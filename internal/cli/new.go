package cli

import (
	"os"
	"path/filepath"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// cmdNew creates a worktree (and, if needed, its branch) in the sibling
// container and enters it. On success it prints the new worktree path to stdout
// for the shim to cd into; the guardrails below turn each worktree footgun into
// an offer to do the right thing instead.
func cmdNew(e Env, args []string) error {
	branch, err := singleBranchArg("new", args)
	if err != nil {
		return err
	}

	repo, err := e.repo()
	if err != nil {
		return err
	}
	_ = git.Prune(e.Dir) // opportunistic self-heal

	dirName := sanitizeDirName(branch)
	if dirName == "" {
		return ui.Hintf(
			"branch name "+quote(branch)+" has no usable characters for a directory name",
			"pick a branch name with letters or digits",
		)
	}
	dir := filepath.Join(repo.Container(), dirName)

	wts, err := git.Worktrees(e.Dir)
	if err != nil {
		return err
	}

	// Guardrail: branch already has a worktree — offer to enter it instead.
	if wt := git.FindByBranch(wts, branch); wt != nil {
		if !e.confirm("Branch "+quote(branch)+" is already checked out at "+prettyPath(wt.Path)+". Enter it instead?", true) {
			return nil // user declined; success, nothing printed, shim does nothing
		}
		return e.emitPath(wt.Path)
	}

	// Guardrail: the target directory is taken by something else (e.g. a
	// sanitization collision between feat/foo and feat-foo, or a stray dir).
	if other := worktreeAtPath(wts, dir); other != nil {
		return ui.Hintf(
			"the directory for "+quote(branch)+" is already used by branch "+quote(other.Branch),
			"choose a branch name that maps to a different directory",
		)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		return ui.Hintf(
			"a directory already exists at "+prettyPath(dir),
			"remove it, or pick a different branch name",
		)
	}

	if git.BranchExists(e.Dir, branch) {
		// Branch exists but isn't checked out anywhere — make a worktree for it.
		if !e.confirm("Branch "+quote(branch)+" already exists. Create a worktree for it?", true) {
			return nil
		}
		if err := ensureContainer(repo); err != nil {
			return err
		}
		if err := git.AddExisting(e.Dir, dir, branch); err != nil {
			return err
		}
	} else {
		if err := ensureContainer(repo); err != nil {
			return err
		}
		if err := git.AddNewBranch(e.Dir, dir, branch); err != nil {
			return err
		}
	}

	e.prompter().Infof("✓ created worktree for %s at %s", branch, prettyPath(dir))
	return e.emitPath(dir)
}

// ensureContainer makes the sibling worktree directory if it doesn't exist.
func ensureContainer(repo *git.Repo) error {
	if err := os.MkdirAll(repo.Container(), 0o755); err != nil {
		return ui.Errorf("could not create worktree directory: %v", err)
	}
	return nil
}

// worktreeAtPath returns a worktree registered at the given filesystem path
// (resolving symlinks), or nil.
func worktreeAtPath(wts []git.Worktree, path string) *git.Worktree {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		target = path
	}
	for i := range wts {
		wp, err := filepath.EvalSymlinks(wts[i].Path)
		if err != nil {
			wp = wts[i].Path
		}
		if wp == target {
			return &wts[i]
		}
	}
	return nil
}
