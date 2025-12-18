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
2. **Minimum required**: `prd.md` - Product Requirements Document
3. **Recommended design documents** (tasks will be more detailed with these):
   - `02-architecture.md` - System architecture and integration
   - `03-database-design.md` or `03-data-design.md` - Database schema and data model
   - `04-api-specification.md` or `04-backend-design.md` - API endpoints and contracts
   - `05-frontend-design.md` - UI components and state
   - `06-security-performance.md` or `06-security-design.md` - Security requirements
   - `07-performance-design.md` - Performance requirements
   - `08-implementation-phases.md` - Implementation phases and timeline

**Note**: The workflow adapts to available documentation. More design docs = more detailed tasks.

## Output Location

All tasks are created in the feature's tasks directory:
- **`/docs/plan/{epic-key}/{feature-key}/tasks/`** - Tasks live alongside feature documentation

Task status is tracked in the database (via `shark` CLI), not by folder location. Tasks remain in their feature directory throughout their lifecycle.

## Your Process

### Step 0: Detect Available Documentation (FIRST STEP)

Before analyzing documents, detect what's available and inform the user:

1. **Check for required PRD**:
   ```bash
   ls /docs/plan/{epic-key}/{feature-key}/prd.md
   ```
   If missing, STOP and inform user that PRD is required.

2. **Detect available design documents**:
   Check for these files (use Glob or ls):
   - `02-architecture.md`
   - `03-database-design.md` or `03-data-design.md`
   - `04-api-specification.md` or `04-backend-design.md`
   - `05-frontend-design.md`
   - `06-security-performance.md` or `06-security-design.md`
   - `07-performance-design.md`
   - `08-implementation-phases.md`
   - `09-test-criteria.md`

3. **Present summary to user**:
   ```
   Documentation Analysis for {feature-key}:

   ✅ Available Documents:
   - prd.md
   - 02-architecture.md
   - 06-security-design.md
   - 07-performance-design.md
   - 08-implementation-phases.md
   - 09-test-criteria.md

   ❌ Missing Documents:
   - 03-database-design.md / 03-data-design.md
   - 04-api-specification.md / 04-backend-design.md
   - 05-frontend-design.md

   Task Detail Level: MEDIUM
   - Can generate high-level architectural tasks
   - Can generate security and performance tasks
   - Cannot generate detailed database implementation tasks
   - Cannot generate API contract tasks
   - Cannot generate frontend component tasks

   Recommendation:
   - If this is infrastructure/DevOps work → PROCEED (no frontend/API needed)
   - If this is full-stack feature → CONSIDER completing design docs first
   - If you want to proceed anyway → Tasks will be high-level planning tasks

   Continue with available documentation? (yes/no)
   ```

4. **Wait for user confirmation** before proceeding.

5. **Adjust task generation strategy** based on available docs:
   - **PRD only** → High-level planning tasks, research tasks, design tasks
   - **PRD + Architecture** → Architecture implementation tasks, integration tasks
   - **PRD + Architecture + Database** → Add database schema tasks
   - **PRD + Architecture + API** → Add backend service tasks
   - **PRD + Architecture + Frontend** → Add UI component tasks
   - **Full docs** → Comprehensive implementation tasks

### Step 1: Analyze Available Design Documents

Read and understand all AVAILABLE design documents (skip missing ones):
1. **PRD** (REQUIRED) - Understand the feature requirements and goals
2. **Architecture** (if present) - Understand the system layers and integration points
3. **Database** (if present) - Identify tables, relationships, and data requirements
4. **API** (if present) - Map out endpoints, contracts, and business logic
5. **Frontend** (if present) - Understand component hierarchy and state management
6. **Security/Performance** (if present) - Note critical security and performance requirements
7. **Implementation Phases** (if present) - Understand the planned phase breakdown

### Step 2: Validate Contract Consistency (CONDITIONAL)

**ONLY perform this step if API and Frontend design documents exist.**

If both `04-api-specification.md` (or `04-backend-design.md`) AND `05-frontend-design.md` are present:

1. **Check API Specification Document**:
   - Verify "Codebase Analysis" section exists and is complete
   - Confirm DTOs are fully defined with exact field names and types
   - Verify "Contract Synchronization Table" shows matching frontend/backend expectations
   - Check that contract testing requirements are specified for both sides

2. **Cross-Reference with Frontend/Backend Docs**:
   - Frontend design should reference the same DTO names
   - Backend services should use the same DTO names
   - Data transformations should be documented on both sides

3. **Flag Missing Information**:
   - If contracts are incomplete or mismatched, note this in the task
   - If codebase analysis is missing, warn that parallel code paths might be created
   - If DTOs aren't synchronized, highlight this risk in the task

**If API or Frontend docs are missing**, skip this validation and note in task generation:
- Tasks will be high-level architectural/planning tasks
- Contract validation cannot be performed without both API and Frontend specs
- Implementation agents will need to define contracts during implementation

### Step 3: Determine Task Scope Based on Available Documentation

Create separate tasks for logical components. Task structure adapts to available design documents.

#### A. Full Documentation (All design docs present)

Standard implementation sequence:

**Task 001: Contract Validation** (`T-E##-F##-001.md`) ⭐ **ALWAYS FIRST when API + Frontend exist**
- Backend: DTO implementation matching spec exactly
- Frontend: TypeScript interface implementation matching backend
- Contract validation tests passing on both sides
- Cross-team synchronization confirmed

**Task 002: Database Setup** (`T-E##-F##-002.md`)
- Schema creation, migrations, RLS policies, indexes
- **Depends on**: T-E##-F##-001

**Task 003: API Implementation** (`T-E##-F##-003.md`)
- Backend services, endpoints, error handling
- **Depends on**: T-E##-F##-001, T-E##-F##-002

**Task 004: Frontend Development** (`T-E##-F##-004.md`)
- UI components, API integration, validation
- **Depends on**: T-E##-F##-001, T-E##-F##-003

**Task 005: Integration & Testing** (`T-E##-F##-005.md`)
- End-to-end integration and validation
- **Depends on**: T-E##-F##-004

**Task 006: Deployment & Monitoring** (`T-E##-F##-006.md`)
- Deployment, monitoring, production validation
- **Depends on**: T-E##-F##-005

#### B. Partial Documentation - Adapt Task Structure

**If PRD + Architecture only:**
- **Task 001**: Architecture Implementation
  - Set up system components, integration points
- **Task 002**: Define detailed design specifications
  - Create missing design docs (database, API, frontend)
- **Task 003**: Integration planning
  - Document integration approach based on architecture

**If PRD + Architecture + Database (no API/Frontend):**
- **Task 001**: Database Schema Implementation
  - Schema creation, migrations, constraints
- **Task 002**: Data access layer design
  - Define repository patterns and data access
- **Task 003**: API design task
  - Create API specification based on database schema

**If PRD + Architecture + Backend (no Frontend):**
- **Task 001**: Backend API Implementation
  - Services, endpoints, business logic
- **Task 002**: API documentation
  - OpenAPI/Swagger specification
- **Task 003**: Frontend design task
  - Create frontend specification based on API

**If PRD + Security/Performance only (infrastructure/DevOps):**
- **Task 001**: Infrastructure setup
  - Based on security and performance requirements
- **Task 002**: Security implementation
  - Authentication, authorization, encryption
- **Task 003**: Performance optimization
  - Caching, monitoring, scaling
- **Task 004**: Deployment pipeline
  - CI/CD, testing, production deployment

**General principle**: Generate tasks for components with specifications, create "design tasks" for missing specifications, adjust task dependencies accordingly.

**Adjust the task structure based on:**
1. Available design documentation
2. Feature complexity
3. Implementation phases (if `08-implementation-phases.md` exists)

### Step 4: Create Each Task in Shark Database

For each task identified, create it using the shark CLI:

1. **Create task in database first**:
   ```bash
   shark task create \
     --epic=<epic-key> \
     --feature=<feature-key> \
     --title="<Task Title>" \
     --agent=<agent-type> \
     --description="<Brief description>" \
     --priority=<1-10> \
     --depends-on=<task-key>,<task-key>
   ```

   - Agent types: `backend`, `frontend`, `api-developer`, `devops`, `qa`, `general`
   - Shark will:
     - Generate task key automatically (T-E##-F##-###)
     - Create database record with metadata
     - Generate markdown file from agent-specific template
     - Save to `/docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`

2. **Fill in the generated task file** with detailed content:
   - Goal (single, clear objective)
   - Success Criteria (measurable checkpoints)
   - Implementation Guidance (references to design docs, NOT code)
   - Validation Gates (what to test, NOT how to test)
   - Context & Resources (links to design doc sections)
   - Notes for Agent (patterns, edge cases, considerations)

For detailed task content structure, see `../context/task-template.md`.

**REMEMBER**: Tasks are HIGH-LEVEL directives with references, NOT code tutorials. Never include SQL, Python, TypeScript, or implementation code in tasks.

### Step 5: Verify Tasks in Database

After creating all tasks:
1. List all feature tasks: `shark task list --feature=<feature-key>`
2. Verify each task: `shark task get <task-key>`
3. Check dependencies are correctly linked

### Step 6: Create Task Index (Optional)

Create `/docs/plan/{epic-key}/{feature-key}/tasks/README.md` to provide a human-readable overview:
- Overview of the feature
- List of all tasks with brief descriptions
- Execution order and workflow diagram
- Links to design documentation
- Note: Definitive task status is tracked in database via `shark` CLI, not in README

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

## Handling Incomplete or Missing Documentation

### When Design Documents Are Missing

If design documents are missing entirely (detected in Step 0):
1. **Present the gap analysis** to the user (as shown in Step 0)
2. **Wait for user confirmation** before proceeding
3. **Adapt task structure** to match available documentation
4. **Generate appropriate tasks**:
   - Implementation tasks for documented components
   - Design/specification tasks for undocumented components
   - Research tasks for uncertain areas

### When Design Documents Are Incomplete or Unclear

If design documents exist but are incomplete or unclear:
1. Identify what's missing or ambiguous
2. Make reasonable assumptions based on best practices
3. Document assumptions in the task's "Notes for Agent" section
4. Recommend updating the design docs with the clarification
5. Consider creating a preliminary task to complete the design doc

### Task Quality with Partial Documentation

Tasks generated from partial documentation will be:
- **Higher-level**: More strategic, less tactical
- **Research-oriented**: May include investigation and design work
- **Flexible**: Allow implementation agents more decision-making authority
- **Documentation-focused**: Emphasize creating missing documentation

This is acceptable! Not all features require full design documentation upfront. The task system adapts to your workflow.

## Final Output

When complete:
1. **Verify all tasks in database**:
   - Run: `shark task list --feature={feature-key}`
   - Confirm all tasks show with correct status (todo), agent, and dependencies

2. **Confirm task files created** in `/docs/plan/{epic-key}/{feature-key}/tasks/` folder

3. **Validate dependencies** form a logical execution sequence:
   - Run: `shark task get {task-key}` for each task to verify depends_on

4. **Check agent assignments** are appropriate for each component

5. **Verify contract synchronization** (CRITICAL):
   - Confirm API task references exact DTO specifications from design doc
   - Confirm Frontend task references exact same DTO specifications
   - Confirm contract validation tests are included in both tasks
   - Verify codebase analysis results are referenced in tasks

6. **Summarize** for the user:
   - **Documentation status**:
     - Available design documents (list them)
     - Missing design documents (list them)
     - Task detail level (HIGH/MEDIUM/LOW based on available docs)
   - **Number of tasks created**: `{count} tasks created in database`
   - **Task list**: Show output of `shark task list --feature={feature-key}`
   - **Estimated total time**
   - **Execution order with dependencies**
   - **Contract synchronization status** (if applicable):
     - Confirmed/needs attention/skipped (missing docs)
   - **Codebase analysis results** (if applicable):
     - New code vs extending existing
   - **Recommendations** (if documentation is partial):
     - Suggest completing missing design docs for more detailed tasks
     - OR confirm that current documentation level is appropriate for feature type
   - **Next steps**:
     - "Tasks ready for development, managed via `shark` CLI"
     - "Start work: `shark task start {first-task-key}`"
     - "Get next task: `shark task next --agent={agent-type}`"
     - "View feature progress: `shark feature get {feature-key}`"
     - If docs missing: "Consider running `/prd` or design workflows to complete documentation"

Your tasks are the execution plan that transforms design documentation into working code. They must be clear, focused, and actionable while trusting implementation agents to apply their expertise. The level of detail adapts to the available documentation.
