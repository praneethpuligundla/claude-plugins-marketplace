// Package protocol handles JSON stdin/stdout communication with Claude Code hooks.
// All hooks read input from stdin and write responses to stdout.
package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// MaxInputSize limits stdin to 10MB to prevent DoS attacks
const MaxInputSize = 10 * 1024 * 1024

// HookInput represents the JSON input from Claude Code to hooks
type HookInput struct {
	SessionID  string                 `json:"session_id"`
	ToolName   string                 `json:"tool_name"`
	ToolInput  map[string]interface{} `json:"tool_input"`
	ToolResult string                 `json:"tool_result,omitempty"`
}

// HookOutput represents the JSON output from hooks to Claude Code
type HookOutput struct {
	SystemMessage      string                 `json:"systemMessage,omitempty"`
	HookSpecificOutput *HookSpecificOutput    `json:"hookSpecificOutput,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// HookSpecificOutput contains hook-specific decisions
type HookSpecificOutput struct {
	PermissionDecision string `json:"permissionDecision,omitempty"` // "allow" or "deny"
}

// PermissionDecision constants
const (
	PermissionAllow = "allow"
	PermissionDeny  = "deny"
)

// ReadInput reads and parses JSON from stdin with size limiting
func ReadInput() (*HookInput, error) {
	reader := io.LimitReader(os.Stdin, MaxInputSize)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}

	// Handle empty input gracefully
	if len(data) == 0 {
		return &HookInput{}, nil
	}

	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &input, nil
}

// WriteOutput writes JSON response to stdout
func WriteOutput(output *HookOutput) error {
	data, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	_, err = os.Stdout.Write(data)
	return err
}

// WriteEmpty writes an empty JSON object {} to stdout
func WriteEmpty() error {
	_, err := os.Stdout.WriteString("{}")
	return err
}

// WriteError writes an error message as systemMessage
func WriteError(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return WriteOutput(&HookOutput{
		SystemMessage: fmt.Sprintf("[Harness] Hook error: %s", msg),
	})
}

// WriteDeny writes a permission denial response
func WriteDeny(message string) error {
	return WriteOutput(&HookOutput{
		SystemMessage: message,
		HookSpecificOutput: &HookSpecificOutput{
			PermissionDecision: PermissionDeny,
		},
	})
}

// WriteMessage writes a system message (informational, not blocking)
func WriteMessage(message string) error {
	return WriteOutput(&HookOutput{
		SystemMessage: message,
	})
}

// GetFilePath extracts file_path from tool input, returns empty string if not present
func (h *HookInput) GetFilePath() string {
	if h.ToolInput == nil {
		return ""
	}
	if path, ok := h.ToolInput["file_path"].(string); ok {
		return path
	}
	return ""
}

// GetCommand extracts command from tool input (for Bash), returns empty string if not present
func (h *HookInput) GetCommand() string {
	if h.ToolInput == nil {
		return ""
	}
	if cmd, ok := h.ToolInput["command"].(string); ok {
		return cmd
	}
	return ""
}

// GetPrompt extracts prompt from tool input (for UserPromptSubmit), returns empty string if not present
func (h *HookInput) GetPrompt() string {
	if h.ToolInput == nil {
		return ""
	}
	if prompt, ok := h.ToolInput["prompt"].(string); ok {
		return prompt
	}
	return ""
}

// GetSubagentType extracts subagent_type from tool input (for SubagentStop), returns empty string if not present
func (h *HookInput) GetSubagentType() string {
	if h.ToolInput == nil {
		return ""
	}
	if t, ok := h.ToolInput["subagent_type"].(string); ok {
		return t
	}
	return ""
}

// GetDescription extracts description from tool input (for SubagentStop), returns empty string if not present
func (h *HookInput) GetDescription() string {
	if h.ToolInput == nil {
		return ""
	}
	if d, ok := h.ToolInput["description"].(string); ok {
		return d
	}
	return ""
}

// GetOutput extracts output from tool input (for SubagentStop), returns empty string if not present
func (h *HookInput) GetOutput() string {
	if h.ToolInput == nil {
		return ""
	}
	if o, ok := h.ToolInput["output"].(string); ok {
		return o
	}
	return ""
}

// GetStopReason extracts stopReason or reason from tool input (for Stop), returns empty string if not present
func (h *HookInput) GetStopReason() string {
	if h.ToolInput == nil {
		return ""
	}
	if r, ok := h.ToolInput["stopReason"].(string); ok {
		return r
	}
	if r, ok := h.ToolInput["reason"].(string); ok {
		return r
	}
	return ""
}

// WriteSystemMessage writes a system message response (alias for WriteMessage for clarity)
func WriteSystemMessage(message string) error {
	return WriteMessage(message)
}
