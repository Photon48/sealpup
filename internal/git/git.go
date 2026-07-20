// Package git shells out to the user's real `git` binary. sealpup deliberately
// does not embed a git implementation: matching the exact behavior of the git
// the user already has (prune semantics, checkout guards, config) is a feature,
// not a limitation.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Error wraps a failed git invocation, preserving git's own stderr so callers
// can surface it or pattern-match on it.
type Error struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		msg = e.Err.Error()
	}
	return fmt.Sprintf("git %s: %s", strings.Join(e.Args, " "), msg)
}

func (e *Error) Unwrap() error { return e.Err }

// Run executes `git <args...>` with its working directory set to dir and
// returns trimmed stdout. On non-zero exit it returns an *Error carrying git's
// stderr. The process environment is inherited, so the user's git identity and
// config apply (tests isolate via GIT_CONFIG_* env vars).
func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return out.String(), &Error{Args: args, Stderr: errb.String(), Err: err}
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// runOK reports whether a git command exited zero, discarding output. Used for
// existence checks like show-ref where the exit code is the whole answer.
func runOK(dir string, args ...string) bool {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// Repo identifies a git repository from any directory inside it (including a
// linked worktree). MainDir is the primary worktree's root — the directory
// whose .git holds the common object store — so worktree paths resolve to the
// same sibling container no matter which worktree you invoke sealpup from.
type Repo struct {
	MainDir   string // absolute path to the primary worktree root
	CommonDir string // absolute path to the common .git directory
}

// Discover resolves the repository containing dir. It returns a *Error (which
// surfaces "not a git repository") when dir is not inside one.
func Discover(dir string) (*Repo, error) {
	common, err := Run(dir, "rev-parse", "--git-common-dir")
	if err != nil {
		return nil, err
	}
	// --git-common-dir is resolved relative to git's cwd (dir) when not absolute.
	if !filepath.IsAbs(common) {
		common = filepath.Join(dir, common)
	}
	common = filepath.Clean(common)
	return &Repo{
		MainDir:   filepath.Dir(common),
		CommonDir: common,
	}, nil
}

// Name is the repository's directory name, used to build the worktree
// container name.
func (r *Repo) Name() string { return filepath.Base(r.MainDir) }

// Container is the sibling directory that holds sealpup's worktrees:
// ../<repo>-worktrees relative to the main repo.
func (r *Repo) Container() string {
	return filepath.Join(filepath.Dir(r.MainDir), r.Name()+"-worktrees")
}

// BranchExists reports whether a local branch of the given name exists.
func BranchExists(dir, branch string) bool {
	return runOK(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
}

// Status returns the porcelain status lines (one per changed path) for the
// worktree at dir. An empty slice means clean.
func Status(dir string) ([]string, error) {
	out, err := Run(dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// Prune runs `git worktree prune` best-effort; errors are returned but callers
// typically ignore them (self-heal should never block the real command).
func Prune(dir string) error {
	_, err := Run(dir, "worktree", "prune")
	return err
}

// AddNewBranch creates branch (from current HEAD) and a worktree for it at path.
func AddNewBranch(dir, path, branch string) error {
	_, err := Run(dir, "worktree", "add", "-b", branch, path)
	return err
}

// AddExisting creates a worktree at path checked out to an existing branch.
func AddExisting(dir, path, branch string) error {
	_, err := Run(dir, "worktree", "add", path, branch)
	return err
}

// RemoveWorktree removes the worktree at path. force allows removal of a dirty
// or locked worktree.
func RemoveWorktree(dir, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := Run(dir, args...)
	return err
}

// DeleteBranch deletes a local branch. force uses -D (delete even if unmerged).
func DeleteBranch(dir, branch string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := Run(dir, "branch", flag, branch)
	return err
}
