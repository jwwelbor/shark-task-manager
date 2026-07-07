package templates_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	cfgtemplate "github.com/jwwelbor/shark-task-manager/internal/config/template"
	workflowcfg "github.com/jwwelbor/shark-task-manager/internal/config/workflow"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

const (
	b036VerificationDir  = "dev-artifacts/2026-07-06-b036-prompt-audit/verification"
	b036AuditJSONName    = "exhaustive-prompt-crawl.json"
	b036AuditSummaryName = "exhaustive-prompt-crawl-summary.md"
)

type b036BundleConfig struct {
	Name                string
	WorkflowConfig      string
	SharkDataPath       string
	ExpectedTemplateDir string
}

type b036AuditRecord struct {
	Bundle               string `json:"bundle"`
	EntityType           string `json:"entity_type"`
	Status               string `json:"status"`
	Step                 string `json:"step"`
	Phase                string `json:"phase"`
	PromptReference      string `json:"prompt_reference"`
	ResolvedTemplateRoot string `json:"resolved_template_root"`
	Exists               bool   `json:"exists"`
	Renderable           bool   `json:"renderable"`
	NonEmpty             bool   `json:"non_empty"`
	FailureDetail        string `json:"failure_detail,omitempty"`
}

type b036AuditReport struct {
	GeneratedAt string            `json:"generated_at"`
	ArtifactDir string            `json:"artifact_dir"`
	RecordCount int               `json:"record_count"`
	Bundles     map[string]int    `json:"bundles"`
	Records     []b036AuditRecord `json:"records"`
}

func TestB036_ExhaustivePromptCrawl_WritesVerificationArtifacts(t *testing.T) {
	repoRoot := findRepoRootForB036PromptAudit(t)
	artifactDir := filepath.Join(repoRoot, b036VerificationDir)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", artifactDir, err)
	}

	defaultBundleRoot := filepath.Join(repoRoot, "internal", "sharkdata", "default_data")
	fixture := testutil.WriteWorkflowIndexFixture(t)
	bundles := []b036BundleConfig{
		{
			Name:                "embedded-default",
			WorkflowConfig:      filepath.Join(defaultBundleRoot, "workflow"),
			SharkDataPath:       defaultBundleRoot,
			ExpectedTemplateDir: filepath.Join(defaultBundleRoot, "prompts"),
		},
		{
			Name:                "workflow-index-fixture",
			WorkflowConfig:      fixture.WorkflowIndexPath,
			ExpectedTemplateDir: fixture.ExpectedPromptsDir,
		},
	}

	config.ClearWorkflowCache()
	t.Cleanup(config.ClearWorkflowCache)

	var records []b036AuditRecord
	bundleCounts := make(map[string]int, len(bundles))

	for _, bundle := range bundles {
		records = append(records, crawlPromptBundle(t, bundle)...)
	}

	sort.Slice(records, func(i, j int) bool {
		left := []string{records[i].Bundle, records[i].EntityType, records[i].Status, records[i].PromptReference}
		right := []string{records[j].Bundle, records[j].EntityType, records[j].Status, records[j].PromptReference}
		for k := range left {
			if left[k] != right[k] {
				return left[k] < right[k]
			}
		}
		return false
	})
	for _, record := range records {
		bundleCounts[record.Bundle]++
	}

	report := b036AuditReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		ArtifactDir: artifactDir,
		RecordCount: len(records),
		Bundles:     bundleCounts,
		Records:     records,
	}
	writeB036Artifacts(t, artifactDir, report)

	failures := failingAuditRecords(records)
	if len(failures) > 0 {
		t.Fatalf("prompt crawl found %d defect(s); see %s and %s", len(failures), filepath.Join(artifactDir, b036AuditJSONName), filepath.Join(artifactDir, b036AuditSummaryName))
	}
}

func crawlPromptBundle(t *testing.T, bundle b036BundleConfig) []b036AuditRecord {
	t.Helper()

	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	writeB036AuditConfig(t, configPath, bundle)

	config.ClearWorkflowCache()
	multi, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return []b036AuditRecord{{
			Bundle:               bundle.Name,
			ResolvedTemplateRoot: bundle.ExpectedTemplateDir,
			Renderable:           false,
			NonEmpty:             false,
			FailureDetail:        fmt.Sprintf("load workflow bundle: %v", err),
		}}
	}

	templateRoot, err := resolveTemplateRootForB036Audit(configPath)
	if err != nil {
		return []b036AuditRecord{{
			Bundle:               bundle.Name,
			ResolvedTemplateRoot: bundle.ExpectedTemplateDir,
			Renderable:           false,
			NonEmpty:             false,
			FailureDetail:        fmt.Sprintf("resolve template root: %v", err),
		}}
	}

	if err := configureProductionPromptResolutionForB036Audit(configPath); err != nil {
		return []b036AuditRecord{{
			Bundle:               bundle.Name,
			ResolvedTemplateRoot: templateRoot,
			Renderable:           false,
			NonEmpty:             false,
			FailureDetail:        fmt.Sprintf("configure prompt resolution: %v", err),
		}}
	}
	actionSvc, err := config.NewActionService(configPath)
	if err != nil {
		return []b036AuditRecord{{
			Bundle:               bundle.Name,
			ResolvedTemplateRoot: templateRoot,
			Renderable:           false,
			NonEmpty:             false,
			FailureDetail:        fmt.Sprintf("create action service: %v", err),
		}}
	}

	var records []b036AuditRecord
	if templateRoot != bundle.ExpectedTemplateDir {
		records = append(records, b036AuditRecord{
			Bundle:               bundle.Name,
			ResolvedTemplateRoot: templateRoot,
			Renderable:           false,
			NonEmpty:             false,
			FailureDetail:        fmt.Sprintf("resolution-root mismatch: got %s want %s", templateRoot, bundle.ExpectedTemplateDir),
		})
	}

	for _, entityType := range workflowcfg.KnownLevels {
		raw := multi.RawForLevel(entityType)
		if raw == nil || len(raw.Steps) == 0 {
			continue
		}
		entityActionSvc := actionSvc.ForEntity(entityType)

		if strings.TrimSpace(raw.Start) == "" {
			records = append(records, b036AuditRecord{
				Bundle:               bundle.Name,
				EntityType:           entityType,
				ResolvedTemplateRoot: templateRoot,
				FailureDetail:        "workflow start is empty",
			})
			continue
		}

		statuses, unreachable := reachablePromptStepsFromStart(raw)
		for _, status := range unreachable {
			step := raw.Steps[status]
			records = append(records, b036AuditRecord{
				Bundle:               bundle.Name,
				EntityType:           entityType,
				Status:               status,
				Step:                 status,
				Phase:                step.Phase,
				PromptReference:      step.Prompt,
				ResolvedTemplateRoot: templateRoot,
				FailureDetail:        fmt.Sprintf("workflow step is not reachable from start %q", raw.Start),
			})
		}

		for _, status := range statuses {
			step := raw.Steps[status]
			if step == nil || strings.TrimSpace(step.Prompt) == "" {
				continue
			}

			record := b036AuditRecord{
				Bundle:               bundle.Name,
				EntityType:           entityType,
				Status:               status,
				Step:                 status,
				Phase:                step.Phase,
				PromptReference:      step.Prompt,
				ResolvedTemplateRoot: templateRoot,
			}

			if hasBadPromptPath(step.Prompt) {
				record.FailureDetail = fmt.Sprintf("bad prompt path: %s", step.Prompt)
				records = append(records, record)
				continue
			}

			promptPath := filepath.Join(templateRoot, filepath.FromSlash(step.Prompt))
			if _, statErr := os.Stat(promptPath); statErr == nil {
				record.Exists = true
			} else {
				record.FailureDetail = buildMissingPromptFailure(step.Prompt, promptPath)
				records = append(records, record)
				continue
			}

			populated, renderErr := entityActionSvc.GetStatusActionPopulated(context.Background(), status, promptVarsForB036Audit(entityType, status))
			if renderErr != nil {
				record.Exists = true
				record.FailureDetail = fmt.Sprintf("production render failed: %v", renderErr)
				records = append(records, record)
				continue
			}
			if populated == nil {
				record.Exists = true
				record.FailureDetail = "production render returned nil action"
				records = append(records, record)
				continue
			}

			record.Exists = true
			record.Renderable = true
			record.NonEmpty = strings.TrimSpace(populated.Instruction) != ""
			if !record.NonEmpty {
				record.FailureDetail = "rendered prompt trimmed to empty output"
			}
			records = append(records, record)
		}
	}

	return records
}

func writeB036AuditConfig(t *testing.T, configPath string, bundle b036BundleConfig) {
	t.Helper()

	raw := map[string]any{
		"color_enabled":            false,
		"interactive_mode":         false,
		"require_rejection_reason": false,
		"workflow_config":          bundle.WorkflowConfig,
	}
	if bundle.SharkDataPath != "" {
		raw["shark_data_path"] = bundle.SharkDataPath
	}

	body, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(configPath, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", configPath, err)
	}
}

func resolveTemplateRootForB036Audit(configPath string) (string, error) {
	cfg, err := config.NewManager(configPath).Load()
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	templateDir := ""
	sharkDataPath := config.DefaultSharkDataPath
	if cfg != nil {
		templateDir = cfg.GetTemplateDirectory()
		sharkDataPath = cfg.GetSharkDataPath()
	}

	workflows, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return "", fmt.Errorf("load multi-level workflow: %w", err)
	}
	if workflows != nil && workflows.TemplateDirectory != nil && *workflows.TemplateDirectory != "" {
		templateDir = *workflows.TemplateDirectory
	}

	if templateDir != "" {
		return templateDir, nil
	}
	return filepath.Join(sharkDataPath, "prompts"), nil
}

func configureProductionPromptResolutionForB036Audit(configPath string) error {
	cfg, err := config.NewManager(configPath).Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	templateDir := ""
	sharkDataPath := config.DefaultSharkDataPath
	if cfg != nil {
		templateDir = cfg.GetTemplateDirectory()
		sharkDataPath = cfg.GetSharkDataPath()
	}

	workflows, err := config.LoadMultiLevelWorkflow(configPath)
	if err != nil {
		return fmt.Errorf("load multi-level workflow: %w", err)
	}
	if workflows != nil && workflows.TemplateDirectory != nil && *workflows.TemplateDirectory != "" {
		templateDir = *workflows.TemplateDirectory
	}

	templates.SetConfiguredTemplateDir(templateDir)
	templates.SetConfiguredSharkDataPath(sharkDataPath)
	templates.ResetOrchestratorEngine()
	return nil
}

func promptVarsForB036Audit(entityType, status string) map[string]string {
	vars := basePromptVarsForB036Audit()

	switch entityType {
	case "epic":
		mergePromptVars(vars, cfgtemplate.EpicPlaceholders(fakeEpicForB036Audit(status)))
	case "feature":
		mergePromptVars(vars, cfgtemplate.FeaturePlaceholders(fakeFeatureForB036Audit(status)))
	case "task":
		mergePromptVars(vars, cfgtemplate.TaskPlaceholders(fakeTaskForB036Audit(status)))
	case "bug":
		mergePromptVars(vars, cfgtemplate.BugPlaceholders(fakeBugForB036Audit(status)))
	case "change":
		mergePromptVars(vars, cfgtemplate.ChangeCardPlaceholders(fakeChangeCardForB036Audit(status)))
	case "tech_debt":
		mergePromptVars(vars, cfgtemplate.TechDebtPlaceholders(fakeTechDebtForB036Audit(status)))
	case "sprint":
		mergePromptVars(vars, map[string]string{
			"id":          "S01",
			"key":         "S01",
			"title":       "Sprint prompt coverage",
			"status":      status,
			"entity_type": "sprint",
			"file_path":   "docs/sprints/S01.md",
		})
	}

	vars["status"] = status
	return vars
}

func basePromptVarsForB036Audit() map[string]string {
	return map[string]string{
		"advance_note_type":   "implementation-note",
		"advance_summary":     "Advance after completing the rendered prompt instructions.",
		"blocked_reason":      "External dependency on a staging API rollout.",
		"business_value":      "high",
		"category":            "architecture",
		"complexity_tier":     "complex",
		"completion_notes":    "Implemented the change and verified the behavior.",
		"context_data":        "Key decision: keep workflow loading isolated to a temp project.",
		"depends_on":          "T-E01-F01-002,T-E01-F01-003",
		"description":         "B036 exhaustive prompt crawl coverage entity.",
		"doc_friendly_name":   "feature spec",
		"epic_id":             "E01",
		"execution_order":     "1",
		"fail_reason_summary": "A validation gate found a workflow issue requiring revision.",
		"fail_target_status":  "draft",
		"feature_key":         "E01-F01",
		"file_path":           "docs/plan/E01-prompt-audit/E01-F01-coverage/tasks/T-E01-F01-001.md",
		"files_changed":       "internal/templates/b036_prompt_crawl_test.go",
		"id":                  "T-E01-F01-001",
		"impact_analysis":     "Touches prompt resolution, workflow loading, and dispatch coverage.",
		"is_resume":           "true",
		"justification":       "Ensures prompt mapping defects are caught mechanically.",
		"latest_note":         "Most recent review note from the previous step.",
		"linked_entity_key":   "E01-F01",
		"linked_entity_type":  "feature",
		"notes_summary":       "Prior notes captured a renderer root mismatch.",
		"parent_title":        "Prompt coverage parent entity",
		"previous_status":     "draft",
		"primary_doc":         "docs/plan/E01-prompt-audit/E01-F01-coverage/spec.md",
		"priority":            "5",
		"related_docs":        "docs/plan/E01-prompt-audit/E01-F01-coverage/spec.md, docs/plan/E01-prompt-audit/E01-F01-coverage/implementation.md",
		"related_tasks":       "T-E01-F01-002, T-E01-F01-003",
		"review_base":         "docs/review/E01-F01",
		"rollback_plan":       "Revert the workflow YAML and prompt path change together.",
		"severity":            "high",
		"sibling_progress":    "2 of 3 sibling tasks completed",
		"task_id":             "T-E01-F01-001",
		"title":               "Prompt coverage fixture",
	}
}

func fakeEpicForB036Audit(status string) *models.Epic {
	filePath := "docs/plan/E01-prompt-audit/epic.md"
	slug := "prompt-audit"
	businessValue := models.PriorityHigh
	size := 5
	return &models.Epic{
		BaseEntity: models.BaseEntity{
			Key:         "E01",
			Title:       "Prompt audit epic",
			Description: strPtrForB036Audit("Audit every prompt-bearing workflow step."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:        models.EpicStatus(status),
		Priority:      models.PriorityHigh,
		BusinessValue: &businessValue,
	}
}

func fakeFeatureForB036Audit(status string) *models.Feature {
	filePath := "docs/plan/E01-prompt-audit/E01-F01-coverage/feature.md"
	slug := "prompt-coverage"
	order := 1
	size := 3
	return &models.Feature{
		BaseEntity: models.BaseEntity{
			Key:         "E01-F01",
			Title:       "Prompt coverage feature",
			Description: strPtrForB036Audit("Exercise prompt rendering across feature phases."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 10, 30, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:         models.FeatureStatus(status),
		ExecutionOrder: &order,
	}
}

func fakeTaskForB036Audit(status string) *models.Task {
	filePath := "docs/plan/E01-prompt-audit/E01-F01-coverage/tasks/T-E01-F01-001.md"
	slug := "implement-prompt-audit"
	agentType := "developer"
	order := 1
	blockedReason := "Waiting on prompt root confirmation."
	dependsOn := "[]"
	completionNotes := "Added exhaustive prompt crawl coverage."
	filesChanged := "[\"internal/templates/b036_prompt_crawl_test.go\"]"
	size := 2
	return &models.Task{
		BaseEntity: models.BaseEntity{
			Key:         "T-E01-F01-001",
			Title:       "Implement prompt audit coverage",
			Description: strPtrForB036Audit("Render every prompt-bearing workflow step."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 11, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 11, 30, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:          models.TaskStatus(status),
		AgentType:       &agentType,
		Priority:        5,
		DependsOn:       &dependsOn,
		BlockedReason:   &blockedReason,
		ExecutionOrder:  &order,
		CompletionNotes: &completionNotes,
		FilesChanged:    &filesChanged,
	}
}

func fakeBugForB036Audit(status string) *models.Bug {
	filePath := "docs/plan/bugs/B001.md"
	slug := "prompt-audit-bug"
	linkedType := "feature"
	linkedKey := "E01-F01"
	size := 2
	return &models.Bug{
		BaseEntity: models.BaseEntity{
			Key:         "B001",
			Title:       "Prompt audit bug",
			Description: strPtrForB036Audit("Prompt resolution regression fixture."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 12, 15, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:           models.BugStatus(status),
		Severity:         models.BugSeverityHigh,
		LinkedEntityType: &linkedType,
		LinkedEntityKey:  &linkedKey,
	}
}

func fakeChangeCardForB036Audit(status string) *models.ChangeCard {
	filePath := "docs/plan/changes/CC-001.md"
	slug := "prompt-audit-change"
	requestedBy := "engineering"
	assignedTo := "codex"
	justification := "Need deterministic workflow prompt coverage."
	impact := "Affects test coverage and audit artifacts only."
	rollback := "Delete the audit test and verification artifacts."
	size := 3
	return &models.ChangeCard{
		BaseEntity: models.BaseEntity{
			Key:         "CC-001",
			Title:       "Prompt audit change card",
			Description: strPtrForB036Audit("Prompt dispatch coverage fixture."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 13, 20, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:         models.ChangeCardStatus(status),
		Priority:       4,
		RequestedBy:    &requestedBy,
		AssignedTo:     &assignedTo,
		Justification:  &justification,
		ImpactAnalysis: &impact,
		RollbackPlan:   &rollback,
	}
}

func fakeTechDebtForB036Audit(status string) *models.TechDebt {
	filePath := "docs/plan/tech-debt/TD-001.md"
	slug := "prompt-audit-tech-debt"
	effort := "medium"
	size := 5
	return &models.TechDebt{
		BaseEntity: models.BaseEntity{
			Key:         "TD-001",
			Title:       "Prompt audit tech debt",
			Description: strPtrForB036Audit("Backfill prompt coverage debt."),
			FilePath:    &filePath,
			Slug:        &slug,
			CreatedAt:   time.Date(2026, 7, 6, 14, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 7, 6, 14, 10, 0, 0, time.UTC),
			Size:        &size,
		},
		Status:         models.TechDebtStatus(status),
		Category:       models.TechDebtCategoryArchitecture,
		Severity:       models.TechDebtSeverityHigh,
		EffortEstimate: &effort,
	}
}

func mergePromptVars(dst, src map[string]string) {
	for key, value := range src {
		if strings.TrimSpace(value) == "" {
			continue
		}
		dst[key] = value
	}
}

func hasBadPromptPath(promptRef string) bool {
	if filepath.IsAbs(promptRef) {
		return true
	}
	cleaned := filepath.Clean(filepath.FromSlash(promptRef))
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return true
	}
	ext := filepath.Ext(promptRef)
	return ext != ".md" && ext != ".tmpl"
}

func buildMissingPromptFailure(promptRef, promptPath string) string {
	alt := alternatePromptPath(promptPath)
	if alt != "" {
		return fmt.Sprintf("mismatched extension: referenced %s but only %s exists", promptRef, filepath.ToSlash(alt))
	}
	return fmt.Sprintf("missing prompt file: %s", filepath.ToSlash(promptPath))
}

func alternatePromptPath(promptPath string) string {
	ext := filepath.Ext(promptPath)
	switch ext {
	case ".md":
		alt := strings.TrimSuffix(promptPath, ext) + ".tmpl"
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	case ".tmpl":
		alt := strings.TrimSuffix(promptPath, ext) + ".md"
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return ""
}

func failingAuditRecords(records []b036AuditRecord) []b036AuditRecord {
	failures := make([]b036AuditRecord, 0)
	for _, record := range records {
		if !record.Exists || !record.Renderable || !record.NonEmpty || record.FailureDetail != "" {
			failures = append(failures, record)
		}
	}
	return failures
}

func reachablePromptStepsFromStart(wf *workflowcfg.WorkflowConfig) ([]string, []string) {
	if wf == nil || len(wf.Steps) == 0 {
		return nil, nil
	}

	visited := make(map[string]struct{}, len(wf.Steps))
	queue := []string{wf.Start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, ok := visited[current]; ok {
			continue
		}
		step, ok := wf.GetStep(current)
		if !ok || step == nil {
			continue
		}
		visited[current] = struct{}{}

		nextSteps := make([]string, 0, len(step.Outcomes))
		for _, target := range step.Outcomes {
			nextSteps = append(nextSteps, target)
		}
		sort.Strings(nextSteps)
		queue = append(queue, nextSteps...)
	}

	reachable := make([]string, 0, len(visited))
	unreachable := make([]string, 0)
	for status, step := range wf.Steps {
		if step == nil || strings.TrimSpace(step.Prompt) == "" {
			continue
		}
		if _, ok := visited[status]; ok {
			reachable = append(reachable, status)
			continue
		}
		unreachable = append(unreachable, status)
	}
	sort.Strings(reachable)
	sort.Strings(unreachable)
	return reachable, unreachable
}

func writeB036Artifacts(t *testing.T, artifactDir string, report b036AuditReport) {
	t.Helper()

	jsonBody, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal audit report: %v", err)
	}
	jsonBody = append(jsonBody, '\n')
	if err := os.WriteFile(filepath.Join(artifactDir, b036AuditJSONName), jsonBody, 0o644); err != nil {
		t.Fatalf("write audit json: %v", err)
	}

	summary := buildB036AuditSummary(report)
	if err := os.WriteFile(filepath.Join(artifactDir, b036AuditSummaryName), []byte(summary), 0o644); err != nil {
		t.Fatalf("write audit summary: %v", err)
	}
}

func buildB036AuditSummary(report b036AuditReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# B036 Exhaustive Prompt Crawl Summary\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", report.GeneratedAt)
	fmt.Fprintf(&b, "Prompt-bearing steps audited: %d\n\n", report.RecordCount)

	bundleNames := make([]string, 0, len(report.Bundles))
	for name := range report.Bundles {
		bundleNames = append(bundleNames, name)
	}
	sort.Strings(bundleNames)
	b.WriteString("## Bundles Audited\n\n")
	for _, name := range bundleNames {
		fmt.Fprintf(&b, "- %s: %d prompt-bearing steps\n", name, report.Bundles[name])
	}
	b.WriteString("\n")

	writeSummarySection(&b, "Missing Prompt Files", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if r.Exists || !strings.Contains(r.FailureDetail, "missing prompt file") {
			return ""
		}
		return fmt.Sprintf("%s | %s | %s | %s", r.Bundle, r.EntityType, r.Status, r.FailureDetail)
	}))
	writeSummarySection(&b, "Bad Prompt Paths", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if !strings.Contains(r.FailureDetail, "bad prompt path") {
			return ""
		}
		return fmt.Sprintf("%s | %s | %s | %s", r.Bundle, r.EntityType, r.Status, r.FailureDetail)
	}))
	writeSummarySection(&b, "Mismatched .md vs .tmpl References", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if !strings.Contains(r.FailureDetail, "mismatched extension") {
			return ""
		}
		return fmt.Sprintf("%s | %s | %s | %s", r.Bundle, r.EntityType, r.Status, r.FailureDetail)
	}))
	writeSummarySection(&b, "Resolution-Root Mismatches", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if !strings.Contains(r.FailureDetail, "resolution-root mismatch") {
			return ""
		}
		return fmt.Sprintf("%s | %s", r.Bundle, r.FailureDetail)
	}))
	writeSummarySection(&b, "Include Failures", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if !strings.Contains(r.FailureDetail, "include") {
			return ""
		}
		return fmt.Sprintf("%s | %s | %s | %s", r.Bundle, r.EntityType, r.Status, r.FailureDetail)
	}))
	writeSummarySection(&b, "Steps With Empty Rendered Instructions", uniqueSummaryLines(report.Records, func(r b036AuditRecord) string {
		if r.NonEmpty || !strings.Contains(r.FailureDetail, "rendered prompt trimmed to empty") {
			return ""
		}
		return fmt.Sprintf("%s | %s | %s | %s", r.Bundle, r.EntityType, r.Status, r.FailureDetail)
	}))

	return b.String()
}

func uniqueSummaryLines(records []b036AuditRecord, extract func(b036AuditRecord) string) []string {
	seen := map[string]struct{}{}
	lines := make([]string, 0)
	for _, record := range records {
		line := extract(record)
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	return lines
}

func writeSummarySection(b *strings.Builder, title string, lines []string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(lines) == 0 {
		b.WriteString("- None.\n\n")
		return
	}
	for _, line := range lines {
		fmt.Fprintf(b, "- %s\n", line)
	}
	b.WriteString("\n")
}

func findRepoRootForB036PromptAudit(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root from %s", wd)
		}
		dir = parent
	}
}

func strPtrForB036Audit(value string) *string {
	return &value
}
