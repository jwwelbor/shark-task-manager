package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// E3 — `shark next` agent-body auto-inline (per 2026-05-10 rendering decision)
// ============================================================================

// setupAgentFixture lays down a minimal shark-data/ tree with one agent file
// (and optionally an override) and returns the data root.
func setupAgentFixture(t *testing.T, agentType, body string, overrideBody string) string {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, "shark-data")
	require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "agents"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dataRoot, "agents", agentType+".md"),
		[]byte(body),
		0644,
	))
	if overrideBody != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(dataRoot, "overrides", "agents"), 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(dataRoot, "overrides", "agents", agentType+".md"),
			[]byte(overrideBody),
			0644,
		))
	}
	return dataRoot
}

func TestLoadAgentBodyForInline_EmptyRootFallsBackToEmbed(t *testing.T) {
	// Empty root (zero-config mode) should fall back to the embedded canonical
	// agent tree rather than returning false. The embedded "qa" agent is known
	// to exist, so ok must be true and the body non-empty.
	got, ok := LoadAgentBodyForInline("", "qa")
	assert.True(t, ok, "zero-config mode should fall back to embedded agent body")
	assert.NotEmpty(t, got)
}

func TestLoadAgentBodyForInline_EmptyAgentTypeReturnsFalse(t *testing.T) {
	root := t.TempDir()
	got, ok := LoadAgentBodyForInline(root, "")
	assert.False(t, ok)
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_AgentFileFound(t *testing.T) {
	body := "You are the QA agent. Tools: Read, Bash, Grep."
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok, "agent file should be resolved")
	assert.Equal(t, body, got)
}

func TestLoadAgentBodyForInline_FrontmatterStripped(t *testing.T) {
	// Authors may put YAML frontmatter on agent files (model, allowed-tools,
	// description). The resolver strips it before returning the body — the
	// frontmatter is metadata, not content for the inlined prompt.
	body := "---\nname: qa\nmodel: opus\nallowed-tools: Read, Bash\n---\nQA agent persona body."
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)
	assert.Equal(t, "QA agent persona body.", got)
	assert.NotContains(t, got, "name: qa", "frontmatter must be stripped")
	assert.NotContains(t, got, "---", "frontmatter delimiters must be stripped")
}

func TestLoadAgentBodyForInline_OverrideWins(t *testing.T) {
	defaultBody := "DEFAULT qa agent body"
	overrideBody := "OVERRIDE qa agent body"
	root := setupAgentFixture(t, "qa", defaultBody, overrideBody)

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)
	assert.Equal(t, overrideBody, got, "override under overrides/agents/ must fully replace the default")
}

func TestLoadAgentBodyForInline_AgentMissingReturnsFalse(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "shark-data", "agents"), 0755))

	got, ok := LoadAgentBodyForInline(filepath.Join(root, "shark-data"), "ghost-agent-that-doesnt-exist")
	assert.False(t, ok, "missing agent should return ok=false (non-fatal)")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_EmptyBodyTreatedAsMissing(t *testing.T) {
	// An agent file with only frontmatter (no body content) shouldn't
	// produce an empty inline — return ok=false so callers don't prepend
	// useless whitespace + separator.
	body := "---\nname: stub\n---\n"
	root := setupAgentFixture(t, "stub", body, "")

	got, ok := LoadAgentBodyForInline(root, "stub")
	assert.False(t, ok, "agent file with no body after frontmatter strip should report not inlined")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_PathTraversalRejected(t *testing.T) {
	root := t.TempDir()
	got, ok := LoadAgentBodyForInline(root, "../../../etc/passwd")
	assert.False(t, ok, "path-traversal agent type must not resolve")
	assert.Equal(t, "", got)
}

func TestLoadAgentBodyForInline_PrependFormatExample(t *testing.T) {
	// Documents the prepend format used in runNext so future reviewers see
	// the contract: agent body, blank line, --- separator, blank line, then
	// the action prompt.
	body := "QA persona"
	root := setupAgentFixture(t, "qa", body, "")

	got, ok := LoadAgentBodyForInline(root, "qa")
	require.True(t, ok)

	actionPrompt := "Run QA on E07-F02-001..."
	combined := got + "\n\n---\n\n" + actionPrompt

	assert.True(t, strings.HasPrefix(combined, "QA persona"))
	assert.Contains(t, combined, "\n\n---\n\n")
	assert.True(t, strings.HasSuffix(combined, actionPrompt))
}

// findRepoPromptsDir walks up from the test working directory looking for the
// canonical prompts directory. It prefers the committed embedded canonical at
// internal/sharkdata/default_data/prompts so the test suite is hermetic and
// cannot be skewed by an untracked local shark-data/ extraction. It falls back
// to shark-data/prompts only when the canonical source tree is unavailable.
func findRepoPromptsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		// Prefer the embedded canonical (always present in the repo checkout).
		if candidate := filepath.Join(dir, "internal", "sharkdata", "default_data", "prompts"); isDirExist(candidate) {
			return candidate
		}
		// Fall back to the deployed copy when the canonical tree is unavailable.
		if candidate := filepath.Join(dir, "shark-data", "prompts"); isDirExist(candidate) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate prompts directory (shark-data/prompts or internal/sharkdata/default_data/prompts) walking up from %s", wd)
		}
		dir = parent
	}
}

// isDirExist returns true when path exists and is a directory.
func isDirExist(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// TestRunNext_InlinesSkillContent is the F02 AC #2 end-to-end check: the
// shipped feature/assessment.md prompt must produce a rendered output that
// contains the selected assessment workflow content inlined via {{include:}},
// not a path reference.
func TestRunNext_InlinesSkillContent(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err, "shipped prompts must parse with includes resolved")

	out, err := renderer.Render("feature/assessment.md", map[string]string{
		"id":        "E32-F02",
		"title":     "Engine — includes",
		"file_path": "docs/plan/E32/E32-F02/E32-F02.md",
		"epic_id":   "E32",
		"is_resume": "false",
	})
	require.NoError(t, err)

	// The workflow body has a stable H1 that proves the file was inlined,
	// not merely referenced by path.
	assert.Contains(t, out, "# Workflow: Complexity Triage",
		"rendered prompt must inline the selected assessment workflow via {{include:}}")
	assert.NotContains(t, out, "Load skill: ",
		"path-reference idiom should not appear in the rendered prompt")
}

func TestRunNext_ResumePreambleIncludeIsShortAndActionable(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)

	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	vars := standardVars()
	vars["is_resume"] = "true"

	out, err := renderer.Render("feature/assessment.md", vars)
	require.NoError(t, err)

	assert.Contains(t, out, "RESUME CONTEXT:")
	assert.Contains(t, out, "shark claims")
	assert.Contains(t, out, "continue it instead of restarting")
	assert.Contains(t, out, "Do not advance status just because code or docs exist")

	preamble := strings.SplitN(out, "Assess feature", 2)[0]
	nonEmptyLines := 0
	for _, line := range strings.Split(preamble, "\n") {
		if strings.TrimSpace(line) != "" {
			nonEmptyLines++
		}
	}
	assert.LessOrEqual(t, nonEmptyLines, 3, "resume preamble should stay short")
}

type fixedNextTransitioner struct {
	info *services.NextStatusInfo
}

func (f fixedNextTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	return &services.TransitionResult{ToStatus: targetStatus}, nil
}

func (f fixedNextTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return f.info, nil
}

type fixedNextPlaceholders struct {
	vars map[string]string
}

func (f fixedNextPlaceholders) GeneratePlaceholders(ctx context.Context, key string) (map[string]string, error) {
	return f.vars, nil
}

type cascadeAutoAdvanceTransitioner struct {
	currentStatus  string
	transitionedTo []string
}

func (c *cascadeAutoAdvanceTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	c.currentStatus = targetStatus
	c.transitionedTo = append(c.transitionedTo, targetStatus)
	return &services.TransitionResult{ToStatus: targetStatus}, nil
}

func (c *cascadeAutoAdvanceTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	switch c.currentStatus {
	case "active":
		// Realistic production shape: the shipped feature workflow's "active"
		// status exposes every unique outcome target, pass-first — never
		// exactly one transition. (StatusFlow[active] = [code_review
		// task_review blocked on_hold] in the canonical feature.yaml.)
		return &services.NextStatusInfo{
			EntityKey:     key,
			CurrentStatus: "active",
			AvailableTransitions: []services.TransitionInfoWithAction{
				{TransitionInfo: workflow.TransitionInfo{TargetStatus: "code_review"}},
				{TransitionInfo: workflow.TransitionInfo{TargetStatus: "task_review"}},
				{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
				{TransitionInfo: workflow.TransitionInfo{TargetStatus: "on_hold"}},
			},
		}, nil
	case "code_review":
		return &services.NextStatusInfo{
			EntityKey:     key,
			CurrentStatus: "code_review",
		}, nil
	default:
		return &services.NextStatusInfo{
			EntityKey:     key,
			CurrentStatus: c.currentStatus,
		}, nil
	}
}

func TestResolveNext_ReturnsSelfContainedPrompt(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	vars := standardVars()
	vars["id"] = "E99-F01"
	vars["key"] = "E99-F01"
	vars["feature_id"] = "E99-F01"
	vars["title"] = "Self-contained dispatch contract"
	vars["file_path"] = "docs/plan/E99/E99-F01/feature.md"
	vars["is_resume"] = "false"

	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, gotVars map[string]string) (*action.PopulatedAction, error) {
			require.Equal(t, "assessment", status)
			rendered, err := renderer.Render("feature/assessment.md", gotVars)
			require.NoError(t, err)
			return &action.PopulatedAction{
				Action:      "spawn_agent",
				AgentType:   "researcher",
				Provider:    "anthropic",
				Model:       "haiku",
				Instruction: rendered,
			}, nil
		},
	}

	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"feature": {
				transitioner: fixedNextTransitioner{info: &services.NextStatusInfo{
					CurrentStatus: "assessment",
					IsTerminal:    false,
					AvailableTransitions: []services.TransitionInfoWithAction{
						{TransitionInfo: workflow.TransitionInfo{TargetStatus: "research"}},
					},
				}},
				generator: fixedNextPlaceholders{vars: vars},
				actionSvc: actionSvc,
			},
		},
		actionSvcRoot: actionSvc,
	}

	resp, err := resolveNext(context.Background(), cache, "feature", "E99-F01", 0)
	require.NoError(t, err)

	assert.Equal(t, "spawn_agent", resp.Action)
	assert.Equal(t, "researcher", resp.AgentType)
	assert.Equal(t, "anthropic", resp.Provider)
	assert.Equal(t, "haiku", resp.Model)
	assert.True(t, strings.HasPrefix(resp.Prompt, "PARENT LOOP OWNERSHIP CONTRACT:"),
		"response.prompt must lead with the worker ownership contract")
	assert.Contains(t, resp.Prompt, "Operate in single-worker mode by default.",
		"response.prompt must explicitly constrain nested worker spawning by default")
	assert.Contains(t, resp.Prompt, "# Researcher Agent",
		"response.prompt must include Shark agent persona content")
	assert.Contains(t, resp.Prompt, "# Workflow: Complexity Triage",
		"response.prompt must include workflow skill content")
	assert.Contains(t, resp.Prompt, `Assess feature E99-F01: "Self-contained dispatch contract".`,
		"response.prompt must include rendered workflow prompt content")
	assert.NotContains(t, resp.Prompt, "{{include:",
		"response.prompt must not leave include directives for the harness")
	assert.Less(t,
		strings.Index(resp.Prompt, "# Researcher Agent"),
		strings.Index(resp.Prompt, "# Workflow: Complexity Triage"),
		"agent persona should be prepended before the workflow prompt")
}

func TestResolveNext_CascadeParentAutoAdvancesWhenAllChildrenAreTerminal(t *testing.T) {
	origTransitionerBuilder := nextBuildTransitioner
	origPlaceholderBuilder := nextBuildPlaceholderGenerator
	origDescribeChildren := nextDescribeDispatchableChildren
	defer func() {
		nextBuildTransitioner = origTransitionerBuilder
		nextBuildPlaceholderGenerator = origPlaceholderBuilder
		nextDescribeDispatchableChildren = origDescribeChildren
	}()

	transitioner := &cascadeAutoAdvanceTransitioner{currentStatus: "active"}
	nextBuildTransitioner = func(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
		return transitioner, nil
	}
	nextBuildPlaceholderGenerator = func(_ context.Context, entityType string) runner.PlaceholderGenerator {
		return fixedNextPlaceholders{vars: map[string]string{
			"id":         "E03-F02",
			"feature_id": "E03-F02",
			"title":      "Unified Local Search And Session Browse Entry",
			"key":        "E03-F02",
		}}
	}
	nextDescribeDispatchableChildren = func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
		return services.CascadeChildrenState{
			TotalChildren:       4,
			NonTerminalChildren: 0,
		}, nil
	}

	actionSvc := &action.MockActionService{
		GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*action.PopulatedAction, error) {
			switch status {
			case "active":
				return &action.PopulatedAction{
					Action:      "cascade",
					Instruction: "delegate child work",
				}, nil
			case "code_review":
				return &action.PopulatedAction{
					Action:      "spawn_agent",
					AgentType:   "tech-lead",
					Provider:    "anthropic",
					Model:       "sonnet",
					Instruction: "review completed feature E03-F02",
				}, nil
			default:
				return nil, nil
			}
		},
	}

	cache := &nextAdapterCache{
		entries:       map[string]*nextAdapters{},
		actionSvcRoot: actionSvc,
	}

	resp, err := resolveNext(context.Background(), cache, "feature", "E03-F02", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"code_review"}, transitioner.transitionedTo)
	assert.Equal(t, "E03-F02", resp.EntityKey)
	assert.Equal(t, "feature", resp.EntityType)
	assert.Equal(t, "code_review", resp.Status)
	assert.Equal(t, "spawn_agent", resp.Action)
	assert.Equal(t, "tech-lead", resp.AgentType)
	assert.Contains(t, resp.Prompt, "review completed feature E03-F02")
}

// failingCascadeTransitioner reports "active" with no forward transitions, or
// fails TransitionStatus, to exercise autoAdvanceCascadeParent's guard paths.
type failingCascadeTransitioner struct {
	transitions   []services.TransitionInfoWithAction
	transitionErr error
}

func (f *failingCascadeTransitioner) TransitionStatus(ctx context.Context, key, targetStatus string, opts services.TransitionOptions) (*services.TransitionResult, error) {
	if f.transitionErr != nil {
		return nil, f.transitionErr
	}
	return &services.TransitionResult{ToStatus: targetStatus}, nil
}

func (f *failingCascadeTransitioner) GetNextStatus(ctx context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{
		EntityKey:            key,
		CurrentStatus:        "active",
		AvailableTransitions: f.transitions,
	}, nil
}

// TestResolveNext_CascadeGuardBranches covers tryCascade's non-happy paths:
// a childless parent pauses quietly; an all-terminal parent with no forward
// transition pauses with a descriptive error; a failing transition propagates.
func TestResolveNext_CascadeGuardBranches(t *testing.T) {
	forwardTransitions := []services.TransitionInfoWithAction{
		{TransitionInfo: workflow.TransitionInfo{TargetStatus: "code_review"}},
		{TransitionInfo: workflow.TransitionInfo{TargetStatus: "blocked"}},
	}

	tests := []struct {
		name          string
		childrenState services.CascadeChildrenState
		transitioner  *failingCascadeTransitioner
		wantAction    string
		wantErrField  string // substring of resp.Error; "" means must be empty
		wantErr       bool   // hard error returned from resolveNext
	}{
		{
			name:          "childless parent pauses without error",
			childrenState: services.CascadeChildrenState{TotalChildren: 0, NonTerminalChildren: 0},
			transitioner:  &failingCascadeTransitioner{transitions: forwardTransitions},
			wantAction:    "pause",
		},
		{
			name:          "non-terminal children but none dispatchable pauses",
			childrenState: services.CascadeChildrenState{TotalChildren: 3, NonTerminalChildren: 2},
			transitioner:  &failingCascadeTransitioner{transitions: forwardTransitions},
			wantAction:    "pause",
		},
		{
			name:          "all terminal but no forward transition pauses with error",
			childrenState: services.CascadeChildrenState{TotalChildren: 3, NonTerminalChildren: 0},
			transitioner:  &failingCascadeTransitioner{transitions: nil},
			wantAction:    "pause",
			wantErrField:  "no forward transition",
		},
		{
			name:          "transition failure propagates as hard error",
			childrenState: services.CascadeChildrenState{TotalChildren: 3, NonTerminalChildren: 0},
			transitioner: &failingCascadeTransitioner{
				transitions:   forwardTransitions,
				transitionErr: errors.New("simulated transition failure"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTransitionerBuilder := nextBuildTransitioner
			origPlaceholderBuilder := nextBuildPlaceholderGenerator
			origDescribeChildren := nextDescribeDispatchableChildren
			defer func() {
				nextBuildTransitioner = origTransitionerBuilder
				nextBuildPlaceholderGenerator = origPlaceholderBuilder
				nextDescribeDispatchableChildren = origDescribeChildren
			}()

			nextBuildTransitioner = func(_ context.Context, entityType string) (runner.EntityTransitioner, error) {
				return tt.transitioner, nil
			}
			nextBuildPlaceholderGenerator = func(_ context.Context, entityType string) runner.PlaceholderGenerator {
				return fixedNextPlaceholders{vars: map[string]string{"id": "E03-F02", "key": "E03-F02"}}
			}
			nextDescribeDispatchableChildren = func(ctx context.Context, entityType, key string) (services.CascadeChildrenState, error) {
				return tt.childrenState, nil
			}

			actionSvc := &action.MockActionService{
				GetStatusActionPopulatedFunc: func(ctx context.Context, status string, vars map[string]string) (*action.PopulatedAction, error) {
					return &action.PopulatedAction{Action: "cascade", Instruction: "delegate child work"}, nil
				},
			}
			cache := &nextAdapterCache{entries: map[string]*nextAdapters{}, actionSvcRoot: actionSvc}

			resp, err := resolveNext(context.Background(), cache, "feature", "E03-F02", 0)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cascade completion advance")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantAction, resp.Action)
			if tt.wantErrField == "" {
				assert.Empty(t, resp.Error)
			} else {
				assert.Contains(t, resp.Error, tt.wantErrField)
			}
		})
	}
}

func TestRenderedDispatchPromptsDoNotTellWorkersToTransitionSharkState(t *testing.T) {
	promptsDir := findRepoPromptsDir(t)
	renderer, err := templates.NewOrchestratorRenderer(promptsDir)
	require.NoError(t, err)

	vars := standardVars()
	dispatchPrompts := []string{
		"epic/assessment.md",
		"epic/decomposition.md",
		"epic/design.md",
		"epic/feature_review.md",
		"epic/refinement.md",
		"epic/research.md",
		"feature/assessment.md",
		"feature/approval.md",
		"feature/code_review.md",
		"feature/qa.md",
		"feature/research.md",
		"feature/specification.md",
		"feature/task_generation.md",
		"feature/task_review.md",
		"feature/test_planning.md",
		"task/development.md",
		"bug/development.md",
		"change/development.md",
		"tech_debt/identified.md",
		"tech_debt/in_progress.md",
		"tech_debt/triaged.md",
		"sprint/planning.md",
		"sprint/active.md",
		"sprint/closing.md",
	}
	forbidden := []string{
		"shark status advance",
		"shark status set",
		"shark task next-status",
		"shark task set-status",
		"shark feature next-status",
		"shark epic next-status",
		"shark claim",
		"shark heartbeat",
		"shark release",
	}

	for _, tmplName := range dispatchPrompts {
		t.Run(tmplName, func(t *testing.T) {
			rendered, err := renderer.Render(tmplName, vars)
			require.NoError(t, err, "render %s", tmplName)
			for _, snippet := range forbidden {
				assert.NotContainsf(t, rendered, snippet,
					"dispatch prompt %s must not instruct the worker to mutate Shark workflow state directly", tmplName)
			}
		})
	}
}

func TestSharkRunVerbUsesNextDispatchContract(t *testing.T) {
	path := findRepoPath(t, filepath.Join("skills", "shark", "verbs", "run.md"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(data)

	assert.Contains(t, body, "shark next {KEY} --json")
	assert.Contains(t, body, "response.action")
	assert.Contains(t, body, "response.prompt")
	assert.Contains(t, body, "response.agent_type")
	assert.Contains(t, body, "response.provider")
	assert.Contains(t, body, "response.model")
	assert.Contains(t, body, "general-purpose")
	assert.Contains(t, body, "single worker by default")
	assert.Contains(t, body, "Only recurse when the workflow")
	assert.Contains(t, body, "explicitly invokes a multi-agent skill or recipe")

	for _, forbidden := range []string{
		"loop:  shark get {KEY} --json",
		"orchestrator_action.instruction",
		"orchestrator_action.agent_type",
		"read orchestrator_action",
	} {
		assert.NotContains(t, body, forbidden, "run harness must not assemble dispatch prompts from orchestrator_action")
	}
}

func findRepoPath(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate %s walking up from %s", rel, wd)
		}
		dir = parent
	}
}
