---
feature_key: E07-F34-template-variable-enrichment
epic_key: E07
title: Template Variable Enrichment
description: Add new template variables (previous_status, parent_title, context_data fields, latest_note, sibling_progress) to the orchestrator template system so templates can provide richer context to agents without requiring CLI round-trips.
---

# Template Variable Enrichment

**Feature Key**: E07-F34-template-variable-enrichment

---

## Goal

### Problem

Orchestrator templates currently expose only direct model fields and relationship data via the `*PlaceholdersWithRelated()` functions in `internal/config/template_helpers.go`. Agents receiving these templates frequently need additional context — parent entity titles, previous status for branching on rejection loops, structured context_data fields, and recent notes — but must make separate CLI calls (`shark get`, `shark notes`, `shark feature context get`) to obtain it. Each CLI call is a full shark process invocation including database connection setup, which is especially costly with our Turso cloud database backend where every query is a remote round-trip.

### Solution

Enrich the `TaskPlaceholdersWithRelated()`, `FeaturePlaceholdersWithRelated()`, and `EpicPlaceholdersWithRelated()` functions to populate additional template variables from data already accessible through existing repository interfaces. This eliminates the most common "first thing the agent does is run a CLI command" patterns identified in the template variable audit.

### Impact

- Eliminate 2-4 CLI calls per template invocation for the most common agent workflows
- Reduce agent startup latency, especially with Turso cloud database
- Enable smarter template branching (first-time vs rejection loop, resume vs fresh start)
- Templates become more self-contained and reliable

---

## Architecture Note: Database Query Strategy

The project uses Turso cloud database where every query is a remote round-trip. The existing codebase uses **large views and joins** to minimize the number of round-trips. When implementing these new placeholder variables, we should follow the same pattern:

- **Prefer a single query that fetches multiple related fields** over multiple small queries
- Consider creating a database view or a single repository method that returns all enrichment data in one call (parent title, previous status, latest note, child counts)
- The `*PlaceholdersWithRelated()` functions already accept multiple repository interfaces — we may want to consolidate into fewer, richer queries rather than adding more repository dependencies
- Evaluate whether a single `GetTemplateEnrichmentData(entityType, entityID)` repository method would be more efficient than separate calls for each new variable

---

## Items

### 1. `previous_status` (High Impact)

**Current state**: Not available. Agents check notes or history to determine if a task is entering a status for the first time or was rejected back from a later phase.

**Proposed variable**: `{{.previous_status}}` — the status the entity was in before the current one.

**Data source**: `task_history` table (most recent entry for the entity). Similar tables exist for features and epics.

**Template use case**: Smart branching in development templates:
```go
{{- if eq .previous_status "in_code_review"}}
RETURNING FROM CODE REVIEW — check rejection notes before resuming.
{{- end}}
```

**Implementation location**: `TaskPlaceholdersWithRelated()` in `internal/config/template_helpers.go`. Requires a repository method to fetch the most recent history entry.

---

### 2. `parent_title` (Medium Impact)

**Current state**: Task templates know `{{.feature_key}}` and `{{.epic_key}}` but not the parent entity's title. Every agent has to `shark feature get` or `shark epic get` just to understand context.

**Proposed variables**:
- Tasks: `{{.feature_title}}`, `{{.epic_title}}`
- Features: `{{.epic_title}}`

**Data source**: Join to `features` and `epics` tables by ID (already available as `task.FeatureID`, `task.EpicID`, `feature.EpicID`).

**Implementation location**: `TaskPlaceholdersWithRelated()` and `FeaturePlaceholdersWithRelated()`. Requires repository lookups by ID — could be a single join query.

---

### 3. `context_data` Structured Fields (High Impact)

**Current state**: `extractContextDataMetadata()` already extracts `ContextData.Metadata` (arbitrary key-value pairs) into placeholders. But the structured fields in `ContextData` are NOT extracted:
- `Progress.CompletedSteps` / `Progress.CurrentStep` / `Progress.RemainingSteps`
- `ImplementationDecisions`
- `OpenQuestions`
- `Blockers`

**Proposed variables**:
- `{{.current_step}}` — from `ContextData.Progress.CurrentStep`
- `{{.completed_steps}}` — comma-joined list from `ContextData.Progress.CompletedSteps`
- `{{.remaining_steps}}` — comma-joined list from `ContextData.Progress.RemainingSteps`
- `{{.completed_steps_count}}` — count of completed steps
- `{{.remaining_steps_count}}` — count of remaining steps
- `{{.open_questions}}` — comma-joined list
- `{{.blockers_summary}}` — count and brief summary of active blockers

**Data source**: Already stored as JSON in `context_data` column, already parsed by `extractContextDataMetadata()`. Just needs to also extract the structured fields.

**Implementation location**: Extend `extractContextDataMetadata()` in `internal/config/template_helpers.go`. No new repository calls needed — this is pure JSON parsing of data already in hand.

**Template use case**: Eliminates the PLAN GATE pattern (`shark feature context get {{.id}} --json`) repeated in 8+ templates:
```go
{{- if .remaining_steps}}
RESUME MODE — remaining steps: {{.remaining_steps}}
{{- else}}
PLAN MODE — run planning decision tree.
{{- end}}
```

---

### 4. `latest_note` (High Impact)

**Current state**: Almost every `in_*` resume template starts with "check notes via `shark notes`". Agents need the latest note to understand why they're resuming and what happened previously.

**Proposed variables**:
- `{{.latest_note}}` — content of the most recent note
- `{{.latest_note_type}}` — type of the most recent note (comment, decision, risk, rejection, analysis)
- `{{.notes_count}}` — total number of notes on the entity
- `{{.rejection_count}}` — count of rejection-type notes

**Data source**: `entity_notes` table. Single query: `SELECT content, note_type FROM entity_notes WHERE entity_type=? AND entity_id=? ORDER BY created_at DESC LIMIT 1` plus a count query (or combined).

**Implementation location**: `*PlaceholdersWithRelated()` functions. Requires injecting `EntityNoteRepository` (or a lighter interface) into the placeholder builders.

**Template use case**:
```go
{{- if .latest_note}}
LAST NOTE ({{.latest_note_type}}): {{.latest_note}}
{{- end}}
{{- if ne .rejection_count "0"}}
This task has been rejected {{.rejection_count}} time(s) — review rejection notes carefully.
{{- end}}
```

---

### 5. `sibling_progress` (Medium Impact)

**Current state**: `active.tmpl` for both epics and features tells the agent to list children and check statuses. The agent always has to run `shark task list` or `shark feature list` as its first action.

**Proposed variables**:
- Features: `{{.task_total}}`, `{{.task_completed}}`, `{{.task_blocked}}`, `{{.task_summary}}` (e.g., "3/7 completed, 1 blocked")
- Epics: `{{.feature_total}}`, `{{.feature_completed}}`, `{{.feature_blocked}}`, `{{.feature_summary}}`

**Data source**: Count queries against `tasks` or `features` table grouped by status. Could be a single `GROUP BY status` query per entity.

**Implementation location**: `FeaturePlaceholdersWithRelated()` and `EpicPlaceholdersWithRelated()`. Requires a counter/summary interface (the `FeatureTaskCounter` and `EpicFeatureCounter` interfaces already exist in the service layer).

**Template use case**:
```go
Feature {{.id}} is ACTIVE — {{.task_summary}}.
{{- if ne .task_blocked "0"}}
WARNING: {{.task_blocked}} task(s) blocked.
{{- end}}
```

---

## Out of Scope

1. **Template syntax changes** — The `map[string]string` flat data model is retained. No move to structured template data.
2. **Template file updates** — Updating `.tmpl` files to use these new variables is a separate effort (done after variables are available).
3. **New template functions** — No new Go template functions added; existing `eq`, `ne`, `isEmpty`, `join` are sufficient.

---

## Dependencies

- `internal/config/template_helpers.go` — Primary implementation location
- `internal/models/context_data.go` — ContextData struct (item 3)
- `internal/repository/entity_note_repository.go` — Note queries (item 4)
- `internal/repository/task_repository.go` — Task counts (item 5)
- `internal/repository/feature_repository.go` — Feature counts (item 5)
- Existing `*PlaceholdersWithRelated()` function signatures will need additional parameters or a consolidated enrichment query

---

*Last Updated*: 2026-03-17
