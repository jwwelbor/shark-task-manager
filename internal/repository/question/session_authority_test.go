package question_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	questionrepo "github.com/jwwelbor/shark-task-manager/internal/repository/question"
	"github.com/jwwelbor/shark-task-manager/internal/repository/repoerr"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/stretchr/testify/require"
)

type questionLeaseFixture struct {
	db     *sql.DB
	repo   *questionrepo.QuestionRepository
	svc    *services.QuestionService
	claims *services.ClaimService
	q      *models.Question
}

func newQuestionLeaseFixture(t *testing.T) questionLeaseFixture {
	t.Helper()
	ctx := context.Background()
	database := test.NewIsolatedTestDB(t)
	conn := dbconn.NewDB(database)
	repo := questionrepo.NewQuestionRepository(conn)
	svc, err := services.NewQuestionService(repo)
	require.NoError(t, err)
	ttl := time.Minute
	claims := services.NewClaimService(claimrepo.NewRepository(conn), &ttl)
	svc.SetClaimReader(claims)
	q := createQuestion(t, ctx, svc, "Clarify scope", "asker")
	_, err = svc.ConfigureWorkflow(ctx, services.ConfigureWorkflowInput{Key: q.Key, ResolutionOwner: "owner", Responders: []string{"alice"}})
	require.NoError(t, err)
	_, err = claims.Claim(ctx, services.ClaimInput{EntityType: "question", EntityKey: q.Key, SessionID: "session-a", ClaimedBy: "alice"})
	require.NoError(t, err)
	return questionLeaseFixture{database, repo, svc, claims, q}
}

type questionClaimReadBarrier struct {
	*services.ClaimService
	afterRead func()
}

func (b questionClaimReadBarrier) Get(ctx context.Context, typ, key string) (*models.EntityClaim, error) {
	claim, err := b.ClaimService.Get(ctx, typ, key)
	b.afterRead()
	return claim, err
}

// C18: A passed the service's identity check, but B acquired the same
// responder's lease before A's response could change the Question.
func TestQuestionResponseRequiresSessionAtWrite(t *testing.T) {
	f := newQuestionLeaseFixture(t)
	ctx := context.Background()
	input := services.RecordQuestionResponseInput{Key: f.q.Key, SessionID: "session-a", Responder: "alice", Summary: "first answer", EvidencePointer: "notes/A.md"}
	before, err := f.repo.GetByKey(ctx, f.q.Key)
	require.NoError(t, err)
	counts := countQuestionDependentAssociations(t, ctx, f.db, f.q)
	f.svc.SetClaimReader(questionClaimReadBarrier{f.claims, func() {
		_, err := f.claims.Claim(ctx, services.ClaimInput{EntityType: "question", EntityKey: f.q.Key, SessionID: "session-b", ClaimedBy: "alice", Force: true})
		require.NoError(t, err)
	}})
	_, err = f.svc.RecordResponse(ctx, input)
	require.ErrorIs(t, err, repoerr.ErrConditionalWriteConflict)
	after, err := f.repo.GetByKey(ctx, f.q.Key)
	require.NoError(t, err)
	require.Equal(t, before.ContextData, after.ContextData)
	require.Equal(t, before.Status, after.Status)
	afterCounts := countQuestionDependentAssociations(t, ctx, f.db, f.q)
	require.Equal(t, counts.notes, afterCounts.notes)
	require.Equal(t, counts.history, afterCounts.history)

	f.svc.SetClaimReader(f.claims)
	input.SessionID, input.Summary, input.EvidencePointer = "session-b", "second answer", "notes/B.md"
	_, err = f.db.Exec("UPDATE entity_claims SET last_heartbeat=datetime('now','-1 hour') WHERE entity_key=?", f.q.Key)
	require.NoError(t, err)
	result, err := f.svc.RecordResponse(ctx, input)
	require.NoError(t, err)
	require.Equal(t, models.QuestionStatusReadyForResolution, result.Status)
	state, err := models.DecodeQuestionState(result.ContextData)
	require.NoError(t, err)
	require.Len(t, state.Responses, 1)
	require.Equal(t, input.EvidencePointer, state.Responses[0].EvidencePointer)
	released, err := f.claims.Release(ctx, "question", f.q.Key, input.SessionID, "", false)
	require.NoError(t, err)
	require.True(t, released)
	_, err = f.svc.RecordResponse(ctx, input)
	require.NoError(t, err)
	afterCounts = countQuestionDependentAssociations(t, ctx, f.db, f.q)
	require.Equal(t, counts.notes+1, afterCounts.notes)
	require.Equal(t, counts.history+1, afterCounts.history)
}

// Pauses after the actual destination lookup and before the Question transaction.
type questionDestinationBarrier struct {
	*questionrepo.QuestionRepository
	afterLookup func()
}

func (b questionDestinationBarrier) NoteExists(ctx context.Context, id string) (bool, error) {
	found, err := b.QuestionRepository.NoteExists(ctx, id)
	b.afterLookup()
	return found, err
}

func TestQuestionResolutionRequiresLiveFinalSessionAtWrite(t *testing.T) {
	for _, mode := range []string{"current", "takeover", "expired", "released", "successor supplies its session", "direct owner"} {
		t.Run(mode, func(t *testing.T) {
			f := newQuestionLeaseFixture(t)
			ctx := context.Background()
			_, err := f.svc.RecordResponse(ctx, services.RecordQuestionResponseInput{Key: f.q.Key, SessionID: "session-a", Responder: "alice", Summary: "accepted answer", EvidencePointer: "notes/answer.md"})
			require.NoError(t, err)
			before, err := f.repo.GetByKey(ctx, f.q.Key)
			require.NoError(t, err)
			counts := countQuestionDependentAssociations(t, ctx, f.db, f.q)
			var noteID string
			require.NoError(t, f.db.QueryRow("SELECT id FROM entity_notes WHERE entity_type='question' AND entity_id=? ORDER BY id DESC LIMIT 1", f.q.ID).Scan(&noteID))
			svc, err := services.NewQuestionService(questionDestinationBarrier{f.repo, func() {
				switch mode {
				case "takeover", "successor supplies its session":
					_, err := f.claims.Claim(ctx, services.ClaimInput{EntityType: "question", EntityKey: f.q.Key, SessionID: "session-b", ClaimedBy: "alice", Force: true})
					require.NoError(t, err)
				case "expired":
					_, err := f.db.Exec("UPDATE entity_claims SET last_heartbeat=datetime('now','-2 minutes') WHERE entity_key=?", f.q.Key)
					require.NoError(t, err)
				case "released", "direct owner":
					released, err := f.claims.Release(ctx, "question", f.q.Key, "session-a", "", false)
					require.NoError(t, err)
					require.True(t, released)
				}
			}})
			require.NoError(t, err)
			svc.SetClaimReader(f.claims)
			session := "session-a"
			if mode == "direct owner" {
				session = ""
			}
			if mode == "successor supplies its session" {
				session = "session-b"
			}
			_, err = svc.Resolve(ctx, services.ResolveQuestionInput{Key: f.q.Key, Owner: "owner", Kind: "local_clarification", Pointer: "note:" + noteID, SessionID: session})
			if mode == "current" || mode == "direct owner" {
				require.NoError(t, err)
				after, err := f.repo.GetByKey(ctx, f.q.Key)
				require.NoError(t, err)
				require.Equal(t, models.QuestionStatusResolved, after.Status)
				afterCounts := countQuestionDependentAssociations(t, ctx, f.db, f.q)
				require.Equal(t, counts.history+1, afterCounts.history)
				require.Equal(t, counts.notes+1, afterCounts.notes)
				return
			}
			var rule *services.QuestionRuleError
			require.ErrorAs(t, err, &rule)
			require.Equal(t, services.QuestionRuleConflict, rule.Class)
			after, err := f.repo.GetByKey(ctx, f.q.Key)
			require.NoError(t, err)
			require.Equal(t, before.Status, after.Status)
			require.Equal(t, before.ContextData, after.ContextData)
			afterCounts := countQuestionDependentAssociations(t, ctx, f.db, f.q)
			require.Equal(t, counts.notes, afterCounts.notes)
			require.Equal(t, counts.history, afterCounts.history)
			if mode == "takeover" || mode == "successor supplies its session" {
				claim, err := f.claims.Get(ctx, "question", f.q.Key)
				require.NoError(t, err)
				require.Equal(t, "session-b", claim.SessionID)
			}
		})
	}
}
