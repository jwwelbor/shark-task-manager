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
			BaseEntity: BaseEntity{
				ID:          1,
				Key:         "E01",
				Title:       "Epic Title",
				Slug:        &slug,
				Description: &desc,
				FilePath:    &filePath,
				ContextData: &ctxData,
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			Status: EpicStatusActive,
		}
	case EntityTypeFeature:
		return &Feature{BaseEntity: BaseEntity{ID: 2,
			Key:         "E01-F01",
			Title:       "Feature Title",
			Slug:        &slug,
			Description: &desc,

			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now}, Status: FeatureStatusActive,
		}
	case EntityTypeTask:
		return &Task{BaseEntity: BaseEntity{ID: 3,
			Key:         "T-E01-F01-001",
			Title:       "Task Title",
			Slug:        &slug,
			Description: &desc,

			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now}, Status: TaskStatus("todo"),
		}
	case EntityTypeBug:
		return &Bug{BaseEntity: BaseEntity{ID: 4,
			Key:         "B001",
			Title:       "Bug Title",
			Slug:        &slug,
			Description: &desc,

			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now}, Status: BugStatus("open"),
		}
	case EntityTypeChange:
		return &ChangeCard{BaseEntity: BaseEntity{ID: 5,
			Key:         "CC-001",
			Title:       "Change Title",
			Slug:        &slug,
			Description: &desc,

			FilePath:    &filePath,
			ContextData: &ctxData,
			CreatedAt:   now,
			UpdatedAt:   now}, Status: ChangeCardStatus("draft"),
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
			BaseEntity: BaseEntity{
				ID: 1, Key: "E01", Title: "Epic", Slug: &slug, Description: &desc,
				FilePath: &filePath, CreatedAt: now, UpdatedAt: now,
			},
			Status: EpicStatusActive, Priority: "high",
		},
		EntityTypeFeature: &Feature{BaseEntity: BaseEntity{ID: 2, Key: "E01-F01", Title: "Feature", Slug: &slug, Description: &desc,
			FilePath:  &filePath,
			CreatedAt: now, UpdatedAt: now}, Status: FeatureStatusActive,
		},
		EntityTypeTask: &Task{BaseEntity: BaseEntity{ID: 3, Key: "T-E01-F01-001", Title: "Task", Slug: &slug, Description: &desc,
			FilePath:  &filePath,
			CreatedAt: now, UpdatedAt: now}, Status: TaskStatus("todo"), Priority: 5,
		},
		EntityTypeBug: &Bug{BaseEntity: BaseEntity{ID: 4, Key: "B001", Title: "Bug", Slug: &slug, Description: &desc,
			FilePath:  &filePath,
			CreatedAt: now, UpdatedAt: now}, Status: BugStatus("open"), Severity: "medium",
		},
		EntityTypeChange: &ChangeCard{BaseEntity: BaseEntity{ID: 5, Key: "CC-001", Title: "Change", Slug: &slug, Description: &desc,
			FilePath:  &filePath,
			CreatedAt: now, UpdatedAt: now}, Status: ChangeCardStatus("draft"),
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

func TestBaseEntity_FieldAccess(t *testing.T) {
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	slug := "my-slug"
	desc := "my description"
	fp := "docs/plan/test.md"
	ctx := `{"step":"testing"}`

	b := &BaseEntity{
		ID:          42,
		Key:         "E01",
		Title:       "Test Entity",
		Slug:        &slug,
		Description: &desc,
		FilePath:    &fp,
		ContextData: &ctx,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if b.GetID() != 42 {
		t.Errorf("GetID() = %d, want 42", b.GetID())
	}
	if b.GetKey() != "E01" {
		t.Errorf("GetKey() = %q, want %q", b.GetKey(), "E01")
	}
	if b.GetTitle() != "Test Entity" {
		t.Errorf("GetTitle() = %q, want %q", b.GetTitle(), "Test Entity")
	}
	if b.GetSlug() != "my-slug" {
		t.Errorf("GetSlug() = %q, want %q", b.GetSlug(), "my-slug")
	}
	if b.GetDescription() != "my description" {
		t.Errorf("GetDescription() = %q, want %q", b.GetDescription(), "my description")
	}
	if b.GetFilePath() != "docs/plan/test.md" {
		t.Errorf("GetFilePath() = %q, want %q", b.GetFilePath(), "docs/plan/test.md")
	}
	if b.GetContextData() == nil || *b.GetContextData() != ctx {
		t.Errorf("GetContextData() = %v, want %q", b.GetContextData(), ctx)
	}
	if !b.GetCreatedAt().Equal(now) {
		t.Errorf("GetCreatedAt() = %v, want %v", b.GetCreatedAt(), now)
	}
	if !b.GetUpdatedAt().Equal(now) {
		t.Errorf("GetUpdatedAt() = %v, want %v", b.GetUpdatedAt(), now)
	}
}

func TestBaseEntity_NilFields(t *testing.T) {
	b := &BaseEntity{
		ID:    1,
		Key:   "E01",
		Title: "Test",
	}

	if b.GetSlug() != "" {
		t.Errorf("GetSlug() on nil = %q, want \"\"", b.GetSlug())
	}
	if b.GetDescription() != "" {
		t.Errorf("GetDescription() on nil = %q, want \"\"", b.GetDescription())
	}
	if b.GetFilePath() != "" {
		t.Errorf("GetFilePath() on nil = %q, want \"\"", b.GetFilePath())
	}
	if b.GetContextData() != nil {
		t.Errorf("GetContextData() on nil = %v, want nil", b.GetContextData())
	}
}

func TestBaseEntity_SetContextData(t *testing.T) {
	b := &BaseEntity{}

	// Initially nil
	if b.GetContextData() != nil {
		t.Fatal("initial GetContextData() should be nil")
	}

	// Set to a value
	data := `{"key":"value"}`
	b.SetContextData(&data)
	if got := b.GetContextData(); got == nil || *got != data {
		t.Errorf("after SetContextData, got %v, want %q", got, data)
	}

	// Set back to nil
	b.SetContextData(nil)
	if b.GetContextData() != nil {
		t.Errorf("after SetContextData(nil), got %v, want nil", b.GetContextData())
	}
}

func TestBaseEntity_ZeroValue(t *testing.T) {
	b := &BaseEntity{}

	// All accessors should work without panic on zero value
	if b.GetID() != 0 {
		t.Errorf("GetID() = %d, want 0", b.GetID())
	}
	if b.GetKey() != "" {
		t.Errorf("GetKey() = %q, want \"\"", b.GetKey())
	}
	if b.GetTitle() != "" {
		t.Errorf("GetTitle() = %q, want \"\"", b.GetTitle())
	}
	if b.GetSlug() != "" {
		t.Errorf("GetSlug() = %q, want \"\"", b.GetSlug())
	}
	if b.GetDescription() != "" {
		t.Errorf("GetDescription() = %q, want \"\"", b.GetDescription())
	}
	if b.GetFilePath() != "" {
		t.Errorf("GetFilePath() = %q, want \"\"", b.GetFilePath())
	}
	if b.GetContextData() != nil {
		t.Errorf("GetContextData() = %v, want nil", b.GetContextData())
	}
	if !b.GetCreatedAt().IsZero() {
		t.Errorf("GetCreatedAt() = %v, want zero", b.GetCreatedAt())
	}
	if !b.GetUpdatedAt().IsZero() {
		t.Errorf("GetUpdatedAt() = %v, want zero", b.GetUpdatedAt())
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
