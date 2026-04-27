package changecard

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	"github.com/jwwelbor/shark-task-manager/internal/test"
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
