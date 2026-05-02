package commands

// console_width_list_test.go — CC-036 follow-up
//
// Verifies that list-view row builders for tasks, ideas, and change-cards,
// plus the recent-items table renderer, truncate the long-text (Title)
// column according to cli.TitleColumnWidth(...). The width plumbing itself
// (config → cli helper) is exercised by internal/cli/console_width_test.go;
// these tests prove each renderer actually consults it.

import (
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

const longTitle = "this is an intentionally very long title that should be truncated by the console_width setting when the terminal is narrow enough"

// findCellContaining returns the first cell that includes substr, or "".
func findCellContaining(row []string, substr string) string {
	for _, c := range row {
		if strings.Contains(c, substr) {
			return c
		}
	}
	return ""
}

// assertTruncated checks that `cell` is shorter than the original long title
// and ends with "..." — the truncateRunes contract.
func assertTruncated(t *testing.T, label, cell string) {
	t.Helper()
	if cell == "" {
		t.Fatalf("%s: no row cell contained the long title", label)
	}
	if cell == longTitle {
		t.Errorf("%s: title was not truncated (length %d)", label, len(cell))
	}
	if !strings.HasSuffix(cell, "...") {
		t.Errorf("%s: truncated cell should end with %q, got %q", label, "...", cell)
	}
}

// assertNotTruncated checks that `cell` contains the full untouched
// title (the cell may be right-padded with spaces by fitColumn so the
// table fills the terminal — that padding is expected).
func assertNotTruncated(t *testing.T, label, cell string) {
	t.Helper()
	trimmed := strings.TrimRight(cell, " ")
	if trimmed != longTitle {
		t.Errorf("%s: title should NOT be truncated at wide width, got %q (trimmed %q)", label, cell, trimmed)
	}
}

// ---------------------------------------------------------------------------
// buildTaskListRows
// ---------------------------------------------------------------------------

func TestBuildTaskListRows_TitleScalesWithConsoleWidth(t *testing.T) {
	tasks := []*models.Task{
		{
			BaseEntity: models.BaseEntity{Key: "E07-F01-001", Title: longTitle},
			Status:     models.TaskStatus("todo"),
			Priority:   5,
		},
	}

	t.Run("narrow terminal truncates", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(80)
		defer restore()

		rows := buildTaskListRows(tasks)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		assertTruncated(t, "buildTaskListRows", findCellContaining(rows[0], "this is an"))
	})

	t.Run("wide terminal preserves full title", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(400)
		defer restore()

		rows := buildTaskListRows(tasks)
		assertNotTruncated(t, "buildTaskListRows", findCellContaining(rows[0], "this is an"))
	})
}

// ---------------------------------------------------------------------------
// buildIdeaListRows
// ---------------------------------------------------------------------------

func TestBuildIdeaListRows_TitleScalesWithConsoleWidth(t *testing.T) {
	ideas := []*models.Idea{
		{
			Key:         "I001",
			Title:       longTitle,
			Status:      models.IdeaStatus("captured"),
			CreatedDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	t.Run("narrow terminal truncates", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(80)
		defer restore()

		rows := buildIdeaListRows(ideas)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		assertTruncated(t, "buildIdeaListRows", findCellContaining(rows[0], "this is an"))
	})

	t.Run("wide terminal preserves full title", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(400)
		defer restore()

		rows := buildIdeaListRows(ideas)
		assertNotTruncated(t, "buildIdeaListRows", findCellContaining(rows[0], "this is an"))
	})
}

// ---------------------------------------------------------------------------
// buildChangeCardListRows
// ---------------------------------------------------------------------------

func TestBuildChangeCardListRows_TitleScalesWithConsoleWidth(t *testing.T) {
	cards := []*models.ChangeCard{
		{
			BaseEntity: models.BaseEntity{
				Key:       "CC-001",
				Title:     longTitle,
				CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Status: models.ChangeCardStatus("proposed"),
		},
	}

	t.Run("narrow terminal truncates", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(80)
		defer restore()

		rows := buildChangeCardListRows(cards)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		assertTruncated(t, "buildChangeCardListRows", findCellContaining(rows[0], "this is an"))
	})

	t.Run("wide terminal preserves full title", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(400)
		defer restore()

		rows := buildChangeCardListRows(cards)
		assertNotTruncated(t, "buildChangeCardListRows", findCellContaining(rows[0], "this is an"))
	})
}

// ---------------------------------------------------------------------------
// buildRecentRows
// ---------------------------------------------------------------------------

func TestBuildRecentRows_TitleScalesWithConsoleWidth(t *testing.T) {
	items := []services.RecentItem{
		{
			Type:      "task",
			Key:       "E07-F01-001",
			Title:     longTitle,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Status:    "todo",
		},
	}

	t.Run("narrow terminal truncates", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(80)
		defer restore()

		rows := buildRecentRows(items)
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
		assertTruncated(t, "buildRecentRows", findCellContaining(rows[0], "this is an"))
	})

	t.Run("wide terminal preserves full title", func(t *testing.T) {
		restore := cli.SetConsoleWidthForTesting(400)
		defer restore()

		rows := buildRecentRows(items)
		assertNotTruncated(t, "buildRecentRows", findCellContaining(rows[0], "this is an"))
	})
}
