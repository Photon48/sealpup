package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/git/gittest"
)

func TestEnter_ExistingWorktree(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")
	wtPath := filepath.Join(repo.Container(), "feat-foo")
	repo.Git("worktree", "add", wtPath, "feat/foo")

	e, out, _ := testEnv(repo.Dir)
	if code := Run(e, []string{"enter", "feat/foo"}); code != 0 {
		t.Fatalf("enter exit = %d, want 0", code)
	}
	got, _ := filepath.EvalSymlinks(wantPathStdout(t, out.String()))
	want, _ := filepath.EvalSymlinks(wtPath)
	if got != want {
		t.Errorf("enter path = %q, want %q", got, want)
	}
}

func TestEnter_BranchWithoutWorktreeOffersCreate(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("bar")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"enter", "bar"}); code != 0 {
		t.Fatalf("enter exit = %d, want 0", code)
	}
	if !strings.Contains(errb.String(), "No worktree for") {
		t.Errorf("expected create offer, got %q", errb.String())
	}
	if filepath.Base(wantPathStdout(t, out.String())) != "bar" {
		t.Errorf("expected worktree created at bar")
	}
}

func TestEnter_DeletedDirRecreates(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("gone")
	wtPath := filepath.Join(repo.Container(), "gone")
	repo.Git("worktree", "add", wtPath, "gone")

	// Simulate someone rm -rf'ing the worktree directory.
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatal(err)
	}

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"enter", "gone"}); code != 0 {
		t.Fatalf("enter exit = %d, want 0", code)
	}
	if !strings.Contains(errb.String(), "deleted from disk") {
		t.Errorf("expected recreate offer, got %q", errb.String())
	}
	got := wantPathStdout(t, out.String())
	if !dirExists(got) {
		t.Errorf("worktree not recreated on disk at %q", got)
	}
}

func TestEnter_NoBranch(t *testing.T) {
	repo := gittest.New(t)
	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"enter", "ghost"}); code != 1 {
		t.Fatalf("enter ghost exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "no branch named") || !strings.Contains(errb.String(), "sealpup new ghost") {
		t.Errorf("expected no-branch error with create hint, got %q", errb.String())
	}
}
