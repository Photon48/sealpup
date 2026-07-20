package cli

import (
	"strconv"
	"strings"

	"github.com/Photon48/sealpup/internal/ui"
)

// singleBranchArg extracts exactly one branch-name argument for a subcommand,
// rejecting missing, extra, or flag-shaped inputs with a helpful error.
func singleBranchArg(cmd string, args []string) (string, error) {
	// Filter positional args from flags so the count check is meaningful.
	var positional []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") && a != "-" {
			continue
		}
		positional = append(positional, a)
	}
	switch len(positional) {
	case 0:
		return "", ui.Hintf(
			cmd+" needs a branch name",
			"usage: sealpup "+cmd+" <branch>",
		)
	case 1:
		branch := positional[0]
		if strings.HasPrefix(branch, "-") {
			return "", ui.Errorf("invalid branch name %s", quote(branch))
		}
		return branch, nil
	default:
		return "", ui.Hintf(
			cmd+" takes a single branch name, got "+strconv.Itoa(len(positional)),
			"usage: sealpup "+cmd+" <branch>",
		)
	}
}

// quote renders a value in double quotes for messages.
func quote(s string) string { return strconv.Quote(s) }
