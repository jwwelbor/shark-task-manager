# D13 — User Feedback

Produce the user feedback (`D13-user-feedback.md`) — themed, prioritized synthesis of D12 raw test results.

> **Read first:** `context/validation-techniques.md` — severity rubric, frequency-vs-severity rule, common failure modes.

**Builds on:** D12 test results (must contain at least one completed session). Also read D11 (friction points being tested) and D08 (persona IDs for evidence anchoring).

## Synthesis Steps

### 1. Read D12 fully

Collect all observations, errors, quotes, and emotional signals across all sessions.

### 2. Cluster into themes

A theme is a pattern that appears across multiple participants or has high intensity in one. Name themes descriptively ("Users can't recover from an accidental deletion" not "Error handling issue").

### 3. Score each theme

- **Severity:** Critical (blocks task) / High (causes workaround or abandonment) / Medium (friction) / Low (annoyance)
- **Frequency:** How many participants surfaced it
- **Persona spread:** Which D08 personas it affects

### 4. Map to D11 friction points

For each theme, state:
- Which D11 friction point it corresponds to (or "newly surfaced — not in D11")
- Whether the friction point was **confirmed**, **disconfirmed**, or **modified** by the test evidence

Closing this D11→D12→D13 loop is mandatory. D13 is a re-test of CX design assumptions.

### 5. Flag new friction points

If testing revealed friction that D11 didn't anticipate, surface it explicitly. It may become a D14 rework item.

## Quality Criteria

- [ ] Every theme anchored to specific D12 session IDs and participant quotes.
- [ ] All D11 friction points accounted for (confirmed / disconfirmed / modified / not tested).
- [ ] Themes scored and ranked.
- [ ] New friction points clearly flagged as "not in D11."

## Output Template

```markdown
# D13 — User Feedback

*Synthesized from: D12-test-results.md*
*Friction points under evaluation: D11 [FP IDs]*

## Themes (Priority Order)

### T01 — [Theme name]
- **Severity:** Critical / High / Medium / Low
- **Frequency:** [n of N participants]
- **Personas:** P1, P2
- **Description:** [What the pattern is — in user terms, not system terms]
- **Evidence:** "[Quote]" — P1, Session [date]; "[Quote]" — P3
- **D11 friction point:** FP-0X — [Confirmed / Disconfirmed / Modified: explanation]

### T02 — [Theme name]
[Same structure]

### T0N — [Theme name] (Newly surfaced — not in D11)
[Same structure — flag these prominently]

---

## D11 Friction Point Disposition

| D11 ID | Friction point | Disposition | Evidence |
|---|---|---|---|
| FP-01 | [Name] | Confirmed / Disconfirmed / Modified / Not tested | [T0X or "no sessions covered this"] |
| FP-02 | [Name] | [Same] | [Same] |

## Summary for D14

**Designs to validate:** [Which touchpoints or designs had sufficient evidence to render a verdict]
**Designs needing rework:** [Which had critical/high themes that must be addressed]
**Designs with insufficient data:** [Which need more testing before a verdict is possible]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D13-user-feedback.md` in the product-design directory.
