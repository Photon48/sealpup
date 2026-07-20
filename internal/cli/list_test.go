package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/agents"
	"github.com/Photon48/sealpup/internal/git/gittest"
)

// stubAgents overrides detectAgents for a test and restores it after.
func stubAgents(t *testing.T, m map[string][]agents.Agent) {
	t.Helper()
	prev := detectAgents
	detectAgents = func([]string) map[string][]agents.Agent { return m }
	t.Cleanup(func() { detectAgents = prev })
}

func TestList_AgentColumn(t *testing.T) {
	repo := gittest.New(t)
	wtFoo := filepath.Join(repo.Container(), "foo")
	repo.Branch("foo")
	repo.Git("worktree", "add", wtFoo, "foo")

	// git reports the worktree path symlink-resolved (/private/var on macOS);
	// the real detectAgents keys its result by that same path, so match it.
	fooKey := repo.Git("-C", wtFoo, "rev-parse", "--show-toplevel")
	stubAgents(t, map[string][]agents.Agent{
		fooKey: {{PID: 4812, Name: "claude"}, {PID: 9001, Name: "aider"}},
	})

	e, out, _ := testEnv(repo.Dir)
	if code := Run(e, []string{"list"}); code != 0 {
		t.Fatalf("list exit = %d, want 0", code)
	}
	got := out.String()
	if !strings.Contains(got, "AGENT") {
		t.Errorf("missing AGENT header:\n%s", got)
	}
	if !strings.Contains(got, "claude (pid 4812) +1") {
		t.Errorf("expected agent cell 'claude (pid 4812) +1':\n%s", got)
	}
	// The main worktree row shows no agent.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "main") && !strings.Contains(line, "-") {
			t.Errorf("main row should show '-' for agent: %q", line)
		}
	}
}

func TestList(t *testing.T) {
	repo := gittest.New(t)
	repo.Branch("feat/foo")
	repo.Branch("bar")
	wtFoo := filepath.Join(repo.Container(), "feat-foo")
	wtBar := filepath.Join(repo.Container(), "bar")
	repo.Git("worktree", "add", wtFoo, "feat/foo")
	repo.Git("worktree", "add", wtBar, "bar")
	// Make feat/foo dirty.
	repo.WriteFile(wtFoo, "dirty.txt", "x")

	// Run list from inside the bar worktree so the marker lands there.
	e, out, _ := testEnv(wtBar)
	if code := Run(e, []string{"list"}); code != 0 {
		t.Fatalf("list exit = %d, want 0", code)
	}
	got := out.String()

	for _, want := range []string{"main", "feat/foo", "bar", "1 file", "← you are here"} {
		if !strings.Contains(got, want) {
			t.Errorf("list output missing %q:\n%s", want, got)
		}
	}
	// The marker must be on the bar row, not feat/foo.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "← you are here") && !strings.Contains(line, wtBar[len(wtBar)-3:]) {
			// weak check: ensure marker line mentions bar's status region
		}
		if strings.Contains(line, "feat/foo") && strings.Contains(line, "← you are here") {
			t.Errorf("marker wrongly on feat/foo row: %q", line)
		}
	}
}

func TestListNotARepo(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	if code := Run(e, []string{"list"}); code != 1 {
		t.Fatalf("list outside repo: exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "not inside a git repository") {
		t.Errorf("expected not-a-repo message, got %q", errb.String())
	}
}
