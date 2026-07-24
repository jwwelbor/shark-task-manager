package services

import (
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

func TestPortfolioPlanningServiceSelectsFirstExecutableUnclaimedLayer(t *testing.T) {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Epics = []models.PortfolioEpicEvidence{
		{
			Key: "E01", Priority: "medium",
			Eligibility: models.PortfolioEligibilityEligible, ActiveWork: []models.PortfolioActiveWork{},
		},
		{
			Key: "E02", Priority: "high",
			Eligibility: models.PortfolioEligibilityEligible,
			ActiveWork:  []models.PortfolioActiveWork{{EntityType: "feature", EntityKey: "E02-F01"}},
		},
		{
			Key: "E03", Priority: "high",
			Eligibility: models.PortfolioEligibilityIneligible, ActiveWork: []models.PortfolioActiveWork{},
		},
		{
			Key: "E04", Priority: "high",
			Eligibility: models.PortfolioEligibilityEligible, ActiveWork: []models.PortfolioActiveWork{},
		},
		{
			Key: "E05", Priority: "high",
			Eligibility: models.PortfolioEligibilityEligible, ActiveWork: []models.PortfolioActiveWork{},
		},
	}
	advice.Ordering.DependencyLayers = [][]string{{"E01", "E02", "E03"}, {"E04", "E05"}}
	advice.Ordering.RoadmapLayers = [][]string{{"E01", "E02", "E03"}, {"E04", "E05"}}
	advice.Ordering.Warnings = []models.PortfolioWarning{{
		Code: models.PortfolioWarningMissingOrdering,
	}}

	got := NewPortfolioPlanningService().Plan(advice)
	if !reflect.DeepEqual(got.RootKeys, []string{"E01"}) {
		t.Fatalf("RootKeys = %#v, want first executable unclaimed layer only", got.RootKeys)
	}
	if got.SelectionReason != "dependency_order" {
		t.Fatalf("SelectionReason = %q, want dependency_order", got.SelectionReason)
	}
}

func TestPortfolioPlanningServiceUsesRoadmapOrderBeforePriority(t *testing.T) {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Epics = []models.PortfolioEpicEvidence{
		{
			Key: "E01", Priority: "low",
			Eligibility: models.PortfolioEligibilityEligible,
			ActiveWork:  []models.PortfolioActiveWork{},
		},
		{
			Key: "E02", Priority: "high",
			Eligibility: models.PortfolioEligibilityEligible, ActiveWork: []models.PortfolioActiveWork{},
		},
	}
	advice.Ordering.DependencyLayers = [][]string{{"E01", "E02"}}
	advice.Ordering.RoadmapLayers = [][]string{{"E01"}, {"E02"}}

	got := NewPortfolioPlanningService().Plan(advice)
	if !reflect.DeepEqual(got.RootKeys, []string{"E01"}) {
		t.Fatalf("RootKeys = %#v, want explicit roadmap order before priority", got.RootKeys)
	}
	if got.SelectionReason != "roadmap_order" {
		t.Fatalf("SelectionReason = %q, want roadmap_order", got.SelectionReason)
	}
}

func TestPortfolioPlanningServiceUsesPriorityAndKeepsTiesParallel(t *testing.T) {
	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = true
	advice.Epics = []models.PortfolioEpicEvidence{
		{Key: "E01", Priority: "medium", Eligibility: models.PortfolioEligibilityEligible},
		{Key: "E02", Priority: "high", Eligibility: models.PortfolioEligibilityEligible},
		{Key: "E03", Priority: "high", Eligibility: models.PortfolioEligibilityEligible},
	}
	advice.Ordering.DependencyLayers = [][]string{{"E01", "E02", "E03"}}
	advice.Ordering.RoadmapLayers = [][]string{{"E01", "E02", "E03"}}

	got := NewPortfolioPlanningService().Plan(advice)
	if !reflect.DeepEqual(got.RootKeys, []string{"E02", "E03"}) {
		t.Fatalf("RootKeys = %#v, want highest-priority tie in stable order", got.RootKeys)
	}
	if got.SelectionReason != "parallel_tie" {
		t.Fatalf("SelectionReason = %q, want parallel_tie", got.SelectionReason)
	}
}

func TestPortfolioPlanningServiceFallsBackOnUnsafeEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.PortfolioAdviceEnvelope)
	}{
		{
			name: "incomplete evidence",
			mutate: func(advice *models.PortfolioAdviceEnvelope) {
				advice.EvidenceComplete = false
			},
		},
		{
			name: "hard ordering cycle",
			mutate: func(advice *models.PortfolioAdviceEnvelope) {
				advice.Ordering.Warnings = []models.PortfolioWarning{{
					Code: models.PortfolioWarningHardOrderCycle,
				}}
			},
		},
		{
			name: "roadmap ordering cycle",
			mutate: func(advice *models.PortfolioAdviceEnvelope) {
				advice.Ordering.Warnings = []models.PortfolioWarning{{
					Code: models.PortfolioWarningRoadmapOrderCycle,
				}}
			},
		},
		{
			name: "contradictory ordering",
			mutate: func(advice *models.PortfolioAdviceEnvelope) {
				advice.Ordering.Warnings = []models.PortfolioWarning{{
					Code: models.PortfolioWarningContradictoryOrder,
				}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advice := models.NewPortfolioAdviceEnvelope()
			advice.EvidenceComplete = true
			advice.Epics = []models.PortfolioEpicEvidence{
				{Key: "E01", Eligibility: models.PortfolioEligibilityEligible},
				{Key: "E02", Eligibility: models.PortfolioEligibilityEligible},
			}
			advice.Ordering.DependencyLayers = [][]string{{"E01", "E02"}}
			advice.Ordering.RoadmapLayers = [][]string{{"E01", "E02"}}
			tt.mutate(advice)

			got := NewPortfolioPlanningService().Plan(advice)
			if len(got.RootKeys) != 0 {
				t.Fatalf("RootKeys = %#v, want paused selection", got.RootKeys)
			}
			if got.PauseReason == "" {
				t.Fatal("PauseReason is empty, want explicit stop reason")
			}
		})
	}
}
