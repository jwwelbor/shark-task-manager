package commands

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
)

// TestViewQuestionReadsPersistedMetadataWithoutContextData is a persisted CLI
// regression for REQ-F-004. It executes the production root registration, so
// a future change that sends Q001 back through file-scope resolution (or
// serializes the persistence model directly) fails this test.
func TestViewQuestionReadsPersistedMetadataWithoutContextData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "questions.db")
	sqlDB, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	questionRepo := repository.NewQuestionRepository(repository.NewDB(sqlDB))
	contextData := `{"current_step":"view-question-context-must-not-leak"}`
	question := &models.Question{
		BaseEntity: models.BaseEntity{
			Title:       "Persisted view question",
			ContextData: &contextData,
		},
		Status:    models.QuestionStatusDraft,
		Summary:   "Read through the generic view command",
		Requester: "cli-regression",
	}
	if err := questionRepo.Create(context.Background(), question); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("seed Question error = %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close seed database: %v", err)
	}

	cli.ResetServices()
	cli.ResetDB()
	t.Cleanup(func() {
		cli.ResetServices()
		cli.ResetDB()
		resetB036RootState(t)
	})

	cli.RootCmd.SetArgs([]string{"--db", dbPath, "--json", "view", "q001"})
	var executeErr error
	output := captureStdout(t, func() {
		executeErr = cli.RootCmd.Execute()
	})
	if executeErr != nil {
		t.Fatalf("shark --db %s --json view q001 error = %v", dbPath, executeErr)
	}

	if strings.Contains(output, "view-question-context-must-not-leak") || strings.Contains(output, "context_data") {
		t.Fatalf("shark view Q001 exposed persisted context data:\n%s", output)
	}
	var got models.QuestionProjection
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decode shark view Q001 output: %v\n%s", err, output)
	}
	if got.Key != "Q001" || got.Title != question.Title || got.Summary != question.Summary || got.Requester != question.Requester {
		t.Fatalf("shark view Q001 projection = %#v, want persisted Question metadata", got)
	}
	if got.Status != models.QuestionStatusDraft {
		t.Fatalf("shark view Q001 status = %q, want %q", got.Status, models.QuestionStatusDraft)
	}
}
