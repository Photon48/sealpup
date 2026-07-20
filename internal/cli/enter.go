package cli

import (
	"os"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// cmdEnter resolves a branch to its worktree and prints the path for the shim
// to cd into. Its guardrails cover the three ways this goes wrong: the worktree
// was deleted from disk, the branch has no worktree yet, or the branch doesn't
// exist at all — each becomes an offer rather than a dead end.
func cmdEnter(e Env, args []string) error {
	branch, err := singleBranchArg("enter", args)
	if err != nil {
		return err
	}

	repo, err := e.repo()
	if err != nil {
		return err
	}

	// Read the worktree list BEFORE pruning: a prune would erase the very
	// evidence (a prunable entry) that tells us a worktree dir was deleted.
	wts, err := git.Worktrees(e.Dir)
	if err != nil {
		return err
	}

	if wt := git.FindByBranch(wts, branch); wt != nil {
		if wt.Prunable == "" && dirExists(wt.Path) {
			return e.emitPath(wt.Path)
		}
		// Registered but the directory is gone from disk.
		if !e.confirm("The worktree for "+quote(branch)+" was deleted from disk. Prune and recreate it?", true) {
			return nil
		}
		_ = git.Prune(e.Dir)
		return e.createAndEnter(repo, branch, freshWorktrees(e))
	}

	if git.BranchExists(e.Dir, branch) {
		if !e.confirm("No worktree for "+quote(branch)+" yet. Create one?", true) {
			return nil
		}
		return e.createAndEnter(repo, branch, wts)
	}

	return ui.Hintf(
		"no branch named "+quote(branch),
		"create it with:  sealpup new "+branch,
	)
}

// createAndEnter plans a worktree path, checks out the existing branch there,
// and emits the path.
func (e Env) createAndEnter(repo *git.Repo, branch string, wts []git.Worktree) error {
	dir, err := planWorktreePath(repo, branch, wts)
	if err != nil {
		return err
	}
	if err := ensureContainer(repo); err != nil {
		return err
	}
	if err := git.AddExisting(e.Dir, dir, branch); err != nil {
		return err
	}
	e.prompter().Successf("created worktree for %s at %s", branch, prettyPath(dir))
	return e.emitPath(dir)
}

// freshWorktrees re-reads the worktree list, tolerating errors (returns nil so
// planning proceeds on a best-effort basis after a prune).
func freshWorktrees(e Env) []git.Worktree {
	wts, err := git.Worktrees(e.Dir)
	if err != nil {
		return nil
	}
	return wts
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
