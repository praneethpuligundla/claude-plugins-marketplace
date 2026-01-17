// PostToolUse hook handles context tracking, change detection, and progress logging.
//
// This hook runs after Edit, Write, Bash, Read, Grep, Glob, and Task tools to:
// 1. Track tool call counts (simple, non-intrusive)
// 2. Auto-log significant changes
// 3. Provide periodic status updates
//
// Philosophy: Compaction is a PROACTIVE quality tool, not an emergency measure.
// This hook provides information but does NOT demand or trigger compaction.
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

	// Context tracking (non-intrusive)
	if cfg.FICEnabled && cfg.FICContextTracking {
		msg := trackContext(input, workDir)
		if msg != "" {
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

// trackContext provides non-intrusive context tracking.
// It tracks tool usage and provides periodic status updates without demanding compaction.
func trackContext(input *protocol.HookInput, workDir string) string {
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
	if err := state.Save(workDir); err != nil {
		// Continue even if save fails
	}

	// Periodic status update every 15 tool calls (non-intrusive)
	if state.TotalToolCalls > 0 && state.TotalToolCalls%15 == 0 {
		return fmt.Sprintf("[FIC] Session progress: %d tool calls | Compactions: %d",
			state.TotalToolCalls, state.CompactionCount)
	}

	return ""
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
		strings.Contains(result, "test result: ok") || strings.Contains(result, "ok  \t")
	hasFailed := strings.Contains(result, "failed") || strings.Contains(result, "FAILED") ||
		strings.Contains(result, "FAIL") || strings.Contains(result, "Error:")

	if hasPassed && !hasFailed {
		return "[FIC] Tests passed! Implementation verification gate satisfied."
	}
	if hasFailed {
		return "[FIC] Tests failed. Review failures before continuing."
	}

	return ""
}
