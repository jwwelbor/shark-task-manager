# Prototype (Feature-Level)

Produce `docs/plan/<epic-id>/<feature-id>/prototype.md`: an interaction
prototype specification for validating user flows and interaction design before
build.

This is a feature-level artifact. It runs after wireframes when interaction
testing is needed.

## Prerequisites

Required:

- `wireframes.md` for this feature.
- The same feature metadata used for wireframes: feature key, feature title,
  feature directory, and source PRD path.
- D08 persona and D09 journey references carried forward from the wireframes.

Use the workflow-provided feature directory to locate `wireframes.md`. If the
feature directory or wireframes path is unavailable, ask the host for the
feature key and `wireframes.md` path before continuing.

## When to Prototype

Create `prototype.md` when:

- The feature involves a multi-step flow that must feel coherent end-to-end.
- The feature contains a novel interaction pattern users have not seen before.
- Stakeholder approval requires seeing flow behavior, not just screens.
- The transition between states carries important meaning.

Skip prototyping when wireframes are sufficient to write tasks and start
development.

If the need is unclear, ask the host or user whether a prototype is required or
whether `wireframes.md` is sufficient to proceed.

## Prototype Scope

A prototype is not a complete build. Scope it to:

- The primary happy path.
- One or two highest-risk interactions from the wireframes.
- The most important error or recovery path, if it affects user trust.

Do not prototype every edge case.

From `wireframes.md`, identify:

- The primary user flow.
- Any interaction that is hard to communicate in static wireframes.
- Any flow where transition, timing, animation, or progressive disclosure
  changes user understanding.

## Format Options

### Option A: Markdown Prototype Specification

Use this when the team can implement confidently from prose plus wireframes.

For each screen-to-screen transition:

```text
From: [Screen / State]
Trigger: [User action - click, scroll, input, etc.]
Animation/transition: [Instant / Fade / Slide from right / etc.]
To: [Screen / State]
Data carried: [Any data passed between screens]
Conditions: [Branching conditions, if any]
```

For each significant single-screen interaction:

```text
Element: [Button / Form / List item / etc.]
Action: [User action]
Immediate feedback: [What the UI shows within 100ms]
Async feedback: [What appears after server response]
Error path: [If it fails, what the UI shows]
```

### Option B: Interactive HTML Companion

Use this when stakeholder review or user testing requires a live clickable
prototype.

Create `prototype.md` as the canonical artifact and include:

- The prototype scope.
- A link to the companion HTML file if one is created.
- The flow and interaction notes needed by implementers.

When producing the HTML companion, use the `frontend-design` skill with:

- `wireframes.md` as the structural reference.
- The D08 primary persona as the design target.
- Scope limited to the happy path and identified high-risk interactions.

## Quality Gate Before Saving

- [ ] Scope is limited to the primary flow and highest-risk interactions.
- [ ] All screen transitions include trigger, transition, destination, and
      carried data.
- [ ] Critical error paths include user-visible recovery behavior.
- [ ] Primary persona uses a D08 persona ID.
- [ ] Journey stage uses a D09 stage name.
- [ ] Any HTML companion is referenced from `prototype.md`.

## Output Template

```markdown
# Prototype: [Feature Name]

*Feature: [Feature key] - [PRD feature name]*
*Based on: wireframes.md*
*Primary persona: [D08 ID]*
*Journey stage: [D09 stage]*
*Scope: Happy path + [specific high-risk interactions]*

## Prototype Format

[Markdown specification only / Markdown specification plus HTML companion at path]

## Flows

### Primary Flow: [Goal]

#### Step 1: [Screen 1] -> [Screen 2]

- **Trigger:** [User action]
- **Transition:** [Instant / Fade / Slide]
- **Data carried:** [What persists]
- **Conditions:** [Branching rules, if any]

#### Step 2: [Screen 2] -> [Screen 3]

[Same structure]

### High-Risk Interaction: [Name]

#### [Element] on [Screen]

- **Action:** [User action]
- **Immediate feedback (< 100ms):** [UI response]
- **Async feedback:** [After server response]
- **Error:** [UI if request fails]

## Error Recovery

[Most important recovery path and why it matters]

## Interaction Notes

[Non-obvious behavior, animation intent, timing, or UX rationale that should inform implementation]

---

Version: 1.0 - YYYY-MM-DD - author: [name]
```

Save to `docs/plan/<epic-id>/<feature-id>/prototype.md`.
