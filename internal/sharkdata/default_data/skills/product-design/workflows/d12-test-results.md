# D12 — Test Results

Produce the test results (`D12-test-results.md`) — raw output of user-testing sessions, structured for synthesis in D13.

> **Read first:** `context/validation-techniques.md` — test methods, per-participant observation minimums.

**Builds on:** D08 personas, D09 journey maps, D10 touchpoints, D11 friction points — these define who is tested, against what, and which friction points are under test.

## What This Captures

D12 records **what happened in user-testing sessions**. It does not interpret or synthesize — that is D13's job. The reader should be able to reconstruct any session from D12 alone.

**The human is the source of truth here.** This skill captures and structures; it does not invent participants, fabricate observations, or generate quotes.

## Setup: Test Protocol

Before capturing results, confirm with the user:

1. "What design(s) or prototype(s) were tested?" (Reference D10 touchpoint IDs or a prototype if available)
2. "What tasks were participants asked to complete?"
3. "How many participants? What's their mapping to D08 personas?"
4. "What were the test conditions — moderated/unmoderated, remote/in-person, think-aloud?"

Use `AskUserQuestion` for each. Document the protocol in the output before recording results.

## Per-Session Capture

For each participant, capture:

- **ID:** P1, P2, P3… (never real names without explicit consent)
- **Persona mapping:** Which D08 persona this participant represents
- **Tasks attempted:** Which tasks from the protocol
- **Completion:** Completed / Partial / Failed per task (and time if recorded)
- **Errors:** Specific mistakes made (what they did, what they expected)
- **Observations:** Verbatim quotes or close paraphrases (label synthesized quotes as such)
- **Probes:** What the facilitator asked and what the participant answered
- **Emotional signals:** Frustration, hesitation, delight — with the triggering moment

Capture sessions one at a time. Ask the user to share notes for each.

## Record material Questions

When a test-protocol or evidence-routing decision remains materially
unresolved, use `skills/question-management/SKILL.md` to create or reuse a
linked Q###. Record a non-material rationale in the test-results artifact
instead. Do not treat the absence of `TBD` text as decision closure.

## Quality Criteria

- [ ] All participants are assigned stable IDs (P1, P2…).
- [ ] Each participant is mapped to a D08 persona.
- [ ] Completion rates captured per task.
- [ ] Quotes labeled as verbatim or synthesized.
- [ ] D11 friction points referenced by ID where they were tested.

## Output Template

```markdown
# D12 — Test Results

*Testing designs from: D10-touchpoints.md (and a prototype if applicable)*
*Friction points under test: [D11 FP IDs]*

## Test Protocol

- **What was tested:** [Designs, prototypes, or flows]
- **Tasks:** [List]
- **Participants:** [n total], mapped to personas: [P1 x n, P2 x n]
- **Method:** Moderated / Unmoderated | Remote / In-person | Think-aloud: Yes/No
- **Date range:** [When sessions were conducted]

---

## Session: Participant P1 ([D08 Persona name])

### Task 1: [Task name]
- **Completion:** Completed / Partial / Failed
- **Time:** [if recorded]
- **Path taken:** [What they actually did]
- **Errors:** [Specific mistakes]
- **Quote:** "[Verbatim or synthesized* — *synthesized from session notes]"
- **Emotional signals:** [Frustration at X, delight at Y]

### Task 2: [Task name]
[Same structure]

### Overall observations for P1
[Anything notable about this participant's session as a whole]

---

## Session: Participant P2 ([D08 Persona name])
[Same structure]

---

## Quantitative Summary

| Task | P1 | P2 | P3 | Completion rate |
|---|---|---|---|---|
| Task 1 | completed | failed | completed | 67% |
| Task 2 | completed | completed | failed | 67% |

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D12-test-results.md` in the product-design directory.
