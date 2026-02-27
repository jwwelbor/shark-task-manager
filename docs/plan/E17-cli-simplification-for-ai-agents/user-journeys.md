# E17: User Journeys

> Part of [E17: CLI Simplification for AI Agents](epic.md). See also: [Personas](personas.md), [Requirements](requirements.md).

---

## Journey 1: AI Agent Daily Task Workflow

**Persona:** [DevAgent](personas.md#primary-ai-development-agent)
**Trigger:** Agent is assigned to work on a project and needs to pick up tasks
**Goal:** Complete a full task lifecycle from assignment to review
**Features Used:** F01 (status subcommand), F02 (--field), F04 (SHARK_OUTPUT)

### Current Journey (Pain Points Highlighted)

```
1. Get next task
   shark task next --agent developer --json 2>/dev/null
   → Must remember "task next" not "next task"
   → Must add --json and 2>/dev/null defensively

2. Extract task key from JSON
   ... | python3 -c "import sys,json; print(json.load(sys.stdin)['key'])"
   → No --field flag, must spawn Python process

3. Read task details
   shark task get T-E18-F05-001 --json 2>/dev/null
   → Works, but defensive error suppression needed

4. Start task (change status)
   shark task set-status T-E18-F05-001 in_development 2>&1 ||
   shark task update T-E18-F05-001 --status in_development 2>&1 ||
   shark task next-status T-E18-F05-001 --force 2>&1
   → THREE FALLBACK ATTEMPTS for one operation
   → Agent doesn't know which command exists

5. Check current status
   shark task get T-E18-F05-001 --json | python3 -c "import sys,json; print(json.load(sys.stdin)['status'])"
   → Python again just for one field

6. Complete task
   shark task next-status T-E18-F05-001 --force --notes "Done"
   → Works, but agent may try "complete" or "set-status" first
```

**Total commands:** 6+ (with fallbacks: 8-10)
**External dependencies:** Python 3
**Error handling:** Defensive throughout

### Proposed Journey (After E17)

```
1. Get next task
   shark next --agent developer --field key
   → Single command, returns raw key value (F02)
   → SHARK_OUTPUT=json handles formatting (F04)

2. Read task details
   shark get E18-F05-001
   → Auto-detects task from ID format (existing smart dispatcher)
   → JSON output via SHARK_OUTPUT env var (F04)

3. Start task
   shark status advance E18-F05-001
   → One command, workflow-aware (F01)
   → Knows "advance from current status" means "start development"

4. Check current status
   shark get E18-F05-001 --field status
   → Returns: "in_development" (raw value, no JSON wrapping) (F02)

5. Check valid next transitions
   shark status options E18-F05-001
   → Shows current status + valid transitions (F01)
   → Replaces the next-status --preview pattern

6. Complete task
   shark status advance E18-F05-001 --notes "Implementation complete"
   → Workflow-aware, advances to next logical status (F01)
```

**Total commands:** 5-6 (no fallbacks needed)
**External dependencies:** None
**Error handling:** Structured JSON errors (F03), idempotent operations (F01)
**Reduction:** 40% fewer commands, 100% elimination of Python dependency

---

## Journey 2: Orchestrator Batch Workflow Transition

**Persona:** [OrchestratorAgent](personas.md#secondary-ai-orchestrator-agent)
**Trigger:** Code review is complete for all tasks in a feature, need to advance to QA
**Goal:** Move all tasks in feature E18-F05 from in_code_review to ready_for_qa
**Features Used:** F07 (batch mode), F06 (progress command)

### Current Journey

```
1. List tasks in feature
   shark task list E18 F05 --json 2>/dev/null

2. Filter to in_code_review tasks
   ... | python3 -c "import sys,json; tasks=json.load(sys.stdin); [print(t['key']) for t in tasks if t['status']=='in_code_review']"

3. Loop through each task
   for t in T-E18-F05-001 T-E18-F05-002 ... T-E18-F05-009; do
     shark task update $t --status ready_for_qa 2>&1 || \
     shark task next-status $t --force --status ready_for_qa 2>&1
   done

4. Verify feature status
   shark feature get E18-F05 --json | python3 -c "..."
```

**Total process invocations:** 2 + (2 * N tasks) + 1 = ~20 for 9 tasks
**Failure mode:** If one task fails in loop, others still proceed but no summary

### Proposed Journey (After E17)

```
1. Batch advance all matching tasks
   shark status set --feature E18-F05 --from in_code_review ready_for_qa
   → Single command, operates on all matching tasks (F07)
   → Returns summary: {"updated": 9, "skipped": 0, "failed": 0}

   OR advance individually but in batch:
   shark status set E18-F05-001 E18-F05-002 ... E18-F05-009 ready_for_qa

2. Verify feature progress
   shark progress E18-F05
   → Shows progress rollup, health, task breakdown (F06)
```

**Total process invocations:** 2
**Failure mode:** Partial success with per-entity result reporting
**Reduction:** 90% fewer process invocations (2 vs ~20)

---

## Journey 3: Project Setup (Create Entities)

**Persona:** [DevAgent](personas.md#primary-ai-development-agent) or [HumanDev](personas.md#tertiary-human-developer)
**Trigger:** New feature area needs to be set up with tasks
**Goal:** Create a feature with 3 tasks
**Features Used:** F08 (unified create), F05 (flag normalization)

### Current Journey

```
1. Create feature
   shark feature create E07 "User Authentication" --execution-order=1
   → Must remember positional syntax and flag name

2. Create tasks (3 separate commands)
   shark task create E07 F01 "Design auth flow" --agent=architect --order=1
   shark task create E07 F01 "Implement JWT tokens" --agent=backend --order=2
   shark task create E07 F01 "Add login UI" --agent=frontend --order=3
   → Must use 3-arg positional format
   → --order for tasks vs --execution-order for features (inconsistent)
```

### Proposed Journey (After E17)

```
1. Create feature
   shark create feature E07 "User Authentication" --order=1
   → Unified create syntax: shark create <type> [parent] "title" (F08)
   → --order flag name is consistent everywhere (F05)

2. Create tasks
   shark create task E07-F01 "Design auth flow" --agent=architect --order=1
   shark create task E07-F01 "Implement JWT tokens" --agent=backend --order=2
   shark create task E07-F01 "Add login UI" --agent=frontend --order=3
   → Same pattern: shark create <type> <parent-id> "title" [flags] (F08)
```

**Improvement:** Consistent `shark create <type>` pattern, normalized `--order` flag. Agent learns one pattern that works for all entity types.

---

## Journey 4: Status Check and Decision Making

**Persona:** [OrchestratorAgent](personas.md#secondary-ai-orchestrator-agent)
**Trigger:** Need to decide if a feature is ready for QA phase
**Goal:** Check if all development tasks are complete
**Features Used:** F06 (progress command), F02 (--field), F01 (status options)

### Current Journey

```
1. Get feature tasks
   shark task list E18 F05 --json 2>/dev/null

2. Parse and check statuses
   ... | python3 -c "
   import sys,json
   tasks = json.load(sys.stdin)
   dev_done = all(t['status'] in ['ready_for_code_review','in_code_review','completed']
                   for t in tasks if t['status'] != 'cancelled')
   print('READY' if dev_done else 'NOT_READY')
   "

3. Get feature progress
   shark feature get E18-F05 --json | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('progress_pct', 0))"
```

### Proposed Journey (After E17)

```
1. Check feature progress (includes task rollup)
   shark progress E18-F05
   → Returns progress, task breakdown by status, health indicator (F06)

2. Extract specific metric
   shark progress E18-F05 --field progress_pct
   → Returns: 78.5 (F02)

3. Check what status transitions are available
   shark status options E18-F05
   → Returns current status + valid transitions (F01)
```

**Reduction:** 3 commands with Python reduced to 1-2 commands with no Python dependency.
