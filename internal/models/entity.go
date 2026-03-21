package models

import "time"

// Entity is the polymorphic interface implemented by all domain entity types.
// It provides accessor methods for the shared fields common to all entities.
//
// This interface is additive -- existing direct field access (e.g., epic.Key)
// continues to work unchanged. The interface is used only by cross-cutting
// services that need to operate on entities polymorphically.
type Entity interface {
	GetID() int64
	GetKey() string
	GetTitle() string
	GetSlug() string
	GetEntityType() EntityType
	GetStatus() string
	SetStatus(status string)
	GetDescription() string
	GetFilePath() string
	GetContextData() *string
	SetContextData(data *string)
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
	Validate() error
}

// BaseEntity contains the shared fields common to all domain entities.
// Entity types embed BaseEntity as an anonymous field to inherit these
// fields and their accessor methods.
//
// Status is intentionally excluded because each entity type uses a
// distinct typed status alias (EpicStatus, TaskStatus, etc.) that
// appears in ~2,178 call sites. Keeping Status per-entity avoids a
// massive codebase-wide migration.
type BaseEntity struct {
	ID          int64     `json:"id" db:"id"`
	Key         string    `json:"key" db:"key"`
	Title       string    `json:"title" db:"title"`
	Slug        *string   `json:"slug,omitempty" db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	FilePath    *string   `json:"file_path,omitempty" db:"file_path"`
	ContextData *string   `json:"context_data,omitempty" db:"context_data"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

func (b *BaseEntity) GetID() int64            { return b.ID }
func (b *BaseEntity) GetKey() string          { return b.Key }
func (b *BaseEntity) GetTitle() string        { return b.Title }
func (b *BaseEntity) GetCreatedAt() time.Time { return b.CreatedAt }
func (b *BaseEntity) GetUpdatedAt() time.Time { return b.UpdatedAt }

func (b *BaseEntity) GetSlug() string {
	if b.Slug != nil {
		return *b.Slug
	}
	return ""
}

func (b *BaseEntity) GetDescription() string {
	if b.Description != nil {
		return *b.Description
	}
	return ""
}

func (b *BaseEntity) GetFilePath() string {
	if b.FilePath != nil {
		return *b.FilePath
	}
	return ""
}

func (b *BaseEntity) GetContextData() *string     { return b.ContextData }
func (b *BaseEntity) SetContextData(data *string) { b.ContextData = data }

// Compile-time interface satisfaction checks.
var (
	_ Entity = (*Epic)(nil)
	_ Entity = (*Feature)(nil)
	_ Entity = (*Task)(nil)
	_ Entity = (*Bug)(nil)
	_ Entity = (*ChangeCard)(nil)
)
