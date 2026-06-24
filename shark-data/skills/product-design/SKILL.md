---
name: product-design
description: Product-level design methodology covering D01–D14 — vision, discovery, UX/CX design, and concept validation. Produces the "what and why" before any feature work begins. Provides the methodology for each of the 14 product-design artifacts; any single artifact workflow can be applied independently. Triggers include "product vision", "discovery", "user personas", "journey mapping", "user testing", "friction points", and any "produce D0X" request.
version: 1.0.0
---

# Product Design Skill

This skill holds the **methodology for product-level design** — from first idea to validated concept. It covers the 14 product-design artifacts (D01–D14): how to produce each one well, what makes each good or bad, and the reusable frameworks behind them.

Each artifact has a dedicated workflow describing the procedure for producing it. Each workflow is self-contained methodology and can be applied on its own. The frameworks the workflows lean on live in `context/`.

**Announce at start:** "I'm using the Product Design skill."

## The 14 Artifacts

The artifacts span four design concerns. Each produces exactly one D-numbered document.

### Vision
| Artifact | Workflow | What it captures |
|---|---|---|
| D01 — Vision Statement | `workflows/d01-vision.md` | The problem, target user, solution direction, scope, and bets |
| D02 — Success Criteria | `workflows/d02-success-criteria.md` | Falsifiable, measurable definition of success |

### Discovery
| Artifact | Workflow | What it captures |
|---|---|---|
| D03 — Market Research | `workflows/d03-market-research.md` | Competitive landscape, market size, positioning gap |
| D04 — Feasibility Report | `workflows/d04-feasibility.md` | Technical, operational, and business feasibility |
| D05 — Stakeholder Insights | `workflows/d05-stakeholder-insights.md` | Synthesized user/stakeholder research themes |

### UX Design — User Research Synthesis
| Artifact | Workflow | What it captures |
|---|---|---|
| D06 — User Insights | `workflows/d06-user-insights.md` | Behavioral clusters and cross-cutting patterns |
| D07 — User Needs | `workflows/d07-user-needs.md` | Solution-agnostic, prioritized need statements |
| D08 — User Personas | `workflows/d08-user-personas.md` | Behaviorally-grounded persona archetypes |

### CX Design — Journey Strategy
| Artifact | Workflow | What it captures |
|---|---|---|
| D09 — Journey Maps | `workflows/d09-journey-maps.md` | End-to-end persona journeys with emotional arcs |
| D10 — Touchpoints | `workflows/d10-touchpoints.md` | Catalog of interaction points, critical ones flagged |
| D11 — Friction Points | `workflows/d11-friction-points.md` | Prioritized inventory of where the experience breaks down |

### Validation
| Artifact | Workflow | What it captures |
|---|---|---|
| D12 — Test Results | `workflows/d12-test-results.md` | Raw, structured output of user-testing sessions |
| D13 — User Feedback | `workflows/d13-user-feedback.md` | Themed, prioritized synthesis of test results |
| D14 — Validated Designs | `workflows/d14-validated-designs.md` | A verdict per design, with persona-anchored evidence |

## Context Reference Files

These hold the frameworks and worked examples the workflows reference but don't duplicate. Read the relevant one before applying the corresponding workflow.

| File | Frameworks it holds | Relevant to |
|---|---|---|
| `context/vision-anatomy.md` | 9-component vision anatomy, anti-patterns, worked example | D01, D02 |
| `context/measurement-frameworks.md` | Metric schema, leading/lagging, SMART/OKR/HEART/AARRR, guardrails | D02 |
| `context/research-techniques.md` | JTBD, Five Whys, competitive analysis, market sizing, survey design | D03, D05 |
| `context/feasibility-frameworks.md` | Tech/ops/business feasibility matrices, ROI, POC criteria, decision records | D04 |
| `context/design-patterns.md` | UX/UI pattern library, accessibility, design systems | D06–D11 |
| `context/validation-techniques.md` | Test methods, severity rubric, persona-evidence rules, failure modes | D12–D14 |

## How These Artifacts Build On Each Other

Each artifact draws on evidence captured in earlier ones — these are **evidence dependencies**, not a required running order:

- D02 success criteria each trace to a D01 vision element.
- D03–D05 scope their research against the D01 target user and problem.
- D06 synthesizes D03 + D05 into behavioral clusters.
- D07 needs derive from D06 clusters.
- D08 personas are built from D06 clusters and D07 needs.
- D09 journeys are mapped per D08 persona, anchored to D07 needs.
- D10 touchpoints are extracted from D09 stages.
- D11 friction points reference D10 touchpoint IDs and D09 emotional valleys.
- D12 testing exercises the D11 friction points against D08 personas.
- D13 synthesizes D12 and reconciles each D11 friction point.
- D14 renders verdicts from D13 with D08 persona-anchored evidence.

When applying a workflow, read the artifacts it builds on so your output stays anchored to that evidence. Each workflow's "Builds on" note lists what it expects to read.

## Core Principles (Apply Across All Workflows)

1. **Elicit, don't invent.** Vision (D01–D02) and validation (D12–D14) are human decisions. Use `AskUserQuestion`. Never fabricate a user quote, metric baseline, participant, or approval.
2. **Persona-anchored evidence.** From D08 onward, every observation, decision, and verdict ties to a named persona ID.
3. **One question at a time.** Vision and validation work rewards depth. Prefer multiple choice when the answer space is small; open-ended when it isn't.
4. **Mark unknowns explicitly.** Write `TBD — <owner> by <date>` — never guess.
5. **Trace every claim to a source.** A user statement, a session observation, a cited study, or a metric reading. No orphan assertions.

## Output Standards (All D-Artifacts)

1. Each artifact uses its exact D-numbered filename (e.g., `D01-vision-statement.md`), stored together in a single product-design directory (conventionally `docs/product/`).
2. Every claim traces to a source: a user statement, a session observation, a cited study, or a metric reading.
3. Unknowns marked `TBD — <owner> by <date>`.
4. Cross-reference the artifacts an output builds on (D02 cites D01 sections; D13 cites D12 session IDs; etc.).
5. Footer: `Version: 1.0 — YYYY-MM-DD — author: <name>`.

## Output Standards (Feature-Level Artifacts)

When product-design evidence feeds feature-level design:

1. Reference D08 persona IDs and D09 journey stage names verbatim.
2. Include the rationale for design decisions, not just the decisions.

## Tools Used

- **`AskUserQuestion`** — All elicitation, approvals, and human decisions.
- **`Read`** — Read the artifacts a workflow builds on, and the relevant `context/` file, before producing output.
- **`Write`** — Create the artifact.
- **`WebSearch`** — Market research (D03), technical research (D04).
