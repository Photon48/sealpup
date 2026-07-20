package cli

import (
	"fmt"
	"strings"

	"github.com/Photon48/sealpup/internal/shim"
	"github.com/Photon48/sealpup/internal/ui"
)

// cmdInit prints the shell integration snippet for the requested shell. Users
// wire it up once with, e.g., eval "$(sealpup init zsh)".
func cmdInit(e Env, args []string) error {
	var positional []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
		}
	}
	if len(positional) != 1 {
		return ui.Hintf(
			"init needs a shell name",
			"one of: "+strings.Join(shim.Supported, ", ")+"   (e.g. sealpup init zsh)",
		)
	}

	shell := positional[0]
	src, ok := shim.For(shell)
	if !ok {
		return ui.Hintf(
			"unsupported shell "+quote(shell),
			"supported shells: "+strings.Join(shim.Supported, ", "),
		)
	}
	fmt.Fprint(e.Stdout, src)
	return nil
}
