---
name: implementation
description: Systematic approaches to writing high-quality code with contract-first discipline and validation gates
when_to_use: when implementing any backend, frontend, database, or test code
version: 1.0.0
inputs:
  - task_spec_path: absolute path to the task spec markdown
  - feature_prd_path: absolute path to the feature PRD markdown (optional)
  - architecture_path: absolute path to the architecture document (optional)
  - api_spec_path: absolute path to the API specification (optional)
  - coding_standards_path: absolute path to coding standards doc (optional)
  - test_plan_path: absolute path to a pre-authored QA test plan, if one exists (optional)
  - existing_files: list of paths to existing source files relevant to the change
  - codebase_kind: greenfield | brownfield (defaults to brownfield if existing_files non-empty)
outputs:
  - selected_workflow: one of {implement-api, implement-backend, implement-database, implement-frontend, implement-tests}
  - implementation_files: list of paths to created/modified source files
  - test_files: list of paths to created/modified test files
  - validation_gate_results: structured record of validation-gate outcomes
  - summary: structured markdown describing what was implemented
---

# Implementation Skill

## Overview

This skill provides systematic workflows for implementing code across backend APIs, frontend components, databases, and tests. It emphasizes:

1. **Contract-first discipline** — DTOs and interfaces defined and validated before business logic
2. **Validation gates** — Quality checkpoints before considering work complete
3. **Coding standards** — Consistent patterns and best practices
4. **Error handling** — Systematic approach to errors and edge cases
5. **Testing requirements** — Comprehensive test coverage expectations

## When to Use This Skill

**Use this skill when:**
- Implementing backend API endpoints and services
- Building frontend components and UI features
- Creating database schemas and migrations
- Writing tests for any code
- Following a task implementation blueprint

## Workflow Selection

Choose the appropriate workflow based on what you're implementing:

### Backend Implementation

**API Endpoints** → `workflows/implement-api.md`
- REST/GraphQL endpoint implementation
- Request/response handling
- Authentication and authorization
- API testing

**Backend Services** → `workflows/implement-backend.md`
- Business logic implementation
- Service layer patterns
- Repository patterns
- Backend utilities

**Database Changes** → `workflows/implement-database.md`
- Schema design and migrations
- Database queries and optimization
- Data modeling patterns

### Frontend Implementation

**UI Components** → `workflows/implement-frontend.md`
- Component implementation
- State management
- API integration
- Frontend testing

### Testing

**Test Creation** → `workflows/implement-tests.md`
- Unit test implementation
- Integration test implementation
- Contract test implementation
- E2E test implementation

A complex change frequently uses multiple workflows in sequence — for example, `implement-database` → `implement-backend` → `implement-api` → `implement-frontend` → `implement-tests`.

## Critical Context

Before implementing, always review these context documents (when present):

### Brownfield Discovery (Existing Codebases)

If `codebase_kind = brownfield` (or `existing_files` is non-empty), complete the brownfield discovery workflow BEFORE selecting an implementation workflow. See `context/brownfield-discovery.md`.

The point: understand what exists, avoid duplicating capabilities, and map the blast radius of your change before writing code.

### QA Test Plan (Shift-Left)

If `test_plan_path` is provided:

1. **Read it first** — it defines acceptance test cases with concrete inputs/outputs
2. **Read the parent feature PRD** — understand the feature-level intent
3. **Write tests that validate the test plan scenarios** before implementing
4. The test plan is your source of truth for "what correct looks like"

This prevents self-grading — where you write both tests and code, validating only your own interpretation of the spec.

### Contract-First Discipline

`context/contract-first.md`

**THE GOLDEN RULE:** DTOs are defined in design docs, implemented identically on frontend and backend, and validated BEFORE any business logic is written.

This prevents frontend/backend divergence and enables parallel development.

### Validation Gates

`context/validation-gates.md`

Quality checkpoints that must pass before work is considered complete:
- Linting and formatting
- Type checking
- Unit tests
- Contract tests
- Integration tests

### Coding Standards

`context/coding-standards.md`

Consistent code style, naming conventions, and best practices.

### Error Handling

`context/error-handling.md`

Systematic approach to error handling, logging, and user-facing error messages.

### Testing Requirements

`context/testing-requirements.md`

Test coverage expectations and testing strategies.

## Implementation Process

The standard implementation sequence:

```
1. Read task and parent context
   - task_spec_path
   - feature_prd_path (intent)
   - architecture_path / api_spec_path (constraints, contracts)
   ↓
2. BROWNFIELD DISCOVERY (if existing codebase)
   - Follow context/brownfield-discovery.md
   - Pass implementation gate before proceeding
   - Skip only for greenfield work
   ↓
3. CHECK FOR QA TEST PLAN (Shift-Left)
   - If test_plan_path provided: read test plan + parent feature PRD
   - Test scenarios come from the test plan, not your interpretation
   ↓
4. INVOKE TDD DISCIPLINE
   - Use the test-driven-development skill via the Skill tool
   - This is non-negotiable for production code (features, bug fixes,
     refactoring) — exceptions: configuration, documentation, or
     explicit user request
   ↓
5. Select appropriate workflow (this SKILL.md)
   ↓
6. Follow contract-first discipline (if applicable)
   ↓
7. Write tests FIRST (following TDD)
   - If test plan exists: translate the planned cases into automated tests
   - If no test plan: write tests from task spec and acceptance criteria
   ↓
8. Implement code following the selected workflow
   ↓
9. Apply coding standards
   ↓
10. Implement error handling
   ↓
11. Run validation gates
   ↓
12. Return structured summary of work completed
```

## Key Principles

1. **Contracts First** — Define and validate DTOs/interfaces before logic
2. **Test-Driven** — Write tests first
3. **Incremental** — Implement in small, testable increments
4. **Validated** — Pass all gates before considering work complete
5. **Documented** — Capture implementation notes and TODOs in the `summary`

## Anti-Patterns to Avoid

- Implementing business logic before validating contracts
- Skipping validation gates "to save time"
- Diverging DTO/interface names from specification
- Writing code without tests
- Assuming the frontend will adapt to backend naming
- Creating database migrations before contracts are validated
- Manual testing only (no automated tests)

## Integration with Other Skills

**test-driven-development** (REQUIRED)
- **Invoke this skill FIRST** before any production code via the Skill tool
- Follow RED-GREEN-REFACTOR cycle
- Exceptions: configuration files, documentation, or explicit user request

**testing-anti-patterns**
- Consult before writing tests
- Avoid testing mock behavior
- Don't add test-only methods to production

**frontend-design**
- Reference for UI aesthetic guidance
- Complement implement-frontend workflow

**architecture**
- Read architecture docs first
- Implementation follows design decisions

## Success Criteria

Implementation is complete when:
- All contracts validated and synchronized
- All validation gates pass
- Tests provide adequate coverage
- Code follows standards
- Error handling is comprehensive
- Implementation summary is structured and complete
- No outstanding TODOs in critical paths

---

**Remember:** Quality is built in through systematic workflows, not added later. Follow the discipline, trust the process, and deliver production-ready code.
