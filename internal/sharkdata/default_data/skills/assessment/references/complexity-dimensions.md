---
inputs:
  # This reference is loaded by the assessment skill during complexity-triage mode.
  # It expects the caller (the SKILL.md craft) to have:
  #   - feature_description (string) — what the feature does
  #   - codebase_summary (string, optional) — file counts, existing patterns
  #   - existing_task_count (integer, optional) — if tasks already decomposed
  # No CLI commands are invoked from inside this reference; "How to assess" sections
  # describe what to look at, not how to fetch it.
outputs:
  # Per-dimension score (0-3) with rationale, returned to caller for aggregation.
---

# Complexity Dimensions — Scoring Reference

Complete scoring criteria for all 9 dimensions (0-3 points each, 27 points max).

## Technical Complexity (6 dimensions)

Measures what's new or novel about the feature.

### 1. File Impact (0-3 points)

**What it measures**: How many files will be created or modified.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | 1-3 files | Add single helper function, fix typo in one file |
| **2** | 4-10 files | Create new service with tests, add validation to 3-4 related files |
| **3** | 10+ files | Refactor architecture across multiple layers, create new subsystem |

**How to assess**:

1. Use the codebase summary (if provided) for file count estimates.
2. Consider: new files + modified files + test files.
3. If no summary is available, request research upstream rather than guessing.

**Common patterns**:

- New feature with service/repository/CLI: typically 4-6 files (score: 2)
- Architecture refactoring: often 15-25+ files (score: 3)
- Bug fix: usually 1-2 files (score: 0)

---

### 2. Pattern Novelty (0-3 points)

**What it measures**: How new/novel are the patterns and architectures being introduced.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | Existing patterns only | Add CRUD endpoint using existing repository pattern |
| **1** | Adapt existing | Extend service layer pattern to new entity type |
| **2** | Minor new pattern | Introduce caching layer, add middleware pattern |
| **3** | Major new architecture | Introduce event sourcing, migrate to microservices |

**How to assess**:

1. Review the codebase summary to identify existing patterns.
2. Compare feature requirements to existing implementations.
3. Ask: "Does this require inventing new approaches?"

**Common patterns**:

- Service layer exists, adding new service of same shape: score 0 (existing pattern)
- Service layer exists, adding caching: score 1-2 (adapt/minor new)
- No service layer, creating first one: score 2-3 (new architecture)

---

### 3. Data Model (0-3 points)

**What it measures**: Database schema changes required.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | No schema change | Add business logic to existing tables, refactor code only |
| **2** | Modify existing tables | Add columns, indexes, or constraints to existing tables |
| **3** | New tables/schemas | Create new tables, relationships, or migrate data models |

**How to assess**:

1. Read architecture doc (if provided) for data model section.
2. Check if feature description mentions "database", "schema", "table", "migration".
3. Look for `.sql` files or migration files in the codebase summary.

**Common patterns**:

- Pure refactoring (service layer): score 0 (no schema)
- Add feature flags table: score 3 (new table)
- Add index for performance: score 2 (modify existing)

---

### 4. API Surface (0-3 points)

**What it measures**: Public API changes (HTTP endpoints, CLI commands, library exports).

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | No API change | Internal refactoring, private method changes |
| **2** | Modify existing endpoints | Add optional parameter, change response format |
| **3** | New endpoints/contracts | New HTTP routes, new CLI commands, new gRPC services |

**How to assess**:

1. Read feature description for "API changes", "endpoints", "commands".
2. Check whether CLI commands are introduced or modified.
3. For HTTP APIs: check for new routes in the architecture doc.

**Common patterns**:

- Service layer refactoring (CLI unchanged): score 0-1 (internal)
- New CLI command: score 3 (new API surface)
- Add `--json` flag to existing command: score 2 (modify existing)

---

### 5. Cross-Feature Dependencies (0-3 points)

**What it measures**: Integration complexity with other features.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | Isolated | Feature works independently, no coordination needed |
| **1** | 1-2 feature deps | Depends on 1-2 existing features (e.g., auth, logging) |
| **2** | 3+ deps | Depends on 3+ features, requires coordination |
| **3** | Circular deps | Features depend on each other, complex orchestration |

**How to assess**:

1. Read the feature description for "Integration Points" or dependency callouts.
2. Check feature dependencies in the PRD.
3. Trace imports/dependencies in the codebase summary.

**Common patterns**:

- New CLI command using existing service: score 1 (depends on service)
- Feature requiring auth + logging + metrics: score 2 (3 deps)
- Circular: Feature A needs B, B needs A: score 3 (rare, design smell)

---

### 6. UI Complexity (0-3 points)

**What it measures**: User interface work (web, CLI, mobile, desktop).

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | Modify existing components | Change CLI output format, update existing React component |
| **1** | New simple component | Add new CLI table renderer, simple form |
| **2** | New complex component | Interactive CLI interface, multi-step wizard, dashboard |
| **3** | Complex interactive UI | Real-time updates, drag-and-drop, rich editor |

**How to assess**:

1. Check whether the feature involves UI changes (web, CLI, mobile).
2. Read the architecture doc for UI/frontend section.
3. Ask: "Does this require new interaction patterns?"

**Common patterns**:

- CLI-only project, modifying table output: score 0
- CLI project, no UI changes: score 0
- Web app, add simple form: score 1
- Web app, add real-time dashboard: score 2-3

---

## Execution Complexity (3 dimensions)

Measures how much work/effort/risk is involved.

### 7. Task Estimation (0-3 points)

**What it measures**: Number of implementation tasks required.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | 1-3 tasks | Simple feature, quick implementation |
| **2** | 4-7 tasks | Standard feature, moderate decomposition |
| **3** | 8+ tasks | Large feature, extensive task breakdown |

**How to assess**:

1. Use `existing_task_count` if tasks are already decomposed.
2. Otherwise estimate from feature scope.
3. Consider: does each task represent days of work?

**Common patterns**:

- Simple feature (1 service method): 1-2 tasks (score: 0)
- Standard feature (CRUD + CLI + tests): 4-6 tasks (score: 2)
- Large feature (multi-layer refactoring): 10+ tasks (score: 3)

**Important**: This dimension captures breadth of work, not just technical novelty.

---

### 8. Regression Risk (0-3 points)

**What it measures**: Risk of breaking existing functionality.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | Additive changes only | New feature, no existing code modified, tests add coverage |
| **2** | Behavior-preserving | Refactoring, 100% test pass required, output unchanged |
| **3** | Breaking changes | Migration required, API changes, backward incompatible |

**How to assess**:

1. Read feature description for "refactoring", "migration", "breaking".
2. Ask: "Will this change existing behavior?"
3. Check acceptance criteria for "100% test pass", "no regression".

**Common patterns**:

- New CLI command: score 0 (additive)
- Service layer refactoring (behavior-preserving): score 2 (high risk)
- API version bump (v1 → v2): score 3 (breaking)

**Critical**: Behavior-preserving refactoring scores 2 (not 0) because test preservation is hard.

---

### 9. Execution Effort (0-3 points)

**What it measures**: Estimated calendar time to implement.

| Score | Criteria | Example |
|-------|----------|---------|
| **0** | <1 week | Quick win, 1-5 days |
| **1** | 1-2 weeks | Normal feature, 5-10 days |
| **2** | 3-4 weeks | Large feature, 15-20 days |
| **3** | 4+ weeks | Very large, 20+ days |

**How to assess**:

1. Consider: file count × task count × complexity.
2. Factor in: regression risk (testing time), pattern novelty (learning time).
3. Estimate conservatively (real work takes longer than ideal).

**Common patterns**:

- Simple feature (2 files, 2 tasks, existing patterns): 3-5 days (score: 0)
- Standard feature (6 files, 5 tasks, adapt patterns): 8-12 days (score: 1)
- Large refactoring (20 files, 10 tasks, behavior-preserving): 20-30 days (score: 2-3)

**Formula heuristic**:

```
Effort (days) ≈ (File Impact × 2) + (Task Count × 1.5) + (Pattern Novelty × 3)

If behavior-preserving (regression risk = 2): multiply by 1.5
```

**Autonomous build consideration**: features scoring ≥2 on Execution Effort often exceed AI context limits.

---

## Total Score Calculation

Sum all 9 dimensions:

```
Total = File Impact + Pattern Novelty + Data Model + API Surface +
        Dependencies + UI Complexity + Task Estimation +
        Regression Risk + Execution Effort

Max: 27 points (9 dimensions × 3 points each)
```

---

## Scoring Examples

### Example 1: SIMPLE-tier feature (score 4/27)

**Feature**: Unified Entity Display Rendering (CLI rendering helpers)

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| File Impact | 2 | 5-7 files (2 new, 2-3 modified) |
| Pattern Novelty | 1 | Adapting existing helper pattern |
| Data Model | 0 | No schema changes |
| API Surface | 0 | CLI output only, backward compatible |
| Dependencies | 1 | DisplayService, workflow.Service |
| UI Complexity | 0 | Modifying existing terminal output |
| **Technical Total** | **4** | |
| Task Estimation | 0 | 2-3 tasks |
| Regression Risk | 0 | Additive changes (new helpers) |
| Execution Effort | 0 | 5-7 days estimated |
| **Execution Total** | **0** | |
| **TOTAL** | **4/27** | **SIMPLE** (0-6 range). Note: a feature can look full-featured by intent yet still score SIMPLE when its technical/execution mix is low — the tier is driven by the score, not the ambition |

### Example 2: Borderline COMPLEX feature (score 12/27)

**Feature**: Service Layer Completion and CLI Integration

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| File Impact | 3 | 15-25 files (repositories, CLI commands) |
| Pattern Novelty | 0 | Existing service layer pattern |
| Data Model | 0 | No schema changes |
| API Surface | 1 | Internal refactoring (CLI signatures unchanged) |
| Dependencies | 1 | Depends on prior service-foundation features |
| UI Complexity | 0 | No UI changes |
| **Technical Total** | **5** | |
| Task Estimation | 3 | 12 tasks (8+ range) |
| Regression Risk | 2 | Behavior-preserving (100% test pass required) |
| Execution Effort | 2 | 3-4 weeks (20-25 days) |
| **Execution Total** | **7** | |
| **TOTAL** | **12/27** | **STANDARD** (7-15) but execution dimensions push toward COMPLEX |

**Note**: 12/27 is technically STANDARD, but the high task count (12), high regression risk, and 3-4 week effort make autonomous build infeasible. This illustrates why tier alone isn't sufficient — autonomous build feasibility requires checking execution dimensions independently.

---

## Calibration Tips

1. **File Impact**: use codebase summary for accurate counts; if missing, request upstream.
2. **Pattern Novelty**: compare to similar implementations to gauge novelty.
3. **Task Estimation**: count actual tasks if they exist.
4. **Regression Risk**: read acceptance criteria for "100% test pass", "no behavioral changes".
5. **Execution Effort**: use formula heuristic, validate with past features.

**When in doubt**: round up on execution dimensions (7-9), round down on technical dimensions (1-6).

**Autonomous build threshold**: features scoring ≥2 on Execution Effort often exceed AI capabilities.
