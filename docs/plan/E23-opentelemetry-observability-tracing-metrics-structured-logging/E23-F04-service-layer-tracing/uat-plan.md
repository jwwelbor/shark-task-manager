# E23-F04 Service Layer Tracing — UAT Plan

**Feature**: Service Layer Tracing
**Epic**: E23 — OpenTelemetry Observability

---

## Prerequisites

- F01 (Observability Foundation) and F02 (CLI Lifecycle Integration) tasks are complete and the binary builds.
- `SHARK_OTEL_ENABLED=true` is supported by the CLI root command.

---

## UAT Scenario 1: Span emitted on task get

**Goal**: Verify `TaskService.GetTask` emits a span with the correct name and attribute.

**Steps**:
1. Build binary: `make shark`
2. Run: `SHARK_OTEL_ENABLED=true ./bin/shark get E07-F01-001 2>trace.log`
3. Open `trace.log`

**Expected result**: Log contains a JSON object with `"Name":"TaskService.GetTask"` and an attribute `"task.key":"E07-F01-001"`.

---

## UAT Scenario 2: Error span recorded on not-found

**Steps**:
1. Run: `SHARK_OTEL_ENABLED=true ./bin/shark get E99-F99-999 2>trace.log`
2. Inspect `trace.log`

**Expected result**: Span `TaskService.GetTask` is present and `Status.Code` is `ERROR` with a message containing "not found".

---

## UAT Scenario 3: No OTel output when disabled (default)

**Steps**:
1. Run: `./bin/shark get E07-F01-001 2>trace.log`
2. Check `trace.log` is empty (or contains only logger output, no OTel JSON).

**Expected result**: No span JSON in trace.log; command exits 0.

---

## UAT Scenario 4: Nested spans — task create

**Steps**:
1. Run: `SHARK_OTEL_ENABLED=true ./bin/shark task create E07 F01 "UAT test task" 2>trace.log`
2. Inspect `trace.log`

**Expected result**: Span `TaskService.CreateTask` present with attributes `task.epic_key=E07` and `task.feature_key=F01`.

---

## UAT Scenario 5: Feature transition span

**Steps**:
1. Run: `SHARK_OTEL_ENABLED=true ./bin/shark status advance E07-F01 2>trace.log`
2. Inspect `trace.log`

**Expected result**: Span `FeatureService.TransitionStatus` present with attribute `feature.key=E07-F01`.

---

## UAT Scenario 6: Epic get span

**Steps**:
1. Run: `SHARK_OTEL_ENABLED=true ./bin/shark get E07 2>trace.log`
2. Inspect `trace.log`

**Expected result**: Span `EpicService.GetEpic` present with attribute `epic.key=E07`.

---

## Pass Criteria

All 6 scenarios produce the expected result. Zero regressions in existing commands when OTel is disabled.
