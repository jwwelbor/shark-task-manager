package commands

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	wf "github.com/jwwelbor/shark-task-manager/internal/workflow"
)

// buildMultiLevelWorkflowDisplay builds display structs for all workflow levels.
func buildMultiLevelWorkflowDisplay(multi *config.MultiLevelWorkflow, configPath string, levelFilter string) *MultiLevelWorkflowDisplay {
	levels := make([]*LevelWorkflowDisplay, 0, len(config.KnownWorkflowLevels))
	for _, lvl := range config.KnownWorkflowLevels {
		if levelFilter != "" && lvl != levelFilter {
			continue
		}
		levels = append(levels, buildLevelWorkflowDisplay(lvl, multi.RawForLevel(lvl), multi.GetWorkflowForLevel(lvl)))
	}
	return &MultiLevelWorkflowDisplay{
		Levels:     levels,
		ConfigPath: configPath,
	}
}

// normalizeWorkflowListLevel validates and normalizes the optional entity-type
// filter accepted by `shark admin workflow list`.
func normalizeWorkflowListLevel(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "-", "_")

	if level, ok := workflowListLevelAliases[normalized]; ok {
		return level, nil
	}

	return "", fmt.Errorf("invalid entity type %q: must be one of %s", raw, strings.Join(config.KnownWorkflowLevels, ", "))
}

var workflowListLevelAliases = map[string]string{
	"epic":     wf.LevelEpic,
	"epics":    wf.LevelEpic,
	"feature":  wf.LevelFeature,
	"features": wf.LevelFeature,
	"task":     wf.LevelTask,
	"tasks":    wf.LevelTask,
	"sprint":   wf.LevelSprint,
	"sprints":  wf.LevelSprint,
	"bug":      wf.LevelBug,
	"bugs":     wf.LevelBug,

	"change":       wf.LevelChange,
	"changes":      wf.LevelChange,
	"change_card":  wf.LevelChange,
	"change_cards": wf.LevelChange,

	"tech_debt":  wf.LevelTechDebt,
	"tech_debts": wf.LevelTechDebt,
	"techdebt":   wf.LevelTechDebt,
	"td":         wf.LevelTechDebt,
}

// buildLevelWorkflowDisplay builds the display struct for a single workflow level.
// raw is the custom config (nil if using default), resolved is the effective config (never nil).
func buildLevelWorkflowDisplay(level string, raw *config.WorkflowConfig, resolved *config.WorkflowConfig) *LevelWorkflowDisplay {
	source := "default"
	if raw != nil {
		source = "custom"
	}

	transitionCount := 0
	for _, transitions := range resolved.StatusFlow {
		transitionCount += len(transitions)
	}

	statusNames := make([]string, 0, len(resolved.StatusFlow))
	for s := range resolved.StatusFlow {
		statusNames = append(statusNames, s)
	}
	sort.Strings(statusNames)

	statuses := make([]StatusDisplay, 0, len(statusNames))
	for _, name := range statusNames {
		sd := StatusDisplay{
			Name:        name,
			Transitions: resolved.StatusFlow[name],
		}
		if sd.Transitions == nil {
			sd.Transitions = []string{}
		}
		if meta, ok := resolved.StatusMetadata[name]; ok {
			sd.Description = meta.Description
			sd.Phase = meta.Phase
			sd.Color = meta.Color
			sd.IsPlanning = meta.IsPlanning
			sd.AggregatesFrom = meta.AggregatesFrom
			if len(meta.AgentTypes) > 0 {
				sd.AgentTypes = meta.AgentTypes
			}
		}
		statuses = append(statuses, sd)
	}

	version := resolved.Version
	if version == "" {
		version = config.DefaultWorkflowVersion
	}

	return &LevelWorkflowDisplay{
		Level:           level,
		Source:          source,
		Version:         version,
		Statuses:        statuses,
		SpecialStatuses: resolved.SpecialStatuses,
		StatusCount:     len(resolved.StatusFlow),
		TransitionCount: transitionCount,
	}
}

// displayMultiLevelWorkflowHumanReadable renders all workflow levels in human-readable format.
func displayMultiLevelWorkflowHumanReadable(display *MultiLevelWorkflowDisplay) error {
	fmt.Println("Workflow Configuration")
	fmt.Println("================================================================")

	for _, level := range display.Levels {
		displayWorkflowLevelSection(level)
	}

	fmt.Println("Legend:")
	fmt.Println("  [status] = aggregation threshold (progress derived from children)")
	fmt.Println("  [planning] = entity has its own workflow status (not aggregating)")
	fmt.Println("  [aggregates: X] = status aggregates progress from children of type X")
	fmt.Println()

	return nil
}

// displayMultiLevelWorkflowSimple renders compact ASCII status-flow lines.
func displayMultiLevelWorkflowSimple(display *MultiLevelWorkflowDisplay) error {
	fmt.Println("Workflow Configuration (simple)")
	fmt.Println("================================================================")

	for _, level := range display.Levels {
		displayWorkflowLevelSimple(level)
	}
	fmt.Println()

	return nil
}

// levelDisplayLabel turns a level key like "tech_debt" into "Tech Debt" for
// human-readable headers.
func levelDisplayLabel(level string) string {
	titleCase := cases.Title(language.English)
	return titleCase.String(strings.ReplaceAll(level, "_", " "))
}

// displayWorkflowLevelSection renders a single workflow level section.
func displayWorkflowLevelSection(level *LevelWorkflowDisplay) {
	fmt.Printf("\n--- %s Workflow (%s) ---\n", levelDisplayLabel(level.Level), level.Source)
	fmt.Printf("  Version: %s\n\n", level.Version)

	displaySpecialStatuses(level.SpecialStatuses)

	fmt.Println("  Status Transitions:")
	for _, status := range level.Statuses {
		displayStatusWithMarkers(status)
	}
}

// displayWorkflowLevelSimple renders one workflow as adjacency-list flow lines.
func displayWorkflowLevelSimple(level *LevelWorkflowDisplay) {
	fmt.Printf("\n%s Workflow (%s)\n", levelDisplayLabel(level.Level), level.Source)
	for _, status := range orderedStatusesForSimpleDisplay(level) {
		targets := "[terminal]"
		if len(status.Transitions) > 0 {
			targets = strings.Join(status.Transitions, " | ")
		}
		fmt.Printf("  %s -> %s\n", status.Name, targets)
	}
}

// orderedStatusesForSimpleDisplay returns statuses in flow order when possible:
// start statuses first, then reachable transitions, then any disconnected
// statuses alphabetically for deterministic output.
func orderedStatusesForSimpleDisplay(level *LevelWorkflowDisplay) []StatusDisplay {
	byName := make(map[string]StatusDisplay, len(level.Statuses))
	for _, status := range level.Statuses {
		byName[status.Name] = status
	}

	visited := make(map[string]bool, len(level.Statuses))
	ordered := make([]StatusDisplay, 0, len(level.Statuses))

	var visit func(string)
	visit = func(name string) {
		status, ok := byName[name]
		if !ok || visited[name] {
			return
		}
		visited[name] = true
		ordered = append(ordered, status)
		for _, next := range status.Transitions {
			visit(next)
		}
	}

	for _, start := range level.SpecialStatuses[config.StartStatusKey] {
		visit(start)
	}

	for _, status := range level.Statuses {
		visit(status.Name)
	}

	return ordered
}

// displaySpecialStatuses renders the special statuses section.
func displaySpecialStatuses(specials map[string][]string) {
	if len(specials) == 0 {
		return
	}
	fmt.Println("  Special Statuses:")
	keys := []struct {
		key   string
		label string
	}{
		{config.StartStatusKey, "entry points"},
		{config.CompleteStatusKey, "exit points"},
		{config.AggregationStatusKey, "threshold"},
	}
	for _, k := range keys {
		if vals, ok := specials[k.key]; ok && len(vals) > 0 {
			fmt.Printf("    %s (%s):  %s\n", k.key, k.label, strings.Join(vals, ", "))
		}
	}
	fmt.Println()
}

// displayStatusWithMarkers renders a single status with planning/aggregation markers.
func displayStatusWithMarkers(status StatusDisplay) {
	nameDisplay := status.Name
	if status.AggregatesFrom != "" {
		nameDisplay = fmt.Sprintf("[%s]", status.Name)
	}

	descPart := ""
	if status.Description != "" {
		descPart = fmt.Sprintf(" (%s)", status.Description)
	}

	var markers []string
	if status.IsPlanning {
		markers = append(markers, "[planning]")
	}
	if status.AggregatesFrom != "" {
		markers = append(markers, fmt.Sprintf("[aggregates: %s]", status.AggregatesFrom))
	}
	markerSuffix := ""
	if len(markers) > 0 {
		markerSuffix = "  " + strings.Join(markers, " ")
	}

	fmt.Printf("    %s%s%s\n", nameDisplay, descPart, markerSuffix)

	if len(status.Transitions) == 0 {
		fmt.Printf("      -> (terminal - no transitions)\n")
	} else {
		for _, t := range status.Transitions {
			fmt.Printf("      -> %s\n", t)
		}
	}

	var metaInfo []string
	if status.Phase != "" {
		metaInfo = append(metaInfo, fmt.Sprintf("phase: %s", status.Phase))
	}
	if len(status.AgentTypes) > 0 {
		metaInfo = append(metaInfo, fmt.Sprintf("agents: %s", strings.Join(status.AgentTypes, ", ")))
	}
	if status.Color != "" {
		metaInfo = append(metaInfo, fmt.Sprintf("color: %s", status.Color))
	}
	if len(metaInfo) > 0 {
		fmt.Printf("      [%s]\n", strings.Join(metaInfo, " | "))
	}
	fmt.Println()
}
