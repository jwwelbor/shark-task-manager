package main

import (
	"database/sql"
	"fmt"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	_ "modernc.org/sqlite"
)

func main() {
	sqlDB, err := sql.Open("sqlite", "shark-tasks.db?_foreign_keys=on")
	if err != nil {
		fmt.Printf("open error: %v\n", err)
		return
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		fmt.Printf("ping error: %v\n", err)
		return
	}

	applied, err := db.ApplySchemaIfNeeded(sqlDB)
	fmt.Printf("applied=%v err=%v\n", applied, err)

	var version int
	if err := sqlDB.QueryRow(`SELECT version FROM schema_version ORDER BY version DESC LIMIT 1`).Scan(&version); err != nil {
		fmt.Printf("version query error: %v\n", err)
	}
	fmt.Printf("version after: %d\n", version)
}
