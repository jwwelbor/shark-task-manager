package main

import "testing"

func TestShouldDumpTable_FiltersFTSObjects(t *testing.T) {
	tests := []struct {
		name string
		tbl  string
		ddl  string
		want bool
	}{
		{name: "application table", tbl: "tasks", ddl: "CREATE TABLE tasks(id INTEGER)", want: true},
		{name: "unified fts virtual table", tbl: "entity_search_fts", ddl: "CREATE VIRTUAL TABLE entity_search_fts USING fts5(title)", want: false},
		{name: "fts shadow table", tbl: "entity_search_fts_data", ddl: "CREATE TABLE 'entity_search_fts_data'(id INTEGER)", want: false},
		{name: "legacy fts table", tbl: "task_search_fts", ddl: "CREATE VIRTUAL TABLE task_search_fts USING fts5(title)", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDumpTable(tt.tbl, tt.ddl); got != tt.want {
				t.Fatalf("shouldDumpTable(%q) = %v, want %v", tt.tbl, got, tt.want)
			}
		})
	}
}
