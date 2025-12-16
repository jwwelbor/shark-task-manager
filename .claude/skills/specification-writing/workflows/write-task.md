# Workflow: Write Tasks

## Purpose

Generate agent-executable implementation tasks from existing technical design documentation. Tasks break down implementation into logical phases and components that specialized agents can execute independently.

## Core Responsibility

You will read comprehensive technical design documentation and create focused, agent-executable tasks. Your tasks must be **high-level directives that reference the design documents**, NOT detailed code tutorials.

## CRITICAL: You Create High-Level Directives, Not Code Tutorials

Think of yourself as a project manager creating work tickets for specialized teams. Each task should tell an agent WHAT to build and WHY, while referencing the detailed HOW from the design documents.

### NEVER WRITE IN TASKS:
- ❌ SQL statements, DDL, migrations, or database queries
- ❌ Python, TypeScript, JavaScript, or any programming language code
- ❌ Bash scripts, shell commands, or CLI instructions
- ❌ Configuration files (YAML, JSON, TOML, etc.)
- ❌ Step-by-step code implementation instructions
- ❌ Line-by-line coding tutorials
- ❌ Detailed implementation procedures

### ONLY WRITE IN TASKS:
- ✅ Clear goal and success criteria
- ✅ WHAT needs to be built (high-level requirements)
- ✅ WHY it's needed (business/technical rationale)
- ✅ References to design doc sections for details
- ✅ List of files that will need changes
- ✅ Integration points and dependencies
- ✅ Validation requirements (what to test, not how)
- ✅ Edge cases and performance considerations

**KEY PRINCIPLE**: Tasks are executive summaries with references, not implementation manuals.

## Input Requirements

Before starting, you must have access to:
1. **Feature directory path**: `/docs/plan/{epic-key}/{feature-key}/`
2. **Design documents** in that directory:
   - `02-architecture.md` - System architecture and integration
   - `03-database-design.md` - Database schema and data model
   - `04-api-specification.md` - API endpoints and contracts
   - `05-frontend-design.md` - UI components and state
   - `06-security-performance.md` - Security and performance requirements
   - `07-implementation-phases.md` - Implementation phases and timeline

## Output Location

All tasks are created in the feature's tasks directory:
- **`/docs/plan/{epic-key}/{feature-key}/tasks/`** - Tasks live alongside feature documentation

Task status is tracked in the database (via `pm` CLI), not by folder location. Tasks remain in their feature directory throughout their lifecycle.

## Your Process

### Step 1: Analyze Design Documents

Read and understand all design documents:
1. **Architecture** - Understand the system layers and integration points
2. **Database** - Identify tables, relationships, and data requirements
3. **API** - Map out endpoints, contracts, and business logic
4. **Frontend** - Understand component hierarchy and state management
5. **Security/Performance** - Note critical security and performance requirements
6. **Implementation Phases** - Understand the planned phase breakdown

### Step 2: Validate Contract Consistency (CRITICAL)

Before creating PRPs, verify the design documents have synchronized contracts:

1. **Check API Specification Document** (`04-api-specification.md`):
   - Verify "Codebase Analysis" section exists and is complete
   - Confirm DTOs are fully defined with exact field names and types
   - Verify "Contract Synchronization Table" shows matching frontend/backend expectations
   - Check that contract testing requirements are specified for both sides

2. **Cross-Reference with Frontend/Backend Docs**:
   - Frontend design (`05-frontend-design.md`) should reference the same DTO names
   - Backend services should use the same DTO names
   - Data transformations should be documented on both sides

3. **Flag Missing Information**:
   - If contracts are incomplete or mismatched, note this in the task
   - If codebase analysis is missing, warn that parallel code paths might be created
   - If DTOs aren't synchronized, highlight this risk in the task

### Step 3: Determine Task Scope

Create separate tasks for logical components based on the implementation phases. **CRITICAL: Always start with Contract Validation task**.

Typical structure:

**Task 001: Contract Validation** (`T-E##-F##-001.md`) ⭐ **ALWAYS FIRST**
- **Blocks**: All implementation tasks depend on this passing
- Backend: DTO implementation matching spec exactly
- Frontend: TypeScript interface implementation matching backend
- Contract validation tests passing on both sides
- Cross-team synchronization confirmed

**Task 002: Database Setup** (`T-E##-F##-002.md`)
- Schema creation
- Migrations and rollback strategy
- RLS policies, indexes and constraints
- Data seeding (if needed)
- **Depends on**: T-E##-F##-001

**Task 003: API Implementation** (`T-E##-F##-003.md`)
- Backend services and business logic
- API endpoints and middleware
- Request/response handling with validated DTOs
- Error handling and API contract tests
- **Depends on**: T-E##-F##-001, T-E##-F##-002

**Task 004: Frontend Development** (`T-E##-F##-004.md`)
- UI components and state management
- API integration with validated DTOs
- Form handling and validation
- Component tests
- **Depends on**: T-E##-F##-001, T-E##-F##-003

**Task 005: Integration & Testing** (`T-E##-F##-005.md`)
- End-to-end integration
- Cross-component testing
- Performance and security validation
- **Depends on**: T-E##-F##-004

**Task 006: Deployment & Monitoring** (`T-E##-F##-006.md`)
- Deployment configuration
- Monitoring setup and alerting rules
- Production validation
- **Depends on**: T-E##-F##-005

**Adjust the task structure based on feature complexity and implementation phases outlined in design docs.**

### Step 4: Create Each Task

For detailed task structure, see `../context/task-template.md`.

Key elements:
- Frontmatter (status, feature path, dependencies, agent assignment, time estimate)
- Goal (single, clear objective)
- Success Criteria (measurable checkpoints)
- Implementation Guidance (references to design docs)
- Validation Gates (what to test)
- Context & Resources (links to design doc sections)
- Notes for Agent (patterns, edge cases, considerations)

### Step 5: Create Task Index

Create `/docs/plan/{epic-key}/{feature-key}/tasks/README.md` to index all created tasks with:
- Overview of the feature
- Active tasks table (with agent, dependencies, time)
- Execution order and workflow
- Links to design documentation
- Note: Task status is tracked in database via `pm` CLI

## Task Quality Standards

Each task must meet these criteria:

1. **Focused Scope**: Single, clear objective that can be completed independently
2. **High-Level**: Tells WHAT to build, not HOW to code it
3. **Reference-Rich**: Links to design docs instead of duplicating content
4. **Testable**: Clear success criteria and validation gates
5. **Concise**: 50-100 lines maximum (not counting frontmatter)
6. **Agent-Appropriate**: Assigned to the right specialized agent
7. **Dependency-Aware**: Clearly states what must complete first
8. **Time-Bounded**: Realistic effort estimate (2-12 hours typically)

## Common Mistakes to Avoid

❌ **Writing implementation tutorials**
- Tasks are not step-by-step coding guides
- Reference design docs instead of duplicating details

❌ **Including code samples**
- No SQL, Python, TypeScript, or any language code
- Only exception: Design docs may have pseudocode for concepts

❌ **Being too prescriptive**
- Trust the implementation agent's expertise
- Provide requirements and constraints, not micro-instructions

❌ **Duplicating design doc content**
- Use references and links
- Design docs have the details; tasks coordinate execution

❌ **Creating overlapping tasks**
- Each task should have distinct, non-overlapping scope
- Clear handoff points between tasks

## When You Need Clarification

If design documents are incomplete or unclear:
1. Identify what's missing or ambiguous
2. Make reasonable assumptions based on best practices
3. Document assumptions in the task's "Notes for Agent" section
4. Recommend updating the design docs with the clarification

## Final Output

When complete:
1. **Confirm all tasks created** in `/docs/plan/{epic-key}/{feature-key}/tasks/` folder
2. **Verify task index** in `/docs/plan/{epic-key}/{feature-key}/tasks/README.md` is complete
3. **Validate dependencies** form a logical execution sequence
4. **Check agent assignments** are appropriate for each component
5. **Verify contract synchronization** (CRITICAL):
   - Confirm API task references exact DTO specifications from design doc
   - Confirm Frontend task references exact same DTO specifications
   - Confirm contract validation tests are included in both tasks
   - Verify codebase analysis results are referenced in tasks
6. **Summarize** for the user:
   - Number of tasks created
   - Estimated total time
   - Execution order
   - Contract synchronization status (confirmed/needs attention)
   - Codebase analysis results (new code vs extending existing)
   - Next steps (tasks ready for development, managed via `pm` CLI)

Your tasks are the execution plan that transforms design documentation into working code. They must be clear, focused, and actionable while trusting implementation agents to apply their expertise.
