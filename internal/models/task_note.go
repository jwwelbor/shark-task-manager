package models

import (
	"time"
)

// NoteType represents the type of note
type NoteType string

const (
	NoteTypeComment        NoteType = "comment"        // General observation
	NoteTypeDecision       NoteType = "decision"       // Why we chose X over Y
	NoteTypeBlocker        NoteType = "blocker"        // What's blocking progress
	NoteTypeSolution       NoteType = "solution"       // How we solved a problem
	NoteTypeReference      NoteType = "reference"      // External links, documentation
	NoteTypeImplementation NoteType = "implementation" // What we actually built
	NoteTypeTesting        NoteType = "testing"        // Test results, coverage
	NoteTypeFuture         NoteType = "future"         // Future improvements / TODO
	NoteTypeQuestion       NoteType = "question"       // Unanswered questions
	NoteTypeRejection      NoteType = "rejection"      // Rejection reason for backward transitions
)

// TaskNote represents a typed note attached to a task
type TaskNote struct {
	ID        int64     `json:"id" db:"id"`
	TaskID    int64     `json:"task_id" db:"task_id"`
	NoteType  NoteType  `json:"note_type" db:"note_type"`
	Content   string    `json:"content" db:"content"`
	CreatedBy *string   `json:"created_by,omitempty" db:"created_by"`
	Metadata  *string   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Validate validates the TaskNote fields
func (tn *TaskNote) Validate() error {
	if tn.TaskID == 0 {
		return ErrInvalidTaskID
	}
	if err := ValidateNoteType(string(tn.NoteType)); err != nil {
		return err
	}
	if tn.Content == "" {
		return ErrEmptyContent
	}
	return nil
}
