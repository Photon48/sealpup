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

func TestSetup_NoArgWiresZshAndBash(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	e, _, errb := testEnv(home)
	if code := Run(e, []string{"setup"}); code != 0 {
		t.Fatalf("setup exit = %d, want 0\n%s", code, errb.String())
	}

	// Both shells get wired: $SHELL only says the login shell, not what the
	// user actually types into (macOS ships bash alongside zsh).
	zrc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if !strings.Contains(string(zrc), `eval "$(sealpup init zsh)"`) {
		t.Errorf(".zshrc not wired:\n%s", zrc)
	}
	brc, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	if !strings.Contains(string(brc), `eval "$(sealpup init bash)"`) {
		t.Errorf(".bashrc not wired:\n%s", brc)
	}
	if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err != nil {
		t.Error("bash login link missing")
	}
	// fish is only wired when its config already exists.
	if _, err := os.Stat(filepath.Join(home, ".config", "fish", "config.fish")); err == nil {
		t.Error("fish should not be wired without an existing config")
	}
}

func TestSetup_ExplicitShellWiresOnlyThat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	e, _, _ := testEnv(home)
	if code := Run(e, []string{"setup", "zsh"}); code != 0 {
		t.Fatal("setup zsh failed")
	}
	if _, err := os.Stat(filepath.Join(home, ".bashrc")); err == nil {
		t.Error("explicit `setup zsh` should not touch .bashrc")
	}
}

func TestSetup_BashLinksLoginRC(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	e, _, _ := testEnv(home)
	if code := Run(e, []string{"setup"}); code != 0 {
		t.Fatalf("setup exit = %d, want 0", code)
	}

	// The real block lives in ~/.bashrc.
	rc, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	if !strings.Contains(string(rc), `eval "$(sealpup init bash)"`) {
		t.Errorf("shim eval line missing from .bashrc:\n%s", rc)
	}

	// With no pre-existing login file, we create ~/.bash_profile that sources it,
	// so macOS login shells (which never read ~/.bashrc) still load the block.
	prof, err := os.ReadFile(filepath.Join(home, ".bash_profile"))
	if err != nil {
		t.Fatalf(".bash_profile not created: %v", err)
	}
	if !strings.Contains(string(prof), ".bashrc") {
		t.Errorf(".bash_profile does not source .bashrc:\n%s", prof)
	}
	if !strings.Contains(string(prof), markerStart) {
		t.Errorf(".bash_profile missing managed markers:\n%s", prof)
	}
}

func TestSetup_BashLeavesExistingSourceAlone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	// A login file that already sources ~/.bashrc — the common distro default.
	prof := filepath.Join(home, ".bash_profile")
	if err := os.WriteFile(prof, []byte("[ -f ~/.bashrc ] && . ~/.bashrc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, _, _ := testEnv(home)
	Run(e, []string{"setup"})

	got, _ := os.ReadFile(prof)
	if strings.Contains(string(got), markerStart) {
		t.Errorf("should not add a block when .bashrc is already sourced:\n%s", got)
	}
}

func TestSetup_BashAppendsToExistingProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")
	// An existing ~/.profile that does NOT source ~/.bashrc must be appended to,
	// not shadowed by a fresh ~/.bash_profile.
	if err := os.WriteFile(filepath.Join(home, ".profile"), []byte("export EDITOR=vim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	e, _, _ := testEnv(home)
	Run(e, []string{"setup"})

	if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
		t.Error("should append to existing .profile, not create .bash_profile that shadows it")
	}
	prof, _ := os.ReadFile(filepath.Join(home, ".profile"))
	if !strings.Contains(string(prof), "export EDITOR=vim") {
		t.Errorf("clobbered existing .profile content:\n%s", prof)
	}
	if !strings.Contains(string(prof), ".bashrc") {
		t.Errorf(".profile was not linked to .bashrc:\n%s", prof)
	}
}

func TestSetup_BashLoginLinkIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/bash")

	e1, _, _ := testEnv(home)
	Run(e1, []string{"setup"})
	e2, _, _ := testEnv(home)
	Run(e2, []string{"setup"})

	prof, _ := os.ReadFile(filepath.Join(home, ".bash_profile"))
	if n := strings.Count(string(prof), markerStart); n != 1 {
		t.Fatalf("expected exactly 1 managed block in .bash_profile, got %d", n)
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
