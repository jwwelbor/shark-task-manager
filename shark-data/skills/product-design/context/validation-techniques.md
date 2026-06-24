# Validation Techniques — Reference

A concise reference for the validation phase of `product-design` (D12–D14). Read on demand from the workflows that link here. Not a textbook — just the patterns the workflows assume you know.

## Test Methods at a Glance

| Method | Best for | Sample size | Caveat |
|---|---|---|---|
| Moderated remote (e.g., Zoom + screen share) | Rich qualitative, complex flows | 5–8 | Time-intensive; moderator bias |
| Moderated in-person | Lo-fi prototypes, contextual inquiry | 3–6 | Logistics-heavy; geographic skew |
| Unmoderated remote (UserTesting, Maze, etc.) | Quick directional reads, late-stage | 10–20 | No probing; surface-level findings |
| Hallway test | Fast feedback on a specific question | 3–5 | Convenience sample; weight cautiously |
| Live alpha / beta | Real-context behavior, retention | 20+ | Slowest signal; needs analytics |

**Rule of thumb:** 5 participants per persona surfaces ~80% of usability issues for that persona. If you have 3 personas, plan for ~15 sessions to get reasonable coverage. Fewer is fine, but reflect that in the confidence rating in D12.

## Severity Scoring Rubric (used in D13)

Use this exact scale; do not invent your own.

| Score | Label | Definition |
|---|---|---|
| 1 | Cosmetic / nice-to-have | Small annoyance; doesn't affect task outcome. Examples: typo, color contrast preference, button copy quibble. |
| 2 | Annoying but workable | Slows users down; they grumble but recover unaided. Examples: ambiguous label, extra click, confusing-but-skippable step. |
| 3 | Blocks task in some contexts | Some users (or some contexts) hit a wall; need help to recover. Examples: error states without next-step guidance, dead-end flows for edge personas. |
| 4 | Blocks task always; users abandon | Every tested user fails or gives up. Examples: broken navigation, missing critical action, fatal misalignment with mental model. |

**Calibration:** if more than half of your themes score `3` or `4`, the test wasn't iterative enough — designs needed another round before user testing. If everything scores `1` or `2`, the test was probably too leading or the scenarios were too easy.

## Frequency vs. Severity (the "fix-first" rule)

A theme's priority ≈ severity × frequency × persona-impact-weight.

- **Frequency**: count of participants who hit it. With small samples, treat 1-of-5 as a "single-source — keep watching" rather than a finding.
- **Persona-impact-weight**: a high-severity issue affecting your *primary* persona outranks a same-severity issue affecting a *tertiary* persona. The skill doesn't enforce a numeric weight — it surfaces the persona ID and lets the user decide.

## Per-Participant Observation — the minimum useful unit

For an observation to be reusable in D13, it should answer:

1. **Who:** participant ID + persona.
2. **What:** what they did, said, or struggled with — concretely. "Confused" is not enough; "couldn't find the export button; clicked Settings 3 times" is.
3. **When in the flow:** which task, which screen, which touchpoint.
4. **Maps to:** D11 friction-point ID, or `new`.

Anything that can't answer these four is anecdote, not evidence.

## Persona Evidence in D14

A design is "validated against persona X" when at least one of these is true:

1. At least one participant matching that persona completed the relevant task without help and reported neutral-or-better sentiment.
2. At least one participant matching that persona made it through with minor stumbles AND no theme at severity 3 or higher hit them.
3. The user explicitly waives validation for that persona with a documented rationale ("we'll validate post-launch via support tickets" / "this persona is out of scope this round").

Anything else means the design is *not yet* validated for that persona — push for `Needs rework` or honest re-test, not optimistic check-marks.

## Common Failure Modes (and what to do about them)

| Pattern | Likely cause | Fix |
|---|---|---|
| Themes from 2-of-2 sessions in a 3-persona test | Under-sampling | Run more sessions before D13 finalization, or downgrade confidence to Low. |
| Lots of feedback on visual style for a lo-fi test | Wrong fidelity question | Tag visual feedback as "out of scope for this round" and don't act on it. |
| Every design "validated" | Confirmation bias / soft test | Probe: which hypothesis from D12 §1 was actually at risk? If none, the test was theatre. |
| One participant dominates the themes | Outlier weighting | Note in D13 §5 (Confidence) and ensure no severity 3+ finding rests solely on that participant. |
| Personas drift between docs | D08 is stale | Pause and update D08; this is the implication-callout in D13 §4. |

## Severity → Action (in D14)

| Severity (D13) | Default action in D14 |
|---|---|
| 4 (always blocks) | Needs rework — a blocking issue that must close before the design is considered done |
| 3 (blocks some) | Needs rework or accepted residual risk (with mitigation) |
| 2 (annoying) | Validated with minor adjustments OR accepted |
| 1 (cosmetic) | Validated; defer fix to backlog |

These are defaults; the user can override with rationale. The skill's job is to surface the default and ask if a different choice is intentional. Outstanding "Needs rework" items are carried forward as open obligations against the design.

## Quick Glossary

- **Validated** — there is persona-anchored evidence the design solves the friction it was meant to solve.
- **Needs rework** — known fixes are required; design will return for re-test or re-approval.
- **Scrapped** — won't ship in any form; the friction will be addressed elsewhere or accepted.
- **Deferred** — fine as is, but not part of *this* initiative; revisit in a future round.
- **Accepted residual risk** — known issue, no fix planned for this initiative, monitored or mitigated post-launch.
