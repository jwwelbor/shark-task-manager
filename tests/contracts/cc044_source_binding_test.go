package contracts

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/db"
	"github.com/jwwelbor/shark-task-manager/internal/gatepersist"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/require"
)

type changedSourceTransition struct {
	delegate gatepersist.Transitioner
	repo     interface {
		UpdateStatus(context.Context, int64, models.BugStatus) error
	}
}

func (t *changedSourceTransition) Transition(ctx context.Context, typ models.EntityType, key, target, reason, agent string, guard gatepersist.TransitionGuard) (string, bool, error) {
	if err := t.repo.UpdateStatus(ctx, 1, "qa"); err != nil {
		return "", false, err
	}
	return t.delegate.Transition(ctx, typ, key, target, reason, agent, guard)
}

func TestBugGateRejectsSourceChangedAfterAdmission(t *testing.T) {
	ctx := context.Background()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	binary := filepath.Join(t.TempDir(), "shark")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/shark")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	project := t.TempDir()
	baseline := initCC044Baseline(t, project)
	dbPath := filepath.Join(project, "fixture.db")
	shark := func(args ...string) {
		t.Helper()
		cmd := exec.Command(binary, append(args, "--db", dbPath, "--json")...)
		cmd.Dir = project
		data, e := cmd.CombinedOutput()
		require.NoError(t, e, string(data))
	}
	shark("admin", "init", "--non-interactive")
	shark("bug", "create", "Changed source")
	shark("status", "set", "B001", "plan_review", "--force", "--reason", "fixture setup")
	shark("claim", "B001", "--session", "fixture-session", "--by", "test")
	sqlDB, err := db.InitDB(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	storage := repository.NewDB(sqlDB)
	bugRepo := repository.NewBugRepository(storage)
	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeBug, services.NewBugRepositoryAdapter(bugRepo))
	notes, err := services.NewNoteService(repository.NewEntityNoteRepository(storage), registry)
	require.NoError(t, err)
	historyRepo := repository.NewEntityHistoryRepository(storage)
	history := services.NewEntityHistoryService(historyRepo, registry)
	flow := workflow.NewService(project)
	entity := services.NewEntityService(flow)
	entity.SetHistoryRepo(historyRepo)
	transition := gatepersist.NewEntityServiceTransitioner(entity, registry, flow)
	claims := services.NewClaimService(claimrepo.NewRepository(storage), nil)
	interleaved := &changedSourceTransition{delegate: transition, repo: bugRepo}
	coord := gatepersist.NewCoordinator(notes, notes, history, gatepersist.NewWorkflowStatusValidator(flow), interleaved, transition, claims, transition)
	coord.ClaimVerifier = claims
	coord.Admission = runner.NewBugGateAdmission(notes, history)
	coord.BeforeTransition = runner.PrepareBugApprovalSnapshot
	planRel := "docs/plan/bugs/B001.repair-plan.md"
	plan := []byte("approved plan")
	require.NoError(t, os.WriteFile(filepath.Join(project, planRel), plan, 0600))
	commitCC044Files(t, project, planRel)
	digest := sha256.Sum256(plan)
	payload, err := json.Marshal(map[string]any{"kind": "final", "recommended_outcome": "pass", "evidence": []map[string]string{{"kind": "bug_repair_plan", "pointer": planRel, "summary": fmt.Sprintf("sha256:%x", digest)}, {"kind": "bug_review_baseline", "pointer": baseline}}, "gate_result": map[string]any{"schema_version": 1, "summary": "approved plan"}})
	require.NoError(t, err)
	_, err = runner.IngestGateResult(ctx, runner.GateIngestRequest{EnvelopeBytes: payload, Coordinator: coord, ProjectRoot: project, RunID: "wrong-source", EntityKey: "B001", EntityType: models.EntityTypeBug, SourceStatus: "plan_review", Gate: "plan_review", Session: gatepersist.Session{ID: "fixture-session"}, OutcomeRoles: map[string]gateresult.OutcomeRole{"pass": gateresult.RoleSuccess}, Outcomes: map[string]string{"pass": "development"}})
	require.ErrorIs(t, err, services.ErrAdvanceGuardStaleFromStatus)
	bug, err := bugRepo.GetByKey(ctx, "B001")
	require.NoError(t, err)
	require.Equal(t, models.BugStatus("qa"), bug.Status)
}
