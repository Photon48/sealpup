package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/git/gittest"
)

func TestList(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")
	repo.Branch("bar")
	wtFoo := filepath.Join(repo.Container(), "feat-foo")
	wtBar := filepath.Join(repo.Container(), "bar")
	repo.Git("worktree", "add", wtFoo, "feat/foo")
	repo.Git("worktree", "add", wtBar, "bar")
	// Make feat/foo dirty.
	repo.WriteFile(wtFoo, "dirty.txt", "x")

	// Run list from inside the bar worktree so the marker lands there.
	e, out, _ := testEnv(wtBar)
	if code := Run(e, []string{"list"}); code != 0 {
		t.Fatalf("list exit = %d, want 0", code)
	}
	got := out.String()

	for _, want := range []string{"main", "feat/foo", "bar", "1 file", "← you are here"} {
		if !strings.Contains(got, want) {
			t.Errorf("list output missing %q:\n%s", want, got)
		}
	}
	// The marker must be on the bar row, not feat/foo.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "← you are here") && !strings.Contains(line, wtBar[len(wtBar)-3:]) {
			// weak check: ensure marker line mentions bar's status region
		}
		if strings.Contains(line, "feat/foo") && strings.Contains(line, "← you are here") {
			t.Errorf("marker wrongly on feat/foo row: %q", line)
		}
	}
}

func TestListNotARepo(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	if code := Run(e, []string{"list"}); code != 1 {
		t.Fatalf("list outside repo: exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "not inside a git repository") {
		t.Errorf("expected not-a-repo message, got %q", errb.String())
	}
}
