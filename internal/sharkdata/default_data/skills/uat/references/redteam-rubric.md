# UAT Red-Team Assessment Rubric

This is the assessment framework for the **independent red-team reviewer** (Codex) at UAT — the
rubric the reviewer follows and the output format it produces. The UAT skill (`SKILL.md` Step 3)
hands the reviewer the evidence file plus the source artifact paths and points it at this rubric.

The reviewer is **not** the team that built the work. It is the skeptic. Its job is to challenge
the evidence, find gaps, and decide whether the increment should be accepted.

## Reviewer stance

- Run read-only with full filesystem access. **Read each artifact yourself** — do not rely on
  summaries. Cross-check QA/review claims against the actual implementation and test code.
- Do not assume missing evidence means success. Do not invent evidence — if it is not in the
  artifacts, say so. Be conservative when evidence is incomplete.
- Distinguish blocking issues from observations. Trace every conclusion to a specific requirement
  **and** specific evidence.
- Do not fail work for trivial implementation differences when business intent is satisfied.

## ENUMERATE — DO NOT ITERATE

For each acceptance criterion, find **all** violations within each class — not the first one.
Finding one issue per round produces a rejection spiral; finding all of them in one pass lets the
developer fix everything together. Group findings by category, then by class within each category.
Within each class, list every distinct case you can construct. Better to over-report and let the
user triage than to find one and stop. If you find yourself summarizing rather than listing, you
are under-reporting.

For every blocking finding, also emit a one-line **defect-class statement** — the general class
the finding instantiates ("schema required-list omits fields the code dereferences
unconditionally"), not the point instance. The class statement drives the developer's enumeration
sweep and the next round's re-review scope.

## Staged-integration gate integrity

Assess and report the independent assessor verdict, the separate owner decision, its conditions, and the demonstrability disposition as distinct facts. A complete, predeclared `contract-only` edge can be presented for an owner `override-accept` / Accept with Conditions decision, but that decision does not change the assessor verdict and never makes pending work a verified end-to-end delivery.

Missing live wiring, authentication, authorization, integrity, unsafe exposure, and an unmet current-feature acceptance criterion remain blocking even when a future owner is named or an owner selects `override-accept`.

For an activation owner, require evidence of the real caller chain, shared-contract evidence, a production-path integration test, and a wiring-removal counterfactual. An internal activation obligation blocks epic completion until those proofs close it. An external obligation may stay open only with a named future owner and a documented roadmap decision.

## Re-verification rounds — never fix-scoped

When reviewing work that was previously rejected, the round always has three parts, regardless of
how the prompt was phrased:

1. **Verify the named fixes** — confirm each previously cited finding is resolved.
2. **Defect-class sweep** — re-audit the touched functions/modules for every remaining instance of
   each prior finding's defect class.
3. **Full-rubric sanity pass** — re-run the verification checks above over the feature surface.

"Confirm finding N is fixed" is never the whole job. Narrow asks get narrow answers, and each
narrowly-answered round costs a full fix/review cycle when the next instance of the same class
surfaces.

## Critical verification checks — enumerate every instance

1. **Wiring & reachability.** For every new function/class/service introduced, search for call
   sites **outside** test files. List each component with: call-site count, entry-point
   reachability path, DI/registry registration status, API route mount status. **Zero call sites =
   BLOCKER** — dead/unwired code regardless of test results. List ALL unwired components.

2. **Contract consistency.** Enumerate every boundary in the diff:
   - async↔sync — every async function called: is it awaited? every sync function passed where a
     coroutine is expected?
   - producer↔consumer data shapes — every DTO/schema crossing a boundary (DB↔service,
     service↔API, API↔frontend): field names + types match on both sides?
   - DI/registry registrations — every component requiring registration: is it registered?
   - API routes — every endpoint defined: is it mounted in the router?
   - Database migrations — the chain of down_revision links is unbroken; list any orphans.

3. **AC satisfaction.** For each AC, enumerate ALL paths where it could be violated:
   - 'immutable'/'frozen' ACs: every mutation path (attribute rebinding, collection methods
     .append/.clear/.pop/.update/.extend, item assignment, nested mutation via shared references,
     mutable subclass coercion, pickle/copy round-trips, reflection setattr/__dict__).
   - 'secure'/'authorized' ACs: every input class (unauthenticated, expired token, replay, param
     tampering for privilege escalation, injection per user-controlled field, race/TOCTOU between
     auth check and action).
   - 'correct'/'contract' ACs: every public method × every input shape × every output assertion.
   - 'performant'/'SLO' ACs: every input scale (empty, single, large, pathological).

4. **Test coverage gaps.** For each AC, list cases that should exist in the test set but don't. If
   QA reports tests pass but coverage of an AC's enumerated cases is incomplete, call out EVERY gap.

5. **Standards & idiom.** Enumerate violations grouped by section of the project's
   `coding-standards.md` (if it exists).

## Output format

```markdown
## UAT Red-Team Assessment

### Scope
- Epic: [key] — [title]
- Feature: [key] — [title]
- Tasks reviewed: [list]
- Evidence reviewed: [QA reports, code reviews, artifacts]

### Intended Outcome
- Business outcome: [from epic]
- User outcome: [from feature]

### Acceptance Criteria Assessment

| # | Criterion | Source | Status | Evidence | Gap |
|---|-----------|--------|--------|----------|-----|
| 1 | [text] | [Epic AC-1 / Feature AC-1] | Met / Partially Met / Not Met / Cannot Verify | [specific evidence reference] | [what's missing or wrong] |

### E2E Wiring Verification

| Component | Type | Call Sites (non-test) | Entry Point Reachable | Registered/Mounted | Verdict |
|-----------|------|----------------------|----------------------|--------------------|---------|
| [name] | [function/class/endpoint] | [file:line, …] or NONE | Yes/No | Yes/No/N/A | WIRED / DEAD CODE |

**Wiring summary:** [X] components verified, [Y] wired, [Z] dead code

### Risks and Issues

**Blocking:**
- [issue with evidence reference + one-line defect-class statement]

**Non-blocking:**
- [observation with severity: MEDIUM / LOW — CRITICAL and HIGH findings are always blocking]

**Missing evidence:**
- [what's absent and how it affects confidence]

### Decision
- **Verdict:** Accept / Accept with Conditions / Reject / Insufficient Evidence
- **Confidence:** High / Medium / Low
- **Rationale:** [2–3 sentences in business terms]

### Required Follow-ups
- [action items, if any]
```

## Verdict definitions

The severity→verdict mapping is pinned: **any CRITICAL or HIGH finding ⇒ Reject** (never "with
conditions"); MEDIUM-only ⇒ Accept with Conditions; LOW-only ⇒ Accept with notes.

- **Accept** — evidence and checks support every AC; no blocking findings.
- **Accept with Conditions** — acceptable; only MEDIUM-severity conditions remain, and each must be
  tracked as work (route them through triage; see `SKILL.md` Step 8).
- **Reject** — at least one blocking finding (unmet AC with evidence, a wiring/contract BLOCKER, or
  any CRITICAL/HIGH-severity finding).
- **Insufficient Evidence** — the review could not be completed or key artifacts are missing. Do
  not present a pass; report what's missing.

Wiring failures (no call sites, unregistered components, unmounted routes) are **always** blockers
— never downgrade them to non-blocking conditions.

**Always end the assessment with a final delimited line, no matter how much investigation time
remains:**

```
VERDICT: Accept | Accept with Conditions | Reject | Insufficient Evidence
```

This guarantees a parseable verdict even under timeout pressure.
