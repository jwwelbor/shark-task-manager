# E17: Requirements Catalog

> Part of [E17: CLI Simplification for AI Agents](epic.md). See also: [Personas](personas.md), [User Journeys](user-journeys.md), [Scope](scope.md).

---

## Phase 1: Add Without Removing (Must-Have)

Zero breaking changes. All old commands continue to work identically.

---

### F01: `status` Subcommand Group

**Priority:** Critical
**Complexity:** M
**Addresses:** Pain Point #1 (Status transition confusion)
**Journeys:** [Journey 1](user-journeys.md#journey-1-ai-agent-daily-task-workflow) (steps 3, 5, 6), [Journey 2](user-journeys.md#journey-2-orchestrator-batch-workflow-transition)
**Dependencies:** None (standalone, first feature to implement)
**Depended On By:** F06 (progress needs status namespace separation), F07 (batch mode extends set/advance)

#### Description
Create a `shark status` subcommand group that consolidates all status-related operations under one discoverable namespace. This replaces the scattered `task next-status`, `task update --status`, `task set-status`, `task start/complete/approve` with a unified API.

#### Command Surface

```
shark status set <id> <status> [--force] [--notes "..."] [--agent <type>]
shark status advance <id> [--to <status>] [--force] [--notes "..."] [--agent <type>]
shark status options <id>           # show current status + valid transitions
shark status history <id> [--json]  # show change history
```

#### Acceptance Criteria — `shark status set`
- [ ] `shark status set E18-F05-001 in_development` changes task status
- [ ] `shark status set E18-F05 active` changes feature status
- [ ] `shark status set E18 active` changes epic status
- [ ] Auto-detects entity type from ID format
- [ ] Validates transition via workflow service
- [ ] Returns updated entity as JSON when `--json` active
- [ ] Accepts `--force` to skip workflow validation
- [ ] Accepts `--notes` for transition notes
- [ ] Accepts `--agent` to set agent on task transitions
- [ ] **Idempotent:** Returns success (exit 0) if entity already at target status
- [ ] Response includes `"changed": false` to indicate no-op transition
- [ ] No history record created for no-op transitions
- [ ] Works with both short (`E18-F05-001`) and traditional (`T-E18-F05-001`) key formats

#### Acceptance Criteria — `shark status advance`
- [ ] `shark status advance E18-F05-001` moves task to next status in workflow
- [ ] When multiple next statuses are valid, uses the primary/default transition
- [ ] `--to <status>` to specify which next status (when ambiguous)
- [ ] `--force` to skip workflow validation
- [ ] `--notes` for transition notes
- [ ] `--agent` to set agent on transition
- [ ] Works for tasks, features, and epics (auto-detected from ID)
- [ ] Returns updated entity with new status

#### Acceptance Criteria — `shark status options`
- [ ] `shark status options E18-F05-001` shows current status and valid transitions
- [ ] Output includes: `current_status`, `valid_transitions[]`, `phase`, `agent_type`
- [ ] Replaces the `next-status --preview` pattern
- [ ] Works for all entity types

#### Acceptance Criteria — `shark status history`
- [ ] `shark status history E18-F05-001` shows status change log
- [ ] Replaces existing `shark history` command (which becomes a hidden alias)
- [ ] Each entry shows: from_status, to_status, timestamp, agent, notes

#### Technical Notes
- Implement as `internal/cli/commands/status_group.go`
- `status` is a Cobra parent command with `set`, `advance`, `list`, `history` subcommands
- Must use service layer (not direct repo calls)
- `set` and `advance` share underlying status transition logic
- Existing `shark status <id>` (smart dispatcher for progress) will be migrated to `shark progress <id>` (see F06)

---

### F02: `--field` Flag for Targeted Extraction

**Priority:** Critical
**Complexity:** S
**Addresses:** Pain Point #2 (Excessive JSON post-processing)
**Journeys:** [Journey 1](user-journeys.md#journey-1-ai-agent-daily-task-workflow) (steps 1, 4), [Journey 4](user-journeys.md#journey-4-status-check-and-decision-making) (step 2)
**Dependencies:** None (standalone)
**Depended On By:** F06 (progress command supports --field)

#### Description
Add `--field <name>` flag to `get`, `list`, `next`, `progress`, and `status list` commands. When specified, returns only the raw value of that field (no JSON wrapping).

#### Acceptance Criteria
- [ ] `shark get E18-F05-001 --field status` → `in_development` (raw string)
- [ ] `shark get E18-F05-001 --field title` → `Implement JWT tokens`
- [ ] `shark get E18-F05-001 --field key` → `E18-F05-001`
- [ ] `shark next --agent developer --field key` → `E18-F05-003`
- [ ] `shark list E18-F05 --field key` → one key per line
- [ ] `shark progress E18-F05 --field progress_pct` → `78.5`
- [ ] `shark status options E18-F05-001 --field valid_transitions` → comma-separated list
- [ ] Exit code 1 if field doesn't exist in response
- [ ] Implicitly enables JSON processing (no need for `--json` alongside `--field`)

#### Technical Notes
- Implement as middleware in output formatting layer
- Works by marshaling to JSON internally, then extracting the field
- Simple top-level field access initially; nested access can be added later

---

### F03: Structured JSON Error Output

**Priority:** High
**Complexity:** S
**Addresses:** Pain Point #3 (Defensive error handling)
**Journeys:** All journeys benefit (eliminates `2>/dev/null` patterns)
**Dependencies:** None (standalone)
**Depended On By:** F04 (env var triggers JSON error mode)

#### Description
When `--json` is active (or `SHARK_OUTPUT=json`), all errors should be returned as structured JSON to stdout with appropriate exit codes, not as unstructured text to stderr.

#### Acceptance Criteria
- [ ] Error output is valid JSON: `{"error": true, "code": "INVALID_TRANSITION", "message": "...", ...}`
- [ ] Error JSON includes `entity` field (the ID operated on)
- [ ] Error JSON includes `current_status` for status-related errors
- [ ] Error JSON includes `valid_transitions` array for transition errors
- [ ] Errors go to stdout (not stderr) when JSON mode is active
- [ ] Exit codes are consistent:
  - 0: Success
  - 1: Not found
  - 2: System error (DB, IO)
  - 3: Invalid state (workflow violation, validation)
  - 4: Conflict (duplicate key)
- [ ] Human-readable errors still go to stderr when NOT in JSON mode

#### Error Codes
```
NOT_FOUND           - Entity does not exist
INVALID_TRANSITION  - Workflow does not allow this transition
INVALID_STATUS      - Status value is not in workflow config
VALIDATION_ERROR    - Input validation failure
CONFLICT            - Duplicate key or concurrent modification
SYSTEM_ERROR        - Database or IO failure
```

---

### F04: `SHARK_OUTPUT` Environment Variable

**Priority:** High
**Complexity:** XS
**Addresses:** Agents forgetting `--json` on every call
**Journeys:** [Journey 1](user-journeys.md#journey-1-ai-agent-daily-task-workflow) (set once at session start)
**Dependencies:** F03 (JSON error output should be active when SHARK_OUTPUT=json)
**Depended On By:** None

#### Description
Support `SHARK_OUTPUT=json` environment variable as an alternative to the `--json` flag.

#### Acceptance Criteria
- [ ] `SHARK_OUTPUT=json shark get E18-F05-001` returns JSON
- [ ] `--json` flag overrides env var (both ways: `--json` forces JSON, `--no-json` could force table)
- [ ] Other values: `SHARK_OUTPUT=table` (default), `SHARK_OUTPUT=json`
- [ ] Documented in `--help` output for root command

---

### F05: Flag Normalization

**Priority:** Medium
**Complexity:** XS
**Journeys:** [Journey 3](user-journeys.md#journey-3-project-setup-create-entities)
**Dependencies:** None (standalone)
**Depended On By:** F08 (unified create uses normalized flags)

#### Description
Normalize flag names across all commands for consistency.

#### Acceptance Criteria
- [ ] `--order` accepted everywhere (replaces `--execution-order`)
- [ ] `--execution-order` kept as hidden alias for backward compatibility
- [ ] `--all` replaces `--show-all` on list commands
- [ ] `--show-all` kept as hidden alias

---

## Phase 2: Promote New Commands (Should-Have)

New commands become the documented primary interface. Old commands still work.

---

### F06: `progress` Command (Replaces `shark status <id>` Smart Dispatcher)

**Priority:** High
**Complexity:** M
**Journeys:** [Journey 2](user-journeys.md#journey-2-orchestrator-batch-workflow-transition) (step 2), [Journey 4](user-journeys.md#journey-4-status-check-and-decision-making) (steps 1-2)
**Dependencies:** F01 (status subcommand must exist first to free the `status` namespace), F02 (--field support)
**Depended On By:** None

#### Description
Add `shark progress <id>` for viewing entity progress rollups, health indicators, and task breakdowns. This replaces the overloaded `shark status <id>` smart dispatcher which conflicted with the `shark status set/advance` subcommand group.

#### Acceptance Criteria
- [ ] `shark progress E18-F05` shows feature progress: weighted %, completion %, task breakdown by status
- [ ] `shark progress E18` shows epic progress: feature rollup, impediments
- [ ] `shark progress E18-F05-001` shows task progress context (status, phase, blocking info)
- [ ] `--field` flag works: `shark progress E18-F05 --field progress_pct` → `78.5`
- [ ] `--json` returns structured progress data
- [ ] Existing `shark status <id>` becomes a hidden alias for `shark progress` (backward compat)
- [ ] Health indicators: healthy, warning, critical
- [ ] Action items: tasks requiring attention

---

### F07: Batch Mode for Status Changes

**Priority:** High
**Complexity:** M
**Addresses:** Pain Point #4 (Manual for-loops)
**Journeys:** [Journey 2](user-journeys.md#journey-2-orchestrator-batch-workflow-transition) (step 1)
**Dependencies:** F01 (batch mode extends `status set` and `status advance`)
**Depended On By:** None

#### Description
Allow `shark status set` and `shark status advance` to accept multiple IDs or feature-level targeting.

#### Acceptance Criteria
- [ ] `shark status set E18-F05-001 E18-F05-002 E18-F05-003 in_qa` - multiple IDs
- [ ] `shark status set --feature E18-F05 in_qa` - all tasks in feature
- [ ] `shark status set --feature E18-F05 --from in_code_review ready_for_qa` - filtered batch
- [ ] `shark status advance --feature E18-F05` - advance all tasks to next status
- [ ] Returns batch result: `{"updated": 9, "skipped": 2, "failed": 0, "results": [...]}`
- [ ] Individual failures don't prevent other updates (partial success)
- [ ] `--dry-run` to preview batch changes

---

### F08: Unified `create` Dispatcher

**Priority:** Medium
**Complexity:** M
**Journeys:** [Journey 3](user-journeys.md#journey-3-project-setup-create-entities)
**Dependencies:** F05 (uses normalized flag names)
**Depended On By:** None

#### Description
Add `shark create <entity-type> [parent-id] "title" [flags]` as a consistent creation interface.

#### Acceptance Criteria
- [ ] `shark create epic "Epic Title"` - create epic
- [ ] `shark create feature E07 "Feature Title"` - create feature in epic
- [ ] `shark create task E07-F01 "Task Title"` - create task in feature
- [ ] `shark create task E07 F01 "Task Title"` - 3-arg format also works
- [ ] Consistent flags across all: `--priority`, `--order`, `--agent`, `--file`, `--force`
- [ ] Returns created entity as JSON when `--json` active

---

## Phase 3: Polish & Cleanup (Nice-to-Have)

---

### F09: `admin` Subgroup

**Priority:** Low
**Complexity:** S
**Dependencies:** None (standalone)
**Depended On By:** None

#### Description
Group infrequently-used admin commands under `shark admin` to declutter help output.

#### Acceptance Criteria
- [ ] `shark admin init` (replaces `shark init`)
- [ ] `shark admin config get/set` (replaces `shark config`)
- [ ] `shark admin cloud init/status` (replaces `shark cloud`)
- [ ] `shark admin workflow show/validate` (replaces `shark workflow`)
- [ ] `shark admin migrate slugs` (replaces `shark migrate`)
- [ ] Old commands remain as hidden aliases

---

### F10: Unified `note` Command

**Priority:** Low
**Complexity:** S
**Dependencies:** None (standalone)
**Depended On By:** None

#### Description
Single `shark note <id> "text"` command replacing `shark epic/feature/task note`.

#### Acceptance Criteria
- [ ] `shark note E18-F05-001 "Review feedback"` - add note to task
- [ ] `shark note E18-F05 "Feature delayed"` - add note to feature
- [ ] `shark note list E18-F05-001` - list notes
- [ ] `--type rejection|comment|review` for note categorization

---

### F11: Deprecation Warnings

**Priority:** Low
**Complexity:** XS
**Dependencies:** F01 through F08 should be implemented first (deprecation warnings reference replacement commands)
**Depended On By:** None

#### Description
Add deprecation warnings (stderr only, never JSON) for old command forms.

#### Acceptance Criteria
- [ ] `shark task get E18-F05-001` prints: `DEPRECATED: Use "shark get E18-F05-001" instead`
- [ ] Warning goes to stderr only (does not break JSON parsing)
- [ ] No warning when `--json` or `SHARK_OUTPUT=json` is active
- [ ] Warnings can be suppressed with `SHARK_NO_DEPRECATION=1`

---

### F12: Unified `update` Command

**Priority:** Medium
**Complexity:** M
**Dependencies:** None (standalone)
**Depended On By:** None

#### Description
Add `shark update <id> [flags]` for updating entity fields (title, priority, agent, etc.). Auto-detects entity type.

#### Acceptance Criteria
- [ ] `shark update E18-F05-001 --title "New title"` - update task title
- [ ] `shark update E18-F05-001 --priority 8` - update task priority
- [ ] `shark update E18-F05-001 --agent backend` - update task agent
- [ ] `shark update E18-F05 --order 3` - update feature execution order
- [ ] Returns updated entity as JSON

---

### F13: Unified `delete` Command

**Priority:** Low
**Complexity:** S
**Dependencies:** None (standalone)
**Depended On By:** None

#### Description
Add `shark delete <id> [--force]` for deleting entities. Auto-detects entity type.

#### Acceptance Criteria
- [ ] `shark delete E18-F05-001` - delete task
- [ ] `shark delete E18-F05` - delete feature (warns if has tasks)
- [ ] `shark delete E18` - delete epic (warns if has features)
- [ ] `--force` to skip confirmation/cascade
- [ ] Returns deleted entity info as JSON

---

## Feature Dependency Map

```
Phase 1 (all independent, can be developed in parallel):
  F01 (status group)  ←─── F06, F07 depend on this
  F02 (--field)       ←─── F06 uses --field
  F03 (JSON errors)   ←─── F04 triggers JSON error mode
  F04 (SHARK_OUTPUT)
  F05 (flag normalize) ←── F08 uses normalized flags

Phase 2:
  F06 (progress)  depends on F01, F02
  F07 (batch)     depends on F01
  F08 (create)    depends on F05

Phase 3 (all independent):
  F09 (admin subgroup)
  F10 (note command)
  F11 (deprecation warnings) -- should come after Phase 1+2 features exist
  F12 (update command)
  F13 (delete command)
```

## Non-Functional Requirements

### NFR-1: Backward Compatibility
- All existing commands continue to produce identical output and behavior
- Old commands remain functional (as hidden aliases after Phase 3)
- No changes to existing exit codes for existing commands
- Existing `--json` output format preserved exactly

### NFR-2: Performance
- Single command latency: less than 200ms (same as current baseline)
- Batch operation latency: less than 500ms for 20 entities
- `--field` extraction overhead: less than 10ms vs full JSON output

### NFR-3: Service Layer Integration
- All new commands must use the service layer (not direct repository calls)
- If an E15 service method does not exist for an operation, create it following the service design patterns in `.claude/rules/services/service-design.md`

### NFR-4: Testing
- All new commands must have CLI tests with mocked services
- All new service methods must have unit tests with mocked repositories
- Backward compatibility tests must verify old commands still produce identical output

---

## Related Documents

- [Epic Overview](epic.md) - Vision, principles, and command taxonomy
- [Personas](personas.md) - User types and their needs
- [User Journeys](user-journeys.md) - Current vs proposed workflows
- [Success Metrics](success-metrics.md) - Measurable KPIs
- [Scope & Boundaries](scope.md) - What is and is not in scope
