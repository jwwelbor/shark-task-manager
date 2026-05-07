package viewer

import "github.com/jwwelbor/shark-task-manager/internal/services"

// This file re-exports viewer-facing sprint DTOs from the services package so
// the handler layer and its tests can refer to them through api/viewer without
// importing the full service package tree directly.

type SprintOverviewResponse = services.SprintOverviewResponse
type SprintPlanView = services.SprintPlanView
type SprintReportResponse = services.SprintReportResponse
