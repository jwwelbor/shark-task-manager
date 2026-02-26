# E17: Scope & Boundaries

> Part of [E17: CLI Simplification for AI Agents](epic.md). See also: [Requirements](requirements.md), [Success Metrics](success-metrics.md).

---

## In Scope

### Phase 1 (Additive, Zero Breaking Changes)
- New `shark status set` command (unified status changes for all entity types)
- New `shark status advance` command (workflow-aware next-status)
- New `shark status options` command (shows valid transitions)
- New `shark status history` command (shows change log)
- `--field` flag on `get`, `list`, `next`, `progress`, and `status options` commands
- Structured JSON error output when `--json` or `SHARK_OUTPUT=json` is active
- Idempotent status operations (no error if entity already at target status)
- `SHARK_OUTPUT` environment variable support for session-wide JSON mode
- Flag normalization: `--order` replaces `--execution-order`, `--all` replaces `--show-all`
- New command registrations in `internal/cli/commands/`
- Service layer integration (all new commands use services, not direct repository calls)
- Exit code 4 added for conflict errors (duplicate key, concurrent modification)

### Phase 2 (New Primary Commands)
- `shark progress` command (replaces overloaded `shark status <id>` smart dispatcher)
- Batch mode for `shark status set` and `shark status advance` (multi-ID and `--feature` targeting)
- Unified `shark create` dispatcher for epics, features, and tasks
- Documentation updates to promote new commands as primary interface

### Phase 3 (Cleanup)
- `shark admin` subgroup for config/init/cloud/workflow/migrate commands
- Unified `shark note` command replacing entity-specific note commands
- Unified `shark update` and `shark delete` commands with auto-detection
- Deprecation warnings on old command forms (stderr only, suppressed in JSON mode)
- Old commands hidden from `--help` output (still fully functional)

---

## Out of Scope

### Will NOT Be Done in This Epic

1. **Binary split** - No separate `shark-config` or `shark-analytics` binaries. The CX review concluded that a single binary with subcommand grouping is better for AI agents. See [CX Review section 6](../cx-review-cli-ai-agents.md).

2. **Removing existing commands** - All existing commands remain fully functional throughout the entire epic. Command removal is a future epic (post-E17 Phase 3), after agent adoption metrics confirm the new commands are dominant.

3. **Database schema changes** - No new tables, columns, or indexes. All features operate on existing data structures.

4. **Service layer creation** - E15 handles service layer architecture. E17 wires new commands to existing (or E15-created) services. If a service method is missing, E17 creates it following E15 patterns, but E17 does not redesign the service architecture.

5. **New workflow statuses or transitions** - No changes to the workflow state machine. `status set` and `status advance` work with whatever statuses are configured in `.sharkconfig.json`.

6. **Tab completion** - Useful for HumanDev persona but out of scope. Can be a separate enhancement. See [personas.md](personas.md#tertiary-human-developer).

7. **Interactive mode / TUI** - AI agents cannot use interactive prompts (no stdin). Not in scope.

8. **HTTP API changes** - This epic focuses on CLI only. The HTTP API already follows similar patterns via the service layer and will benefit indirectly from service methods created for E17.

9. **Configuration file format changes** - `.sharkconfig.json` structure unchanged. No new configuration sections.

10. **Multi-language output** - English only.

11. **Opt-in CLI telemetry** - While success metrics reference agent log analysis, E17 does not implement telemetry collection within the CLI binary. Measurement uses external agent activity logs.

---

## Dependencies

### Depends On

| Dependency | Status | Impact | Mitigation |
|-----------|--------|--------|------------|
| **E15 (Service Layer Architecture)** | In progress | New commands should call service methods, not repositories directly | If an E15 service method is not available for an operation, the E17 command creates the service method following E15 patterns (see `.claude/rules/services/service-design.md`). |
| **E16 (Multi-Level Workflow)** | Planned | E16 introduces configurable workflows for epics and features. E17's `status set` and `status advance` commands should work with E16 workflows if available. | Design E17's status commands to use the `workflow.Service` abstraction. If E16 is not implemented, the commands use the current hardcoded epic/feature statuses. When E16 lands, E17 commands automatically support the new workflows without code changes. |

### Depended On By

| Dependent | Impact |
|-----------|--------|
| Future CLAUDE.md and agent instruction updates | Will reference E17 commands as primary interface |
| Future `shark-agent` SDK or library | May wrap E17 CLI commands |
| Future removal of deprecated commands | Post-E17 Phase 3 cleanup epic |

---

## Risks

### Risk 1: Service Layer Readiness (E15)
**Likelihood:** Medium
**Impact:** Low
**Mitigation:** New commands can create missing service methods inline, following E15 patterns. Each method is a candidate for extraction into the service layer later. TODO comments mark temporary implementations.

### Risk 2: Backward Compatibility Regression
**Likelihood:** Low
**Impact:** High
**Mitigation:** All existing tests must continue to pass at every phase. Add snapshot tests that verify old commands produce identical output before and after changes. No existing command behavior or exit code may change.

### Risk 3: Agent Adoption of New Commands
**Likelihood:** Low
**Impact:** Medium
**Mitigation:** Update CLAUDE.md and all agent instruction files to reference new commands. The `--help` output promotes new commands. Old commands still work, so no forced migration is needed. Measure adoption via agent logs after each phase.

### Risk 4: Batch Mode Complexity
**Likelihood:** Medium
**Impact:** Medium
**Mitigation:** Implement multi-ID batch first (simple: accept multiple positional args). Feature-level batch (`--feature E18-F05`) added only after multi-ID works and is tested. Partial success reporting prevents one failure from blocking the entire batch.

### Risk 5: `status` Namespace Conflict
**Likelihood:** Low
**Impact:** Medium
**Description:** The existing `shark status <id>` smart dispatcher shows progress rollups. The new `shark status set/advance` subcommand group uses the same `status` keyword. These must coexist.
**Mitigation:** In Phase 1, `shark status <id>` continues to work as progress rollup (detected by receiving an ID, not a subcommand keyword). In Phase 2, F06 (`shark progress`) becomes the documented replacement. In Phase 3, `shark status <id>` becomes a hidden alias for `shark progress <id>`.

---

## Constraints

1. **Go 1.23.4+** - Must compile with existing Go version
2. **Cobra CLI framework** - Must use Cobra for command registration (existing pattern)
3. **SQLite/Turso compatibility** - All commands must work with both database backends
4. **Non-interactive** - All new commands must work without stdin (AI agents cannot do interactive prompts)
5. **Backward compatible** - Every existing command must produce identical output after changes
6. **Service layer pattern** - New commands must follow the service layer architecture from E15 (thin command wrapper: parse args, call service, format output)

---

## Assumptions

1. AI agents are the primary CLI users (based on activity log evidence: ~70% of all invocations)
2. The service layer (E15) will be available for most operations by the time E17 Phase 1 starts
3. The existing `--json` flag behavior and output format are correct and must be preserved exactly
4. The existing smart dispatchers (`shark get`, `shark list`) are a good foundation to build on
5. Exit code conventions (0=success, 1=not found, 2=system error, 3=invalid state) are established; code 4 (conflict) is a safe extension
6. The wormwoodGM activity log (231 interactions) is representative of typical AI agent usage patterns

---

## Related Documents

- [Epic Overview](epic.md) - Vision, principles, and phased delivery plan
- [Personas](personas.md) - User types referenced in scope decisions
- [Requirements](requirements.md) - Detailed feature specifications
- [Success Metrics](success-metrics.md) - How we measure scope completion
- [CX Review](../cx-review-cli-ai-agents.md) - Evidence base for scope decisions
