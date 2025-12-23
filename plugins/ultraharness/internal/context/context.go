// Package context handles FIC context intelligence tracking.
// Tracks context utilization and detects when compaction is needed.
//
// Context tracking uses weighted tool call counts as a proxy for actual
// context utilization, since hooks cannot access the real context window.
package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ContextStateFileName is the name of the context state file
const ContextStateFileName = "fic-context-state.json"

// FilePermission is the permission for state files
const FilePermission = 0600

// DirPermission is the permission for state directories
const DirPermission = 0700

// MaxContextTokens is the assumed maximum context size
const MaxContextTokens = 200000

// Tool token weights - estimated average tokens per tool use
// These are conservative estimates including tool input, output, and response overhead
var toolWeights = map[string]int{
	"Read":  1500, // Large file reads
	"Grep":  800,  // Search results
	"Glob":  300,  // File listings
	"Task":  2500, // Subagent responses are large
	"Edit":  600,  // Edit context + result
	"Write": 500,  // Write content + confirmation
	"Bash":  700,  // Command + output
}

// BaseOverhead is tokens added per tool call for conversation structure
const BaseOverhead = 400

// ConversationMultiplier accounts for conversation history accumulation
// As conversation grows, each new message includes more history context
const ConversationMultiplier = 1.15

// ToolCallsByType tracks tool usage by type
type ToolCallsByType struct {
	Read  int `json:"read"`
	Grep  int `json:"grep"`
	Glob  int `json:"glob"`
	Task  int `json:"task"`
	Edit  int `json:"edit"`
	Write int `json:"write"`
	Bash  int `json:"bash"`
	Other int `json:"other"`
}

// ContextState tracks context utilization
type ContextState struct {
	// Session tracking - now persists across sessions
	SessionID       string    `json:"session_id"`
	SessionStarted  time.Time `json:"session_started"`
	LastSessionID   string    `json:"last_session_id,omitempty"`
	CompactionCount int       `json:"compaction_count"`

	// Tool tracking
	ToolCalls      ToolCallsByType `json:"tool_calls"`
	TotalToolCalls int             `json:"total_tool_calls"`

	// Token estimation
	TotalTokenEstimate int     `json:"total_token_estimate"`
	UtilizationPercent float64 `json:"utilization_percent"`

	// Legacy fields for compatibility
	EntryCount           int       `json:"entry_count"`
	RedundantDiscoveries []string  `json:"redundant_discoveries,omitempty"`
	LastUpdated          time.Time `json:"last_updated"`
}

// LoadContextState loads the context state from the working directory.
// Unlike before, this now PERSISTS state across sessions instead of resetting.
func LoadContextState(sessionID, workDir string) (*ContextState, error) {
	statePath := filepath.Join(workDir, ".claude", ContextStateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ContextState{
				SessionID:      sessionID,
				SessionStarted: time.Now(),
				LastUpdated:    time.Now(),
			}, nil
		}
		return nil, err
	}

	var state ContextState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// If session ID changed, track it but DON'T reset
	// A new session continues accumulating context
	if state.SessionID != sessionID {
		state.LastSessionID = state.SessionID
		state.SessionID = sessionID
		// Don't reset - context persists across sessions until compaction
	}

	return &state, nil
}

// Save writes the context state to disk
func (s *ContextState) Save(workDir string) error {
	stateDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(stateDir, DirPermission); err != nil {
		return err
	}

	s.LastUpdated = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(stateDir, ContextStateFileName)
	return os.WriteFile(statePath, data, FilePermission)
}

// AddEntry updates context tracking for a tool use
func (s *ContextState) AddEntry(toolName string, toolResult string) string {
	s.EntryCount++
	s.TotalToolCalls++

	// Track by tool type
	switch toolName {
	case "Read":
		s.ToolCalls.Read++
	case "Grep":
		s.ToolCalls.Grep++
	case "Glob":
		s.ToolCalls.Glob++
	case "Task":
		s.ToolCalls.Task++
	case "Edit":
		s.ToolCalls.Edit++
	case "Write":
		s.ToolCalls.Write++
	case "Bash":
		s.ToolCalls.Bash++
	default:
		s.ToolCalls.Other++
	}

	// Calculate token estimate with weights
	weight := toolWeights[toolName]
	if weight == 0 {
		weight = 500 // Default for unknown tools
	}

	// Add base overhead + weighted tool tokens
	toolTokens := BaseOverhead + weight

	// For tools with output, also consider actual result size
	if len(toolResult) > 0 {
		resultTokens := len(toolResult) / 4
		// Use the larger of weight estimate or actual result
		if resultTokens > weight {
			toolTokens = BaseOverhead + resultTokens
		}
	}

	// Apply conversation multiplier based on depth
	// Context grows non-linearly as conversation accumulates
	depthMultiplier := 1.0
	if s.TotalToolCalls > 10 {
		depthMultiplier = 1.0 + (float64(s.TotalToolCalls-10) * 0.01) // +1% per call after 10
		if depthMultiplier > ConversationMultiplier {
			depthMultiplier = ConversationMultiplier
		}
	}

	s.TotalTokenEstimate += int(float64(toolTokens) * depthMultiplier)

	// Update utilization
	s.UtilizationPercent = float64(s.TotalTokenEstimate) / float64(MaxContextTokens)

	return ""
}

// NeedsCompaction returns true if context utilization is above threshold
func (s *ContextState) NeedsCompaction(threshold float64) bool {
	return s.UtilizationPercent >= threshold
}

// NeedsCompactionByToolCount returns true if tool count exceeds limit
// This is a more reliable heuristic than token estimation
func (s *ContextState) NeedsCompactionByToolCount(maxTools int) bool {
	return s.TotalToolCalls >= maxTools
}

// GetUtilizationMessage returns a human-readable utilization message
func (s *ContextState) GetUtilizationMessage() string {
	if s.UtilizationPercent < 0.3 {
		return ""
	}

	level := "moderate"
	if s.UtilizationPercent >= 0.7 {
		level = "CRITICAL"
	} else if s.UtilizationPercent >= 0.5 {
		level = "high"
	}

	return fmt.Sprintf("[FIC] Context %s: %.0f%% (~%dk tokens, %d tool calls)",
		level,
		s.UtilizationPercent*100,
		s.TotalTokenEstimate/1000,
		s.TotalToolCalls)
}

// Reset clears the context state after compaction
func (s *ContextState) Reset(sessionID string) {
	s.CompactionCount++
	s.SessionID = sessionID
	s.SessionStarted = time.Now()
	s.ToolCalls = ToolCallsByType{}
	s.TotalToolCalls = 0
	s.TotalTokenEstimate = 0
	s.UtilizationPercent = 0
	s.EntryCount = 0
	s.RedundantDiscoveries = nil
	s.LastUpdated = time.Now()
}

// GetSummary returns a summary of context usage
func (s *ContextState) GetSummary() string {
	return fmt.Sprintf("Tool calls: %d (Read:%d, Grep:%d, Glob:%d, Edit:%d, Write:%d, Bash:%d, Task:%d) | Est. tokens: %dk | Util: %.0f%%",
		s.TotalToolCalls,
		s.ToolCalls.Read, s.ToolCalls.Grep, s.ToolCalls.Glob,
		s.ToolCalls.Edit, s.ToolCalls.Write, s.ToolCalls.Bash, s.ToolCalls.Task,
		s.TotalTokenEstimate/1000,
		s.UtilizationPercent*100)
}
