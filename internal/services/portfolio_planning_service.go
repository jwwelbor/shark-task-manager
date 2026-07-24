package services

import "github.com/jwwelbor/shark-task-manager/internal/models"

// PortfolioPlan is the service-layer selection result for bare shark plan.
// RootKeys contains only epic keys and preserves portfolio ordering.
type PortfolioPlan struct {
	RootKeys        []string
	SelectionReason string
	PauseReason     string
}

// PortfolioPlanningService selects workflow-eligible, unclaimed epic roots
// from the first executable hard-dependency layer.
type PortfolioPlanningService struct{}

// NewPortfolioPlanningService constructs a portfolio next planner.
func NewPortfolioPlanningService() *PortfolioPlanningService {
	return &PortfolioPlanningService{}
}

// Plan selects epic roots only when the portfolio evidence and ordering are
// usable. It respects hard dependency order first, then explicit roadmap order,
// then the stored high/medium/low priority tier. A tie remains a parallel
// candidate set; the service never resolves candidates into feature or task
// dispatches.
func (s *PortfolioPlanningService) Plan(advice *models.PortfolioAdviceEnvelope) PortfolioPlan {
	if advice == nil {
		return pausedPortfolioPlan("portfolio_evidence_unavailable")
	}
	if !advice.EvidenceComplete {
		return pausedPortfolioPlan("portfolio_evidence_incomplete")
	}
	if portfolioOrderingUnavailable(advice.Ordering) {
		return pausedPortfolioPlan("portfolio_ordering_unavailable")
	}

	epicsByKey := make(map[string]models.PortfolioEpicEvidence, len(advice.Epics))
	for _, epic := range advice.Epics {
		epicsByKey[epic.Key] = epic
	}

	for _, dependencyLayer := range advice.Ordering.DependencyLayers {
		keys := make([]string, 0, len(dependencyLayer))
		for _, key := range dependencyLayer {
			epic, ok := epicsByKey[key]
			if !ok || epic.Eligibility != models.PortfolioEligibilityEligible || len(epic.ActiveWork) > 0 {
				continue
			}
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			orderedKeys, ordered := selectEarliestRoadmapLayer(keys, advice.Ordering.RoadmapLayers)
			priorityKeys, prioritized := selectHighestEpicPriority(orderedKeys, epicsByKey)

			reason := "dependency_order"
			if ordered {
				reason = "roadmap_order"
			}
			if prioritized {
				reason = "priority"
			}
			if len(priorityKeys) > 1 {
				reason = "parallel_tie"
			}
			return PortfolioPlan{
				RootKeys:        priorityKeys,
				SelectionReason: reason,
			}
		}
	}

	return pausedPortfolioPlan("no_eligible_epic")
}

func pausedPortfolioPlan(reason string) PortfolioPlan {
	return PortfolioPlan{
		RootKeys:    []string{},
		PauseReason: reason,
	}
}

func portfolioOrderingUnavailable(ordering models.PortfolioOrdering) bool {
	for _, warning := range ordering.Warnings {
		switch warning.Code {
		case models.PortfolioWarningHardOrderCycle,
			models.PortfolioWarningRoadmapOrderCycle,
			models.PortfolioWarningContradictoryOrder:
			return true
		}
	}
	return false
}

func selectEarliestRoadmapLayer(keys []string, roadmapLayers [][]string) ([]string, bool) {
	ranks := make(map[string]int, len(keys))
	for layerIndex, layer := range roadmapLayers {
		for _, key := range layer {
			ranks[key] = layerIndex
		}
	}

	bestRank := len(roadmapLayers)
	for _, key := range keys {
		rank, ok := ranks[key]
		if !ok {
			return keys, false
		}
		if rank < bestRank {
			bestRank = rank
		}
	}

	selected := make([]string, 0, len(keys))
	for _, key := range keys {
		if ranks[key] == bestRank {
			selected = append(selected, key)
		}
	}
	return selected, len(selected) < len(keys)
}

func selectHighestEpicPriority(
	keys []string,
	epicsByKey map[string]models.PortfolioEpicEvidence,
) ([]string, bool) {
	bestRank := 4
	for _, key := range keys {
		rank := epicPriorityRank(epicsByKey[key].Priority)
		if rank < bestRank {
			bestRank = rank
		}
	}

	selected := make([]string, 0, len(keys))
	for _, key := range keys {
		if epicPriorityRank(epicsByKey[key].Priority) == bestRank {
			selected = append(selected, key)
		}
	}
	return selected, len(selected) < len(keys)
}

func epicPriorityRank(priority string) int {
	switch models.Priority(priority) {
	case models.PriorityHigh:
		return 1
	case models.PriorityMedium:
		return 2
	case models.PriorityLow:
		return 3
	default:
		return 4
	}
}
