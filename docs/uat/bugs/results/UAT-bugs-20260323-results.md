# UAT Results — Bug Verification Batch

**Date:** 2026-03-23
**Assessor:** Codex (GPT-5.4) red-team review + human approval
**Bugs:** B002, B003, B004, B005, B006

---

## Codex Assessment Summary

| Bug | Codex Verdict | Conditions | Resolution |
|-----|--------------|------------|------------|
| B002 | Reject | Missing fork/vendor mitigation option | Fixed: Added Option 5 to B002.md |
| B003 | Accept | None | Completed |
| B004 | Accept | None | Completed |
| B005 | Accept with Conditions | (1) Test too broad, (2) stale doc comment | Fixed: Tightened test assertions, updated comment |
| B006 | Accept with Conditions | Test too broad | Fixed: Tightened test assertions |

## Conditions Fixed

### B002 — Added Option 5 (fork/vendor mitigation)
- File: `docs/plan/bugs/B002.md`
- Added vendor/fork option with Go `replace` directive approach
- Documents benefits (version stability, no pseudo-version) and risks (maintenance burden)

### B005 — Tightened test + fixed stale comment
- File: `internal/services/task_service.go` — Updated doc comment from "silently ignores errors" to "logs a warning on error"
- File: `internal/services/task_service_test.go` — Test now asserts specific message ("feature progress recalculation failed") and specific error text

### B006 — Tightened test
- File: `internal/services/entity_service_test.go` — Test now asserts specific message ("failed to create rejection note") and specific error text

## Quality Gate

```
make fmt  — no changes
make lint — 0 issues
make test — all pass (0 failures)
```

## Final Verdicts

| Bug | Final Verdict | Status |
|-----|--------------|--------|
| B002 | Accept | completed |
| B003 | Accept | completed |
| B004 | Accept | completed |
| B005 | Accept | completed |
| B006 | Accept | completed |

**Overall: All 5 bugs verified and completed.**
