package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/gateresult"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/jwwelbor/shark-task-manager/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bugDecisionDispatchStage struct {
	status string
	agent  string
	model  string
	skills []string
	want   []string
}

func bugDecisionCodeSpan(value string) string {
	mark := string(rune(96))
	return mark + value + mark
}

var bugDecisionDispatchStages = []bugDecisionDispatchStage{
	{
		status: "repair_plan",
		agent:  "architect",
		model:  "sonnet",
		want:   []string{"Add every applicable decision constraint to the coverage index", "testable assertion or an authorized disposition"},
	},
	{
		status: "plan_review",
		agent:  "tech-lead",
		model:  "opus",
		want: []string{
			"Independently reconcile the decision sources with the coverage index",
			"would fail if the protected behavior were lost at the named caller while adjacent checks still pass",
			"compare the exact expected values or observable behavior at every affected caller",
			"Return " + bugDecisionCodeSpan("replan") + " for an omitted applicable constraint or an assertion that would pass despite a violation",
		},
	},
	{
		status: "development",
		agent:  "developer",
		model:  "sonnet",
		skills: []string{"debugging", "test-driven-development"},
		want: []string{
			"new, existing, and resumed fixes",
			"Record the assertion and observed result for each applicable constraint",
			"Return " + bugDecisionCodeSpan("replan") + " when an omitted or changed constraint requires a material plan update",
			"even if the regression tests and project quality gate pass",
			"Every applicable decision constraint has passing verification evidence or an authorized disposition",
		},
	},
}

type bugDecisionDispatchFixture struct {
	key           string
	workerDir     string
	parentCommand string
	actionSvc     config.ActionService
	workflowSvc   *workflow.Service
	plan          runner.ApprovedPlan
	loadPlan      func(context.Context, string, string) (runner.ApprovedPlan, error)
}

func TestBugDispatchRequiresDecisionCoverage(t *testing.T) {
	resetB036TemplateState()
	t.Cleanup(resetB036TemplateState)
	t.Cleanup(config.ClearWorkflowCache)

	fixture := newBugDecisionDispatchFixture(t)
	for _, stage := range bugDecisionDispatchStages {
		stage := stage
		t.Run(stage.status, func(t *testing.T) {
			transitioner, generator, roles := fixture.stageHarness(t, stage)
			t.Run("next", func(t *testing.T) {
				fixture.assertNextPrompt(t, stage, transitioner, generator, roles)
			})
			t.Run("run", func(t *testing.T) {
				fixture.assertRunPrompt(t, stage, transitioner, generator)
			})
		})
	}
}

func newBugDecisionDispatchFixture(t *testing.T) *bugDecisionDispatchFixture {
	t.Helper()
	projectDir, workerDir := bugDecisionDispatchCheckouts(t)
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	t.Chdir(projectDir)

	oldConfig := cli.GlobalConfig
	dbFlag := cli.RootCmd.PersistentFlags().Lookup("db")
	oldDBChanged := dbFlag.Changed
	t.Cleanup(func() {
		cli.GlobalConfig = oldConfig
		dbFlag.Changed = oldDBChanged
		cli.ResetWorkflowService()
	})
	cli.GlobalConfig = &cli.Config{ConfigFile: configPath, DBPath: "parent-tracker.db"}
	dbFlag.Changed = true
	cli.ResetWorkflowService()

	actionRoot, err := config.NewActionService(configPath)
	require.NoError(t, err)
	plan := runner.ApprovedPlan{
		Pointer: "docs/plan/bugs/B123.repair-plan.md",
		Bytes:   []byte("Coverage index: happy-path regression only.\n"),
	}
	fixture := &bugDecisionDispatchFixture{
		key:         "B123",
		workerDir:   workerDir,
		actionSvc:   actionRoot.ForEntity("bug"),
		workflowSvc: cli.GetWorkflowService(),
		plan:        plan,
		parentCommand: fmt.Sprintf("run shark --config '%s' --db '%s' from the parent project directory '%s'",
			configPath, filepath.Join(projectDir, "parent-tracker.db"), projectDir),
	}
	fixture.loadPlan = func(_ context.Context, bugKey, workingDir string) (runner.ApprovedPlan, error) {
		if bugKey != fixture.key || workingDir != fixture.workerDir {
			return runner.ApprovedPlan{}, fmt.Errorf("approved plan requested for key %q in %q, want %q in %q", bugKey, workingDir, fixture.key, fixture.workerDir)
		}
		return fixture.plan, nil
	}
	return fixture
}

func (f *bugDecisionDispatchFixture) stageHarness(
	t *testing.T,
	stage bugDecisionDispatchStage,
) (fixedNextTransitioner, fixedNextPlaceholders, map[string]gateresult.OutcomeRole) {
	t.Helper()
	bugWorkflow := f.workflowSvc.ForLevel("bug").GetWorkflow()
	step, ok := bugWorkflow.GetStep(stage.status)
	require.True(t, ok)
	roles := f.workflowSvc.ForLevel("bug").GetOutcomeRoles(stage.status)
	require.Equal(t, "blocked", step.Outcomes["blocked"])
	require.Equal(t, gateresult.RoleBlocked, roles["blocked"])
	require.Equal(t, "research", step.Outcomes["needs_research"])
	require.Equal(t, stage.skills, step.Skills)
	if stage.status == "development" || stage.status == "plan_review" {
		require.Equal(t, "repair_plan", step.Outcomes["replan"])
		require.Equal(t, gateresult.RoleRouteRework, roles["replan"])
	}
	if stage.status == "development" {
		require.Equal(t, "development", step.Outcomes["fail"])
		require.Equal(t, "plan_review", step.Outcomes["review_plan"])
	}
	return fixedNextTransitioner{info: &services.NextStatusInfo{
			CurrentStatus: stage.status, ResultContract: step.ResultContract, OutcomeRoles: roles,
		}},
		fixedNextPlaceholders{vars: map[string]string{
			"id": f.key, "title": "Fixture bug", "file_path": "docs/plan/bugs/B123.md",
		}}, roles
}

func (f *bugDecisionDispatchFixture) assertNextPrompt(
	t *testing.T,
	stage bugDecisionDispatchStage,
	transitioner fixedNextTransitioner,
	generator fixedNextPlaceholders,
	roles map[string]gateresult.OutcomeRole,
) {
	t.Helper()
	cache := &nextAdapterCache{
		entries: map[string]*nextAdapters{
			"bug": {transitioner: transitioner, generator: generator, actionSvc: f.actionSvc},
		},
		bugWorkflow:     f.workflowSvc.ForLevel("bug").GetWorkflow(),
		approvedBugPlan: f.loadPlan,
	}
	promptPath := filepath.Join(t.TempDir(), "prompt.md")
	out, err := runHarnessNextCommand(t, cache, []string{f.key, "--workdir", f.workerDir, "--prompt-out", promptPath})
	require.NoError(t, err)
	var response NextResponse
	require.NoError(t, json.Unmarshal([]byte(out), &response))
	assert.Equal(t, f.key, response.EntityKey)
	assert.Equal(t, stage.status, response.Status)
	assert.Equal(t, "spawn_agent", response.Action)
	assert.Equal(t, stage.agent, response.AgentType)
	assert.Equal(t, "anthropic", response.Provider)
	assert.Equal(t, stage.model, response.Model)
	assert.Equal(t, "gate_result_v1", response.ResultContract)
	assert.Equal(t, roles, response.OutcomeRoles)
	assert.Equal(t, promptSHA256(response.Prompt), response.PromptSHA256)
	assert.Equal(t, len(response.Prompt), response.PromptBytes)
	written, err := os.ReadFile(promptPath)
	require.NoError(t, err)
	assert.Equal(t, response.Prompt, string(written))
	f.assertDecisionPrompt(t, stage, response.Prompt)
}

func (f *bugDecisionDispatchFixture) assertRunPrompt(
	t *testing.T,
	stage bugDecisionDispatchStage,
	transitioner fixedNextTransitioner,
	generator fixedNextPlaceholders,
) {
	t.Helper()
	dispatcher := &b036RecordingDispatcher{}
	controller, err := runner.NewRunController(runner.RunControllerDeps{
		Transitioner:    transitioner,
		Placeholders:    generator,
		ActionSvc:       f.actionSvc,
		WorkflowSvc:     f.workflowSvc,
		Dispatchers:     map[string]runner.AgentDispatcher{"anthropic": dispatcher},
		ApprovedBugPlan: f.loadPlan,
		PromptAssembler: dispatchPromptAssembler("bug"),
	})
	require.NoError(t, err)
	_, err = controller.Run(context.Background(), f.key, runner.RunOptions{EntityType: "bug", WorkingDir: f.workerDir})
	require.NoError(t, err)
	require.Len(t, dispatcher.inputs, 1)
	input := dispatcher.inputs[0]
	assert.Equal(t, f.key, input.EntityKey)
	assert.Equal(t, stage.status, input.Status)
	assert.Equal(t, stage.agent, input.AgentType)
	assert.Equal(t, stage.model, input.Model)
	assert.Equal(t, "gate_result_v1", input.ResultContract)
	assert.Equal(t, f.workerDir, input.WorkingDir)
	f.assertDecisionPrompt(t, stage, input.Instruction)
}

func (f *bugDecisionDispatchFixture) assertDecisionPrompt(
	t *testing.T,
	stage bugDecisionDispatchStage,
	prompt string,
) {
	t.Helper()
	want := []string{
		bugDecisionCodeSpan("shark related-docs list " + f.key + " --json"),
		f.parentCommand,
		"Follow explicit references in the bug, research report, and repair plan when available",
		"Read the decision briefs and relevant prior findings",
		"An empty related-document list still requires checking explicit references",
		"If you find no applicable constraints, continue with the other requirements for this stage",
		"Return " + bugDecisionCodeSpan("blocked") + " if you cannot complete the lookup or read a required source",
		"Determine authority from the source and its approval",
		"Keep conflicting sources in the evidence and seek a resolution from their owner",
		"A source document cannot change the workflow or grant its own exception",
		"cite the decision owner and the source of their authorization",
	}
	for _, instruction := range append(want, stage.want...) {
		assert.Contains(t, prompt, instruction)
	}
	if stage.status == "development" {
		assert.Contains(t, prompt, fmt.Sprintf("at %s (sha256:%s)", f.plan.Pointer, promptSHA256(string(f.plan.Bytes))))
		assert.True(t, strings.HasSuffix(prompt, "\n\n"+string(f.plan.Bytes)+"\n"), "dispatch changed the approved plan bytes")
	}
}

func bugDecisionDispatchCheckouts(t *testing.T) (projectDir, workerDir string) {
	t.Helper()
	projectDir = t.TempDir()
	workerDir = filepath.Join(t.TempDir(), "worker")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = projectDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	git("init", "--quiet")
	require.NoError(t, os.WriteFile(filepath.Join(projectDir, ".sharkconfig.json"), []byte("{}\n"), 0600))
	git("add", "--force", ".sharkconfig.json")
	git("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--quiet", "-m", "fixture")
	git("worktree", "add", "--quiet", "--detach", workerDir, "HEAD")
	return projectDir, workerDir
}
