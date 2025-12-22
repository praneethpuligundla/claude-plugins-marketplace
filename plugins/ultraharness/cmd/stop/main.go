// Stop hook validates session stop conditions.
//
// This hook runs when a session is stopping to:
// 1. Check if tests were run (if code was modified)
// 2. Check for uncommitted changes
// 3. Check for features still in progress
// 4. Check if progress log was updated
// 5. Validate merge-ready state
//
// Behavior by strictness mode:
// - strict: Block if validation fails
// - standard: Strong warnings but no blocking
// - relaxed: Minimal suggestions only
package main

import (
	"os"
	"strings"

	"ultraharness/internal/config"
	"ultraharness/internal/features"
	"ultraharness/internal/git"
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

	// Check if harness is initialized
	if !config.IsHarnessInitialized(workDir) {
		return protocol.WriteEmpty()
	}

	// Load config
	cfg, err := config.Load(workDir)
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Read input from stdin
	input, err := protocol.ReadInput()
	if err != nil {
		return protocol.WriteEmpty()
	}

	// Get stop reason
	stopReason := input.GetStopReason()

	// Only validate for normal stops (not errors/interrupts)
	if stopReason != "end_turn" && stopReason != "stop_sequence" && stopReason != "" && stopReason != "unknown" {
		return protocol.WriteEmpty()
	}

	// Get transcript for test detection
	transcript := input.GetTranscript()

	// Run validation
	canStop, blockingReasons, warnings := validateStop(workDir, cfg, transcript)

	// Handle based on strictness mode
	if cfg.IsStrictMode() {
		return handleStrictMode(canStop, blockingReasons, warnings)
	} else if !cfg.IsRelaxedMode() {
		return handleStandardMode(blockingReasons, warnings)
	}
	return handleRelaxedMode(blockingReasons, warnings)
}

func validateStop(workDir string, cfg *config.Config, transcript string) (bool, []string, []string) {
	var blockingReasons []string
	var warnings []string

	codeModified := git.CodeWasModified(workDir)

	// Check 1: Tests not run (if code was modified)
	if codeModified {
		testsRan := testrunner.DidTestsRun(transcript)
		if !testsRan {
			blockingReasons = append(blockingReasons, "Code was modified but tests were not run")
		}
	}

	// Check 2: Uncommitted changes
	if git.HasUncommittedChanges(workDir) {
		warnings = append(warnings, "Uncommitted changes exist - consider creating a checkpoint")
	}

	// Check 3: Features still in progress
	if features.Exists(workDir) {
		inProgress, err := features.GetInProgress(workDir)
		if err == nil && len(inProgress) > 0 {
			featureNames := make([]string, 0, 3)
			for i, f := range inProgress {
				if i >= 3 {
					break
				}
				featureNames = append(featureNames, f.Name)
			}
			warnings = append(warnings, "Features still in progress: "+strings.Join(featureNames, ", "))
		}
	}

	// Check 4: Progress log not updated
	if codeModified {
		progressPath := progress.GetProgressPath(workDir)
		if !git.FileModified(workDir, progressPath) {
			warnings = append(warnings, "Progress log not updated - consider logging your accomplishments")
		}
	}

	// Determine if stopping is allowed
	canStop := len(blockingReasons) == 0

	return canStop, blockingReasons, warnings
}

func handleStrictMode(canStop bool, blockingReasons, warnings []string) error {
	if !canStop {
		var messageParts []string
		messageParts = append(messageParts, "[Harness - STRICT MODE] Cannot stop due to:")
		for _, r := range blockingReasons {
			messageParts = append(messageParts, "  ! "+r)
		}

		if len(warnings) > 0 {
			messageParts = append(messageParts, "")
			messageParts = append(messageParts, "Additional reminders:")
			for _, w := range warnings {
				messageParts = append(messageParts, "  - "+w)
			}
		}

		return protocol.WriteDeny(strings.Join(messageParts, "\n"))
	}

	if len(warnings) > 0 {
		msg := "[Harness] Approved to stop.\n\nReminders:\n"
		for _, w := range warnings {
			msg += "  - " + w + "\n"
		}
		return protocol.WriteMessage(msg)
	}

	return protocol.WriteEmpty()
}

func handleStandardMode(blockingReasons, warnings []string) error {
	var messageParts []string

	if len(blockingReasons) > 0 {
		messageParts = append(messageParts, "[Harness] IMPORTANT - Before stopping:")
		for _, r := range blockingReasons {
			messageParts = append(messageParts, "  ! "+r)
		}
		messageParts = append(messageParts, "")
	}

	if len(warnings) > 0 {
		if len(messageParts) == 0 {
			messageParts = append(messageParts, "[Harness] Reminders before stopping:")
		} else {
			messageParts = append(messageParts, "Additional reminders:")
		}
		for _, w := range warnings {
			messageParts = append(messageParts, "  - "+w)
		}
	}

	if len(messageParts) > 0 {
		return protocol.WriteMessage(strings.Join(messageParts, "\n"))
	}
	return protocol.WriteEmpty()
}

func handleRelaxedMode(blockingReasons, warnings []string) error {
	allItems := append(blockingReasons, warnings...)
	if len(allItems) > 0 {
		return protocol.WriteMessage("[Harness] FYI: " + allItems[0])
	}
	return protocol.WriteEmpty()
}
