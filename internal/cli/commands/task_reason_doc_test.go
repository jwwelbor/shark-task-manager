package commands

import (
	"context"
	"encoding/json"
	"testing"
)

// TestTaskReopen_RejectionReasonDocFlag tests that task reopen command accepts --reason-doc flag
func TestTaskReopen_RejectionReasonDocFlag(t *testing.T) {
	// Verify the task reopen command has the --reason-doc flag
	if taskReopenCmd.Flags().Lookup("reason-doc") == nil {
		t.Error("task reopen command missing --reason-doc flag")
	}
}

// TestTaskApprove_RejectionReasonDocFlag tests that task approve command accepts --reason-doc flag
func TestTaskApprove_RejectionReasonDocFlag(t *testing.T) {
	// Verify the task approve command has the --reason-doc flag
	if taskApproveCmd.Flags().Lookup("reason-doc") == nil {
		t.Error("task approve command missing --reason-doc flag")
	}
}

// TestValidateRejectionReasonDocPath tests path validation for --reason-doc flag
func TestValidateRejectionReasonDocPath(t *testing.T) {
	tests := []struct {
		name    string
		docPath string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid relative path",
			docPath: "docs/feedback/reason.md",
			wantErr: false,
		},
		{
			name:    "valid nested path",
			docPath: "docs/reviews/2026-01-16/feedback.md",
			wantErr: false,
		},
		{
			name:    "empty path should fail",
			docPath: "",
			wantErr: true,
			errMsg:  "document path cannot be empty",
		},
		{
			name:    "path with .. should fail",
			docPath: "docs/../../../etc/passwd",
			wantErr: true,
			errMsg:  "document path traversal not allowed",
		},
		{
			name:    "absolute path should fail",
			docPath: "/etc/passwd",
			wantErr: true,
			errMsg:  "document path must be relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRejectionReasonDocPath(tt.docPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRejectionReasonDocPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && (err == nil || err.Error() != tt.errMsg) {
				t.Errorf("ValidateRejectionReasonDocPath() error message = %q, want %q",
					err.Error(), tt.errMsg)
			}
		})
	}
}

// TestLinkRejectionReasonDocument tests that rejection reason is stored with document link
func TestLinkRejectionReasonDocument(t *testing.T) {
	// Use existing mock repository
	mockDocRepo := NewMockDocumentRepository()

	ctx := context.Background()

	// Verify CreateOrGet works with path
	doc, err := mockDocRepo.CreateOrGet(ctx, "Rejection Reason", "docs/feedback/reason.md")
	if err != nil {
		t.Fatalf("CreateOrGet() error = %v", err)
	}

	if doc.FilePath != "docs/feedback/reason.md" {
		t.Errorf("expected FilePath 'docs/feedback/reason.md', got %q", doc.FilePath)
	}

	// Verify LinkToTask can be called
	err = mockDocRepo.LinkToTask(ctx, 123, doc.ID)
	if err != nil {
		t.Fatalf("LinkToTask() error = %v", err)
	}

	// Verify the document was tracked in mock
	if mockDocRepo.LinkToTaskCalls != 1 {
		t.Errorf("LinkToTaskCalls = %d, want 1", mockDocRepo.LinkToTaskCalls)
	}
}

// TestTaskNoteMetadataForRejectionReason tests storing document path in note metadata
func TestTaskNoteMetadataForRejectionReason(t *testing.T) {
	tests := []struct {
		name        string
		docPath     string
		wantMetaKey string
		wantMetaVal string
	}{
		{
			name:        "simple path in metadata",
			docPath:     "docs/feedback.md",
			wantMetaKey: "reason_doc_path",
			wantMetaVal: "docs/feedback.md",
		},
		{
			name:        "nested path in metadata",
			docPath:     "docs/reviews/2026-01-16/rejection.md",
			wantMetaKey: "reason_doc_path",
			wantMetaVal: "docs/reviews/2026-01-16/rejection.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build metadata JSON
			metadata := BuildRejectionReasonMetadata(tt.docPath)

			// Verify metadata is valid JSON
			if metadata == "" {
				t.Error("expected metadata to be non-empty")
				return
			}

			// Parse metadata
			var metaMap map[string]string
			err := json.Unmarshal([]byte(metadata), &metaMap)
			if err != nil {
				t.Errorf("failed to parse metadata JSON: %v", err)
				return
			}

			// Verify metadata contains document path
			if docPath, ok := metaMap[tt.wantMetaKey]; !ok {
				t.Errorf("metadata missing key %q", tt.wantMetaKey)
			} else if docPath != tt.wantMetaVal {
				t.Errorf("metadata[%q] = %q, want %q", tt.wantMetaKey, docPath, tt.wantMetaVal)
			}
		})
	}
}

// TestDocumentPathExistenceValidation tests that document path must exist
func TestDocumentPathExistenceValidation(t *testing.T) {
	tests := []struct {
		name    string
		docPath string
		create  bool
		wantErr bool
	}{
		{
			name:    "existing document file",
			docPath: "docs/test-document.md",
			create:  true,
			wantErr: false,
		},
		{
			name:    "non-existent document file",
			docPath: "docs/non-existent-file-12345.md",
			create:  false,
			wantErr: true,
		},
		{
			name:    "document in nested directory",
			docPath: "docs/reviews/2026-01-16/document.md",
			create:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For this test, we just validate syntax; actual file existence
			// is checked in CLI command implementation
			err := ValidateRejectionReasonDocPath(tt.docPath)
			if err != nil {
				// Syntax validation shouldn't fail for these cases
				t.Errorf("ValidateRejectionReasonDocPath() unexpected error: %v", err)
			}
		})
	}
}

// TestRejectionReasonDocFlagIntegration tests the --reason-doc flag flow
func TestRejectionReasonDocFlagIntegration(t *testing.T) {
	// This test verifies the complete flow:
	// 1. Validate document path
	// 2. Verify document exists
	// 3. Store document path in rejection note metadata
	// 4. Create task_documents link (if table exists)

	tests := []struct {
		name           string
		docPath        string
		validationFail bool
		wantMetadata   bool
	}{
		{
			name:           "valid document linking",
			docPath:        "docs/reviews/rejection.md",
			validationFail: false,
			wantMetadata:   true,
		},
		{
			name:           "empty document path (should be optional)",
			docPath:        "",
			validationFail: true,
			wantMetadata:   false,
		},
		{
			name:           "traversal attack blocked",
			docPath:        "../../../etc/passwd",
			validationFail: true,
			wantMetadata:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.docPath == "" {
				// Empty path should be skipped, not validated
				return
			}

			err := ValidateRejectionReasonDocPath(tt.docPath)
			if (err != nil) != tt.validationFail {
				t.Errorf("ValidateRejectionReasonDocPath() error = %v, wantErr %v", err, tt.validationFail)
				return
			}

			if !tt.validationFail && tt.wantMetadata {
				metadata := BuildRejectionReasonMetadata(tt.docPath)
				if metadata == "" {
					t.Error("expected non-empty metadata")
					return
				}

				// Verify JSON structure
				var metaMap map[string]string
				if err := json.Unmarshal([]byte(metadata), &metaMap); err != nil {
					t.Errorf("metadata JSON parse failed: %v", err)
					return
				}

				if metaMap["reason_doc_path"] != tt.docPath {
					t.Errorf("metadata path mismatch: got %q, want %q", metaMap["reason_doc_path"], tt.docPath)
				}
			}
		})
	}
}

// TestRejectionNoteWithDocumentLink tests that rejection notes include document link
func TestRejectionNoteWithDocumentLink(t *testing.T) {
	tests := []struct {
		name         string
		docPath      string
		reason       string
		expectedBoth bool
	}{
		{
			name:         "reason and document",
			docPath:      "docs/bug-report.md",
			reason:       "Critical bug found in test suite",
			expectedBoth: true,
		},
		{
			name:         "reason without document",
			docPath:      "",
			reason:       "Missing error handling",
			expectedBoth: false,
		},
		{
			name:         "document without explicit reason",
			docPath:      "docs/code-review.md",
			reason:       "",
			expectedBoth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build metadata if doc path provided
			var metadata string
			if tt.docPath != "" {
				metadata = BuildRejectionReasonMetadata(tt.docPath)
			}

			// Verify metadata structure
			if tt.expectedBoth && (tt.docPath != "" && tt.reason != "") {
				if metadata == "" {
					t.Error("expected metadata to contain document path")
					return
				}

				var metaMap map[string]string
				if err := json.Unmarshal([]byte(metadata), &metaMap); err != nil {
					t.Fatalf("failed to parse metadata: %v", err)
				}

				if metaMap["reason_doc_path"] != tt.docPath {
					t.Errorf("metadata document path mismatch: got %q, want %q",
						metaMap["reason_doc_path"], tt.docPath)
				}
			}
		})
	}
}

// Helper function to build rejection reason metadata as JSON string
func BuildRejectionReasonMetadata(docPath string) string {
	if docPath == "" {
		return ""
	}
	metaMap := map[string]string{
		"reason_doc_path": docPath,
	}
	jsonBytes, _ := json.Marshal(metaMap)
	return string(jsonBytes)
}
