package models

import (
	"testing"
	"time"
)

// testEntityFactory creates a populated entity of the given type for testing.
func testEntityFactory(entityType EntityType) Entity {
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	slug := "test-slug"
	desc := "test description"
	filePath := "docs/plan/test.md"
	ctxData := `{"key":"value"}`

	switch entityType {
	case EntityTypeEpic:
		return &Epic{
			ID:          1,
			Key:         "E01",
			Title:       "Epic Title",
			Slug:        &slug,
			Description: &desc,
			Status:      EpicStatusActive,
			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	case EntityTypeFeature:
		return &Feature{
			ID:          2,
			Key:         "E01-F01",
			Title:       "Feature Title",
			Slug:        &slug,
			Description: &desc,
			Status:      FeatureStatusActive,
			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	case EntityTypeTask:
		return &Task{
			ID:          3,
			Key:         "T-E01-F01-001",
			Title:       "Task Title",
			Slug:        &slug,
			Description: &desc,
			Status:      TaskStatus("todo"),
			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	case EntityTypeBug:
		return &Bug{
			ID:          4,
			Key:         "B001",
			Title:       "Bug Title",
			Slug:        &slug,
			Description: &desc,
			Status:      BugStatus("open"),
			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	case EntityTypeChange:
		return &ChangeCard{
			ID:          5,
			Key:         "CC-001",
			Title:       "Change Title",
			Slug:        &slug,
			Description: &desc,
			Status:      ChangeCardStatus("draft"),
			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
	default:
		return nil
	}
}

func TestEntity_Accessors(t *testing.T) {
	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		entityType EntityType
		wantID     int64
		wantKey    string
		wantTitle  string
		wantSlug   string
		wantStatus string
		wantDesc   string
		wantFile   string
	}{
		{
			name:       "epic",
			entityType: EntityTypeEpic,
			wantID:     1,
			wantKey:    "E01",
			wantTitle:  "Epic Title",
			wantSlug:   "test-slug",
			wantStatus: "active",
			wantDesc:   "test description",
			wantFile:   "docs/plan/test.md",
		},
		{
			name:       "feature",
			entityType: EntityTypeFeature,
			wantID:     2,
			wantKey:    "E01-F01",
			wantTitle:  "Feature Title",
			wantSlug:   "test-slug",
			wantStatus: "active",
			wantDesc:   "test description",
			wantFile:   "docs/plan/test.md",
		},
		{
			name:       "task",
			entityType: EntityTypeTask,
			wantID:     3,
			wantKey:    "T-E01-F01-001",
			wantTitle:  "Task Title",
			wantSlug:   "test-slug",
			wantStatus: "todo",
			wantDesc:   "test description",
			wantFile:   "docs/plan/test.md",
		},
		{
			name:       "bug",
			entityType: EntityTypeBug,
			wantID:     4,
			wantKey:    "B001",
			wantTitle:  "Bug Title",
			wantSlug:   "test-slug",
			wantStatus: "open",
			wantDesc:   "test description",
			wantFile:   "docs/plan/test.md",
		},
		{
			name:       "change_card",
			entityType: EntityTypeChange,
			wantID:     5,
			wantKey:    "CC-001",
			wantTitle:  "Change Title",
			wantSlug:   "test-slug",
			wantStatus: "draft",
			wantDesc:   "test description",
			wantFile:   "docs/plan/test.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := testEntityFactory(tt.entityType)
			if e == nil {
				t.Fatal("testEntityFactory returned nil")
			}

			if got := e.GetID(); got != tt.wantID {
				t.Errorf("GetID() = %d, want %d", got, tt.wantID)
			}
			if got := e.GetKey(); got != tt.wantKey {
				t.Errorf("GetKey() = %q, want %q", got, tt.wantKey)
			}
			if got := e.GetTitle(); got != tt.wantTitle {
				t.Errorf("GetTitle() = %q, want %q", got, tt.wantTitle)
			}
			if got := e.GetSlug(); got != tt.wantSlug {
				t.Errorf("GetSlug() = %q, want %q", got, tt.wantSlug)
			}
			if got := e.GetEntityType(); got != tt.entityType {
				t.Errorf("GetEntityType() = %q, want %q", got, tt.entityType)
			}
			if got := e.GetStatus(); got != tt.wantStatus {
				t.Errorf("GetStatus() = %q, want %q", got, tt.wantStatus)
			}
			if got := e.GetDescription(); got != tt.wantDesc {
				t.Errorf("GetDescription() = %q, want %q", got, tt.wantDesc)
			}
			if got := e.GetFilePath(); got != tt.wantFile {
				t.Errorf("GetFilePath() = %q, want %q", got, tt.wantFile)
			}
			if got := e.GetContextData(); got == nil {
				t.Error("GetContextData() returned nil, want non-nil")
			} else if *got != `{"key":"value"}` {
				t.Errorf("GetContextData() = %q, want %q", *got, `{"key":"value"}`)
			}
			if got := e.GetCreatedAt(); !got.Equal(now) {
				t.Errorf("GetCreatedAt() = %v, want %v", got, now)
			}
			if got := e.GetUpdatedAt(); !got.Equal(now) {
				t.Errorf("GetUpdatedAt() = %v, want %v", got, now)
			}
		})
	}
}

func TestEntity_GetEntityType(t *testing.T) {
	tests := []struct {
		name     string
		entity   Entity
		wantType EntityType
	}{
		{"epic", &Epic{}, EntityTypeEpic},
		{"feature", &Feature{}, EntityTypeFeature},
		{"task", &Task{}, EntityTypeTask},
		{"bug", &Bug{}, EntityTypeBug},
		{"change_card", &ChangeCard{}, EntityTypeChange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entity.GetEntityType(); got != tt.wantType {
				t.Errorf("GetEntityType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestEntity_SetStatus(t *testing.T) {
	entityTypes := []EntityType{
		EntityTypeEpic,
		EntityTypeFeature,
		EntityTypeTask,
		EntityTypeBug,
		EntityTypeChange,
	}

	for _, et := range entityTypes {
		t.Run(string(et), func(t *testing.T) {
			e := testEntityFactory(et)

			// Set a new status
			e.SetStatus("new_status")
			if got := e.GetStatus(); got != "new_status" {
				t.Errorf("after SetStatus(\"new_status\"), GetStatus() = %q, want %q", got, "new_status")
			}

			// Set empty status
			e.SetStatus("")
			if got := e.GetStatus(); got != "" {
				t.Errorf("after SetStatus(\"\"), GetStatus() = %q, want %q", got, "")
			}
		})
	}
}

func TestEntity_SetContextData(t *testing.T) {
	entityTypes := []EntityType{
		EntityTypeEpic,
		EntityTypeFeature,
		EntityTypeTask,
		EntityTypeBug,
		EntityTypeChange,
	}

	for _, et := range entityTypes {
		t.Run(string(et), func(t *testing.T) {
			e := testEntityFactory(et)

			// Verify initial non-nil
			if got := e.GetContextData(); got == nil {
				t.Fatal("initial GetContextData() should be non-nil")
			}

			// Set to nil
			e.SetContextData(nil)
			if got := e.GetContextData(); got != nil {
				t.Errorf("after SetContextData(nil), GetContextData() = %v, want nil", got)
			}

			// Set to new value
			newData := `{"new":"data"}`
			e.SetContextData(&newData)
			got := e.GetContextData()
			if got == nil {
				t.Fatal("after SetContextData(&newData), GetContextData() returned nil")
			}
			if *got != newData {
				t.Errorf("GetContextData() = %q, want %q", *got, newData)
			}

			// Set back to nil
			e.SetContextData(nil)
			if got := e.GetContextData(); got != nil {
				t.Errorf("after second SetContextData(nil), GetContextData() = %v, want nil", got)
			}
		})
	}
}

func TestEntity_NilPointerFields(t *testing.T) {
	// Test that nil *string fields return "" and do not panic.
	tests := []struct {
		name   string
		entity Entity
	}{
		{"epic_zero_value", &Epic{}},
		{"feature_zero_value", &Feature{}},
		{"task_zero_value", &Task{}},
		{"bug_zero_value", &Bug{}},
		{"change_card_zero_value", &ChangeCard{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entity.GetSlug(); got != "" {
				t.Errorf("GetSlug() on zero-value = %q, want \"\"", got)
			}
			if got := tt.entity.GetDescription(); got != "" {
				t.Errorf("GetDescription() on zero-value = %q, want \"\"", got)
			}
			if got := tt.entity.GetFilePath(); got != "" {
				t.Errorf("GetFilePath() on zero-value = %q, want \"\"", got)
			}
			if got := tt.entity.GetContextData(); got != nil {
				t.Errorf("GetContextData() on zero-value = %v, want nil", got)
			}
		})
	}
}

func TestEntity_Validate(t *testing.T) {
	// Test that Validate() is callable via the Entity interface for all 5 types.
	entityTypes := []EntityType{
		EntityTypeEpic,
		EntityTypeFeature,
		EntityTypeTask,
		EntityTypeBug,
		EntityTypeChange,
	}

	// Build fully valid entities for Validate() testing.
	slug := "test-slug"
	desc := "test description"
	filePath := "docs/plan/test.md"
	now := time.Now()

	validEntities := map[EntityType]Entity{
		EntityTypeEpic: &Epic{
			ID: 1, Key: "E01", Title: "Epic", Slug: &slug, Description: &desc,
			Status: EpicStatusActive, FilePath: &filePath, Priority: "high",
			CreatedAt: now, UpdatedAt: now,
		},
		EntityTypeFeature: &Feature{
			ID: 2, Key: "E01-F01", Title: "Feature", Slug: &slug, Description: &desc,
			Status: FeatureStatusActive, FilePath: &filePath,
			CreatedAt: now, UpdatedAt: now,
		},
		EntityTypeTask: &Task{
			ID: 3, Key: "T-E01-F01-001", Title: "Task", Slug: &slug, Description: &desc,
			Status: TaskStatus("todo"), FilePath: &filePath, Priority: 5,
			CreatedAt: now, UpdatedAt: now,
		},
		EntityTypeBug: &Bug{
			ID: 4, Key: "B001", Title: "Bug", Slug: &slug, Description: &desc,
			Status: BugStatus("open"), FilePath: &filePath, Severity: "medium",
			CreatedAt: now, UpdatedAt: now,
		},
		EntityTypeChange: &ChangeCard{
			ID: 5, Key: "CC-001", Title: "Change", Slug: &slug, Description: &desc,
			Status: ChangeCardStatus("draft"), FilePath: &filePath,
			CreatedAt: now, UpdatedAt: now,
		},
	}

	for _, et := range entityTypes {
		t.Run(string(et)+"_populated", func(t *testing.T) {
			e := validEntities[et]
			err := e.Validate()
			if err != nil {
				t.Errorf("Validate() on populated %s returned error: %v", et, err)
			}
		})
	}

	// Zero-value structs should fail validation (empty title).
	zeroEntities := []struct {
		name   string
		entity Entity
	}{
		{"epic_zero", &Epic{}},
		{"feature_zero", &Feature{}},
		{"task_zero", &Task{}},
		{"bug_zero", &Bug{}},
		{"change_card_zero", &ChangeCard{}},
	}

	for _, tt := range zeroEntities {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entity.Validate()
			if err == nil {
				t.Error("Validate() on zero-value entity should return error (empty title)")
			}
		})
	}
}

func TestEntity_ZeroValueAccessors(t *testing.T) {
	// Ensure zero-value structs do not panic on any accessor.
	entities := []Entity{
		&Epic{},
		&Feature{},
		&Task{},
		&Bug{},
		&ChangeCard{},
	}

	for _, e := range entities {
		t.Run(string(e.GetEntityType()), func(t *testing.T) {
			// These should not panic
			_ = e.GetID()
			_ = e.GetKey()
			_ = e.GetTitle()
			_ = e.GetSlug()
			_ = e.GetEntityType()
			_ = e.GetStatus()
			_ = e.GetDescription()
			_ = e.GetFilePath()
			_ = e.GetContextData()
			_ = e.GetCreatedAt()
			_ = e.GetUpdatedAt()
		})
	}
}
