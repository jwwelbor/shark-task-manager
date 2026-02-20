package services

import "github.com/jwwelbor/shark-task-manager/internal/repository"

// SessionAnalytics is an alias for repository.SessionAnalytics, made available
// in the services package so that WorkSessionRepository interface methods can
// reference it without importing repository in task_service.go.
type SessionAnalytics = repository.SessionAnalytics

// SessionAnalyticsInput defines filters for session analytics queries.
// Exactly one of EpicKey or FeatureKey must be set.
type SessionAnalyticsInput struct {
	EpicKey    string
	FeatureKey string
	AgentType  string
}

// SessionAnalyticsResult wraps analytics data with scope information.
type SessionAnalyticsResult struct {
	Scope     string            `json:"scope"`
	Analytics *SessionAnalytics `json:"analytics"`
}
