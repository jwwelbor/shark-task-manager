# Code Review Finding Criteria

Do not assign per-file scores or derive findings from rubric thresholds. A review
finding is admissible only when it identifies an affected user, caller,
behavior, invariant, documented standard, or maintenance operation; states a
concrete consequence; and cites evidence.

## Severity

- **Blocker**: evidenced acceptance-criteria violation, security flaw, broken
  production contract, missing standards-required behavior, or material
  architecture/workflow risk.
- **Non-blocker**: an actionable, evidence-backed issue with a concrete effect
  that does not violate a hard contract. Route it through the triage decision
  tree.
- Do not emit a third style/opinion severity. Omit a preference that has no
  evidenced effect on a user, behavior, invariant, documented standard, or
  maintenance operation.

## Withdrawn blockers

If a previously reported blocker is refuted as having no such impact, preserve
the original concern, the refuting evidence, and the report-only disposition.
That disposition creates no source edit and no tracker write. A substantive
finding that remains after review follows normal triage.

## Preserve review evidence

Keep candidate identity, complete scope, raw-finding correspondence, reviewer
identity, and requested/effective model metadata. Do not infer effective
identity from the requested route. Preserve applicable existing cost or token
accounting metadata unchanged. Historical review packets remain readable under
their original contract; new packets identify contract version 2 and require an
impact field on each retained finding.

Substantive standards citations, risk hotspots, caller chains, test evidence,
and incomplete-coverage failures remain part of review output. Do not treat an
absent standards citation as permission to create an opinion-only finding.
