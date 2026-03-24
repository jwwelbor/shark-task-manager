package services

import "time"

// SessionAnalytics holds aggregated analytics for work sessions across multiple tasks.
// This is a service-layer type that mirrors repository.worksession.SessionAnalytics,
// decoupling the service interface from the concrete repository package.
type SessionAnalytics struct {
	TotalSessions          int
	TotalDuration          time.Duration
	AverageDuration        time.Duration
	MedianDuration         time.Duration
	TasksWithSessions      int
	TasksWithPauses        int
	AverageSessionsPerTask float64
	PauseRate              float64 // Percentage of sessions that were paused
}

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
