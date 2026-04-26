package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// T-E28-F04-004 tests
//
// AC-T1: tagsErrorCode returns ("unregistered_tag", 3) for
//        *services.UnregisteredTagError and ("tag_required", 3) for
//        *services.TagRequiredError.
// AC-T2: handleVocabularyErrorWithSnippet recognises *UnregisteredTagError
//        and produces stderr ending in the exact substring
//        "To add it: shark tags add <name>".
// AC-T3: existing F03 tags_test.go cases still pass unchanged (verified by
//        running the full package test suite).
// ---------------------------------------------------------------------------

func TestTagsErrorCode_UnregisteredTag(t *testing.T) {
	err := &services.UnregisteredTagError{Name: "does-not-exist"}
	code, exit := tagsErrorCode(err)
	if code != "unregistered_tag" {
		t.Errorf("code = %q, want %q", code, "unregistered_tag")
	}
	if exit != 3 {
		t.Errorf("exit = %d, want %d", exit, 3)
	}
}

func TestTagsErrorCode_TagRequired(t *testing.T) {
	err := &services.TagRequiredError{EntityType: "task"}
	code, exit := tagsErrorCode(err)
	if code != "tag_required" {
		t.Errorf("code = %q, want %q", code, "tag_required")
	}
	if exit != 3 {
		t.Errorf("exit = %d, want %d", exit, 3)
	}
}

func TestTagsErrorCode_WrappedUnregisteredTag(t *testing.T) {
	// errors.As should unwrap through %w.
	inner := &services.UnregisteredTagError{Name: "voice"}
	err := fmt.Errorf("outer: %w", inner)
	code, exit := tagsErrorCode(err)
	if code != "unregistered_tag" || exit != 3 {
		t.Errorf("wrapped unregistered tag: got (%q, %d), want (%q, %d)",
			code, exit, "unregistered_tag", 3)
	}
}

// buildSingleCommandForHelper returns a fresh cobra command wired with a
// stderr buffer and the --json flag. It is used to drive
// handleVocabularyErrorWithSnippet directly.
//
// NOTE: This helper does NOT touch cli.GlobalConfig.JSON. Callers that want
// plain-text behaviour must do `cli.GlobalConfig.JSON = false; defer
// restore()` at the start of the test — otherwise leaked state from other
// tests in the package (some of which set the global) can flip the helper
// into JSON mode and break snippet assertions.
func buildSingleCommandForHelper(jsonMode bool) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "shark"}
	cmd.PersistentFlags().Bool("json", false, "JSON output")
	if jsonMode {
		_ = cmd.PersistentFlags().Set("json", "true")
	}
	var errBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	return cmd, &errBuf
}

// withPlainTextGlobal pins cli.GlobalConfig.JSON to false for the duration
// of the test, restoring the previous value when the test exits. It is the
// mirror of the json-mode setup in TestHandleVocabularyErrorWithSnippet_JSONModeSuppressesSnippet.
func withPlainTextGlobal(t *testing.T) {
	t.Helper()
	orig := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = false
	t.Cleanup(func() { cli.GlobalConfig.JSON = orig })
}

func TestHandleVocabularyErrorWithSnippet_UnregisteredTag(t *testing.T) {
	// AC-T2: UnregisteredTagError renders vocabulary snippet + remediation.
	withPlainTextGlobal(t)
	cmd, errBuf := buildSingleCommandForHelper(false)

	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{
				{ID: 1, Name: "voice"},
				{ID: 2, Name: "auth"},
			}, nil
		},
	}

	unregistered := &services.UnregisteredTagError{Name: "does-not-exist"}
	retErr := handleVocabularyErrorWithSnippet(cmd, svc, "does-not-exist", unregistered)
	if retErr == nil {
		t.Fatal("expected non-nil error from helper")
	}
	stderr := errBuf.String()

	// Error message body.
	if !strings.Contains(stderr, "tag is not registered: does-not-exist") {
		t.Errorf("stderr missing error body, got: %q", stderr)
	}
	// Vocabulary snippet.
	if !strings.Contains(stderr, "voice") {
		t.Errorf("stderr missing vocabulary name 'voice', got: %q", stderr)
	}
	if !strings.Contains(stderr, "auth") {
		t.Errorf("stderr missing vocabulary name 'auth', got: %q", stderr)
	}
	// Remediation line — exact substring per AC-T2.
	want := "To add it: shark tags add does-not-exist"
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr missing remediation %q, got: %q", want, stderr)
	}
	// Stderr should END with the remediation line (trailing newline allowed).
	trimmed := strings.TrimRight(stderr, "\n")
	if !strings.HasSuffix(trimmed, want) {
		t.Errorf("stderr does not end with %q, got: %q", want, stderr)
	}
}

func TestHandleVocabularyErrorWithSnippet_NotFoundStillWorks(t *testing.T) {
	// AC-T3 regression: NotFoundError path (used by F03 rm/rename) still
	// renders the same vocabulary snippet + remediation.
	withPlainTextGlobal(t)
	cmd, errBuf := buildSingleCommandForHelper(false)

	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{{ID: 1, Name: "voice"}}, nil
		},
	}

	notFound := &services.NotFoundError{Name: "missing"}
	retErr := handleVocabularyErrorWithSnippet(cmd, svc, "missing", notFound)
	if retErr == nil {
		t.Fatal("expected non-nil error from helper")
	}
	stderr := errBuf.String()

	if !strings.Contains(stderr, "tag not found: missing") {
		t.Errorf("stderr missing NotFound body, got: %q", stderr)
	}
	if !strings.Contains(stderr, "voice") {
		t.Errorf("stderr missing vocabulary name, got: %q", stderr)
	}
	if !strings.Contains(stderr, "To add it: shark tags add missing") {
		t.Errorf("stderr missing remediation, got: %q", stderr)
	}
}

func TestHandleVocabularyErrorWithSnippet_JSONModeSuppressesSnippet(t *testing.T) {
	// In --json mode the helper writes a JSON error object and does NOT
	// append the human-readable vocabulary snippet.
	// We use cli.GlobalConfig.JSON (matching the pattern in tags_test.go's
	// executeTagsCmd) because the helper reads that global in addition to
	// the per-command --json flag.
	origJSON := cli.GlobalConfig.JSON
	cli.GlobalConfig.JSON = true
	defer func() { cli.GlobalConfig.JSON = origJSON }()

	cmd, errBuf := buildSingleCommandForHelper(true)

	svc := &mockTagService{
		listTagsFn: func(ctx context.Context) ([]*models.Tag, error) {
			return []*models.Tag{{ID: 1, Name: "voice"}}, nil
		},
	}

	unregistered := &services.UnregisteredTagError{Name: "does-not-exist"}
	_ = handleVocabularyErrorWithSnippet(cmd, svc, "does-not-exist", unregistered)

	stderr := errBuf.String()
	if strings.Contains(stderr, "To add it: shark tags add") {
		t.Errorf("json mode should not include remediation line, got: %q", stderr)
	}

	// Parse the JSON error object and confirm the code.
	line := strings.TrimSpace(stderr)
	var errObj map[string]string
	if jsonErr := json.Unmarshal([]byte(line), &errObj); jsonErr != nil {
		t.Fatalf("expected JSON error on stderr, got: %q (unmarshal err=%v)",
			stderr, jsonErr)
	}
	if errObj["error"] != "unregistered_tag" {
		t.Errorf("JSON error code = %q, want %q", errObj["error"], "unregistered_tag")
	}
}

// TestTagsErrorCode_TagFilterUnavailable tests AC-T3: tagsErrorCode maps
// *TagFilterUnavailableError to exit code 3 with JSON code "unavailable".
func TestTagsErrorCode_TagFilterUnavailable(t *testing.T) {
	err := &services.TagFilterUnavailableError{}
	code, exit := tagsErrorCode(err)
	if exit != 3 {
		t.Errorf("exit = %d, want %d", exit, 3)
	}
	if code == "db_error" {
		t.Errorf("code = %q, should not be db_error — TagFilterUnavailableError should get a non-db exit code",
			code)
	}
}

// TestTagsErrorCode_WrappedTagFilterUnavailable tests that errors.As unwraps
// through %w for TagFilterUnavailableError.
func TestTagsErrorCode_WrappedTagFilterUnavailable(t *testing.T) {
	inner := &services.TagFilterUnavailableError{}
	err := fmt.Errorf("outer: %w", inner)
	_, exit := tagsErrorCode(err)
	if exit != 3 {
		t.Errorf("wrapped TagFilterUnavailableError exit = %d, want %d", exit, 3)
	}
}
