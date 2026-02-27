# E17: Success Metrics

> Part of [E17: CLI Simplification for AI Agents](epic.md). See also: [Requirements](requirements.md), [Scope](scope.md).

---

## Primary KPIs

### 1. Command Surface Reduction

| Metric | Baseline (Current) | Target | Measurement Method |
|--------|-------------------|--------|-------------------|
| Unique command paths | ~45 | ~25 | Count of leaf commands in `shark --help` (recursive) |
| Commands shown in top-level `--help` | ~15 | ~10 | Count of visible (non-hidden) root subcommands |

**How to measure:** Run `shark --help` and recursively count all leaf subcommands. Hidden aliases are excluded from the count.

### 2. Agent Workflow Efficiency

Baseline data collected from wormwoodGM project activity logs (231 interactions, 2026-02-16 to 2026-02-25).

| Metric | Baseline | Target | Measurement Method |
|--------|----------|--------|-------------------|
| Commands per task lifecycle | 8-10 (with fallbacks) | 5 or fewer | Count commands from "get next" to "mark complete" in agent logs |
| Python post-processing invocations | ~15% of all commands (~30 of 231) | 0% | Count of `\| python3` patterns in agent logs |
| Defensive error suppression | ~36% of all commands (~83 of 231) | Less than 5% | Count of `2>/dev/null` and `2>&1 \|\|` patterns in agent logs |
| Batch for-loops | ~5% of all commands (~12 of 231) | 0% | Count of `for ... do shark ...` patterns in agent logs |
| Status command fallback chains | ~10% of status changes | 0% | Count of `\|\| shark task update` after `shark task set-status` patterns |

### 3. Agent Error Rates

| Metric | Baseline | Target | Measurement Method |
|--------|----------|--------|-------------------|
| Non-existent commands tried | ~3% of all commands (~7 of 231) | 0% | Count of commands that return "unknown command" in agent logs |
| Status command confusion | ~4 different commands tried per status change | 1 command used consistently | Count distinct status-change command patterns per agent session |

---

## Secondary KPIs

### 4. Developer Experience

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Flag name consistency | 100% -- same flag name means the same thing on every command | Audit all commands for `--order` vs `--execution-order`, `--all` vs `--show-all` |
| `--help` discoverability | Agent finds correct command on first try in greater than 90% of attempts | Agent log analysis: ratio of first-try successes to total attempts |
| Backward compatibility | 100% -- no old commands broken through Phase 2 | Full regression test suite; all existing tests pass without modification |

### 5. Performance

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Single command latency | Less than 200ms (same as current baseline) | Benchmark `shark get` with `time` command, average of 10 runs |
| Batch operation latency | Less than 500ms for 20 entities | Benchmark `shark status set --feature` with 20 tasks, average of 5 runs |
| JSON field extraction overhead | Less than 10ms additional vs full JSON output | Benchmark `shark get --field status` vs `shark get --json`, difference of 10 runs |

---

## Measurement Methodology

### Agent Log Analysis

Deploy updated CLI to a real project (wormwoodGM or equivalent) and collect `activity.jsonl` for 1 week of active agent usage (minimum 200 interactions). Compare patterns against baseline data from 2026-02-16 to 2026-02-25.

Specific patterns to search for in logs:
- `| python3` -- indicates Python post-processing (target: 0%)
- `2>/dev/null` -- indicates defensive error suppression (target: less than 5%)
- `for .* do shark` -- indicates manual batch loops (target: 0%)
- `|| shark` -- indicates fallback chains (target: 0%)
- `unknown command` in stderr -- indicates command confusion (target: 0%)

### Regression Testing

- Full test suite must pass at each phase (`make test` returns 0)
- Old command forms must produce identical output (verified by snapshot tests)
- Hidden aliases must be tested alongside new commands
- Exit codes for existing commands must not change

### Performance Benchmarking

Run benchmarks on the same hardware before and after changes:
```bash
# Single command benchmark
for i in $(seq 1 10); do time shark get E07-F01-001 --json > /dev/null; done

# Batch benchmark (after F07)
time shark status set --feature E07-F01 --from todo in_progress

# Field extraction benchmark (after F02)
for i in $(seq 1 10); do time shark get E07-F01-001 --field status > /dev/null; done
```

---

## Success Gates

### Phase 1 Complete When:
- F01 through F05 all implemented and tested
- All existing tests still pass (`make test` green)
- `shark status set` works for tasks, features, and epics (auto-detected from ID)
- `shark status advance` works for all entity types
- `--field` flag eliminates the need for Python parsing in all observed patterns
- JSON errors are structured and include error code, message, entity, and valid_transitions
- `SHARK_OUTPUT=json` activates JSON mode for all commands
- `--order` accepted everywhere; `--execution-order` works as hidden alias

### Phase 2 Complete When:
- F06 through F08 implemented and tested
- Agent log analysis (1 week of real usage) shows:
  - Greater than 80% of status changes use `shark status set` or `shark status advance`
  - 0% batch for-loop patterns
  - Less than 5% Python post-processing patterns
- `shark progress` provides feature/epic rollups with health indicators
- Batch mode handles partial success with per-entity result reporting
- `shark create` dispatcher works for all entity types with consistent syntax

### Phase 3 Complete When:
- F09 through F13 implemented and tested
- Old commands hidden from `--help` but still fully functional
- Agent logs show less than 5% usage of deprecated command forms
- Deprecation warnings appear on stderr for old commands (suppressed in JSON mode)
- All admin commands accessible via `shark admin` prefix

---

## Related Documents

- [Epic Overview](epic.md) - Vision and feature summary
- [Requirements](requirements.md) - Detailed acceptance criteria per feature
- [Scope & Boundaries](scope.md) - What is measured and what is not
