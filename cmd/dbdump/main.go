// Command dbdump connects to the configured turso/libsql database and writes a
// SQL dump (CREATE + INSERT statements) to stdout. Standalone safety-net backup
// tool for the E35 work; not part of the shipped CLI surface.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

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

	rows, err := db.Query(`SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "list tables: %v\n", err)
		os.Exit(1)
	}
	type tbl struct{ name, ddl string }
	var tables []tbl
	for rows.Next() {
		var t tbl
		if err := rows.Scan(&t.name, &t.ddl); err != nil {
			fmt.Fprintf(os.Stderr, "scan: %v\n", err)
			os.Exit(1)
		}
		tables = append(tables, t)
	}
	rows.Close()

	fmt.Println("PRAGMA foreign_keys=OFF;")
	fmt.Println("BEGIN TRANSACTION;")
	for _, t := range tables {
		fmt.Printf("%s;\n", t.ddl)
		dumpTable(db, t.name)
	}
	fmt.Println("COMMIT;")
}

func dumpTable(db *sql.DB, name string) {
	rows, err := db.Query("SELECT * FROM " + name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query %s: %v\n", name, err)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			fmt.Fprintf(os.Stderr, "scan %s: %v\n", name, err)
			return
		}
		parts := make([]string, len(cols))
		for i, v := range vals {
			parts[i] = sqlLiteral(v)
		}
		fmt.Printf("INSERT INTO %s VALUES(%s);\n", name, strings.Join(parts, ","))
	}
}

func sqlLiteral(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		return fmt.Sprintf("%g", x)
	case []byte:
		return "'" + strings.ReplaceAll(string(x), "'", "''") + "'"
	case string:
		return "'" + strings.ReplaceAll(x, "'", "''") + "'"
	default:
		return "'" + strings.ReplaceAll(fmt.Sprint(x), "'", "''") + "'"
	}
}
