// Package config loads a repo's optional .sealpup.toml. It parses a deliberately
// tiny TOML subset — [hooks] with copy = ["…"] and post_create = "…" — and
// errors loudly (with line number and what IS supported) on anything else, so a
// snippet copied from a README that uses unsupported syntax fails clearly rather
// than being silently ignored.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Photon48/sealpup/internal/ui"
)

// Config is the recognized content of .sealpup.toml.
type Config struct {
	Copy       []string // files copied from the main worktree into a new one
	PostCreate string   // shell command run in a new worktree after creation
}

// IsEmpty reports whether the config asks for no work.
func (c *Config) IsEmpty() bool {
	return c == nil || (len(c.Copy) == 0 && c.PostCreate == "")
}

const supportedHint = `sealpup supports only: [hooks] with copy = ["file", ...] and post_create = "command"`

// Load reads and parses <mainDir>/.sealpup.toml. A missing file is not an error
// (returns nil, nil).
func Load(mainDir string) (*Config, error) {
	path := filepath.Join(mainDir, ".sealpup.toml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, ui.Errorf("could not read .sealpup.toml: %v", err)
	}
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Parse parses the TOML subset. Pure (no filesystem), so it's golden-testable.
func Parse(src []byte) (*Config, error) {
	cfg := &Config{}
	lines := strings.Split(string(src), "\n")
	section := ""

	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimSpace(stripComment(lines[i]))
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return nil, errAt(lineNo, "malformed section header")
			}
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name != "hooks" {
				return nil, errAt(lineNo, "unknown section "+strconv.Quote(name))
			}
			section = name
			continue
		}

		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, errAt(lineNo, "expected 'key = value'")
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if section != "hooks" {
			return nil, errAt(lineNo, "keys must be inside a [hooks] section")
		}

		switch key {
		case "copy":
			arr, last, err := parseArray(lines, i, val)
			if err != nil {
				return nil, err
			}
			cfg.Copy = arr
			i = last
		case "post_create":
			s, err := parseString(val, lineNo)
			if err != nil {
				return nil, err
			}
			cfg.PostCreate = s
		default:
			return nil, errAt(lineNo, "unrecognized key "+strconv.Quote(key))
		}
	}
	return cfg, nil
}

// validate enforces that copy paths stay inside the worktree (no absolute paths,
// no ".." escapes).
func (c *Config) validate() error {
	for _, p := range c.Copy {
		if p == "" {
			return ui.Hintf(".sealpup.toml: copy contains an empty path", supportedHint)
		}
		if !filepath.IsLocal(p) {
			return ui.Hintf(
				".sealpup.toml: copy path "+strconv.Quote(p)+" must be relative and stay inside the repo",
				"use paths like \".env\" or \"config/local.json\" — no absolute paths or \"..\"",
			)
		}
	}
	return nil
}

// parseArray reads a string array beginning on lines[start] (val is the text
// after '='), possibly spanning multiple lines. Returns the items and the index
// of the last line consumed.
func parseArray(lines []string, start int, val string) ([]string, int, error) {
	if !strings.HasPrefix(val, "[") {
		return nil, 0, errAt(start+1, "copy must be an array, e.g. [\"file\", ...]")
	}
	var content strings.Builder
	content.WriteString(val[1:]) // drop the opening '['
	idx := start
	for {
		if close := findUnquoted(content.String(), ']'); close >= 0 {
			items, err := splitArrayItems(content.String()[:close], start+1)
			return items, idx, err
		}
		idx++
		if idx >= len(lines) {
			return nil, 0, errAt(start+1, "unterminated array (missing ']')")
		}
		content.WriteString(" ")
		content.WriteString(stripComment(lines[idx]))
	}
}

// splitArrayItems splits comma-separated string literals (top level), tolerating
// a trailing comma.
func splitArrayItems(s string, lineNo int) ([]string, error) {
	var items []string
	for _, part := range splitUnquoted(s, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // trailing comma or empty gap
		}
		v, err := parseString(part, lineNo)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, nil
}

// parseString parses a single basic ("…") or literal ('…') string.
func parseString(s string, lineNo int) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return "", errAt(lineNo, "expected a quoted string")
	}
	switch s[0] {
	case '"':
		if s[len(s)-1] != '"' {
			return "", errAt(lineNo, "unterminated string")
		}
		return unescape(s[1:len(s)-1], lineNo)
	case '\'':
		if s[len(s)-1] != '\'' {
			return "", errAt(lineNo, "unterminated string")
		}
		return s[1 : len(s)-1], nil
	default:
		return "", errAt(lineNo, "value must be a quoted string")
	}
}

// unescape resolves the supported escapes in a basic string.
func unescape(s string, lineNo int) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i >= len(s) {
			return "", errAt(lineNo, "trailing backslash in string")
		}
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case '"':
			b.WriteByte('"')
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		default:
			return "", errAt(lineNo, "unsupported escape \\"+string(s[i]))
		}
	}
	return b.String(), nil
}

// stripComment removes a trailing/whole-line '#' comment, ignoring '#' inside
// quoted strings.
func stripComment(line string) string {
	if i := findUnquoted(line, '#'); i >= 0 {
		return line[:i]
	}
	return line
}

// findUnquoted returns the index of the first ch outside any quoted string, or
// -1. Backslash escapes apply only inside double quotes.
func findUnquoted(s string, ch byte) int {
	var q byte // current quote char, 0 when outside a string
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q == '"' && c == '\\':
			i++ // skip escaped char
		case q != 0 && c == q:
			q = 0
		case q == 0 && (c == '"' || c == '\''):
			q = c
		case q == 0 && c == ch:
			return i
		}
	}
	return -1
}

// splitUnquoted splits s on sep chars that lie outside quoted strings.
func splitUnquoted(s string, sep byte) []string {
	var parts []string
	var q byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q == '"' && c == '\\':
			i++
		case q != 0 && c == q:
			q = 0
		case q == 0 && (c == '"' || c == '\''):
			q = c
		case q == 0 && c == sep:
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

func errAt(lineNo int, msg string) error {
	return ui.Hintf(fmt.Sprintf(".sealpup.toml line %d: %s", lineNo, msg), supportedHint)
}
