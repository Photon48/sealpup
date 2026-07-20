// Package cli wires sealpup's subcommands to a testable Env seam. Commands
// never touch os.Stdout/os.Getwd directly — everything flows through Env so a
// test can drive Run with buffers and a scripted stdin.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Photon48/sealpup/internal/ui"
)

// version is the sealpup release string, overridable at build time via
// -ldflags "-X github.com/Photon48/sealpup/internal/cli.version=...".
var version = "0.1.0-dev"

// Env is the injectable environment a command runs against. RealEnv builds one
// from the process; tests build one from buffers.
type Env struct {
	Dir           string    // working directory (cwd)
	Stdout        io.Writer // machine output only on new/enter (the resolved path)
	Stderr        io.Writer // all human-facing prose, prompts, and errors
	Stdin         io.Reader // prompt answers
	IsInteractive bool      // stderr AND stdin are terminals
	Color         bool      // emit ANSI color to stderr
	ShimActive    bool      // the shell shim set _SEALPUP_SHIM=1
}

// RealEnv constructs an Env from the real process state.
func RealEnv() Env {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	interactive := isTerminal(os.Stderr) && isTerminal(os.Stdin)
	return Env{
		Dir:           dir,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Stdin:         os.Stdin,
		IsInteractive: interactive,
		Color:         interactive && os.Getenv("NO_COLOR") == "",
		ShimActive:    os.Getenv("_SEALPUP_SHIM") == "1",
	}
}

// prompter derives the ui.Prompter for this Env.
func (e Env) prompter() *ui.Prompter {
	return &ui.Prompter{
		In:            e.Stdin,
		Err:           e.Stderr,
		IsInteractive: e.IsInteractive,
		Color:         e.Color,
	}
}

// command is the signature every subcommand implements.
type command func(e Env, args []string) error

// Run dispatches args[0] to a subcommand and returns a process exit code.
// A *ui.UserError is rendered cleanly (message + optional hint); any other
// error is rendered as an unexpected internal failure. Both exit non-zero.
func Run(e Env, args []string) int {
	if len(args) == 0 {
		printUsage(e.Stderr)
		return 2
	}

	name, rest := args[0], args[1:]

	switch name {
	case "-h", "--help", "help":
		printUsage(e.Stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintln(e.Stdout, "sealpup "+version)
		return 0
	}

	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(e.Stderr, "sealpup: unknown command %q\n", name)
		fmt.Fprintf(e.Stderr, "hint: run 'sealpup help' to see available commands\n")
		return 2
	}

	if err := cmd(e, rest); err != nil {
		var ue *ui.UserError
		if errors.As(err, &ue) {
			fmt.Fprintf(e.Stderr, "sealpup: %s\n", ue.Msg)
			if ue.Hint != "" {
				fmt.Fprintf(e.Stderr, "hint: %s\n", ue.Hint)
			}
			return 1
		}
		fmt.Fprintf(e.Stderr, "sealpup: %s\n", err)
		return 1
	}
	return 0
}

// commands is the subcommand registry, populated by each command's file.
var commands = map[string]command{
	"new":    cmdNew,
	"enter":  cmdEnter,
	"list":   cmdList,
	"delete": cmdDelete,
	"init":   cmdInit,
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, usage)
}

const usage = `sealpup — a light, idiot-proof git worktree manager

usage:
  sealpup new <branch>      create a worktree (and branch) and enter it
  sealpup enter <branch>    jump into an existing worktree
  sealpup list              show all worktrees and where you are
  sealpup delete <branch>   remove a worktree (and optionally its branch)
  sealpup init <shell>      print the shell integration (zsh|bash|fish)

setup (once):
  eval "$(sealpup init zsh)"   # add to ~/.zshrc

flags:
  -h, --help       show this help
  -v, --version    show version
`
