//go:build darwin || linux

package agents

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// TestDetect_RealProcess spawns a real `sleep` with its cwd inside a temp
// worktree and, treating "sleep" as a known agent, asserts Detect finds it
// there and nowhere else. This exercises the actual ps/lsof (or /proc) path.
func TestDetect_RealProcess(t *testing.T) {
	t.Setenv("SEALPUP_AGENTS", "sleep")

	root := t.TempDir()
	wt := filepath.Join(root, "wt")
	sibling := filepath.Join(root, "sibling")
	for _, d := range []string{wt, sibling} {
		if err := exec.Command("mkdir", "-p", d).Run(); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command("sleep", "60")
	cmd.Dir = wt
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn sleep: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	got := Detect([]string{wt, sibling})

	found := false
	for _, a := range got[wt] {
		if a.PID == cmd.Process.Pid && a.Name == "sleep" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sleep pid %d detected in %s, got %+v", cmd.Process.Pid, wt, got)
	}
	// Our sleep must not appear under the sibling worktree.
	for _, a := range got[sibling] {
		if a.PID == cmd.Process.Pid {
			t.Errorf("sleep wrongly attributed to sibling worktree")
		}
	}
}
