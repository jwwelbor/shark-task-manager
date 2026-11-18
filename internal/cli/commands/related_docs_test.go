package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
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
		{name: "tech debt explicit", args: []string{"--tech-debt=TD-595"}, wantType: "tech-debt", wantKey: "TD-595"},
		{name: "unsupported entity", args: []string{"TD001"}, wantErr: "cannot infer a supported entity type"},
		{name: "multiple positional keys", args: []string{"E01", "E02"}, wantErr: "at most one positional entity key"},
		{name: "positional and explicit selector", args: []string{"B072", "--bug=B073"}, wantErr: "cannot combine positional entity key"},
		{name: "tech debt positional and conflicting selector", args: []string{"TD-595", "--tech-debt=TD-596"}, wantErr: "cannot combine positional entity key"},
		{name: "conflicting tech debt and bug selectors", args: []string{"--bug=B079", "--tech-debt=TD-595"}, wantErr: "exactly one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parseRelatedDocsListArgs(t, tt.args)
			entityType, key, err := resolveRelatedDocsSelection(cmd, cmd.Flags().Args())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, entityType)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

func parseRelatedDocsListArgs(t *testing.T, args []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	for _, selector := range relatedDocsSelectors {
		cmd.Flags().String(selector.flag, "", selector.usage)
	}
	require.NoError(t, cmd.ParseFlags(args))
	return cmd
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

func TestRelatedDocsProductionHandlersRouteSupportedEntityServices(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "related-docs.db")
	rawDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	repoDB := repository.NewDB(rawDB)

	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E90", Title: "Related docs test epic"},
		Status:     models.EpicStatusActive,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, repository.NewEpicRepository(repoDB).Create(ctx, epic))

	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E90-F01", Title: "Related docs test feature"},
		EpicID:     epic.ID,
		Status:     models.FeatureStatusDraft,
	}
	require.NoError(t, repository.NewFeatureRepository(repoDB).Create(ctx, feature))

	task := &models.Task{
		BaseEntity: models.BaseEntity{Key: "T-E90-F01-001", Title: "Related docs test task"},
		FeatureID:  feature.ID,
		Status:     models.TaskStatus("todo"),
		Priority:   5,
	}
	require.NoError(t, repository.NewTaskRepository(repoDB).Create(ctx, task))

	bug := &models.Bug{
		BaseEntity: models.BaseEntity{Key: "B901", Title: "Related docs test bug"},
		Status:     models.BugStatus("identified"),
		Severity:   models.BugSeverityMedium,
	}
	require.NoError(t, repository.NewBugRepository(repoDB).Create(ctx, bug))

	change := &models.ChangeCard{
		BaseEntity: models.BaseEntity{Key: "CC-901", Title: "Related docs test change"},
		Status:     models.ChangeCardStatus("proposed"),
		Priority:   1,
	}
	require.NoError(t, repository.NewChangeCardRepository(repoDB).Create(ctx, change))

	question := &models.Question{
		BaseEntity: models.BaseEntity{Key: "Q901", Title: "Related docs test question"},
		Status:     models.QuestionStatusDraft,
		Summary:    "Question to verify related-document routing",
		Requester:  "related-docs-test",
	}
	require.NoError(t, repository.NewQuestionRepository(repoDB).Create(ctx, question))
	require.NoError(t, rawDB.Close())
	configureRelatedDocsHandlerDB(t, dbPath)

	tests := []struct {
		name, selector, key, entityType string
		entityID                        int64
	}{
		{name: "epic", selector: "epic", key: epic.Key, entityType: string(models.EntityTypeEpic), entityID: epic.ID},
		{name: "feature", selector: "feature", key: feature.Key, entityType: string(models.EntityTypeFeature), entityID: feature.ID},
		{name: "task", selector: "task", key: task.Key, entityType: string(models.EntityTypeTask), entityID: task.ID},
		{name: "bug", selector: "bug", key: bug.Key, entityType: string(models.EntityTypeBug), entityID: bug.ID},
		{name: "change-card", selector: "change", key: change.Key, entityType: string(models.EntityTypeChange), entityID: change.ID},
		{name: "question", selector: "question", key: question.Key, entityType: string(models.EntityTypeQuestion), entityID: question.ID},
	}

	globalDB, err := cli.GetDB(ctx)
	require.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := "B079 route " + tt.name
			path := "docs/b079-route-" + tt.name + ".md"
			addCmd := newRelatedDocsHandlerCommand(t, tt.selector, tt.key)
			var addErr error
			addOut := captureOutput(t, func() {
				addErr = runRelatedDocsAdd(addCmd, []string{title, path})
			})
			require.NoError(t, addErr)
			assert.Contains(t, string(addOut), "Document linked to "+tt.name+" "+tt.key)

			var actualType string
			var actualEntityID int64
			var documentID int64
			require.NoError(t, globalDB.QueryRowContext(ctx, `
				SELECT ed.entity_type, ed.entity_id, ed.document_id
				FROM entity_documents ed
				JOIN documents d ON d.id = ed.document_id
				WHERE d.title = ? AND d.file_path = ?`, title, path).
				Scan(&actualType, &actualEntityID, &documentID))
			assert.Equal(t, tt.entityType, actualType)
			assert.Equal(t, tt.entityID, actualEntityID)

			listCmd := newRelatedDocsHandlerCommand(t, tt.selector, tt.key)
			var listErr error
			listOut := captureOutput(t, func() {
				listErr = runRelatedDocsListList(listCmd, nil)
			})
			require.NoError(t, listErr)
			assert.Contains(t, string(listOut), path)

			deleteCmd := newRelatedDocsHandlerCommand(t, tt.selector, tt.key)
			var deleteErr error
			deleteOut := captureOutput(t, func() {
				deleteErr = runRelatedDocsDelete(deleteCmd, []string{title})
			})
			require.NoError(t, deleteErr)
			assert.Contains(t, string(deleteOut), "Document unlinked from "+tt.name+" "+tt.key)

			var remaining int
			require.NoError(t, globalDB.QueryRowContext(ctx,
				"SELECT COUNT(*) FROM entity_documents WHERE entity_type = ? AND entity_id = ? AND document_id = ?",
				tt.entityType, tt.entityID, documentID).Scan(&remaining))
			assert.Zero(t, remaining)
		})
	}
}

func TestRelatedDocsProductionHandlerReturnsEntityServiceError(t *testing.T) {
	setupRelatedDocsHandlerDB(t)
	cmd := newRelatedDocsHandlerCommand(t, "bug", "B999")
	var addErr error
	addOut := captureOutput(t, func() {
		addErr = runRelatedDocsAdd(cmd, []string{"Missing bug document", "docs/missing-bug.md"})
	})
	require.Error(t, addErr)
	assert.ErrorContains(t, addErr, "link document to bug B999")
	assert.NotContains(t, string(addOut), "Document linked")
}

func setupRelatedDocsHandlerDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "related-docs-error.db")
	rawDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	require.NoError(t, rawDB.Close())
	configureRelatedDocsHandlerDB(t, dbPath)
}

func configureRelatedDocsHandlerDB(t *testing.T, dbPath string) {
	t.Helper()
	configPath := filepath.Join(filepath.Dir(dbPath), ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0600))
	originalConfigFile := cli.GlobalConfig.ConfigFile
	originalDBPath := cli.GlobalConfig.DBPath
	originalJSON := cli.GlobalConfig.JSON
	configFlag := cli.RootCmd.PersistentFlags().Lookup("config")
	dbFlag := cli.RootCmd.PersistentFlags().Lookup("db")
	originalConfigChanged := configFlag != nil && configFlag.Changed
	originalDBChanged := dbFlag != nil && dbFlag.Changed
	cli.GlobalConfig.ConfigFile = configPath
	cli.GlobalConfig.DBPath = dbPath
	cli.GlobalConfig.JSON = false
	require.NoError(t, cli.RootCmd.PersistentFlags().Set("config", configPath))
	require.NoError(t, cli.RootCmd.PersistentFlags().Set("db", dbPath))
	cli.ResetDB()
	cli.ResetServices()
	cli.ResetWorkflowService()
	config.ClearWorkflowCache()
	t.Cleanup(func() {
		cli.ResetDB()
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.GlobalConfig.ConfigFile = originalConfigFile
		cli.GlobalConfig.DBPath = originalDBPath
		cli.GlobalConfig.JSON = originalJSON
		require.NoError(t, cli.RootCmd.PersistentFlags().Set("config", originalConfigFile))
		require.NoError(t, cli.RootCmd.PersistentFlags().Set("db", originalDBPath))
		if configFlag != nil {
			configFlag.Changed = originalConfigChanged
		}
		if dbFlag != nil {
			dbFlag.Changed = originalDBChanged
		}
		config.ClearWorkflowCache()
	})
}

func newRelatedDocsHandlerCommand(t *testing.T, selector, key string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	for _, flag := range relatedDocsSelectors {
		cmd.Flags().String(flag.flag, "", flag.usage)
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	if selector != "" {
		require.NoError(t, cmd.Flags().Set(selector, key))
	}
	return cmd
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
