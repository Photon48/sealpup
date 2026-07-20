package cli

import (
	"bytes"
	"strings"
	"testing"
)

// testEnv builds an Env backed by buffers, defaulting to an interactive session
// that answers "y" to any prompt.
func testEnv(dir string) (Env, *bytes.Buffer, *bytes.Buffer) {
	return envWith(dir, "y\n")
}

// envWith builds an interactive test Env whose prompts are answered by stdin.
func envWith(dir, stdin string) (Env, *bytes.Buffer, *bytes.Buffer) {
	var out, errb bytes.Buffer
	e := Env{
		Dir:           dir,
		Stdout:        &out,
		Stderr:        &errb,
		Stdin:         strings.NewReader(stdin),
		IsInteractive: true,
		Color:         false,
		ShimActive:    true,
	}
	return e, &out, &errb
}

func TestRun_NoArgs(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	if code := Run(e, nil); code != 2 {
		t.Fatalf("no args: got exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Fatalf("no args: expected usage text on stderr, got %q", errb.String())
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	e, _, errb := testEnv(t.TempDir())
	code := Run(e, []string{"frobnicate"})
	if code != 2 {
		t.Fatalf("unknown command: got exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Fatalf("unknown command: stderr %q missing 'unknown command'", errb.String())
	}
	if !strings.Contains(errb.String(), "hint:") {
		t.Fatalf("unknown command: stderr %q missing a hint", errb.String())
	}
}

func TestRun_Version(t *testing.T) {
	e, out, _ := testEnv(t.TempDir())
	if code := Run(e, []string{"--version"}); code != 0 {
		t.Fatalf("version: got exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "sealpup ") {
		t.Fatalf("version: stdout %q missing version", out.String())
	}
}

func TestRun_Help(t *testing.T) {
	e, out, _ := testEnv(t.TempDir())
	if code := Run(e, []string{"help"}); code != 0 {
		t.Fatalf("help: got exit %d, want 0", code)
	}
	if !strings.Contains(out.String(), "sealpup new") {
		t.Fatalf("help: stdout %q missing command list", out.String())
	}
}
