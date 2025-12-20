package artifacts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResearchIsComplete(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		want       bool
	}{
		{"below threshold", 0.50, false},
		{"at threshold", 0.70, true},
		{"above threshold", 0.90, true},
		{"zero", 0.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Research{ConfidenceScore: tt.confidence}
			if got := r.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlanIsActionable(t *testing.T) {
	tests := []struct {
		name   string
		plan   *Plan
		want   bool
	}{
		{
			name: "nil validation result",
			plan: &Plan{ValidationResult: nil},
			want: false,
		},
		{
			name: "REVISE recommendation",
			plan: &Plan{ValidationResult: &ValidationResult{Recommendation: "REVISE"}},
			want: false,
		},
		{
			name: "BLOCK recommendation",
			plan: &Plan{ValidationResult: &ValidationResult{Recommendation: "BLOCK"}},
			want: false,
		},
		{
			name: "PROCEED recommendation",
			plan: &Plan{ValidationResult: &ValidationResult{Recommendation: "PROCEED"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.plan.IsActionable(); got != tt.want {
				t.Errorf("IsActionable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetArtifactDir(t *testing.T) {
	workDir := "/test/project"

	tests := []struct {
		artifactType ArtifactType
		want         string
	}{
		{ArtifactResearch, "/test/project/.claude/fic-artifacts/research"},
		{ArtifactPlan, "/test/project/.claude/fic-artifacts/plan"},
		{ArtifactImplementation, "/test/project/.claude/fic-artifacts/implementation"},
	}

	for _, tt := range tests {
		t.Run(string(tt.artifactType), func(t *testing.T) {
			if got := GetArtifactDir(workDir, tt.artifactType); got != tt.want {
				t.Errorf("GetArtifactDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLatestArtifact(t *testing.T) {
	t.Run("non-existent directory returns nil", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		artifact, err := GetLatestArtifact(tmpDir, ArtifactResearch)
		if err != nil {
			t.Fatalf("GetLatestArtifact() error = %v", err)
		}
		if artifact != nil {
			t.Error("Expected nil artifact for non-existent directory")
		}
	})

	t.Run("empty directory returns nil", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create empty artifact directory
		artifactDir := GetArtifactDir(tmpDir, ArtifactResearch)
		if err := os.MkdirAll(artifactDir, DirPermission); err != nil {
			t.Fatalf("Failed to create artifact dir: %v", err)
		}

		artifact, err := GetLatestArtifact(tmpDir, ArtifactResearch)
		if err != nil {
			t.Fatalf("GetLatestArtifact() error = %v", err)
		}
		if artifact != nil {
			t.Error("Expected nil artifact for empty directory")
		}
	})

	t.Run("loads research artifact", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create artifact
		research := &Research{
			ID:              "research-1",
			FeatureOrTask:   "Test feature",
			ConfidenceScore: 0.85,
		}

		if err := SaveArtifact(tmpDir, ArtifactResearch, research); err != nil {
			t.Fatalf("SaveArtifact() error = %v", err)
		}

		artifact, err := GetLatestArtifact(tmpDir, ArtifactResearch)
		if err != nil {
			t.Fatalf("GetLatestArtifact() error = %v", err)
		}

		loaded, ok := artifact.(*Research)
		if !ok {
			t.Fatal("Expected *Research type")
		}
		if loaded.ID != "research-1" {
			t.Errorf("ID = %v, want 'research-1'", loaded.ID)
		}
		if loaded.ConfidenceScore != 0.85 {
			t.Errorf("ConfidenceScore = %v, want 0.85", loaded.ConfidenceScore)
		}
	})

	t.Run("loads plan artifact", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		plan := &Plan{
			ID:   "plan-1",
			Goal: "Implement feature X",
			Steps: []PlanStep{
				{ID: "step-1", Description: "First step"},
				{ID: "step-2", Description: "Second step"},
			},
		}

		if err := SaveArtifact(tmpDir, ArtifactPlan, plan); err != nil {
			t.Fatalf("SaveArtifact() error = %v", err)
		}

		artifact, err := GetLatestArtifact(tmpDir, ArtifactPlan)
		if err != nil {
			t.Fatalf("GetLatestArtifact() error = %v", err)
		}

		loaded, ok := artifact.(*Plan)
		if !ok {
			t.Fatal("Expected *Plan type")
		}
		if loaded.ID != "plan-1" {
			t.Errorf("ID = %v, want 'plan-1'", loaded.ID)
		}
		if len(loaded.Steps) != 2 {
			t.Errorf("Steps count = %d, want 2", len(loaded.Steps))
		}
	})

	t.Run("loads implementation artifact", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		impl := &Implementation{
			ID:             "impl-1",
			PlanArtifactID: "plan-1",
			StepsCompleted: []string{"step-1"},
		}

		if err := SaveArtifact(tmpDir, ArtifactImplementation, impl); err != nil {
			t.Fatalf("SaveArtifact() error = %v", err)
		}

		artifact, err := GetLatestArtifact(tmpDir, ArtifactImplementation)
		if err != nil {
			t.Fatalf("GetLatestArtifact() error = %v", err)
		}

		loaded, ok := artifact.(*Implementation)
		if !ok {
			t.Fatal("Expected *Implementation type")
		}
		if loaded.ID != "impl-1" {
			t.Errorf("ID = %v, want 'impl-1'", loaded.ID)
		}
	})
}

func TestSaveArtifact(t *testing.T) {
	t.Run("creates directory if not exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		research := &Research{ID: "test"}

		if err := SaveArtifact(tmpDir, ArtifactResearch, research); err != nil {
			t.Fatalf("SaveArtifact() error = %v", err)
		}

		artifactDir := GetArtifactDir(tmpDir, ArtifactResearch)
		if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
			t.Error("Artifact directory should be created")
		}
	})

	t.Run("file has correct permissions", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "artifacts-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		research := &Research{ID: "test"}
		if err := SaveArtifact(tmpDir, ArtifactResearch, research); err != nil {
			t.Fatalf("SaveArtifact() error = %v", err)
		}

		artifactDir := GetArtifactDir(tmpDir, ArtifactResearch)
		entries, _ := os.ReadDir(artifactDir)
		if len(entries) == 0 {
			t.Fatal("No artifact file created")
		}

		filePath := filepath.Join(artifactDir, entries[0].Name())
		info, _ := os.Stat(filePath)
		perm := info.Mode().Perm()

		// Check file is readable/writable by owner only (0600)
		if perm != FilePermission {
			t.Errorf("File permission = %o, want %o", perm, FilePermission)
		}
	})
}

func TestGetCurrentPhase(t *testing.T) {
	t.Run("new session", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		phase := GetCurrentPhase(tmpDir)
		if phase != "NEW_SESSION" {
			t.Errorf("Phase = %v, want 'NEW_SESSION'", phase)
		}
	})

	t.Run("research phase", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		research := &Research{ID: "r1", ConfidenceScore: 0.5}
		SaveArtifact(tmpDir, ArtifactResearch, research)

		phase := GetCurrentPhase(tmpDir)
		if phase != "RESEARCH" {
			t.Errorf("Phase = %v, want 'RESEARCH'", phase)
		}
	})

	t.Run("planning ready", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		research := &Research{ID: "r1", ConfidenceScore: 0.8}
		SaveArtifact(tmpDir, ArtifactResearch, research)

		phase := GetCurrentPhase(tmpDir)
		if phase != "PLANNING_READY" {
			t.Errorf("Phase = %v, want 'PLANNING_READY'", phase)
		}
	})

	t.Run("planning phase", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		plan := &Plan{ID: "p1", ValidationResult: nil}
		SaveArtifact(tmpDir, ArtifactPlan, plan)

		phase := GetCurrentPhase(tmpDir)
		if phase != "PLANNING" {
			t.Errorf("Phase = %v, want 'PLANNING'", phase)
		}
	})

	t.Run("implementation ready", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		plan := &Plan{
			ID:               "p1",
			ValidationResult: &ValidationResult{Recommendation: "PROCEED"},
		}
		SaveArtifact(tmpDir, ArtifactPlan, plan)

		phase := GetCurrentPhase(tmpDir)
		if phase != "IMPLEMENTATION_READY" {
			t.Errorf("Phase = %v, want 'IMPLEMENTATION_READY'", phase)
		}
	})

	t.Run("implementation phase", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		impl := &Implementation{ID: "i1", PlanArtifactID: "p1"}
		SaveArtifact(tmpDir, ArtifactImplementation, impl)

		phase := GetCurrentPhase(tmpDir)
		if phase != "IMPLEMENTATION" {
			t.Errorf("Phase = %v, want 'IMPLEMENTATION'", phase)
		}
	})
}

func TestGetPhaseInfo(t *testing.T) {
	t.Run("returns phase and details", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "phase-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		info := GetPhaseInfo(tmpDir)

		phase, ok := info["phase"].(string)
		if !ok {
			t.Error("phase should be a string")
		}
		if phase != "NEW_SESSION" {
			t.Errorf("phase = %v, want 'NEW_SESSION'", phase)
		}

		if _, ok := info["details"].(map[string]interface{}); !ok {
			t.Error("details should be a map")
		}
	})
}

func TestArtifactTypeConstants(t *testing.T) {
	if ArtifactResearch != "research" {
		t.Errorf("ArtifactResearch = %v, want 'research'", ArtifactResearch)
	}
	if ArtifactPlan != "plan" {
		t.Errorf("ArtifactPlan = %v, want 'plan'", ArtifactPlan)
	}
	if ArtifactImplementation != "implementation" {
		t.Errorf("ArtifactImplementation = %v, want 'implementation'", ArtifactImplementation)
	}
	if ArtifactsDir != ".claude/fic-artifacts" {
		t.Errorf("ArtifactsDir = %v, want '.claude/fic-artifacts'", ArtifactsDir)
	}
}
