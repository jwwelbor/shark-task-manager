package taskcreation

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	testdb "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type creatorCommitTestCase struct {
	name        string
	fileState   string
	defaultPath bool
	failCommit  bool
}

type creatorCommitFixture struct {
	raw          *sql.DB
	creator      *Creator
	input        CreateTaskInput
	target       string
	relativePath string
	original     []byte
}

type creatorCommitRepositories struct {
	raw     *sql.DB
	db      *repository.DB
	epic    *repository.EpicRepository
	feature *repository.FeatureRepository
	task    *repository.TaskRepository
	history *repository.TaskHistoryRepository
}

// B114: transaction compensation must only remove documents this invocation wrote.
// These cases use the real writer and transaction so a deferred constraint fails
// at Commit, after the writer has created or linked the file.
func TestCreatorCreateTaskCommitFileOwnership(t *testing.T) {
	cases := []creatorCommitTestCase{
		{"custom_raced_commit_failure", "raced", false, true},
		{"default_raced_commit_failure", "raced", true, true},
		{"custom_existing_commit_failure", "existing", false, true},
		{"default_existing_commit_failure", "existing", true, true},
		{"custom_created_commit_failure", "created", false, true},
		{"default_created_commit_failure", "created", true, true},
		{"custom_raced_commit_success", "raced", false, false},
		{"default_raced_commit_success", "raced", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fixture := newCreatorCommitFixture(t, tc, ctx)
			result, err := fixture.creator.CreateTask(ctx, fixture.input)
			assertCreatorCommitResult(t, tc, result, err, fixture.relativePath)
			assertCreatorCommitRows(t, ctx, fixture.raw, tc.failCommit)
			assertCreatorCommitFile(t, tc, fixture.target, fixture.original)
		})
	}
}

func newCreatorCommitFixture(t *testing.T, tc creatorCommitTestCase, ctx context.Context) creatorCommitFixture {
	repos := newCreatorCommitRepositories(t, ctx)
	projectRoot := t.TempDir()
	relativePath, target, original := prepareCreatorCommitFile(t, projectRoot, tc)
	creator := NewCreator(repos.db, NewKeyGenerator(repos.task, repos.feature),
		NewValidator(repos.epic, repos.feature, repos.task), templates.NewRenderer(templates.NewLoader("")),
		repos.task, repos.history, repos.epic, repos.feature, projectRoot, nil)
	creator.SetAfterCreateHook(func(ctx context.Context, tx *sql.Tx, task *models.Task) error {
		if tc.fileState == "raced" {
			_, statErr := os.Stat(target)
			require.True(t, os.IsNotExist(statErr), "target must still be absent after preflight")
			require.NoError(t, os.WriteFile(target, original, 0644))
		}
		if tc.failCommit {
			_, err := tx.ExecContext(ctx, "INSERT INTO commit_child VALUES (999)")
			return err
		}
		return nil
	})
	input := CreateTaskInput{
		EpicKey: "E97", FeatureKey: "F01", Title: "Preserve foreign document", AgentType: "general", Priority: 5,
	}
	if !tc.defaultPath {
		input.Filename = relativePath
		input.Create = true
	}
	return creatorCommitFixture{
		raw: repos.raw, creator: creator, input: input,
		target: target, relativePath: relativePath, original: original,
	}
}

func newCreatorCommitRepositories(t *testing.T, ctx context.Context) creatorCommitRepositories {
	raw := testdb.NewIsolatedTestDB(t)
	repos := creatorCommitRepositories{
		raw: raw, db: repository.NewDB(raw),
	}
	repos.epic = repository.NewEpicRepository(repos.db)
	repos.feature = repository.NewFeatureRepository(repos.db)
	repos.task = repository.NewTaskRepository(repos.db)
	repos.history = repository.NewTaskHistoryRepository(repos.db)
	seedCreatorCommitEntities(t, ctx, repos)
	createCreatorCommitFailureTables(t, ctx, raw)
	return repos
}

func seedCreatorCommitEntities(t *testing.T, ctx context.Context, repos creatorCommitRepositories) {
	epic := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E97", Title: "File ownership"},
		Status:     models.EpicStatusDraft,
		Priority:   models.PriorityMedium,
	}
	require.NoError(t, repos.epic.Create(ctx, epic))
	featurePath := "docs/plan/E97/E97-F01/feature.md"
	feature := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E97-F01", Title: "File ownership", FilePath: &featurePath},
		EpicID:     epic.ID,
		Status:     models.FeatureStatusDraft,
	}
	require.NoError(t, repos.feature.Create(ctx, feature))
}

func createCreatorCommitFailureTables(t *testing.T, ctx context.Context, raw *sql.DB) {
	_, err := raw.ExecContext(ctx, `CREATE TABLE commit_parent (id INTEGER PRIMARY KEY)`)
	require.NoError(t, err)
	_, err = raw.ExecContext(ctx, `CREATE TABLE commit_child (
		parent_id INTEGER REFERENCES commit_parent(id) DEFERRABLE INITIALLY DEFERRED
	)`)
	require.NoError(t, err)
}

func prepareCreatorCommitFile(t *testing.T, projectRoot string, tc creatorCommitTestCase) (string, string, []byte) {
	relativePath := "raced.md"
	if tc.defaultPath {
		relativePath = filepath.Join("docs", "plan", "E97", "E97-F01", "tasks", "T-E97-F01-001.md")
	}
	target := filepath.Join(projectRoot, relativePath)
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
	original := []byte("FOREIGN-WRITER\nexact bytes\x00\xff\n")
	if tc.fileState == "existing" {
		require.NoError(t, os.WriteFile(target, original, 0644))
	}
	return relativePath, target, original
}

func assertCreatorCommitResult(
	t *testing.T, tc creatorCommitTestCase, result *CreateTaskResult, err error, relativePath string,
) {
	if tc.failCommit {
		require.ErrorContains(t, err, "failed to commit transaction")
		require.Nil(t, result)
		return
	}
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.FileWasLinked)
	assert.Equal(t, relativePath, result.FilePath)
}

func assertCreatorCommitRows(t *testing.T, ctx context.Context, raw *sql.DB, failCommit bool) {
	var sequence int
	var databaseName, databasePath string
	require.NoError(t, raw.QueryRowContext(ctx, "PRAGMA database_list").Scan(&sequence, &databaseName, &databasePath))
	observer, err := sql.Open("sqlite", databasePath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, observer.Close()) })
	for _, query := range []string{"SELECT COUNT(*) FROM tasks", "SELECT COUNT(*) FROM task_history"} {
		var count int
		require.NoError(t, observer.QueryRowContext(ctx, query).Scan(&count))
		if failCommit {
			assert.Zero(t, count, query)
		} else {
			assert.Equal(t, 1, count, query)
		}
	}
}

func assertCreatorCommitFile(t *testing.T, tc creatorCommitTestCase, target string, original []byte) {
	content, err := os.ReadFile(target)
	if tc.fileState == "created" {
		assert.True(t, os.IsNotExist(err), "newly created file must be removed after failed commit: %v", err)
		return
	}
	require.NoError(t, err, "foreign document must survive")
	assert.Equal(t, original, content)
}
