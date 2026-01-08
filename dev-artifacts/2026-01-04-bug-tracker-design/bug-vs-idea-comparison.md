# Bug Tracker vs Idea Tracker - Comparison

This document compares the bug tracker design with the existing idea tracker to highlight similarities, differences, and design rationale.

---

## Overview

Both trackers follow similar architectural patterns but serve different purposes:

- **Idea Tracker**: Lightweight capture of feature ideas before committing to epics/features/tasks
- **Bug Tracker**: Structured tracking of defects with rich technical context and lifecycle management

---

## Comparison Matrix

| Aspect | Idea Tracker | Bug Tracker | Rationale |
|--------|--------------|-------------|-----------|
| **Purpose** | Capture feature ideas | Track and resolve bugs | Different workflows |
| **Key Format** | `I-YYYY-MM-DD-xx` | `B-YYYY-MM-DD-xx` | Date-based for chronological tracking |
| **Primary Users** | Product managers, stakeholders | Developers, QA, AI agents | Different user personas |
| **Lifecycle** | new → on_hold → converted → archived | new → confirmed → in_progress → resolved → closed | Bug resolution is multi-stage |
| **Technical Fields** | No | Yes (steps, expected/actual, error) | Bugs require technical context |
| **Severity** | Priority only (1-10) | Severity + Priority | Bugs need impact classification |
| **Environment Tracking** | No | Yes (OS, version, environment) | Bugs are environment-specific |
| **Reporter Attribution** | No | Yes (human vs AI agent) | Track bug source |
| **File Attachments** | No explicit support | Yes (`--file` flag) | Bugs often have logs/screenshots |
| **Status Transitions** | Simple (new/hold/converted) | Complex (7 statuses) | Bug resolution is iterative |
| **Timestamps** | created_at, updated_at | + resolved_at, closed_at | Track resolution timeline |
| **Conversion** | To epic/feature/task | To epic/feature/task | Both support conversion |
| **Relationships** | Dependencies only | Dependencies + epic/feature/task links | Bugs relate to existing work |

---

## Detailed Comparisons

### 1. Data Model

#### Idea Model (Simple)
```go
type Idea struct {
    ID           int64
    Key          string  // I-YYYY-MM-DD-xx
    Title        string
    Description  *string
    CreatedDate  time.Time
    Priority     *int
    Order        *int
    Notes        *string
    RelatedDocs  *string  // JSON array
    Dependencies *string  // JSON array
    Status       IdeaStatus  // new, on_hold, converted, archived

    // Conversion tracking
    ConvertedToType *string
    ConvertedToKey  *string
    ConvertedAt     *time.Time

    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

#### Bug Model (Rich)
```go
type Bug struct {
    ID    int64
    Key   string  // B-YYYY-MM-DD-xx
    Title string

    // Core bug information
    Description        *string
    Severity           BugSeverity  // critical, high, medium, low
    Priority           *int
    Category           *string

    // Technical details (NEW)
    StepsToReproduce   *string
    ExpectedBehavior   *string
    ActualBehavior     *string
    ErrorMessage       *string

    // Environment (NEW)
    Environment        *string  // production, staging, development, test
    OSInfo             *string
    Version            *string

    // Reporter (NEW)
    ReporterType       ReporterType  // human, ai_agent
    ReporterID         *string
    DetectedAt         time.Time

    // File references (NEW)
    AttachmentFile     *string
    RelatedDocs        *string  // JSON array

    // Relationships (EXPANDED)
    RelatedToEpic      *string
    RelatedToFeature   *string
    RelatedToTask      *string
    Dependencies       *string  // JSON array

    // Status (MORE COMPLEX)
    Status             BugStatus  // new, confirmed, in_progress, resolved, closed, wont_fix, duplicate
    Resolution         *string

    // Conversion tracking (SAME)
    ConvertedToType    *string
    ConvertedToKey     *string
    ConvertedAt        *time.Time

    // Timestamps (EXPANDED)
    CreatedAt          time.Time
    UpdatedAt          time.Time
    ResolvedAt         *time.Time  // NEW
    ClosedAt           *time.Time  // NEW
}
```

**Key Differences**:
1. **Technical Fields**: Bug has `steps_to_reproduce`, `expected_behavior`, `actual_behavior`, `error_message`
2. **Environment**: Bug tracks `environment`, `os_info`, `version`
3. **Reporter**: Bug tracks `reporter_type` and `reporter_id` for AI agent attribution
4. **Severity**: Bug has explicit severity enum separate from priority
5. **Lifecycle Timestamps**: Bug tracks `resolved_at` and `closed_at` for SLA monitoring
6. **Relationships**: Bug can relate to existing epics/features/tasks (idea tracker only has dependencies on other ideas)
7. **File Support**: Bug has explicit `attachment_file` field for logs/screenshots

---

### 2. CLI Commands

#### Idea Tracker Commands
```bash
shark idea list [--status=<status>] [--priority=<priority>] [--json]
shark idea get <idea-key> [--json]
shark idea create <title> [--description] [--priority] [--notes] [--json]
shark idea update <idea-key> [flags] [--json]
shark idea delete <idea-key> [--force] [--hard] [--json]

shark idea convert epic <idea-key>
shark idea convert feature <idea-key> --epic=<epic>
shark idea convert task <idea-key> --epic=<epic> --feature=<feature>
```

#### Bug Tracker Commands (Extended)
```bash
shark bug list [--status] [--severity] [--category] [--epic] [--feature] [--environment] [--json]
shark bug get <bug-key> [--json]
shark bug create <title> [--description] [--severity] [--priority] [--category]
                         [--steps] [--expected] [--actual] [--error]
                         [--environment] [--os] [--version] [--reporter] [--reporter-type]
                         [--file] [--epic] [--feature] [--task] [--json]
shark bug update <bug-key> [flags] [--json]
shark bug delete <bug-key> [--force] [--hard] [--json]

# Additional status commands (NOT in idea tracker)
shark bug confirm <bug-key> [--notes] [--json]
shark bug resolve <bug-key> [--resolution] [--json]
shark bug close <bug-key> [--notes] [--json]
shark bug reopen <bug-key> [--notes] [--json]

shark bug convert task <bug-key> --epic=<epic> --feature=<feature>
shark bug convert feature <bug-key> --epic=<epic>
shark bug convert epic <bug-key>
```

**Key Differences**:
1. **More Filters**: Bug list supports `--severity`, `--category`, `--epic`, `--feature`, `--environment`
2. **More Creation Flags**: Bug create has 15+ flags vs idea's 8 flags
3. **Status Commands**: Bug has dedicated commands for status transitions (confirm, resolve, close, reopen)
4. **File Support**: Bug create supports `--file` flag for attachments
5. **Reporter Tracking**: Bug create supports `--reporter` and `--reporter-type` flags

---

### 3. Status Lifecycle

#### Idea Status (Simple - 4 States)
```
new ──────> on_hold ──────> converted ──────> archived
 │                               ▲
 └───────────────────────────────┘
```

**Transitions**:
- `new`: Just created
- `on_hold`: Postponed, waiting for capacity/approval
- `converted`: Transformed into epic/feature/task
- `archived`: Discarded, no longer relevant

**Simple Workflow**: Ideas are lightweight; they either get converted or archived.

---

#### Bug Status (Complex - 7 States)
```
new ──────> confirmed ──────> in_progress ──────> resolved ──────> closed
 │               │                  │                  │
 │               │                  v                  │
 │               │              blocked                │
 │               │                  │                  │
 │               v                  v                  v
 └───────────> wont_fix <──────────┴──────────────────┘
                  │
                  v
              duplicate
```

**Transitions**:
1. `new`: Bug reported, not yet verified
2. `confirmed`: Bug reproduced, verified by QA/dev
3. `in_progress`: Developer actively working on fix
4. `resolved`: Fix implemented, awaiting verification
5. `closed`: Fix verified in production
6. `wont_fix`: Bug will not be fixed (design decision, low priority)
7. `duplicate`: Duplicate of another bug

**Complex Workflow**: Bug resolution requires multiple verification stages and has non-fix paths (wont_fix, duplicate).

---

### 4. Use Case Examples

#### Idea Tracker Use Case
```bash
# PM captures feature idea
shark idea create "AI-powered search suggestions" \
  --description="Use ML to suggest search terms based on user history" \
  --priority=7 \
  --notes="Research ElasticSearch ML plugins"

# Later, PM converts to feature
shark idea convert feature I-2026-01-01-05 --epic=E08

# Idea status changes to 'converted'
```

**Scenario**: Lightweight idea capture → decision → conversion

---

#### Bug Tracker Use Case
```bash
# AI agent detects bug during testing
shark bug create "Database connection leak" \
  --severity=critical \
  --priority=9 \
  --category=database \
  --description="Connection pool exhausted after 30 seconds" \
  --steps="1. Run load test\n2. Monitor pool\n3. Observe exhaustion" \
  --expected="Connections released" \
  --actual="Connections leak" \
  --error="TimeoutError: QueuePool limit reached" \
  --environment=test \
  --reporter=test-agent \
  --reporter-type=ai_agent \
  --epic=E07 \
  --feature=E07-F20

# Developer confirms bug
shark bug confirm B-2026-01-04-01 --notes="Reproduced locally"

# Developer converts to task
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20

# Developer fixes bug
shark task complete T-E07-F20-018 --notes="Fixed leak"

# QA resolves bug
shark bug resolve B-2026-01-04-01 --resolution="Fixed in v1.2.4"

# QA verifies fix
shark bug close B-2026-01-04-01 --notes="Verified in production"
```

**Scenario**: Automated detection → human confirmation → task conversion → resolution → verification

---

### 5. AI Agent Integration

#### Idea Tracker - Limited AI Use
Ideas are typically human-generated (product ideas, feature requests). AI agents rarely create ideas.

```bash
# Rare: AI agent suggesting feature idea
shark idea create "Implement caching for user queries" \
  --description="Performance analysis suggests 40% query time reduction with Redis cache" \
  --priority=8 \
  --notes="Recommendation from performance profiler agent"
```

---

#### Bug Tracker - Heavy AI Use
Bugs are frequently detected and reported by AI agents during automated testing, monitoring, and profiling.

```bash
# Common: AI agent reports bug during automated testing
shark bug create "Test failure: test_user_authentication" \
  --severity=high \
  --category=backend \
  --error="AssertionError: Expected 401, got 500" \
  --reporter=test-agent \
  --reporter-type=ai_agent \
  --environment=test

# Common: AI agent reports performance issue
shark bug create "API response time exceeds SLA" \
  --severity=medium \
  --category=performance \
  --description="Average response time: 1250ms (SLA: 1000ms)" \
  --reporter=performance-monitor \
  --reporter-type=ai_agent \
  --environment=production

# Common: AI agent reports production error spike
shark bug create "Error rate spike: 500 errors/hour" \
  --severity=critical \
  --file=/tmp/error-log-samples.txt \
  --reporter=error-monitor \
  --reporter-type=ai_agent \
  --environment=production
```

**Key Point**: Bug tracker is designed for AI-first workflows with `--reporter-type` and rich automation support.

---

### 6. File Attachment Support

#### Idea Tracker - No File Support
Ideas use `--related-docs` for document paths, but no dedicated file attachment mechanism.

```bash
shark idea create "New feature" \
  --related-docs="docs/design/feature-sketch.md,docs/market-research.pdf"
```

**Limitation**: Can reference documents but cannot attach large logs or screenshots directly.

---

#### Bug Tracker - Explicit File Support
Bugs support `--file` flag for attaching large stack traces, log excerpts, screenshot references.

```bash
# Attach large stack trace
shark bug create "App crashes on startup" \
  --file=/tmp/crash-stacktrace.txt \
  --severity=critical

# Attach comprehensive bug report
cat > /tmp/bug-details.md <<EOF
## Log Files
- /var/log/app.log (lines 1234-5678)
- /var/log/db.log (lines 890-1234)

## Screenshots
- screenshots/error-screen.png
- screenshots/expected-screen.png

## Stack Trace
[500 lines of stack trace]
EOF

shark bug create "Multi-system failure" --file=/tmp/bug-details.md
```

**Advantage**: Bugs can include extensive technical details without cluttering CLI flags.

---

### 7. Filtering & Querying

#### Idea Tracker - Simple Filters
```bash
shark idea list --status=new
shark idea list --priority=8
shark idea list --json | jq '.[] | select(.priority > 7)'
```

**Limited Dimensions**: Status, priority

---

#### Bug Tracker - Rich Filters
```bash
shark bug list --status=new
shark bug list --severity=critical
shark bug list --category=backend
shark bug list --environment=production
shark bug list --epic=E07
shark bug list --feature=E07-F20

# Complex queries
shark bug list --severity=critical --category=backend --environment=production
shark bug list --json | jq '.[] | select(.reporter_type == "ai_agent")'
shark bug list --json | jq '.[] | select(.priority > 8 and .status == "new")'
```

**Multiple Dimensions**: Status, severity, priority, category, environment, epic, feature, task, reporter type

---

## Design Rationale

### Why Bug Tracker Is More Complex

1. **Technical Nature**: Bugs are technical problems requiring structured debugging information (steps, expected/actual, errors)

2. **Lifecycle Complexity**: Bug resolution is iterative (report → confirm → fix → verify → close) vs idea's linear path (create → convert)

3. **Environment Sensitivity**: Bugs are environment-specific (production vs staging), ideas are environment-agnostic

4. **AI Agent Use**: Bugs are frequently detected by automated systems, requiring reporter attribution and machine-readable formats

5. **Stakeholder Diversity**: Bugs involve multiple roles (developers, QA, ops, product) vs ideas (primarily product/business)

6. **SLA Tracking**: Bugs need timestamps (detected, resolved, closed) for SLA monitoring and metrics

7. **Evidence Requirements**: Bugs require supporting evidence (logs, screenshots, stack traces), ideas are descriptive

### Why Keep Idea Tracker Simple

1. **Lightweight Capture**: Ideas should be quick to record without ceremony

2. **Early Stage**: Ideas are pre-commitment; don't need detailed technical specs yet

3. **Human-Centric**: Ideas are primarily human-generated and human-reviewed

4. **Exploratory**: Ideas are about possibilities, not problems

---

## Shared Patterns

Despite differences, both trackers share core patterns:

1. **Date-Based Keys**: Both use `X-YYYY-MM-DD-xx` format for chronological tracking
2. **Conversion Tracking**: Both support conversion to epic/feature/task with audit trail
3. **Status Enum**: Both use status enums (though bug's is more complex)
4. **JSON Output**: Both support `--json` for programmatic access
5. **Repository Pattern**: Both use repository layer with `Create/GetByKey/List/Update/Delete`
6. **Timestamps**: Both track `created_at`, `updated_at`
7. **Priority**: Both support 1-10 priority scale
8. **Dependencies**: Both support dependencies (ideas on ideas, bugs on bugs)
9. **Soft Delete**: Both support soft delete (idea=archive, bug=wont_fix/duplicate)

---

## Migration Considerations

### No Conflict Between Trackers

- Separate tables: `ideas` vs `bugs`
- Separate key prefixes: `I-` vs `B-`
- Separate commands: `shark idea` vs `shark bug`
- No schema overlap or migration required

### Shared Infrastructure

Both trackers reuse:
- Database layer (`internal/db/`)
- Repository pattern (`internal/repository/`)
- CLI framework (`internal/cli/`)
- Validation (`internal/models/validation.go`)
- JSON utilities (`encoding/json`)

---

## Summary

| **Aspect** | **Idea Tracker** | **Bug Tracker** |
|------------|------------------|-----------------|
| **Complexity** | Simple (8 fields) | Rich (25+ fields) |
| **Use Case** | Feature ideation | Defect tracking |
| **Primary Users** | Product/Business | Dev/QA/Ops |
| **AI Agent Role** | Minimal | Central |
| **Lifecycle** | Linear (4 states) | Iterative (7 states) |
| **Technical Details** | None | Extensive |
| **File Support** | Reference only | Direct attachment |
| **Environment** | N/A | Critical |
| **Filtering** | 2 dimensions | 8+ dimensions |

**Conclusion**: Bug tracker extends idea tracker patterns with richer data model and workflow tailored for technical defect tracking, while maintaining architectural consistency with shark's existing design.

---

**End of Comparison**
