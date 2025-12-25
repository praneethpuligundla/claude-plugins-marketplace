// PostToolUse hook handles context tracking, change detection, and progress logging.
//
// This hook runs after Edit, Write, Bash, Read, Grep, Glob, and Task tools to:
// 1. Track context utilization with weighted tool estimates
// 2. Warn when context is filling up (50%+)
// 3. Trigger compaction directive when critical (70%+)
// 4. Auto-log significant changes
// 5. Suggest checkpoints after major changes
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

// Default thresholds if not configured
const (
	DefaultToolCountWarning  = 30  // Warn after 30 tool calls
	DefaultToolCountCritical = 50  // Critical after 50 tool calls
	DefaultUtilizationWarn   = 0.5 // 50% utilization warning
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
			// If compaction is needed, return immediately with high priority
			if strings.Contains(msg, "CRITICAL") || strings.Contains(msg, "ACTION REQUIRED") {
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
	if err := state.Save(workDir); err != nil {
		// Continue even if save fails
	}

	// Get thresholds from config
	autoCompactThreshold := cfg.GetAutoCompactThreshold()
	compactionToolThreshold := cfg.GetCompactionToolThreshold()
	if compactionToolThreshold == 0 {
		compactionToolThreshold = DefaultToolCountCritical
	}
	autoCompactEnabled := cfg.IsAutoCompactEnabled()

	// Check for CRITICAL: auto-compaction needed (token-based)
	if state.NeedsCompaction(autoCompactThreshold) {
		if autoCompactEnabled {
			return buildAutoCompactDirective(state, "utilization", autoCompactThreshold)
		}
		return buildCompactionDirective(state, autoCompactThreshold)
	}

	// Check for CRITICAL: tool count exceeded
	if state.NeedsCompactionByToolCount(compactionToolThreshold) {
		if autoCompactEnabled {
			return buildAutoCompactDirective(state, "tool_count", float64(compactionToolThreshold))
		}
		return buildToolCountDirective(state, compactionToolThreshold)
	}

	// Check for WARNING: approaching limits
	warningToolCount := compactionToolThreshold * 2 / 3 // ~67% of critical
	if state.TotalToolCalls >= warningToolCount || state.UtilizationPercent >= DefaultUtilizationWarn {
		return buildWarningMessage(state, compactionToolThreshold)
	}

	// Periodic status update every 10 tool calls
	if state.TotalToolCalls > 0 && state.TotalToolCalls%10 == 0 {
		return fmt.Sprintf("[FIC] %s", state.GetSummary())
	}

	return ""
}

func buildAutoCompactDirective(state *context.ContextState, reason string, threshold float64) string {
	var triggerInfo string
	if reason == "utilization" {
		triggerInfo = fmt.Sprintf("Context utilization: %.0f%% (threshold: %.0f%%)", state.UtilizationPercent*100, threshold*100)
	} else {
		triggerInfo = fmt.Sprintf("Tool calls: %d (threshold: %.0f)", state.TotalToolCalls, threshold)
	}

	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] AUTO-COMPACTION TRIGGERED                                             ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                              ║
║  %s
║  %s
║                                                                              ║
║  Compactions so far: %d
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

MANDATORY: You MUST run /compact NOW before doing anything else.

This is an automated compaction trigger. The context window is filling up.
Running /compact will:
1. Preserve essential context (phase, discoveries, progress)
2. Reset the context window for continued work
3. Maintain your current focus and task

Execute: /compact`,
		triggerInfo,
		state.GetSummary(),
		state.CompactionCount)
}

func buildCompactionDirective(state *context.ContextState, threshold float64) string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] CRITICAL: CONTEXT UTILIZATION AT %.0f%%                                 ║
║  LONG-RUNNING SESSION - COMPACTION REQUIRED                                   ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                                ║
║  %s
║                                                                                ║
║  Threshold: %.0f%% | Compactions so far: %d
║                                                                                ║
║  ACTION REQUIRED: Run /compact NOW before continuing.                         ║
║                                                                                ║
║  Context is filling up. Compacting now preserves essential discoveries        ║
║  and prevents context overflow and degraded performance.                      ║
║                                                                                ║
╚══════════════════════════════════════════════════════════════════════════════╝

STOP current work. Run /compact immediately.
The PreCompact hook will preserve essential context automatically.`,
		state.UtilizationPercent*100,
		state.GetSummary(),
		threshold*100,
		state.CompactionCount)
}

func buildToolCountDirective(state *context.ContextState, maxTools int) string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════════════════╗
║  [FIC] CRITICAL: %d TOOL CALLS - COMPACTION RECOMMENDED                       ║
╠══════════════════════════════════════════════════════════════════════════════╣
║                                                                                ║
║  %s
║                                                                                ║
║  Tool limit: %d | Compactions so far: %d
║                                                                                ║
║  ACTION REQUIRED: Consider running /compact to free up context space.         ║
║                                                                                ║
║  High tool count indicates a long-running session. Compacting preserves       ║
║  essential context and improves response quality.                             ║
║                                                                                ║
╚══════════════════════════════════════════════════════════════════════════════╝`,
		state.TotalToolCalls,
		state.GetSummary(),
		maxTools,
		state.CompactionCount)
}

func buildWarningMessage(state *context.ContextState, maxTools int) string {
	remaining := maxTools - state.TotalToolCalls
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf("[FIC] Context filling: %.0f%% util, %d/%d tool calls. ~%d calls until compaction recommended.",
		state.UtilizationPercent*100,
		state.TotalToolCalls,
		maxTools,
		remaining)
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
