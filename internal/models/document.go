package models

import (
	"fmt"
	"time"
)

// Document represents a related document linked to epics, features, or tasks
type Document struct {
	ID        int64     `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	FilePath  string    `db:"file_path" json:"file_path"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// NotFoundError represents an entity not found error
type NotFoundError struct {
	Entity string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Entity)
}
