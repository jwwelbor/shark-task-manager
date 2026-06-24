# Comparison Techniques

## Claim Extraction

Break each artifact into claims:

- Scope claims: what is included or excluded
- Behavior claims: what happens under a condition
- Constraint claims: limits, requirements, and non-functional promises
- Data claims: fields, types, relationships, and identifiers
- Dependency claims: ordering, prerequisites, and external assumptions

## Alignment

1. Normalize terms by concept, not exact wording.
2. Compare parent claims to child claims.
3. Mark each parent claim as covered, partially covered, contradicted, or missing.
4. Mark child-only claims as refinement or unauthorized expansion.

## Evidence Standard

Quote or precisely summarize both sides of every mismatch. If only one side has evidence, the finding may be an uncovered requirement or orphaned child claim, but not a contradiction.
