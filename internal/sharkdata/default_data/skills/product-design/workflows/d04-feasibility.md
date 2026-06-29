# D04 — Feasibility Report

Produce the feasibility report (`D04-feasibility-report.md`) by assessing technical, operational, and business feasibility of the proposed initiative.

> **Read first:** `context/feasibility-frameworks.md` — risk assessment matrices, feasibility checklists.

**Builds on:** D01 vision statement (the § Constraints section is the hard-limit list). Also read D03 market research if available — market risks inform feasibility risks.

**Reads the real stack.** Before assessing, read `docs/architecture/tech-stack.md` and `docs/architecture/integration-map.md` when they exist (bootstrap writes them), plus the bootstrap marker `docs/architecture/bootstrap.md` for the **track**. Assess the proposal against the recorded stack, not in the abstract. The track decides how you treat that stack:

- **Brownfield** — the stack is a *fixed* constraint. A gap is a risk to mitigate or a constraint that is violated; you cannot wish it away.
- **Greenfield** — the stack is a *provisional proposal*. You may challenge it; a gap becomes a required change that feeds back into `tech-stack.md` (see § Recommendation).
- If no architecture docs or marker exist, assess in the abstract as before.

## Assessment Dimensions

Work through each dimension. For each risk, rate it: **High / Medium / Low** likelihood × **High / Medium / Low** impact.

### 1. Technical Feasibility

Check the proposal against the **recorded stack** from `docs/architecture/` (above), not in the abstract:

- Does the required technology exist and is it mature enough? *(If the stack is fixed/brownfield, does the recorded stack already provide it, or is a new dependency implied?)*
- Are there known integration constraints (APIs, platforms, standards)? *(Cross-check `integration-map.md` for the inbound surface, data stores, and external services already committed.)*
- What are the performance or scalability requirements, and can the recorded stack meet them?
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

List each constraint from D01 § Constraints — and, when the architecture docs exist, treat the **recorded tech stack and integrations as additional hard constraints** (brownfield) or as the proposal under test (greenfield). For each: **Within limits / At risk / Violates**.

If any constraint is at risk or violated, surface it explicitly — do not bury it.

### 5. Recommendation

State clearly: **Feasible as-described / Feasible with changes / Not feasible**.

If "with changes": name the specific changes required and who decides on them.
If "not feasible": name the blocking constraint and what would need to change for it to become feasible.

**Return a stack-feedback signal** when the verdict is "feasible with changes" or "not feasible" — so the host can act on it (the gap must not be silently absorbed). Record it in the report's *Stack feedback* block (below) as {verdict, gap, driver, recommended route}; D04 does **not** perform the action itself — the `/shark product-design` host reads this and acts:

- **Greenfield** (provisional stack): recommended route = re-run bootstrap in **reconcile mode**, so `greenfield-scaffold.md` revises `tech-stack.md` against this report (its Phase 3.5).
- **Brownfield** (fixed stack): the stack cannot be rewritten, so the gap is *tracked*, not absorbed. Recommended route = file a tech-debt entry (or a constraint note on the affected entity) capturing the required change and its driver.

## Quality Criteria

- [ ] Every risk is specific — not "technical risks exist" but "the OCR library does not support handwritten forms."
- [ ] All D01 constraints are addressed (none skipped).
- [ ] When `docs/architecture/` exists, the assessment references the recorded stack/integrations rather than reasoning in the abstract.
- [ ] At least one unknown is identified with a de-risking action (prototype, spike, vendor evaluation).
- [ ] Recommendation is a clear one of three options, not hedged ambiguously.
- [ ] A "with changes" / "not feasible" verdict names its stack-feedback route (greenfield reconcile, or brownfield tech-debt / constraint note).

## Output Template

```markdown
# D04 — Feasibility Report

*Scoped from: D01-vision-statement.md, D03-market-research.md (if available), docs/architecture/tech-stack.md + integration-map.md (if present)*
*Track: brownfield (stack fixed) | greenfield (stack provisional) | unknown (no marker)*

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

**Stack feedback (if "with changes" / "not feasible"):**
- Greenfield → re-run `/shark project bootstrap` in reconcile mode to revise tech-stack.md
- Brownfield → tech-debt entry / constraint note: [key or summary]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D04-feasibility-report.md` in the product-design directory.
