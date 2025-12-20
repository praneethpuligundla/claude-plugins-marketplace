// PostToolUse hook handles context tracking, change detection, and progress logging.
//
// This hook runs after Edit, Write, Bash, Read, Grep, Glob, and Task tools to:
// 1. Track context utilization
// 2. Trigger auto-compaction when utilization >= 70%
// 3. Auto-log significant changes
// 4. Suggest checkpoints after major changes
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ultraharness/internal/config"
	"ultraharness/internal/context"
	"ultraharness/internal/progress"
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

	// Read input from stdin
	input, err := protocol.ReadInput()
	if err != nil {
		return protocol.WriteEmpty()
	}

	var messages []string

	// Context intelligence tracking
	if cfg.FICEnabled && cfg.FICContextTracking {
		msg := trackContext(input, workDir, cfg)
		if msg != "" {
			// If compaction is needed, return immediately
			if strings.Contains(msg, "CRITICAL") {
				return protocol.WriteMessage(msg)
			}
			messages = append(messages, msg)
		}
	}

	// Skip further processing in relaxed mode
	if cfg.IsRelaxedMode() {
		if len(messages) > 0 {
			return protocol.WriteMessage(strings.Join(messages, "\n"))
		}
		return protocol.WriteEmpty()
	}

	// Only track progress for file modifications
	toolName := input.ToolName
	if toolName != "Edit" && toolName != "Write" && toolName != "Bash" {
		if len(messages) > 0 {
			return protocol.WriteMessage(strings.Join(messages, "\n"))
		}
		return protocol.WriteEmpty()
	}

	// Classify change and auto-log
	if cfg.AutoProgressLogging {
		logEntry := classifyAndLog(toolName, input, workDir)
		if logEntry != "" {
			messages = append(messages, logEntry)
		}
	}

	// Check for test results in Bash output
	if toolName == "Bash" {
		testMsg := checkTestResults(input.ToolResult)
		if testMsg != "" {
			messages = append(messages, testMsg)
		}
	}

	// Output result
	if len(messages) > 0 {
		return protocol.WriteMessage(strings.Join(messages, "\n"))
	}
	return protocol.WriteEmpty()
}

func trackContext(input *protocol.HookInput, workDir string, cfg *config.Config) string {
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// Validate session ID
	if err := validation.ValidateSessionID(sessionID); err != nil {
		sessionID = "default"
	}

	state, err := context.LoadContextState(sessionID, workDir)
	if err != nil {
		return ""
	}

	// Add this tool use to context tracking
	state.AddEntry(input.ToolName, input.ToolResult)

	// Save updated state
	state.Save(workDir)

	// Check for auto-compaction threshold
	threshold := cfg.GetAutoCompactThreshold()
	if state.NeedsCompaction(threshold) {
		return buildCompactionDirective(state.UtilizationPercent, state.TotalTokenEstimate, threshold)
	}

	// Check soft threshold
	compactionToolThreshold := cfg.GetCompactionToolThreshold()
	if state.EntryCount >= compactionToolThreshold && state.UtilizationPercent >= 0.60 {
		return fmt.Sprintf("[FIC] Context utilization at %.0f%%. Consider compacting or using subagents for research.",
			state.UtilizationPercent*100)
	}

	return ""
}

func buildCompactionDirective(utilization float64, tokenEstimate int, threshold float64) string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════╗
║  [FIC] CRITICAL: CONTEXT UTILIZATION AT %.0f%%                     ║
║  LONG-RUNNING SESSION - AUTO-COMPACTION REQUIRED                  ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                    ║
║  Estimated tokens: %d
║  Threshold: %.0f%%
║                                                                    ║
║  ACTION REQUIRED: Run /compact NOW before continuing.             ║
║                                                                    ║
║  Context is filling up during autonomous operation.               ║
║  Compacting now preserves essential discoveries and prevents      ║
║  context overflow.                                                 ║
║                                                                    ║
╚══════════════════════════════════════════════════════════════════╝

STOP current work. Run /compact immediately.
The PreCompact hook will preserve essential context automatically.`,
		utilization*100, tokenEstimate, threshold*100)
}

func classifyAndLog(toolName string, input *protocol.HookInput, workDir string) string {
	// Classify change level based on tool and file
	filePath := input.GetFilePath()
	if filePath == "" && toolName != "Bash" {
		return ""
	}

	// Determine if significant
	isSignificant := false
	var reason string

	switch toolName {
	case "Write":
		isSignificant = true
		reason = "new file created"
	case "Edit":
		// Large edits are significant
		if len(input.ToolResult) > 500 {
			isSignificant = true
			reason = "substantial edit"
		}
	case "Bash":
		cmd := input.GetCommand()
		// Test commands, builds, deployments are significant
		if strings.Contains(cmd, "test") || strings.Contains(cmd, "build") ||
			strings.Contains(cmd, "deploy") || strings.Contains(cmd, "npm") ||
			strings.Contains(cmd, "cargo") || strings.Contains(cmd, "go build") {
			isSignificant = true
			reason = "build/test command"
		}
	}

	if !isSignificant {
		return ""
	}

	// Format log entry
	var logEntry string
	switch toolName {
	case "Write":
		filename := filepath.Base(filePath)
		logEntry = fmt.Sprintf("AUTO: Created %s (%s)", filename, reason)
	case "Edit":
		filename := filepath.Base(filePath)
		logEntry = fmt.Sprintf("AUTO: Modified %s (%s)", filename, reason)
	case "Bash":
		cmd := input.GetCommand()
		if len(cmd) > 40 {
			cmd = cmd[:40] + "..."
		}
		logEntry = fmt.Sprintf("AUTO: Ran '%s' (%s)", cmd, reason)
	}

	// Append to progress file (ignore errors)
	progress.Append(logEntry, workDir)

	return ""
}

func checkTestResults(result string) string {
	if result == "" {
		return ""
	}

	// Check for test result indicators
	hasPassed := strings.Contains(result, "passed") || strings.Contains(result, "PASSED") ||
		strings.Contains(result, "test result: ok")
	hasFailed := strings.Contains(result, "failed") || strings.Contains(result, "FAILED") ||
		strings.Contains(result, "FAIL") || strings.Contains(result, "error")

	if hasPassed && !hasFailed {
		return "[FIC] Tests passed! Implementation verification gate satisfied."
	}
	if hasFailed {
		return "[FIC] Tests failed. Review failures before continuing."
	}

	return ""
}
