package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Photon48/sealpup/internal/git"
)

// cmdList prints every worktree in the repo with its branch, path, dirty
// status, and a marker for the one you're standing in. It is read-only, so it
// writes to stdout and stays pipeable (sealpup list | grep ...).
func cmdList(e Env, args []string) error {
	if len(args) > 0 {
		return usageErr("list", "list takes no arguments")
	}

	repo, err := e.repo()
	if err != nil {
		return err
	}
	_ = git.Prune(e.Dir) // opportunistic self-heal; ignore failures

	wts, err := git.Worktrees(e.Dir)
	if err != nil {
		return err
	}

	// One process scan for the whole table: which agents live in which worktree.
	var paths []string
	for _, wt := range wts {
		if !wt.Bare {
			paths = append(paths, wt.Path)
		}
	}
	byPath := detectAgents(paths)

	tw := tabwriter.NewWriter(e.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "BRANCH\tPATH\tSTATUS\tAGENT")
	for _, wt := range wts {
		branch := branchLabel(wt)
		path := prettyPath(wt.Path)
		status := statusLabel(e, wt)
		agent := agentLabel(byPath[wt.Path])
		if e.isCurrent(wt.Path) {
			agent += "\t← you are here"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", branch, path, status, agent)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_ = repo
	return nil
}

// branchLabel renders the branch column, naming the special detached/bare cases.
func branchLabel(wt git.Worktree) string {
	switch {
	case wt.Bare:
		return "(bare)"
	case wt.Branch != "":
		return wt.Branch
	case wt.Detached:
		return "(detached)"
	default:
		return "(unknown)"
	}
}

// statusLabel returns "clean", "N files", or "missing" (dir gone from disk).
func statusLabel(e Env, wt git.Worktree) string {
	if wt.Bare {
		return "-"
	}
	if wt.Prunable != "" {
		return "missing"
	}
	if _, err := os.Stat(wt.Path); err != nil {
		return "missing"
	}
	lines, err := git.Status(wt.Path)
	if err != nil {
		return "?"
	}
	if len(lines) == 0 {
		return "clean"
	}
	if len(lines) == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", len(lines))
}
