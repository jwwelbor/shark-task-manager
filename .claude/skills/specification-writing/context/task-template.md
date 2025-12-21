# Task (Product Requirement Prompt) Template Structure

This document defines the complete structure for Tasks (agent-executable implementation tasks).

## Task File Structure

```markdown
---
status: created
feature: /docs/plan/{epic-key}/{feature-key}
created: YYYY-MM-DD
assigned_agent: api-developer | frontend-developer | devops-engineer | general-purpose
dependencies: [list-of-task-files-that-must-complete-first.md]
estimated_time: X hours
---

# Task: {Component Name}

## Goal

{Single, clear objective (1-2 sentences) stating WHAT will be built and WHY it matters.}

## Success Criteria

- [ ] {Measurable checkpoint 1 - specific, testable outcome}
- [ ] {Measurable checkpoint 2}
- [ ] {Measurable checkpoint 3}
- [ ] All validation gates pass
- [ ] `IMPLEMENTATION.md` and `TODO.md` updated (if applicable)
- [ ] Any artifacts referenced in design docs created/updated

## Implementation Guidance

### Overview

{High-level description of what needs to be built. Keep this to 2-3 paragraphs maximum.}

### Codebase Analysis Results

{CRITICAL for API/Frontend Tasks - reference from design docs if available}

**If API specification exists**, reference the codebase analysis:
- **Existing Features Found**: See [API Spec - Codebase Analysis](../04-api-specification.md#codebase-analysis)
- **Decision**: {Extend existing vs create new, with rationale from design doc}
- **SRP Validation**: {Confirmation that approach doesn't violate Single Responsibility Principle}
- **NO PARALLEL PATHS**: {Explicit confirmation this doesn't duplicate existing functionality}

**If API specification is missing**, note in task:
- **Research Required**: Implementation agent must analyze existing codebase
- **Document Findings**: Agent should document codebase analysis results
- **Avoid Duplication**: Agent must verify no parallel functionality exists

### Key Requirements

{List 5-8 key requirements with references to available design docs - DO NOT duplicate content}

**Reference available design documents**:
- Requirement from PRD - See [PRD - Requirements](../prd.md#requirements)
- Requirement from Architecture (if exists) - See [Architecture - Section Name](../02-architecture.md#section)
- Requirement from Database (if exists) - See [Database Design - Table Specs](../03-database-design.md#table-specs)
- Requirement from API (if exists) - See [API Spec - Endpoint Details](../04-api-specification.md#endpoints)

**If design docs are missing**, state requirements directly from PRD and note:
- "Detailed {component} design not yet documented - implementation agent will need to make design decisions"
- "Agent should document design decisions made during implementation"

### Contract Specifications

{CRITICAL for API/Frontend Tasks - reference exact contract definitions IF AVAILABLE}

**If API specification exists**, reference the exact contract definitions:
- **DTOs to Implement**: See [API Spec - DTO Definitions](../04-api-specification.md#dto-definitions)
- **Field Names**: Must match EXACTLY as specified in API spec
- **Data Types**: Must match EXACTLY as specified in API spec
- **Validation Rules**: Must match EXACTLY as specified in API spec
- **Contract Sync Table**: See [API Spec - Contract Synchronization Table](../04-api-specification.md#contract-synchronization-table)
- **IMPORTANT**: Frontend and backend must implement IDENTICAL DTOs. Any deviation will cause integration failures.

**If API specification is missing**, note in task:
- **Contract Definition Required**: Implementation agent must define DTOs/interfaces
- **Documentation Requirement**: Agent must document contract definitions
- **Coordination Required**: If multiple agents involved, ensure contract synchronization
- **Recommendation**: Consider creating API specification before implementation

### Data Flow

{For API/Frontend Tasks - reference data flow documentation IF AVAILABLE}

**If API specification exists**, reference the data flow documentation:
- **Frontend Pre-Call**: See [API Spec - Frontend Pre-Call](../04-api-specification.md#{endpoint-name})
- **Backend Processing**: See [API Spec - Backend Processing](../04-api-specification.md#{endpoint-name})
- **Backend Pre-Response**: See [API Spec - Backend Pre-Response](../04-api-specification.md#{endpoint-name})
- **Frontend Post-Response**: See [API Spec - Frontend Post-Response](../04-api-specification.md#{endpoint-name})

**If API specification is missing**, provide high-level data flow from PRD/Architecture:
- Describe expected data flow based on available documentation
- Note that detailed data transformations need to be designed during implementation
- Recommend documenting actual data flow after implementation

### Files to Create/Modify

{List file paths that will need changes - don't provide the code}

**Backend** (example):
- `backend/services/{service-name}.py` - New service for {purpose}
- `backend/api/routes/{route-name}.py` - API endpoints
- `backend/models/{model-name}.py` - Data models
- `backend/dtos/{dto-name}.py` - DTO definitions matching API spec
- `backend/tests/test_{component}.py` - Tests for this component
- `backend/tests/test_{component}_contract.py` - Contract validation tests

**Frontend** (example):
- `frontend/src/components/{component-name}.tsx` - UI component
- `frontend/src/services/{service-name}.ts` - API service layer
- `frontend/src/types/api/{feature-key}.ts` - TypeScript interfaces matching backend
- `frontend/src/tests/{component}.test.tsx` - Component tests

**Database** (example):
- `migrations/{timestamp}_create_{table}.sql` - Database migration
- `migrations/{timestamp}_add_rls_{table}.sql` - Row-level security policies

### Integration Points

{Describe how this component integrates with other systems}

- **Existing systems/features**: {Reference codebase analysis from design docs if available}
- **Other components being built**: {What other Tasks does this interact with}
- **External APIs or services**: {Any third-party integrations}

**If architecture document exists**: Reference [Architecture - Integration Points](../02-architecture.md#integration-points)

**If architecture is missing**: Describe integration points from PRD and note that implementation agent should document actual integration approach taken.

## Validation Gates

{Describe WHAT needs to be validated - not test code}

**Linting & Type Checking**:
- Code passes all linting checks (ruff, mypy for Python; eslint for TypeScript)
- No type errors in strict mode

**Unit Tests**:
- Coverage for {specific behaviors - reference design doc}
- All edge cases from design doc tested

**Contract Tests** (CRITICAL for API/Frontend):
- DTO structure matches API specification EXACTLY
- Field names match specification
- Data types match specification
- Validation rules match specification
- Reference: [API Spec - Contract Testing Requirements](../04-api-specification.md#contract-testing-requirements)

**Integration Tests**:
- Validation of {specific workflows - reference design doc}
- End-to-end scenarios work as expected

**Manual Testing**:
- Verify {specific scenarios - reference design doc}

**Performance**:
- Meets targets specified in [Security & Performance](../06-security-performance.md#performance-targets)

## Context & Resources

{Provide direct links to available design doc sections}

**Always available**:
- **PRD**: [Feature Requirements](../prd.md)

**If design documents exist, link to relevant sections**:
- **Architecture** (if exists): [System Architecture](../02-architecture.md#system-architecture)
- **Database** (if exists): [Table Specifications](../03-database-design.md#table-specifications)
- **API** (if exists): [Endpoint Details](../04-api-specification.md#api-endpoints)
- **API Contracts** (if exists): [DTO Definitions](../04-api-specification.md#dto-definitions) - **MUST MATCH EXACTLY**
- **Contract Sync** (if exists): [Contract Synchronization Table](../04-api-specification.md#contract-synchronization-table)
- **Frontend** (if exists): [Component Design](../05-frontend-design.md#components)
- **Security** (if exists): [Security Measures](../06-security-performance.md#security-measures)

**If design documents are missing**, note:
- Implementation will be guided primarily by PRD requirements
- Agent should create design documentation as part of implementation
- Document design decisions for future reference

## Notes for Agent

{Brief notes (3-5 bullet points) about:}

- Patterns to follow from existing codebase (reference codebase analysis)
- **CONTRACT CRITICAL**: DTOs must match API spec EXACTLY - field names, types, validation
- Edge cases requiring special attention
- Performance considerations from design doc
- Security requirements to prioritize
- **NO PARALLEL PATHS**: Do not duplicate functionality identified in codebase analysis
```

## Frontmatter Fields

**status**: `todo` | `in_progress` | `blocked` | `ready_for_review` | `completed` | `archived`

Note: Status is managed via the `shark` CLI and tracked in database. Task files remain in feature directory regardless of status.

**feature**: Path to feature directory containing design docs

**created**: Date in YYYY-MM-DD format

**assigned_agent**: Appropriate specialized agent
- `general-purpose` - General implementation work
- `api-developer` - Backend API development
- `frontend-developer` - Frontend UI development
- `devops-engineer` - Infrastructure and deployment
- `backend-architect` - Backend architecture decisions
- `frontend-architect` - Frontend architecture decisions
- `db-admin` - Database schema and migrations

**dependencies**: List of Task files that must complete before this one can start

**estimated_time**: Realistic effort estimate in hours (typically 2-12 hours)

## Quality Standards

Each Task must:

1. **Focused Scope**: Single, clear objective that can be completed independently
2. **High-Level**: Tells WHAT to build, not HOW to code it
3. **Reference-Rich**: Links to design docs instead of duplicating content
4. **Testable**: Clear success criteria and validation gates
5. **Concise**: 50-100 lines maximum (not counting frontmatter)
6. **Agent-Appropriate**: Assigned to the right specialized agent
7. **Dependency-Aware**: Clearly states what must complete first
8. **Time-Bounded**: Realistic effort estimate

## What NOT to Include in Tasks

❌ **NEVER include**:
- SQL statements, DDL, migrations, or database queries
- Python, TypeScript, JavaScript, or any programming language code
- Bash scripts, shell commands, or CLI instructions
- Configuration files (YAML, JSON, TOML, etc.)
- Step-by-step code implementation instructions
- Line-by-line coding tutorials
- Detailed implementation procedures

✅ **ONLY include**:
- Clear goal and success criteria
- WHAT needs to be built (high-level requirements)
- WHY it's needed (business/technical rationale)
- References to design doc sections for details
- List of files that will need changes
- Integration points and dependencies
- Validation requirements (what to test, not how)
- Edge cases and performance considerations

## Task Index File

Create `tasks/README.md` to index all Tasks:

```markdown
# Implementation Tasks

## Overview
This folder contains agent-executable tasks that implement the {feature-key} feature in phases.

## Active Tasks

| Task | Status | Assigned Agent | Dependencies | Estimated Time |
|-----|--------|----------------|--------------|----------------|
| [T-E##-F##-001](./T-E##-F##-001.md) | todo | api-developer, frontend-developer | None | 1 hour |
| [T-E##-F##-002](./T-E##-F##-002.md) | todo | general-purpose | T-E##-F##-001 | 4 hours |
| [T-E##-F##-003](./T-E##-F##-003.md) | todo | api-developer | T-E##-F##-001, T-E##-F##-002 | 8 hours |
| [T-E##-F##-004](./T-E##-F##-004.md) | todo | frontend-developer | T-E##-F##-001, T-E##-F##-003 | 8 hours |
| [T-E##-F##-005](./T-E##-F##-005.md) | todo | general-purpose | T-E##-F##-004 | 6 hours |

## Workflow

### Execution Order
1. **Task 001**: Contract Validation (1 hour) - MUST complete first
2. **Task 002**: Database Setup (4 hours) - Depends on 001
3. **Task 003**: API Implementation (8 hours) - Depends on 001, 002
4. **Task 004**: Frontend Development (8 hours) - Depends on 001, 003
5. **Task 005**: Integration Testing (6 hours) - Depends on 004

## Status Definitions

Status is tracked in database via `shark` CLI. Use these commands:
- `shark task list --status=todo` - Show todo tasks
- `shark task start <key>` - Start a task (status: in_progress)
- `shark task complete <key>` - Complete a task (status: ready_for_review)
- `shark task approve <key>` - Approve a task (status: completed)
- `shark task block <key> --reason="..."` - Block a task
- `shark task unblock <key>` - Unblock a task

Task files remain in `/docs/plan/{epic}/{feature}/tasks/` regardless of status.

## Design Documentation
All tasks reference these design documents in /docs/plan/{epic-key}/{feature-key}/:
- [Architecture](../../plan/{epic-key}/{feature-key}/02-architecture.md)
- [Database Design](../../plan/{epic-key}/{feature-key}/03-database-design.md)
- [API Specification](../../plan/{epic-key}/{feature-key}/04-api-specification.md)
- [Frontend Design](../../plan/{epic-key}/{feature-key}/05-frontend-design.md)
- [Security & Performance](../../plan/{epic-key}/{feature-key}/06-security-performance.md)
```
