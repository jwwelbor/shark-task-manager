# Test Plan: E34-F02 — Evidence-Based Demo Script Skill

**Created:** 2026-07-22  
**Feature brief:** `feature.md`  
**Specification:** `spec.md`  
**Research:** `research-report.md`  
**Parent UAT plan:** None exists for E34; this plan preserves the UAT boundary in the feature brief and specification.  
**Status:** APPROVED FOR TASK GENERATION

## Scope and test model

E34-F02 changes Rider procedure content, embedded skill/template content, bundle registration, and focused test/golden surfaces. It adds no deterministic application behavior, Go command, data model, workflow transition, API, or persistence contract. Every automated case is therefore **content-only**: it drives the shipped renderer/validator or verifies a documented file reference. Human review judges policy wording; tests do not simulate a readiness engine, decision table, mutation, or runtime caller path.

## Spec drift analysis

No drift found. The specification preserves the feature brief's Mode-3 boundary, portable evidence model, no-UAT-authority rule, persisted-artifact contract, and E34-F03 producer/consumer boundary. The parent has no E34 `uat-plan.md`; this is recorded rather than invented as a gate.

| Specification AC | Content-only coverage | Notes |
|---|---|---|
| AC-001 | TC-001, TC-003 | Router/help/verb references and `shark skill get demo-script`; no Go command/workflow transition is human-reviewed. |
| AC-002 | TC-004, TC-005 | Required scenario fields and normal-mode evidence boundary are checked; a specific generated script is manual review. |
| AC-003 | TC-004 | One template check enumerates all seven allowed evidence surfaces. |
| AC-004 | TC-005 | Procedure/skill wording must require an explicit gap and prohibit invention. |
| AC-005 | TC-002 (shared I-01), TC-006 | TC-002 remains the sole shared I-01 contract test; TC-006 checks consumer references only. |
| AC-006 | TC-007 | Artifact path and existing related-document/reference-note/triage instructions. |
| AC-007 | TC-001, TC-003, TC-006, TC-008 | Bundle validation, focused tests, changed prompt goldens, and quality gate. |

## Existing test infrastructure

- `internal/cli/commands/interaction_prompts_test.go` constructs `templates.NewOrchestratorRenderer(findRepoPromptsDir(t))`, renders shipped prompts with `goldenVars()`, and contains the E34 prompt/reference pattern.
- `internal/cli/commands/next_golden_test.go:TestRenderedPromptsGolden` is the golden path when changed Rider content participates in rendered prompts.
- `internal/sharkdata/embed_test.go` and `shark admin validate-data` are the embedded-bundle manifest/file integrity seams.
- `make fmt`, `make lint`, and `make test` are the repository quality gate. No DB fixture, browser suite, simulated runtime harness, or helper is required.

## Acceptance test cases

### TC-001: Demo Rider content resolves through shipped surfaces

**Covers:** AC-001, AC-007; REQ-F-001, REQ-NF-003/004.  
**Technique:** Content-reference enumeration.  
**Entrypoint:** `TestE34F02DemoBundleAndReferences` in `internal/cli/commands/interaction_prompts_test.go`, using `templates.NewOrchestratorRenderer(findRepoPromptsDir(t))` and `goldenVars()`; direct files `skills/shark-rider/SKILL.md` and `skills/shark-rider/verbs/demo.md`.  
**Content-only justification:** This validates host procedure prose and real include resolution, not a deterministic production caller.

**Check:** Render changed Rider prompt templates; assert the router recognizes `demo`, `demo.md` exists, only epic/feature targets plus `--draft` are documented, and the procedure retrieves `demo-script` with `shark skill get` without a `shark demo` or status-advance instruction.

**Expected result:** Includes resolve and the content exposes the explicit Mode-3 route/boundary in `spec.md`.

### TC-003: Demo-script bundle is registered and retrievable

**Covers:** AC-001, AC-007; REQ-F-001, REQ-NF-004.  
**Technique:** Content-reference enumeration.  
**Entrypoint:** `shark admin validate-data`; focused assertions in `internal/sharkdata/embed_test.go`; direct files `internal/sharkdata/default_data/manifest.yaml`, `internal/sharkdata/default_data/skills/demo-script/SKILL.md`, and `internal/sharkdata/default_data/skills/README.md`.  
**Content-only justification:** Manifest identity and skill layout are static bundle contracts exercised by the real validator.

**Check:** Assert normalized `demo-script` identity agrees across manifest, directory, and frontmatter; validate the bundle; retrieve it with `shark skill get demo-script`.

**Expected result:** Validation succeeds and the embedded skill is discoverable through the documented bundle interface.

### TC-004: Template enumerates portable scenario fields and surfaces

**Covers:** AC-002, AC-003; REQ-F-002, REQ-NF-001/002.  
**Technique:** Content-surface enumeration.  
**Entrypoint:** Direct file `internal/sharkdata/default_data/skills/demo-script/context/demo-script-template.md`, checked by the focused bundle/reference test.  
**Content-only justification:** The template specifies required prose fields and allowed evidence categories; it does not execute or classify data.

**Check:** Verify scenario fields: stakeholder value, source, prerequisites/demo data, presenter actions, observable result, evidence type/path, environment/date, readiness classification, reset/recovery, and limitations. Verify UI, CLI, API, SDK, pipeline, infrastructure, and background-process evidence without a framework, package manager, browser, deployment provider, credential, endpoint, or capture tool.

**Expected result:** The reusable template is complete and surface-neutral.

### TC-005: Missing guidance is explicit; proof is never invented

**Covers:** AC-002, AC-004; REQ-F-002/003, REQ-NF-001/002.  
**Technique:** Content-surface enumeration.  
**Entrypoint:** Direct files `skills/shark-rider/verbs/demo.md` and `internal/sharkdata/default_data/skills/demo-script/SKILL.md`, checked by the focused bundle/reference test.  
**Content-only justification:** Normal-versus-draft handling is documented operator policy; simulating it would invent runtime behavior.

**Check:** Verify normal mode requires documented existing environment/date-scoped evidence before `Demonstrated now`; draft labels uncaptured evidence; both prohibit invented commands, credentials, deployments, endpoints, and proof.

**Expected result:** Missing setup/capture guidance remains an explicit gap.

### TC-006: I-01 consumer retains the canonical shared contract without a twin

**Covers:** AC-005, AC-007; REQ-F-003, REQ-NF-002.  
**Technique:** Cross-document reference enumeration.  
**Entrypoint:** Direct files `skills/shark-rider/verbs/demo.md` and `internal/sharkdata/default_data/skills/demo-script/SKILL.md`; canonical shared entrypoint **TC-002** in `../E34-F03-deliverable-feature-decomposition-and-staged-integ/test-plan.md`.  
**Content-only justification:** I-01 is a documentation-policy handoff. Consumer content must preserve source and pointer, not simulate readiness classification.

**Check:** Verify the consumer cites `E34-interaction-map.md#i-01-readiness-evidence-shape`, retains the exact nine read-only fields, and cites shared **TC-002**. Verify contract-only/open activation stays `Not demonstrated / pending integration`, while assessor verdict, owner decision, conditions, and risk remain `Accepted risks and overrides`.

**Expected result:** I-01 source, nine-field shape, activation owner/closure key, and TC-002 pointer are preserved exactly; no consumer-only I-01 test is added.

### TC-007: Artifact discovery and triage boundary are documented

**Covers:** AC-006; REQ-F-004, REQ-NF-003.  
**Technique:** Content-reference enumeration.  
**Entrypoint:** Direct files `skills/shark-rider/verbs/demo.md` and `internal/sharkdata/default_data/skills/demo-script/SKILL.md`, checked by the focused bundle/reference test.  
**Content-only justification:** This checks procedure use of existing Shark data-plane commands without creating artifacts or mutating Shark state.

**Check:** Verify `docs/demos/<entity-key>/demo-script.md`, its `evidence/` directory, `shark related-docs add`, `shark create note --type=reference`, successful-creation ordering, and triage candidates requiring normal deduplication and user confirmation.

**Expected result:** Discoverability reuses existing contracts; discrepancies cannot create backlog work automatically.

### TC-008: Rendered-output regression and repository quality gate

**Covers:** AC-007; REQ-NF-004.  
**Technique:** Regression corpus enumeration.  
**Entrypoint:** `go test ./internal/cli/commands/ -run TestRenderedPromptsGolden` when changed Rider content enters that corpus, then `make fmt`, `make lint`, and `make test`.  
**Content-only justification:** These existing deterministic bundle/repository paths validate rendering and compilation, not a simulated demo or policy engine.

**Check:** Run focused tests; generate/review goldens only when rendered prompt output changes; then run the mandatory quality gate.

**Expected result:** Intentional rendered output is reviewed and all repository checks pass.

## ISO 25010 coverage matrix

| AC | Functional | Performance | Compatibility | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-001 | TC-001, TC-003 | N/A: no runtime path | N/A: no protocol | TC-001 | TC-003 | N/A: no new secret processing | TC-001 | TC-001 |
| AC-002 | TC-004, TC-005 | N/A: prose/template | N/A: no protocol | TC-004 | N/A: no runtime path | Manual policy review | TC-004 | TC-004 |
| AC-003 | TC-004 | N/A: prose/template | TC-004 | TC-004 | N/A: no runtime path | N/A: no security mechanism | TC-004 | TC-004 |
| AC-004 | TC-005 | N/A: prose/template | N/A: no protocol | TC-005 | N/A: no runtime path | Manual policy review | TC-005 | TC-005 |
| AC-005 | TC-002, TC-006 | N/A: policy prose | N/A: no protocol | TC-006 | N/A: no runtime path | Manual policy review | TC-006 | TC-006 |
| AC-006 | TC-007 | N/A: no runtime path | N/A: existing CLI contracts | TC-007 | N/A: no runtime path | N/A: no new secret handling | TC-007 | TC-007 |
| AC-007 | TC-008 | N/A: no runtime path | TC-003 | N/A: developer gate | TC-008 | N/A: no security mechanism | TC-008 | TC-003 |

## Observability and caller-path disposition

No runtime behavior or instrumentation is introduced. Every case names a renderer, validator, command, or direct-file entrypoint and a content-only justification. There is no production caller-path, mock seam, counter-factual, metric, log, trace, or alert to invent. Focused-test and quality-gate output is the implementation verification record.

## Integration coverage

**I-01:** E34-F03 produces and E34-F02 consumes the readiness-evidence shape. Source: `E34-interaction-map.md#i-01-readiness-evidence-shape`. The one shared test is **TC-002** in the E34-F03 test plan. TC-006 is consumer reference conformance, not a second contract test.

**X-##:** None applies. `spec.md` declares no E34-F02 X-## and the product map has no row for this feature; none is invented.

## Manual policy review (completed 2026-07-22)

Reviewed the required future Rider verb, skill, and template contract against
`spec.md` and I-01. The planned content remains an explicit Mode-3 action, does
not grant UAT authority or rewrite verdicts, treats completion/owner decisions
as context, keeps contract-only/open activation pending, limits itself to
documented setup/evidence with no secrets or hard-coded endpoints, and offers
discrepancies only as confirmed triage candidates. This is human judgment,
deliberately outside automated content checks.

## Codex red-team

**Verdict:** UNAVAILABLE — non-blocking. The workflow dispatch did not provide
the required `codex_command`. A local Codex review attempt could not initialize
its in-process app-server client because its required alias state was read-only.
No red-team finding was therefore produced. Retain the required manual policy
review before approval; this does not change the content-only test scope.

## Exit-gate decision

- Every AC has concrete content-only coverage with renderer, validator, command, or direct-file entrypoint and justification.
- Existing renderer, bundle, golden, and repository-gate infrastructure is cited.
- I-01 preserves the exact source, nine-field shape, and shared TC-002 pointer; no twin test is proposed.
- No runtime caller-path, decision-table, mutation, or simulated policy test is invented.

**Recommendation:** Ready for task generation.
