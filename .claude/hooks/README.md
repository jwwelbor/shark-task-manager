# Claude Code Hooks - Workflow Automation System

## Overview

This directory contains hook scripts that automate the SDLC workflow by detecting events, managing state, and orchestrating agent handoffs.

Hooks enable the workflow system to:
- Automatically detect when artifacts are created
- Route control to the next appropriate agent
- Maintain workflow state across sessions
- Validate artifacts before proceeding
- Handle subgraph invocations and returns

## Hook Types

Claude Code supports several hook types based on event timing:

### 1. PostToolUse (After Tool Execution)
Fires immediately after a tool completes execution.

**Use Cases:**
- Detect artifact file creation (Write tool)
- Update workflow state when artifacts are produced
- Trigger next agent in the workflow chain

**Example: `artifact-watcher.py`**
```python
# Fires after Write tool creates a file
# Checks if file matches artifact pattern (D01-*, F01-*, etc.)
# Updates state.json with new artifact
# Signals next workflow step
```

### 2. Stop (Agent Session End)
Fires when an agent completes and stops execution.

**Use Cases:**
- Determine next agent based on workflow graph
- Hand off context to next agent
- Complete workflow if at terminal node

**Example: `workflow-router.py`**
```python
# Reads current workflow state
# Consults workflow CSV to find next_nodes
# Launches next agent with appropriate context
# Handles subgraph returns to parent
```

### 3. SessionStart (Agent Initialization)
Fires when an agent session begins.

**Use Cases:**
- Load workflow state and context
- Provide agent with current position in workflow
- Resume interrupted workflows

**Example: `context-loader.py`**
```python
# Loads state.json
# Identifies current workflow node
# Provides agent with relevant artifacts and history
```

### 4. PreToolUse (Before Tool Execution)
Fires before a tool executes.

**Use Cases:**
- Validate artifacts before proceeding
- Block workflow if prerequisites missing
- Enforce quality gates

**Example: `quality-gate.py`**
```python
# Checks if required artifacts exist
# Validates artifact completeness
# Prevents advancement if validation fails
```

### 5. SubagentStop (Subagent Completion)
Fires when a subagent (nested agent invocation) completes.

**Use Cases:**
- Return control to parent workflow
- Merge subgraph outputs into parent context
- Pop subgraph stack

**Example: `subgraph-complete.py`**
```python
# Pops from subgraph_stack in state.json
# Returns to parent_graph at return_to_node
# Provides subgraph outputs to parent
```

## Workflow State Integration

All hooks interact with the workflow state file:
```
docs/workflow/state.json
```

### State Schema
```json
{
  "current_workflow": {
    "graph_name": "PDLC",
    "current_node": "Ideation_Brainstorming",
    "current_agent": "ProductManager",
    "status": "active"
  },
  "workflow_context": { ... },
  "pending_artifacts": [ ... ],
  "completed_nodes": [ ... ],
  "subgraph_stack": [ ... ]
}
```

### Hook Workflow Pattern

```
1. Agent executes task
2. Agent uses Write tool to create artifact
3. PostToolUse hook (artifact-watcher.py) fires
   - Detects artifact creation
   - Updates state.json with artifact
   - Marks node as completed
4. Agent completes and stops
5. Stop hook (workflow-router.py) fires
   - Reads state.json
   - Consults workflow CSV for next_nodes
   - Launches next agent OR completes workflow
6. Next agent starts
7. SessionStart hook (context-loader.py) fires
   - Loads state.json
   - Provides context to new agent
8. Cycle repeats
```

## Hook Configuration

Hooks are registered in `.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": {
      "Write": [".claude/hooks/artifact-watcher.py"]
    },
    "Stop": [".claude/hooks/workflow-router.py"],
    "SessionStart": [".claude/hooks/context-loader.py"],
    "PreToolUse": {
      "Write": [".claude/hooks/quality-gate.py"]
    },
    "SubagentStop": [".claude/hooks/subgraph-complete.py"]
  }
}
```

## Artifact Pattern Matching

Hooks detect artifacts by filename patterns:

| Prefix | Phase | Description |
|--------|-------|-------------|
| D01-* | Discovery | Vision, research, user insights |
| F01-* | Feature | PRDs, stories, designs |
| T01-* | Technical | Architecture, specs, ADRs |
| DEV-* | Development | Code, tests, branches |
| R01-* | Release | Release notes, deployment plans |

## Writing New Hooks

### Template Structure
```python
#!/usr/bin/env python3
"""
Hook Name: artifact-watcher.py
Event: PostToolUse
Purpose: Detect artifact creation and update workflow state
"""

import json
import sys
from pathlib import Path

def main():
    # Hook receives event data via stdin
    event_data = json.load(sys.stdin)

    # Access tool output
    tool_name = event_data.get('tool_name')
    tool_result = event_data.get('result')

    # Read workflow state
    state_path = Path('docs/workflow/state.json')
    with open(state_path) as f:
        state = json.load(f)

    # Process event and update state
    # ...

    # Write updated state
    with open(state_path, 'w') as f:
        json.dump(state, f, indent=2)

    # Output to Claude (optional)
    print("State updated successfully")

if __name__ == '__main__':
    main()
```

### Best Practices
1. Always validate event data structure
2. Handle missing state gracefully
3. Use atomic file writes for state updates
4. Log errors for debugging
5. Keep hooks fast (< 500ms)
6. Avoid external dependencies when possible

## Debugging Hooks

### Enable Hook Logging
Set environment variable:
```bash
export CLAUDE_HOOK_DEBUG=1
```

### Check Hook Execution
Hooks output to stderr, visible in Claude Code debug logs.

### Test Hooks Manually
```bash
echo '{"tool_name": "Write", "result": {...}}' | python3 .claude/hooks/artifact-watcher.py
```

## Workflow Graphs Reference

Hook behavior is driven by CSV workflow definitions:
- `/docs/plan/E01-SDLC-Workflow/csv/01-pdlc.csv`
- `/docs/plan/E01-SDLC-Workflow/csv/02-feature-refinement.csv`
- And others...

Each CSV defines:
- `node_name`: Unique node identifier
- `agent_type`: Which agent executes this node
- `outputs`: Expected artifact filenames
- `hooks`: Hook triggers (e.g., `PostToolUse:Write→launch_researcher`)
- `next_nodes`: Where to route after completion

## Troubleshooting

### Hook Not Firing
- Check `.claude/settings.json` registration
- Verify hook script is executable
- Check event type matches hook type

### State Not Updating
- Verify state.json path is correct
- Check file permissions
- Ensure atomic write operations

### Wrong Agent Launched
- Verify workflow CSV next_nodes mapping
- Check current_node in state.json
- Validate subgraph_stack for nested workflows

## Future Enhancements

- Retry logic for failed nodes
- Parallel node execution support
- Workflow visualization from state
- State persistence to database
- Human approval gate integration
