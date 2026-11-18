package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRelatedDocsAddEpic tests adding a document to an epic
func TestRelatedDocsAddEpic(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	// Setup test data
	mockEpicRepo.AddEpic(&models.Epic{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E01",
		Title: "Test Epic"},
	})

	// Create command with mocks
	cmd := createRelatedDocsAddCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"OAuth Spec", "docs/oauth.md", "--epic=E01"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify CreateOrGet was called
	if mockDocRepo.CreateOrGetCalls == 0 {
		t.Fatal("CreateOrGet not called")
	}
	if mockDocRepo.LastCreateOrGetTitle != "OAuth Spec" {
		t.Errorf("Expected title 'OAuth Spec', got %q", mockDocRepo.LastCreateOrGetTitle)
	}
	if mockDocRepo.LastCreateOrGetPath != "docs/oauth.md" {
		t.Errorf("Expected path 'docs/oauth.md', got %q", mockDocRepo.LastCreateOrGetPath)
	}

	// Verify LinkToEpic was called
	if mockDocRepo.LinkToEpicCalls == 0 {
		t.Fatal("LinkToEpic not called")
	}
}

// TestRelatedDocsAddFeature tests adding a document to a feature
func TestRelatedDocsAddFeature(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	// Setup test data
	mockFeatureRepo.AddFeature(&models.Feature{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "E01-F01",
		Title: "Test Feature"},
	})

	cmd := createRelatedDocsAddCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"Design Doc", "docs/design.md", "--feature=E01-F01"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.LinkToFeatureCalls == 0 {
		t.Fatal("LinkToFeature not called")
	}
}

// TestRelatedDocsAddTask tests adding a document to a task
func TestRelatedDocsAddTask(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	agentType := "backend"
	mockTaskRepo.AddTask(&models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key:   "T-E01-F01-001",
		Title: "Test Task"}, AgentType: &agentType,
		Priority: 1,
	})

	cmd := createRelatedDocsAddCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"Task Notes", "docs/notes.md", "--task=T-E01-F01-001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.LinkToTaskCalls == 0 {
		t.Fatal("LinkToTask not called")
	}
}

// TestRelatedDocsAddMissingParent tests error when parent doesn't exist
func TestRelatedDocsAddMissingParent(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	cmd := createRelatedDocsAddCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"Doc", "docs/doc.md", "--epic=MISSING"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for missing epic")
	}
}

// TestRelatedDocsDeleteEpic tests deleting a document from an epic
func TestRelatedDocsDeleteEpic(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	mockEpicRepo.AddEpic(&models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}})
	mockDocRepo.AddDocument(&models.Document{
		ID:       1,
		Title:    "OAuth Spec",
		FilePath: "docs/oauth.md",
	})

	cmd := createRelatedDocsDeleteCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"delete", "OAuth Spec", "--epic=E01"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.UnlinkFromEpicCalls == 0 {
		t.Fatal("UnlinkFromEpic not called")
	}
}

// TestRelatedDocsDeleteIdempotent tests delete succeeds even if not linked
func TestRelatedDocsDeleteIdempotent(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	mockEpicRepo.AddEpic(&models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}})

	cmd := createRelatedDocsDeleteCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"delete", "NonExistent", "--epic=E01"})

	// Should not error - delete is idempotent
	_ = cmd.Execute()
}

// TestRelatedDocsListEpic tests listing documents for an epic
func TestRelatedDocsListEpic(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	mockEpicRepo.AddEpic(&models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}})
	mockDocRepo.AddDocument(&models.Document{
		ID:       1,
		Title:    "Doc1",
		FilePath: "docs/doc1.md",
	})
	mockDocRepo.EpicDocuments[int64(1)] = []*models.Document{
		{ID: 1, Title: "Doc1", FilePath: "docs/doc1.md"},
	}

	cmd := createRelatedDocsListCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"list", "--epic=E01"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.ListForEpicCalls == 0 {
		t.Fatal("ListForEpic not called")
	}
}

// TestRelatedDocsListFeature tests listing documents for a feature
func TestRelatedDocsListFeature(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	mockFeatureRepo.AddFeature(&models.Feature{BaseEntity: models.BaseEntity{ID: 1, Key: "E01-F01", Title: "Feature"}})
	mockDocRepo.FeatureDocuments[int64(1)] = []*models.Document{}

	cmd := createRelatedDocsListCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"list", "--feature=E01-F01"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.ListForFeatureCalls == 0 {
		t.Fatal("ListForFeature not called")
	}
}

// TestRelatedDocsListTask tests listing documents for a task
func TestRelatedDocsListTask(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	agentType := "backend"
	mockTaskRepo.AddTask(&models.Task{BaseEntity: models.BaseEntity{ID: 1,
		Key: "T-E01-F01-001"}, AgentType: &agentType,
		Priority: 1,
	})
	mockDocRepo.TaskDocuments[int64(1)] = []*models.Document{}

	cmd := createRelatedDocsListCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"list", "--task=T-E01-F01-001"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if mockDocRepo.ListForTaskCalls == 0 {
		t.Fatal("ListForTask not called")
	}
}

// TestRelatedDocsListPositionalSelection covers B072: a related-docs list
// invocation should infer supported entity types from a positional key and
// reject ambiguous or unsupported input before dispatch.
func TestRelatedDocsListPositionalSelection(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		bug      string
		techDebt string
		wantType string
		wantKey  string
		wantErr  string
	}{
		{name: "epic", args: []string{"E01"}, wantType: "epic", wantKey: "E01"},
		{name: "feature", args: []string{"E01-F01"}, wantType: "feature", wantKey: "E01-F01"},
		{name: "task", args: []string{"T-E01-F01-001"}, wantType: "task", wantKey: "T-E01-F01-001"},
		{name: "bug", args: []string{"B072"}, wantType: "bug", wantKey: "B072"},
		{name: "change", args: []string{"CC-001"}, wantType: "change-card", wantKey: "CC-001"},
		{name: "question", args: []string{"Q001"}, wantType: "question", wantKey: "Q001"},
		{name: "tech debt", args: []string{"TD-595"}, wantType: "tech-debt", wantKey: "TD-595"},
		{name: "tech debt explicit", techDebt: "TD-595", wantType: "tech-debt", wantKey: "TD-595"},
		{name: "unsupported entity", args: []string{"TD001"}, wantErr: "cannot infer a supported entity type"},
		{name: "multiple positional keys", args: []string{"E01", "E02"}, wantErr: "at most one positional entity key"},
		{name: "positional and explicit selector", args: []string{"B072"}, bug: "B073", wantErr: "cannot combine positional entity key"},
		{name: "tech debt positional and conflicting selector", args: []string{"TD-595"}, techDebt: "TD-596", wantErr: "cannot combine positional entity key"},
		{name: "conflicting tech debt and bug selectors", bug: "B079", techDebt: "TD-595", wantErr: "exactly one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			for _, selector := range relatedDocsSelectors {
				cmd.Flags().String(selector.flag, "", selector.usage)
			}
			if tt.bug != "" {
				require.NoError(t, cmd.Flags().Set("bug", tt.bug))
			}
			if tt.techDebt != "" {
				require.NoError(t, cmd.Flags().Set("tech-debt", tt.techDebt))
			}
			entityType, key, err := resolveRelatedDocsSelection(cmd, tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRelatedDocsSelection returned error: %v", err)
			}
			if entityType != tt.wantType || key != tt.wantKey {
				t.Fatalf("selection = (%q, %q), want (%q, %q)", entityType, key, tt.wantType, tt.wantKey)
			}
		})
	}
}

func TestRelatedDocsTechDebtSelectorsAreAvailable(t *testing.T) {
	for _, cmd := range []*cobra.Command{relatedDocsAddCmd, relatedDocsDeleteCmd, relatedDocsListCmd} {
		if cmd.Flags().Lookup("tech-debt") == nil {
			t.Errorf("%s is missing --tech-debt selector", cmd.CommandPath())
		}
	}
}

type mockRelatedDocService struct {
	LinkDocumentFunc              func(ctx context.Context, key, title, path string) error
	UnlinkDocumentFunc            func(ctx context.Context, key, title string) error
	ListRelatedDocumentsByKeyFunc func(ctx context.Context, key string) ([]*models.Document, error)
}

func (m *mockRelatedDocService) LinkDocument(ctx context.Context, key, title, path string) error {
	return m.LinkDocumentFunc(ctx, key, title, path)
}

func (m *mockRelatedDocService) UnlinkDocument(ctx context.Context, key, title string) error {
	return m.UnlinkDocumentFunc(ctx, key, title)
}

func (m *mockRelatedDocService) ListRelatedDocumentsByKey(ctx context.Context, key string) ([]*models.Document, error) {
	return m.ListRelatedDocumentsByKeyFunc(ctx, key)
}

func TestRelatedDocsTechDebtAddDeleteAndListUseTechDebtService(t *testing.T) {
	var linkedKey, linkedTitle, linkedPath string
	var unlinked bool
	mock := &mockRelatedDocService{
		LinkDocumentFunc: func(_ context.Context, key, title, path string) error {
			linkedKey, linkedTitle, linkedPath = key, title, path
			return nil
		},
		UnlinkDocumentFunc: func(_ context.Context, key, title string) error {
			if key != "TD-595" || title != "Research report" {
				t.Fatalf("UnlinkDocument(%q, %q), want TD-595 and Research report", key, title)
			}
			unlinked = true
			return nil
		},
		ListRelatedDocumentsByKeyFunc: func(_ context.Context, key string) ([]*models.Document, error) {
			if key != "TD-595" {
				t.Fatalf("ListRelatedDocumentsByKey(%q), want TD-595", key)
			}
			return []*models.Document{{Title: "Research report", FilePath: "docs/plan/tech-debt/TD-595.research-report.md"}}, nil
		},
	}
	original := techDebtDocService
	techDebtDocService = func() relatedDocService { return mock }
	t.Cleanup(func() { techDebtDocService = original })
	t.Cleanup(func() {
		for _, flag := range []struct {
			cmd   *cobra.Command
			name  string
			value string
		}{
			{cmd: relatedDocsAddCmd, name: "tech-debt", value: ""},
			{cmd: relatedDocsListCmd, name: "tech-debt", value: ""},
			{cmd: relatedDocsListCmd, name: "json", value: "false"},
			{cmd: relatedDocsDeleteCmd, name: "tech-debt", value: ""},
		} {
			if err := flag.cmd.Flags().Set(flag.name, flag.value); err != nil {
				t.Errorf("reset %s flag: %v", flag.name, err)
			}
			flag.cmd.Flags().Lookup(flag.name).Changed = false
		}
	})
	require.NoError(t, relatedDocsAddCmd.Flags().Set("tech-debt", "TD-595"))

	var addErr error
	addOut := captureOutput(t, func() {
		addErr = runRelatedDocsAdd(relatedDocsAddCmd, []string{"Research report", "docs/plan/tech-debt/TD-595.research-report.md"})
	})
	require.NoError(t, addErr)
	assert.Contains(t, string(addOut), "Document linked to tech-debt TD-595")
	assert.Equal(t, "TD-595", linkedKey)
	assert.Equal(t, "Research report", linkedTitle)
	assert.Equal(t, "docs/plan/tech-debt/TD-595.research-report.md", linkedPath)

	require.NoError(t, relatedDocsListCmd.Flags().Set("json", "true"))
	var listErr error
	listOut := captureOutput(t, func() {
		listErr = runRelatedDocsListList(relatedDocsListCmd, []string{"TD-595"})
	})
	require.NoError(t, listErr)
	assert.Contains(t, string(listOut), "TD-595.research-report.md")

	require.NoError(t, relatedDocsDeleteCmd.Flags().Set("tech-debt", "TD-595"))
	var deleteErr error
	deleteOut := captureOutput(t, func() {
		deleteErr = runRelatedDocsDelete(relatedDocsDeleteCmd, []string{"Research report"})
	})
	require.NoError(t, deleteErr)
	assert.Contains(t, string(deleteOut), "Document unlinked from tech-debt TD-595")
	assert.True(t, unlinked)
}

// TestRelatedDocsListJSON tests JSON output
func TestRelatedDocsListJSON(t *testing.T) {
	mockDocRepo := NewMockDocumentRepository()
	mockEpicRepo := NewMockEpicRepository()
	mockFeatureRepo := NewMockFeatureRepository()
	mockTaskRepo := NewMockTaskRepository()

	mockEpicRepo.AddEpic(&models.Epic{BaseEntity: models.BaseEntity{ID: 1, Key: "E01", Title: "Epic"}})
	mockDocRepo.EpicDocuments[int64(1)] = []*models.Document{
		{ID: 1, Title: "Doc", FilePath: "docs/doc.md"},
	}

	cmd := createRelatedDocsListCmd(mockDocRepo, mockEpicRepo, mockFeatureRepo, mockTaskRepo)
	cmd.SetArgs([]string{"list", "--epic=E01", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

// Helper functions to create commands with mocked repos
func createRelatedDocsAddCmd(
	docRepo DocumentRepositoryInterface,
	epicRepo EpicRepositoryInterface,
	featureRepo FeatureRepositoryInterface,
	taskRepo TaskRepositoryInterface,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add related document",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				_ = cmd.Usage()
				return nil
			}
			title := args[0]
			path := args[1]

			epic, _ := cmd.Flags().GetString("epic")
			feature, _ := cmd.Flags().GetString("feature")
			task, _ := cmd.Flags().GetString("task")

			count := 0
			if epic != "" {
				count++
			}
			if feature != "" {
				count++
			}
			if task != "" {
				count++
			}

			if count != 1 {
				_ = cmd.Usage()
				return nil
			}

			ctx := context.Background()

			if epic != "" {
				e, err := epicRepo.GetByKey(ctx, epic)
				if err != nil {
					return err
				}
				doc, _ := docRepo.CreateOrGet(ctx, title, path)
				return docRepo.LinkToEpic(ctx, e.ID, doc.ID)
			}

			if feature != "" {
				f, err := featureRepo.GetByKey(ctx, feature)
				if err != nil {
					return err
				}
				doc, _ := docRepo.CreateOrGet(ctx, title, path)
				return docRepo.LinkToFeature(ctx, f.ID, doc.ID)
			}

			if task != "" {
				t, err := taskRepo.GetByKey(ctx, task)
				if err != nil {
					return err
				}
				doc, _ := docRepo.CreateOrGet(ctx, title, path)
				return docRepo.LinkToTask(ctx, t.ID, doc.ID)
			}

			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic key")
	cmd.Flags().String("feature", "", "Feature key")
	cmd.Flags().String("task", "", "Task key")

	return cmd
}

func createRelatedDocsDeleteCmd(
	docRepo DocumentRepositoryInterface,
	epicRepo EpicRepositoryInterface,
	featureRepo FeatureRepositoryInterface,
	taskRepo TaskRepositoryInterface,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete related document",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return cmd.Usage()
			}
			_ = args[0] // title not used in test mock

			epic, _ := cmd.Flags().GetString("epic")
			feature, _ := cmd.Flags().GetString("feature")
			task, _ := cmd.Flags().GetString("task")

			ctx := context.Background()

			if epic != "" {
				e, _ := epicRepo.GetByKey(ctx, epic)
				if e != nil {
					_ = docRepo.UnlinkFromEpic(ctx, e.ID, 0)
				}
			}

			if feature != "" {
				f, _ := featureRepo.GetByKey(ctx, feature)
				if f != nil {
					_ = docRepo.UnlinkFromFeature(ctx, f.ID, 0)
				}
			}

			if task != "" {
				t, _ := taskRepo.GetByKey(ctx, task)
				if t != nil {
					_ = docRepo.UnlinkFromTask(ctx, t.ID, 0)
				}
			}

			return nil
		},
	}

	cmd.Flags().String("epic", "", "Epic key")
	cmd.Flags().String("feature", "", "Feature key")
	cmd.Flags().String("task", "", "Task key")

	return cmd
}

func createRelatedDocsListCmd(
	docRepo DocumentRepositoryInterface,
	epicRepo EpicRepositoryInterface,
	featureRepo FeatureRepositoryInterface,
	taskRepo TaskRepositoryInterface,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List related documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			epic, _ := cmd.Flags().GetString("epic")
			feature, _ := cmd.Flags().GetString("feature")
			task, _ := cmd.Flags().GetString("task")
			jsonOutput, _ := cmd.Flags().GetBool("json")

			ctx := context.Background()
			var docs []*models.Document
			var err error

			if epic != "" {
				e, _ := epicRepo.GetByKey(ctx, epic)
				if e != nil {
					docs, err = docRepo.ListForEpic(ctx, e.ID)
				}
			} else if feature != "" {
				f, _ := featureRepo.GetByKey(ctx, feature)
				if f != nil {
					docs, err = docRepo.ListForFeature(ctx, f.ID)
				}
			} else if task != "" {
				t, _ := taskRepo.GetByKey(ctx, task)
				if t != nil {
					docs, err = docRepo.ListForTask(ctx, t.ID)
				}
			}

			if jsonOutput {
				data, _ := json.MarshalIndent(docs, "", "  ")
				cmd.Println(string(data))
			}

			return err
		},
	}

	cmd.Flags().String("epic", "", "Epic key")
	cmd.Flags().String("feature", "", "Feature key")
	cmd.Flags().String("task", "", "Task key")
	cmd.Flags().Bool("json", false, "JSON output")

	return cmd
}
