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

// handleVocabularyErrorWithSnippet renders typed tag errors to the user.
// For NotFoundError / UnregisteredTagError / TagRequiredError it writes the
// error line, the vocabulary snippet (first 10 registered names, "…and N
// more" when truncated), and for the first two a remediation line
// "To add it: shark tags add <name>". Returns an error carrying the exit
// code for the cobra RunE path.
func handleVocabularyErrorWithSnippet(
	cmd *cobra.Command,
	s tagServiceIface,
	name string,
	err error,
) error {
	jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON

	// NotFoundError and UnregisteredTagError emit snippet + remediation.
	// TagRequiredError emits snippet only.
	var notFound *services.NotFoundError
	var unregistered *services.UnregisteredTagError
	var required *services.TagRequiredError
	wantSnippet := errors.As(err, &notFound) || errors.As(err, &unregistered) || errors.As(err, &required)
	wantRemediation := errors.As(err, &notFound) || errors.As(err, &unregistered)

	code, exitCode := tagsErrorCode(err)
	writeTagsError(cmd, code, err.Error())

	if wantSnippet && !jsonMode {
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
		// Remediation line is the last stderr write so callers can assert
		// the exact suffix. TagRequiredError omits it.
		if wantRemediation {
			fmt.Fprintf(cmd.ErrOrStderr(), "To add it: shark tags add %s\n", name)
		}
		return fmt.Errorf("exit code %d: %w", exitCode, err)
	}

	// For unauthorized errors surface the maintainer hint alongside the error line.
	var unauth *maintainer.UnauthorizedError
	if errors.As(err, &unauth) {
		if hint := unauth.UserHint(); hint != "" && !jsonMode {
			fmt.Fprintln(cmd.ErrOrStderr(), hint)
		}
	}
	return fmt.Errorf("exit code %d: %w", exitCode, err)
}

// handleEntityServiceError routes entity service errors through the correct
// rendering path. Typed tag errors get the vocabulary snippet and an
// embedded "exit code N:" prefix; all other errors propagate unchanged.
// This is the single wiring point for all six entity create/update runners.
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
		// Empty name — handleVocabularyErrorWithSnippet skips remediation for TagRequiredError.
		return handleVocabularyErrorWithSnippet(cmd, tagSvc, "", err)
	}

	return err
}

// handleTagsRmRenameError is retained as a thin alias so existing call
// sites in tags.go (rm and rename) compile unchanged. New code should
// call handleVocabularyErrorWithSnippet directly.
func handleTagsRmRenameError(
	cmd *cobra.Command,
	s tagServiceIface,
	name string,
	err error,
) error {
	return handleVocabularyErrorWithSnippet(cmd, s, name, err)
}

// appendTagsToBasicInfo appends a "Tags" row to a BasicInfo slice for entity
// get display (REQ-F-015, AC-28 series). Renders "Tags: voice, auth" when tags
// are present, or "Tags: (none)" when the slice is empty. When tags is nil
// (tagSvc unavailable / graceful degradation per REQ-F-014), no row is added.
func appendTagsToBasicInfo(info [][]string, tags []string) [][]string {
	if tags == nil {
		// tagSvc was nil — omit the row entirely (graceful degradation).
		return info
	}
	var tagStr string
	if len(tags) == 0 {
		tagStr = "(none)"
	} else {
		tagStr = strings.Join(tags, ", ")
	}
	return append(info, []string{"Tags", tagStr})
}
