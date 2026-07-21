// Package update implements sealpup's self-update plumbing: resolving the
// latest tagged release from the Go module proxy — the exact source
// `go install @latest` uses, so the reminder and the updater can never
// disagree — caching the answer once a day, and deciding when a newer-version
// reminder should be shown. Nothing here ever blocks a command on the network:
// reminders are printed from the cache, and the cache refreshes in the
// background.
package update

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

// Module is the sealpup module path — also what `sealpup update` installs.
const Module = "github.com/Photon48/sealpup"

// interval is how often the proxy is asked for the latest version.
const interval = 24 * time.Hour

// proxyBase returns the module proxy to query, overridable via
// SEALPUP_UPDATE_URL so tests can point it at a fake server.
func proxyBase() string {
	if u := os.Getenv("SEALPUP_UPDATE_URL"); u != "" {
		return u
	}
	return "https://proxy.golang.org"
}

// escapeModule applies the module proxy's case encoding: an uppercase letter
// becomes '!' + its lowercase form (Photon48 → !photon48).
func escapeModule(m string) string {
	var b strings.Builder
	for _, r := range m {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Current picks the version to compare against: the module version stamped by
// `go install …@version` when it is a clean release, else the build-time
// fallback (the cli version var).
func Current(fallback string) string {
	if bi, ok := debug.ReadBuildInfo(); ok && IsRelease(bi.Main.Version) {
		return bi.Main.Version
	}
	return normalize(fallback)
}

// normalize adds the leading "v" module versions carry ("0.2.0" → "v0.2.0").
func normalize(v string) string {
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

// parseSemver parses a strict release version "vMAJOR.MINOR.PATCH". Anything
// else — pseudo-versions, "-dev" suffixes, "(devel)" — is rejected, which is
// what keeps the reminder quiet until a real tag exists.
func parseSemver(v string) (nums [3]int, ok bool) {
	v, found := strings.CutPrefix(v, "v")
	if !found {
		return nums, false
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nums, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || (len(p) > 1 && p[0] == '0') || p[0] == '+' {
			return nums, false
		}
		nums[i] = n
	}
	return nums, true
}

// IsRelease reports whether v is a clean vX.Y.Z release version.
func IsRelease(v string) bool {
	_, ok := parseSemver(v)
	return ok
}

// Newer reports whether latest is a strictly newer release than current. Both
// sides must be clean releases — invalid versions never trigger a reminder.
func Newer(latest, current string) bool {
	l, ok := parseSemver(latest)
	if !ok {
		return false
	}
	c, ok := parseSemver(normalize(current))
	if !ok {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

// cacheEntry is what persists between runs: when we last asked the proxy, and
// the newest release it reported.
type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest,omitempty"`
}

// cachePath returns the per-user cache file for update checks.
func cachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sealpup", "update.json"), nil
}

func readCache() (cacheEntry, bool) {
	var c cacheEntry
	p, err := cachePath()
	if err != nil {
		return c, false
	}
	data, err := os.ReadFile(p)
	if err != nil || json.Unmarshal(data, &c) != nil {
		return cacheEntry{}, false
	}
	return c, true
}

func writeCache(c cacheEntry) {
	p, err := cachePath()
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

// Notice returns the one-line reminder to print on stderr, or "" when the
// cached latest release isn't newer than current (or no check has run yet).
// It never touches the network.
func Notice(current string) string {
	if os.Getenv("SEALPUP_NO_UPDATE_CHECK") == "1" {
		return ""
	}
	c, ok := readCache()
	if !ok || !Newer(c.Latest, current) {
		return ""
	}
	return "🦭 sealpup " + c.Latest + " is available (you have " + normalize(current) + ") — run `sealpup update`"
}

// StartRefresh kicks off the once-daily background check and returns a channel
// that closes when it's done. When no refresh is due (fresh cache, opt-out via
// SEALPUP_NO_UPDATE_CHECK=1) the channel is already closed. Callers bound
// their wait with Wait — a command is never blocked on the network.
func StartRefresh() <-chan struct{} {
	done := make(chan struct{})
	prev, ok := readCache()
	if os.Getenv("SEALPUP_NO_UPDATE_CHECK") == "1" || (ok && time.Since(prev.CheckedAt) < interval) {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		if v, err := FetchLatest(2 * time.Second); err == nil {
			writeCache(cacheEntry{CheckedAt: time.Now(), Latest: v})
		} else {
			// Record the attempt (keeping any previously known release) so a
			// dead network isn't retried on every single command.
			writeCache(cacheEntry{CheckedAt: time.Now(), Latest: prev.Latest})
		}
	}()
	return done
}

// Wait blocks until a StartRefresh finishes or the deadline passes. The fetch
// overlaps the command's own git work, so this rarely waits at all.
func Wait(done <-chan struct{}, d time.Duration) {
	select {
	case <-done:
	case <-time.After(d):
	}
}

// FetchLatest asks the module proxy for the newest tagged release of Module.
func FetchLatest(timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(proxyBase() + "/" + escapeModule(Module) + "/@latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &httpError{resp.Status}
	}
	var body struct{ Version string }
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Version == "" {
		return "", &httpError{"empty version in proxy response"}
	}
	return body.Version, nil
}

type httpError struct{ msg string }

func (e *httpError) Error() string { return "module proxy: " + e.msg }

// MarkLatest records v as the freshly confirmed latest release — called after
// a successful `sealpup update` so the reminder disappears immediately.
func MarkLatest(v string) {
	writeCache(cacheEntry{CheckedAt: time.Now(), Latest: v})
}
