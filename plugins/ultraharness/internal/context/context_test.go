package context

import (
	"os"
	"path/filepath"
	"strings"
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

	t.Run("different session ID persists state (no reset)", func(t *testing.T) {
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
			TotalToolCalls:     10,
		}
		oldState.Save(tmpDir)

		// Load with different session ID
		state, err := LoadContextState("new-session", tmpDir)
		if err != nil {
			t.Fatalf("LoadContextState() error = %v", err)
		}

		// Session ID should change but data should persist
		if state.SessionID != "new-session" {
			t.Errorf("SessionID = %v, want 'new-session'", state.SessionID)
		}
		if state.LastSessionID != "old-session" {
			t.Errorf("LastSessionID = %v, want 'old-session'", state.LastSessionID)
		}
		// Context state should persist, not reset
		if state.TotalTokenEstimate != 5000 {
			t.Errorf("TotalTokenEstimate = %v, want 5000 (persisted)", state.TotalTokenEstimate)
		}
		if state.TotalToolCalls != 10 {
			t.Errorf("TotalToolCalls = %v, want 10 (persisted)", state.TotalToolCalls)
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
	t.Run("tracks tool calls by type", func(t *testing.T) {
		state := &ContextState{SessionID: "test"}

		state.AddEntry("Read", "file content here")
		state.AddEntry("Read", "more content")
		state.AddEntry("Grep", "search results")
		state.AddEntry("Edit", "edit result")

		if state.TotalToolCalls != 4 {
			t.Errorf("TotalToolCalls = %v, want 4", state.TotalToolCalls)
		}
		if state.ToolCalls.Read != 2 {
			t.Errorf("ToolCalls.Read = %v, want 2", state.ToolCalls.Read)
		}
		if state.ToolCalls.Grep != 1 {
			t.Errorf("ToolCalls.Grep = %v, want 1", state.ToolCalls.Grep)
		}
		if state.ToolCalls.Edit != 1 {
			t.Errorf("ToolCalls.Edit = %v, want 1", state.ToolCalls.Edit)
		}
	})

	t.Run("uses weighted token estimates", func(t *testing.T) {
		state := &ContextState{SessionID: "test"}

		// Read has weight 1500 + base overhead 400 = 1900 min
		state.AddEntry("Read", "short")

		// Should use weighted estimate (1500) not just len/4 (1)
		if state.TotalTokenEstimate < 1500 {
			t.Errorf("TotalTokenEstimate = %v, want >= 1500 (weighted)", state.TotalTokenEstimate)
		}
	})

	t.Run("uses actual result size when larger than weight", func(t *testing.T) {
		state := &ContextState{SessionID: "test"}

		// Create a large result (8000 chars = ~2000 tokens, larger than Read weight of 1500)
		largeResult := strings.Repeat("x", 8000)
		state.AddEntry("Read", largeResult)

		// Should use actual size (2000) + overhead, not weight (1500)
		if state.TotalTokenEstimate < 2000 {
			t.Errorf("TotalTokenEstimate = %v, want >= 2000 (actual result size)", state.TotalTokenEstimate)
		}
	})

	t.Run("calculates utilization correctly", func(t *testing.T) {
		state := &ContextState{SessionID: "test"}

		// Add many entries to accumulate tokens
		for i := 0; i < 50; i++ {
			state.AddEntry("Read", "content")
		}

		// Should have accumulated significant utilization
		if state.UtilizationPercent <= 0 {
			t.Errorf("UtilizationPercent = %v, want > 0", state.UtilizationPercent)
		}
		// Check reasonable bounds
		if state.UtilizationPercent > 1.0 {
			t.Errorf("UtilizationPercent = %v, should not exceed 1.0", state.UtilizationPercent)
		}
	})
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

func TestNeedsCompactionByToolCount(t *testing.T) {
	tests := []struct {
		name       string
		toolCalls  int
		maxTools   int
		wantResult bool
	}{
		{"below limit", 30, 50, false},
		{"at limit", 50, 50, true},
		{"above limit", 60, 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ContextState{TotalToolCalls: tt.toolCalls}
			if got := state.NeedsCompactionByToolCount(tt.maxTools); got != tt.wantResult {
				t.Errorf("NeedsCompactionByToolCount() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestGetUtilizationMessage(t *testing.T) {
	t.Run("low utilization returns empty", func(t *testing.T) {
		state := &ContextState{UtilizationPercent: 0.2}
		if msg := state.GetUtilizationMessage(); msg != "" {
			t.Errorf("GetUtilizationMessage() = %v, want empty for low util", msg)
		}
	})

	t.Run("moderate utilization returns message", func(t *testing.T) {
		state := &ContextState{
			UtilizationPercent: 0.4,
			TotalTokenEstimate: 80000,
			TotalToolCalls:     30,
		}
		msg := state.GetUtilizationMessage()
		if msg == "" {
			t.Error("GetUtilizationMessage() should return message for 40% util")
		}
		if !strings.Contains(msg, "moderate") {
			t.Errorf("Message should contain 'moderate', got: %v", msg)
		}
	})

	t.Run("high utilization returns warning", func(t *testing.T) {
		state := &ContextState{
			UtilizationPercent: 0.6,
			TotalTokenEstimate: 120000,
			TotalToolCalls:     45,
		}
		msg := state.GetUtilizationMessage()
		if !strings.Contains(msg, "high") {
			t.Errorf("Message should contain 'high', got: %v", msg)
		}
	})

	t.Run("critical utilization returns critical warning", func(t *testing.T) {
		state := &ContextState{
			UtilizationPercent: 0.75,
			TotalTokenEstimate: 150000,
			TotalToolCalls:     55,
		}
		msg := state.GetUtilizationMessage()
		if !strings.Contains(msg, "CRITICAL") {
			t.Errorf("Message should contain 'CRITICAL', got: %v", msg)
		}
	})
}

func TestReset(t *testing.T) {
	state := &ContextState{
		SessionID:          "old-session",
		TotalTokenEstimate: 100000,
		TotalToolCalls:     50,
		EntryCount:         50,
		ToolCalls: ToolCallsByType{
			Read: 20,
			Edit: 10,
			Bash: 20,
		},
		CompactionCount: 2,
	}

	state.Reset("new-session")

	if state.SessionID != "new-session" {
		t.Errorf("SessionID = %v, want 'new-session'", state.SessionID)
	}
	if state.TotalTokenEstimate != 0 {
		t.Errorf("TotalTokenEstimate = %v, want 0", state.TotalTokenEstimate)
	}
	if state.TotalToolCalls != 0 {
		t.Errorf("TotalToolCalls = %v, want 0", state.TotalToolCalls)
	}
	if state.CompactionCount != 3 {
		t.Errorf("CompactionCount = %v, want 3 (incremented)", state.CompactionCount)
	}
	if state.ToolCalls.Read != 0 {
		t.Errorf("ToolCalls.Read = %v, want 0", state.ToolCalls.Read)
	}
}

func TestGetSummary(t *testing.T) {
	state := &ContextState{
		TotalToolCalls:     25,
		TotalTokenEstimate: 50000,
		UtilizationPercent: 0.25,
		ToolCalls: ToolCallsByType{
			Read:  10,
			Grep:  5,
			Glob:  2,
			Edit:  3,
			Write: 1,
			Bash:  3,
			Task:  1,
		},
	}

	summary := state.GetSummary()

	if !strings.Contains(summary, "25") {
		t.Errorf("Summary should contain tool count '25', got: %v", summary)
	}
	if !strings.Contains(summary, "50k") {
		t.Errorf("Summary should contain '50k' tokens, got: %v", summary)
	}
	if !strings.Contains(summary, "25%") {
		t.Errorf("Summary should contain '25%%' utilization, got: %v", summary)
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

func TestToolWeights(t *testing.T) {
	// Verify weights are set for common tools
	expectedWeights := map[string]int{
		"Read":  1500,
		"Grep":  800,
		"Glob":  300,
		"Task":  2500,
		"Edit":  600,
		"Write": 500,
		"Bash":  700,
	}

	for tool, expectedWeight := range expectedWeights {
		if toolWeights[tool] != expectedWeight {
			t.Errorf("toolWeights[%s] = %v, want %v", tool, toolWeights[tool], expectedWeight)
		}
	}
}
