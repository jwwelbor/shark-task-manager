# Error Handling

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 4 — Behavior Analysis

## Error Propagation Chain

```
Database → Repository → Service → CLI Command → User
   SQL error    wrapped      business     formatted      displayed
                context      context      as JSON/text   to terminal
```

## Exit Codes

| Code | Meaning | When |
|------|---------|------|
| 0 | Success | Operation completed |
| 1 | Not found / general error | Entity doesn't exist, or Cobra error |
| 2 | Database error | Connection failure, query error |
| 3 | Invalid state | Invalid status transition, business rule violation |
| 4 | Field not found | `--field` flag references non-existent field |

Source: `cmd/shark/main.go`, `internal/cli/commands/errors.go`

## Error Types

### Custom Error Types

| Type | Package | Used For |
|------|---------|----------|
| `FieldNotFoundError` | `internal/cli/` | `--field` extraction failure (exit code 4) |
| `CLIError` | `internal/cli/` | Structured JSON error output |

### Error Wrapping Pattern

Each layer adds context:
```go
// Repository: technical context
"failed to get task: sql: no rows in result set"

// Service: business context
"failed to start task E07-F01-001: failed to get task: sql: no rows"

// Command: user-facing context
"Error: unable to start task: failed to start task E07-F01-001: ..."
```

### Workflow Errors

Status transition failures return descriptive errors:
- `"invalid transition from 'todo' to 'completed'"`
- `"rejection reason required for backward transition"`
- `"task dependencies not met: E07-F01-001 not completed"`

## Error Response Formats

### CLI (Text Mode)
```
Error: task not found: E07-F01-999
```

### CLI (JSON Mode)
```json
{
  "error": {
    "code": "not_found",
    "message": "task not found: E07-F01-999"
  }
}
```

Source: `internal/cli/root.go` — `ErrorJSON()` function

## Validation Layers

| Layer | Validates | Example |
|-------|-----------|---------|
| **Cobra** | Argument count, flag types | `cobra.ExactArgs(1)` |
| **Models** | Structural validity | Title non-empty, priority 1-10 |
| **Services** | Business rules | Valid status transitions, dependencies met |
| **Database** | Constraints | UNIQUE, CHECK, FOREIGN KEY |

## Recovery Patterns

### Transaction Rollback
Services that manage transactions use `defer tx.Rollback()`:
```go
tx, err := s.db.BeginTx(ctx, nil)
if err != nil { return err }
defer tx.Rollback() // Automatic rollback if not committed

// ... operations ...

return tx.Commit()
```

### Graceful Degradation
Optional service dependencies (note repo, creator service) are nil-checked:
```go
if s.noteRepo != nil && reason != "" {
    _ = s.noteRepo.CreateRejectionNote(ctx, ...)
}
```

### Database Integrity
- `CheckIntegrity()` function runs `PRAGMA integrity_check`
- Foreign keys prevent orphaned records
- `ON DELETE CASCADE` maintains referential integrity
- WAL mode prevents corruption from concurrent access

## Logging Patterns

- **Verbose mode** (`--verbose`): Debug logging to stderr
- **No structured logging framework**: Uses `fmt.Fprintf(os.Stderr, ...)`
- **Audit logging**: Status changes recorded in `task_history` table
- **No log files**: Output goes to terminal only

---

See also: [Business Logic](business-logic.md) | [Decision Logic](decision-logic.md) | [Security Patterns](../analysis/security-patterns.md)
