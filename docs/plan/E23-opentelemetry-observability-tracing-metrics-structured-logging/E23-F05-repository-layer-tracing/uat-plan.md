# E23-F05: Repository Layer Tracing — UAT Plan

**Feature Key:** E23-F05-repository-layer-tracing
**Date:** 2026-03-22

---

## UAT Overview

This UAT verifies that repository methods emit OpenTelemetry spans with correct attributes, that the no-op path adds no visible overhead, and that errors are recorded on spans. UAT is performed against a running Shark CLI with observability enabled.

---

## Prerequisites

1. E23-F01 (Observability Foundation) merged and available.
2. E23-F02 (CLI Lifecycle Integration) merged and available.
3. E23-F05 implementation complete and built: `make build`.
4. A project with at least one epic, feature, and task in the database.
5. `.sharkconfig.json` with observability enabled for the tracing scenarios:

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "metrics_enabled": false,
    "exporter": "stdout",
    "log_level": "info"
  }
}
```

---

## UAT Scenarios

### UAT-001: Span Appears for Task Read

**Steps:**
1. Enable observability as above.
2. Run: `./bin/shark get E01-F01-001 2>/tmp/spans.json`
3. Inspect `/tmp/spans.json`.

**Expected:**
- A span with name `TaskRepository.GetByKey` is present.
- Span attributes include `db.operation=SELECT`, `db.table=tasks`, `db.system=sqlite`.
- Span is a child of a `TaskService.*` span (parent span ID matches).
- No output on stdout except the normal task JSON.

**Pass Criteria:** All three attributes present; span is nested under service span.

---

### UAT-002: Span Appears for Feature Read

**Steps:**
1. Run: `./bin/shark get E01-F01 2>/tmp/spans.json`
2. Inspect `/tmp/spans.json`.

**Expected:**
- A span with name `FeatureRepository.GetByKey` is present.
- Attributes: `db.operation=SELECT`, `db.table=features`, `db.system=sqlite`.

**Pass Criteria:** Span present with correct attributes.

---

### UAT-003: Span Appears for Epic Read

**Steps:**
1. Run: `./bin/shark get E01 2>/tmp/spans.json`
2. Inspect `/tmp/spans.json`.

**Expected:**
- A span named `EpicRepository.GetByKey` is present.
- Attributes: `db.operation=SELECT`, `db.table=epics`, `db.system=sqlite`.

**Pass Criteria:** Span present with correct attributes.

---

### UAT-004: Write Operation Span (Task Create)

**Steps:**
1. Run: `./bin/shark task create E01 F01 "UAT Test Task" 2>/tmp/spans.json`
2. Inspect `/tmp/spans.json`.

**Expected:**
- A span named `TaskRepository.Create` is present.
- Attributes: `db.operation=INSERT`, `db.table=tasks`, `db.system=sqlite`.

**Pass Criteria:** Span present with correct attributes.

---

### UAT-005: Error Span (Task Not Found)

**Steps:**
1. Run: `./bin/shark get E99-F99-999 2>/tmp/spans.json` (non-existent task).
2. Inspect `/tmp/spans.json`.

**Expected:**
- A span named `TaskRepository.GetByKey` is present.
- Span status is `ERROR`.
- `exception.message` attribute contains the error text.

**Pass Criteria:** Span present with error status and exception message.

---

### UAT-006: No Stdout Contamination

**Steps:**
1. Run: `./bin/shark get E01-F01-001 2>/dev/null`
2. Inspect stdout only.

**Expected:**
- Stdout contains only the normal JSON task output (or table output).
- No OTel span JSON, no log lines on stdout.

**Pass Criteria:** Stdout output is identical to pre-F05 baseline.

---

### UAT-007: No-Op When Observability Disabled

**Steps:**
1. Remove or set `observability.enabled=false` in `.sharkconfig.json`.
2. Run: `./bin/shark get E01-F01-001`
3. Verify no span output on stderr.

**Expected:**
- No span output.
- Command completes with same output as before E23.

**Pass Criteria:** Stderr is empty (or contains only non-span output from other sources).

---

### UAT-008: Task List Spans

**Steps:**
1. Enable observability.
2. Run: `./bin/shark list E01 F01 2>/tmp/spans.json`
3. Inspect `/tmp/spans.json`.

**Expected:**
- A span named `TaskRepository.ListByFeature` is present.
- Attributes: `db.operation=SELECT`, `db.table=tasks`, `db.system=sqlite`.

**Pass Criteria:** Span present with correct attributes.

---

## UAT Sign-Off Criteria

All 8 scenarios must pass before this feature is considered UAT-complete. UAT is performed by the product owner or designated reviewer on a non-production environment with a real OTel-compatible viewer (Jaeger, Grafana Tempo, or stdout JSON inspection).
