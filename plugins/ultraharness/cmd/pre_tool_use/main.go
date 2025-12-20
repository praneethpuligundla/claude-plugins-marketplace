// PreToolUse hook enforces FIC verification gates for file modifications.
//
// Gate behavior by strictness mode:
// - relaxed: No validation, all operations allowed
// - standard: Warn on gate violations, allow operation
// - strict: Block operations that violate gates
package main

import (
	"os"

	"ultraharness/internal/config"
	"ultraharness/internal/gates"
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

	// Only check gates for file modifications
	toolName := input.ToolName
	if toolName != "Edit" && toolName != "Write" {
		return protocol.WriteEmpty()
	}

	// Check if FIC is enabled
	if !cfg.FICEnabled {
		return protocol.WriteEmpty()
	}

	// Determine which gate to check
	var gate string
	if toolName == "Edit" {
		gate = gates.GateAllowEdit
	} else {
		gate = gates.GateAllowWrite
	}

	// Check the gate
	result := gates.CheckGate(gate, workDir, cfg.Strictness)

	// Handle result
	switch result.Action {
	case gates.ActionBlock:
		msg := gates.FormatGateMessage(result)
		msg += "\n\n[FIC Gate: Operation blocked. Complete prior phase first.]"
		return protocol.WriteDeny(msg)

	case gates.ActionWarn:
		msg := gates.FormatGateMessage(result)
		if msg != "" {
			return protocol.WriteMessage(msg)
		}
		return protocol.WriteEmpty()

	default:
		return protocol.WriteEmpty()
	}
}
