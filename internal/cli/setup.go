package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Photon48/sealpup/internal/shim"
	"github.com/Photon48/sealpup/internal/ui"
)

// Markers delimit the block sealpup manages inside a user's shell rc file, so
// setup is idempotent and the block is easy to find and remove.
const (
	markerStart = "# >>> sealpup >>>"
	markerEnd   = "# <<< sealpup <<<"
)

// cmdSetup wires sealpup into the user's shell in one step: it puts the binary's
// directory on PATH and enables the auto-cd shim, writing a single managed block
// to the right rc file. Running it again is a no-op unless --force.
func cmdSetup(e Env, args []string) error {
	flags, positional := partitionFlags(args)
	force := false
	for _, f := range flags {
		if f == "--force" || f == "-f" {
			force = true
		}
	}

	shell, err := resolveShell(positional)
	if err != nil {
		return err
	}
	if _, ok := shim.For(shell); !ok {
		return ui.Hintf(
			"can't set up unsupported shell "+quote(shell),
			"supported shells: "+strings.Join(shim.Supported, ", "),
		)
	}

	rc, err := rcPath(shell)
	if err != nil {
		return err
	}
	binDir := executableDir()

	existing, _ := os.ReadFile(rc)
	if strings.Contains(string(existing), markerStart) && !force {
		e.prompter().Infof("sealpup is already set up in %s", prettyPath(rc))
		e.prompter().Infof("restart your shell (or run %s) to pick up changes", reloadHint(shell, rc))
		return nil
	}

	block := setupBlock(shell, binDir)
	if err := writeSetupBlock(rc, string(existing), block, force); err != nil {
		return ui.Errorf("could not update %s: %v", prettyPath(rc), err)
	}

	e.prompter().Successf("wired sealpup into %s", prettyPath(rc))
	e.prompter().Infof("restart your shell, or run:  %s", reloadHint(shell, rc))
	return nil
}

// resolveShell picks the shell from an explicit arg, else the $SHELL basename.
func resolveShell(positional []string) (string, error) {
	if len(positional) > 1 {
		return "", ui.Errorf("setup takes at most one shell name")
	}
	if len(positional) == 1 {
		return positional[0], nil
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return filepath.Base(sh), nil
	}
	return "", ui.Hintf(
		"couldn't detect your shell",
		"pass it explicitly:  sealpup setup zsh",
	)
}

// rcPath returns the shell's startup file, creating parent dirs for fish.
func rcPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ui.Errorf("could not find your home directory: %v", err)
	}
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "fish":
		dir := filepath.Join(home, ".config", "fish")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		return filepath.Join(dir, "config.fish"), nil
	default:
		return "", ui.Errorf("unsupported shell %s", quote(shell))
	}
}

// setupBlock builds the managed rc block for a shell: ensure the binary dir is
// on PATH, then load the shim.
func setupBlock(shell, binDir string) string {
	var b strings.Builder
	b.WriteString(markerStart + "  (managed by `sealpup setup` — delete this block to uninstall)\n")
	switch shell {
	case "fish":
		if binDir != "" {
			b.WriteString("fish_add_path " + binDir + "\n")
		}
		b.WriteString("sealpup init fish | source\n")
	default: // zsh, bash
		if binDir != "" {
			// Prepend to PATH only if it's not already present.
			b.WriteString("case \":$PATH:\" in *\":" + binDir + ":\"*) ;; *) export PATH=\"" + binDir + ":$PATH\" ;; esac\n")
		}
		b.WriteString("eval \"$(sealpup init " + shell + ")\"\n")
	}
	b.WriteString(markerEnd + "\n")
	return b.String()
}

// writeSetupBlock appends the block, or replaces an existing managed block when
// force is set.
func writeSetupBlock(rc, existing, block string, force bool) error {
	var out string
	if force && strings.Contains(existing, markerStart) {
		out = replaceManagedBlock(existing, block)
	} else {
		sep := ""
		if existing != "" && !strings.HasSuffix(existing, "\n") {
			sep = "\n"
		}
		lead := "\n"
		if existing == "" {
			lead = ""
		}
		out = existing + sep + lead + block
	}
	return os.WriteFile(rc, []byte(out), 0o644)
}

// replaceManagedBlock swaps the content between markers (inclusive) for block.
func replaceManagedBlock(content, block string) string {
	start := strings.Index(content, markerStart)
	if start == -1 {
		return content + "\n" + block
	}
	endIdx := strings.Index(content[start:], markerEnd)
	if endIdx == -1 {
		return content[:start] + block
	}
	end := start + endIdx + len(markerEnd)
	// Consume a trailing newline after the end marker if present.
	if end < len(content) && content[end] == '\n' {
		end++
	}
	trimmed := strings.TrimRight(block, "\n")
	return content[:start] + trimmed + "\n" + content[end:]
}

// executableDir returns the directory of the running sealpup binary, or "".
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe)
}

// reloadHint gives the command to reload the shell config now.
func reloadHint(shell, rc string) string {
	switch shell {
	case "fish":
		return "source " + prettyPath(rc)
	default:
		return "source " + prettyPath(rc)
	}
}
