package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// repo resolves the git repository containing the Env's working directory,
// translating git's "not a repository" error into a clean UserError.
func (e Env) repo() (*git.Repo, error) {
	r, err := git.Discover(e.Dir)
	if err != nil {
		return nil, ui.Hintf(
			"not inside a git repository",
			"run sealpup from within a git repo, or run 'git init' first",
		)
	}
	return r, nil
}

// isCurrent reports whether the Env's cwd is at or below worktree path. Both are
// resolved through symlinks so /var vs /private/var (macOS) doesn't defeat it.
func (e Env) isCurrent(path string) bool {
	cwd, err := filepath.EvalSymlinks(e.Dir)
	if err != nil {
		cwd = e.Dir
	}
	wt, err := filepath.EvalSymlinks(path)
	if err != nil {
		wt = path
	}
	if cwd == wt {
		return true
	}
	return strings.HasPrefix(cwd, wt+string(filepath.Separator))
}

// prettyPath shortens an absolute path by replacing the home prefix with ~.
func prettyPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}
