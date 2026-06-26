---
name: derive-standards
description: Derive organization-level standards from existing conventions, constraints, and recurring pain points.
inputs:
  - existing_conventions
  - target_scope
  - pain_points
  - constraints
outputs:
  - standards_document
  - open_questions
---

# Derive Standards

1. **Collect patterns.** Identify repeated naming, layout, documentation, review, and testing conventions in the target scope.
2. **Classify strength.**
   - Mandatory: divergence creates real cost, risk, or confusion.
   - Recommended: consistency helps, but exceptions are acceptable.
   - Local choice: multiple options are acceptable.
3. **Write rules as checks.** Phrase each rule so a reviewer can determine pass/fail from the artifact.
4. **Add examples.** Include one correct example and, where useful, one incorrect example.
5. **State exceptions.** Define when the standard can be bypassed and what justification is required.
6. **List open decisions.** Do not invent policy when ownership is unclear.

## Standards Template

```markdown
# {Standard Name}

## Scope
{Where this applies and where it does not}

## Mandatory Rules
1. {Rule} — {Rationale}

## Recommended Defaults
- {Preference} — {When to use}

## Examples
### Correct
{Example}

### Avoid
{Counterexample}

## Exceptions
{Allowed exceptions and required justification}

## Open Decisions
- {Decision needed, owner, impact}
```
