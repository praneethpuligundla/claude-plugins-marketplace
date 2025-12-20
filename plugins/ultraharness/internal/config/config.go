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
	AutoCompactThreshold     float64 `json:"auto_compact_threshold"`
	CompactionToolThreshold  int     `json:"compaction_tool_threshold"`
	TargetUtilizationHigh    float64 `json:"target_utilization_high"`
	TargetUtilizationLow     float64 `json:"target_utilization_low"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Strictness:               StrictnessStandard,
		FICEnabled:               true,
		FICContextTracking:       true,
		FICAutoDelegateResearch:  true,
		AutoProgressLogging:      true,
		AutoCheckpointSuggestions: true,
		CheckpointIntervalMinutes: 30,
		FeatureEnforcement:       true,
		InitScriptExecution:      true,
		BaselineTestsOnStartup:   true,
		FICConfig: &FICConfig{
			AutoCompactThreshold:    0.70,
			CompactionToolThreshold: 25,
			TargetUtilizationHigh:   0.60,
			TargetUtilizationLow:    0.40,
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

// GetAutoCompactThreshold returns the auto-compact threshold
func (c *Config) GetAutoCompactThreshold() float64 {
	if c.FICConfig != nil && c.FICConfig.AutoCompactThreshold > 0 {
		return c.FICConfig.AutoCompactThreshold
	}
	return 0.70
}

// GetCompactionToolThreshold returns the compaction tool threshold
func (c *Config) GetCompactionToolThreshold() int {
	if c.FICConfig != nil && c.FICConfig.CompactionToolThreshold > 0 {
		return c.FICConfig.CompactionToolThreshold
	}
	return 25
}
