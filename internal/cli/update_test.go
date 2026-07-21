package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// taggedOrigin creates a local git repo carrying a release tag, usable as the
// SEALPUP_ORIGIN_URL update resolves releases from.
func taggedOrigin(t *testing.T, tag string) string {
	t.Helper()
	repo := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"},
		{"tag", tag},
	} {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return repo
}

func TestUpdate_AlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("SEALPUP_ORIGIN_URL", taggedOrigin(t, "v0.1.0"))

	// The origin's latest (v0.1.0) is not newer than the running build, so no
	// `go install` runs — the command reports up to date and exits 0.
	old := version
	version = "0.5.0"
	defer func() { version = old }()

	e, _, errb := testEnv(home)
	if code := Run(e, []string{"update"}); code != 0 {
		t.Fatalf("update exit = %d, want 0\n%s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "already up to date") {
		t.Fatalf("expected 'already up to date', got %q", errb.String())
	}
}

func TestUpdate_RejectsArguments(t *testing.T) {
	e, _, _ := testEnv(t.TempDir())
	if code := Run(e, []string{"update", "now"}); code != 1 {
		t.Fatalf("update with args: exit = %d, want 1", code)
	}
}
