# D04 — Feasibility Report

Produce the feasibility report (`D04-feasibility-report.md`) by assessing technical, operational, and business feasibility of the proposed initiative.

> **Read first:** `context/feasibility-frameworks.md` — risk assessment matrices, feasibility checklists.

**Builds on:** D01 vision statement (the § Constraints section is the hard-limit list). Also read D03 market research if available — market risks inform feasibility risks.

## Assessment Dimensions

Work through each dimension. For each risk, rate it: **High / Medium / Low** likelihood × **High / Medium / Low** impact.

### 1. Technical Feasibility

- Does the required technology exist and is it mature enough?
- Are there known integration constraints (APIs, platforms, standards)?
- What are the performance or scalability requirements, and can they be met?
- Are there security or privacy constraints (data residency, encryption requirements)?
- What are the key technical unknowns — things we'd need to prototype or research to de-risk?

### 2. Operational Feasibility

- Does the team have the skills to build this? If not, what gaps exist?
- What tooling, infrastructure, or platforms are needed?
- What is the realistic timeline given constraints from D01 § Constraints?
- What dependencies exist on external teams, vendors, or systems?

### 3. Business Feasibility

- Is the cost of building within acceptable range given the market opportunity (D03)?
- Does the initiative comply with known regulatory requirements?
- What are the go-to-market dependencies (sales, support, legal, security review)?

### 4. Constraint Compliance

List each constraint from D01 § Constraints. For each: **Within limits / At risk / Violates**.

If any constraint is at risk or violated, surface it explicitly — do not bury it.

### 5. Recommendation

State clearly: **Feasible as-described / Feasible with changes / Not feasible**.

If "with changes": name the specific changes required and who decides on them.
If "not feasible": name the blocking constraint and what would need to change for it to become feasible.

## Quality Criteria

- [ ] Every risk is specific — not "technical risks exist" but "the OCR library does not support handwritten forms."
- [ ] All D01 constraints are addressed (none skipped).
- [ ] At least one unknown is identified with a de-risking action (prototype, spike, vendor evaluation).
- [ ] Recommendation is a clear one of three options, not hedged ambiguously.

## Output Template

```markdown
# D04 — Feasibility Report

*Scoped from: D01-vision-statement.md, D03-market-research.md (if available)*

## Technical Feasibility

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| [Specific risk] | H/M/L | H/M/L | [What to do] |

**Key unknowns requiring spikes:**
- [Unknown] — owner: [name], de-risk by: [date]

## Operational Feasibility

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| [Specific risk] | H/M/L | H/M/L | [What to do] |

**Skill gaps:** [list, or "none identified"]
**Timeline assessment:** [on track / at risk] — [reason]

## Business Feasibility

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| [Specific risk] | H/M/L | H/M/L | [What to do] |

## Constraint Compliance

| Constraint (from D01) | Status | Notes |
|---|---|---|
| [Constraint] | Within limits / At risk / Violates | [explanation] |

## Recommendation

**Verdict:** Feasible as-described / Feasible with changes / Not feasible

**Rationale:** [One paragraph]

**Required changes (if applicable):**
- [Change] — decision owner: [name]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D04-feasibility-report.md` in the product-design directory.
