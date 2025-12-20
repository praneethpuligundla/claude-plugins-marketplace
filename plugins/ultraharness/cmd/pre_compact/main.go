// PreCompact hook preserves essential context before compaction.
//
// This hook runs before context compaction to:
// 1. Extract essential context (decisions, blockers, discoveries)
// 2. Save to preserved context file
// 3. Inject focus directive for post-compaction
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ultraharness/internal/artifacts"
	"ultraharness/internal/config"
	"ultraharness/internal/context"
	"ultraharness/internal/protocol"
	"ultraharness/internal/validation"
)

// PreservedContextFile is the name of the preserved context file.
const PreservedContextFile = "fic-preserved-context.json"

// FilePermission for preserved context file.
const FilePermission = 0600

// DirPermission for state directories.
const DirPermission = 0700

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

	// Check if FIC is enabled
	if !cfg.FICEnabled {
		return protocol.WriteEmpty()
	}

	// Read input from stdin
	input, err := protocol.ReadInput()
	if err != nil {
		return protocol.WriteEmpty()
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	var messages []string

	// Get current phase info
	phaseInfo := artifacts.GetPhaseInfo(workDir)
	phase := phaseInfo["phase"].(string)
	details := phaseInfo["details"].(map[string]interface{})

	// Load context state for additional info
	var tokenEstimate int
	var utilization float64
	if cfg.FICContextTracking {
		state, err := context.LoadContextState(sessionID, workDir)
		if err == nil && state != nil {
			tokenEstimate = state.TotalTokenEstimate
			utilization = state.UtilizationPercent
			messages = append(messages, fmt.Sprintf("[FIC] Context state: %.0f%% utilization, %d tokens estimated",
				utilization*100, tokenEstimate))
		}
	}

	// Build focus directive
	focusDirective := buildFocusDirective(phase, details)

	// Assemble preserved context
	preservedContext := map[string]interface{}{
		"timestamp":               time.Now().Format(time.RFC3339),
		"session_id":              sessionID,
		"phase":                   phase,
		"phase_details":           details,
		"focus_directive":         focusDirective,
		"essential_discoveries":   []interface{}{}, // Would be populated from context state
		"token_estimate_at_compact": tokenEstimate,
		"utilization_at_compact":  utilization,
	}

	// Save preserved context
	if savePreservedContext(preservedContext, workDir) {
		messages = append(messages, "[FIC] Context preserved for next session.")
	}

	// Build focus directive message
	messages = append(messages, "")
	messages = append(messages, strings.Repeat("=", 50))
	messages = append(messages, "FIC CONTEXT PRESERVATION")
	messages = append(messages, strings.Repeat("=", 50))
	messages = append(messages, fmt.Sprintf("Phase: %s", phase))
	messages = append(messages, fmt.Sprintf("Focus: %s", focusDirective))

	messages = append(messages, strings.Repeat("=", 50))
	messages = append(messages, "")
	messages = append(messages, "After compaction, continue with the focus directive above.")
	messages = append(messages, "Disregard exploration noise. Focus on completing the current phase.")

	return protocol.WriteSystemMessage(strings.Join(messages, "\n"))
}

func buildFocusDirective(phase string, details map[string]interface{}) string {
	switch phase {
	case "IMPLEMENTATION":
		if stepsInProgress, ok := details["steps_in_progress"].([]string); ok && len(stepsInProgress) > 0 {
			if len(stepsInProgress) > 3 {
				stepsInProgress = stepsInProgress[:3]
			}
			return fmt.Sprintf("Continue implementation. In progress: %s", strings.Join(stepsInProgress, ", "))
		}
		if completed, ok := details["steps_completed"].(int); ok {
			return fmt.Sprintf("Continue implementation. %d steps completed.", completed)
		}
		return "Continue implementation."

	case "IMPLEMENTATION_READY":
		if goal, ok := details["goal"].(string); ok && goal != "" {
			truncated := goal
			if len(truncated) > 60 {
				truncated = truncated[:60] + "..."
			}
			return fmt.Sprintf("Plan validated. Begin implementation of: %s", truncated)
		}
		return "Plan validated. Begin implementation."

	case "PLANNING":
		if goal, ok := details["goal"].(string); ok && goal != "" {
			truncated := goal
			if len(truncated) > 60 {
				truncated = truncated[:60] + "..."
			}
			return fmt.Sprintf("Continue planning. Goal: %s", truncated)
		}
		return "Continue planning."

	case "PLANNING_READY":
		if confidence, ok := details["confidence"].(float64); ok {
			return fmt.Sprintf("Research complete (confidence: %.0f%%). Create implementation plan.", confidence*100)
		}
		return "Research complete. Create implementation plan."

	case "RESEARCH":
		if feature, ok := details["feature"].(string); ok && feature != "" {
			return fmt.Sprintf("Continue research on: %s. Build confidence to >= 70%%.", feature)
		}
		return "Continue research. Build confidence to >= 70%."

	default:
		return "Review context and determine next steps."
	}
}

func savePreservedContext(ctx map[string]interface{}, workDir string) bool {
	preservedDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(preservedDir, DirPermission); err != nil {
		return false
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return false
	}

	preservedPath := filepath.Join(preservedDir, PreservedContextFile)
	if err := os.WriteFile(preservedPath, data, FilePermission); err != nil {
		return false
	}

	return true
}
