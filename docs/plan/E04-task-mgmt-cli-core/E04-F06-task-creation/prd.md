# Feature: Task Creation & Templating

## Epic

- [Epic PRD](/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/epic.md)

## Goal

### Problem

Developers and AI agents need to create new tasks programmatically with consistent structure, proper metadata, and automatic key generation. Currently, creating tasks requires manually determining the next available task number (T-E01-F02-003, T-E01-F02-004, etc.), writing markdown files with correct frontmatter format, adding to database, and placing files in the correct folder. This manual process is error-prone: developers forget required frontmatter fields, use incorrect key formats, skip database insertion, or put files in wrong folders. Agents cannot create tasks programmatically without complex markdown generation code. Different agent types (frontend, backend, testing) need different task templates with appropriate structure and boilerplate. Without standardized templates, task files have inconsistent format, making them harder to parse and manage.

### Solution

Implement `pm task create` command that automates the entire task creation workflow: automatically generates the next available task key, creates database record, generates markdown file from agent-specific template, saves file to the feature's tasks/ directory, and returns the new task key. Provide customizable Jinja2 templates for each agent type (frontend, backend, api, testing, devops, general) that include appropriate frontmatter, description structure, and boilerplate content. Support required flags (--epic, --feature, --title, --agent) and optional metadata (--priority, --depends-on, --description). Validate all inputs before creation (epic/feature exist, dependency tasks exist, key format is valid). Integrate with E04-F01 (Database), E04-F05 (File Path Management), and E04-F02 (CLI Framework) to ensure atomic creation and consistent error handling.

### Impact

- **Automation**: Reduce task creation time from 5+ minutes (manual file + DB) to <5 seconds with single command
- **Consistency**: 100% of created tasks follow standard format with all required fields
- **Agent Enablement**: Agents can create tasks programmatically without complex markdown generation
- **Error Prevention**: Automatic validation prevents duplicate keys, invalid references, and format errors
- **Developer Experience**: Templates provide appropriate structure for each agent type (frontend vs backend templates differ)

## User Personas

### Primary Persona: AI Agent (Claude Code Agents)

**Role**: Autonomous code generation and task execution agent
**Environment**: Claude Code CLI, needs to create tasks programmatically

**Key Characteristics**:
- Discovers missing work during implementation
- Needs to create blocking tasks (prerequisites)
- Must include proper metadata (dependencies, agent type)
- Requires JSON output for created task

**Goals**:
- Run `pm task create --epic=E01 --feature=F02 --title="Document API contract" --agent=api-developer --json` and get task key
- Create tasks with dependencies: `--depends-on=T-E01-F01-005`
- Use returned task key to reference new task
- Create multiple related tasks in sequence

**Pain Points this Feature Addresses**:
- Cannot manually write markdown files
- Determining next task number is complex
- No way to create tasks programmatically
- Manual creation breaks agent workflows

### Secondary Persona: Product Manager / Technical Lead

**Role**: Human developer creating tasks for epics/features
**Environment**: Terminal, planning work for agents

**Key Characteristics**:
- Creates multiple tasks when breaking down features
- Needs tasks to have correct template for agent type
- Wants automatic key generation (no mental math)
- Requires consistent frontmatter format

**Goals**:
- Create task with single command: `pm task create --epic=E01 --feature=F02 --title="Build login form" --agent=frontend`
- Have frontend template automatically applied
- Get properly formatted markdown file ready to edit
- See task created in feature's tasks/ directory

**Pain Points this Feature Addresses**:
- Manual task key generation is tedious
- Forgetting required frontmatter fields
- Inconsistent markdown format across tasks
- Must manually create DB record and file

## User Stories

### Must-Have User Stories

**Story 1: Create Task with Required Fields**
- As a user, I want to run `pm task create --epic=E01 --feature=F02 --title="Build auth middleware" --agent=backend`, so that a new task is created with automatic key generation.

**Story 2: Automatic Key Generation**
- As a user, I want task keys generated automatically (next available number in sequence), so that I don't manually calculate T-E01-F02-003, T-E01-F02-004, etc.

**Story 3: Agent-Specific Templates**
- As a user, I want tasks for `--agent=frontend` to use the frontend template (different from backend template), so that task files have appropriate structure.

**Story 4: Create Database Record**
- As a user, I want the task automatically inserted into the database with all metadata, so that `pm task list` shows the new task immediately.

**Story 5: Create Markdown File**
- As a user, I want the task markdown file created in the feature's tasks/ directory with proper frontmatter, so that the file is ready to edit.

**Story 6: Add Optional Description**
- As a user, I want to include `--description="Detailed description text"` when creating a task, so that I can provide context beyond the title.

**Story 7: Set Priority**
- As a user, I want to include `--priority=1` to create high-priority tasks, so that important work is marked appropriately.

**Story 8: Specify Dependencies**
- As a user, I want to include `--depends-on=T-E01-F01-005,T-E01-F01-007` to link prerequisite tasks, so that task dependencies are captured at creation time.

**Story 9: Return Task Key**
- As an AI agent, I want the command to return the created task key in JSON (`{"key": "T-E01-F02-003"}`), so that I can reference the new task in subsequent commands.

**Story 10: Validate Inputs**
- As a user, I want invalid inputs (non-existent epic, invalid agent type) to be rejected with clear errors, so that I don't create malformed tasks.

### Should-Have User Stories

**Story 11: Custom Template Override**
- As a user, I want to specify `--template=custom-template.md` to use a custom template instead of defaults, so that I can create specialized task formats.

**Story 12: Validate Dependencies Exist**
- As a user, I want the system to verify that dependency task IDs exist before creating the task, so that I don't create tasks with broken dependency links.

**Story 13: Bulk Task Creation**
- As a user, I want to create multiple tasks from a CSV file with `pm task import tasks.csv`, so that I can quickly populate a feature with tasks.

### Could-Have User Stories

**Story 14: Interactive Task Creation**
- As a user, I want to run `pm task create --interactive` and be prompted for all fields, so that I don't memorize all flag names.

**Story 15: Clone Existing Task**
- As a user, I want to run `pm task clone T-E01-F01-005 --title="New title"` to duplicate a task's structure, so that I can create similar tasks quickly.

## Requirements

### Functional Requirements

**Task Creation Command (pm task create):**

1. The system must provide `pm task create` command with required flags:
   - `--epic=<epic-key>` (e.g., --epic=E01)
   - `--feature=<feature-key>` (e.g., --feature=E01-F02 or --feature=F02)
   - `--title="<task title>"` (quoted string)
   - `--agent=<agent-type>` (enum: frontend, backend, api, testing, devops, general)

2. The system must support optional flags:
   - `--description="<long description>"` (multi-line text)
   - `--priority=<1-10>` (default: 5)
   - `--depends-on=<task-key>,<task-key>` (comma-separated list)

3. Missing required flags must raise error: "Error: Missing required option '--epic'" with exit code 1

4. The command must support `--json` flag to return created task details in JSON format

**Automatic Key Generation:**

5. The system must query the database to find the highest task number for the specified feature

6. The next task key must be generated as: `T-<epic-key>-<feature-key>-<next-number>`
   - Epic E01, Feature F02, next number 003 → `T-E01-F02-003`
   - Numbers must be zero-padded to 3 digits (001, 002, ..., 099, 100)

7. If no tasks exist for the feature, start with 001

8. If 999 tasks already exist for a feature, raise error: "Error: Feature E01-F02 has reached maximum task count (999)"

9. Key generation must handle concurrent creation (use database locks or transactions to prevent duplicates)

**Input Validation:**

10. The system must validate that specified epic exists in database (raise error if not)

11. The system must validate that specified feature exists and belongs to specified epic

12. If `--feature=F02` is provided (without epic prefix), the system must prepend the epic key: `E01-F02`

13. The system must validate `--agent` against allowed values (frontend, backend, api, testing, devops, general)

14. The system must validate `--priority` is integer between 1 and 10

15. If `--depends-on` is provided, the system must validate that each referenced task exists in database

16. Invalid dependency references must raise error: "Error: Dependency task T-E01-F99-001 does not exist"

**Database Record Creation:**

17. The system must create a task record in the database with fields:
    - feature_id (foreign key to features table)
    - key (generated task key)
    - title (from --title)
    - description (from --description, or empty string)
    - status (always "todo" for new tasks)
    - agent_type (from --agent)
    - priority (from --priority, default 5)
    - depends_on (JSON array from --depends-on, or empty array)
    - file_path (calculated as `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`)
    - created_at (automatic timestamp)

18. Database insertion must use a transaction (rollback on failure)

**Template System:**

19. The system must support Jinja2 templates for task file generation

20. The system must provide default templates for each agent type:
    - `templates/task-frontend.md`
    - `templates/task-backend.md`
    - `templates/task-api.md`
    - `templates/task-testing.md`
    - `templates/task-devops.md`
    - `templates/task-general.md`

21. Templates must have access to variables: `key`, `title`, `description`, `epic`, `feature`, `agent_type`, `priority`, `depends_on`, `created_at`

22. Templates must include frontmatter block with all required metadata:
    ```yaml
    ---
    key: T-E01-F02-003
    title: Build authentication middleware
    epic: E01
    feature: E01-F02
    agent: backend
    status: todo
    priority: 5
    depends_on: [T-E01-F01-005]
    created_at: 2025-12-14T10:30:00Z
    ---
    ```

23. Templates must include structure appropriate for agent type:
    - Frontend: Component specs, UI/UX notes, acceptance criteria
    - Backend: API endpoints, data models, business logic
    - API: Endpoint specs, request/response examples, error codes
    - Testing: Test scenarios, test data, coverage requirements

24. Template rendering must handle missing optional fields gracefully (don't show "Description: None")

**Markdown File Creation:**

25. The system must render the template with task data to generate markdown content

26. The system must save the markdown file to `docs/plan/{epic-key}/{feature-key}/tasks/{task-key}.md`

27. File creation must use E04-F05 file path utilities to generate the path and ensure the feature/tasks/ directory exists

28. File must be written with UTF-8 encoding

29. If file already exists, raise error: "Error: Task file already exists: <path>" (don't overwrite)

**Atomic Creation:**

30. Task creation must be atomic: either database record AND file are created, or neither

31. If database insertion fails, no file should be created

32. If file creation fails, database transaction must rollback

33. The system must use `TransactionalFileOperation` from E04-F05 for atomicity

**Command Output:**

34. Human-readable output must display:
    - Success message: "Created task T-E01-F02-003: <title>"
    - File location: "File created at: docs/plan/{epic}/{feature}/tasks/T-E01-F02-003.md"

35. JSON output must return created task object:
    ```json
    {
      "key": "T-E01-F02-003",
      "title": "Build auth middleware",
      "status": "todo",
      "file_path": "docs/plan/E01-epic/F02-feature/tasks/T-E01-F02-003.md",
      "epic": "E01",
      "feature": "E01-F02",
      "agent_type": "backend",
      "priority": 5,
      "depends_on": ["T-E01-F01-005"],
      "created_at": "2025-12-14T10:30:00Z"
    }
    ```

**History Recording:**

36. The system must create a task_history record with:
    - old_status = null
    - new_status = "todo"
    - agent = current user
    - notes = "Task created"
    - timestamp = created_at

**Integration with Other Features:**

37. The command must use E04-F01 database models and queries

38. The command must use E04-F02 CLI framework (Click, Rich, error handling, --json support)

39. The command must use E04-F05 file path management for file path generation and directory creation

40. Created tasks must immediately be visible in `pm task list` output

### Non-Functional Requirements

**Performance:**

- Task creation (DB insert + file write) must complete in <500ms
- Key generation query must complete in <50ms
- Template rendering must complete in <100ms
- Concurrent creation must not create duplicate keys

**Usability:**

- Error messages must be specific: "Epic E99 does not exist. Use 'pm epic list' to see available epics."
- Success messages must include next steps: "Task created. Start work with: pm task start T-E01-F02-003"
- Template output must be well-formatted and ready to edit
- Generated markdown must have consistent indentation and spacing

**Reliability:**

- Atomic creation prevents orphaned DB records or files
- Transaction rollback ensures consistency on failure
- Validation prevents creating tasks with invalid references
- Concurrent creation uses locks to prevent race conditions

**Maintainability:**

- Templates must be easily customizable (external .md files, not hardcoded)
- Adding new agent types requires only adding new template file
- Template variables must be documented in template header comments

**Compatibility:**

- Templates must work across platforms (no platform-specific paths or line endings)
- Generated files must use Unix line endings (LF) for Git compatibility
- Template rendering must not break on special characters in title/description

## Acceptance Criteria

### Basic Task Creation

**Given** Epic E01 and Feature E01-F02 exist
**And** no tasks exist for Feature E01-F02
**When** I run `pm task create --epic=E01 --feature=F02 --title="Build login form" --agent=frontend`
**Then** a task is created with key "T-E01-F02-001"
**And** database contains the task with status="todo"
**And** file exists at the feature's tasks/ directory
**And** file contains frontmatter with correct metadata
**And** success message displays the task key

### Automatic Key Sequencing

**Given** Feature E01-F02 already has tasks T-E01-F02-001 and T-E01-F02-002
**When** I run `pm task create --epic=E01 --feature=F02 --title="New task" --agent=backend`
**Then** a task is created with key "T-E01-F02-003" (next in sequence)

**Given** Feature E01-F02 has task T-E01-F02-099
**When** I create a new task
**Then** key is "T-E01-F02-100" (handles 3-digit → 4-digit transition)

### Input Validation

**Given** Epic E99 does not exist
**When** I run `pm task create --epic=E99 --feature=F01 --title="Test" --agent=backend`
**Then** error is displayed: "Error: Epic E99 does not exist"
**And** exit code is 1
**And** no database record is created
**And** no file is created

**Given** I run `pm task create` without --title flag
**When** the command executes
**Then** error is displayed: "Error: Missing required option '--title'"
**And** exit code is 1

**Given** I run `pm task create` with `--agent=invalid-agent`
**When** the command executes
**Then** error is displayed: "Error: Invalid agent type. Must be one of: frontend, backend, api, testing, devops, general"
**And** exit code is 1

### Dependency Validation

**Given** task T-E01-F01-005 exists
**When** I run `pm task create --epic=E01 --feature=F02 --title="Task with dep" --agent=backend --depends-on=T-E01-F01-005`
**Then** task is created successfully
**And** depends_on field contains ["T-E01-F01-005"]

**Given** task T-E01-F99-999 does not exist
**When** I run `pm task create` with `--depends-on=T-E01-F99-999`
**Then** error is displayed: "Error: Dependency task T-E01-F99-999 does not exist"
**And** no task is created

### Template Application

**Given** I create a task with `--agent=frontend`
**When** the task file is generated
**Then** the file uses the frontend template
**And** includes frontend-specific sections (Component Specs, UI/UX notes)

**Given** I create a task with `--agent=backend`
**When** the task file is generated
**Then** the file uses the backend template
**And** includes backend-specific sections (API endpoints, data models)

### Frontmatter Generation

**Given** I create a task with full metadata
**When** the file is generated
**Then** frontmatter includes all fields:
```yaml
---
key: T-E01-F02-003
title: Build auth middleware
epic: E01
feature: E01-F02
agent: backend
status: todo
priority: 3
depends_on: [T-E01-F01-005, T-E01-F01-007]
created_at: 2025-12-14T10:30:00Z
---
```

### Atomic Creation

**Given** database is available but filesystem has permission error on feature tasks/ directory
**When** I run `pm task create`
**Then** database transaction is rolled back
**And** no database record exists
**And** no file exists
**And** error message explains filesystem permission issue

**Given** file creation succeeds but database constraint fails (e.g., duplicate key)
**When** the command executes
**Then** file is deleted (rollback)
**And** error message explains database constraint violation

### JSON Output

**Given** I run `pm task create --epic=E01 --feature=F02 --title="Test" --agent=backend --json`
**When** the command completes successfully
**Then** output is valid JSON
**And** JSON contains created task object with key, title, status, file_path, etc.
**And** I can parse with `jq '.key'` to extract "T-E01-F02-003"

### Optional Fields

**Given** I create a task without `--description`
**When** the task is created
**Then** description field is empty string (not null)
**And** template doesn't show "Description: " section

**Given** I create a task without `--priority`
**When** the task is created
**Then** priority defaults to 5

**Given** I create a task without `--depends-on`
**When** the task is created
**Then** depends_on is empty array []

### Command Output

**Given** I successfully create a task
**When** the command completes
**Then** output includes:
- "Created task T-E01-F02-003: Build auth middleware"
- "File created at: docs/plan/{epic}/{feature}/tasks/T-E01-F02-003.md"
- "Start work with: pm task start T-E01-F02-003"

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Task Editing** - Updating existing tasks is done by editing markdown files manually or via future commands. No `pm task update` in this feature.

2. **Task Deletion** - Deleting tasks is out of scope for E04.

3. **Bulk Import** - Importing tasks from CSV or JSON files is a Could-Have, deferred to future work.

4. **Interactive Creation** - Wizard-style interactive prompts are Could-Have, not Must-Have.

5. **Task Cloning** - Duplicating existing tasks is Could-Have.

6. **Custom Template Management** - Commands to create, edit, or manage templates are out of scope (users edit template files directly).

7. **Task Validation** - Validating task file content against schema is out of scope (basic frontmatter validation only).

8. **Task Conversion** - Converting between agent types or templates is out of scope.

9. **Task Archiving on Creation** - Tasks are always created with status="todo" (no creating directly to archived).

10. **Epic/Feature Creation** - This feature creates tasks only. Creating epics/features is manual (PRD-based workflow).
