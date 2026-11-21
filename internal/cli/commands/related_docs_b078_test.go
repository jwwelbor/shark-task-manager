package commands

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/repository"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRelatedDocsDeleteSelectionRequiresExactlyOneSelector(t *testing.T) {
	tests := []struct {
		name     string
		flags    []struct{ name, value string }
		wantType string
		wantKey  string
		wantErr  bool
	}{
		{name: "missing", wantErr: true},
		{name: "empty key", flags: []struct{ name, value string }{{"task", ""}}, wantErr: true},
		{name: "conflicting", flags: []struct{ name, value string }{{"epic", "E01"}, {"task", "T-E01-F01-001"}}, wantErr: true},
		{name: "valid plus explicitly empty first", flags: []struct{ name, value string }{{"epic", ""}, {"task", "T-E01-F01-001"}}, wantErr: true},
		{name: "valid plus explicitly empty second", flags: []struct{ name, value string }{{"task", "T-E01-F01-001"}, {"epic", ""}}, wantErr: true},
		{name: "epic", flags: []struct{ name, value string }{{"epic", "E01"}}, wantType: "epic", wantKey: "E01"},
		{name: "feature", flags: []struct{ name, value string }{{"feature", "E01-F01"}}, wantType: "feature", wantKey: "E01-F01"},
		{name: "task", flags: []struct{ name, value string }{{"task", "T-E01-F01-001"}}, wantType: "task", wantKey: "T-E01-F01-001"},
		{name: "bug", flags: []struct{ name, value string }{{"bug", "B078"}}, wantType: "bug", wantKey: "B078"},
		{name: "change", flags: []struct{ name, value string }{{"change", "CC-001"}}, wantType: "change", wantKey: "CC-001"},
		{name: "question", flags: []struct{ name, value string }{{"question", "Q001"}}, wantType: "question", wantKey: "Q001"},
		{name: "tech debt", flags: []struct{ name, value string }{{"tech-debt", "TD-001"}}, wantType: "tech_debt", wantKey: "TD-001"},
		{name: "invalid tech debt key", flags: []struct{ name, value string }{{"tech-debt", "TD-1"}}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := newRelatedDocsDeleteSelectionCommand(t, test.flags)
			gotType, gotKey, err := resolveRelatedDocsSelection(cmd, nil)
			if test.wantErr {
				wantMessage := "exactly one"
				if test.name == "invalid tech debt key" {
					wantMessage = "invalid tech-debt key"
				}
				require.ErrorContains(t, err, wantMessage)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantType, string(gotType))
			require.Equal(t, test.wantKey, gotKey)
		})
	}
}

func TestResolveRelatedDocsDeleteSelectionPreservesFlagReadError(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("task", false, "")
	require.NoError(t, cmd.Flags().Set("task", "true"))
	_, _, err := resolveRelatedDocsSelection(cmd, nil)
	require.ErrorContains(t, err, "read --task selector")
}

func TestRelatedDocsDeleteRejectsInvalidTechDebtKeysBeforeDatabaseAccess(t *testing.T) {
	originalGetter := relatedDocsGetDB
	databaseAccessed := false
	relatedDocsGetDB = func(context.Context) (*repository.DB, error) {
		databaseAccessed = true
		return nil, errors.New("database getter must not run")
	}
	t.Cleanup(func() { relatedDocsGetDB = originalGetter })

	for _, testCase := range []struct {
		name        string
		key         string
		wantErrText string
	}{
		{name: "empty", key: "", wantErrText: "exactly one"},
		{name: "whitespace", key: "   ", wantErrText: "invalid tech-debt key"},
		{name: "malformed numeric key", key: "TD-1", wantErrText: "invalid tech-debt key"},
		{name: "empty slug", key: "TD-001-", wantErrText: "invalid tech-debt key"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			cmd := &cobra.Command{RunE: runRelatedDocsDelete}
			cmd.Flags().String("tech-debt", "", "")
			require.NoError(t, cmd.Flags().Set("tech-debt", testCase.key))
			err := cmd.RunE(cmd, []string{"specification"})
			require.ErrorContains(t, err, testCase.wantErrText)
			require.False(t, databaseAccessed, "database access happened before selector validation")
		})
	}
}

func newRelatedDocsDeleteSelectionCommand(t *testing.T, values []struct{ name, value string }) *cobra.Command {
	t.Helper()
	return newRelatedDocsHandlerCommand(t, nil, values)
}

func TestRelatedDocsDeleteReturnsDatabaseAcquisitionCauseBeforeDispatch(t *testing.T) {
	sentinel := errors.New("fixture database initialization error")
	originalGetter := relatedDocsGetDB
	relatedDocsGetDB = func(context.Context) (*repository.DB, error) { return nil, sentinel }
	t.Cleanup(func() { relatedDocsGetDB = originalGetter })

	cmd := &cobra.Command{RunE: runRelatedDocsDelete}
	cmd.Flags().String("task", "", "")
	require.NoError(t, cmd.Flags().Set("task", "T-E10-F02-013"))
	err := cmd.RunE(cmd, []string{"S010 current continuation"})
	require.ErrorIs(t, err, sentinel)
	require.ErrorContains(t, err, "failed to unlink document from task T-E10-F02-013")
}

func TestRelatedDocsDeleteQuestionRunEPreservesNotFoundCause(t *testing.T) {
	originalDBGetter := relatedDocsGetDB
	originalQuestionFactory := relatedDocsQuestionUnlinkService
	relatedDocsGetDB = func(context.Context) (*repository.DB, error) { return nil, nil }
	relatedDocsQuestionUnlinkService = func(context.Context) questionUnlinkService {
		return questionUnlinkErrorStub{err: repoerr.ErrNotFound}
	}
	t.Cleanup(func() {
		relatedDocsGetDB = originalDBGetter
		relatedDocsQuestionUnlinkService = originalQuestionFactory
	})

	cmd := &cobra.Command{RunE: runRelatedDocsDelete}
	cmd.Flags().String("question", "", "")
	require.NoError(t, cmd.Flags().Set("question", "Q404"))
	err := cmd.RunE(cmd, []string{"Question spec"})
	require.ErrorIs(t, err, repoerr.ErrNotFound)
	require.True(t, strings.HasPrefix(err.Error(), "exit code 1:"))
	require.ErrorContains(t, err, "failed to unlink document from question Q404")
}

func TestRelatedDocsDeleteQuestionRunEPreservesWriteFailureCause(t *testing.T) {
	sentinel := errors.New("fixture association write failure")
	originalDBGetter := relatedDocsGetDB
	originalQuestionFactory := relatedDocsQuestionUnlinkService
	relatedDocsGetDB = func(context.Context) (*repository.DB, error) { return nil, nil }
	relatedDocsQuestionUnlinkService = func(context.Context) questionUnlinkService {
		return questionUnlinkErrorStub{err: sentinel}
	}
	t.Cleanup(func() {
		relatedDocsGetDB = originalDBGetter
		relatedDocsQuestionUnlinkService = originalQuestionFactory
	})

	cmd := &cobra.Command{RunE: runRelatedDocsDelete}
	cmd.Flags().String("question", "", "")
	require.NoError(t, cmd.Flags().Set("question", "Q405"))
	err := cmd.RunE(cmd, []string{"Question write failure"})
	require.ErrorIs(t, err, sentinel)
	require.True(t, strings.HasPrefix(err.Error(), "exit code 2:"))
	require.Equal(t, 1, strings.Count(err.Error(), "failed to unlink document from question Q405"))
}

type questionUnlinkErrorStub struct{ err error }

func (stub questionUnlinkErrorStub) UnlinkDocumentByKey(context.Context, string, string) error {
	return stub.err
}

func TestRelatedDocsAddHandlerRejectsInvalidSelectorCardinalityBeforeDispatch(t *testing.T) {
	for _, selectors := range invalidRelatedDocsSelectorCases() {
		t.Run(selectors.name, func(t *testing.T) {
			cmd := newRelatedDocsHandlerCommand(t, runRelatedDocsAdd, selectors.values)
			output, err := captureRelatedDocsHandlerOutput(t, func() error {
				return cmd.RunE(cmd, []string{"title", "docs/title.md"})
			})
			require.EqualError(t, err, selectors.wantError)
			require.Empty(t, output)
		})
	}
}

func TestRelatedDocsListHandlerRejectsInvalidSelectorCardinalityBeforeDispatch(t *testing.T) {
	for _, selectors := range invalidRelatedDocsSelectorCases() {
		t.Run(selectors.name, func(t *testing.T) {
			cmd := newRelatedDocsHandlerCommand(t, runRelatedDocsListList, selectors.values)
			output, err := captureRelatedDocsHandlerOutput(t, func() error {
				return cmd.RunE(cmd, nil)
			})
			require.EqualError(t, err, selectors.wantError)
			require.Empty(t, output)
		})
	}
}

const exactlyOneRelatedDocsSelectorError = "exactly one of --epic, --feature, --task, --bug, --change, --question, or --tech-debt must specify an entity key"

func invalidRelatedDocsSelectorCases() []struct {
	name      string
	values    []struct{ name, value string }
	wantError string
} {
	return []struct {
		name      string
		values    []struct{ name, value string }
		wantError string
	}{
		{name: "missing", values: nil, wantError: exactlyOneRelatedDocsSelectorError},
		{name: "empty key", values: []struct{ name, value string }{{"task", ""}}, wantError: exactlyOneRelatedDocsSelectorError},
		{name: "conflicting", values: []struct{ name, value string }{{"task", "T-E10-F02-013"}, {"epic", "E10"}}, wantError: "exactly one of the entity-type flags must be specified"},
		{name: "valid plus explicitly empty first", values: []struct{ name, value string }{{"epic", ""}, {"task", "T-E10-F02-013"}}, wantError: "exactly one of the entity-type flags must be specified"},
		{name: "valid plus explicitly empty second", values: []struct{ name, value string }{{"task", "T-E10-F02-013"}, {"epic", ""}}, wantError: "exactly one of the entity-type flags must be specified"},
	}
}

func newRelatedDocsHandlerCommand(t *testing.T, handler func(*cobra.Command, []string) error, values []struct{ name, value string }) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{RunE: handler}
	for _, selector := range relatedDocsSelectors {
		cmd.Flags().String(selector.flag, "", selector.usage)
	}
	for _, value := range values {
		require.NoError(t, cmd.Flags().Set(value.name, value.value))
	}
	return cmd
}

func captureRelatedDocsHandlerOutput(t *testing.T, run func() error) (string, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	writerClosed, readerClosed := false, false
	defer func() {
		os.Stdout = originalStdout
		if !readerClosed {
			assert.NoError(t, reader.Close(), "close captured stdout reader")
		}
		if !writerClosed {
			assert.NoError(t, writer.Close(), "close captured stdout writer")
		}
	}()
	runErr := run()
	closeErr := writer.Close()
	writerClosed = true
	os.Stdout = originalStdout
	output, readErr := io.ReadAll(reader)
	readerErr := reader.Close()
	readerClosed = true
	return string(output), errors.Join(runErr, closeErr, readErr, readerErr)
}
