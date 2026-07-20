package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Photon48/sealpup/internal/ui"
)

func TestParse_ReadmeExample(t *testing.T) {
	src := `
# my repo's sealpup config
[hooks]
copy = [".env", ".env.local"]   # local secrets
post_create = "pnpm install"
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(cfg.Copy, []string{".env", ".env.local"}) {
		t.Errorf("Copy = %v", cfg.Copy)
	}
	if cfg.PostCreate != "pnpm install" {
		t.Errorf("PostCreate = %q", cfg.PostCreate)
	}
}

func TestParse_MultilineArrayTrailingComma(t *testing.T) {
	src := `[hooks]
copy = [
  ".env",
  'literal.txt',   # a comment
]
`
	cfg, err := Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !reflect.DeepEqual(cfg.Copy, []string{".env", "literal.txt"}) {
		t.Errorf("Copy = %v", cfg.Copy)
	}
}

func TestParse_EscapesAndHashInString(t *testing.T) {
	cfg, err := Parse([]byte("[hooks]\npost_create = \"echo \\\"hi # there\\\"\"\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cfg.PostCreate != `echo "hi # there"` {
		t.Errorf("PostCreate = %q", cfg.PostCreate)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		line string // substring expected in the error
	}{
		{"unknown section", "[build]\nx = 1\n", "line 1"},
		{"unknown key", "[hooks]\ninstall = \"x\"\n", "line 2"},
		{"key outside section", "copy = [\".env\"]\n", "line 1"},
		{"bare value", "[hooks]\npost_create = pnpm install\n", "line 2"},
		{"copy not array", "[hooks]\ncopy = \".env\"\n", "line 2"},
		{"unterminated array", "[hooks]\ncopy = [\".env\",\n", "line 2"},
		{"bad escape", "[hooks]\npost_create = \"a\\qb\"\n", "line 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.src))
			if err == nil {
				t.Fatalf("expected error for %q", tc.src)
			}
			var ue *ui.UserError
			if !isUserError(err, &ue) {
				t.Fatalf("expected *ui.UserError, got %T", err)
			}
			if !strings.Contains(ue.Msg, tc.line) {
				t.Errorf("error %q missing %q", ue.Msg, tc.line)
			}
			if !strings.Contains(ue.Hint, "supports only") {
				t.Errorf("error hint %q missing 'supports only'", ue.Hint)
			}
		})
	}
}

func TestValidate_RejectsEscapingPaths(t *testing.T) {
	for _, bad := range []string{"/etc/passwd", "../secrets"} {
		cfg := &Config{Copy: []string{bad}}
		if err := cfg.validate(); err == nil {
			t.Errorf("expected validate to reject %q", bad)
		}
	}
	ok := &Config{Copy: []string{".env", "config/local.json"}}
	if err := ok.validate(); err != nil {
		t.Errorf("valid paths rejected: %v", err)
	}
}

// isUserError is a tiny errors.As wrapper kept local to avoid importing errors
// in the test just for this.
func isUserError(err error, target **ui.UserError) bool {
	ue, ok := err.(*ui.UserError)
	if ok {
		*target = ue
	}
	return ok
}
