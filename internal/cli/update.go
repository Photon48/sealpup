package cli

import (
	"os/exec"
	"time"

	"github.com/Photon48/sealpup/internal/ui"
	"github.com/Photon48/sealpup/internal/update"
)

// cmdUpdate upgrades sealpup in place by reinstalling the module at @latest —
// the exact path install.sh takes, so the two can never drift.
func cmdUpdate(e Env, args []string) error {
	_, positional := partitionFlags(args)
	if len(positional) != 0 {
		return usageErr("update", "update takes no arguments")
	}

	gobin, err := exec.LookPath("go")
	if err != nil {
		return ui.Hintf(
			"updating needs Go, the same requirement as installing",
			"install it from https://go.dev/dl/ — or re-run the curl installer from the README",
		)
	}

	current := update.Current(version)
	latest, ferr := update.FetchLatest(5 * time.Second)
	if ferr == nil && update.IsRelease(current) && !update.Newer(latest, current) {
		update.MarkLatest(latest)
		e.prompter().Successf("already up to date (%s)", current)
		return nil
	}

	e.prompter().Infof("installing %s@latest ...", update.Module)
	cmd := exec.Command(gobin, "install", update.Module+"@latest")
	cmd.Stdout = e.Stderr // keep stdout clean for the shell shim
	cmd.Stderr = e.Stderr
	if err := cmd.Run(); err != nil {
		return ui.Errorf("go install failed: %v", err)
	}

	if ferr == nil {
		update.MarkLatest(latest)
		e.prompter().Successf("updated to %s", latest)
	} else {
		e.prompter().Successf("updated to the latest version")
	}
	return nil
}
