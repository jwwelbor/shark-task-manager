# Requirements

**Epic**: [Configurable Status Workflow System](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for the configurable status workflow system.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

## Must Have Requirements

### Category 1: Workflow Configuration

**REQ-F-001**: Load Workflow from Config File
- **Description**: System SHALL load status workflow definition from `.sharkconfig.json` on startup
- **User Story**: As a project manager, I want to define custom workflows in config so that Shark enforces my team's process automatically
- **Acceptance Criteria**:
  - [ ] Config file parsed on first repository initialization
  - [ ] Config cached in memory for performance (no re-read per operation)
  - [ ] Invalid JSON syntax produces clear error message with line number
  - [ ] Missing config file falls back to default 4-status workflow (backward compatible)
- **Related Journey**: Journey 5 (Project Manager Customizes Workflow), Step 1-2

**REQ-F-002**: Validate Workflow Configuration
- **Description**: System SHALL validate workflow config structure and semantics
- **User Story**: As a project manager, I want validation errors to be caught early so that I don't deploy invalid workflows
- **Acceptance Criteria**:
  - [ ] Detects missing required keys (`_start_`, `_complete_`)
  - [ ] Detects undefined status references in transition arrays
  - [ ] Detects unreachable statuses (no path from `_start_`)
  - [ ] Detects dead-end statuses (no path to `_complete_`)
  - [ ] Detects circular references with no terminal path
  - [ ] Provides actionable error messages with fix suggestions
- **Related Journey**: Journey 5, Step 3 (Validate Configuration)

**REQ-F-003**: Support Config Schema Versioning
- **Description**: System SHALL support versioned workflow configs for future evolution
- **User Story**: As a system architect, I want config versioning so that future schema changes don't break existing configs
- **Acceptance Criteria**:
  - [ ] Config includes `status_flow_version` field (default: "1.0")
  - [ ] System checks version and applies appropriate parser
  - [ ] Unsupported versions produce clear error: "Config version X.Y not supported, upgrade Shark to vZ.Z"
- **Related Journey**: Future migration scenarios

### Category 2: Status Transition Validation

**REQ-F-004**: Enforce Valid Transitions
- **Description**: System SHALL validate all status changes against configured workflow
- **User Story**: As a developer agent, I want invalid transitions to be blocked so that tasks don't enter impossible states
- **Acceptance Criteria**:
  - [ ] Transition validation occurs before database update (atomic check-and-set)
  - [ ] Invalid transitions rejected with error code 3 (invalid state)
  - [ ] Error message includes: current status, attempted status, list of valid next statuses
  - [ ] Error message includes hint: "Use --force to override validation"
  - [ ] Validation adds <100ms overhead (95th percentile, measured via benchmark)
- **Related Journey**: All agent journeys (steps where status changes occur)

**REQ-F-005**: Support Force Flag Override
- **Description**: System SHALL allow `--force` flag to bypass workflow validation for exceptional cases
- **User Story**: As a tech lead, I want to force status changes during emergencies so that I can deploy hotfixes quickly
- **Acceptance Criteria**:
  - [ ] `--force` flag supported on all status-changing commands (`set-status`, `start`, `complete`, etc.)
  - [ ] Forced transitions logged separately in task_history with `forced=true` flag
  - [ ] Warning message displayed: "⚠️  Forced transition bypassed workflow validation"
  - [ ] Forced transitions DO NOT bypass schema validation (status must still be defined in config)
- **Related Journey**: Journey 6 (Emergency Hotfix with Force Flag), Step 3

**REQ-F-006**: Record All Transition Attempts
- **Description**: System SHALL log all status transition attempts (successful and failed) in task_history
- **User Story**: As a security auditor, I want all transition attempts logged so that I can detect workflow violations
- **Acceptance Criteria**:
  - [ ] task_history records successful transitions with: previous_status, new_status, timestamp, agent/user, notes
  - [ ] task_history records forced transitions with: `forced=true` flag
  - [ ] System logs (not DB) record failed transition attempts with validation error
  - [ ] No transition attempt is invisible to audit trail
- **Related Journey**: All agent journeys (audit requirement)

### Category 3: Backward Compatibility

**REQ-F-007**: Backward Compatible Default Workflow
- **Description**: System SHALL provide default workflow matching current hardcoded behavior when config is missing
- **User Story**: As an existing Shark user, I want my project to continue working when I upgrade so that I'm not forced to configure workflows immediately
- **Acceptance Criteria**:
  - [ ] If `.sharkconfig.json` lacks `status_flow` section, use default workflow
  - [ ] Default workflow matches current behavior: `todo → in_progress → ready_for_review → completed` with `blocked`
  - [ ] Warning logged (not error): "Using default workflow, define status_flow in config to customize"
  - [ ] Existing commands (`start`, `complete`, `approve`) work unchanged with default workflow
- **Related Journey**: Backward compatibility for existing projects

### Category 4: CLI Commands

**REQ-F-010**: Generic Status Transition Command
- **Description**: System SHALL provide `shark task set-status` command for arbitrary status changes
- **User Story**: As any agent, I want a generic status change command so that I can transition to any valid status
- **Acceptance Criteria**:
  - [ ] Command signature: `shark task set-status <task-key> <new-status> [--force] [--notes="..."]`
  - [ ] Validates transition against workflow config (unless `--force`)
  - [ ] Updates task status in database
  - [ ] Records transition in task_history with optional notes
  - [ ] Supports `--json` output for agent consumption
- **Related Journey**: All agent journeys (primary status change mechanism)

**REQ-F-011**: Workflow List Command
- **Description**: System SHALL provide `shark workflow list` command to display configured status flow
- **User Story**: As a developer, I want to see the workflow visually so that I understand valid transitions
- **Acceptance Criteria**:
  - [ ] Displays all statuses and their valid next statuses
  - [ ] Highlights `_start_` and `_complete_` special statuses
  - [ ] Human-readable format (indented tree structure)
  - [ ] Supports `--json` for programmatic parsing
- **Related Journey**: Journey 5, workflow understanding

**REQ-F-012**: Workflow Validation Command
- **Description**: System SHALL provide `shark workflow validate` command to check config correctness
- **User Story**: As a project manager, I want to validate my workflow config before deploying so that I catch errors early
- **Acceptance Criteria**:
  - [ ] Checks all validation rules from REQ-F-002
  - [ ] Exits with code 0 if valid, code 2 if invalid
  - [ ] Displays summary: "✓ 14 statuses defined, 2 start statuses, 2 terminal statuses, all statuses reachable"
  - [ ] On error, displays specific issues and fix suggestions
- **Related Journey**: Journey 5, Step 3 (Validate Configuration)

**REQ-F-013**: Update Existing Convenience Commands
- **Description**: System SHALL update existing commands (`start`, `complete`, `approve`) to use workflow validation
- **User Story**: As a developer, I want existing commands to respect new workflow so that behavior is consistent
- **Acceptance Criteria**:
  - [ ] `shark task start` validates transition to `in_development` (or mapped status)
  - [ ] `shark task complete` validates transition to `ready_for_review` (or mapped status)
  - [ ] `shark task approve` validates transition to `completed` (or mapped status)
  - [ ] All commands support `--force` flag
  - [ ] Error messages match generic `set-status` command format
- **Related Journey**: Journey 2 (Developer Implements Feature), Step 2

---

## Should Have Requirements

### Category 5: Status Metadata & Agent Targeting

**REQ-F-014**: Load and Use Status Metadata
- **Description**: System SHALL load status metadata (color, description, phase, agent_types) from config
- **User Story**: As a business analyst agent, I want to query tasks by agent type so that I only see relevant work
- **Acceptance Criteria**:
  - [ ] Metadata loaded from `status_metadata` config section
  - [ ] Each status can define: color, description, phase, agent_types (all optional)
  - [ ] Metadata accessible via JSON output (`shark task get --json`)
  - [ ] Missing metadata fields default gracefully (no errors)
- **Related Journey**: All agent journeys (metadata enables targeting)

**REQ-F-015**: Filter Tasks by Agent Type
- **Description**: System SHALL support `--agent=<type>` filter for task queries
- **User Story**: As a QA agent, I want to query `--agent=qa` so that I only see QA-relevant statuses
- **Acceptance Criteria**:
  - [ ] `shark task list --agent=business-analyst` returns tasks with statuses where `agent_types` includes "business-analyst"
  - [ ] `shark task next --agent=developer` returns highest-priority task for developer agents
  - [ ] Agent filter returns empty result (not error) if no matches
  - [ ] Agent filter works in combination with `--status`, `--epic`, `--feature` filters
- **Related Journey**: Journey 1 (Business Analyst), Step 1 (Query by agent type)

**REQ-F-016**: Filter Tasks by Workflow Phase
- **Description**: System SHALL support `--phase=<phase>` filter for task queries
- **User Story**: As a project manager, I want to see all "development" phase tasks so that I can identify bottlenecks
- **Acceptance Criteria**:
  - [ ] `shark task list --phase=development` returns tasks in statuses where `phase=development`
  - [ ] Phase filter supports multiple statuses per phase (e.g., `in_development`, `ready_for_development`, `blocked`)
  - [ ] Phase filter combined with other filters works correctly
  - [ ] Unknown phase returns empty result with warning
- **Related Journey**: Journey 4 (Tech Lead), workflow analysis

**REQ-F-017**: Colored Status Output
- **Description**: System SHALL display statuses with colors defined in metadata (unless `--no-color`)
- **User Story**: As a human developer, I want colored status output so that I can visually distinguish workflow phases quickly
- **Acceptance Criteria**:
  - [ ] Status color applied from `status_metadata.{status}.color` field
  - [ ] Colors used in `task list`, `task get`, `workflow list` output
  - [ ] `--no-color` flag disables colors (plain text output)
  - [ ] Unknown colors or missing color metadata defaults to no color (not error)
- **Related Journey**: Human user experience improvement

---

## Could Have Requirements

### Category 6: Workflow Visualization & Analytics

**REQ-F-018**: Generate Workflow Diagram
- **Description**: System COULD provide `shark workflow graph` command to generate Mermaid diagram
- **User Story**: As a project manager, I want to visualize workflow as a diagram so that I can share with stakeholders
- **Acceptance Criteria**:
  - [ ] Command generates Mermaid state diagram syntax from config
  - [ ] Output includes all statuses and transitions
  - [ ] Output can be saved to file: `shark workflow graph > workflow.mmd`
  - [ ] Diagram renders correctly in Mermaid-compatible tools (GitHub, VS Code)
- **Related Journey**: Optional enhancement for workflow communication

**REQ-F-019**: Workflow Template Library
- **Description**: System COULD ship with preset workflow templates (simple, kanban, gitflow)
- **User Story**: As a new Shark user, I want to choose from templates so that I don't start from scratch
- **Acceptance Criteria**:
  - [ ] Templates stored in `internal/config/templates/` directory
  - [ ] `shark workflow use <template-name>` copies template to `.sharkconfig.json`
  - [ ] Templates include: simple (4 statuses), standard (14 statuses), kanban (5 statuses), gitflow (8 statuses)
  - [ ] Template selection preserves existing config (prompts for overwrite confirmation)
- **Related Journey**: Journey 5, easier workflow setup

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Low Latency Status Validation
- **Description**: Workflow validation SHALL add <100ms overhead to status update operations (95th percentile)
- **Measurement**: Benchmark test with 1000 status updates, measure P95 latency delta with/without validation
- **Target**: <100ms (P95), <20ms (P50)
- **Justification**: Status updates are frequent operations; high latency would degrade agent performance

**REQ-NF-002**: Efficient Config Loading
- **Description**: Workflow config loading SHALL complete in <50ms (cold start)
- **Measurement**: Time from file open to parsed config structure
- **Target**: <50ms (cold), <1ms (cached)
- **Justification**: Config loaded once per CLI invocation; must be fast for responsiveness

**REQ-NF-003**: Scalable to Large Workflows
- **Description**: System SHALL support workflows with up to 100 statuses without performance degradation
- **Measurement**: Validate workflow with 100 statuses and 500 transition rules
- **Target**: Validation completes in <500ms
- **Justification**: Enterprise teams may have complex workflows

### Security

**REQ-NF-010**: Config File Permission Validation
- **Description**: System SHALL warn if `.sharkconfig.json` has world-writable permissions
- **Implementation**: Check file permissions on load, log warning if mode includes world-write
- **Compliance**: Security best practice (prevent unauthorized workflow changes)
- **Risk Mitigation**: Prevents malicious actors from modifying workflow to bypass controls

**REQ-NF-011**: Audit Trail Integrity
- **Description**: Task history records SHALL be immutable once written
- **Implementation**: No UPDATE or DELETE operations on task_history table; only INSERT
- **Compliance**: Audit trail requirement for compliance and security review
- **Risk Mitigation**: Ensures transition history cannot be tampered with

**REQ-NF-012**: Force Flag Abuse Detection
- **Description**: System SHALL log all forced transitions for monitoring and abuse detection
- **Implementation**: Separate log entries for forced transitions, queryable for anomaly detection
- **Compliance**: Security monitoring requirement
- **Risk Mitigation**: Detects if `--force` is overused (indicates workflow doesn't match reality or malicious behavior)

### Reliability

**REQ-NF-020**: ~~Atomic Migration Transactions~~ **REMOVED - OUT OF SCOPE**
- **Original Description**: Workflow migration SHALL execute atomically (all tasks migrate or none do)
- **Decision**: Data migration is explicitly OUT OF SCOPE per `scope.md` lines 17-25
- **Rationale**: E11 targets new projects or uses default workflow (backward compatible). Existing task data migration adds significant complexity without sufficient value
- **Alternative**: Projects can use default workflow which matches legacy statuses, or manually update task statuses as needed
- **Note**: This requirement was orphaned from earlier design phase before scope was finalized

**REQ-NF-021**: Graceful Config Error Handling
- **Description**: Invalid workflow configs SHALL fail safely without corrupting database
- **Implementation**: Validate config completely before any database operations; fail fast on errors
- **Testing**: Test with malformed JSON, missing keys, circular references
- **Justification**: Config errors should not leave system in broken state

**REQ-NF-022**: Fallback to Default Workflow
- **Description**: System SHALL continue operating with default workflow if config is missing or invalid
- **Implementation**: Hardcoded default workflow used as fallback, warning logged
- **Testing**: Delete config file, verify system uses default workflow
- **Justification**: Ensures backward compatibility and robustness

### Usability

**REQ-NF-030**: Clear Error Messages
- **Description**: All validation errors SHALL include actionable guidance for resolution
- **Implementation**: Error messages follow format: "Problem description. Fix: specific action to take."
- **Testing**: Test all error conditions, verify error messages are helpful
- **Justification**: Reduces user frustration and support burden

**REQ-NF-031**: Comprehensive Help Text
- **Description**: All new commands SHALL have detailed `--help` output with examples
- **Implementation**: Help text includes: description, flags, examples (happy path + edge cases)
- **Testing**: Run `shark workflow --help`, verify examples are copy-pasteable
- **Justification**: Self-service documentation reduces onboarding time

### Maintainability

**REQ-NF-040**: Test Coverage >85%
- **Description**: Workflow package SHALL have >85% test coverage (line coverage)
- **Measurement**: `go test -cover ./internal/config/workflow.go`
- **Target**: >85% line coverage, 100% of public methods
- **Justification**: High coverage ensures reliability during refactoring

**REQ-NF-041**: Repository Layer Abstraction
- **Description**: Workflow validation SHALL be isolated in repository layer, not CLI
- **Implementation**: CLI calls `TaskRepository.UpdateStatus()`, which handles validation
- **Testing**: Repository tests use mocked configs, CLI tests use mocked repositories
- **Justification**: Separation of concerns enables independent testing

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
