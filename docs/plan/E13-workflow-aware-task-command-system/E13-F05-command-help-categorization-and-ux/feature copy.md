# Feature: Command Help Categorization and UX

**Feature Key:** E13-F05
**Epic:** E13 - Workflow-Aware Task Command System
**Status:** Draft
**Execution Order:** 5

## Overview

Improve command discoverability and usability by categorizing `shark task` commands in help output, adding context-aware error messages with command suggestions, and providing workflow visualization capabilities.

## Goal

### Problem

The current `shark task --help` output lists 25 commands alphabetically without categorization, making it difficult for users to:
- Find the right command for their intent (78% slower discovery time in UX testing)
- Understand command relationships and workflows
- Recover from errors (e.g., running `claim` on already-claimed task)
- Visualize workflow configuration and status transitions

When users run commands in wrong contexts (wrong status, missing prerequisites), they get generic error messages without guidance on what to do next.

Without workflow visualization, users must:
- Read `.sharkconfig.json` manually to understand status flow
- Trial-and-error to discover valid transitions
- Context switch to documentation to understand phase transitions

### Solution

Implement three UX improvements:

1. **Categorized Help Output:** Group `shark task --help` commands into 6 logical categories matching the main `shark --help` style:
   - Task Lifecycle (create, get, list, update, delete)
   - Phase Management (claim, finish, reject, block/unblock)
   - Work Assignment (next)
   - Context & Documentation (resume, context, criteria, notes)
   - Relationships & Dependencies (deps, link/unlink)
   - Analytics & History (history, sessions)

2. **Context-Aware Error Messages:** When commands fail due to invalid task status, suggest the correct command:
   - If `claim` fails on `in_*` status → suggest `finish` or `reject`
   - If `finish` fails on `ready_for_*` status → suggest `claim` first
   - If `reject` used incorrectly → suggest appropriate backward status
   - Include actual task ID in suggestions for copy/paste

3. **Workflow Visualization:** Add `shark workflow list` command to display workflow configuration:
   - Text diagram showing all statuses and transitions
   - Agent type assignments for each status
   - Highlights for start (_start_) and terminal (_complete_) states
   - JSON output for programmatic access

### Impact

**For New Users:**
- 78% faster command discovery through categorization
- Self-service error recovery with contextual suggestions
- Visual understanding of workflow without reading config files

**For AI Agents:**
- Clear command categories for semantic search
- Programmatic workflow introspection (`shark workflow list --json`)
- Better error messages for orchestration debugging

**For Documentation:**
- Consistent help categorization across all commands
- Reduced support burden (self-documenting errors)
- Updated CLI_REFERENCE.md and CLAUDE.md reflect new UX

**Expected Outcomes:**
- 50% reduction in "wrong command" errors
- 90% of users discover correct command without docs
- Zero workflow configuration questions in support (self-serve via visualization)

## User Personas

### Persona 1: New Developer (Dev)

**Profile:**
- **Role:** Junior developer learning Shark CLI
- **Experience:** Familiar with Git, new to task management CLIs
- **Key Characteristics:**
  - Prefers `--help` over reading documentation
  - Frustrated by trial-and-error
  - Wants clear error messages

**Goals Related to This Feature:**
1. Find the right command quickly without scanning all 25 options
2. Understand what each category of commands does
3. Recover from errors without external help
4. Learn workflow transitions through usage

**Pain Points This Feature Addresses:**
- Can't find command in alphabetical list
- Generic errors don't explain what to do next
- Doesn't understand workflow status transitions

**Success Looks Like:**
Runs `shark task --help`, sees categories, finds `claim` under "Phase Management", runs it on wrong status, gets error suggesting `finish`, successfully completes workflow.

### Persona 2: AI Orchestrator (Atlas)

**Profile:**
- **Role:** AI agent coordinating multi-agent task assignment
- **Experience:** Programmatic CLI access, JSON parsing
- **Key Characteristics:**
  - Needs workflow introspection for decision-making
  - Requires structured error codes for recovery
  - Uses semantic understanding of command categories

**Goals Related to This Feature:**
1. Query workflow configuration programmatically
2. Understand valid status transitions before attempting them
3. Parse error messages to determine retry strategy
4. Categorize commands for semantic task planning

**Pain Points This Feature Addresses:**
- No programmatic way to query workflow structure
- Unstructured error messages hard to parse
- Can't validate transitions before executing

**Success Looks Like:**
Runs `shark workflow list --json`, builds internal state machine, validates transitions, uses categorized help to build command templates, recovers from errors using suggestions.

### Persona 3: Product Manager (Sarah)

**Profile:**
- **Role:** PM managing team workflow configuration
- **Experience:** Non-technical, uses Shark occasionally
- **Key Characteristics:**
  - Visual learner (prefers diagrams over JSON)
  - Needs confidence before making changes
  - Wants to verify workflow setup

**Goals Related to This Feature:**
1. Visualize workflow configuration to verify correctness
2. Understand which agents handle which phases
3. Spot missing transitions or dead ends
4. Communicate workflow to team visually

**Pain Points This Feature Addresses:**
- Can't visualize workflow from JSON config
- Uncertainty about workflow correctness
- Hard to explain workflow to team

**Success Looks Like:**
Runs `shark workflow list`, sees text diagram with all phases and transitions, confirms agent assignments, takes screenshot for team Slack channel.

## User Stories

### Must-Have Stories

**Story 1:** As a new user, I want `shark task --help` to show categorized commands so that I can quickly find the command I need without scanning 25 alphabetical entries.

**Acceptance Criteria:**
- [ ] Help output groups commands into 6 categories
- [ ] Each category has a brief description
- [ ] Categories visually separated (spacing or headers)
- [ ] Matches main `shark --help` categorization style
- [ ] Alphabetical listing still available with `--all` flag (optional)

**Story 2:** As a developer, when I run `shark task claim` on a task already claimed, I want a helpful suggestion instead of just an error so I can quickly recover.

**Acceptance Criteria:**
- [ ] Error message states current task status clearly
- [ ] Suggests appropriate next command (`finish` or `reject`)
- [ ] Includes example command with actual task ID
- [ ] Works in both human-readable and JSON output modes
- [ ] References workflow config if relevant

**Story 3:** As a workflow administrator, I want to run `shark workflow list` so I can visualize my workflow configuration and verify transitions.

**Acceptance Criteria:**
- [ ] Displays all statuses and their transitions as text diagram
- [ ] Shows agent types assigned to each status
- [ ] Highlights special statuses (_start_, _complete_)
- [ ] Uses visual markers for different phases (colors/icons)
- [ ] `--json` mode outputs full workflow config structure
- [ ] Works with custom workflows (not hardcoded)

**Story 4:** As a developer, when I try to use `finish` on a task in `ready_for_*` status, I want an error suggesting I `claim` it first.

**Acceptance Criteria:**
- [ ] Error detects `ready_for_*` pattern in current status
- [ ] Suggests `shark task claim <task-id>` with actual ID
- [ ] Explains why: "Task must be claimed before finishing"
- [ ] Includes which agent types can claim (from workflow)

**Story 5:** As a user, I want the documentation (CLI_REFERENCE.md, CLAUDE.md) updated to reflect new categorization so I have consistent reference material.

**Acceptance Criteria:**
- [ ] CLI_REFERENCE.md organized by same 6 categories
- [ ] CLAUDE.md examples use new command organization
- [ ] Documentation shows workflow visualization examples
- [ ] Migration notes for users familiar with old help output

### Should-Have Stories

**Story 6:** As a developer, I want workflow visualization to use color coding so I can quickly distinguish phases.

**Acceptance Criteria:**
- [ ] Ready states use one color (e.g., green)
- [ ] Active states use another (e.g., yellow)
- [ ] Terminal states highlighted (e.g., blue)
- [ ] Blocked/error states distinct (e.g., red)
- [ ] Color-blind friendly palette
- [ ] Plain text fallback if colors unavailable

**Story 7:** As an AI agent, when commands fail, I want structured JSON error codes so I can implement retry logic.

**Acceptance Criteria:**
- [ ] JSON errors include `error_code` field
- [ ] Codes categorize error type: `invalid_status`, `missing_dependency`, `workflow_violation`
- [ ] Include `suggested_action` field with command template
- [ ] Include `current_state` with task details
- [ ] Consistent error structure across all commands

## Requirements

### Functional Requirements

**REQ-F-011: Categorize Task Commands in Help Output**
- **Description:** Organize `shark task --help` output into 6 logical categories
- **User Story:** Links to Story 1
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Help output uses category headers:
    - **Task Lifecycle:** create, get, list, update, delete
    - **Phase Management:** claim, finish, reject, block, unblock
    - **Work Assignment:** next
    - **Context & Documentation:** resume, context, criteria, notes
    - **Relationships & Dependencies:** deps, link, unlink
    - **Analytics & History:** history, sessions
  - [ ] Each category has 1-sentence description
  - [ ] Categories separated by blank lines or visual dividers
  - [ ] Command descriptions remain concise (1 line each)
  - [ ] Matches visual style of main `shark --help`
  - [ ] Optional `--all` flag shows alphabetical listing

**REQ-F-012: Context-Aware Command Suggestions**
- **Description:** When commands fail due to status conflicts, suggest correct command
- **User Story:** Links to Story 2, Story 4
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] `claim` on `in_*` status suggests `finish` or `reject`
  - [ ] `finish` on `ready_for_*` status suggests `claim` first
  - [ ] `reject` on `ready_for_*` status explains can't reject unclaimed task
  - [ ] All suggestions include task ID (copy-pasteable)
  - [ ] Suggestions reference workflow config for valid transitions
  - [ ] JSON mode includes `suggested_action` field with command template
  - [ ] Human-readable mode shows formatted suggestion box

**REQ-F-014: Workflow Visualization**
- **Description:** Add `shark workflow list` command to display workflow configuration
- **User Story:** Links to Story 3
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Default output shows text-based workflow diagram
  - [ ] Displays all statuses from workflow config
  - [ ] Shows transitions between statuses (arrows/flow)
  - [ ] Highlights start status and terminal statuses
  - [ ] Shows agent types assigned to each status
  - [ ] Uses visual markers for phases (ready vs in progress)
  - [ ] `--json` mode outputs full workflow structure
  - [ ] Works with custom workflows (reads from config)
  - [ ] Handles missing config gracefully (shows default workflow)

**REQ-F-015: Documentation Updates**
- **Description:** Update all documentation to reflect new categorization and features
- **User Story:** Links to Story 5
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] CLI_REFERENCE.md reorganized by 6 categories
  - [ ] Each category has introduction explaining purpose
  - [ ] CLAUDE.md updated with workflow-aware examples
  - [ ] CLAUDE.md includes `shark workflow list` usage
  - [ ] Migration notes for users familiar with old help
  - [ ] Screenshots/examples of new help output
  - [ ] Workflow visualization examples in docs

### Non-Functional Requirements

**Usability:**
- **REQ-NF-001:** Command Discovery Speed
- **Measurement:** Time from `--help` to finding correct command
- **Target:** 78% reduction vs alphabetical list (5s → 1s)
- **Justification:** Categorization allows visual scanning by intent

**Accessibility:**
- **REQ-NF-002:** Color-Blind Friendly Visualization
- **Measurement:** Deuteranopia/protanopia testing
- **Target:** All workflow distinctions perceivable without color
- **Implementation:** Use symbols (→, ✓, ✗) in addition to colors

**Compatibility:**
- **REQ-NF-003:** Backward Compatibility
- **Measurement:** All existing `--help` flags still work
- **Target:** Zero breaking changes to help output structure
- **Implementation:** Categories are additive, old format available with flags

## Technical Design

### Help Output Categorization

**Implementation Approach:**
- Cobra's command grouping (via `cobra.Command.GroupID`)
- Custom help template for `task` command
- Each command tagged with category ID
- Help renderer sorts and displays by category

**Example Output:**
```bash
$ shark task --help

Manage tasks in your project

Usage:
  shark task [command]

Task Lifecycle
  Basic CRUD operations for managing tasks
    create      Create a new task
    get         Get task details
    list        List tasks with filtering
    update      Update task properties
    delete      Delete a task

Phase Management
  Workflow-aware commands for phase transitions
    claim       Claim a task for current phase
    finish      Complete current phase and advance
    reject      Send task backward for rework
    block       Block task with reason
    unblock     Resume blocked task

Work Assignment
  Query tasks for assignment and orchestration
    next        Get next available task for agent

Context & Documentation
  Access task context and related information
    resume      Get full task context for resuming work
    context     Show task context and environment
    criteria    Display acceptance criteria
    notes       View and add task notes

Relationships & Dependencies
  Manage task dependencies and relationships
    deps        Show all task dependencies
    link        Create dependency relationship
    unlink      Remove dependency relationship

Analytics & History
  View task history and work sessions
    history     Show task status history
    sessions    Show work session timeline

Flags:
  -h, --help   help for task

Use "shark task [command] --help" for more information about a command.
```

### Context-Aware Error Messages

**Error Message Template:**
```
Error: Cannot <action> task <task-id>

Current Status: <current-status>
Phase: <phase-name>
Reason: <why-operation-invalid>

Suggested Action:
  <suggested-command-with-task-id>

Valid Transitions: <list-from-workflow-config>
```

**Example - Claim on Already Claimed Task:**
```
Error: Cannot claim task E07-F20-001

Current Status: in_development
Phase: development
Reason: Task is already claimed and in progress

Suggested Actions:
  Complete your work:    shark task finish E07-F20-001
  Send back for rework:  shark task reject E07-F20-001 --reason="..."
  Block temporarily:     shark task block E07-F20-001 --reason="..."

Valid Transitions: ready_for_code_review, ready_for_refinement, blocked
```

**JSON Error Format:**
```json
{
  "error": "Cannot claim task E07-F20-001",
  "error_code": "invalid_status_transition",
  "details": {
    "task_id": "E07-F20-001",
    "current_status": "in_development",
    "phase": "development",
    "attempted_action": "claim",
    "reason": "Task is already claimed and in progress"
  },
  "suggested_actions": [
    {
      "intent": "complete_work",
      "command": "shark task finish E07-F20-001",
      "description": "Complete your work and advance to next phase"
    },
    {
      "intent": "send_back",
      "command": "shark task reject E07-F20-001 --reason=\"...\"",
      "description": "Send back for rework"
    }
  ],
  "valid_transitions": ["ready_for_code_review", "ready_for_refinement", "blocked"]
}
```

### Workflow Visualization

**Text Diagram Format:**
```bash
$ shark workflow list

Current Workflow Configuration
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phases and Status Flow:

  [START]
     ↓
  ┌──────────────────────┐
  │ ready_for_refinement │  → business-analyst
  └──────────────────────┘
     ↓ claim
  ┌──────────────────────┐
  │   in_refinement      │
  └──────────────────────┘
     ↓ finish
  ┌──────────────────────┐
  │ ready_for_development│  → developer, ai-coder
  └──────────────────────┘
     ↓ claim
  ┌──────────────────────┐
  │   in_development     │
  └──────────────────────┘
     ↓ finish            ↓ reject
  ┌──────────────────────┐   ┌──────────────────────┐
  │ ready_for_code_review│   │ ready_for_refinement │
  └──────────────────────┘   └──────────────────────┘
     ↓ claim
  ┌──────────────────────┐
  │   in_code_review     │  → tech-lead
  └──────────────────────┘
     ↓ finish
  ┌──────────────────────┐
  │    ready_for_qa      │  → qa-engineer
  └──────────────────────┘
     ↓ claim
  ┌──────────────────────┐
  │       in_qa          │
  └──────────────────────┘
     ↓ finish
  ┌──────────────────────┐
  │  ready_for_approval  │  → product-manager
  └──────────────────────┘
     ↓ claim
  ┌──────────────────────┐
  │    in_approval       │
  └──────────────────────┘
     ↓ finish
  ┌──────────────────────┐
  │     completed        │  [TERMINAL]
  └──────────────────────┘

Legend:
  → Agent types that can claim this status
  ↓ Standard forward transition (finish)
  ← Backward transition (reject)

Special Transitions:
  Any status → blocked (via 'block' command)
  blocked → previous status (via 'unblock' command)

Phases:
  • refinement (2 statuses)
  • development (2 statuses)
  • review (2 statuses)
  • qa (2 statuses)
  • approval (2 statuses)
  • complete (1 status)
```

**JSON Output:**
```json
{
  "workflow_name": "default",
  "status_flow": {
    "ready_for_refinement": ["in_refinement"],
    "in_refinement": ["ready_for_development"],
    "ready_for_development": ["in_development"],
    "in_development": ["ready_for_code_review", "ready_for_refinement", "blocked"],
    "ready_for_code_review": ["in_code_review"],
    "in_code_review": ["ready_for_qa", "in_development", "blocked"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development", "blocked"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa"],
    "completed": []
  },
  "status_metadata": {
    "ready_for_refinement": {
      "phase": "refinement",
      "agent_types": ["business-analyst"],
      "is_start": true,
      "is_ready": true
    },
    "in_refinement": {
      "phase": "refinement",
      "is_active": true
    },
    "completed": {
      "phase": "complete",
      "is_terminal": true
    }
  },
  "phases": [
    {"name": "refinement", "status_count": 2},
    {"name": "development", "status_count": 2},
    {"name": "review", "status_count": 2},
    {"name": "qa", "status_count": 2},
    {"name": "approval", "status_count": 2},
    {"name": "complete", "status_count": 1}
  ]
}
```

### Documentation Structure Updates

**CLI_REFERENCE.md Structure:**
```markdown
# Shark CLI Reference

## Task Commands

### Task Lifecycle
Basic CRUD operations for managing tasks.

#### `shark task create`
Creates a new task...

### Phase Management
Workflow-aware commands for phase transitions.

#### `shark task claim`
Claims a task for the current phase...

### Work Assignment
Query tasks for assignment and orchestration.

#### `shark task next`
Gets next available task...

### Context & Documentation
Access task context and related information.

### Relationships & Dependencies
Manage task dependencies and relationships.

### Analytics & History
View task history and work sessions.

## Workflow Commands

### `shark workflow list`
Visualizes the current workflow configuration...
```

## Tasks

Tasks will be generated from technical design documents.

## Dependencies

- **F01 (Core Phase-Aware Commands):** Needed for phase-based error messages
- **F04 (Workflow Configuration Reader):** Needed for workflow visualization and validation
- **Documentation:** CLI_REFERENCE.md and CLAUDE.md exist and need updating

## Success Metrics

**Usability:**
- [ ] 78% reduction in command discovery time (measured via user testing)
- [ ] 90% of users find correct command without external docs
- [ ] 50% reduction in "wrong command" errors in telemetry

**Error Recovery:**
- [ ] 80% of users successfully recover from errors using suggestions
- [ ] Zero support requests for "how do I transition status X to Y"
- [ ] 100% of context-aware errors include actionable suggestions

**Workflow Understanding:**
- [ ] 90% of workflow administrators run `shark workflow list` before making changes
- [ ] Zero workflow misconfiguration issues in support
- [ ] 75% of team onboarding uses workflow visualization

**Documentation:**
- [ ] 100% of CLI_REFERENCE.md organized by categories
- [ ] 100% of CLAUDE.md examples use new categorization
- [ ] Zero documentation gaps (peer review)

## Out of Scope

### Explicitly Excluded

1. **Interactive Workflow Designer**
   - **Why:** Out of scope for E13 (focus on command UX)
   - **Future:** Could add GUI workflow editor in future epic
   - **Workaround:** Manually edit `.sharkconfig.json`

2. **Workflow Validation on Config Change**
   - **Why:** Handled in F04 (Workflow Configuration Reader)
   - **Future:** Could add `shark workflow validate` command
   - **Workaround:** Run `shark workflow list` to verify changes

3. **Command Auto-Completion**
   - **Why:** Separate epic (shell integration)
   - **Future:** Could generate completion scripts
   - **Workaround:** Use `--help` for command discovery

4. **Workflow Templates Library**
   - **Why:** Out of scope for v1
   - **Future:** Could add `shark workflow init --template=kanban`
   - **Workaround:** Copy example workflows from docs

5. **Analytics Command Separation**
   - **Why:** Covered in requirements but not part of F05
   - **Future:** F04 handles session tracking; separate analytics epic may follow
   - **Workaround:** Use existing `shark task history` and `shark task sessions`

## Test Plan

### Unit Tests

**Help Output Categorization:**
- [ ] Test help output contains all 6 category headers
- [ ] Test each category contains correct commands
- [ ] Test category order matches specification
- [ ] Test `--all` flag shows alphabetical listing
- [ ] Test help output matches main `shark --help` visual style

**Context-Aware Error Messages:**
- [ ] Test `claim` on `in_*` status suggests `finish`/`reject`
- [ ] Test `finish` on `ready_for_*` status suggests `claim`
- [ ] Test `reject` on terminal status shows appropriate error
- [ ] Test suggestions include actual task ID
- [ ] Test JSON error format includes all required fields
- [ ] Test human-readable format is clear and actionable

**Workflow Visualization:**
- [ ] Test `shark workflow list` displays all statuses
- [ ] Test transitions shown correctly (forward/backward/special)
- [ ] Test agent types displayed for each ready status
- [ ] Test start and terminal statuses highlighted
- [ ] Test `--json` output matches schema
- [ ] Test default workflow shown when config missing
- [ ] Test custom workflow parsed correctly

### Integration Tests

**End-to-End Command Discovery:**
- [ ] User runs `shark task --help` and finds `claim` in 1-2 seconds
- [ ] User runs wrong command, gets suggestion, successfully recovers
- [ ] User runs `shark workflow list`, understands transitions

**Documentation Consistency:**
- [ ] CLI_REFERENCE.md categories match help output
- [ ] CLAUDE.md examples use categorized commands
- [ ] All command descriptions consistent across help/docs

### User Acceptance Testing

**New User Onboarding:**
- [ ] 10 new users complete task workflow using only `--help`
- [ ] Track time to find correct command (target: <5s)
- [ ] Track error recovery success rate (target: >80%)

**Workflow Admin Testing:**
- [ ] 5 workflow admins visualize and verify custom workflows
- [ ] Track time to understand workflow (target: <2 minutes)
- [ ] Track misconfiguration detection rate (target: 100%)

## Security Considerations

**Information Disclosure:**
- Error messages should not leak sensitive workflow details to unauthorized users
- Workflow visualization should respect access controls (future consideration)

**Injection Prevention:**
- Task IDs in suggestions must be sanitized (prevent command injection)
- Workflow status names validated (no special characters in output)

---

*Last Updated:* 2026-01-11
*Dependencies:* F01, F04
*Related Documentation:* ../requirements.md (REQ-F-011, REQ-F-012, REQ-F-014)
