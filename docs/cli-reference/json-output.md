# JSON Output Format

All commands support `--json` flag for machine-readable output.

## Epic JSON Format

```json
{
  "id": 7,
  "key": "E07",
  "slug": "user-management-system",
  "title": "User Management System",
  "description": "Complete user management infrastructure",
  "file_path": "docs/plan/E07-user-management-system/epic.md",
  "priority": 1,
  "business_value": 10,
  "progress": 42.5,
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

## Feature JSON Format

```json
{
  "id": 1,
  "key": "E07-F01",
  "slug": "authentication",
  "epic_id": 7,
  "epic_key": "E07",
  "title": "Authentication",
  "description": "User authentication system",
  "file_path": "docs/plan/E07-user-management-system/E07-F01-authentication/feature.md",
  "execution_order": 1,
  "progress": 60.0,
  "task_count": 5,
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

## Task JSON Format

```json
{
  "id": 1,
  "key": "T-E07-F01-001",
  "slug": "implement-jwt-validation",
  "feature_id": 1,
  "epic_id": 7,
  "title": "Implement JWT validation",
  "description": "Add JWT token validation middleware",
  "status": "in_progress",
  "priority": 3,
  "agent_type": "backend",
  "depends_on": ["T-E07-F01-000"],
  "dependency_status": {
    "T-E07-F01-000": "completed"
  },
  "file_path": "docs/plan/E07-user-management-system/E07-F01-authentication/tasks/T-E07-F01-001.md",
  "created_at": "2026-01-02T10:00:00Z",
  "updated_at": "2026-01-02T15:30:00Z"
}
```

## Usage in Scripts

### Bash Example

```bash
#!/bin/bash

# Get task as JSON
task_json=$(shark task get E07-F01-001 --json)

# Parse with jq
task_key=$(echo "$task_json" | jq -r '.key')
task_status=$(echo "$task_json" | jq -r '.status')

echo "Task $task_key is $task_status"
```

### Python Example

```python
import json
import subprocess

# Run command with --json flag
result = subprocess.run(
    ["shark", "task", "list", "--status=todo", "--json"],
    capture_output=True,
    text=True
)

# Parse JSON response
tasks = json.loads(result.stdout)

# Process tasks
for task in tasks:
    print(f"Task {task['key']}: {task['title']}")
```

### Node.js Example

```javascript
const { execSync } = require('child_process');

// Run command and get JSON output
const output = execSync('shark task list --status=todo --json', {
  encoding: 'utf-8'
});

// Parse JSON
const tasks = JSON.parse(output);

// Process tasks
tasks.forEach(task => {
  console.log(`Task ${task.key}: ${task.title}`);
});
```

## Enhanced JSON Fields

See [JSON API Fields](json-api-fields.md) for documentation on enhanced fields like:
- Progress information
- Work summaries
- Action items
- Health indicators
- Rollup data

## Related Documentation

- [Best Practices](best-practices.md) - JSON output best practices
- [JSON API Fields](json-api-fields.md) - Enhanced JSON response fields
- [Orchestrator Actions](orchestrator-actions.md) - Orchestrator action metadata
