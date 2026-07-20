package cli

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// cmdDelete removes a worktree and, optionally, its branch, refusing anything
// that would lose work or strand the user: a dirty worktree (without --force),
// the worktree you're standing in, or the main checkout.
func cmdDelete(e Env, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(e.Stderr)
	force := fs.Bool("force", false, "remove even if the worktree has uncommitted changes")
	delBranch := fs.Bool("branch", false, "also delete the branch without asking")
	keepBranch := fs.Bool("keep-branch", false, "keep the branch, delete only the worktree")

	// All delete flags are booleans, so we can hoist them ahead of the branch
	// name and accept both `delete --force x` and `delete x --force`.
	flags, positional := partitionFlags(args)
	if err := fs.Parse(flags); err != nil {
		return ui.Errorf("invalid flags for delete")
	}
	if *delBranch && *keepBranch {
		return ui.Errorf("--branch and --keep-branch cannot be used together")
	}
	branch, err := singleBranchArg("delete", positional)
	if err != nil {
		return err
	}

	repo, err := e.repo()
	if err != nil {
		return err
	}
	_ = git.Prune(e.Dir)

	wts, err := git.Worktrees(e.Dir)
	if err != nil {
		return err
	}
	wt := git.FindByBranch(wts, branch)

	// No worktree: maybe the branch still exists and the user wants it gone.
	if wt == nil {
		if git.BranchExists(e.Dir, branch) {
			return e.deleteBranchOnly(branch, *delBranch, *keepBranch, *force)
		}
		return ui.Hintf(
			"nothing named "+quote(branch)+" to delete",
			"run 'sealpup list' to see your worktrees",
		)
	}

	// Refuse to delete the primary worktree.
	if samePath(wt.Path, repo.MainDir) {
		return ui.Hintf(
			"refusing to delete the main worktree ("+quote(branch)+")",
			"the primary checkout can't be removed by sealpup",
		)
	}
	// Refuse to delete the worktree you're currently inside.
	if e.isCurrent(wt.Path) {
		return ui.Hintf(
			"you're currently inside the worktree for "+quote(branch),
			"switch away first, e.g.  sealpup enter <other-branch>",
		)
	}

	// Guard uncommitted work.
	if lines, statErr := git.Status(wt.Path); statErr == nil && len(lines) > 0 && !*force {
		return ui.Hintf(
			describeDirty(branch, lines),
			"commit or stash the changes, or rerun with --force",
		)
	}

	if err := git.RemoveWorktree(e.Dir, wt.Path, *force); err != nil {
		return err
	}
	_ = git.Prune(e.Dir)
	e.prompter().Successf("removed worktree for %s", branch)
	removeIfEmpty(repo.Container())

	return e.maybeDeleteBranch(branch, *delBranch, *keepBranch, *force)
}

// deleteBranchOnly handles the case where the branch exists but has no worktree.
func (e Env) deleteBranchOnly(branch string, delBranch, keepBranch, force bool) error {
	if keepBranch {
		return ui.Msg("no worktree for " + quote(branch) + " (branch left untouched)")
	}
	if !delBranch && !e.confirm("No worktree for "+quote(branch)+", but the branch exists. Delete the branch?", false) {
		return nil
	}
	if err := git.DeleteBranch(e.Dir, branch, force); err != nil {
		return branchDeleteError(branch, force, err)
	}
	e.prompter().Successf("deleted branch %s", branch)
	return nil
}

// maybeDeleteBranch decides whether to delete the branch after its worktree is
// gone, honoring the flags or asking (default No).
func (e Env) maybeDeleteBranch(branch string, delBranch, keepBranch, force bool) error {
	if keepBranch {
		return nil
	}
	if !delBranch && !e.confirm("Also delete branch "+quote(branch)+"?", false) {
		return nil
	}
	if err := git.DeleteBranch(e.Dir, branch, force); err != nil {
		return branchDeleteError(branch, force, err)
	}
	e.prompter().Successf("deleted branch %s", branch)
	return nil
}

// branchDeleteError turns git's unmerged-branch refusal into an actionable hint.
func branchDeleteError(branch string, force bool, err error) error {
	if !force {
		return ui.Hintf(
			"branch "+quote(branch)+" isn't fully merged, so it wasn't deleted",
			"rerun with --force to delete it anyway (git branch -D)",
		)
	}
	return err
}

// describeDirty summarizes uncommitted changes, listing the first few paths.
func describeDirty(branch string, lines []string) string {
	const max = 3
	var files []string
	for i, l := range lines {
		if i >= max {
			break
		}
		// porcelain lines are "XY path"; take the path part.
		if len(l) > 3 {
			files = append(files, strings.TrimSpace(l[3:]))
		}
	}
	suffix := ""
	if len(lines) > max {
		suffix = ", …"
	}
	return "worktree for " + quote(branch) + " has " + plural(len(lines), "uncommitted change") +
		" (" + strings.Join(files, ", ") + suffix + ")"
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

// removeIfEmpty deletes dir if it exists and contains no entries.
func removeIfEmpty(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		_ = os.Remove(dir)
	}
}
