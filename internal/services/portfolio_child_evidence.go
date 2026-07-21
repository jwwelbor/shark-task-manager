package services

import (
	"math"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	portfoliorepo "github.com/jwwelbor/shark-task-manager/internal/repository/portfolio"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
)

type portfolioEpicAssembly struct {
	evidence                models.PortfolioEpicEvidence
	eligibilityReasons      map[string]struct{}
	eligibilityUnknown      bool
	nonTerminalFeatureCount int
	blockedFeatureCount     int
}

func buildPortfolioEpicStates(
	epics []*models.Epic,
	epicWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) ([]*portfolioEpicAssembly, map[string]*portfolioEpicAssembly, map[string]*models.Epic) {
	states := make([]*portfolioEpicAssembly, 0, len(epics))
	byKey := make(map[string]*portfolioEpicAssembly, len(epics))
	allEpics := make(map[string]*models.Epic, len(epics))
	for _, epic := range epics {
		if epic == nil {
			continue
		}
		allEpics[epic.Key] = epic
		status := string(epic.Status)
		validStatus := epicWorkflow.IsValidStatus(status)
		if validStatus && epicWorkflow.IsTerminalStatus(status) {
			continue
		}

		var businessValue *string
		if epic.BusinessValue != nil {
			value := string(*epic.BusinessValue)
			businessValue = &value
		}
		state := &portfolioEpicAssembly{
			evidence: models.PortfolioEpicEvidence{
				Key:                epic.Key,
				Title:              epic.Title,
				Status:             status,
				Priority:           string(epic.Priority),
				BusinessValue:      businessValue,
				EligibilityReasons: []string{},
				BlockedItems:       []models.PortfolioBlockedItem{},
				ActiveWork:         []models.PortfolioActiveWork{},
			},
			eligibilityReasons: make(map[string]struct{}),
		}
		if !validStatus {
			state.eligibilityUnknown = true
			advice.EvidenceComplete = false
			advice.Warnings = append(advice.Warnings, portfolioUnknownStatusWarning("epic", epic.Key, status, epic.Key))
		} else if epicWorkflow.IsBlockedStatus(status) {
			state.eligibilityReasons["epic_workflow_blocked"] = struct{}{}
		}
		states = append(states, state)
		byKey[epic.Key] = state
	}
	return states, byKey, allEpics
}

func assemblePortfolioChildren(
	states []*portfolioEpicAssembly,
	byKey map[string]*portfolioEpicAssembly,
	children []portfoliorepo.ChildStateRow,
	ownership map[string]string,
	featureWorkflow *workflow.Service,
	taskWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) {
	featureProgress := make(map[string][]repository.FeatureProgressData)
	for _, child := range children {
		state, ok := byKey[child.EpicKey]
		if !ok {
			continue
		}
		assemblePortfolioChild(
			state, child, ownership, featureProgress, featureWorkflow, taskWorkflow, advice,
		)
	}

	for _, state := range states {
		finalizePortfolioChildEvidence(state, featureProgress[state.evidence.Key])
	}
}

func assemblePortfolioChild(
	state *portfolioEpicAssembly,
	child portfoliorepo.ChildStateRow,
	ownership map[string]string,
	featureProgress map[string][]repository.FeatureProgressData,
	featureWorkflow *workflow.Service,
	taskWorkflow *workflow.Service,
	advice *models.PortfolioAdviceEnvelope,
) {
	ownership[child.EntityKey] = child.EpicKey
	classifier := taskWorkflow
	if child.EntityType == models.EntityTypeFeature {
		classifier = featureWorkflow
	}
	recordPortfolioFeatureProgress(child, featureProgress)
	validStatus := classifier.IsValidStatus(child.Status)
	if !validStatus {
		markPortfolioChildStatusUnknown(state, child, advice)
	}
	blocked := validStatus && classifier.IsBlockedStatus(child.Status)
	if blocked {
		state.evidence.BlockedItems = append(state.evidence.BlockedItems, models.PortfolioBlockedItem{
			Kind: models.PortfolioBlockerWorkflowBlocked, EntityType: string(child.EntityType),
			EntityKey: child.EntityKey, Title: child.Title, Status: child.Status,
		})
	}
	recordPortfolioDirectFeatureState(state, child, classifier, validStatus, blocked)
}

func recordPortfolioFeatureProgress(
	child portfoliorepo.ChildStateRow,
	featureProgress map[string][]repository.FeatureProgressData,
) {
	if child.EntityType != models.EntityTypeFeature {
		return
	}
	progress := 0.0
	if child.ProgressPct != nil {
		progress = *child.ProgressPct
	}
	featureProgress[child.EpicKey] = append(featureProgress[child.EpicKey], repository.FeatureProgressData{
		Status: child.Status, ProgressPct: progress,
	})
}

func markPortfolioChildStatusUnknown(
	state *portfolioEpicAssembly,
	child portfoliorepo.ChildStateRow,
	advice *models.PortfolioAdviceEnvelope,
) {
	state.eligibilityUnknown = true
	advice.EvidenceComplete = false
	advice.Warnings = append(advice.Warnings, portfolioUnknownStatusWarning(
		string(child.EntityType), child.EntityKey, child.Status, child.EpicKey,
	))
}

func recordPortfolioDirectFeatureState(
	state *portfolioEpicAssembly,
	child portfoliorepo.ChildStateRow,
	classifier *workflow.Service,
	validStatus bool,
	blocked bool,
) {
	if child.EntityType != models.EntityTypeFeature || child.DirectParentKey != child.EpicKey {
		return
	}
	if validStatus && classifier.IsTerminalStatus(child.Status) {
		return
	}
	state.nonTerminalFeatureCount++
	if blocked {
		state.blockedFeatureCount++
	}
}

func finalizePortfolioChildEvidence(state *portfolioEpicAssembly, progressRows []repository.FeatureProgressData) {
	progress := calculateEpicProgress(progressRows)
	progress = math.Max(0, math.Min(100, progress))
	state.evidence.ProgressPct = math.Round(progress*100) / 100
	if state.nonTerminalFeatureCount > 0 && state.nonTerminalFeatureCount == state.blockedFeatureCount {
		state.eligibilityReasons["all_direct_features_blocked"] = struct{}{}
	}
}
