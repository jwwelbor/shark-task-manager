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

// Compile-time interface satisfaction checks.
var (
	_ Entity = (*Epic)(nil)
	_ Entity = (*Feature)(nil)
	_ Entity = (*Task)(nil)
	_ Entity = (*Bug)(nil)
	_ Entity = (*ChangeCard)(nil)
)
