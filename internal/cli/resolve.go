package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Photon48/sealpup/internal/git"
	"github.com/Photon48/sealpup/internal/ui"
)

// repo resolves the git repository containing the Env's working directory,
// translating git's failures into clean UserErrors: a missing git binary, or a
// directory that isn't inside a repo.
func (e Env) repo() (*git.Repo, error) {
	r, err := git.Discover(e.Dir)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ui.Hintf(
				"git is not installed or not on your PATH",
				"install git, then try again",
			)
		}
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

// confirm asks a yes/no question, routing it to stderr. In a non-interactive
// session it does not block — it proceeds with the default answer, so scripted
// use (cd "$(sealpup enter x)") never hangs.
func (e Env) confirm(question string, defaultYes bool) bool {
	ans, notInteractive := e.prompter().Confirm(question, defaultYes)
	if notInteractive {
		return defaultYes
	}
	return ans
}

// emitPath is the shim contract: print the resolved worktree path (and nothing
// else) to stdout so the shell function can cd into it. When the shim isn't
// installed, a stderr note explains why the directory didn't change.
func (e Env) emitPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	fmt.Fprintln(e.Stdout, abs)
	if !e.ShimActive {
		fmt.Fprintf(e.Stderr,
			"note: sealpup can't change your shell's directory on its own.\n"+
				"      add this to your shell rc to enable it:  eval \"$(sealpup init zsh)\"\n")
	}
	return nil
}

// samePath reports whether two paths point at the same location, resolving
// symlinks so /var and /private/var compare equal on macOS.
func samePath(a, b string) bool {
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		ra = a
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		rb = b
	}
	return ra == rb
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
