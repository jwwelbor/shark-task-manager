# D14 — Validated Designs

Produce the validated designs (`D14-validated-designs.md`) — a verdict per design under test, with persona evidence supporting each verdict.

> **Read first:** `context/validation-techniques.md` — persona-evidence rules, severity-to-action defaults, glossary of verdict terms.

**Builds on:** D13 user feedback. If D13 marks any designs "insufficient data," those need additional testing before a verdict — confirm with the user before rendering verdicts for them.

## Verdict Logic

For each design tested, render one of three verdicts:

- **Validated** — Evidence supports this design as sufficient for the use case. The primary persona could accomplish their goal. No critical/high themes blocking it.
- **Needs rework** — Evidence shows a specific problem that must be addressed. The design is not wrong overall but has a blocking issue. Rework is scoped and owned.
- **Scrapped** — Evidence shows the design fundamentally doesn't work for the primary persona. It is retired. An alternative path is identified (or explicitly deferred).

"Needs rework" requires: what changes, who owns it, by when, and what validates the fix.
"Scrapped" requires: why it failed, what the alternative is (or "no alternative — this feature is deferred").

No design may be marked "Validated" without at least one persona-anchored success observation from D12.

## Elicitation

Present the D13 summary to the human stakeholder and walk through each design together. Use `AskUserQuestion` to confirm each verdict — do not auto-assign.

> "Based on the test results, [Design X] had [summary]. I'd suggest the verdict is [Validated / Needs rework / Scrapped]. Do you agree, or do you see it differently?"

## Quality Criteria

- [ ] Every design under test has a verdict.
- [ ] No "Validated" verdict without at least one persona-anchored success observation.
- [ ] No "Needs rework" without: specific changes, owner, timeline, and validation criteria.
- [ ] No "Scrapped" without a reason and a deferred/alternative path.
- [ ] Human stakeholder confirmed each verdict via `AskUserQuestion`.

## Output Template

```markdown
# D14 — Validated Designs

*Verdicts based on: D13-user-feedback.md*

## Design Verdicts

### [Design name / Touchpoint ID]

**Verdict:** Validated / Needs rework / Scrapped

**Evidence (persona-anchored):**
- P1: [Specific observation from D12 session — task, outcome, quote]
- P2: [Same]

**Themes addressed:** [D13 theme IDs that this design handles well]
**Themes outstanding:** [D13 theme IDs still unresolved]

**If Needs rework:**
- Required changes: [Specific, not vague]
- Owner: [Name]
- Complete by: [Date]
- Validation: [How we'll know the rework is sufficient — re-test, expert review, etc.]

**If Scrapped:**
- Reason: [What fundamentally failed]
- Alternative / deferred path: [What replaces it, or "deferred — no alternative in scope"]

---

### [Next design]
[Same structure]

---

## Summary Table

| Design | Verdict | Primary blocker (if any) | Owner |
|---|---|---|---|
| [Name] | Validated | — | — |
| [Name] | Needs rework | [Theme] | [Name] |
| [Name] | Scrapped | [Reason] | — |

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D14-validated-designs.md` in the product-design directory.
