package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/git/gittest"
)

// wantPathStdout asserts stdout is exactly one path line and returns it.
func wantPathStdout(t *testing.T, out string) string {
	t.Helper()
	trimmed := strings.TrimRight(out, "\n")
	if trimmed == "" || strings.Contains(trimmed, "\n") {
		t.Fatalf("expected exactly one path line on stdout, got %q", out)
	}
	return trimmed
}

func TestNew_CreatesBranchAndWorktree(t *testing.T) {
	repo := gittest.New(t)
	e, out, _ := testEnv(repo.Dir)

	if code := Run(e, []string{"new", "feat/foo"}); code != 0 {
		t.Fatalf("new exit = %d, want 0", code)
	}
	path := wantPathStdout(t, out.String())

	if filepath.Base(path) != "feat-foo" {
		t.Errorf("worktree dir = %q, want basename feat-foo", path)
	}
	wtList := repo.Git("worktree", "list", "--porcelain")
	if !strings.Contains(wtList, "refs/heads/feat/foo") {
		t.Errorf("feat/foo not registered as a worktree:\n%s", wtList)
	}
}

func TestNew_ExistingBranchOffersWorktree(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("bar") // exists, not checked out
	e, out, errb := testEnv(repo.Dir)

	if code := Run(e, []string{"new", "bar"}); code != 0 {
		t.Fatalf("new exit = %d, want 0", code)
	}
	if !strings.Contains(errb.String(), "already exists") {
		t.Errorf("expected 'already exists' prompt on stderr, got %q", errb.String())
	}
	path := wantPathStdout(t, out.String())
	if filepath.Base(path) != "bar" {
		t.Errorf("worktree dir = %q, want basename bar", path)
	}
}

func TestNew_AlreadyCheckedOutOffersEnter(t *testing.T) {
	repo := gittest.New(t)
	e, out, errb := testEnv(repo.Dir)

	// main is checked out at the main repo; `new main` should offer to enter it
	// and (default yes) print the main repo path.
	if code := Run(e, []string{"new", "main"}); code != 0 {
		t.Fatalf("new main exit = %d, want 0", code)
	}
	if !strings.Contains(errb.String(), "already checked out") {
		t.Errorf("expected 'already checked out' prompt, got %q", errb.String())
	}
	path := wantPathStdout(t, out.String())
	resolved, _ := filepath.EvalSymlinks(path)
	wantMain, _ := filepath.EvalSymlinks(repo.Dir)
	if resolved != wantMain {
		t.Errorf("enter path = %q, want main repo %q", resolved, wantMain)
	}
}

func TestNew_DeclineCheckedOutPrintsNothing(t *testing.T) {
	repo := gittest.New(t)
	e, out, _ := envWith(repo.Dir, "n\n")

	if code := Run(e, []string{"new", "main"}); code != 0 {
		t.Fatalf("declining exit = %d, want 0", code)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("declining should print nothing to stdout, got %q", out.String())
	}
}

func TestNew_NotInteractiveDefaultsProceed(t *testing.T) {
	repo := gittest.New(t)
	e, out, _ := envWith(repo.Dir, "")
	e.IsInteractive = false

	if code := Run(e, []string{"new", "solo"}); code != 0 {
		t.Fatalf("non-interactive new exit = %d, want 0", code)
	}
	if wantPathStdout(t, out.String()) == "" {
		t.Error("expected a path even non-interactively")
	}
}

func TestNew_ShimlessNoteShownOnceThenSuppressed(t *testing.T) {
	t.Setenv("SEALPUP_STATE_DIR", t.TempDir())
	repo := gittest.New(t)

	// First shim-less run: note appears, pointing at `sealpup setup`.
	e1, _, errb1 := testEnv(repo.Dir)
	e1.ShimActive = false
	if code := Run(e1, []string{"new", "first"}); code != 0 {
		t.Fatalf("new exit = %d, want 0", code)
	}
	if !strings.Contains(errb1.String(), "sealpup setup") {
		t.Errorf("expected one-time shim note pointing at 'sealpup setup', got %q", errb1.String())
	}

	// Second shim-less run: note is suppressed by the stamp.
	e2, _, errb2 := testEnv(repo.Dir)
	e2.ShimActive = false
	if code := Run(e2, []string{"new", "second"}); code != 0 {
		t.Fatalf("new exit = %d, want 0", code)
	}
	if strings.Contains(errb2.String(), "sealpup setup") {
		t.Errorf("shim note should only appear once, but showed again: %q", errb2.String())
	}
}
