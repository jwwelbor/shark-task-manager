package repoerr

import (
	"errors"
	"testing"
)

func TestIsSQLiteUniqueViolation(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "modernc", err: errors.New("constraint failed: UNIQUE constraint failed: questions.key (1555)"), want: true},
		{name: "libsql", err: errors.New("SQLITE_CONSTRAINT: UNIQUE constraint failed: questions.key"), want: true},
		{name: "foreign key constraint", err: errors.New("SQLITE_CONSTRAINT: FOREIGN KEY constraint failed"), want: false},
		{name: "unrelated unique prose", err: errors.New("unique workflow configuration is invalid"), want: false},
		{name: "nil", err: nil, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSQLiteUniqueViolation(tt.err); got != tt.want {
				t.Fatalf("IsSQLiteUniqueViolation(%v) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}
