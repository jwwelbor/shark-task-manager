package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "test-schema.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check epics table schema
	rows, err := db.Query("PRAGMA table_info(epics)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("=== EPICS TABLE SCHEMA ===")
	for rows.Next() {
		var cid, notnull, pk int
		var name, type_ string
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s (%s)\n", name, type_)
	}

	// Check features table schema
	rows, err = db.Query("PRAGMA table_info(features)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\n=== FEATURES TABLE SCHEMA ===")
	for rows.Next() {
		var cid, notnull, pk int
		var name, type_ string
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &type_, &notnull, &dfltValue, &pk); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s (%s)\n", name, type_)
	}

	// Check indexes
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name IN ('epics', 'features') ORDER BY name")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\n=== INDEXES ===")
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  %s\n", name)
	}
}
