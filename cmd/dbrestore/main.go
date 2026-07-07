package main

import (
	"database/sql"
	"fmt"
	"os"

	sharkdb "github.com/jwwelbor/shark-task-manager/internal/db"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: dbrestore <dump.sql> <output.db>")
		os.Exit(1)
	}

	dumpPath := os.Args[1]
	outputPath := os.Args[2]

	if err := restoreDump(dumpPath, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("restored dump into %s\n", outputPath)
}

func restoreDump(dumpPath, outputPath string) error {
	dumpSQL, err := os.ReadFile(dumpPath)
	if err != nil {
		return fmt.Errorf("read dump: %w", err)
	}

	if err := os.RemoveAll(outputPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing output DB: %w", err)
	}

	db, err := sql.Open("sqlite", outputPath+"?_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("open output DB: %w", err)
	}
	defer db.Close()

	if _, err := db.Exec(string(dumpSQL)); err != nil {
		return fmt.Errorf("import dump: %w", err)
	}

	if err := sharkdb.ApplySchemaAndMigrations(db); err != nil {
		return fmt.Errorf("rebuild schema and FTS: %w", err)
	}

	return nil
}
