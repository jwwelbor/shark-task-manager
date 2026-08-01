package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

type questionBlockerFunc func(context.Context, models.EntityType, string) (*services.QuestionBlock, error)

func (f questionBlockerFunc) Check(ctx context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
	return f(ctx, entityType, key)
}

// TC-305: keyed next must check the direct Question gate after status identity
// is known but before it renders placeholders or an action/prompt.
func TestResolveNextBlockedCandidatePausesBeforeDispatchWork_TC305(t *testing.T) {
	placeholderCalls := 0
	actionCalls := 0
	cache := &nextAdapterCache{
		questionBlocker: questionBlockerFunc(func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
			if entityType != models.EntityTypeFeature || key != "E39-F03" {
				t.Fatalf("blocker candidate = %s %s, want feature E39-F03", entityType, key)
			}
			return &services.QuestionBlock{QuestionKey: "Q001", Summary: "Choose the release", ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
		}),
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{EntityType: models.EntityTypeFeature, EntityKey: "E39-F03", CurrentStatus: "active"}},
				generator: runnerPlaceholderFunc(func(context.Context, string) (map[string]string, error) {
					placeholderCalls++
					return nil, errors.New("blocked next must not generate placeholders")
				}),
				actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					actionCalls++
					return nil, errors.New("blocked next must not resolve action")
				}},
			},
		},
	}

	got, err := resolveNext(context.Background(), cache, "feature", "E39-F03", 0)
	if err != nil {
		t.Fatalf("resolveNext(blocked) error = %v", err)
	}
	if got.Action != action.ActionPause || got.EntityKey != "E39-F03" || got.Status != "active" {
		t.Fatalf("resolveNext(blocked) = %#v, want compact pause", got)
	}
	if got.QuestionBlock == nil || *got.QuestionBlock != (services.QuestionBlock{QuestionKey: "Q001", Summary: "Choose the release", ResolutionOwner: "owner", CurrentResponder: "alice"}) {
		t.Fatalf("question_block = %#v, want I-03 handoff", got.QuestionBlock)
	}
	if got.AgentType != "" || got.Provider != "" || got.Model != "" || got.Effort != "" || got.Prompt != "" {
		t.Fatalf("blocked next leaked dispatch fields: %#v", got)
	}
	if placeholderCalls != 0 || actionCalls != 0 {
		t.Fatalf("blocked next placeholder/action calls = %d/%d, want 0/0", placeholderCalls, actionCalls)
	}
}

// TestResolveNextPropagatesQuestionBlockerCheckError locks in that a
// questionBlocker.Check failure (a real read/state error, not "no block")
// propagates out of resolveNext and stops before any placeholder/action
// dispatch work runs -- a regression that silently swallowed the error
// (e.g. `block, _ := cache.questionBlocker.Check(...)`) would otherwise pass
// every other test in this file, since none of them exercise the error path.
func TestResolveNextPropagatesQuestionBlockerCheckError(t *testing.T) {
	placeholderCalls := 0
	actionCalls := 0
	checkErr := errors.New("Question blocker load candidate: repository unavailable")
	cache := &nextAdapterCache{
		questionBlocker: questionBlockerFunc(func(context.Context, models.EntityType, string) (*services.QuestionBlock, error) {
			return nil, checkErr
		}),
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{EntityType: models.EntityTypeFeature, EntityKey: "E39-F03", CurrentStatus: "active"}},
				generator: runnerPlaceholderFunc(func(context.Context, string) (map[string]string, error) {
					placeholderCalls++
					return nil, errors.New("blocker error must not reach placeholder generation")
				}),
				actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					actionCalls++
					return nil, errors.New("blocker error must not reach action resolution")
				}},
			},
		},
	}

	_, err := resolveNext(context.Background(), cache, "feature", "E39-F03", 0)
	if err == nil {
		t.Fatal("resolveNext() error = nil, want the propagated Question blocker error")
	}
	if !errors.Is(err, checkErr) {
		t.Fatalf("resolveNext() error = %v, want it to wrap %v", err, checkErr)
	}
	if placeholderCalls != 0 || actionCalls != 0 {
		t.Fatalf("blocker error placeholder/action calls = %d/%d, want 0/0", placeholderCalls, actionCalls)
	}
}

// TC-306: a blocked child is parked, not a cascade result. Keyed next must
// continue to an unlinked live sibling without changing the parent.
func TestResolveNextCascadeFallsThroughBlockedChild_TC306(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	defer func() { planDescribeDispatchableChildren = originalDescribe }()
	planDescribeDispatchableChildren = func(_ context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
		if entityType != "epic" || key != "E39" {
			t.Fatalf("cascade lookup = %s %s, want epic E39", entityType, key)
		}
		return services.PlanHierarchyChildrenState{Children: []services.PlanHierarchyChild{
			{Key: "E39-F03", EntityType: models.EntityTypeFeature},
			{Key: "E39-F04", EntityType: models.EntityTypeFeature},
		}, TotalChildren: 2, NonTerminalChildren: 2}, nil
	}

	cache := &nextAdapterCache{
		questionBlocker: questionBlockerFunc(func(_ context.Context, entityType models.EntityType, key string) (*services.QuestionBlock, error) {
			if entityType == models.EntityTypeFeature && key == "E39-F03" {
				return &services.QuestionBlock{QuestionKey: "Q001", Summary: "Gate", ResolutionOwner: "owner", CurrentResponder: "alice"}, nil
			}
			return nil, nil
		}),
		entries: map[string]*nextAdapters{
			"epic": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{EntityType: models.EntityTypeEpic, EntityKey: "E39", CurrentStatus: "active"}},
				actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					return &action.PopulatedAction{Action: "cascade"}, nil
				}},
			},
			"feature": {
				transitioner: keyedByEntityTransitioner{statuses: map[string]string{"E39-F03": "active", "E39-F04": "active"}},
				actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					return &action.PopulatedAction{Action: action.ActionSpawnAgent, AgentType: "developer", Instruction: "work"}, nil
				}},
			},
		},
	}

	got, err := resolveNext(context.Background(), cache, "epic", "E39", 0)
	if err != nil {
		t.Fatalf("resolveNext(cascade) error = %v", err)
	}
	if got.EntityKey != "E39-F04" || got.Action != action.ActionSpawnAgent {
		t.Fatalf("resolveNext(cascade) = %#v, want unblocked E39-F04 dispatch", got)
	}
	if len(got.ResolvedVia) != 1 || got.ResolvedVia[0] != "E39" {
		t.Fatalf("resolved_via = %#v, want E39", got.ResolvedVia)
	}
}
