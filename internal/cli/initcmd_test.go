package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_EmitsFunction(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		e, out, _ := testEnv(t.TempDir())
		if code := Run(e, []string{"init", shell}); code != 0 {
			t.Fatalf("init %s exit = %d, want 0", shell, code)
		}
		got := out.String()
		if !strings.Contains(got, "sealpup") || !strings.Contains(got, "cd") {
			t.Errorf("init %s output missing a sealpup function with cd:\n%s", shell, got)
		}
	}
}

func TestInit_UnknownShell(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	if code := Run(e, []string{"init", "powershell"}); code != 1 {
		t.Fatalf("init powershell exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "unsupported shell") {
		t.Errorf("expected unsupported-shell error, got %q", errb.String())
	}
}

func TestInit_MissingShell(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	if code := Run(e, []string{"init"}); code != 1 {
		t.Fatalf("init (no arg) exit = %d, want 1", code)
	}
	if !strings.Contains(errb.String(), "needs a shell name") {
		t.Errorf("expected missing-shell error, got %q", errb.String())
	}
}

// TestInit_ShimSyntaxValid runs each emitted shim through its shell's syntax
// checker, skipping shells that aren't installed.
func TestInit_ShimSyntaxValid(t *testing.T) {
	cases := []struct {
		shell   string
		bin     string
		checkFn func(path string) *exec.Cmd
	}{
		{"zsh", "zsh", func(p string) *exec.Cmd { return exec.Command("zsh", "-n", p) }},
		{"bash", "bash", func(p string) *exec.Cmd { return exec.Command("bash", "-n", p) }},
		{"fish", "fish", func(p string) *exec.Cmd { return exec.Command("fish", "--no-execute", p) }},
	}
	for _, tc := range cases {
		t.Run(tc.shell, func(t *testing.T) {
			if _, err := exec.LookPath(tc.bin); err != nil {
				t.Skipf("%s not installed", tc.bin)
			}
			e, out, _ := testEnv(t.TempDir())
			if code := Run(e, []string{"init", tc.shell}); code != 0 {
				t.Fatalf("init %s failed", tc.shell)
			}
			f := filepath.Join(t.TempDir(), "shim."+tc.shell)
			if err := os.WriteFile(f, out.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			if b, err := tc.checkFn(f).CombinedOutput(); err != nil {
				t.Errorf("%s syntax check failed: %v\n%s", tc.shell, err, b)
			}
		})
	}
}
