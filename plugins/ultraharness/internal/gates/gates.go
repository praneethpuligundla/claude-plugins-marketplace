// Package gates implements FIC (Feature-Implementation-Completion) phase awareness.
//
// Philosophy: Gates are ADVISORY, not blocking. The correct model is to pause at
// phase boundaries (via checkpoints), not to block individual tool calls.
// This package provides phase awareness messages but NEVER blocks operations.
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

// CheckGate checks phase status and returns advisory messages.
// NOTE: This function NEVER blocks operations - it only provides awareness.
// The blocking model has been replaced with pause-and-prompt checkpoints.
func CheckGate(gate string, workDir string, strictness string) *GateResult {
	// Relaxed mode: no messages
	if strictness == "relaxed" {
		return &GateResult{Action: ActionAllow}
	}

	// Load FIC state
	state, err := LoadFICState(workDir)
	if err != nil {
		// On error, allow silently
		return &GateResult{Action: ActionAllow}
	}

	// Check gate based on phase (advisory only, never block)
	switch gate {
	case GateAllowEdit, GateAllowWrite:
		return checkEditWriteGateAdvisory(state)
	case GateAllowBash:
		return &GateResult{Action: ActionAllow} // Bash always allowed
	default:
		return &GateResult{Action: ActionAllow}
	}
}

// checkEditWriteGateAdvisory returns advisory messages based on phase (never blocks)
func checkEditWriteGateAdvisory(state *FICState) *GateResult {
	// If research is not complete, provide advisory
	if !state.ResearchComplete {
		return &GateResult{
			Action: ActionWarn, // Advisory only, will still allow
			Reason: "Research phase not complete",
			Suggestions: []string{
				"For complex tasks, consider research first",
				"Use @fic-researcher for structured exploration",
			},
		}
	}

	// If plan is not validated, provide advisory
	if !state.PlanValidated {
		return &GateResult{
			Action: ActionWarn, // Advisory only, will still allow
			Reason: "Plan not validated",
			Suggestions: []string{
				"Consider creating a plan for multi-step changes",
				"Use @fic-plan-validator for plan review",
			},
		}
	}

	// Phase is appropriate for implementation
	return &GateResult{Action: ActionAllow}
}

// Legacy function kept for backward compatibility
func checkEditWriteGate(state *FICState, strictness string) *GateResult {
	// Always use advisory mode now - blocking is deprecated
	return checkEditWriteGateAdvisory(state)
}

func checkBashGate(state *FICState, strictness string) *GateResult {
	// Bash is always allowed
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

// GateConfig holds gate-specific configuration options
type GateConfig struct {
	WarnOnResearchIncomplete bool
	WarnOnPlanIncomplete     bool
	BlockInStrictMode        bool
}

// DefaultGateConfig returns the default gate configuration
func DefaultGateConfig() *GateConfig {
	return &GateConfig{
		WarnOnResearchIncomplete: true,
		WarnOnPlanIncomplete:     true,
		BlockInStrictMode:        true,
	}
}

// CheckGateWithConfig checks phase status with custom config (advisory only, never blocks)
func CheckGateWithConfig(gate string, workDir string, strictness string, gateConfig *GateConfig) *GateResult {
	if gateConfig == nil {
		gateConfig = DefaultGateConfig()
	}

	// Relaxed mode: no messages
	if strictness == "relaxed" {
		return &GateResult{Action: ActionAllow}
	}

	// Load FIC state
	state, err := LoadFICState(workDir)
	if err != nil {
		return &GateResult{Action: ActionAllow}
	}

	// Check gate based on phase (advisory only)
	switch gate {
	case GateAllowEdit, GateAllowWrite:
		return checkEditWriteGateWithConfigAdvisory(state, gateConfig)
	case GateAllowBash:
		return &GateResult{Action: ActionAllow}
	default:
		return &GateResult{Action: ActionAllow}
	}
}

// checkEditWriteGateWithConfigAdvisory returns advisory messages based on config (never blocks)
func checkEditWriteGateWithConfigAdvisory(state *FICState, gateConfig *GateConfig) *GateResult {
	// If research is not complete and warnings are enabled
	if !state.ResearchComplete && gateConfig.WarnOnResearchIncomplete {
		return &GateResult{
			Action: ActionWarn, // Advisory only
			Reason: "Research phase not complete",
			Suggestions: []string{
				"For complex tasks, consider research first",
				"Use @fic-researcher for structured exploration",
			},
		}
	}

	// If plan is not validated and warnings are enabled
	if !state.PlanValidated && gateConfig.WarnOnPlanIncomplete {
		return &GateResult{
			Action: ActionWarn, // Advisory only
			Reason: "Plan not validated",
			Suggestions: []string{
				"Consider creating a plan for multi-step changes",
				"Use @fic-plan-validator for plan review",
			},
		}
	}

	return &GateResult{Action: ActionAllow}
}

// Legacy function kept for backward compatibility (redirects to advisory version)
func checkEditWriteGateWithConfig(state *FICState, strictness string, gateConfig *GateConfig) *GateResult {
	return checkEditWriteGateWithConfigAdvisory(state, gateConfig)
}

// SaveFICState saves the FIC state to disk
func SaveFICState(workDir string, state *FICState) error {
	stateDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	state.LastUpdated = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(stateDir, FICStateFileName)
	return os.WriteFile(statePath, data, 0600)
}
