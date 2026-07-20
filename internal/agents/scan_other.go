//go:build !darwin && !linux

package agents

// On unsupported platforms sealpup simply reports no agents. The default seams
// in agents.go already return empty, so there's nothing to wire up here.
