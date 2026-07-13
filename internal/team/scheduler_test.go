package team

import (
	"context"
	"errors"
	"fmt"
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
	run   *TeamRun
	items []*TeamRunItem
	mu    sync.Mutex
}

func newSchedulerLedger(run *TeamRun, items ...*TeamRunItem) *schedulerLedger {
	return &schedulerLedger{run: run, items: items}
}
func (l *schedulerLedger) GetRun(context.Context, int64) (*TeamRun, error) { return l.run, nil }
func (l *schedulerLedger) ListItems(context.Context, int64) ([]*TeamRunItem, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]*TeamRunItem, len(l.items))
	copy(out, l.items)
	return out, nil
}
func (l *schedulerLedger) UpdateRun(_ context.Context, u RunUpdate) (*TeamRun, error) {
	l.run.Status, l.run.NextAction = u.Status, u.NextAction
	return l.run, nil
}
func (l *schedulerLedger) PersistConfirmedPlan(context.Context, *TeamPlan, string) (*TeamRun, error) {
	return nil, errors.New("not used")
}
func (l *schedulerLedger) RecordItemResult(_ context.Context, u ItemResultUpdate) (*TeamRunItem, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, i := range l.items {
		if i.ID == u.ItemID {
			i.ItemStatus = u.effectiveStatus()
			i.Outcome = optionalString(u.Outcome)
			i.SkipReason = optionalString(u.SkipReason)
			return i, nil
		}
	}
	return nil, errors.New("missing item")
}
func (l *schedulerLedger) ClaimItem(context.Context, int64, int64, int, string) (bool, error) {
	return true, nil
}
func (l *schedulerLedger) StartItem(context.Context, int64, int64, int, string, string) (bool, error) {
	return true, nil
}

type schedulerClaims struct {
	mu     sync.Mutex
	claims int
	heart  []heartbeatCall
}

func (c *schedulerClaims) Claim(_ context.Context, in services.ClaimInput) (*models.EntityClaim, error) {
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
func (*schedulerClaims) Release(context.Context, string, string, string, string, bool) (bool, error) {
	return true, nil
}

type schedulerResolver struct{ step dispatch.DispatchStep }

func (r schedulerResolver) Resolve(_ context.Context, typ models.EntityType, key string) (dispatch.DispatchStep, error) {
	if r.step.GateClassification != "" || r.step.Error != "" {
		return r.step, nil
	}
	return dispatch.DispatchStep{EntityType: typ, EntityKey: key, Prompt: "work", AgentType: "developer"}, nil
}

type schedulerDispatcher struct {
	mu         sync.Mutex
	started    chan string
	fail       map[string]error
	active     *activeCounter
	dispatched []string
	block      chan struct{}
}

func (d *schedulerDispatcher) Dispatch(_ context.Context, in runner.DispatchInput) (*runner.DispatchResult, error) {
	if d.block != nil {
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
	d.mu.Unlock()
	time.Sleep(time.Millisecond)
	if err != nil {
		return nil, err
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
func testRun(id int64, mode ExecutionMode, limit int) *TeamRun {
	hash := fmt.Sprintf("%064x", id)
	return &TeamRun{ID: id, RootKey: "E38-F02", RootType: models.EntityTypeFeature, Status: RunStatusPlanned, ExecutionMode: mode, ConcurrencyLimit: limit, PlanHash: hash}
}
func item(id int64, key string, wave int) *TeamRunItem {
	return &TeamRunItem{ID: id, TeamRunID: 41, ChildKey: key, ChildType: models.EntityTypeTask, Wave: wave, ItemStatus: ItemStatusPlanned, Attempt: 0}
}
