# BA Feasibility Review: E21 Entity Polymorphism and Duplication Reduction

**Reviewer**: Business Analyst
**Date**: 2026-03-19
**Status**: in_feasibility_review_ba
**Overall Assessment**: **APPROVED** -- with scope adjustments recommended

---

## 1. Cross-Epic Conflict Analysis

### E15: Service Layer Architecture Refactoring (Active, ~79% complete)

**Finding**: Moderate overlap exists. E15 is migrating CLI commands to use the service layer. E21 restructures the internals of those same services. Both epics touch files in `internal/services/`.

**Assessment**: Manageable. E15 is nearly complete (79% progress). The overlap is primarily in `epic_service.go`, `feature_service.go`, and `task_service.go`. The research report correctly identifies the sequencing strategy: E21 F01 (Entity Interface Foundation) and F02 (Cross-Cutting Service Unification) do not conflict with E15 targets because they focus on NoteService, ContextService, and ResumeService -- which are not E15 migration targets. F03 (Status Transition Unification) is the overlap point and should be scheduled after E15 completes or after confirming E15 has no pending work on entity service files.

**Recommendation**: Sequence F01 and F02 first. Begin F03 only after confirming E15 status on entity service files. This is a scheduling constraint, not a blocker.

### E12: Repository Layer Database Interface Migration (Draft)

**Finding**: E12 proposes migrating repositories from `*sql.DB` to `db.Database` interface. E21 creates `EntityRepository` adapters wrapping existing typed repositories. If E12 runs concurrently, the adapters would need to be updated.

**Assessment**: Low risk. E12 is in draft status with no progress. E21's adapter pattern wraps existing repositories without modifying them, so E12 could proceed independently. If both run concurrently, the adapters are thin enough to update quickly.

**Recommendation**: No action required. Monitor E12 if it moves to active during E21 execution.

### E19: Sprint Management & Planning System (ready_for_decomposition)

**Finding**: E19 would add a Sprint entity type -- exactly the use case E21 is designed to simplify. Positive synergy exists.

**Assessment**: Beneficial. If E21 completes first, E19's Sprint entity implementation effort drops significantly (from ~15 files to ~5 files). This validates E21's business value.

**Recommendation**: Prioritize E21 ahead of E19 to capture the effort reduction.

### E17: CLI Simplification (Completed), E18: Bug and Change-Card (Active)

**Finding**: No conflicts. E17 established unified CLI patterns. E18 validated the duplication problem E21 addresses.

---

## 2. Business Case Validation

### Line Reduction Claims

**PRD Claim**: ~1,255+ lines of cross-entity duplication in the service layer.

**Research Finding**: ~815-900 lines of directly unifiable duplication. The discrepancy comes from two factors:
1. Bug and ChangeCard use simpler `SetBugStatus`/`AdvanceBugStatus` methods (~30 lines each), not the full `TransitionStatus` pattern (~80 lines each). The PRD assumed all 5 entities had identical patterns.
2. Some counted "duplication" involves entity-specific logic that differs meaningfully and should not be unified.

**Assessment**: The revised estimate is still substantial and supports the business case. The PRD's net reduction target of 800+ lines (REQ-NF-007) is achievable but tighter than originally presented. The success metrics document already defines a minimum viable target of 500+ lines, which provides appropriate safety margin.

**Recommendation**: Revise the epic description's "1,255+" claim to "~815-900" for accuracy. Keep the 800+ line target as the primary goal. Keep the 500+ line minimum viable threshold as the commitment level. This is an honest adjustment, not a downgrade -- the value proposition remains strong.

### New Entity Effort Reduction

**PRD Claim**: Adding a 6th entity type goes from ~2 weeks / 15+ files to ~2 days / 3-4 files.

**Research Finding**: Validated. The EntityRegistry pattern and Entity interface eliminate the need to modify cross-cutting services (NoteService, ContextService, ResumeService) and their CLI accessors. The E18 experience (adding Bug and ChangeCard) confirms the 15+ file baseline.

**Assessment**: Validated. This is the strongest part of the business case and is unaffected by the line count adjustment.

### Maintenance Benefit

**PRD Claim**: Bug fixes to cross-cutting logic apply once instead of 5 times.

**Research Finding**: Confirmed. The research verified 14+ switch branches across NoteService and ContextService, plus 5 identical `resolveAction` methods. Each of these is a point where a bug fix must be applied 5 times today.

**Assessment**: Validated. The maintenance benefit compounds over time as more cross-cutting features are added.

---

## 3. Scope Coherence

### ChangeCard Type Inconsistency

**Research Finding**: `ChangeCard.Slug` and `ChangeCard.FilePath` are `string` types while all other entities use `*string`. This blocks a clean Entity interface implementation.

**PRD Coverage**: The PRD notes the type inconsistency in the shared fields table but does not identify it as a prerequisite fix.

**Assessment**: This is a gap in the requirements. The fix is small (change ChangeCard to use `*string` for Slug and FilePath, or add nil-safe dereferencing in accessors), but it must be explicitly called out as the first task in F01.

**Recommendation**: Add an explicit prerequisite task to F01: "Normalize ChangeCard.Slug and ChangeCard.FilePath to `*string` to match other entities." This should be documented in the requirements as a precondition for REQ-F-002.

### Bug/ChangeCard TransitionStatus Scope

**Research Finding**: Bug and ChangeCard do not use the full `TransitionStatus` pattern with `TransitionResult`, backward detection, rejection notes, and child counting. They use simpler methods. Unifying them under `EntityService.TransitionStatus` would add behaviors that may be unwanted for standalone entities.

**PRD Coverage**: REQ-F-008 states "All 5 entity types produce identical transition behavior for shared logic." REQ-F-009 preserves entity-specific extensions but does not address the case where an entity type has fewer transition behaviors than the shared implementation.

**Assessment**: The requirements need clarification. Forcing Bug and ChangeCard into the full `TransitionStatus` pattern would violate REQ-NF-001 (zero behavioral changes) by adding backward detection and rejection notes to entities that currently lack them.

**Recommendation**: Amend REQ-F-008 to make the full `TransitionResult` pattern opt-in for Bug and ChangeCard. The shared `EntityService.TransitionStatus` should have a mechanism (configuration flag, interface method, or options parameter) to skip backward detection and rejection notes for entity types that do not use them. This preserves REQ-NF-001.

### Requirements Alignment with Research

All other requirements (REQ-F-001 through REQ-F-015, REQ-NF-001 through REQ-NF-009) remain valid and coherent given the research findings. The interface-based approach is confirmed as the correct pattern. The existing partial abstractions (EntityType enum, TransitionResult, document_helpers) validate the incremental approach.

---

## 4. Business Risk Assessment

### Showstoppers

**None identified.** All risks identified in the research report are manageable with the scope adjustments recommended above.

### Risk Summary

| Risk | Severity | Status |
|------|----------|--------|
| ChangeCard type inconsistency | Low | Solvable as F01 prerequisite task |
| Bug/ChangeCard TransitionStatus behavioral change | Medium | Solvable with opt-in pattern in REQ-F-008 |
| E15 overlap on entity service files | Medium | Manageable with sequencing (F01/F02 first, then F03) |
| Line reduction target tighter than claimed | Low | 500+ minimum viable target provides safety margin |
| Over-abstraction reducing debuggability | Low | Mitigated by keeping typed repos and entity-specific services |

---

## 5. Recommended Actions

### Before Feature Decomposition

1. **Revise epic description**: Update the "1,255+" duplication estimate to "~815-900" in `epic.md`. Update the impact section accordingly.

2. **Amend REQ-F-002**: Add precondition that ChangeCard.Slug and ChangeCard.FilePath must be normalized to `*string` before interface implementation begins.

3. **Amend REQ-F-008**: Clarify that the full `TransitionResult` pattern (backward detection, rejection notes, child counting) is opt-in for Bug and ChangeCard. The shared method must support a reduced-feature mode for entity types with simpler status workflows.

4. **Revise Metric 1 (success-metrics.md)**: Update the duplication estimate from "1,255 lines of identified duplication" to "~815-900 lines of verified duplication." Keep the 800+ target as the primary goal and 500+ as the minimum viable.

5. **Add E15 sequencing constraint**: Document in scope.md or a planning note that F03 should not begin until E15's work on entity service files is confirmed complete or inactive.

### During Execution

6. **F01 first task**: Normalize ChangeCard types before defining the Entity interface.

7. **F03 phasing**: Start with Epic and Feature (identical TransitionStatus logic), then Task (additional auto-unblock hooks), then decide Bug/ChangeCard approach based on opt-in design.

---

## Overall Assessment: APPROVED

The research validates the core premise of E21. Cross-entity duplication is real, measurable, and growing. The proposed interface-based approach is idiomatic Go, incremental, and well-proven. Existing partial abstractions provide a solid foundation. The identified concerns (revised line count, ChangeCard type inconsistency, E15 overlap, Bug/ChangeCard scope clarification) are all manageable with the scope adjustments recommended above. None are showstoppers.

The business case remains strong: the maintenance benefit, new-entity effort reduction, and consistency guarantee all hold regardless of whether the final line reduction is 815 or 1,255. The epic should proceed to technical feasibility review.

---

*Reviewed against: epic.md, requirements.md, scope.md, success-metrics.md, personas.md, user-journeys.md, research-report.md, and cross-epic analysis of E12, E15, E17, E18, E19.*
