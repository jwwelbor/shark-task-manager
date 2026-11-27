# B077 Repair Plan Revision 29 Review

plan_path: docs/plan/bugs/B077.repair-plan.md
plan_sha256: sha256:c1a35ce131ff959582dcc33431df6b958b37215af2638f694fe131d8ed1ff6ce

**Reviewer:** Codex independent reviewer
**Candidate:** `bc075a24c749db5a3223c8c3cb50c179a6a6714b` on `fix/B077-model-discovery-20261123`.
**Review baseline:** `bbf490c6ec2bd597d6f3e6e4cdd93e68ca54fa50`; the commit exists and is an ancestor of the candidate.
**Review scope:** revision 29 correction to selected Codex CLI model discovery, exact plan bytes, successor obligations, and their active coverage-index assertions.

## Review result: REPLAN

Revision 29 corrects the two central procedure steps. Step 2 now selects only the Codex CLI host class and provider and discovers the model through `config/read`. Step 3 requires the same selected executable, working directory, profile, environment, and route overrides for discovery and invocation; it passes the discovered value via `--model "$SELECTED_MODEL"` and captures the final executable and argv. The top-level successor requirements also prohibit hardcoded IDs and call for a recording CLI proving discovery and invocation use the same executable. These changes align with Tech Director decision 4273 and Architect decision 4011, and retain B083's separate E40 scope.

One blocker remains in the active coverage index. C02 still specifies “forcing Astra onto a different exact request” as a required counterfactual. C14 still requires the mapped fallback actor to be `researcher@openai/gpt-6.2-astra`; C16 still requires `--harness-model=gpt-6.2-astra` and a selected model of `gpt-6.2-astra`. These are active requirements in the sole C01–C30 coverage index, with no disposition that makes them historical or superseded. A developer following those acceptance rows would preserve the prohibited fixed ID, while a developer following revision 29 steps 2/3 would use the model returned by discovery. The rows must be reconciled with the successor scope, using the recording CLI's discovered model as the oracle and requiring that exact value at the relevant actor, harness, and argv boundaries.

The existing contract test shows why the index needs an explicit, counterfactual discovery assertion: `b077SelectedCodexModel` reads `selected_model` out of the documentation mapping table, and `TestB077RecordingCodexReceivesSelectedRoute` uses that text as its expected launch model. That test checks expanded argv against the table value; it does not establish that config/read and invocation use the same executable or that changing config/read's discovered value changes the selected argv. The successor table states the required evidence at a high level, but the C02/C14/C16 assertions still conflict with it and do not carry that evidence through those consumers.

### Finding F-1 — Active coverage rows still require the fixed model ID

- **Severity:** blocker
- **Disposition:** open
- **Authority:** Tech Director decision 4273; Architect decision 4011; owner instruction 4008 prohibits hardcoded Codex model IDs. Decision 4011 specifically requires discovery from the same selected CLI and explicit propagation, while 4273 scopes the successor work to that discovery/selection/identity follow-up.
- **Evidence:** `docs/plan/bugs/B077.repair-plan.md:87` (C02), `:99` (C14), and `:101` (C16); `tests/contracts/b077_codex_route_attestation_test.go:60-70,108-121`.
- **Required correction:** Rewrite C02, C14, and C16's active acceptance language to use a synthetic model returned by the recording selected CLI's config/read. Require same-executable identity across discovery and invocation, exact discovered-value propagation through argv and resulting actor/harness fields, and a counterfactual where a fixed model or a different discovery result fails. If any old fixed-ID assertions are retained solely as historical delivered behavior, explicitly mark them as non-operative for revision 29 and exclude them from this plan's acceptance gate.

The baseline identity is valid. No other change in revision 29's successor procedure was found to conflict with the stated discovery and explicit-selection approach. Because the active coverage index contradicts the accepted route decision and can pass an implementation that hardcodes the old model, revision 29 is not ready for approval. Return to repair planning for a new revision before development or code review resumes.
