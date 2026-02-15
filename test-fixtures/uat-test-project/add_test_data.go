package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "/tmp/test-shark-project/shark-tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create documents
	_, err = db.Exec(`
		INSERT INTO documents (title, file_path, created_at)
		VALUES
			('Authentication Spec', 'docs/design/authentication-spec.md', datetime('now')),
			('API Design', 'docs/design/api-design.md', datetime('now')),
			('Test Plan', 'docs/testing/test-plan.md', datetime('now'))
	`)
	if err != nil {
		log.Fatal("Failed to create documents:", err)
	}

	// Link documents to task T-E01-F01-001
	_, err = db.Exec(`
		INSERT INTO task_documents (task_id, document_id)
		SELECT t.id, d.id
		FROM tasks t, documents d
		WHERE t.key = 'T-E01-F01-001'
			AND d.file_path IN (
				'docs/design/authentication-spec.md',
				'docs/design/api-design.md',
				'docs/testing/test-plan.md'
			)
	`)
	if err != nil {
		log.Fatal("Failed to link documents:", err)
	}

	fmt.Println("✅ Added 3 documents to database")
	fmt.Println("✅ Linked documents to task T-E01-F01-001")
	fmt.Println()
	fmt.Println("Documents:")
	fmt.Println("  - docs/design/authentication-spec.md")
	fmt.Println("  - docs/design/api-design.md")
	fmt.Println("  - docs/testing/test-plan.md")
}
