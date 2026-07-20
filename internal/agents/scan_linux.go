//go:build linux

package agents

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func init() {
	procLister = linuxProcs
	cwdResolver = linuxCwds
}

// linuxProcs reads candidate processes from /proc. It uses argv[0] from
// /proc/N/cmdline as comm (avoiding the 15-char truncation of /proc/N/comm) and
// the full cmdline as args.
func linuxProcs() []proc {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a pid dir
		}
		raw, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil || len(raw) == 0 {
			continue
		}
		argv := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
		out = append(out, proc{
			pid:  pid,
			comm: argv[0],
			args: strings.Join(argv, " "),
		})
	}
	return out
}

// linuxCwds reads /proc/N/cwd symlinks for the given pids, skipping any that
// error (permission denied, or the process exited mid-scan).
func linuxCwds(pids []int) map[int]string {
	res := make(map[int]string, len(pids))
	for _, pid := range pids {
		if target, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "cwd")); err == nil {
			res[pid] = target
		}
	}
	return res
}
