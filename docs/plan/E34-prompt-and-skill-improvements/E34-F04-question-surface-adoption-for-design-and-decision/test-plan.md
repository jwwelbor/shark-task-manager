# Test Plan: E34-F04 — Question Surface Adoption for Design and Decision Prompts

**Created:** 2026-08-04
**Feature PRD:** `feature.md`
**Feature specification:** `spec.md`
**Status:** APPROVED

## Scope and drift analysis

This is a content-only feature. The applicable feature specification adds a
registered embedded skill, references it from named decision producers, and
uses rendering and direct-file tests to protect policy wording. It explicitly
excludes E39 runtime, Question persistence, CLI/API, workflow-YAML, and
migration changes.

### Drift findings

None. `spec.md` preserves the feature PRD's reusable-skill, durable Q###,
existing-council, and no-runtime-change boundaries. It adds concrete test names
and golden paths but no implementation scope absent from the PRD.

### Traceability matrix

| Feature requirement | Specification acceptance criterion | Test cases | Coverage |
|---|---|---|---|
| REQ-F-001 reusable Question skill | AC-1 | TC-001 | Complete |
| REQ-F-002 decision-producer adoption | AC-2 and AC-3 | TC-002, TC-003 | Complete |
| REQ-F-003 authority and consumer boundaries | AC-1 and AC-3 | TC-001, TC-003 | Complete |
| REQ-NF-001 content-only bundle integrity | AC-1, AC-2, and AC-4 | TC-001, TC-002, TC-004 | Complete |
| REQ-NF-002 safe durable evidence | AC-1 and AC-3 | TC-001, TC-003 | Complete |
| Content-only delivery scope | AC-5 | TC-005 | Complete |

### Acceptance-criteria review

All five specification criteria have concrete verification. The terms
“material” and “non-material” are not tested by simulating an autonomous
policy engine: the direct-file contract enumerates required procedure language
and each producer's required shared-skill reference. Human review remains the
evidence for whether the resulting wording is appropriate in its workflow.

No parent E34 `uat-plan.md` exists in the checked plan tree. This feature adds
no runtime UAT path; its contribution is bounded to bundle rendering and
content integrity. A later feature-level UAT must use this plan's evidence and
the feature's actual changed-file diff, not invent an E34 UAT scenario.

## AC test matrix

| AC | Test case | Input/setup | Expected outcome | Edge and negative cases |
|---|---|---|---|---|
| AC-1 | TC-001 bundle identity and lifecycle contract | Read embedded `manifest.yaml`, `skills/README.md`, and `skills/question-management/SKILL.md`. | The normalized canonical identity appears in manifest and index; the skill contains materiality, deduplication, configure-before-block, parent ownership, council, resolution, and non-material language. | Missing manifest entry, mismatched `name:`, missing skill file, or missing any required boundary token fails. |
| AC-2 | TC-002 rendered prompt references | Render `epic/refinement.md`, `epic/design.md`, and `feature/specification.md` using `goldenVars()`. | Every render succeeds and includes the shared Question procedure plus durable-Q### policy reference. | Missing include/reference, malformed template, or stale golden output fails. |
| AC-3 | TC-003 producer surface and boundary enumeration | Read each REQ-F-002 producer and the shared skill. | Every producer has a material-decision cue and shared-skill reference; shared skill preserves routine-versus-council, no auto-resolution, parent-owned mutations, and explicitly linked blocking. | A producer that only says “resolve interactively,” a copied council rule, a generic block relation, or a solution-walkthrough auto-mutation instruction fails. |
| AC-4 | TC-004 golden corpus update | Run `TestRenderedPromptsGolden` against the production renderer after prompt edits. | Golden corpus matches the four named fixtures, including epic decomposition through its included workflow, and all changed prompt includes resolve. | Absent fixture, changed rendered bytes, or unresolved include fails. |
| AC-5 | TC-005 content-only scope audit | Review `git diff --check`, changed paths, and `make fmt && make lint && make test`. | No changed Go runtime, E39 model/service/repository/API, schema, migration, or workflow-YAML path; gates pass. | Any changed prohibited runtime path or a failed standard gate fails the feature review. |

## ISTQB technique application

| AC | Technique | Test cases | Rationale |
|---|---|---|---|
| AC-1 | Contract-surface enumeration | TC-001 | The valid bundle identity and required lifecycle vocabulary are a finite, named content contract. |
| AC-2 | Contract-surface enumeration and snapshot comparison | TC-002, TC-004 | The renderer and four named prompt outputs are the production-facing content surface. |
| AC-3 | Decision-table enumeration | TC-003 | Enumerates material/routine, routine/council, worker/parent authority, linked/unlinked block, and walkthrough-consumer boundaries without simulating their prose semantics. |
| AC-4 | Snapshot comparison | TC-004 | Intentional output changes must match production renderer snapshots exactly. |
| AC-5 | Change-surface enumeration | TC-005 | The excluded runtime path set is finite and can be inspected mechanically. |

## ISO 25010 coverage matrix

`N/A (content-only)` means no executable product behavior or runtime
instrumentation is introduced; the named direct-file or renderer test supplies
the relevant evidence.

| AC | Functional suitability | Performance efficiency | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-1 | ✅ TC-001 | N/A (content-only) | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 | ✅ TC-001 |
| AC-2 | ✅ TC-002 | N/A (content-only) | ✅ TC-002 | ✅ TC-002 | ✅ TC-002 | N/A (no credential path) | ✅ TC-002 | ✅ TC-002 |
| AC-3 | ✅ TC-003 | N/A (content-only) | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 | ✅ TC-003 |
| AC-4 | ✅ TC-004 | N/A (content-only) | ✅ TC-004 | N/A (no UI) | ✅ TC-004 | N/A (no credential path) | ✅ TC-004 | ✅ TC-004 |
| AC-5 | ✅ TC-005 | N/A (content-only) | ✅ TC-005 | N/A (no UI) | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 | ✅ TC-005 |

### Coverage gaps

None. Runtime performance, telemetry, and endpoint compatibility are not
applicable because the specification prohibits runtime changes. Human policy
review supplements, but does not replace, the mechanical content checks.

## Observability design

| Behavior | Metric | Log | Trace | Rationale and test evidence |
|---|---|---|---|---|
| Bundle skill discovery and direct policy references | Internal — no runtime observability | None | None | Embedded content is validated in CI; TC-001 and TC-003 read the shipped files. |
| Prompt rendering | Internal — no runtime observability | None | None | TC-002 and TC-004 drive the production renderer; no production telemetry is introduced. |
| Scope exclusion | Internal — no runtime observability | None | None | TC-005 uses changed-path review and the standard quality gate. |

## Caller-path contracts

All test cases are content-only. They name a production renderer or a direct
shipped-file entrypoint; no runtime caller exists above those entrypoints, so
no service/mock contract is appropriate.

| TC | Entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual | Content-only justification |
|---|---|---|---|---|---|
| TC-001 | `readEmbeddedString(t, "manifest.yaml")`, `readEmbeddedString(t, "skills/README.md")`, and `readEmbeddedString(t, "skills/question-management/SKILL.md")` | None; read the compiled embedded bundle | Do not substitute fixture strings or a temporary bundle | A source-only edit that was not embedded or registered would pass a fixture test but fail this test. | No runtime behavior changes. |
| TC-002 | `templates.NewOrchestratorRenderer(promptsDir).Render("epic/refinement.md", goldenVars())`, repeated for the two other named prompts | None; use the production renderer and real prompt tree | Do not mock renderer, includes, or prompt files | A malformed include or absent shared-skill reference would be hidden by a mock renderer. | Prompt-only behavior is rendered content. |
| TC-003 | `os.ReadFile` for every path enumerated in REQ-F-002 and the shared skill | None; real repository files | Do not replace producer files with fixtures | A partial migration that updates only one producer would otherwise appear complete. | The requirement is a finite documentation contract. |
| TC-004 | `TestRenderedPromptsGolden` production renderer corpus | None; real corpus and goldens | Do not compare handcrafted strings instead of generated goldens | A changed prompt could render differently while individual token checks still pass. | Golden snapshots are the specified regression mechanism. |
| TC-005 | `git diff --check` plus repository `make fmt`, `make lint`, and `make test` | No mocks; actual working-tree and CI gates | Do not omit changed-path inspection or use a narrow package-only gate | A runtime/schema change could be smuggled into a content feature while targeted tests remain green. | This is a delivery-boundary audit, not a runtime test. |

## Integration scenarios

| Scenario | Components and boundary | Verification | UAT contribution |
|---|---|---|---|
| Registered skill discovery | Manifest → embedded bundle → contributor index → shared skill | TC-001 proves all three delivery surfaces identify the same canonical skill. | Bundle integrity evidence; no parent E34 UAT scenario is currently defined. |
| Prompt adoption | Prompt renderer → epic/feature prompts → shared Question procedure | TC-002 and TC-004 prove changed prompts parse and produce intended references. | Renderer evidence for later feature review. |
| Policy boundary preservation | Producer content → E39 Question lifecycle; producer content → Shark Attack council; solution walkthrough consumer | TC-003 verifies references rather than recreating E39 or council logic. | Human review checks wording against the feature PRD. |

## Test infrastructure

| Pattern | Existing source | Use |
|---|---|---|
| Embedded bundle content assertions | `internal/sharkdata/embed_test.go:TestE34F02DemoScriptBundle_TC003_TC004_TC005_TC006_TC007` | Follow its real embedded-file read pattern for TC-001. |
| Prompt rendering and cross-feature content assertions | `internal/cli/commands/interaction_prompts_test.go:TestCrossFeatureInteractionLifecyclePrompts` | Follow its production renderer and `goldenVars()` setup for TC-002. |
| Golden corpus | `internal/cli/commands/next_golden_test.go:TestRenderedPromptsGolden` | Regenerate the four specified fixtures, including epic decomposition for the included `decompose-epic` workflow, and use TC-004 for regression. |
| Bundle validation | `internal/sharkdata/embed_test.go` and `internal/sharkdata/default_data/manifest.yaml` | Validate normalized skill identity and required file references. |

New helper code is not needed. Add the two concrete focused tests named by the
feature specification to the existing test files; avoid a simulated policy
engine, real database, browser, network, or telemetry fixture.

## Cross-feature contract tests (I-##)

None. `spec.md` correctly declares that E34-F04 neither produces nor consumes
I-01; the exact I-01 shape source and TC-002 remain E34-F03/E34-F02 ownership.
No new I-## or twin contract test is permitted.

## Cross-epic integration tests (X-##)

None. `spec.md` correctly declares that E34-F04 is not a producer or consumer
of X-06; the product-map row remains E39-F04/E38-F09 ownership. No X-06 test,
coverage pointer, or progress-log deferral is created here.

## Codex Test-Plan Red-Team

**Verdict:** PASS
**Issues raised:** 0
**Issues addressed before dev:** 0
**Issues deferred:** 0

Codex ran in read-only mode against `feature.md`, `spec.md`, and this plan.
It returned `PASS` with no actionable findings after checking AC-1 through
AC-5 for bounded finite tests, ISTQB technique fit, ISO decisions, negative
cases, content-only entrypoints, and correct non-ownership of I-01 and X-06.

## Recommendations

- [x] Ready for development: no unresolved drift; every AC has an enumerated
  content-only test, technique, ISO coverage decision, direct-file or renderer
  entrypoint, negative case, and integration boundary.
- [ ] Needs BA refinement.
- [ ] Needs technical refinement.
