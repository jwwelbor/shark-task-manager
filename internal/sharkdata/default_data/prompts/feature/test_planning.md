{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Create test plan for feature {{.id}}: "{{.title}}".

Check for existing test-plan.md in feature directory. If test plan exists meeting exit gate criteria, advance immediately.

---

{{include: skills/quality/workflows/test-planning.md}}

READ:
(1) Feature spec.md for requirements and acceptance criteria
(2) Parent epic uat-plan.md for UAT scenarios this feature must satisfy
(3) Feature research report (if exists) for existing test infrastructure
(4) CLAUDE.md testing architecture rules (repo tests use real DB, everything else uses mocks)
(5) Feature spec.md "Cross-feature interactions" section and parent
    interaction map if present
(6) Feature spec.md "Cross-epic integrations" section, parent
    {{.epic_id}}-cross-epic-map.md, and docs/product/cross-epic-integration-map.md
    if present

PRODUCE test-plan.md:

(1) **AC Test Matrix**: For each acceptance criterion in spec.md:
    - Test case description
    - Input/setup
    - Expected outcome
    - Edge cases

(2) **Caller-Path Contract** (per test case, mandatory if a production caller exists above the entrypoint):
    - Production entrypoint (function + argument shape production callers actually use)
    - Lowest allowed mock seam
    - Forbidden mocks (seams that hid prior bugs — e.g. helper-test signatures production never passes)
    - Counter-factual (one sentence: what a buggy impl would do that this test would catch)
    See skills/quality/workflows/test-planning.md Step 5.8 for guidance.

(3) **Integration Scenarios**: Cross-component interactions:
    - Which components interact
    - What to verify at boundaries
    - Reference to epic UAT scenarios this feature contributes to

(4) **Test Infrastructure**: What test utilities exist vs need creation:
    - Existing test patterns to follow (with file paths)
    - New test helpers needed (if any)

(5) ### Cross-feature contract tests (I-##)
    For every I-## the feature spec declares under "Cross-feature
    interactions", design at least one contract test case in this plan.
    - The TC name and location must match the Contract test pointer the feature
      spec declared.
    - Tag the TC with the I-## ID in its description.
    - The SAME TC is referenced by both producer and consumer features. Do not
      write twin tests.
    - These contract tests are what a developer on either side writes first
      (red), then implements code to satisfy (green).
    - Preserve the map-assigned gate mode, counterpart identities, a current
      status read live from Shark, shared-contract evidence, activation owner, closure key, and review basis.
      A `contract-only` edge must test its declared evidence and closure; it
      does not waive live production-path proof for a `live` edge.
    - Required staged fields include `review_basis`; do not omit it from the plan.

(6) ### Cross-epic integration tests (X-##)
    For every X-## the feature spec declares under "Cross-epic integrations",
    design at least one contract, journey handoff, or integration test case in
    this plan unless the row is explicitly deferred in docs/product/progress.md.
    - The TC name and location must match the Test coverage pointer declared in
      the feature spec and global product map.
    - Tag the TC with the X-## ID in its description.
    - Producer and consumer features reference the SAME TC or shared test file
      pointer when the same contract proves both sides.
    - If coverage is deferred, record the deferral decision, owner, and
      follow-up trigger from docs/product/progress.md. Do not silently omit it.
    - Verify the contract / shape source matches the global product map.

### Prompt-only changes

When a feature changes only embedded prompts, skills, templates, or
documentation, do not turn policy prose into simulated application behavior.

- Use the real renderer to verify changed templates and includes resolve.
- Run the rendered-prompt golden test when the bundle output changes.
- Verify that newly documented local bundle or project-file references exist.
- Review policy wording against the specification as a human judgment step.
- Require caller-path, mutation, decision-table, or counterfactual tests only
  when the change adds or alters deterministic runtime behavior.

CRITICAL: Tests trace to FEATURE acceptance criteria (in spec.md), which trace to epic requirements. No orphaned tests. Tests drive the production caller signature, not a convenient helper signature — caller-path contracts close that gap at design time.

EXIT GATE:
- Every AC in spec.md has at least one test case
- Every runtime test case has a caller-path contract (or a documented internal-only justification); every content-only test case names its renderer or direct-file entrypoint and `content-only` justification
- Edge cases identified for each runtime AC
- Integration scenarios cover cross-component boundaries
- Test patterns reference existing infrastructure
- Every I-## declared by the feature spec has at least one contract test case
  whose TC name and location match the declared contract test pointer
- Every X-## declared by the feature spec has test coverage matching the
  product map pointer, or an explicit deferral recorded in docs/product/progress.md

DECISION:
- Exit gate met -> recommended_outcome: pass
- Exit gate not met -> recommended_outcome: fail. This outcome's role is
  `route_rework` — `gate_result.kickbacks` must stay empty; state the
  specific gaps in `gate_result.summary`.

{{template "_gate_result_directive" .}}
