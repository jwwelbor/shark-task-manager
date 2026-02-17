# Feature CLI Commands

Complete reference for feature management commands.

## Overview

Features are mid-level organizational units in Shark's hierarchy (Epic → **Feature** → Task). They represent user-facing capabilities or technical improvements, following a complexity-adaptive workflow.

**Feature Lifecycle:**
```
draft → scope_validation → triage → [SIMPLE→tasks | STANDARD→refinement | COMPLEX→research]
→ refinement → architecture → test_planning → task_generation → ready_to_build → active → completed
```

## Commands

### `shark feature create`

Create a new feature in an epic.

#### Syntax (Positional - Recommended)

```bash
shark feature create <epic-key> "<title>" [flags]
```

#### Syntax (Flag-Based - Legacy)

```bash
shark feature create --epic=<epic-key> --title="<title>" [flags]
```

#### Arguments

- `<epic-key>` - Parent epic key (case-insensitive)
- `<title>` - Feature title (quoted string)

#### Optional Flags

- `--file=<path>` - Custom file path (relative to project root, must end in .md)
- `--force` - Force reassignment if file already claimed
- `--execution-order=<n>` - Execution order within epic (default: auto)
- `--json` - Output JSON response

#### Examples

**Positional syntax (recommended):**
```bash
shark feature create E07 "JWT Token Management"
```

**Flag syntax (legacy):**
```bash
shark feature create --epic=E07 --title="JWT Token Management"
```

**Custom file path:**
```bash
shark feature create E07 "JWT Tokens" --file="docs/roadmap/2025-q1/jwt-feature.md"
```

**With execution order:**
```bash
shark feature create E07 "OAuth Integration" --execution-order=2
```

**JSON output:**
```bash
shark feature create E07 "JWT Tokens" --json
```

```json
{
  "id": 201,
  "key": "E07-F01",
  "slug": "jwt-token-management",
  "title": "JWT Token Management",
  "epic_id": 7,
  "epic_key": "E07",
  "status": "draft",
  "execution_order": 1,
  "file_path": "docs/plan/E07-user-authentication-system/E07-F01-jwt-token-management/feature.md",
  "created_at": "2026-02-16T10:30:00Z",
  "updated_at": "2026-02-16T10:30:00Z"
}
```

---

### `shark feature list`

List features with optional epic filtering.

#### Syntax (Positional - Recommended)

```bash
shark feature list [EPIC] [flags]
```

#### Syntax (Flag-Based - Legacy)

```bash
shark feature list [--epic=<epic-key>] [flags]
```

#### Optional Arguments

- `[EPIC]` - Epic key to filter by (case-insensitive)

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**All features:**
```bash
shark feature list
```

**Positional epic filter:**
```bash
shark feature list E07
```

**Flag-based epic filter:**
```bash
shark feature list --epic=E07
```

**Output:**
```
Key         Title                    Status                  Progress  Tasks  Health
────────────────────────────────────────────────────────────────────────────────────
E07-F01     JWT Token Management     active                    80.0%     15  ●
E07-F02     OAuth Integration        in_refinement_tech        20.0%      8  ●
E07-F03     Session Management       draft                      0.0%      0  ○
```

**Legend:**
- `●` - Healthy (no blockers, progressing normally)
- `◐` - Warning (minor issues, some blocked tasks)
- `○` - Critical (major blockers, stalled)

---

### `shark feature get`

Get detailed feature information.

#### Syntax

```bash
shark feature get <feature-key> [flags]
```

#### Arguments

- `<feature-key>` - Feature key (case-insensitive, short or full form)

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**Full key:**
```bash
shark feature get E07-F01
```

**Short key:**
```bash
shark feature get F01
```

**Case insensitive:**
```bash
shark feature get e07-f01
```

**Human-readable output:**
```
═══════════════════════════════════════════════════════════════════
Feature: E07-F01 - JWT Token Management
═══════════════════════════════════════════════════════════════════

Epic: E07 - User Authentication System
Status: active
Phase: execution
Execution Order: 1
Health: ● Healthy

File: docs/plan/.../E07-F01-jwt-token-management/feature.md

Progress: 80.0% (weighted) | 73.3% (completion)
Total Tasks: 15

Created: 2026-01-20 09:00:00
Updated: 2026-02-16 10:45:00

───────────────────────────────────────────────────────────────────
Work Breakdown
───────────────────────────────────────────────────────────────────

Agent Work:          8 tasks
Human Work:          2 tasks
QA Work:             1 task
No Assignment:       1 task
Blocked:             0 tasks

───────────────────────────────────────────────────────────────────
Action Items (3 tasks require attention)
───────────────────────────────────────────────────────────────────

ready_for_qa (1):
  T-E07-F01-012 - Token refresh endpoint tests

ready_for_approval (2):
  T-E07-F01-013 - Token validation middleware
  T-E07-F01-014 - JWT secret rotation

───────────────────────────────────────────────────────────────────
Task Breakdown
───────────────────────────────────────────────────────────────────

Status                  Count
────────────────────────────
completed                  11
ready_for_approval          2
ready_for_qa                1
in_development              1

Total: 15 tasks
```

**JSON output:**
```bash
shark feature get E07-F01 --json
```

```json
{
  "feature": {
    "id": 201,
    "key": "E07-F01",
    "slug": "jwt-token-management",
    "title": "JWT Token Management",
    "epic_id": 7,
    "epic_key": "E07",
    "status": "active",
    "execution_order": 1,
    "file_path": "docs/plan/.../feature.md",
    "created_at": "2026-01-20T09:00:00Z",
    "updated_at": "2026-02-16T10:45:00Z"
  },
  "progress": {
    "weighted_pct": 80.0,
    "completion_pct": 73.3,
    "total_tasks": 15
  },
  "work_breakdown": {
    "agent": 8,
    "human": 2,
    "qa_team": 1,
    "none": 1,
    "blocked": 0
  },
  "action_items": {
    "ready_for_qa": [
      {
        "key": "T-E07-F01-012",
        "title": "Token refresh endpoint tests"
      }
    ],
    "ready_for_approval": [
      {
        "key": "T-E07-F01-013",
        "title": "Token validation middleware"
      },
      {
        "key": "T-E07-F01-014",
        "title": "JWT secret rotation"
      }
    ]
  },
  "health": "healthy"
}
```

---

### `shark feature update`

Update feature fields.

#### Syntax

```bash
shark feature update <feature-key> [flags]
```

#### Arguments

- `<feature-key>` - Feature key (case-insensitive)

#### Optional Flags

- `--title="New Title"` - Update title
- `--status=<status>` - Update status
- `--execution-order=<n>` - Update execution order
- `--note="Update reason"` - Add note to history
- `--json` - Output JSON response

#### Examples

**Update title:**
```bash
shark feature update E07-F01 --title="JWT Token Generation and Validation"
```

**Update status:**
```bash
shark feature update E07-F01 --status=ready_for_test_planning
```

**Update execution order:**
```bash
shark feature update E07-F01 --execution-order=1
```

**Multiple fields:**
```bash
shark feature update E07-F01 --status=ready_to_build --note="All tasks generated and specified"
```

---

### `shark feature next-status`

Advance feature to next valid status.

#### Syntax

```bash
shark feature next-status <feature-key> [flags]
```

#### Arguments

- `<feature-key>` - Feature key (case-insensitive)

#### Optional Flags

- `--note="Advancement reason"` - Add note to history
- `--json` - Output JSON response

#### Examples

```bash
shark feature next-status E07-F01
```

**Output:**
```
Feature E07-F01 advanced: draft → ready_for_scope_validation
```

---

### `shark feature note`

Add a note to feature.

#### Syntax

```bash
shark feature note <feature-key> "<note-text>" [flags]
```

#### Arguments

- `<feature-key>` - Feature key (case-insensitive)
- `<note-text>` - Note content (quoted string)

#### Optional Flags

- `--metadata <key>=<value>` - Add structured metadata
- `--json` - Output JSON response

#### Examples

**Simple note:**
```bash
shark feature note E07-F01 "Architecture review complete"
```

**Note with complexity metadata:**
```bash
shark feature note E07-F01 "Complexity assessed" --metadata complexity_tier=STANDARD --metadata complexity_score=6
```

---

### `shark feature history`

View feature change history.

#### Syntax

```bash
shark feature history <feature-key> [flags]
```

#### Arguments

- `<feature-key>` - Feature key (case-insensitive)

#### Optional Flags

- `--json` - Output JSON response
- `--limit=<n>` - Limit results (default: 50)

#### Examples

```bash
shark feature history E07-F01
```

---

## Key Concepts

### Complexity-Adaptive Workflow

Features follow different paths based on complexity tier:

**SIMPLE tier (score 0-3):**
```
draft → scope_validation → triage → ready_for_task_generation → tasks → ready_to_build → active
```

**STANDARD tier (score 4-7):**
```
draft → scope_validation → triage → ready_for_refinement_ba →
ready_for_refinement_tech → ready_for_test_planning →
ready_for_task_generation → ready_to_build → active
```

**COMPLEX tier (score 8+):**
```
draft → scope_validation → triage → ready_for_research →
ready_for_refinement_ba → ready_for_refinement_tech →
ready_for_test_planning → ready_for_task_generation → ready_to_build → active
```

### Complexity Scoring

Features are scored across 6 dimensions (0-3 each, max 18):

1. **File Impact**: 1-3 files=0, 4-10 files=2, 10+ files=3
2. **Pattern Novelty**: existing=0, adapt=1, minor new=2, major new=3
3. **Data Model**: no change=0, modify tables=2, new tables=3
4. **API Surface**: no change=0, modify endpoints=2, new endpoints=3
5. **Cross-Feature Deps**: isolated=0, 1-2 deps=1, 3+ deps=2, circular=3
6. **UI Complexity**: modify=0, new simple=1, new complex=2, interactive=3

**Example:**
- File Impact: 8 files → 2 points
- Pattern Novelty: Adapt existing → 1 point
- Data Model: New tables → 3 points
- API Surface: New endpoints → 3 points
- Cross-Feature: Isolated → 0 points
- UI: Modify existing → 0 points
- **Total: 9 points → COMPLEX tier**

### Feature Status Flow

**Validation Phase:**
1. `draft` - Initial creation
2. `ready_for_scope_validation` - Validate it's a feature (not task)
3. `in_scope_validation` - BA validating scope

**Triage Phase:**
4. `ready_for_triage` - Ready for complexity assessment
5. `in_triage` - Researcher scoring complexity
   - Routes to: `ready_for_task_generation` (SIMPLE) | `ready_for_refinement_ba` (STANDARD) | `ready_for_research` (COMPLEX)

**Research Phase (COMPLEX only):**
6. `ready_for_research` - Ready for codebase research
7. `in_research` - Researcher investigating

**Refinement Phase (STANDARD/COMPLEX):**
8. `ready_for_refinement_ba` - Ready for feature PRD
9. `in_refinement_ba` - BA writing PRD
10. `ready_for_refinement_tech` - Ready for architecture
11. `in_refinement_tech` - Architect designing

**Test Planning Phase (STANDARD/COMPLEX):**
12. `ready_for_test_planning` - Ready for test strategy
13. `in_test_planning` - QA creating test plan

**Decomposition Phase (All tiers):**
14. `ready_for_task_generation` - Ready to generate tasks
15. `in_task_generation` - PM generating tasks

**Execution Phase:**
16. `ready_to_build` - All tasks specified, ready for autonomous build
17. `active` - Build in progress (aggregates from task progress)

**Terminal Phases:**
18. `completed` - All tasks done
19. `cancelled` - Feature abandoned
20. `on_hold` - Paused
21. `blocked` - External blocker

### Progress Calculation

**Planning statuses:**
- Progress = status metadata `progress_weight`

**Active status:**
- Progress = aggregated from child tasks
- Weighted: `Σ(task_weight × task_count) / total_tasks`
- Completion: `completed_tasks / total_tasks`

### Health Status

**Healthy (●):**
- No blocked tasks
- All approval tasks < 3 days old
- Progressing normally

**Warning (◐):**
- Minor blockers (1-2 tasks)
- Approval tasks 3-7 days old
- Some stalled tasks

**Critical (○):**
- Multiple blockers (3+ tasks)
- Approval tasks > 7 days old
- No progress in 7+ days

### Agent Routing

Features spawn different agents based on status:
- `ready_for_scope_validation` → business-analyst (scope validation)
- `ready_for_triage` → researcher (complexity assessment)
- `ready_for_research` → researcher (codebase research)
- `ready_for_refinement_ba` → business-analyst (PRD)
- `ready_for_refinement_tech` → architect (architecture)
- `ready_for_test_planning` → qa (test strategy)
- `ready_for_task_generation` → product-manager (tasks)
- `ready_to_build` → tech-director (autonomous build)

## Common Workflows

### Create Feature and Validate Scope

```bash
# 1. Create feature
shark feature create E07 "JWT Token Management"

# 2. Advance to scope validation
shark feature next-status E07-F01

# 3. Agent validates scope (spawned automatically)
# If valid feature: advances to triage
# If misclassified task: converts to task under enhancement feature

# 4. Verify status
shark feature get E07-F01
```

### Triage and Route Based on Complexity

```bash
# 1. Feature in ready_for_triage
shark feature get E07-F01

# 2. Agent scores complexity (spawned automatically)
# SIMPLE (0-3): Routes to ready_for_task_generation
# STANDARD (4-7): Routes to ready_for_refinement_ba
# COMPLEX (8+): Routes to ready_for_research

# 3. View complexity metadata
shark feature get E07-F01 --json | jq '.metadata.complexity_tier'
```

### Monitor Feature Progress

```bash
# View detailed progress
shark feature get E07-F01

# Check action items
shark feature get E07-F01 | grep "Action Items" -A 20

# List all features in epic
shark feature list E07
```

## Related Documentation

- **[epic-cli.md](epic-cli.md)** - Epic commands
- **[task-cli.md](task-cli.md)** - Task commands
- **[workflow-configuration.md](workflow-configuration.md)** - Status flows
- **[template-system.md](template-system.md)** - Agent instructions
