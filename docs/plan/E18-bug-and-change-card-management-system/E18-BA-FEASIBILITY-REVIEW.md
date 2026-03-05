# E18 BA Feasibility Review: Bug and Change-Card Management System

**Date**: 2026-03-02
**Reviewer**: Business Analyst Agent
**Epic**: E18 -- Bug and Change-Card Management System
**Overall Assessment**: **APPROVED**

---

## 1. Cross-Epic Conflict Analysis

### E12: Bug Tracker System -- SUPERSESSION (No Conflict)

E18 explicitly supersedes E12. The research report confirms this is appropriate:

- E12 was designed on 2026-01-04 with a heavier data model (28+ fields) and emphasis on CI/CD automated reporting.
- E18 simplifies the approach by using context fields for optional metadata instead of dedicated columns, aligning with the current architecture established in E15-E17.
- E12's database key in shark is `E12` with title "Repository Layer Database Interface Migration" -- the original bug tracker E12 epic appears to have been repurposed. The E12 design documents at `dev-artifacts/2026-01-04-bug-tracker-design/` remain as prior art.
- **Action required**: E12 (original bug tracker scope) should be formally marked as superseded once E18 enters active development.

**Verdict**: No conflict. Clean supersession with documented rationale.

### E08: Idea Capture and Conversion System -- COMPLEMENTARY (No Conflict)

- E08 established the standalone entity pattern (ideas with promotion/conversion to epics/features/tasks).
- E18's Could Have requirements (REQ-F-019, REQ-F-020) propose similar promotion paths: bug-to-task and change-card-to-feature.
- The `converted_to_type`/`converted_to_key` pattern from E08's idea model is directly reusable.
- No overlap in entity types, workflows, or key formats.

**Verdict**: Complementary. Pattern reuse opportunity, no conflict.

### E16: Multi-Level Workflow System -- DEPENDENCY (Manageable)

- E18 depends on E16's `ForLevel()` infrastructure to register bug and change-card workflow levels.
- E16 is currently `active` status with the core `GetWorkflowForLevel()` function already implemented for epic/feature/task levels.
- Adding `bug` and `change` levels is additive -- it extends the existing mechanism without modifying it.
- The research report confirms the dependency is low risk because the core infrastructure exists.

**Verdict**: Dependency is manageable. The foundational `ForLevel()` mechanism is in place. E18 extends it linearly.

### E17: CLI Simplification for AI Agents -- EXTENSION (No Conflict)

- E17 is `completed`. It established the unified CLI patterns (`shark get`, `shark status`, `shark search`) with entity auto-detection.
- E18 extends these patterns to handle B### and C### key prefixes.
- The cross-cutting nature of adding new entity type dispatch points (identified in the research as Risk 1) is a known engineering concern, not a business conflict.

**Verdict**: Clean extension of completed work. No conflict.

### E15: Service Layer Architecture Refactoring -- FOUNDATION (No Conflict)

- E15 is `active` and established the service layer patterns that E18 must follow.
- E18 adds new services (`BugService`, `ChangeCardService`) following the established `NewXxxService(repo, workflowSvc, ...)` pattern.
- No contradiction with E15's architectural direction.

**Verdict**: No conflict. E18 is a well-aligned consumer of E15 patterns.

### Overall Cross-Epic Assessment

No cross-epic conflicts detected. All interactions are either complementary (E08), clean supersessions (E12), manageable dependencies (E16), or pattern extensions (E15, E17). The epic is well-positioned in the broader project roadmap.

---

## 2. Market Viability

### Business Case Validation

The research report's competitive analysis supports the business case:

1. **Dedicated entity types are the industry norm.** Jira, Linear, Bugzilla, and ClickUp all distinguish bugs from features at the entity level. E18's decision to create separate entity types (rather than task subtypes) aligns with established industry practice. This validates the PRD's core design decision.

2. **CLI-first bug tracking is a genuine differentiator.** The competitive analysis shows most tools are web-first. Only GitHub Issues (`gh issue create`) and Linear offer CLI-native bug workflows. Shark's CLI-first, local-first approach with `--json` output for AI agents fills a market niche that competitors do not address well.

3. **Change-card concept fills a real gap.** The research explicitly notes that "change request / enhancement request tracking is underdeveloped in CLI tools." Most CLI tools conflate enhancement requests with regular issues. E18's change-card concept with a dedicated lightweight workflow (propose-approve-implement) is novel in the CLI space.

4. **Severity tracking and triage workflows are standard expectations.** E18's four-level severity (critical/high/medium/low) and the `reported -> triaged -> in_fix -> in_verification -> resolved` workflow match industry norms from Jira, Bugzilla, and Linear.

### Market Viability Assessment

The market viability is **strong**. The combination of CLI-first, local-first, and dedicated entity types with configurable workflows is unique. No direct competitor offers all of these together. The business case that "bugs and change requests are fundamental to every development workflow" is supported by the research findings.

---

## 3. Scope Coherence

### Requirements vs. Research Findings

Reviewing each requirement area against what the research uncovered:

**Must Have Requirements (REQ-F-001 through REQ-F-014)**: All confirmed as feasible. The research found that approximately 60% of E18 is pattern reuse from existing entity implementations. The remaining 40% is entity-specific logic (triage, approval, severity filtering, dashboard sections). This is a favorable ratio.

**Should Have Requirements (REQ-F-015 through REQ-F-018)**: Confirmed feasible with low effort. The existing `EntityNoteRepository` and `ContextService` already support multiple entity types via `entity_type` fields. Bug and change-card notes/context slot directly into these systems.

**Could Have Requirements (REQ-F-019 through REQ-F-021)**: Appropriately deferred. Bug-to-task promotion and change-card-to-feature promotion are valuable but not essential for launch. Duplicate detection hints add polish but are not core.

### Scope Boundary Assessment

The scope boundaries documented in `scope.md` are well-reasoned:

- **Automated Bug Detection / CI/CD Integration**: Correctly deferred. The workaround (CI pipelines calling `shark bug create` via CLI) is viable for initial adoption.
- **Web UI / Dashboard Frontend**: Correctly deferred. Shark is CLI-first by design.
- **Email / Slack Notifications**: Correctly deferred. Orthogonal to core entity types.
- **Bug Attachments**: Correctly deferred. Markdown files and context fields are sufficient for text-based bug descriptions.
- **SLA Tracking**: Correctly deferred. Requires timer-based automation that does not exist in Shark's architecture.

### Alternative Approaches Assessment

The PRD documents three rejected alternatives with clear rationale:

1. **Bugs as Task Subtypes**: Rejected for semantic clarity and workflow isolation. This is the correct decision given the research finding that "dedicated entity types are the industry norm."
2. **Bugs as Feature-Level Items**: Rejected because feature workflows do not match bug lifecycle semantics. Correct -- the triage/fix/verify workflow is fundamentally different from the refinement/development/review workflow.
3. **Single "Item" Entity**: Rejected because bugs and change-cards have fundamentally different workflows. Correct -- combining them forces conditional workflow logic that increases complexity without user benefit.

### Scope Coherence Assessment

The scope is **coherent and well-bounded**. Requirements are appropriately prioritized. Exclusions have documented workarounds. Alternative approaches are rejected with clear rationale supported by research findings.

---

## 4. Business Risk Assessment

### Risk 1: Adoption Risk -- Will Users Actually Use It?

- **Severity**: Medium
- **Assessment**: The success metric targets are ambitious (90% bug adoption within 8 weeks, 5+ change-cards/month by month 2). These depend on the existing Shark user base being willing to switch from external tools.
- **Mitigation**: The < 500ms creation speed target and CLI-first design reduce friction. The `--json` output for AI agents enables automated adoption. The 50% minimum viable threshold (4 weeks) provides an earlier checkpoint.
- **Conclusion**: Not a showstopper. Success metrics have appropriate minimum viable thresholds.

### Risk 2: Scope Creep Risk -- Two Entity Types in One Epic

- **Severity**: Low-Medium
- **Assessment**: E18 introduces two new entity types (bugs AND change-cards) in a single epic. This doubles the surface area compared to introducing one type. However, the research confirms that 60% is pattern reuse, and the two types share infrastructure (notes, context, search, dashboard integration).
- **Mitigation**: The MoSCoW prioritization enables phased delivery. Phase 1 (Must Have) can be delivered and validated before Phase 3-4 refinements. If needed, change-cards could be deferred to a follow-on epic without affecting the bug entity.
- **Conclusion**: Manageable. The phased approach provides natural scope control points.

### Risk 3: Cross-Cutting Change Risk

- **Severity**: Medium
- **Assessment**: The research identifies that every entity-type switch/dispatch point in the codebase must be updated. Missing any dispatch point creates a silent failure. This is an engineering risk, but it has a direct business impact: if `shark get B001` silently fails because a dispatch point was missed, user trust in the bug system degrades.
- **Mitigation**: The research recommends a grep-based inventory of all `EntityType` dispatch points before implementation. The recommendation for an entity type registry pattern (Recommendation 3 in the research) would eliminate this risk class entirely for future entity types.
- **Conclusion**: Not a showstopper. Mitigatable through disciplined implementation.

### Risk 4: Dashboard Information Overload

- **Severity**: Low
- **Assessment**: The `shark status` dashboard currently shows epics, features, and tasks. Adding bug and change-card sections increases information density. However, the research notes this is a UX design decision, not a technical blocker.
- **Mitigation**: Conditional display (only show sections when entities exist) and concise formatting (counts by status, not full lists) address this concern.
- **Conclusion**: Not a showstopper. Standard UX consideration.

### Showstopper Assessment

**No showstopper business concerns identified.** All risks are at medium or low severity with documented mitigations.

---

## 5. Recommended Actions

### Pre-Development Actions

1. **Cancel/archive E12** (original bug tracker scope) to avoid confusion. Reference E18 as the superseding epic.
2. **Validate E16 workflow engine readiness** by confirming that `ForLevel()` can accept new level registrations without E16-internal changes.
3. **Create entity type dispatch inventory** by grepping for all `EntityType` switch/dispatch points in the codebase before implementation begins.

### Implementation Guidance

4. **Follow the phased approach** recommended in the research report:
   - Phase 1: Bug entity (data model + workflow + CRUD) and change-card entity (data model + workflow + CRUD)
   - Phase 2: Unified CLI integration (auto-detection, dashboard, analytics)
   - Phase 3: Notes, context, markdown templates, linked-entity filtering
   - Phase 4: Promotion features, duplicate detection

5. **Validate workflow engine extension early** -- implement and test bug/change-card workflow levels before building CRUD and CLI layers.

6. **Consider the entity type registry pattern** (research Recommendation 3) as a foundation task. With 5+ entity types, the scattered switch statements become a maintenance burden. This is not blocking for E18 but would reduce risk.

### Post-Launch Actions

7. **Monitor adoption metrics** at 2-week and 8-week checkpoints as defined in success-metrics.md.
8. **Coordinate with E16 on workflow profile updates** so that `shark init update --workflow=advanced` includes default bug and change-card workflow definitions.

---

## 6. Overall Assessment

### APPROVED

E18 "Bug and Change-Card Management System" passes the BA feasibility review on all four evaluation criteria:

| Criterion | Result | Summary |
|-----------|--------|---------|
| Cross-Epic Conflicts | PASS | No conflicts. Clean supersession of E12. Complementary with E08. Manageable dependency on E16. |
| Market Viability | PASS | Industry-aligned design. CLI-first bug tracking is a genuine differentiator. Change-card concept fills a real gap. |
| Scope Coherence | PASS | Requirements are well-prioritized. Exclusions have workarounds. Alternative approaches rejected with sound rationale. |
| Business Risk | PASS | No showstoppers. All risks are medium or low with documented mitigations. |

The epic is ready to advance to the next workflow stage.

---

*Reviewed against: E18 PRD (6 documents), E18-EPIC-RESEARCH-REPORT.md, E12/E08/E15/E16/E17 epic documents.*
