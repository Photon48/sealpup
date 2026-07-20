package git

import "strings"

// Worktree is one entry from `git worktree list --porcelain`. Branch has the
// refs/heads/ prefix stripped and is empty for detached or bare entries.
type Worktree struct {
	Path     string
	Head     string
	Branch   string
	Bare     bool
	Detached bool
	Locked   string // lock reason; set (possibly empty string) only when locked
	Prunable string // prunable reason; set when git considers it removable
	isLocked bool
}

// IsLocked reports whether git flagged this worktree as locked.
func (w Worktree) IsLocked() bool { return w.isLocked }

// Worktrees returns all worktrees known to the repository containing dir.
func Worktrees(dir string) ([]Worktree, error) {
	out, err := Run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(out), nil
}

// ParseWorktreeList parses the porcelain output of `git worktree list
// --porcelain`. Stanzas are separated by blank lines; each begins with a
// "worktree <path>" line followed by attribute lines. Kept as a pure function
// so it can be golden-tested without a real repository.
func ParseWorktreeList(out string) []Worktree {
	var result []Worktree
	var cur *Worktree
	flush := func() {
		if cur != nil {
			result = append(result, *cur)
			cur = nil
		}
	}

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			flush()
			continue
		}
		key, val, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: val}
		case "HEAD":
			if cur != nil {
				cur.Head = val
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(val, "refs/heads/")
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "locked":
			if cur != nil {
				cur.isLocked = true
				cur.Locked = val
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = val
			}
		}
	}
	flush()
	return result
}

// FindByBranch returns the worktree checked out to branch, or nil.
func FindByBranch(wts []Worktree, branch string) *Worktree {
	for i := range wts {
		if wts[i].Branch == branch {
			return &wts[i]
		}
	}
	return nil
}
