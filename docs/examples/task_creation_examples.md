# Task Creation Examples

This document provides practical examples and common patterns for creating tasks using the `pm task create` command.

## Basic Usage

### Minimum Required Fields

Create a task with only the required fields:

```bash
pm task create \
  --epic=E01 \
  --feature=F02 \
  --title="Build Login Component" \
  --agent=frontend
```

Output:
```
✓ Created task T-E01-F02-001: Build Login Component
File created at: docs/tasks/todo/T-E01-F02-001.md
Start work with: pm task start T-E01-F02-001
```

### Using Short Flags

The same command using short flag names:

```bash
pm task create -e E01 -f F02 -t "Build Login Component" -a frontend
```

## Common Patterns

### Backend API Task

Creating a backend task with description and priority:

```bash
pm task create \
  --epic=E01 \
  --feature=F03 \
  --title="Implement User Service" \
  --agent=backend \
  --description="Create CRUD operations for user management" \
  --priority=7
```

### Task with Dependencies

Creating a task that depends on another task:

```bash
pm task create \
  --epic=E01 \
  --feature=F02 \
  --title="Integrate Login with Auth Service" \
  --agent=frontend \
  --depends-on="T-E01-F03-001"
```

### Task with Multiple Dependencies

When a task depends on multiple other tasks:

```bash
pm task create \
  --epic=E02 \
  --feature=F05 \
  --title="End-to-End Integration Test" \
  --agent=testing \
  --depends-on="T-E02-F05-001,T-E02-F05-002,T-E02-F05-003"
```

Note: Dependencies are comma-separated without spaces.

### High Priority Task

Creating an urgent task with high priority:

```bash
pm task create \
  --epic=E01 \
  --feature=F01 \
  --title="Fix Critical Security Bug" \
  --agent=backend \
  --priority=10 \
  --description="Address SQL injection vulnerability in login endpoint"
```

Priority scale: 1-10, where 10 is highest priority.

## Agent-Specific Examples

### Frontend Development

```bash
pm task create \
  --epic=E03 \
  --feature=F02 \
  --title="Create Dashboard Component" \
  --agent=frontend \
  --description="Build responsive dashboard with charts and metrics" \
  --priority=5
```

The generated task file will include frontend-specific sections:
- Component Specifications
- UI/UX Requirements
- State Management
- Acceptance Criteria

### API Development

```bash
pm task create \
  --epic=E03 \
  --feature=F03 \
  --title="Implement User Registration Endpoint" \
  --agent=api \
  --description="POST /api/v1/users endpoint with validation" \
  --priority=8
```

The generated task file will include:
- API Specification
- Request/Response Schemas
- Authentication Requirements
- Error Responses

### Testing

```bash
pm task create \
  --epic=E03 \
  --feature=F04 \
  --title="Test User Authentication Flow" \
  --agent=testing \
  --description="Comprehensive tests for login, logout, and session management" \
  --priority=6
```

The generated task file will include:
- Test Scenarios
- Test Data Requirements
- Coverage Requirements
- Performance Benchmarks

### DevOps

```bash
pm task create \
  --epic=E04 \
  --feature=F01 \
  --title="Configure CI/CD Pipeline" \
  --agent=devops \
  --description="Set up automated testing and deployment pipeline" \
  --priority=9
```

The generated task file will include:
- Infrastructure Requirements
- Deployment Configuration
- Monitoring & Observability

### General Purpose

```bash
pm task create \
  --epic=E05 \
  --feature=F01 \
  --title="Research Database Options" \
  --agent=general \
  --description="Evaluate PostgreSQL vs MySQL for our use case" \
  --priority=4
```

The generated task file provides flexible structure for any task type.

## Feature Key Normalization

You can provide the feature key in either short or full form:

### Short Form

```bash
pm task create --epic=E01 --feature=F02 --title="My Task" --agent=backend
```

### Full Form

```bash
pm task create --epic=E01 --feature=E01-F02 --title="My Task" --agent=backend
```

Both commands produce identical results. The system automatically normalizes the feature key.

## JSON Output

Get machine-readable output for scripting:

```bash
pm task create \
  --epic=E01 \
  --feature=F02 \
  --title="Automated Task" \
  --agent=backend \
  --json
```

Output:
```json
{
  "id": 42,
  "feature_id": 5,
  "key": "T-E01-F02-005",
  "title": "Automated Task",
  "status": "todo",
  "agent_type": "backend",
  "priority": 5,
  "created_at": "2025-12-14T10:30:00Z",
  "file_path": "docs/tasks/todo/T-E01-F02-005.md"
}
```

## Common Workflows

### Creating Multiple Related Tasks

When breaking down a feature into tasks:

```bash
# Step 1: Database schema
pm task create -e E02 -f F01 -t "Design Database Schema" -a backend -p 8

# Step 2: API implementation (depends on schema)
pm task create -e E02 -f F01 -t "Implement API Endpoints" -a api -p 7 --depends-on="T-E02-F01-001"

# Step 3: Frontend integration (depends on API)
pm task create -e E02 -f F01 -t "Build Frontend Interface" -a frontend -p 6 --depends-on="T-E02-F01-002"

# Step 4: Testing (depends on all previous)
pm task create -e E02 -f F01 -t "Write Integration Tests" -a testing -p 5 --depends-on="T-E02-F01-001,T-E02-F01-002,T-E02-F01-003"
```

### Creating a Task from a Bug Report

```bash
pm task create \
  --epic=E99 \
  --feature=F01 \
  --title="Fix: Login Button Not Responsive on Mobile" \
  --agent=frontend \
  --description="Users report the login button doesn't respond to taps on iOS devices. Need to investigate touch event handling." \
  --priority=9
```

## Error Handling

### Invalid Epic

```bash
pm task create -e E99 -f F01 -t "Test" -a backend
```

Error:
```
✗ Failed to create task: epic E99 does not exist. Use 'pm epic list' to see available epics
```

### Invalid Agent Type

```bash
pm task create -e E01 -f F02 -t "Test" -a invalid-agent
```

Error:
```
✗ Failed to create task: invalid agent type 'invalid-agent'. Must be one of: frontend, backend, api, testing, devops, general
```

### Invalid Priority

```bash
pm task create -e E01 -f F02 -t "Test" -a backend -p 15
```

Error:
```
✗ Failed to create task: priority must be between 1 and 10, got 15
```

### Non-Existent Dependency

```bash
pm task create -e E01 -f F02 -t "Test" -a backend --depends-on="T-E01-F02-999"
```

Error:
```
✗ Failed to create task: dependency task T-E01-F02-999 does not exist
```

## Tips and Best Practices

1. **Use Descriptive Titles**: Make titles clear and action-oriented
   - Good: "Implement User Registration API"
   - Bad: "User stuff"

2. **Add Descriptions for Complex Tasks**: Provide context that will help the implementing agent
   ```bash
   --description="Implement password hashing using bcrypt with salt rounds=12. Include rate limiting to prevent brute force attacks."
   ```

3. **Set Appropriate Priorities**: Reserve high priorities (8-10) for critical/urgent work

4. **Model Dependencies Carefully**: Only add dependencies when truly required for the task to proceed

5. **Choose the Right Agent Type**: Select the agent type that best matches the primary skill needed

6. **Validate Before Creating**: Use `pm epic list` and `pm feature list` to verify epic and feature keys exist

## Next Steps

After creating a task:

1. **Review the generated file**: Check `docs/tasks/todo/T-XXX-XXX-XXX.md`
2. **Start working**: `pm task start T-E01-F02-001`
3. **Update progress**: `pm task complete T-E01-F02-001`
4. **Track status**: `pm task list --status=in_progress`

## See Also

- `pm task list` - List and filter tasks
- `pm task get <key>` - View task details
- `pm task start <key>` - Begin work on a task
- `pm epic list` - View available epics
- `pm feature list` - View available features
