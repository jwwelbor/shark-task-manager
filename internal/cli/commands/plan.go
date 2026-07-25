// Package commands provides CLI command implementations.
//
// This file implements `shark plan [entity-key|collection]`: work selection,
// as opposed to next.go's keyed dispatch. Bare `shark plan` selects the next
// eligible epic (or an epic-only parallel tie). `shark plan <epic|feature>`
// evaluates exactly one hierarchy edge and returns direct children as a
// selection — it never recurses into a selected child. `shark plan
// bugs|change-cards|tech-debt` selects the next claimable standalone tier.
// A leaf entity, or a parent currently at its own agent step, returns that
// entity's rendered dispatch response exactly like `shark next` would.
//
// `shark plan` never claims, heartbeats, releases, steals a lease, or spawns
// an agent — selection responses carry no worker prompt and are not
// claimable. The operator (or harness) re-invokes `shark plan`/`shark next`
// with a selected child key to continue.
package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	sharkconfig "github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

const parallelPlanPrompt = "Multiple workflow-ready entities are available. If parallel execution is desired, claim each entity and launch one isolated worker with that entity's prompt. You may instead choose one entity from entities and execute it sequentially."

// planAdapterCache extends the shared nextAdapterCache (next.go) with the
// configured max_parallel_items bound. It embeds rather than modifies
// nextAdapterCache so keyed `shark next`'s adapter cache stays byte-for-byte
// identical to its 0e3f0103 baseline.
type planAdapterCache struct {
	*nextAdapterCache
	maxParallelItems int
}

func newPlanAdapterCache(ctx context.Context, maxParallelItems int) (*planAdapterCache, error) {
	base, err := nextNewAdapterCache(ctx)
	if err != nil {
		return nil, err
	}
	return &planAdapterCache{nextAdapterCache: base, maxParallelItems: maxParallelItems}, nil
}

// planGetPortfolioAdvisor, planGetPortfolioPlanner, planGetStandalonePlanner,
// and planGetMaxParallelItems are indirection hooks for testing — production
// code points them at the real cli.Get* service accessors.
var (
	planGetPortfolioAdvisor  = func() portfolioAdvisor { return cli.GetPortfolioAdviceService() }
	planGetPortfolioPlanner  = func() portfolioPlanner { return cli.GetPortfolioPlanningService() }
	planGetStandalonePlanner = func() standalonePlanner { return cli.GetStandalonePlanningService() }
	planGetMaxParallelItems  = func() int {
		cfg, err := cli.GetConfig()
		if err != nil {
			return sharkconfig.DefaultMaxParallelItems
		}
		return cfg.GetMaxParallelItems()
	}
	planDescribeDispatchableChildren = describePlanDispatchableChildren
)

type portfolioAdvisor interface {
	Advise(ctx context.Context) (*models.PortfolioAdviceEnvelope, error)
}

type portfolioPlanner interface {
	Plan(advice *models.PortfolioAdviceEnvelope) services.PortfolioPlan
}

type standalonePlanner interface {
	Plan(ctx context.Context, collection services.StandalonePlanCollection) (services.StandalonePlan, error)
}

// PortfolioPlanCandidate is the bounded epic projection returned by bare
// `shark plan`. It deliberately contains no dispatch action or worker prompt.
type PortfolioPlanCandidate struct {
	EntityKey     string  `json:"entity_key"`
	EntityType    string  `json:"entity_type"`
	Title         string  `json:"title"`
	Status        string  `json:"status"`
	Priority      string  `json:"priority"`
	BusinessValue *string `json:"business_value,omitempty"`
}

// PortfolioPlanSelectionResponse is the root-selection contract returned by
// bare `shark plan`.
type PortfolioPlanSelectionResponse struct {
	Mode              string                    `json:"mode"`
	Action            string                    `json:"action"`
	SelectionReason   string                    `json:"selection_reason,omitempty"`
	Entity            *PortfolioPlanCandidate   `json:"entity,omitempty"`
	ParallelExecution string                    `json:"parallel_execution,omitempty"`
	Entities          []PortfolioPlanCandidate  `json:"entities,omitempty"`
	Reason            string                    `json:"reason,omitempty"`
	Warnings          []models.PortfolioWarning `json:"warnings,omitempty"`
}

// CandidateEdge is one dependency, blocker, or link endpoint of a hierarchy
// plan candidate. Status is reported raw so a consumer can tell a satisfied
// prerequisite from an outstanding one.
type CandidateEdge struct {
	Key    string `json:"key"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

// HierarchyPlanCandidate is one direct feature or task selected beneath a
// keyed planning parent.
//
// DependsOn/Blocks/Links carry the candidate's relationship neighbourhood so a
// consumer stopping at a fork can decide which candidates are safe to run in
// parallel. They are omitempty and are left unpopulated by `shark plan`, whose
// selection output is deliberately edge-less; callers that want edges attach
// them with applyCandidateEdges after buildHierarchyPlanSelection returns.
type HierarchyPlanCandidate struct {
	EntityKey      string          `json:"entity_key"`
	EntityType     string          `json:"entity_type"`
	Title          string          `json:"title"`
	Status         string          `json:"status"`
	ExecutionOrder *int            `json:"execution_order,omitempty"`
	Priority       *int            `json:"priority,omitempty"`
	DependsOn      []CandidateEdge `json:"depends_on,omitempty"`
	Blocks         []CandidateEdge `json:"blocks,omitempty"`
	Links          []CandidateEdge `json:"links,omitempty"`
}

// HierarchyPlanSelectionResponse is the one-level selection contract returned
// when a keyed epic or feature is currently at a cascade workflow step.
type HierarchyPlanSelectionResponse struct {
	Mode              string                   `json:"mode"`
	Action            string                   `json:"action"`
	RootKey           string                   `json:"root_key"`
	RootType          string                   `json:"root_type"`
	SelectionReason   string                   `json:"selection_reason"`
	ResolvedVia       []string                 `json:"resolved_via,omitempty"`
	Entity            *HierarchyPlanCandidate  `json:"entity,omitempty"`
	ParallelExecution string                   `json:"parallel_execution,omitempty"`
	Entities          []HierarchyPlanCandidate `json:"entities,omitempty"`
}

// ParallelPlanResponse is the distinct JSON envelope returned when two or
// more workflow-ready standalone entities are available in the same tier.
type ParallelPlanResponse struct {
	Action            string         `json:"action"`
	RootKey           string         `json:"root_key"`
	RootType          string         `json:"root_type"`
	ParallelExecution string         `json:"parallel_execution"`
	Entities          []NextResponse `json:"entities"`
	Prompt            string         `json:"prompt"`
}

var planCmd = newPlanCommand()

func newPlanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "plan [entity-key|collection]",
		Short: "Select the next epic, hierarchy tier, or standalone-collection root as JSON",
		Long: `With no entity key, select the next eligible epic. Return one epic
normally, or an epic-only parallel candidate list when equally ranked roots
are ready. Bare plan never resolves features or tasks and never assembles a
worker prompt.

With an entity key, evaluate only that entity and one hierarchy edge. An epic
at a cascade step selects feature(s); a feature at a cascade step selects
task(s). The command never skips from an epic directly to tasks. A leaf entity
or a parent at its own agent step returns that entity's dispatch prompt,
exactly like 'shark next' would.

Collection roots select non-terminal, unclaimed standalone work by stored
priority tier:
  bugs         severity: critical, high, medium, low
  change-cards numeric priority: lower numbers first
  tech-debt    severity: critical, high, medium, low

Tied candidate lists in every mode are capped by .sharkconfig.json
max_parallel_items. The default is 5.

'shark plan' never claims, heartbeats, releases, or spawns an agent.
Selection responses carry no worker prompt and cannot be claimed directly —
re-invoke 'shark plan' or 'shark next' with the selected child key to
continue. 'shark next <entity-key>' remains the canonical keyed dispatch API
used by the claim/execute/advance/release loop.

Bare epic-selection JSON shape:
  {
    "mode":             "portfolio_selection",
    "action":           "select_epic" | "parallel_candidates" | "pause",
    "selection_reason": "dependency_order" | "roadmap_order" | "priority" | "parallel_tie",
    "entity":           {"entity_key": "E07", "entity_type": "epic", ...},
    "entities":         [{"entity_key": "E07", "entity_type": "epic", ...}]
  }

Keyed hierarchy-selection JSON shape:
  {
    "mode":             "hierarchy_selection",
    "action":           "select_feature" | "select_task" | "parallel_candidates",
    "root_key":         "<requested parent key>",
    "root_type":        "epic" | "feature",
    "selection_reason": "execution_order" | "priority" | "repository_order" | "parallel_tie",
    "entity":           {"entity_key": "<direct child>", ...},
    "entities":         [{"entity_key": "<direct child>", ...}]
  }

Standalone parallel-dispatch JSON shape:
  {
    "action":             "parallel_dispatch",
    "root_key":           "<collection>",
    "root_type":          "<entity type>",
    "parallel_execution": "available",
    "entities":           [<keyed dispatch responses, unchanged>],
    "prompt":             "<parallel execution guidance>"
  }

Examples:
  shark plan                           # Select the next epic root
  shark plan bugs                      # Highest-severity claimable bug tier
  shark plan change-cards              # Highest-priority claimable change-card tier
  shark plan tech-debt                 # Highest-severity claimable tech-debt tier
  shark plan E07                       # Direct feature(s) beneath the epic
  shark plan E07-F01                   # Direct task(s) beneath the feature
  shark plan E07-F01-001              # Rendered dispatch for a leaf task

Errors:
  - Unknown entity key  → exit 1
  - Entity in a state with no orchestrator action → action="pause" (not error)
	  - Internal failure rendering the prompt → exit 2 with stderr context`,
		Args: cobra.MaximumNArgs(1),
		RunE: runPlan,
	}
}

func init() {
	cli.RootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	maxParallelItems := planGetMaxParallelItems()

	tracer := cli.GetTracer("shark.cli")
	ctx, span := tracer.Start(ctx, "shark.plan")
	defer span.End()

	if len(args) == 0 {
		return runPortfolioPlan(ctx, cmd, span, maxParallelItems)
	}

	if collection, ok := parseStandaloneCollection(args[0]); ok {
		return runStandalonePlan(ctx, span, collection, maxParallelItems)
	}

	entityType, normalizedKey, err := ParseGetArgs(args)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("invalid entity key %q: %w", args[0], err)
	}

	cache, err := newPlanAdapterCache(ctx, maxParallelItems)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	resp, err := resolvePlanDispatch(ctx, cache, entityType, normalizedKey, 0)
	if err != nil {
		return handlePlanResolutionError(span, entityType, normalizedKey, err)
	}
	return outputPlanResult(span, resp)
}

func runPortfolioPlan(ctx context.Context, cmd *cobra.Command, span trace.Span, maxParallelItems int) error {
	advice, err := planGetPortfolioAdvisor().Advise(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to get portfolio advice: %w", err)
	}
	if advice == nil {
		err := errors.New("portfolio advisor returned no advice")
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	plan := planGetPortfolioPlanner().Plan(advice)
	response := buildPortfolioPlanSelection(advice, plan, maxParallelItems)
	span.SetAttributes(
		attribute.String("mode", response.Mode),
		attribute.String("action", response.Action),
		attribute.Int("portfolio.candidate_count", portfolioPlanSelectionCandidateCount(response)),
		attribute.Int("portfolio.warning_count", len(response.Warnings)),
		attribute.Bool("portfolio.evidence_complete", advice.EvidenceComplete),
	)
	if err := outputPortfolioPlanSelectionJSON(cmd, response); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func buildPortfolioPlanSelection(
	advice *models.PortfolioAdviceEnvelope,
	plan services.PortfolioPlan,
	maxParallelItems int,
) PortfolioPlanSelectionResponse {
	response := PortfolioPlanSelectionResponse{
		Mode:            "portfolio_selection",
		SelectionReason: plan.SelectionReason,
	}
	if len(plan.RootKeys) == 0 {
		response.Action = "pause"
		response.Reason = plan.PauseReason
		response.Warnings = append(response.Warnings, advice.Warnings...)
		response.Warnings = append(response.Warnings, advice.Ordering.Warnings...)
		return response
	}

	epicsByKey := make(map[string]models.PortfolioEpicEvidence, len(advice.Epics))
	for _, epic := range advice.Epics {
		epicsByKey[epic.Key] = epic
	}
	candidates := make([]PortfolioPlanCandidate, 0, len(plan.RootKeys))
	for _, key := range plan.RootKeys {
		epic, ok := epicsByKey[key]
		if !ok {
			continue
		}
		candidates = append(candidates, PortfolioPlanCandidate{
			EntityKey:     epic.Key,
			EntityType:    string(models.EntityTypeEpic),
			Title:         epic.Title,
			Status:        epic.Status,
			Priority:      epic.Priority,
			BusinessValue: epic.BusinessValue,
		})
	}
	if len(candidates) == 0 {
		response.Action = "pause"
		response.Reason = "selected_epic_evidence_unavailable"
		return response
	}
	candidates = candidates[:boundedParallelItemCount(len(candidates), maxParallelItems)]
	if len(candidates) == 1 {
		response.Action = "select_epic"
		response.Entity = &candidates[0]
		return response
	}
	response.Action = "parallel_candidates"
	response.ParallelExecution = "available"
	response.Entities = candidates
	return response
}

func portfolioPlanSelectionCandidateCount(response PortfolioPlanSelectionResponse) int {
	if response.Entity != nil {
		return 1
	}
	return len(response.Entities)
}

func runStandalonePlan(
	ctx context.Context,
	span trace.Span,
	collection services.StandalonePlanCollection,
	maxParallelItems int,
) error {
	plan, err := planGetStandalonePlanner().Plan(ctx, collection)
	if err != nil {
		return handlePlanResolutionError(span, "collection", string(collection), err)
	}
	if len(plan.Layers) == 0 {
		return outputPlanResult(span, standalonePlanNoWorkResponse(collection))
	}
	cache, err := newPlanAdapterCache(ctx, maxParallelItems)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, layer := range plan.Layers {
		entities, err := resolvePlanStandaloneLayer(ctx, cache.nextAdapterCache, layer, maxParallelItems)
		if err != nil {
			return handlePlanResolutionError(span, "collection", string(collection), err)
		}
		switch len(entities) {
		case 0:
			continue
		case 1:
			return outputPlanResult(span, entities[0])
		default:
			batch := newParallelPlanResponse(string(collection), "collection", entities)
			prepared := preparePlanParallelForOutput(*batch)
			span.SetAttributes(
				attribute.String("mode", "collection_dispatch"),
				attribute.String("action", prepared.Action),
				attribute.Int("parallel.entity_count", len(prepared.Entities)),
			)
			return outputParallelPlanJSON(prepared)
		}
	}

	return outputPlanResult(span, standalonePlanNoWorkResponse(collection))
}

func standalonePlanNoWorkResponse(collection services.StandalonePlanCollection) NextResponse {
	return NextResponse{
		EntityKey:  string(collection),
		EntityType: standaloneCollectionEntityType(collection),
		Status:     "no_dispatchable_work",
		Action:     "pause",
	}
}

func resolvePlanStandaloneLayer(
	ctx context.Context,
	cache *nextAdapterCache,
	layer []services.StandalonePlanCandidate,
	maxParallelItems int,
) ([]NextResponse, error) {
	limit := boundedParallelItemCount(len(layer), maxParallelItems)
	entities := make([]NextResponse, 0, limit)
	for _, candidate := range layer {
		resp, err := resolveNext(ctx, cache, string(candidate.EntityType), candidate.Key, 0)
		if err != nil {
			return nil, fmt.Errorf("resolve %s %s: %w", candidate.EntityType, candidate.Key, err)
		}
		if resp.Action == "error" {
			return nil, fmt.Errorf("resolve %s %s: %s", candidate.EntityType, candidate.Key, resp.Error)
		}
		if resp.Action == "spawn_agent" {
			entities = append(entities, resp)
			if len(entities) == limit {
				break
			}
		}
	}
	return entities, nil
}

func outputPlanResult(span trace.Span, resp NextResponse) error {
	if resp.selection != nil {
		selection := *resp.selection
		span.SetAttributes(
			attribute.String("mode", selection.Mode),
			attribute.String("action", selection.Action),
			attribute.String("root_type", selection.RootType),
			attribute.Int("selection.entity_count", hierarchyPlanSelectionCandidateCount(selection)),
			attribute.String("exit_status", "ok"),
		)
		return outputHierarchyPlanSelectionJSON(selection)
	}
	resp = preparePlanNextResponseForOutput(resp)
	span.SetAttributes(
		attribute.String("entity_key", resp.EntityKey),
		attribute.String("entity_type", resp.EntityType),
		attribute.String("status", resp.Status),
		attribute.String("action", resp.Action),
		attribute.String("agent_type", resp.AgentType),
		attribute.String("model", resp.Model),
		attribute.Int("prompt_bytes", len(resp.Prompt)),
		attribute.Int("unresolved_placeholders", len(resp.UnresolvedPlaceholders)),
		attribute.String("exit_status", "ok"),
	)
	return outputNextJSON(resp)
}

func hierarchyPlanSelectionCandidateCount(response HierarchyPlanSelectionResponse) int {
	if response.Entity != nil {
		return 1
	}
	return len(response.Entities)
}

func preparePlanNextResponseForOutput(resp NextResponse) NextResponse {
	resp = annotateUnresolvedPlaceholders(resp)
	warnUnresolvedPlaceholdersToStderr(resp)
	return resp
}

func preparePlanParallelForOutput(batch ParallelPlanResponse) ParallelPlanResponse {
	for i := range batch.Entities {
		batch.Entities[i] = preparePlanNextResponseForOutput(batch.Entities[i])
	}
	return batch
}

func handlePlanResolutionError(span trace.Span, entityType, key string, err error) error {
	var tokErr *templates.UnrenderedTokenError
	if errors.As(err, &tokErr) {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(
			attribute.String("entity_key", key),
			attribute.String("entity_type", entityType),
			attribute.String("exit_status", "error"),
		)
		return fmt.Errorf("[shark plan] %s (entity %s)", tokErr.Error(), key)
	}
	span.SetStatus(codes.Error, err.Error())
	span.SetAttributes(
		attribute.String("entity_key", key),
		attribute.String("entity_type", entityType),
		attribute.String("exit_status", "error"),
	)
	return err
}

func parseStandaloneCollection(value string) (services.StandalonePlanCollection, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "bugs":
		return services.StandalonePlanBugs, true
	case "change-cards", "changes":
		return services.StandalonePlanChangeCards, true
	case "tech-debt", "tech-debts":
		return services.StandalonePlanTechDebt, true
	default:
		return "", false
	}
}

func standaloneCollectionEntityType(collection services.StandalonePlanCollection) string {
	switch collection {
	case services.StandalonePlanBugs:
		return string(models.EntityTypeBug)
	case services.StandalonePlanChangeCards:
		return string(models.EntityTypeChange)
	case services.StandalonePlanTechDebt:
		return string(models.EntityTypeTechDebt)
	default:
		return "unknown"
	}
}

func newParallelPlanResponse(rootKey, rootType string, entities []NextResponse) *ParallelPlanResponse {
	return &ParallelPlanResponse{
		Action:            "parallel_dispatch",
		RootKey:           rootKey,
		RootType:          rootType,
		ParallelExecution: "available",
		Entities:          entities,
		Prompt:            parallelPlanPrompt,
	}
}

func boundedParallelItemCount(itemCount, configuredMax int) int {
	if itemCount <= 0 {
		return 0
	}
	if configuredMax <= 0 {
		configuredMax = sharkconfig.DefaultMaxParallelItems
	}
	if configuredMax < itemCount {
		return configuredMax
	}
	return itemCount
}

// resolvePlanDispatch is the top-level entry point for a keyed `shark plan
// <entity-key>` invocation (depth 0). It is a thin, explicitly-named alias
// over resolvePlanEntity, mirroring next.go's resolveNext/resolveNextDispatch
// naming so tests and callers can distinguish "start a plan resolution" from
// "recurse within one".
func resolvePlanDispatch(ctx context.Context, cache *planAdapterCache, entityType, normalizedKey string, depth int) (NextResponse, error) {
	return resolvePlanEntity(ctx, cache, entityType, normalizedKey, depth)
}

// resolvePlanEntity is the recursive core of `shark plan <entity-key>`. It
// mirrors resolveNext's dispatch resolution for non-cascade statuses (spawn,
// pause, archive, auto-advance placeholders — sharing next.go's applyWireAction
// wire-vocabulary mapping and prompt assembly) but replaces cascade
// resolution with one-level hierarchy selection instead of next.go's full
// traversal: an epic or feature at a "cascade" step returns its direct
// children as a selection and never recurses into a selected child.
func resolvePlanEntity(ctx context.Context, cache *planAdapterCache, entityType, normalizedKey string, depth int) (NextResponse, error) {
	if depth > maxCascadeDepth {
		return NextResponse{
			EntityKey:  normalizedKey,
			EntityType: entityType,
			Action:     "error",
			Error:      fmt.Sprintf("cascade depth limit (%d) exceeded — likely a misconfigured workflow", maxCascadeDepth),
		}, nil
	}

	a, err := cache.get(ctx, entityType)
	if err != nil {
		return NextResponse{}, err
	}
	transitioner := a.transitioner
	placeholderGen := a.generator
	actionSvc := a.actionSvc

	nextInfo, err := transitioner.GetNextStatus(ctx, normalizedKey)
	if err != nil {
		return NextResponse{}, fmt.Errorf("failed to read status for %s: %w", normalizedKey, err)
	}
	currentStatus := nextInfo.CurrentStatus

	resp := NextResponse{
		EntityKey:  normalizedKey,
		EntityType: entityType,
		Status:     currentStatus,
	}

	if nextInfo.IsTerminal || isArchivedStatus(entityType, currentStatus) {
		if isArchivedStatus(entityType, currentStatus) {
			resp.Action = "archive"
		} else {
			resp.Action = "pause"
		}
		return resp, nil
	}

	var vars map[string]string
	if placeholderGen != nil {
		vars, err = placeholderGen.GeneratePlaceholders(ctx, normalizedKey)
		if err != nil {
			return NextResponse{}, fmt.Errorf("failed to generate placeholders for %s: %w", normalizedKey, err)
		}
	}
	if vars == nil {
		vars = map[string]string{}
	}
	templates.AugmentPlaceholderAliases(vars)

	populated, err := actionSvc.GetStatusActionPopulated(ctx, currentStatus, vars)
	if err != nil {
		if isStatusNotFoundError(err) {
			resp.Action = "pause"
			resp.Error = fmt.Sprintf("status %q is not defined in the workflow configuration; this may be a legacy status that has been removed", currentStatus)
			return resp, nil
		}
		return NextResponse{}, fmt.Errorf("failed to populate action for status %q: %w", currentStatus, err)
	}

	if populated == nil {
		resp.Action = "pause"
		return resp, nil
	}

	internalAction := strings.TrimSpace(populated.Action)
	if actionRequiresInstruction(internalAction) && strings.TrimSpace(populated.Instruction) == "" {
		return NextResponse{}, fmt.Errorf("workflow action for %s status %q rendered an empty instruction; check the configured prompt path/template", normalizedKey, currentStatus)
	}
	if internalAction == "cascade" {
		return tryPlanHierarchy(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
	}

	wireResp, handled, err := applyPlanWireAction(ctx, cache, entityType, normalizedKey, depth, internalAction, populated, nextInfo, transitioner, resp)
	if err != nil {
		return NextResponse{}, err
	}
	if handled {
		return wireResp, nil
	}
	resp = wireResp

	attached, err := assembleDispatchPrompt(resp.Prompt, resp.AgentType, vars)
	if err != nil {
		return NextResponse{}, err
	}
	resp.Prompt = attached

	return resp, nil
}

// tryPlanHierarchy owns one-level hierarchy selection for the "cascade" verb.
// It returns direct features beneath an epic or direct tasks beneath a
// feature as a selection envelope — it never resolves a selected child's own
// dispatch response or recurses into it.
//
// When all children are non-dispatchable, the parent normally pauses. The one
// exception is when the service can prove there were children and every one of
// them is terminal; in that case the parent is auto-advanced one configured
// step and resolvePlanEntity recurses on the same entity so feature/epic
// workflows move into code_review/completed instead of stalling at 100% child
// completion.
func tryPlanHierarchy(
	ctx context.Context,
	cache *planAdapterCache,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	childrenState, err := planDescribeDispatchableChildren(ctx, entityType, normalizedKey)
	if err != nil {
		return NextResponse{}, fmt.Errorf("plan hierarchy lookup failed for %s: %w", normalizedKey, err)
	}
	selected, reason := selectPlanChildTier(childrenState.Children)
	if len(selected) > 0 {
		selection := buildHierarchyPlanSelection(
			normalizedKey,
			entityType,
			selected,
			reason,
			cache.maxParallelItems,
		)
		resp.selection = &selection
		return resp, nil
	}
	if childrenState.TotalChildren > 0 && childrenState.NonTerminalChildren == 0 {
		return autoAdvancePlanCascadeParent(ctx, cache, entityType, normalizedKey, depth, resp, nextInfo, transitioner)
	}
	// All children either non-dispatchable or absent — pause the parent.
	resp.Action = "pause"
	return resp, nil
}

func selectPlanChildTier(children []services.PlanHierarchyChild) ([]services.PlanHierarchyChild, string) {
	if len(children) == 0 {
		return []services.PlanHierarchyChild{}, ""
	}
	first := children[0]
	if first.ExecutionOrder != nil {
		selected := make([]services.PlanHierarchyChild, 0, len(children))
		for _, child := range children {
			if child.ExecutionOrder == nil || *child.ExecutionOrder != *first.ExecutionOrder {
				break
			}
			selected = append(selected, child)
		}
		if len(selected) > 1 {
			return selected, "parallel_tie"
		}
		return selected, "execution_order"
	}
	if first.EntityType == models.EntityTypeTask && first.Priority != nil {
		selected := make([]services.PlanHierarchyChild, 0, len(children))
		for _, child := range children {
			if child.ExecutionOrder != nil || child.Priority == nil || *child.Priority != *first.Priority {
				break
			}
			selected = append(selected, child)
		}
		if len(selected) > 1 {
			return selected, "parallel_tie"
		}
		return selected, "priority"
	}
	// Neither an execution order nor (for tasks) a priority distinguishes
	// these children, so nothing about them is sequenced — they are a tie,
	// the same way an equal execution_order is. Returning children[:1] here
	// would silently collapse the tier: features are created with a NULL
	// execution_order unless --order is passed, so an epic's features would
	// otherwise never surface as a parallel opportunity.
	//
	// Children arrive ordered "execution_order IS NULL" last, so reaching
	// this branch means the leading run of children all lack an order; the
	// loop stops at the first child that has one.
	selected := make([]services.PlanHierarchyChild, 0, len(children))
	for _, child := range children {
		if child.ExecutionOrder != nil {
			break
		}
		if child.EntityType == models.EntityTypeTask && child.Priority != nil {
			break
		}
		selected = append(selected, child)
	}
	if len(selected) > 1 {
		return selected, "parallel_tie"
	}
	// children[0] always survives the loop: reaching this branch means it has
	// no execution order and, if it is a task, no priority — the same two
	// predicates the loop breaks on — so selected is never empty here.
	return selected, "repository_order"
}

func buildHierarchyPlanSelection(
	rootKey, rootType string,
	children []services.PlanHierarchyChild,
	reason string,
	maxParallelItems int,
) HierarchyPlanSelectionResponse {
	response := HierarchyPlanSelectionResponse{
		Mode:            "hierarchy_selection",
		RootKey:         rootKey,
		RootType:        rootType,
		SelectionReason: reason,
	}
	children = children[:boundedParallelItemCount(len(children), maxParallelItems)]
	candidates := make([]HierarchyPlanCandidate, 0, len(children))
	for _, child := range children {
		candidates = append(candidates, HierarchyPlanCandidate{
			EntityKey:      child.Key,
			EntityType:     string(child.EntityType),
			Title:          child.Title,
			Status:         child.Status,
			ExecutionOrder: child.ExecutionOrder,
			Priority:       child.Priority,
		})
	}
	if len(candidates) == 1 {
		response.Action = "select_" + candidates[0].EntityType
		response.Entity = &candidates[0]
		return response
	}
	response.Action = "parallel_candidates"
	response.ParallelExecution = "available"
	response.Entities = candidates
	return response
}

// autoAdvancePlanCascadeParent handles tryPlanHierarchy's all-children-terminal
// case: the parent is advanced one happy-path step and resolvePlanEntity
// recurses on the same entity, so feature/epic workflows move into
// code_review/completed instead of stalling at 100% child completion.
func autoAdvancePlanCascadeParent(
	ctx context.Context,
	cache *planAdapterCache,
	entityType, normalizedKey string,
	depth int,
	resp NextResponse,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
) (NextResponse, error) {
	if nextInfo == nil || len(nextInfo.AvailableTransitions) == 0 {
		resp.Action = "pause"
		resp.Error = fmt.Sprintf(
			"all child work is terminal, but %s %s at status %q has no forward transition to auto-advance to",
			entityType, normalizedKey, resp.Status,
		)
		return resp, nil
	}
	targetStatus := nextInfo.AvailableTransitions[0].TargetStatus //shark:ordered pass-first contract, see uniqueSortedOutcomeTargets
	opts := services.TransitionOptions{
		Reason: "cascade completion: all child work terminal",
		Agent:  "shark-cascade",
	}
	if _, err := transitioner.TransitionStatus(ctx, normalizedKey, targetStatus, opts); err != nil {
		return NextResponse{}, fmt.Errorf(
			"cascade completion advance from %s to %s failed for %s: %w",
			resp.Status, targetStatus, normalizedKey, err,
		)
	}
	return resolvePlanEntity(ctx, cache, entityType, normalizedKey, depth+1)
}

// applyPlanWireAction mirrors next.go's applyWireAction (shared wire-vocabulary
// mapping via normalizeWireAction), differing only in its recursive target
// for the "advance_and_recurse" branch: resolvePlanEntity rather than
// resolveNext, so a same-entity auto-advance that lands on a new cascade step
// still gets one-level selection instead of next.go's full traversal.
func applyPlanWireAction(
	ctx context.Context,
	cache *planAdapterCache,
	entityType, normalizedKey string,
	depth int,
	internalAction string,
	populated *action.PopulatedAction,
	nextInfo *services.NextStatusInfo,
	transitioner runner.EntityTransitioner,
	resp NextResponse,
) (NextResponse, bool, error) {
	wireAction, transitionalTarget := normalizeWireAction(internalAction, populated.AgentType, nextInfo)
	switch wireAction {
	case "advance_and_recurse":
		if transitionalTarget == "" {
			resp.Action = "pause"
			return resp, true, nil
		}
		if _, err := transitioner.TransitionStatus(ctx, normalizedKey, transitionalTarget, services.TransitionOptions{}); err != nil {
			return NextResponse{}, true, fmt.Errorf("auto-advance from %s to %s failed for %s: %w", resp.Status, transitionalTarget, normalizedKey, err)
		}
		recursed, err := resolvePlanEntity(ctx, cache, entityType, normalizedKey, depth+1)
		if err != nil {
			return NextResponse{}, true, err
		}
		return recursed, true, nil

	case "error":
		resp.Action = "pause"
		resp.Error = fmt.Sprintf("unknown internal action verb %q for status %q", internalAction, resp.Status)
		return resp, true, nil
	}

	resp.Action = wireAction
	resp.AgentType = populated.AgentType
	resp.Provider = populated.Provider
	resp.Model = populated.Model
	resp.Effort = populated.Effort
	resp.Prompt = populated.Instruction
	return resp, false, nil
}

func describePlanDispatchableChildren(ctx context.Context, entityType, key string) (services.PlanHierarchyChildrenState, error) {
	return cli.GetPlanHierarchyService().DescribeChildren(ctx, entityType, key)
}

// describePlanCandidateEdges loads dependency/blocker/link edges for already
// selected candidates of one entity type. `shark plan` never calls this — its
// selection output stays edge-less; keyed fork callers do.
func describePlanCandidateEdges(
	ctx context.Context,
	entityType string,
	keys []string,
) (map[string]services.PlanHierarchyEdges, error) {
	return cli.GetPlanHierarchyService().DescribeChildEdges(ctx, entityType, keys)
}

// applyCandidateEdges attaches loaded edges to an existing hierarchy selection,
// covering both envelope shapes: the singleton Entity pointer and the
// parallel_candidates Entities slice.
//
// edges must be keyed by canonical entity key, which is what
// PlanHierarchyService.DescribeChildEdges returns. Candidates with no entry are
// left edge-less rather than zeroed, so a partial edge load never silently
// rewrites a candidate that already carries edges.
func applyCandidateEdges(
	response *HierarchyPlanSelectionResponse,
	edges map[string]services.PlanHierarchyEdges,
) {
	if response == nil || len(edges) == 0 {
		return
	}
	if response.Entity != nil {
		applyCandidateEdgesTo(response.Entity, edges)
	}
	for index := range response.Entities {
		applyCandidateEdgesTo(&response.Entities[index], edges)
	}
}

func applyCandidateEdgesTo(
	candidate *HierarchyPlanCandidate,
	edges map[string]services.PlanHierarchyEdges,
) {
	found, ok := edges[candidate.EntityKey]
	if !ok {
		return
	}
	candidate.DependsOn = toCandidateEdges(found.DependsOn)
	candidate.Blocks = toCandidateEdges(found.Blocks)
	candidate.Links = toCandidateEdges(found.Links)
}

// toCandidateEdges returns nil for an empty input so the omitempty json tags
// keep an edge-free candidate byte-identical to its pre-edges shape.
func toCandidateEdges(edges []services.PlanHierarchyEdge) []CandidateEdge {
	if len(edges) == 0 {
		return nil
	}
	converted := make([]CandidateEdge, 0, len(edges))
	for _, edge := range edges {
		converted = append(converted, CandidateEdge{
			Key:    edge.Key,
			Status: edge.Status,
			Type:   edge.Type,
		})
	}
	return converted
}

func outputParallelPlanJSON(resp ParallelPlanResponse) error {
	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal parallel plan response: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func outputHierarchyPlanSelectionJSON(response HierarchyPlanSelectionResponse) error {
	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal hierarchy plan selection: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func outputPortfolioPlanSelectionJSON(cmd *cobra.Command, response PortfolioPlanSelectionResponse) error {
	out, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal portfolio plan selection: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(out)); err != nil {
		return fmt.Errorf("failed to write portfolio plan selection: %w", err)
	}
	return nil
}
