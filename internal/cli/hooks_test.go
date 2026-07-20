package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/git/gittest"
)

// commitConfig writes .sealpup.toml at the repo root and commits it.
func commitConfig(t *testing.T, repo *gittest.Repo, body string) {
	t.Helper()
	repo.WriteFile(repo.Dir, ".sealpup.toml", body)
	repo.Git("add", ".sealpup.toml")
	repo.Git("commit", "-m", "add sealpup config")
}

func TestHooks_CopiesFile(t *testing.T) {
	repo := gittest.New(t)
	// A gitignored local file that should be copied into new worktrees.
	repo.WriteFile(repo.Dir, ".env", "SECRET=1")
	commitConfig(t, repo, "[hooks]\ncopy = [\".env\"]\n")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"new", "feat"}); code != 0 {
		t.Fatalf("new exit = %d, want 0\n%s", code, errb.String())
	}
	dir := wantPathStdout(t, out.String())
	got, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil || string(got) != "SECRET=1" {
		t.Fatalf(".env not copied into worktree: %q err=%v", got, err)
	}
	if !strings.Contains(errb.String(), "copied .env") {
		t.Errorf("expected 'copied .env' note, got %q", errb.String())
	}
}

func TestHooks_MissingCopySourceNoted(t *testing.T) {
	repo := gittest.New(t)
	commitConfig(t, repo, "[hooks]\ncopy = [\".env.local\"]\n")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"new", "feat"}); code != 0 {
		t.Fatalf("new exit = %d, want 0 (missing source is not fatal)", code)
	}
	if wantPathStdout(t, out.String()) == "" {
		t.Error("expected path on stdout despite missing copy source")
	}
	if !strings.Contains(errb.String(), "skipped .env.local") {
		t.Errorf("expected skip note, got %q", errb.String())
	}
}

func TestHooks_PostCreateRunsAndPathStaysClean(t *testing.T) {
	repo := gittest.New(t)
	commitConfig(t, repo, "[hooks]\npost_create = \"touch made-by-hook\"\n")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"new", "feat"}); code != 0 {
		t.Fatalf("new exit = %d, want 0", code)
	}
	dir := wantPathStdout(t, out.String())
	// stdout must be EXACTLY the path line — hook output goes to stderr only.
	if out.String() != dir+"\n" {
		t.Fatalf("stdout must be only the path; got %q", out.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "made-by-hook")); err != nil {
		t.Errorf("post_create did not run: %v", err)
	}
	if !strings.Contains(errb.String(), "running post_create hook: touch made-by-hook") {
		t.Errorf("expected hook banner on stderr, got %q", errb.String())
	}
}

func TestHooks_PostCreateFailureKeepsWorktree(t *testing.T) {
	repo := gittest.New(t)
	commitConfig(t, repo, "[hooks]\npost_create = \"exit 3\"\n")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"new", "feat"}); code != 1 {
		t.Fatalf("failing hook exit = %d, want 1", code)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("no path should be emitted when the hook fails, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "exit 3") || !strings.Contains(errb.String(), "already exists") {
		t.Errorf("expected exit-3 failure with 'worktree already exists' hint, got %q", errb.String())
	}
	// The worktree must still be on disk.
	if !strings.Contains(repo.Git("worktree", "list", "--porcelain"), "feat") {
		t.Error("worktree should survive a failed post_create")
	}
}

func TestHooks_MalformedConfigAbortsBeforeCreation(t *testing.T) {
	repo := gittest.New(t)
	commitConfig(t, repo, "[hooks]\ncopy = \"not an array\"\n")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"new", "feat"}); code != 1 {
		t.Fatalf("malformed config exit = %d, want 1", code)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("no path on malformed config, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "line 2") {
		t.Errorf("expected line-numbered error, got %q", errb.String())
	}
	// No worktree should have been created.
	if strings.Contains(repo.Git("worktree", "list", "--porcelain"), "feat") {
		t.Error("worktree must NOT be created when config is malformed")
	}
	if _, err := os.Stat(repo.Container()); !os.IsNotExist(err) {
		t.Error("container dir should not exist after aborted creation")
	}
}

func TestHooks_EnterExistingRunsNoHooks(t *testing.T) {
	repo := gittest.New(t)
	// A post_create that would fail if it ran.
	commitConfig(t, repo, "[hooks]\npost_create = \"exit 9\"\n")
	// Pre-create the worktree so `enter` takes the existing-worktree path.
	wt := filepath.Join(repo.Container(), "feat")
	repo.Branch("feat")
	repo.Git("worktree", "add", wt, "feat")

	e, out, errb := testEnv(repo.Dir)
	if code := Run(e, []string{"enter", "feat"}); code != 0 {
		t.Fatalf("enter existing exit = %d, want 0 (hooks must not run)\n%s", code, errb.String())
	}
	if strings.Contains(errb.String(), "post_create") {
		t.Errorf("hooks should not run when entering an existing worktree: %q", errb.String())
	}
	_ = out
}
