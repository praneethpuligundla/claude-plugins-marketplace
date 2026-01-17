// Package context handles FIC context tracking.
//
// Philosophy: Simple tool call counting, not complex token estimation.
// The goal is to track session progress, not predict context overflow.
// Compaction should be proactive (at phase completion), not reactive (at token limits).
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

// ToolCallsByType tracks tool usage by type (simple counting)
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

// ContextState tracks session progress (simplified - no token estimation)
type ContextState struct {
	// Session tracking
	SessionID       string    `json:"session_id"`
	SessionStarted  time.Time `json:"session_started"`
	LastSessionID   string    `json:"last_session_id,omitempty"`
	CompactionCount int       `json:"compaction_count"`

	// Simple tool tracking (counts only, no estimation)
	ToolCalls      ToolCallsByType `json:"tool_calls"`
	TotalToolCalls int             `json:"total_tool_calls"`

	// Legacy fields (kept for backward compatibility, no longer used)
	TotalTokenEstimate   int       `json:"total_token_estimate,omitempty"`
	UtilizationPercent   float64   `json:"utilization_percent,omitempty"`
	EntryCount           int       `json:"entry_count,omitempty"`
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

// AddEntry updates context tracking for a tool use (simple counting)
func (s *ContextState) AddEntry(toolName string, toolResult string) string {
	s.TotalToolCalls++

	// Track by tool type (simple counting)
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

	return ""
}

// NeedsCompaction is kept for backward compatibility but is no longer recommended.
// Philosophy: Compaction should be proactive (at phase completion), not reactive.
func (s *ContextState) NeedsCompaction(threshold float64) bool {
	// Legacy: always returns false now since we don't track utilization
	return false
}

// NeedsCompactionByToolCount is kept for backward compatibility.
// Philosophy: Tool count alone is not a good indicator of when to compact.
func (s *ContextState) NeedsCompactionByToolCount(maxTools int) bool {
	return s.TotalToolCalls >= maxTools
}

// Reset clears the context state after compaction
func (s *ContextState) Reset(sessionID string) {
	s.CompactionCount++
	s.SessionID = sessionID
	s.SessionStarted = time.Now()
	s.ToolCalls = ToolCallsByType{}
	s.TotalToolCalls = 0
	s.LastUpdated = time.Now()
}

// GetSummary returns a summary of session progress
func (s *ContextState) GetSummary() string {
	return fmt.Sprintf("Tool calls: %d (Read:%d, Grep:%d, Glob:%d, Edit:%d, Write:%d, Bash:%d, Task:%d) | Compactions: %d",
		s.TotalToolCalls,
		s.ToolCalls.Read, s.ToolCalls.Grep, s.ToolCalls.Glob,
		s.ToolCalls.Edit, s.ToolCalls.Write, s.ToolCalls.Bash, s.ToolCalls.Task,
		s.CompactionCount)
}
