// SessionStart hook provides session context with FIC workflow state.
//
// This hook runs at the start of each Claude Code session to:
// 1. Check if harness is initialized for the current project
// 2. Load FIC state: phase, confidence, artifacts
// 3. Show preserved context from prior sessions
// 4. Execute init.sh if it exists
// 5. Run baseline tests if configured
// 6. Display git status and recent commits
// 7. Read progress file for context
// 8. Read feature checklist status
// 9. Inject context into the session via systemMessage
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
	"ultraharness/internal/features"
	"ultraharness/internal/git"
	"ultraharness/internal/initscript"
	"ultraharness/internal/progress"
	"ultraharness/internal/protocol"
	"ultraharness/internal/testrunner"
	"ultraharness/internal/validation"
)

// PreservedContextFile is the name of the preserved context file.
const PreservedContextFile = "fic-preserved-context.json"

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
		return writeInitMessage()
	}

	// Check if harness is initialized
	if !config.IsHarnessInitialized(workDir) {
		return writeInitMessage()
	}

	// Load config
	cfg, err := config.Load(workDir)
	if err != nil {
		return writeInitMessage()
	}

	// Build context message
	return writeContextMessage(workDir, cfg)
}

func writeInitMessage() error {
	msg := "[FIC System] This project has not been initialized. " +
		"Run `/ultraharness:init` to enable the FIC (Flow-Information-Context) system. " +
		"This provides automatic Research → Plan → Implement workflow with verification gates."
	return protocol.WriteSystemMessage(msg)
}

func writeContextMessage(workDir string, cfg *config.Config) error {
	var messages []string

	messages = append(messages, "=== FIC SYSTEM SESSION STARTUP ===")
	messages = append(messages, fmt.Sprintf("Session started: %s", time.Now().Format(time.RFC3339)))
	messages = append(messages, fmt.Sprintf("Working directory: %s", workDir))
	messages = append(messages, fmt.Sprintf("Mode: %s", cfg.Strictness))
	messages = append(messages, "")

	// FIC Workflow State (High Priority)
	if cfg.FICEnabled {
		ficMessages := formatFICState(workDir)
		if len(ficMessages) > 0 {
			messages = append(messages, ficMessages...)
		}
	}

	// Run init script
	if cfg.InitScriptExecution {
		initResult := initscript.Run(workDir, 0)
		if resultStr := initscript.GetResultString(initResult); resultStr != "" {
			messages = append(messages, "--- INIT SCRIPT ---")
			messages = append(messages, resultStr)
			messages = append(messages, "")
		}
	}

	// Run baseline tests
	if cfg.BaselineTestsOnStartup {
		testSummary := testrunner.Run(workDir, testrunner.DefaultTimeout)
		if testSummary.Result != testrunner.NotRun {
			messages = append(messages, "--- BASELINE TESTS ---")
			summaryStr := testrunner.GetSummaryString(testSummary)
			if testSummary.Result == testrunner.Passed {
				messages = append(messages, fmt.Sprintf("Baseline tests PASSED: %s", summaryStr))
			} else if testSummary.Result == testrunner.Failed {
				messages = append(messages, fmt.Sprintf("WARNING: Baseline tests FAILING: %s", summaryStr))
				messages = append(messages, "Review failures before making changes.")
			} else {
				messages = append(messages, fmt.Sprintf("Baseline test error: %s", testSummary.RawOutput[:min(200, len(testSummary.RawOutput))]))
			}
			messages = append(messages, "")
		}
	}

	// Git status and log
	if git.IsRepo(workDir) {
		messages = append(messages, "--- GIT STATUS ---")
		status := git.Status(workDir)
		if status != "" {
			messages = append(messages, status)
		} else {
			messages = append(messages, "(clean)")
		}
		messages = append(messages, "")

		messages = append(messages, "--- RECENT COMMITS ---")
		log := git.Log(workDir, 10)
		if log != "" {
			messages = append(messages, log)
		} else {
			messages = append(messages, "(no commits)")
		}
		messages = append(messages, "")
	}

	// Progress file
	progressContent, err := progress.Read(workDir)
	if err == nil && progressContent != "" {
		messages = append(messages, "--- PROGRESS LOG ---")
		// Truncate to last 50 lines
		lines := strings.Split(progressContent, "\n")
		if len(lines) > 50 {
			messages = append(messages, "[...truncated...]")
			lines = lines[len(lines)-50:]
		}
		messages = append(messages, strings.Join(lines, "\n"))
		messages = append(messages, "")
	}

	// Features checklist
	if features.Exists(workDir) {
		summary, err := features.GetSummary(workDir)
		if err == nil {
			messages = append(messages, "--- FEATURE CHECKLIST STATUS ---")
			messages = append(messages, fmt.Sprintf("Total: %d | Passing: %d | Failing: %d | In Progress: %d",
				summary.Total, summary.Passing, summary.Failing, summary.InProgress))

			if len(summary.NextItems) > 0 {
				messages = append(messages, "")
				messages = append(messages, "Next priority items:")
				for _, item := range summary.NextItems {
					statusIcon := "[TODO]"
					if item.Status == "in_progress" {
						statusIcon = "[WIP]"
					}
					desc := item.Description
					if len(desc) > 60 {
						desc = desc[:60] + "..."
					}
					messages = append(messages, fmt.Sprintf("  %s %s. %s: %s", statusIcon, item.ID, item.Name, desc))
				}
			}
			messages = append(messages, "")
		}
	}

	messages = append(messages, "=== END SESSION CONTEXT ===")
	messages = append(messages, "")

	// Automation features
	var autoFeatures []string
	if cfg.AutoProgressLogging {
		autoFeatures = append(autoFeatures, "auto-logging")
	}
	if cfg.AutoCheckpointSuggestions {
		autoFeatures = append(autoFeatures, "checkpoint suggestions")
	}
	if cfg.FeatureEnforcement {
		autoFeatures = append(autoFeatures, "feature enforcement")
	}
	if cfg.FICEnabled {
		autoFeatures = append(autoFeatures, "FIC context tracking")
	}
	if len(autoFeatures) > 0 {
		messages = append(messages, fmt.Sprintf("Automation enabled: %s", strings.Join(autoFeatures, ", ")))
		messages = append(messages, "")
	}

	// Phase-specific guidance
	phase := artifacts.GetCurrentPhase(workDir)
	messages = append(messages, getPhaseGuidance(phase))

	return protocol.WriteSystemMessage(strings.Join(messages, "\n"))
}

func formatFICState(workDir string) []string {
	var messages []string

	messages = append(messages, "--- FIC WORKFLOW STATE ---")

	phase := artifacts.GetCurrentPhase(workDir)
	messages = append(messages, fmt.Sprintf("Phase: %s", phase))

	// Show preserved context from prior session
	preserved := loadPreservedContext(workDir)
	if preserved != nil {
		messages = append(messages, "")
		messages = append(messages, "Prior Session Context:")
		if discoveries, ok := preserved["essential_discoveries"].([]interface{}); ok {
			for i, d := range discoveries {
				if i >= 5 {
					break
				}
				if disc, ok := d.(map[string]interface{}); ok {
					if summary, ok := disc["summary"].(string); ok {
						messages = append(messages, fmt.Sprintf("  - %s", summary))
					}
				}
			}
		}
		if focus, ok := preserved["focus_directive"].(string); ok && focus != "" {
			messages = append(messages, fmt.Sprintf("Focus: %s", focus))
		}
	}

	// Show research state
	if research, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactResearch); research != nil {
		if r, ok := research.(*artifacts.Research); ok {
			messages = append(messages, "")
			messages = append(messages, fmt.Sprintf("Active Research: %s", r.FeatureOrTask))
			messages = append(messages, fmt.Sprintf("  Confidence: %.0f%%", r.ConfidenceScore*100))
			messages = append(messages, fmt.Sprintf("  Discoveries: %d", len(r.Discoveries)))

			blockingQ := 0
			for _, q := range r.OpenQuestions {
				if q.Blocking {
					blockingQ++
				}
			}
			messages = append(messages, fmt.Sprintf("  Open Questions: %d (%d blocking)", len(r.OpenQuestions), blockingQ))
		}
	}

	// Show plan state
	if plan, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactPlan); plan != nil {
		if p, ok := plan.(*artifacts.Plan); ok {
			messages = append(messages, "")
			goal := p.Goal
			if len(goal) > 60 {
				goal = goal[:60] + "..."
			}
			messages = append(messages, fmt.Sprintf("Active Plan: %s", goal))
			messages = append(messages, fmt.Sprintf("  Steps: %d", len(p.Steps)))
			if p.ValidationResult != nil {
				messages = append(messages, fmt.Sprintf("  Validation: %s", p.ValidationResult.Recommendation))
			}
		}
	}

	// Show implementation progress
	if impl, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactImplementation); impl != nil {
		if i, ok := impl.(*artifacts.Implementation); ok {
			messages = append(messages, "")
			messages = append(messages, "Implementation Progress:")
			messages = append(messages, fmt.Sprintf("  Completed Steps: %d", len(i.StepsCompleted)))
			messages = append(messages, fmt.Sprintf("  In Progress: %d", len(i.StepsInProgress)))
			if len(i.PlanDeviations) > 0 {
				messages = append(messages, fmt.Sprintf("  Plan Deviations: %d", len(i.PlanDeviations)))
			}
		}
	}

	messages = append(messages, "")
	return messages
}

func loadPreservedContext(workDir string) map[string]interface{} {
	preservedPath := filepath.Join(workDir, ".claude", PreservedContextFile)
	data, err := os.ReadFile(preservedPath)
	if err != nil {
		return nil
	}

	var preserved map[string]interface{}
	if err := json.Unmarshal(data, &preserved); err != nil {
		return nil
	}
	return preserved
}

func getPhaseGuidance(phase string) string {
	switch phase {
	case "NEW_SESSION":
		return "IMPORTANT: This is a new session. For complex tasks, start with RESEARCH to understand the codebase.\nDelegate exploration to subagents to keep main context clean."
	case "RESEARCH":
		return "IMPORTANT: Continue RESEARCH phase. Build confidence before planning.\nUse subagents for exploration. Only essential findings should enter main context."
	case "PLANNING_READY":
		return "IMPORTANT: Research complete. Ready to create an implementation PLAN.\nCreate specific, actionable steps with verification criteria."
	case "PLANNING":
		return "IMPORTANT: Continue PLANNING. Validate the plan before implementation."
	case "IMPLEMENTATION_READY":
		return "IMPORTANT: Plan validated. Ready to IMPLEMENT.\nFollow the plan steps. Document any deviations."
	case "IMPLEMENTATION":
		return "IMPORTANT: Continue IMPLEMENTATION. Track progress against the plan."
	default:
		return "IMPORTANT: Review the above context. For complex tasks, start with RESEARCH phase.\nThe FIC system will automatically track your workflow progression."
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
