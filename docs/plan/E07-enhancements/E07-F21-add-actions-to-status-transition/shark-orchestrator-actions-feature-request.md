# Feature Request: Orchestrator Action Instructions in Shark Config

**Version:** 2.0
**Date:** 2026-01-13
**Requestor:** AI Agent Orchestrator Team
**Status:** Proposal

---

## Executive Summary

Add `orchestrator_action` metadata to the existing `.sharkconfig.json` `status_metadata` section. When tasks transition to new statuses, Shark returns the orchestrator action as part of the transition response, telling the orchestrator what to do next.

**Key Benefit:** Decouples orchestrator logic from workflow definitions. Workflow configuration becomes the single source of truth for both state management (Shark) and execution instructions (Orchestrator).

**Primary API:** Orchestrator actions returned automatically in `shark task update` responses - no separate query needed.

---

## Problem Statement

### Current State

The AI Agent Orchestrator currently has **hardcoded knowledge** about which agents to spawn for each workflow status:

```go
// Hardcoded in orchestrator
statusToAgent := map[string]string{
    "ready_for_refinement_ba": "business-analyst",
    "ready_for_refinement_tech": "architect",
    "ready_for_development": "developer",
    // ... etc
}
```

**Problems:**
1. Workflow knowledge is duplicated between Shark (state) and Orchestrator (execution logic)
2. Adding new workflow stages requires orchestrator code changes
3. Modifying agent instructions requires orchestrator rebuild
4. Cannot support different workflows for different projects easily
5. Workflow definition is split across multiple systems
6. Orchestrator must make separate queries to determine what to do after transitions

### Desired State

Shark configuration becomes the **single source of truth** for both:
- **Workflow state** (already exists: `status_flow`, `status_metadata`)
- **Execution instructions** (new: `orchestrator_action` in `status_metadata`)

**When a task status changes, Shark immediately tells the orchestrator what to do:**

```bash
# Agent or orchestrator updates task status
shark task update T-E01-F03-002 --status ready_for_development --json
```

**Shark responds with transition result AND orchestrator action:**
```json
{
  "success": true,
  "task_id": "T-E01-F03-002",
  "transition": {
    "from": "in_refinement_tech",
    "to": "ready_for_development",
    "timestamp": "2026-01-13T15:30:00Z"
  },
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation", "shark-task-management"],
    "instruction": "Launch a developer agent with test-driven-development skill to work on task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications."
  }
}
```

**Orchestrator receives this and immediately knows to spawn a developer agent - no additional query needed.**

---

## Proposed Schema Changes

### 1. Add `orchestrator_action` to `status_metadata`

**Location:** `.sharkconfig.json` → `status_metadata[status_name]`

**Schema:**
```typescript
interface OrchestratorAction {
  // Required: Type of action orchestrator should take
  action: "spawn_agent" | "pause" | "wait_for_triage" | "archive";

  // Optional: Agent type to spawn (required if action = "spawn_agent")
  agent_type?: string;

  // Optional: Skills to provide to agent
  skills?: string[];

  // Required: Instruction template with {task_id} placeholder
  instruction_template: string;
}
```

**Action Types:**

| Action | Description | Required Fields | Use Cases |
|--------|-------------|-----------------|-----------|
| `spawn_agent` | Launch an AI agent to work on task | `agent_type`, `skills`, `instruction_template` | ready_for_development, ready_for_qa, etc. |
| `pause` | Do not spawn agent, task is waiting | `instruction_template` | blocked, on_hold |
| `wait_for_triage` | Task needs human decision | `instruction_template` | draft |
| `archive` | Task is complete, no action | `instruction_template` | completed, cancelled |

### 2. Example: Updated `status_metadata` Entry

**Before:**
```json
{
  "ready_for_development": {
    "agent_types": ["developer", "ai-coder"],
    "color": "yellow",
    "description": "Spec complete, ready for implementation",
    "phase": "development"
  }
}
```

**After:**
```json
{
  "ready_for_development": {
    "agent_types": ["developer", "ai-coder"],
    "color": "yellow",
    "description": "Spec complete, ready for implementation",
    "phase": "development",
    "orchestrator_action": {
      "action": "spawn_agent",
      "agent_type": "developer",
      "skills": [
        "test-driven-development",
        "implementation",
        "shark-task-management"
      ],
      "instruction_template": "Launch a developer agent with test-driven-development skill to work on task {task_id}. Write tests first, then implement to pass tests following the technical specifications."
    }
  }
}
```

### 3. Backward Compatibility

**IMPORTANT:** `orchestrator_action` is **optional**.

- Existing Shark functionality is unchanged
- If `orchestrator_action` is missing, `shark task update` works normally without the field
- Shark CLI validation should warn but not fail if `orchestrator_action` is missing
- Orchestrator can fall back to defaults if action not provided

---

## Required Shark CLI Changes

### Primary API: Include in Status Transitions

#### 1. `shark task update` Enhancement (PRIMARY)

**Purpose:** Return orchestrator action as part of status transition response

**Current Behavior:**
```bash
shark task update T-E01-F03-002 --status ready_for_development --json
```

**Current Output:**
```json
{
  "success": true,
  "task_id": "T-E01-F03-002",
  "old_status": "in_refinement_tech",
  "new_status": "ready_for_development",
  "updated_at": "2026-01-13T15:30:00Z"
}
```

**New Behavior (with orchestrator_action):**
```bash
shark task update T-E01-F03-002 --status ready_for_development --json
```

**New Output:**
```json
{
  "success": true,
  "task_id": "T-E01-F03-002",
  "transition": {
    "from": "in_refinement_tech",
    "to": "ready_for_development",
    "timestamp": "2026-01-13T15:30:00Z"
  },
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": [
      "test-driven-development",
      "implementation",
      "shark-task-management"
    ],
    "instruction": "Launch a developer agent with test-driven-development skill to work on task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications."
  }
}
```

**Human-Readable Output:**
```bash
shark task update T-E01-F03-002 --status ready_for_development
```

**Output:**
```
✓ Task T-E01-F03-002 updated
  From: in_refinement_tech
  To: ready_for_development

Next Action: spawn_agent
  Agent: developer
  Skills: test-driven-development, implementation, shark-task-management
```

**Implementation Notes:**
- Look up `orchestrator_action` from `status_metadata` for the **new status**
- Populate `{task_id}` in `instruction_template`
- If `orchestrator_action` not defined for status, omit the field (backward compatible)
- Always include action in JSON output when defined
- Human-readable output shows action summary

#### 2. `shark task list` Enhancement

**Purpose:** Optionally include orchestrator actions in task listings

**Syntax:**
```bash
shark task list --status ready_for_development [--with-actions] [--json]
```

**Without `--with-actions` (existing behavior):**
```json
{
  "tasks": [
    {
      "task_id": "T-E01-F03-002",
      "status": "ready_for_development",
      "title": "Implement user authentication API",
      "priority": 1
    }
  ]
}
```

**With `--with-actions` (new):**
```json
{
  "tasks": [
    {
      "task_id": "T-E01-F03-002",
      "status": "ready_for_development",
      "title": "Implement user authentication API",
      "priority": 1,
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation", "shark-task-management"],
        "instruction": "Launch a developer agent with test-driven-development skill to work on task T-E01-F03-002..."
      }
    }
  ]
}
```

**Use Case:** Orchestrator polls for ready tasks and immediately knows what to do without additional queries.

### Secondary API: Direct Query (Utility)

#### 3. `shark config get-status-action` (SECONDARY/UTILITY)

**Purpose:** Query what action would be taken for a status without actually transitioning

**Syntax:**
```bash
shark config get-status-action <status> [--task TASK_ID] [--json]
```

**Examples:**
```bash
# Get action for status
shark config get-status-action ready_for_development

# Get action with task ID populated in template
shark config get-status-action ready_for_development --task T-E01-F03-002

# JSON output
shark config get-status-action ready_for_development --task T-E01-F03-002 --json
```

**Output (Human-Readable):**
```
Status: ready_for_development
Action: spawn_agent
Agent Type: developer
Skills: test-driven-development, implementation, shark-task-management
Instruction: Launch a developer agent with test-driven-development skill to work on task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications.
```

**Output (JSON):**
```json
{
  "status": "ready_for_development",
  "task_id": "T-E01-F03-002",
  "action": "spawn_agent",
  "agent_type": "developer",
  "skills": [
    "test-driven-development",
    "implementation",
    "shark-task-management"
  ],
  "instruction": "Launch a developer agent with test-driven-development skill to work on task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications."
}
```

**Use Cases:**
- **Debugging:** "What would happen if task moved to this status?"
- **Documentation:** "Show me the action for this workflow stage"
- **Planning:** "Which agent handles this status?"
- **Validation:** "Does this status have an action defined?"

**Error Handling:**
- If status not found: Exit 1, message "Status 'xyz' not found in config"
- If `orchestrator_action` missing: Exit 1, message "Status 'xyz' has no orchestrator_action defined"
- If action is `spawn_agent` but `agent_type` missing: Exit 1, validation error

#### 4. `shark workflow validate-actions`

**Purpose:** Validate that all "ready_for_*" statuses have orchestrator actions

**Syntax:**
```bash
shark workflow validate-actions [--strict]
```

**Behavior:**
- Checks all statuses in `status_metadata`
- Warns if "ready_for_*" statuses lack `orchestrator_action`
- With `--strict`: Fails if any actionable status lacks definition
- Validates `orchestrator_action` schema correctness

**Output:**
```
✓ ready_for_refinement_ba: has orchestrator_action (spawn_agent)
✓ ready_for_refinement_tech: has orchestrator_action (spawn_agent)
✓ ready_for_development: has orchestrator_action (spawn_agent)
✓ ready_for_code_review: has orchestrator_action (spawn_agent)
✓ ready_for_qa: has orchestrator_action (spawn_agent)
✓ ready_for_approval: has orchestrator_action (spawn_agent)
✓ blocked: has orchestrator_action (pause)
✓ on_hold: has orchestrator_action (pause)
✓ draft: has orchestrator_action (wait_for_triage)
✓ completed: has orchestrator_action (archive)
✓ cancelled: has orchestrator_action (archive)

All orchestrator actions validated successfully.
```

#### 5. `shark workflow show-actions`

**Purpose:** Display all orchestrator actions in workflow

**Syntax:**
```bash
shark workflow show-actions [--json]
```

**Output (Human-Readable):**
```
Orchestrator Actions for Workflow: wormwoodGM

Planning Phase:
  ready_for_refinement_ba → spawn_agent (business-analyst)
  ready_for_refinement_tech → spawn_agent (architect)

Development Phase:
  ready_for_development → spawn_agent (developer)

Review Phase:
  ready_for_code_review → spawn_agent (tech-lead)

QA Phase:
  ready_for_qa → spawn_agent (qa)

Approval Phase:
  ready_for_approval → spawn_agent (product-manager)

Special Actions:
  blocked → pause
  on_hold → pause
  draft → wait_for_triage
  completed → archive
  cancelled → archive
```

---

## Implementation Guidance

### Phase 1: Config Schema Support

**Goal:** Shark can parse and validate `orchestrator_action` field

**Tasks:**
1. Update config schema to include optional `orchestrator_action`
2. Add validation for `orchestrator_action` structure
3. Update config parser to load `orchestrator_action` data
4. Add unit tests for config parsing

**Acceptance Criteria:**
- Shark loads `.sharkconfig.json` with `orchestrator_action` without error
- Validation catches malformed `orchestrator_action` definitions
- Existing configs without `orchestrator_action` still work

### Phase 2: Enhance `shark task update` (PRIMARY)

**Goal:** Return orchestrator action in status transition responses

**Tasks:**
1. Modify `shark task update` to look up `orchestrator_action` from new status
2. Populate `{task_id}` template variable in instruction
3. Include `orchestrator_action` in JSON output
4. Add action summary to human-readable output
5. Handle missing `orchestrator_action` gracefully

**Acceptance Criteria:**
- JSON output includes `orchestrator_action` when defined
- Human-readable output shows action summary
- Works correctly when `orchestrator_action` not defined (backward compat)
- Template variables populated correctly

### Phase 3: CLI Utility Commands (SECONDARY)

**Goal:** Implement utility commands for querying and validating actions

**Tasks:**
1. Implement `shark config get-status-action`
   - Parse status from config
   - Populate `{task_id}` template variable
   - Output JSON and human-readable formats
2. Implement `shark workflow validate-actions`
   - Check all statuses have actions
   - Validate action schema
3. Implement `shark workflow show-actions`
   - List all actions grouped by phase
4. Optional: Enhance `shark task list --with-actions`

**Acceptance Criteria:**
- All commands work with valid config
- Error messages are clear and actionable
- JSON output is machine-parseable
- Commands integrate with existing shark CLI patterns

### Phase 4: Template Engine

**Goal:** Support dynamic template variables in `instruction_template`

**Initial Support:**
- `{task_id}` - replaced with actual task ID

**Future Support (not required for v1):**
- `{epic_id}` - epic identifier
- `{feature_id}` - feature identifier
- `{task_title}` - task title
- `{priority}` - task priority

**Implementation:**
```python
# Simple template replacement
def populate_template(template: str, task_id: str) -> str:
    return template.replace("{task_id}", task_id)
```

**Future:** Consider using standard template library (Jinja2, Go text/template, etc.)

### Phase 5: Documentation

**Goal:** Document new feature for users and orchestrator developers

**Tasks:**
1. Update Shark README with `orchestrator_action` section
2. Add examples to docs showing different action types
3. Document CLI commands in help text
4. Create migration guide for existing configs

---

## Use Cases

### Use Case 1: Agent Completes Work and Transitions Task

**Scenario:** Business analyst finishes requirements and transitions task to technical design

**Flow:**
```bash
# BA agent completes work
shark task update T-E01-F03-002 --status ready_for_refinement_tech --json
```

**Shark Response:**
```json
{
  "success": true,
  "task_id": "T-E01-F03-002",
  "transition": {
    "from": "in_refinement_ba",
    "to": "ready_for_refinement_tech"
  },
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "architect",
    "skills": ["architecture", "specification-writing", "shark-task-management"],
    "instruction": "Launch an architect agent with architecture skill to design technical solution for task T-E01-F03-002..."
  }
}
```

**Orchestrator receives this and immediately:**
1. Parses `orchestrator_action`
2. Spawns architect agent with specified skills
3. Provides instruction to agent
4. No additional queries needed!

**Benefit:** One API call handles both state transition and orchestrator instruction.

### Use Case 2: Orchestrator Polls for Ready Tasks

**Scenario:** Orchestrator polls every 30 seconds for tasks in "ready" statuses

**Flow:**
```bash
# Orchestrator queries for ready tasks
shark task list --status ready_for_development --with-actions --json
```

**Shark Response:**
```json
{
  "tasks": [
    {
      "task_id": "T-E01-F03-002",
      "status": "ready_for_development",
      "title": "Implement user authentication API",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation", "shark-task-management"],
        "instruction": "Launch a developer agent..."
      }
    },
    {
      "task_id": "T-E01-F04-001",
      "status": "ready_for_development",
      "title": "Add password reset flow",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation", "shark-task-management"],
        "instruction": "Launch a developer agent..."
      }
    }
  ]
}
```

**Orchestrator receives this and:**
1. Iterates through tasks
2. Checks available agent slots
3. Spawns developer agents for top-priority tasks
4. All info in one response!

**Benefit:** Batch query returns all needed information.

### Use Case 3: Debugging Workflow Configuration

**Scenario:** Developer adds new status but wants to verify orchestrator action

**Flow:**
```bash
# Check what action is defined for a status
shark config get-status-action ready_for_security_review
```

**Output:**
```
Status: ready_for_security_review
Action: spawn_agent
Agent Type: security-engineer
Skills: security, quality, shark-task-management
Instruction: Launch a security-engineer agent to perform security review for task {task_id}...
```

**Benefit:** Quick validation without needing to trigger an actual transition.

### Use Case 4: Workflow Validation

**Scenario:** Developer adds new status but forgets orchestrator action

**Flow:**
```bash
# Developer adds "ready_for_security_review" to config
vim .sharkconfig.json

# Validate config
shark workflow validate-actions --strict

# Output:
# ✗ ready_for_security_review: missing orchestrator_action
# Error: Validation failed. All actionable statuses must have orchestrator_action.

# Developer adds orchestrator_action and re-validates
shark workflow validate-actions --strict
# ✓ All orchestrator actions validated successfully.
```

**Benefit:** Catches configuration errors before deployment.

---

## Orchestrator Integration

### Simplified Orchestrator Flow

**Old Flow (Multiple API Calls):**
```
1. Poll for tasks: shark task list --status ready_for_development
2. For each task: shark config get-status-action ready_for_development --task T-001
3. Parse response
4. Spawn agent
```

**New Flow (Single API Call per Transition):**
```
1. Agent completes work: shark task update T-001 --status ready_for_code_review --json
2. Response includes orchestrator_action
3. Spawn agent immediately
```

**Or Polling (Single Batch Query):**
```
1. Poll: shark task list --status ready_for_development --with-actions --json
2. Response includes all tasks with actions
3. Spawn agents for selected tasks
```

### Orchestrator Code Example (Go)

**Handling Task Transitions:**
```go
type TaskUpdateResponse struct {
    Success    bool        `json:"success"`
    TaskID     string      `json:"task_id"`
    Transition Transition  `json:"transition"`
    Action     *OrchestratorAction `json:"orchestrator_action,omitempty"`
}

type OrchestratorAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"`
}

// Agent updates task status
func (a *Agent) CompleteWork(taskID string, newStatus string) error {
    cmd := exec.Command("shark", "task", "update", taskID, "--status", newStatus, "--json")
    output, err := cmd.Output()
    if err != nil {
        return err
    }

    var response TaskUpdateResponse
    json.Unmarshal(output, &response)

    // Check if orchestrator action provided
    if response.Action != nil {
        switch response.Action.Action {
        case "spawn_agent":
            // Spawn next agent immediately
            orchestrator.SpawnAgent(
                response.Action.AgentType,
                response.Action.Skills,
                response.Action.Instruction,
                taskID,
            )
        case "pause":
            log.Info("Task %s paused: %s", taskID, response.Action.Instruction)
        case "archive":
            log.Info("Task %s completed", taskID)
        }
    }

    return nil
}
```

**Polling for Ready Tasks:**
```go
// Orchestrator polls for ready tasks
func (o *Orchestrator) PollForReadyTasks() error {
    for _, status := range o.readyStatuses {
        cmd := exec.Command("shark", "task", "list",
            "--status", status,
            "--with-actions",
            "--json")
        output, err := cmd.Output()
        if err != nil {
            return err
        }

        var response TaskListResponse
        json.Unmarshal(output, &response)

        for _, task := range response.Tasks {
            if task.Action != nil && task.Action.Action == "spawn_agent" {
                // Check if we have capacity
                if o.CanSpawnAgent(task.Action.AgentType) {
                    o.SpawnAgent(
                        task.Action.AgentType,
                        task.Action.Skills,
                        task.Action.Instruction,
                        task.TaskID,
                    )
                }
            }
        }
    }
    return nil
}
```

---

## Testing Requirements

### Unit Tests

1. **Config Parsing:**
   - Test loading config with `orchestrator_action`
   - Test loading config without `orchestrator_action` (backward compat)
   - Test malformed `orchestrator_action` (validation)

2. **Template Population:**
   - Test `{task_id}` replacement
   - Test missing task_id (should work, template not populated)

3. **Task Update Response:**
   - Test `shark task update` includes action when defined
   - Test `shark task update` works without action (backward compat)
   - Test action populated correctly from new status metadata

4. **CLI Commands:**
   - Test `get-status-action` with valid status
   - Test `get-status-action` with invalid status
   - Test `validate-actions` with complete config
   - Test `validate-actions` with incomplete config
   - Test JSON output format

### Integration Tests

1. **Orchestrator Integration:**
   - Mock orchestrator updates task status
   - Verify response includes orchestrator_action
   - Test all action types (spawn_agent, pause, etc.)

2. **Task List with Actions:**
   - Query tasks with `--with-actions`
   - Verify actions included in response
   - Test without flag (backward compat)

3. **End-to-End Workflow:**
   - Create task in draft
   - Transition to ready_for_development
   - Verify action in response
   - Verify instruction includes task ID

---

## Example Configs

### Minimal Example (Single Status)

```json
{
  "status_metadata": {
    "ready_for_development": {
      "color": "yellow",
      "description": "Ready for implementation",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["implementation"],
        "instruction_template": "Implement task {task_id}."
      }
    }
  }
}
```

### Complete Example (All WormwoodGM Statuses)

See `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json` for full example with all 14 statuses configured.

---

## Migration Path

### Existing Shark Users

**Step 1:** Update Shark to version with `orchestrator_action` support

**Step 2:** Existing configs work unchanged (backward compatible)
- `shark task update` works as before
- JSON output may not include `orchestrator_action` if not defined

**Step 3:** Optionally add `orchestrator_action` to statuses:
```bash
# Validate current config
shark workflow validate-actions

# If warnings about missing actions, add them to config
vim .sharkconfig.json

# Re-validate
shark workflow validate-actions
```

### Orchestrator Integration

**Step 1:** Orchestrator updated to parse `orchestrator_action` from task update responses

**Step 2:** Orchestrator checks if `orchestrator_action` present:
- If present: Use it
- If missing: Fall back to hardcoded logic (temporary)

**Step 3:** Once all configs updated, remove hardcoded fallback

---

## Questions for Shark Developers

1. **Response Format:** Should `orchestrator_action` be:
   - ✅ Top-level field in task update response (as proposed)
   - Nested under `transition` object
   - Separate `next_action` field

2. **Task List Flag:** Should actions in list be:
   - ✅ Opt-in via `--with-actions` flag (as proposed)
   - Always included (could be verbose)
   - Separate command `shark task list-with-actions`

3. **Template Engine:** Should we:
   - ✅ Start simple with string replacement
   - Use existing template library (Jinja2, text/template)
   - Support complex expressions later

4. **Validation Strictness:** Should missing `orchestrator_action` be:
   - ✅ Warning by default, error with `--strict` flag
   - Error by default (breaking change)
   - Always just a warning (permissive)

5. **Action Types:** Are the proposed action types sufficient?
   - `spawn_agent`, `pause`, `wait_for_triage`, `archive`
   - Should we add `notify`, `webhook`, `custom`?

---

## Success Criteria

This feature is successful when:

1. ✅ `shark task update` returns orchestrator action in response
2. ✅ Orchestrator receives instruction immediately on transition (no extra query)
3. ✅ Workflow definitions are single-source-of-truth in Shark config
4. ✅ No orchestrator code changes needed to add workflow stages
5. ✅ Multiple workflows supported without code changes
6. ✅ Backward compatible with existing Shark configs
7. ✅ Clear error messages guide users to fix config issues
8. ✅ Reduced API calls: one transition = one response with action

---

## API Priority Summary

| API | Priority | Purpose | When Used |
|-----|----------|---------|-----------|
| `shark task update` response includes action | **PRIMARY** | Real-time instruction on state change | Every status transition |
| `shark task list --with-actions` | **SECONDARY** | Batch query for polling | Orchestrator polling |
| `shark config get-status-action` | **UTILITY** | Query action without transition | Debugging, validation |
| `shark workflow validate-actions` | **UTILITY** | Validate config completeness | Development, CI/CD |
| `shark workflow show-actions` | **UTILITY** | Documentation, overview | Development |

**Key Insight:** The primary integration is through task update responses. Other commands are utilities for specific use cases.

---

## References

- **AI Agent Orchestrator Design:** `/home/jwwelbor/.claude/docs/architecture/ai-agent-orchestrator-design.md`
- **Orchestration Approach Analysis:** `/home/jwwelbor/.claude/docs/architecture/orchestration-approach-analysis.md`
- **Example Config:** `/home/jwwelbor/projects/wormwoodGM/.sharkconfig.json`
- **Shark CLI Repository:** [Link to repo]

---

## Appendix A: Complete Status Action Mapping

| Status | Action | Agent Type | Skills |
|--------|--------|------------|--------|
| `ready_for_refinement_ba` | `spawn_agent` | business-analyst | specification-writing, shark-task-management |
| `ready_for_refinement_tech` | `spawn_agent` | architect | architecture, specification-writing, shark-task-management |
| `ready_for_development` | `spawn_agent` | developer | test-driven-development, implementation, shark-task-management |
| `ready_for_code_review` | `spawn_agent` | tech-lead | quality, shark-task-management |
| `ready_for_qa` | `spawn_agent` | qa | quality, shark-task-management |
| `ready_for_approval` | `spawn_agent` | product-manager | shark-task-management |
| `in_refinement_ba` | (none) | - | Agent already working |
| `in_refinement_tech` | (none) | - | Agent already working |
| `in_development` | (none) | - | Agent already working |
| `in_code_review` | (none) | - | Agent already working |
| `in_qa` | (none) | - | Agent already working |
| `in_approval` | (none) | - | Agent already working |
| `blocked` | `pause` | - | - |
| `on_hold` | `pause` | - | - |
| `draft` | `wait_for_triage` | - | - |
| `completed` | `archive` | - | - |
| `cancelled` | `archive` | - | - |

---

## Appendix B: JSON Schema Definition

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "OrchestratorAction": {
      "type": "object",
      "required": ["action", "instruction_template"],
      "properties": {
        "action": {
          "type": "string",
          "enum": ["spawn_agent", "pause", "wait_for_triage", "archive"]
        },
        "agent_type": {
          "type": "string",
          "description": "Required if action is spawn_agent"
        },
        "skills": {
          "type": "array",
          "items": { "type": "string" },
          "description": "List of skills for agent"
        },
        "instruction_template": {
          "type": "string",
          "description": "Template string with {task_id} placeholder"
        }
      },
      "if": {
        "properties": { "action": { "const": "spawn_agent" } }
      },
      "then": {
        "required": ["agent_type", "skills"]
      }
    },
    "TaskUpdateResponse": {
      "type": "object",
      "required": ["success", "task_id", "transition"],
      "properties": {
        "success": {
          "type": "boolean"
        },
        "task_id": {
          "type": "string"
        },
        "transition": {
          "type": "object",
          "properties": {
            "from": { "type": "string" },
            "to": { "type": "string" },
            "timestamp": { "type": "string", "format": "date-time" }
          }
        },
        "orchestrator_action": {
          "$ref": "#/definitions/OrchestratorAction"
        }
      }
    }
  }
}
```

---

**End of Document**

For questions or clarifications, contact: AI Agent Orchestrator Team
