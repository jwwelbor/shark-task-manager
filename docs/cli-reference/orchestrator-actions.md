# Task Update API Response Format

Enhanced task update API response format that includes `orchestrator_action` metadata for AI Agent Orchestrators.

## Overview

When tasks transition status (via `shark task start`, `shark task complete`, `shark task approve`, etc.), the API response includes optional `orchestrator_action` metadata. This metadata tells orchestrators what action to take next and which agent should be spawned (if applicable).

**Key Features:**
- Atomic response: Task state + action metadata in single API call
- Backward compatible: Missing actions don't break existing code
- Flexible: Actions are defined per-status in configuration
- Optional: Clients can safely ignore missing actions

## JSON Response Structure

### Complete Task Response with Orchestrator Action

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "slug": "implement-feature-x",
  "feature_id": 45,
  "epic_id": 7,
  "title": "Implement feature X",
  "description": "Detailed task description",
  "status": "ready_for_development",
  "priority": 5,
  "agent_type": "developer",
  "depends_on": ["T-E01-F03-001"],
  "file_path": "docs/plan/E01/E01-F03/tasks/T-E01-F03-002.md",
  "created_at": "2026-01-15T10:00:00Z",
  "updated_at": "2026-01-15T12:30:00Z",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation", "shark-task-management"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002. Write tests first, then implement to pass tests following the technical specifications."
  }
}
```

### Response Without Orchestrator Action

When no action is defined for a status, the `orchestrator_action` field is **omitted entirely** (not null):

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X"
  // NO orchestrator_action field
}
```

## OrchestratorAction Object Schema

The `orchestrator_action` object contains metadata to guide orchestrator behavior:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `action` | String | Yes | Action type: `spawn_agent`, `pause`, `wait_for_triage`, or `archive` |
| `agent_type` | String | Conditional | Required if `action=spawn_agent`. Type of agent to spawn (e.g., `developer`, `architect`, `reviewer`) |
| `skills` | String Array | Conditional | Required if `action=spawn_agent`. Array of skills the agent should have (e.g., `["test-driven-development", "implementation"]`) |
| `instruction` | String | Conditional | Required if `action=spawn_agent` or `pause`. Human-readable instruction template with template variables populated (e.g., `{task_id}` replaced with actual task key) |

## Action Types

### spawn_agent

Launch a new agent to work on the task.

**Required Fields:**
- `action`: `"spawn_agent"`
- `agent_type`: Agent role/type
- `skills`: Array of required skills
- `instruction`: Detailed instruction for the agent

**Example:**
```json
{
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation", "shark-task-management"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002. Write tests first, then implement to pass tests. Reference the technical specifications in the task file."
  }
}
```

**Common agent_type values:**
- `developer`: Backend/general development
- `frontend-developer`: Frontend-specific work
- `architect`: System design and architecture
- `reviewer`: Code review and quality assurance
- `test-engineer`: Testing and QA
- `devops`: Infrastructure and deployment

### pause

Wait before taking action (e.g., waiting for external dependencies).

**Required Fields:**
- `action`: `"pause"`
- `instruction`: Reason for pause and what to wait for

**Example:**
```json
{
  "orchestrator_action": {
    "action": "pause",
    "instruction": "Task T-E01-F03-002 is blocked waiting for API specification from architect. Check back when ready_for_development status is reached."
  }
}
```

### wait_for_triage

Task requires manual review and assignment before proceeding.

**Required Fields:**
- `action`: `"wait_for_triage"`
- `instruction`: Triage instructions

**Example:**
```json
{
  "orchestrator_action": {
    "action": "wait_for_triage",
    "instruction": "Task T-E01-F03-002 requires manual triage to assign to appropriate team. Review dependencies and priority before assigning."
  }
}
```

### archive

Task is complete and should be archived/ignored.

**Required Fields:**
- `action`: `"archive"`
- `instruction`: Reason for archival

**Example:**
```json
{
  "orchestrator_action": {
    "action": "archive",
    "instruction": "Task T-E01-F03-002 is completed and archived. No further action needed."
  }
}
```

## Template Variables

The `instruction` field may contain template variables that are automatically populated at runtime:

| Variable | Description | Example |
|----------|-------------|---------|
| `{task_id}` | Task key (normalized) | `T-E01-F03-002` or `E01-F03-002` |

**Example Template:**
```
"instruction": "Launch a developer agent to implement task {task_id}. Follow test-driven development practices."
```

**After Substitution:**
```
"instruction": "Launch a developer agent to implement task T-E01-F03-002. Follow test-driven development practices."
```

## Response Examples by Command

### Task Start Command
```bash
$ shark task start E01-F03-002 --json
```

Response (task transitions to `in_progress`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation"],
    "instruction": "Launch a developer agent to implement task T-E01-F03-002..."
  }
}
```

### Task Complete Command
```bash
$ shark task complete E01-F03-002 --json
```

Response (task transitions to `ready_for_review`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_review",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "reviewer",
    "skills": ["code-review", "quality-assurance"],
    "instruction": "Launch a code reviewer agent to review task T-E01-F03-002..."
  }
}
```

### Task Approve Command
```bash
$ shark task approve E01-F03-002 --json
```

Response (task transitions to `completed`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "completed",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "archive",
    "instruction": "Task T-E01-F03-002 is completed. Archive and prepare final report."
  }
}
```

### Task Block Command
```bash
$ shark task block E01-F03-002 --reason="Waiting for API design" --json
```

Response (task transitions to `blocked`):
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "blocked",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "pause",
    "instruction": "Task T-E01-F03-002 is blocked: Waiting for API design. Check back when unblocked."
  }
}
```

## Integration Guide

### Parsing Orchestrator Action in Go

```go
// Define response structure
type TaskResponse struct {
    ID                   int64                `json:"id"`
    Key                  string               `json:"key"`
    Status               string               `json:"status"`
    Title                string               `json:"title"`
    OrchestratorAction   *OrchestratorAction  `json:"orchestrator_action,omitempty"`
}

type OrchestratorAction struct {
    Action      string   `json:"action"`
    AgentType   string   `json:"agent_type,omitempty"`
    Skills      []string `json:"skills,omitempty"`
    Instruction string   `json:"instruction"`
}

// Parse response
func parseTaskResponse(jsonData []byte) (*TaskResponse, error) {
    var task TaskResponse
    if err := json.Unmarshal(jsonData, &task); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    return &task, nil
}

// Handle orchestrator action
func handleTaskResponse(task *TaskResponse) error {
    // Action is optional - check presence before accessing
    if task.OrchestratorAction == nil {
        log.Printf("Task %s updated, no action defined", task.Key)
        return nil
    }

    action := task.OrchestratorAction
    switch action.Action {
    case "spawn_agent":
        return spawnAgent(task.Key, action.AgentType, action.Skills, action.Instruction)
    case "pause":
        log.Printf("Pausing: %s", action.Instruction)
        return nil
    case "wait_for_triage":
        log.Printf("Waiting for triage: %s", action.Instruction)
        return nil
    case "archive":
        return archiveTask(task.Key)
    default:
        return fmt.Errorf("unknown action type: %s", action.Action)
    }
}

// Spawn agent based on action
func spawnAgent(taskID, agentType string, skills []string, instruction string) error {
    agent := &Agent{
        Type:        agentType,
        Skills:      skills,
        TaskID:      taskID,
        Instruction: instruction,
    }
    return orchestrator.Spawn(agent)
}
```

### Parsing in Python

```python
import json
import logging

def parse_task_response(json_data):
    """Parse task response from shark CLI."""
    task = json.loads(json_data)
    return task

def handle_task_response(task):
    """Process orchestrator action from task response."""
    action = task.get('orchestrator_action')

    # Action is optional
    if action is None:
        logging.info(f"Task {task['key']} updated, no action defined")
        return

    action_type = action['action']

    if action_type == 'spawn_agent':
        spawn_agent(
            task_id=task['key'],
            agent_type=action['agent_type'],
            skills=action['skills'],
            instruction=action['instruction']
        )
    elif action_type == 'pause':
        logging.info(f"Pausing: {action['instruction']}")
    elif action_type == 'wait_for_triage':
        logging.info(f"Waiting for triage: {action['instruction']}")
    elif action_type == 'archive':
        archive_task(task['key'])
    else:
        raise ValueError(f"Unknown action type: {action_type}")

def spawn_agent(task_id, agent_type, skills, instruction):
    """Spawn an agent to work on the task."""
    # Implementation depends on your orchestrator
    pass
```

## Backward Compatibility

**Guarantee**: The `orchestrator_action` field is optional and omitted when not defined. Existing code continues to work without changes.

**Migration Path** (if you have existing orchestrators):

1. **No changes required** - Code ignoring `orchestrator_action` continues working
2. **Optional enhancement** - Add handling for `orchestrator_action` when available
3. **Gradual rollout** - Enable actions per-status in configuration as needed

**Before (no actions):**
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_development",
  "title": "Implement feature X"
}
```

**After (with actions):**
```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "ready_for_development",
  "title": "Implement feature X",
  "orchestrator_action": {
    "action": "spawn_agent",
    "agent_type": "developer",
    "skills": ["test-driven-development", "implementation"],
    "instruction": "Launch a developer agent..."
  }
}
```

Existing orchestrators ignore the new field. New orchestrators can opt-in to using it.

## Error Handling

### Missing Actions

Missing actions are **not errors**. The `orchestrator_action` field is simply omitted from the response:

```json
{
  "id": 123,
  "key": "T-E01-F03-002",
  "status": "in_progress",
  "title": "Implement feature X"
  // No orchestrator_action field
}
```

**Recommended handling:**
```go
if action := task.OrchestratorAction; action != nil {
    // Handle action
} else {
    // No action defined - orchestrator should have fallback logic
    log.Warn("No action defined for task", "task_id", task.Key)
}
```

### Invalid Actions

Invalid actions are caught at configuration load time (not at response time). The CLI will fail to start with helpful error messages.

### Template Variable Errors

If template variable population fails, the template is returned as-is (with unpopulated variables). Log a warning and continue:

```bash
# If {task_id} fails to populate:
"instruction": "Launch agent for task {task_id}"  # Unpopulated placeholder

# Orchestrator should handle gracefully:
if strings.Contains(instruction, "{") {
    log.Warn("Template not fully populated", "instruction", instruction)
}
```

## Debugging

### View Orchestrator Action for a Task

Get the current action for a task without modifying status:

```bash
# Get task details (no status change)
shark task get E01-F03-002 --json

# Check the orchestrator_action field (if present)
shark task get E01-F03-002 --json | jq '.orchestrator_action'
```

### View Actions in Task List

Retrieve multiple ready tasks with their actions:

```bash
# List tasks in ready_for_development status WITH actions
shark task list --status=ready_for_development --with-actions --json

# View actions for specific epic/feature
shark task list E01 F03 --with-actions --json
```

## Configuration

Orchestrator actions are defined in the `.sharkconfig.json` workflow configuration:

```json
{
  "status_metadata": {
    "ready_for_development": {
      "color": "yellow",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation", "shark-task-management"],
        "instruction_template": "Launch a developer agent to implement task {task_id}. Write tests first, then implement to pass tests following the technical specifications."
      }
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "reviewer",
        "skills": ["code-review", "quality-assurance"],
        "instruction_template": "Launch a reviewer agent to review task {task_id}. Check code quality, tests, and compliance with specifications."
      }
    }
  }
}
```

## Related Documentation

- [Task Commands](task-commands-full.md) - Task status transition commands
- [JSON API Fields](json-api-fields.md) - Other enhanced API fields
- [Workflow Configuration](workflow-config.md) - Configure actions per status
- [Best Practices](best-practices.md) - AI agent best practices
