// Package artifacts manages FIC workflow artifacts (research, plan, implementation).
package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ArtifactType represents different FIC artifact types.
type ArtifactType string

const (
	ArtifactResearch       ArtifactType = "research"
	ArtifactPlan           ArtifactType = "plan"
	ArtifactImplementation ArtifactType = "implementation"
)

// ArtifactsDir is the directory where artifacts are stored.
const ArtifactsDir = ".claude/fic-artifacts"

// FilePermission for artifact files.
const FilePermission = 0600

// DirPermission for artifact directories.
const DirPermission = 0700

// Research represents a research artifact.
type Research struct {
	ID               string         `json:"id"`
	FeatureOrTask    string         `json:"feature_or_task"`
	ConfidenceScore  float64        `json:"confidence_score"`
	Discoveries      []Discovery    `json:"discoveries,omitempty"`
	OpenQuestions    []OpenQuestion `json:"open_questions,omitempty"`
	ResearchSessions int            `json:"research_sessions"`
	UpdatedAt        string         `json:"updated_at"`
}

// Discovery represents a research discovery.
type Discovery struct {
	Summary  string `json:"summary"`
	Critical bool   `json:"critical,omitempty"`
}

// OpenQuestion represents an open research question.
type OpenQuestion struct {
	Question string `json:"question"`
	Blocking bool   `json:"blocking,omitempty"`
}

// IsComplete returns true if research confidence is >= 70%.
func (r *Research) IsComplete() bool {
	return r.ConfidenceScore >= 0.7
}

// Plan represents a plan artifact.
type Plan struct {
	ID               string           `json:"id"`
	Goal             string           `json:"goal"`
	Steps            []PlanStep       `json:"steps,omitempty"`
	ValidationResult *ValidationResult `json:"validation_result,omitempty"`
	ResearchArtifactID string         `json:"research_artifact_id,omitempty"`
	UpdatedAt        string           `json:"updated_at"`
}

// PlanStep represents a step in a plan.
type PlanStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Completed   bool   `json:"completed,omitempty"`
}

// ValidationResult represents plan validation outcome.
type ValidationResult struct {
	Recommendation string `json:"recommendation"` // PROCEED, REVISE, BLOCK
	Score          int    `json:"score,omitempty"`
}

// IsActionable returns true if plan is validated for implementation.
func (p *Plan) IsActionable() bool {
	return p.ValidationResult != nil && p.ValidationResult.Recommendation == "PROCEED"
}

// Implementation represents an implementation artifact.
type Implementation struct {
	ID              string   `json:"id"`
	PlanArtifactID  string   `json:"plan_artifact_id"`
	StepsCompleted  []string `json:"steps_completed,omitempty"`
	StepsInProgress []string `json:"steps_in_progress,omitempty"`
	PlanDeviations  []string `json:"plan_deviations,omitempty"`
	UpdatedAt       string   `json:"updated_at"`
}

// GetArtifactDir returns the directory for a given artifact type.
func GetArtifactDir(workDir string, artifactType ArtifactType) string {
	return filepath.Join(workDir, ArtifactsDir, string(artifactType))
}

// GetLatestArtifact returns the most recent artifact of the given type.
func GetLatestArtifact(workDir string, artifactType ArtifactType) (interface{}, error) {
	dir := GetArtifactDir(workDir, artifactType)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Filter JSON files and sort by name (which includes timestamp)
	var jsonFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			jsonFiles = append(jsonFiles, entry.Name())
		}
	}

	if len(jsonFiles) == 0 {
		return nil, nil
	}

	// Sort descending to get latest first
	sort.Sort(sort.Reverse(sort.StringSlice(jsonFiles)))

	// Load the latest artifact
	latestPath := filepath.Join(dir, jsonFiles[0])
	data, err := os.ReadFile(latestPath)
	if err != nil {
		return nil, err
	}

	switch artifactType {
	case ArtifactResearch:
		var research Research
		if err := json.Unmarshal(data, &research); err != nil {
			return nil, err
		}
		return &research, nil

	case ArtifactPlan:
		var plan Plan
		if err := json.Unmarshal(data, &plan); err != nil {
			return nil, err
		}
		return &plan, nil

	case ArtifactImplementation:
		var impl Implementation
		if err := json.Unmarshal(data, &impl); err != nil {
			return nil, err
		}
		return &impl, nil
	}

	return nil, nil
}

// SaveArtifact saves an artifact to disk.
func SaveArtifact(workDir string, artifactType ArtifactType, artifact interface{}) error {
	dir := GetArtifactDir(workDir, artifactType)
	if err := os.MkdirAll(dir, DirPermission); err != nil {
		return err
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(dir, timestamp+".json")

	data, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, FilePermission)
}

// GetCurrentPhase determines the current FIC workflow phase.
func GetCurrentPhase(workDir string) string {
	impl, _ := GetLatestArtifact(workDir, ArtifactImplementation)
	if impl != nil {
		return "IMPLEMENTATION"
	}

	plan, _ := GetLatestArtifact(workDir, ArtifactPlan)
	if plan != nil {
		if p, ok := plan.(*Plan); ok && p.IsActionable() {
			return "IMPLEMENTATION_READY"
		}
		return "PLANNING"
	}

	research, _ := GetLatestArtifact(workDir, ArtifactResearch)
	if research != nil {
		if r, ok := research.(*Research); ok && r.IsComplete() {
			return "PLANNING_READY"
		}
		return "RESEARCH"
	}

	return "NEW_SESSION"
}

// GetPhaseInfo returns phase and details for context preservation.
func GetPhaseInfo(workDir string) map[string]interface{} {
	phase := GetCurrentPhase(workDir)
	info := map[string]interface{}{
		"phase":   phase,
		"details": map[string]interface{}{},
	}

	details := info["details"].(map[string]interface{})

	switch phase {
	case "IMPLEMENTATION":
		if impl, _ := GetLatestArtifact(workDir, ArtifactImplementation); impl != nil {
			if i, ok := impl.(*Implementation); ok {
				details["implementation_id"] = i.ID
				details["steps_completed"] = len(i.StepsCompleted)
				details["steps_in_progress"] = i.StepsInProgress
				details["plan_id"] = i.PlanArtifactID
			}
		}

	case "IMPLEMENTATION_READY", "PLANNING":
		if plan, _ := GetLatestArtifact(workDir, ArtifactPlan); plan != nil {
			if p, ok := plan.(*Plan); ok {
				details["plan_id"] = p.ID
				if len(p.Goal) > 100 {
					details["goal"] = p.Goal[:100]
				} else {
					details["goal"] = p.Goal
				}
				details["total_steps"] = len(p.Steps)
				details["is_validated"] = p.ValidationResult != nil
			}
		}

	case "PLANNING_READY", "RESEARCH":
		if research, _ := GetLatestArtifact(workDir, ArtifactResearch); research != nil {
			if r, ok := research.(*Research); ok {
				details["research_id"] = r.ID
				details["feature"] = r.FeatureOrTask
				details["confidence"] = r.ConfidenceScore
				details["discoveries"] = len(r.Discoveries)
				details["open_questions"] = len(r.OpenQuestions)
			}
		}
	}

	return info
}
