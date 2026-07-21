package update

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tempCache points the user cache dir at a temp home so tests never touch the
// real ~/Library/Caches (darwin) or ~/.cache (linux).
func tempCache(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
}

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.3.0", "v0.2.0", true},
		{"v0.3.0", "0.2.0", true}, // current normalized
		{"v0.2.10", "v0.2.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.2.0", "v0.2.0", false},
		{"v0.1.9", "v0.2.0", false},
		{"v0.0.0-20260720140707-2be974cc7839", "v0.1.0", false}, // pseudo-version never counts
		{"v0.3.0", "0.2.0-dev", false},                          // dev build never nagged
		{"v0.3.0", "(devel)", false},
		{"v0.3.0", "", false},
		{"", "v0.2.0", false},
		{"v01.2.3", "v0.1.0", false}, // leading zero rejected
	}
	for _, c := range cases {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestMaxReleaseTag(t *testing.T) {
	out := "abc\trefs/tags/v0.2.0\n" +
		"def\trefs/tags/v0.10.0\n" + // numeric compare, not lexical
		"def\trefs/tags/v0.10.0^{}\n" + // peeled ref ignored as duplicate
		"ghi\trefs/tags/v0.9.9\n" +
		"jkl\trefs/tags/nightly\n" + // non-release tag ignored
		"mno\trefs/tags/v1.0.0-rc1\n" // prerelease ignored
	if got := maxReleaseTag(out); got != "v0.10.0" {
		t.Fatalf("maxReleaseTag = %q, want v0.10.0", got)
	}
	if got := maxReleaseTag("abc\trefs/heads/main\n"); got != "" {
		t.Fatalf("expected no release tag, got %q", got)
	}
}

func TestEscapeModule(t *testing.T) {
	if got := escapeModule("github.com/Photon48/sealpup"); got != "github.com/!photon48/sealpup" {
		t.Fatalf("escapeModule = %q", got)
	}
}

func TestNotice(t *testing.T) {
	tempCache(t)

	// No cache yet → silent.
	if n := Notice("v0.2.0"); n != "" {
		t.Fatalf("expected no notice without a cache, got %q", n)
	}

	writeCache(cacheEntry{CheckedAt: time.Now(), Latest: "v9.9.9"})
	n := Notice("0.2.0")
	if !strings.Contains(n, "v9.9.9") || !strings.Contains(n, "v0.2.0") || !strings.Contains(n, "sealpup update") {
		t.Fatalf("notice missing versions or hint: %q", n)
	}

	// Cached latest not newer → silent.
	writeCache(cacheEntry{CheckedAt: time.Now(), Latest: "v0.2.0"})
	if n := Notice("v0.2.0"); n != "" {
		t.Fatalf("expected no notice when up to date, got %q", n)
	}

	// Opt-out silences the reminder even with a newer cached release.
	writeCache(cacheEntry{CheckedAt: time.Now(), Latest: "v9.9.9"})
	t.Setenv("SEALPUP_NO_UPDATE_CHECK", "1")
	if n := Notice("v0.2.0"); n != "" {
		t.Fatalf("opt-out should silence the notice, got %q", n)
	}
}

func TestStartRefresh(t *testing.T) {
	tempCache(t)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if want := "/github.com/!photon48/sealpup/@latest"; r.URL.Path != want {
			t.Errorf("proxy path = %q, want %q", r.URL.Path, want)
		}
		w.Write([]byte(`{"Version":"v1.2.3"}`))
	}))
	defer srv.Close()
	t.Setenv("SEALPUP_UPDATE_URL", srv.URL)

	Wait(StartRefresh(), 5*time.Second)
	c, ok := readCache()
	if !ok || c.Latest != "v1.2.3" {
		t.Fatalf("cache after refresh = %+v (ok=%v), want latest v1.2.3", c, ok)
	}
	if hits != 1 {
		t.Fatalf("expected 1 proxy hit, got %d", hits)
	}

	// Fresh cache → second refresh is a no-op.
	Wait(StartRefresh(), 5*time.Second)
	if hits != 1 {
		t.Fatalf("fresh cache should skip the proxy, got %d hits", hits)
	}
}

func TestStartRefresh_OptOut(t *testing.T) {
	tempCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("proxy hit despite opt-out")
	}))
	defer srv.Close()
	t.Setenv("SEALPUP_UPDATE_URL", srv.URL)
	t.Setenv("SEALPUP_NO_UPDATE_CHECK", "1")

	Wait(StartRefresh(), 5*time.Second)
	if _, ok := readCache(); ok {
		t.Fatal("opt-out should not write a cache")
	}
}

func TestStartRefresh_KeepsKnownLatestOnFailure(t *testing.T) {
	tempCache(t)
	// A stale cache with a known release, and a proxy that now errors.
	writeCache(cacheEntry{CheckedAt: time.Now().Add(-48 * time.Hour), Latest: "v1.0.0"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("SEALPUP_UPDATE_URL", srv.URL)

	Wait(StartRefresh(), 5*time.Second)
	c, ok := readCache()
	if !ok || c.Latest != "v1.0.0" {
		t.Fatalf("failed refresh should keep known latest, got %+v", c)
	}
	if time.Since(c.CheckedAt) > time.Minute {
		t.Fatal("failed refresh should still bump checked_at")
	}
}
