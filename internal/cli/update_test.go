package cli

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdate_AlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Version":"v0.1.0"}`))
	}))
	defer srv.Close()
	t.Setenv("SEALPUP_UPDATE_URL", srv.URL)

	// The proxy's latest (v0.1.0) is not newer than the running build, so no
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
