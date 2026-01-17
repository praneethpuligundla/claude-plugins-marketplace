// Package config handles loading and accessing harness configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"ultraharness/internal/validation"
)

// ConfigFileName is the name of the config file
const ConfigFileName = "claude-harness.json"

// InitMarkerFileName is the marker file that indicates harness is initialized
const InitMarkerFileName = ".claude-harness-initialized"

// Strictness levels
const (
	StrictnessRelaxed  = "relaxed"
	StrictnessStandard = "standard"
	StrictnessStrict   = "strict"
)

// Config represents the harness configuration
type Config struct {
	Strictness               string     `json:"strictness"`
	FICEnabled               bool       `json:"fic_enabled"`
	FICContextTracking       bool       `json:"fic_context_tracking"`
	FICAutoDelegateResearch  bool       `json:"fic_auto_delegate_research"`
	AutoProgressLogging      bool       `json:"auto_progress_logging"`
	AutoCheckpointSuggestions bool      `json:"auto_checkpoint_suggestions"`
	CheckpointIntervalMinutes int       `json:"checkpoint_interval_minutes"`
	FeatureEnforcement       bool       `json:"feature_enforcement"`
	InitScriptExecution      bool       `json:"init_script_execution"`
	BaselineTestsOnStartup   bool       `json:"baseline_tests_on_startup"`
	FICConfig                *FICConfig `json:"fic_config,omitempty"`
}

// FICConfig contains FIC-specific configuration
type FICConfig struct {
	// Context utilization thresholds (target 40-60%)
	// Philosophy: Compaction is a PROACTIVE quality tool, not an emergency measure
	TargetUtilizationLow  float64 `json:"target_utilization_low"`  // Ideal lower bound (40%)
	TargetUtilizationHigh float64 `json:"target_utilization_high"` // Ideal upper bound (60%)
	SuggestCompactAt      float64 `json:"suggest_compact_at"`      // Soft suggestion point (55%)

	// Legacy fields (kept for backward compatibility, no longer drive emergency behavior)
	AutoCompactThreshold    float64 `json:"auto_compact_threshold,omitempty"`
	CompactionToolThreshold int     `json:"compaction_tool_threshold,omitempty"`
	AutoCompactEnabled      bool    `json:"auto_compact_enabled,omitempty"`

	// Research phase thresholds
	ResearchConfidenceThreshold float64 `json:"research_confidence_threshold"`
	MaxOpenQuestions            int     `json:"max_open_questions"`

	// Gate behavior customization
	WarnOnResearchIncomplete bool `json:"warn_on_research_incomplete"`
	WarnOnPlanIncomplete     bool `json:"warn_on_plan_incomplete"`
	BlockInStrictMode        bool `json:"block_in_strict_mode"`

	// Phase checkpoint behavior
	RequireCheckpointReview bool `json:"require_checkpoint_review"` // Pause at phase transitions

	// Parallel implementation settings
	ParallelImplementationEnabled bool `json:"parallel_implementation_enabled"`
	MaxParallelAgents             int  `json:"max_parallel_agents"`
	MinStepsForParallel           int  `json:"min_steps_for_parallel"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Strictness:                StrictnessStandard,
		FICEnabled:                true,
		FICContextTracking:        true,
		FICAutoDelegateResearch:   true,
		AutoProgressLogging:       true,
		AutoCheckpointSuggestions: true,
		CheckpointIntervalMinutes: 30,
		FeatureEnforcement:        true,
		InitScriptExecution:       true,
		BaselineTestsOnStartup:    true,
		FICConfig: &FICConfig{
			// Target utilization range: 40-60% (per ACE-FCA best practices)
			TargetUtilizationLow:  0.40,
			TargetUtilizationHigh: 0.60,
			SuggestCompactAt:      0.55, // Soft suggestion, not demand

			// Research phase thresholds
			ResearchConfidenceThreshold: 0.70,
			MaxOpenQuestions:            2,

			// Gate behavior: advisory warnings, not blocking
			WarnOnResearchIncomplete: true,
			WarnOnPlanIncomplete:     true,
			BlockInStrictMode:        false, // Changed: don't block individual tools

			// Phase checkpoints: pause for human review at transitions
			RequireCheckpointReview: true,

			// Parallel implementation
			ParallelImplementationEnabled: true,
			MaxParallelAgents:             3,
			MinStepsForParallel:           3,
		},
	}
}

// Load reads the config file from the given working directory.
// Returns default config if file doesn't exist.
func Load(workDir string) (*Config, error) {
	if workDir == "" {
		workDir = validation.GetWorkDir()
	}

	configPath := filepath.Join(workDir, ".claude", ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	config := DefaultConfig()
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// IsHarnessInitialized checks if the harness marker file exists
func IsHarnessInitialized(workDir string) bool {
	if workDir == "" {
		workDir = validation.GetWorkDir()
	}

	markerPath := filepath.Join(workDir, ".claude", InitMarkerFileName)
	_, err := os.Stat(markerPath)
	return err == nil
}

// IsRelaxedMode returns true if strictness is relaxed
func (c *Config) IsRelaxedMode() bool {
	return c.Strictness == StrictnessRelaxed
}

// IsStrictMode returns true if strictness is strict
func (c *Config) IsStrictMode() bool {
	return c.Strictness == StrictnessStrict
}

// IsStandardMode returns true if strictness is standard
func (c *Config) IsStandardMode() bool {
	return c.Strictness == StrictnessStandard || c.Strictness == ""
}

// GetSuggestCompactAt returns the threshold at which compaction is suggested (not demanded)
func (c *Config) GetSuggestCompactAt() float64 {
	if c.FICConfig != nil && c.FICConfig.SuggestCompactAt > 0 {
		return c.FICConfig.SuggestCompactAt
	}
	return 0.55
}

// GetTargetUtilizationRange returns the target utilization range (low, high)
func (c *Config) GetTargetUtilizationRange() (float64, float64) {
	if c.FICConfig != nil {
		low := c.FICConfig.TargetUtilizationLow
		high := c.FICConfig.TargetUtilizationHigh
		if low > 0 && high > 0 {
			return low, high
		}
	}
	return 0.40, 0.60
}

// RequireCheckpointReview returns whether phase transitions require human review
func (c *Config) RequireCheckpointReview() bool {
	if c.FICConfig != nil {
		return c.FICConfig.RequireCheckpointReview
	}
	return true
}

// GetAutoCompactThreshold returns the auto-compact threshold (legacy, kept for compatibility)
func (c *Config) GetAutoCompactThreshold() float64 {
	if c.FICConfig != nil && c.FICConfig.AutoCompactThreshold > 0 {
		return c.FICConfig.AutoCompactThreshold
	}
	// Return suggest threshold as fallback
	return c.GetSuggestCompactAt()
}

// GetCompactionToolThreshold returns the compaction tool threshold (legacy, kept for compatibility)
func (c *Config) GetCompactionToolThreshold() int {
	if c.FICConfig != nil && c.FICConfig.CompactionToolThreshold > 0 {
		return c.FICConfig.CompactionToolThreshold
	}
	return 100 // Increased default, less aggressive
}

// GetResearchConfidenceThreshold returns the research confidence threshold
func (c *Config) GetResearchConfidenceThreshold() float64 {
	if c.FICConfig != nil && c.FICConfig.ResearchConfidenceThreshold > 0 {
		return c.FICConfig.ResearchConfidenceThreshold
	}
	return 0.70
}

// GetMaxOpenQuestions returns the maximum allowed open questions
func (c *Config) GetMaxOpenQuestions() int {
	if c.FICConfig != nil && c.FICConfig.MaxOpenQuestions > 0 {
		return c.FICConfig.MaxOpenQuestions
	}
	return 2
}

// IsAutoCompactEnabled returns whether auto-compaction is enabled
func (c *Config) IsAutoCompactEnabled() bool {
	if c.FICConfig != nil {
		return c.FICConfig.AutoCompactEnabled
	}
	return true // Enabled by default
}

// ShouldWarnOnResearchIncomplete returns whether to warn when research is incomplete
func (c *Config) ShouldWarnOnResearchIncomplete() bool {
	if c.FICConfig != nil {
		return c.FICConfig.WarnOnResearchIncomplete
	}
	return true
}

// ShouldWarnOnPlanIncomplete returns whether to warn when plan is incomplete
func (c *Config) ShouldWarnOnPlanIncomplete() bool {
	if c.FICConfig != nil {
		return c.FICConfig.WarnOnPlanIncomplete
	}
	return true
}

// ShouldBlockInStrictMode returns whether to block operations in strict mode
func (c *Config) ShouldBlockInStrictMode() bool {
	if c.FICConfig != nil {
		return c.FICConfig.BlockInStrictMode
	}
	return true
}

// Save writes the config to disk
func (c *Config) Save(workDir string) error {
	if workDir == "" {
		workDir = validation.GetWorkDir()
	}

	configDir := filepath.Join(workDir, ".claude")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// SetStrictness updates the strictness level
func (c *Config) SetStrictness(level string) {
	switch level {
	case StrictnessRelaxed, StrictnessStandard, StrictnessStrict:
		c.Strictness = level
	default:
		c.Strictness = StrictnessStandard
	}
}

// SetResearchConfidenceThreshold updates the research confidence threshold
func (c *Config) SetResearchConfidenceThreshold(threshold float64) {
	if c.FICConfig == nil {
		c.FICConfig = &FICConfig{}
	}
	if threshold >= 0 && threshold <= 1.0 {
		c.FICConfig.ResearchConfidenceThreshold = threshold
	}
}

// SetMaxOpenQuestions updates the max open questions threshold
func (c *Config) SetMaxOpenQuestions(max int) {
	if c.FICConfig == nil {
		c.FICConfig = &FICConfig{}
	}
	if max >= 0 {
		c.FICConfig.MaxOpenQuestions = max
	}
}
