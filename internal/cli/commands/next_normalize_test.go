package commands

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// TestNormalizeWireAction_TableDriven covers every row of the verb-mapping
// table in next.go. Adding a new internal verb without updating this table
// will leave it routed through the "error" branch — which is the loudest
// possible signal to the next reviewer.
func TestNormalizeWireAction_TableDriven(t *testing.T) {
	// A nextInfo with one productive forward transition (in_development)
	// and several non-productive siblings, so the auto-advance picker
	// produces "in_development" deterministically.
	productiveNextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "in_development"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}

	// A nextInfo whose only available transitions are non-productive —
	// auto-advance has nothing safe to pick, so we expect autoAdvanceTarget="".
	stuckNextInfo := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}

	tests := []struct {
		name           string
		internalAction string
		agentType      string
		nextInfo       *services.NextStatusInfo
		wantWire       string
		wantTarget     string
	}{
		{"spawn_agent passes through", "spawn_agent", "developer", productiveNextInfo, "spawn_agent", ""},
		{"check_or_resume becomes spawn_agent", "check_or_resume", "developer", productiveNextInfo, "spawn_agent", ""},
		{"advance_status with agent becomes spawn_agent", "advance_status", "developer", productiveNextInfo, "spawn_agent", ""},
		{"advance_status without agent triggers auto-advance", "advance_status", "", productiveNextInfo, "advance_and_recurse", "in_development"},
		{"advance_status with no safe target", "advance_status", "", stuckNextInfo, "advance_and_recurse", ""},
		{"pause stays pause", "pause", "", productiveNextInfo, "pause", ""},
		{"archive stays archive", "archive", "", productiveNextInfo, "archive", ""},
		{"empty verb with agent_type becomes spawn_agent", "", "developer", productiveNextInfo, "spawn_agent", ""},
		{"empty verb without agent_type pauses", "", "", productiveNextInfo, "pause", ""},
		{"unknown verb returns error sentinel", "frobnicate", "developer", productiveNextInfo, "error", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire, target := normalizeWireAction(tt.internalAction, tt.agentType, tt.nextInfo)
			if wire != tt.wantWire {
				t.Errorf("wireAction=%q want %q", wire, tt.wantWire)
			}
			if target != tt.wantTarget {
				t.Errorf("autoAdvanceTarget=%q want %q", target, tt.wantTarget)
			}
		})
	}
}

func TestPickAutoAdvanceTarget_PrefersFirstProductive(t *testing.T) {
	info := &services.NextStatusInfo{
		AvailableTransitions: []services.TransitionInfoWithAction{
			// Authoring may list these in workflow-natural order.
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "ready_for_assessment"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "cancelled"}},
			{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
		},
	}
	if got := pickAutoAdvanceTarget(info); got != "ready_for_assessment" {
		t.Errorf("expected ready_for_assessment, got %q", got)
	}
}

func TestPickAutoAdvanceTarget_NilSafe(t *testing.T) {
	if got := pickAutoAdvanceTarget(nil); got != "" {
		t.Errorf("nil nextInfo should return empty, got %q", got)
	}
}

// ─── B022 regression: legacy status graceful degradation ─────────────────────

// TestIsStatusNotFoundError_DetectsStatusNotFoundError verifies that the helper
// correctly identifies action.StatusNotFoundError so resolveNext can apply
// graceful degradation (pause) instead of propagating the error.
//
// Bug: B022 — "shark next exits 1 on legacy task statuses (in_approval,
// ready_for_approval) instead of degrading". When a task's current status is
// not defined in the workflow YAML, GetStatusActionPopulated returns a
// StatusNotFoundError; the fix converts that to a pause action.
func TestIsStatusNotFoundError_DetectsStatusNotFoundError(t *testing.T) {
	snfe := &action.StatusNotFoundError{Status: "in_approval"}

	if !isStatusNotFoundError(snfe) {
		t.Error("isStatusNotFoundError should return true for *action.StatusNotFoundError")
	}
}

func TestIsStatusNotFoundError_IgnoresOtherErrors(t *testing.T) {
	otherErr := errors.New("some other error")

	if isStatusNotFoundError(otherErr) {
		t.Error("isStatusNotFoundError should return false for a non-StatusNotFoundError")
	}
}

func TestIsStatusNotFoundError_NilReturnsFalse(t *testing.T) {
	if isStatusNotFoundError(nil) {
		t.Error("isStatusNotFoundError should return false for nil")
	}
}

func TestIsStatusNotFoundError_WrappedErrorDetected(t *testing.T) {
	// Wrapped StatusNotFoundError (e.g., from fmt.Errorf("...: %w", snfe))
	// should also be detected, since errors.As unwraps.
	snfe := &action.StatusNotFoundError{Status: "ready_for_approval"}
	wrapped := fmt.Errorf("failed to populate action for status %q: %w", "ready_for_approval", snfe)

	if !isStatusNotFoundError(wrapped) {
		t.Error("isStatusNotFoundError should return true for a wrapped *action.StatusNotFoundError")
	}
}

// TC-103: an unconfigured Question workflow must use the standard keyed-next
// compatibility pause. The base record remains readable while no responder
// state has been configured; treating the missing workflow status as a fatal
// error would make the normal F02 configuration sequence undispatchable.
func TestResolveNextQuestionUnconfiguredWorkflowPauses_TC103(t *testing.T) {
	transitioner := nextStatusOnlyTransitioner{next: &services.NextStatusInfo{
		EntityType:    models.EntityTypeQuestion,
		EntityKey:     "Q001",
		CurrentStatus: "unsupported",
	}}
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"question": {
				transitioner: transitioner,
				actionSvc: &action.MockActionService{
					GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
						return nil, &action.StatusNotFoundError{Status: "unsupported"}
					},
				},
			},
		},
	}
	got, err := resolveNext(context.Background(), cache, "question", "Q001", 0)
	if err != nil {
		t.Fatalf("resolveNext(question unconfigured) error = %v, want compatibility pause", err)
	}
	if got.Action != action.ActionPause || got.EntityKey != "Q001" || got.Status != "unsupported" || got.Error == "" {
		t.Fatalf("resolveNext(question unconfigured) = %#v, want pause with compatibility diagnostic", got)
	}
}

// TC-010: the Question draft fixture is a non-dispatching pause. It must not
// manufacture the parent-loop preamble or a worker prompt.
func TestResolveNextQuestionDraftReturnsEmptyDispatchFields_TC010(t *testing.T) {
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"question": {
				transitioner: nextStatusOnlyTransitioner{next: &services.NextStatusInfo{
					EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: "draft",
				}},
				actionSvc: &action.MockActionService{
					GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
						return &action.PopulatedAction{Action: action.ActionPause, Instruction: "ignored"}, nil
					},
				},
			},
		},
	}
	got, err := resolveNext(context.Background(), cache, "question", "Q001", 0)
	if err != nil {
		t.Fatalf("resolveNext(question draft) error = %v", err)
	}
	if got.Action != action.ActionPause || got.AgentType != "" || got.Provider != "" || got.Model != "" || got.Prompt != "" || got.Error != "" {
		t.Fatalf("resolveNext(question draft) = %#v, want exact empty pause envelope", got)
	}
}

// TC-103/TC-104: a live lease pauses a competing keyed-next request without
// presenting that operational condition as a terminal workflow state. The
// parent that owns the lease can therefore still perform status advance.
func TestResolveNextClaimedQuestionPausesWithoutTerminalizingParentLifecycle_TC103_TC104(t *testing.T) {
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"question": {
				transitioner: nextStatusOnlyTransitioner{next: &services.NextStatusInfo{
					EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: "open", IsClaimed: true,
				}},
			},
		},
	}
	got, err := resolveNext(context.Background(), cache, "question", "Q001", 0)
	if err != nil {
		t.Fatalf("resolveNext(claimed question) error = %v", err)
	}
	if got.Action != action.ActionPause || got.Status != "open" || got.AgentType != "" || got.Prompt != "" {
		t.Fatalf("resolveNext(claimed question) = %#v, want empty competing-dispatch pause", got)
	}
}

// TC-103: ready_for_resolution has no responder. Keyed next must emit the
// normal pause envelope before either placeholder or workflow-action lookup;
// otherwise the Question responder adapter turns a human checkpoint into an
// internal error.
func TestResolveNextQuestionReadyForResolutionPausesBeforeRendering_TC103(t *testing.T) {
	placeholderCalls := 0
	actionCalls := 0
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"question": {
				transitioner: nextStatusOnlyTransitioner{next: &services.NextStatusInfo{
					EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: "ready_for_resolution", IsTerminal: true,
				}},
				generator: runnerPlaceholderFunc(func(context.Context, string) (map[string]string, error) {
					placeholderCalls++
					return nil, errors.New("responder placeholders must not run")
				}),
				actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
					actionCalls++
					return nil, errors.New("workflow action lookup must not run")
				}},
			},
		},
	}

	got, err := resolveNext(context.Background(), cache, "question", "Q001", 0)
	if err != nil {
		t.Fatalf("resolveNext(ready Question) error = %v", err)
	}
	if got.Action != action.ActionPause || got.Status != "ready_for_resolution" || got.Prompt != "" || got.AgentType != "" {
		t.Fatalf("resolveNext(ready Question) = %#v, want empty pause envelope", got)
	}
	if placeholderCalls != 0 || actionCalls != 0 {
		t.Fatalf("ready Question rendered placeholders=%d actions=%d, want neither", placeholderCalls, actionCalls)
	}
}

// TC-103/TC-104: QuestionService marks every responder-less checkpoint as a
// terminal dispatch signal so keyed next exits before placeholder/action
// rendering. Keep the F01 draft compatibility state in the same parity matrix
// as F02's migrated open/answering and resolution-owner checkpoint.
func TestResolveNextQuestionNoResponderCheckpointsPauseParity_TC103_TC104(t *testing.T) {
	for _, status := range []string{"draft", "open", "answering", "ready_for_resolution"} {
		t.Run(status, func(t *testing.T) {
			placeholderCalls := 0
			actionCalls := 0
			cache := &nextAdapterCache{entries: map[string]*nextAdapters{
				"question": {
					transitioner: nextStatusOnlyTransitioner{next: &services.NextStatusInfo{
						EntityType: models.EntityTypeQuestion, EntityKey: "Q001", CurrentStatus: status, IsTerminal: true,
					}},
					generator: runnerPlaceholderFunc(func(context.Context, string) (map[string]string, error) {
						placeholderCalls++
						return nil, errors.New("Question responder placeholders must not run")
					}),
					actionSvc: &action.MockActionService{GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
						actionCalls++
						return nil, errors.New("Question workflow action must not run")
					}},
				},
			}}

			got, err := resolveNext(context.Background(), cache, "question", "Q001", 0)
			if err != nil {
				t.Fatalf("resolveNext(%s) error = %v", status, err)
			}
			if got.Action != action.ActionPause || got.Status != status || got.AgentType != "" || got.Prompt != "" {
				t.Fatalf("resolveNext(%s) = %#v, want empty pause envelope", status, got)
			}
			if placeholderCalls != 0 || actionCalls != 0 {
				t.Fatalf("Question %s rendered placeholders=%d actions=%d, want neither", status, placeholderCalls, actionCalls)
			}
		})
	}
}

type runnerPlaceholderFunc func(context.Context, string) (map[string]string, error)

func (f runnerPlaceholderFunc) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	return f(ctx, key)
}

type nextStatusOnlyTransitioner struct{ next *services.NextStatusInfo }

func (t nextStatusOnlyTransitioner) TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, errors.New("not called")
}

func (t nextStatusOnlyTransitioner) GetNextStatus(context.Context, string) (*services.NextStatusInfo, error) {
	return t.next, nil
}
