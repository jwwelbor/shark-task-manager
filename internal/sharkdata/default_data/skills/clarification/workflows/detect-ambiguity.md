---
inputs:
  - requirement_text: the raw requirement, request, or instruction to analyze
  - context: surrounding information (parent goal, domain, prior decisions) (optional)
  - constraints: known non-negotiables (optional)
outputs:
  - ambiguity_report: list of {span, ambiguity_type, interpretations, why_it_matters, severity}
  - overall_assessment: clear | clarify-recommended | clarify-required
  - recommended_next_workflow: question-ladder | surface-assumptions | none
---

# Workflow: Detect Ambiguity

**Purpose**: Locate the exact spans of a requirement that admit more than one reasonable interpretation, classify each, and rate its severity.
**Use for**: Triage — deciding whether clarification is needed and where to spend effort.
**Output**: An ambiguity report plus an overall assessment.

## Overview

Ambiguity detection is a scanning pass, not a discussion. You read the requirement against a fixed catalog of ambiguity types, and for each suspect span you write down the competing interpretations and what would go wrong if you picked the wrong one. The result tells you whether to stop (it's clear), ask questions (someone can answer), or document assumptions (no one can).

This is a reading discipline. Resist the urge to "just interpret it" — your job here is to find the forks, not to pick a path.

## Detection Process

### Phase 1: Read for the goal

Before tagging anything, answer one question: **what outcome is this requirement trying to produce?**

- If you can state the goal in one sentence with confidence, write it down — it becomes your yardstick.
- If you *cannot* state the goal, that is itself the highest-severity finding: an **intent ambiguity**. Everything else is downstream of it. Tag it and consider stopping the scan early, because resolving intent may dissolve the other ambiguities.

### Phase 2: Scan span-by-span against the taxonomy

Walk the requirement clause by clause. For each clause, check it against each ambiguity type from `../context/ambiguity-taxonomy.md`:

| Type | The question you ask of the span |
|------|----------------------------------|
| **Referential** | Does a pronoun or "the X" point to something unstated? ("send *it* to *them*") |
| **Term-definition** | Is a key term undefined or used loosely? ("*fast*", "*recent*", "*the admin*") |
| **Quantifier** | Is an amount, threshold, or boundary missing? ("*some*", "*most*", "*large*") |
| **Scope** | Is it unclear what is included vs excluded? ("handle errors" — which errors?) |
| **Conditional** | Is a rule stated without its else-branch or its trigger? ("if invalid, reject" — and if valid?) |
| **Completeness** | Does an "etc." / "and so on" hide unlisted cases? |
| **Intent** | Is the *why* missing, so success can't be judged? |

A single span can carry more than one type. Tag all that apply.

### Phase 3: For each tagged span, write the fork

A finding is not real until you can write the competing readings. For every tagged span, record:

1. **Span** — quote the exact words.
2. **Interpretations** — at least two concrete, plausible readings (A, B, …).
3. **Why it matters** — what diverges downstream if A vs B is chosen. If nothing diverges, *delete the finding* — it is not ambiguous in any way that matters.

> If you can only produce one plausible interpretation, it is not ambiguous to you. Note any residual doubt as low severity and move on.

### Phase 4: Rate severity

Score each finding using the severity rubric in `../context/clarification-criteria.md`. The short version:

- **High** — wrong guess produces a silently incorrect outcome that is expensive to discover or reverse.
- **Medium** — wrong guess causes noticeable rework but is caught before it does damage.
- **Low** — wrong guess fails fast and cheap, or both readings are nearly equivalent in practice.

Severity is about *consequence × detectability*, not about how confusing the wording feels.

### Phase 5: Produce the overall assessment

Roll the findings up into one verdict:

- **clear** — no findings, or only low-severity findings that are safe to ride as assumptions → recommend `none`.
- **clarify-recommended** — medium-severity findings present → recommend `question-ladder` if an answerer exists, else `surface-assumptions`.
- **clarify-required** — any high-severity finding, or an unresolved intent ambiguity → must clarify before proceeding. Recommend `question-ladder` if an answerer exists, else `surface-assumptions` with the high-severity items flagged as bets.

## Output Format

```markdown
# Ambiguity Report: {short requirement label}

**Stated goal (best read)**: {one sentence, or "UNCLEAR — see A1"}
**Overall assessment**: clear | clarify-recommended | clarify-required
**Recommended next**: question-ladder | surface-assumptions | none

## Findings

### A1 — {span quoted}
- **Type**: {referential | term-definition | quantifier | scope | conditional | completeness | intent}
- **Interpretations**:
  - A: {reading one}
  - B: {reading two}
- **Why it matters**: {what diverges downstream}
- **Severity**: high | medium | low

### A2 — {span quoted}
...

## Notes
- {residual low-severity doubts not worth a full finding}
```

## Worked Example

**Requirement**: "When a user uploads a large file, show a warning and process it in the background."

```markdown
### A1 — "a large file"
- Type: term-definition, quantifier
- Interpretations:
  - A: large = anything over a fixed size threshold (e.g., a documented limit)
  - B: large = relative to the user's typical uploads
- Why it matters: the threshold determines when the warning fires and when background processing kicks in; A and B trigger on completely different files.
- Severity: high

### A2 — "show a warning"
- Type: completeness, intent
- Interpretations:
  - A: informational only — proceed regardless
  - B: a confirmation the user must accept before processing continues
- Why it matters: B blocks the flow until the user acts; A does not. This changes the entire interaction model.
- Severity: high

### A3 — "process it in the background"
- Type: scope
- Interpretations:
  - A: only large files go to background; normal files stay inline
  - B: all uploads now go to background, large or not
- Why it matters: changes behavior for the common (non-large) case, which the requirement never mentions.
- Severity: medium
```

Assessment: **clarify-required** (two high-severity findings). Next: `question-ladder`.

## Success Criteria

Detection is complete when:
- [ ] The requirement's goal is stated, or its absence is recorded as an intent ambiguity
- [ ] Every clause has been scanned against the full taxonomy
- [ ] Each finding quotes an exact span and lists ≥2 concrete interpretations
- [ ] Findings with no downstream divergence have been deleted (not kept as noise)
- [ ] Each finding has a severity
- [ ] An overall assessment and recommended next workflow are produced

## Related

- `../context/ambiguity-taxonomy.md` — full type catalog with detection cues
- `../context/clarification-criteria.md` — severity rubric
- `question-ladder.md` — next step when an answerer is available
- `surface-assumptions.md` — next step when no answerer is available
