# Technical Feasibility Review: E21 Entity Polymorphism and Duplication Reduction

**Reviewer**: Architect
**Date**: 2026-03-19
**Status**: in_feasibility_review_tech
**Overall Assessment**: **APPROVED**

---

## 1. Technical Feasibility by Requirement Area

### 1.1 Entity Interface (REQ-F-001, REQ-F-002) -- FEASIBLE

**Assessment**: Fully feasible with one prerequisite fix.

**Verification**: All 5 model structs share 10 common fields. The interface definition maps cleanly to existing struct fields:

| Accessor | Return Type | Epic | Feature | Task | Bug | ChangeCard |
|----------|-------------|------|---------|------|-----|------------|
| `GetID()` | `int64` | `.ID` | `.ID` | `.ID` | `.ID` | `.ID` |
| `GetKey()` | `string` | `.Key` | `.Key` | `.Key` | `.Key` | `.Key` |
| `GetTitle()` | `string` | `.Title` | `.Title` | `.Title` | `.Title` | `.Title` |
| `GetSlug()` | `string` | `deref(.Slug)` | `deref(.Slug)` | `deref(.Slug)` | `deref(.Slug)` | `.Slug` (string) |
| `GetStatus()` | `string` | `string(.Status)` | `string(.Status)` | `string(.Status)` | `string(.Status)` | `string(.Status)` |
| `GetDescription()` | `string` | `deref(.Description)` | `deref(.Description)` | `deref(.Description)` | `deref(.Description)` | `deref(.Description)` |
| `GetFilePath()` | `string` | `deref(.FilePath)` | `deref(.FilePath)` | `deref(.FilePath)` | `deref(.FilePath)` | `.FilePath` (string) |
| `GetContextData()` | `*string` | `.ContextData` | `.ContextData` | `.ContextData` | `.ContextData` | `.ContextData` |
| `GetCreatedAt()` | `time.Time` | `.CreatedAt` | `.CreatedAt` | `.CreatedAt` | `.CreatedAt` | `.CreatedAt` |
| `GetUpdatedAt()` | `time.Time` | `.UpdatedAt` | `.UpdatedAt` | `.UpdatedAt` | `.UpdatedAt` | `.UpdatedAt` |

**Prerequisite**: ChangeCard uses `string` (not `*string`) for `Slug` and `FilePath`. This must be normalized to `*string` before interface implementation. See Section 3.1 for details.

**Type Safety**: The interface is purely additive. Existing code accessing `epic.Key` directly is unaffected. Compile-time interface satisfaction checks (`var _ Entity = (*Epic)(nil)`) guarantee correctness.

**Risk**: Zero. Adding accessor methods to existing structs cannot break existing code.

### 1.2 EntityRepository Adapters (REQ-F-003, REQ-F-004) -- FEASIBLE

**Assessment**: Feasible. The adapter pattern wraps existing typed repositories without modifying them.

**Verification**: Each typed repository already implements the required methods:

| Method | EpicRepo | FeatureRepo | TaskRepo | BugRepo | ChangeCardRepo |
|--------|----------|-------------|----------|---------|----------------|
| `GetByKey` | Y | Y | Y | Y | Y |
| `GetByID` | Y | Y | Y | Y | Y |
| `Update` | Y | Y | Y | Y | Y |
| `UpdateStatus` | Y (via Update) | Y (via Update) | Y | Y | Y |
| `GetContextData` | Y | Y | N (via field) | N (via field) | N (via field) |
| `UpdateContextData` | Y | Y | N (via Update) | N (via Update) | Y |

**Gap**: Task and Bug repositories do not have dedicated `GetContextData`/`UpdateContextData` methods -- they use the general `Update` method to persist ContextData changes. The adapter for these entities would need to implement context operations by calling `GetByKey` then extracting/setting the `ContextData` field and calling `Update`. This is a minor implementation detail, not a blocker.

**Type Assertion Safety**: Adapters perform `entity.(type)` assertions. The EntityRegistry guarantees correct routing, so wrong-type assertions are prevented by design. Error messages from failed assertions should include the expected and actual types for debugging.

**Risk**: Low. Adapters are thin wrappers (~15-20 lines each). Type assertion errors are caught at compile time via interface satisfaction checks.

### 1.3 Cross-Cutting Service Unification (REQ-F-005 to REQ-F-007) -- FEASIBLE

**Assessment**: Feasible and highest value-to-effort ratio.

**Verification**: I confirmed the duplication patterns in the actual codebase:

**NoteService** (`note_service.go`):
- Lines 11-47: 5 separate repository interfaces (`NoteEpicRepository`, `NoteFeatureRepository`, `NoteTaskRepository`, `NoteChangeCardRepository`, `NoteBugRepository`), each requiring only `GetByKey` and `GetByID`. These are structurally identical and can be replaced by a single `EntityRepository` map.
- `resolveEntityID`: 5-branch switch dispatching to the same `repo.GetByKey()` pattern.
- `GetEntityDetails`: 5-branch switch dispatching to the same `repo.GetByKey()` then extracting `.Key` and `.Title`.

**ContextService** (`context_service.go`):
- Lines 12-41: 5 separate repository interfaces with similar patterns.
- `getContextJSON` and `setContextJSON`: 5-branch switches performing identical `repo.GetByKey()` -> extract/set ContextData operations.

**ResumeService** (`resume_service.go`):
- 3 setter methods (`SetSessionRepo`, `SetBugRepo`, `SetChangeCardRepo`) that would be replaced by registry registration.

**CLI Accessors** (`services_global.go`):
- Lines 32-57 (NoteService), 60-84 (ContextService), 87-110 (ResumeService): Each follows an identical pattern of creating repos, constructing the service, then calling `SetBugRepo`/`SetChangeCardRepo`. The EntityRegistry replaces all of this with a single registration loop.

**Risk**: Medium. Requires careful behavioral parity testing to ensure all 5 entity paths produce identical results after refactoring. Table-driven tests parameterized by EntityType are the correct approach.

### 1.4 Status Transition Unification (REQ-F-008, REQ-F-009) -- FEASIBLE with opt-in design

**Assessment**: Feasible for Epic/Feature/Task (identical `TransitionStatus` pattern). Feasible for Bug/ChangeCard with an opt-in approach for advanced features.

**Verification**: I examined the actual `TransitionStatus` implementations:

**Epic** (`epic_service.go` lines 227-309) and **Feature** (`feature_service.go` lines 232-314): Structurally identical 80-line implementations with the full 10-step pattern (get entity, extract status, validate transition, normalize, check backward, require reason, update status, create rejection note, count children, resolve action).

**Task** (`task_service.go` lines 603-662): Uses the same pattern but adds entity-specific pre/post hooks (auto-unblock logic, feature status cascade).

**Bug** (`bug_service.go` lines 306-352): Uses a simpler pattern -- `AdvanceBugStatus` (20 lines) and `SetBugStatus` (25 lines) that lack backward detection, rejection notes, and child counting.

**ChangeCard** (`change_card_service.go` lines 317-358): Same simpler pattern as Bug -- `AdvanceChangeCardStatus` (22 lines) and `SetChangeCardStatus` (16 lines).

**Design Decision**: The shared `EntityService.TransitionStatus` must support a `TransitionFeatures` configuration so that Bug and ChangeCard can opt out of backward detection, rejection notes, and child counting. This preserves REQ-NF-001 (zero behavioral changes). See architecture-design.md for the concrete design.

**Risk**: Medium. The complexity lies in correctly implementing the opt-in mechanism without introducing subtle behavioral differences. Comprehensive table-driven tests comparing before/after behavior for each entity type are required.

### 1.5 Document Operations Unification (REQ-F-010) -- FEASIBLE

**Assessment**: Feasible. `document_helpers.go` (108 lines) already provides partial abstraction via `linkDocumentToEntity` and `unlinkDocumentFromEntity` helper functions. Each entity service wraps these helpers with entity-specific lookup logic.

**Verification**: Bug (`bug_service.go` lines 462-511) and ChangeCard (`change_card_service.go` lines 444-493) both follow the same pattern: check `writableDocRepo` is set, get entity by key, extract ID, call shared helper. This is directly unifiable via the EntityRepository adapter.

**Risk**: Low. The existing `document_helpers.go` provides the foundation. The refactoring is straightforward.

### 1.6 Template Placeholder Unification (REQ-F-011) -- FEASIBLE

**Assessment**: Feasible. Entity-specific placeholder functions (`config.EpicPlaceholders`, `config.BugPlaceholders`, `config.ChangeCardPlaceholders`) all start with the same shared-field placeholders and add entity-specific ones.

**Risk**: Low. Isolated change with no dependencies on other features.

### 1.7 EntityRegistry (REQ-F-012) -- FEASIBLE

**Assessment**: Feasible. The `EntityType` enum and `ValidEntityTypes` map already exist at `internal/models/entity_note.go:9-27`. The registry pattern is a natural extension.

**Risk**: Low. The registry is a simple map-based struct initialized at application startup. The `services_global.go` accessor pattern already demonstrates the initialization approach.

---

## 2. Architectural Concerns

### 2.1 Type Safety

**Concern**: The Entity interface returns `string` for status, losing the type safety of `EpicStatus`, `TaskStatus`, etc.

**Assessment**: Acceptable trade-off. The Entity interface is used only for cross-cutting operations (status transitions, notes, context). Entity-specific services and CLI commands continue to use typed status values for entity-specific logic. The `SetStatus(string)` method on the interface is only called by the shared `EntityService.TransitionStatus`, which receives its input from the workflow service (already string-based).

**Mitigation**: Entity-specific services retain their typed repositories and typed status handling. The Entity interface is additive -- it does not replace typed access.

### 2.2 Performance

**Concern**: Interface dispatch overhead and registry map lookups.

**Assessment**: Negligible. Go interface dispatch is ~1-2ns per call. Map lookup for the registry is O(1) with 5 entries. Both are 3-4 orders of magnitude below the database I/O cost (~1-10ms per query). No benchmarks needed to validate this -- it is well-established Go performance characteristic.

### 2.3 Complexity Budget

**Concern**: Adding a new abstraction layer (Entity interface, EntityRepository, EntityService, EntityRegistry) increases the number of concepts a developer must understand.

**Assessment**: The new concepts are counterbalanced by removing complexity:

| Removed | Added |
|---------|-------|
| 5 per-entity repository interfaces in NoteService | 1 EntityRepository interface |
| 5 per-entity repository interfaces in ContextService | 1 EntityRegistry |
| 5 per-entity repository interfaces in ResumeService | 1 EntityService with shared methods |
| 14+ switch branches across cross-cutting services | 1 map lookup per operation |
| 6+ setter methods on cross-cutting services | 1 Register() call per entity |
| 5 identical resolveAction methods | 1 shared resolveAction method |

Net result: Fewer concepts, fewer files to modify for new entities, fewer places for bugs to hide.

### 2.4 Debuggability

**Concern**: Polymorphic dispatch through interfaces is harder to trace than direct method calls.

**Assessment**: Acceptable. The adapter pattern preserves the typed repository underneath -- developers can set breakpoints in the concrete repository methods. The EntityRegistry provides a clear mapping from EntityType to adapter. Stack traces include the adapter method, making the dispatch chain visible.

**Mitigation**: Clear naming conventions (`EpicRepositoryAdapter`, `EntityService`) and godoc documentation on each adapter.

---

## 3. Dependency and Integration Risks

### 3.1 ChangeCard Type Inconsistency (PREREQUISITE)

**Risk**: `ChangeCard.Slug` is `string` (not `*string`) and `ChangeCard.FilePath` is `string` (not `*string`). All other entities use `*string` for these fields.

**Impact**: Blocks clean Entity interface implementation. The `GetSlug()` and `GetFilePath()` accessors must return consistent types.

**Resolution**: Normalize ChangeCard to use `*string` for Slug and FilePath. This requires:
1. Change `change_card.go` struct field types
2. Update ChangeCard repository scan logic to handle `*string`
3. Update service methods that set/read these fields
4. Update tests

**Effort**: Small (S). Estimated 1 task, ~30-50 lines changed across 3-4 files.

**Timing**: Must be the first task in F01, before any Entity interface work begins.

### 3.2 E15 Overlap (SEQUENCING CONSTRAINT)

**Risk**: E15 (Service Layer Refactoring) is actively migrating CLI commands to use services. E21 restructures the internal service layer.

**Current State**: E15 is ~79% complete. The overlap point is `epic_service.go`, `feature_service.go`, and `task_service.go` -- files that both E15 and E21 F03 would modify.

**Resolution**: F01 (Entity Interface Foundation) and F02 (Cross-Cutting Service Unification) have zero overlap with E15 because they target `NoteService`, `ContextService`, `ResumeService`, and the models layer -- none of which are E15 migration targets. F03 (Status Transition Unification) should be scheduled after confirming E15 has no pending work on entity service files.

**Assessment**: Manageable scheduling constraint, not a blocker.

### 3.3 E12 Repository Layer Migration (LOW RISK)

**Risk**: E12 proposes migrating repositories from `*sql.DB` to `db.Database` interface. If E12 runs concurrently, EntityRepository adapters would need updating.

**Assessment**: E12 is in draft status with no progress. E21's adapters are thin wrappers (~15-20 lines each) that can be updated quickly. No action required now.

---

## 4. Technical Debt Analysis

### Does This Epic CREATE or REDUCE Technical Debt?

**Verdict**: REDUCES technical debt significantly.

**Debt Reduced**:

| Category | Before E21 | After E21 | Reduction |
|----------|------------|-----------|-----------|
| Service-layer duplication | ~815-900 lines across 8 service files | ~100 lines (new shared code) | ~700-800 net lines removed |
| Per-entity repository interfaces | 15+ redundant interfaces across NoteService, ContextService, ResumeService | 1 EntityRepository interface | 14+ interfaces removed |
| Switch statements in cross-cutting services | 14+ branches (NoteService: 10, ContextService: 10, ResumeService: varies) | 0 switch branches (map lookup) | 14+ branches removed |
| Setter methods for entity registration | 6+ setter methods + calls | 1 Register() method | 5+ methods removed |
| Files to modify for new entity type | 15+ files | 3-5 files | 10+ files avoided per new entity |

**Debt Introduced**:

| Category | Impact | Mitigation |
|----------|--------|------------|
| New abstraction layer (Entity interface) | Developers must learn the interface contract | Well-documented, compile-time verified |
| EntityRepository adapters (5 files) | Thin wrappers that must be kept in sync with typed repos | Compile-time interface satisfaction checks |
| EntityRegistry initialization | Must be populated before cross-cutting services are called | Lazy initialization pattern matches existing `services_global.go` |

**Net Assessment**: The debt reduction far outweighs the debt introduced. The new abstractions are standard Go patterns (interfaces, adapters, registries) with compile-time safety. The eliminated duplication is the kind that causes maintenance bugs and slows development velocity.

---

## 5. Overall Assessment: APPROVED

### Summary

All requirement areas are technically feasible with the current Go technology stack, project architecture, and team capabilities. The interface-based polymorphism approach is idiomatic Go, well-proven in large codebases (Terraform, Kubernetes, Hugo), and fully incremental.

### Key Findings

1. **Entity Interface**: Directly implementable on all 5 model structs with ~10 accessor methods each. Purely additive, zero risk to existing code.

2. **Duplication is Real and Verified**: ~815-900 lines of directly unifiable service-layer duplication confirmed by codebase inspection. The 800+ line reduction target (REQ-NF-007) is achievable.

3. **Existing Abstractions Provide a Head Start**: `EntityType` enum, `TransitionResult`/`TransitionOptions` structs, `document_helpers.go`, and `workflow.Service.ForLevel` are all directly reusable.

4. **ChangeCard Type Normalization is a Must-Fix Prerequisite**: Must be completed as the first task in F01.

5. **Bug/ChangeCard Transition Pattern Requires Opt-In Design**: The shared `TransitionStatus` method must support configurable features to avoid adding unwanted behaviors to standalone entities.

6. **E15 Overlap is Manageable**: F01 and F02 have zero overlap. F03 needs sequencing after E15 entity service work completes.

### Recommended Actions

1. Proceed to feature decomposition with F01 as the critical path
2. Make ChangeCard type normalization the first task in F01
3. Design TransitionStatus with a features configuration for opt-in behavior
4. Sequence F03 after E15 confirmation
5. Revise line reduction estimate in epic.md from "1,255+" to "~815-900"

### Risks Requiring Monitoring

| Risk | Likelihood | Impact | Monitor Via |
|------|-----------|--------|-------------|
| ChangeCard normalization causes unexpected breakage | Low | Low | Comprehensive ChangeCard test suite |
| TransitionStatus opt-in design adds more complexity than it saves | Low | Medium | Code review at F03 boundary |
| E15 has pending work on entity services when F03 begins | Medium | Medium | Check E15 status before starting F03 |

---

*Reviewed against: epic.md, requirements.md, scope.md, success-metrics.md, research-report.md, ba-feasibility-review.md, and direct codebase inspection of internal/models/, internal/services/, internal/repository/, and internal/cli/services_global.go.*
