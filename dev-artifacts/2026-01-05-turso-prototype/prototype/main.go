package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	fmt.Println("=== Turso Prototype - Proof of Concept ===")
	fmt.Println()

	// Load credentials from environment
	dbURL := os.Getenv("TURSO_DATABASE_URL")
	authToken := os.Getenv("TURSO_AUTH_TOKEN")

	if dbURL == "" || authToken == "" {
		fmt.Println("❌ Missing environment variables")
		fmt.Println("   Please set TURSO_DATABASE_URL and TURSO_AUTH_TOKEN")
		fmt.Println()
		fmt.Println("   Run the setup script:")
		fmt.Println("   ./scripts/setup-turso.sh")
		fmt.Println()
		fmt.Println("   Then source the .env file:")
		fmt.Println("   source prototype/.env")
		os.Exit(1)
	}

	// Phase 1: Basic Connection
	fmt.Println("Phase 1: Testing Basic Connection")
	fmt.Println("----------------------------------")
	db, err := testConnection(dbURL, authToken)
	if err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println()

	// Phase 2: CRUD Operations
	fmt.Println("Phase 2: Testing CRUD Operations")
	fmt.Println("----------------------------------")
	if err := testCRUD(db); err != nil {
		fmt.Printf("❌ CRUD test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	// Phase 3: Performance Benchmarks
	fmt.Println("Phase 3: Performance Benchmarks")
	fmt.Println("----------------------------------")
	if err := testPerformance(db); err != nil {
		fmt.Printf("❌ Performance test failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()

	// Phase 4: Edge Cases
	fmt.Println("Phase 4: Testing Edge Cases")
	fmt.Println("----------------------------------")
	if err := testEdgeCases(db); err != nil {
		fmt.Printf("⚠️  Some edge cases failed: %v\n", err)
		// Don't exit, edge cases are informational
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Prototype Summary ===")
	fmt.Println("✅ All core functionality works!")
	fmt.Println()
	fmt.Println("Next Steps:")
	fmt.Println("  1. Review findings in prototype/test_results.txt")
	fmt.Println("  2. Update dev-artifacts/2026-01-05-turso-prototype/README.md")
	fmt.Println("  3. Decide on E13 implementation approach")
}

// testConnection tests basic connection to Turso
func testConnection(dbURL, authToken string) (*sql.DB, error) {
	fmt.Printf("  Connecting to: %s\n", dbURL)

	// Build connection string with auth token
	connStr := fmt.Sprintf("%s?authToken=%s", dbURL, authToken)

	start := time.Now()
	db, err := sql.Open("libsql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("  ✅ Connected successfully in %v\n", elapsed)

	// Get database stats
	stats := db.Stats()
	fmt.Printf("  📊 Open connections: %d\n", stats.OpenConnections)

	return db, nil
}

// testCRUD tests Create, Read, Update, Delete operations
func testCRUD(db *sql.DB) error {
	// Create test table
	fmt.Println("  Creating test table...")
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS test_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'todo',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	fmt.Println("  ✅ Table created")

	// INSERT
	fmt.Println("  Testing INSERT...")
	testKey := fmt.Sprintf("TEST-%d", time.Now().Unix())
	result, err := db.Exec(
		"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
		testKey, "Test Task", "todo",
	)
	if err != nil {
		return fmt.Errorf("failed to insert: %w", err)
	}
	insertID, _ := result.LastInsertId()
	fmt.Printf("  ✅ Inserted record with ID: %d\n", insertID)

	// SELECT
	fmt.Println("  Testing SELECT...")
	var id int64
	var key, title, status string
	err = db.QueryRow(
		"SELECT id, key, title, status FROM test_tasks WHERE key = ?",
		testKey,
	).Scan(&id, &key, &title, &status)
	if err != nil {
		return fmt.Errorf("failed to select: %w", err)
	}
	fmt.Printf("  ✅ Retrieved: ID=%d, Key=%s, Title=%s, Status=%s\n", id, key, title, status)

	// UPDATE
	fmt.Println("  Testing UPDATE...")
	_, err = db.Exec(
		"UPDATE test_tasks SET status = ? WHERE key = ?",
		"in_progress", testKey,
	)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	// Verify update
	err = db.QueryRow("SELECT status FROM test_tasks WHERE key = ?", testKey).Scan(&status)
	if err != nil {
		return fmt.Errorf("failed to verify update: %w", err)
	}
	fmt.Printf("  ✅ Updated status to: %s\n", status)

	// DELETE
	fmt.Println("  Testing DELETE...")
	_, err = db.Exec("DELETE FROM test_tasks WHERE key = ?", testKey)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	// Verify deletion
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_tasks WHERE key = ?", testKey).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify deletion: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("delete failed, record still exists")
	}
	fmt.Println("  ✅ Deleted successfully")

	return nil
}

// testPerformance runs performance benchmarks
func testPerformance(db *sql.DB) error {
	// Single insert latency
	fmt.Println("  Testing single insert latency...")
	start := time.Now()
	testKey := fmt.Sprintf("PERF-%d", time.Now().UnixNano())
	_, err := db.Exec(
		"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
		testKey, "Performance Test", "todo",
	)
	if err != nil {
		return fmt.Errorf("insert failed: %w", err)
	}
	singleInsertLatency := time.Since(start)
	fmt.Printf("  ✅ Single insert: %v\n", singleInsertLatency)

	// Single read latency
	fmt.Println("  Testing single read latency...")
	start = time.Now()
	var id int64
	err = db.QueryRow("SELECT id FROM test_tasks WHERE key = ?", testKey).Scan(&id)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	singleReadLatency := time.Since(start)
	fmt.Printf("  ✅ Single read: %v\n", singleReadLatency)

	// // Batch insert (1,000 records)
	// fmt.Println("  Testing batch insert (1,000 records)...")
	// start = time.Now()
	// tx, err := db.Begin()
	// if err != nil {
	// 	return fmt.Errorf("failed to begin transaction: %w", err)
	// }

	// stmt, err := tx.Prepare("INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)")
	// if err != nil {
	// 	tx.Rollback()
	// 	return fmt.Errorf("failed to prepare statement: %w", err)
	// }
	// defer stmt.Close()

	// for i := 0; i < 1000; i++ {
	// 	key := fmt.Sprintf("BATCH-%d-%d", time.Now().UnixNano(), i)
	// 	_, err := stmt.Exec(key, fmt.Sprintf("Batch Task %d", i), "todo")
	// 	if err != nil {
	// 		tx.Rollback()
	// 		return fmt.Errorf("batch insert failed at %d: %w", i, err)
	// 	}
	// }

	// if err := tx.Commit(); err != nil {
	// 	return fmt.Errorf("failed to commit transaction: %w", err)
	// }
	// batchInsertLatency := time.Since(start)
	// fmt.Printf("  ✅ Batch insert (1,000): %v (%.2f ms per record)\n",
	// 	batchInsertLatency, float64(batchInsertLatency.Milliseconds())/1000.0)

	// // Query latency with WHERE clause
	// fmt.Println("  Testing query with WHERE clause...")
	// start = time.Now()
	// rows, err := db.Query("SELECT id, key, title FROM test_tasks WHERE status = ? LIMIT 10", "todo")
	// if err != nil {
	// 	return fmt.Errorf("query failed: %w", err)
	// }
	// defer rows.Close()

	// count := 0
	// for rows.Next() {
	// 	var id int64
	// 	var key, title string
	// 	if err := rows.Scan(&id, &key, &title); err != nil {
	// 		return fmt.Errorf("scan failed: %w", err)
	// 	}
	// 	count++
	// }
	// queryLatency := time.Since(start)
	// fmt.Printf("  ✅ Query (WHERE + LIMIT): %v (%d rows)\n", queryLatency, count)

	// Performance summary
	fmt.Println()
	fmt.Println("  📊 Performance Summary:")
	fmt.Printf("    Single Insert: %v\n", singleInsertLatency)
	fmt.Printf("    Single Read:   %v\n", singleReadLatency)

	// Cleanup
	_, _ = db.Exec("DELETE FROM test_tasks WHERE key LIKE 'PERF-%' OR key LIKE 'BATCH-%'")

	return nil
}

// testEdgeCases tests edge cases and error handling
func testEdgeCases(db *sql.DB) error {
	fmt.Println("  Testing duplicate key constraint...")
	testKey := fmt.Sprintf("DUP-%d", time.Now().Unix())

	// Insert first record
	_, err := db.Exec(
		"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
		testKey, "Duplicate Test", "todo",
	)
	if err != nil {
		return fmt.Errorf("first insert failed: %w", err)
	}

	// Try to insert duplicate
	_, err = db.Exec(
		"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
		testKey, "Duplicate Test 2", "todo",
	)
	if err == nil {
		return fmt.Errorf("duplicate key should have failed")
	}
	fmt.Printf("  ✅ Duplicate key rejected: %v\n", err)

	// Cleanup
	_, _ = db.Exec("DELETE FROM test_tasks WHERE key LIKE 'DUP-%'")

	// Test transaction rollback
	fmt.Println("  Testing transaction rollback...")
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	rollbackKey := fmt.Sprintf("ROLLBACK-%d", time.Now().Unix())
	_, err = tx.Exec(
		"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
		rollbackKey, "Rollback Test", "todo",
	)
	if err != nil {
		return fmt.Errorf("insert in transaction failed: %w", err)
	}

	// Rollback
	tx.Rollback()

	// Verify rollback
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_tasks WHERE key = ?", rollbackKey).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify rollback: %w", err)
	}
	if count != 0 {
		return fmt.Errorf("rollback failed, record still exists")
	}
	fmt.Println("  ✅ Transaction rollback works correctly")

	// Test concurrent writes (simulate)
	fmt.Println("  Testing concurrent writes...")
	start := time.Now()
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("CONCURRENT-%d-%d", time.Now().UnixNano(), i)
		_, err := db.Exec(
			"INSERT INTO test_tasks (key, title, status) VALUES (?, ?, ?)",
			key, fmt.Sprintf("Concurrent %d", i), "todo",
		)
		if err != nil {
			return fmt.Errorf("concurrent write %d failed: %w", i, err)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("  ✅ 10 concurrent writes: %v\n", elapsed)

	// Cleanup
	_, _ = db.Exec("DELETE FROM test_tasks WHERE key LIKE 'CONCURRENT-%'")

	return nil
}
