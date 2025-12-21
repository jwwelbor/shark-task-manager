package formatters

import (
	"encoding/json"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// EpicWithProgress combines Epic model with calculated progress
type EpicWithProgress struct {
	*models.Epic
	ProgressPct float64 `json:"progress_pct"`
}

// FeatureWithTaskCount combines Feature model with task count
type FeatureWithTaskCount struct {
	*models.Feature
	TaskCount int `json:"task_count"`
}

// EpicListResponse represents the JSON structure for epic list output
type EpicListResponse struct {
	Results []*EpicWithProgress `json:"results"`
	Count   int                 `json:"count"`
}

// FeatureListResponse represents the JSON structure for feature list output
type FeatureListResponse struct {
	Results []*FeatureWithTaskCount `json:"results"`
	Count   int                     `json:"count"`
}

// EpicGetResponse represents the JSON structure for epic get output
type EpicGetResponse struct {
	*EpicWithProgress
	Features []*FeatureWithTaskCount `json:"features"`
}

// FeatureGetResponse represents the JSON structure for feature get output
type FeatureGetResponse struct {
	*models.Feature
	Tasks           []*models.Task            `json:"tasks"`
	StatusBreakdown map[models.TaskStatus]int `json:"status_breakdown"`
}

// FormatEpicListJSON formats a list of epics as JSON
func FormatEpicListJSON(epics []*EpicWithProgress) (string, error) {
	response := EpicListResponse{
		Results: epics,
		Count:   len(epics),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatEpicGetJSON formats a single epic with features as JSON
func FormatEpicGetJSON(epic *EpicWithProgress, features []*FeatureWithTaskCount) (string, error) {
	response := EpicGetResponse{
		EpicWithProgress: epic,
		Features:         features,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatFeatureListJSON formats a list of features as JSON
func FormatFeatureListJSON(features []*FeatureWithTaskCount) (string, error) {
	response := FeatureListResponse{
		Results: features,
		Count:   len(features),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// FormatFeatureGetJSON formats a single feature with tasks as JSON
func FormatFeatureGetJSON(feature *models.Feature, tasks []*models.Task, breakdown map[models.TaskStatus]int) (string, error) {
	response := FeatureGetResponse{
		Feature:         feature,
		Tasks:           tasks,
		StatusBreakdown: breakdown,
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}
