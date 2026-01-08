# Bug Tracker - Quick Reference Card

**One-page reference for the bug tracker feature**

---

## CLI Commands Cheat Sheet

### Create Bug
```bash
# Minimal
shark bug create "Bug title"

# Full
shark bug create "Bug title" \
  --severity=critical \
  --priority=9 \
  --category=backend \
  --steps="1. Do X\n2. Observe Y" \
  --expected="Should do Z" \
  --actual="Does W instead" \
  --error="Error: XYZ" \
  --environment=production \
  --os="Linux 6.6" \
  --version="v1.2.3" \
  --reporter=agent-id \
  --reporter-type=ai_agent \
  --epic=E07 \
  --feature=E07-F20 \
  --task=T-E07-F20-001 \
  --file=/tmp/stacktrace.txt
```

### List Bugs
```bash
shark bug list                                    # All active bugs
shark bug list --severity=critical                # Critical only
shark bug list --status=new --category=backend    # New backend bugs
shark bug list --epic=E07                         # Bugs in E07
shark bug list --environment=production           # Production bugs
shark bug list --json                             # JSON output
```

### Get Bug
```bash
shark bug get B-2026-01-04-01                     # Full details
shark bug get B-2026-01-04-01 --json              # JSON output
```

### Update Bug
```bash
shark bug update B-2026-01-04-01 --severity=high
shark bug update B-2026-01-04-01 --status=confirmed
shark bug update B-2026-01-04-01 --priority=8
```

### Status Management
```bash
shark bug confirm B-2026-01-04-01 --notes="Reproduced"
shark bug resolve B-2026-01-04-01 --resolution="Fixed in commit abc123"
shark bug close B-2026-01-04-01 --notes="Verified in prod"
shark bug reopen B-2026-01-04-01 --notes="Still occurs"
```

### Conversion
```bash
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20
shark bug convert feature B-2026-01-04-01 --epic=E07
shark bug convert epic B-2026-01-04-01
```

### Delete Bug
```bash
shark bug delete B-2026-01-04-01                  # Soft delete (archive)
shark bug delete B-2026-01-04-01 --hard --force   # Permanent delete
```

---

## Status Workflow

```
new → confirmed → in_progress → resolved → closed
 │         │            │           │
 │         └────────────┴───────────┴──→ wont_fix
 │                                       duplicate
 └─────────────────────────────────────────────→
```

---

## Severity Levels

| Severity | Description | Example |
|----------|-------------|---------|
| **critical** | System down, data loss, security breach | Production database offline |
| **high** | Major functionality broken | Login system fails |
| **medium** | Functionality impaired, workarounds exist | Search slow but works |
| **low** | Minor issue, cosmetic | Button misaligned |

---

## Priority Scale (1-10)

| Priority | Action |
|----------|--------|
| 9-10 | Drop everything, fix now |
| 7-8 | Fix in current sprint |
| 5-6 | Fix in next sprint |
| 3-4 | Fix when capacity allows |
| 1-2 | Backlog |

---

## Common Use Cases

### AI Agent Reports Bug During Test
```bash
shark bug create "Test failure: test_auth" \
  --severity=high \
  --category=backend \
  --error="AssertionError: Expected 401, got 500" \
  --reporter=test-agent \
  --reporter-type=ai_agent \
  --environment=test \
  --json
```

### Developer Converts Bug to Task
```bash
shark bug list --severity=critical
shark bug get B-2026-01-04-05
shark bug confirm B-2026-01-04-05
shark bug convert task B-2026-01-04-05 --epic=E07 --feature=E07-F20
shark task start T-E07-F20-025
```

### QA Verifies Fix
```bash
shark bug list --status=resolved
shark bug get B-2026-01-04-03
shark bug close B-2026-01-04-03 --notes="Verified in production"
```

### Report Bug with Large Stack Trace
```bash
# Save stack trace to file
python app.py 2> /tmp/crash.txt

# Create bug with attachment
shark bug create "App crashes on startup" \
  --severity=critical \
  --file=/tmp/crash.txt
```

---

## Filtering with jq

### Bugs by Severity
```bash
shark bug list --json | \
  jq 'group_by(.severity) | map({severity: .[0].severity, count: length})'
```

### Critical Bugs in Production
```bash
shark bug list --json | \
  jq '.[] | select(.severity == "critical" and .environment == "production")'
```

### AI Agent Reports
```bash
shark bug list --json | \
  jq '.[] | select(.reporter_type == "ai_agent")'
```

### Average Time to Resolution
```bash
shark bug list --status=resolved --json | \
  jq '[.[] | select(.resolved_at != null) |
      (((.resolved_at | fromdateiso8601) - (.detected_at | fromdateiso8601)) / 3600)] |
      add / length'
```

---

## Key Format

**Pattern**: `B-YYYY-MM-DD-xx`

**Examples**:
- `B-2026-01-04-01` - First bug on Jan 4, 2026
- `B-2026-01-04-15` - 15th bug on Jan 4, 2026
- `B-2026-12-31-99` - Max bugs per day

**Validation**: `^B-\d{4}-\d{2}-\d{2}-\d{2}$`

---

## Data Model Quick View

### Required Fields
- `key` (auto-generated)
- `title`
- `severity` (default: medium)
- `status` (default: new)
- `detected_at` (auto-set)
- `reporter_type` (default: human)

### Optional Fields
- `description`
- `priority` (1-10)
- `category`
- `steps_to_reproduce`
- `expected_behavior`
- `actual_behavior`
- `error_message`
- `environment`
- `os_info`
- `version`
- `reporter_id`
- `attachment_file`
- `related_docs` (JSON array)
- `related_to_epic`
- `related_to_feature`
- `related_to_task`
- `dependencies` (JSON array)
- `resolution`

---

## File Attachment Guidelines

### Use `--file` When:
- Stack trace > 500 characters
- Multiple log file references
- Screenshot references needed
- Performance profiling data

### Use Inline Flags When:
- Quick bug reports
- All details fit in CLI flags
- Simple reproduction steps
- Short error messages

---

## Integration with Tasks

### Link Bug to Task
```bash
shark bug create "Bug in task T-E07-F20-001" \
  --task=T-E07-F20-001
```

### Convert Bug to Task
```bash
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20
```

### Task Description Includes Bug Context
When converted, task description contains:
- Original bug description
- Steps to reproduce
- Expected/actual behavior
- Error message
- Bug key reference

---

## Testing Patterns

### Repository Test (Real DB)
```go
func TestBugRepository_Create(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewBugRepository(db)

    // Clean up first
    _, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B-TEST-%'")

    // Test creation
    bug := &models.Bug{...}
    err := repo.Create(ctx, bug)
    assert.NoError(t, err)

    // Cleanup
    defer database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", bug.ID)
}
```

### CLI Test (Mock)
```go
type MockBugRepository struct {
    CreateFunc func(ctx context.Context, bug *models.Bug) error
}

func TestBugCreateCommand(t *testing.T) {
    mockRepo := &MockBugRepository{
        CreateFunc: func(ctx context.Context, bug *models.Bug) error {
            bug.ID = 123
            return nil
        },
    }

    // Test command with mock
}
```

---

## Common Errors

### Invalid Key Format
```
Error: invalid bug key format "B-2026-1-4-01": must match B-YYYY-MM-DD-xx (e.g., B-2026-01-04-01)
```

### Invalid Severity
```
Error: invalid bug severity "urgent": must be one of critical, high, medium, low
```

### Already Converted
```
Error: bug B-2026-01-04-01 already converted to task T-E07-F20-015
```

### Missing Required Flags
```
Error: required flag(s) "epic", "feature" not set
```

---

## Performance Tips

### Indexing
Key indexes exist on:
- `key` (unique)
- `status`
- `severity`
- `priority`
- `detected_at`
- `category`
- `environment`
- `related_to_epic`, `related_to_feature`, `related_to_task`

### Query Optimization
- Use specific filters to reduce result set
- Leverage indexes with `--severity`, `--category`, `--status`
- Use `--json` for programmatic parsing (faster than table output)

---

## Comparison with Idea Tracker

| Feature | Idea Tracker | Bug Tracker |
|---------|--------------|-------------|
| **Purpose** | Feature ideation | Defect tracking |
| **Key Format** | I-YYYY-MM-DD-xx | B-YYYY-MM-DD-xx |
| **Statuses** | 4 (new, on_hold, converted, archived) | 7 (new, confirmed, in_progress, resolved, closed, wont_fix, duplicate) |
| **Technical Fields** | No | Yes (steps, expected/actual, error) |
| **Environment** | No | Yes (os, version, environment) |
| **Reporter** | No | Yes (human vs ai_agent) |
| **File Support** | No | Yes (`--file` flag) |

---

## Implementation Phases

1. **Week 1**: Core infrastructure (DB schema, model, repository)
2. **Week 2**: Basic CLI (create, list, get, update)
3. **Week 3**: Status management (confirm, resolve, close, reopen)
4. **Week 4**: Conversion (bug → task/feature/epic)
5. **Week 5**: Documentation and polish

---

**Full documentation**: See `bug-tracker-comprehensive-design.md`
