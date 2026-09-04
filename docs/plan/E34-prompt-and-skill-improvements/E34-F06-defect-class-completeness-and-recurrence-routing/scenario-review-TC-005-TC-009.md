---
feature_key: E34-F06-defect-class-completeness-and-recurrence-routing
epic_key: E34
title: TC-005–TC-009 Scenario Review — Reviewed Evidence Record
task: T-E34-F06-003
---

# TC-005–TC-009 Scenario Review

Per test-plan.md's "Prompt-only changes" gate and its "New test helpers
needed: none" note, TC-005 through TC-009 have no fixture-execution harness
— there is no Go decision function to unit-test, only prose a future AI
worker follows. This document is that reviewed-evidence record: for each
test case, it feeds the fixture described in test-plan.md's AC Test Matrix
against the actual rendered content of
`internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md`
(the canonical workflow file produced by T-E34-F06-001) and traces the
section of that file that produces the expected outcome. Section references
use the file's own `##` headers, which are stable anchors — no line numbers,
since those shift on future edits.

This satisfies task T-E34-F06-003's AC-T1 (each of the seven feature.md
acceptance scenarios has a reviewed test case per TC-005–TC-009) and
contributes to AC-T3 (`TC-I-03-DEFECT-CLASS-CLOSURE` = TC-005 + TC-008 +
TC-009, see the closing section below).

## TC-005 — Close an enumerated class

**Fixture:** 1 finding, 1 matching sibling instance, both fixed, guard
verified.

**Walkthrough:** "Enumeration procedure" requires every matched instance to
be dispositioned as `fixed`, `dispositioned`, or `open`, with
`fixed_count + dispositioned_count + open_count = matching_count`. With both
instances `fixed`, `open_count = 0` and `matching_count = fixed_count`
(satisfying `matching_count = fixed_count + dispositioned_count` since
`dispositioned_count = 0`). "Structural guard closure" then requires, before
`status: complete` may be reported: (a) every instance `fixed` or
`dispositioned` (`open_count = 0` — true here), (b) the completeness
identity above (true here), and (c) `guard.status = verified` by an observed
counterfactual in both directions. The fixture states the guard is verified,
so all three closure conditions hold and the workflow content instructs
`status: complete`.

**Edge case (guard not verified):** "Structural guard closure" makes
`guard.status = verified` a hard precondition for `status: complete`,
independent of the instance-level counts — an unverified guard forces
`status: open` (with a linked work item per the same section) even when
`open_count = 0`. The rendered guidance therefore cannot report `complete`
on instance counts alone.

**Verdict: PASS.** The workflow content produces the documented
`status: complete`, `open_count: 0`, `matching_count = fixed_count +
dispositioned_count` decision for this fixture, and correctly withholds
`complete` when the guard is unverified.

## TC-006 — Distinguish recurrence from a new finding

**Fixture:**
- A: same fingerprint resurfaces after a recorded repair.
- B: new fingerprint, same `class_key`, inside a completed `search_scope`.
- C: new fingerprint outside that scope.
- D: new fingerprint, **different** `class_key`, inside a completed
  `search_scope` (the HIGH-3 boundary case).

**Walkthrough:** "Recurrence classification" defines exactly two buckets:
**Recurrence** — "the same fingerprint resurfaces after a recorded repair, or
a new fingerprint belongs to the **same `class_key`** as a previously
`status: complete` class **and** lies inside that class's recorded
`search_scope`. Both conjuncts are required" — and **Normal finding** —
"anything else... A different defect class that merely happens to live
inside an old scope's file/module footprint is a normal finding, not a
recurrence." Fixture A matches the first recurrence clause verbatim (same
fingerprint, recorded repair). Fixture B matches the second recurrence clause
verbatim (new fingerprint, same `class_key`, inside a completed class's
`search_scope` — both conjuncts satisfied). Fixture C matches neither
recurrence clause (new fingerprint, outside the completed scope), so it
falls to "Normal finding" and "routes through ordinary rework, not
recurrence handling."

**Edge case (class_key discriminator, UAT HIGH-3 fix):** The section makes
both conjuncts mandatory: "matching `class_key` alone (outside the recorded
scope) or scope membership alone (under a different `class_key`) is not
recurrence." A fourth fixture — D: a new fingerprint under a *different*
`class_key` that happens to fall inside a completed class's `search_scope` —
matches scope membership but not `class_key`, so it fails the compound rule
and classifies as a normal finding, not recurrence. This is exactly the
boundary case HIGH-3 identified as unenforced before the fix; the corrected
text now requires both conjuncts, so scope membership alone cannot
misclassify fixture D as recurrence.

**Edge case (no round-count field):** The same section states explicitly:
"No round-count field or round-counting logic is used anywhere in this
classification — recurrence is decided by fingerprint, `class_key`, and
scope membership, never by 'this is the Nth time we've seen a finding
here.'" A grep for a round-counting concept (`round`, `Nth time`, `attempt
number`) against the classification logic in this section returns no such
field — the only inputs named are `fingerprint`, `repair_record`,
`class_key`, and `search_scope` membership.

**Verdict: PASS.** A/B classify as recurrence, C and D classify as normal
findings (D specifically because `class_key` fails to match despite scope
membership), and no round-count field appears in the classification logic.

## TC-007 — Route a severity conflict

**Fixture:** Fresh evidence materially changes risk on a previously-accepted
fingerprint (HIGH finding conflicts with a prior accepted LOW decision).

**Walkthrough:** "Disposition and severity-conflict routing" covers exactly
this shape: "When a finding's disposition would conflict with a prior
accepted decision on a matching fingerprint — for example, fresh evidence
materially raises the risk of a previously-accepted low-severity finding —
do not resolve the conflict unilaterally inside this workflow." It then
names the two routing mechanisms by name — `question-management`
(`skills/question-management/SKILL.md`) for a bounded, single-owner
disagreement, and "the project's multi-specialist council deliberation
workflow" for specialist disagreement, cross-entity inconsistency, high
blast radius, irreversibility, or no safe evidence path. It then states the
corrected routing shape (post-HIGH-4 fix): `severity_conflict` "is an outer
`GateResult.Finding.disposition` value ..., not an I-03 instance
disposition — the sweep's own `instances[].disposition` stays within
`{fixed, dispositioned, open}`." The conflicted instance is recorded as
`open` inside `instances` (pending the outer conflict's resolution, with
`evidence` pointing at the conflict), and the calling gate adds or updates a
`Finding` in its own `GateResult.findings` array with `disposition:
severity_conflict` and a `disposition_pointer` back to the instance's
`fingerprint`/`site_pointer`. It closes with: "Block normal advancement at
the `Finding` level until the referenced mechanism resolves it — a conflict
must never be routed silently through ordinary rework, and must never be
closed by marking the I-03 instance itself `severity_conflict`."

**Edge case (silent routing):** The section's closing sentence is an
explicit prohibition on exactly the failure mode TC-007 tests for — routing
a conflict through normal rework without a block, or closing it by marking
the I-03 instance itself `severity_conflict`. There is no code path in the
section that permits silent routing; every conflict is required to resolve
via the outer `Finding.disposition = severity_conflict` plus a block, while
the instance itself stays `open`.

**Verdict: PASS.** The workflow content records the conflicted instance as
`open` inside `instances`, routes the conflict to the outer
`GateResult.Finding.disposition = severity_conflict` with a
`disposition_pointer`, blocks normal advancement, and references both the
Question and council mechanisms by name.

## TC-008 — Zero remaining instances

**Fixture:** Full re-enumeration re-run after all instances were already
fixed (a re-verification pass finds nothing new).

**Walkthrough:** "Zero-result reporting" states: "A pass that finds no
additional instances is a normal, reportable outcome — never an omitted or
empty result. Report `searched_count` (the real number of candidates
checked) and explicit `matching_count: 0`, `open_count: 0`. A missing or
blank instance list is indistinguishable from 'the sweep was never run' —
always emit the counts, even when they are all zero." This directly requires
all three counts (`searched_count`, `matching_count: 0`, `open_count: 0`) to
be explicitly present, not omitted, on a zero-result pass. The
"Self-verification" checklist reinforces this as a pre-return gate:
"`searched_count`, `matching_count`, `fixed_count`, `dispositioned_count`,
and `open_count` are all present and consistent, including on a zero-result
pass."

**Edge case (omitted counts):** Both the dedicated "Zero-result reporting"
section and the closing self-verification checklist treat an
omitted/blank/empty result on a zero-instance pass as a failure of the
procedure, not an acceptable shorthand — the content gives no path to skip
emitting the counts.

**Verdict: PASS.** The workflow content requires `searched_count`,
`matching_count: 0`, and `open_count: 0` to be reported explicitly, never
omitted, on a zero-result pass.

## TC-009 — Missing/disabled/ineffective guard

**Fixture:** A sibling defect is reintroduced with (a) no guard, (b) a guard
present but disabled, (c) a guard present but that does not actually detect
the class when reintroduced.

**Walkthrough:** "Structural guard closure" requires "The selected guard has
passed a counterfactual verification: it catches (flags, fails the build, or
otherwise blocks) the class when the defect is deliberately re-introduced,
and it does not flag/fail when the defect is absent. A guard that misses the
reintroduced defect, or that false-positives when the defect is absent, is
not verified. Set `guard.status = verified` only after both directions of
that counterfactual are actually observed — an unverified or
merely-plausible guard does not count." Applying the three fixture cases:

- (a) **No guard**: "Guard selection" requires selecting "an executable
  guard that would have caught this class"; with none selected, there is no
  guard to verify, so `guard.status` cannot become `verified`. "Structural
  guard closure"'s fallback applies: "If no feasible guard exists for this
  class, the class stays `status: open` and must carry a linked work item
  ... tracking the gap."
- (b) **Guard present but disabled**: a disabled guard cannot be observed
  catching-then-not-flagging across the counterfactual's two directions (it
  can't catch the class when re-introduced if it never runs at all, and "it
  does not flag/fail when the defect is absent" is likewise unobservable) —
  the counterfactual is not actually run, so `guard.status = verified`
  cannot be set on "actually observed" evidence.
- (c) **Guard present but ineffective (does not detect the reintroduced
  defect)**: this is precisely the counterfactual's first direction failing
  — the guard must catch the class "when the defect is deliberately
  re-introduced," and "a guard that misses the reintroduced defect ... is
  not verified" directly covers this fixture — the observed counterfactual
  result is negative and `guard.status = verified` is withheld.

In all three cases, "Structural guard closure" blocks `status: complete`
("If no feasible guard exists ... the class stays `status: open`"; and more
generally `status: complete` requires `guard.status = verified`, which none
of (a)/(b)/(c) satisfy) until an executable counterfactual proves detection.

**Edge case (guard accepted without counterfactual evidence):** The phrase
"an unverified or merely-plausible guard does not count" is the direct
prohibition on this failure mode — a guard cannot be accepted as verified on
the strength of existing/being enabled alone; only an *observed* two-way
counterfactual result counts.

**Verdict: PASS.** All three guard-failure cases are required to fail
closure until an executable counterfactual proves detection, matching
TC-009's expected outcome.

## TC-011 — Backward-looking rework: compatible fix or cited divergence (REQ-F-002)

**Fixture:**
- A: the search surfaces a recorded prior fix design (a decision note) for
  this class, and the rework implements that design as-is.
- B: the search surfaces a recorded prior fix design, and the rework
  implements a *different* fix, citing durable new evidence (a changed
  requirement) in the instance's `evidence` field as the reason for
  diverging.
- C (violation): the search surfaces a recorded prior fix design, and the
  rework implements a different fix with no cited evidence — it silently
  diverges.
- D: the search finds no recorded prior design for this class (first
  occurrence) — nothing to be compatible with or diverge from.

**Walkthrough:** REQ-F-002 requires three things: search prior treatment
first, then "Implement a recorded compatible fix design or cite the durable
evidence that justifies divergence," and "preserve unrelated owner
decisions; do not reinterpret an existing disposition without new evidence."
The workflow's "Backward-looking rework" section encodes the search list
(code/tests, feature/epic decisions, tech-debt records, prior review-finding
notes, spec/architecture sections, standards docs) and then states: "When
that search surfaces a recorded prior design or fix for this class ...
implement that design, or a fix compatible with it. Diverging from a
recorded design is only valid when the divergence is justified by durable
evidence ...; cite that evidence in the instance's `evidence` field. A
repair that silently does something different from a recorded prior design,
with no cited justification, does not satisfy this section."

Applying the fixtures: (A) implementing the recorded design as-is directly
satisfies "implement that design" — compliant. (B) implementing a different
fix while citing durable new evidence in `evidence` satisfies the
divergence branch — compliant. (C) implementing a different fix with **no**
cited evidence is the exact failure mode the section names and rejects — "a
repair that silently does something different ... does not satisfy this
section" — **non-compliant**, and the section further calls this itself "a
class instance of 'the rework guessed instead of searching.'" (D) with no
recorded prior design found, there is nothing to be compatible with or
diverge from, so neither branch applies and ordinary rework proceeds — this
is outside the enforcement's scope by construction, not a gap in it.

**Edge case (silent override without a stated reason):** The adjacent
sentence — "Preserve an existing disposition on a matching fingerprint
unless new evidence contradicts it — do not silently re-open or re-decide a
already-dispositioned instance without a stated reason" — reinforces the
same enforcement for the already-dispositioned case specifically: a fixture
E (re-deciding a dispositioned instance with no stated new-evidence reason)
is likewise rejected by this section, not merely discouraged.

**Verdict: PASS.** Fixtures A and B (implement-the-design and
cite-divergence-evidence) are the two compliant paths; fixture C
(silent divergence, no cited evidence) is explicitly named and rejected by
the workflow's own text, not merely implied by the search-first framing;
fixture D falls outside the rule's scope by construction (no recorded
design exists to diverge from).

## TC-012 — Accepted-risk (visible, non-blocking) branch (REQ-F-006)

**Fixture:**
- A (accepted-risk): a recurring finding matches a fingerprint already
  covered by a dated, owner-grounded acceptance decision; no material new
  evidence has appeared since that decision.
- B (fresh-evidence conflict, contrast with TC-007): the same shape, but
  fresh evidence materially changes the risk since the acceptance decision.

**Walkthrough:** REQ-F-006 requires: "Keep a recurring finding visible but
non-blocking when a dated, owner-grounded decision covers the same
fingerprint and no material new evidence changes the risk." The workflow's
"Disposition and severity-conflict routing" section's "Accepted-risk
(visible, non-blocking) branch" paragraph implements this directly: mark the
instance `dispositioned` (not `open`, not `severity_conflict`) with
`evidence` citing the accepting decision's pointer; it stays visible in
`instances` but does not block advancement and does not route through
Question or council — "the decision already stands." Fixture A satisfies
this branch exactly.

Fixture B is explicitly distinguished as out of this branch's scope by the
same paragraph's closing sentence — "only escalate when new evidence
materially conflicts with that prior decision, per the severity-conflict
path below" — routing instead to the `severity_conflict` path this
document's TC-007 already covers (fresh HIGH finding conflicting with a
prior accepted LOW decision).

**Verdict: PASS.** Fixture A (no new evidence) is visible-but-non-blocking
via `dispositioned`; fixture B (new evidence) correctly falls through to the
separate severity-conflict path (TC-007), not this branch — the workflow
text distinguishes the two by the presence of material new evidence, not by
severity or recurrence count.

## TC-I-03-DEFECT-CLASS-CLOSURE cross-reference

Per test-plan.md's Cross-feature contract test table,
`TC-I-03-DEFECT-CLASS-CLOSURE` = TC-005 + TC-008 + TC-009 combined, proving
the full I-03 `DefectClassSweep` completeness invariant
(`matching_count = fixed_count + dispositioned_count`, `open_count = 0`,
`guard.status = verified`) that E34-F08 will consume:

- TC-005 (above) demonstrates the invariant holding on a successful closure
  (`open_count: 0`, `matching_count = fixed_count`, guard verified →
  `status: complete`).
- TC-008 (above) demonstrates `matching_count`/`open_count` are reported
  explicitly and correctly even in the degenerate zero-instance case.
- TC-009 (above) demonstrates `guard.status = verified` is gated on an
  actually-observed two-way counterfactual, not assumed or granted by
  guard presence alone.

Together these three reviewed scenarios prove the invariant end-to-end:
completeness of the instance count reconciliation (TC-005, TC-008) plus
integrity of the guard-verification gate (TC-005, TC-009) that together
define when `status: complete` may legitimately be reported. E34-F08's test
plan should reference this pointer
(`E34-F06-defect-class-completeness-and-recurrence-routing/scenario-review-TC-005-TC-009.md#tc-i-03-defect-class-closure-cross-reference`)
rather than writing a twin test, per test-plan.md's existing instruction.
