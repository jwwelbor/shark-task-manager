package services

import (
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// FeatureUpdates contains fields that can be updated on an existing feature.
// Only non-nil pointer fields will be updated.
type FeatureUpdates struct {
	Title          *string               `json:"title,omitempty"`
	Description    *string               `json:"description,omitempty"`
	Status         *models.FeatureStatus `json:"status,omitempty"`
	ExecutionOrder *int                  `json:"execution_order,omitempty"`
	FilePath       *string               `json:"file_path,omitempty"`

	// E28-F04 REQ-F-010: Tags to attach additively on update. Empty/nil
	// means no tag change (see AC-18b). Removal on update is explicitly
	// NOT supported — use `shark feature tag rm` (REQ-F-014).
	Tags []string `json:"tags,omitempty"`
	// Size updates the size when non-nil. Use models.ParseSize to convert
	// t-shirt labels before setting. E07-F42 REQ-F-005.
	Size *int `json:"size,omitempty"`
	// ClearSize when true sets the feature's size to NULL regardless of the
	// Size field value. ClearSize takes precedence over Size.
	// Corresponds to `--size clear` on the CLI. E07-F42 REQ-F-005.
	ClearSize bool `json:"clear_size,omitempty"`
	// SkipResequence, when true, applies an ExecutionOrder change without
	// renumbering sibling features. Enables intentional duplicate-order
	// groups (parallel work). Wired from `--parallel` on
	// `shark feature update`. Has no effect when ExecutionOrder is nil.
	SkipResequence bool `json:"skip_resequence,omitempty"`
}
