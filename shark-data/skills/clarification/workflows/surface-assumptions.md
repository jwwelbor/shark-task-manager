---
name: surface-assumptions
description: Make silent assumptions explicit when no answerer is available or immediate clarification is impractical.
inputs:
  - requirement_text
  - context
  - constraints
  - prior_clarifications
outputs:
  - assumption_register
  - clarified_requirement
---

# Surface Assumptions

1. **Read for hidden dependencies.** Identify where the text requires a choice that is not stated.
2. **Write each assumption plainly.** State the interpretation you would use if forced to proceed.
3. **Rate confidence.** Use high, medium, or low based on available evidence.
4. **Rate blast radius.** Describe what breaks or changes if the assumption is wrong.
5. **Define validation.** Name the fastest way to confirm or reject the assumption.
6. **Rewrite with assumptions embedded.** Produce a clarified requirement that clearly marks assumed points.

## Assumption Register Template

```markdown
| Assumption | Confidence | Blast Radius | Evidence | Validation Method |
|---|---|---|---|---|
| {assumption} | {high/medium/low} | {impact} | {evidence} | {method} |
```
