package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadContextState(t *testing.T) {
	t.Run("non-existent state returns new state", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "context-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		state, err := LoadContextState("session-123", tmpDir)
		if err != nil {
			t.Fatalf("LoadContextState() error = %v", err)
		}

		if state.SessionID != "session-123" {
			t.Errorf("SessionID = %v, want 'session-123'", state.SessionID)
		}
		if state.TotalTokenEstimate != 0 {
			t.Errorf("TotalTokenEstimate = %v, want 0", state.TotalTokenEstimate)
		}
	})

	t.Run("different session ID returns fresh state", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "context-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Save a state with one session ID
		oldState := &ContextState{
			SessionID:          "old-session",
			TotalTokenEstimate: 5000,
			EntryCount:         10,
		}
		oldState.Save(tmpDir)

		// Load with different session ID
		state, err := LoadContextState("new-session", tmpDir)
		if err != nil {
			t.Fatalf("LoadContextState() error = %v", err)
		}

		// Should get fresh state, not old one
		if state.SessionID != "new-session" {
			t.Errorf("SessionID = %v, want 'new-session'", state.SessionID)
		}
		if state.TotalTokenEstimate != 0 {
			t.Errorf("TotalTokenEstimate = %v, want 0 for new session", state.TotalTokenEstimate)
		}
	})

	t.Run("same session ID loads existing state", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "context-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Save a state
		originalState := &ContextState{
			SessionID:          "same-session",
			TotalTokenEstimate: 5000,
			EntryCount:         10,
		}
		originalState.Save(tmpDir)

		// Load with same session ID
		state, err := LoadContextState("same-session", tmpDir)
		if err != nil {
			t.Fatalf("LoadContextState() error = %v", err)
		}

		if state.SessionID != "same-session" {
			t.Errorf("SessionID = %v, want 'same-session'", state.SessionID)
		}
		if state.TotalTokenEstimate != 5000 {
			t.Errorf("TotalTokenEstimate = %v, want 5000", state.TotalTokenEstimate)
		}
		if state.EntryCount != 10 {
			t.Errorf("EntryCount = %v, want 10", state.EntryCount)
		}
	})
}

func TestContextStateSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "context-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	state := &ContextState{
		SessionID:          "test-session",
		TotalTokenEstimate: 1000,
		EntryCount:         5,
	}

	if err := state.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	statePath := filepath.Join(tmpDir, ".claude", ContextStateFileName)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("State file should be created")
	}

	// LastUpdated should be set
	if state.LastUpdated.IsZero() {
		t.Error("LastUpdated should be set after Save()")
	}
}

func TestAddEntry(t *testing.T) {
	state := &ContextState{
		SessionID: "test",
	}

	// Add an entry
	state.AddEntry("Read", "This is a 40 character result string....")

	if state.EntryCount != 1 {
		t.Errorf("EntryCount = %v, want 1", state.EntryCount)
	}

	// Token estimate should be roughly len/4
	expectedTokens := 40 / 4
	if state.TotalTokenEstimate != expectedTokens {
		t.Errorf("TotalTokenEstimate = %v, want %v", state.TotalTokenEstimate, expectedTokens)
	}

	// Utilization should be calculated
	expectedUtil := float64(expectedTokens) / float64(MaxContextTokens)
	if state.UtilizationPercent != expectedUtil {
		t.Errorf("UtilizationPercent = %v, want %v", state.UtilizationPercent, expectedUtil)
	}
}

func TestNeedsCompaction(t *testing.T) {
	tests := []struct {
		name       string
		util       float64
		threshold  float64
		wantResult bool
	}{
		{"below threshold", 0.50, 0.70, false},
		{"at threshold", 0.70, 0.70, true},
		{"above threshold", 0.80, 0.70, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ContextState{UtilizationPercent: tt.util}
			if got := state.NeedsCompaction(tt.threshold); got != tt.wantResult {
				t.Errorf("NeedsCompaction() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestGetUtilizationMessage(t *testing.T) {
	state := &ContextState{}
	// Currently returns empty string
	if msg := state.GetUtilizationMessage(); msg != "" {
		t.Errorf("GetUtilizationMessage() = %v, want empty", msg)
	}
}

func TestContextConstants(t *testing.T) {
	if ContextStateFileName != "fic-context-state.json" {
		t.Errorf("ContextStateFileName = %v, want 'fic-context-state.json'", ContextStateFileName)
	}
	if FilePermission != 0600 {
		t.Errorf("FilePermission = %o, want 0600", FilePermission)
	}
	if DirPermission != 0700 {
		t.Errorf("DirPermission = %o, want 0700", DirPermission)
	}
	if MaxContextTokens != 200000 {
		t.Errorf("MaxContextTokens = %v, want 200000", MaxContextTokens)
	}
}
