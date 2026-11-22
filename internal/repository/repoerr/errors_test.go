package repoerr

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotFoundRecognizesBothRepositoryAndLegacySentinels(t *testing.T) {
	for _, testCase := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "repository sentinel", err: ErrNotFound, want: true},
		{name: "wrapped repository sentinel", err: fmt.Errorf("wrapped: %w", ErrNotFound), want: true},
		{name: "legacy sql no rows", err: sql.ErrNoRows, want: true},
		{name: "nil", err: nil, want: false},
		{name: "unrelated error", err: errors.New("database offline"), want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, IsNotFound(testCase.err))
		})
	}
}

func TestIsDatabaseFailure(t *testing.T) {
	for _, testCase := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "repository marker", err: ErrDatabaseFailure, want: true},
		{name: "wrapped marker", err: fmt.Errorf("lookup failed: %w", ErrDatabaseFailure), want: true},
		{name: "nil", err: nil, want: false},
		{name: "unmarked error", err: errors.New("database offline"), want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, IsDatabaseFailure(testCase.err))
		})
	}
}

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
