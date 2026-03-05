package keys

import (
	"testing"
)

func TestNewKeyService(t *testing.T) {
	ks := NewKeyService()
	if ks == nil {
		t.Fatal("NewKeyService() returned nil")
	}
}

func TestKeyService_DetectEntityType(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want EntityType
	}{
		// Empty / invalid
		{"empty string", "", EntityTypeUnknown},
		{"random string", "hello", EntityTypeUnknown},
		{"just numbers", "123", EntityTypeUnknown},

		// Epic keys
		{"epic uppercase", "E07", EntityTypeEpic},
		{"epic lowercase", "e07", EntityTypeEpic},
		{"epic E01", "E01", EntityTypeEpic},
		{"epic E99", "E99", EntityTypeEpic},
		{"epic with slug", "E07-user-management", EntityTypeEpic},
		{"epic with slug lowercase", "e07-user-management", EntityTypeEpic},
		{"epic with multi-word slug", "E07-my-cool-epic", EntityTypeEpic},

		// Feature keys
		{"feature full key", "E07-F01", EntityTypeFeature},
		{"feature lowercase", "e07-f01", EntityTypeFeature},
		{"feature suffix only", "F01", EntityTypeFeature},
		{"feature suffix lowercase", "f01", EntityTypeFeature},
		{"feature with slug", "E07-F01-auth-module", EntityTypeFeature},
		{"feature with slug lowercase", "e07-f01-auth-module", EntityTypeFeature},

		// Task keys
		{"task full format", "T-E07-F01-001", EntityTypeTask},
		{"task full lowercase", "t-e07-f01-001", EntityTypeTask},
		{"task short format", "E07-F01-001", EntityTypeTask},
		{"task short lowercase", "e07-f01-001", EntityTypeTask},
		{"task with slug", "E07-F01-001-implement-jwt", EntityTypeTask},
		{"task full with slug", "T-E07-F01-001-implement-jwt", EntityTypeTask},
		{"task short with slug lowercase", "e07-f01-001-implement-jwt", EntityTypeTask},

		// Edge cases
		{"invalid epic single digit", "E1", EntityTypeUnknown},
		{"invalid epic three digits", "E001", EntityTypeUnknown},
		{"trailing dash", "E07-", EntityTypeUnknown},
		{"just T prefix", "T-", EntityTypeUnknown},
		{"feature without epic", "F01-some-slug", EntityTypeFeature},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.DetectEntityType(tt.key)
			if got != tt.want {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_Parse_Epic(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name           string
		key            string
		wantEntityType EntityType
		wantNormalized string
		wantEpicNum    string
		wantSlug       string
	}{
		{
			name:           "simple epic",
			key:            "E07",
			wantEntityType: EntityTypeEpic,
			wantNormalized: "E07",
			wantEpicNum:    "07",
			wantSlug:       "",
		},
		{
			name:           "lowercase epic",
			key:            "e07",
			wantEntityType: EntityTypeEpic,
			wantNormalized: "E07",
			wantEpicNum:    "07",
			wantSlug:       "",
		},
		{
			name:           "epic with slug",
			key:            "E07-user-management",
			wantEntityType: EntityTypeEpic,
			wantNormalized: "E07",
			wantEpicNum:    "07",
			wantSlug:       "user-management",
		},
		{
			name:           "epic with slug lowercase",
			key:            "e07-user-management",
			wantEntityType: EntityTypeEpic,
			wantNormalized: "E07",
			wantEpicNum:    "07",
			wantSlug:       "user-management",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != tt.wantEntityType {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, tt.wantEntityType)
			}
			if got.Normalized != tt.wantNormalized {
				t.Errorf("Parse(%q).Normalized = %q, want %q", tt.key, got.Normalized, tt.wantNormalized)
			}
			if got.EpicNum != tt.wantEpicNum {
				t.Errorf("Parse(%q).EpicNum = %q, want %q", tt.key, got.EpicNum, tt.wantEpicNum)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("Parse(%q).Slug = %q, want %q", tt.key, got.Slug, tt.wantSlug)
			}
			if got.Raw != tt.key {
				t.Errorf("Parse(%q).Raw = %q, want %q", tt.key, got.Raw, tt.key)
			}
		})
	}
}

func TestKeyService_Parse_Feature(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name           string
		key            string
		wantEntityType EntityType
		wantNormalized string
		wantEpicNum    string
		wantFeatureNum string
		wantSlug       string
	}{
		{
			name:           "full feature key",
			key:            "E07-F01",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "E07-F01",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantSlug:       "",
		},
		{
			name:           "feature lowercase",
			key:            "e07-f01",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "E07-F01",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantSlug:       "",
		},
		{
			name:           "feature suffix only",
			key:            "F01",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "F01",
			wantEpicNum:    "",
			wantFeatureNum: "01",
			wantSlug:       "",
		},
		{
			name:           "feature suffix lowercase",
			key:            "f01",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "F01",
			wantEpicNum:    "",
			wantFeatureNum: "01",
			wantSlug:       "",
		},
		{
			name:           "feature with slug",
			key:            "E07-F01-auth-module",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "E07-F01",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantSlug:       "auth-module",
		},
		{
			name:           "feature with slug lowercase",
			key:            "e07-f01-auth-module",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "E07-F01",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantSlug:       "auth-module",
		},
		{
			name:           "feature suffix with slug",
			key:            "F01-some-feature",
			wantEntityType: EntityTypeFeature,
			wantNormalized: "F01",
			wantEpicNum:    "",
			wantFeatureNum: "01",
			wantSlug:       "some-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != tt.wantEntityType {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, tt.wantEntityType)
			}
			if got.Normalized != tt.wantNormalized {
				t.Errorf("Parse(%q).Normalized = %q, want %q", tt.key, got.Normalized, tt.wantNormalized)
			}
			if got.EpicNum != tt.wantEpicNum {
				t.Errorf("Parse(%q).EpicNum = %q, want %q", tt.key, got.EpicNum, tt.wantEpicNum)
			}
			if got.FeatureNum != tt.wantFeatureNum {
				t.Errorf("Parse(%q).FeatureNum = %q, want %q", tt.key, got.FeatureNum, tt.wantFeatureNum)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("Parse(%q).Slug = %q, want %q", tt.key, got.Slug, tt.wantSlug)
			}
		})
	}
}

func TestKeyService_Parse_Task(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name           string
		key            string
		wantEntityType EntityType
		wantNormalized string
		wantEpicNum    string
		wantFeatureNum string
		wantTaskNum    string
		wantSlug       string
	}{
		{
			name:           "full task key",
			key:            "T-E07-F01-001",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "",
		},
		{
			name:           "full task key lowercase",
			key:            "t-e07-f01-001",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "",
		},
		{
			name:           "short task key",
			key:            "E07-F01-001",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "",
		},
		{
			name:           "short task key lowercase",
			key:            "e07-f01-001",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "",
		},
		{
			name:           "task with slug",
			key:            "E07-F01-001-implement-jwt",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "implement-jwt",
		},
		{
			name:           "full task with slug",
			key:            "T-E07-F01-001-implement-jwt",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "implement-jwt",
		},
		{
			name:           "task with slug lowercase",
			key:            "e07-f01-001-implement-jwt-token",
			wantEntityType: EntityTypeTask,
			wantNormalized: "T-E07-F01-001",
			wantEpicNum:    "07",
			wantFeatureNum: "01",
			wantTaskNum:    "001",
			wantSlug:       "implement-jwt-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != tt.wantEntityType {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, tt.wantEntityType)
			}
			if got.Normalized != tt.wantNormalized {
				t.Errorf("Parse(%q).Normalized = %q, want %q", tt.key, got.Normalized, tt.wantNormalized)
			}
			if got.EpicNum != tt.wantEpicNum {
				t.Errorf("Parse(%q).EpicNum = %q, want %q", tt.key, got.EpicNum, tt.wantEpicNum)
			}
			if got.FeatureNum != tt.wantFeatureNum {
				t.Errorf("Parse(%q).FeatureNum = %q, want %q", tt.key, got.FeatureNum, tt.wantFeatureNum)
			}
			if got.TaskNum != tt.wantTaskNum {
				t.Errorf("Parse(%q).TaskNum = %q, want %q", tt.key, got.TaskNum, tt.wantTaskNum)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("Parse(%q).Slug = %q, want %q", tt.key, got.Slug, tt.wantSlug)
			}
		})
	}
}

func TestKeyService_Parse_Unknown(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
	}{
		{"empty string", ""},
		{"random text", "hello"},
		{"just numbers", "123"},
		{"invalid epic single digit", "E1"},
		{"invalid epic three digits", "E001"},
		{"trailing dash", "E07-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != EntityTypeUnknown {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, EntityTypeUnknown)
			}
			if got.Raw != tt.key {
				t.Errorf("Parse(%q).Raw = %q, want %q", tt.key, got.Raw, tt.key)
			}
		})
	}
}

func TestKeyService_Normalize(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"epic uppercase", "E07", "E07"},
		{"epic lowercase", "e07", "E07"},
		{"feature key", "e07-f01", "E07-F01"},
		{"task key short", "e07-f01-001", "T-E07-F01-001"},
		{"task key full", "t-e07-f01-001", "T-E07-F01-001"},
		{"task key with slug", "e07-f01-001-my-task", "T-E07-F01-001"},
		{"epic with slug", "e07-user-management", "E07"},
		{"feature with slug", "e07-f01-auth-module", "E07-F01"},
		{"feature suffix", "f01", "F01"},
		{"feature suffix with slug", "f01-my-feature", "F01"},
		{"empty returns empty", "", ""},
		{"unknown returns uppercase", "hello", "HELLO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Normalize(tt.key)
			if got != tt.want {
				t.Errorf("Normalize(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_IsValid(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		// Valid keys
		{"valid epic", "E07", true},
		{"valid epic lowercase", "e07", true},
		{"valid feature", "E07-F01", true},
		{"valid feature suffix", "F01", true},
		{"valid task short", "E07-F01-001", true},
		{"valid task full", "T-E07-F01-001", true},
		{"valid epic with slug", "E07-user-management", true},
		{"valid feature with slug", "E07-F01-auth", true},
		{"valid task with slug", "E07-F01-001-impl", true},

		// Invalid keys
		{"empty string", "", false},
		{"random text", "hello", false},
		{"invalid epic", "E1", false},
		{"trailing dash", "E07-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.IsValid(tt.key)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_Format(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name   string
		parsed ParsedKey
		want   string
	}{
		{
			name: "format epic",
			parsed: ParsedKey{
				EntityType: EntityTypeEpic,
				EpicNum:    "07",
			},
			want: "E07",
		},
		{
			name: "format feature with epic",
			parsed: ParsedKey{
				EntityType: EntityTypeFeature,
				EpicNum:    "07",
				FeatureNum: "01",
			},
			want: "E07-F01",
		},
		{
			name: "format feature suffix only",
			parsed: ParsedKey{
				EntityType: EntityTypeFeature,
				EpicNum:    "",
				FeatureNum: "01",
			},
			want: "F01",
		},
		{
			name: "format task",
			parsed: ParsedKey{
				EntityType: EntityTypeTask,
				EpicNum:    "07",
				FeatureNum: "01",
				TaskNum:    "001",
			},
			want: "T-E07-F01-001",
		},
		{
			name: "format unknown returns empty",
			parsed: ParsedKey{
				EntityType: EntityTypeUnknown,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Format(tt.parsed)
			if got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKeyService_NormalizeTaskKey(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"full format", "T-E07-F01-001", "T-E07-F01-001"},
		{"short format", "E07-F01-001", "T-E07-F01-001"},
		{"lowercase short", "e07-f01-001", "T-E07-F01-001"},
		{"lowercase full", "t-e07-f01-001", "T-E07-F01-001"},
		{"with slug short", "E07-F01-001-task-name", "T-E07-F01-001"},
		{"with slug full", "T-E07-F01-001-task-name", "T-E07-F01-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.NormalizeTaskKey(tt.key)
			if got != tt.want {
				t.Errorf("NormalizeTaskKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_Parse_RoundTrip(t *testing.T) {
	ks := NewKeyService()

	// Test that Parse -> Format produces the normalized key
	tests := []struct {
		name    string
		key     string
		wantFmt string
	}{
		{"epic", "E07", "E07"},
		{"epic lowercase", "e07", "E07"},
		{"feature", "E07-F01", "E07-F01"},
		{"feature lowercase", "e07-f01", "E07-F01"},
		{"feature suffix", "F01", "F01"},
		{"task short", "E07-F01-001", "T-E07-F01-001"},
		{"task full", "T-E07-F01-001", "T-E07-F01-001"},
		{"task lowercase", "e07-f01-001", "T-E07-F01-001"},
		// Slugged keys - Format returns the numeric-only form
		{"epic with slug", "E07-user-management", "E07"},
		{"feature with slug", "E07-F01-auth", "E07-F01"},
		{"task with slug", "E07-F01-001-impl", "T-E07-F01-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := ks.Parse(tt.key)
			formatted := ks.Format(parsed)
			if formatted != tt.wantFmt {
				t.Errorf("Format(Parse(%q)) = %q, want %q", tt.key, formatted, tt.wantFmt)
			}
		})
	}
}

func TestKeyService_DetectEntityType_AmbiguousCases(t *testing.T) {
	ks := NewKeyService()

	// These test cases verify priority ordering in detection:
	// task > feature > epic for ambiguous patterns
	tests := []struct {
		name string
		key  string
		want EntityType
	}{
		// E07-F01-001 is a short task key, NOT a feature with slug "001"
		{"short task not feature with numeric slug", "E07-F01-001", EntityTypeTask},
		// E07-F01 is a feature key, not an epic with slug "F01"
		{"feature not epic with F-slug", "E07-F01", EntityTypeFeature},
		// F01 is a feature suffix
		{"feature suffix", "F01", EntityTypeFeature},
		// E07-something is an epic with slug (not starting with F##)
		{"epic with non-feature slug", "E07-something", EntityTypeEpic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.DetectEntityType(tt.key)
			if got != tt.want {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_DetectEntityType_BugAndChange(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want EntityType
	}{
		// Bug keys: B followed by 1+ digits (variable digit count)
		{"bug 3 digits uppercase", "B001", EntityTypeBug},
		{"bug 3 digits lowercase", "b001", EntityTypeBug},
		{"bug 1 digit", "B1", EntityTypeBug},
		{"bug 2 digits", "B42", EntityTypeBug},
		{"bug 4 digits", "B1000", EntityTypeBug},
		{"bug zero is valid", "B0", EntityTypeBug},

		// Change keys: C followed by 1+ digits (variable digit count)
		{"change 3 digits uppercase", "C001", EntityTypeChange},
		{"change 3 digits lowercase", "c001", EntityTypeChange},
		{"change 2 digits lowercase", "c015", EntityTypeChange},
		{"change 1 digit", "C1", EntityTypeChange},

		// Invalid bug/change keys
		{"bug no digits", "B", EntityTypeUnknown},
		{"change no digits", "C", EntityTypeUnknown},
		{"bug alpha chars", "BABC", EntityTypeUnknown},

		// Must not interfere with existing entity types
		{"epic still works", "E07", EntityTypeEpic},
		{"feature still works", "F01", EntityTypeFeature},
		{"task still works", "T-E07-F01-001", EntityTypeTask},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.DetectEntityType(tt.key)
			if got != tt.want {
				t.Errorf("DetectEntityType(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestKeyService_Parse_Bug(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name           string
		key            string
		wantEntityType EntityType
		wantNormalized string
		wantNum        string
	}{
		{
			name:           "bug 3 digits uppercase",
			key:            "B001",
			wantEntityType: EntityTypeBug,
			wantNormalized: "B001",
			wantNum:        "001",
		},
		{
			name:           "bug 3 digits lowercase",
			key:            "b001",
			wantEntityType: EntityTypeBug,
			wantNormalized: "B001",
			wantNum:        "001",
		},
		{
			name:           "bug 1 digit",
			key:            "B1",
			wantEntityType: EntityTypeBug,
			wantNormalized: "B1",
			wantNum:        "1",
		},
		{
			name:           "bug 4 digits",
			key:            "B1000",
			wantEntityType: EntityTypeBug,
			wantNormalized: "B1000",
			wantNum:        "1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != tt.wantEntityType {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, tt.wantEntityType)
			}
			if got.Normalized != tt.wantNormalized {
				t.Errorf("Parse(%q).Normalized = %q, want %q", tt.key, got.Normalized, tt.wantNormalized)
			}
			if got.BugNum != tt.wantNum {
				t.Errorf("Parse(%q).BugNum = %q, want %q", tt.key, got.BugNum, tt.wantNum)
			}
			if got.Raw != tt.key {
				t.Errorf("Parse(%q).Raw = %q, want %q", tt.key, got.Raw, tt.key)
			}
		})
	}
}

func TestKeyService_Parse_Change(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name           string
		key            string
		wantEntityType EntityType
		wantNormalized string
		wantNum        string
	}{
		{
			name:           "change 3 digits uppercase",
			key:            "C001",
			wantEntityType: EntityTypeChange,
			wantNormalized: "C001",
			wantNum:        "001",
		},
		{
			name:           "change 3 digits lowercase",
			key:            "c001",
			wantEntityType: EntityTypeChange,
			wantNormalized: "C001",
			wantNum:        "001",
		},
		{
			name:           "change 2 digits lowercase",
			key:            "c015",
			wantEntityType: EntityTypeChange,
			wantNormalized: "C015",
			wantNum:        "015",
		},
		{
			name:           "change 1 digit",
			key:            "C1",
			wantEntityType: EntityTypeChange,
			wantNormalized: "C1",
			wantNum:        "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Parse(tt.key)
			if got.EntityType != tt.wantEntityType {
				t.Errorf("Parse(%q).EntityType = %q, want %q", tt.key, got.EntityType, tt.wantEntityType)
			}
			if got.Normalized != tt.wantNormalized {
				t.Errorf("Parse(%q).Normalized = %q, want %q", tt.key, got.Normalized, tt.wantNormalized)
			}
			if got.ChangeNum != tt.wantNum {
				t.Errorf("Parse(%q).ChangeNum = %q, want %q", tt.key, got.ChangeNum, tt.wantNum)
			}
			if got.Raw != tt.key {
				t.Errorf("Parse(%q).Raw = %q, want %q", tt.key, got.Raw, tt.key)
			}
		})
	}
}

func TestKeyService_Format_BugAndChange(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name   string
		parsed ParsedKey
		want   string
	}{
		{
			name: "format bug",
			parsed: ParsedKey{
				EntityType: EntityTypeBug,
				BugNum:     "001",
			},
			want: "B001",
		},
		{
			name: "format bug 1 digit",
			parsed: ParsedKey{
				EntityType: EntityTypeBug,
				BugNum:     "1",
			},
			want: "B1",
		},
		{
			name: "format change",
			parsed: ParsedKey{
				EntityType: EntityTypeChange,
				ChangeNum:  "001",
			},
			want: "C001",
		},
		{
			name: "format change 1 digit",
			parsed: ParsedKey{
				EntityType: EntityTypeChange,
				ChangeNum:  "1",
			},
			want: "C1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.Format(tt.parsed)
			if got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestKeyService_IsValid_BugAndChange(t *testing.T) {
	ks := NewKeyService()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{"valid bug 3 digits", "B001", true},
		{"valid bug lowercase", "b001", true},
		{"valid bug 1 digit", "B1", true},
		{"valid change 3 digits", "C001", true},
		{"valid change lowercase", "c001", true},
		{"valid change 1 digit", "C1", true},
		{"invalid bug no digits", "B", false},
		{"invalid change no digits", "C", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ks.IsValid(tt.key)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
