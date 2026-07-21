package services

import (
	"math"
	"sort"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

func assemblePortfolioRelationships(
	byKey map[string]*portfolioEpicAssembly,
	allEpics map[string]*models.Epic,
	rows []portfoliorepo.EpicRelationshipRow,
	epicWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) {
	for _, row := range rows {
		relationship, include := assemblePortfolioRelationship(row, byKey, allEpics, epicWorkflow, advice)
		if include {
			advice.Relationships = append(advice.Relationships, relationship)
		}
	}

	sort.Slice(advice.Relationships, func(i, j int) bool {
		left, right := advice.Relationships[i], advice.Relationships[j]
		if left.FromKey != right.FromKey {
			return left.FromKey < right.FromKey
		}
		if left.RelationshipType != right.RelationshipType {
			return left.RelationshipType < right.RelationshipType
		}
		return left.ToKey < right.ToKey
	})
}

type portfolioRelationshipCandidates struct {
	from bool
	to   bool
}

type portfolioHardRelationship struct {
	dependentKey      string
	predecessorKey    string
	predecessorStatus string
}

func assemblePortfolioRelationship(
	row portfoliorepo.EpicRelationshipRow,
	byKey map[string]*portfolioEpicAssembly,
	allEpics map[string]*models.Epic,
	epicWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) (models.PortfolioEpicRelationship, bool) {
	candidates := portfolioRelationshipCandidateState(row, byKey)
	if !candidates.from && !candidates.to {
		return models.PortfolioEpicRelationship{}, false
	}
	if row.FromKey == nil || row.FromStatus == nil || row.ToKey == nil || row.ToStatus == nil {
		recordPortfolioDanglingRelationship(row, byKey, advice)
		return models.PortfolioEpicRelationship{}, false
	}

	fromKey, toKey := *row.FromKey, *row.ToKey
	fromStatus, toStatus := *row.FromStatus, *row.ToStatus
	portfolioMarkInvalidRelationshipStatus(fromKey, fromStatus, candidates.from, epicWorkflow, byKey, advice)
	portfolioMarkInvalidRelationshipStatus(toKey, toStatus, candidates.to, epicWorkflow, byKey, advice)

	relationship, hard, supported := portfolioRelationshipEvidence(row, fromKey, fromStatus, toKey, toStatus)
	if !supported {
		return models.PortfolioEpicRelationship{}, false
	}
	applyPortfolioHardRelationship(&relationship, hard, row.RelationshipType, byKey, allEpics, epicWorkflow)
	return relationship, true
}

func portfolioRelationshipCandidateState(
	row portfoliorepo.EpicRelationshipRow,
	byKey map[string]*portfolioEpicAssembly,
) portfolioRelationshipCandidates {
	candidates := portfolioRelationshipCandidates{}
	if row.FromKey != nil {
		candidates.from = byKey[*row.FromKey] != nil
	}
	if row.ToKey != nil {
		candidates.to = byKey[*row.ToKey] != nil
	}
	return candidates
}
func recordPortfolioDanglingRelationship(
	row portfoliorepo.EpicRelationshipRow,
	byKey map[string]*portfolioEpicAssembly,
	advice *models.PortfolioAdviceEnvelope,
) {
	keys := portfolioKnownCandidateKeys(row, byKey)
	for _, key := range keys {
		byKey[key].eligibilityUnknown = true
	}
	advice.EvidenceComplete = false
	advice.Warnings = append(advice.Warnings, models.PortfolioWarning{
		Code:     models.PortfolioWarningDanglingRelationship,
		Message:  "a relevant epic relationship has a missing endpoint and was excluded from ordering",
		EpicKeys: keys,
	})
}

func portfolioMarkInvalidRelationshipStatus(
	key string,
	status string,
	candidate bool,
	epicWorkflow *workflow.Service,
	byKey map[string]*portfolioEpicAssembly,
	advice *models.PortfolioAdviceEnvelope,
) {
	if epicWorkflow.IsValidStatus(status) {
		return
	}
	portfolioMarkUnknownRelationshipStatus(key, status, candidate, byKey, advice)
}

func portfolioRelationshipEvidence(
	row portfoliorepo.EpicRelationshipRow,
	fromKey string,
	fromStatus string,
	toKey string,
	toStatus string,
) (models.PortfolioEpicRelationship, portfolioHardRelationship, bool) {
	relationship := models.PortfolioEpicRelationship{
		FromKey: fromKey, FromStatus: fromStatus, RelationshipType: row.RelationshipType,
		ToKey: toKey, ToStatus: toStatus,
	}
	switch row.RelationshipType {
	case models.EntityRelDependsOn:
		relationship.Hard = true
		return relationship, portfolioHardRelationship{fromKey, toKey, toStatus}, true
	case models.EntityRelBlocks:
		relationship.Hard = true
		return relationship, portfolioHardRelationship{toKey, fromKey, fromStatus}, true
	case models.EntityRelFollows:
		return relationship, portfolioHardRelationship{}, true
	default:
		return models.PortfolioEpicRelationship{}, portfolioHardRelationship{}, false
	}
}

func applyPortfolioHardRelationship(
	relationship *models.PortfolioEpicRelationship,
	hard portfolioHardRelationship,
	relationshipType models.EntityRelationshipType,
	byKey map[string]*portfolioEpicAssembly,
	allEpics map[string]*models.Epic,
	epicWorkflow *workflow.Service,
) {
	if !relationship.Hard {
		return
	}
	predecessorValid := epicWorkflow.IsValidStatus(hard.predecessorStatus)
	satisfied := predecessorValid && epicWorkflow.IsTerminalStatus(hard.predecessorStatus)
	relationship.Satisfied = &satisfied
	state := byKey[hard.dependentKey]
	if state == nil || satisfied {
		return
	}
	if !predecessorValid {
		state.eligibilityUnknown = true
	}
	kind := models.PortfolioBlockerHardDependency
	reason := "unresolved_dependency:" + hard.predecessorKey
	if relationshipType == models.EntityRelBlocks {
		kind = models.PortfolioBlockerIncomingBlock
		reason = "blocked_by:" + hard.predecessorKey
	}
	state.eligibilityReasons[reason] = struct{}{}
	title := ""
	if epic := allEpics[hard.predecessorKey]; epic != nil {
		title = epic.Title
	}
	state.evidence.BlockedItems = append(state.evidence.BlockedItems, models.PortfolioBlockedItem{
		Kind: kind, EntityType: string(models.EntityTypeEpic), EntityKey: hard.predecessorKey,
		Title: title, Status: hard.predecessorStatus,
	})
}

func assemblePortfolioClaims(
	byKey map[string]*portfolioEpicAssembly,
	ownership map[string]string,
	claims []*models.EntityClaim,
	advice *models.PortfolioAdviceEnvelope,
) {
	for _, claim := range claims {
		if claim == nil {
			continue
		}
		owner := portfolioClaimOwner(claim, byKey, ownership)
		state := byKey[owner]
		if state == nil {
			continue
		}
		work, valid := portfolioActiveWorkFromClaim(claim)
		if !valid {
			advice.EvidenceComplete = false
			advice.Warnings = append(advice.Warnings, models.PortfolioWarning{
				Code:     models.PortfolioWarningClaimStateUnavailable,
				Message:  "an active claim had invalid progress and was excluded from portfolio evidence",
				EpicKeys: []string{owner},
			})
			continue
		}
		state.evidence.ActiveWork = append(state.evidence.ActiveWork, work)
	}
}

func portfolioClaimOwner(
	claim *models.EntityClaim,
	byKey map[string]*portfolioEpicAssembly,
	ownership map[string]string,
) string {
	switch claim.EntityType {
	case string(models.EntityTypeEpic):
		if byKey[claim.EntityKey] != nil {
			return claim.EntityKey
		}
	case string(models.EntityTypeFeature), string(models.EntityTypeTask):
		return ownership[claim.EntityKey]
	}
	return ""
}

func portfolioActiveWorkFromClaim(claim *models.EntityClaim) (models.PortfolioActiveWork, bool) {
	if claim.Progress != nil && (*claim.Progress < 0 || *claim.Progress > 1 || math.IsNaN(*claim.Progress) || math.IsInf(*claim.Progress, 0)) {
		return models.PortfolioActiveWork{}, false
	}
	var progress *float64
	if claim.Progress != nil {
		value := *claim.Progress
		progress = &value
	}
	return models.PortfolioActiveWork{
		EntityType: claim.EntityType, EntityKey: claim.EntityKey, ClaimedBy: claim.ClaimedBy,
		LastHeartbeat: claim.LastHeartbeat.UTC(), Progress: progress,
	}, true
}
