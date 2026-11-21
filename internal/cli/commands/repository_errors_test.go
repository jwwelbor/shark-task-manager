package commands

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRepositoryDatabaseFailure() error {
	return fmt.Errorf("database operation failed: %w: %w", repoerr.ErrDatabaseFailure, errors.New("invalid stored value"))
}

func assertDatabaseExitCode2(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err, "expected a database lookup error")
	assert.Contains(t, err.Error(), "exit code 2:", "error = %q, want an exit code 2 marker", err)
	assert.ErrorIs(t, err, repoerr.ErrDatabaseFailure, "repository classification should remain wrapped")
}

func TestMapRepositoryErrorClassifiesDatabaseFailure(t *testing.T) {
	cause := fmt.Errorf("failed to update tech-debt TD-001-slug: %w", newRepositoryDatabaseFailure())
	mapped := mapRepositoryError("tech-debt", "TD-001-slug", cause)
	assertDatabaseExitCode2(t, mapped)
	assert.Contains(t, mapped.Error(), "failed to update tech-debt")
	assert.NotContains(t, mapped.Error(), "look up")
}

func TestRunChangeGetSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runChangeGet(newChangeCmd(), []string{"CC-001-slug"}))
}

func TestRunChangeUpdateSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		UpdateChangeCardFunc: func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	cmd := newChangeCmd()
	cmd.Flags().String("title", "", "Title")
	require.NoError(t, cmd.Flags().Set("title", "Updated title"))
	assertDatabaseExitCode2(t, runChangeUpdate(cmd, []string{"CC-001-slug"}))
}

func TestRunChangeDeleteSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	changeForce = false
	t.Cleanup(func() { changeForce = false })
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		GetChangeCardFunc: func(ctx context.Context, key string) (*models.ChangeCard, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runChangeDelete(newChangeCmd(), []string{"CC-001-slug"}))
}

func TestRunTdGetSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runTdGet(newTdCmd(), []string{"TD-001-slug"}))
}

func TestRunTdUpdateSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	restore := injectMockTdSvc(t, &MockTechDebtService{
		UpdateTechDebtFunc: func(ctx context.Context, key string, updates services.TechDebtUpdates) (*models.TechDebt, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	cmd := newTestTdUpdateCmd()
	require.NoError(t, cmd.Flags().Set("title", "Updated title"))
	assertDatabaseExitCode2(t, runTdUpdate(cmd, []string{"TD-001-slug"}))
}

func TestRunTdDeleteSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	cmd := newTdCmd()
	cmd.Flags().Bool("force", false, "")
	restore := injectMockTdSvc(t, &MockTechDebtService{
		GetTechDebtFunc: func(ctx context.Context, key string) (*models.TechDebt, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runTdDelete(cmd, []string{"TD-001-slug"}))
}

func TestRunChangeDeleteWriteFailureUsesDatabaseExitCode(t *testing.T) {
	changeForce = true
	t.Cleanup(func() { changeForce = false })
	restore := injectMockChangeCardSvc(t, &MockChangeCardService{
		DeleteChangeCardFunc: func(ctx context.Context, key string) error {
			return newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runChangeDelete(newChangeCmd(), []string{"CC-001-slug"}))
}

func TestRunTdDeleteWriteFailureUsesDatabaseExitCode(t *testing.T) {
	cmd := newTdCmd()
	cmd.Flags().Bool("force", true, "")
	restore := injectMockTdSvc(t, &MockTechDebtService{
		DeleteTechDebtFunc: func(ctx context.Context, key string) error {
			return newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runTdDelete(cmd, []string{"TD-001-slug"}))
}

func TestRunTdTriageSlugLookupFailureUsesDatabaseExitCode(t *testing.T) {
	cmd := newTdCmd()
	cmd.Flags().String("severity", "", "")
	cmd.Flags().String("category", "", "")
	cmd.Flags().String("effort-estimate", "", "")
	restore := injectMockTdSvc(t, &MockTechDebtService{
		TriageTechDebtFunc: func(ctx context.Context, key string, input services.TriageTechDebtInput) (*models.TechDebt, error) {
			return nil, newRepositoryDatabaseFailure()
		},
	})
	defer restore()

	assertDatabaseExitCode2(t, runTdTriage(cmd, []string{"TD-001-slug"}))
}
