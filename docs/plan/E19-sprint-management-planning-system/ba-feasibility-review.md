# E19 Sprint Management & Planning System -- BA Feasibility Review

**Epic Key**: E19
**Review Date**: 2026-03-18
**Reviewer**: Business Analyst Agent
**Overall Assessment**: **APPROVED**

---

## 1. Cross-Epic Conflict Analysis

### 1.1 E13 (Workflow-Aware Task Command System) -- Soft Dependency, No Conflict

**Finding**: E13 is in `draft` status with 0% progress. The research report correctly identifies that E19's `shark sprint summary --detailed` (cycle-time-by-phase analytics) depends on the `work_sessions` table being populated by E13 commands.

**Business Impact**: LOW. The research report proposes graceful degradation -- showing "insufficient data" when work_sessions are empty. This is acceptable from a business perspective because:
- Core sprint value (lifecycle management, velocity, burndown) does not depend on E13
- The `--detailed` flag is additive; its absence does not undermine the primary use cases
- The `work_sessions` table schema already exists (migration v5), so the data model is forward-compatible

**Verdict**: No conflict. E13 is a recommended enhancement, not a prerequisite. The graceful degradation strategy is sound.

### 1.2 E15 (Service Layer Architecture Refactoring) -- Aligned, No Conflict

**Finding**: E15 is `active` at 79% progress. The service layer pattern is mature and well-documented. E19 should follow the established `NewXxxService()` constructor injection pattern.

**Business Impact**: NONE. E19 benefits from E15's completed work. The patterns demonstrated by `BugService` and `ChangeCardService` (from E18) provide a clear implementation template.

**Verdict**: No conflict. E15's progress is an enabler for E19.

### 1.3 E16 (Multi-Level Workflow) -- Aligned, No Conflict

**Finding**: E16 is `active` at 96%. Sprint backlog views must respect the active workflow profile's status groupings (planning, development, review, qa, done phases).

**Business Impact**: NONE. The sprint backlog view requirement (REQ-F-005) groups tasks by status category, which aligns with the workflow profile's phase definitions. No contradictions.

**Verdict**: No conflict. E16's near-completion means the workflow infrastructure E19 needs is stable.

### 1.4 E18 (Bug and Change-Card Management) -- Interaction Identified, Manageable

**Finding**: E18 is `active` at 100% (essentially complete). The research report identified an important interaction: the PRD's user journey shows `shark sprint add S024 B042` (assigning a bug to a sprint), but the proposed schema only has `task_id` in `sprint_assignments`.

**Business Impact**: MEDIUM. This is not a conflict but a schema design decision that affects scope. The research recommends polymorphic assignment (`entity_type + entity_id`) following the `entity_notes` pattern. This is the correct business decision because:
- Bugs and change-cards are legitimate sprint work items
- Excluding them would make sprint velocity incomplete (velocity should reflect all work done, not just tasks)
- The polymorphic pattern already exists in the codebase and is proven

**Verdict**: No conflict, but a PRD-to-schema gap that research correctly flagged and resolved. The polymorphic assignment recommendation should be adopted.

### 1.5 E10 (Advanced Task Intelligence & Context Management) -- Complementary

**Finding**: E10 is `active` at 82%. Sprint assignment context (which sprint a task belongs to) should be visible in `shark task get` and `shark task resume` output.

**Business Impact**: POSITIVE. Sprint context enriches existing task intelligence features. This is additive and non-conflicting.

**Verdict**: No conflict. Complementary enhancement opportunity.

### 1.6 Key Format Namespace

**Finding**: The proposed `S###` key format must coexist with existing key formats: `E##` (epics), `E##-F##` (features), `E##-F##-###` (tasks), `B###` (bugs), `CC-###` (change-cards).

**Business Impact**: NONE. The `S###` namespace is unique and follows the standalone entity pattern established by `B###` and `CC-###`. No collision risk.

**Verdict**: No conflict.

---

## 2. Market Viability Assessment

### 2.1 Business Case Validation

**Research Finding**: Sprint management is table-stakes for project planning tools. Every major competitor (Jira, Linear, Shortcut, Plane.so) has sprint support. Shark's current lack of sprint management is its largest identified functional gap.

**BA Assessment**: STRONGLY SUPPORTED. The research validates the PRD's core premise. Specific supporting evidence:

1. **Functional gap is real and documented**: The AI PM Journey Map analysis (stages 1 and 5) independently identified sprint planning as the primary missing capability. This is not a speculative feature request -- it addresses an observed workflow gap.

2. **CLI-first sprint management is a genuine differentiator**: No major tool offers complete sprint lifecycle management through CLI. This is strategically important for Shark's AI-agent-friendly positioning. The PRD's persona for the AI Orchestrator Agent is well-justified.

3. **Agent-type capacity model is novel**: Capacity tracking per agent type (backend, frontend, qa) rather than per individual is architecturally distinct and aligned with AI-augmented team structures. This is not a "me too" feature -- it extends the competitive positioning.

### 2.2 Target User Validation

**PM/Scrum Master persona**: VALIDATED. The persona accurately reflects the user operating Shark as a planning tool. Pain points (no sprint grouping, no velocity data, manual date filtering) are concrete and directly addressable by the proposed solution.

**AI Orchestrator persona**: VALIDATED with moderate confidence. The autonomous sprint planning capability is projected rather than observed, but the technical foundation (CLI with `--json` output, agent-type routing) supports it. The persona's goals (auto-create sprints, populate backlog within capacity constraints, generate readiness reports) are technically achievable with the proposed commands.

### 2.3 Verdict

The business case is strong. Research findings reinforce rather than undermine the PRD's justification.

---

## 3. Scope Coherence Assessment

### 3.1 Requirements vs. Research Findings

**Must Have requirements (REQ-F-001 through REQ-F-010)**: All assessed as FEASIBLE with LOW to MEDIUM risk. No requirement was found to be technically infeasible or misdirected. The phased delivery strategy (Core -> Analytics -> Planning) is well-structured and manages complexity appropriately.

**Should Have requirements (REQ-F-011 through REQ-F-015)**: All assessed as FEASIBLE. The planning view, bulk assignment, readiness scoring, and capacity management are achievable using existing infrastructure patterns.

**Could Have requirements (REQ-F-016 through REQ-F-018)**: Appropriately scoped as stretch goals. Auto-creation (REQ-F-016) and dashboard integration (REQ-F-017) are common patterns in competing tools and are sensible enhancements.

### 3.2 Scope Boundaries

The scope exclusions are well-reasoned:

- **No web UI**: Correct. CLI-first is the core value proposition.
- **No story point estimation workflow**: Correct. Using task priority as a capacity proxy is pragmatic for v1.
- **No cross-team coordination**: Correct. Single-project scope is appropriate for the current architecture.
- **No notifications**: Correct. Event infrastructure does not exist; polling via CLI is adequate.
- **No time tracking**: Correct. Abstract points are simpler and avoid compliance concerns.
- **No sprint templates**: Correct. Deferring until demand is demonstrated is prudent.

### 3.3 Alternative Approaches

The three rejected alternatives (sprints as epic metadata, tag-based assignment, external tool integration) were properly evaluated and rejected with sound rationale. The research confirms the decision to create sprints as a standalone first-class entity.

### 3.4 Identified Gaps

One gap identified by research that the PRD partially addresses:

**Polymorphic sprint assignment**: The PRD's user journey (Journey 3, Alt Path A) shows assigning bugs to sprints, but the database schema in the PRD only specifies `task_id` in `sprint_assignments`. The research recommends changing to `entity_type + entity_id`. This is a schema design correction, not a scope issue. The requirements should be updated during technical refinement to reflect polymorphic assignment.

### 3.5 Verdict

Scope is coherent. Requirements are well-prioritized. The MoSCoW categorization is appropriate. One schema design gap needs correction during technical refinement.

---

## 4. Business Risk Assessment

### 4.1 Risk: E13 Dependency May Never Complete

**Probability**: Medium (E13 is in `draft` at 0%)
**Impact**: Low (affects only `--detailed` analytics, not core functionality)
**Assessment**: ACCEPTABLE. The graceful degradation strategy fully mitigates this. The core sprint value proposition does not depend on E13. If E13 never ships, sprint management still delivers meaningful velocity, burndown, and planning capabilities.

### 4.2 Risk: Adoption Uncertainty

**Probability**: Low
**Impact**: Medium
**Assessment**: ACCEPTABLE. The success metric (Metric 1: Sprint Feature Adoption) sets a low bar -- one complete sprint lifecycle per iteration cycle within 2 sprints of launch. The feature addresses a documented functional gap, and the PM persona's pain points are concrete. The risk of building something nobody uses is low.

### 4.3 Risk: Scope Creep into Estimation

**Probability**: Medium (users may expect story points)
**Impact**: Low (using task priority as capacity unit is documented and pragmatic)
**Assessment**: ACCEPTABLE. The scope exclusion is explicit, the workaround (priority as size proxy) is documented, and the `capacity_points` column name is generic enough to accommodate future story points without schema changes.

### 4.4 Risk: Carryover Complexity

**Probability**: Medium
**Impact**: Medium (data integrity if carryover logic has bugs)
**Assessment**: ACCEPTABLE. The research correctly identifies this as the most complex requirement and recommends implementing it as a service-layer transaction with extensive test coverage. This follows established patterns.

### 4.5 Risk: Single Active Sprint Constraint Too Restrictive

**Probability**: Low
**Impact**: Low
**Assessment**: ACCEPTABLE. The PRD allows multiple `planning` sprints and only constrains `active` to one. This covers the legitimate use case of planning ahead while enforcing focus during execution. The warning-only approach for overlapping dates (during planning) is pragmatic.

### 4.6 Showstopper Assessment

**No showstopper risks identified.** All risks have viable mitigations. The highest combined risk (carryover complexity) is a well-understood implementation challenge, not a business blocker.

---

## 5. Recommended Actions

### 5.1 For Technical Refinement Phase

1. **Adopt polymorphic sprint assignment**: Update the `sprint_assignments` schema from `task_id` to `entity_type + entity_id` as recommended by research. This aligns the schema with the PRD's user journey and prevents a future migration.

2. **Confirm burndown approach**: Research recommends using `task_history` for burndown reconstruction (approach a). Validate this is sufficient during technical design. Daily snapshot approach (approach b) may be needed for performance at scale but is unlikely to be necessary given the 50-sprint target.

3. **Plan carryover test scenarios**: The sprint close carryover logic needs comprehensive test coverage. Define edge cases during technical refinement: empty sprint close, all tasks completed, no next sprint exists, carryover with polymorphic entities.

### 5.2 For Implementation Planning

4. **Adopt phased delivery**: Implement in the three phases recommended by research (Core -> Analytics -> Planning). This manages complexity and delivers incremental value.

5. **Decouple from E13**: Do not block E19 on E13. Implement graceful degradation for `--detailed` analytics. Document E13 as a recommended follow-up.

6. **Follow E18 implementation pattern**: Use the bug/change-card implementation as the template for sprint model, repository, service, and CLI commands.

---

## 6. Overall Assessment

| Evaluation Area | Finding | Rating |
|----------------|---------|--------|
| Cross-Epic Conflicts | No conflicts. E13 soft dependency handled via graceful degradation. E18 interaction (polymorphic assignment) identified and resolvable. | PASS |
| Market Viability | Strong business case. Sprint management is table-stakes. CLI-first and agent-type capacity are genuine differentiators. | PASS |
| Scope Coherence | Requirements are well-prioritized and technically feasible. One schema gap identified (polymorphic assignment) for correction in tech refinement. | PASS |
| Business Risk | No showstoppers. All risks have viable mitigations. Highest risk (carryover complexity) is an implementation challenge, not a business blocker. | PASS |

**OVERALL: APPROVED**

The epic is ready to advance to technical feasibility review. The PRD is well-structured, the research validates the business case and technical feasibility, and no blocking issues were identified. The recommended actions above should be addressed during the technical refinement phase.

---

*BA Feasibility Review Complete -- 2026-03-18*
