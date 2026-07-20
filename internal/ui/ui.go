// Package ui holds the human-facing layer of sealpup: typed errors that carry
// a fix-it hint, and prompts that route to stderr so they survive the shell
// shim's stdout capture (see internal/shim).
package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// UserError is an expected, actionable failure — a footgun sealpup caught. Msg
// states what went wrong; Hint (optional) tells the user how to fix it. These
// are printed cleanly to stderr, never as a Go stack-trace-style dump.
type UserError struct {
	Msg  string
	Hint string
}

func (e *UserError) Error() string { return e.Msg }

// Errorf builds a UserError with no hint.
func Errorf(format string, a ...any) *UserError {
	return &UserError{Msg: fmt.Sprintf(format, a...)}
}

// WithHint returns a copy of e carrying the given hint.
func (e *UserError) WithHint(format string, a ...any) *UserError {
	return &UserError{Msg: e.Msg, Hint: fmt.Sprintf(format, a...)}
}

// Hintf builds a UserError with both a message and a hint.
func Hintf(msg, hint string) *UserError {
	return &UserError{Msg: msg, Hint: hint}
}

// Prompter carries the streams and interactivity state a command needs to talk
// to the user without reaching for the real os.Std* handles. All prose goes to
// Err (stderr) so that stdout stays reserved for machine output on new/enter.
type Prompter struct {
	In            io.Reader
	Err           io.Writer
	IsInteractive bool
	Color         bool
}

// Confirm asks a yes/no question on stderr and reads the answer from stdin.
// defaultYes controls both the [Y/n] vs [y/N] rendering and the empty-line
// answer. When the session is non-interactive it does not block: it returns
// notInteractive=true so the caller can fail fast with an explicit-command hint
// instead of hanging forever waiting on a closed stdin.
func (p *Prompter) Confirm(question string, defaultYes bool) (answer, notInteractive bool) {
	if !p.IsInteractive {
		return false, true
	}
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	fmt.Fprintf(p.Err, "%s %s ", question, suffix)

	reader := bufio.NewReader(p.In)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		// EOF with nothing typed: treat as the default rather than looping.
		return defaultYes, false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "":
		return defaultYes, false
	case "y", "yes":
		return true, false
	default:
		return false, false
	}
}

// Infof writes a status/progress line to stderr.
func (p *Prompter) Infof(format string, a ...any) {
	fmt.Fprintf(p.Err, format+"\n", a...)
}
