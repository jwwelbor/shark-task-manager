# Overall Code Review — fix/B036-workflow-dispatch-prompts

**Generated:** 2026-07-06 · **Tool:** `/deep-review` (manual fallback + verification) · **Diff:** `HEAD..worktree` · **Effort:** high
**Verdict:** PASS

---

### A. Executive Summary
The reviewed changes repair workflow-index prompt resolution for route-based dispatch surfaces, add exhaustive prompt-crawl coverage, and expand change-card alias handling.

Risk is low after fix verification. The one concrete blocker found during review was brittle test coupling to a deleted repo example bundle plus an implicit prompt root; that is now replaced with a generated workflow-index fixture that explicitly exercises `template_directory`.

Verdict: **PASS**
Counts: 0 blockers / 0 non-blockers triaged / 0 nits

### E. Tests Review
Coverage is strong on the fixed surface:
- `internal/templates/b036_prompt_crawl_test.go` now exercises the embedded-default bundle plus a generated workflow-index bundle with an explicit prompt root.
- `internal/cli/commands/b036_dispatch_prompt_integration_test.go` verifies `shark next --json` prompt rendering across representative entity types against the workflow-index fixture.
- `internal/viewer/server/wire_template_test.go` verifies viewer wiring applies the workflow index prompt root before renderer use.
- `internal/templates/b036_example_bundle_test.go` now directly verifies renderer behavior against the generated fixture instead of depending on deleted repo content.

Counter-factual check: the prior implementation would fail these tests because the prompt root fell back to `shark-data/prompts` and the deleted example-bundle path could not be resolved. The current tests now fail against that old behavior and pass against the fixed behavior.

### F. Quality Rubric
| file | readability | maintainability | performance | testability | standards | notes |
|---|---:|---:|---:|---:|---:|---|
| `internal/test/workflow_index_fixture.go` | 4 | 5 | 5 | 5 | 5 | Centralizes a previously duplicated brittle fixture into deterministic temp-bundle setup. |
| `internal/templates/b036_prompt_crawl_test.go` | 4 | 4 | 5 | 5 | 5 | Keeps the exhaustive crawl while removing dependence on deleted repo files. |
| `internal/cli/commands/b036_dispatch_prompt_integration_test.go` | 4 | 4 | 5 | 5 | 5 | Exercises the production CLI path against the same generated index-bundle contract. |
| `internal/viewer/server/wire_template_test.go` | 4 | 4 | 5 | 5 | 5 | Confirms viewer service wiring applies workflow-index prompt roots end to end. |
| `internal/templates/b036_example_bundle_test.go` | 4 | 4 | 5 | 5 | 5 | Focused renderer smoke test with no repo-layout coupling. |

### G. Risk Hotspots
- Historical branch scope includes an older cleanup commit that deleted repo example content; tests that assumed those paths existed were the main regression vector.
- Workflow-index prompt resolution is now covered in CLI, renderer, viewer, and exhaustive-crawl tests, which materially lowers future drift risk.

### J. Verdict
**PASS**

The reviewed prompt-dispatch changes are in a shippable state. The only blocker found during review was fixed, and `make fmt`, `make lint`, and `make test` all pass on the post-fix worktree.
