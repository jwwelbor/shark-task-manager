---
inputs:
  - finding: the point-instance finding that triggered this sweep — {severity, file_line, diagnosis, evidence}
  - touched_module_paths: list of module/package paths the finding lives in (defines the default search scope)
  - repair_record: prior fix/disposition history for this class, if any — {class_key, fingerprint, disposition, date} entries
  - decision_sources: paths to search for prior designs — feature/epic notes, tech-debt records, prior review-finding notes, spec/architecture sections, project standards docs (all optional; skip a source that doesn't exist for this project rather than failing)
  - calling_gate: which gate invoked this workflow — code_review | approval | uat_redteam | qa (drives which rubric re-runs during full-class re-verification)
outputs:
  - class_key: stable normalized class identity
  - class_statement: one-line general class, not the point instance
  - search_scope: array of concrete modules, patterns, contracts, and durable records searched
  - prior_designs: array of prior TD, DEC, Question, note, spec, or standard pointers considered
  - searched_count: number of candidate sites evaluated
  - matching_count: number of class instances found
  - instances: array of {fingerprint, site_pointer, disposition, evidence}
  - fixed_count: matching instances repaired
  - dispositioned_count: matching instances covered by cited decisions
  - open_count: matching instances still open
  - guard: {kind, implementation_pointer, counterfactual_pointer, status}
  - status: open | complete
---

# Workflow: Defect-Class Sweep

## Purpose

Generalize one finding into a defect class and sweep the whole affected
surface for sibling instances in a single pass, so a fix and a review round
close the class instead of one point instance. This is the canonical source
for the sweep procedure invoked from code review kickback handling, approval
rejection handling, and UAT/red-team re-verification — those callers
reference this file rather than restating the procedure.

Produces the I-03 `DefectClassSweep` record nested inside the calling gate's
`GateResult.remediation_sweeps` array. It does not itself write to an
entity, table, or lifecycle status — the parent gate owns persistence.

## When to invoke

- A code-review, QA, or approval gate is about to issue a kickback for a
  finding that plausibly recurs elsewhere in the touched surface.
- A UAT/red-team round is re-reviewing work that was previously rejected
  (every re-verification round runs the full three-part procedure below, not
  just a check of the cited fix).
- A development/rework pass is starting on a task carrying a rejection
  section or a kickback reason naming a defect class — enumerate the class
  before re-fixing the cited instance (see Enumeration procedure, below).
- A finding's disposition conflicts with a prior accepted decision on a
  matching fingerprint (severity-conflict routing, below).

## Class naming

Before searching, state the class at the general level, not the point
instance:

- `class_key`: a short, stable, normalized identifier for the class (e.g.
  `unvalidated-optional-field-dereference`, not `line-42-null-check`).
  Reuse an existing `class_key` from `repair_record` when the new finding is
  the same defect shape; mint a new one only when no prior key fits.
- `class_statement`: one sentence describing the general defect shape a
  reviewer could use to recognize other instances — "a schema's required
  list omits a field the code dereferences unconditionally," not "field `x`
  is missing on line 42."

A class statement that only restates the point instance fails this step —
rewrite it one level more general before proceeding.

## Search-scope declaration

Declare `search_scope` explicitly before enumerating, so the scope is
auditable and so a later instance inside that same scope can be classified as
recurrence (see Recurrence classification, below):

- Start from `touched_module_paths` and widen to every module, package, or
  template that shares the pattern the class statement describes (same
  language construct, same schema shape, same call pattern) — not just files
  literally touched by the current diff.
- Include durable records worth checking for prior treatment of this class:
  feature/epic decision notes, tech-debt records, prior review-finding notes,
  the relevant spec.md/architecture.md sections, and project standards docs
  from `decision_sources`. Skip any source that doesn't exist for this
  project; do not fabricate a path.
- Record what was actually searched in `search_scope` and `prior_designs` —
  a scope that isn't written down can't be re-used to classify recurrence
  later and can't be audited by the next reviewer.

## Enumeration procedure

Search the full declared scope in one pass — do not fix the first instance,
re-review, then search again for a second.

1. For each entry in `search_scope`, search for the pattern the class
   statement describes (grep/glob for the language construct or schema
   shape; read decision sources for text matches on the class statement).
2. Record every candidate site checked in `searched_count`, whether or not it
   matched.
3. For every site that matches the class, add one entry to `instances` (see
   Instance evidence, below) and increment `matching_count`.
4. For each matched instance, disposition it immediately as one of: fixed
   (repaired in this pass), dispositioned (already covered by a cited prior
   decision — cite the pointer), or open (matches but not yet resolved).
   `fixed_count + dispositioned_count + open_count = matching_count`.

Enumerate, don't iterate: report every matching instance found in this pass,
not the first one. A partial sweep that stops after the first fix produces
another rejection round when the next sibling instance surfaces.

## Zero-result reporting

A pass that finds no additional instances is a normal, reportable outcome —
never an omitted or empty result. Report `searched_count` (the real number of
candidates checked) and explicit `matching_count: 0`, `open_count: 0`. A
missing or blank instance list is indistinguishable from "the sweep was never
run" — always emit the counts, even when they are all zero.

## Instance evidence

Every entry in `instances` carries:

- `fingerprint`: a stable identity for this specific occurrence (e.g. a
  normalized `file:construct` key) — used to detect recurrence later.
- `site_pointer`: the file/line, doc section, or record where the instance
  was found.
- `disposition`: `fixed`, `dispositioned`, or `open` (see Enumeration
  procedure).
- `evidence`: the concrete proof for the disposition — the diff hunk for
  `fixed`, the decision pointer for `dispositioned`, or the reproduction
  detail for `open`.

An instance without a fingerprint or evidence pointer cannot be told apart
from a future recurrence — do not close a class on instances missing either
field.

## Backward-looking rework

Before designing a repair for any cited finding, search prior treatment of
this class rather than re-deciding it from scratch:

- Affected code and tests in the touched module (grep/glob against the
  pattern the class statement describes).
- Feature/epic decision notes recorded against the entity.
- Tech-debt records that may already track this class.
- Prior review-finding notes across entities (a project-wide search, or
  entity notes filtered to the review-finding note type).
- The relevant spec.md/architecture.md sections.
- Project standards docs, if present.

Preserve an existing disposition on a matching fingerprint unless new
evidence contradicts it — do not silently re-open or re-decide a
already-dispositioned instance without a stated reason.

## Recurrence classification

Classify each matched instance against `repair_record` and any
previously-`complete` class with an overlapping `search_scope`:

- **Recurrence**: the same fingerprint resurfaces after a recorded repair, or
  a new fingerprint belongs to the **same `class_key`** as a previously
  `status: complete` class **and** lies inside that class's recorded
  `search_scope`. Both conjuncts are required — matching `class_key` alone
  (outside the recorded scope) or scope membership alone (under a different
  `class_key`) is not recurrence.
- **Normal finding**: anything else — a new fingerprint under a different
  `class_key`, or outside every completed class's recorded scope, routes
  through ordinary rework, not recurrence handling. A different defect class
  that merely happens to live inside an old scope's file/module footprint is
  a normal finding, not a recurrence.

No round-count field or round-counting logic is used anywhere in this
classification — recurrence is decided by fingerprint, `class_key`, and scope
membership, never by "this is the Nth time we've seen a finding here."

## Disposition and severity-conflict routing

When a finding's disposition would conflict with a prior accepted decision on
a matching fingerprint — for example, fresh evidence materially raises the
risk of a previously-accepted low-severity finding — do not resolve the
conflict unilaterally inside this workflow:

- A bounded, single-owner disagreement about the right disposition routes
  through the `question-management` skill
  (`skills/question-management/SKILL.md`).
- Specialist disagreement, cross-entity inconsistency, high blast radius,
  irreversibility, or no safe evidence path routes through the project's
  multi-specialist council deliberation workflow.

`severity_conflict` is an outer `GateResult.Finding.disposition` value (see
architecture.md's I-02 `GateResult` schema), not an I-03 instance
disposition — the sweep's own `instances[].disposition` stays within
`{fixed, dispositioned, open}` so `fixed_count + dispositioned_count +
open_count = matching_count` keeps reconciling cleanly (see Enumeration
procedure, above). Record the conflicted instance as `open` inside
`instances` (pending the outer conflict's resolution, with `evidence`
pointing at the conflict), and have the calling gate add or update a
`Finding` in its own `GateResult.findings` array with `disposition:
severity_conflict` and a `disposition_pointer` back to this instance's
`fingerprint`/`site_pointer`. Block normal advancement at the `Finding` level
until the referenced mechanism resolves it — a conflict must never be routed
silently through ordinary rework, and must never be closed by marking the
I-03 instance itself `severity_conflict`.

## Guard selection

For a class to close, select an executable guard that would have caught this
class if it existed before the fix — a test, a lint/type rule, a schema
constraint, or an equivalent automated check discovered from the project's
own toolchain and standards inputs (never a hardcoded tool name; use the
guard commands already available to the calling gate's render context).
Prefer the narrowest guard that actually exercises the defect shape over a
broad rule that happens to also cover it.

## Structural guard closure

A class may only report `status: complete` when all of the following hold:

- Every instance in `instances` is `fixed` or `dispositioned` — `open_count`
  is exactly `0`.
- `matching_count = fixed_count + dispositioned_count`.
- The selected guard has passed a counterfactual verification: it catches
  (flags, fails the build, or otherwise blocks) the class when the defect is
  deliberately re-introduced, and it does not flag/fail when the defect is
  absent. A guard that misses the reintroduced defect, or that false-positives
  when the defect is absent, is not verified. Set `guard.status = verified`
  only after both directions of that counterfactual are actually observed —
  an unverified or merely-plausible guard does not count.

If no feasible guard exists for this class, the class stays `status: open`
and must carry a linked work item (a task, a tech-debt record, or a note)
tracking the gap — never report `complete` on the strength of "we fixed the
instances we found" alone when no guard exists to catch the next one.

## Full-class re-verification

Every re-review round after a kickback — from code review, approval, or
UAT/red-team — runs this same three-part procedure, invoked identically
regardless of which gate calls it:

1. **Verify the named fixes** — confirm each previously cited finding is
   actually resolved.
2. **Re-run the full enumeration** — re-search the entire declared
   `search_scope`, not just the site of the cited fix, using the Enumeration
   procedure above.
3. **Re-run the calling gate's full rubric** — the calling gate's ordinary
   verification checks over the whole feature surface, not only the
   previously-flagged area.

"Confirm the cited finding is fixed" is never the whole job on a re-review
round. A narrowly-scoped re-check that skips steps 2 and 3 lets a sibling
instance of the same class survive into the next round.

## Output shape

The completed sweep is reported as one I-03 `DefectClassSweep` entry, nested
inside the calling gate's `remediation_sweeps` array:

| Field | Value |
|---|---|
| `class_key` | from Class naming |
| `class_statement` | from Class naming |
| `search_scope` | from Search-scope declaration |
| `prior_designs` | pointers gathered during Search-scope declaration / Backward-looking rework |
| `searched_count` | from Enumeration procedure |
| `matching_count` | from Enumeration procedure |
| `instances` | from Instance evidence |
| `fixed_count` / `dispositioned_count` / `open_count` | from Enumeration procedure |
| `guard` | from Guard selection |
| `status` | `open` or `complete`, per Structural guard closure |

No new note type, table, or lifecycle status is introduced — this record
nests inside the existing `GateResult` envelope the calling gate already
produces.

## Self-verification (before returning)

- [ ] `class_statement` describes the general class, not the point instance.
- [ ] `search_scope` and `prior_designs` list what was actually searched.
- [ ] `searched_count`, `matching_count`, `fixed_count`, `dispositioned_count`,
      and `open_count` are all present and consistent, including on a
      zero-result pass.
- [ ] Every entry in `instances` has a `fingerprint`, `site_pointer`,
      `disposition`, and `evidence`.
- [ ] Recurrence classification used fingerprint, `class_key`, and scope
      membership only (both `class_key` match and scope membership required
      for a new fingerprint) — no round-count field appears anywhere.
- [ ] Any severity conflict is routed to `question-management` or the
      multi-specialist council deliberation workflow (not resolved
      unilaterally) and recorded as the outer `GateResult.Finding.disposition
      = severity_conflict` with a `disposition_pointer` — never as an I-03
      `instances[].disposition` value.
- [ ] The guard's counterfactual was verified in the correct direction: it
      catches the class when the defect is deliberately re-introduced, and
      does not flag when the defect is absent.
- [ ] `status: complete` only when `open_count = 0` and `guard.status =
      verified` by an observed counterfactual in both directions; otherwise
      `status: open` with a linked work item.
