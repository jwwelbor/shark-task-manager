package team

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

const defaultSchedulerCleanupTimeout = 5 * time.Second

// Scheduler executes a confirmed ledger snapshot. It never invokes planning
// or changes the root entity workflow.
type Scheduler struct {
	deps         SchedulerDeps
	now          func() time.Time
	diagnosticMu sync.Mutex
	recordMu     sync.Mutex
}

func NewScheduler(deps SchedulerDeps) (*Scheduler, error) {
	if deps.Ledger == nil {
		return nil, errors.New("team scheduler: ledger is required")
	}
	if deps.Claims == nil {
		return nil, errors.New("team scheduler: claims are required")
	}
	if deps.Resolver == nil {
		return nil, errors.New("team scheduler: dispatch resolver is required")
	}
	if deps.Dispatcher == nil {
		return nil, errors.New("team scheduler: dispatcher is required")
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &Scheduler{deps: deps, now: now}, nil
}

func (s *Scheduler) Start(ctx context.Context, runID int64, rootSessionID string) (*TeamRunResult, error) {
	return s.start(ctx, runID, rootSessionID)
}

// Resume reconciles interrupted claimed/running items before continuing the
// confirmed snapshot. Terminal items remain untouched; a later retry must be
// requested explicitly by the caller through the ledger attempt contract.
func (s *Scheduler) Resume(ctx context.Context, runID int64, rootSessionID string) (*TeamRunResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items, err := s.deps.Ledger.ListItems(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load team run %d items for resume: %w", runID, err)
	}
	for _, item := range items {
		if item == nil || (item.ItemStatus != ItemStatusClaimed && item.ItemStatus != ItemStatusRunning) {
			continue
		}
		claim := stringValue(item.ClaimSessionID)
		worker := stringValue(item.WorkerSessionID)
		cleanupCtx, cleanupCancel := s.cleanupContext(ctx)
		_, releaseErr := s.deps.Claims.Release(cleanupCtx, string(item.ChildType), item.ChildKey, claim, "resume_reconciled", false)
		cleanupCancel()
		if releaseErr != nil {
			return nil, fmt.Errorf("reconcile claim for %s: %w", item.ChildKey, releaseErr)
		}
		cleanupCtx, cleanupCancel = s.cleanupContext(ctx)
		persisted, err := s.deps.Ledger.RecordItemResult(cleanupCtx, ItemResultUpdate{
			RunID: runID, ItemID: item.ID, Attempt: item.Attempt, Status: ItemStatusPaused,
			SkipReason: "resume_reconciled_stale", ClaimSessionID: claim, WorkerSessionID: worker,
			CompletedAt: ptrTime(s.now()),
		})
		cleanupCancel()
		if err != nil {
			return nil, fmt.Errorf("reconcile item %s: %w", item.ChildKey, err)
		}
		if persisted != nil {
			*item = *persisted
		}
	}
	return s.start(ctx, runID, rootSessionID)
}

func (s *Scheduler) start(ctx context.Context, runID int64, rootSessionID string) (*TeamRunResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	run, err := s.deps.Ledger.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load team run %d: %w", runID, err)
	}
	items, err := s.deps.Ledger.ListItems(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("load team run %d items: %w", runID, err)
	}
	if run == nil || run.ID != runID || strings.TrimSpace(rootSessionID) == "" {
		return nil, errors.New("team scheduler: invalid run identity")
	}
	// The caller owns the root lease for this execution. Keep the exact
	// correlation on the confirmed run so lifecycle CAS updates and any
	// pre-claim diagnostics remain tied to that lease, not just heartbeats.
	run.RootSessionID = optionalString(rootSessionID)
	if err := run.Validate(); err != nil {
		return nil, fmt.Errorf("validate team run %d: %w", runID, err)
	}
	if expected := strings.TrimSpace(s.deps.ExpectedPlanHash); expected != "" && expected != run.PlanHash {
		return nil, &PlanDriftError{RootKey: run.RootKey, ExistingHash: run.PlanHash, RequestedHash: expected}
	}
	mode, limit, reason, err := s.selectCapacity(ctx, run, items)
	if err != nil {
		return nil, err
	}
	// The confirmed plan's mode and limit are immutable ledger fields. A
	// conservative resource policy may reduce the capacity used for this
	// execution, but that runtime fallback must not be written back as a plan
	// mutation. Keep the selected values on the in-memory run for the operator
	// result while persisting only lifecycle fields through the ledger.
	confirmedMode, confirmedLimit := run.ExecutionMode, run.ConcurrencyLimit
	rootSession := rootSessionID
	if mode != run.ExecutionMode || limit != run.ConcurrencyLimit || reason != "" {
		next := reason
		if next == "" {
			next = "resource policy selected a different execution mode"
		}
		run.NextAction = &next
	}
	// Persist the actual capacity selected by the resource policy. The result
	// must describe the mode that was really used, even when the confirmed
	// request was degraded before any child claim.
	if _, err := s.deps.Ledger.UpdateRun(ctx, RunUpdate{RunID: run.ID, Status: RunStatusRunning, ExecutionMode: confirmedMode, ConcurrencyLimit: confirmedLimit, PlanHash: run.PlanHash, RootSessionID: &rootSession, StartedAt: run.StartedAt, NextAction: run.NextAction}); err != nil {
		// Lightweight test ledgers may not implement lifecycle transitions yet;
		// execution remains safe because item CAS is still the mutation gate.
		if !errors.Is(err, ErrInvalidRunTransition) {
			return nil, fmt.Errorf("start team run %d: %w", runID, err)
		}
	}
	// Keep the selected runtime capacity in the result while leaving the
	// confirmed snapshot immutable in the durable ledger.
	run.ExecutionMode, run.ConcurrencyLimit = mode, limit
	// The coordinator owns the parent lease. Renew it before any child work and
	// keep renewing it while workers are active; workers never receive this seam.
	if err := s.deps.Claims.Heartbeat(ctx, string(run.RootType), run.RootKey, rootSessionID, nil, "team scheduler active"); err != nil {
		s.setNextAction(run, "root heartbeat failed: "+bounded(err.Error()))
	}
	stopRootHeartbeat := s.startHeartbeat(ctx, run.RootType, run.RootKey, rootSessionID, run)
	defer stopRootHeartbeat()
	s.execute(ctx, run, items, limit)
	stopRootHeartbeat()
	status := RunStatusCompleted
	for _, item := range items {
		if item.ItemStatus == ItemStatusFailed {
			status = RunStatusFailed
		}
		if item.ItemStatus == ItemStatusPlanned || item.ItemStatus == ItemStatusClaimed || item.ItemStatus == ItemStatusRunning {
			status = RunStatusPaused
		}
	}
	run.Status = status
	cleanupCtx, cleanupCancel := s.cleanupContext(ctx)
	_, updateErr := s.deps.Ledger.UpdateRun(cleanupCtx, RunUpdate{RunID: run.ID, Status: status, ExecutionMode: confirmedMode, ConcurrencyLimit: confirmedLimit, PlanHash: run.PlanHash, RootSessionID: &rootSession, NextAction: run.NextAction, CompletedAt: ptrTime(s.now())})
	cleanupCancel()
	if updateErr != nil && !errors.Is(updateErr, ErrInvalidRunTransition) {
		return nil, fmt.Errorf("complete team run %d: %w", runID, updateErr)
	}
	return NewTeamRunResult(run, items)
}

func (s *Scheduler) selectCapacity(ctx context.Context, run *TeamRun, items []*TeamRunItem) (ExecutionMode, int, string, error) {
	mode, limit := run.ExecutionMode, run.ConcurrencyLimit
	if s.deps.Resource == nil {
		return mode, limit, "", nil
	}
	selected, selectedLimit, reason, err := s.deps.Resource.Select(ctx, run, items)
	if err != nil {
		return mode, limit, "", fmt.Errorf("select scheduler capacity: %w", err)
	}
	if selectedLimit <= 0 {
		return ExecutionMode(""), 0, "", errors.New("team scheduler: resource policy returned non-positive limit")
	}
	if !validExecutionMode(selected) {
		return ExecutionMode(""), 0, "", fmt.Errorf("team scheduler: resource policy returned invalid mode %q", selected)
	}
	if selectedLimit > limit {
		selectedLimit = limit
	}
	return selected, selectedLimit, reason, nil
}

func (s *Scheduler) execute(ctx context.Context, run *TeamRun, items []*TeamRunItem, limit int) {
	for {
		ready := s.ready(items)
		if len(ready) == 0 {
			s.blockDependents(ctx, run, items)
			return
		}
		if limit < 1 {
			limit = 1
		}
		for start := 0; start < len(ready); start += limit {
			end := start + limit
			if end > len(ready) {
				end = len(ready)
			}
			var wg sync.WaitGroup
			for _, item := range ready[start:end] {
				wg.Add(1)
				go func(i *TeamRunItem) { defer wg.Done(); s.executeItem(ctx, run, i) }(item)
			}
			wg.Wait()
		}
	}
}

func (s *Scheduler) ready(items []*TeamRunItem) []*TeamRunItem {
	completed := make(map[string]ItemStatus, len(items))
	for _, item := range items {
		completed[item.ChildKey] = item.ItemStatus
	}
	ready := make([]*TeamRunItem, 0)
	lowestWave := int(^uint(0) >> 1)
	for _, item := range items {
		if item.ItemStatus != ItemStatusPlanned || !dependenciesReady(item.DependencyKeys, completed) {
			continue
		}
		if item.Wave < lowestWave {
			lowestWave = item.Wave
		}
		ready = append(ready, item)
	}
	if len(ready) > 0 {
		filtered := ready[:0]
		for _, item := range ready {
			if item.Wave == lowestWave {
				filtered = append(filtered, item)
			}
		}
		ready = filtered
	}
	sort.Slice(ready, func(i, j int) bool {
		if ready[i].Wave != ready[j].Wave {
			return ready[i].Wave < ready[j].Wave
		}
		if ready[i].ExecutionOrder != ready[j].ExecutionOrder {
			return ready[i].ExecutionOrder < ready[j].ExecutionOrder
		}
		return ready[i].ChildKey < ready[j].ChildKey
	})
	return ready
}

func dependenciesReady(keys []string, statuses map[string]ItemStatus) bool {
	for _, key := range keys {
		if statuses[key] != ItemStatusCompleted {
			return false
		}
	}
	return true
}

func (s *Scheduler) executeItem(ctx context.Context, run *TeamRun, item *TeamRunItem) {
	claimSession := fmt.Sprintf("team-%d-item-%d-%d", run.ID, item.ID, item.Attempt+1)
	// Resolve first: pause, terminal, and unresolved steps are not dispatchable
	// and therefore must never acquire a claim.
	step, err := s.deps.Resolver.Resolve(ctx, item.ChildType, item.ChildKey)
	if err != nil || step.GateClassification != dispatch.GateNone || step.Error != "" || len(step.UnresolvedPlaceholders) > 0 || strings.TrimSpace(step.Prompt) == "" {
		reason := "dispatch_step_unresolved"
		evidence := ""
		if err != nil {
			reason = "unresolved_workflow"
			evidence = bounded(err.Error())
		} else {
			evidence = bounded(step.Error)
			if step.GateClassification != "" {
				reason = string(step.GateClassification)
			} else if len(step.UnresolvedPlaceholders) > 0 {
				reason = "unresolved_placeholder"
			}
		}
		if recordErr := s.record(ctx, run, item, ItemStatusSkipped, reason, evidence, ""); recordErr != nil {
			s.setNextAction(run, "pre-claim diagnostic failed for "+item.ChildKey+": "+bounded(recordErr.Error()))
		}
		return
	}
	if err := ctx.Err(); err != nil {
		if recordErr := s.record(ctx, run, item, ItemStatusCancelled, "cancelled", bounded(err.Error()), ""); recordErr != nil {
			s.setNextAction(run, "pre-claim diagnostic failed for "+item.ChildKey+": "+bounded(recordErr.Error()))
		}
		return
	}
	claim, err := s.deps.Claims.Claim(ctx, services.ClaimInput{EntityType: string(item.ChildType), EntityKey: item.ChildKey, ClaimedBy: "team-scheduler", SessionID: claimSession, Force: false})
	if err != nil {
		if recordErr := s.record(ctx, run, item, ItemStatusSkipped, "claim_conflict", err.Error(), ""); recordErr != nil {
			s.setNextAction(run, "pre-claim diagnostic failed for "+item.ChildKey+": "+bounded(recordErr.Error()))
		}
		return
	}
	if claim != nil && claim.SessionID != "" {
		claimSession = claim.SessionID
	}
	workerSession := ""
	defer func() {
		if recovered := recover(); recovered != nil {
			evidence := bounded(fmt.Sprintf("dispatcher panic: %v", recovered))
			if releaseErr := s.release(ctx, item, claimSession, "failed"); releaseErr != nil {
				evidence = bounded(evidence + "; release failed: " + releaseErr.Error())
			}
			if recordErr := s.record(ctx, run, item, ItemStatusFailed, "dispatcher_panic", evidence, claimSession, workerSession); recordErr != nil {
				s.setNextAction(run, "result persistence failed for "+item.ChildKey+": "+bounded(recordErr.Error()))
			}
			return
		}
	}()
	if ok, err := s.deps.Ledger.ClaimItem(ctx, run.ID, item.ID, item.Attempt, claimSession); err != nil || !ok {
		evidence := "item claim CAS lost"
		if err != nil {
			evidence = bounded(err.Error())
		}
		s.releaseAndRecord(ctx, run, item, claimSession, ItemStatusSkipped, "claim_conflict", evidence, "")
		return
	}
	item.ItemStatus = ItemStatusClaimed
	item.ClaimSessionID = optionalString(claimSession)
	if err := ctx.Err(); err != nil {
		s.releaseAndRecord(ctx, run, item, claimSession, ItemStatusCancelled, "cancelled", err.Error(), workerSession)
		return
	}
	workerSession = fmt.Sprintf("worker-%d-%d", item.ID, item.Attempt+1)
	if ok, err := s.deps.Ledger.StartItem(ctx, run.ID, item.ID, item.Attempt, claimSession, workerSession); err != nil || !ok {
		s.releaseAndRecord(ctx, run, item, claimSession, ItemStatusSkipped, "item_start_conflict", "item start CAS lost", workerSession)
		return
	}
	item.ItemStatus = ItemStatusRunning
	item.WorkerSessionID = optionalString(workerSession)
	stopHeartbeat := s.startHeartbeat(ctx, item.ChildType, item.ChildKey, claimSession, run)
	result, dispatchErr := s.deps.Dispatcher.Dispatch(ctx, runner.DispatchInput{Instruction: step.Prompt, EntityKey: item.ChildKey, EntityType: string(item.ChildType), Status: step.Status, AgentType: step.AgentType, Model: step.Model})
	stopHeartbeat()
	status, outcome, evidence := ItemStatusCompleted, "success", ""
	if errors.Is(dispatchErr, context.Canceled) || errors.Is(dispatchErr, context.DeadlineExceeded) {
		status, outcome = ItemStatusCancelled, "cancelled"
		evidence = safeWorkerEvidence(errorText(dispatchErr))
	} else if result != nil && result.ExitCode != 0 {
		status, outcome = ItemStatusFailed, "process_failed"
		evidence = safeWorkerEvidence(fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Stderr))
	} else if dispatchErr != nil {
		status, outcome = ItemStatusFailed, "dispatch_error"
		var missing *runner.ToolNotFoundError
		if errors.As(dispatchErr, &missing) {
			outcome = "provider_not_found"
		}
		evidence = safeWorkerEvidence(errorText(dispatchErr))
	} else if result != nil {
		evidence = safeWorkerEvidence(result.Stdout)
	}
	s.releaseAndRecord(ctx, run, item, claimSession, status, outcome, evidence, workerSession)
	if s.deps.Events != nil {
		duration := time.Duration(0)
		if result != nil {
			duration = result.Duration
		}
		event := SchedulerEvent{RunID: run.ID, RootKey: run.RootKey, ChildKey: item.ChildKey, Wave: item.Wave, ItemStatus: status, Provider: step.Provider, Duration: duration, ClaimSession: claimSession, WorkerSession: workerSession, Outcome: outcome, Communication: s.communicationFor(run, item)}
		s.deps.Events.Emit(ctx, event)
	}
}

func (s *Scheduler) communicationFor(run *TeamRun, item *TeamRunItem) *CouncilCommunication {
	if s.deps.Communication == nil {
		return nil
	}
	copy := *s.deps.Communication
	copy.RootKey, copy.ChildKey = run.RootKey, item.ChildKey
	copy.EvidenceLinks = append([]string(nil), copy.EvidenceLinks...)
	return &copy
}

func (s *Scheduler) release(ctx context.Context, item *TeamRunItem, session, outcome string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("claim release panic: %v", recovered)
		}
	}()
	cleanupCtx, cleanupCancel := s.cleanupContext(ctx)
	defer cleanupCancel()
	released, err := s.deps.Claims.Release(cleanupCtx, string(item.ChildType), item.ChildKey, session, outcome, false)
	if err != nil {
		return err
	}
	if !released {
		return errors.New("claim session was not released")
	}
	return nil
}

func (s *Scheduler) cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := s.deps.CleanupTimeout
	if timeout <= 0 {
		timeout = defaultSchedulerCleanupTimeout
	}
	return context.WithTimeout(context.WithoutCancel(ctx), timeout)
}

func (s *Scheduler) releaseAndRecord(ctx context.Context, run *TeamRun, item *TeamRunItem, session string, status ItemStatus, outcome, evidence, worker string) {
	if err := s.release(ctx, item, session, outcome); err != nil {
		evidence = bounded(evidence + "; release failed: " + err.Error())
		s.setNextAction(run, "claim release failed for "+item.ChildKey)
	}
	if recordErr := s.record(ctx, run, item, status, outcome, evidence, session, worker); recordErr != nil {
		s.setNextAction(run, "result persistence failed for "+item.ChildKey+": "+bounded(recordErr.Error()))
	}
}

func (s *Scheduler) startHeartbeat(ctx context.Context, typ models.EntityType, key, session string, run *TeamRun) func() {
	interval := s.deps.HeartbeatInterval
	if interval <= 0 || strings.TrimSpace(session) == "" {
		return func() {}
	}
	childCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.deps.Claims.Heartbeat(childCtx, string(typ), key, session, nil, "team scheduler active"); err != nil {
					s.setNextAction(run, "heartbeat failed for "+key)
				}
			case <-childCtx.Done():
				return
			}
		}
	}()
	return cancel
}

func (s *Scheduler) setNextAction(run *TeamRun, diagnostic string) {
	s.diagnosticMu.Lock()
	defer s.diagnosticMu.Unlock()
	if run.NextAction == nil || *run.NextAction == "" {
		run.NextAction = &diagnostic
	}
}

func (s *Scheduler) blockDependents(ctx context.Context, run *TeamRun, items []*TeamRunItem) {
	statuses := map[string]ItemStatus{}
	for _, i := range items {
		statuses[i.ChildKey] = i.ItemStatus
	}
	for _, i := range items {
		if i.ItemStatus == ItemStatusPlanned && len(i.DependencyKeys) > 0 {
			if recordErr := s.record(ctx, run, i, ItemStatusBlocked, "dependency_not_satisfied", "prerequisite did not complete successfully", ""); recordErr != nil {
				s.setNextAction(run, "pre-claim diagnostic failed for "+i.ChildKey+": "+bounded(recordErr.Error()))
			}
		}
	}
}
func (s *Scheduler) record(ctx context.Context, run *TeamRun, item *TeamRunItem, status ItemStatus, outcome, evidence, claim string, worker ...string) error {
	s.recordMu.Lock()
	defer s.recordMu.Unlock()
	workerSession := ""
	if len(worker) > 0 {
		workerSession = worker[0]
	}
	skipReason := ""
	if status == ItemStatusBlocked || status == ItemStatusSkipped || status == ItemStatusPaused || status == ItemStatusCancelled {
		skipReason, outcome = outcome, ""
	}
	wasPlanned := item.ItemStatus == ItemStatusPlanned
	// Update the in-memory snapshot before persistence so a failed ledger write
	// cannot make the scheduling loop repeatedly select the same item. The
	// diagnostic below remains the durable truth that the write was rejected.
	item.ItemStatus, item.Outcome, item.SkipReason = status, optionalString(outcome), optionalString(skipReason)
	item.ClaimSessionID, item.WorkerSessionID = optionalString(claim), optionalString(workerSession)
	cleanupCtx, cleanupCancel := s.cleanupContext(ctx)
	defer cleanupCancel()
	update := ItemResultUpdate{RunID: run.ID, ItemID: item.ID, Attempt: item.Attempt, Status: status, Outcome: outcome, SkipReason: skipReason, Evidence: bounded(evidence), ClaimSessionID: claim, WorkerSessionID: workerSession, CompletedAt: ptrTime(s.now())}
	var persisted *TeamRunItem
	var err error
	if wasPlanned {
		// A claim may have been acquired and released before the ledger CAS
		// succeeds. It is still pre-claim from the ledger's perspective.
		update.ClaimSessionID, update.WorkerSessionID = "", ""
		persisted, err = s.deps.Ledger.RecordPreClaimResult(cleanupCtx, update, stringValue(run.RootSessionID))
	} else {
		persisted, err = s.deps.Ledger.RecordItemResult(cleanupCtx, update)
	}
	if err != nil {
		s.setNextAction(run, "result persistence failed for "+item.ChildKey+": "+bounded(err.Error()))
		return err
	}
	if persisted != nil {
		*item = *persisted
	}
	return nil
}
func bounded(value string) string {
	if len(value) > maxBoundedText {
		return value[:maxBoundedText]
	}
	return value
}

func safeWorkerEvidence(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if sensitiveEvidencePattern.MatchString(value) || strings.Contains(lower, "transcript") {
		return "worker output omitted: sensitive content"
	}
	return bounded(value)
}
func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
func ptrTime(t time.Time) *time.Time { return &t }
