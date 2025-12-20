package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFICState(t *testing.T) {
	t.Run("non-existent state returns default", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		state, err := LoadFICState(tmpDir)
		if err != nil {
			t.Fatalf("LoadFICState() error = %v", err)
		}

		if state.Phase != "research" {
			t.Errorf("Phase = %v, want 'research'", state.Phase)
		}
		if state.ResearchComplete {
			t.Error("ResearchComplete should be false by default")
		}
		if state.PlanValidated {
			t.Error("PlanValidated should be false by default")
		}
	})

	t.Run("load existing state", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .claude directory and state file
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		state := &FICState{
			Phase:            "implementation",
			ResearchComplete: true,
			PlanValidated:    true,
		}

		data, err := json.Marshal(state)
		if err != nil {
			t.Fatalf("Failed to marshal state: %v", err)
		}

		statePath := filepath.Join(claudeDir, FICStateFileName)
		if err := os.WriteFile(statePath, data, 0644); err != nil {
			t.Fatalf("Failed to write state file: %v", err)
		}

		loadedState, err := LoadFICState(tmpDir)
		if err != nil {
			t.Fatalf("LoadFICState() error = %v", err)
		}

		if loadedState.Phase != "implementation" {
			t.Errorf("Phase = %v, want 'implementation'", loadedState.Phase)
		}
		if !loadedState.ResearchComplete {
			t.Error("ResearchComplete should be true")
		}
		if !loadedState.PlanValidated {
			t.Error("PlanValidated should be true")
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		statePath := filepath.Join(claudeDir, FICStateFileName)
		if err := os.WriteFile(statePath, []byte("invalid json{"), 0644); err != nil {
			t.Fatalf("Failed to write state file: %v", err)
		}

		_, err = LoadFICState(tmpDir)
		if err == nil {
			t.Error("LoadFICState() should return error for invalid JSON")
		}
	})
}

func TestCheckGate(t *testing.T) {
	t.Run("relaxed mode always allows", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := CheckGate(GateAllowEdit, tmpDir, "relaxed")
		if result.Action != ActionAllow {
			t.Errorf("Action = %v, want %v", result.Action, ActionAllow)
		}
	})

	t.Run("standard mode warns when research not complete", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := CheckGate(GateAllowEdit, tmpDir, "standard")
		if result.Action != ActionWarn {
			t.Errorf("Action = %v, want %v", result.Action, ActionWarn)
		}
		if result.Reason != "Research phase not complete" {
			t.Errorf("Reason = %v, want 'Research phase not complete'", result.Reason)
		}
	})

	t.Run("strict mode blocks when research not complete", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := CheckGate(GateAllowEdit, tmpDir, "strict")
		if result.Action != ActionBlock {
			t.Errorf("Action = %v, want %v", result.Action, ActionBlock)
		}
	})

	t.Run("standard mode warns when plan not validated", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create state with research complete but plan not validated
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		state := &FICState{
			Phase:            "planning",
			ResearchComplete: true,
			PlanValidated:    false,
		}

		data, _ := json.Marshal(state)
		statePath := filepath.Join(claudeDir, FICStateFileName)
		os.WriteFile(statePath, data, 0644)

		result := CheckGate(GateAllowEdit, tmpDir, "standard")
		if result.Action != ActionWarn {
			t.Errorf("Action = %v, want %v", result.Action, ActionWarn)
		}
		if result.Reason != "Planning phase not complete" {
			t.Errorf("Reason = %v, want 'Planning phase not complete'", result.Reason)
		}
	})

	t.Run("allows when all gates pass", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create state with all gates passed
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		state := &FICState{
			Phase:            "implementation",
			ResearchComplete: true,
			PlanValidated:    true,
		}

		data, _ := json.Marshal(state)
		statePath := filepath.Join(claudeDir, FICStateFileName)
		os.WriteFile(statePath, data, 0644)

		result := CheckGate(GateAllowEdit, tmpDir, "standard")
		if result.Action != ActionAllow {
			t.Errorf("Action = %v, want %v", result.Action, ActionAllow)
		}
	})

	t.Run("bash gate always allows", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := CheckGate(GateAllowBash, tmpDir, "strict")
		if result.Action != ActionAllow {
			t.Errorf("Bash gate Action = %v, want %v", result.Action, ActionAllow)
		}
	})

	t.Run("unknown gate allows", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gates-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := CheckGate("unknown_gate", tmpDir, "strict")
		if result.Action != ActionAllow {
			t.Errorf("Unknown gate Action = %v, want %v", result.Action, ActionAllow)
		}
	})
}

func TestFormatGateMessage(t *testing.T) {
	t.Run("allow returns empty", func(t *testing.T) {
		result := &GateResult{Action: ActionAllow}
		if msg := FormatGateMessage(result); msg != "" {
			t.Errorf("FormatGateMessage() = %v, want empty", msg)
		}
	})

	t.Run("warn formats correctly", func(t *testing.T) {
		result := &GateResult{
			Action:      ActionWarn,
			Reason:      "Research not complete",
			Suggestions: []string{"Do more research"},
		}
		msg := FormatGateMessage(result)
		if msg == "" {
			t.Error("FormatGateMessage() should not be empty for warn")
		}
		// Check it contains key elements
		if !contains(msg, "warn") && !contains(msg, "Research not complete") {
			t.Errorf("FormatGateMessage() missing expected content: %v", msg)
		}
	})

	t.Run("block formats correctly", func(t *testing.T) {
		result := &GateResult{
			Action: ActionBlock,
			Reason: "Plan not validated",
		}
		msg := FormatGateMessage(result)
		if msg == "" {
			t.Error("FormatGateMessage() should not be empty for block")
		}
	})
}

func TestGateConstants(t *testing.T) {
	if GateAllowEdit != "allow_edit" {
		t.Errorf("GateAllowEdit = %v, want 'allow_edit'", GateAllowEdit)
	}
	if GateAllowWrite != "allow_write" {
		t.Errorf("GateAllowWrite = %v, want 'allow_write'", GateAllowWrite)
	}
	if GateAllowBash != "allow_bash" {
		t.Errorf("GateAllowBash = %v, want 'allow_bash'", GateAllowBash)
	}
	if FICStateFileName != "fic-state.json" {
		t.Errorf("FICStateFileName = %v, want 'fic-state.json'", FICStateFileName)
	}
}

func TestActionConstants(t *testing.T) {
	if ActionAllow != "allow" {
		t.Errorf("ActionAllow = %v, want 'allow'", ActionAllow)
	}
	if ActionWarn != "warn" {
		t.Errorf("ActionWarn = %v, want 'warn'", ActionWarn)
	}
	if ActionBlock != "block" {
		t.Errorf("ActionBlock = %v, want 'block'", ActionBlock)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
