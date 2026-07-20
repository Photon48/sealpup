package git

import (
	"reflect"
	"testing"
)

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Worktree
	}{
		{
			name: "empty",
			in:   "",
			want: nil,
		},
		{
			name: "single main",
			in:   "worktree /home/u/repo\nHEAD abc123\nbranch refs/heads/main\n",
			want: []Worktree{{Path: "/home/u/repo", Head: "abc123", Branch: "main"}},
		},
		{
			name: "slash branch and second worktree",
			in: "worktree /home/u/repo\nHEAD abc\nbranch refs/heads/main\n\n" +
				"worktree /home/u/repo-worktrees/feat-foo\nHEAD def\nbranch refs/heads/feat/foo\n",
			want: []Worktree{
				{Path: "/home/u/repo", Head: "abc", Branch: "main"},
				{Path: "/home/u/repo-worktrees/feat-foo", Head: "def", Branch: "feat/foo"},
			},
		},
		{
			name: "detached",
			in:   "worktree /home/u/wt\nHEAD abc\ndetached\n",
			want: []Worktree{{Path: "/home/u/wt", Head: "abc", Detached: true}},
		},
		{
			name: "bare",
			in:   "worktree /home/u/repo.git\nbare\n",
			want: []Worktree{{Path: "/home/u/repo.git", Bare: true}},
		},
		{
			name: "locked with reason and prunable",
			in:   "worktree /home/u/wt\nHEAD abc\nbranch refs/heads/x\nlocked on removable media\nprunable gitdir file points to non-existent location\n",
			want: []Worktree{{
				Path: "/home/u/wt", Head: "abc", Branch: "x",
				isLocked: true, Locked: "on removable media",
				Prunable: "gitdir file points to non-existent location",
			}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseWorktreeList(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseWorktreeList(%q)\n got %#v\nwant %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFindByBranch(t *testing.T) {
	wts := []Worktree{{Branch: "main"}, {Branch: "feat/foo"}}
	if got := FindByBranch(wts, "feat/foo"); got == nil || got.Branch != "feat/foo" {
		t.Fatalf("FindByBranch: got %v", got)
	}
	if got := FindByBranch(wts, "nope"); got != nil {
		t.Fatalf("FindByBranch(nope): expected nil, got %v", got)
	}
}
