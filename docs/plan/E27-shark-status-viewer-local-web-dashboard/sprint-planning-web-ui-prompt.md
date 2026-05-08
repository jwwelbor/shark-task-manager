# Sprint Planning Web UI Prompt

Use this prompt when thinking through how Sprint Planning should appear in the Shark web UI.

The goal is not to force a framework migration. The current Shark web app is a single-file, read-only viewer/dashboard served by `shark web`. Assume the implementation remains frameworkless unless the design team produces a strong justification otherwise.

---

## Prompt

You are designing the Sprint Planning experience for the Shark web UI.

Use the `frontend-design` skill and commit to a clear, intentional visual direction. The current Shark web app is a dark, IDE-style viewer. Do not default to a generic admin dashboard. This needs to feel like a serious planning surface for real work.

### Context

- Sprint Planning is now a major feature in the CLI and docs.
- The current web UI is a local viewer/dashboard for Shark project data.
- The web UI does not currently expose sprint planning as a dedicated surface.
- Sprint Planning includes:
  - planning view
  - readiness score
  - capacity by agent type
  - backlog inspection
  - bulk add / remove planning actions
  - velocity, burndown, and summary reporting
- The current web app is a single HTML/JS viewer, not a framework app.

### What To Think Through

Evaluate the best way to surface sprint planning in the web UI. Consider these options:

1. Add a separate Sprint tab or sprint dashboard, similar to the existing dashboard pattern.
2. Add a sprint tree or sprint navigation section on the left sidebar.
3. Do both: a sprint dashboard as the primary planning view, plus a sprint tree on the left for navigation.

### Key Questions

- Should this be read-only reporting, or an interactive planning surface?
- If it is interactive, what are the minimum usable interactions?
- How should a PM / Scrum Master use it day to day?
- How should an AI-orchestrator-style power user use it?
- How should a casual user inspect sprint state without getting overwhelmed?
- What should be visible by default?
- What should be tucked behind navigation, filters, or secondary views?
- How should the sprint dashboard relate to the existing entity dashboard?
- How should the sprint tree coexist with the existing entity hierarchy?

### Constraints

- Prefer a frameworkless implementation unless there is a compelling reason not to.
- Preserve the current Shark viewer character: dark, sharp, readable, practical.
- Avoid generic SaaS dashboard patterns.
- Make the result feel intentional and designed, not bolted on.
- Keep the interface usable on desktop and reasonable on smaller viewports.
- Assume this should work as a local-first app.

### Design Expectations

- Use a bold but coherent visual language.
- Make the navigation feel obvious without being noisy.
- Keep the planning surface usable, not just pretty.
- If you propose both a dashboard and a tree, explain exactly how they coexist.
- If you recommend an interactive surface, define the primary actions and the states clearly.
- If you recommend read-only first, explain what would be needed later to make it operational.

### Deliverables

Return:

- A recommendation with reasoning.
- A concise UX strategy.
- An information architecture / navigation model.
- A proposed screen or component structure.
- The key states and interactions.
- A rationale for why this is the right surface for the feature.
- A comparison of the candidate approaches if useful.
- A clear answer on whether the surface should be interactive or read-only.

### Desired Outcome

I want a design recommendation that answers:

- whether sprint planning belongs in the web UI
- whether the experience should be interactive
- whether to use a tab, a tree, or both
- how to keep the result cohesive with the existing viewer

Do not write implementation code unless the design recommendation explicitly requires it.

---

## Notes

- Existing CLI docs already cover the sprint feature set.
- The web UI prompt should focus on UX/CX and the viewer surface, not the backend service implementation.
- If you think a framework migration is necessary, justify it explicitly and compare it to staying frameworkless.

