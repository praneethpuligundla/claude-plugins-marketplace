// Package features handles loading and managing the feature checklist.
package features

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FeaturesFile is the name of the features file.
const FeaturesFile = "claude-features.json"

// Feature represents a single feature in the checklist.
type Feature struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // passing, failing, in_progress, pending
	Priority    int    `json:"priority,omitempty"`
}

// FeaturesData represents the features checklist file structure.
type FeaturesData struct {
	Features []Feature `json:"features"`
}

// Summary provides aggregate information about features.
type Summary struct {
	Total      int
	Passing    int
	Failing    int
	InProgress int
	Pending    int
	NextItems  []Feature
}

// Load reads and parses the features checklist file.
func Load(workDir string) (*FeaturesData, error) {
	featuresPath := filepath.Join(workDir, FeaturesFile)
	data, err := os.ReadFile(featuresPath)
	if err != nil {
		return nil, err
	}

	var features FeaturesData
	if err := json.Unmarshal(data, &features); err != nil {
		return nil, err
	}
	return &features, nil
}

// Exists checks if the features file exists.
func Exists(workDir string) bool {
	featuresPath := filepath.Join(workDir, FeaturesFile)
	_, err := os.Stat(featuresPath)
	return err == nil
}

// GetSummary returns aggregate stats about features.
func GetSummary(workDir string) (*Summary, error) {
	data, err := Load(workDir)
	if err != nil {
		return nil, err
	}

	summary := &Summary{}
	summary.Total = len(data.Features)

	for _, f := range data.Features {
		switch f.Status {
		case "passing":
			summary.Passing++
		case "failing":
			summary.Failing++
		case "in_progress":
			summary.InProgress++
		default:
			summary.Pending++
		}
	}

	// Get next priority items (failing and in_progress, up to 5)
	for _, f := range data.Features {
		if f.Status == "failing" || f.Status == "in_progress" {
			summary.NextItems = append(summary.NextItems, f)
			if len(summary.NextItems) >= 5 {
				break
			}
		}
	}

	return summary, nil
}

// GetInProgress returns features currently in progress.
func GetInProgress(workDir string) ([]Feature, error) {
	data, err := Load(workDir)
	if err != nil {
		return nil, err
	}

	var inProgress []Feature
	for _, f := range data.Features {
		if f.Status == "in_progress" {
			inProgress = append(inProgress, f)
		}
	}
	return inProgress, nil
}

// GetFailing returns failing features.
func GetFailing(workDir string) ([]Feature, error) {
	data, err := Load(workDir)
	if err != nil {
		return nil, err
	}

	var failing []Feature
	for _, f := range data.Features {
		if f.Status == "failing" {
			failing = append(failing, f)
		}
	}
	return failing, nil
}
