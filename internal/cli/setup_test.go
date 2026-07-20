package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetup_WritesManagedBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	// Seed an existing rc to prove we append, not clobber.
	rc := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rc, []byte("export FOO=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, _, errb := testEnv(home)
	if code := Run(e, []string{"setup"}); code != 0 {
		t.Fatalf("setup exit = %d, want 0\n%s", code, errb.String())
	}

	got, _ := os.ReadFile(rc)
	s := string(got)
	if !strings.Contains(s, "export FOO=1") {
		t.Error("setup clobbered existing rc content")
	}
	if !strings.Contains(s, markerStart) || !strings.Contains(s, markerEnd) {
		t.Errorf("managed block markers missing:\n%s", s)
	}
	if !strings.Contains(s, `eval "$(sealpup init zsh)"`) {
		t.Errorf("shim eval line missing:\n%s", s)
	}
}

func TestSetup_Idempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/usr/bin/bash")

	e1, _, _ := testEnv(home)
	Run(e1, []string{"setup"})
	e2, _, errb := testEnv(home)
	Run(e2, []string{"setup"})

	got, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	if n := strings.Count(string(got), markerStart); n != 1 {
		t.Fatalf("expected exactly 1 managed block after two runs, got %d", n)
	}
	if !strings.Contains(errb.String(), "already set up") {
		t.Errorf("second run should report already set up, got %q", errb.String())
	}
}

func TestSetup_Fish(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	e, _, _ := testEnv(home)
	if code := Run(e, []string{"setup", "fish"}); code != 0 {
		t.Fatalf("setup fish exit = %d, want 0", code)
	}
	got, err := os.ReadFile(filepath.Join(home, ".config", "fish", "config.fish"))
	if err != nil {
		t.Fatalf("fish config not written: %v", err)
	}
	if !strings.Contains(string(got), "sealpup init fish | source") {
		t.Errorf("fish shim line missing:\n%s", got)
	}
}

func TestSetup_ForceReplacesBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	e1, _, _ := testEnv(home)
	Run(e1, []string{"setup"})
	e2, _, _ := testEnv(home)
	Run(e2, []string{"setup", "--force"})

	got, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if n := strings.Count(string(got), markerStart); n != 1 {
		t.Fatalf("force should keep exactly 1 block, got %d", n)
	}
}
