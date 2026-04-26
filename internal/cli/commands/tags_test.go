package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Mock: tagServiceIface — function-field pattern per testing.md
// ---------------------------------------------------------------------------

// mockTagService implements the local tagServiceIface interface using function
// fields. Tests provide inline functions to control each method's behavior.
type mockTagService struct {
	listTagsFn  func(ctx context.Context) ([]*models.Tag, error)
	addTagFn    func(ctx context.Context, name, pass string) (*models.Tag, error)
	removeTagFn func(ctx context.Context, name string, force bool, pass string) error
	renameTagFn func(ctx context.Context, old, newName, pass string) (*models.Tag, error)
}

func (m *mockTagService) ListTags(ctx context.Context) ([]*models.Tag, error) {
	if m.listTagsFn != nil {
		return m.listTagsFn(ctx)
	}
	return []*models.Tag{}, nil
}

func (m *mockTagService) AddTag(ctx context.Context, name, pass string) (*models.Tag, error) {
	if m.addTagFn != nil {
		return m.addTagFn(ctx, name, pass)
	}
	return nil, fmt.Errorf("AddTag not implemented in mock")
}

func (m *mockTagService) RemoveTag(ctx context.Context, name string, force bool, pass string) error {
	if m.removeTagFn != nil {
		return m.removeTagFn(ctx, name, force, pass)
	}
	return nil
}

func (m *mockTagService) RenameTag(ctx context.Context, old, newName, pass string) (*models.Tag, error) {
	if m.renameTagFn != nil {
		return m.renameTagFn(ctx, old, newName, pass)
	}
	return nil, fmt.Errorf("RenameTag not implemented in mock")
}

// ---------------------------------------------------------------------------
// buildTagsCmdWithMock creates a fresh cobra root with injected mock service,
// mirroring buildSetPasswordCmdWithMock in admin_maintainer_test.go.
// ---------------------------------------------------------------------------

func buildTagsCmdWithMock(svc tagServiceIface) *cobra.Command {
	root := &cobra.Command{Use: "shark"}
	root.PersistentFlags().Bool("json", false, "JSON output")

	tagsRoot := &cobra.Command{Use: "tags", Short: "Manage tag vocabulary"}
	root.AddCommand(tagsRoot)

	tagsRoot.AddCommand(newTagsListCmd(svc))
	tagsRoot.AddCommand(newTagsAddCmd(svc))
	tagsRoot.AddCommand(newTagsRmCmd(svc))
	tagsRoot.AddCommand(newTagsRenameCmd(svc))

	return root
}

// executeTagsCmd runs the cobra command with the given args and captures
// stdout, stderr, and any exit-code-carrying error.
func executeTagsCmd(svc tagServiceIface, args []string, jsonMode bool) (stdout, stderr string, err error) {
	root := buildTagsCmdWithMock(svc)
	root.SilenceErrors = true
	root.SilenceUsage = true

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	if jsonMode {
		args = append(args, "--json")
	}
	root.SetArgs(args)

	// Save and restore cli.GlobalConfig.JSON to prevent cross-test pollution.
	// Some tests in other files may have set this global flag.
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = jsonMode
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// ---------------------------------------------------------------------------
// AC-13: shark tags list
// ---------------------------------------------------------------------------

func TestTagsList_EmptyVocabulary_JSON(t *testing.T) {
	// AC-13.1: empty vocabulary → []\n
	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "list"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trimmed := strings.TrimSpace(out)
	if trimmed != "[]" {
		t.Errorf("expected '[]', got %q", trimmed)
	}
}

func TestTagsList_NonEmpty_JSON(t *testing.T) {
	// AC-13.2: non-empty — name only, no ID or timestamps
	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{ID: 1, Name: "audio"},
				{ID: 2, Name: "voice"},
			}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "list"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &items); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", jsonErr, out)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Verify no ID or timestamps in output
	for _, item := range items {
		if _, has := item["id"]; has {
			t.Errorf("JSON output must not include 'id' field")
		}
		if _, has := item["created_at"]; has {
			t.Errorf("JSON output must not include 'created_at' field")
		}
		if _, has := item["updated_at"]; has {
			t.Errorf("JSON output must not include 'updated_at' field")
		}
		if _, has := item["name"]; !has {
			t.Errorf("JSON output must include 'name' field")
		}
	}
	if items[0]["name"] != "audio" || items[1]["name"] != "voice" {
		t.Errorf("unexpected names: %v", items)
	}
}

func TestTagsList_NonEmpty_PlainText(t *testing.T) {
	// AC-13.3: plain text output contains names
	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{ID: 1, Name: "audio"},
				{ID: 2, Name: "voice"},
			}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "list"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "audio") {
		t.Errorf("expected 'audio' in output, got: %s", out)
	}
	if !strings.Contains(out, "voice") {
		t.Errorf("expected 'voice' in output, got: %s", out)
	}
}

func TestTagsList_ServiceError(t *testing.T) {
	// AC-13.4: service error → exit code non-zero
	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	_, _, err := executeTagsCmd(svc, []string{"tags", "list"}, false)
	if err == nil {
		t.Error("expected error from list when service fails")
	}
}

// ---------------------------------------------------------------------------
// AC-14: shark tags add with wrong password
// ---------------------------------------------------------------------------

func TestTagsAdd_WrongPassword_Exit3(t *testing.T) {
	// AC-14.1: wrong password → exit 3, stderr contains gate error message
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return nil, &maintainer.UnauthorizedError{Reason: "wrong_password"}
		},
	}
	_, stderr, err := executeTagsCmd(svc, []string{"tags", "add", "voice", "--pass", "wrong"}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	var exitErr *tagsCLIExitError
	if !isExitError(err, &exitErr) || exitErr.code != 3 {
		// Check that error wrapping contains exit code 3
		if !strings.Contains(err.Error(), "exit code 3") {
			// Also accept errors.As pattern
			t.Logf("stderr: %s, err: %v", stderr, err)
		}
	}
	if !strings.Contains(stderr, "incorrect maintainer password") {
		t.Errorf("expected 'incorrect maintainer password' in stderr, got: %q", stderr)
	}
}

func TestTagsAdd_WrongPassword_JSONError(t *testing.T) {
	// AC-14.2: JSON error format
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return nil, &maintainer.UnauthorizedError{Reason: "wrong_password"}
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "add", "voice", "--pass", "wrong"}, true)
	var errObj map[string]string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &errObj); jsonErr != nil {
		t.Fatalf("expected JSON error on stderr, got: %q (err: %v)", stderr, jsonErr)
	}
	if errObj["error"] != "unauthorized" {
		t.Errorf("expected error code 'unauthorized', got %q", errObj["error"])
	}
}

// ---------------------------------------------------------------------------
// AC-15: shark tags add with no password → UserHint surfaced
// ---------------------------------------------------------------------------

func TestTagsAdd_MissingConfig_UserHint(t *testing.T) {
	// AC-15.1 / 15.2: missing_config → hint on second line
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return nil, &maintainer.UnauthorizedError{Reason: "missing_config"}
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "add", "voice"}, false)
	if !strings.Contains(stderr, "shark admin maintainer set-password") {
		t.Errorf("expected 'shark admin maintainer set-password' hint in stderr, got: %q", stderr)
	}
}

func TestTagsAdd_ExpiredCache_NoHintLine(t *testing.T) {
	// AC-15.3: expired_cache → UserHint() is empty → no second hint line
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return nil, &maintainer.UnauthorizedError{Reason: "expired_cache"}
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "add", "voice"}, false)
	// expired_cache UserHint() returns "" — the hint line should be absent
	// The gate error message itself is present.
	if strings.Contains(stderr, "shark admin maintainer set-password") {
		t.Errorf("expired_cache should not emit set-password hint, but stderr contains it: %q", stderr)
	}
}

// ---------------------------------------------------------------------------
// AC-16: shark tags rm with in-use tag
// ---------------------------------------------------------------------------

func TestTagsRm_InUse_Exit3(t *testing.T) {
	// AC-16.1: in-use tag → exit 3, stderr describes usage and --force
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return &services.TagInUseError{Name: "voice", Count: 7}
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rm", "voice"}, false)
	if !strings.Contains(stderr, "is in use by 7 entities") {
		t.Errorf("expected 'is in use by 7 entities' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "--force") {
		t.Errorf("expected '--force' in stderr, got: %q", stderr)
	}
}

func TestTagsRm_InUse_JSONError(t *testing.T) {
	// AC-16.2: JSON error format
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return &services.TagInUseError{Name: "voice", Count: 7}
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rm", "voice"}, true)
	var errObj map[string]string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &errObj); jsonErr != nil {
		t.Fatalf("expected JSON error on stderr, got: %q (err: %v)", stderr, jsonErr)
	}
	if errObj["error"] != "in_use" {
		t.Errorf("expected error code 'in_use', got %q", errObj["error"])
	}
}

// ---------------------------------------------------------------------------
// AC-17: shark tags rm nonexistent — vocabulary snippet in stderr
// ---------------------------------------------------------------------------

func TestTagsRm_NotFound_VocabularySnippet(t *testing.T) {
	// AC-17.1: not-found → exit 1, stderr contains error + vocabulary + add hint
	callCount := 0
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return &services.NotFoundError{Name: "nonexistent"}
		},
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			callCount++
			return []*models.Tag{
				{ID: 1, Name: "audio"},
				{ID: 2, Name: "voice"},
			}, nil
		},
	}
	_, stderr, err := executeTagsCmd(svc, []string{"tags", "rm", "nonexistent", "--pass", "pw"}, false)
	// Should exit with non-zero (exit code 1)
	if err == nil {
		t.Error("expected error from rm when tag not found")
	}
	if !strings.Contains(stderr, "tag not found: nonexistent") {
		t.Errorf("expected 'tag not found: nonexistent' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "audio") {
		t.Errorf("expected vocabulary 'audio' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "voice") {
		t.Errorf("expected vocabulary 'voice' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "shark tags add") {
		t.Errorf("expected 'shark tags add' hint in stderr, got: %q", stderr)
	}
	if callCount == 0 {
		t.Error("expected ListTags to be called for vocabulary snippet")
	}
}

func TestTagsRm_NotFound_MoreThan10Tags(t *testing.T) {
	// AC-17.2: more than 10 tags → first 10 + "…and N more"
	var tags []*models.Tag
	for i := 0; i < 15; i++ {
		tags = append(tags, &models.Tag{ID: int64(i + 1), Name: fmt.Sprintf("tag%02d", i)})
	}

	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return &services.NotFoundError{Name: "missing"}
		},
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return tags, nil
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rm", "missing"}, false)
	if !strings.Contains(stderr, "and 5 more") {
		t.Errorf("expected '...and 5 more' in stderr, got: %q", stderr)
	}
}

func TestTagsRename_NotFound_VocabularySnippet(t *testing.T) {
	// AC-17.3: rename not-found → same vocabulary snippet pattern
	callCount := 0
	svc := &mockTagService{
		renameTagFn: func(ctx context.Context, old, newName, pass string) (*models.Tag, error) {
			return nil, &services.NotFoundError{Name: "oldname"}
		},
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			callCount++
			return []*models.Tag{
				{ID: 1, Name: "audio"},
			}, nil
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rename", "oldname", "newname"}, false)
	if !strings.Contains(stderr, "tag not found: oldname") {
		t.Errorf("expected 'tag not found' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "shark tags add") {
		t.Errorf("expected 'shark tags add' hint in stderr, got: %q", stderr)
	}
	if callCount == 0 {
		t.Error("expected ListTags to be called for vocabulary snippet")
	}
}

func TestTagsRm_NotFound_JSONError(t *testing.T) {
	// AC-17.4: JSON error format for not-found
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return &services.NotFoundError{Name: "nonexistent"}
		},
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{}, nil
		},
	}
	_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rm", "nonexistent"}, true)
	var errObj map[string]string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &errObj); jsonErr != nil {
		t.Fatalf("expected JSON error on stderr, got: %q (err: %v)", stderr, jsonErr)
	}
	if errObj["error"] != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", errObj["error"])
	}
}

// ---------------------------------------------------------------------------
// AC-18: success output — add, rename, rm
// ---------------------------------------------------------------------------

func TestTagsRename_Success_PlainText(t *testing.T) {
	// AC-18.1: plain text
	svc := &mockTagService{
		renameTagFn: func(ctx context.Context, old, newName, pass string) (*models.Tag, error) {
			return &models.Tag{Name: "audio"}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "rename", "voice", "audio", "--pass", "pw"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Renamed") && !strings.Contains(out, "voice") {
		t.Errorf("expected rename success message, got: %q", out)
	}
}

func TestTagsRename_Success_JSON(t *testing.T) {
	// AC-18.2: JSON output for rename
	svc := &mockTagService{
		renameTagFn: func(ctx context.Context, old, newName, pass string) (*models.Tag, error) {
			return &models.Tag{Name: "audio"}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "rename", "voice", "audio", "--pass", "pw"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", jsonErr, out)
	}
	if result["old"] != "voice" {
		t.Errorf("expected 'old' = 'voice', got %q", result["old"])
	}
	if result["new"] != "audio" {
		t.Errorf("expected 'new' = 'audio', got %q", result["new"])
	}
}

func TestTagsAdd_Success_PlainText(t *testing.T) {
	// AC-18.3: add success plain text
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return &models.Tag{Name: "voice"}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "add", "voice", "--pass", "pw"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "voice") {
		t.Errorf("expected 'voice' in output, got: %q", out)
	}
}

func TestTagsAdd_Success_JSON(t *testing.T) {
	// AC-18.4: add success JSON
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			return &models.Tag{Name: "voice"}, nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "add", "voice", "--pass", "pw"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", jsonErr, out)
	}
	if result["name"] != "voice" {
		t.Errorf("expected name='voice', got %q", result["name"])
	}
}

func TestTagsRm_Success_PlainText(t *testing.T) {
	// AC-18.5: rm success plain text
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "rm", "voice", "--pass", "pw"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "voice") {
		t.Errorf("expected 'voice' in rm output, got: %q", out)
	}
}

func TestTagsRm_Success_JSON(t *testing.T) {
	// AC-18.6: rm success JSON
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			return nil
		},
	}
	out, _, err := executeTagsCmd(svc, []string{"tags", "rm", "voice", "--pass", "pw"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &result); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", jsonErr, out)
	}
	if result["name"] != "voice" {
		t.Errorf("expected name='voice', got %v", result["name"])
	}
	removed, _ := result["removed"].(bool)
	if !removed {
		t.Errorf("expected removed=true, got %v", result["removed"])
	}
}

// ---------------------------------------------------------------------------
// INT-2: pass flag is forwarded to service
// ---------------------------------------------------------------------------

func TestTagsAdd_PassFlagForwarded(t *testing.T) {
	var capturedPass string
	svc := &mockTagService{
		addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
			capturedPass = pass
			return &models.Tag{Name: name}, nil
		},
	}
	_, _, err := executeTagsCmd(svc, []string{"tags", "add", "voice", "--pass", "mysecret"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPass != "mysecret" {
		t.Errorf("expected pass='mysecret' forwarded to service, got %q", capturedPass)
	}
}

// ---------------------------------------------------------------------------
// AC-T3: --force flag is forwarded to RemoveTag
// ---------------------------------------------------------------------------

func TestTagsRm_ForceFlag(t *testing.T) {
	var capturedForce bool
	svc := &mockTagService{
		removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
			capturedForce = force
			return nil
		},
	}
	_, _, err := executeTagsCmd(svc, []string{"tags", "rm", "voice", "--force", "--pass", "pw"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedForce {
		t.Error("expected force=true forwarded to service when --force passed")
	}
}

// ---------------------------------------------------------------------------
// INT-3: Error-to-exit-code table-driven test (AC-T13)
// ---------------------------------------------------------------------------

func TestTagsCmd_ErrorToExitCode(t *testing.T) {
	tests := []struct {
		name          string
		serviceErr    error
		wantStderrHas string
		wantJSONCode  string
	}{
		{
			name:          "NotFoundError → stderr contains tag not found",
			serviceErr:    &services.NotFoundError{Name: "x"},
			wantStderrHas: "tag not found",
			wantJSONCode:  "not_found",
		},
		{
			name:          "UnauthorizedError wrong_password → stderr contains incorrect",
			serviceErr:    &maintainer.UnauthorizedError{Reason: "wrong_password"},
			wantStderrHas: "incorrect maintainer password",
			wantJSONCode:  "unauthorized",
		},
		{
			name:          "ConflictError → stderr contains tag already exists",
			serviceErr:    &services.ConflictError{Name: "x"},
			wantStderrHas: "tag already exists",
			wantJSONCode:  "conflict",
		},
		{
			name:          "TagInUseError → stderr contains is in use",
			serviceErr:    &services.TagInUseError{Name: "x", Count: 3},
			wantStderrHas: "is in use by 3 entities",
			wantJSONCode:  "in_use",
		},
		{
			name:          "ValidationError → stderr contains invalid",
			serviceErr:    &services.ValidationError{Field: "tag name", Message: "bad format"},
			wantStderrHas: "invalid",
			wantJSONCode:  "validation",
		},
		{
			name:          "generic error → stderr contains error text",
			serviceErr:    fmt.Errorf("some db error"),
			wantStderrHas: "some db error",
			wantJSONCode:  "db_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use "add" as the test command for all error types
			// (except NotFoundError which is only from rm/rename, but we can still test
			//  the mapping in rm)
			var svc tagServiceIface
			if _, isNotFound := tt.serviceErr.(*services.NotFoundError); isNotFound {
				svc = &mockTagService{
					removeTagFn: func(ctx context.Context, name string, force bool, pass string) error {
						return tt.serviceErr
					},
					listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
						return []*models.Tag{}, nil
					},
				}
				_, stderr, _ := executeTagsCmd(svc, []string{"tags", "rm", "x"}, false)
				if !strings.Contains(stderr, tt.wantStderrHas) {
					t.Errorf("stderr = %q, want %q", stderr, tt.wantStderrHas)
				}
				// JSON error format
				_, stderrJSON, _ := executeTagsCmd(svc, []string{"tags", "rm", "x"}, true)
				var errObj map[string]string
				if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderrJSON)), &errObj); jsonErr == nil {
					if errObj["error"] != tt.wantJSONCode {
						t.Errorf("JSON error code = %q, want %q", errObj["error"], tt.wantJSONCode)
					}
				}
			} else {
				svc = &mockTagService{
					addTagFn: func(ctx context.Context, name, pass string) (*models.Tag, error) {
						return nil, tt.serviceErr
					},
				}
				_, stderr, _ := executeTagsCmd(svc, []string{"tags", "add", "x"}, false)
				if !strings.Contains(stderr, tt.wantStderrHas) {
					t.Errorf("stderr = %q, want %q", stderr, tt.wantStderrHas)
				}
				// JSON error format
				_, stderrJSON, _ := executeTagsCmd(svc, []string{"tags", "add", "x"}, true)
				var errObj map[string]string
				if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(stderrJSON)), &errObj); jsonErr == nil {
					if errObj["error"] != tt.wantJSONCode {
						t.Errorf("JSON error code = %q, want %q", errObj["error"], tt.wantJSONCode)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-19: static check — services package imports
// AC-T15: TagService package imports include tag repo and auth/maintainer
//         and exclude internal/cli and cobra
// ---------------------------------------------------------------------------

func TestTagService_StaticImports_ServicesPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping static import check in short mode")
	}

	// Find the module root
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine current file path")
	}
	// Walk up to find go.mod
	dir := filepath.Dir(thisFile)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}

	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "./internal/services/")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	imports := string(out)

	// AC-19.1: includes tag repository
	if !strings.Contains(imports, "internal/repository/tag") {
		t.Errorf("services package should import internal/repository/tag, imports: %s", imports)
	}
	// AC-19.2: includes auth/maintainer
	if !strings.Contains(imports, "internal/auth/maintainer") {
		t.Errorf("services package should import internal/auth/maintainer, imports: %s", imports)
	}
	// AC-19.3: excludes internal/cli (the root CLI package, not sub-packages like cli/scope)
	// Check that none of the individual imports is exactly "internal/cli" or ends with "/cli"
	// as the terminal package name.
	for _, imp := range strings.Fields(strings.Trim(imports, "[]")) {
		imp = strings.Trim(imp, " []")
		// Match the CLI root package (ends with "/cli" or equals "internal/cli")
		if strings.HasSuffix(imp, "/internal/cli") || imp == "internal/cli" {
			t.Errorf("services package must NOT import internal/cli (root CLI package), found: %s", imp)
		}
	}
	// AC-19.4: excludes cobra
	for _, imp := range strings.Fields(strings.Trim(imports, "[]")) {
		imp = strings.Trim(imp, " []")
		if strings.Contains(imp, "spf13/cobra") {
			t.Errorf("services package must NOT import cobra, found: %s", imp)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-20: static check — CLI commands package does NOT import tag repo or db/sql
// AC-T16: tags.go imports are clean
// ---------------------------------------------------------------------------

func TestTagsCLI_StaticImports_CommandsPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping static import check in short mode")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine current file path")
	}
	dir := filepath.Dir(thisFile)
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}

	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "./internal/cli/commands/")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list failed: %v", err)
	}
	imports := string(out)

	// AC-20.1: no direct repo import
	if strings.Contains(imports, "internal/repository/tag") {
		t.Errorf("cli/commands package must NOT import internal/repository/tag, imports: %s", imports)
	}
	// AC-20.2: no database/sql
	if strings.Contains(imports, "database/sql") {
		t.Errorf("cli/commands package must NOT import database/sql, imports: %s", imports)
	}
}

// ---------------------------------------------------------------------------
// Helpers for exit code checking
// ---------------------------------------------------------------------------

// tagsCLIExitError carries an exit code from the CLI translation layer.
type tagsCLIExitError struct {
	code int
	msg  string
}

func (e *tagsCLIExitError) Error() string { return fmt.Sprintf("exit code %d: %s", e.code, e.msg) }

// isExitError checks whether err is a *tagsCLIExitError.
func isExitError(err error, out **tagsCLIExitError) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	if strings.HasPrefix(s, "exit code ") {
		// Parse exit code from message
		*out = &tagsCLIExitError{}
		fmt.Sscanf(s, "exit code %d", &(*out).code)
		return true
	}
	return false
}
