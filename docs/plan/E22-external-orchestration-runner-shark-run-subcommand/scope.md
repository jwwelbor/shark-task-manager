# Scope Boundaries

**Epic**: [External Orchestration Runner - shark run subcommand](./epic.md)

---

## Overview

This document explicitly defines what is NOT included in E22 to prevent scope creep and set clear expectations for feature decomposition.

---

## Out of Scope

### Explicitly Excluded Features

**1. Web UI or Dashboard for Run Monitoring**
- **Why It's Out of Scope**: Observability for `shark run` is limited to CLI output and log files. A real-time dashboard adds significant frontend complexity unrelated to the core enforcement problem.
- **Future Consideration**: A future epic could add a `shark run --watch` or web dashboard once the core runner is stable.
- **Workaround**: Tail log files or use `shark status history <key>` to inspect progress after the fact.

**2. Parallel Agent Execution**
- **Why It's Out of Scope**: Running multiple agents simultaneously (e.g., for independent tasks within a feature) introduces concurrency complexity in worktree management, logging, and error handling.
- **Future Consideration**: After sequential execution is proven, a `shark run --feature E07-F01 --parallel` mode could be added.
- **Workaround**: Run `shark run` on multiple tasks in separate terminal sessions.

**3. Custom Agent Providers Beyond Claude and Codex**
- **Why It's Out of Scope**: Only Claude CLI and Codex CLI have confirmed, stable CLI interfaces. Supporting arbitrary providers (Gemini, Llama, etc.) requires a plugin architecture.
- **Future Consideration**: The dispatcher interface should be designed for extensibility, but only two implementations ship in this epic.
- **Workaround**: Agents with other providers can be invoked manually or wrapped in a script that `shark run` calls as a custom CLI.

**4. Retry Policies with Exponential Backoff**
- **Why It's Out of Scope**: Sophisticated retry logic (exponential backoff, jitter, max attempts) adds complexity. Simple fail-and-stop is sufficient for the first version.
- **Future Consideration**: A `--retry=N` flag could be added to retry failed stages up to N times.
- **Workaround**: Re-run `shark run` manually after investigating and fixing the failure cause.

**5. Distributed Execution Across Machines**
- **Why It's Out of Scope**: `shark run` executes on a single machine with local filesystem access. Distributing stages across machines requires network coordination, shared state, and remote agent management.
- **Future Consideration**: Could be built on top of Turso cloud database for state sharing, but agent dispatch would need a job queue.
- **Workaround**: Run `shark run` on any machine that has the repo and CLI tools installed; Turso syncs the database state.

**6. Modifying or Removing the Existing `/run` Skill**
- **Why It's Out of Scope**: The `/run` Claude Code skill is a separate concern. Deprecating or removing it should happen after `shark run` is battle-tested, not during initial development.
- **Future Consideration**: After E22 is complete, a follow-up task can update `/run` to either call `shark run` under the hood or be formally deprecated.
- **Workaround**: Both `/run` and `shark run` can coexist. `/run` continues to work as before (with its known limitations).

**7. Persistent Run State Across Process Restarts**
- **Why It's Out of Scope**: Run-level metadata (e.g., "stage 5 of 12, retry attempt 2") is held in memory. If the process is killed, this metadata is lost. The task's database status provides sufficient resumption context.
- **Future Consideration**: A `run_sessions` table could track run attempts, but the simpler approach of re-reading current status is sufficient.
- **Workaround**: `shark run` on a task always resumes from the task's current database status. No manual state recovery is needed.

---

### Edge Cases & Scenarios Not Covered

**1. Agent Hangs Indefinitely**
- **Impact**: Low probability but high impact -- the `shark run` process blocks forever.
- **Rationale**: Adding a timeout/watchdog mechanism adds complexity. Claude CLI has its own `--max-turns` limit.
- **Mitigation**: Users can set `--max-turns` in the workflow template instruction. If the agent still hangs, kill the `shark run` process and re-run.

**2. Agent Produces Output But Exits 0 Without Completing Work**
- **Impact**: Medium -- status advances despite incomplete work.
- **Rationale**: Verifying work completeness requires domain-specific validation (e.g., "did the code actually compile?") that belongs in the workflow template instructions, not the runner.
- **Mitigation**: Workflow templates should include verification steps in agent instructions (e.g., "run tests before exiting"). Red-team stages (Codex) catch incomplete work.

**3. Concurrent `shark run` on the Same Entity**
- **Impact**: Low probability but could cause conflicting status transitions.
- **Rationale**: Adding locking for single-entity runs adds complexity for an unlikely scenario.
- **Mitigation**: Document that only one `shark run` should target a given entity at a time. A future enhancement could add advisory locking.

---

## Alternative Approaches Considered But Rejected

**Alternative 1: Stronger Prompt Guards in `/run` Skill**
- **Description**: Add more verification steps, checksums, and self-audit loops to the `/run` Claude Code skill.
- **Pros**: No code changes to shark; purely prompt-level fix.
- **Cons**: Fundamentally unenforceable. The LLM controls its own enforcement. Already tried and failed -- the existing `/run` instructions are already explicit and were ignored.
- **Decision Rationale**: Prompt-level guards cannot prevent the LLM from shortcutting. Architectural enforcement is required.

**Alternative 2: Webhook-Based Orchestration Service**
- **Description**: Build a separate service that listens for shark webhook events and dispatches agents.
- **Pros**: Decoupled architecture; could support distributed execution.
- **Cons**: Requires new infrastructure (webhook server, message queue), adds network hops, complicates deployment. Over-engineered for a single-machine developer workflow tool.
- **Decision Rationale**: shark already exists as a Go binary with full service access. Adding a subprocess dispatcher is simpler and more reliable than a separate service.

**Alternative 3: Python/Bash Wrapper Script**
- **Description**: Write a shell or Python script that calls `shark get --json`, parses output, and dispatches agents.
- **Pros**: Quick to prototype; language flexibility.
- **Cons**: JSON round-tripping overhead, fragile parsing, no access to shark's internal service layer, additional runtime dependency, harder to maintain alongside Go codebase.
- **Decision Rationale**: Go implementation inside shark eliminates JSON parsing, uses type-safe service calls, ships as a single binary, and follows the established architecture.

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| Parallel agent execution for feature-level runs | Medium | E22 (sequential runner must exist first) |
| Run monitoring dashboard | Low | E22 (structured logs must exist) |
| Custom agent provider plugin system | Low | E22 (dispatcher interface must be defined) |
| Persistent run sessions with history | Medium | E22 (core runner must be stable) |
| `/run` skill retirement or delegation to `shark run` | Medium | E22 (shark run must be battle-tested) |

---

*See also*: [Requirements](./requirements.md)
