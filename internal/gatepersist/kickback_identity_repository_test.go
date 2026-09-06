package gatepersist

import (
	"context"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

// TestValidateKickbacks_RealFeatureRepository_BareSuffixResolvesToMainEntity
// is the code-review round 12 regression test. It deliberately deviates
// from this codebase's "service-layer tests use mocks; only repository
// tests touch a real database" convention
// (.claude/rules/testing/architecture.md) because the round 12 bug is a
// MISMATCH between keys.KeyService.Normalize (syntactic only) and
// FeatureRepository.GetByKey's real suffix-match resolution: a fake of
// either side in isolation cannot reproduce a bug that only exists in the
// gap BETWEEN them. Only the real repository, wired exactly as production
// wires it (EntityServiceTransitioner over an EntityRegistry, the same
// adapter internal/cli/commands.buildGateCoordinator uses), proves the fix
// actually closes the gap production exhibits.
func TestValidateKickbacks_RealFeatureRepository_BareSuffixResolvesToMainEntity(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewIsolatedTestDB(t)
	db := repository.NewDB(database)

	epicRepo := repository.NewEpicRepository(db)
	featureRepo := repository.NewFeatureRepository(db)

	epicRow := &models.Epic{
		BaseEntity: models.BaseEntity{Key: "E34", Title: "Test Epic"},
		Status:     models.EpicStatusActive,
		Priority:   models.PriorityMedium,
	}
	if err := epicRepo.Create(ctx, epicRow); err != nil {
		t.Fatalf("create epic: %v", err)
	}
	featureRow := &models.Feature{
		BaseEntity: models.BaseEntity{Key: "E34-F05", Title: "Test Feature"},
		EpicID:     epicRow.ID,
		Status:     models.FeatureStatusDraft,
	}
	if err := featureRepo.Create(ctx, featureRow); err != nil {
		t.Fatalf("create feature: %v", err)
	}

	// Premise assert: prove the real repository actually resolves the bare
	// suffix form to the SAME row as the full form -- the production
	// behavior this fix must account for. Without this assertion, a future
	// change to GetByKey's resolution rules could make the assertion below
	// pass vacuously (e.g. if GetByKey stopped suffix-matching, the
	// rejection below would then come from a different cause entirely).
	byFull, err := featureRepo.GetByKey(ctx, "E34-F05")
	if err != nil {
		t.Fatalf("GetByKey(E34-F05): %v", err)
	}
	bySuffix, err := featureRepo.GetByKey(ctx, "F05")
	if err != nil {
		t.Fatalf("GetByKey(F05): %v", err)
	}
	if bySuffix.ID != byFull.ID {
		t.Fatalf("premise failed: GetByKey(F05).ID = %d, GetByKey(E34-F05).ID = %d -- this test no longer models production's bare-suffix resolution", bySuffix.ID, byFull.ID)
	}

	registry := services.NewEntityRegistry()
	registry.Register(models.EntityTypeFeature, services.NewFeatureRepositoryAdapter(featureRepo))

	workflowSvc := testWorkflowService(t)
	entitySvc := services.NewEntityService(workflowSvc)
	transitioner := NewEntityServiceTransitioner(entitySvc, registry, workflowSvc)

	validator := newFakeStatusValidator().allow(models.EntityTypeFeature, "todo")

	// The bug (round 12): a feature-typed gate worker submits a kickback
	// whose entity_key is the BARE SUFFIX form of the main entity being
	// gated. keys.KeyService.Normalize("F05") != Normalize("E34-F05") (no
	// epic context to fold against), so the syntactic layer-1 check alone
	// accepts this. Before this fix, validateKickbacks trusted Normalize
	// alone and let it through; after this fix, the repository-backed
	// layer-2 check (via IdentityResolver) resolves both keys to the same
	// row and rejects it.
	kickbacks := []gateresult.Kickback{{EntityKey: "F05", TargetStatus: "todo", Reason: "r"}}
	if _, err := validateKickbacks(ctx, kickbacks, models.EntityTypeFeature, "E34-F05", validator, transitioner); err == nil {
		t.Fatalf("expected validateKickbacks to reject a kickback whose bare-suffix key resolves (via the real FeatureRepository) to the bound main entity, got nil error")
	}
}
