{{define "_product_critical_path_guard"}}PRODUCT CRITICAL-PATH GUARD — consult before choosing or starting any work item:

Check whether these four files exist in this project:
- `docs/product/D01-vision-statement.md`
- `docs/product/D02-success-criteria.md`
- `docs/plan/product-delivery-roadmap.md`
- `docs/plan/product-critical-path.md`

If a file is missing, report it and move on — a missing file is an advisory
note, not a reason to stop:
- `docs/product/D01-vision-statement.md` missing → report: `unresolved prerequisite: docs/product/D01-vision-statement.md missing`
- `docs/product/D02-success-criteria.md` missing → report: `unresolved prerequisite: docs/product/D02-success-criteria.md missing`
- `docs/plan/product-delivery-roadmap.md` missing → report: `unresolved prerequisite: docs/plan/product-delivery-roadmap.md missing`
- `docs/plan/product-critical-path.md` missing → report: `unresolved prerequisite: docs/plan/product-critical-path.md missing`

When the files are present, read them and report all of the following before
choosing or starting work:

1. **Gate name** — the current roadmap gate's name/number, from
   `docs/plan/product-critical-path.md`'s "## Current Gate" section.
2. **Relationship to the gate** — how the proposed work item moves that named
   gate forward.
3. **Executable advancement evidence** — a link to a runnable command or test
   that proves forward movement against the live golden path. None of the
   following count as executable advancement evidence:
   - fixture data
   - a captured/recorded run
   - a hand-authored test actor standing in for a real caller
   - a contract-only test that never exercises the production path
   - a component-level test suite that stops short of the live golden path

   Only evidence that runs against the live golden path — an executed
   command, a passing end-to-end test, or a demonstrated production
   interaction — moves the gate forward.
4. **Unresolved prerequisites** — any of the four files above that is
   missing, stale, or names an unclear gate.
5. **Side-quest call** — when the work item does not move the named gate
   forward, say so explicitly and give one of two calls: proceed, with a
   stated reason (e.g. an urgent bug fix), or defer.

This guard only reports; it never names a target status of its own and never
makes the choice for you.
{{end}}
