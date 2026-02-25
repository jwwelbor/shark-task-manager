# Epic E15: Service Layer Architecture Refactoring - Decomposition Summary

**Date**: 2026-02-16
**Epic**: E15 - Service Layer Architecture Refactoring
**Status**: Decomposition Complete (Pending User Approval)
**Product Manager**: AI Agent (ProductManager Role)

---

## Executive Summary

Epic E15 has been decomposed into **7 core features** (F01-F07) following a phased implementation approach validated by the research report and technical feasibility review. The decomposition addresses all 9 functional requirements (FR1-FR9) and 5 non-functional requirements (NFR1-NFR5), with two additional optional features (F08-F09) deferred to P1 priority per BA recommendation.

**Key Metrics:**
- **Total Features**: 7 core + 2 optional = 9 features
- **Estimated Timeline**: 7 weeks (core features), +1-2 weeks (optional features)
- **Epic Requirements Coverage**: 100% (all FR1-FR9, NFR1-NFR5)
- **Success Metrics Coverage**: 100% (all 10 metrics trace to features)
- **User Journey Coverage**: 100% (all 5 journeys)
- **Dependency Graph**: Acyclic (no circular dependencies)

---

## Feature Tree

```
Epic: E15 - "Service Layer Architecture Refactoring"
│
├── F01: Service Interface Design and Foundation
│   ├── Order: 1 (Foundation)
│   ├── Size: Small (1 sprint - Week 1)
│   ├── Risk: LOW
│   ├── Requirements: FR1 (Service Layer Architecture), FR9 (Documentation)
│   └── Deliverables: Service interfaces, DI patterns, architecture docs
│
├── F02: TaskService CRUD Operations
│   ├── Order: 2 (First Service Extraction)
│   ├── Size: Medium (1-2 sprints - Week 2)
│   ├── Risk: MEDIUM (establishes pattern)
│   ├── Requirements: FR2 (TaskService - CRUD subset)
│   ├── Dependencies: F01 (requires interface design)
│   └── Deliverables: Create, Get, Update, Delete methods in TaskService
│
├── F03: TaskService Lifecycle Operations
│   ├── Order: 3 (Lifecycle Management)
│   ├── Size: Medium (1-2 sprints - Week 3)
│   ├── Risk: MEDIUM-HIGH (workflow integration)
│   ├── Requirements: FR2 (TaskService - Lifecycle subset), NFR1 (Zero Regression)
│   ├── Dependencies: F02 (builds on CRUD foundation)
│   └── Deliverables: Start, Complete, Approve, Reopen, Block, Unblock in TaskService
│
├── F04: TaskService Querying and Dependencies
│   ├── Order: 4 (Query Operations)
│   ├── Size: Medium (1 sprint - Week 4)
│   ├── Risk: MEDIUM (complex filtering)
│   ├── Requirements: FR2 (TaskService - Query subset)
│   ├── Dependencies: F03 (completes TaskService)
│   └── Deliverables: GetNextTask, List, Filter, ValidateDependencies in TaskService
│
├── F05: Epic and Feature Service Expansion
│   ├── Order: 5 (Parallel Expansion)
│   ├── Size: Medium (2 sprints - Week 5)
│   ├── Risk: LOW-MEDIUM (proven pattern)
│   ├── Requirements: FR3 (FeatureService), FR4 (EpicService)
│   ├── Dependencies: F01 (parallel path from foundation)
│   └── Deliverables: CRUD + Progress + Health in Epic/FeatureService
│
├── F06: Repository Layer Cleanup
│   ├── Order: 6 (Data Access Layer)
│   ├── Size: Small (1 sprint - Week 6)
│   ├── Risk: LOW (straightforward logic move)
│   ├── Requirements: FR7 (Repository Cleanup)
│   ├── Dependencies: F02-F05 (services must exist to move logic into)
│   └── Deliverables: Remove CalculateProgress, GetStatusBreakdown from repos
│
├── F07: CLI Commands as Thin Wrappers
│   ├── Order: 7 (Entry Point Refactoring)
│   ├── Size: Large (2 sprints - Week 7)
│   ├── Risk: MEDIUM (high volume, 47 files)
│   ├── Requirements: FR6 (CLI Slimming), NFR4 (Incremental), NFR5 (Compatibility)
│   ├── Dependencies: F02-F06 (all services complete)
│   └── Deliverables: 47 command files <500 LOC each
│
├── F08: HTTP API Service Integration (OPTIONAL - P1)
│   ├── Order: 8 (API Wiring)
│   ├── Size: Medium (1 sprint - Optional Week 8)
│   ├── Risk: LOW (service layer enables it)
│   ├── Requirements: FR8 (HTTP API Integration)
│   ├── Dependencies: F07 (CLI refactoring complete)
│   ├── Note: BA recommendation to defer to separate epic E17
│   └── Deliverables: 47 API endpoints using service layer
│
└── F09: QueryService for Cross-Entity Operations (OPTIONAL - P1)
    ├── Order: 9 (Consolidation)
    ├── Size: Small (3-5 days)
    ├── Risk: LOW (consolidation, not creation)
    ├── Requirements: FR5 (QueryService)
    ├── Dependencies: F02-F05 (entity services must exist)
    ├── Note: Not blocking core refactoring
    └── Deliverables: Smart dispatch logic consolidated
```

---

## Dependency Graph

```
                    F01 (Interface Design)
                      ↓
           ┌──────────┴──────────┐
           ↓                     ↓
    F02 (TaskService CRUD)   F05 (Epic/FeatureService)
           ↓                     ↓
    F03 (TaskService Lifecycle) │
           ↓                     │
    F04 (TaskService Querying)  │
           ↓                     ↓
           └──────────┬──────────┘
                      ↓
              F06 (Repository Cleanup)
                      ↓
              F07 (CLI Slimming)
                      ↓
              F08 (HTTP API - Optional)
                      ↓
              F09 (QueryService - Optional)
```

**Key Observations:**
- **Sequential Path**: F01 → F02 → F03 → F04 → F06 → F07 (TaskService completion)
- **Parallel Path**: F01 → F05 (Epic/FeatureService can develop alongside TaskService)
- **Convergence Point**: F06 requires both TaskService (F02-F04) and Epic/FeatureService (F05)
- **No Circular Dependencies**: DAG (Directed Acyclic Graph) validated
- **Optional Extensions**: F08 and F09 can be deferred without blocking core epic

---

## Requirements Traceability Matrix

### Functional Requirements

| Epic Requirement | Feature(s) | Coverage | Notes |
|------------------|-----------|----------|-------|
| **FR1: Service Layer Architecture** | F01 | Full | Interface design, DI patterns, foundation |
| **FR2: TaskService Implementation** | F02, F03, F04 | Full | Split by complexity: CRUD, Lifecycle, Querying |
| **FR3: FeatureService Expansion** | F05 | Full | Expand from transitions to full CRUD + progress |
| **FR4: EpicService Expansion** | F05 | Full | Expand from transitions to full CRUD + rollups |
| **FR5: QueryService** | F09 | Full (P1) | Cross-entity smart dispatch (optional) |
| **FR6: CLI Commands as Thin Wrappers** | F07 | Full | 47 files to <500 LOC each |
| **FR7: Repository Layer Cleanup** | F06 | Full | Remove 106 LOC business logic from repos |
| **FR8: HTTP API Integration** | F08 | Full (P1) | 100% feature parity with CLI (optional/E17) |
| **FR9: Service Interface Documentation** | F01, F02-F05 | Cross-cutting | Godoc comments per feature, architecture docs in F01 |

### Non-Functional Requirements

| Epic Requirement | Feature(s) | Coverage | Notes |
|------------------|-----------|----------|-------|
| **NFR1: Zero Behavior Regression** | F03, F07 | Cross-cutting | Testing emphasis in lifecycle ops and CLI slimming |
| **NFR2: Test Coverage >80%** | F02-F05 | Cross-cutting | Service unit tests per feature |
| **NFR3: Performance ±10%** | F06, F07 | Cross-cutting | Benchmarking in cleanup and CLI refactoring |
| **NFR4: Incremental Migration** | F07 | Full | PRs <500 LOC, phased command refactoring |
| **NFR5: Backward Compatibility** | F07 | Full | CLI interface unchanged (same flags, output, exit codes) |

---

## Success Metrics Traceability

Mapping success metrics from `success-metrics.md` to features:

| Metric | Category | Feature(s) | Measurement Point |
|--------|----------|-----------|-------------------|
| **AQ1: Code Reduction (65%)** | Architecture Quality | F07 | CLI command LOC before/after |
| **AQ2: Service Test Coverage (>80%)** | Architecture Quality | F02-F05 | go test -coverprofile per service |
| **AQ3: Duplication Elimination (80%)** | Architecture Quality | F02-F07 | Repository instantiations count |
| **AQ4: Business Logic in Services (100%)** | Architecture Quality | F02-F06 | Code review checklist per PR |
| **DX1: Time to Add Command (75% faster)** | Developer Experience | F01-F07 | Track 3-5 new commands post-epic |
| **DX2: PR Review Cycle Time (70% faster)** | Developer Experience | F01, F07 | GitHub PR metadata analysis |
| **DX3: Test Execution Speed (<100ms)** | Developer Experience | F02-F05 | go test -bench service suite |
| **SH1: API Feature Parity (100%)** | System Health | F08 | CLI vs API endpoint count (optional) |
| **SH2: Regression Rate (<15 bugs)** | System Health | F03, F07 | GitHub issues tagged "E15" + "regression" |
| **SH3: Performance Stability (±10%)** | System Health | F06, F07 | Benchmark key commands before/after |

---

## User Journey Coverage

Validation that all 5 user journeys from `user-journeys.md` are addressed:

| Journey | Persona | Features | Coverage |
|---------|---------|----------|----------|
| **Journey 1: Adding New Command** | Shark Core Developer | F01, F02-F05, F07 | Service layer enables 75% time reduction |
| **Journey 2: Understanding Logic** | AI Code Agent | F01, F02-F05 | Centralized business logic (93% time reduction) |
| **Journey 3: Writing Tests** | Test Engineer | F02-F05 | Service unit tests (90% less code, 900x faster) |
| **Journey 4: API Integration** | API Consumer / DevOps | F08 | Service layer enables API parity (optional) |
| **Journey 5: Maintaining Architecture** | Shark Maintainer | F01, F07 | Self-documenting pattern (80% faster reviews) |

---

## Execution Order Rationale

### Phase 1: Foundation (Week 1)
- **F01 (Order 1)**: Define service layer architecture
  - Why first: All other features depend on interface design and DI patterns
  - Deliverables: Service interfaces, constructor injection, godoc standards

### Phase 2: TaskService Extraction (Weeks 2-4)
- **F02 (Order 2)**: CRUD operations (Create, Get, Update, Delete)
  - Why second: Proves the service layer pattern with lowest-risk operations
  - Impact: Establishes pattern for F03-F04
- **F03 (Order 3)**: Lifecycle operations (Start, Complete, Approve, etc.)
  - Why third: Builds on CRUD, adds workflow integration complexity
  - Impact: Most critical business logic (status transitions)
- **F04 (Order 4)**: Querying (GetNextTask, List, Filter, Dependencies)
  - Why fourth: Completes TaskService with complex filtering logic
  - Impact: Finishes highest-risk extraction (2,664 LOC command → service)

### Phase 3: Parallel Expansion (Week 5)
- **F05 (Order 5)**: Epic/FeatureService expansion
  - Why parallel: Can start after F01, doesn't block TaskService work
  - Impact: Proven pattern (services already exist, just expanding)

### Phase 4: Cleanup (Week 6)
- **F06 (Order 6)**: Repository cleanup
  - Why after F02-F05: Services must exist before moving business logic into them
  - Impact: 106 LOC removed from repositories

### Phase 5: CLI Refactoring (Week 7)
- **F07 (Order 7)**: Slim 47 command files
  - Why last: Requires all services complete (F02-F06)
  - Impact: 65% code reduction (20,952 → ~7,000 LOC)

### Phase 6: Optional Extensions (Weeks 8+)
- **F08 (Order 8)**: HTTP API integration (optional)
  - Why optional: Service layer enables it; can defer to E17 per BA recommendation
  - Impact: 100% API/CLI feature parity
- **F09 (Order 9)**: QueryService consolidation (optional)
  - Why optional: Not blocking core refactoring; P1 priority
  - Impact: Reduced duplication in smart dispatchers

---

## Feature Sizing Breakdown

| Feature | Size | Estimated Effort | Rationale |
|---------|------|-----------------|-----------|
| F01 | Small | 1 sprint (Week 1) | Interface design, no implementation |
| F02 | Medium | 1-2 sprints (Week 2) | First major extraction, establishes pattern |
| F03 | Medium | 1-2 sprints (Week 3) | Workflow integration, status transitions |
| F04 | Medium | 1 sprint (Week 4) | Complex filtering logic |
| F05 | Medium | 2 sprints (Week 5) | Expand 2 services (proven pattern) |
| F06 | Small | 1 sprint (Week 6) | Straightforward logic move |
| F07 | Large | 2 sprints (Week 7) | 47 files, but repetitive pattern |
| F08 | Medium | 1 sprint (Optional Week 8) | HTTP plumbing (service layer enables) |
| F09 | Small | 3-5 days (Optional) | Consolidation, not creation |

**Total Core Timeline**: 7 weeks (F01-F07)
**Total with Optional**: 8-9 weeks (includes F08-F09)

---

## Risk Assessment by Feature

| Feature | Risk Level | Risk Factors | Mitigation |
|---------|-----------|--------------|------------|
| F01 | LOW | Proven pattern exists (Epic/FeatureService) | Follow existing service structure |
| F02 | MEDIUM | First major extraction, establishes pattern | Start with CRUD (lower complexity) |
| F03 | MEDIUM-HIGH | Workflow integration, status transitions | Comprehensive testing, manual regression |
| F04 | MEDIUM | Complex filtering logic | Table-driven tests, edge case coverage |
| F05 | LOW-MEDIUM | Services exist, expanding functionality | Maintain >80% test coverage |
| F06 | LOW | Logic already exists, just moving it | Verify calculations match current behavior |
| F07 | MEDIUM | High volume (47 files), CLI interface compatibility | Incremental PRs (<500 LOC), CI gating |
| F08 | LOW | Service layer enables it (optional) | Defer to E17 per BA recommendation |
| F09 | LOW | Consolidation work (optional) | Low priority, not blocking |

**Overall Epic Risk**: MEDIUM (mitigated by phased approach and incremental PRs)

---

## Architectural Decisions

### Decision 1: Split TaskService into 3 Features (F02-F04)
**Rationale**: Research report identified task.go as 2,664 LOC (32.6% of all command code). Splitting by complexity reduces risk and enables incremental validation.
- **F02 (CRUD)**: Lower complexity, establishes pattern
- **F03 (Lifecycle)**: Medium-high complexity, workflow integration
- **F04 (Querying)**: Medium complexity, completes TaskService

**Alternative Considered**: Single TaskService feature
**Why Rejected**: Too large (3+ sprints), high risk, difficult to review/test

### Decision 2: Parallel F05 (Epic/FeatureService) with F02-F04
**Rationale**: Epic/FeatureService expansion uses proven pattern (services already exist) and doesn't block TaskService work. Enables parallel development.

**Alternative Considered**: Sequential (F02-F04 then F05)
**Why Rejected**: Unnecessary sequencing; F05 can start after F01

### Decision 3: F08 (HTTP API) as Optional P1
**Rationale**: BA feasibility review recommended deferring FR8 to separate epic E17. Service layer completion is sufficient to ENABLE API parity; actual endpoint wiring can follow.

**Alternative Considered**: Include F08 in core epic
**Why Rejected**: API wiring is valuable but not blocking for CLI efficiency gains (60-75% target)

### Decision 4: F09 (QueryService) as Optional P1
**Rationale**: Research report classified QueryService as P1 (not blocking). Smart dispatch consolidation reduces duplication but doesn't block core refactoring.

**Alternative Considered**: Include F09 in core epic
**Why Rejected**: Consolidation can happen after core services exist; not time-critical

---

## Cross-Epic Coordination

### E12: Repository Layer Database Interface Migration
**Sequencing**: E15 → E12 (complete → start)
**Reason**: E15-F06 removes 106 LOC business logic from repositories. E12 should work on cleaned-up repositories.
**Coordination Point**: After E15-F06 completes, E12 can refactor DB interface without touching business logic.

### E16: Multi-Level Workflow System
**Sequencing**: E15-F04/F05 → E16 (Epic/FeatureService expansion → E16 start)
**Reason**: E16 workflow logic should integrate into service layer created by E15.
**Coordination Point**: E15-F05 leaves room for epic/feature workflow integration (no hardcoded status lists).

### E17: HTTP API Parity (Potential)
**Dependency**: E15 → E17 (service layer enables API parity)
**Reason**: E15 creates reusable services; E17 wires HTTP endpoints to services.
**Note**: E15-F08 may be moved to E17 entirely per BA recommendation.

---

## Scope Boundary Validation

### In Scope (from epic scope.md)
✅ Service layer creation (F01-F05)
✅ CLI command slimming (F07)
✅ Repository cleanup (F06)
✅ HTTP API integration (F08 - optional)
✅ Service documentation (F01, F02-F05)

### Out of Scope (correctly excluded)
❌ New CLI commands (not creating new functionality)
❌ Database schema changes (code-only refactoring)
❌ New test frameworks (use existing patterns)
❌ New external dependencies (Go stdlib only)
❌ Performance optimizations (only neutrality required)

### Deferred to Future Epics
🔮 BaseService<T> generics (Epic E18: Advanced Service Patterns)
🔮 Service layer caching (Epic E18)
🔮 Event sourcing for audit trail (Epic E18)
🔮 GraphQL API layer (Epic E16 or E17)

---

## Completeness Verification

### ✅ All Requirements Covered
- FR1-FR9: 100% mapped to features
- NFR1-NFR5: 100% addressed (cross-cutting or per feature)

### ✅ No Scope Overlap
- TaskService (F02-F04) vs Epic/FeatureService (F05): Distinct domains
- Service creation (F02-F05) vs Repository cleanup (F06): Sequential, no overlap
- Service implementation (F02-F06) vs CLI refactoring (F07): Distinct layers

### ✅ Acyclic Dependencies
- Dependency graph is a DAG (Directed Acyclic Graph)
- No circular dependencies detected
- Clear sequential and parallel execution paths

### ✅ Appropriate Sizing
- Small features: F01 (1 sprint), F06 (1 sprint), F09 (3-5 days)
- Medium features: F02 (1-2 sprints), F03 (1-2 sprints), F04 (1 sprint), F05 (2 sprints), F08 (1 sprint)
- Large features: F07 (2 sprints) - justified by high volume (47 files) but repetitive pattern

### ✅ All Success Metrics Traceable
- 10/10 success metrics map to specific features
- Measurement points defined per metric

### ✅ All User Journeys Covered
- 5/5 user journeys addressed across features
- Efficiency gains validated per journey

---

## Next Steps

### Immediate Actions (Post-Approval)
1. **User Review**: Present decomposition summary to user for approval
2. **Advance Epic Status**: `shark epic next-status E15` → Move to `active`
3. **Begin F01**: Start with Service Interface Design (foundation)

### Feature Workflow Sequence
Each feature will go through:
1. **Research Phase** (`ready_for_research` → `in_research`)
   - Architect researches existing functionality
2. **BA Refinement** (`ready_for_refinement_ba` → `in_refinement_ba`)
   - Business Analyst refines PRD
3. **Technical Refinement** (`ready_for_refinement_tech` → `in_refinement_tech`)
   - Architect creates technical architecture
4. **Task Generation** (tasks created with dependencies and order)
5. **Build Phase** (tasks executed, code implemented)
6. **Review & QA** (code review, testing, UAT)

### Feature Kickoff Order
1. **Week 1**: F01 (Interface Design) - Foundation
2. **Week 2**: F02 (TaskService CRUD) - First extraction
3. **Week 3**: F03 (TaskService Lifecycle) - Workflow integration
4. **Week 4**: F04 (TaskService Querying) - Complete TaskService
5. **Week 5**: F05 (Epic/FeatureService) - Parallel expansion
6. **Week 6**: F06 (Repository Cleanup) - Remove business logic
7. **Week 7**: F07 (CLI Slimming) - Refactor 47 commands

### Optional Feature Decision Points
- **F08 (HTTP API)**: Decide at Week 7 completion - include in E15 or defer to E17?
- **F09 (QueryService)**: Decide at Week 7 completion - include or defer?

---

## Approval Checklist

Before advancing epic status, confirm:
- [ ] User has reviewed decomposition summary
- [ ] User approves feature boundaries (7 core + 2 optional)
- [ ] User approves dependency ordering (F01 → F02 → ... → F07)
- [ ] User approves sizing estimates (7 weeks core, +1-2 weeks optional)
- [ ] User approves F08/F09 as optional (defer decision)
- [ ] User approves traceability matrix (all requirements covered)
- [ ] User understands next steps (feature workflow sequence)

---

**Decomposition Status**: ✅ Complete (Pending User Approval)
**Next Action**: User review and approval → `shark epic next-status E15`

