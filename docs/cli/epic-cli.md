# Epic CLI Commands

Complete reference for epic management commands.

## Overview

Epics are the top-level organizational units in Shark's three-tier hierarchy (Epic → Feature → Task). They represent major product initiatives with multi-phase workflows from ideation through decomposition.

**Epic Lifecycle:**
```
draft → refinement → research → feasibility → test planning → decomposition → active → completed
```

## Commands

### `shark epic create`

Create a new epic.

#### Syntax

```bash
shark epic create --title="Epic Title" [flags]
```

#### Required Flags

- `--title` - Epic title (string)

#### Optional Flags

- `--file=<path>` - Custom file path (relative to project root, must end in .md)
- `--force` - Force reassignment if file already claimed
- `--priority=<1-10>` - Epic priority (default: 5)
- `--business-value=<1-10>` - Business value rating (default: 5)
- `--json` - Output JSON response

#### Examples

**Basic epic creation:**
```bash
shark epic create --title="User Authentication System"
```

**Output:**
```
Epic created: E07 - User Authentication System
File: docs/plan/E07-user-authentication-system/epic.md
Status: draft
```

**Custom file path:**
```bash
shark epic create --title="Q1 2025 Roadmap" --file="docs/roadmap/2025-q1/epic.md"
```

**With priority and business value:**
```bash
shark epic create --title="Payment Gateway Integration" --priority=10 --business-value=10
```

**JSON output:**
```bash
shark epic create --title="User Auth" --json
```

```json
{
  "id": 7,
  "key": "E07",
  "slug": "user-auth",
  "title": "User Auth",
  "status": "draft",
  "priority": 5,
  "business_value": 5,
  "file_path": "docs/plan/E07-user-auth/epic.md",
  "created_at": "2026-02-16T10:30:00Z",
  "updated_at": "2026-02-16T10:30:00Z"
}
```

#### Behavior

1. Generates epic key (e.g., `E07`)
2. Creates slug from title (e.g., "User Auth" → `user-auth`)
3. Creates directory structure: `docs/plan/E##-slug/`
4. Creates `epic.md` file with frontmatter
5. Inserts record in database
6. Sets initial status to `draft`

#### File Structure

**Generated file** (`docs/plan/E07-user-auth/epic.md`):

```markdown
---
epic_key: E07
title: User Auth
status: draft
priority: 5
business_value: 5
created_at: 2026-02-16T10:30:00Z
updated_at: 2026-02-16T10:30:00Z
---

# User Auth

[Epic description goes here]
```

---

### `shark epic list`

List all epics with progress information.

#### Syntax

```bash
shark epic list [flags]
```

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**Human-readable table:**
```bash
shark epic list
```

**Output:**
```
Key  Title                          Status                Progress  Features
────────────────────────────────────────────────────────────────────────────
E04  Task Management CLI Core       active                   75.0%         7
E07  User Authentication System     in_decomposition         45.0%         3
E08  Analytics Dashboard            draft                     0.0%         0
```

**JSON output:**
```bash
shark epic list --json
```

```json
[
  {
    "id": 4,
    "key": "E04",
    "slug": "task-management-cli-core",
    "title": "Task Management CLI Core",
    "status": "active",
    "priority": 10,
    "business_value": 10,
    "file_path": "docs/plan/E04-task-management-cli-core/epic.md",
    "created_at": "2025-12-01T08:00:00Z",
    "updated_at": "2026-02-16T10:15:00Z",
    "progress_pct": 75.0,
    "feature_count": 7
  },
  {
    "id": 7,
    "key": "E07",
    "slug": "user-authentication-system",
    "title": "User Authentication System",
    "status": "in_decomposition",
    "priority": 8,
    "business_value": 9,
    "file_path": "docs/plan/E07-user-authentication-system/epic.md",
    "created_at": "2026-01-15T14:20:00Z",
    "updated_at": "2026-02-16T10:30:00Z",
    "progress_pct": 45.0,
    "feature_count": 3
  }
]
```

#### Progress Calculation

Epic progress is **aggregated from child features** when in `active` status:

```
Epic Progress = Σ(Feature Progress) / Feature Count
```

For planning statuses (draft, refinement, research), progress is based on status metadata `progress_weight`.

---

### `shark epic get`

Get detailed information about a specific epic.

#### Syntax

```bash
shark epic get <epic-key> [flags]
```

#### Arguments

- `<epic-key>` - Epic key (case-insensitive, numeric or slugged)

#### Optional Flags

- `--json` - Output JSON response

#### Examples

**Numeric key:**
```bash
shark epic get E07
```

**Slugged key:**
```bash
shark epic get E07-user-authentication-system
```

**Case insensitive:**
```bash
shark epic get e07
```

**Human-readable output:**
```
═══════════════════════════════════════════════════════════════════
Epic: E07 - User Authentication System
═══════════════════════════════════════════════════════════════════

Status: in_decomposition
Phase: decomposition
Priority: 8
Business Value: 9
Progress: 45.0%

File: docs/plan/E07-user-authentication-system/epic.md

Created: 2026-01-15 14:20:00
Updated: 2026-02-16 10:30:00

───────────────────────────────────────────────────────────────────
Features (3)
───────────────────────────────────────────────────────────────────

Key          Title                    Status              Progress
──────────────────────────────────────────────────────────────────
E07-F01      JWT Token Management     active                 80.0%
E07-F02      OAuth Integration        in_refinement_tech     20.0%
E07-F03      Session Management       draft                   0.0%

───────────────────────────────────────────────────────────────────
Task Status Rollup
───────────────────────────────────────────────────────────────────

Status                  Count
─────────────────────────────
completed                  12
ready_for_qa                3
in_development              5
ready_for_development       8
ready_for_refinement_ba     4
draft                      10

Total Tasks: 42
```

**JSON output:**
```bash
shark epic get E07 --json
```

```json
{
  "epic": {
    "id": 7,
    "key": "E07",
    "slug": "user-authentication-system",
    "title": "User Authentication System",
    "status": "in_decomposition",
    "priority": 8,
    "business_value": 9,
    "file_path": "docs/plan/E07-user-authentication-system/epic.md",
    "created_at": "2026-01-15T14:20:00Z",
    "updated_at": "2026-02-16T10:30:00Z"
  },
  "progress_pct": 45.0,
  "features": [
    {
      "id": 201,
      "key": "E07-F01",
      "slug": "jwt-token-management",
      "title": "JWT Token Management",
      "status": "active",
      "progress_pct": 80.0,
      "task_count": 15
    },
    {
      "id": 202,
      "key": "E07-F02",
      "slug": "oauth-integration",
      "title": "OAuth Integration",
      "status": "in_refinement_tech",
      "progress_pct": 20.0,
      "task_count": 8
    }
  ],
  "task_status_rollup": {
    "completed": 12,
    "ready_for_qa": 3,
    "in_development": 5,
    "ready_for_development": 8,
    "ready_for_refinement_ba": 4,
    "draft": 10
  },
  "total_tasks": 42
}
```

---

### `shark epic update`

Update epic fields.

#### Syntax

```bash
shark epic update <epic-key> [flags]
```

#### Arguments

- `<epic-key>` - Epic key (case-insensitive)

#### Optional Flags

- `--title="New Title"` - Update title
- `--status=<status>` - Update status
- `--priority=<1-10>` - Update priority
- `--business-value=<1-10>` - Update business value
- `--note="Update reason"` - Add note to history
- `--json` - Output JSON response

#### Examples

**Update title:**
```bash
shark epic update E07 --title="Authentication and Authorization System"
```

**Update status:**
```bash
shark epic update E07 --status=ready_for_research
```

**Update priority:**
```bash
shark epic update E07 --priority=10
```

**Multiple fields with note:**
```bash
shark epic update E07 --priority=10 --business-value=10 --note="Elevated to highest priority"
```

#### Status Transitions

Status updates must follow configured `status_flow`. See [workflow-configuration.md](workflow-configuration.md) for valid transitions.

**Example flow:**
```
draft → ready_for_refinement → in_refinement → ready_for_research →
in_research → ready_for_feasibility_review_ba → ... → active → completed
```

**Invalid transition:**
```bash
shark epic update E07 --status=completed
# Error: Invalid transition from 'draft' to 'completed'
```

---

### `shark epic next-status`

Advance epic to next valid status.

#### Syntax

```bash
shark epic next-status <epic-key> [flags]
```

#### Arguments

- `<epic-key>` - Epic key (case-insensitive)

#### Optional Flags

- `--note="Advancement reason"` - Add note to history
- `--json` - Output JSON response

#### Examples

```bash
shark epic next-status E07
```

**Output:**
```
Epic E07 advanced: draft → ready_for_refinement
```

**With note:**
```bash
shark epic next-status E07 --note="PRD complete, ready for research"
```

#### Behavior

1. Checks current status
2. Looks up valid transitions from `status_flow`
3. Advances to first valid status
4. Records transition in history
5. Triggers status cascade (updates features if needed)

**If multiple next statuses exist:**
```json
{
  "draft": ["ready_for_refinement", "active", "cancelled"]
}
```

Defaults to first option (`ready_for_refinement`). Use `shark epic update --status` for explicit choice.

---

### `shark epic note`

Add a note to epic without changing status.

#### Syntax

```bash
shark epic note <epic-key> "<note-text>" [flags]
```

#### Arguments

- `<epic-key>` - Epic key (case-insensitive)
- `<note-text>` - Note content (quoted string)

#### Optional Flags

- `--metadata <key>=<value>` - Add structured metadata
- `--json` - Output JSON response

#### Examples

**Simple note:**
```bash
shark epic note E07 "Stakeholder meeting scheduled for Feb 20"
```

**Note with metadata:**
```bash
shark epic note E07 "Complexity assessment complete" --metadata complexity_tier=COMPLEX --metadata complexity_score=14
```

#### Behavior

- Adds note to epic history table
- Does NOT change epic status
- Searchable via history queries
- Metadata stored as JSON

**Use cases:**
- Meeting notes
- Decision rationale
- Complexity assessment results
- Risk flags
- Stakeholder feedback

---

### `shark epic history`

View epic change history.

#### Syntax

```bash
shark epic history <epic-key> [flags]
```

#### Arguments

- `<epic-key>` - Epic key (case-insensitive)

#### Optional Flags

- `--json` - Output JSON response
- `--limit=<n>` - Limit results (default: 50)

#### Examples

```bash
shark epic history E07
```

**Output:**
```
────────────────────────────────────────────────────────────────────
Epic E07 Change History
────────────────────────────────────────────────────────────────────

Date                  From Status       To Status             Notes
──────────────────────────────────────────────────────────────────
2026-02-16 10:30:00   in_research      ready_for_decomp...    Research complete
2026-02-15 14:20:00   ready_for_res... in_research            Agent started
2026-02-15 09:00:00   in_refinement    ready_for_research     PRD approved
2026-02-14 16:45:00   ready_for_ref... in_refinement          BA assigned
2026-02-14 10:00:00   draft            ready_for_refinement   Ready for PRD
```

---

### `shark get` (Smart Dispatcher)

Auto-detect entity type and get details (epic, feature, or task).

#### Syntax

```bash
shark get <key> [flags]
```

#### Key Format Detection

**Epic keys:**
- `E##` (e.g., `E07`)
- `E##-slug` (e.g., `E07-user-auth`)

**Feature keys:**
- `F##` (e.g., `F01`)
- `E##-F##` (e.g., `E07-F01`)
- `E##-F##-slug` (e.g., `E07-F01-jwt-tokens`)

**Task keys:**
- `E##-F##-###` (e.g., `E07-F01-001`)
- `T-E##-F##-###` (e.g., `T-E07-F01-001`)

#### Examples

```bash
# Get epic
shark get E07

# Get feature
shark get E07-F01

# Get task
shark get E07-F01-001
```

---

## Common Workflows

### Create Epic and Begin Refinement

```bash
# 1. Create epic
shark epic create --title="User Authentication System" --priority=8

# 2. Advance to refinement
shark epic next-status E07

# 3. Verify status
shark epic get E07
```

### Update Epic During Development

```bash
# Add progress note
shark epic note E07 "Feature F01 completed, F02 in progress"

# Check overall progress
shark epic get E07

# View recent changes
shark epic history E07 --limit=10
```

### Complete Epic

```bash
# Verify all features complete
shark feature list E07

# Advance to completed
shark epic update E07 --status=completed --note="All features delivered"
```

## Key Concepts

### Epic Status Flow

**Planning Phase:**
1. `draft` - Initial capture
2. `ready_for_refinement` - Ready for PRD
3. `in_refinement` - BA writing PRD
4. `ready_for_research` - PRD complete, ready for research
5. `in_research` - Research in progress

**Feasibility Phase:**
6. `ready_for_feasibility_review_ba` - Business feasibility assessment
7. `in_feasibility_review_ba` - BA reviewing
8. `ready_for_feasibility_review_tech` - Technical feasibility assessment
9. `in_feasibility_review_tech` - Architect reviewing
10. `intervention_required` - Human decision needed

**Execution Phase:**
11. `ready_for_test_planning` - UAT planning
12. `in_test_planning` - QA creating UAT plan
13. `ready_for_decomposition` - Ready to break into features
14. `in_decomposition` - PM generating features
15. `active` - Features exist, work in progress

**Terminal Phases:**
16. `completed` - All features done
17. `cancelled` - Epic abandoned
18. `on_hold` - Paused
19. `blocked` - External blocker

### Progress Calculation

**Planning statuses** (draft → decomposition):
- Progress = status metadata `progress_weight`
- Example: `in_refinement` = 15%

**Active status:**
- Progress = aggregated from child features
- Formula: `Σ(Feature Progress) / Feature Count`

### Agent Routing

Epics spawn different agents based on status:
- `ready_for_refinement` → business-analyst (writes PRD)
- `ready_for_research` → researcher (conducts research)
- `ready_for_feasibility_review_ba` → business-analyst (business feasibility)
- `ready_for_feasibility_review_tech` → architect (technical feasibility)
- `ready_for_test_planning` → qa (creates UAT plan)
- `ready_for_decomposition` → product-manager (generates features)

## Related Documentation

- **[feature-cli.md](feature-cli.md)** - Feature commands
- **[task-cli.md](task-cli.md)** - Task commands
- **[workflow-configuration.md](workflow-configuration.md)** - Status flows
- **[template-system.md](template-system.md)** - Agent instructions
