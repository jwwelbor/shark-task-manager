---
inputs:
  - impl_paths: list of paths to changed implementation source files
  - api_contract_paths: list of paths to API contract documents (T01-api-contracts.md or 04-backend-design.md)
  - data_model_paths: list of paths to data model documents (T03-data-models.md or 03-data-design.md)
  - flow_diagram_paths: list of paths to flow / system diagram documents (T06-system-flows.md or 02-architecture.md)
  - arch_review_path: absolute path where the architecture compliance review should be written
outputs:
  - arch_review_doc: structured markdown written to arch_review_path
  - deviations: list of {severity: blocking|should-fix|suggestion, description, location}
  - required_changes: list of changes required before merge (blocking deviations)
  - improvements: list of suggestions for future iterations
---

# Design Compliance Review Workflow (craft)

**When**: Implementation is complete and needs architectural review before merge.

## Process

### 1. Compare Implementation Against Specs

- API endpoints match contract definitions (paths, methods, schemas)
- Data models match schema design (fields, types, constraints, relationships)
- System flows match sequence diagrams (order of operations, error paths)

### 2. Check Architectural Compliance

- Correct layers and boundaries respected (no leaking between layers)
- Proper use of established patterns (repository, service, handler, etc.)
- No architectural shortcuts (direct DB access from handlers, business logic in controllers)

### 3. Verify Integration Points

- External APIs called correctly (auth, headers, error handling)
- Retry logic and circuit breakers where appropriate
- Timeout configuration for external calls

### 4. Apply Design Principles

- **Appropriate**: Right solution for the problem and constraints?
- **Proven**: Uses established patterns and technologies?
- **Simple**: No unnecessary complexity?

### 5. Document Findings

- List deviations from spec with severity (blocking, should-fix, suggestion)
- Identify required changes before merge
- Note improvements for future iterations

## Output

Write the compliance review to `arch_review_path` containing:

- Summary of comparison against contracts, data models, and flow diagrams
- Architectural compliance findings
- Integration point verification results
- Design principles assessment (appropriate / proven / simple)
- Enumerated deviations with severity (blocking, should-fix, suggestion)
- Required changes before merge
- Improvements for future iterations
