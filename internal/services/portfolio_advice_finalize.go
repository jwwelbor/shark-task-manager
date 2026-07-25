package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
)

func finalizePortfolioEpics(states []*portfolioEpicAssembly, advice *models.PortfolioAdviceEnvelope) {
	advice.Epics = make([]models.PortfolioEpicEvidence, 0, len(states))
	for _, state := range states {
		if state.eligibilityUnknown {
			state.evidence.Eligibility = models.PortfolioEligibilityUnknown
			state.eligibilityReasons["evidence_incomplete"] = struct{}{}
		} else if len(state.eligibilityReasons) > 0 {
			state.evidence.Eligibility = models.PortfolioEligibilityIneligible
		} else {
			state.evidence.Eligibility = models.PortfolioEligibilityEligible
		}
		state.evidence.EligibilityReasons = sortedPortfolioReasonCodes(state.eligibilityReasons)
		sort.Slice(state.evidence.BlockedItems, func(i, j int) bool {
			left, right := state.evidence.BlockedItems[i], state.evidence.BlockedItems[j]
			if left.EntityType != right.EntityType {
				return left.EntityType < right.EntityType
			}
			if left.EntityKey != right.EntityKey {
				return left.EntityKey < right.EntityKey
			}
			return left.Kind < right.Kind
		})
		sort.Slice(state.evidence.ActiveWork, func(i, j int) bool {
			left, right := state.evidence.ActiveWork[i], state.evidence.ActiveWork[j]
			if left.EntityType != right.EntityType {
				return left.EntityType < right.EntityType
			}
			return left.EntityKey < right.EntityKey
		})
		advice.Epics = append(advice.Epics, state.evidence)
	}
	sort.Slice(advice.Epics, func(i, j int) bool {
		left, right := strings.ToUpper(advice.Epics[i].Key), strings.ToUpper(advice.Epics[j].Key)
		if left != right {
			return left < right
		}
		return advice.Epics[i].Key < advice.Epics[j].Key
	})
}

func portfolioKnownCandidateKeys(
	row portfoliorepo.EpicRelationshipRow,
	byKey map[string]*portfolioEpicAssembly,
) []string {
	set := make(map[string]struct{})
	if row.FromKey != nil && byKey[*row.FromKey] != nil {
		set[*row.FromKey] = struct{}{}
	}
	if row.ToKey != nil && byKey[*row.ToKey] != nil {
		set[*row.ToKey] = struct{}{}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func portfolioMarkUnknownRelationshipStatus(
	key string,
	status string,
	candidate bool,
	byKey map[string]*portfolioEpicAssembly,
	advice *models.PortfolioAdviceEnvelope,
) {
	if !candidate {
		return
	}
	byKey[key].eligibilityUnknown = true
	advice.EvidenceComplete = false
	if !portfolioHasUnknownStatusWarning(advice.Warnings, key, status) {
		advice.Warnings = append(advice.Warnings, portfolioUnknownStatusWarning("epic", key, status, key))
	}
}

func portfolioHasUnknownStatusWarning(warnings []models.PortfolioWarning, key, status string) bool {
	messageFragment := fmt.Sprintf("status %q", status)
	for _, warning := range warnings {
		if warning.Code == models.PortfolioWarningUnknownWorkflowStatus &&
			len(warning.EpicKeys) == 1 && warning.EpicKeys[0] == key && strings.Contains(warning.Message, messageFragment) {
			return true
		}
	}
	return false
}

func portfolioUnknownStatusWarning(entityType, entityKey, status, epicKey string) models.PortfolioWarning {
	return models.PortfolioWarning{
		Code:     models.PortfolioWarningUnknownWorkflowStatus,
		Message:  fmt.Sprintf("workflow status %q for %s %s is not configured", status, entityType, entityKey),
		EpicKeys: []string{epicKey},
	}
}

func sortedPortfolioReasonCodes(set map[string]struct{}) []string {
	reasons := make([]string, 0, len(set))
	for reason := range set {
		reasons = append(reasons, reason)
	}
	sort.Strings(reasons)
	return reasons
}

func portfolioAdviceContextError(ctx context.Context, dependencyErr error) error {
	if dependencyErr != nil && (errors.Is(dependencyErr, context.Canceled) || errors.Is(dependencyErr, context.DeadlineExceeded)) {
		return fmt.Errorf("assemble portfolio advice: %w", dependencyErr)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("assemble portfolio advice: %w", err)
	}
	return nil
}
