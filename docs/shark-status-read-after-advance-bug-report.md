---
title: Shark transient status inconsistency after `status advance`
date: 2026-05-25
repo: /home/jwwel/projects/agentmap
feature: E05-F01
task: T-E05-F01-003
severity: medium
status: observed
---

# Shark transient status inconsistency after `status advance`

## Summary

During a `/command run E05-F01` workflow run, `shark status advance` reported a successful transition for `T-E05-F01-003`, but an immediate follow-up `shark get E05-F01 --json` still showed the task as `in_development`.

The inconsistency was transient. A later read showed the expected final state:
- `T-E05-F01-003` = `completed`
- `E05-F01` = `ready_for_code_review`

This looks like a read-after-write consistency issue or stale status derivation, not permanent data corruption.

## Expected behavior

After:

```bash
shark status advance T-E05-F01-003
```

all immediate follow-up reads should agree:

- `shark get T-E05-F01-003 --json` should show `status: completed`
- `shark get E05-F01 --json` should embed task `T-E05-F01-003` as `completed`
- feature-level derived status/progress should reflect the completed child state consistently

## Actual behavior

Observed sequence:

1. `shark status advance T-E05-F01-003` printed a successful transition:

```text
SUCCESS  Transitioned: in_development -> completed
INFO  Run `shark get T-E05-F01-003 --field orchestrator_action` to get your next instructions.
```

2. Immediate follow-up read of the feature still showed contradictory state:

- top-level feature status had already moved to `ready_for_code_review`
- embedded task list inside `shark get E05-F01 --json` still showed:
  - `T-E05-F01-003.status = "in_development"`
- `action_items.InProgress` also still listed `T-E05-F01-003`

3. A later re-read self-healed and showed the correct final state:

- `shark get E05-F01 --json`
  - `status = "ready_for_code_review"`
  - all three tasks `completed`
  - `progress_pct = 100`
- `shark get T-E05-F01-003 --json`
  - `status = "completed"`
  - `orchestrator_action.action = "archive"`

## Why this matters

This can confuse orchestration agents and humans in the narrow window after advancing status:

- an orchestrator may re-run or re-advance work that already completed
- feature-level status and child-task status can disagree
- follow-up automation may branch on stale embedded child state

## Reproduction

I do not have a guaranteed minimal reproducer yet, but this is the real-world sequence that produced it in AgentMap.

### Preconditions

- Repo: `/home/jwwel/projects/agentmap`
- Feature: `E05-F01`
- Task: `T-E05-F01-003`
- Task state before reproduction: `in_development`
- Feature close to review transition, with sibling tasks already complete

### Steps

1. Ensure:
   - `T-E05-F01-001` = `completed`
   - `T-E05-F01-002` = `completed`
   - `T-E05-F01-003` = `in_development`

2. Advance the task:

```bash
shark status advance T-E05-F01-003
```

3. Immediately read the parent feature:

```bash
shark get E05-F01 --json
```

4. Check whether the embedded task list still reports:

```json
{
  "key": "T-E05-F01-003",
  "status": "in_development"
}
```

while the feature itself has already advanced or is eligible to advance.

5. Re-run the reads a moment later:

```bash
shark get T-E05-F01-003 --json
shark get E05-F01 --json
```

6. Observe whether the state self-corrects.

## Exact commands observed during the incident

```bash
shark status advance T-E05-F01-003
shark get T-E05-F01-003 --field status
shark get E05-F01 --json
shark get T-E05-F01-003 --json
```

There were also feature-level transitions during the same run:

```bash
shark status advance E05-F01
```

That may be relevant if parent status derivation and child completion writes are racing.

## Observed contradictory state

At the inconsistent point in time:

- `shark status advance T-E05-F01-003` had already reported success
- `shark get E05-F01 --json` showed:
  - `status = "ready_for_code_review"`
  - `orchestrator_action.action = "spawn_agent"` for feature-level code review
  - but embedded task `T-E05-F01-003.status = "in_development"`

Later, both entities converged to:

- `E05-F01.updated_at = 2026-05-25T05:23:06Z`
- `T-E05-F01-003.updated_at = 2026-05-25T05:23:06Z`

## Hypotheses

Possible causes:

1. Parent `shark get` reads from a cached or stale derived status snapshot
2. Child status write and parent aggregate/status derivation are not atomic
3. `--field status` and full `--json` may be hitting different code paths/caches
4. Feature advancement can occur while embedded child state is not yet refreshed

## Suggested diagnostics for Shark team

1. Add debug logging around:
   - task status write commit
   - parent aggregate recomputation
   - any caching layer used by `get`

2. Check whether:
   - `shark get <task> --json`
   - `shark get <feature> --json`
   - `shark get <entity> --field status`

   all read through the same freshness guarantees.

3. Verify whether parent feature reads can observe:
   - new feature status
   - old child snapshot

   in the same response.

## Current state

As of the latest read, the inconsistency has resolved:

- `T-E05-F01-003` is `completed`
- `E05-F01` is `ready_for_code_review`
- embedded task list is fully consistent

So this report is about transient inconsistency after an advance, not lasting corruption.
