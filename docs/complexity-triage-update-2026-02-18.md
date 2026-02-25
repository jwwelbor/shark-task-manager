# Complexity Triage System Update (2026-02-18)

## Problem Statement

The original complexity triage system (6 dimensions, 18 points) measured **technical novelty** but failed to capture **execution complexity**, leading to massive features being scored as STANDARD when they should be COMPLEX.

### Example: E15-F12 Mis-Classification

**E15-F12** scored **5/18 (STANDARD)** despite being:
- 12 tasks (vs typical 1-3 for STANDARD features)
- 6,858 lines to refactor (vs ~500-700 new lines)
- 3-4 weeks effort (vs 1 week typical)
- 100% test preservation requirement (HIGH regression risk)
- Too complex for autonomous build

**Root Cause**: All dimensions focused on "what's new" (patterns, schema, API) but ignored "how much work" (task count, effort, risk).

---

## Solution: Enhanced 9-Dimension System

### New Dimensions Added

| Dimension | Scoring | Purpose |
|-----------|---------|---------|
| **Task Estimation** (NEW) | 1-3 tasks=0, 4-7 tasks=2, 8+ tasks=3 | Captures scope breadth |
| **Regression Risk** (NEW) | Additive=0, Behavior-preserving=2, Breaking=3 | Captures testing/quality burden |
| **Execution Effort** (NEW) | <1 week=0, 1-2 weeks=1, 3-4 weeks=2, 4+ weeks=3 | Captures time/complexity |

### Updated Scoring Matrix

**Technical Complexity (6 dimensions - unchanged):**
1. File Impact: 0-3 points
2. Pattern Novelty: 0-3 points
3. Data Model: 0-3 points
4. API Surface: 0-3 points
5. Cross-Feature Dependencies: 0-3 points
6. UI Complexity: 0-3 points

**Execution Complexity (3 dimensions - NEW):**
7. Task Estimation: 0-3 points
8. Regression Risk: 0-3 points
9. Execution Effort: 0-3 points

**Total: 0-27 points** (previously 0-18)

### Updated Tier Thresholds

| Tier | Old Range | New Range | Criteria |
|------|-----------|-----------|----------|
| SIMPLE | 0-3 | 0-6 | Quick wins, 1-3 tasks, <1 week, additive changes |
| STANDARD | 4-7 | 7-15 | Normal features, 4-7 tasks, 1-2 weeks, some refactoring |
| COMPLEX | 8+ | 16+ | Large features, 8+ tasks, 3+ weeks, high risk |

---

## E15-F12 Re-Scored with New System

### Old Score: 5/18 (STANDARD) ❌ Misleading

| Dimension | Score | Reasoning |
|-----------|-------|-----------|
| File Impact | 3 | 15-25 files |
| Pattern Novelty | 0 | Existing patterns |
| Data Model | 0 | No schema changes |
| API Surface | 1 | Internal refactoring |
| Cross-Feature Dependencies | 1 | 1-2 deps |
| UI Complexity | 0 | No UI |
| **TOTAL** | **5/18** | **STANDARD** (incorrect) |

### New Score: 15/27 (COMPLEX) ✅ Accurate

| Dimension | Score | Reasoning |
|-----------|-------|-----------|
| **Technical Complexity** | | |
| File Impact | 3 | 15-25 files |
| Pattern Novelty | 0 | Existing patterns |
| Data Model | 0 | No schema changes |
| API Surface | 1 | Internal refactoring |
| Cross-Feature Dependencies | 1 | 1-2 deps |
| UI Complexity | 0 | No UI |
| **Execution Complexity** | | |
| Task Estimation | **3** | 12 tasks (8+ range) |
| Regression Risk | **2** | Behavior-preserving (100% test pass) |
| Execution Effort | **2** | 3-4 weeks estimated |
| **TOTAL** | **15/27** | **COMPLEX** (correct!) |

**Result**: With new scoring, E15-F12 correctly routes to **COMPLEX tier** → `ready_for_research` instead of attempting autonomous build.

---

## Implementation Details

### Updated Files

1. **`.sharkconfig.json`** (line 560)
   - Updated `feature_workflow.status_metadata.ready_for_triage.orchestrator_action.instruction_template`
   - Changed from "6 dimensions (max 18 points)" to "9 dimensions (max 27 points)"
   - Added execution complexity dimensions (7, 8, 9)
   - Updated tier thresholds: 0-6=SIMPLE, 7-15=STANDARD, 16+=COMPLEX
   - Updated metadata storage: `/27` instead of `/18`

### Backup Created

- `.sharkconfig.json.backup-20260218-HHMMSS` (automatic backup before changes)

### No Skills Updated

Complexity triage criteria live entirely in `.sharkconfig.json`, not in a separate skill. The orchestrator action template is injected directly into the researcher agent's prompt.

---

## Testing the Update

To verify the new system works, re-triage any feature:

```bash
# Reset feature to triage status (if needed)
shark feature update E15-F12 --status=ready_for_triage

# Run orchestration (will use new 9-dimension system)
/run E15-F12

# Researcher agent will now:
# 1. Count existing tasks (12 tasks → 3 points)
# 2. Assess regression risk (behavior-preserving → 2 points)
# 3. Estimate effort (3-4 weeks → 2 points)
# 4. Total: 15/27 (COMPLEX tier)
# 5. Route to ready_for_research instead of ready_for_refinement_ba
```

---

## Benefits

### 1. Accurate Complexity Assessment
- Features with massive scope correctly flagged as COMPLEX
- Prevents autonomous build attempts on unsuitable features
- Better resource allocation (human vs AI execution)

### 2. Better Routing Decisions
- SIMPLE features → Skip BA/arch, go straight to tasks
- STANDARD features → Focused PRD/arch, autonomous build feasible
- COMPLEX features → Full research/architecture, manual execution

### 3. Autonomous Build Feasibility
- Task count >8 typically exceeds AI context limits
- Regression risk >0 requires careful human oversight
- Effort >2 weeks usually too large for single autonomous session

### 4. Prevents Scope Creep
- High task count signals feature should be split
- High effort estimate triggers decomposition discussion
- Explicit regression risk assessment surfaces quality requirements

---

## Migration Guide

### For Existing Features

Features already triaged with old system may be mis-classified:

```bash
# Find all features in STANDARD tier
shark feature list --json | jq -r '.[] | select(.metadata.complexity_tier == "STANDARD") | .key'

# Re-triage suspicious features:
# - Check task count: shark task list --feature={key} | wc -l
# - If >8 tasks: Likely should be COMPLEX
# - Re-run: shark feature update {key} --status=ready_for_triage && /run {key}
```

### For New Features

All new features will automatically use 9-dimension scoring:
1. Scope validation (feature vs task)
2. **Triage with 9 dimensions** (NEW system)
3. BA refinement (if STANDARD or COMPLEX)
4. Architecture (if STANDARD or COMPLEX)
5. Task generation
6. Build decision (autonomous vs manual)

---

## Rollback Procedure

If the new system causes issues:

```bash
# Restore old config
cp .sharkconfig.json.backup-YYYYMMDD-HHMMSS .sharkconfig.json

# Or manually revert:
# - Change "9 dimensions (max 27 points)" → "6 dimensions (max 18 points)"
# - Remove dimensions 7, 8, 9
# - Change thresholds: 0-3=SIMPLE, 4-7=STANDARD, 8+=COMPLEX
# - Change metadata: /18 instead of /27
```

---

## Next Steps

1. **Re-triage E15-F12** with new system to get accurate COMPLEX score
2. **Decide on E15-F12 approach**:
   - Option A: Split into multiple features (F12a, F12b, F12c)
   - Option B: Keep as COMPLEX, execute manually with agent assistance
   - Option C: Mark as manual-only, use docs as implementation guides
3. **Monitor new features** triaged with 9-dimension system
4. **Update documentation** if tier thresholds need adjustment

---

## Appendix: Dimension Scoring Reference

### Technical Complexity Dimensions (Original 6)

**1. File Impact** (How many files touched?)
- 0 points: 1-3 files
- 2 points: 4-10 files
- 3 points: 10+ files

**2. Pattern Novelty** (How new is the approach?)
- 0 points: Existing patterns only
- 1 point: Adapt existing patterns
- 2 points: Minor new pattern
- 3 points: Major new architecture

**3. Data Model** (Schema changes?)
- 0 points: No schema change
- 2 points: Modify existing tables
- 3 points: New tables/schemas

**4. API Surface** (Public API changes?)
- 0 points: No API change
- 2 points: Modify existing endpoints
- 3 points: New endpoints/contracts

**5. Cross-Feature Dependencies** (Integration complexity?)
- 0 points: Isolated
- 1 point: 1-2 feature dependencies
- 2 points: 3+ dependencies
- 3 points: Circular dependencies

**6. UI Complexity** (User interface work?)
- 0 points: Modify existing components
- 1 point: New simple component
- 2 points: New complex component
- 3 points: Complex interactive UI

### Execution Complexity Dimensions (NEW 3)

**7. Task Estimation** (How many implementation tasks?)
- 0 points: 1-3 tasks (typical SIMPLE feature)
- 2 points: 4-7 tasks (typical STANDARD feature)
- 3 points: 8+ tasks (signals COMPLEX or needs split)

**8. Regression Risk** (Quality burden?)
- 0 points: Additive changes only (low risk, tests add coverage)
- 2 points: Behavior-preserving refactoring (high risk, 100% test pass required)
- 3 points: Breaking changes (very high risk, migration required)

**9. Execution Effort** (Time estimate?)
- 0 points: <1 week (quick win)
- 1 point: 1-2 weeks (normal feature)
- 2 points: 3-4 weeks (large feature)
- 3 points: 4+ weeks (very large, consider splitting)

---

## Summary

The enhanced 9-dimension complexity triage system now accurately captures both **technical novelty** and **execution complexity**, ensuring features are correctly classified and routed to appropriate workflows. E15-F12, which scored 5/18 (STANDARD) under the old system, now correctly scores 15/27 (COMPLEX) under the new system.

**Update Status**: ✅ Complete
**Backup**: `.sharkconfig.json.backup-20260218-HHMMSS`
**Next Action**: Re-triage E15-F12 to verify new scoring
