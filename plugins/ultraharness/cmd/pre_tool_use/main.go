// PreToolUse hook provides phase awareness for file modifications.
//
// Philosophy: NEVER block individual tools. Gates are advisory, not blocking.
// The correct model is to pause at phase boundaries, not tool boundaries.
//
// This hook runs before Edit/Write tools to:
// - Provide phase awareness messages (advisory only)
// - Help users understand the workflow context
// - NEVER deny tool execution (that's the old wrong model)
package main

import (
	"os"

	"ultraharness/internal/artifacts"
	"ultraharness/internal/config"
	"ultraharness/internal/protocol"
	"ultraharness/internal/validation"
)

func main() {
	if err := run(); err != nil {
		protocol.WriteError("%v", err)
	}
	os.Exit(0)
}

func run() error {
	// Get working directory
	workDir := validation.GetWorkDir()
	if workDir == "" {
		return protocol.WriteEmpty()
	}

	// Check if harness is initialized
	if !config.IsHarnessInitialized(workDir) {
		return protocol.WriteEmpty()
	}

	// Load config
	cfg, err := config.Load(workDir)
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Skip all validation in relaxed mode
	if cfg.IsRelaxedMode() {
		return protocol.WriteEmpty()
	}

	// Read input from stdin
	input, err := protocol.ReadInput()
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Only provide awareness for file modifications
	toolName := input.ToolName
	if toolName != "Edit" && toolName != "Write" {
		return protocol.WriteEmpty()
	}

	// Check if FIC is enabled
	if !cfg.FICEnabled {
		return protocol.WriteEmpty()
	}

	// Get current phase for awareness (NOT blocking)
	phase := artifacts.GetCurrentPhase(workDir)

	// Provide advisory messages based on phase (never block)
	msg := getPhaseAwarenessMessage(phase, toolName, cfg)
	if msg != "" {
		return protocol.WriteMessage(msg)
	}

	return protocol.WriteEmpty()
}

// getPhaseAwarenessMessage returns an advisory message based on the current phase.
// This NEVER blocks - it only provides helpful context.
func getPhaseAwarenessMessage(phase, toolName string, cfg *config.Config) string {
	switch phase {
	case "NEW_SESSION":
		if cfg.ShouldWarnOnResearchIncomplete() {
			return "[FIC] Note: Starting modifications without prior research. For complex tasks, consider research first."
		}
	case "RESEARCH":
		if cfg.ShouldWarnOnResearchIncomplete() {
			return "[FIC] Note: Modifying files while in research phase. Ensure research is complete (70%+ confidence) for best results."
		}
	case "PLANNING_READY":
		if cfg.ShouldWarnOnPlanIncomplete() {
			return "[FIC] Note: Research complete but no validated plan. Consider creating a plan before implementation."
		}
	case "PLANNING":
		if cfg.ShouldWarnOnPlanIncomplete() {
			return "[FIC] Note: Plan exists but may not be validated. Consider validation with @fic-plan-validator."
		}
	}

	// No message for IMPLEMENTATION_READY or IMPLEMENTATION - those are the expected phases
	return ""
}
