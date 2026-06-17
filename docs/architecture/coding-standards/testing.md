# Testing Standards

Use these standards when adding or changing tests.

## Strategy by Layer

| Layer | Test With | Speed |
|-------|-----------|-------|
| Repository | Real isolated database with cleanup before each test | Slow integration |
| Service | Mocked interfaces or in-memory fakes | Fast unit |
| Handler/CLI | Mocked services | Fast unit |
| Model | Pure logic tests | Instant |

## Core Rules

**Rule**: Prefer table-driven tests when behavior has multiple cases.

```go
tests := []struct {
    name    string
    pattern string
    wantErr bool
}{
    {name: "valid", pattern: `^E(?P<number>\d{2})$`, wantErr: false},
    {name: "missing number", pattern: `^E\d{2}$`, wantErr: true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := ValidatePattern(tt.pattern, "epic")
        if tt.wantErr {
            assert.Error(t, err)
        } else {
            assert.NoError(t, err)
        }
    })
}
```

**Rule**: Use `testify/require` for setup and fatal assertions. Use `testify/assert` for non-fatal assertions.

```go
require.NoError(t, err, "setup must succeed")
assert.Equal(t, expected, actual)
```

**Rule**: Test behavior, not implementation details.

**Rule**: Every expected error path needs coverage.

**Rule**: Do not use `time.Sleep` in tests. Use channels, contexts, polling with deadlines, or injected clocks.

## Organization

**Rule**: Test files live in the same package as the code under test unless external-package testing is specifically needed.

```text
internal/patterns/validator.go
internal/patterns/validator_test.go
```

**Rule**: Use subtests with `t.Run()` for table-driven cases.

**Rule**: Tests must be deterministic and independent.

## Repository and Integration Tests

**Rule**: Repository tests use isolated test databases.

```go
func TestTaskRepository_Create(t *testing.T) {
    db := testdb.Setup(t)
    t.Cleanup(func() { testdb.Teardown(t, db) })

    repo := NewTaskRepository(db)
    // test
}
```

**Rule**: Clean up test data with `t.Cleanup`.

**Rule**: Do not use production database paths in tests.

## Coverage Expectations

- Public functions should have coverage.
- New behavior needs success and error-path tests.
- Boundary behavior should be tested where errors are mapped to CLI/API output.
- Repository tests should cover SQL constraints and scan behavior.

## Mock Pattern

Prefer small hand-written mocks or function-field fakes over broad mocking frameworks.

```go
type MockTaskReader struct {
    GetByKeyFunc func(ctx context.Context, key string) (*models.Task, error)
}

func (m *MockTaskReader) GetByKey(ctx context.Context, key string) (*models.Task, error) {
    if m.GetByKeyFunc == nil {
        panic("MockTaskReader.GetByKeyFunc: not implemented")
    }
    return m.GetByKeyFunc(ctx, key)
}
```

## Test Checklist

- [ ] Tests are deterministic.
- [ ] Setup errors use `require.NoError`.
- [ ] Error paths are covered.
- [ ] Table-driven tests use `t.Run`.
- [ ] Repository tests use isolated databases.
- [ ] Tests do not use `time.Sleep`.
- [ ] Cleanup happens through `t.Cleanup`.
