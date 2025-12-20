// Package gates implements FIC (Feature-Implementation-Completion) verification gates.
// Gates enforce the research → planning → implementation workflow.
package gates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Gate types
const (
	GateAllowEdit  = "allow_edit"
	GateAllowWrite = "allow_write"
	GateAllowBash  = "allow_bash"
)

// GateAction represents the action to take
type GateAction string

const (
	ActionAllow GateAction = "allow"
	ActionWarn  GateAction = "warn"
	ActionBlock GateAction = "block"
)

// GateResult contains the result of a gate check
type GateResult struct {
	Action      GateAction
	Reason      string
	Suggestions []string
}

// FICState represents the current FIC workflow state
type FICState struct {
	Phase            string    `json:"phase"` // "research", "planning", "implementation"
	ResearchComplete bool      `json:"research_complete"`
	PlanValidated    bool      `json:"plan_validated"`
	LastUpdated      time.Time `json:"last_updated"`
}

// FICStateFileName is the name of the FIC state file
const FICStateFileName = "fic-state.json"

// LoadFICState loads the FIC state from the working directory
func LoadFICState(workDir string) (*FICState, error) {
	statePath := filepath.Join(workDir, ".claude", FICStateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Default state: not initialized
			return &FICState{
				Phase:            "research",
				ResearchComplete: false,
				PlanValidated:    false,
			}, nil
		}
		return nil, err
	}

	var state FICState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// CheckGate checks if an operation is allowed based on FIC state
func CheckGate(gate string, workDir string, strictness string) *GateResult {
	// Relaxed mode: always allow
	if strictness == "relaxed" {
		return &GateResult{Action: ActionAllow}
	}

	// Load FIC state
	state, err := LoadFICState(workDir)
	if err != nil {
		// On error, allow but warn
		return &GateResult{
			Action: ActionAllow,
			Reason: fmt.Sprintf("Could not load FIC state: %v", err),
		}
	}

	// Check gate based on phase
	switch gate {
	case GateAllowEdit, GateAllowWrite:
		return checkEditWriteGate(state, strictness)
	case GateAllowBash:
		return checkBashGate(state, strictness)
	default:
		return &GateResult{Action: ActionAllow}
	}
}

func checkEditWriteGate(state *FICState, strictness string) *GateResult {
	// If research is not complete, block/warn
	if !state.ResearchComplete {
		result := &GateResult{
			Reason: "Research phase not complete",
			Suggestions: []string{
				"Complete research using Read, Grep, Glob, Task tools first",
				"Use /fic-research-done when research is complete",
			},
		}
		if strictness == "strict" {
			result.Action = ActionBlock
		} else {
			result.Action = ActionWarn
		}
		return result
	}

	// If plan is not validated, block/warn
	if !state.PlanValidated {
		result := &GateResult{
			Reason: "Planning phase not complete",
			Suggestions: []string{
				"Create and validate your implementation plan",
				"Use /fic-plan-done when plan is validated",
			},
		}
		if strictness == "strict" {
			result.Action = ActionBlock
		} else {
			result.Action = ActionWarn
		}
		return result
	}

	// All gates passed
	return &GateResult{Action: ActionAllow}
}

func checkBashGate(state *FICState, strictness string) *GateResult {
	// Bash is allowed in all phases for read-only operations
	// Only block destructive commands in early phases (not implemented here)
	return &GateResult{Action: ActionAllow}
}

// FormatGateMessage formats the gate result as a user-friendly message
func FormatGateMessage(result *GateResult) string {
	if result.Action == ActionAllow {
		return ""
	}

	msg := fmt.Sprintf("[FIC Gate] %s: %s", result.Action, result.Reason)

	if len(result.Suggestions) > 0 {
		msg += "\nSuggestions:"
		for _, s := range result.Suggestions {
			msg += fmt.Sprintf("\n  - %s", s)
		}
	}

	return msg
}
