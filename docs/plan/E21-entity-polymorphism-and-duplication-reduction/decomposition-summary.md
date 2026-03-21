# E21 Decomposition Summary

**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Decomposition Date**: 2026-03-19
**Features Created**: 5 (F01-F05)
**Features Deferred**: 1 (F06 -- Generic CLI Commands, rated XL, out of scope per scope.md)

---

## Feature Tree with Execution Order

| Order | Feature Key | Title | Effort | Risk | Status |
|-------|-------------|-------|--------|------|--------|
| 1 | E21-F01 | Entity Interface Foundation | M (3-5 tasks) | Zero | draft |
| 2 | E21-F02 | Cross-Cutting Service Unification | L (5-8 tasks) | Medium | draft |
| 3 | E21-F03 | Status Transition Unification | L (5-8 tasks) | Medium | draft |
| 4 | E21-F04 | Document Operations Unification | S (2-3 tasks) | Low | draft |
| 5 | E21-F05 | Template Placeholder Unification | S (2-3 tasks) | Low | draft |
| -- | E21-F06 | Generic CLI Commands (DEFERRED) | XL (10+ tasks) | Medium | not created |

**Total estimated tasks**: 18-27 tasks across F01-F05

---

## Dependency Graph

```
F01: Entity Interface Foundation  [CRITICAL PATH - no dependencies]
 |
 +---> F02: Cross-Cutting Service Unification  [depends on F01]
 |
 +---> F03: Status Transition Unification      [depends on F01]
 |
 +---> F04: Document Operations Unification    [depends on F01]
 |
 +---> F05: Template Placeholder Unification   [depends on F01]
```

### Parallelization Opportunities

After F01 completes, F02-F05 can proceed in parallel with the following considerations:

- **F02 and F03** are the highest-value features and should be prioritized
- **F02 and F03** can run in parallel if developers work on different service files
- **F04** can run any time after F01 (if EntityService does not yet exist, F04 creates it with document methods; if it exists from F03, F04 adds to it)
- **F05** can run any time after F01 (minimal file overlap with F02-F04)
- **F03** should be sequenced after any E15 (service layer migration) work on the same files to avoid merge conflicts

### Recommended Execution Sequence

```
Phase 1: F01 (foundation)
Phase 2: F02 + F03 (parallel, highest value)
Phase 3: F04 + F05 (parallel, lower effort)
```

---

## Requirement Traceability Matrix

| Requirement | Feature | Priority | Status |
|-------------|---------|----------|--------|
| REQ-F-001 | F01 | Must Have | Mapped |
| REQ-F-002 | F01 | Must Have | Mapped |
| REQ-F-003 | F01 | Must Have | Mapped |
| REQ-F-004 | F01 | Must Have | Mapped |
| REQ-F-005 | F02 | Must Have | Mapped |
| REQ-F-006 | F02 | Must Have | Mapped |
| REQ-F-007 | F02 | Must Have | Mapped |
| REQ-F-008 | F03 | Must Have | Mapped |
| REQ-F-009 | F03 | Must Have | Mapped |
| REQ-F-010 | F04 | Must Have | Mapped |
| REQ-F-011 | F05 | Must Have | Mapped |
| REQ-F-012 | F01 | Must Have | Mapped |
| REQ-F-013 | F02 (partial) | Should Have | Mapped (CLI accessor wiring only) |
| REQ-F-014 | F03 | Should Have | Mapped (parameterized test suite) |
| REQ-F-015 | -- (DEFERRED) | Could Have | Not mapped (F06 out of scope) |

### Coverage Validation

- **All Must Have requirements (REQ-F-001 through REQ-F-012)**: Fully covered by F01-F05
- **Should Have requirements (REQ-F-013, REQ-F-014)**: Partially covered (CLI accessor consolidation in F02, test consolidation in F03)
- **Could Have requirements (REQ-F-015)**: Deferred with F06

### Non-Functional Requirements Coverage

| NFR | Covered By | Validation Method |
|-----|-----------|-------------------|
| REQ-NF-001 (Zero Behavioral Changes) | All features | All existing tests pass at each phase boundary |
| REQ-NF-002 (Typed Access Preservation) | F01 | Compile-time verification |
| REQ-NF-003 (Interface Dispatch Overhead) | F01 | Benchmark test (<5ns) |
| REQ-NF-004 (Registry Lookup Performance) | F01 | Benchmark test (<50ns) |
| REQ-NF-005 (Entity Interface Test Coverage) | F01 | 100% coverage of interface methods |
| REQ-NF-006 (Shared Service Test Coverage) | F02, F03 | 80%+ coverage for refactored services |
| REQ-NF-007 (Code Reduction Target) | F02-F05 cumulative | `wc -l` before/after: target 800+ lines net reduction |
| REQ-NF-008 (New Entity Effort Reduction) | F01-F03 | 5 or fewer files for new entity type |
| REQ-NF-009 (Phase Boundary Quality Gate) | All features | `make fmt && make lint && make test` at each boundary |

---

## Estimated Code Impact

| Feature | Lines Removed | Lines Added | Net |
|---------|--------------|-------------|-----|
| F01 | ~0 | ~650 (new files) | +650 |
| F02 | ~230 | ~100 | -130 |
| F03 | ~525 | ~200 | -325 |
| F04 | ~150 | ~50 | -100 |
| F05 | ~75 | ~30 | -45 |
| **Total** | **~980** | **~1,030** | **+50 (gross), -650 net reduction in service layer** |

Note: F01 adds new infrastructure files (interface, adapters, registry). The net reduction target of 800+ lines (REQ-NF-007) applies to the service layer only, excluding F01's new infrastructure. The ~980 lines removed from F02-F05 exceed the 800-line target.

---

## Deferred Scope

### F06: Generic CLI Commands (XL)

**Reason for deferral**: The CLI command layer has ~24,000 lines across 76 files. Refactoring it alongside the service layer creates too large a blast radius. F01-F05 provide the foundation that makes F06 straightforward in a follow-on epic.

**Future epic candidate**: "Generic CLI Command Builder" -- depends on E21 F01-F03 completion.

---

## PRD Locations

| Feature | PRD Path |
|---------|----------|
| F01 | `docs/plan/E21-.../E21-F01-entity-interface-foundation/prd.md` |
| F02 | `docs/plan/E21-.../E21-F02-cross-cutting-service-unification/prd.md` |
| F03 | `docs/plan/E21-.../E21-F03-status-transition-unification/prd.md` |
| F04 | `docs/plan/E21-.../E21-F04-document-operations-unification/prd.md` |
| F05 | `docs/plan/E21-.../E21-F05-template-placeholder-unification/prd.md` |

---

*Last Updated*: 2026-03-19
