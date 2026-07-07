// Command dbdump connects to the configured turso/libsql database and writes a
// SQL dump (CREATE + INSERT statements) to stdout. Standalone safety-net backup
// tool for the E35 work; not part of the shipped CLI surface.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	url := os.Getenv("SHARK_DB_URL")
	tokenFile := os.Getenv("SHARK_AUTH_TOKEN_FILE")
	if url == "" {
		fmt.Fprintln(os.Stderr, "SHARK_DB_URL not set")
		os.Exit(1)
	}
	dsn := url
	if tokenFile != "" {
		tok, err := os.ReadFile(tokenFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read token: %v\n", err)
			os.Exit(1)
		}
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		dsn = url + sep + "authToken=" + strings.TrimSpace(string(tok))
	}
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tables, err := listDumpTables(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list tables: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PRAGMA foreign_keys=OFF;")
	fmt.Println("BEGIN TRANSACTION;")
	for _, t := range tables {
		fmt.Printf("%s;\n", t.ddl)
		if err := dumpTable(db, t.name); err != nil {
			// Fail loudly: a partial dump that silently drops rows is a
			// corrupt backup, which is worse than no backup.
			fmt.Fprintf(os.Stderr, "dump table %s: %v\n", t.name, err)
			os.Exit(1)
		}
	}
	fmt.Println("COMMIT;")
	fmt.Println("-- Restores should re-run Shark migrations locally to rebuild FTS.")
	fmt.Println("-- Use: go run ./cmd/dbrestore <dump.sql> <output.db>")
}

type tbl struct{ name, ddl string }

func listDumpTables(db *sql.DB) ([]tbl, error) {
	rows, err := db.Query(`SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []tbl
	for rows.Next() {
		var t tbl
		if err := rows.Scan(&t.name, &t.ddl); err != nil {
			return nil, err
		}
		if shouldDumpTable(t.name, t.ddl) {
			tables = append(tables, t)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func shouldDumpTable(name, ddl string) bool {
	lowerName := strings.ToLower(name)
	lowerDDL := strings.ToLower(ddl)

	if lowerName == "entity_search_fts" || lowerName == "task_search_fts" {
		return false
	}
	if strings.HasPrefix(lowerName, "entity_search_fts_") || strings.HasPrefix(lowerName, "task_search_fts_") {
		return false
	}
	if strings.Contains(lowerDDL, "virtual table") && strings.Contains(lowerDDL, "using fts5") {
		return false
	}
	return true
}

// quoteIdent wraps a SQL identifier in double quotes, escaping embedded quotes,
// so an unexpected table name can never break out of the identifier position.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func dumpTable(db *sql.DB, name string) error {
	ident := quoteIdent(name)
	rows, err := db.Query("SELECT * FROM " + ident)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("columns: %w", err)
	}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		parts := make([]string, len(cols))
		for i, v := range vals {
			parts[i] = sqlLiteral(v)
		}
		fmt.Printf("INSERT INTO %s VALUES(%s);\n", ident, strings.Join(parts, ","))
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("row iteration: %w", err)
	}
	return nil
}

func sqlLiteral(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case bool:
		if x {
			return "1"
		}
		return "0"
	case time.Time:
		// SQLite/shark store timestamps as "2006-01-02 15:04:05" (UTC). The
		// libsql driver hands TIMESTAMP columns back as time.Time; format them
		// in that exact layout so a reinsert reads back into *time.Time.
		return "'" + x.UTC().Format("2006-01-02 15:04:05") + "'"
	case []byte:
		return "'" + strings.ReplaceAll(string(x), "'", "''") + "'"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}
