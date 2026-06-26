# Test Matrix Template

Use this template for the `test_matrix` output.

```markdown
# Test Matrix

## Requirement
{requirement summary}

## Acceptance Criteria
| ID | Criterion | Source Span | Notes |
|---|---|---|---|
| AC-001 | {criterion} | {source} | {notes} |

## Conditions
| ID | Criterion | Class | Preconditions | Input / Action | Expected Outcome |
|---|---|---|---|---|---|
| TC-001 | AC-001 | positive | {precondition} | {input} | {outcome} |
| TC-002 | AC-001 | edge | {precondition} | {input} | {outcome} |
| TC-003 | AC-001 | negative | {precondition} | {input} | {outcome} |

## Open Questions
- {question}
```

## Case Classes

- **Positive:** expected, valid usage.
- **Edge:** boundary, minimum, maximum, empty, duplicate, ordering, or concurrency condition.
- **Negative:** invalid, unauthorized, malformed, missing, conflicting, or must-not-happen condition.
