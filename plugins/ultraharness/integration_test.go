// Package main contains integration tests for the FIC workflow system.
// These tests verify the complete workflow: research → planning → implementation
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ultraharness/internal/context"
	"ultraharness/internal/gates"
	"ultraharness/internal/progress"
)

// TestFICWorkflowCycle tests the complete FIC workflow from research to implementation
func TestFICWorkflowCycle(t *testing.T) {
	// Create test workspace
	workDir, err := os.MkdirTemp("", "fic-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	claudeDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	t.Run("Phase 1: Research - edits should be blocked/warned", func(t *testing.T) {
		// Default state is research phase
		result := gates.CheckGate(gates.GateAllowEdit, workDir, "standard")
		if result.Action != gates.ActionWarn {
			t.Errorf("Research phase: Action = %v, want %v", result.Action, gates.ActionWarn)
		}
		if result.Reason != "Research phase not complete" {
			t.Errorf("Research phase: wrong reason: %s", result.Reason)
		}

		// Strict mode should block
		result = gates.CheckGate(gates.GateAllowEdit, workDir, "strict")
		if result.Action != gates.ActionBlock {
			t.Errorf("Research phase strict: Action = %v, want %v", result.Action, gates.ActionBlock)
		}

		// Relaxed mode should allow
		result = gates.CheckGate(gates.GateAllowEdit, workDir, "relaxed")
		if result.Action != gates.ActionAllow {
			t.Errorf("Research phase relaxed: Action = %v, want %v", result.Action, gates.ActionAllow)
		}
	})

	t.Run("Phase 2: Planning - transition from research", func(t *testing.T) {
		// Complete research phase
		state := &gates.FICState{
			Phase:            "planning",
			ResearchComplete: true,
			PlanValidated:    false,
			LastUpdated:      time.Now(),
		}
		saveFICState(t, workDir, state)

		// Edit should still warn (plan not validated)
		result := gates.CheckGate(gates.GateAllowEdit, workDir, "standard")
		if result.Action != gates.ActionWarn {
			t.Errorf("Planning phase: Action = %v, want %v", result.Action, gates.ActionWarn)
		}
		if result.Reason != "Planning phase not complete" {
			t.Errorf("Planning phase: wrong reason: %s", result.Reason)
		}
	})

	t.Run("Phase 3: Implementation - all gates pass", func(t *testing.T) {
		// Complete planning phase
		state := &gates.FICState{
			Phase:            "implementation",
			ResearchComplete: true,
			PlanValidated:    true,
			LastUpdated:      time.Now(),
		}
		saveFICState(t, workDir, state)

		// Edit should now be allowed
		result := gates.CheckGate(gates.GateAllowEdit, workDir, "standard")
		if result.Action != gates.ActionAllow {
			t.Errorf("Implementation phase: Action = %v, want %v", result.Action, gates.ActionAllow)
		}

		// Write should also be allowed
		result = gates.CheckGate(gates.GateAllowWrite, workDir, "standard")
		if result.Action != gates.ActionAllow {
			t.Errorf("Implementation phase write: Action = %v, want %v", result.Action, gates.ActionAllow)
		}

		// Even strict mode should allow
		result = gates.CheckGate(gates.GateAllowEdit, workDir, "strict")
		if result.Action != gates.ActionAllow {
			t.Errorf("Implementation phase strict: Action = %v, want %v", result.Action, gates.ActionAllow)
		}
	})
}

// TestContextTrackingAcrossWorkflow tests context utilization tracking during workflow
func TestContextTrackingAcrossWorkflow(t *testing.T) {
	workDir, err := os.MkdirTemp("", "context-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	claudeDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create .claude dir: %v", err)
	}

	sessionID := "test-session-123"

	t.Run("Context starts fresh", func(t *testing.T) {
		ctx, err := context.LoadContextState(sessionID, workDir)
		if err != nil {
			t.Fatalf("LoadContextState error: %v", err)
		}

		if ctx.SessionID != sessionID {
			t.Errorf("SessionID = %v, want %v", ctx.SessionID, sessionID)
		}
		if ctx.EntryCount != 0 {
			t.Errorf("EntryCount = %v, want 0", ctx.EntryCount)
		}
		if ctx.TotalTokenEstimate != 0 {
			t.Errorf("TotalTokenEstimate = %v, want 0", ctx.TotalTokenEstimate)
		}
	})

	t.Run("Context accumulates during research", func(t *testing.T) {
		ctx, _ := context.LoadContextState(sessionID, workDir)

		// Simulate tool use results
		ctx.AddEntry("Read", string(make([]byte, 4000))) // ~1000 tokens
		ctx.AddEntry("Grep", string(make([]byte, 2000))) // ~500 tokens
		ctx.AddEntry("Glob", string(make([]byte, 800)))  // ~200 tokens

		if ctx.EntryCount != 3 {
			t.Errorf("EntryCount = %v, want 3", ctx.EntryCount)
		}

		// Save and reload to verify persistence
		if err := ctx.Save(workDir); err != nil {
			t.Fatalf("Save error: %v", err)
		}

		reloaded, err := context.LoadContextState(sessionID, workDir)
		if err != nil {
			t.Fatalf("Reload error: %v", err)
		}

		if reloaded.EntryCount != 3 {
			t.Errorf("Reloaded EntryCount = %v, want 3", reloaded.EntryCount)
		}
	})

	t.Run("Different session ID resets state", func(t *testing.T) {
		newSessionID := "new-session-456"
		ctx, err := context.LoadContextState(newSessionID, workDir)
		if err != nil {
			t.Fatalf("LoadContextState error: %v", err)
		}

		// Should start fresh for new session
		if ctx.SessionID != newSessionID {
			t.Errorf("New session ID = %v, want %v", ctx.SessionID, newSessionID)
		}
		if ctx.EntryCount != 0 {
			t.Errorf("New session EntryCount = %v, want 0", ctx.EntryCount)
		}
	})

	t.Run("Compaction threshold detection", func(t *testing.T) {
		ctx := &context.ContextState{
			SessionID:          sessionID,
			TotalTokenEstimate: 160000, // 80% of 200k
			UtilizationPercent: 0.8,
		}

		if !ctx.NeedsCompaction(0.75) {
			t.Error("Should need compaction at 80% when threshold is 75%")
		}
		if ctx.NeedsCompaction(0.85) {
			t.Error("Should not need compaction at 80% when threshold is 85%")
		}
	})
}

// TestProgressLogIntegration tests progress logging during workflow
func TestProgressLogIntegration(t *testing.T) {
	workDir, err := os.MkdirTemp("", "progress-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Create progress file
	progressPath := filepath.Join(workDir, progress.ProgressFileName)
	initialContent := "# Claude Agent Progress Log\n# Project: test\n"
	if err := os.WriteFile(progressPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create progress file: %v", err)
	}
	_ = progressPath // Used to create the file

	t.Run("Log entries across workflow phases", func(t *testing.T) {
		// Log research activity
		progress.Append("RESEARCH: Explored codebase structure", workDir)
		progress.Append("RESEARCH: Identified key files", workDir)

		// Log planning activity
		progress.Append("PLAN: Created implementation strategy", workDir)

		// Log implementation activity
		progress.Append("IMPLEMENT: Created new component", workDir)
		progress.Append("IMPLEMENT: Added tests", workDir)

		// Verify entries by reading file
		content, err := progress.Read(workDir)
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}

		// Check that all phases are logged
		if !containsStr(content, "RESEARCH") {
			t.Error("Progress should contain RESEARCH entries")
		}
		if !containsStr(content, "PLAN") {
			t.Error("Progress should contain PLAN entries")
		}
		if !containsStr(content, "IMPLEMENT") {
			t.Error("Progress should contain IMPLEMENT entries")
		}
	})
}

// TestStrictnessLevelsAcrossPhases tests all strictness levels at each phase
func TestStrictnessLevelsAcrossPhases(t *testing.T) {
	workDir, err := os.MkdirTemp("", "strictness-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	claudeDir := filepath.Join(workDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	phases := []struct {
		name             string
		state            *gates.FICState
		relaxedAction    gates.GateAction
		standardAction   gates.GateAction
		strictAction     gates.GateAction
	}{
		{
			name: "research",
			state: &gates.FICState{
				Phase:            "research",
				ResearchComplete: false,
				PlanValidated:    false,
			},
			relaxedAction:  gates.ActionAllow,
			standardAction: gates.ActionWarn,
			strictAction:   gates.ActionBlock,
		},
		{
			name: "planning",
			state: &gates.FICState{
				Phase:            "planning",
				ResearchComplete: true,
				PlanValidated:    false,
			},
			relaxedAction:  gates.ActionAllow,
			standardAction: gates.ActionWarn,
			strictAction:   gates.ActionBlock,
		},
		{
			name: "implementation",
			state: &gates.FICState{
				Phase:            "implementation",
				ResearchComplete: true,
				PlanValidated:    true,
			},
			relaxedAction:  gates.ActionAllow,
			standardAction: gates.ActionAllow,
			strictAction:   gates.ActionAllow,
		},
	}

	for _, phase := range phases {
		t.Run(phase.name, func(t *testing.T) {
			saveFICState(t, workDir, phase.state)

			// Test relaxed
			result := gates.CheckGate(gates.GateAllowEdit, workDir, "relaxed")
			if result.Action != phase.relaxedAction {
				t.Errorf("%s relaxed: got %v, want %v", phase.name, result.Action, phase.relaxedAction)
			}

			// Test standard
			result = gates.CheckGate(gates.GateAllowEdit, workDir, "standard")
			if result.Action != phase.standardAction {
				t.Errorf("%s standard: got %v, want %v", phase.name, result.Action, phase.standardAction)
			}

			// Test strict
			result = gates.CheckGate(gates.GateAllowEdit, workDir, "strict")
			if result.Action != phase.strictAction {
				t.Errorf("%s strict: got %v, want %v", phase.name, result.Action, phase.strictAction)
			}
		})
	}
}

// TestSessionPersistence tests that state persists correctly between sessions
func TestSessionPersistence(t *testing.T) {
	workDir, err := os.MkdirTemp("", "persistence-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	claudeDir := filepath.Join(workDir, ".claude")
	os.MkdirAll(claudeDir, 0755)

	t.Run("FIC state persists", func(t *testing.T) {
		// Session 1: Set state
		state1 := &gates.FICState{
			Phase:            "planning",
			ResearchComplete: true,
			PlanValidated:    false,
			LastUpdated:      time.Now(),
		}
		saveFICState(t, workDir, state1)

		// Session 2: Load state
		loaded, err := gates.LoadFICState(workDir)
		if err != nil {
			t.Fatalf("LoadFICState error: %v", err)
		}

		if loaded.Phase != "planning" {
			t.Errorf("Phase = %v, want planning", loaded.Phase)
		}
		if !loaded.ResearchComplete {
			t.Error("ResearchComplete should be true")
		}
		if loaded.PlanValidated {
			t.Error("PlanValidated should be false")
		}
	})

	t.Run("Context state persists for same session", func(t *testing.T) {
		sessionID := "persistent-session"

		// First access
		ctx1, _ := context.LoadContextState(sessionID, workDir)
		ctx1.AddEntry("Read", "test content")
		ctx1.Save(workDir)

		// Second access (same session)
		ctx2, _ := context.LoadContextState(sessionID, workDir)
		if ctx2.EntryCount != 1 {
			t.Errorf("EntryCount = %v, want 1", ctx2.EntryCount)
		}
	})
}

// TestGateMessageFormatting tests that gate messages are formatted correctly
func TestGateMessageFormatting(t *testing.T) {
	workDir, err := os.MkdirTemp("", "message-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	os.MkdirAll(filepath.Join(workDir, ".claude"), 0755)

	t.Run("Warn message includes suggestions", func(t *testing.T) {
		result := gates.CheckGate(gates.GateAllowEdit, workDir, "standard")
		msg := gates.FormatGateMessage(result)

		if msg == "" {
			t.Error("Message should not be empty for warn")
		}

		// Should contain useful info
		if !containsStr(msg, "Research") {
			t.Errorf("Message should mention research: %s", msg)
		}
	})

	t.Run("Allow message is empty", func(t *testing.T) {
		result := gates.CheckGate(gates.GateAllowEdit, workDir, "relaxed")
		msg := gates.FormatGateMessage(result)

		if msg != "" {
			t.Errorf("Allow message should be empty, got: %s", msg)
		}
	})
}

// Helper functions

func saveFICState(t *testing.T, workDir string, state *gates.FICState) {
	t.Helper()
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	statePath := filepath.Join(workDir, ".claude", gates.FICStateFileName)
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
