package cli

import (
	"fmt"

	"github.com/Photon48/sealpup/internal/agents"
)

// detectAgents is the seam for agent detection, overridable in tests so command
// behavior can be exercised without spawning real processes.
var detectAgents = agents.Detect

// agentLabel renders the AGENT cell for a worktree: "-" when none, the first
// agent with its pid, and "+N" when more share the worktree.
func agentLabel(ags []agents.Agent) string {
	if len(ags) == 0 {
		return "-"
	}
	label := fmt.Sprintf("%s (pid %d)", ags[0].Name, ags[0].PID)
	if len(ags) > 1 {
		label += fmt.Sprintf(" +%d", len(ags)-1)
	}
	return label
}
