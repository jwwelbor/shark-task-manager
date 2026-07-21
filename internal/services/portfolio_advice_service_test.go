package services

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

const wantPortfolioAdvicePrompt = "Inspect the relevant artifacts that exist under docs/product/, especially docs/product/progress.md and docs/product/cross-epic-integration-map.md.\n" +
	"Treat this envelope's state, relationships, blockers, and active work as the live Shark authority; treat product documents only as intent and decision context.\n" +
	"Respect hard precedence before considering priority, business value, progress, and continuity from active work; do not convert those fields into an undocumented weighted score.\n" +
	"Recommend exactly one eligibility=eligible epic key, give the decisive \"why now\" evidence, and compare it with the strongest eligible alternative.\n" +
	"If evidence_complete=false, no eligible root exists, or evidence contradicts, report the condition and the next evidence or relationship fix instead of guessing; when no eligible root exists, no root can be recommended from current Shark state.\n" +
	"End at advice. Do not claim, dispatch, or advance the root."

const portfolioAdviceMaxCyclomaticComplexity = 10

func TestPortfolioAdviceFunctionsStayWithinCyclomaticComplexityLimit(t *testing.T) {
	t.Parallel()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate portfolio advice test file")
	}
	adviceFile := filepath.Join(filepath.Dir(testFile), "portfolio_advice_service.go")
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, adviceFile, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", adviceFile, err)
	}

	functionCount := 0
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if !isFunction {
			continue
		}
		functionCount++
		complexity := portfolioAdviceCyclomaticComplexity(function)
		if complexity > portfolioAdviceMaxCyclomaticComplexity {
			t.Errorf(
				"%s has cyclomatic complexity %d; maximum is %d",
				function.Name.Name,
				complexity,
				portfolioAdviceMaxCyclomaticComplexity,
			)
		}
	}
	if functionCount == 0 {
		t.Fatal("portfolio_advice_service.go contains no functions; complexity guard did not inspect anything")
	}
}

func portfolioAdviceCyclomaticComplexity(function ast.Node) int {
	visitor := &portfolioAdviceComplexityVisitor{complexity: 1}
	ast.Walk(visitor, function)
	return visitor.complexity
}

type portfolioAdviceComplexityVisitor struct {
	complexity int
}

// Visit mirrors gocyclo's complexity definition without requiring the binary at test runtime.
func (v *portfolioAdviceComplexityVisitor) Visit(node ast.Node) ast.Visitor {
	switch current := node.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
		v.complexity++
	case *ast.CaseClause:
		if current.List != nil {
			v.complexity++
		}
	case *ast.CommClause:
		if current.Comm != nil {
			v.complexity++
		}
	case *ast.BinaryExpr:
		if current.Op == token.LAND || current.Op == token.LOR {
			v.complexity++
		}
	}
	return v
}

type stubPortfolioEpicReader struct {
	epics []*models.Epic
	err   error
	calls int
}

func (s *stubPortfolioEpicReader) List(context.Context, *models.EpicStatus) ([]*models.Epic, error) {
	s.calls++
	return s.epics, s.err
}

type stubPortfolioSnapshotReader struct {
	children          []portfoliorepo.ChildStateRow
	childErr          error
	childCalls        int
	relationships     []portfoliorepo.EpicRelationshipRow
	relationshipErr   error
	relationshipCalls int
}

func (s *stubPortfolioSnapshotReader) ListChildStates(context.Context) ([]portfoliorepo.ChildStateRow, error) {
	s.childCalls++
	return s.children, s.childErr
}

func (s *stubPortfolioSnapshotReader) ListEpicRelationships(context.Context) ([]portfoliorepo.EpicRelationshipRow, error) {
	s.relationshipCalls++
	return s.relationships, s.relationshipErr
}

type stubPortfolioClaimReader struct {
	claims      []*models.EntityClaim
	err         error
	calls       int
	evaluatedAt time.Time
}

func (s *stubPortfolioClaimReader) ListActiveReadOnly(_ context.Context, evaluatedAt time.Time) ([]*models.EntityClaim, error) {
	s.calls++
	s.evaluatedAt = evaluatedAt
	return s.claims, s.err
}

func TestPortfolioAdviceServiceEmptyPortfolio(t *testing.T) {
	epics := &stubPortfolioEpicReader{epics: []*models.Epic{}}
	snapshot := &stubPortfolioSnapshotReader{}
	claims := &stubPortfolioClaimReader{}
	service := NewPortfolioAdviceService(epics, snapshot, claims, portfolioTestWorkflows())

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	if advice == nil {
		t.Fatal("Advise() returned nil advice")
	}
	if !advice.EvidenceComplete {
		t.Error("EvidenceComplete = false, want true")
	}
	if advice.Prompt != wantPortfolioAdvicePrompt {
		t.Errorf("Prompt = %q, want exact static contract %q", advice.Prompt, wantPortfolioAdvicePrompt)
	}
	if advice.Epics == nil || advice.Relationships == nil || advice.Warnings == nil ||
		advice.Ordering.DependencyLayers == nil || advice.Ordering.RoadmapLayers == nil ||
		advice.Ordering.UnlayeredEpics == nil || advice.Ordering.Warnings == nil {
		t.Fatalf("Advise() returned a nil collection: %#v", advice)
	}
	if epics.calls != 1 || snapshot.childCalls != 1 || snapshot.relationshipCalls != 1 || claims.calls != 1 {
		t.Fatalf("read counts = epic:%d child:%d relationship:%d claim:%d, want one each",
			epics.calls, snapshot.childCalls, snapshot.relationshipCalls, claims.calls)
	}
	if claims.evaluatedAt.Location() != time.UTC {
		t.Errorf("claim evaluation location = %v, want UTC", claims.evaluatedAt.Location())
	}

	encoded, err := json.Marshal(advice)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "null") {
		t.Fatalf("empty advice contains null collection: %s", encoded)
	}
}

func TestPortfolioAdviceServiceAssemblesConfiguredEvidence(t *testing.T) {
	value := models.PriorityHigh
	heartbeat := time.Date(2026, 7, 20, 10, 4, 5, 0, time.FixedZone("test", -5*60*60))
	progress0, progressHalf, progress1 := 0.0, 0.5, 1.0
	epics := &stubPortfolioEpicReader{epics: []*models.Epic{
		portfolioTestEpic(4, "E04", "Held", "held_custom", models.PriorityLow, nil),
		portfolioTestEpic(1, "E01", "Shipped", "shipped_custom", models.PriorityLow, nil),
		portfolioTestEpic(3, "E03", "Third", "active_custom", models.PriorityMedium, nil),
		portfolioTestEpic(2, "E02", "Second", "active_custom", models.PriorityHigh, &value),
	}}
	snapshot := &stubPortfolioSnapshotReader{
		children: []portfoliorepo.ChildStateRow{
			portfolioTestChild(2, "E02", models.EntityTypeTask, "T-E02-F02-001", "Blocked task", "task_stuck", "F02", nil),
			portfolioTestChild(3, "E03", models.EntityTypeFeature, "F03", "Complete feature", "completed", "E03", floatPointer(2)),
			portfolioTestChild(2, "E02", models.EntityTypeFeature, "F02", "Ready feature", "ready_custom", "E02", floatPointer(50)),
			portfolioTestChild(2, "E02", models.EntityTypeFeature, "F01", "Blocked feature", "feature_stuck", "E02", floatPointer(20)),
		},
		relationships: []portfoliorepo.EpicRelationshipRow{
			portfolioTestRelationship(3, stringPointer("E03"), stringPointer("active_custom"), models.EntityRelDependsOn, 2, stringPointer("E02"), stringPointer("active_custom")),
			portfolioTestRelationship(2, stringPointer("E02"), stringPointer("active_custom"), models.EntityRelDependsOn, 1, stringPointer("E01"), stringPointer("shipped_custom")),
			portfolioTestRelationship(2, stringPointer("E02"), stringPointer("active_custom"), models.EntityRelBlocks, 4, stringPointer("E04"), stringPointer("held_custom")),
		},
	}
	claims := &stubPortfolioClaimReader{claims: []*models.EntityClaim{
		{EntityType: "task", EntityKey: "T-E02-F02-001", ClaimedBy: "qa-1", SessionID: "secret-task", Note: "secret-note", LastHeartbeat: heartbeat, Progress: &progress1},
		{EntityType: "bug", EntityKey: "B001", ClaimedBy: "ignored", SessionID: "secret-bug", LastHeartbeat: heartbeat},
		{EntityType: "epic", EntityKey: "E02", ClaimedBy: "lead", SessionID: "secret-epic", Note: "secret-note", LastHeartbeat: heartbeat, Progress: nil},
		{EntityType: "feature", EntityKey: "F01", ClaimedBy: "dev-1", SessionID: "secret-feature", Note: "secret-note", LastHeartbeat: heartbeat, Progress: &progress0},
		{EntityType: "feature", EntityKey: "F02", ClaimedBy: "dev-2", SessionID: "secret-feature-2", Note: "secret-note", LastHeartbeat: heartbeat, Progress: &progressHalf},
	}}
	service := NewPortfolioAdviceService(epics, snapshot, claims, portfolioTestWorkflows())

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	if !advice.EvidenceComplete {
		t.Fatalf("EvidenceComplete = false; warnings = %#v", advice.Warnings)
	}
	if got := portfolioEvidenceKeys(advice.Epics); !reflect.DeepEqual(got, []string{"E02", "E03", "E04"}) {
		t.Fatalf("epic keys = %#v, want terminal E01 filtered and lexical order", got)
	}

	e02 := requirePortfolioEvidence(t, advice.Epics, "E02")
	if e02.ProgressPct != 35 {
		t.Errorf("E02 ProgressPct = %v, want shared-formula result 35", e02.ProgressPct)
	}
	if e02.BusinessValue == nil || *e02.BusinessValue != "high" {
		t.Errorf("E02 BusinessValue = %#v, want high", e02.BusinessValue)
	}
	if e02.Eligibility != models.PortfolioEligibilityEligible || len(e02.EligibilityReasons) != 0 {
		t.Errorf("E02 eligibility = %s %#v, want eligible", e02.Eligibility, e02.EligibilityReasons)
	}
	wantBlocked := []models.PortfolioBlockedItem{
		{Kind: models.PortfolioBlockerWorkflowBlocked, EntityType: "feature", EntityKey: "F01", Title: "Blocked feature", Status: "feature_stuck"},
		{Kind: models.PortfolioBlockerWorkflowBlocked, EntityType: "task", EntityKey: "T-E02-F02-001", Title: "Blocked task", Status: "task_stuck"},
	}
	if !reflect.DeepEqual(e02.BlockedItems, wantBlocked) {
		t.Errorf("E02 BlockedItems = %#v, want %#v", e02.BlockedItems, wantBlocked)
	}
	if got := activeWorkKeys(e02.ActiveWork); !reflect.DeepEqual(got, []string{"epic:E02", "feature:F01", "feature:F02", "task:T-E02-F02-001"}) {
		t.Errorf("E02 ActiveWork keys = %#v, want sorted epic/feature/task claims", got)
	}
	for _, work := range e02.ActiveWork {
		if work.LastHeartbeat.Location() != time.UTC {
			t.Errorf("active work heartbeat location = %v, want UTC", work.LastHeartbeat.Location())
		}
	}

	e03 := requirePortfolioEvidence(t, advice.Epics, "E03")
	if e03.ProgressPct != 100 {
		t.Errorf("E03 ProgressPct = %v, want clamped shared-formula result 100", e03.ProgressPct)
	}
	if e03.Eligibility != models.PortfolioEligibilityIneligible || !reflect.DeepEqual(e03.EligibilityReasons, []string{"unresolved_dependency:E02"}) {
		t.Errorf("E03 eligibility = %s %#v", e03.Eligibility, e03.EligibilityReasons)
	}
	if len(e03.BlockedItems) != 1 || e03.BlockedItems[0].Kind != models.PortfolioBlockerHardDependency || e03.BlockedItems[0].EntityKey != "E02" {
		t.Errorf("E03 BlockedItems = %#v, want E02 hard dependency", e03.BlockedItems)
	}
	e04 := requirePortfolioEvidence(t, advice.Epics, "E04")
	if e04.Eligibility != models.PortfolioEligibilityIneligible ||
		!reflect.DeepEqual(e04.EligibilityReasons, []string{"blocked_by:E02", "epic_workflow_blocked"}) {
		t.Errorf("E04 eligibility = %s %#v", e04.Eligibility, e04.EligibilityReasons)
	}

	wantRelationships := []models.PortfolioEpicRelationship{
		portfolioPublicRelationship("E02", "active_custom", models.EntityRelBlocks, "E04", "held_custom", true, boolPointer(false)),
		portfolioPublicRelationship("E02", "active_custom", models.EntityRelDependsOn, "E01", "shipped_custom", true, boolPointer(true)),
		portfolioPublicRelationship("E03", "active_custom", models.EntityRelDependsOn, "E02", "active_custom", true, boolPointer(false)),
	}
	if !reflect.DeepEqual(advice.Relationships, wantRelationships) {
		t.Errorf("Relationships = %#v, want %#v", advice.Relationships, wantRelationships)
	}
	wantLayers := [][]string{{"E02"}, {"E03", "E04"}}
	if !reflect.DeepEqual(advice.Ordering.DependencyLayers, wantLayers) || !reflect.DeepEqual(advice.Ordering.RoadmapLayers, wantLayers) {
		t.Errorf("ordering = %#v, want layers %#v", advice.Ordering, wantLayers)
	}

	encoded, err := json.Marshal(advice)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	for _, secret := range []string{"secret-session", "secret-note", "secret-epic", "secret-feature", "secret-task"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("marshaled advice leaked %q: %s", secret, encoded)
		}
	}
}

func TestPortfolioAdviceServiceAllDirectFeaturesBlocked(t *testing.T) {
	service := NewPortfolioAdviceService(
		&stubPortfolioEpicReader{epics: []*models.Epic{portfolioTestEpic(1, "E01", "First", "active_custom", models.PriorityHigh, nil)}},
		&stubPortfolioSnapshotReader{children: []portfoliorepo.ChildStateRow{
			portfolioTestChild(1, "E01", models.EntityTypeFeature, "F01", "Blocked 1", "feature_stuck", "E01", nil),
			portfolioTestChild(1, "E01", models.EntityTypeFeature, "F02", "Blocked 2", "feature_stuck", "E01", nil),
			portfolioTestChild(1, "E01", models.EntityTypeFeature, "F03", "Done", "completed", "E01", nil),
		}},
		&stubPortfolioClaimReader{},
		portfolioTestWorkflows(),
	)

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	epic := requirePortfolioEvidence(t, advice.Epics, "E01")
	if epic.Eligibility != models.PortfolioEligibilityIneligible ||
		!reflect.DeepEqual(epic.EligibilityReasons, []string{"all_direct_features_blocked"}) {
		t.Errorf("eligibility = %s %#v, want all-direct-features reason", epic.Eligibility, epic.EligibilityReasons)
	}
}

func TestPortfolioAdviceServiceUsesReadOnlyClaimPolicy(t *testing.T) {
	now := time.Now().UTC()
	liveProgress := 0.25
	reclaimCalls := 0
	repo := &mockClaimRepo{
		ListFn: func(context.Context) ([]*models.EntityClaim, error) {
			return []*models.EntityClaim{
				{EntityType: "epic", EntityKey: "E01", ClaimedBy: "live", LastHeartbeat: now, Progress: &liveProgress},
				{EntityType: "epic", EntityKey: "E01", ClaimedBy: "expired", LastHeartbeat: now.Add(-2 * time.Hour)},
			}, nil
		},
		ReclaimFn: func(context.Context, time.Duration) (int64, error) {
			reclaimCalls++
			return 0, errors.New("read-only advice must not reclaim")
		},
	}
	ttl := time.Hour
	service := NewPortfolioAdviceService(
		&stubPortfolioEpicReader{epics: []*models.Epic{portfolioTestEpic(1, "E01", "First", "active_custom", models.PriorityHigh, nil)}},
		&stubPortfolioSnapshotReader{},
		NewClaimService(repo, &ttl),
		portfolioTestWorkflows(),
	)

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	work := requirePortfolioEvidence(t, advice.Epics, "E01").ActiveWork
	if len(work) != 1 || work[0].ClaimedBy != "live" {
		t.Errorf("ActiveWork = %#v, want only live claim", work)
	}
	if reclaimCalls != 0 {
		t.Errorf("ReclaimExpired calls = %d, want 0", reclaimCalls)
	}
}

func TestPortfolioAdviceServicePartialEvidenceDecisionTable(t *testing.T) {
	sentinel := errors.New("SELECT * FROM secrets WHERE token='do-not-leak'")
	tests := []struct {
		name            string
		childErr        error
		relationshipErr error
		claimErr        error
		wantCode        models.PortfolioWarningCode
		wantEligibility models.PortfolioEligibility
		wantEmptyLayers bool
	}{
		{name: "children", childErr: sentinel, wantCode: models.PortfolioWarningChildStateUnavailable, wantEligibility: models.PortfolioEligibilityUnknown},
		{name: "relationships", relationshipErr: sentinel, wantCode: models.PortfolioWarningRelationshipStateUnavailable, wantEligibility: models.PortfolioEligibilityUnknown, wantEmptyLayers: true},
		{name: "claims", claimErr: sentinel, wantCode: models.PortfolioWarningClaimStateUnavailable, wantEligibility: models.PortfolioEligibilityEligible},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPortfolioAdviceService(
				&stubPortfolioEpicReader{epics: []*models.Epic{portfolioTestEpic(1, "E01", "First", "active_custom", models.PriorityHigh, nil)}},
				&stubPortfolioSnapshotReader{childErr: tt.childErr, relationshipErr: tt.relationshipErr},
				&stubPortfolioClaimReader{err: tt.claimErr},
				portfolioTestWorkflows(),
			)
			advice, err := service.Advise(context.Background())
			if err != nil {
				t.Fatalf("Advise() error = %v", err)
			}
			if advice.EvidenceComplete {
				t.Error("EvidenceComplete = true, want false")
			}
			warning := requirePortfolioWarning(t, advice.Warnings, tt.wantCode)
			if strings.Contains(warning.Message, "SELECT") || strings.Contains(warning.Message, "do-not-leak") {
				t.Errorf("warning leaked dependency detail: %q", warning.Message)
			}
			if got := requirePortfolioEvidence(t, advice.Epics, "E01").Eligibility; got != tt.wantEligibility {
				t.Errorf("Eligibility = %s, want %s", got, tt.wantEligibility)
			}
			if tt.wantEmptyLayers && (len(advice.Ordering.DependencyLayers) != 0 || len(advice.Ordering.RoadmapLayers) != 0) {
				t.Errorf("ordering = %#v, want empty layers", advice.Ordering)
			}
		})
	}
}

func TestPortfolioAdviceServiceUnknownStatusesAndDanglingRelationships(t *testing.T) {
	service := NewPortfolioAdviceService(
		&stubPortfolioEpicReader{epics: []*models.Epic{
			portfolioTestEpic(1, "E01", "Unknown", "unconfigured", models.PriorityHigh, nil),
			portfolioTestEpic(2, "E02", "Known", "active_custom", models.PriorityHigh, nil),
		}},
		&stubPortfolioSnapshotReader{
			children: []portfoliorepo.ChildStateRow{
				portfolioTestChild(2, "E02", models.EntityTypeTask, "T-E02-F01-001", "Unknown task", "unconfigured_task", "F01", nil),
			},
			relationships: []portfoliorepo.EpicRelationshipRow{
				portfolioTestRelationship(2, stringPointer("E02"), stringPointer("active_custom"), models.EntityRelDependsOn, 999, nil, nil),
			},
		},
		&stubPortfolioClaimReader{},
		portfolioTestWorkflows(),
	)

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	if advice.EvidenceComplete {
		t.Error("EvidenceComplete = true, want false for dangling relationship")
	}
	if len(advice.Relationships) != 0 {
		t.Errorf("Relationships = %#v, want dangling row omitted", advice.Relationships)
	}
	if got := requirePortfolioEvidence(t, advice.Epics, "E01").Eligibility; got != models.PortfolioEligibilityUnknown {
		t.Errorf("E01 eligibility = %s, want unknown", got)
	}
	if got := requirePortfolioEvidence(t, advice.Epics, "E02").Eligibility; got != models.PortfolioEligibilityUnknown {
		t.Errorf("E02 eligibility = %s, want unknown", got)
	}
	unknown := requirePortfolioWarning(t, advice.Warnings, models.PortfolioWarningUnknownWorkflowStatus)
	if !reflect.DeepEqual(unknown.EpicKeys, []string{"E01"}) {
		t.Errorf("first unknown warning keys = %#v, want E01", unknown.EpicKeys)
	}
	if countPortfolioWarnings(advice.Warnings, models.PortfolioWarningUnknownWorkflowStatus) != 2 {
		t.Errorf("warnings = %#v, want epic and child unknown-status warnings", advice.Warnings)
	}
	dangling := requirePortfolioWarning(t, advice.Warnings, models.PortfolioWarningDanglingRelationship)
	if !reflect.DeepEqual(dangling.EpicKeys, []string{"E02"}) {
		t.Errorf("dangling keys = %#v, want E02", dangling.EpicKeys)
	}

	wantWarnings := []models.PortfolioWarning{
		{
			Code:     models.PortfolioWarningDanglingRelationship,
			Message:  "a relevant epic relationship has a missing endpoint and was excluded from ordering",
			EpicKeys: []string{"E02"},
		},
		{
			Code:     models.PortfolioWarningUnknownWorkflowStatus,
			Message:  `workflow status "unconfigured" for epic E01 is not configured`,
			EpicKeys: []string{"E01"},
		},
		{
			Code:     models.PortfolioWarningUnknownWorkflowStatus,
			Message:  `workflow status "unconfigured_task" for task T-E02-F01-001 is not configured`,
			EpicKeys: []string{"E02"},
		},
	}
	if !reflect.DeepEqual(advice.Warnings, wantWarnings) {
		t.Errorf("Warnings = %#v, want exact sorted set %#v", advice.Warnings, wantWarnings)
	}
}

func TestPortfolioAdviceServiceInvalidClaimProgressDegradesWithoutLeakingClaim(t *testing.T) {
	invalid := 1.5
	service := NewPortfolioAdviceService(
		&stubPortfolioEpicReader{epics: []*models.Epic{portfolioTestEpic(1, "E01", "First", "active_custom", models.PriorityHigh, nil)}},
		&stubPortfolioSnapshotReader{},
		&stubPortfolioClaimReader{claims: []*models.EntityClaim{{
			EntityType: "epic", EntityKey: "E01", ClaimedBy: "worker", SessionID: "secret",
			LastHeartbeat: time.Now(), Progress: &invalid,
		}}},
		portfolioTestWorkflows(),
	)

	advice, err := service.Advise(context.Background())
	if err != nil {
		t.Fatalf("Advise() error = %v", err)
	}
	if advice.EvidenceComplete {
		t.Error("EvidenceComplete = true, want false for invalid claim progress")
	}
	if work := requirePortfolioEvidence(t, advice.Epics, "E01").ActiveWork; len(work) != 0 {
		t.Errorf("ActiveWork = %#v, want invalid claim omitted", work)
	}
	if !hasPortfolioWarning(advice.Warnings, models.PortfolioWarningClaimStateUnavailable) {
		t.Errorf("Warnings = %#v, want typed claim warning", advice.Warnings)
	}
}

func TestPortfolioAdviceServiceFatalErrors(t *testing.T) {
	t.Run("epic list", func(t *testing.T) {
		sentinel := errors.New("database unavailable")
		service := NewPortfolioAdviceService(
			&stubPortfolioEpicReader{err: sentinel},
			&stubPortfolioSnapshotReader{},
			&stubPortfolioClaimReader{},
			portfolioTestWorkflows(),
		)
		advice, err := service.Advise(context.Background())
		if advice != nil || !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "list portfolio epics") {
			t.Fatalf("Advise() = (%#v, %v), want wrapped epic-list error", advice, err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		service := NewPortfolioAdviceService(
			&stubPortfolioEpicReader{},
			&stubPortfolioSnapshotReader{},
			&stubPortfolioClaimReader{},
			portfolioTestWorkflows(),
		)
		advice, err := service.Advise(ctx)
		if advice != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("Advise() = (%#v, %v), want context.Canceled", advice, err)
		}
	})

	t.Run("missing workflow configuration", func(t *testing.T) {
		snapshot := &stubPortfolioSnapshotReader{}
		claims := &stubPortfolioClaimReader{}
		service := NewPortfolioAdviceService(
			&stubPortfolioEpicReader{},
			snapshot,
			claims,
			stubPortfolioWorkflowProvider{},
		)
		advice, err := service.Advise(context.Background())
		if advice != nil || err == nil || !strings.Contains(err.Error(), "workflow configuration") {
			t.Fatalf("Advise() = (%#v, %v), want workflow configuration error", advice, err)
		}
		if snapshot.childCalls != 0 || snapshot.relationshipCalls != 0 || claims.calls != 0 {
			t.Fatalf("optional reads occurred after fatal config failure: child:%d relationship:%d claim:%d",
				snapshot.childCalls, snapshot.relationshipCalls, claims.calls)
		}
	})
}

type stubPortfolioWorkflowProvider struct{}

func (stubPortfolioWorkflowProvider) ForLevel(string) *workflow.Service { return nil }

func portfolioTestWorkflows() *workflow.Service {
	epic := portfolioTestWorkflow(map[string]*config.Step{
		"active_custom":  {Phase: "development"},
		"held_custom":    {Phase: "blocked", Parking: true},
		"completed":      {Phase: "development"},
		"shipped_custom": {Phase: "done", Terminal: true},
	})
	feature := portfolioTestWorkflow(map[string]*config.Step{
		"ready_custom":  {Phase: "development"},
		"feature_stuck": {Phase: "blocked", Parking: true},
		"completed":     {Phase: "done", Terminal: true},
		"cancelled":     {Phase: "done", Terminal: true},
	})
	task := portfolioTestWorkflow(map[string]*config.Step{
		"ready_task":   {Phase: "development"},
		"task_stuck":   {Phase: "blocked", Parking: true},
		"task_shipped": {Phase: "done", Terminal: true},
	})
	return workflow.NewServiceFromMultiLevel(&config.MultiLevelWorkflow{Epic: epic, Feature: feature, Task: task})
}

func portfolioTestWorkflow(steps map[string]*config.Step) *config.WorkflowConfig {
	wf := &config.WorkflowConfig{Version: "1.0", Steps: steps}
	wf.DeriveLegacy()
	return wf
}

func portfolioTestEpic(id int64, key, title, status string, priority models.Priority, businessValue *models.Priority) *models.Epic {
	return &models.Epic{
		BaseEntity: models.BaseEntity{ID: id, Key: key, Title: title},
		Status:     models.EpicStatus(status), Priority: priority, BusinessValue: businessValue,
	}
}

func portfolioTestChild(
	epicID int64,
	epicKey string,
	entityType models.EntityType,
	entityKey string,
	title string,
	status string,
	directParent string,
	progress *float64,
) portfoliorepo.ChildStateRow {
	return portfoliorepo.ChildStateRow{
		EpicID: epicID, EpicKey: epicKey, EntityType: entityType, EntityKey: entityKey,
		Title: title, Status: status, DirectParentKey: directParent, ProgressPct: progress,
	}
}

func portfolioTestRelationship(
	fromID int64,
	fromKey, fromStatus *string,
	relationshipType models.EntityRelationshipType,
	toID int64,
	toKey, toStatus *string,
) portfoliorepo.EpicRelationshipRow {
	return portfoliorepo.EpicRelationshipRow{
		FromEpicID: fromID, FromKey: fromKey, FromStatus: fromStatus,
		RelationshipType: relationshipType,
		ToEpicID:         toID, ToKey: toKey, ToStatus: toStatus,
	}
}

func portfolioPublicRelationship(
	fromKey, fromStatus string,
	relationshipType models.EntityRelationshipType,
	toKey, toStatus string,
	hard bool,
	satisfied *bool,
) models.PortfolioEpicRelationship {
	return models.PortfolioEpicRelationship{
		FromKey: fromKey, FromStatus: fromStatus, RelationshipType: relationshipType,
		ToKey: toKey, ToStatus: toStatus, Hard: hard, Satisfied: satisfied,
	}
}

func requirePortfolioEvidence(t *testing.T, epics []models.PortfolioEpicEvidence, key string) models.PortfolioEpicEvidence {
	t.Helper()
	for _, epic := range epics {
		if epic.Key == key {
			return epic
		}
	}
	t.Fatalf("epics = %#v, want key %s", epics, key)
	return models.PortfolioEpicEvidence{}
}

func portfolioEvidenceKeys(epics []models.PortfolioEpicEvidence) []string {
	keys := make([]string, 0, len(epics))
	for _, epic := range epics {
		keys = append(keys, epic.Key)
	}
	return keys
}

func activeWorkKeys(work []models.PortfolioActiveWork) []string {
	keys := make([]string, 0, len(work))
	for _, item := range work {
		keys = append(keys, item.EntityType+":"+item.EntityKey)
	}
	return keys
}

func stringPointer(value string) *string { return &value }

func floatPointer(value float64) *float64 { return &value }
