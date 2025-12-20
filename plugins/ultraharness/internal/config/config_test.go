package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check default values
	if cfg.Strictness != StrictnessStandard {
		t.Errorf("Strictness = %v, want %v", cfg.Strictness, StrictnessStandard)
	}
	if !cfg.FICEnabled {
		t.Error("FICEnabled should be true by default")
	}
	if !cfg.FICContextTracking {
		t.Error("FICContextTracking should be true by default")
	}
	if !cfg.FICAutoDelegateResearch {
		t.Error("FICAutoDelegateResearch should be true by default")
	}
	if !cfg.AutoProgressLogging {
		t.Error("AutoProgressLogging should be true by default")
	}
	if cfg.CheckpointIntervalMinutes != 30 {
		t.Errorf("CheckpointIntervalMinutes = %d, want 30", cfg.CheckpointIntervalMinutes)
	}

	// Check FIC config defaults
	if cfg.FICConfig == nil {
		t.Fatal("FICConfig should not be nil")
	}
	if cfg.FICConfig.AutoCompactThreshold != 0.70 {
		t.Errorf("AutoCompactThreshold = %f, want 0.70", cfg.FICConfig.AutoCompactThreshold)
	}
	if cfg.FICConfig.CompactionToolThreshold != 25 {
		t.Errorf("CompactionToolThreshold = %d, want 25", cfg.FICConfig.CompactionToolThreshold)
	}
}

func TestStrictnessModes(t *testing.T) {
	tests := []struct {
		name       string
		strictness string
		isRelaxed  bool
		isStandard bool
		isStrict   bool
	}{
		{
			name:       "relaxed mode",
			strictness: StrictnessRelaxed,
			isRelaxed:  true,
			isStandard: false,
			isStrict:   false,
		},
		{
			name:       "standard mode",
			strictness: StrictnessStandard,
			isRelaxed:  false,
			isStandard: true,
			isStrict:   false,
		},
		{
			name:       "strict mode",
			strictness: StrictnessStrict,
			isRelaxed:  false,
			isStandard: false,
			isStrict:   true,
		},
		{
			name:       "empty defaults to standard",
			strictness: "",
			isRelaxed:  false,
			isStandard: true,
			isStrict:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Strictness: tt.strictness}

			if got := cfg.IsRelaxedMode(); got != tt.isRelaxed {
				t.Errorf("IsRelaxedMode() = %v, want %v", got, tt.isRelaxed)
			}
			if got := cfg.IsStandardMode(); got != tt.isStandard {
				t.Errorf("IsStandardMode() = %v, want %v", got, tt.isStandard)
			}
			if got := cfg.IsStrictMode(); got != tt.isStrict {
				t.Errorf("IsStrictMode() = %v, want %v", got, tt.isStrict)
			}
		})
	}
}

func TestGetAutoCompactThreshold(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantValue float64
	}{
		{
			name:      "nil FICConfig uses default",
			cfg:       &Config{FICConfig: nil},
			wantValue: 0.70,
		},
		{
			name:      "zero threshold uses default",
			cfg:       &Config{FICConfig: &FICConfig{AutoCompactThreshold: 0}},
			wantValue: 0.70,
		},
		{
			name:      "custom threshold",
			cfg:       &Config{FICConfig: &FICConfig{AutoCompactThreshold: 0.80}},
			wantValue: 0.80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetAutoCompactThreshold(); got != tt.wantValue {
				t.Errorf("GetAutoCompactThreshold() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestGetCompactionToolThreshold(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		wantValue int
	}{
		{
			name:      "nil FICConfig uses default",
			cfg:       &Config{FICConfig: nil},
			wantValue: 25,
		},
		{
			name:      "zero threshold uses default",
			cfg:       &Config{FICConfig: &FICConfig{CompactionToolThreshold: 0}},
			wantValue: 25,
		},
		{
			name:      "custom threshold",
			cfg:       &Config{FICConfig: &FICConfig{CompactionToolThreshold: 50}},
			wantValue: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetCompactionToolThreshold(); got != tt.wantValue {
				t.Errorf("GetCompactionToolThreshold() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("non-existent config returns default", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "config-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Should return default config
		if cfg.Strictness != StrictnessStandard {
			t.Errorf("Strictness = %v, want %v", cfg.Strictness, StrictnessStandard)
		}
	})

	t.Run("load custom config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "config-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create .claude directory and config file
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		customConfig := &Config{
			Strictness:               StrictnessStrict,
			FICEnabled:               false,
			CheckpointIntervalMinutes: 60,
		}

		data, err := json.Marshal(customConfig)
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		configPath := filepath.Join(claudeDir, ConfigFileName)
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		cfg, err := Load(tmpDir)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Strictness != StrictnessStrict {
			t.Errorf("Strictness = %v, want %v", cfg.Strictness, StrictnessStrict)
		}
		if cfg.FICEnabled {
			t.Error("FICEnabled should be false")
		}
		if cfg.CheckpointIntervalMinutes != 60 {
			t.Errorf("CheckpointIntervalMinutes = %d, want 60", cfg.CheckpointIntervalMinutes)
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "config-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		configPath := filepath.Join(claudeDir, ConfigFileName)
		if err := os.WriteFile(configPath, []byte("invalid json{"), 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		_, err = Load(tmpDir)
		if err == nil {
			t.Error("Load() should return error for invalid JSON")
		}
	})
}

func TestIsHarnessInitialized(t *testing.T) {
	t.Run("not initialized", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "harness-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		if IsHarnessInitialized(tmpDir) {
			t.Error("IsHarnessInitialized() = true, want false")
		}
	})

	t.Run("initialized", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "harness-test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create marker file
		claudeDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			t.Fatalf("Failed to create .claude dir: %v", err)
		}

		markerPath := filepath.Join(claudeDir, InitMarkerFileName)
		if err := os.WriteFile(markerPath, []byte("initialized"), 0644); err != nil {
			t.Fatalf("Failed to write marker file: %v", err)
		}

		if !IsHarnessInitialized(tmpDir) {
			t.Error("IsHarnessInitialized() = false, want true")
		}
	})
}

func TestConfigConstants(t *testing.T) {
	if ConfigFileName != "claude-harness.json" {
		t.Errorf("ConfigFileName = %v, want 'claude-harness.json'", ConfigFileName)
	}
	if InitMarkerFileName != ".claude-harness-initialized" {
		t.Errorf("InitMarkerFileName = %v, want '.claude-harness-initialized'", InitMarkerFileName)
	}
	if StrictnessRelaxed != "relaxed" {
		t.Errorf("StrictnessRelaxed = %v, want 'relaxed'", StrictnessRelaxed)
	}
	if StrictnessStandard != "standard" {
		t.Errorf("StrictnessStandard = %v, want 'standard'", StrictnessStandard)
	}
	if StrictnessStrict != "strict" {
		t.Errorf("StrictnessStrict = %v, want 'strict'", StrictnessStrict)
	}
}
