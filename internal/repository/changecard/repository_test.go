package changecard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	repoerr "github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChangeCardRepository_List_TerminalStatusFiltering verifies that
// IncludeTerminal=false filters out exactly the statuses listed in TerminalStatuses,
// and that a custom terminal set (different from the hardcoded default) is respected.
func TestChangeCardRepository_List_TerminalStatusFiltering(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	// Clean up test change-cards
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-980','CC-981','CC-982','CC-983')")

	// Create change-cards covering default terminal statuses and a custom one
	cards := []*models.ChangeCard{
		newTestChangeCard("CC-980", "Active change card", "draft"),
		newTestChangeCard("CC-981", "Completed change card", "completed"),
		newTestChangeCard("CC-982", "Declined change card", "declined"),
		newTestChangeCard("CC-983", "Archived change card (custom terminal)", "archived"),
	}
	for _, c := range cards {
		if err := repo.Create(ctx, c); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key IN ('CC-980','CC-981','CC-982','CC-983')")
	}()

	t.Run("default terminal statuses exclude completed and declined", func(t *testing.T) {
		// IncludeTerminal=false with no TerminalStatuses uses the hardcoded fallback
		results, err := repo.List(ctx, &ChangeCardRepoFilter{IncludeTerminal: false})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		for _, c := range results {
			if c.Key == "CC-981" || c.Key == "CC-982" {
				t.Errorf("List(IncludeTerminal=false) returned terminal change-card %s with status %s", c.Key, c.Status)
			}
		}
		// CC-980 (draft) and CC-983 (archived) should appear since 'archived' is not in the default list
		found980, found983 := false, false
		for _, c := range results {
			if c.Key == "CC-980" {
				found980 = true
			}
			if c.Key == "CC-983" {
				found983 = true
			}
		}
		if !found980 {
			t.Error("List(IncludeTerminal=false) should include CC-980 (draft)")
		}
		if !found983 {
			t.Error("List(IncludeTerminal=false) with default terminals should include CC-983 (archived is not a default terminal)")
		}
	})

	t.Run("custom terminal statuses exclude archived but not completed", func(t *testing.T) {
		// Supply a custom terminal set: only 'archived' is terminal
		results, err := repo.List(ctx, &ChangeCardRepoFilter{
			IncludeTerminal:  false,
			TerminalStatuses: []string{"archived"},
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		for _, c := range results {
			if c.Key == "CC-983" {
				t.Errorf("List(TerminalStatuses=['archived']) returned terminal change-card %s with status %s", c.Key, c.Status)
			}
		}
		// completed and declined should appear since they are not in the custom terminal set
		foundCompleted := false
		for _, c := range results {
			if c.Key == "CC-981" {
				foundCompleted = true
			}
		}
		if !foundCompleted {
			t.Error("List(TerminalStatuses=['archived']) should include CC-981 (completed) because it is not in the custom terminal set")
		}
	})

	t.Run("IncludeTerminal=true returns all change-cards regardless of TerminalStatuses", func(t *testing.T) {
		results, err := repo.List(ctx, &ChangeCardRepoFilter{
			IncludeTerminal:  true,
			TerminalStatuses: []string{"completed", "declined", "archived"},
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		keys := make(map[string]bool)
		for _, c := range results {
			keys[c.Key] = true
		}
		for _, k := range []string{"CC-980", "CC-981", "CC-982", "CC-983"} {
			if !keys[k] {
				t.Errorf("List(IncludeTerminal=true) should include change-card %s", k)
			}
		}
	})
}

func TestChangeCardRepository_GetByKey_NormalizesChangeAliases(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	const canonicalKey = "CC-984"
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", canonicalKey)

	card := newTestChangeCard(canonicalKey, "Alias lookup change card", "draft")
	if err := repo.Create(ctx, card); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", canonicalKey)
	}()

	for _, input := range []string{"CC-984", "CC984", "C984", "c984", "C-984"} {
		t.Run(input, func(t *testing.T) {
			got, err := repo.GetByKey(ctx, input)
			if err != nil {
				t.Fatalf("GetByKey(%q) error = %v", input, err)
			}
			if got.Key != canonicalKey {
				t.Errorf("GetByKey(%q).Key = %q, want %q", input, got.Key, canonicalKey)
			}
		})
	}
}

func TestChangeCardRepository_GetByKey_SluggedAliasesResolveCanonicalKey(t *testing.T) {
	ctx := context.Background()
	database := test.GetTestDB()
	db := dbconn.NewDB(database)
	repo := NewChangeCardRepository(db)

	const canonicalKey = "CC-985"
	const slug = "alias-lookup-change-card"
	_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", canonicalKey)

	card := newTestChangeCard(canonicalKey, "Slug alias lookup change card", "draft")
	card.Slug = &[]string{slug}[0]
	if err := repo.Create(ctx, card); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer func() {
		_, _ = database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", canonicalKey)
	}()

	for _, input := range []string{
		"CC-985-alias-lookup-change-card",
		"CC985-alias-lookup-change-card",
		"C985-alias-lookup-change-card",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := repo.GetByKey(ctx, input)
			if err != nil {
				t.Fatalf("GetByKey(%q) error = %v", input, err)
			}
			if got.Key != canonicalKey {
				t.Errorf("GetByKey(%q).Key = %q, want %q", input, got.Key, canonicalKey)
			}
		})
	}
}

func TestChangeCardRepositoryGetByKeySluggedLookupReturnsScanFailure(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewChangeCardRepository(dbconn.NewDB(database))
	const key = "CC-986"
	_, err := database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
	require.NoError(t, err, "clear fixture change-card")
	card := newTestChangeCard(key, "Unreadable slug alias card", "draft")
	slug := "fixture"
	card.Slug = &slug
	require.NoError(t, repo.Create(ctx, card))
	t.Cleanup(func() {
		_, cleanupErr := database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
		assert.NoError(t, cleanupErr, "remove fixture change-card")
	})
	_, err = database.ExecContext(ctx, "UPDATE change_cards SET priority = NULL WHERE key = ?", key)
	require.NoError(t, err, "make card unreadable")

	_, err = repo.GetByKey(ctx, "CC-986-fixture")
	require.Error(t, err)
	assert.False(t, errors.Is(err, repoerr.ErrNotFound), "scan failure must not become not found")
	assert.ErrorIs(t, err, repoerr.ErrDatabaseFailure, "scan failure must be classified for CLI exit mapping")
	assert.Contains(t, strings.ToLower(err.Error()), "null", "want original NULL scan cause")
}

func TestChangeCardRepositoryGetByKeyExactLookupClassifiesScanFailure(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewChangeCardRepository(dbconn.NewDB(database))
	const key = "CC-987"
	_, err := database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
	require.NoError(t, err, "clear fixture change-card")
	card := newTestChangeCard(key, "Unreadable exact-key card", "draft")
	require.NoError(t, repo.Create(ctx, card))
	t.Cleanup(func() {
		_, cleanupErr := database.ExecContext(ctx, "DELETE FROM change_cards WHERE key = ?", key)
		assert.NoError(t, cleanupErr, "remove fixture change-card")
	})
	_, err = database.ExecContext(ctx, "UPDATE change_cards SET priority = NULL WHERE key = ?", key)
	require.NoError(t, err, "make card unreadable")

	_, err = repo.GetByKey(ctx, key)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoerr.ErrDatabaseFailure)
	assert.False(t, errors.Is(err, repoerr.ErrNotFound), "scan failure must not become not found")
}

func TestChangeCardRepositoryUpdateClassifiesDatabaseFailure(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewChangeCardRepository(dbconn.NewDB(database))
	card := newTestChangeCard("CC-988", "Update failure fixture", "draft")
	require.NoError(t, repo.Create(ctx, card))
	_, err := database.ExecContext(ctx, `CREATE TRIGGER fail_change_card_update BEFORE UPDATE ON change_cards BEGIN SELECT RAISE(ABORT, 'forced update failure'); END`)
	require.NoError(t, err)
	card.Title = "Updated fixture"

	err = repo.Update(ctx, card)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoerr.ErrDatabaseFailure)
	assert.Contains(t, err.Error(), "forced update failure")
}

func TestChangeCardRepositoryDeleteClassifiesDatabaseFailure(t *testing.T) {
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	repo := NewChangeCardRepository(dbconn.NewDB(database))
	card := newTestChangeCard("CC-989", "Delete failure fixture", "draft")
	require.NoError(t, repo.Create(ctx, card))
	_, err := database.ExecContext(ctx, `CREATE TRIGGER fail_change_card_delete BEFORE DELETE ON change_cards BEGIN SELECT RAISE(ABORT, 'forced delete failure'); END`)
	require.NoError(t, err)

	err = repo.Delete(ctx, card.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoerr.ErrDatabaseFailure)
	assert.Contains(t, err.Error(), "forced delete failure")
}
