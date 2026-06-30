---
name: human-checkpoint
description: Represents points requiring real human intervention. Invoke for user testing, final approvals, or decisions requiring human judgment.
---

# Human Checkpoint

This is a **checkpoint**, not a traditional agent. When the workflow reaches it, the automated flow pauses and waits for a real human decision, input, or validation before proceeding.

## When This Triggers

Use a human checkpoint when a decision carries significant business impact, depends on subjective judgment, involves ethics or compliance, or requires human observation — for example:

- **User testing** — observing real users with prototypes or features
- **Stakeholder validation** — business sign-off before development
- **Human approval** — go/no-go before opening a pull request
- **Production deployment approval** — go/no-go for a production release
- **Rollback decision** — revert vs. hotfix during a bad deployment

Don't add checkpoints for routine, low-risk, high-frequency, or purely technical decisions with clear criteria — they just create bottlenecks.

## How It Works

1. **Pause** the workflow.
2. **Present context** — what decision is needed, what information bears on it, the options, and the implications and risks of each.
3. **Gather input** through the host's normal question mechanism; allow custom answers beyond the presented options, and capture the rationale.
4. **Document** the decision, its rationale, and the resulting next step.
5. **Resume** the workflow on the chosen route.

Present a recommendation when you have one, but the human makes the call. Keep the framing in business terms for stakeholder decisions and surface time-sensitivity explicitly.

## Judgment for the Critical Checkpoints

- **Production deployment** is the highest-stakes checkpoint: weigh failure impact, rollback complexity, business timing, and team availability to monitor. Prefer low-traffic windows with a tested rollback ready.
- **Rollback** is the most time-sensitive: assess how many users are affected, whether data is at risk, and how quickly a fix is possible. **Default to rollback** unless impact is minimal, the fix is trivial and well-understood, or rolling back would cause more harm (e.g., a risky data migration).

## What Requires a Human

Information gathering, running checks, summarizing, and routing can all be automated. Reserve the human for what can't be: user-experience quality, business-value and risk-tolerance judgments, ethical considerations, strategic direction, and crisis-response prioritization.
