package team

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

func TestScheduler_BoundsParallelism_TC001(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeParallel, 2),
		item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 0), item(3, "T-E38-F02-003", 0))
	claims := &schedulerClaims{}
	dispatcher := &schedulerDispatcher{active: &activeCounter{}}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionMode != ExecutionModeParallel || result.ConcurrencyLimit != 2 {
		t.Fatalf("mode=%s limit=%d", result.ExecutionMode, result.ConcurrencyLimit)
	}
	if dispatcher.active.max() > 2 {
		t.Fatalf("active dispatches exceeded limit: %d", dispatcher.active.max())
	}
	if len(dispatcher.keys()) != 3 {
		t.Fatalf("dispatched %v", dispatcher.keys())
	}
}

func TestScheduler_RejectsConfirmedPlanDriftBeforeClaim_TC001(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}, ExpectedPlanHash: strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Start(context.Background(), 41, "root")
	if !errors.Is(err, ErrPlanDrift) {
		t.Fatalf("error = %v, want ErrPlanDrift", err)
	}
	if claims.claims != 0 || ledger.items[0].ItemStatus != ItemStatusPlanned {
		t.Fatalf("drift mutated child: claims=%d status=%s", claims.claims, ledger.items[0].ItemStatus)
	}
}

func TestScheduler_GatesDependentWave_TC002(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeParallel, 3),
		item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 0), item(3, "T-E38-F02-003", 1))
	ledger.items[2].DependencyKeys = []string{"T-E38-F02-001", "T-E38-F02-002"}
	dispatcher := &schedulerDispatcher{started: make(chan string, 3)}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root"); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.keys(); len(got) != 3 || got[2] != "T-E38-F02-003" {
		t.Fatalf("dispatch order=%v", got)
	}
}

func TestScheduler_DispatchesOnlyLowestReadyWavePerPartition_TC001(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeParallel, 2),
		item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 1))
	dispatcher := &schedulerDispatcher{started: make(chan string, 2), block: make(chan struct{}), blockAfterStart: true}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, startErr := s.Start(context.Background(), 41, "root")
		done <- startErr
	}()

	select {
	case got := <-dispatcher.started:
		if got != "T-E38-F02-001" {
			t.Fatalf("started later wave %q before lowest ready wave", got)
		}
	case <-time.After(time.Second):
		t.Fatal("lowest ready wave did not start")
	}
	select {
	case got := <-dispatcher.started:
		t.Fatalf("later wave started in the same partition: %q", got)
	case <-time.After(20 * time.Millisecond):
	}

	close(dispatcher.block)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.keys(); len(got) != 2 || got[0] != "T-E38-F02-001" || got[1] != "T-E38-F02-002" {
		t.Fatalf("dispatch order=%v", got)
	}
}

func TestScheduler_ContainsDependencyFailure_TC003(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1),
		item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 1), item(3, "T-E38-F02-003", 0))
	ledger.items[1].DependencyKeys = []string{"T-E38-F02-001"}
	dispatcher := &schedulerDispatcher{fail: map[string]error{"T-E38-F02-001": errors.New("provider unavailable")}}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if ledger.items[1].ItemStatus != ItemStatusBlocked {
		t.Fatalf("dependent status=%s", ledger.items[1].ItemStatus)
	}
	if ledger.items[1].SkipReason == nil || *ledger.items[1].SkipReason != "dependency_not_satisfied" {
		t.Fatalf("dependent reason=%v", ledger.items[1].SkipReason)
	}
	if got := dispatcher.keys(); len(got) != 2 || got[0] != "T-E38-F02-001" || got[1] != "T-E38-F02-003" {
		t.Fatalf("dispatch order=%v", got)
	}
	if result.Status == RunStatusCompleted && ledger.items[1].ItemStatus == ItemStatusCompleted {
		t.Fatal("failed dependency falsely completed")
	}
}

func TestScheduler_UsesResourceSafetyFallback_TC011(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeParallel, 4), item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 0))
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}, Resource: resourcePolicy{mode: ExecutionModeSequential, limit: 1, reason: DegradedReasonUnknownResourceOwnership}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionMode != ExecutionModeSequential || result.ConcurrencyLimit != 1 {
		t.Fatalf("unsafe mode was not reduced: %s/%d", result.ExecutionMode, result.ConcurrencyLimit)
	}
	if result.NextAction == nil || *result.NextAction == "" {
		t.Fatal("fallback reason was not observable")
	}
}

func TestScheduler_SkipsNonDispatchableStepsBeforeClaim_TC008(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	resolver := schedulerResolver{step: dispatch.DispatchStep{GateClassification: dispatch.GatePause, Error: "human approval required"}}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: resolver, Dispatcher: &schedulerDispatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root-session-001"); err != nil {
		t.Fatal(err)
	}
	if claims.claims != 0 {
		t.Fatalf("non-dispatchable step was claimed %d times", claims.claims)
	}
	if ledger.items[0].ItemStatus != ItemStatusSkipped || ledger.items[0].SkipReason == nil || *ledger.items[0].SkipReason != string(dispatch.GatePause) {
		t.Fatalf("status=%s reason=%v", ledger.items[0].ItemStatus, ledger.items[0].SkipReason)
	}
}

func TestScheduler_ReleasesClaimWhenLedgerClaimCASIsLost_TC006(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	ledger.claimItemOK = false
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root-session-001"); err != nil {
		t.Fatal(err)
	}

	releases := claims.releases()
	if len(releases) != 1 {
		t.Fatalf("release calls=%d, want 1", len(releases))
	}
	if releases[0].session != "claim-1" || releases[0].force {
		t.Fatalf("release=%+v, want exact claim session without force", releases[0])
	}
}

func TestScheduler_ClaimConflictIsNonDestructive_TC004A(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{claimErr: errors.New("entity is already claimed")}
	dispatcher := &schedulerDispatcher{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(dispatcher.keys()) != 0 {
		t.Fatalf("claim-conflicted item was dispatched: %v", dispatcher.keys())
	}
	if len(claims.releases()) != 0 {
		t.Fatalf("losing coordinator released a claim it did not own: %v", claims.releases())
	}
	if result.Items[0].SkipReason == nil || *result.Items[0].SkipReason != "claim_conflict" {
		t.Fatalf("claim conflict was not recorded: %+v", result.Items[0])
	}
}

func TestScheduler_MapsDispatcherResultAndError_TC005(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	dispatcher := &schedulerDispatcher{
		result: &runner.DispatchResult{ExitCode: 7, Stderr: "bounded failure"},
		fail:   map[string]error{"T-E38-F02-001": &runner.AgentFailedError{ExitCode: 7, Stderr: "bounded failure"}},
	}
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].ItemStatus != ItemStatusFailed || result.Items[0].Outcome == nil || *result.Items[0].Outcome != "process_failed" {
		t.Fatalf("dispatcher process result was not mapped: %+v", result.Items[0])
	}
	if len(claims.releases()) != 1 || claims.releases()[0].session != "claim-1" || claims.releases()[0].force {
		t.Fatalf("dispatcher result did not use exact non-forced cleanup: %+v", claims.releases())
	}
}

func TestScheduler_MapsProviderNotFoundAndCancellation_TC005(t *testing.T) {
	t.Run("provider_not_found", func(t *testing.T) {
		ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
		dispatcher := &schedulerDispatcher{fail: map[string]error{"T-E38-F02-001": &runner.ToolNotFoundError{Tool: "claude"}}}
		s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
		if err != nil {
			t.Fatal(err)
		}
		result, err := s.Start(context.Background(), 41, "root")
		if err != nil {
			t.Fatal(err)
		}
		if result.Items[0].Outcome == nil || *result.Items[0].Outcome != "provider_not_found" {
			t.Fatalf("outcome=%v", result.Items[0].Outcome)
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Start(ctx, 41, "root"); !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	})
}

func TestScheduler_CoversPrerequisiteFinitePartitions_TC003(t *testing.T) {
	cases := []struct {
		name       string
		dependency string
		want       ItemStatus
	}{
		{name: "external_unsatisfied", dependency: "E99-F01", want: ItemStatusBlocked},
		{name: "unknown_prerequisite", dependency: "T-E38-F02-999", want: ItemStatusBlocked},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 1))
			ledger.items[1].DependencyKeys = []string{tc.dependency}
			s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Start(context.Background(), 41, "root"); err != nil {
				t.Fatal(err)
			}
			if ledger.items[1].ItemStatus != tc.want {
				t.Fatalf("status=%s", ledger.items[1].ItemStatus)
			}
		})
	}
}

func TestScheduler_ReportsResultPersistenceFailureAfterCleanup_TC006(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	ledger.recordErr = errors.New("ledger write failed")
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(claims.releases()) != 1 || claims.releases()[0].session != "claim-1" || claims.releases()[0].force {
		t.Fatalf("persistence failure skipped exact cleanup: %+v", claims.releases())
	}
	if result.NextAction == nil || !strings.Contains(*result.NextAction, "result persistence failed") {
		t.Fatalf("persistence failure was not observable: %+v", result.NextAction)
	}
}

func TestScheduler_CancellationUsesBoundedCleanupContext_TC005_TC006(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	dispatcher := &schedulerDispatcher{cancel: cancel, dispatchErr: context.Canceled}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: dispatcher,
		CleanupTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(ctx, 41, "root-session-001"); err != nil {
		t.Fatal(err)
	}
	if got := claims.releaseContextErrors(); len(got) != 1 || got[0] != nil {
		t.Fatalf("release used canceled context: %v", got)
	}
	if deadlines := claims.releaseDeadlines(); len(deadlines) != 1 || deadlines[0].IsZero() {
		t.Fatalf("release cleanup context was not bounded: %v", deadlines)
	}
	if got := ledger.recordContextErrors(); len(got) == 0 || got[len(got)-1] != nil {
		t.Fatalf("result persistence used canceled context: %v", got)
	}
	if deadlines := ledger.recordDeadlines(); len(deadlines) == 0 || deadlines[len(deadlines)-1].IsZero() {
		t.Fatalf("result cleanup context was not bounded: %v", deadlines)
	}
	if got := ledger.updateContextErrors(); len(got) < 2 || got[len(got)-1] != nil {
		t.Fatalf("final UpdateRun used canceled context: %v", got)
	}
	if deadlines := ledger.updateDeadlines(); len(deadlines) < 2 || deadlines[len(deadlines)-1].IsZero() {
		t.Fatalf("final UpdateRun cleanup context was not bounded: %v", deadlines)
	}
}

func TestScheduler_PanicReleasesExactSessionAndPersistsOutcome_TC006(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{panicOnDispatch: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if releases := claims.releases(); len(releases) != 1 || releases[0].session != "claim-1" || releases[0].force {
		t.Fatalf("panic cleanup release=%v", releases)
	}
	if result.Items[0].ItemStatus != ItemStatusFailed || result.Items[0].Outcome == nil || *result.Items[0].Outcome != "dispatcher_panic" {
		t.Fatalf("panic outcome=%+v", result.Items[0])
	}
}

func TestScheduler_ReleasePanicBecomesCleanupDiagnostic_TC006(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{panicOnRelease: true}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root-session-001")
	if err != nil {
		t.Fatal(err)
	}
	if result.NextAction == nil || !strings.Contains(*result.NextAction, "claim release failed") {
		t.Fatalf("release panic was not diagnosed: %v", result.NextAction)
	}
	if result.Items[0].ItemStatus != ItemStatusCompleted || result.Items[0].Outcome == nil || *result.Items[0].Outcome != "success" {
		t.Fatalf("release panic changed worker outcome: %+v", result.Items[0])
	}
}

func TestScheduler_ZeroHeartbeatIntervalKeepsInitialLeaseRenewal_TC007(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{},
		HeartbeatInterval: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root-session-001"); err != nil {
		t.Fatal(err)
	}
	heartbeats := claims.heartbeats()
	if len(heartbeats) != 1 || heartbeats[0].session != "root-session-001" {
		t.Fatalf("zero heartbeat interval changed initial renewal behavior: %v", heartbeats)
	}
}

func TestScheduler_EventUsesAllowListedFieldsOnly_TC005(t *testing.T) {
	event := SchedulerEvent{RunID: 41, RootKey: "E38-F02", ChildKey: "T-E38-F02-001", Wave: 0, ItemStatus: ItemStatusCompleted, Provider: "test", Outcome: "success"}
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"Instruction", "Prompt", "Stdout", "Stderr", "Command", "Evidence", "Transcript"} {
		if _, ok := fields[forbidden]; ok {
			t.Fatalf("event exposes forbidden field %q: %s", forbidden, payload)
		}
	}
}

func TestScheduler_ConsumesCouncilCommunicationContext_TC004(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	events := &schedulerEvents{}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}, Events: events,
		Communication: &CouncilCommunication{SenderRole: "developer", RecipientRole: "chair", Subject: "implementation handoff", Urgency: "normal"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root"); err != nil {
		t.Fatal(err)
	}
	if len(events.events) != 1 || events.events[0].Communication == nil {
		t.Fatalf("scheduler did not consume council communication context: %+v", events.events)
	}
	communication := events.events[0].Communication
	if communication.RootKey != "E38-F02" || communication.ChildKey != "T-E38-F02-001" || communication.RecipientRole != "chair" {
		t.Fatalf("communication scope/recipient = %+v", communication)
	}
}

func TestScheduler_HeartbeatsRootAndChildSessions_TC007(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	claims := &schedulerClaims{}
	dispatcher := &schedulerDispatcher{block: make(chan struct{})}
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: claims, Resolver: schedulerResolver{}, Dispatcher: dispatcher,
		HeartbeatInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		_, _ = s.Start(context.Background(), 41, "root-session-001")
		close(done)
	}()
	deadline := time.After(time.Second)
	for len(claims.heartbeats()) < 2 {
		select {
		case <-deadline:
			t.Fatal("scheduler did not heartbeat root and child")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	close(dispatcher.block)
	<-done
	seen := claims.heartbeats()
	if !containsHeartbeat(seen, string(models.EntityTypeFeature), "E38-F02", "root-session-001") || !containsHeartbeat(seen, string(models.EntityTypeTask), "T-E38-F02-001", "claim-1") {
		t.Fatalf("heartbeats=%v", seen)
	}
}

func TestScheduler_PersistsProvidedRootSessionForLifecycleUpdates_TC007(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	s, err := NewScheduler(SchedulerDeps{
		Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root-session-001"); err != nil {
		t.Fatal(err)
	}
	if ledger.updatedRootSession == nil || *ledger.updatedRootSession != "root-session-001" {
		t.Fatalf("root session was not persisted through lifecycle updates: %v", ledger.updatedRootSession)
	}
}

func TestScheduler_ResumeIsIdempotent_TC009(t *testing.T) {
	completed := item(1, "T-E38-F02-001", 0)
	completed.ItemStatus = ItemStatusCompleted
	completed.Attempt = 1
	completed.Outcome = stringPtr("success")
	stale := item(2, "T-E38-F02-002", 0)
	stale.ItemStatus = ItemStatusRunning
	stale.Attempt = 1
	stale.ClaimSessionID = stringPtr("claim-stale")
	stale.WorkerSessionID = stringPtr("worker-stale")
	fresh := item(3, "T-E38-F02-003", 0)
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), completed, stale, fresh)
	dispatcher := &schedulerDispatcher{}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resume(context.Background(), 41, "root"); err != nil {
		t.Fatal(err)
	}
	if got := dispatcher.keys(); len(got) != 1 || got[0] != fresh.ChildKey {
		t.Fatalf("resume dispatched %v, want only unfinished item", got)
	}
	if completed.Attempt != 1 || stale.ItemStatus != ItemStatusPaused {
		t.Fatalf("resume changed completed/stale state: completed=%+v stale=%+v", completed, stale)
	}
}

func TestScheduler_RejectsSensitiveEvidence_TC010(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	events := &schedulerEvents{}
	dispatcher := &schedulerDispatcher{result: &runner.DispatchResult{ExitCode: 0, Stdout: "SYSTEM PROMPT: Authorization: Bearer secret-123\nfull transcript"}}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher, Events: events})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].Evidence != "" && strings.Contains(result.Items[0].Evidence, "secret-123") {
		t.Fatal("sensitive worker output was persisted")
	}
	if len(events.events) != 1 || events.events[0].Outcome != "success" {
		t.Fatalf("safe execution event missing: %+v", events.events)
	}
}

func TestScheduler_ReturnsCompleteTeamRunResult_TC012(t *testing.T) {
	items := []*TeamRunItem{item(1, "T-E38-F02-001", 0), item(2, "T-E38-F02-002", 0)}
	items[1].ItemStatus = ItemStatusPaused
	items[1].SkipReason = stringPtr("human_gate")
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), items...)
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID != 41 || result.RootKey == "" || result.PlanHash == "" || len(result.Items) != 2 {
		t.Fatalf("incomplete aggregate result: %+v", result)
	}
	if result.Items[0].ClaimSessionID == nil || result.Items[0].WorkerSessionID == nil || result.Items[1].SkipReason == nil {
		t.Fatalf("missing per-item diagnostics: %+v", result.Items)
	}
	if result.Counts.Total != 2 || result.Counts.ByStatus[string(ItemStatusCompleted)] != 1 || result.Counts.ByStatus[string(ItemStatusPaused)] != 1 {
		t.Fatalf("incomplete result counts: %+v", result.Counts)
	}
}

func TestScheduler_SanitizesSensitiveDispatcherErrors_TC010(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	dispatcher := &schedulerDispatcher{dispatchErr: errors.New("SYSTEM PROMPT: Authorization: Bearer secret-123")}
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: dispatcher})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ledger.lastEvidence, "secret-123") || strings.Contains(ledger.lastEvidence, "SYSTEM PROMPT") {
		t.Fatalf("sensitive dispatcher error was persisted: %q", result.Items[0].Evidence)
	}
}

func TestScheduler_UsesPersistedItemInAggregateResult_TC012(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeSequential, 1), item(1, "T-E38-F02-001", 0))
	ledger.persistedResultEvidence = `{"summary":"safe summary","artifact_refs":["docs/result.md"]}`
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.Start(context.Background(), 41, "root")
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].Evidence != ledger.persistedResultEvidence {
		t.Fatalf("aggregate used pre-persistence evidence %q, want %q", result.Items[0].Evidence, ledger.persistedResultEvidence)
	}
}

func TestScheduler_PersistsResourceFallbackBeforeRunning_TC011(t *testing.T) {
	ledger := newSchedulerLedger(testRun(41, ExecutionModeParallel, 4), item(1, "T-E38-F02-001", 0))
	ledger.enforceImmutableCapacity = true
	s, err := NewScheduler(SchedulerDeps{Ledger: ledger, Claims: &schedulerClaims{}, Resolver: schedulerResolver{}, Dispatcher: &schedulerDispatcher{}, Resource: resourcePolicy{mode: ExecutionModeSequential, limit: 1, reason: DegradedReasonUnknownResourceOwnership}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Start(context.Background(), 41, "root"); err != nil {
		t.Fatal(err)
	}
	if ledger.runningMode != ExecutionModeParallel || ledger.runningLimit != 4 {
		t.Fatalf("running lifecycle changed immutable snapshot to %s/%d", ledger.runningMode, ledger.runningLimit)
	}
}

type heartbeatCall struct{ entityType, key, session string }

func containsHeartbeat(calls []heartbeatCall, entityType, key, session string) bool {
	for _, call := range calls {
		if call.entityType == entityType && call.key == key && call.session == session {
			return true
		}
	}
	return false
}

type schedulerLedger struct {
	run                      *TeamRun
	items                    []*TeamRunItem
	mu                       sync.Mutex
	claimItemOK              bool
	recordErr                error
	persistedResultEvidence  string
	lastEvidence             string
	updatedMode              ExecutionMode
	updatedLimit             int
	updatedRootSession       *string
	runningMode              ExecutionMode
	runningLimit             int
	enforceImmutableCapacity bool
	originalMode             ExecutionMode
	originalLimit            int
	recordCtxErr             []error
	updateCtxErr             []error
	recordDeadline           []time.Time
	updateDeadline           []time.Time
}

func newSchedulerLedger(run *TeamRun, items ...*TeamRunItem) *schedulerLedger {
	return &schedulerLedger{run: run, items: items, claimItemOK: true, originalMode: run.ExecutionMode, originalLimit: run.ConcurrencyLimit}
}
func (l *schedulerLedger) GetRun(context.Context, int64) (*TeamRun, error) { return l.run, nil }
func (l *schedulerLedger) ListItems(context.Context, int64) ([]*TeamRunItem, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]*TeamRunItem, len(l.items))
	copy(out, l.items)
	return out, nil
}
func (l *schedulerLedger) UpdateRun(ctx context.Context, u RunUpdate) (*TeamRun, error) {
	l.mu.Lock()
	l.updateCtxErr = append(l.updateCtxErr, ctx.Err())
	deadline, _ := ctx.Deadline()
	l.updateDeadline = append(l.updateDeadline, deadline)
	l.mu.Unlock()
	if l.enforceImmutableCapacity && (u.ExecutionMode != l.originalMode || u.ConcurrencyLimit != l.originalLimit) {
		return nil, ErrImmutablePlanSnapshot
	}
	l.run.Status, l.run.NextAction = u.Status, u.NextAction
	l.updatedMode, l.updatedLimit = u.ExecutionMode, u.ConcurrencyLimit
	l.updatedRootSession = u.RootSessionID
	if u.Status == RunStatusRunning {
		l.runningMode, l.runningLimit = u.ExecutionMode, u.ConcurrencyLimit
	}
	return l.run, nil
}
func (l *schedulerLedger) PersistConfirmedPlan(context.Context, *TeamPlan, string) (*TeamRun, error) {
	return nil, errors.New("not used")
}
func (l *schedulerLedger) RecordItemResult(ctx context.Context, u ItemResultUpdate) (*TeamRunItem, error) {
	l.mu.Lock()
	l.recordCtxErr = append(l.recordCtxErr, ctx.Err())
	deadline, _ := ctx.Deadline()
	l.recordDeadline = append(l.recordDeadline, deadline)
	l.mu.Unlock()
	if l.recordErr != nil {
		return nil, l.recordErr
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, i := range l.items {
		if i.ID == u.ItemID {
			l.lastEvidence = u.Evidence
			copy := *i
			copy.ItemStatus = u.effectiveStatus()
			copy.Outcome = optionalString(u.Outcome)
			copy.SkipReason = optionalString(u.SkipReason)
			if l.persistedResultEvidence != "" {
				copy.Evidence = l.persistedResultEvidence
			}
			return &copy, nil
		}
	}
	return nil, errors.New("missing item")
}
func (l *schedulerLedger) RecordPreClaimResult(_ context.Context, u ItemResultUpdate, _ string) (*TeamRunItem, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, i := range l.items {
		if i.ID != u.ItemID || i.ItemStatus != ItemStatusPlanned || i.Attempt != u.Attempt {
			continue
		}
		copy := *i
		copy.ItemStatus = u.effectiveStatus()
		copy.Outcome = optionalString(u.Outcome)
		copy.SkipReason = optionalString(u.SkipReason)
		return &copy, nil
	}
	return nil, errors.New("pre-claim CAS lost")
}
func (l *schedulerLedger) recordContextErrors() []error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]error(nil), l.recordCtxErr...)
}
func (l *schedulerLedger) updateContextErrors() []error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]error(nil), l.updateCtxErr...)
}
func (l *schedulerLedger) recordDeadlines() []time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]time.Time(nil), l.recordDeadline...)
}
func (l *schedulerLedger) updateDeadlines() []time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]time.Time(nil), l.updateDeadline...)
}
func (l *schedulerLedger) ClaimItem(context.Context, int64, int64, int, string) (bool, error) {
	return l.claimItemOK, nil
}
func (l *schedulerLedger) StartItem(context.Context, int64, int64, int, string, string) (bool, error) {
	return true, nil
}

type schedulerClaims struct {
	mu              sync.Mutex
	claims          int
	heart           []heartbeatCall
	releaseCalls    []releaseCall
	claimErr        error
	releaseCtxErr   []error
	releaseDeadline []time.Time
	panicOnRelease  bool
}

type releaseCall struct {
	session string
	force   bool
}

func (c *schedulerClaims) Claim(_ context.Context, in services.ClaimInput) (*models.EntityClaim, error) {
	if c.claimErr != nil {
		return nil, c.claimErr
	}
	c.mu.Lock()
	c.claims++
	c.mu.Unlock()
	return &models.EntityClaim{EntityType: in.EntityType, EntityKey: in.EntityKey, SessionID: "claim-1"}, nil
}
func (c *schedulerClaims) Heartbeat(_ context.Context, entityType, entityKey, sessionID string, _ *float64, _ string) error {
	c.mu.Lock()
	c.heart = append(c.heart, heartbeatCall{entityType, entityKey, sessionID})
	c.mu.Unlock()
	return nil
}
func (c *schedulerClaims) heartbeats() []heartbeatCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]heartbeatCall(nil), c.heart...)
}
func (c *schedulerClaims) Release(ctx context.Context, _ string, _ string, session string, _ string, force bool) (bool, error) {
	if c.panicOnRelease {
		panic("test release panic")
	}
	c.mu.Lock()
	c.releaseCalls = append(c.releaseCalls, releaseCall{session: session, force: force})
	c.releaseCtxErr = append(c.releaseCtxErr, ctx.Err())
	deadline, _ := ctx.Deadline()
	c.releaseDeadline = append(c.releaseDeadline, deadline)
	c.mu.Unlock()
	return true, nil
}
func (c *schedulerClaims) releaseContextErrors() []error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]error(nil), c.releaseCtxErr...)
}
func (c *schedulerClaims) releaseDeadlines() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.releaseDeadline...)
}
func (c *schedulerClaims) releases() []releaseCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]releaseCall(nil), c.releaseCalls...)
}

type schedulerResolver struct{ step dispatch.DispatchStep }

func (r schedulerResolver) Resolve(_ context.Context, typ models.EntityType, key string) (dispatch.DispatchStep, error) {
	if r.step.GateClassification != "" || r.step.Error != "" {
		return r.step, nil
	}
	return dispatch.DispatchStep{EntityType: typ, EntityKey: key, Prompt: "work", AgentType: "developer"}, nil
}

type schedulerDispatcher struct {
	mu              sync.Mutex
	started         chan string
	fail            map[string]error
	active          *activeCounter
	dispatched      []string
	block           chan struct{}
	blockAfterStart bool
	result          *runner.DispatchResult
	dispatchErr     error
	cancel          context.CancelFunc
	panicOnDispatch bool
}

func (d *schedulerDispatcher) Dispatch(_ context.Context, in runner.DispatchInput) (*runner.DispatchResult, error) {
	if d.cancel != nil {
		d.cancel()
	}
	if d.panicOnDispatch {
		panic("test dispatcher panic")
	}
	if d.block != nil && !d.blockAfterStart {
		<-d.block
	}
	if d.active != nil {
		d.active.enter()
		defer d.active.leave()
	}
	d.mu.Lock()
	d.dispatched = append(d.dispatched, in.EntityKey)
	if d.started != nil {
		d.started <- in.EntityKey
	}
	err := d.fail[in.EntityKey]
	if d.dispatchErr != nil {
		err = d.dispatchErr
	}
	d.mu.Unlock()
	if d.block != nil && d.blockAfterStart {
		<-d.block
	}
	time.Sleep(time.Millisecond)
	if err != nil {
		if d.result != nil {
			return d.result, err
		}
		return nil, err
	}
	if d.result != nil {
		return d.result, nil
	}
	return &runner.DispatchResult{ExitCode: 0}, nil
}
func (d *schedulerDispatcher) Name() string                                      { return "test" }
func (d *schedulerDispatcher) BuildCommand(runner.DispatchInput) (string, error) { return "test", nil }
func (d *schedulerDispatcher) keys() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.dispatched...)
}

type activeCounter struct {
	mu               sync.Mutex
	current, highest int
}

func (c *activeCounter) enter() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current++
	if c.current > c.highest {
		c.highest = c.current
	}
}
func (c *activeCounter) leave()   { c.mu.Lock(); defer c.mu.Unlock(); c.current-- }
func (c *activeCounter) max() int { c.mu.Lock(); defer c.mu.Unlock(); return c.highest }

type resourcePolicy struct {
	mode   ExecutionMode
	limit  int
	reason string
}

func (p resourcePolicy) Select(context.Context, *TeamRun, []*TeamRunItem) (ExecutionMode, int, string, error) {
	return p.mode, p.limit, p.reason, nil
}

type schedulerEvents struct{ events []SchedulerEvent }

func (e *schedulerEvents) Emit(_ context.Context, event SchedulerEvent) {
	e.events = append(e.events, event)
}
func testRun(id int64, mode ExecutionMode, limit int) *TeamRun {
	hash := fmt.Sprintf("%064x", id)
	return &TeamRun{ID: id, RootKey: "E38-F02", RootType: models.EntityTypeFeature, Status: RunStatusPlanned, ExecutionMode: mode, ConcurrencyLimit: limit, PlanHash: hash}
}
func item(id int64, key string, wave int) *TeamRunItem {
	return &TeamRunItem{ID: id, TeamRunID: 41, ChildKey: key, ChildType: models.EntityTypeTask, Wave: wave, ItemStatus: ItemStatusPlanned, Attempt: 0}
}
