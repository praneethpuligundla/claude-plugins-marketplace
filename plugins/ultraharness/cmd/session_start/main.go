// SessionStart hook provides lightweight session context.
//
// Philosophy: Lightweight startup with pointers to artifacts, not heavy preloading.
// Target: ~200 tokens of context injection (down from ~2000).
//
// This hook runs at the start of each Claude Code session to:
// 1. Check if harness is initialized for the current project
// 2. Load current FIC phase
// 3. Show focus directive (1-2 lines)
// 4. Point to artifacts (let model load on demand)
// 5. Check for pending phase checkpoints
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
	"ultraharness/internal/initscript"
	"ultraharness/internal/progress"
	"ultraharness/internal/protocol"
	"ultraharness/internal/testrunner"
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

	// Auto-initialize if not already done (zero user input required)
	if !config.IsHarnessInitialized(workDir) {
		if err := autoInitialize(workDir); err != nil {
			// Initialization failed, continue without harness
			return protocol.WriteEmpty()
		}
	}

	// Load config
	cfg, err := config.Load(workDir)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Build context message
	return writeContextMessage(workDir, cfg)
}

func writeContextMessage(workDir string, cfg *config.Config) error {
	var messages []string

	// Lightweight header
	messages = append(messages, "[FIC] Session Start")

	// Current phase and focus (most important info)
	phase := artifacts.GetCurrentPhase(workDir)
	focusDirective := getFocusDirective(workDir, phase)
	messages = append(messages, fmt.Sprintf("Phase: %s | Focus: %s", phase, focusDirective))

	// Artifact pointers (let model load on demand)
	messages = append(messages, fmt.Sprintf("Artifacts: %s | Progress: claude-progress.txt", artifacts.ArtifactsDir))

	// Check for pending phase checkpoint requiring human review
	if cfg.FICEnabled && cfg.RequireCheckpointReview() {
		if checkpoint := artifacts.GetPendingCheckpoint(workDir); checkpoint != nil {
			messages = append(messages, "")
			messages = append(messages, formatCheckpointReminder(checkpoint))
		}
	}

	// Run init script (if configured and exists)
	if cfg.InitScriptExecution {
		initResult := initscript.Run(workDir, 0)
		if resultStr := initscript.GetResultString(initResult); resultStr != "" {
			messages = append(messages, "")
			messages = append(messages, fmt.Sprintf("Init: %s", resultStr))
		}
	}

	// Brief baseline test status (if configured)
	if cfg.BaselineTestsOnStartup {
		testSummary := testrunner.Run(workDir, testrunner.DefaultTimeout)
		if testSummary.Result == testrunner.Failed {
			messages = append(messages, "")
			messages = append(messages, fmt.Sprintf("⚠ Baseline tests FAILING: %s", testrunner.GetSummaryString(testSummary)))
		}
	}

	// Brief feature status (only if items need attention)
	if features.Exists(workDir) {
		if summary, err := features.GetSummary(workDir); err == nil && summary.Failing > 0 {
			messages = append(messages, "")
			messages = append(messages, fmt.Sprintf("Features: %d/%d passing (%d failing)",
				summary.Passing, summary.Total, summary.Failing))
		}
	}

	return protocol.WriteSystemMessage(strings.Join(messages, "\n"))
}

// getFocusDirective returns a brief focus directive for the current phase.
func getFocusDirective(workDir string, phase string) string {
	switch phase {
	case "IMPLEMENTATION":
		if impl, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactImplementation); impl != nil {
			if i, ok := impl.(*artifacts.Implementation); ok && len(i.StepsInProgress) > 0 {
				return fmt.Sprintf("Continue: %s", i.StepsInProgress[0])
			}
		}
		return "Continue implementation"

	case "IMPLEMENTATION_READY":
		if plan, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactPlan); plan != nil {
			if p, ok := plan.(*artifacts.Plan); ok {
				goal := p.Goal
				if len(goal) > 50 {
					goal = goal[:50] + "..."
				}
				return fmt.Sprintf("Begin: %s", goal)
			}
		}
		return "Plan validated, begin implementation"

	case "PLANNING", "PLANNING_READY":
		if research, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactResearch); research != nil {
			if r, ok := research.(*artifacts.Research); ok {
				return fmt.Sprintf("Create plan (research %.0f%% confident)", r.ConfidenceScore*100)
			}
		}
		return "Create implementation plan"

	case "RESEARCH":
		if research, _ := artifacts.GetLatestArtifact(workDir, artifacts.ArtifactResearch); research != nil {
			if r, ok := research.(*artifacts.Research); ok {
				return fmt.Sprintf("Build understanding (%.0f%% confident)", r.ConfidenceScore*100)
			}
		}
		return "Explore codebase, build understanding"

	default:
		return "Start with research for complex tasks"
	}
}

// formatCheckpointReminder formats a reminder about a pending checkpoint.
func formatCheckpointReminder(checkpoint *artifacts.PhaseCheckpoint) string {
	return fmt.Sprintf("⏸ CHECKPOINT: %s phase complete - awaiting human review\n   %s\n   Reply 'proceed' to continue or provide feedback.",
		checkpoint.Phase, checkpoint.Summary)
}

// autoInitialize sets up the harness with zero user input.
// Creates .claude directory, marker file, and default config.
func autoInitialize(workDir string) error {
	claudeDir := filepath.Join(workDir, ".claude")

	// Create .claude directory if it doesn't exist
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	// Create marker file
	markerPath := filepath.Join(claudeDir, config.InitMarkerFileName)
	markerContent := fmt.Sprintf("# Ultraharness initialized\n# Auto-initialized: %s\n", time.Now().Format(time.RFC3339))
	if err := os.WriteFile(markerPath, []byte(markerContent), 0644); err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	// Write default config
	configPath := filepath.Join(claudeDir, config.ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := config.DefaultConfig()
		configData, err := json.MarshalIndent(defaultCfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	// Create initial progress file with auto-init entry
	progressPath := progress.GetProgressPath(workDir)
	if _, err := os.Stat(progressPath); os.IsNotExist(err) {
		initialProgress := fmt.Sprintf("# Ultraharness Progress Log\n# Auto-initialized: %s\n\n", time.Now().Format(time.RFC3339))
		if err := os.WriteFile(progressPath, []byte(initialProgress), 0600); err != nil {
			// Non-fatal - progress file is optional
		}
	}

	// Update .gitignore to ignore harness-specific files
	updateGitignore(workDir)

	return nil
}

// updateGitignore adds harness files to .gitignore
func updateGitignore(workDir string) {
	gitignorePath := filepath.Join(workDir, ".gitignore")

	harnessIgnores := []string{
		"# Ultraharness local files",
		"claude-progress.txt",
		".claude/fic-*.json",
		".claude/.claude-harness-initialized",
	}

	// Read existing .gitignore
	existing := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = string(data)
	}

	// Check which entries need to be added
	var toAdd []string
	for _, entry := range harnessIgnores {
		if !strings.Contains(existing, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return
	}

	// Append new entries
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		f.WriteString("\n")
	}
	f.WriteString("\n" + strings.Join(toAdd, "\n") + "\n")
}
