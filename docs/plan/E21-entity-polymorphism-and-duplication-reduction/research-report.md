# E21 Research Report: Entity Polymorphism and Duplication Reduction

**Date**: 2026-03-19
**Status**: Complete
**Recommendation**: GO -- with scope adjustments noted below

---

## 1. Competitive/Pattern Landscape

Go projects handling entity polymorphism across multiple domain types typically use one of three patterns:

**Interface-Based Polymorphism (Proposed Approach)**
The idiomatic Go pattern for this problem. Used by major projects like Terraform (resource types implement `Resource` interface), Kubernetes (all API objects implement `runtime.Object`), and Hugo (content types implement `Page` interface). The pattern works well when entities share 60-70% behavior but have meaningful type-specific logic. Go interfaces are implicitly satisfied, making adoption incremental and non-breaking.

**Struct Embedding (Rejected in scope.md)**
Embedding a `BaseEntity` struct would eliminate accessor boilerplate but breaks existing direct field access (`epic.Key` vs `epic.BaseEntity.Key`), changes JSON serialization, and introduces method promotion conflicts. This was correctly rejected in the PRD.

**Go Generics (1.18+)**
Generic services (`EntityService[T models.Entity]`) are theoretically possible but still require an interface constraint, making them equivalent to the interface approach with more complex syntax. The Go community consensus (as of Go 1.22) is that interfaces are preferred for this pattern. Generics add value for collection types and algorithms, not for polymorphic service dispatch.

**Assessment**: The interface-based approach proposed in the PRD is the correct choice. It is idiomatic, incremental, and well-proven in large Go codebases.

---

## 2. Feasibility Assessment

### Entity Interface (REQ-F-001 to REQ-F-003): FEASIBLE with one caveat

All 5 entity types share the 10 fields identified in the PRD. Verified by examining actual model files:

| File | Lines | Shared Fields Present |
|------|-------|-----------------------|
| `internal/models/epic.go` | 61 | ID, Key, Title, Slug(*string), Description(*string), Status, FilePath(*string), ContextData(*string), CreatedAt, UpdatedAt |
| `internal/models/feature.go` | 54 | Same pattern |
| `internal/models/task.go` | 77 | Same pattern (ContextData is *string) |
| `internal/models/bug.go` | 83 | Same pattern |
| `internal/models/change_card.go` | 70 | **Slug is `string` (not `*string`), FilePath is `string` (not `*string`)** |

**Blocker identified**: `ChangeCard.Slug` and `ChangeCard.FilePath` are value types (`string`) while all other entities use pointer types (`*string`). The Entity interface accessor `GetSlug() string` and `GetFilePath() string` would need to return `string`, requiring the other four models' implementations to dereference their pointers (with nil-safety handling). This is solvable but must be addressed in F01. Two options:
1. Change ChangeCard to use `*string` for Slug/FilePath (minor migration, preferred)
2. Have accessors return `string` with nil-safe dereferencing in implementations

The `EntityType` enum already exists at `internal/models/entity_note.go:9-18` with all 5 types defined. This is a head start -- no new enum needed.

### EntityRepository Adapters (REQ-F-004 to REQ-F-005): FEASIBLE

Each typed repository already implements the methods needed for a polymorphic adapter: `GetByKey`, `GetByID`, `Create`, `Update`, `Delete`, `UpdateStatus`. The adapter pattern requires type assertions (`entity.(*models.Epic)`) which carry a small runtime cost but are well-contained. The risk of wrong-type assertions is mitigated by the EntityRegistry routing entity types to their correct repository.

### Service Unification (REQ-F-006 to REQ-F-015): FEASIBLE -- Duplication Validated

**Duplication validation results** (measured from actual code):

| Duplicated Pattern | Claim | Verified | Evidence |
|---|---|---|---|
| TransitionStatus methods | 5x ~80 lines = ~400 | **3x ~80 lines = ~240** | Epic (lines 227-309), Feature (lines 232-314), Task (lines 603-662). Bug and ChangeCard use simpler AdvanceStatus/SetStatus methods (~30 lines each), not the full TransitionResult pattern. |
| resolveAction methods | 5x ~25 = ~125 | **5x ~18 lines = ~90** | Confirmed in all 5 services. Structurally identical except for placeholder function call. |
| Document linking methods | 5x ~30 = ~150 | **Partially confirmed: ~100** | Bug (lines 462-511) and ChangeCard (lines 444-493) have full Link/Unlink/List patterns using `document_helpers.go` shared functions. Epic/Feature/Task use similar pattern. |
| NoteService switch branches | 2 switches x 5 branches | **Confirmed: 80 lines** | `resolveEntityID` (lines 186-227) and `GetEntityDetails` (lines 57-98) each have 5 branches. |
| ContextService switch branches | 2 switches x 5 branches | **Confirmed: ~90 lines** | `getContextJSON` (lines 137-178) and `setContextJSON` (lines 181-224) each have 5 branches. |
| ResumeService per-entity repos | Setter methods | **Confirmed: 3 setter methods** | `SetSessionRepo`, `SetBugRepo`, `SetChangeCardRepo` at lines 151-163. Uses separate methods per entity type, not switch statements. |
| Repository interface defs (per-service) | ~15 x ~5 = ~75 | **Confirmed: ~85 lines** | NoteService defines 5 repo interfaces (lines 11-47), ContextService defines 5 (lines 12-41), ResumeService defines 4 (lines 13-50). |
| CLI accessor functions | 5x ~25 = ~125 | **Confirmed: ~130 lines** | `services_global.go` lines 32-57 (NoteService), 60-84 (ContextService), 87-110 (ResumeService), 319-338 (ChangeCardService), 348-368 (BugService). Each follows identical init-repos-wire-setters pattern. |
| Setter methods on services | 10-15 | **36 total setter calls across 11 files** | Verified via grep. Most are on TaskService (13), but cross-cutting services have 6 (NoteService: 2, ContextService: 2, ResumeService: 2). |

**Revised duplication estimate**: ~815-900 lines of directly unifiable service-layer duplication. The PRD's "1,255+" claim is slightly inflated because:
- Bug and ChangeCard do not have the full `TransitionStatus` pattern with `TransitionResult`, backward detection, rejection notes, and child counting. They use simpler `SetBugStatus`/`AdvanceBugStatus` methods.
- Some "duplication" counted in the PRD is actually entity-specific logic that differs meaningfully.

However, the revised estimate of ~815-900 lines is still substantial and supports the case for refactoring. The net reduction target of 800+ lines (REQ-NF-007) is achievable if Bug/ChangeCard are also migrated to the full TransitionStatus pattern as part of unification.

### Non-Functional Requirements: FEASIBLE

- **Backward compatibility (REQ-NF-001, REQ-NF-002)**: The interface is purely additive. Existing code accessing `epic.Key` directly is unaffected. No compile errors in code outside refactored services.
- **Performance (REQ-NF-003, REQ-NF-004)**: Interface dispatch in Go is ~1-2ns. Map lookup for registry is O(1). Both are negligible vs. database I/O (~1-10ms per query).
- **Quality gate (REQ-NF-009)**: Fully aligned with existing `make fmt && make lint && make test` mandatory workflow in CLAUDE.md.

---

## 3. System-Wide Impact

### Interaction with Other Epics

| Epic | Status | Impact |
|------|--------|--------|
| **E15** (Service Layer Refactoring) | In progress | **Moderate overlap**. E15 is migrating CLI commands to use services. E21 changes how services are structured internally. Recommend completing E15 first or ensuring E21 changes are compatible with ongoing E15 work. E21's Entity interface could simplify some E15 migration targets. |
| **E19** (Sprint Management) | Planned | **Positive synergy**. E19 would add a Sprint entity type -- exactly the use case E21 is designed to simplify. If E21 completes first, E19 effort drops significantly. |
| **E17** (CLI Simplification) | Complete | **No conflict**. E17 already established the unified `shark get/status/list` command pattern with entity auto-detection. E21's registry pattern would make these dispatch points cleaner but does not conflict. |
| **E18** (Bug & ChangeCard) | Complete | **Validated E21's premise**. E18 added Bug and ChangeCard entities, requiring the 15+ file modifications described in the PRD. E21 directly addresses the extension pain discovered during E18. |

### Risk of Concurrent Work

The main risk is if E15 service-layer migration is actively modifying the same service files that E21 would restructure. Recommend checking with the team whether E15 has pending work on `epic_service.go`, `feature_service.go`, or `task_service.go` before starting E21 F03 (Status Transition Unification).

---

## 4. Existing Capability Overlap

Several partial abstractions already exist that E21 can build on:

| Existing Abstraction | Location | Reuse Potential |
|---|---|---|
| **EntityType enum** | `internal/models/entity_note.go:9-18` | Direct reuse. All 5 types already defined. |
| **ValidEntityTypes map** | `internal/models/entity_note.go:21-27` | Direct reuse for registry validation. |
| **document_helpers.go** | `internal/services/document_helpers.go` (108 lines) | Already provides `linkDocumentToEntity` and `unlinkDocumentFromEntity` helper functions. These accept a link function callback, partially abstracting the entity-specific document linking. E21 F04 can build on this. |
| **TransitionResult struct** | `internal/services/transition_types.go` | Already entity-agnostic (has `EntityType` and `EntityKey` string fields). Directly reusable by EntityService. |
| **TransitionOptions struct** | `internal/services/transition_types.go` | Already entity-agnostic. Directly reusable. |
| **BackwardReasonError** | `internal/services/backward_transition_test.go` (likely also in a source file) | Already entity-agnostic error type. |
| **config.XxxPlaceholders** | `internal/config/` | Per-entity placeholder functions exist (`EpicPlaceholders`, `BugPlaceholders`, `ChangeCardPlaceholders`). These are the entity-specific parts that would extend a shared `EntityPlaceholders` base (F05). |
| **workflow.Service.ForLevel** | `internal/workflow/` | Already supports multi-level workflows (task, feature, epic, bug, change). Each entity service calls `.ForLevel()` in its constructor. EntityService would use the same pattern. |

---

## 5. Risk Assessment

| # | Risk | Likelihood | Impact | Mitigation | Related Req |
|---|------|-----------|--------|------------|-------------|
| 1 | **ChangeCard type inconsistency blocks interface implementation** | High | Low | Normalize `ChangeCard.Slug` and `ChangeCard.FilePath` to `*string` in F01 as a prerequisite step. This is a small, safe change. | REQ-F-002 |
| 2 | **TransitionStatus unification changes Bug/ChangeCard behavior** | Medium | Medium | Bug and ChangeCard currently use simpler status methods (SetBugStatus, AdvanceBugStatus) that do not include backward detection, rejection notes, or child counting. Unifying them under EntityService.TransitionStatus would add these behaviors, which may be unwanted for standalone entities. Define which TransitionStatus features are opt-in vs. mandatory. | REQ-F-008, REQ-F-009 |
| 3 | **Conflict with ongoing E15 service layer migration** | Medium | Medium | Check E15 progress before starting E21 F03. If E15 is actively modifying entity services, sequence E21 F01-F02 first (no overlap with E15 targets), then coordinate F03 timing. | REQ-NF-009 |
| 4 | **Over-abstraction reduces debuggability** | Low | Medium | Keep typed repositories and entity-specific service methods. Entity interface is additive. Developers can still step through entity-specific code paths. Use clear naming (`EntityService` for shared, `EpicService` for specific). | REQ-NF-002 |
| 5 | **Registry initialization ordering at startup** | Low | Low | Ensure EntityRegistry is populated before any cross-cutting service is called. Use lazy initialization pattern consistent with existing `services_global.go`. | REQ-F-012 |

---

## 6. Recommendations

### Recommendation: GO

The research validates the core premise. Cross-entity duplication is real, measurable (~815-900 lines in services alone), and growing with each new entity type. The interface-based approach is idiomatic Go, incremental, and proven in large codebases. Existing partial abstractions (EntityType enum, TransitionResult, document_helpers) provide a foundation to build on.

### Suggested Implementation Order

1. **F01: Entity Interface Foundation** (prerequisite for all others)
   - First task: Normalize ChangeCard.Slug and ChangeCard.FilePath to `*string` to match other entities
   - Define Entity interface, implement on all 5 models
   - Create EntityRepository interface and adapters
   - Add compile-time interface satisfaction checks

2. **F02: Cross-Cutting Service Unification** (highest value-to-effort ratio)
   - NoteService, ContextService, ResumeService
   - Eliminates 14+ switch branches and 6 setter methods
   - Creates EntityRegistry pattern

3. **F03: Status Transition Unification** (highest line reduction)
   - Start with Epic and Feature (which share identical TransitionStatus logic)
   - Extend to Task (which has additional auto-unblock logic as pre/post hook)
   - Decide on Bug/ChangeCard: either migrate to full TransitionStatus pattern or keep their simpler methods and delegate only the shared subset

4. **F04: Document Operations Unification** (quick win, builds on document_helpers.go)

5. **F05: Template Placeholder Unification** (quick win, isolated)

### Scope Adjustments

1. **Clarify F03 scope for Bug/ChangeCard**: The PRD assumes all 5 entities have identical TransitionStatus logic, but Bug and ChangeCard use simpler methods. Recommend making the full TransitionResult pattern opt-in for these entities rather than forcing behavioral changes on them. This preserves REQ-NF-001 (zero behavioral changes).

2. **Add ChangeCard type normalization to F01**: The Slug/FilePath type inconsistency should be fixed as the first task in F01, not treated as an edge case.

3. **Revise line reduction target**: The 1,255-line estimate should be revised to ~815-900 lines of directly unifiable duplication. The net reduction target of 800+ lines (REQ-NF-007) remains achievable but is tighter than the PRD suggests. Recommend keeping the 500-line minimum viable target as the commitment and treating 800+ as a stretch goal.

---

*References: All line counts and file paths verified against the codebase at commit 7299cc1 on the file-path-updates branch.*
