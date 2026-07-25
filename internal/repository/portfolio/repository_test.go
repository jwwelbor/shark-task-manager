package portfolio_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	claimrepo "github.com/jwwelbor/shark-task-manager/internal/repository/claim"
	"github.com/jwwelbor/shark-task-manager/internal/repository/dbconn"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func TestPortfolioAdvice_TC020TargetScale(t *testing.T) {
	// Deliberately do not use t.Parallel: the one-second local SQLite contract
	// must not compete with another fixture in this package.
	t.Run("empty lower boundary", func(t *testing.T) {
		database := testutil.NewIsolatedTestDB(t)
		service, reads := newObservedPortfolioAdviceService(database)
		before := readPortfolioTableCounts(t, database)

		advice, err := service.Advise(context.Background())
		if err != nil {
			t.Fatalf("Advise() empty fixture error = %v", err)
		}

		assertPortfolioAdviceReadCounts(t, reads, 1)
		assertAllocatedPortfolioAdvice(t, advice)
		if !advice.EvidenceComplete || len(advice.Epics) != 0 {
			t.Fatalf("empty advice = %#v, want complete evidence with zero candidates", advice)
		}
		if after := readPortfolioTableCounts(t, database); after != before {
			t.Fatalf("empty Advise() mutated material tables: before = %+v, after = %+v", before, after)
		}
	})

	t.Run("one epic lower boundary", func(t *testing.T) {
		database := testutil.NewIsolatedTestDB(t)
		seedTC020OneEpic(t, database)
		service, reads := newObservedPortfolioAdviceService(database)
		before := readPortfolioTableCounts(t, database)

		advice, err := service.Advise(context.Background())
		if err != nil {
			t.Fatalf("Advise() one-epic fixture error = %v", err)
		}

		assertPortfolioAdviceReadCounts(t, reads, 1)
		assertAllocatedPortfolioAdvice(t, advice)
		if !advice.EvidenceComplete || len(advice.Epics) != 1 || advice.Epics[0].Key != "E001" {
			t.Fatalf("one-epic advice candidates = %#v, want complete E001 evidence", advice.Epics)
		}
		wantLayers := [][]string{{"E001"}}
		if !reflect.DeepEqual(advice.Ordering.DependencyLayers, wantLayers) ||
			!reflect.DeepEqual(advice.Ordering.RoadmapLayers, wantLayers) {
			t.Fatalf("one-epic layers = (%v, %v), want %v",
				advice.Ordering.DependencyLayers, advice.Ordering.RoadmapLayers, wantLayers)
		}
		if after := readPortfolioTableCounts(t, database); after != before {
			t.Fatalf("one-epic Advise() mutated material tables: before = %+v, after = %+v", before, after)
		}
	})

	t.Run("exact target scale", func(t *testing.T) {
		database := testutil.NewIsolatedTestDB(t)
		seedTC020TargetFixture(t, database)
		service, reads := newObservedPortfolioAdviceService(database)

		before := readPortfolioTableCounts(t, database)
		wantCounts := portfolioTableCounts{
			epics: 200, features: 4_000, tasks: 1_000,
			relationships: 10_000, claims: 4,
		}
		if before != wantCounts {
			t.Fatalf("target fixture counts = %+v, want %+v", before, wantCounts)
		}

		started := time.Now()
		first, err := service.Advise(context.Background())
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Advise() target fixture error = %v", err)
		}
		if elapsed >= time.Second {
			t.Fatalf("Advise() target fixture took %v, want < 1s", elapsed)
		}
		t.Logf("first exact-scale Advise() completed in %v with four set reads", elapsed)
		assertPortfolioAdviceReadCounts(t, reads, 1)
		assertTC020TargetAdvice(t, first)
		if after := readPortfolioTableCounts(t, database); after != before {
			t.Fatalf("first target Advise() mutated material tables: before = %+v, after = %+v", before, after)
		}

		second, err := service.Advise(context.Background())
		if err != nil {
			t.Fatalf("second Advise() target fixture error = %v", err)
		}
		assertPortfolioAdviceReadCounts(t, reads, 2)
		if !reflect.DeepEqual(second, first) {
			t.Fatalf("repeated target advice changed over an unchanged snapshot")
		}
		if after := readPortfolioTableCounts(t, database); after != before {
			t.Fatalf("repeated target Advise() mutated material tables: before = %+v, after = %+v", before, after)
		}
	})
}

type portfolioAdviceReadCounts struct {
	snapshotReads int
}

type observedPortfolioSnapshotSource struct {
	delegate services.PortfolioSnapshotSource
	counts   *portfolioAdviceReadCounts
}

func (r *observedPortfolioSnapshotSource) ReadSnapshot(ctx context.Context) (portfoliorepo.Snapshot, error) {
	r.counts.snapshotReads++
	return r.delegate.ReadSnapshot(ctx)
}

func newObservedPortfolioAdviceService(
	database *sql.DB,
) (*services.PortfolioAdviceService, *portfolioAdviceReadCounts) {
	db := dbconn.NewDB(database)
	counts := &portfolioAdviceReadCounts{}
	snapshotSource := &observedPortfolioSnapshotSource{
		delegate: portfoliorepo.NewRepository(db),
		counts:   counts,
	}
	ttl := services.DefaultClaimTTL
	claimService := services.NewClaimService(claimrepo.NewRepository(db), &ttl)

	return services.NewPortfolioAdviceServiceFromSnapshot(
		snapshotSource,
		claimService,
		workflow.NewService(""),
	), counts
}

func assertPortfolioAdviceReadCounts(t *testing.T, counts *portfolioAdviceReadCounts, calls int) {
	t.Helper()
	if counts.snapshotReads != calls {
		t.Fatalf(
			"database snapshot reads after %d Advise() calls = %d; want exactly %d (one hierarchy-view query per call)",
			calls,
			counts.snapshotReads,
			calls,
		)
	}
}

func assertAllocatedPortfolioAdvice(t *testing.T, advice *models.PortfolioAdviceEnvelope) {
	t.Helper()
	if advice == nil {
		t.Fatal("Advise() returned nil envelope")
	}
	if advice.Mode != models.PortfolioAdviceModePortfolioAdvice ||
		advice.Epics == nil || advice.Relationships == nil || advice.Warnings == nil ||
		advice.Ordering.DependencyLayers == nil || advice.Ordering.RoadmapLayers == nil ||
		advice.Ordering.UnlayeredEpics == nil || advice.Ordering.Warnings == nil {
		t.Fatalf("Advise() returned incomplete allocated contract: %#v", advice)
	}
}

func assertTC020TargetAdvice(t *testing.T, advice *models.PortfolioAdviceEnvelope) {
	t.Helper()
	assertAllocatedPortfolioAdvice(t, advice)
	if !advice.EvidenceComplete {
		t.Fatalf("target advice evidence_complete = false; warnings = %#v / %#v", advice.Warnings, advice.Ordering.Warnings)
	}
	if len(advice.Epics) != 200 {
		t.Fatalf("target advice candidates = %d, want 200", len(advice.Epics))
	}
	if advice.Epics[0].Key != "E001" || advice.Epics[len(advice.Epics)-1].Key != "E200" {
		t.Fatalf("target advice candidate bounds = %s..%s, want E001..E200",
			advice.Epics[0].Key, advice.Epics[len(advice.Epics)-1].Key)
	}
	if len(advice.Relationships) != 10_000 {
		t.Fatalf("target advice relationships = %d, want 10000", len(advice.Relationships))
	}
	if len(advice.Warnings) != 0 || len(advice.Ordering.Warnings) != 0 || len(advice.Ordering.UnlayeredEpics) != 0 {
		t.Fatalf("acyclic target advice has warnings/unlayered epics: warnings=%#v ordering=%#v",
			advice.Warnings, advice.Ordering)
	}

	eligible := make([]string, 0, 1)
	activeWork := make(map[string]bool)
	for _, epic := range advice.Epics {
		if epic.Eligibility == models.PortfolioEligibilityEligible {
			eligible = append(eligible, epic.Key)
		}
		for _, claim := range epic.ActiveWork {
			activeWork[claim.EntityKey] = true
		}
	}
	if !reflect.DeepEqual(eligible, []string{"E001"}) {
		t.Fatalf("eligible target roots = %v, want [E001]", eligible)
	}
	wantActive := []string{"E001", "E002-F01", "T-E003-F01-001"}
	for _, key := range wantActive {
		if !activeWork[key] {
			t.Errorf("live claim %s missing from target active_work", key)
		}
	}
	if activeWork["E004"] || len(activeWork) != len(wantActive) {
		t.Fatalf("target active_work keys = %v, want only %v (expired E004 excluded)", activeWork, wantActive)
	}

	seenTypes := make(map[models.EntityRelationshipType]bool)
	for _, relationship := range advice.Relationships {
		seenTypes[relationship.RelationshipType] = true
	}
	for _, relationshipType := range []models.EntityRelationshipType{
		models.EntityRelBlocks,
		models.EntityRelDependsOn,
		models.EntityRelFollows,
	} {
		if !seenTypes[relationshipType] {
			t.Errorf("target advice does not contain supported relationship type %s", relationshipType)
		}
	}
	if len(advice.Ordering.DependencyLayers) == 0 || len(advice.Ordering.RoadmapLayers) == 0 {
		t.Fatalf("target ordering has no layers: %#v", advice.Ordering)
	}
	if !reflect.DeepEqual(advice.Ordering.DependencyLayers[0], []string{"E001"}) ||
		!reflect.DeepEqual(advice.Ordering.RoadmapLayers[0], []string{"E001"}) {
		t.Fatalf("target root layers = (%v, %v), want [E001]",
			advice.Ordering.DependencyLayers[0], advice.Ordering.RoadmapLayers[0])
	}
	if countPortfolioLayerKeys(advice.Ordering.DependencyLayers) != 200 ||
		countPortfolioLayerKeys(advice.Ordering.RoadmapLayers) != 200 {
		t.Fatalf("target layers do not cover all 200 candidates: dependency=%d roadmap=%d",
			countPortfolioLayerKeys(advice.Ordering.DependencyLayers),
			countPortfolioLayerKeys(advice.Ordering.RoadmapLayers))
	}
}

func countPortfolioLayerKeys(layers [][]string) int {
	total := 0
	for _, layer := range layers {
		total += len(layer)
	}
	return total
}

func seedTC020OneEpic(t *testing.T, database *sql.DB) {
	t.Helper()
	tx, err := database.Begin()
	if err != nil {
		t.Fatalf("begin one-epic fixture transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		`INSERT INTO epics (key, title, description, status, priority)
		 VALUES ('E001', 'One epic', 'TC-020 lower boundary', 'active', 'high')`,
	); err != nil {
		t.Fatalf("insert one-epic fixture: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit one-epic fixture: %v", err)
	}
}

func seedTC020TargetFixture(t *testing.T, database *sql.DB) {
	t.Helper()
	const (
		epicCount          = 200
		featuresPerEpic    = 20
		tasksPerEpic       = 5
		relationshipCount  = 10_000
		fixtureDescription = "TC-020 exact-scale fixture"
	)

	tx, err := database.Begin()
	if err != nil {
		t.Fatalf("begin target fixture transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	epicStmt := prepareTC020Statement(t, tx,
		`INSERT INTO epics (key, title, description, status, priority)
		 VALUES (?, ?, ?, 'active', 'medium')`)
	featureStmt := prepareTC020Statement(t, tx,
		`INSERT INTO features (epic_id, key, title, description, status, progress_pct)
		 VALUES (?, ?, ?, ?, 'active', ?)`)
	taskStmt := prepareTC020Statement(t, tx,
		`INSERT INTO tasks (feature_id, key, title, description, status, priority)
		 VALUES (?, ?, ?, ?, 'development', 5)`)
	relationshipStmt := prepareTC020Statement(t, tx,
		`INSERT INTO entity_relationships
		 (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		 VALUES ('epic', ?, 'epic', ?, ?)`)
	claimStmt := prepareTC020Statement(t, tx,
		`INSERT INTO entity_claims
		 (entity_type, entity_key, claimed_by, session_id, claimed_at, last_heartbeat, progress, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)

	epicIDs := make([]int64, epicCount)
	for epicNumber := 1; epicNumber <= epicCount; epicNumber++ {
		epicKey := fmt.Sprintf("E%03d", epicNumber)
		result, err := epicStmt.Exec(epicKey, "Portfolio epic "+epicKey, fixtureDescription)
		if err != nil {
			t.Fatalf("insert target epic %s: %v", epicKey, err)
		}
		epicID := lastInsertID(t, result)
		epicIDs[epicNumber-1] = epicID

		featureIDs := make([]int64, featuresPerEpic)
		for featureNumber := 1; featureNumber <= featuresPerEpic; featureNumber++ {
			featureKey := fmt.Sprintf("%s-F%02d", epicKey, featureNumber)
			progress := float64((epicNumber + featureNumber) % 101)
			result, err := featureStmt.Exec(
				epicID, featureKey, "Portfolio feature "+featureKey, fixtureDescription, progress,
			)
			if err != nil {
				t.Fatalf("insert target feature %s: %v", featureKey, err)
			}
			featureIDs[featureNumber-1] = lastInsertID(t, result)
		}

		for taskNumber := 1; taskNumber <= tasksPerEpic; taskNumber++ {
			featureKey := fmt.Sprintf("%s-F%02d", epicKey, taskNumber)
			taskKey := fmt.Sprintf("T-%s-001", featureKey)
			if _, err := taskStmt.Exec(
				featureIDs[taskNumber-1], taskKey, "Portfolio task "+taskKey, fixtureDescription,
			); err != nil {
				t.Fatalf("insert target task %s: %v", taskKey, err)
			}
		}
	}

	insertedRelationships := 0
	for before := 0; before < epicCount && insertedRelationships < relationshipCount; before++ {
		for after := before + 1; after < epicCount && insertedRelationships < relationshipCount; after++ {
			fromID, toID := epicIDs[before], epicIDs[after]
			relationshipType := models.EntityRelBlocks
			switch insertedRelationships % 3 {
			case 1:
				fromID, toID = epicIDs[after], epicIDs[before]
				relationshipType = models.EntityRelDependsOn
			case 2:
				fromID, toID = epicIDs[after], epicIDs[before]
				relationshipType = models.EntityRelFollows
			}
			if _, err := relationshipStmt.Exec(fromID, toID, relationshipType); err != nil {
				t.Fatalf("insert target relationship %d: %v", insertedRelationships, err)
			}
			insertedRelationships++
		}
	}
	if insertedRelationships != relationshipCount {
		t.Fatalf("inserted %d target relationships, want %d", insertedRelationships, relationshipCount)
	}

	// Advise currently owns its evaluation time and has no injectable clock.
	// Heartbeats far in the future/past make the live/expired split stable while
	// retaining the real ClaimService and repository path.
	farPast := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	farFuture := time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC)
	claims := []struct {
		entityType string
		entityKey  string
		holder     string
		session    string
		heartbeat  time.Time
		progress   float64
	}{
		{entityType: "epic", entityKey: "E001", holder: "portfolio-live-epic", session: "live-epic", heartbeat: farFuture, progress: 0.1},
		{entityType: "feature", entityKey: "E002-F01", holder: "portfolio-live-feature", session: "live-feature", heartbeat: farFuture, progress: 0.5},
		{entityType: "task", entityKey: "T-E003-F01-001", holder: "portfolio-live-task", session: "live-task", heartbeat: farFuture, progress: 0.9},
		{entityType: "epic", entityKey: "E004", holder: "portfolio-expired", session: "expired-epic", heartbeat: farPast, progress: 0.2},
	}
	for _, claim := range claims {
		if _, err := claimStmt.Exec(
			claim.entityType,
			claim.entityKey,
			claim.holder,
			claim.session,
			farPast,
			claim.heartbeat,
			claim.progress,
			fixtureDescription,
		); err != nil {
			t.Fatalf("insert target claim %s/%s: %v", claim.entityType, claim.entityKey, err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit target fixture: %v", err)
	}
}

func prepareTC020Statement(t *testing.T, tx *sql.Tx, query string) *sql.Stmt {
	t.Helper()
	statement, err := tx.Prepare(query)
	if err != nil {
		t.Fatalf("prepare TC-020 fixture statement: %v", err)
	}
	t.Cleanup(func() { _ = statement.Close() })
	return statement
}

func TestRepository_ListChildStates(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	cleanupPortfolioFixtures(t, database)
	t.Cleanup(func() { cleanupPortfolioFixtures(t, database) })

	e97ID := insertPortfolioEpic(t, database, "E97", "Second epic", "planning")
	e96ID := insertPortfolioEpic(t, database, "E96", "First epic", "active")

	e96f02ID := insertPortfolioFeature(t, database, e96ID, "E96-F02", "Second feature", "completed", 100)
	e97f01ID := insertPortfolioFeature(t, database, e97ID, "E97-F01", "Third feature", "draft", 0)
	e96f01ID := insertPortfolioFeature(t, database, e96ID, "E96-F01", "First feature", "active", 37.5)

	insertPortfolioTask(t, database, e96f02ID, "T-E96-F02-001", "Later task", "completed")
	insertPortfolioTask(t, database, e97f01ID, "T-E97-F01-001", "Other epic task", "todo")
	insertPortfolioTask(t, database, e96f01ID, "T-E96-F01-001", "Earlier task", "active")

	repo := portfoliorepo.NewRepository(dbconn.NewDB(database))
	before := readPortfolioTableCounts(t, database)

	first, err := repo.ListChildStates(context.Background())
	if err != nil {
		t.Fatalf("ListChildStates() error = %v", err)
	}
	second, err := repo.ListChildStates(context.Background())
	if err != nil {
		t.Fatalf("second ListChildStates() error = %v", err)
	}

	want := []portfoliorepo.ChildStateRow{
		{
			EpicID: e96ID, EpicKey: "E96", EntityType: models.EntityTypeFeature,
			EntityKey: "E96-F01", Title: "First feature", Status: "active",
			DirectParentKey: "E96", ProgressPct: float64Pointer(37.5),
		},
		{
			EpicID: e96ID, EpicKey: "E96", EntityType: models.EntityTypeFeature,
			EntityKey: "E96-F02", Title: "Second feature", Status: "completed",
			DirectParentKey: "E96", ProgressPct: float64Pointer(100),
		},
		{
			EpicID: e96ID, EpicKey: "E96", EntityType: models.EntityTypeTask,
			EntityKey: "T-E96-F01-001", Title: "Earlier task", Status: "active",
			DirectParentKey: "E96-F01", ProgressPct: nil,
		},
		{
			EpicID: e96ID, EpicKey: "E96", EntityType: models.EntityTypeTask,
			EntityKey: "T-E96-F02-001", Title: "Later task", Status: "completed",
			DirectParentKey: "E96-F02", ProgressPct: nil,
		},
		{
			EpicID: e97ID, EpicKey: "E97", EntityType: models.EntityTypeFeature,
			EntityKey: "E97-F01", Title: "Third feature", Status: "draft",
			DirectParentKey: "E97", ProgressPct: float64Pointer(0),
		},
		{
			EpicID: e97ID, EpicKey: "E97", EntityType: models.EntityTypeTask,
			EntityKey: "T-E97-F01-001", Title: "Other epic task", Status: "todo",
			DirectParentKey: "E97-F01", ProgressPct: nil,
		},
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("ListChildStates() = %#v, want %#v", first, want)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("repeated ListChildStates() ordering changed:\nfirst  = %#v\nsecond = %#v", first, second)
	}
	if after := readPortfolioTableCounts(t, database); after != before {
		t.Fatalf("ListChildStates() mutated tables: before = %+v, after = %+v", before, after)
	}
}

func TestRepository_ListEpicRelationships(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	cleanupPortfolioFixtures(t, database)
	t.Cleanup(func() { cleanupPortfolioFixtures(t, database) })

	e97ID := insertPortfolioEpic(t, database, "E97", "Second epic", "active")
	e98ID := insertPortfolioEpic(t, database, "E98", "Terminal epic", "completed")
	e96ID := insertPortfolioEpic(t, database, "E96", "First epic", "planning")
	e96FeatureID := insertPortfolioFeature(t, database, e96ID, "E96-F01", "Feature endpoint", "active", 20)
	const danglingFromID int64 = 900096
	const danglingToID int64 = 900097

	insertPortfolioRelationship(t, database, models.EntityTypeEpic, danglingFromID, models.EntityTypeEpic, e96ID, models.EntityRelFollows)
	insertPortfolioRelationship(t, database, models.EntityTypeEpic, e96ID, models.EntityTypeEpic, danglingToID, models.EntityRelBlocks)
	insertPortfolioRelationship(t, database, models.EntityTypeEpic, e96ID, models.EntityTypeEpic, e97ID, models.EntityRelDependsOn)
	insertPortfolioRelationship(t, database, models.EntityTypeEpic, e97ID, models.EntityTypeEpic, e98ID, models.EntityRelFollows)
	insertPortfolioRelationship(t, database, models.EntityTypeEpic, e96ID, models.EntityTypeEpic, e98ID, models.EntityRelRelatedTo)
	insertPortfolioRelationship(t, database, models.EntityTypeEpic, e96ID, models.EntityTypeFeature, e96FeatureID, models.EntityRelDependsOn)

	repo := portfoliorepo.NewRepository(dbconn.NewDB(database))
	before := readPortfolioTableCounts(t, database)

	first, err := repo.ListEpicRelationships(context.Background())
	if err != nil {
		t.Fatalf("ListEpicRelationships() error = %v", err)
	}
	second, err := repo.ListEpicRelationships(context.Background())
	if err != nil {
		t.Fatalf("second ListEpicRelationships() error = %v", err)
	}

	want := []portfoliorepo.EpicRelationshipRow{
		{
			FromEpicID: danglingFromID, FromKey: nil, FromStatus: nil,
			RelationshipType: models.EntityRelFollows,
			ToEpicID:         e96ID, ToKey: stringPointer("E96"), ToStatus: stringPointer("planning"),
		},
		{
			FromEpicID: e96ID, FromKey: stringPointer("E96"), FromStatus: stringPointer("planning"),
			RelationshipType: models.EntityRelBlocks,
			ToEpicID:         danglingToID, ToKey: nil, ToStatus: nil,
		},
		{
			FromEpicID: e96ID, FromKey: stringPointer("E96"), FromStatus: stringPointer("planning"),
			RelationshipType: models.EntityRelDependsOn,
			ToEpicID:         e97ID, ToKey: stringPointer("E97"), ToStatus: stringPointer("active"),
		},
		{
			FromEpicID: e97ID, FromKey: stringPointer("E97"), FromStatus: stringPointer("active"),
			RelationshipType: models.EntityRelFollows,
			ToEpicID:         e98ID, ToKey: stringPointer("E98"), ToStatus: stringPointer("completed"),
		},
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("ListEpicRelationships() = %#v, want %#v", first, want)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("repeated ListEpicRelationships() ordering changed:\nfirst  = %#v\nsecond = %#v", first, second)
	}
	if after := readPortfolioTableCounts(t, database); after != before {
		t.Fatalf("ListEpicRelationships() mutated tables: before = %+v, after = %+v", before, after)
	}
}

// TestReadSnapshot_DecodesClaimWrittenByProductionClaimPath pins the wire
// format actually stored by production writes. claimrepo.Claim omits
// last_heartbeat on insert, so SQLite's DEFAULT CURRENT_TIMESTAMP writes
// "YYYY-MM-DD HH:MM:SS" — the only heartbeat format Shark ever persists. The
// snapshot query projects that column through json_object as raw text, so the
// decode path (not the driver) has to cope with it. Binding a Go time.Time in a
// fixture, as the scale fixture does, never produces this format and therefore
// cannot cover it: with one epic and one claim, bare `shark plan` failed hard.
func TestReadSnapshot_DecodesClaimWrittenByProductionClaimPath(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewIsolatedTestDB(t)
	cleanupPortfolioFixtures(t, database)
	t.Cleanup(func() {
		cleanupPortfolioFixtures(t, database)
		if _, err := database.Exec("DELETE FROM entity_claims"); err != nil {
			t.Fatalf("cleanup entity_claims: %v", err)
		}
	})

	insertPortfolioEpic(t, database, "E96", "Claimed epic", "active")

	db := dbconn.NewDB(database)
	progress := 0.25
	stored, err := claimrepo.NewRepository(db).Claim(ctx, &models.EntityClaim{
		EntityType: "epic",
		EntityKey:  "E96",
		ClaimedBy:  "portfolio-production-path",
		SessionID:  "portfolio-production-session",
		Progress:   &progress,
	})
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}

	snapshot, err := portfoliorepo.NewRepository(db).ReadSnapshot(ctx)
	if err != nil {
		t.Fatalf("ReadSnapshot() error = %v", err)
	}
	if len(snapshot.Claims) != 1 {
		t.Fatalf("ReadSnapshot() claims = %#v, want the one claim written by claimrepo.Claim", snapshot.Claims)
	}

	claim := snapshot.Claims[0]
	if claim.EntityType != "epic" || claim.EntityKey != "E96" || claim.ClaimedBy != "portfolio-production-path" {
		t.Fatalf("ReadSnapshot() claim identity = %#v, want the seeded epic claim", claim)
	}
	if claim.LastHeartbeat.IsZero() {
		t.Fatalf("ReadSnapshot() decoded a zero heartbeat for %s", claim.EntityKey)
	}
	// The heartbeat must survive the JSON projection as the same instant the
	// claim repository reads back, so live/expired TTL decisions agree across
	// both read paths. The tolerance guards gross mis-parse — a timezone offset
	// applied twice, or naive text read as local instead of UTC — not the
	// sub-millisecond truncation the SQL normalization introduces.
	if delta := claim.LastHeartbeat.Sub(stored.LastHeartbeat); delta > time.Millisecond || delta < -time.Millisecond {
		t.Fatalf("snapshot heartbeat = %s, claim repository heartbeat = %s (delta %v)",
			claim.LastHeartbeat, stored.LastHeartbeat, delta)
	}
	if claim.LastHeartbeat.Location() != time.UTC {
		t.Fatalf("snapshot heartbeat location = %s, want UTC", claim.LastHeartbeat.Location())
	}
}

func TestRepository_ListMethodsReturnAllocatedEmptySlices(t *testing.T) {
	database := testutil.NewIsolatedTestDB(t)
	repo := portfoliorepo.NewRepository(dbconn.NewDB(database))

	childStates, err := repo.ListChildStates(context.Background())
	if err != nil {
		t.Fatalf("ListChildStates() error = %v", err)
	}
	if childStates == nil || len(childStates) != 0 {
		t.Fatalf("ListChildStates() = %#v, want allocated empty slice", childStates)
	}

	relationships, err := repo.ListEpicRelationships(context.Background())
	if err != nil {
		t.Fatalf("ListEpicRelationships() error = %v", err)
	}
	if relationships == nil || len(relationships) != 0 {
		t.Fatalf("ListEpicRelationships() = %#v, want allocated empty slice", relationships)
	}
}

type portfolioTableCounts struct {
	epics         int
	features      int
	tasks         int
	relationships int
	claims        int
	history       int
	workSessions  int
}

func readPortfolioTableCounts(t *testing.T, database *sql.DB) portfolioTableCounts {
	t.Helper()
	var counts portfolioTableCounts
	for _, query := range []struct {
		name string
		dest *int
	}{
		{name: "epics", dest: &counts.epics},
		{name: "features", dest: &counts.features},
		{name: "tasks", dest: &counts.tasks},
		{name: "entity_relationships", dest: &counts.relationships},
		{name: "entity_claims", dest: &counts.claims},
		{name: "entity_history", dest: &counts.history},
		{name: "work_sessions", dest: &counts.workSessions},
	} {
		if err := database.QueryRow("SELECT COUNT(*) FROM " + query.name).Scan(query.dest); err != nil {
			t.Fatalf("count %s: %v", query.name, err)
		}
	}
	return counts
}

func cleanupPortfolioFixtures(t *testing.T, database *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		"DELETE FROM entity_relationships",
		"DELETE FROM tasks",
		"DELETE FROM features",
		"DELETE FROM epics",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("cleanup with %q: %v", statement, err)
		}
	}
}

func insertPortfolioEpic(t *testing.T, database *sql.DB, key, title, status string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO epics (key, title, description, status, priority)
		 VALUES (?, ?, 'portfolio repository fixture', ?, 'medium')`,
		key, title, status,
	)
	if err != nil {
		t.Fatalf("insert epic %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func insertPortfolioFeature(t *testing.T, database *sql.DB, epicID int64, key, title, status string, progress float64) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO features (epic_id, key, title, description, status, progress_pct)
		 VALUES (?, ?, ?, 'portfolio repository fixture', ?, ?)`,
		epicID, key, title, status, progress,
	)
	if err != nil {
		t.Fatalf("insert feature %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func insertPortfolioTask(t *testing.T, database *sql.DB, featureID int64, key, title, status string) int64 {
	t.Helper()
	result, err := database.Exec(
		`INSERT INTO tasks (feature_id, key, title, description, status, priority)
		 VALUES (?, ?, ?, 'portfolio repository fixture', ?, 5)`,
		featureID, key, title, status,
	)
	if err != nil {
		t.Fatalf("insert task %s: %v", key, err)
	}
	return lastInsertID(t, result)
}

func insertPortfolioRelationship(
	t *testing.T,
	database *sql.DB,
	fromType models.EntityType,
	fromID int64,
	toType models.EntityType,
	toID int64,
	relationshipType models.EntityRelationshipType,
) {
	t.Helper()
	_, err := database.Exec(
		`INSERT INTO entity_relationships
		 (from_entity_type, from_entity_id, to_entity_type, to_entity_id, relationship_type)
		 VALUES (?, ?, ?, ?, ?)`,
		fromType, fromID, toType, toID, relationshipType,
	)
	if err != nil {
		t.Fatalf("insert relationship %s(%d) -[%s]-> %s(%d): %v", fromType, fromID, relationshipType, toType, toID, err)
	}
}

func lastInsertID(t *testing.T, result sql.Result) int64 {
	t.Helper()
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId(): %v", err)
	}
	return id
}

func float64Pointer(value float64) *float64 { return &value }

func stringPointer(value string) *string { return &value }
