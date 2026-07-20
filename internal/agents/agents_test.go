package agents

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func names(agents []Agent) []string {
	var out []string
	for _, a := range agents {
		out = append(out, a.Name)
	}
	sort.Strings(out)
	return out
}

func TestMatchProcs(t *testing.T) {
	known := knownNames() // defaults; SEALPUP_AGENTS unset
	procs := []proc{
		{pid: 19278, comm: "claude", args: "claude --dangerously-skip-permissions"},                   // real claude
		{pid: 500, comm: "AMPDeviceDiscoveryAgent", args: "/usr/libexec/AMPDeviceDiscoveryAgent"},     // must NOT match "amp"
		{pid: 601, comm: "node", args: "node /Users/x/.nvm/versions/node/v20/bin/gemini --model pro"}, // interpreter fallback
		{pid: 42, comm: "/bin/sleep", args: "/bin/sleep 60"},                                          // full-path, not an agent
		{pid: 77, comm: "aider", args: "aider"},
	}
	got := names(matchProcs(procs, known))
	want := []string{"aider", "claude", "gemini"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("matchProcs = %v, want %v", got, want)
	}
}

func TestMatchOne_ArgsWithSpacesInPath(t *testing.T) {
	// argv[0] contains a space; TrimPrefix must still isolate argv[1] exactly.
	p := proc{
		comm: "/Applications/My App/node",
		args: "/Applications/My App/node /opt/tools/codex serve",
	}
	name, ok := matchOne(p, knownNames())
	if !ok || name != "codex" {
		t.Fatalf("matchOne = (%q,%v), want (codex,true)", name, ok)
	}
}

func TestParsePSLines(t *testing.T) {
	comm := "19278 claude\n  601 node\n42 /bin/sleep\n"
	args := "19278 claude --dangerously-skip-permissions\n601 node /path/gemini\n42 /bin/sleep 60\n"
	procs := parsePSLines(comm, args)
	if len(procs) != 3 {
		t.Fatalf("got %d procs, want 3: %+v", len(procs), procs)
	}
	if procs[0].pid != 19278 || procs[0].comm != "claude" || procs[0].args != "claude --dangerously-skip-permissions" {
		t.Errorf("proc[0] = %+v", procs[0])
	}
	if procs[1].pid != 601 || procs[1].comm != "node" {
		t.Errorf("proc[1] = %+v", procs[1])
	}
}

func TestParseLsofCwd(t *testing.T) {
	out := "p19278\nfcwd\nn/Users/rishugoyal/wt/foo\np601\nftxt\nn/should/be/ignored\nfcwd\nn/Users/rishugoyal/wt/bar\n"
	got := parseLsofCwd(out)
	want := map[int]string{
		19278: "/Users/rishugoyal/wt/foo",
		601:   "/Users/rishugoyal/wt/bar",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLsofCwd = %v, want %v", got, want)
	}
}

func TestAssign_SymlinkAndNesting(t *testing.T) {
	root := t.TempDir()
	wt := filepath.Join(root, "wt")
	other := filepath.Join(root, "other")
	for _, d := range []string{wt, other, filepath.Join(wt, "sub")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	matched := []Agent{
		{PID: 1, Name: "claude"}, // cwd == wt
		{PID: 2, Name: "aider"},  // cwd in wt/sub → still wt
		{PID: 3, Name: "codex"},  // cwd in other → not wt
		{PID: 4, Name: "gemini"}, // no cwd → dropped
	}
	cwds := map[int]string{
		1: wt,
		2: filepath.Join(wt, "sub"),
		3: other,
	}
	got := assign(cwds, matched, []string{wt, other})
	if n := names(got[wt]); !reflect.DeepEqual(n, []string{"aider", "claude"}) {
		t.Errorf("wt agents = %v, want [aider claude]", n)
	}
	if n := names(got[other]); !reflect.DeepEqual(n, []string{"codex"}) {
		t.Errorf("other agents = %v, want [codex]", n)
	}
}

func TestKnownNames_EnvMerge(t *testing.T) {
	t.Setenv("SEALPUP_AGENTS", "myagent, other ,")
	m := knownNames()
	if !m["myagent"] || !m["other"] {
		t.Errorf("SEALPUP_AGENTS names not merged: %v", m)
	}
	if !m["claude"] {
		t.Errorf("defaults dropped when env set")
	}
	if m[""] {
		t.Errorf("empty name should be ignored")
	}
}
