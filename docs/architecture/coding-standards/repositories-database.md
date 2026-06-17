# Repository and Database Standards

Use these code-level standards when writing repository methods, SQL, migrations, or transaction-aware repository helpers.

For service/repository ownership boundaries, see [.claude/rules/architecture.md](../../../.claude/rules/architecture.md) and [.claude/rules/services/service-design.md](../../../.claude/rules/services/service-design.md).

## Repository Responsibilities

**Rule**: Repositories are leaf operations.

They own:

- CRUD operations.
- Parameterized queries.
- Row scanning.
- Persistence-specific error wrapping.
- Transaction-aware helper methods when called by higher layers.

They must not:

- Call other repositories.
- Enforce workflow or business rules.
- Format CLI/API output.
- Return `sql.Row`, `sql.Rows`, or transport DTOs.

## Repository Interfaces

Repository interfaces should be defined by consumers, usually services.

```go
type TaskStore interface {
    GetByKey(ctx context.Context, key string) (*models.Task, error)
    UpdateStatusTx(ctx context.Context, tx *sql.Tx, key string, status models.TaskStatus) error
}
```

## SQL

**Rule**: Use parameterized queries. Never concatenate user input into SQL.

```go
query := `SELECT * FROM tasks WHERE key = ?`
err := r.db.QueryRowContext(ctx, query, taskKey).Scan(...)
```

Avoid:

```go
query := fmt.Sprintf("SELECT * FROM tasks WHERE key = '%s'", taskKey)
```

**Rule**: Validate enums and structured identifiers before database operations.

## Transaction-Aware Repository Methods

**Rule**: Repositories participate through `*sql.Tx` methods.

```go
func (r *TaskRepository) UpdateStatusTx(
    ctx context.Context,
    tx *sql.Tx,
    key string,
    status models.TaskStatus,
) error {
    _, err := tx.ExecContext(ctx, updateTaskStatusSQL, status, key)
    if err != nil {
        return fmt.Errorf("update task status: %w", err)
    }
    return nil
}
```

Transaction ownership is an architecture rule, not a repository coding standard. In this project, multi-step transaction boundaries are owned by services.

## Resources

**Rule**: Close resources immediately with `defer` after successful acquisition.

```go
rows, err := r.db.QueryContext(ctx, query, args...)
if err != nil {
    return nil, fmt.Errorf("query tasks: %w", err)
}
defer rows.Close()
```

**Rule**: Check row iteration errors.

```go
for rows.Next() {
    // scan
}
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("iterate tasks: %w", err)
}
```

## SQLite/libsql

**Rules**:

- Use WAL mode for local SQLite where appropriate.
- Configure connection pooling intentionally.
- Prefer prepared statements for repeated queries.
- Keep schema constraints close to the data invariants they protect.
- Do not rely on test ordering or shared database state.

## Repository Checklist

- [ ] Accept `context.Context` as the first parameter.
- [ ] Use parameterized SQL.
- [ ] Return domain models or domain-friendly values.
- [ ] Wrap errors with context.
- [ ] Keep business rules out of repositories.
- [ ] Add `Tx` variants when a repository method must participate in a higher-level transaction.
- [ ] Close rows and check `rows.Err()`.
- [ ] Add integration tests with isolated test databases.
