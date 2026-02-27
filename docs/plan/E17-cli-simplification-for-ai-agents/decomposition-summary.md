# E17: Decomposition Summary

> Epic: [E17: CLI Simplification for AI Agents](epic.md)
> Date: 2026-02-25

---

## Overview

Epic E17 has been decomposed into 6 features covering Phase 1 (Must-Have) and Phase 2 (Should-Have) of the CLI simplification effort. Phase 3 features (F09-F13 in the PRD) are deferred and not tracked in shark at this time.

The decomposition is based on the complete PRD (6 documents), research report, BA feasibility review, tech feasibility review, and UAT plan.

---

## Feature Inventory

| Shark Key | PRD Key | Title | Phase | Complexity | Exec Order | Status |
|-----------|---------|-------|-------|------------|------------|--------|
| E17-F04 | F04 | SHARK_OUTPUT Environment Variable | 1 | XS | 1 | draft |
| E17-F05 | F05 | Flag Normalization | 1 | XS | 2 | draft |
| E17-F03 | F03 | Structured JSON Error Output | 1 | S | 3 | draft |
| E17-F07 | F01 | Status Subcommand Group | 1 | M | 4 | draft |
| E17-F02 | F02 | Field Flag for Targeted Extraction | 1 | S | 5 | draft |
| E17-F06 | F06 | Progress Command | 2 | M | 6 | draft |

### Key Mapping Note

The PRD's F01 (Status Subcommand Group) is tracked as **E17-F07** in shark because features F02-F06 already existed in the database when F01 was created. The shark key is the source of truth for all task management operations.

---

## Execution Order Rationale

The execution order follows the research report's recommended implementation sequence, which builds infrastructure incrementally:

1. **E17-F04 (SHARK_OUTPUT env var)** -- XS, immediate value, 5-10 lines of code. Agents can set JSON mode once per session. Zero risk.

2. **E17-F05 (Flag Normalization)** -- XS, mechanical change using Cobra's `MarkDeprecated()`. Normalizes `--order` and `--all` across all commands. Zero risk.

3. **E17-F03 (Structured JSON Errors)** -- S, foundational error infrastructure. Creates `StructuredError` type and error mapping layer. All subsequent features benefit from consistent error reporting.

4. **E17-F07 (Status Subcommand Group)** -- M, the core feature. Creates `shark status set/advance/options/history`. Claims the `status` namespace for transitions.

5. **E17-F02 (Field Flag)** -- S, output extraction. Adds `--field` to get/list/next/status commands. Benefits from error infrastructure (F03) for `FIELD_NOT_FOUND`.

6. **E17-F06 (Progress Command)** -- M, Phase 2. Resolves the `shark status <id>` namespace collision by introducing `shark progress`. Depends on F07 (status subcommand) and F02 (field flag).

---

## Dependency Graph

```
Phase 1 (all independent of each other):
  E17-F04 (SHARK_OUTPUT)     [no deps]
  E17-F05 (Flag Normalize)   [no deps]
  E17-F03 (JSON Errors)      [no deps]
  E17-F07 (Status Group)     [no deps]     --> depended on by E17-F06
  E17-F02 (Field Flag)       [no deps]     --> depended on by E17-F06

Phase 2:
  E17-F06 (Progress)         [depends on E17-F07, E17-F02]
```

All Phase 1 features are independent and could technically be developed in parallel. The execution order provides a logical incremental build sequence, not a strict dependency chain.

---

## Requirement Traceability

### Phase 1 Epic Requirements

| Requirement | Feature | Coverage |
|-------------|---------|----------|
| F01 (Status Subcommand Group) | E17-F07 | Full |
| F02 (--field Flag) | E17-F02 | Full |
| F03 (Structured JSON Error Output) | E17-F03 | Full |
| F04 (SHARK_OUTPUT Environment Variable) | E17-F04 | Full |
| F05 (Flag Normalization) | E17-F05 | Full |

### Phase 2 Epic Requirements

| Requirement | Feature | Coverage |
|-------------|---------|----------|
| F06 (Progress Command) | E17-F06 | Full |

### Non-Functional Requirements

| NFR | Covered By |
|-----|-----------|
| NFR-1 (Backward Compatibility) | All features -- each preserves existing behavior |
| NFR-2 (Performance) | Implied -- no features introduce latency |
| NFR-3 (Service Layer Integration) | E17-F07, E17-F02, E17-F03, E17-F06 -- all use service layer |
| NFR-4 (Testing) | All features -- acceptance criteria include `make test` green |

### Phase 3 Requirements (Deferred)

F07 (Batch Mode), F08 (Unified Create), F09 (Admin Subgroup), F10 (Note Command), F11 (Deprecation Warnings), F12 (Update Command), F13 (Delete Command) are defined in the PRD but not decomposed or tracked in shark at this time. They will be addressed in a future decomposition pass.

---

## Exit Gate Verification

### Every Phase 1+2 requirement traces to a feature
PASS -- All 6 functional requirements (F01-F06) and all 4 NFRs are traced. See traceability matrix above.

### No scope overlap between features
PASS -- Each feature has a distinct responsibility:
- F04: Environment variable for session-wide JSON mode
- F05: Flag renaming with backward-compatible aliases
- F03: Error output format and error code infrastructure
- F07: Status transition commands (set, advance, options, history)
- F02: Single-field extraction from command output
- F06: Progress viewing command (separate from status transitions)

### Dependencies are acyclic
PASS -- The dependency graph is a DAG:
- Phase 1 features have no inter-dependencies
- Phase 2 feature (F06) depends on two Phase 1 features (F07, F02)
- No circular dependencies exist

### All features are appropriately sized
PASS -- Complexity distribution:
- XS: 2 features (F04, F05) -- simple, mechanical changes
- S: 2 features (F03, F02) -- moderate new code, well-defined scope
- M: 2 features (F07, F06) -- largest features, but well-scoped with clear acceptance criteria
- No L, XL, or XXL features that would need further decomposition

### Complete feature files exist
PASS -- All 6 features have complete feature.md files with:
- Frontmatter (feature_key, epic_key, title, description, execution_order, phase, complexity, status, dependencies, depended_on_by, epic_requirements)
- Scope section (Problem, Solution, What It Does, What It Does Not)
- Acceptance criteria (checkboxes from requirements.md)
- Dependencies (depends on, depended on by)
- Implementation notes
- Success metrics
- UAT scenario references

---

## Risk Notes

1. **Status namespace collision**: The `shark status <id>` smart dispatcher (progress) coexists with `shark status set/advance` during Phase 1. Full resolution happens in Phase 2 with F06 (Progress Command). Mitigation: Cobra handles subcommand vs argument disambiguation.

2. **F01/F07 key mapping**: The PRD's F01 is tracked as E17-F07 in shark. All task management and status tracking should use the shark key (E17-F07). The mapping is documented here and in the feature.md frontmatter.

3. **Service layer readiness**: Tech feasibility review confirms 60-70% of required service infrastructure exists. Primary new work is CLI commands, structured errors, and global flags. No service layer blockers.

---

## Files

### Feature Files
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F04-shark-output-environment-variable/feature.md`
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F05-flag-normalization/feature.md`
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F03-structured-json-error-output/feature.md`
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F07-status-subcommand-group/feature.md`
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F02-field-flag-for-targeted-extraction/feature.md`
- `/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/E17-F06-progress-command/feature.md`

### PRD Source Documents
- `epic.md`, `requirements.md`, `scope.md`, `success-metrics.md`, `personas.md`, `user-journeys.md`
- `research-report.md`, `ba-feasibility-review.md`, `tech-feasibility-review.md`, `uat-plan.md`

---

*Last Updated*: 2026-02-25
