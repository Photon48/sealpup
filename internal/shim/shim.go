// Package shim embeds the per-shell integration snippets that make
// `sealpup enter` actually change the caller's directory. See the .zsh/.bash/
// .fish files for the mechanism.
package shim

import _ "embed"

//go:embed sealpup.zsh
var zsh string

//go:embed sealpup.bash
var bash string

//go:embed sealpup.fish
var fish string

// Supported lists the shells sealpup can emit an integration for.
var Supported = []string{"zsh", "bash", "fish"}

// For returns the integration script for the named shell and whether it is
// supported.
func For(shell string) (string, bool) {
	switch shell {
	case "zsh":
		return zsh, true
	case "bash":
		return bash, true
	case "fish":
		return fish, true
	default:
		return "", false
	}
}
