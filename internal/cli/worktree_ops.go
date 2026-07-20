package cli

import (
	"os"
	"path/filepath"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// planWorktreePath computes where a new worktree for branch should live and
// verifies the spot is free, returning a clean UserError on a sanitization
// collision or a pre-existing directory. It does not touch the filesystem.
func planWorktreePath(repo *git.Repo, branch string, wts []git.Worktree) (string, error) {
	dirName := sanitizeDirName(branch)
	if dirName == "" {
		return "", ui.Hintf(
			"branch name "+quote(branch)+" has no usable characters for a directory name",
			"pick a branch name with letters or digits",
		)
	}
	dir := filepath.Join(repo.Container(), dirName)

	if other := worktreeAtPath(wts, dir); other != nil && other.Branch != branch {
		return "", ui.Hintf(
			"the directory for "+quote(branch)+" is already used by branch "+quote(other.Branch),
			"choose a branch name that maps to a different directory",
		)
	}
	if _, err := os.Stat(dir); err == nil {
		return "", ui.Hintf(
			"a directory already exists at "+prettyPath(dir),
			"remove it, or pick a different branch name",
		)
	}
	return dir, nil
}
