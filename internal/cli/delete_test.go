package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/agents"
	"github.com/Photon48/sealpup/internal/git/gittest"
)

func TestDelete_RefusesWhenAgentRunning(t *testing.T) {
	repo := gittest.New(t)
	wt := makeWorktree(t, repo, "busy")
	key := repo.Git("-C", wt, "rev-parse", "--show-toplevel")
	stubAgents(t, map[string][]agents.Agent{
		key: {{PID: 4812, Name: "claude"}},
	})

	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "busy"}); code != 1 {
		t.Fatalf("delete with agent exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "claude (pid 4812)") || !strings.Contains(errb.String(), "--force") {
		t.Errorf("expected agent refusal with --force hint, got %q", errb.String())
	}
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "busy") {
		t.Error("worktree should still exist after refusal")
	}
}

func TestDelete_ForceBypassesAgent(t *testing.T) {
	repo := gittest.New(t)
	wt := makeWorktree(t, repo, "busy")
	key := repo.Git("-C", wt, "rev-parse", "--show-toplevel")
	stubAgents(t, map[string][]agents.Agent{
		key: {{PID: 4812, Name: "claude"}},
	})

	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "--force", "busy"}); code != 0 {
		t.Fatalf("force delete exit = %d, want 0\n%s", code, errb.String())
	}
	if strings.Contains(repo.Git("worktree", "list", "--porcelain"), "busy") {
		t.Error("worktree should be removed under --force despite agent")
	}
}

// makeWorktree adds a worktree for a new branch and returns its path.
func makeWorktree(t *testing.T, repo *gittest.Repo, branch string) string {
	t.Helper()
	repo.Branch(branch)
	p := filepath.Join(repo.Container(), sanitizeDirName(branch))
	repo.Git("worktree", "add", p, branch)
	return p
}

func TestDelete_CleanWorktreeKeepsBranch(t *testing.T) {
	repo := gittest.New(t)
	makeWorktree(t, repo, "feat/foo")

	// Answer "n" to the "also delete branch?" prompt.
	e, _, errb := envWith(repo.Dir, "n\n")
	if code := Run(e, []string{"delete", "feat/foo"}); code != 0 {
		t.Fatalf("delete exit = %d, want 0\n%s", code, errb.String())
	}
	if strings.Contains(repo.Git("worktree", "list", "--porcelain"), "feat/foo") {
		t.Error("worktree for feat/foo should be gone")
	}
	if !BranchExistsForTest(repo, "feat/foo") {
		t.Error("branch feat/foo should remain (declined deletion)")
	}
}

func TestDelete_WithBranchFlagRemovesBoth(t *testing.T) {
	repo := gittest.New(t)
	makeWorktree(t, repo, "bar")

	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "--branch", "bar"}); code != 0 {
		t.Fatalf("delete --branch exit = %d, want 0\n%s", code, errb.String())
	}
	if BranchExistsForTest(repo, "bar") {
		t.Error("branch bar should be deleted with --branch")
	}
}

func TestDelete_DirtyRefusedWithoutForce(t *testing.T) {
	repo := gittest.New(t)
	wt := makeWorktree(t, repo, "dirty")
	repo.WriteFile(wt, "wip.txt", "x")

	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "dirty"}); code != 1 {
		t.Fatalf("dirty delete exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "uncommitted") || !strings.Contains(errb.String(), "--force") {
		t.Errorf("expected dirty refusal with --force hint, got %q", errb.String())
	}
	// Still there.
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "dirty") {
		t.Error("dirty worktree should not have been removed")
	}
}

func TestDelete_DirtyForceRemoves(t *testing.T) {
	repo := gittest.New(t)
	wt := makeWorktree(t, repo, "dirty")
	repo.WriteFile(wt, "wip.txt", "x")

	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "--force", "dirty"}); code != 0 {
		t.Fatalf("force delete exit = %d, want 0\n%s", code, errb.String())
	}
	if strings.Contains(repo.Git("worktree", "list", "--porcelain"), "dirty") {
		t.Error("worktree should be removed under --force")
	}
}

func TestDelete_RefusesCurrentWorktree(t *testing.T) {
	repo := gittest.New(t)
	wt := makeWorktree(t, repo, "here")

	// Run from inside the worktree we're trying to delete.
	e, _, errb := testEnv(wt)
	if code := Run(e, []string{"delete", "here"}); code != 1 {
		t.Fatalf("delete current exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "currently inside") {
		t.Errorf("expected current-worktree refusal, got %q", errb.String())
	}
}

func TestDelete_RefusesMainWorktree(t *testing.T) {
	repo := gittest.New(t)
	// Run from a linked worktree so cwd isn't main, but target main.
	other := makeWorktree(t, repo, "other")
	e, _, errb := testEnv(other)
	if code := Run(e, []string{"delete", "main"}); code != 1 {
		t.Fatalf("delete main exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "main worktree") {
		t.Errorf("expected main-worktree refusal, got %q", errb.String())
	}
}

func TestDelete_ContainerRemovedWhenEmpty(t *testing.T) {
	repo := gittest.New(t)
	makeWorktree(t, repo, "solo")

	e, _, _ := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "--keep-branch", "solo"}); code != 0 {
		t.Fatalf("delete exit = %d, want 0", code)
	}
	if _, err := os.Stat(repo.Container()); !os.IsNotExist(err) {
		t.Errorf("empty container should be removed, stat err = %v", err)
	}
}

func TestDelete_NothingNamed(t *testing.T) {
	repo := gittest.New(t)
	e, _, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"delete", "ghost"}); code != 1 {
		t.Fatalf("delete ghost exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "nothing named") {
		t.Errorf("expected nothing-named error, got %q", errb.String())
	}
}

// BranchExistsForTest is a tiny wrapper so tests read clearly.
func BranchExistsForTest(repo *gittest.Repo, branch string) bool {
	out := repo.Git("branch", "--list", branch)
	return strings.Contains(out, branch)
}
