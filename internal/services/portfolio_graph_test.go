package services

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

const portfolioGraphMaxCyclomaticComplexity = 10

func TestPortfolioGraphFunctionsStayWithinCyclomaticComplexityLimit(t *testing.T) {
	t.Parallel()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() could not locate portfolio graph test file")
	}
	graphFile := filepath.Join(filepath.Dir(testFile), "portfolio_graph.go")
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, graphFile, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", graphFile, err)
	}

	functionCount := 0
	for _, declaration := range parsed.Decls {
		function, isFunction := declaration.(*ast.FuncDecl)
		if !isFunction {
			continue
		}
		functionCount++
		complexity := portfolioGraphCyclomaticComplexity(function)
		if complexity > portfolioGraphMaxCyclomaticComplexity {
			t.Errorf(
				"%s has cyclomatic complexity %d; maximum is %d",
				function.Name.Name,
				complexity,
				portfolioGraphMaxCyclomaticComplexity,
			)
		}
	}
	if functionCount == 0 {
		t.Fatal("portfolio_graph.go contains no functions; complexity guard did not inspect anything")
	}
}

func portfolioGraphCyclomaticComplexity(function ast.Node) int {
	visitor := &portfolioGraphComplexityVisitor{complexity: 1}
	ast.Walk(visitor, function)
	return visitor.complexity
}

type portfolioGraphComplexityVisitor struct {
	complexity int
}

// Visit mirrors gocyclo's complexity definition without requiring the binary at test runtime.
func (v *portfolioGraphComplexityVisitor) Visit(node ast.Node) ast.Visitor {
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

func TestNormalizePortfolioEdges_DirectionSatisfactionAndCandidateFiltering(t *testing.T) {
	epics := portfolioEpics("E01", "E02", "E03", "E04", "E05", "E06")
	relationships := []models.PortfolioEpicRelationship{
		portfolioRelationship("E02", "E01", models.EntityRelDependsOn, boolPointer(false)),
		portfolioRelationship("E03", "E01", models.EntityRelDependsOn, nil),
		portfolioRelationship("E03", "E04", models.EntityRelBlocks, nil),
		portfolioRelationship("E05", "E06", models.EntityRelFollows, boolPointer(true)),
		portfolioRelationship("E04", "E01", models.EntityRelDependsOn, boolPointer(true)),
		portfolioRelationship("E06", "E02", models.EntityRelBlocks, boolPointer(true)),
		portfolioRelationship("E99", "E01", models.EntityRelBlocks, boolPointer(false)),
		portfolioRelationship("E01", "E02", models.EntityRelRelatedTo, nil),
	}

	got := normalizePortfolioEdges(epics, relationships)
	want := []portfolioGraphEdge{
		{
			Before: "E01", After: "E02", Hard: true,
			ContributingTypes: []models.EntityRelationshipType{models.EntityRelDependsOn},
		},
		{
			Before: "E01", After: "E03", Hard: true,
			ContributingTypes: []models.EntityRelationshipType{models.EntityRelDependsOn},
		},
		{
			Before: "E03", After: "E04", Hard: true,
			ContributingTypes: []models.EntityRelationshipType{models.EntityRelBlocks},
		},
		{
			Before: "E06", After: "E05", Hard: false,
			ContributingTypes: []models.EntityRelationshipType{models.EntityRelFollows},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizePortfolioEdges() = %#v, want %#v", got, want)
	}
}

func TestNormalizePortfolioEdges_DeduplicatesAdjacencyAndAggregatesTypes(t *testing.T) {
	epics := portfolioEpics("E01", "E02")
	relationships := []models.PortfolioEpicRelationship{
		portfolioRelationship("E02", "E01", models.EntityRelDependsOn, boolPointer(false)),
		portfolioRelationship("E02", "E01", models.EntityRelDependsOn, boolPointer(false)),
		portfolioRelationship("E01", "E02", models.EntityRelBlocks, boolPointer(false)),
		portfolioRelationship("E02", "E01", models.EntityRelFollows, nil),
		portfolioRelationship("E02", "E01", models.EntityRelFollows, boolPointer(true)),
	}

	edges := normalizePortfolioEdges(epics, relationships)
	wantEdges := []portfolioGraphEdge{{
		Before: "E01", After: "E02", Hard: true,
		ContributingTypes: []models.EntityRelationshipType{
			models.EntityRelBlocks,
			models.EntityRelDependsOn,
			models.EntityRelFollows,
		},
	}}
	if !reflect.DeepEqual(edges, wantEdges) {
		t.Fatalf("normalizePortfolioEdges() = %#v, want %#v", edges, wantEdges)
	}

	ordering := analyzePortfolioGraph(epics, relationships)
	wantOrdering := models.PortfolioOrdering{
		DependencyLayers: [][]string{{"E01"}, {"E02"}},
		RoadmapLayers:    [][]string{{"E01"}, {"E02"}},
		UnlayeredEpics:   []string{},
		Warnings:         []models.PortfolioWarning{},
	}
	if !reflect.DeepEqual(ordering, wantOrdering) {
		t.Fatalf("analyzePortfolioGraph() = %#v, want %#v", ordering, wantOrdering)
	}
}

func TestAnalyzePortfolioGraph_EmptyAndIndependentCandidates(t *testing.T) {
	t.Run("empty returns allocated slices", func(t *testing.T) {
		got := analyzePortfolioGraph(nil, nil)
		if got.DependencyLayers == nil || got.RoadmapLayers == nil || got.UnlayeredEpics == nil || got.Warnings == nil {
			t.Fatalf("analyzePortfolioGraph(nil, nil) returned nil collection: %#v", got)
		}
		if len(got.DependencyLayers) != 0 || len(got.RoadmapLayers) != 0 || len(got.UnlayeredEpics) != 0 || len(got.Warnings) != 0 {
			t.Fatalf("analyzePortfolioGraph(nil, nil) = %#v, want allocated empty ordering", got)
		}
	})

	t.Run("independent candidates share one lexical layer", func(t *testing.T) {
		got := analyzePortfolioGraph(portfolioEpics("E10", "E02", "E01", "E02"), nil)
		want := models.PortfolioOrdering{
			DependencyLayers: [][]string{{"E01", "E02", "E10"}},
			RoadmapLayers:    [][]string{{"E01", "E02", "E10"}},
			UnlayeredEpics:   []string{},
			Warnings:         []models.PortfolioWarning{},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("analyzePortfolioGraph() = %#v, want %#v", got, want)
		}
	})
}

func TestAnalyzePortfolioGraph_FollowsChainAndInputPermutations(t *testing.T) {
	epics := portfolioEpics("E10", "E02", "E01", "E03")
	relationships := []models.PortfolioEpicRelationship{
		portfolioRelationship("E10", "E02", models.EntityRelFollows, nil),
		portfolioRelationship("E02", "E01", models.EntityRelFollows, nil),
		portfolioRelationship("E03", "E02", models.EntityRelDependsOn, boolPointer(false)),
		portfolioRelationship("E10", "E02", models.EntityRelFollows, nil),
	}
	want := models.PortfolioOrdering{
		DependencyLayers: [][]string{{"E01", "E02", "E10"}, {"E03"}},
		RoadmapLayers:    [][]string{{"E01"}, {"E02"}, {"E03", "E10"}},
		UnlayeredEpics:   []string{},
		Warnings:         []models.PortfolioWarning{},
	}

	for seed := 0; seed < 32; seed++ {
		shuffledEpics := append([]models.PortfolioEpicEvidence(nil), epics...)
		shuffledRelationships := append([]models.PortfolioEpicRelationship(nil), relationships...)
		rng := rand.New(rand.NewSource(int64(seed)))
		rng.Shuffle(len(shuffledEpics), func(i, j int) {
			shuffledEpics[i], shuffledEpics[j] = shuffledEpics[j], shuffledEpics[i]
		})
		rng.Shuffle(len(shuffledRelationships), func(i, j int) {
			shuffledRelationships[i], shuffledRelationships[j] = shuffledRelationships[j], shuffledRelationships[i]
		})

		got := analyzePortfolioGraph(shuffledEpics, shuffledRelationships)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d: analyzePortfolioGraph() = %#v, want %#v", seed, got, want)
		}
	}
}

func TestAnalyzePortfolioGraph_HardCyclesReturnPartialLayersAndTerminate(t *testing.T) {
	tests := []struct {
		name          string
		epics         []models.PortfolioEpicEvidence
		relationships []models.PortfolioEpicRelationship
		wantLayers    [][]string
		wantUnlayered []string
		wantCycleKeys []string
	}{
		{
			name:  "two node cycle includes downstream Kahn remainder",
			epics: portfolioEpics("E04", "E02", "E03", "E01"),
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E01", "E02", models.EntityRelDependsOn, boolPointer(false)),
				portfolioRelationship("E02", "E01", models.EntityRelDependsOn, boolPointer(false)),
				portfolioRelationship("E04", "E02", models.EntityRelDependsOn, boolPointer(false)),
			},
			wantLayers:    [][]string{{"E03"}},
			wantUnlayered: []string{"E01", "E02", "E04"},
			wantCycleKeys: []string{"E01", "E02", "E04"},
		},
		{
			name:  "self cycle",
			epics: portfolioEpics("E02", "E01"),
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E01", "E01", models.EntityRelDependsOn, nil),
			},
			wantLayers:    [][]string{{"E02"}},
			wantUnlayered: []string{"E01"},
			wantCycleKeys: []string{"E01"},
		},
		{
			name:  "larger and disjoint cycles",
			epics: portfolioEpics("E06", "E05", "E04", "E03", "E02", "E01"),
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E01", "E02", models.EntityRelDependsOn, nil),
				portfolioRelationship("E02", "E01", models.EntityRelDependsOn, nil),
				portfolioRelationship("E03", "E04", models.EntityRelDependsOn, nil),
				portfolioRelationship("E04", "E05", models.EntityRelDependsOn, nil),
				portfolioRelationship("E05", "E03", models.EntityRelDependsOn, nil),
			},
			wantLayers:    [][]string{{"E06"}},
			wantUnlayered: []string{"E01", "E02", "E03", "E04", "E05"},
			wantCycleKeys: []string{"E01", "E02", "E03", "E04", "E05"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePortfolioGraphWithTimeout(t, tt.epics, tt.relationships)
			if !reflect.DeepEqual(got.DependencyLayers, tt.wantLayers) {
				t.Errorf("DependencyLayers = %#v, want %#v", got.DependencyLayers, tt.wantLayers)
			}
			if !reflect.DeepEqual(got.RoadmapLayers, tt.wantLayers) {
				t.Errorf("RoadmapLayers = %#v, want %#v", got.RoadmapLayers, tt.wantLayers)
			}
			if !reflect.DeepEqual(got.UnlayeredEpics, tt.wantUnlayered) {
				t.Errorf("UnlayeredEpics = %#v, want %#v", got.UnlayeredEpics, tt.wantUnlayered)
			}
			warning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningHardOrderCycle)
			if !reflect.DeepEqual(warning.EpicKeys, tt.wantCycleKeys) {
				t.Errorf("hard-cycle keys = %#v, want %#v", warning.EpicKeys, tt.wantCycleKeys)
			}
			if hasPortfolioWarning(got.Warnings, models.PortfolioWarningRoadmapOrderCycle) {
				t.Errorf("hard-only graph unexpectedly returned ROADMAP_ORDER_CYCLE: %#v", got.Warnings)
			}
		})
	}
}

func TestAnalyzePortfolioGraph_RoadmapCycleKeepsHardLayersAndClassifiesActualSoftCycle(t *testing.T) {
	epics := portfolioEpics("E03", "E02", "E01")
	relationships := []models.PortfolioEpicRelationship{
		portfolioRelationship("E03", "E01", models.EntityRelDependsOn, boolPointer(false)),
		portfolioRelationship("E02", "E01", models.EntityRelFollows, nil),
		portfolioRelationship("E01", "E02", models.EntityRelFollows, nil),
	}

	got := analyzePortfolioGraphWithTimeout(t, epics, relationships)
	if want := [][]string{{"E01", "E02"}, {"E03"}}; !reflect.DeepEqual(got.DependencyLayers, want) {
		t.Errorf("DependencyLayers = %#v, want %#v", got.DependencyLayers, want)
	}
	if len(got.RoadmapLayers) != 0 {
		t.Errorf("RoadmapLayers = %#v, want empty partial layers", got.RoadmapLayers)
	}
	if want := []string{"E01", "E02", "E03"}; !reflect.DeepEqual(got.UnlayeredEpics, want) {
		t.Errorf("UnlayeredEpics = %#v, want full Kahn remainder %#v", got.UnlayeredEpics, want)
	}
	roadmapWarning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningRoadmapOrderCycle)
	if want := []string{"E01", "E02"}; !reflect.DeepEqual(roadmapWarning.EpicKeys, want) {
		t.Errorf("roadmap-cycle keys = %#v, want actual soft-cycle participants %#v", roadmapWarning.EpicKeys, want)
	}
	if hasPortfolioWarning(got.Warnings, models.PortfolioWarningHardOrderCycle) {
		t.Errorf("follows-only cycle unexpectedly returned HARD_ORDER_CYCLE: %#v", got.Warnings)
	}
}

func TestAnalyzePortfolioGraph_CoexistingHardAndSoftCyclesReturnBothDiagnostics(t *testing.T) {
	epics := portfolioEpics("E04", "E02", "E03", "E01")
	relationships := []models.PortfolioEpicRelationship{
		portfolioRelationship("E01", "E02", models.EntityRelDependsOn, nil),
		portfolioRelationship("E02", "E01", models.EntityRelDependsOn, nil),
		portfolioRelationship("E03", "E04", models.EntityRelFollows, nil),
		portfolioRelationship("E04", "E03", models.EntityRelFollows, nil),
	}

	got := analyzePortfolioGraphWithTimeout(t, epics, relationships)
	hardWarning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningHardOrderCycle)
	roadmapWarning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningRoadmapOrderCycle)
	if want := []string{"E01", "E02"}; !reflect.DeepEqual(hardWarning.EpicKeys, want) {
		t.Errorf("hard-cycle keys = %#v, want %#v", hardWarning.EpicKeys, want)
	}
	if want := []string{"E03", "E04"}; !reflect.DeepEqual(roadmapWarning.EpicKeys, want) {
		t.Errorf("roadmap-cycle keys = %#v, want %#v", roadmapWarning.EpicKeys, want)
	}

	wantCodes := []models.PortfolioWarningCode{
		models.PortfolioWarningContradictoryOrder,
		models.PortfolioWarningContradictoryOrder,
		models.PortfolioWarningHardOrderCycle,
		models.PortfolioWarningRoadmapOrderCycle,
	}
	if gotCodes := portfolioWarningCodes(got.Warnings); !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Errorf("warning codes = %#v, want stable code order %#v", gotCodes, wantCodes)
	}
}

func TestAnalyzePortfolioGraph_ContradictionsNameSortedContributingTypes(t *testing.T) {
	tests := []struct {
		name          string
		relationships []models.PortfolioEpicRelationship
		wantTypes     string
	}{
		{
			name: "hard against hard",
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E01", "E02", models.EntityRelDependsOn, nil),
				portfolioRelationship("E01", "E02", models.EntityRelBlocks, nil),
				portfolioRelationship("E01", "E02", models.EntityRelBlocks, nil),
			},
			wantTypes: "blocks, depends_on",
		},
		{
			name: "hard against soft",
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E01", "E02", models.EntityRelBlocks, nil),
				portfolioRelationship("E01", "E02", models.EntityRelFollows, nil),
			},
			wantTypes: "blocks, follows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePortfolioGraph(portfolioEpics("E02", "E01"), tt.relationships)
			warning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningContradictoryOrder)
			if want := []string{"E01", "E02"}; !reflect.DeepEqual(warning.EpicKeys, want) {
				t.Errorf("EpicKeys = %#v, want %#v", warning.EpicKeys, want)
			}
			if !strings.Contains(warning.Message, tt.wantTypes) {
				t.Errorf("Message = %q, want sorted contributing types %q", warning.Message, tt.wantTypes)
			}
			if countPortfolioWarnings(got.Warnings, models.PortfolioWarningContradictoryOrder) != 1 {
				t.Errorf("warnings = %#v, want exactly one contradiction", got.Warnings)
			}
		})
	}
}

func TestAnalyzePortfolioGraph_MissingOrderingUsesEligibleHardRootsAndRoadmapReachability(t *testing.T) {
	tests := []struct {
		name          string
		epics         []models.PortfolioEpicEvidence
		relationships []models.PortfolioEpicRelationship
		wantMissing   bool
		wantRoadmap   [][]string
	}{
		{
			name: "eligible incomparable roots",
			epics: []models.PortfolioEpicEvidence{
				portfolioEpic("E02", models.PortfolioEligibilityEligible),
				portfolioEpic("E01", models.PortfolioEligibilityEligible),
			},
			wantMissing: true,
			wantRoadmap: [][]string{{"E01", "E02"}},
		},
		{
			name: "direct roadmap path supplies order",
			epics: []models.PortfolioEpicEvidence{
				portfolioEpic("E02", models.PortfolioEligibilityEligible),
				portfolioEpic("E01", models.PortfolioEligibilityEligible),
			},
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E02", "E01", models.EntityRelFollows, nil),
			},
			wantRoadmap: [][]string{{"E01"}, {"E02"}},
		},
		{
			name: "transitive roadmap path supplies order",
			epics: []models.PortfolioEpicEvidence{
				portfolioEpic("E03", models.PortfolioEligibilityIneligible),
				portfolioEpic("E02", models.PortfolioEligibilityEligible),
				portfolioEpic("E01", models.PortfolioEligibilityEligible),
			},
			relationships: []models.PortfolioEpicRelationship{
				portfolioRelationship("E03", "E01", models.EntityRelFollows, nil),
				portfolioRelationship("E02", "E03", models.EntityRelFollows, nil),
			},
			wantRoadmap: [][]string{{"E01"}, {"E03"}, {"E02"}},
		},
		{
			name: "ineligible and unknown roots are excluded",
			epics: []models.PortfolioEpicEvidence{
				portfolioEpic("E03", models.PortfolioEligibilityUnknown),
				portfolioEpic("E02", models.PortfolioEligibilityIneligible),
				portfolioEpic("E01", models.PortfolioEligibilityEligible),
			},
			wantRoadmap: [][]string{{"E01", "E02", "E03"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzePortfolioGraph(tt.epics, tt.relationships)
			if want := []string{"E01", "E02"}; tt.wantMissing {
				warning := requirePortfolioWarning(t, got.Warnings, models.PortfolioWarningMissingOrdering)
				if !reflect.DeepEqual(warning.EpicKeys, want) {
					t.Errorf("missing-order keys = %#v, want %#v", warning.EpicKeys, want)
				}
			} else if hasPortfolioWarning(got.Warnings, models.PortfolioWarningMissingOrdering) {
				t.Errorf("unexpected MISSING_ORDERING warning: %#v", got.Warnings)
			}
			if !reflect.DeepEqual(got.RoadmapLayers, tt.wantRoadmap) {
				t.Errorf("RoadmapLayers = %#v, want %#v", got.RoadmapLayers, tt.wantRoadmap)
			}
		})
	}
}

func analyzePortfolioGraphWithTimeout(
	t *testing.T,
	epics []models.PortfolioEpicEvidence,
	relationships []models.PortfolioEpicRelationship,
) models.PortfolioOrdering {
	t.Helper()
	result := make(chan models.PortfolioOrdering, 1)
	go func() {
		result <- analyzePortfolioGraph(epics, relationships)
	}()
	select {
	case ordering := <-result:
		return ordering
	case <-time.After(100 * time.Millisecond):
		t.Fatal("analyzePortfolioGraph() did not terminate within 100ms")
		return models.PortfolioOrdering{}
	}
}

func portfolioEpics(keys ...string) []models.PortfolioEpicEvidence {
	result := make([]models.PortfolioEpicEvidence, 0, len(keys))
	for _, key := range keys {
		result = append(result, portfolioEpic(key, models.PortfolioEligibilityUnknown))
	}
	return result
}

func portfolioEpic(key string, eligibility models.PortfolioEligibility) models.PortfolioEpicEvidence {
	return models.PortfolioEpicEvidence{Key: key, Eligibility: eligibility}
}

func portfolioRelationship(
	fromKey string,
	toKey string,
	relationshipType models.EntityRelationshipType,
	satisfied *bool,
) models.PortfolioEpicRelationship {
	return models.PortfolioEpicRelationship{
		FromKey:          fromKey,
		ToKey:            toKey,
		RelationshipType: relationshipType,
		Satisfied:        satisfied,
	}
}

func requirePortfolioWarning(
	t *testing.T,
	warnings []models.PortfolioWarning,
	code models.PortfolioWarningCode,
) models.PortfolioWarning {
	t.Helper()
	for _, warning := range warnings {
		if warning.Code == code {
			return warning
		}
	}
	t.Fatalf("warnings = %#v, want code %s", warnings, code)
	return models.PortfolioWarning{}
}

func hasPortfolioWarning(warnings []models.PortfolioWarning, code models.PortfolioWarningCode) bool {
	return countPortfolioWarnings(warnings, code) > 0
}

func countPortfolioWarnings(warnings []models.PortfolioWarning, code models.PortfolioWarningCode) int {
	count := 0
	for _, warning := range warnings {
		if warning.Code == code {
			count++
		}
	}
	return count
}

func portfolioWarningCodes(warnings []models.PortfolioWarning) []models.PortfolioWarningCode {
	result := make([]models.PortfolioWarningCode, 0, len(warnings))
	for _, warning := range warnings {
		result = append(result, warning.Code)
	}
	return result
}

func boolPointer(value bool) *bool { return &value }
