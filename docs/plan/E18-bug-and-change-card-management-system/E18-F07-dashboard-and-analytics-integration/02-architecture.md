# E18-F07: Dashboard and Analytics Integration -- Technical Architecture

**Feature Key**: E18-F07
**Complexity Tier**: STANDARD
**Date**: 2026-03-03
**Author**: Architect Agent

---

## 1. Architecture Overview

This feature extends two existing systems -- the `shark status` dashboard and `shark analytics` command -- to include bug and change-card summary data. No new commands, services, or packages are introduced. The work is incremental extension of proven patterns.

### Design Principles Applied

- **Appropriate**: Extends existing rendering and calculation infrastructure rather than building new systems.
- **Proven**: Follows the same StatusService + formatter pattern used for epics, features, and tasks.
- **Simple**: Conditional display (omit sections when zero entities) keeps output clean.

### System Boundary

```
CLI Command Layer              Service Layer                    Repository Layer
==================             ==============                   ================

shark status (existing)  --->  StatusService.GetDashboard()     BugRepository.CountByStatus()
  + new bug section            + query bug/cc repos             BugRepository.CountBySeverity()
  + new change-card section    + populate optional fields       ChangeCardRepository.CountByStatus()

shark status E07-F01     --->  StatusService (feature level)    BugRepository.CountLinkedToFeature()
  + linked bug context         + query linked bugs              BugRepository.CountLinkedBySeverity()

shark analytics          --->  DashboardAnalyticsService (NEW)  BugRepository.GetResolutionStats()
  + --type=bug                 (focused sub-service)            ChangeCardRepository.GetThroughputStats()
  + --type=change
```

---

## 2. Key Architecture Decisions

### ADR-F07-001: Extend StatusService vs. Create New Service

**Decision**: Extend the existing `StatusService` in `internal/status/status.go` with optional bug/change-card repository dependencies.

**Rationale**: The dashboard is a single cohesive view. Splitting it across services would require cross-service orchestration for a single CLI command. The StatusService already aggregates data from epic, feature, and task repositories; adding two more optional dependencies is a linear extension.

**Consequence**: The `NewStatusService()` constructor gains optional parameters for bug and change-card repositories. When nil (i.e., before F02/F03 are built), the dashboard renders without those sections.

### ADR-F07-002: New DashboardAnalyticsService for Bug/Change-Card Analytics

**Decision**: Create a focused `DashboardAnalyticsService` in `internal/services/` for bug and change-card analytics, rather than extending `EpicAnalyticsService`.

**Rationale**: `EpicAnalyticsService` is scoped to epic progress, feature rollups, and task impediments. Bug/change-card analytics (resolution time, approval rate, severity distributions) are a different concern. Mixing them into `EpicAnalyticsService` would violate single responsibility. A focused sub-service keeps the analytics command's service layer clean.

**Consequence**: The analytics command calls `DashboardAnalyticsService` for bug/change analytics and `EpicAnalyticsService` for session analytics. Both are accessed via separate global accessors.

### ADR-F07-003: Conditional Section Display via nil/omitempty

**Decision**: Use Go pointer types with `json:"...,omitempty"` for bug and change-card summary fields in `StatusDashboard`. When the count is zero, the field is nil and omitted from JSON output. The formatter skips rendering when the field is nil.

**Rationale**: This is the simplest approach and aligns with the existing `RecentCompletions` field pattern in `StatusDashboard`, which already uses `omitempty`. No special flag or configuration needed.

### ADR-F07-004: Repository Aggregate Methods Return Summary DTOs

**Decision**: Bug and change-card repositories expose aggregate query methods that return summary DTOs (`BugStatusSummary`, `ChangeCardStatusSummary`), not raw `map[string]int`.

**Rationale**: Typed DTOs provide compile-time safety, self-documenting field names, and JSON serialization tags. They prevent misuse of generic maps and make the API contract explicit. The DTOs are defined in the repository package alongside the repositories (they are data-access-level types, not business-logic types).

---

## 3. Data Model Extensions

### 3.1 New Repository DTOs (in bug/change-card repository packages, created by F02/F03)

These types are created by F02 and F03 respectively. F07 consumes them. Documented here for contract clarity.

```go
// BugStatusSummary contains aggregate bug counts for dashboard display.
// Created by F02 in internal/repository/bug_repository.go
type BugStatusSummary struct {
    Total           int            `json:"total"`
    ByStatus        map[string]int `json:"by_status"`         // reported: 3, triaged: 2, ...
    BySeverity      map[string]int `json:"by_severity"`       // critical: 1, high: 2, ...
    OpenBySeverity  map[string]int `json:"open_by_severity"`  // non-terminal only
}

// BugResolutionStats contains resolution time metrics for analytics.
// Created by F02 in internal/repository/bug_repository.go
type BugResolutionStats struct {
    ResolvedCount      int            `json:"resolved_count"`
    AvgResolutionSecs  *float64       `json:"avg_resolution_time_seconds"`  // nil if no resolved bugs
    // P50/P90 fields (Could-Have, US-F07-010):
    // P50ResolutionSecs *float64     `json:"p50_resolution_time_seconds,omitempty"`
    // P90ResolutionSecs *float64     `json:"p90_resolution_time_seconds,omitempty"`
}

// BugFeatureSummary contains bug data linked to a specific feature.
// Created by F02 in internal/repository/bug_repository.go
type BugFeatureSummary struct {
    TotalLinked    int            `json:"total_linked"`
    OpenCount      int            `json:"open_count"`
    OpenBySeverity map[string]int `json:"open_by_severity"` // critical: 1, high: 0, ...
}

// ChangeCardStatusSummary contains aggregate change-card counts for dashboard.
// Created by F03 in internal/repository/change_card_repository.go
type ChangeCardStatusSummary struct {
    Total    int            `json:"total"`
    ByStatus map[string]int `json:"by_status"` // proposed: 2, approved: 1, ...
}

// ChangeCardThroughputStats contains throughput metrics for analytics.
// Created by F03 in internal/repository/change_card_repository.go
type ChangeCardThroughputStats struct {
    DecidedCount        int      `json:"decided_count"`       // approved + declined
    ApprovedCount       int      `json:"approved_count"`
    DeclinedCount       int      `json:"declined_count"`
    ApprovalRate        *float64 `json:"approval_rate"`       // nil if no decided cards
    CompletedCount      int      `json:"completed_count"`
    AvgCompletionSecs   *float64 `json:"avg_completion_time_seconds"` // nil if no completed
}
```

### 3.2 Repository Interface Methods (consumed by F07 services)

F07 defines service-layer interfaces for the methods it needs. The concrete implementations live in F02/F03.

```go
// In internal/services/dashboard_analytics_service.go

// BugSummaryRepository defines the aggregate query methods F07 needs from BugRepository.
type BugSummaryRepository interface {
    GetStatusSummary(ctx context.Context) (*repository.BugStatusSummary, error)
    GetResolutionStats(ctx context.Context) (*repository.BugResolutionStats, error)
    GetFeatureBugSummary(ctx context.Context, featureKey string) (*repository.BugFeatureSummary, error)
}

// ChangeCardSummaryRepository defines the aggregate query methods F07 needs.
type ChangeCardSummaryRepository interface {
    GetStatusSummary(ctx context.Context) (*repository.ChangeCardStatusSummary, error)
    GetThroughputStats(ctx context.Context) (*repository.ChangeCardThroughputStats, error)
}
```

### 3.3 Dashboard Model Extensions

Extend `StatusDashboard` in `internal/status/models.go`:

```go
// StatusDashboard is the complete dashboard output structure
type StatusDashboard struct {
    Summary           *ProjectSummary             `json:"summary"`
    Epics             []*EpicSummary              `json:"epics"`
    ActiveTasks       map[string][]*TaskInfo      `json:"active_tasks"`
    BlockedTasks      []*BlockedTaskInfo          `json:"blocked_tasks"`
    RecentCompletions []*CompletionInfo           `json:"recent_completions,omitempty"`
    Filter            *DashboardFilter            `json:"filter,omitempty"`
    // NEW: Bug and change-card summaries (conditional, omitted when nil)
    BugSummary        *BugDashboardSummary        `json:"bugs,omitempty"`
    ChangeCardSummary *ChangeCardDashboardSummary `json:"change_cards,omitempty"`
}

// BugDashboardSummary contains bug data for the project dashboard.
type BugDashboardSummary struct {
    Total          int            `json:"total"`
    ByStatus       map[string]int `json:"by_status"`
    OpenBySeverity map[string]int `json:"open_by_severity"`
}

// ChangeCardDashboardSummary contains change-card data for the project dashboard.
type ChangeCardDashboardSummary struct {
    Total    int            `json:"total"`
    ByStatus map[string]int `json:"by_status"`
}
```

### 3.4 Feature Status Extension

Extend `FeatureStatusInfo` in `internal/status/types.go` to include linked bug context:

```go
type FeatureStatusInfo struct {
    Feature         interface{}           // *models.Feature
    StatusBreakdown map[string]int        // Status -> count
    Tasks           []interface{}         // []*models.Task
    Progress        *ProgressInfo         // Calculated progress metrics
    WorkSummary     *WorkSummary          // Work breakdown
    StatusContext   string                // "active (waiting)", etc.
    ActionItems     *ActionItems          // Tasks needing attention
    // NEW: Linked bug context (optional, nil when no bugs linked)
    LinkedBugs      *BugFeatureSummary    `json:"linked_bugs,omitempty"`
}

// BugFeatureSummary is the feature-level bug context.
// Reuses the repository DTO structure.
type BugFeatureSummary struct {
    TotalLinked    int            `json:"total_linked"`
    OpenCount      int            `json:"open_count"`
    OpenBySeverity map[string]int `json:"open_by_severity"`
}
```

---

## 4. Service Layer Design

### 4.1 StatusService Extension (Dashboard)

**File**: `internal/status/status.go`

**Changes**:

1. Add optional bug and change-card repository dependencies to `StatusService`:

```go
type StatusService struct {
    db              *repository.DB
    epicRepo        *repository.EpicRepository
    featureRepo     *repository.FeatureRepository
    taskRepo        *repository.TaskRepository
    taskHistoryRepo *repository.TaskHistoryRepository
    // NEW: Optional bug/change-card repos (nil when entities not yet available)
    bugRepo         BugDashboardRepository         // interface, nil-safe
    changeCardRepo  ChangeCardDashboardRepository   // interface, nil-safe
}
```

2. Define narrow interfaces for what the dashboard needs:

```go
// BugDashboardRepository defines what StatusService needs from the bug repo.
type BugDashboardRepository interface {
    GetStatusSummary(ctx context.Context) (*BugStatusSummary, error)
    GetFeatureBugSummary(ctx context.Context, featureKey string) (*BugFeatureSummary, error)
}

// ChangeCardDashboardRepository defines what StatusService needs from the change-card repo.
type ChangeCardDashboardRepository interface {
    GetStatusSummary(ctx context.Context) (*ChangeCardStatusSummary, error)
}
```

3. Extend `NewStatusService()` with optional parameters:

```go
// Option function pattern for backward compatibility
type StatusServiceOption func(*StatusService)

func WithBugRepository(repo BugDashboardRepository) StatusServiceOption {
    return func(s *StatusService) { s.bugRepo = repo }
}

func WithChangeCardRepository(repo ChangeCardDashboardRepository) StatusServiceOption {
    return func(s *StatusService) { s.changeCardRepo = repo }
}

func NewStatusService(database *repository.DB, opts ...StatusServiceOption) *StatusService {
    s := &StatusService{
        db:              database,
        epicRepo:        repository.NewEpicRepository(database),
        featureRepo:     repository.NewFeatureRepository(database),
        taskRepo:        repository.NewTaskRepository(database),
        taskHistoryRepo: repository.NewTaskHistoryRepository(database),
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

4. Extend `GetDashboard()` to conditionally populate bug/change-card sections:

```go
func (s *StatusService) GetDashboard(ctx context.Context, req *StatusRequest) (*StatusDashboard, error) {
    // ... existing code to build summary, epics, tasks, blocked, completions ...

    dashboard := &StatusDashboard{
        Summary:           summary,
        Epics:             epics,
        ActiveTasks:       activeTasks,
        BlockedTasks:      blockedTasks,
        RecentCompletions: recentCompletions,
    }

    // NEW: Conditionally add bug summary
    if s.bugRepo != nil {
        bugSummary, err := s.bugRepo.GetStatusSummary(ctx)
        if err != nil {
            // Degrade gracefully: log warning, don't fail the dashboard
            // (bug table might not exist yet during phased rollout)
        } else if bugSummary != nil && bugSummary.Total > 0 {
            dashboard.BugSummary = &BugDashboardSummary{
                Total:          bugSummary.Total,
                ByStatus:       bugSummary.ByStatus,
                OpenBySeverity: bugSummary.OpenBySeverity,
            }
        }
    }

    // NEW: Conditionally add change-card summary
    if s.changeCardRepo != nil {
        ccSummary, err := s.changeCardRepo.GetStatusSummary(ctx)
        if err != nil {
            // Degrade gracefully
        } else if ccSummary != nil && ccSummary.Total > 0 {
            dashboard.ChangeCardSummary = &ChangeCardDashboardSummary{
                Total:    ccSummary.Total,
                ByStatus: ccSummary.ByStatus,
            }
        }
    }

    return dashboard, nil
}
```

### 4.2 DashboardAnalyticsService (Analytics)

**File**: `internal/services/dashboard_analytics_service.go` (NEW)

This service provides bug and change-card analytics data for the `shark analytics` command.

```go
// DashboardAnalyticsService provides bug and change-card analytics.
type DashboardAnalyticsService struct {
    bugRepo        BugSummaryRepository         // optional, nil-safe
    changeCardRepo ChangeCardSummaryRepository   // optional, nil-safe
}

func NewDashboardAnalyticsService(
    bugRepo BugSummaryRepository,
    changeCardRepo ChangeCardSummaryRepository,
) *DashboardAnalyticsService {
    return &DashboardAnalyticsService{
        bugRepo:        bugRepo,
        changeCardRepo: changeCardRepo,
    }
}

// GetBugAnalytics returns bug-specific analytics.
func (s *DashboardAnalyticsService) GetBugAnalytics(ctx context.Context) (*BugAnalyticsResult, error) {
    if s.bugRepo == nil {
        return nil, fmt.Errorf("bug analytics not available: bug repository not configured")
    }

    summary, err := s.bugRepo.GetStatusSummary(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get bug status summary: %w", err)
    }

    resolution, err := s.bugRepo.GetResolutionStats(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get bug resolution stats: %w", err)
    }

    return &BugAnalyticsResult{
        TotalBugs:                summary.Total,
        BugsByStatus:             summary.ByStatus,
        BugsBySeverity:           summary.BySeverity,
        AvgResolutionTimeSecs:    resolution.AvgResolutionSecs,
        ResolvedCount:            resolution.ResolvedCount,
    }, nil
}

// GetChangeCardAnalytics returns change-card-specific analytics.
func (s *DashboardAnalyticsService) GetChangeCardAnalytics(ctx context.Context) (*ChangeCardAnalyticsResult, error) {
    if s.changeCardRepo == nil {
        return nil, fmt.Errorf("change-card analytics not available: repository not configured")
    }

    summary, err := s.changeCardRepo.GetStatusSummary(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get change-card status summary: %w", err)
    }

    throughput, err := s.changeCardRepo.GetThroughputStats(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get change-card throughput stats: %w", err)
    }

    return &ChangeCardAnalyticsResult{
        TotalChangeCards:          summary.Total,
        ChangeCardsByStatus:      summary.ByStatus,
        ApprovalRate:             throughput.ApprovalRate,
        AvgCompletionTimeSecs:    throughput.AvgCompletionSecs,
        CompletedCount:           throughput.CompletedCount,
        DecidedCount:             throughput.DecidedCount,
    }, nil
}
```

### 4.3 Analytics DTOs

**File**: `internal/services/dashboard_analytics_dto.go` (NEW)

```go
// BugAnalyticsResult contains bug analytics for CLI/JSON output.
type BugAnalyticsResult struct {
    TotalBugs             int            `json:"total_bugs"`
    BugsByStatus          map[string]int `json:"bugs_by_status"`
    BugsBySeverity        map[string]int `json:"bugs_by_severity"`
    ResolvedCount         int            `json:"resolved_count"`
    AvgResolutionTimeSecs *float64       `json:"avg_resolution_time_seconds"`
}

// ChangeCardAnalyticsResult contains change-card analytics for CLI/JSON output.
type ChangeCardAnalyticsResult struct {
    TotalChangeCards      int            `json:"total_change_cards"`
    ChangeCardsByStatus   map[string]int `json:"change_cards_by_status"`
    ApprovalRate          *float64       `json:"approval_rate"`
    DecidedCount          int            `json:"decided_count"`
    CompletedCount        int            `json:"completed_count"`
    AvgCompletionTimeSecs *float64       `json:"avg_completion_time_seconds"`
}

// DashboardAnalyticsResult is the combined analytics output when no type filter is used.
type DashboardAnalyticsResult struct {
    Bugs        *BugAnalyticsResult        `json:"bugs,omitempty"`
    ChangeCards *ChangeCardAnalyticsResult  `json:"change_cards,omitempty"`
}
```

---

## 5. CLI Command Extensions

### 5.1 Dashboard Command (`shark status`)

**File**: `internal/cli/commands/status.go` (extend existing)

No structural changes to the command itself. The existing `runStatus()` function already calls `StatusService.GetDashboard()` and passes the result to `FormatDashboard()`. The new fields are populated by the service and rendered by the formatter.

### 5.2 Dashboard Formatter Extension

**File**: `internal/status/formatter.go` (extend existing)

Add new rendering functions following the existing section pattern:

```go
// formatBugSummary renders the bug section of the dashboard.
// Returns empty string if summary is nil (conditional display).
func formatBugSummary(bugs *BugDashboardSummary, noColor bool) string {
    if bugs == nil {
        return ""
    }
    var sb strings.Builder
    // Header
    if noColor {
        sb.WriteString("\n=== BUGS ===\n\n")
    } else {
        sb.WriteString("\n" + pterm.DefaultHeader.WithFullWidth().Sprint("BUGS") + "\n\n")
    }

    sb.WriteString(fmt.Sprintf("Total: %d\n\n", bugs.Total))

    // Status breakdown
    sb.WriteString("By Status:\n")
    statusOrder := []string{"reported", "triaged", "in_fix", "in_verification", "resolved", "wont_fix", "duplicate"}
    for _, status := range statusOrder {
        if count, ok := bugs.ByStatus[status]; ok && count > 0 {
            sb.WriteString(fmt.Sprintf("  %-18s %d\n", status+":", count))
        }
    }

    // Severity breakdown (open bugs only)
    if len(bugs.OpenBySeverity) > 0 {
        hasOpen := false
        for _, count := range bugs.OpenBySeverity {
            if count > 0 {
                hasOpen = true
                break
            }
        }
        if hasOpen {
            sb.WriteString("\nOpen Bug Severity:\n")
            severityOrder := []string{"critical", "high", "medium", "low"}
            for _, sev := range severityOrder {
                count := bugs.OpenBySeverity[sev]
                sb.WriteString(fmt.Sprintf("  %-18s %d\n", sev+":", count))
            }
        }
    }

    return sb.String()
}

// formatChangeCardSummary renders the change-card section of the dashboard.
func formatChangeCardSummary(cards *ChangeCardDashboardSummary, noColor bool) string {
    if cards == nil {
        return ""
    }
    var sb strings.Builder
    if noColor {
        sb.WriteString("\n=== CHANGE CARDS ===\n\n")
    } else {
        sb.WriteString("\n" + pterm.DefaultHeader.WithFullWidth().Sprint("CHANGE CARDS") + "\n\n")
    }

    sb.WriteString(fmt.Sprintf("Total: %d\n\n", cards.Total))

    sb.WriteString("By Status:\n")
    statusOrder := []string{"proposed", "approved", "in_progress", "completed", "declined"}
    for _, status := range statusOrder {
        if count, ok := cards.ByStatus[status]; ok && count > 0 {
            sb.WriteString(fmt.Sprintf("  %-18s %d\n", status+":", count))
        }
    }

    return sb.String()
}
```

The main `FormatDashboard()` function calls these after the existing sections:

```go
func FormatDashboard(dashboard *StatusDashboard, noColor bool) string {
    var sb strings.Builder
    // ... existing rendering (summary, epics, active tasks, blocked, completions) ...

    // NEW: Append bug and change-card sections
    sb.WriteString(formatBugSummary(dashboard.BugSummary, noColor))
    sb.WriteString(formatChangeCardSummary(dashboard.ChangeCardSummary, noColor))

    return sb.String()
}
```

### 5.3 Analytics Command Extension

**File**: `internal/cli/commands/analytics.go` (extend existing)

Add `--type` flag to existing analytics command:

```go
func init() {
    cli.RootCmd.AddCommand(analyticsCmd)
    analyticsCmd.Flags().Bool("session-duration", false, "Analyze session duration metrics")
    analyticsCmd.Flags().Bool("pause-frequency", false, "Analyze pause frequency patterns")
    analyticsCmd.Flags().String("epic", "", "Filter by epic key")
    analyticsCmd.Flags().String("feature", "", "Filter by feature key")
    analyticsCmd.Flags().String("agent-type", "", "Filter by agent type")
    // NEW
    analyticsCmd.Flags().String("type", "", "Entity type analytics: bug, change")
}
```

The `runAnalytics()` handler is extended:

```go
func runAnalytics(cmd *cobra.Command, args []string) error {
    entityType, _ := cmd.Flags().GetString("type")

    // If --type is specified, route to entity analytics
    if entityType == "bug" || entityType == "change" {
        return runEntityAnalytics(cmd, entityType)
    }

    // ... existing session analytics logic ...

    // If no --type and no session flags, show combined dashboard analytics
    if !sessionDuration && !pauseFrequency {
        return runCombinedAnalytics(cmd)
    }

    // ... existing session analytics ...
}

func runEntityAnalytics(cmd *cobra.Command, entityType string) error {
    svc := cli.GetDashboardAnalyticsService()

    switch entityType {
    case "bug":
        result, err := svc.GetBugAnalytics(cmd.Context())
        if err != nil {
            return err
        }
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(result)
        }
        printBugAnalytics(result)
    case "change":
        result, err := svc.GetChangeCardAnalytics(cmd.Context())
        if err != nil {
            return err
        }
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(result)
        }
        printChangeCardAnalytics(result)
    }
    return nil
}

func runCombinedAnalytics(cmd *cobra.Command) error {
    svc := cli.GetDashboardAnalyticsService()
    result := &services.DashboardAnalyticsResult{}

    // Attempt bug analytics (degrade gracefully)
    bugResult, err := svc.GetBugAnalytics(cmd.Context())
    if err == nil && bugResult != nil && bugResult.TotalBugs > 0 {
        result.Bugs = bugResult
    }

    // Attempt change-card analytics (degrade gracefully)
    ccResult, err := svc.GetChangeCardAnalytics(cmd.Context())
    if err == nil && ccResult != nil && ccResult.TotalChangeCards > 0 {
        result.ChangeCards = ccResult
    }

    if cli.GlobalConfig.JSON {
        return cli.OutputJSON(result)
    }

    if result.Bugs != nil {
        printBugAnalytics(result.Bugs)
    }
    if result.ChangeCards != nil {
        printChangeCardAnalytics(result.ChangeCards)
    }

    if result.Bugs == nil && result.ChangeCards == nil {
        fmt.Println("No bug or change-card data available for analytics.")
        fmt.Println("Use --session-duration or --pause-frequency for session analytics.")
    }

    return nil
}
```

### 5.4 Analytics Formatters

**File**: `internal/cli/commands/analytics.go` (extend existing)

```go
func printBugAnalytics(result *services.BugAnalyticsResult) {
    fmt.Printf("=== BUG ANALYTICS ===\n\n")
    fmt.Printf("Total Bugs: %d\n\n", result.TotalBugs)

    fmt.Printf("By Status:\n")
    for status, count := range result.BugsByStatus {
        fmt.Printf("  %-18s %d\n", status+":", count)
    }

    fmt.Printf("\nBy Severity:\n")
    for _, sev := range []string{"critical", "high", "medium", "low"} {
        fmt.Printf("  %-18s %d\n", sev+":", result.BugsBySeverity[sev])
    }

    fmt.Printf("\nResolution:\n")
    fmt.Printf("  Resolved:          %d\n", result.ResolvedCount)
    if result.AvgResolutionTimeSecs != nil {
        fmt.Printf("  Avg Resolution:    %s\n", formatDurationFromSecs(*result.AvgResolutionTimeSecs))
    } else {
        fmt.Printf("  Avg Resolution:    N/A\n")
    }
    fmt.Println()
}

func printChangeCardAnalytics(result *services.ChangeCardAnalyticsResult) {
    fmt.Printf("=== CHANGE CARD ANALYTICS ===\n\n")
    fmt.Printf("Total Change Cards: %d\n\n", result.TotalChangeCards)

    fmt.Printf("By Status:\n")
    for status, count := range result.ChangeCardsByStatus {
        fmt.Printf("  %-18s %d\n", status+":", count)
    }

    fmt.Printf("\nThroughput:\n")
    if result.ApprovalRate != nil {
        fmt.Printf("  Approval Rate:     %.1f%%\n", *result.ApprovalRate*100)
    } else {
        fmt.Printf("  Approval Rate:     N/A\n")
    }
    fmt.Printf("  Decided:           %d\n", result.DecidedCount)
    fmt.Printf("  Completed:         %d\n", result.CompletedCount)
    if result.AvgCompletionTimeSecs != nil {
        fmt.Printf("  Avg Completion:    %s\n", formatDurationFromSecs(*result.AvgCompletionTimeSecs))
    } else {
        fmt.Printf("  Avg Completion:    N/A\n")
    }
    fmt.Println()
}

// formatDurationFromSecs converts seconds to human-readable duration.
func formatDurationFromSecs(secs float64) string {
    d := time.Duration(secs * float64(time.Second))
    hours := int(d.Hours())
    if hours >= 24 {
        days := hours / 24
        remainingHours := hours % 24
        return fmt.Sprintf("%dd %dh", days, remainingHours)
    }
    minutes := int(d.Minutes()) % 60
    return fmt.Sprintf("%dh %dm", hours, minutes)
}
```

---

## 6. Service Wiring

### 6.1 CLI Global Accessor

**File**: `internal/cli/services_global.go` (extend existing)

```go
// GetDashboardAnalyticsService returns a DashboardAnalyticsService instance.
// Bug and change-card repositories are optional and may be nil if
// the corresponding tables do not exist yet.
func GetDashboardAnalyticsService() *services.DashboardAnalyticsService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(fmt.Sprintf("failed to get database: %v", err))
    }

    // These repos may be nil if F02/F03 tables don't exist yet
    var bugRepo services.BugSummaryRepository
    var ccRepo services.ChangeCardSummaryRepository

    // Attempt to create bug repo (table may not exist)
    bugRepo = repository.NewBugRepository(db)   // nil-safe if table missing
    ccRepo = repository.NewChangeCardRepository(db)

    return services.NewDashboardAnalyticsService(bugRepo, ccRepo)
}
```

### 6.2 StatusService Wiring Update

The status command's service construction adds the new optional repositories:

```go
// In status command setup or services_global.go
func GetStatusService() *status.StatusService {
    db, err := GetDB(context.Background())
    if err != nil {
        panic(...)
    }

    opts := []status.StatusServiceOption{}

    // Wire bug repo if available
    bugRepo := repository.NewBugRepository(db)
    if bugRepo != nil {
        opts = append(opts, status.WithBugRepository(bugRepo))
    }

    // Wire change-card repo if available
    ccRepo := repository.NewChangeCardRepository(db)
    if ccRepo != nil {
        opts = append(opts, status.WithChangeCardRepository(ccRepo))
    }

    return status.NewStatusService(db, opts...)
}
```

---

## 7. Repository Aggregate Query Specifications

These SQL queries are implemented by F02 and F03 respectively. Documented here as the contract F07 depends on.

### 7.1 Bug Repository Aggregate Queries

```sql
-- CountByStatus: BugStatusSummary.ByStatus
SELECT status, COUNT(*) as count
FROM bugs
GROUP BY status;

-- CountBySeverity (open only): BugStatusSummary.OpenBySeverity
SELECT severity, COUNT(*) as count
FROM bugs
WHERE status NOT IN ('resolved', 'wont_fix', 'duplicate')
GROUP BY severity;

-- Average Resolution Time: BugResolutionStats
-- Resolution time = time from created_at to the timestamp of the first
-- terminal status in bug_history (or entity_status_history).
SELECT
    COUNT(*) as resolved_count,
    AVG(
        JULIANDAY(h.changed_at) - JULIANDAY(b.created_at)
    ) * 86400 as avg_resolution_seconds
FROM bugs b
JOIN entity_status_history h ON h.entity_type = 'bug'
    AND h.entity_id = b.id
    AND h.new_status IN ('resolved', 'wont_fix', 'duplicate')
WHERE b.status IN ('resolved', 'wont_fix', 'duplicate');

-- Feature-linked bug summary
SELECT
    COUNT(*) as total_linked,
    SUM(CASE WHEN status NOT IN ('resolved', 'wont_fix', 'duplicate') THEN 1 ELSE 0 END) as open_count
FROM bugs
WHERE linked_entity_type = 'feature'
  AND linked_entity_key = ?;

-- Feature-linked open bug severity
SELECT severity, COUNT(*) as count
FROM bugs
WHERE linked_entity_type = 'feature'
  AND linked_entity_key = ?
  AND status NOT IN ('resolved', 'wont_fix', 'duplicate')
GROUP BY severity;
```

### 7.2 Change-Card Repository Aggregate Queries

```sql
-- CountByStatus: ChangeCardStatusSummary.ByStatus
SELECT status, COUNT(*) as count
FROM change_cards
GROUP BY status;

-- Approval rate and throughput: ChangeCardThroughputStats
SELECT
    COUNT(CASE WHEN status IN ('approved', 'in_progress', 'completed', 'declined') THEN 1 END) as decided_count,
    COUNT(CASE WHEN status IN ('approved', 'in_progress', 'completed') THEN 1 END) as approved_count,
    COUNT(CASE WHEN status = 'declined' THEN 1 END) as declined_count,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_count
FROM change_cards;

-- Average completion time
SELECT AVG(
    JULIANDAY(h.changed_at) - JULIANDAY(c.created_at)
) * 86400 as avg_completion_seconds
FROM change_cards c
JOIN entity_status_history h ON h.entity_type = 'change'
    AND h.entity_id = c.id
    AND h.new_status = 'completed'
WHERE c.status = 'completed';
```

### 7.3 Performance Contract

All aggregate queries must complete in under 100ms for databases with up to 1000 bugs and 1000 change-cards. The queries use simple `GROUP BY` and `COUNT` operations on small tables with indexed `status` columns. No performance concern expected.

---

## 8. JSON Output Contracts

### 8.1 Dashboard JSON (`shark status --json`)

The existing JSON structure gains two optional top-level keys:

```json
{
  "summary": { ... },
  "epics": [ ... ],
  "active_tasks": { ... },
  "blocked_tasks": [ ... ],
  "bugs": {
    "total": 7,
    "by_status": {
      "reported": 2,
      "triaged": 1,
      "in_fix": 2,
      "resolved": 2
    },
    "open_by_severity": {
      "critical": 1,
      "high": 2,
      "medium": 2,
      "low": 0
    }
  },
  "change_cards": {
    "total": 5,
    "by_status": {
      "proposed": 2,
      "approved": 1,
      "in_progress": 1,
      "completed": 1
    }
  }
}
```

When zero bugs exist, the `"bugs"` key is absent. Same for `"change_cards"`.

### 8.2 Bug Analytics JSON (`shark analytics --type=bug --json`)

```json
{
  "total_bugs": 10,
  "bugs_by_status": {
    "reported": 3,
    "triaged": 2,
    "in_fix": 1,
    "in_verification": 1,
    "resolved": 2,
    "wont_fix": 1
  },
  "bugs_by_severity": {
    "critical": 2,
    "high": 3,
    "medium": 4,
    "low": 1
  },
  "resolved_count": 3,
  "avg_resolution_time_seconds": 14400.0
}

```

### 8.3 Change-Card Analytics JSON (`shark analytics --type=change --json`)

```json
{
  "total_change_cards": 8,
  "change_cards_by_status": {
    "proposed": 2,
    "approved": 1,
    "in_progress": 2,
    "completed": 2,
    "declined": 1
  },
  "approval_rate": 0.833,
  "decided_count": 6,
  "completed_count": 2,
  "avg_completion_time_seconds": 259200.0
}
```

### 8.4 Feature Status JSON (`shark status E07-F01 --json`)

Existing feature status JSON gains an optional `linked_bugs` key:

```json
{
  "feature": { ... },
  "linked_bugs": {
    "total_linked": 3,
    "open_count": 2,
    "open_by_severity": {
      "high": 1,
      "medium": 1
    }
  }
}
```

---

## 9. Dependency Graph

```
F07 (this feature)
  depends on:
    F01 (Database Schema)  -- bugs and change_cards tables must exist
    F02 (Bug Entity Core)  -- BugRepository with aggregate query methods
    F03 (Change-Card Core) -- ChangeCardRepository with aggregate query methods
  soft dependency:
    F06 (Unified CLI)      -- entity type auto-detection for status rendering
  extends:
    internal/status/status.go      -- StatusService.GetDashboard()
    internal/status/models.go      -- StatusDashboard struct
    internal/status/formatter.go   -- FormatDashboard() function
    internal/cli/commands/analytics.go -- analytics command flags and handlers
    internal/cli/services_global.go -- service accessor functions
```

---

## 10. Testing Strategy

### 10.1 Service Tests (Mocked Repositories)

**DashboardAnalyticsService tests** (`internal/services/dashboard_analytics_service_test.go`):
- Test `GetBugAnalytics()` with mock returning various bug distributions
- Test `GetChangeCardAnalytics()` with mock returning various throughput scenarios
- Test nil repository graceful degradation
- Test zero-entity scenario (empty maps)
- Test resolution time N/A when no resolved bugs

**StatusService extension tests** (`internal/status/status_test.go`):
- Test dashboard with nil bug/change-card repos (backward compatibility)
- Test dashboard with bug repo returning zero bugs (section omitted)
- Test dashboard with bug repo returning non-zero bugs (section included)
- Test feature-level linked bug context

### 10.2 Formatter Tests (Unit Tests)

**Formatter tests** (`internal/status/formatter_test.go`):
- Test `formatBugSummary()` with nil input (returns empty string)
- Test `formatBugSummary()` with populated data (correct rendering)
- Test `formatChangeCardSummary()` with nil input
- Test `formatChangeCardSummary()` with populated data
- Test `formatDurationFromSecs()` for hours, days, and N/A scenarios

### 10.3 CLI Tests (Mocked Services)

- Test `runEntityAnalytics()` with `--type=bug` and JSON output
- Test `runEntityAnalytics()` with `--type=change` and JSON output
- Test combined analytics when both entities have data
- Test combined analytics when no entity data exists

---

## 11. Files Changed Summary

### New Files

| File | Purpose |
|------|---------|
| `internal/services/dashboard_analytics_service.go` | Bug/change-card analytics service |
| `internal/services/dashboard_analytics_dto.go` | Analytics result DTOs |
| `internal/services/dashboard_analytics_service_test.go` | Service tests |

### Modified Files

| File | Change |
|------|--------|
| `internal/status/models.go` | Add `BugDashboardSummary`, `ChangeCardDashboardSummary` to `StatusDashboard` |
| `internal/status/types.go` | Add `LinkedBugs` field to `FeatureStatusInfo` |
| `internal/status/status.go` | Add optional bug/cc repos to `StatusService`; extend `GetDashboard()` |
| `internal/status/formatter.go` | Add `formatBugSummary()`, `formatChangeCardSummary()`; extend `FormatDashboard()` |
| `internal/status/status_test.go` | Add tests for new dashboard sections |
| `internal/status/formatter_test.go` | Add tests for new formatters |
| `internal/cli/commands/analytics.go` | Add `--type` flag; add entity analytics routing and formatters |
| `internal/cli/services_global.go` | Add `GetDashboardAnalyticsService()` accessor |

### Estimated Scope

- **New code**: ~400 lines (service, DTOs, formatters)
- **Modified code**: ~150 lines (status service extension, analytics command extension, wiring)
- **Test code**: ~300 lines
- **Total**: ~850 lines

---

## 12. Security Considerations

No new security concerns. This feature reads aggregate data from existing tables using parameterized queries. No user input is passed to SQL without parameterization. The feature-level bug query uses the feature key parameter which is already validated by the key detection system.

---

## 13. Implementation Notes

1. **Phased Development**: F07 can be developed in parallel with F02/F03 by coding against the repository interfaces. Integration testing requires F02/F03 tables to exist.

2. **Graceful Degradation**: All new repository calls are guarded by nil checks. If bug/change-card tables do not exist, the dashboard and analytics commands work exactly as before.

3. **Backward Compatibility**: The `NewStatusService()` constructor uses the option function pattern to maintain backward compatibility with all existing callers. No existing code needs to change.

4. **Could-Have (US-F07-010)**: Resolution time percentiles (P50/P90) are deferred. The `BugResolutionStats` struct has commented placeholder fields. Implementation would add `PERCENTILE_DISC` or manual percentile calculation. SQLite does not have built-in percentile functions, so this would require loading all resolution times into memory and computing percentiles in Go. This is feasible for up to 1000 bugs but adds complexity. Defer to a follow-on task.

---

*Traces to*: [E18-F07 PRD](./prd.md) | [E18 Tech Feasibility Review](../E18-TECH-FEASIBILITY-REVIEW.md) Section 1.6
*Dependencies*: E18-F01, E18-F02, E18-F03
*Last Updated*: 2026-03-03
