package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// handleVocabularyErrorWithSnippet is the shared CLI error-rendering helper
// for typed errors that reference the tag vocabulary. It is used by:
//
//   - shark tags rm / shark tags rename (F03) when the vocabulary does not
//     contain the requested name (*services.NotFoundError).
//   - shark <entity> create|update --tag=... (F04) and
//     shark <entity> tag add|rm <key> <name> (F04) when attach fails because
//     the supplied name is not registered (*services.UnregisteredTagError).
//   - shark <entity> create (F04) when the entity type is in tag_required_for
//     but no --tag was supplied (*services.TagRequiredError).
//
// The helper:
//  1. Maps the error to a (jsonCode, exitCode) via tagsErrorCode.
//  2. Writes the error message to stderr (JSON when --json is set, plain
//     text otherwise).
//  3. In plain-text mode ONLY, for UnregisteredTagError / NotFoundError:
//     appends the SC-2 vocabulary snippet (first 10 registered tag names,
//     comma-separated; "…and N more" when truncated) followed by the exact
//     remediation line "To add it: shark tags add <name>".
//  4. In plain-text mode ONLY, for TagRequiredError: appends the SC-2
//     vocabulary snippet but NO remediation line (no single name was rejected).
//  5. Returns an error carrying the exit code so the cobra RunE path can
//     produce the correct process exit status.
//
// Spec references:
//   - REQ-F-015 — shared error-rendering helper for attach-path errors.
//   - REQ-F-016 — exit-code mapping.
//   - spec §2.7 — CLI error handling reuse.
//   - Task T-E28-F04-004 AC-T2 — remediation-line text is exact.
//   - Task T-E28-F04-013 FR-1, FR-2 — TagRequiredError snippet (no remediation).
func handleVocabularyErrorWithSnippet(
	cmd *cobra.Command,
	s tagServiceIface,
	name string,
	err error,
) error {
	jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON

	// Determine whether this error should trigger the vocabulary snippet.
	// Both the F03 vocabulary-management NotFoundError and the F04
	// attach-path UnregisteredTagError produce the snippet with remediation.
	// TagRequiredError produces the snippet WITHOUT a remediation line.
	var notFound *services.NotFoundError
	var unregistered *services.UnregisteredTagError
	var required *services.TagRequiredError
	wantSnippet := errors.As(err, &notFound) || errors.As(err, &unregistered) || errors.As(err, &required)
	wantRemediation := errors.As(err, &notFound) || errors.As(err, &unregistered)

	code, exitCode := tagsErrorCode(err)
	writeTagsError(cmd, code, err.Error())

	if wantSnippet && !jsonMode {
		// Assemble vocabulary snippet per REQ-F-015 / SC-2.
		if vocab, listErr := s.ListTags(cmd.Context()); listErr == nil && len(vocab) > 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "Available tags:")
			const maxSnippet = 10
			shown := vocab
			remainder := 0
			if len(vocab) > maxSnippet {
				shown = vocab[:maxSnippet]
				remainder = len(vocab) - maxSnippet
			}
			names := make([]string, 0, len(shown))
			for _, t := range shown {
				names = append(names, t.Name)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", strings.Join(names, ", "))
			if remainder > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "  …and %d more\n", remainder)
			}
		}
		// The remediation line is the LAST thing written to stderr so
		// AC-T2 can assert the exact-suffix property. TagRequiredError
		// does NOT emit a remediation line (FR-2: no single name was rejected).
		if wantRemediation {
			fmt.Fprintf(cmd.ErrOrStderr(), "To add it: shark tags add %s\n", name)
		}
		return fmt.Errorf("exit code %d: %w", exitCode, err)
	}

	// For non-snippet errors (e.g., *maintainer.UnauthorizedError) surface
	// the maintainer hint alongside the error line, matching the prior
	// handleTagsRmRenameError behaviour.
	var unauth *maintainer.UnauthorizedError
	if errors.As(err, &unauth) {
		if hint := unauth.UserHint(); hint != "" && !jsonMode {
			fmt.Fprintln(cmd.ErrOrStderr(), hint)
		}
	}
	return fmt.Errorf("exit code %d: %w", exitCode, err)
}

// handleEntityServiceError routes entity service errors through the correct
// error-rendering path. For typed tag errors (*UnregisteredTagError,
// *TagRequiredError) it delegates to handleVocabularyErrorWithSnippet so
// the SC-2 vocabulary snippet is rendered and the "exit code N:" prefix is
// embedded in the returned error for main.go to unwrap.
//
// For all other (non-tag) errors the raw error is returned unchanged so
// cobra's RunE chain propagates it to main.go, which emits the generic
// "Error: <msg>" line and exits 1. This preserves the pre-T-013 behavior
// for all call sites that previously used `return err`.
//
// This is the single wiring point for all six entity create/update runners
// (bug, task, feature, epic, change, idea). Replace `return err` at every
// service call site with this helper so that tag-typed errors receive the
// SC-2 rendering.
//
// Parameters:
//   - cmd: the cobra command (for stderr, --json flag, context)
//   - tagSvc: tag service for vocabulary lookup (passed to the snippet helper)
//   - err: error returned by the entity service method
//   - entityType: human-readable entity type (for future error messages)
//   - key: entity key (for future error messages)
//
// Returns the error to propagate as the cobra RunE return value. For tag
// errors the error already encodes the correct exit code via the
// "exit code N:" prefix understood by cmd/shark/main.go.
//
// Spec references: T-E28-F04-013 FR-1, FR-2, FR-3.
func handleEntityServiceError(
	cmd *cobra.Command,
	tagSvc tagServiceIface,
	err error,
	entityType string,
	key string,
) error {
	if err == nil {
		return nil
	}

	var unregistered *services.UnregisteredTagError
	if errors.As(err, &unregistered) {
		return handleVocabularyErrorWithSnippet(cmd, tagSvc, unregistered.Name, err)
	}

	var required *services.TagRequiredError
	if errors.As(err, &required) {
		// Pass empty name — handleVocabularyErrorWithSnippet skips the
		// remediation line for TagRequiredError (wantRemediation=false).
		return handleVocabularyErrorWithSnippet(cmd, tagSvc, "", err)
	}

	// Non-tag error: return as-is. The existing pre-T-013 behavior at
	// most call sites was `return err`, which lets cobra's error handling
	// produce `Error: <msg>` and exit 1. Preserving that behavior avoids
	// regressions in tests that check for specific error return values.
	return err
}

// handleTagsRmRenameError is the pre-F04 name of
// handleVocabularyErrorWithSnippet. It is kept as a thin alias so existing
// call sites in tags.go (F03's rm and rename commands) continue to compile
// without churn. New code (F04 entity-tag commands) should call
// handleVocabularyErrorWithSnippet directly.
func handleTagsRmRenameError(
	cmd *cobra.Command,
	s tagServiceIface,
	name string,
	err error,
) error {
	return handleVocabularyErrorWithSnippet(cmd, s, name, err)
}
