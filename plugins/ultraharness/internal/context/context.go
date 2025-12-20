// Package context handles FIC context intelligence tracking.
// Tracks context utilization and detects when compaction is needed.
package context

import (
	"encoding/json"
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

// ContextState tracks context utilization
type ContextState struct {
	SessionID            string    `json:"session_id"`
	TotalTokenEstimate   int       `json:"total_token_estimate"`
	UtilizationPercent   float64   `json:"utilization_percent"`
	EntryCount           int       `json:"entry_count"`
	RedundantDiscoveries []string  `json:"redundant_discoveries,omitempty"`
	LastUpdated          time.Time `json:"last_updated"`
}

// MaxContextTokens is the assumed maximum context size
const MaxContextTokens = 200000

// LoadContextState loads the context state from the working directory
func LoadContextState(sessionID, workDir string) (*ContextState, error) {
	statePath := filepath.Join(workDir, ".claude", ContextStateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ContextState{
				SessionID:   sessionID,
				LastUpdated: time.Now(),
			}, nil
		}
		return nil, err
	}

	var state ContextState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// If session ID doesn't match, return fresh state
	if state.SessionID != sessionID {
		return &ContextState{
			SessionID:   sessionID,
			LastUpdated: time.Now(),
		}, nil
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

	// Estimate tokens (roughly 4 chars per token)
	resultTokens := len(toolResult) / 4
	s.TotalTokenEstimate += resultTokens

	// Update utilization
	s.UtilizationPercent = float64(s.TotalTokenEstimate) / float64(MaxContextTokens)

	// Return warning if any
	return ""
}

// NeedsCompaction returns true if context utilization is above threshold
func (s *ContextState) NeedsCompaction(threshold float64) bool {
	return s.UtilizationPercent >= threshold
}

// GetUtilizationMessage returns a human-readable utilization message
func (s *ContextState) GetUtilizationMessage() string {
	return ""
}
