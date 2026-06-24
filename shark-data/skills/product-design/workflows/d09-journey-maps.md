# D09 — Journey Maps

Produce the journey maps (`D09-journey-maps.md`) — end-to-end maps of how each primary persona moves through the product experience, from first awareness to goal completion.

> **Read first:** `context/design-patterns.md` — interaction and flow patterns referenced when mapping stages.

**Builds on:** D08 user personas. Also read D07 user needs — these define what each journey stage must accomplish.

## Scope Decision

Before mapping, decide how many journeys to map:

- Map one journey per **primary persona** (P1 always).
- Map secondary persona journeys only where their path meaningfully diverges from P1's.
- Do not map a journey for every persona if their paths are 80% identical — note the divergences inline instead.

Ask the user: "Should we map journeys for all personas, or just the primary (P1)?"

## Journey Structure

A journey map has a **start point**, an **end point**, and **stages** in between. Choose stages that reflect the actual experience, not your product's navigation.

**Common stage patterns:**
- Awareness → Consideration → Onboarding → First use → Repeated use → Advocacy
- Trigger → Research → Try → Commit → Use → Stuck → Resolve
- Recognize problem → Find solution → Evaluate → Adopt → Succeed

Adapt to the specific initiative. Aim for 4–7 stages — more than 7 means your stages are too granular.

## For Each Stage

Document:
- **What the persona is doing** — their actions in this stage
- **What they're thinking** — the question or concern driving them
- **Emotional state** — how they feel (use a simple scale: positive / neutral / frustrated / blocked)
- **Touchpoints** — which channels, screens, or interactions they use
- **What needs to be true** — what the product/service must deliver for them to advance to the next stage

## Quality Criteria

- [ ] Every stage anchored to a D08 persona ID.
- [ ] Touchpoints are real channels/screens, not aspirational features.
- [ ] At least one emotional low point identified per journey (if none, the research is incomplete).
- [ ] Each stage's "what needs to be true" traces to a D07 need statement.
- [ ] Secondary persona journeys only mapped where they genuinely diverge.

## Output Template

```markdown
# D09 — Journey Maps

*Personas from: D08-user-personas.md*
*Needs from: D07-user-needs.md*

## Journey: P1 — [Persona Name]

**Journey scope:** [Start point] to [End point]
**Primary need:** [D07 need ID and statement]

### Stage 1: [Stage Name]

| Dimension | Content |
|---|---|
| **Actions** | [What P1 is doing] |
| **Thoughts** | "[The question/concern driving them]" |
| **Emotional state** | Positive / Neutral / Frustrated / Blocked |
| **Touchpoints** | [Channels, screens, interactions] |
| **Must-be-true** | [What the product must deliver to advance P1] — traces to [D07 need ID] |

### Stage 2: [Stage Name]
[Same structure]

### Emotional Arc (P1)

```
+ |        *          *
  |       / \        /
0 |  *   /   \    *
  | / \ /     \  /
- |/   *       \/
  +--Stage1--Stage2--Stage3...
```

**Peak moment:** [Stage where experience is best — why]
**Valley moment:** [Stage where experience is worst — why]

---

## Journey: P2 — [Persona Name] (divergences from P1 only)

**Diverges at:** Stage [X] — [what's different and why]

[Document only the stages that differ meaningfully]

---
Version: 1.0 — YYYY-MM-DD — author: [name]
```

Save as `D09-journey-maps.md` in the product-design directory.
