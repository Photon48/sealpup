package git

import (
	"path/filepath"
	"testing"

	"github.com/Photon48/sealpup/internal/git/gittest"
)

func TestDiscoverAndContainer(t *testing.T) {
	repo := gittest.New(t)

	r, err := Discover(repo.Dir)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	// t.TempDir may live behind a symlink (/var -> /private/var on macOS), so
	// compare resolved paths.
	wantMain, _ := filepath.EvalSymlinks(repo.Dir)
	gotMain, _ := filepath.EvalSymlinks(r.MainDir)
	if gotMain != wantMain {
		t.Fatalf("MainDir = %q, want %q", gotMain, wantMain)
	}
	if r.Name() != "repo" {
		t.Fatalf("Name = %q, want repo", r.Name())
	}
	wantContainer, _ := filepath.EvalSymlinks(filepath.Dir(wantMain))
	gotContainer, _ := filepath.EvalSymlinks(filepath.Dir(r.Container()))
	if gotContainer != wantContainer || filepath.Base(r.Container()) != "repo-worktrees" {
		t.Fatalf("Container = %q, want sibling repo-worktrees", r.Container())
	}
}

func TestDiscoverFromLinkedWorktree(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")
	wtPath := filepath.Join(repo.Container(), "feat-foo")
	repo.Git("worktree", "add", wtPath, "feat/foo")

	// Discover from inside the linked worktree must still point at the main repo.
	r, err := Discover(wtPath)
	if err != nil {
		t.Fatalf("Discover from worktree: %v", err)
	}
	want, _ := filepath.EvalSymlinks(repo.Dir)
	got, _ := filepath.EvalSymlinks(r.MainDir)
	if got != want {
		t.Fatalf("MainDir from worktree = %q, want %q", got, want)
	}
}

func TestBranchExists(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")

	if !BranchExists(repo.Dir, "main") {
		t.Error("expected main to exist")
	}
	if !BranchExists(repo.Dir, "feat/foo") {
		t.Error("expected feat/foo to exist")
	}
	if BranchExists(repo.Dir, "nope") {
		t.Error("did not expect nope to exist")
	}
}

func TestStatus(t *testing.T) {
	repo := gittest.New(t)
	if lines, err := Status(repo.Dir); err != nil || len(lines) != 0 {
		t.Fatalf("clean status: got %v, err %v", lines, err)
	}
	repo.WriteFile(repo.Dir, "a.txt", "hi")
	lines, err := Status(repo.Dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("dirty status: got %d lines, want 1: %v", len(lines), lines)
	}
}

func TestWorktreesIntegration(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")
	wtPath := filepath.Join(repo.Container(), "feat-foo")
	repo.Git("worktree", "add", wtPath, "feat/foo")

	wts, err := Worktrees(repo.Dir)
	if err != nil {
		t.Fatalf("Worktrees: %v", err)
	}
	if len(wts) != 2 {
		t.Fatalf("got %d worktrees, want 2: %+v", len(wts), wts)
	}
	if FindByBranch(wts, "feat/foo") == nil {
		t.Fatalf("feat/foo worktree not found in %+v", wts)
	}
}
