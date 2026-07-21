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
	// Resolve the newest release straight from the git origin — the module
	// proxy's @latest answer can lag a fresh tag by ~30 minutes. Fall back to
	// the proxy if the origin is unreachable.
	latest, lerr := update.LatestFromOrigin(10 * time.Second)
	if lerr != nil {
		latest, lerr = update.FetchLatest(5 * time.Second)
	}
	if lerr == nil && update.IsRelease(current) && !update.Newer(latest, current) {
		update.MarkLatest(latest)
		e.prompter().Successf("already up to date (%s)", current)
		return nil
	}

	// Install the exact resolved version: explicit versions are fetched by the
	// proxy on demand (no @latest cache lag).
	target := update.Module + "@latest"
	if lerr == nil {
		target = update.Module + "@" + latest
	}
	e.prompter().Infof("installing %s ...", target)
	cmd := exec.Command(gobin, "install", target)
	cmd.Stdout = e.Stderr // keep stdout clean for the shell shim
	cmd.Stderr = e.Stderr
	if err := cmd.Run(); err != nil {
		return ui.Errorf("go install failed: %v", err)
	}

	if lerr == nil {
		update.MarkLatest(latest)
		e.prompter().Successf("updated to %s", latest)
	} else {
		e.prompter().Successf("updated to the latest version")
	}
	return nil
}
