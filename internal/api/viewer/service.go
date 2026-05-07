// Package viewer provides the HTTP handler layer for the read-only dashboard API
// under /api/v1/viewer/. It defines the ViewerServicer interface that the handler
// depends on, and the option types used to pass query parameters to the service.
package viewer

import (
	"context"

	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// ViewerServicer is the interface that ViewerHandler depends on.
// Defined here so tests can inject a mock without importing the full service.
// The concrete *services.ViewerService satisfies this interface.
type ViewerServicer interface {
	Summary(ctx context.Context) (*services.SummaryResponse, error)
	// Hierarchy signature changed in F06: opts carries the optional tag filter (ADR-F06-5).
	Hierarchy(ctx context.Context, opts services.HierarchyOptions) (*services.HierarchyResponse, error)
	SprintOverview(ctx context.Context, key string) (*services.SprintOverviewResponse, error)
	SprintPlan(ctx context.Context, key string) (*services.SprintPlanView, error)
	SprintReport(ctx context.Context, key string) (*services.SprintReportResponse, error)
	History(ctx context.Context, key string) (*services.HistoryResponse, error)
	File(ctx context.Context, key string) (*services.FileResponse, error)
	FileByPath(ctx context.Context, filePath string) (*services.FileResponse, error)
	FolderFiles(ctx context.Context, dirPath string) (*services.FolderFilesResponse, error)
	FeatureTasks(ctx context.Context, featureKey string, opts services.FeatureTaskOptions) (*services.FeatureTasksResponse, error)
	RecentActivity(ctx context.Context, opts services.RecentActivityOptions) (*services.RecentActivityResponse, error)
	WorkflowMeta(ctx context.Context) (*services.WorkflowMetaResponse, error)
	Notes(ctx context.Context, key string) (*services.NotesResponse, error)
	RelatedDocs(ctx context.Context, key string) (*services.RelatedDocsResponse, error)
	Tags(ctx context.Context) (*services.TagsResponse, error)
}

// Compile-time check: *services.ViewerService must satisfy ViewerServicer.
var _ ViewerServicer = (*services.ViewerService)(nil)
