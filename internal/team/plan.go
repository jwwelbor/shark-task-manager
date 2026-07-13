package team

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/dispatch"
	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// TeamPlanner builds complete immutable plan snapshots from injected read
// seams. It has no mutation-capable dependency by design.
type TeamPlanner struct {
	children     ChildSnapshotReader
	dependencies DependencyReader
	dispatch     DispatchStepResolver
	claims       ClaimDiagnosticReader
}

func NewPlanner(deps PlannerDeps) (*TeamPlanner, error) {
	if deps.Children == nil {
		return nil, errors.New("team planner: child snapshot reader is required")
	}
	if deps.Dispatch == nil {
		return nil, errors.New("team planner: dispatch-step resolver is required")
	}
	return &TeamPlanner{children: deps.Children, dependencies: deps.Dependencies, dispatch: deps.Dispatch, claims: deps.Claims}, nil
}

// NewTeamPlanner is the descriptive constructor alias used by service
// wiring; NewPlanner remains the concise contract-oriented name.
func NewTeamPlanner(deps PlannerDeps) (*TeamPlanner, error) {
	return NewPlanner(deps)
}

var _ Planner = (*TeamPlanner)(nil)

func (p *TeamPlanner) Plan(ctx context.Context, input PlanInput) (*TeamPlan, error) {
	if err := validatePlanInput(input); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rootKey := canonicalKey(input.RootKey)
	children, items, byKey, err := p.snapshotChildren(ctx, input, rootKey)
	if err != nil {
		return nil, err
	}
	if err := p.attachDependencies(ctx, children, items, byKey, rootKey); err != nil {
		return nil, err
	}
	if err := validateGraphAndEligibility(rootKey, items, byKey); err != nil {
		return nil, err
	}
	return finalizePlan(rootKey, input, items)
}

func (p *TeamPlanner) snapshotChildren(ctx context.Context, input PlanInput, rootKey string) ([]ChildSnapshot, []TeamPlanItem, map[string]int, error) {
	children, err := p.children.ListChildren(ctx, input.RootType, input.RootKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("list direct children for root %s: %w", rootKey, err)
	}
	items := make([]TeamPlanItem, 0, len(children))
	byKey := make(map[string]int, len(children))
	for i := range children {
		child := &children[i]
		child.Key = canonicalKey(child.Key)
		if err := validateEntityIdentity(child.Key, child.EntityType); err != nil {
			return nil, nil, nil, validationErrorf(ErrInvalidPlanInput, rootKey, child.Key, "", "child identity: %v", err)
		}
		identity := ChildIdentity{Key: child.Key, EntityType: child.EntityType}
		identityKey := string(identity.EntityType) + ":" + identity.Key
		if _, exists := byKey[identityKey]; exists {
			return nil, nil, nil, validationError(ErrDuplicateChild, rootKey, child.Key, "", "direct child appears more than once")
		}
		byKey[identityKey] = len(items)
		step, err := p.dispatch.Resolve(ctx, child.EntityType, child.Key)
		if err != nil {
			return nil, nil, nil, validationErrorf(ErrUnresolvedWorkflow, rootKey, child.Key, "", "resolve dispatch step: %v", err)
		}
		if step.Error != "" {
			return nil, nil, nil, validationError(ErrUnresolvedWorkflow, rootKey, child.Key, "", step.Error)
		}
		item := TeamPlanItem{ChildKey: child.Key, ChildType: child.EntityType, Status: child.Status, ExecutionOrder: child.ExecutionOrder, Priority: child.Priority, Planned: metadataFromStep(step), Eligible: true}
		if p.claims != nil {
			item.Claim, err = p.claims.Diagnose(ctx, identity)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("diagnose claim for child %s: %w", child.Key, err)
			}
		}
		applyStepExclusion(&item, step)
		items = append(items, item)
	}
	return children, items, byKey, nil
}

func applyStepExclusion(item *TeamPlanItem, step dispatch.DispatchStep) {
	switch {
	case step.GateClassification == dispatch.GateTerminal:
		item.Eligible, item.ExclusionReason = false, ExclusionTerminal
	case item.Claim.Claimed:
		item.Eligible, item.ExclusionReason = false, ExclusionClaimed
	case step.GateClassification == dispatch.GateHuman:
		item.Eligible, item.ExclusionReason = false, ExclusionHumanGate
	case step.GateClassification == dispatch.GatePause:
		item.Eligible, item.ExclusionReason = false, ExclusionPause
	}
}

func (p *TeamPlanner) attachDependencies(ctx context.Context, children []ChildSnapshot, items []TeamPlanItem, byKey map[string]int, rootKey string) error {
	for i := range items {
		identity := ChildIdentity{Key: items[i].ChildKey, EntityType: items[i].ChildType}
		edges, err := p.listDependencies(ctx, identity, children[i].LegacyDependencies)
		if err != nil {
			return validationErrorf(ErrMalformedDependency, rootKey, identity.Key, "", "read dependencies: %v", err)
		}
		normalized, err := normalizeEdges(identity, edges)
		if err != nil {
			return validationError(ErrMalformedDependency, rootKey, identity.Key, "", err.Error())
		}
		for _, edge := range normalized {
			key := string(edge.DependencyType) + ":" + edge.DependencyKey
			if edge.Resolved {
				_, edgeIsInternal := byKey[key]
				edge.External = !edgeIsInternal
			}
			if _, exists := byKey[key]; !exists && !edge.External {
				return validationError(ErrMissingDependency, rootKey, identity.Key, edge.DependencyKey, "dependency is not a direct child and was not marked external")
			}
			items[i].Dependencies = append(items[i].Dependencies, edge)
			items[i].DependencyKeys = append(items[i].DependencyKeys, edge.DependencyKey)
		}
		sort.Strings(items[i].DependencyKeys)
	}
	return nil
}

func validateGraphAndEligibility(rootKey string, items []TeamPlanItem, byKey map[string]int) error {
	if err := detectCycles(rootKey, items, byKey); err != nil {
		return err
	}
	assignWaves(items, byKey)
	for i := range items {
		for _, edge := range items[i].Dependencies {
			if edge.External {
				if !dependencySatisfied(edge) {
					items[i].Eligible, items[i].ExclusionReason = false, ExclusionDependencyIneligible
				}
				continue
			}
			depIndex := byKey[string(edge.DependencyType)+":"+edge.DependencyKey]
			if !dependencySatisfied(edge) && !successfulStatus(items[depIndex].Status) {
				items[i].Eligible, items[i].ExclusionReason = false, ExclusionDependencyIneligible
			}
		}
	}
	return nil
}

func finalizePlan(rootKey string, input PlanInput, items []TeamPlanItem) (*TeamPlan, error) {
	mode, limit, reason, err := selectMode(input.RequestedConcurrency, input.Capabilities, rootKey)
	if err != nil {
		return nil, err
	}
	plan := &TeamPlan{RootKey: rootKey, RootType: input.RootType, Items: items, ExecutionMode: mode, ConcurrencyLimit: limit, DegradedReason: reason}
	if reason != "" {
		plan.CapabilityExclusions = []string{reason}
	}
	sort.Slice(plan.Items, func(i, j int) bool { return planItemLess(plan.Items[i], plan.Items[j]) })
	plan.PlanHash, err = plan.computeHash()
	if err != nil {
		return nil, fmt.Errorf("hash team plan for root %s: %w", rootKey, err)
	}
	return plan, nil
}

func validatePlanInput(input PlanInput) error {
	if input.RootType != models.EntityTypeEpic && input.RootType != models.EntityTypeFeature {
		return fmt.Errorf("%w: root type %q must be epic or feature", ErrInvalidPlanInput, input.RootType)
	}
	if strings.TrimSpace(input.RootKey) == "" {
		return fmt.Errorf("%w: root key is required", ErrInvalidPlanInput)
	}
	if err := validateEntityIdentity(input.RootKey, input.RootType); err != nil {
		return fmt.Errorf("%w: root identity: %v", ErrInvalidPlanInput, err)
	}
	if input.RequestedConcurrency <= 0 {
		return fmt.Errorf("%w: requested concurrency must be positive", ErrInvalidPlanInput)
	}
	return nil
}

func validationError(cause error, root, child, reference, detail string) error {
	return &PlanValidationError{Cause: cause, RootKey: root, ChildKey: child, Reference: reference, Detail: detail}
}

func validationErrorf(cause error, root, child, reference, format string, args ...any) error {
	return validationError(cause, root, child, reference, fmt.Sprintf(format, args...))
}

func canonicalKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}

func (p *TeamPlanner) listDependencies(ctx context.Context, child ChildIdentity, legacy string) ([]DependencyEdge, error) {
	var edges []DependencyEdge
	if p.dependencies != nil {
		var err error
		edges, err = p.dependencies.ListDependencies(ctx, child)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(legacy) != "" && strings.TrimSpace(legacy) != "null" && strings.TrimSpace(legacy) != "[]" {
		legacyEdges, err := parseLegacyDependencies(child, legacy)
		if err != nil {
			return nil, err
		}
		edges = append(edges, legacyEdges...)
	}
	return edges, nil
}

func normalizeEdges(child ChildIdentity, edges []DependencyEdge) ([]DependencyEdge, error) {
	seen := make(map[string]bool, len(edges))
	out := make([]DependencyEdge, 0, len(edges))
	for _, edge := range edges {
		if edge.ChildKey == "" {
			edge.ChildKey = child.Key
		}
		if edge.ChildType == "" {
			edge.ChildType = child.EntityType
		}
		edge.ChildKey = canonicalKey(edge.ChildKey)
		edge.DependencyKey = canonicalKey(edge.DependencyKey)
		if edge.DependencyType == "" {
			edge.DependencyType = child.EntityType
		}
		if err := validateEntityIdentity(edge.ChildKey, edge.ChildType); err != nil {
			return nil, fmt.Errorf("dependency child identity: %w", err)
		}
		if err := validateEntityIdentity(edge.DependencyKey, edge.DependencyType); err != nil {
			return nil, fmt.Errorf("dependency target identity: %w", err)
		}
		if edge.ChildKey != child.Key || edge.ChildType != child.EntityType {
			return nil, fmt.Errorf("dependency edge child %s/%s does not match requested child %s/%s", edge.ChildType, edge.ChildKey, child.EntityType, child.Key)
		}
		identity := string(edge.DependencyType) + ":" + edge.DependencyKey
		if seen[identity] {
			continue
		}
		seen[identity] = true
		out = append(out, edge)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DependencyType != out[j].DependencyType {
			return out[i].DependencyType < out[j].DependencyType
		}
		return out[i].DependencyKey < out[j].DependencyKey
	})
	return out, nil
}

func parseLegacyDependencies(child ChildIdentity, raw string) ([]DependencyEdge, error) {
	var keys []string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, fmt.Errorf("parse legacy depends_on for %s: %w", child.Key, err)
	}
	edges := make([]DependencyEdge, 0, len(keys))
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return nil, errors.New("legacy dependency key is empty")
		}
		edges = append(edges, DependencyEdge{ChildKey: child.Key, ChildType: child.EntityType, DependencyKey: key, DependencyType: child.EntityType, Source: "legacy"})
	}
	return edges, nil
}

// DependencyAdapter merges the legacy JSON dependency source and normalized
// entity_relationships source. It is intentionally read-only.
type DependencyAdapter struct {
	legacy       LegacyDependencySource
	relationship RelationshipDependencySource
}

func NewDependencyAdapter(sources ...any) *DependencyAdapter {
	adapter := &DependencyAdapter{}
	for _, source := range sources {
		switch typed := source.(type) {
		case LegacyDependencySource:
			adapter.legacy = typed
		case RelationshipDependencySource:
			adapter.relationship = typed
		}
	}
	return adapter
}

func (a *DependencyAdapter) ListDependencies(ctx context.Context, child ChildIdentity) ([]DependencyEdge, error) {
	var edges []DependencyEdge
	if a.legacy != nil {
		raw, err := a.legacy.ListLegacyDependencies(ctx, child)
		if err != nil {
			return nil, fmt.Errorf("read legacy dependencies for %s: %w", child.Key, err)
		}
		parsed, err := parseLegacyDependencies(child, raw)
		if err != nil && strings.TrimSpace(raw) != "" && strings.TrimSpace(raw) != "null" && strings.TrimSpace(raw) != "[]" {
			return nil, fmt.Errorf("%w: %v", ErrMalformedDependency, err)
		}
		edges = append(edges, parsed...)
	}
	if a.relationship != nil {
		relationships, err := a.relationship.ListRelationshipDependencies(ctx, child)
		if err != nil {
			return nil, fmt.Errorf("read relationship dependencies for %s: %w", child.Key, err)
		}
		for i := range relationships {
			// The relationship source resolved the target entity. Scope is
			// intentionally classified by TeamPlanner against the root roster.
			relationships[i].Resolved = true
		}
		edges = append(edges, relationships...)
	}
	return normalizeEdges(child, edges)
}

func IsMalformedDependency(err error) bool { return errors.Is(err, ErrMalformedDependency) }

func detectCycles(root string, items []TeamPlanItem, byKey map[string]int) error {
	state := make([]uint8, len(items))
	var visit func(int) error
	visit = func(index int) error {
		if state[index] == 1 {
			return validationError(ErrDependencyCycle, root, items[index].ChildKey, "", "dependency cycle detected")
		}
		if state[index] == 2 {
			return nil
		}
		state[index] = 1
		for _, edge := range items[index].Dependencies {
			if edge.External {
				continue
			}
			depIndex, ok := byKey[string(edge.DependencyType)+":"+edge.DependencyKey]
			if !ok {
				return validationError(ErrMissingDependency, root, items[index].ChildKey, edge.DependencyKey, "dependency reference is missing")
			}
			if err := visit(depIndex); err != nil {
				return err
			}
		}
		state[index] = 2
		return nil
	}
	for i := range items {
		if err := visit(i); err != nil {
			return err
		}
	}
	return nil
}

func assignWaves(items []TeamPlanItem, byKey map[string]int) {
	state := make([]uint8, len(items))
	var wave func(int) int
	wave = func(index int) int {
		if state[index] == 2 {
			return items[index].Wave
		}
		if state[index] == 1 {
			return items[index].Wave
		}
		state[index] = 1
		max := 0
		for _, edge := range items[index].Dependencies {
			if edge.External {
				continue
			}
			depWave := wave(byKey[string(edge.DependencyType)+":"+edge.DependencyKey]) + 1
			if depWave > max {
				max = depWave
			}
		}
		items[index].Wave = max
		state[index] = 2
		return max
	}
	for i := range items {
		wave(i)
	}
}

func dependencySatisfied(edge DependencyEdge) bool {
	if edge.Satisfied {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(edge.DependencyStatus)) {
	case "completed", "archived", "shipped", "success", "passed":
		return true
	default:
		return false
	}
}

func successfulStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "archived", "shipped", "success", "passed":
		return true
	default:
		return false
	}
}

func planItemLess(a, b TeamPlanItem) bool {
	if a.Wave != b.Wave {
		return a.Wave < b.Wave
	}
	if a.ExecutionOrder != b.ExecutionOrder {
		return a.ExecutionOrder < b.ExecutionOrder
	}
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if a.ChildType != b.ChildType {
		return a.ChildType < b.ChildType
	}
	return a.ChildKey < b.ChildKey
}

func selectMode(requested int, facts CapabilityFacts, root string) (ExecutionMode, int, string, error) {
	teamAvailable := facts.TeamExecutionAvailable || facts.SafeTeamExecution
	singleAvailable := facts.SingleWorkerAvailable || facts.SafeSingleWorkerExecution || facts.SafeWorkerExecutionAvailable
	isolationAvailable := facts.WorktreeIsolationAvailable || facts.WorktreeIsolation
	max := facts.MaxConcurrency
	if max == 0 {
		max = facts.MaxParallelism
	}
	if max <= 0 {
		max = requested
	}
	if teamAvailable && isolationAvailable && !facts.UnknownResourceOwnership && !facts.OverlappingResourceOwnership && !facts.ResourceOwnershipOverlap && facts.ResourceOwnershipKnown {
		return ExecutionModeParallel, minPositive(requested, max), "", nil
	}
	reason := DegradedReasonParallelUnavailable
	if teamAvailable && isolationAvailable && (facts.UnknownResourceOwnership || !facts.ResourceOwnershipKnown) {
		reason = DegradedReasonUnknownResourceOwnership
	} else if teamAvailable && isolationAvailable && (facts.OverlappingResourceOwnership || facts.ResourceOwnershipOverlap) {
		reason = DegradedReasonOverlappingOwnership
	}
	if singleAvailable {
		return ExecutionModeSequential, 1, reason, nil
	}
	return "", 0, "", &CapabilityError{RootKey: root, Reason: reason}
}

func minPositive(a, b int) int {
	if a < b {
		return a
	}
	return b
}
