package test_test

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestNewIsolatedTestDB_ReturnsInitialisedDB verifies that NewIsolatedTestDB
// returns a working database with the full schema applied.
func TestNewIsolatedTestDB_ReturnsInitialisedDB(t *testing.T) {
	t.Parallel()

	database := test.NewIsolatedTestDB(t)
	if database == nil {
		t.Fatal("expected non-nil database")
	}

	if err := database.Ping(); err != nil {
		t.Fatalf("database ping failed: %v", err)
	}

	// Verify schema was applied by checking that the tasks table exists.
	ctx := context.Background()
	var tableName string
	err := database.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='tasks'",
	).Scan(&tableName)
	if err != nil {
		t.Fatalf("tasks table not found: %v", err)
	}
	if tableName != "tasks" {
		t.Errorf("expected table name 'tasks', got %q", tableName)
	}
}

// TestNewIsolatedTestDB_EachCallReturnsDistinctDB verifies that multiple calls
// to NewIsolatedTestDB return independent databases (writes to one are not
// visible in another).
func TestNewIsolatedTestDB_EachCallReturnsDistinctDB(t *testing.T) {
	t.Parallel()

	db1 := test.NewIsolatedTestDB(t)
	db2 := test.NewIsolatedTestDB(t)

	ctx := context.Background()

	// Insert an epic in db1.
	_, err := db1.ExecContext(ctx,
		"INSERT INTO epics (key, title, status, priority) VALUES ('E91', 'Isolation Test', 'active', 'high')",
	)
	if err != nil {
		t.Fatalf("db1 insert failed: %v", err)
	}

	// db2 must not see that epic.
	var count int
	if err := db2.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM epics WHERE key = 'E91'",
	).Scan(&count); err != nil {
		t.Fatalf("db2 query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected db2 to be empty, got count=%d", count)
	}
}

// TestNewIsolatedTestDB_ParallelSubtests demonstrates that isolated DBs enable
// parallel subtests without cross-contamination.
func TestNewIsolatedTestDB_ParallelSubtests(t *testing.T) {
	t.Parallel()

	epicKeys := []string{"E81", "E82", "E83", "E84"}

	for _, key := range epicKeys {
		key := key // capture range variable
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			database := test.NewIsolatedTestDB(t)
			ctx := context.Background()

			_, err := database.ExecContext(ctx,
				"INSERT INTO epics (key, title, status, priority) VALUES (?, 'Parallel Epic', 'active', 'high')",
				key,
			)
			if err != nil {
				t.Fatalf("insert failed for %s: %v", key, err)
			}

			var title string
			if err := database.QueryRowContext(ctx,
				"SELECT title FROM epics WHERE key = ?", key,
			).Scan(&title); err != nil {
				t.Fatalf("select failed for %s: %v", key, err)
			}
			if title != "Parallel Epic" {
				t.Errorf("expected 'Parallel Epic', got %q", title)
			}
		})
	}
}

// TestNewIsolatedTestDB_ConcurrentSubtests verifies that many parallel subtests
// can each create their own isolated database simultaneously without racing.
func TestNewIsolatedTestDB_ConcurrentSubtests(t *testing.T) {
	t.Parallel()

	const subtests = 8

	for i := 0; i < subtests; i++ {
		i := i // capture
		t.Run("subtest", func(t *testing.T) {
			t.Parallel()

			database := test.NewIsolatedTestDB(t)
			if database == nil {
				t.Errorf("subtest %d: expected non-nil database", i)
				return
			}
			if err := database.Ping(); err != nil {
				t.Errorf("subtest %d: ping failed: %v", i, err)
			}
		})
	}
}
