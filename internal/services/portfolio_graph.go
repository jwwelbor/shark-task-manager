package services

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

type portfolioGraphEdge struct {
	Before            string
	After             string
	Hard              bool
	ContributingTypes []models.EntityRelationshipType
}

type portfolioGraphEdgeKey struct {
	before string
	after  string
}

type portfolioGraphEdgeAggregate struct {
	hard              bool
	contributingTypes map[models.EntityRelationshipType]struct{}
}

func analyzePortfolioGraph(
	epics []models.PortfolioEpicEvidence,
	relationships []models.PortfolioEpicRelationship,
) models.PortfolioOrdering {
	candidateKeys, eligibleCandidates := portfolioGraphCandidates(epics)
	edges := normalizePortfolioEdges(epics, relationships)
	hardAdjacency := buildPortfolioAdjacency(candidateKeys, edges, true)
	roadmapAdjacency := buildPortfolioAdjacency(candidateKeys, edges, false)

	dependencyLayers, hardRemainder := layerPortfolioGraph(candidateKeys, hardAdjacency)
	roadmapLayers, roadmapRemainder := layerPortfolioGraph(candidateKeys, roadmapAdjacency)
	warnings := make([]models.PortfolioWarning, 0)

	if len(hardRemainder) > 0 {
		keys := append([]string(nil), hardRemainder...)
		warnings = append(warnings, models.PortfolioWarning{
			Code:     models.PortfolioWarningHardOrderCycle,
			Message:  fmt.Sprintf("hard ordering cycle leaves epics unlayered: %s", strings.Join(keys, ", ")),
			EpicKeys: keys,
		})
	}
	if keys := portfolioSoftCycleKeys(candidateKeys, roadmapAdjacency, edges); len(keys) > 0 {
		warnings = append(warnings, models.PortfolioWarning{
			Code: models.PortfolioWarningRoadmapOrderCycle,
			Message: fmt.Sprintf(
				"roadmap ordering cycle contains soft precedence among epics: %s",
				strings.Join(keys, ", "),
			),
			EpicKeys: keys,
		})
	}
	warnings = append(warnings, contradictoryOrderWarnings(edges)...)
	warnings = append(
		warnings,
		missingOrderingWarnings(dependencyLayers, eligibleCandidates, roadmapAdjacency)...,
	)
	sortPortfolioWarnings(warnings)

	return models.PortfolioOrdering{
		DependencyLayers: dependencyLayers,
		RoadmapLayers:    roadmapLayers,
		UnlayeredEpics:   roadmapRemainder,
		Warnings:         warnings,
	}
}

func normalizePortfolioEdges(
	epics []models.PortfolioEpicEvidence,
	relationships []models.PortfolioEpicRelationship,
) []portfolioGraphEdge {
	candidateKeys, _ := portfolioGraphCandidates(epics)
	candidates := portfolioCandidateSet(candidateKeys)
	aggregates := collectPortfolioEdgeAggregates(candidates, relationships)
	return portfolioEdgesFromAggregates(aggregates)
}

func portfolioCandidateSet(candidateKeys []string) map[string]struct{} {
	candidates := make(map[string]struct{}, len(candidateKeys))
	for _, key := range candidateKeys {
		candidates[key] = struct{}{}
	}
	return candidates
}

func collectPortfolioEdgeAggregates(
	candidates map[string]struct{},
	relationships []models.PortfolioEpicRelationship,
) map[portfolioGraphEdgeKey]*portfolioGraphEdgeAggregate {
	aggregates := make(map[portfolioGraphEdgeKey]*portfolioGraphEdgeAggregate)
	for _, relationship := range relationships {
		if !portfolioRelationshipHasCandidates(candidates, relationship) {
			continue
		}
		key, hard, ok := normalizePortfolioRelationship(relationship)
		if !ok {
			continue
		}
		addPortfolioEdgeAggregate(aggregates, key, hard, relationship.RelationshipType)
	}
	return aggregates
}

func portfolioRelationshipHasCandidates(
	candidates map[string]struct{},
	relationship models.PortfolioEpicRelationship,
) bool {
	if _, ok := candidates[relationship.FromKey]; !ok {
		return false
	}
	_, ok := candidates[relationship.ToKey]
	return ok
}

func normalizePortfolioRelationship(
	relationship models.PortfolioEpicRelationship,
) (portfolioGraphEdgeKey, bool, bool) {
	switch relationship.RelationshipType {
	case models.EntityRelDependsOn:
		if portfolioHardRelationshipSatisfied(relationship) {
			return portfolioGraphEdgeKey{}, false, false
		}
		return portfolioGraphEdgeKey{
			before: relationship.ToKey,
			after:  relationship.FromKey,
		}, true, true
	case models.EntityRelBlocks:
		if portfolioHardRelationshipSatisfied(relationship) {
			return portfolioGraphEdgeKey{}, false, false
		}
		return portfolioGraphEdgeKey{
			before: relationship.FromKey,
			after:  relationship.ToKey,
		}, true, true
	case models.EntityRelFollows:
		return portfolioGraphEdgeKey{
			before: relationship.ToKey,
			after:  relationship.FromKey,
		}, false, true
	default:
		return portfolioGraphEdgeKey{}, false, false
	}
}

func portfolioHardRelationshipSatisfied(relationship models.PortfolioEpicRelationship) bool {
	return relationship.Satisfied != nil && *relationship.Satisfied
}

func addPortfolioEdgeAggregate(
	aggregates map[portfolioGraphEdgeKey]*portfolioGraphEdgeAggregate,
	key portfolioGraphEdgeKey,
	hard bool,
	relationshipType models.EntityRelationshipType,
) {
	aggregate, ok := aggregates[key]
	if !ok {
		aggregate = &portfolioGraphEdgeAggregate{
			contributingTypes: make(map[models.EntityRelationshipType]struct{}),
		}
		aggregates[key] = aggregate
	}
	aggregate.hard = aggregate.hard || hard
	aggregate.contributingTypes[relationshipType] = struct{}{}
}

func portfolioEdgesFromAggregates(
	aggregates map[portfolioGraphEdgeKey]*portfolioGraphEdgeAggregate,
) []portfolioGraphEdge {
	result := make([]portfolioGraphEdge, 0, len(aggregates))
	for key, aggregate := range aggregates {
		result = append(result, portfolioGraphEdge{
			Before:            key.before,
			After:             key.after,
			Hard:              aggregate.hard,
			ContributingTypes: sortedPortfolioRelationshipTypes(aggregate.contributingTypes),
		})
	}
	sort.Slice(result, func(i, j int) bool { return portfolioGraphEdgeLess(result[i], result[j]) })
	return result
}

func sortedPortfolioRelationshipTypes(
	typeSet map[models.EntityRelationshipType]struct{},
) []models.EntityRelationshipType {
	result := make([]models.EntityRelationshipType, 0, len(typeSet))
	for relationshipType := range typeSet {
		result = append(result, relationshipType)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func portfolioGraphEdgeLess(left, right portfolioGraphEdge) bool {
	if left.Before != right.Before {
		return left.Before < right.Before
	}
	return left.After < right.After
}

func portfolioGraphCandidates(epics []models.PortfolioEpicEvidence) ([]string, map[string]bool) {
	candidateSet := make(map[string]struct{}, len(epics))
	eligible := make(map[string]bool, len(epics))
	for _, epic := range epics {
		candidateSet[epic.Key] = struct{}{}
		if epic.Eligibility == models.PortfolioEligibilityEligible {
			eligible[epic.Key] = true
		}
	}
	keys := make([]string, 0, len(candidateSet))
	for key := range candidateSet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, eligible
}

func buildPortfolioAdjacency(
	candidateKeys []string,
	edges []portfolioGraphEdge,
	hardOnly bool,
) map[string][]string {
	sets := make(map[string]map[string]struct{}, len(candidateKeys))
	for _, key := range candidateKeys {
		sets[key] = make(map[string]struct{})
	}
	for _, edge := range edges {
		if hardOnly && !edge.Hard {
			continue
		}
		sets[edge.Before][edge.After] = struct{}{}
	}

	adjacency := make(map[string][]string, len(candidateKeys))
	for _, key := range candidateKeys {
		outgoing := make([]string, 0, len(sets[key]))
		for target := range sets[key] {
			outgoing = append(outgoing, target)
		}
		sort.Strings(outgoing)
		adjacency[key] = outgoing
	}
	return adjacency
}

func layerPortfolioGraph(candidateKeys []string, adjacency map[string][]string) ([][]string, []string) {
	layers := make([][]string, 0)
	indegree := portfolioIndegrees(candidateKeys, adjacency)
	frontier := portfolioZeroIndegreeKeys(candidateKeys, indegree)
	processed := make(map[string]bool, len(candidateKeys))
	for len(frontier) > 0 {
		layers = append(layers, append([]string(nil), frontier...))
		next := make([]string, 0)
		for _, key := range frontier {
			processed[key] = true
			next = decrementPortfolioTargets(adjacency[key], indegree, next)
		}
		sort.Strings(next)
		frontier = next
	}
	return layers, portfolioUnprocessedKeys(candidateKeys, processed)
}

func portfolioIndegrees(candidateKeys []string, adjacency map[string][]string) map[string]int {
	indegree := make(map[string]int, len(candidateKeys))
	for _, key := range candidateKeys {
		indegree[key] = 0
	}
	for _, key := range candidateKeys {
		for _, target := range adjacency[key] {
			indegree[target]++
		}
	}
	return indegree
}

func portfolioZeroIndegreeKeys(candidateKeys []string, indegree map[string]int) []string {
	result := make([]string, 0)
	for _, key := range candidateKeys {
		if indegree[key] == 0 {
			result = append(result, key)
		}
	}
	return result
}

func decrementPortfolioTargets(
	targets []string,
	indegree map[string]int,
	result []string,
) []string {
	for _, target := range targets {
		indegree[target]--
		if indegree[target] == 0 {
			result = append(result, target)
		}
	}
	return result
}

func portfolioUnprocessedKeys(candidateKeys []string, processed map[string]bool) []string {
	result := make([]string, 0)
	for _, key := range candidateKeys {
		if !processed[key] {
			result = append(result, key)
		}
	}
	return result
}

func contradictoryOrderWarnings(edges []portfolioGraphEdge) []models.PortfolioWarning {
	warnings := make([]models.PortfolioWarning, 0)
	edgeByKey := make(map[portfolioGraphEdgeKey]portfolioGraphEdge, len(edges))
	for _, edge := range edges {
		edgeByKey[portfolioGraphEdgeKey{before: edge.Before, after: edge.After}] = edge
	}

	for _, edge := range edges {
		if edge.Before >= edge.After {
			continue
		}
		reverse, ok := edgeByKey[portfolioGraphEdgeKey{before: edge.After, after: edge.Before}]
		if !ok {
			continue
		}
		typeSet := make(map[models.EntityRelationshipType]struct{})
		for _, relationshipType := range edge.ContributingTypes {
			typeSet[relationshipType] = struct{}{}
		}
		for _, relationshipType := range reverse.ContributingTypes {
			typeSet[relationshipType] = struct{}{}
		}
		types := make([]string, 0, len(typeSet))
		for relationshipType := range typeSet {
			types = append(types, string(relationshipType))
		}
		sort.Strings(types)
		keys := []string{edge.Before, edge.After}
		warnings = append(warnings, models.PortfolioWarning{
			Code: models.PortfolioWarningContradictoryOrder,
			Message: fmt.Sprintf(
				"contradictory ordering between %s and %s from relationship types: %s",
				keys[0],
				keys[1],
				strings.Join(types, ", "),
			),
			EpicKeys: keys,
		})
	}
	return warnings
}

func portfolioSoftCycleKeys(
	candidateKeys []string,
	adjacency map[string][]string,
	edges []portfolioGraphEdge,
) []string {
	edgeByKey := make(map[portfolioGraphEdgeKey]portfolioGraphEdge, len(edges))
	for _, edge := range edges {
		edgeByKey[portfolioGraphEdgeKey{before: edge.Before, after: edge.After}] = edge
	}

	keys := make(map[string]struct{})
	for _, component := range portfolioStronglyConnectedComponents(candidateKeys, adjacency) {
		componentSet := make(map[string]struct{}, len(component))
		for _, key := range component {
			componentSet[key] = struct{}{}
		}
		cyclic := len(component) > 1
		if !cyclic && len(component) == 1 {
			cyclic = portfolioHasAdjacency(adjacency[component[0]], component[0])
		}
		if !cyclic || !portfolioComponentHasSoftEdge(component, componentSet, adjacency, edgeByKey) {
			continue
		}
		for _, key := range component {
			keys[key] = struct{}{}
		}
	}

	result := make([]string, 0, len(keys))
	for key := range keys {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}

func portfolioComponentHasSoftEdge(
	component []string,
	componentSet map[string]struct{},
	adjacency map[string][]string,
	edgeByKey map[portfolioGraphEdgeKey]portfolioGraphEdge,
) bool {
	for _, from := range component {
		for _, to := range adjacency[from] {
			if _, ok := componentSet[to]; !ok {
				continue
			}
			edge := edgeByKey[portfolioGraphEdgeKey{before: from, after: to}]
			for _, relationshipType := range edge.ContributingTypes {
				if relationshipType == models.EntityRelFollows {
					return true
				}
			}
		}
	}
	return false
}

func portfolioStronglyConnectedComponents(candidateKeys []string, adjacency map[string][]string) [][]string {
	index := 0
	indices := make(map[string]int, len(candidateKeys))
	lowLink := make(map[string]int, len(candidateKeys))
	onStack := make(map[string]bool, len(candidateKeys))
	stack := make([]string, 0, len(candidateKeys))
	components := make([][]string, 0)

	var visit func(string)
	visit = func(node string) {
		indices[node] = index
		lowLink[node] = index
		index++
		stack = append(stack, node)
		onStack[node] = true

		for _, target := range adjacency[node] {
			if _, visited := indices[target]; !visited {
				visit(target)
				lowLink[node] = min(lowLink[node], lowLink[target])
			} else if onStack[target] {
				lowLink[node] = min(lowLink[node], indices[target])
			}
		}
		if lowLink[node] != indices[node] {
			return
		}

		component := make([]string, 0)
		for {
			last := len(stack) - 1
			member := stack[last]
			stack = stack[:last]
			onStack[member] = false
			component = append(component, member)
			if member == node {
				break
			}
		}
		sort.Strings(component)
		components = append(components, component)
	}

	for _, key := range candidateKeys {
		if _, visited := indices[key]; !visited {
			visit(key)
		}
	}
	return components
}

func missingOrderingWarnings(
	dependencyLayers [][]string,
	eligibleCandidates map[string]bool,
	roadmapAdjacency map[string][]string,
) []models.PortfolioWarning {
	warnings := make([]models.PortfolioWarning, 0)
	if len(dependencyLayers) == 0 {
		return warnings
	}
	eligibleRoots := make([]string, 0, len(dependencyLayers[0]))
	for _, key := range dependencyLayers[0] {
		if eligibleCandidates[key] {
			eligibleRoots = append(eligibleRoots, key)
		}
	}
	for i := 0; i < len(eligibleRoots); i++ {
		for j := i + 1; j < len(eligibleRoots); j++ {
			left, right := eligibleRoots[i], eligibleRoots[j]
			if portfolioReachable(roadmapAdjacency, left, right) ||
				portfolioReachable(roadmapAdjacency, right, left) {
				continue
			}
			keys := []string{left, right}
			warnings = append(warnings, models.PortfolioWarning{
				Code: models.PortfolioWarningMissingOrdering,
				Message: fmt.Sprintf(
					"missing ordering between eligible epics %s and %s",
					left,
					right,
				),
				EpicKeys: keys,
			})
		}
	}
	return warnings
}

func portfolioReachable(adjacency map[string][]string, from, to string) bool {
	visited := map[string]bool{from: true}
	stack := []string{from}
	for len(stack) > 0 {
		last := len(stack) - 1
		node := stack[last]
		stack = stack[:last]
		for _, target := range adjacency[node] {
			if target == to {
				return true
			}
			if !visited[target] {
				visited[target] = true
				stack = append(stack, target)
			}
		}
	}
	return false
}

func sortPortfolioWarnings(warnings []models.PortfolioWarning) {
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Code != warnings[j].Code {
			return warnings[i].Code < warnings[j].Code
		}
		leftKeys := strings.Join(warnings[i].EpicKeys, ",")
		rightKeys := strings.Join(warnings[j].EpicKeys, ",")
		if leftKeys != rightKeys {
			return leftKeys < rightKeys
		}
		return warnings[i].Message < warnings[j].Message
	})
}

func portfolioHasAdjacency(outgoing []string, target string) bool {
	index := sort.SearchStrings(outgoing, target)
	return index < len(outgoing) && outgoing[index] == target
}
