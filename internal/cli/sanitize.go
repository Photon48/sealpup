package cli

import "strings"

// sanitizeDirName turns a branch name into a safe single-segment directory
// name. Path separators and shell/filesystem-hostile characters collapse to
// '-', so "feat/foo" becomes "feat-foo". Resolution never relies on this
// mapping — git's worktree list is the source of truth — it is used only when
// choosing where a new worktree lives on disk.
func sanitizeDirName(branch string) string {
	var b strings.Builder
	for _, r := range branch {
		switch {
		case r <= 0x1f || r == 0x7f: // control chars
			b.WriteByte('-')
		case strings.ContainsRune(`/\:*?"<>| `, r):
			b.WriteByte('-')
		default:
			b.WriteRune(r)
		}
	}
	// Collapse runs of '-' and trim leading separators/dots that would create
	// hidden or oddly-named directories.
	out := b.String()
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	out = strings.TrimLeft(out, "-.")
	out = strings.TrimRight(out, "-")
	return out
}
