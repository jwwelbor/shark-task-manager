# BA Feasibility Review: E17 - CLI Simplification for AI Agents

**Date:** 2026-02-25
**Reviewer:** BusinessAnalyst Agent
**Epic:** E17 - CLI Simplification for AI Agents
**Status:** in_feasibility_review_ba
**Assessment:** APPROVED

---

## 1. Cross-Epic Conflict Analysis

### 1.1 E15: Service Layer Architecture Refactoring

**Relationship:** E17 depends on E15 (prerequisite for service layer patterns)
**Conflict Status:** NO CONFLICT

The research report confirms that all required service methods already exist:
- `EpicService.TransitionStatus()`, `FeatureService.TransitionStatus()` for F01
- `TaskService.StartTask()`, `CompleteTask()`, `ApproveTask()`, `BlockTask()`, `UnblockTask()` for task lifecycle
- `StatusService.GetDashboard()` for F06
- Service accessor patterns (`cli.GetTaskService()`, etc.) established in `services_global.go`

E17 explicitly follows E15's architectural patterns (thin CLI commands calling services). The scope document correctly states that E17 will create missing service methods inline following E15 conventions if any are discovered, rather than blocking on E15 completion. The research independently verified this against the codebase and found zero E15 blockers.

**Verdict:** Complementary. E17 will produce new thin CLI commands that serve as exemplars of the E15 target architecture.

### 1.2 E16: Multi-Level Workflow System

**Relationship:** E17 benefits from E16 (enhancer, not blocker)
**Conflict Status:** NO CONFLICT

E16 extends workflow configuration to epic and feature levels. E17's `shark status set/advance` commands are designed to delegate to `workflow.Service.ForLevel()`, which already abstracts over entity types. When E16 adds level-specific workflows, E17 commands will automatically support them without code changes.

The research correctly identifies this as a transparent enhancement path. The open question in the epic PRD ("Should `status set` for features/epics respect E16 multi-level workflows?") is resolved: the answer is yes, by design, because both epics use the same `workflow.Service` abstraction layer.

**Verdict:** Forward-compatible. No design changes needed.

### 1.3 E11: Configurable Status Workflow System

**Relationship:** Foundation for E17's workflow-aware commands
**Conflict Status:** NO CONFLICT

E11 delivered the configurable workflow engine for tasks that E17's `status set` and `status advance` commands rely on. E17 does not modify workflow configuration or state machine behavior. It only adds new CLI entry points to existing workflow service methods.

### 1.4 E13: Workflow-Aware Task Command System

**Relationship:** Partial overlap with E17's status commands
**Conflict Status:** MINOR OVERLAP -- NO CONFLICT

E13 delivered `task start/complete/approve/block/unblock` and `epic/feature set-status/next-status` commands. E17's F01 (`status` subcommand group) consolidates these into a unified `shark status set/advance` interface. The PRD correctly handles this by keeping existing E13 commands as hidden aliases, not removing them.

**Verdict:** E17 builds on E13's work by providing a unified entry point. No conflicting behavior.

### 1.5 Other Epics

E04 (Core CLI), E05 (CLI Capabilities), E07 (Enhancements), E12 (Repository Migration), E14 (Cloud DB) -- none of these have significant interactions with E17. The CLI simplification is an additive layer that does not modify database schema, service contracts, or repository interfaces. Confirmed by the research report's system-wide impact analysis.

---

## 2. Market Viability Assessment

### 2.1 Evidence Base

The business case rests on quantitative evidence from 231 real AI agent CLI interactions collected from the wormwoodGM project over 9 days (2026-02-16 to 2026-02-25). This is a solid evidence base -- real production usage, not hypothetical scenarios.

Key findings from the evidence:
- 75% of all interactions are entity retrieval or status changes
- 36% of commands use defensive error suppression (`2>/dev/null`)
- 15% of commands pipe through Python for field extraction
- 5% of commands are manual bash for-loops for batch operations
- Agents try 3-4 different commands before finding the correct status transition

### 2.2 Competitive Positioning

The research report's competitive analysis positions Shark against GitHub CLI (`gh`), Linear CLI, and Jira CLI. Key finding: GitHub CLI's `--json` + `--jq` pattern is the de facto industry standard for agent-friendly CLI tools, but Shark's proposed `--field` flag is simpler and more direct -- a genuine competitive advantage for AI agents that benefit from single-step extraction.

The market is clearly moving toward tool-first design for AI agents. Shark's current CLI was designed primarily for human developers and is falling behind this shift. E17 directly addresses this gap.

### 2.3 Business Value Validation

The PRD claims "High" business value. The research findings support this:
- AI agents are 70% of CLI users (confirmed by activity logs)
- The two most common operations (entity retrieval, status changes) account for 75% of usage
- Current friction costs real resources: every Python pipe, error suppression wrapper, and fallback chain consumes token budget and increases failure probability
- The phased approach with zero breaking changes in Phase 1 eliminates adoption risk

**Verdict:** STRONG business case. The evidence is quantitative, reproducible, and directly tied to measurable efficiency improvements.

---

## 3. Scope Coherence Assessment

### 3.1 Requirements vs Research Findings

The requirements catalog (13 features across 3 phases) aligns well with the research findings:

| Pain Point (from evidence) | Addressed By | Phase | Coherent? |
|---------------------------|--------------|-------|-----------|
| Status transition confusion (4 different commands) | F01 (status subcommand group) | 1 | YES |
| Python post-processing (15% of commands) | F02 (--field flag) | 1 | YES |
| Defensive error suppression (36% of commands) | F03 (structured JSON errors) | 1 | YES |
| Forgetting --json flag | F04 (SHARK_OUTPUT env var) | 1 | YES |
| Inconsistent flag names | F05 (flag normalization) | 1 | YES |
| Overloaded `status` command | F06 (progress command) | 2 | YES |
| Manual batch for-loops (5% of commands) | F07 (batch mode) | 2 | YES |
| Inconsistent create syntax | F08 (unified create) | 2 | YES |
| Cluttered help output | F09 (admin subgroup) | 3 | YES |
| Per-entity note commands | F10 (unified note) | 3 | YES |
| No migration guidance | F11 (deprecation warnings) | 3 | YES |
| No unified update | F12 (update command) | 3 | YES |
| No unified delete | F13 (delete command) | 3 | YES |

All 13 features map to documented pain points or natural extensions. No feature appears speculative or unsupported by evidence. The phasing is logical: Phase 1 addresses the highest-impact pain points (status confusion, field extraction, error handling), Phase 2 extends with batch and progress features, Phase 3 polishes and cleans up.

### 3.2 Research-Revealed Gaps

The research revealed several design clarifications needed that were not fully resolved in the PRD:

1. **Exit code conflict for `--field` not found**: F02 specifies exit code 1 for "field not found", but exit code 1 is already "entity not found". Research recommends exit code 4 or structured JSON error differentiation. This needs resolution before implementation but does not affect feasibility.

2. **`--field` behavior on list commands**: PRD does not specify. Research recommends newline-separated raw values (one per entity), matching `gh` conventions. Reasonable and implementable.

3. **`SHARK_OUTPUT` vs `PM_OUTPUT`**: Current env prefix is `PM`. Research recommends `SHARK_OUTPUT` for discoverability. This is the right call -- `SHARK_OUTPUT` is self-documenting for agents.

4. **Error output stream change**: F03 moves errors from stderr to stdout in JSON mode. This is intentional and correctly scoped to JSON mode only.

**Verdict:** These are implementation-level design decisions, not scope problems. They can be resolved during technical refinement without affecting the overall scope.

### 3.3 Scope Boundaries

The out-of-scope items are appropriate:
- No database schema changes (correct -- this is a CLI layer change)
- No binary split (correct -- research confirms single binary is better for AI agents)
- No service layer redesign (correct -- E15's responsibility)
- No new workflow statuses (correct -- E11/E16's responsibility)
- No tab completion (correct -- AI agents do not use tab completion)
- No HTTP API changes (correct -- CLI focus, HTTP benefits indirectly)

**Verdict:** Scope is well-bounded, internally consistent, and supported by research.

---

## 4. Business Risk Assessment

### 4.1 Showstopper Risks

**None identified.** All risks are manageable and have documented mitigations.

### 4.2 Risk Matrix

| Risk | Likelihood | Impact | Mitigation Quality | Residual Risk |
|------|-----------|--------|---------------------|---------------|
| Service layer readiness (E15) | Low | Low | Verified -- all required methods exist | Very Low |
| Backward compatibility regression | Low | High | Cobra's built-in deprecation + snapshot tests | Low |
| Agent adoption of new commands | Low | Medium | CLAUDE.md updates + old commands remain | Low |
| Status namespace collision | Certain | Medium | Phased resolution via F06 (progress) | Low |
| Batch mode complexity | Medium | Medium | Phased implementation (multi-ID first, then --feature) | Low |
| Error output stream change | Certain | Medium | JSON-mode only, documented | Low |
| Exit code conflict (--field) | Medium | Low | Design clarification needed before implementation | Very Low |

### 4.3 Dependency Risks

- **E15 dependency**: MITIGATED. Research verified all needed service methods exist.
- **E16 dependency**: NONE. E17 is designed to be E16-agnostic. E16 enhances E17 automatically when it lands.
- **No external dependencies**: All work is internal to the shark-task-manager codebase.

### 4.4 Execution Risks

- **Phased delivery**: The three-phase approach is sound. Phase 1 has zero breaking changes, providing a safe rollout. Phases 2 and 3 can be adjusted based on Phase 1 adoption data.
- **Testing strategy**: The research recommends mocked service tests for all new commands, matching the project's established testing patterns. No novel testing approaches needed.
- **Implementation effort**: The research confirms ~60% of the service infrastructure already exists. The remaining 40% is new CLI routing, field extraction, structured error serialization, and env var support -- all well-understood patterns.

---

## 5. Success Metrics Validation

The success metrics are measurable and have clear baselines:

| Metric | Baseline | Target | Measurable? |
|--------|----------|--------|-------------|
| Command paths | ~45 | ~25 | YES (count leaf commands) |
| Commands per task lifecycle | 8-10 | 5 | YES (agent log analysis) |
| Python post-processing | 15% | 0% | YES (pattern search in logs) |
| Error suppression | 36% | <5% | YES (pattern search in logs) |
| Batch for-loops | 5% | 0% | YES (pattern search in logs) |
| Status command confusion | 4 commands per change | 1 | YES (agent log analysis) |

The measurement methodology (agent log analysis from real projects) is practical and directly connected to the evidence base that motivated the epic.

---

## 6. Overall Assessment

### APPROVED

E17 is a well-evidenced, well-scoped, and feasible epic with strong business value and low execution risk.

**Strengths:**
1. Quantitative evidence base (231 real agent interactions) directly supports every feature
2. Phased approach with zero breaking changes in Phase 1 eliminates rollout risk
3. All required service infrastructure already exists -- no blocking dependencies
4. Clear competitive positioning against industry-standard CLI tools (GitHub CLI)
5. Measurable success criteria with concrete baselines and targets
6. Forward-compatible with E16 (multi-level workflow) without requiring design changes
7. Research independently verified feasibility of all 6 Phase 1 features against the codebase

**Minor items for technical refinement:**
1. Resolve exit code conflict for `--field` not found (recommend exit code 4)
2. Specify `--field` behavior on list commands (recommend newline-separated values)
3. Confirm `SHARK_OUTPUT` env var name (recommend as specified, not `PM_OUTPUT`)
4. Clarify Open Question 3 from epic PRD: batch `status set --feature` should require `--from` filter for safety

---

## 7. Recommended Actions

1. **Advance E17 to technical feasibility review** -- the BA perspective is satisfied
2. **During technical refinement, resolve the 4 design clarifications** listed in section 6
3. **Recommend implementation order**: F04 -> F05 -> F03 -> F01 -> F02 -> F06 (as the research suggests) -- this builds infrastructure incrementally
4. **Plan agent log collection** for post-Phase-1 measurement against baseline metrics

---

*Report generated: 2026-02-25*
*Next status: ready_for_feasibility_review_tech*
