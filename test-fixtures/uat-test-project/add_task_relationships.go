package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "/home/jwwel/projects/shark-task-manager/test-fixtures/uat-test-project/shark-tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// First, create 2 more tasks to relate to T-E01-F01-001
	_, err = db.Exec(`
		INSERT INTO tasks (key, title, status, priority, epic_id, feature_id, agent_type, file_path, created_at, updated_at)
		VALUES
			('T-E01-F01-002', 'Design database schema', 'todo', 5, 1, 1, 'developer', 'docs/plan/E01-test-epic-for-uat/E01-F01-test-feature-with-related-docs/tasks/T-E01-F01-002.md', datetime('now'), datetime('now')),
			('T-E01-F01-003', 'Write API tests', 'todo', 5, 1, 1, 'qa', 'docs/plan/E01-test-epic-for-uat/E01-F01-test-feature-with-related-docs/tasks/T-E01-F01-003.md', datetime('now'), datetime('now'))
	`)
	if err != nil {
		log.Fatal("Failed to create tasks:", err)
	}

	// Create task relationships
	// T-E01-F01-001 depends on T-E01-F01-002 (database schema must be designed first)
	// T-E01-F01-003 (tests) related to T-E01-F01-001 (implementation)
	_, err = db.Exec(`
		INSERT INTO task_relationships (from_task_id, to_task_id, relationship_type)
		SELECT t1.id, t2.id, 'depends_on'
		FROM tasks t1, tasks t2
		WHERE t1.key = 'T-E01-F01-001' AND t2.key = 'T-E01-F01-002'

		UNION ALL

		SELECT t1.id, t2.id, 'related_to'
		FROM tasks t1, tasks t2
		WHERE t1.key = 'T-E01-F01-001' AND t2.key = 'T-E01-F01-003'
	`)
	if err != nil {
		log.Fatal("Failed to create relationships:", err)
	}

	fmt.Println("✅ Created 2 additional tasks:")
	fmt.Println("   - T-E01-F01-002: Design database schema")
	fmt.Println("   - T-E01-F01-003: Write API tests")
	fmt.Println()
	fmt.Println("✅ Created task relationships:")
	fmt.Println("   - T-E01-F01-001 depends_on T-E01-F01-002")
	fmt.Println("   - T-E01-F01-001 related_to T-E01-F01-003")
	fmt.Println()
	fmt.Println("Now T-E01-F01-001 has 2 related tasks:")
	fmt.Println("   - T-E01-F01-002 (dependency)")
	fmt.Println("   - T-E01-F01-003 (related)")
}
