package models

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// ptrInt is a test helper that returns a pointer to an int value.
// Avoids the verbose inline pattern: v := n; &v
func ptrInt(n int) *int {
	return &n
}

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
			// GetSize and SetSize are tested via TC-F001 below
			_ = e.GetSize()
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// REQ-F-001 — Size field on BaseEntity (TC-F001-A, TC-F001-B, TC-F001-C)
// ─────────────────────────────────────────────────────────────────────────────

// TC-F001-A: nil Size round-trips through JSON; GetSize() returns nil; "size" key absent.
func TestBaseEntity_Size_NilRoundTripsJSON(t *testing.T) {
	b := &BaseEntity{ID: 1, Key: "E01", Title: "Test"}

	if got := b.GetSize(); got != nil {
		t.Errorf("GetSize() on zero-value BaseEntity = %v, want nil", got)
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// omitempty means "size" key should be absent from JSON when nil
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal into map failed: %v", err)
	}
	if _, present := raw["size"]; present {
		t.Errorf("JSON output contains \"size\" key when Size is nil; want absent (omitempty)")
	}

	// Round-trip: unmarshal back into BaseEntity
	var b2 BaseEntity
	if err := json.Unmarshal(data, &b2); err != nil {
		t.Fatalf("json.Unmarshal into BaseEntity failed: %v", err)
	}
	if b2.GetSize() != nil {
		t.Errorf("after round-trip, GetSize() = %v, want nil", b2.GetSize())
	}
}

// TC-F001-B: ptr(5) Size round-trips through JSON; GetSize() returns pointer to 5; JSON contains "size":5.
func TestBaseEntity_Size_NonNilRoundTripsJSON(t *testing.T) {
	b := &BaseEntity{ID: 1, Key: "E01", Title: "Test", Size: ptrInt(5)}

	if got := b.GetSize(); got == nil || *got != 5 {
		t.Errorf("GetSize() = %v, want ptr(5)", got)
	}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// JSON must contain "size":5
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal into map failed: %v", err)
	}
	sizeVal, present := raw["size"]
	if !present {
		t.Errorf("JSON output missing \"size\" key; want present when Size = ptr(5)")
	} else if int(sizeVal.(float64)) != 5 {
		t.Errorf("JSON \"size\" = %v, want 5", sizeVal)
	}

	// Round-trip back
	var b2 BaseEntity
	if err := json.Unmarshal(data, &b2); err != nil {
		t.Fatalf("json.Unmarshal into BaseEntity failed: %v", err)
	}
	if got := b2.GetSize(); got == nil || *got != 5 {
		t.Errorf("after round-trip, GetSize() = %v, want ptr(5)", got)
	}
}

// TC-F001-C: SetSize mutates the field; GetSize reflects the change.
func TestBaseEntity_SetSize(t *testing.T) {
	b := &BaseEntity{}

	// Initially nil
	if got := b.GetSize(); got != nil {
		t.Errorf("initial GetSize() = %v, want nil", got)
	}

	// Set to ptr(3)
	b.SetSize(ptrInt(3))
	if got := b.GetSize(); got == nil || *got != 3 {
		t.Errorf("after SetSize(ptr(3)), GetSize() = %v, want ptr(3)", got)
	}

	// Set to nil
	b.SetSize(nil)
	if got := b.GetSize(); got != nil {
		t.Errorf("after SetSize(nil), GetSize() = %v, want nil", got)
	}
}

// TC-F001: GetSize/SetSize accessible via Entity interface (all 5 entity types)
func TestEntity_GetSetSize_ViaInterface(t *testing.T) {
	entities := []Entity{
		&Epic{},
		&Feature{},
		&Task{},
		&Bug{},
		&ChangeCard{},
	}

	for _, e := range entities {
		t.Run(string(e.GetEntityType()), func(t *testing.T) {
			// Initially nil
			if got := e.GetSize(); got != nil {
				t.Errorf("GetSize() on zero-value = %v, want nil", got)
			}

			// Set via interface
			e.SetSize(ptrInt(8))
			if got := e.GetSize(); got == nil || *got != 8 {
				t.Errorf("after SetSize(ptr(8)), GetSize() = %v, want ptr(8)", got)
			}

			// Clear via interface
			e.SetSize(nil)
			if got := e.GetSize(); got != nil {
				t.Errorf("after SetSize(nil), GetSize() = %v, want nil", got)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Per-entity Validate() rejects invalid Size, accepts nil and valid size
// (test-plan.md §6 "Existing test files to extend")
// ─────────────────────────────────────────────────────────────────────────────

// TestEpic_Validate_Size verifies that Epic.Validate() checks the Size field.
func TestEpic_Validate_Size(t *testing.T) {
	base := func() Epic {
		return Epic{
			BaseEntity: BaseEntity{Key: "E01", Title: "Epic Title"},
			Status:     EpicStatusActive,
			Priority:   "high",
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		e := base()
		e.Size = nil
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		e := base()
		e.SetSize(ptrInt(5))
		if err := e.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		e := base()
		e.SetSize(ptrInt(4))
		err := e.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestFeature_Validate_Size verifies that Feature.Validate() checks the Size field.
func TestFeature_Validate_Size(t *testing.T) {
	base := func() Feature {
		return Feature{
			BaseEntity: BaseEntity{Key: "E01-F01", Title: "Feature Title"},
			Status:     FeatureStatusActive,
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		f := base()
		f.Size = nil
		if err := f.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		f := base()
		f.SetSize(ptrInt(5))
		if err := f.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		f := base()
		f.SetSize(ptrInt(4))
		err := f.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestTask_Validate_Size verifies that Task.Validate() checks the Size field.
func TestTask_Validate_Size(t *testing.T) {
	base := func() Task {
		return Task{
			BaseEntity: BaseEntity{Key: "T-E01-F01-001", Title: "Task Title"},
			Status:     TaskStatus("todo"),
			Priority:   5,
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		tk := base()
		tk.Size = nil
		if err := tk.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		tk := base()
		tk.SetSize(ptrInt(5))
		if err := tk.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		tk := base()
		tk.SetSize(ptrInt(4))
		err := tk.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestBug_Validate_Size verifies that Bug.Validate() checks the Size field.
func TestBug_Validate_Size(t *testing.T) {
	base := func() Bug {
		return Bug{
			BaseEntity: BaseEntity{Key: "B001", Title: "Bug Title"},
			Status:     BugStatus("reported"),
			Severity:   BugSeverityHigh,
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		b := base()
		b.Size = nil
		if err := b.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		b := base()
		b.SetSize(ptrInt(5))
		if err := b.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		b := base()
		b.SetSize(ptrInt(4))
		err := b.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestChangeCard_Validate_Size verifies that ChangeCard.Validate() checks the Size field.
func TestChangeCard_Validate_Size(t *testing.T) {
	base := func() ChangeCard {
		return ChangeCard{
			BaseEntity: BaseEntity{Key: "CC-001", Title: "Change Title"},
			Status:     ChangeCardStatus("proposed"),
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		c := base()
		c.Size = nil
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		c := base()
		c.SetSize(ptrInt(5))
		if err := c.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		c := base()
		c.SetSize(ptrInt(4))
		err := c.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestIdea_Validate_Size verifies that Idea.Validate() checks the Size field.
// Idea does not embed BaseEntity so this also tests the direct Size field.
func TestIdea_Validate_Size(t *testing.T) {
	base := func() Idea {
		return Idea{
			Key:    "I-2026-01-01-01",
			Title:  "Idea Title",
			Status: IdeaStatusNew,
		}
	}

	t.Run("nil size is valid", func(t *testing.T) {
		i := base()
		i.Size = nil
		if err := i.Validate(); err != nil {
			t.Errorf("Validate() with Size=nil returned error: %v", err)
		}
	})

	t.Run("valid size ptr(5) is accepted", func(t *testing.T) {
		i := base()
		n := 5
		i.Size = &n
		if err := i.Validate(); err != nil {
			t.Errorf("Validate() with Size=ptr(5) returned error: %v", err)
		}
	})

	t.Run("invalid size ptr(4) is rejected", func(t *testing.T) {
		i := base()
		n := 4
		i.Size = &n
		err := i.Validate()
		if err == nil {
			t.Error("Validate() with Size=ptr(4) expected error but got nil")
		}
		if !errors.Is(err, ErrInvalidSize) {
			t.Errorf("Validate() with Size=ptr(4) error does not wrap ErrInvalidSize: %v", err)
		}
	})
}

// TestIdea_GetSetSize verifies Idea's Size field accessors.
func TestIdea_GetSetSize(t *testing.T) {
	i := &Idea{
		Key:    "I-2026-01-01-01",
		Title:  "Idea Title",
		Status: IdeaStatusNew,
	}

	// Initially nil
	if got := i.GetSize(); got != nil {
		t.Errorf("initial GetSize() = %v, want nil", got)
	}

	// Set to ptr(8)
	n := 8
	i.SetSize(&n)
	if got := i.GetSize(); got == nil || *got != 8 {
		t.Errorf("after SetSize(ptr(8)), GetSize() = %v, want ptr(8)", got)
	}

	// Clear to nil
	i.SetSize(nil)
	if got := i.GetSize(); got != nil {
		t.Errorf("after SetSize(nil), GetSize() = %v, want nil", got)
	}
}
