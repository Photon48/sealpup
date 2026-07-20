// Package gittest builds throwaway, fully isolated git repositories for tests.
// It pins git's config to empty temp files so the developer's real ~/.gitconfig
// and system config can never leak in and change behavior.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Repo is a temporary git repository rooted at Dir. Worktrees created through
// sealpup land in the sibling "<Dir>-worktrees" directory, which lives under
// the same t.TempDir() and is cleaned up automatically.
type Repo struct {
	t   *testing.T
	Dir string
}

// New initializes an isolated repository with an initial empty commit on
// branch "main" and a local user identity. It calls t.Setenv, so tests using it
// cannot be run with t.Parallel — an acceptable trade for hermetic config.
func New(t *testing.T) *Repo {
	t.Helper()
	root := t.TempDir()

	// Isolate git from the host's global/system config. Pointing GIT_CONFIG_*
	// at non-existent paths yields empty config; NOSYSTEM belts-and-braces it.
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "no-such-global"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(root, "no-such-system"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	// Nest the repo one level down so its worktree container stays inside root.
	dir := filepath.Join(root, "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("gittest: mkdir repo: %v", err)
	}
	r := &Repo{t: t, Dir: dir}

	r.Git("init", "-b", "main")
	r.Git("config", "user.name", "sealpup test")
	r.Git("config", "user.email", "test@sealpup.invalid")
	r.Git("commit", "--allow-empty", "-m", "init")
	return r
}

// Git runs a git command in the repo and fails the test on error, returning
// trimmed stdout.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("gittest: git %v: %v\n%s", args, err, out)
	}
	return trimRightNewline(string(out))
}

// GitIn runs a git command in an arbitrary directory (e.g. a linked worktree).
func (r *Repo) GitIn(dir string, args ...string) string {
	r.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("gittest: git -C %s %v: %v\n%s", dir, args, err, out)
	}
	return trimRightNewline(string(out))
}

// Branch creates a new branch from HEAD without switching to it.
func (r *Repo) Branch(name string) {
	r.t.Helper()
	r.Git("branch", name)
}

// WriteFile writes a file (relative to a directory) and returns its full path.
func (r *Repo) WriteFile(dir, rel, content string) string {
	r.t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		r.t.Fatalf("gittest: mkdir for %s: %v", p, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		r.t.Fatalf("gittest: write %s: %v", p, err)
	}
	return p
}

// Container is the sibling directory sealpup uses for worktrees.
func (r *Repo) Container() string {
	return r.Dir + "-worktrees"
}

func trimRightNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
