# Cascade Flow Diagrams

**Feature:** E07-F14 - Cascading Status Calculation

---

## 1. Basic Cascade Flow (Happy Path)

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER ACTION                                  │
│                  shark task start E07-F14-001                   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                  TASK LAYER                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ TaskRepository.UpdateStatus(E07-F14-001, "in_progress")  │  │
│  │   UPDATE tasks SET status = 'in_progress' WHERE key = ?  │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ CASCADE UP
┌─────────────────────────────────────────────────────────────────┐
│                  FEATURE LAYER                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 1. GetTaskStatusBreakdown(feature_id)                    │  │
│  │    → {in_progress: 1, todo: 2, completed: 0}            │  │
│  │                                                           │  │
│  │ 2. DeriveFeatureStatus(counts)                           │  │
│  │    → "active" (because any task is in_progress)         │  │
│  │                                                           │  │
│  │ 3. IF status_override == false:                          │  │
│  │      UPDATE features SET status = 'active'               │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ CASCADE UP
┌─────────────────────────────────────────────────────────────────┐
│                    EPIC LAYER                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 1. GetFeatureStatusBreakdown(epic_id)                    │  │
│  │    → {active: 1, draft: 2, completed: 0}                │  │
│  │                                                           │  │
│  │ 2. DeriveEpicStatus(counts)                              │  │
│  │    → "active" (because any feature is active)           │  │
│  │                                                           │  │
│  │ 3. IF status_override == false:                          │  │
│  │      UPDATE epics SET status = 'active'                  │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                       RESULT                                    │
│  ✓ Task E07-F14-001 started                                    │
│  ℹ Feature E07-F14 status changed: draft → active              │
│  ℹ Epic E07 status changed: draft → active                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Cascade with Manual Override

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER ACTION                                  │
│               shark task complete E07-F14-001                   │
│          (Feature has manual override: blocked)                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                  TASK LAYER                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ UPDATE tasks SET status = 'completed'                    │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ CASCADE UP
┌─────────────────────────────────────────────────────────────────┐
│                  FEATURE LAYER                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 1. GetTaskStatusBreakdown(feature_id)                    │  │
│  │    → {completed: 1, in_progress: 1, todo: 1}            │  │
│  │                                                           │  │
│  │ 2. DeriveFeatureStatus(counts)                           │  │
│  │    → "active" (calculated)                               │  │
│  │                                                           │  │
│  │ 3. CHECK status_override:                                │  │
│  │    ✗ status_override == true                             │  │
│  │    ⚠ SKIP UPDATE (manual override active)                │  │
│  │    ℹ Current status remains: "blocked"                   │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ STILL CASCADE UP
┌─────────────────────────────────────────────────────────────────┐
│                    EPIC LAYER                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ 1. GetFeatureStatusBreakdown(epic_id)                    │  │
│  │    → {blocked: 1, active: 2, draft: 0}                  │  │
│  │                                                           │  │
│  │ 2. DeriveEpicStatus(counts)                              │  │
│  │    → "active" (blocked counts as active)                 │  │
│  │                                                           │  │
│  │ 3. UPDATE epics SET status = 'active'                    │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                       RESULT                                    │
│  ✓ Task E07-F14-001 completed                                  │
│  ⚠ Feature E07-F14 status unchanged (manual override active)   │
│  ℹ Epic E07 status remains: active                             │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Status Derivation Logic (Feature)

```
                    ┌──────────────────────┐
                    │  Get Task Counts     │
                    │  GROUP BY status     │
                    └──────────┬───────────┘
                               │
                               ▼
                    ┌──────────────────────┐
                    │  Count == 0?         │
                    └──────────┬───────────┘
                               │
                ┌──────────────┼──────────────┐
                │ YES                         │ NO
                ▼                             ▼
        ┌──────────────┐          ┌──────────────────────┐
        │ RETURN       │          │ All completed/       │
        │ "draft"      │          │ archived?            │
        └──────────────┘          └──────────┬───────────┘
                                              │
                                   ┌──────────┼──────────┐
                                   │ YES                 │ NO
                                   ▼                     ▼
                          ┌──────────────┐    ┌──────────────────────┐
                          │ RETURN       │    │ Any active?          │
                          │ "completed"  │    │ (in_progress/        │
                          └──────────────┘    │  ready_for_review/   │
                                              │  blocked)            │
                                              └──────────┬───────────┘
                                                         │
                                              ┌──────────┼──────────┐
                                              │ YES                 │ NO
                                              ▼                     ▼
                                     ┌──────────────┐    ┌──────────────┐
                                     │ RETURN       │    │ RETURN       │
                                     │ "active"     │    │ "draft"      │
                                     └──────────────┘    │ (all todo)   │
                                                         └──────────────┘
```

---

## 4. Manual Override Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER ACTION                                  │
│      shark feature update E07-F14 --status=blocked              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│              FEATURE REPOSITORY                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ SetStatusManual(featureID, "blocked")                    │  │
│  │                                                           │  │
│  │ UPDATE features                                          │  │
│  │ SET status = 'blocked',                                  │  │
│  │     status_override = 1,                                 │  │
│  │     updated_at = CURRENT_TIMESTAMP                       │  │
│  │ WHERE id = ?                                             │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ CASCADE TO EPIC
┌─────────────────────────────────────────────────────────────────┐
│                EPIC REPOSITORY                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ RecalculateAndUpdateStatus(epicID)                       │  │
│  │                                                           │  │
│  │ GetFeatureStatusBreakdown:                               │  │
│  │   → {blocked: 1, active: 2}                             │  │
│  │                                                           │  │
│  │ DeriveEpicStatus:                                        │  │
│  │   → "active" (blocked counts as active)                  │  │
│  │                                                           │  │
│  │ UPDATE epics SET status = 'active'                       │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                       RESULT                                    │
│  ✓ Feature E07-F14 status set to blocked (manual override)     │
│  ℹ Epic E07 status changed: draft → active                     │
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Clear Override Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER ACTION                                  │
│       shark feature update E07-F14 --status=auto                │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│              FEATURE REPOSITORY                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │ Step 1: ClearStatusOverride(featureID)                   │  │
│  │                                                           │  │
│  │ UPDATE features                                          │  │
│  │ SET status_override = 0,                                 │  │
│  │     updated_at = CURRENT_TIMESTAMP                       │  │
│  │ WHERE id = ?                                             │  │
│  └──────────────────────┬───────────────────────────────────┘  │
│                         │                                       │
│  ┌──────────────────────┴───────────────────────────────────┐  │
│  │ Step 2: RecalculateAndUpdateStatus(featureID)           │  │
│  │                                                           │  │
│  │ GetTaskStatusBreakdown:                                  │  │
│  │   → {completed: 2, in_progress: 1, todo: 0}            │  │
│  │                                                           │  │
│  │ DeriveFeatureStatus:                                     │  │
│  │   → "active" (has in_progress)                          │  │
│  │                                                           │  │
│  │ UPDATE features SET status = 'active'                    │  │
│  └──────────────────────┬───────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────────┘
                           │
                           ▼ CASCADE TO EPIC
┌─────────────────────────────────────────────────────────────────┐
│                EPIC REPOSITORY                                  │
│  RecalculateAndUpdateStatus(epicID)                            │
│  → Update epic based on all features                           │
└─────────────────────────┬───────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                       RESULT                                    │
│  ✓ Feature E07-F14: status override cleared                    │
│  ℹ Status changed: blocked → active                            │
│  ℹ Epic E07 status updated based on all features               │
└─────────────────────────────────────────────────────────────────┘
```

---

## 6. Database Query Flow (Feature Status)

```
┌─────────────────────────────────────────────────────────────────┐
│                 SQL QUERY                                       │
│                                                                 │
│  SELECT status, COUNT(*) as count                              │
│  FROM tasks                                                     │
│  WHERE feature_id = 123                                         │
│  GROUP BY status;                                               │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                 RESULT SET                                      │
│  ┌───────────────────┬────────┐                                │
│  │ status            │ count  │                                │
│  ├───────────────────┼────────┤                                │
│  │ todo              │   2    │                                │
│  │ in_progress       │   1    │                                │
│  │ completed         │   3    │                                │
│  └───────────────────┴────────┘                                │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│         PARSE TO TaskStatusCounts                               │
│  {                                                              │
│    Todo: 2,                                                     │
│    InProgress: 1,                                               │
│    ReadyForReview: 0,                                           │
│    Blocked: 0,                                                  │
│    Completed: 3,                                                │
│    Archived: 0                                                  │
│  }                                                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│         APPLY DERIVATION LOGIC                                  │
│                                                                 │
│  Total = 6                                                      │
│  Completed = 3                                                  │
│  Active = 1 (in_progress)                                       │
│                                                                 │
│  → Not all completed (3 != 6)                                   │
│  → Has active tasks (1 > 0)                                     │
│  → RESULT: "active"                                             │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│         UPDATE DATABASE (if no override)                        │
│                                                                 │
│  UPDATE features                                                │
│  SET status = 'active', updated_at = CURRENT_TIMESTAMP          │
│  WHERE id = 123 AND status_override = 0;                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## 7. Concurrent Update Handling

```
         TIME →

Thread A:              Thread B:
   │                      │
   │ UPDATE task T1       │
   │ status = completed   │
   │                      │
   ├─ BEGIN CASCADE      │
   │                      │
   │ GetTaskBreakdown    │
   │ (counts: 3/10)      │
   │                      │ UPDATE task T2
   │                      │ status = completed
   │                      │
   │                      ├─ BEGIN CASCADE
   │ Calculate: active   │
   │                      │ GetTaskBreakdown
   │                      │ (counts: 4/10) ← sees T1 + T2
   │ UPDATE feature      │
   │ status = active     │
   │                      │ Calculate: active
   │ ← CASCADE DONE      │
   │                      │ UPDATE feature
                          │ status = active ← overwrites, but same value
                          │
                          │ ← CASCADE DONE

RESULT: Both updates succeeded
        Final status: active (correct)
        No conflicts (same derived status)
```

**Conflict Scenario (rare):**
```
Thread A completes last task → calculates "completed"
Thread B creates new task → calculates "active"

Whichever commits last wins (last-write-wins)
Both are correct for their transaction view
User sees final result after both complete
```

**Mitigation:**
- SQLite WAL mode + transactions = ACID guarantees
- Status changes are idempotent (setting "active" twice is fine)
- Eventual consistency acceptable (updates are fast)
- Pessimistic locking not needed (rarely conflicts)

---

## 8. Performance Optimization Path

```
┌─────────────────────────────────────────────────────────────────┐
│                  CURRENT (Phase 1)                              │
│                                                                 │
│  Read Feature Status:                                           │
│    SELECT status FROM features WHERE id = ?                     │
│    → 1 query, < 1ms                                             │
│                                                                 │
│  Update Task Status:                                            │
│    1. UPDATE task                                               │
│    2. SELECT task counts (GROUP BY)                             │
│    3. UPDATE feature                                            │
│    4. SELECT feature counts (GROUP BY)                          │
│    5. UPDATE epic                                               │
│    → 5 queries, ~30ms                                           │
└─────────────────────────────────────────────────────────────────┘

                             │
                             │ IF performance becomes issue
                             ▼

┌─────────────────────────────────────────────────────────────────┐
│              OPTIMIZED (Future Enhancement)                     │
│                                                                 │
│  Add calculated_status column:                                 │
│    ALTER TABLE features ADD calculated_status TEXT;            │
│                                                                 │
│  Read Feature Status:                                           │
│    SELECT                                                       │
│      CASE WHEN status_override = 1                              │
│           THEN status                                           │
│           ELSE calculated_status                                │
│      END as display_status                                      │
│    FROM features WHERE id = ?                                   │
│    → 1 query, < 1ms (same)                                      │
│                                                                 │
│  Update Task Status:                                            │
│    Same cascade, but updates calculated_status                  │
│    → 5 queries, ~30ms (same)                                    │
│                                                                 │
│  Benefit: Can index on calculated_status for analytics         │
└─────────────────────────────────────────────────────────────────┘

                             │
                             │ IF still need more performance
                             ▼

┌─────────────────────────────────────────────────────────────────┐
│              CACHED (Future Enhancement)                        │
│                                                                 │
│  Add Redis cache layer:                                         │
│    HSET feature:{id} calculated_status "active"                │
│    EXPIRE feature:{id} 300  # 5 minute TTL                      │
│                                                                 │
│  Read Feature Status:                                           │
│    1. Try Redis cache (< 1ms)                                   │
│    2. Fallback to database (< 1ms)                              │
│    → < 1ms average                                              │
│                                                                 │
│  Update Task Status:                                            │
│    1. Update database (5 queries, ~30ms)                        │
│    2. Invalidate cache keys                                     │
│    → ~30ms (same write performance)                             │
│                                                                 │
│  Benefit: Scales to 1000s of reads/sec                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Legend

```
┌─────┐
│ Box │  Process or component
└─────┘

   │
   ▼     Flow direction

   ├──   Decision point / split

   ✓     Success indicator

   ⚠     Warning / skip indicator

   ℹ     Information / note
```

---

**Document Version:** 1.0
**Last Updated:** 2026-01-16
