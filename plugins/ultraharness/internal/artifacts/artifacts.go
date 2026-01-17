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
// Enhanced to capture structured information for better compaction artifacts.
type Research struct {
	ID               string         `json:"id"`
	FeatureOrTask    string         `json:"feature_or_task"`
	ConfidenceScore  float64        `json:"confidence_score"`
	Discoveries      []Discovery    `json:"discoveries,omitempty"`
	OpenQuestions    []OpenQuestion `json:"open_questions,omitempty"`
	ResearchSessions int            `json:"research_sessions"`
	UpdatedAt        string         `json:"updated_at"`

	// Enhanced fields for better artifact quality
	CodebaseStructure   string        `json:"codebase_structure,omitempty"`   // High-level architecture understanding
	RelevantFiles       []FileContext `json:"relevant_files,omitempty"`       // Files with WHY they matter
	PotentialApproaches []Approach    `json:"potential_approaches,omitempty"` // Solution options considered
	Assumptions         []string      `json:"assumptions,omitempty"`          // Explicit assumptions made
}

// Discovery represents a research discovery.
type Discovery struct {
	Summary    string  `json:"summary"`
	Critical   bool    `json:"critical,omitempty"`
	Confidence float64 `json:"confidence,omitempty"` // Per-discovery confidence
	Source     string  `json:"source,omitempty"`     // file:line reference
}

// OpenQuestion represents an open research question.
type OpenQuestion struct {
	Question string `json:"question"`
	Blocking bool   `json:"blocking,omitempty"`
}

// FileContext captures why a file is relevant to the task.
type FileContext struct {
	Path     string   `json:"path"`
	Purpose  string   `json:"purpose"`    // Why this file matters for the task
	KeyAreas []string `json:"key_areas"`  // Specific functions/sections relevant
}

// Approach represents a potential solution approach.
type Approach struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Pros        []string `json:"pros,omitempty"`
	Cons        []string `json:"cons,omitempty"`
	Recommended bool     `json:"recommended,omitempty"`
}

// IsComplete returns true if research confidence is >= 70%.
func (r *Research) IsComplete() bool {
	return r.ConfidenceScore >= 0.7
}

// Plan represents a plan artifact.
// Enhanced to capture files to modify, verification steps, and rollback strategy.
type Plan struct {
	ID                 string             `json:"id"`
	Goal               string             `json:"goal"`
	Steps              []PlanStep         `json:"steps,omitempty"`
	ValidationResult   *ValidationResult  `json:"validation_result,omitempty"`
	ResearchArtifactID string             `json:"research_artifact_id,omitempty"`
	UpdatedAt          string             `json:"updated_at"`

	// Enhanced fields for better artifact quality
	FilesToModify     []FileModification `json:"files_to_modify,omitempty"`     // Exact files with change description
	VerificationSteps []Verification     `json:"verification_steps,omitempty"` // How to verify each phase
	Rollback          string             `json:"rollback,omitempty"`           // How to undo if needed
	ParallelBatches   []ParallelBatch    `json:"parallel_batches,omitempty"`   // For parallel execution
}

// PlanStep represents a step in a plan.
type PlanStep struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Completed    bool     `json:"completed,omitempty"`
	Files        []string `json:"files,omitempty"`        // Files this step touches
	Dependencies []string `json:"dependencies,omitempty"` // Step IDs this depends on
}

// ValidationResult represents plan validation outcome.
type ValidationResult struct {
	Recommendation string `json:"recommendation"` // PROCEED, REVISE, BLOCK
	Score          int    `json:"score,omitempty"`
	Feedback       string `json:"feedback,omitempty"` // Validator feedback
}

// FileModification describes a planned file change.
type FileModification struct {
	Path        string `json:"path"`
	ChangeType  string `json:"change_type"`  // "modify", "create", "delete"
	Description string `json:"description"` // What changes and why
}

// Verification describes how to verify a phase or step.
type Verification struct {
	Phase       string `json:"phase"`       // Which phase this verifies
	Method      string `json:"method"`      // "test", "manual", "build"
	Description string `json:"description"` // What to check
}

// ParallelBatch represents a group of tasks that can run in parallel.
type ParallelBatch struct {
	BatchNum int            `json:"batch_num"`
	Tasks    []ParallelTask `json:"tasks"`
}

// ParallelTask represents a task within a parallel batch.
type ParallelTask struct {
	StepID       string `json:"step_id"`
	Description  string `json:"description"`
	Scope        string `json:"scope"`        // File patterns this task affects
	Dependencies string `json:"dependencies"` // Dependencies from prior batches
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

// PhaseCheckpoint tracks human review at phase transitions.
// This enforces the pause-and-prompt model where human review is required
// at Research→Planning and Planning→Implementation transitions.
type PhaseCheckpoint struct {
	Phase          string `json:"phase"`           // "RESEARCH", "PLANNING"
	CompletedAt    string `json:"completed_at"`    // ISO timestamp
	HumanReviewed  bool   `json:"human_reviewed"`  // Set to true after user acknowledges
	ReviewResponse string `json:"review_response"` // User's feedback if any
	Summary        string `json:"summary"`         // Brief summary shown to user
}

// CheckpointState tracks pending checkpoints requiring human review.
type CheckpointState struct {
	PendingCheckpoint *PhaseCheckpoint `json:"pending_checkpoint,omitempty"`
	CompletedCheckpoints []PhaseCheckpoint `json:"completed_checkpoints,omitempty"`
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

// CheckpointStateFile is the name of the checkpoint state file.
const CheckpointStateFile = "fic-checkpoint-state.json"

// LoadCheckpointState loads the checkpoint state from disk.
func LoadCheckpointState(workDir string) (*CheckpointState, error) {
	statePath := filepath.Join(workDir, ".claude", CheckpointStateFile)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &CheckpointState{}, nil
		}
		return nil, err
	}

	var state CheckpointState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveCheckpointState saves the checkpoint state to disk.
func SaveCheckpointState(workDir string, state *CheckpointState) error {
	stateDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(stateDir, DirPermission); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	statePath := filepath.Join(stateDir, CheckpointStateFile)
	return os.WriteFile(statePath, data, FilePermission)
}

// SetPendingCheckpoint sets a pending checkpoint requiring human review.
func SetPendingCheckpoint(workDir string, phase string, summary string) error {
	state, err := LoadCheckpointState(workDir)
	if err != nil {
		state = &CheckpointState{}
	}

	state.PendingCheckpoint = &PhaseCheckpoint{
		Phase:         phase,
		CompletedAt:   time.Now().Format(time.RFC3339),
		HumanReviewed: false,
		Summary:       summary,
	}

	return SaveCheckpointState(workDir, state)
}

// ClearPendingCheckpoint clears the pending checkpoint after human review.
func ClearPendingCheckpoint(workDir string, reviewResponse string) error {
	state, err := LoadCheckpointState(workDir)
	if err != nil || state.PendingCheckpoint == nil {
		return nil
	}

	// Mark as reviewed and move to completed
	state.PendingCheckpoint.HumanReviewed = true
	state.PendingCheckpoint.ReviewResponse = reviewResponse
	state.CompletedCheckpoints = append(state.CompletedCheckpoints, *state.PendingCheckpoint)
	state.PendingCheckpoint = nil

	return SaveCheckpointState(workDir, state)
}

// HasPendingCheckpoint returns true if there's a checkpoint awaiting review.
func HasPendingCheckpoint(workDir string) bool {
	state, err := LoadCheckpointState(workDir)
	if err != nil {
		return false
	}
	return state.PendingCheckpoint != nil && !state.PendingCheckpoint.HumanReviewed
}

// GetPendingCheckpoint returns the pending checkpoint if any.
func GetPendingCheckpoint(workDir string) *PhaseCheckpoint {
	state, err := LoadCheckpointState(workDir)
	if err != nil {
		return nil
	}
	return state.PendingCheckpoint
}
