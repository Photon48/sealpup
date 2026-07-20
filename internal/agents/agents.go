// Package agents detects running coding agents (Claude Code, aider, …) by
// scanning live processes and matching those whose working directory sits
// inside a worktree. It keeps no state — nothing to go stale — and never fails
// a command: any scan error degrades to "no agents found".
package agents

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Agent is a detected agent process.
type Agent struct {
	PID  int
	Name string // the matched known-agent name, e.g. "claude"
}

// proc is a candidate process: its argv[0] (comm) and full command line (args).
type proc struct {
	pid   int
	comm  string
	args  string
}

// defaultAgents are the process names sealpup recognizes out of the box. Match
// is exact on the basename, so "amp" never matches macOS's "AMPLibraryAgent".
var defaultAgents = []string{
	"claude", "aider", "codex", "cursor-agent",
	"gemini", "amp", "goose", "opencode", "copilot",
}

// Platform seams, set by scan_{darwin,linux}.go via init(). Defaults are inert
// so unsupported platforms (and pure-layer tests) simply find nothing.
var (
	procLister  = func() []proc { return nil }
	cwdResolver = func(pids []int) map[int]string { return map[int]string{} }
)

// Detect maps each worktree path to the agents whose cwd is at or below it.
func Detect(paths []string) map[string][]Agent {
	if len(paths) == 0 {
		return map[string][]Agent{}
	}
	matched := matchProcs(procLister(), knownNames())
	if len(matched) == 0 {
		return map[string][]Agent{}
	}
	pids := make([]int, len(matched))
	for i, a := range matched {
		pids[i] = a.PID
	}
	return assign(cwdResolver(pids), matched, paths)
}

// knownNames is the recognized set: the built-in list plus any names from the
// SEALPUP_AGENTS env var (comma-separated, exact basenames).
func knownNames() map[string]bool {
	m := make(map[string]bool, len(defaultAgents)+2)
	for _, n := range defaultAgents {
		m[n] = true
	}
	for _, n := range strings.Split(os.Getenv("SEALPUP_AGENTS"), ",") {
		if n = strings.TrimSpace(n); n != "" {
			m[n] = true
		}
	}
	return m
}

// matchProcs returns an Agent for every process recognized as a known agent.
func matchProcs(procs []proc, names map[string]bool) []Agent {
	var out []Agent
	for _, p := range procs {
		if name, ok := matchOne(p, names); ok {
			out = append(out, Agent{PID: p.pid, Name: name})
		}
	}
	return out
}

// matchOne recognizes a process two ways: (a) the basename of argv[0] is a known
// agent (covers `claude`, `/bin/sleep`); (b) an interpreter running an agent
// script — argv[1]'s basename is known (covers `node /…/bin/gemini`).
func matchOne(p proc, names map[string]bool) (string, bool) {
	if base := filepath.Base(p.comm); names[base] {
		return base, true
	}
	// Interpreter fallback: strip argv[0] (comm, verbatim) off the front of the
	// command line and inspect the next token.
	rest := strings.TrimLeft(strings.TrimPrefix(p.args, p.comm), " ")
	if rest == "" {
		return "", false
	}
	tok := rest
	if i := strings.IndexByte(rest, ' '); i >= 0 {
		tok = rest[:i]
	}
	if base := filepath.Base(tok); names[base] {
		return base, true
	}
	return "", false
}

// assign buckets each matched agent into the worktree containing its cwd. Paths
// are compared through EvalSymlinks so /var vs /private/var (macOS) agree.
func assign(cwds map[int]string, matched []Agent, paths []string) map[string][]Agent {
	type resolved struct{ orig, real string }
	rs := make([]resolved, 0, len(paths))
	for _, p := range paths {
		rs = append(rs, resolved{p, resolvePath(p)})
	}

	out := map[string][]Agent{}
	for _, a := range matched {
		cwd, ok := cwds[a.PID]
		if !ok {
			continue
		}
		rc := resolvePath(cwd)
		for _, r := range rs {
			if rc == r.real || strings.HasPrefix(rc, r.real+string(filepath.Separator)) {
				out[r.orig] = append(out[r.orig], a)
				break
			}
		}
	}
	return out
}

func resolvePath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

// parsePSLines merges two `ps` outputs (pid+comm, pid+args) into procs, keyed by
// pid. Both comm and args are last-position fields, so each line is "pid rest".
func parsePSLines(commOut, argsOut string) []proc {
	comm := map[int]string{}
	var order []int
	for _, line := range strings.Split(commOut, "\n") {
		if pid, rest, ok := splitPidRest(line); ok {
			if _, seen := comm[pid]; !seen {
				order = append(order, pid)
			}
			comm[pid] = rest
		}
	}
	args := map[int]string{}
	for _, line := range strings.Split(argsOut, "\n") {
		if pid, rest, ok := splitPidRest(line); ok {
			args[pid] = rest
		}
	}
	out := make([]proc, 0, len(order))
	for _, pid := range order {
		out = append(out, proc{pid: pid, comm: comm[pid], args: args[pid]})
	}
	return out
}

// splitPidRest parses a "  <pid> <rest…>" line, where rest may contain spaces.
func splitPidRest(line string) (int, string, bool) {
	line = strings.TrimLeft(line, " \t")
	i := strings.IndexAny(line, " \t")
	if i <= 0 {
		return 0, "", false
	}
	pid, err := strconv.Atoi(line[:i])
	if err != nil {
		return 0, "", false
	}
	return pid, strings.TrimLeft(line[i:], " \t"), true
}

// parseLsofCwd parses `lsof -Fn -d cwd` output: p<pid> / fcwd / n<path> records.
func parseLsofCwd(out string) map[int]string {
	res := map[int]string{}
	var pid int
	var isCwd bool
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
			isCwd = false
		case 'f':
			isCwd = line == "fcwd"
		case 'n':
			if isCwd && pid != 0 {
				res[pid] = line[1:]
			}
			isCwd = false
		}
	}
	return res
}
