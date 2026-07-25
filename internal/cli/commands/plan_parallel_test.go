package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stubPortfolioAdvisor struct {
	advice *models.PortfolioAdviceEnvelope
	err    error
	calls  int
}

func (s *stubPortfolioAdvisor) Advise(context.Context) (*models.PortfolioAdviceEnvelope, error) {
	s.calls++
	return s.advice, s.err
}

type keyedPlanTransitioner struct {
	infos       map[string]*services.NextStatusInfo
	errors      map[string]error
	transitions map[string]string
}

func (t *keyedPlanTransitioner) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	if err := t.errors[key]; err != nil {
		return nil, err
	}
	info := t.infos[key]
	if target := t.transitions[key]; target != "" {
		copy := *info
		copy.CurrentStatus = target
		copy.AvailableTransitions = nil
		return &copy, nil
	}
	return info, nil
}

func (t *keyedPlanTransitioner) TransitionStatus(
	_ context.Context,
	key, targetStatus string,
	_ services.TransitionOptions,
) (*services.TransitionResult, error) {
	if t.transitions == nil {
		t.transitions = make(map[string]string)
	}
	t.transitions[key] = targetStatus
	return &services.TransitionResult{ToStatus: targetStatus}, nil
}

func TestResolvePlanDispatchSelectsOneDirectTaskTierWithoutRecursing(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	defer func() {
		planDescribeDispatchableChildren = originalDescribe
	}()

	orderOne, orderTwo := 1, 2
	priority := 5
	children := []services.PlanHierarchyChild{
		{
			Key: "T-E01-F01-001", Title: "First", Status: "ready",
			EntityType: models.EntityTypeTask, ExecutionOrder: &orderOne, Priority: &priority,
		},
		{
			Key: "T-E01-F01-002", Title: "Parallel first", Status: "ready",
			EntityType: models.EntityTypeTask, ExecutionOrder: &orderOne, Priority: &priority,
		},
		{
			Key: "T-E01-F01-003", Title: "Capped parallel item", Status: "ready",
			EntityType: models.EntityTypeTask, ExecutionOrder: &orderOne, Priority: &priority,
		},
		{
			Key: "T-E01-F01-004", Title: "Later", Status: "ready",
			EntityType: models.EntityTypeTask, ExecutionOrder: &orderTwo, Priority: &priority,
		},
	}
	planDescribeDispatchableChildren = func(context.Context, string, string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children:            children,
			TotalChildren:       len(children),
			NonTerminalChildren: len(children),
		}, nil
	}

	parentTransitioner := &keyedPlanTransitioner{infos: map[string]*services.NextStatusInfo{
		"E01-F01": {CurrentStatus: "active"},
	}}
	parentActions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate children"}, nil
		},
	}
	cache := &planAdapterCache{
		nextAdapterCache: &nextAdapterCache{
			entries: map[string]*nextAdapters{
				"feature": {
					transitioner: parentTransitioner,
					generator:    fixedNextPlaceholders{vars: map[string]string{}},
					actionSvc:    parentActions,
				},
			},
			actionSvcRoot: parentActions,
		},
		maxParallelItems: 2,
	}

	resp, err := resolvePlanDispatch(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	require.NotNil(t, resp.selection)
	if resp.selection.Action != "parallel_candidates" {
		t.Fatalf("selection = %#v, want direct-task parallel candidates", resp.selection)
	}
	if got := hierarchyPlanCandidateKeys(resp.selection.Entities); !reflect.DeepEqual(got, []string{
		"T-E01-F01-001", "T-E01-F01-002",
	}) {
		t.Fatalf("candidate keys = %#v, want first execution-order tier only", got)
	}
}

func TestResolvePlanDispatchSingletonUsesHierarchySelectionContract(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	defer func() {
		planDescribeDispatchableChildren = originalDescribe
	}()

	orderOne, orderTwo := 1, 2
	planDescribeDispatchableChildren = func(context.Context, string, string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				{
					Key: "T-E01-F01-001", Title: "First", Status: "ready",
					EntityType: models.EntityTypeTask, ExecutionOrder: &orderOne,
				},
				{
					Key: "T-E01-F01-002", Title: "Later", Status: "ready",
					EntityType: models.EntityTypeTask, ExecutionOrder: &orderTwo,
				},
			},
			TotalChildren:       2,
			NonTerminalChildren: 2,
		}, nil
	}

	cache := &planAdapterCache{nextAdapterCache: planParallelTestCache(map[string]string{"E01-F01": "active"})}
	resp, err := resolvePlanDispatch(context.Background(), cache, "feature", "E01-F01", 0)
	require.NoError(t, err)
	if resp.selection == nil {
		t.Fatal("singleton child did not use hierarchy selection")
	}

	out, err := json.MarshalIndent(resp.selection, "", "  ")
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &fields))
	wantFields := []string{
		"action", "entity", "mode", "root_key", "root_type", "selection_reason",
	}
	for _, field := range wantFields {
		if _, ok := fields[field]; !ok {
			t.Fatalf("selection JSON omits %q\n%s", field, out)
		}
	}
}

func TestResolvePlanDispatchDoesNotReadGrandchildrenOrChildWorkflow(t *testing.T) {
	originalDescribe := planDescribeDispatchableChildren
	defer func() {
		planDescribeDispatchableChildren = originalDescribe
	}()

	order := 1
	planDescribeDispatchableChildren = func(context.Context, string, string) (services.PlanHierarchyChildrenState, error) {
		return services.PlanHierarchyChildrenState{
			Children: []services.PlanHierarchyChild{
				{
					Key: "E07-F01", Title: "Direct feature", Status: "active",
					EntityType: models.EntityTypeFeature, ExecutionOrder: &order,
				},
			},
			TotalChildren:       1,
			NonTerminalChildren: 1,
		}, nil
	}

	parentActions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
		},
	}
	cache := &planAdapterCache{nextAdapterCache: &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"epic": {
				transitioner: &keyedPlanTransitioner{infos: map[string]*services.NextStatusInfo{
					"E07": {CurrentStatus: "active"},
				}},
				generator: fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc: parentActions,
			},
			"feature": {
				transitioner: &keyedPlanTransitioner{errors: map[string]error{
					"E07-F01": fmt.Errorf("child workflow must not be read"),
				}},
				generator: fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc: parentActions,
			},
		},
		actionSvcRoot: parentActions,
	}}

	resp, err := resolvePlanDispatch(context.Background(), cache, "epic", "E07", 0)
	require.NoError(t, err)
	if resp.selection == nil || resp.selection.Entity == nil ||
		resp.selection.Entity.EntityKey != "E07-F01" {
		t.Fatalf("selection = %#v, want direct feature without child resolution", resp.selection)
	}
}

func TestHierarchyPlanSelectionCapsParallelCandidates(t *testing.T) {
	priority := 2
	children := []services.PlanHierarchyChild{
		{Key: "T-E01-F01-001", EntityType: models.EntityTypeTask, Priority: &priority},
		{Key: "T-E01-F01-002", EntityType: models.EntityTypeTask, Priority: &priority},
		{Key: "T-E01-F01-003", EntityType: models.EntityTypeTask, Priority: &priority},
	}

	selection := buildHierarchyPlanSelection("E01-F01", "feature", children, "parallel_tie", 2)
	if selection.Action != "parallel_candidates" {
		t.Fatalf("action = %q, want parallel_candidates", selection.Action)
	}
	if got := hierarchyPlanCandidateKeys(selection.Entities); !reflect.DeepEqual(got, []string{
		"T-E01-F01-001", "T-E01-F01-002",
	}) {
		t.Fatalf("candidate keys = %#v, want first two candidates", got)
	}
}

type stubPortfolioPlanner struct {
	plan services.PortfolioPlan
}

func (s stubPortfolioPlanner) Plan(*models.PortfolioAdviceEnvelope) services.PortfolioPlan {
	return s.plan
}

func TestRunPlanBareEmitsEpicOnlyParallelCandidatesWithBoundedTelemetry(t *testing.T) {
	originalAdvisor := planGetPortfolioAdvisor
	originalPlanner := planGetPortfolioPlanner
	originalAdapters := nextNewAdapterCache
	originalMaxParallelItems := planGetMaxParallelItems
	originalTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		planGetPortfolioAdvisor = originalAdvisor
		planGetPortfolioPlanner = originalPlanner
		nextNewAdapterCache = originalAdapters
		planGetMaxParallelItems = originalMaxParallelItems
		_ = tracerProvider.Shutdown(context.Background())
		otel.SetTracerProvider(originalTracerProvider)
	}()

	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Epics = []models.PortfolioEpicEvidence{
		{Key: "E01", Title: "First", Status: "active", Priority: "high"},
		{Key: "E02", Title: "Second", Status: "ready", Priority: "high"},
		{Key: "E03", Title: "Third", Status: "ready", Priority: "high"},
	}
	planGetPortfolioAdvisor = func() portfolioAdvisor {
		return &stubPortfolioAdvisor{advice: advice}
	}
	planGetPortfolioPlanner = func() portfolioPlanner {
		return stubPortfolioPlanner{plan: services.PortfolioPlan{
			RootKeys: []string{"E01", "E02", "E03"}, SelectionReason: "parallel_tie",
		}}
	}
	planGetMaxParallelItems = func() int { return 2 }
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next descended into keyed dispatch")
		return nil, nil
	}

	var runErr error
	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	capturingOutput(func() {
		runErr = runPlan(cmd, []string{})
	})
	require.NoError(t, runErr)

	var selection PortfolioPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &selection))
	if selection.Action != "parallel_candidates" || selection.Mode != "portfolio_selection" {
		t.Fatalf("selection = %#v", selection)
	}
	if got := portfolioPlanCandidateKeys(selection.Entities); !reflect.DeepEqual(got, []string{"E01", "E02"}) {
		t.Fatalf("candidate keys = %#v", got)
	}
	for _, candidate := range selection.Entities {
		if candidate.EntityType != "epic" {
			t.Fatalf("candidate = %#v, want epic only", candidate)
		}
	}

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "shark.plan", spans[0].Name)
	gotAttributes := make(map[string]any, len(spans[0].Attributes))
	for _, attr := range spans[0].Attributes {
		gotAttributes[string(attr.Key)] = attr.Value.AsInterface()
		for _, forbidden := range []string{"E01", "E02", "E03", "First", "Second", "Third"} {
			if fmt.Sprint(attr.Value.AsInterface()) == forbidden {
				t.Fatalf("telemetry leaked payload through %s", attr.Key)
			}
		}
	}
	wantAttributes := map[string]any{
		"mode":                        "portfolio_selection",
		"action":                      "parallel_candidates",
		"portfolio.candidate_count":   int64(2),
		"portfolio.warning_count":     int64(0),
		"portfolio.evidence_complete": true,
	}
	if !reflect.DeepEqual(gotAttributes, wantAttributes) {
		t.Fatalf("attributes = %#v, want bounded attributes %#v", gotAttributes, wantAttributes)
	}
}

func TestRunPlanBareReturnsOneEpicWithoutResolvingItsWorkflow(t *testing.T) {
	originalAdvisor := planGetPortfolioAdvisor
	originalPlanner := planGetPortfolioPlanner
	originalAdapters := nextNewAdapterCache
	defer func() {
		planGetPortfolioAdvisor = originalAdvisor
		planGetPortfolioPlanner = originalPlanner
		nextNewAdapterCache = originalAdapters
	}()

	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Epics = []models.PortfolioEpicEvidence{{
		Key: "E01", Title: "First", Status: "active", Priority: "high",
	}}
	planGetPortfolioAdvisor = func() portfolioAdvisor {
		return &stubPortfolioAdvisor{advice: advice}
	}
	planGetPortfolioPlanner = func() portfolioPlanner {
		return stubPortfolioPlanner{plan: services.PortfolioPlan{
			RootKeys: []string{"E01"}, SelectionReason: "priority",
		}}
	}
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next descended into keyed dispatch")
		return nil, nil
	}

	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	var runErr error
	capturingOutput(func() {
		runErr = runPlan(cmd, []string{})
	})
	require.NoError(t, runErr)

	var got PortfolioPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &got))
	if got.Action != "select_epic" || got.Entity == nil || got.Entity.EntityKey != "E01" {
		t.Fatalf("selection = %#v, want E01 epic selection", got)
	}
}

func TestPortfolioPlanSelectionCapsParallelCandidates(t *testing.T) {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.Epics = []models.PortfolioEpicEvidence{
		{Key: "E01"}, {Key: "E02"}, {Key: "E03"},
	}
	plan := services.PortfolioPlan{
		RootKeys:        []string{"E01", "E02", "E03"},
		SelectionReason: "parallel_tie",
	}

	selection := buildPortfolioPlanSelection(advice, plan, 2)
	if selection.Action != "parallel_candidates" {
		t.Fatalf("action = %q, want parallel_candidates", selection.Action)
	}
	if got := portfolioPlanCandidateKeys(selection.Entities); !reflect.DeepEqual(got, []string{"E01", "E02"}) {
		t.Fatalf("candidate keys = %#v, want first two candidates", got)
	}
}

func TestRunPlanBareUnsafeEvidenceReturnsPauseWithoutKeyedResolution(t *testing.T) {
	originalAdvisor := planGetPortfolioAdvisor
	originalPlanner := planGetPortfolioPlanner
	originalAdapters := nextNewAdapterCache
	defer func() {
		planGetPortfolioAdvisor = originalAdvisor
		planGetPortfolioPlanner = originalPlanner
		nextNewAdapterCache = originalAdapters
	}()

	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	planGetPortfolioAdvisor = func() portfolioAdvisor {
		return &stubPortfolioAdvisor{advice: advice}
	}
	planGetPortfolioPlanner = func() portfolioPlanner {
		return stubPortfolioPlanner{plan: services.PortfolioPlan{
			PauseReason: "portfolio_ordering_unavailable",
		}}
	}
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next descended into keyed dispatch")
		return nil, nil
	}

	cmd := newPlanCommand()
	cmd.SetContext(context.Background())
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	var runErr error
	capturingOutput(func() {
		runErr = runPlan(cmd, []string{})
	})
	require.NoError(t, runErr)
	var got PortfolioPlanSelectionResponse
	require.NoError(t, json.Unmarshal([]byte(stdout.String()), &got))
	if got.Action != "pause" || got.Reason != "portfolio_ordering_unavailable" {
		t.Fatalf("selection = %#v, want explicit ordering pause", got)
	}
}

type stubStandalonePlanner struct {
	plan services.StandalonePlan
	err  error
}

func (s stubStandalonePlanner) Plan(context.Context, services.StandalonePlanCollection) (services.StandalonePlan, error) {
	return s.plan, s.err
}

func TestRunPlanStandaloneCollectionsReturnSingletonOrParallelTier(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		entityType models.EntityType
		layer      []services.StandalonePlanCandidate
		statuses   map[string]string
		wantAction string
		wantKeys   []string
	}{
		{
			name: "bug singleton", arg: "bugs", entityType: models.EntityTypeBug,
			layer:    []services.StandalonePlanCandidate{{Key: "B001", EntityType: models.EntityTypeBug}},
			statuses: map[string]string{"B001": "ready_bug"}, wantAction: "spawn_agent", wantKeys: []string{"B001"},
		},
		{
			name: "change-card priority tie", arg: "change-cards", entityType: models.EntityTypeChange,
			layer: []services.StandalonePlanCandidate{
				{Key: "CC-001", EntityType: models.EntityTypeChange},
				{Key: "CC-002", EntityType: models.EntityTypeChange},
			},
			statuses:   map[string]string{"CC-001": "ready_one", "CC-002": "ready_two"},
			wantAction: "parallel_dispatch", wantKeys: []string{"CC-001", "CC-002"},
		},
		{
			name: "tech-debt severity tie", arg: "tech-debt", entityType: models.EntityTypeTechDebt,
			layer: []services.StandalonePlanCandidate{
				{Key: "TD-001", EntityType: models.EntityTypeTechDebt},
				{Key: "TD-002", EntityType: models.EntityTypeTechDebt},
			},
			statuses:   map[string]string{"TD-001": "ready_one", "TD-002": "ready_two"},
			wantAction: "parallel_dispatch", wantKeys: []string{"TD-001", "TD-002"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalPlanner := planGetStandalonePlanner
			originalAdapters := nextNewAdapterCache
			defer func() {
				planGetStandalonePlanner = originalPlanner
				nextNewAdapterCache = originalAdapters
			}()

			planGetStandalonePlanner = func() standalonePlanner {
				return stubStandalonePlanner{plan: services.StandalonePlan{Layers: [][]services.StandalonePlanCandidate{tt.layer}}}
			}
			cache := planDirectRootTestCache(string(tt.entityType), tt.statuses)
			nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) { return cache, nil }

			var runErr error
			cmd := newPlanCommand()
			cmd.SetContext(context.Background())
			stdout := capturingOutput(func() {
				runErr = runPlan(cmd, []string{tt.arg})
			})
			require.NoError(t, runErr)

			var wire struct {
				Action    string         `json:"action"`
				EntityKey string         `json:"entity_key"`
				Entities  []NextResponse `json:"entities"`
			}
			require.NoError(t, json.Unmarshal([]byte(stdout), &wire))
			if wire.Action != tt.wantAction {
				t.Fatalf("action = %q, want %q; stdout=%s", wire.Action, tt.wantAction, stdout)
			}
			gotKeys := responseKeys(wire.Entities)
			if wire.EntityKey != "" {
				gotKeys = []string{wire.EntityKey}
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Fatalf("keys = %#v, want %#v", gotKeys, tt.wantKeys)
			}
		})
	}
}

func TestResolvePlanStandaloneLayerCapsWorkBeforePromptResolution(t *testing.T) {
	layer := []services.StandalonePlanCandidate{
		{Key: "B001", EntityType: models.EntityTypeBug},
		{Key: "B002", EntityType: models.EntityTypeBug},
		{Key: "B003", EntityType: models.EntityTypeBug},
	}
	cache := planDirectRootTestCache("bug", map[string]string{
		"B001": "ready_one",
		"B002": "ready_two",
	})

	entities, err := resolvePlanStandaloneLayer(context.Background(), cache, layer, 2)
	require.NoError(t, err)
	if got := responseKeys(entities); !reflect.DeepEqual(got, []string{"B001", "B002"}) {
		t.Fatalf("resolved keys = %#v, want first two candidates", got)
	}
}

func TestResolvePlanStandaloneLayerFillsCapAfterPausedCandidate(t *testing.T) {
	layer := []services.StandalonePlanCandidate{
		{Key: "B001", EntityType: models.EntityTypeBug},
		{Key: "B002", EntityType: models.EntityTypeBug},
		{Key: "B003", EntityType: models.EntityTypeBug},
	}
	actions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "paused" {
				return &action.PopulatedAction{Action: "pause", Instruction: "wait"}, nil
			}
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "test-worker", Instruction: "work",
			}, nil
		},
	}
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"bug": {
				transitioner: &keyedPlanTransitioner{infos: map[string]*services.NextStatusInfo{
					"B001": {CurrentStatus: "paused"},
					"B002": {CurrentStatus: "ready"},
				}},
				generator: fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc: actions,
			},
		},
		actionSvcRoot: actions,
	}

	entities, err := resolvePlanStandaloneLayer(context.Background(), cache, layer, 1)
	require.NoError(t, err)
	if got := responseKeys(entities); !reflect.DeepEqual(got, []string{"B002"}) {
		t.Fatalf("resolved keys = %#v, want next dispatchable candidate", got)
	}
}

func planParallelTestCache(statuses map[string]string) *nextAdapterCache {
	parentActions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "active" {
				return &action.PopulatedAction{Action: "cascade", Instruction: "delegate"}, nil
			}
			return nil, nil
		},
	}
	taskActions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			if status == "ready" {
				return &action.PopulatedAction{
					Action: "spawn_agent", AgentType: "test-worker", Provider: "openai",
					Model: "codex", Instruction: "prompt for ready",
				}, nil
			}
			return &action.PopulatedAction{Action: "pause", Instruction: "wait"}, nil
		},
	}
	parentInfos := map[string]*services.NextStatusInfo{
		"E01-F01": {CurrentStatus: statuses["E01-F01"]},
	}
	taskInfos := make(map[string]*services.NextStatusInfo)
	for key, status := range statuses {
		if key != "E01-F01" {
			taskInfos[key] = &services.NextStatusInfo{CurrentStatus: status}
		}
	}
	return &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: &keyedPlanTransitioner{infos: parentInfos},
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    parentActions,
			},
			"task": {
				transitioner: &keyedPlanTransitioner{infos: taskInfos},
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    taskActions,
			},
		},
		actionSvcRoot: parentActions,
	}
}

func planDirectRootTestCache(entityType string, statuses map[string]string) *nextAdapterCache {
	infos := make(map[string]*services.NextStatusInfo, len(statuses))
	for key, status := range statuses {
		infos[key] = &services.NextStatusInfo{CurrentStatus: status}
	}
	actions := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(_ context.Context, status string, _ map[string]string) (*action.PopulatedAction, error) {
			return &action.PopulatedAction{
				Action: "spawn_agent", AgentType: "test-worker", Provider: "openai",
				Model: "codex", Instruction: "prompt for " + status,
			}, nil
		},
	}
	return &nextAdapterCache{
		entries: map[string]*nextAdapters{
			entityType: {
				transitioner: &keyedPlanTransitioner{infos: infos},
				generator:    fixedNextPlaceholders{vars: map[string]string{}},
				actionSvc:    actions,
			},
		},
		actionSvcRoot: actions,
	}
}

func responseKeys(responses []NextResponse) []string {
	keys := make([]string, 0, len(responses))
	for _, response := range responses {
		keys = append(keys, response.EntityKey)
	}
	return keys
}

func portfolioPlanCandidateKeys(candidates []PortfolioPlanCandidate) []string {
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		keys = append(keys, candidate.EntityKey)
	}
	return keys
}

func hierarchyPlanCandidateKeys(candidates []HierarchyPlanCandidate) []string {
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		keys = append(keys, candidate.EntityKey)
	}
	return keys
}
