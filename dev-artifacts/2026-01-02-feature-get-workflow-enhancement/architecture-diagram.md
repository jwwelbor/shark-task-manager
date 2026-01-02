# Architecture Diagrams

## Current Architecture (Before Enhancement)

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLI Commands Layer                           │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │ feature get │  │  epic get   │  │  task list  │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
│         │                │                │                     │
│         ▼                ▼                ▼                     │
│  ┌──────────────────────────────────────────────────┐          │
│  │     Hardcoded Status Constants                   │          │
│  │  (todo, in_progress, blocked, ready_for_review)  │          │
│  └──────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Repository Layer                               │
│                                                                  │
│  ┌────────────────────────────────────────────────────┐         │
│  │  TaskRepository.GetStatusBreakdown()               │         │
│  │                                                     │         │
│  │  Returns: map[TaskStatus]int                       │         │
│  │  - Hardcoded 6 statuses                            │         │
│  │  - Random order (Go map iteration)                 │         │
│  │  - No metadata                                     │         │
│  └────────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Database Layer                               │
│                                                                  │
│  SELECT status, COUNT(*) FROM tasks GROUP BY status             │
└─────────────────────────────────────────────────────────────────┘

PROBLEMS:
❌ Each command duplicates config loading logic
❌ Status display is inconsistent across commands
❌ No workflow metadata (colors, descriptions, phases)
❌ Status ordering is random
```

---

## Proposed Architecture (After Enhancement)

```
┌──────────────────────────────────────────────────────────────────────┐
│                        CLI Commands Layer                            │
│                                                                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│  │ feature get │  │  epic get   │  │  task list  │                 │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │
│         │                │                │                          │
│         └────────────────┴────────────────┘                          │
│                          │                                           │
│                          ▼                                           │
│         ┌────────────────────────────────────────┐                  │
│         │      Inject WorkflowService            │                  │
│         │  - Get workflow config                 │                  │
│         │  - Get status metadata                 │                  │
│         │  - Format statuses with colors         │                  │
│         └────────────────┬───────────────────────┘                  │
└──────────────────────────┼────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────────────┐
│                   WorkflowService (NEW)                              │
│                   Single Source of Truth                             │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  Public API:                                             │        │
│  │  • GetWorkflow() → *WorkflowConfig                       │        │
│  │  • GetInitialStatus() → TaskStatus                       │        │
│  │  • GetAllStatuses() → []string (ordered)                 │        │
│  │  • GetStatusMetadata(status) → StatusMetadata            │        │
│  │  • FormatStatusForDisplay(status) → FormattedStatus      │        │
│  │  • GetStatusesByPhase(phase) → []string                  │        │
│  └─────────────────────────────────────────────────────────┘        │
│                           │                                          │
│                           ▼                                          │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  Internal Logic:                                         │        │
│  │  • Cache workflow config (loaded once)                   │        │
│  │  • Order statuses by phase                               │        │
│  │  • Apply ANSI color codes                                │        │
│  │  • Respect --no-color flag                               │        │
│  └─────────────────────────────────────────────────────────┘        │
└───────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────────┐
│                   Config Layer (Existing)                            │
│                                                                       │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │  config.LoadWorkflowConfig(configPath)                  │        │
│  │  - Reads .sharkconfig.json                              │        │
│  │  - Parses workflow config                               │        │
│  │  - Caches in memory                                     │        │
│  │  - Returns nil if not found (graceful)                  │        │
│  └─────────────────────────────────────────────────────────┘        │
└───────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────────┐
│                   Repository Layer                                   │
│                                                                       │
│  ┌────────────────────────────────────────────────────────┐         │
│  │  TaskRepository.GetStatusBreakdown()                   │         │
│  │                                                         │         │
│  │  Returns: []StatusCount (ordered slice)                │         │
│  │  - ALL workflow statuses (including zero counts)       │         │
│  │  - Ordered by workflow phase                           │         │
│  │  - Includes metadata (phase, description)              │         │
│  └────────────────────────────────────────────────────────┘         │
└───────────────────────────┬──────────────────────────────────────────┘
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────────┐
│                     Database Layer                                   │
│                                                                       │
│  SELECT status, COUNT(*) FROM tasks GROUP BY status                 │
└──────────────────────────────────────────────────────────────────────┘

BENEFITS:
✅ Single source of truth (WorkflowService)
✅ Consistent status display across all commands
✅ Rich metadata (colors, descriptions, phases)
✅ Deterministic ordering (by workflow phase)
✅ Easy to test and maintain
```

---

## Data Flow: Feature Get Command

### Before (Current)

```
User runs: shark feature get E04-F06
         │
         ▼
    ┌─────────────────────────┐
    │  runFeatureGet()        │
    │  - Get feature from DB  │
    └────────────┬────────────┘
                 │
                 ▼
    ┌─────────────────────────────────────┐
    │  taskRepo.GetStatusBreakdown()      │
    │  Returns: map[TaskStatus]int        │
    │  {                                  │
    │    "todo": 5,                       │
    │    "in_progress": 2,                │
    │    "ready_for_review": 3            │
    │  }                                  │
    └────────────┬────────────────────────┘
                 │
                 ▼
    ┌─────────────────────────────────────┐
    │  renderFeatureDetails()             │
    │  - Iterate map (RANDOM ORDER)       │
    │  - Display status names only        │
    │  - No colors, no descriptions       │
    └─────────────────────────────────────┘
                 │
                 ▼
           Terminal Output:
           Task Status Breakdown
           ready_for_review       3
           in_progress           2
           todo                  5
```

### After (Enhanced)

```
User runs: shark feature get E04-F06
         │
         ▼
    ┌─────────────────────────┐
    │  runFeatureGet()        │
    │  - Get feature from DB  │
    │  - Create WorkflowService│ ←─────┐
    └────────────┬────────────┘        │
                 │                      │
                 ▼                      │
    ┌─────────────────────────────────┐│
    │  WorkflowService               │ │
    │  - Load .sharkconfig.json      │ │
    │  - Parse workflow config       │ │
    │  - Cache in memory             │ │
    └────────────┬────────────────────┘│
                 │                      │
                 ▼                      │
    ┌─────────────────────────────────┐│
    │  taskRepo.GetStatusBreakdown()  │ │
    │  (with workflow config)          │ │
    │  Returns: []StatusCount          │ │
    │  [                               │ │
    │    {Status: "draft", Count: 5,   │ │
    │     Phase: "planning"},          │ │
    │    {Status: "ready_for_dev",     │ │
    │     Count: 2, Phase: "dev"},     │ │
    │    {Status: "ready_for_qa",      │ │
    │     Count: 3, Phase: "qa"}       │ │
    │  ]                               │ │
    └────────────┬────────────────────┘ │
                 │                      │
                 ▼                      │
    ┌─────────────────────────────────┐ │
    │  renderFeatureDetails()         │ │
    │  - Iterate slice (ORDERED)      │ │
    │  - Use WorkflowService for      │─┘
    │    color and metadata           │
    │  - Display with descriptions    │
    └─────────────────────────────────┘
                 │
                 ▼
           Terminal Output:
           Task Status Breakdown
           Status              Count  Phase    Description
           draft               5      planning Task created but not refined
           ready_for_dev       2      dev      Spec complete, ready for coding
           ready_for_qa        3      qa       Ready for testing
           completed           0      done     Finished and approved
```

---

## Status Ordering Algorithm

```
Workflow Config (.sharkconfig.json):
{
  "status_metadata": {
    "draft":                   { "phase": "planning" },
    "ready_for_refinement":    { "phase": "planning" },
    "in_refinement":           { "phase": "planning" },
    "ready_for_development":   { "phase": "development" },
    "in_development":          { "phase": "development" },
    "ready_for_code_review":   { "phase": "review" },
    "in_code_review":          { "phase": "review" },
    "ready_for_qa":            { "phase": "qa" },
    "in_qa":                   { "phase": "qa" },
    "completed":               { "phase": "done" }
  }
}

         │
         ▼
┌────────────────────────────────────────────────┐
│  WorkflowService.GetAllStatuses()              │
│                                                 │
│  1. Group statuses by phase:                   │
│     planning:    [draft, in_refinement,        │
│                   ready_for_refinement]        │
│     development: [in_development,              │
│                   ready_for_development]       │
│     review:      [in_code_review,              │
│                   ready_for_code_review]       │
│     qa:          [in_qa, ready_for_qa]         │
│     done:        [completed]                   │
│                                                 │
│  2. Sort alphabetically within each phase      │
│                                                 │
│  3. Concatenate phases in order:               │
│     planning → development → review → qa → done│
│                                                 │
└────────────────────────────────────────────────┘
         │
         ▼
Result: [
  "draft",                   // planning
  "in_refinement",           // planning
  "ready_for_refinement",    // planning
  "in_development",          // development
  "ready_for_development",   // development
  "in_code_review",          // review
  "ready_for_code_review",   // review
  "in_qa",                   // qa
  "ready_for_qa",            // qa
  "completed"                // done
]
```

---

## Class Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      WorkflowService                         │
├─────────────────────────────────────────────────────────────┤
│ - projectRoot: string                                        │
│ - workflow: *WorkflowConfig                                  │
├─────────────────────────────────────────────────────────────┤
│ + NewService(projectRoot) *Service                           │
│ + GetWorkflow() *WorkflowConfig                              │
│ + GetInitialStatus() TaskStatus                              │
│ + GetAllStatuses() []string                                  │
│ + GetStatusMetadata(status) StatusMetadata                   │
│ + GetStatusesByPhase(phase) []string                         │
│ + FormatStatusForDisplay(status, color) FormattedStatus      │
└────────────────────┬────────────────────────────────────────┘
                     │ uses
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    WorkflowConfig                            │
├─────────────────────────────────────────────────────────────┤
│ + Version: string                                            │
│ + StatusFlow: map[string][]string                            │
│ + StatusMetadata: map[string]StatusMetadata                  │
│ + SpecialStatuses: map[string][]string                       │
├─────────────────────────────────────────────────────────────┤
│ + GetStatusMetadata(status) (StatusMetadata, bool)           │
│ + GetStatusesByAgentType(agentType) []string                 │
│ + GetStatusesByPhase(phase) []string                         │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   StatusMetadata                             │
├─────────────────────────────────────────────────────────────┤
│ + Color: string                                              │
│ + Description: string                                        │
│ + Phase: string                                              │
│ + AgentTypes: []string                                       │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                  FormattedStatus                             │
├─────────────────────────────────────────────────────────────┤
│ + Status: string                                             │
│ + Colored: string                                            │
│ + Description: string                                        │
│ + Phase: string                                              │
│ + ColorName: string                                          │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    StatusCount                               │
├─────────────────────────────────────────────────────────────┤
│ + Status: TaskStatus                                         │
│ + Count: int                                                 │
│ + Phase: string                                              │
│ + Description: string                                        │
└─────────────────────────────────────────────────────────────┘
```

---

## Sequence Diagram: Feature Get with WorkflowService

```
User          CLI           WorkflowService    TaskRepository    Database
 │             │                   │                 │              │
 │  feature get E04-F06            │                 │              │
 ├────────────>│                   │                 │              │
 │             │                   │                 │              │
 │             │  NewService(root) │                 │              │
 │             ├──────────────────>│                 │              │
 │             │                   │ LoadWorkflowConfig             │
 │             │                   ├─────────────────┤              │
 │             │                   │ (cached)        │              │
 │             │                   │<────────────────┤              │
 │             │<──────────────────┤                 │              │
 │             │                   │                 │              │
 │             │  GetByKey(featureKey)               │              │
 │             ├─────────────────────────────────────>│              │
 │             │                   │                 │  SELECT...   │
 │             │                   │                 ├─────────────>│
 │             │                   │                 │<─────────────┤
 │             │<─────────────────────────────────────┤              │
 │             │                   │                 │              │
 │             │  GetStatusBreakdown(featureID)      │              │
 │             ├─────────────────────────────────────>│              │
 │             │                   │                 │  SELECT...   │
 │             │                   │                 ├─────────────>│
 │             │                   │                 │<─────────────┤
 │             │                   │  GetAllStatuses()│              │
 │             │                   │<────────────────┤              │
 │             │                   │  []string       │              │
 │             │                   ├────────────────>│              │
 │             │<─────────────────────────────────────┤              │
 │             │  []StatusCount (ordered)            │              │
 │             │                   │                 │              │
 │             │  FormatStatusForDisplay(status)     │              │
 │             ├──────────────────>│                 │              │
 │             │  FormattedStatus  │                 │              │
 │             │<──────────────────┤                 │              │
 │             │                   │                 │              │
 │             │  renderFeatureDetails()             │              │
 │             ├─────────────────┐ │                 │              │
 │             │                 │ │                 │              │
 │             │<────────────────┘ │                 │              │
 │             │                   │                 │              │
 │  Display    │                   │                 │              │
 │<────────────┤                   │                 │              │
 │             │                   │                 │              │
```

---

## File Structure (After Implementation)

```
shark-task-manager/
├── internal/
│   ├── workflow/                     # NEW PACKAGE
│   │   ├── service.go                # WorkflowService implementation
│   │   ├── service_test.go           # Unit tests
│   │   ├── formatter.go              # Status formatting utilities
│   │   ├── formatter_test.go         # Formatter tests
│   │   └── types.go                  # FormattedStatus type
│   │
│   ├── config/                       # EXISTING
│   │   ├── workflow_schema.go        # WorkflowConfig type
│   │   ├── workflow_parser.go        # LoadWorkflowConfig (cached)
│   │   ├── workflow_default.go       # Default workflow
│   │   └── config.go                 # Config constants (ENHANCED)
│   │
│   ├── repository/                   # EXISTING (ENHANCED)
│   │   ├── task_repository.go        # GetStatusBreakdown returns []StatusCount
│   │   └── types.go                  # StatusCount type (NEW)
│   │
│   ├── cli/commands/                 # EXISTING (ENHANCED)
│   │   ├── feature.go                # Injects WorkflowService
│   │   ├── epic.go                   # Injects WorkflowService
│   │   └── task.go                   # Injects WorkflowService
│   │
│   └── taskcreation/                 # EXISTING (REFACTORED)
│       └── creator.go                # Uses WorkflowService.GetInitialStatus()
│
└── .sharkconfig.json                 # Workflow configuration
```

---

**Diagram Version:** 1.0
**Last Updated:** 2026-01-02
