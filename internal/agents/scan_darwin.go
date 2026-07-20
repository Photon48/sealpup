//go:build darwin

package agents

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func init() {
	procLister = darwinProcs
	cwdResolver = darwinCwds
}

// scanTimeout bounds each external command so a hung ps/lsof can never hang a
// sealpup command.
const scanTimeout = 3 * time.Second

// darwinProcs lists candidate processes via two `ps` calls. comm and args are
// each last-position fields (so embedded spaces in paths parse unambiguously),
// which is why they're fetched separately and merged by pid.
func darwinProcs() []proc {
	comm, ok1 := runOut("ps", "-axo", "pid=,comm=")
	args, ok2 := runOut("ps", "-axo", "pid=,args=")
	if !ok1 || !ok2 {
		return nil
	}
	return parsePSLines(comm, args)
}

// darwinCwds resolves the cwd of the given pids with a single lsof invocation.
// The -a flag ANDs the -p and -d selectors; without it lsof dumps every fd of
// every process.
func darwinCwds(pids []int) map[int]string {
	if len(pids) == 0 {
		return map[int]string{}
	}
	strs := make([]string, len(pids))
	for i, p := range pids {
		strs[i] = strconv.Itoa(p)
	}
	// lsof exits non-zero if any listed pid has already died; parse whatever
	// output we got regardless.
	out, _ := runOut("lsof", "-a", "-p", strings.Join(strs, ","), "-d", "cwd", "-Fn")
	return parseLsofCwd(out)
}

// runOut runs a command with a timeout and returns its stdout. ok is false on
// timeout or spawn failure (a non-zero exit with output still returns ok=true).
func runOut(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if ctx.Err() != nil {
		return "", false
	}
	if err != nil && len(out) == 0 {
		return "", false
	}
	return string(out), true
}
