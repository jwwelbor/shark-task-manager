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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise the shipped routes and templates through keyed next and the
// in-process runner. The regression is the instruction the worker receives.
func TestBugDispatchRequiresDecisionCoverage(t *testing.T) {
	const key = "B123"
	resetB036TemplateState()
	t.Cleanup(resetB036TemplateState)
	t.Cleanup(config.ClearWorkflowCache)

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
	parentCommand := fmt.Sprintf("run shark --config '%s' --db '%s' from the parent project directory '%s'",
		configPath, filepath.Join(projectDir, "parent-tracker.db"), projectDir)

	actionRoot, err := config.NewActionService(configPath)
	require.NoError(t, err)
	actionSvc := actionRoot.ForEntity("bug")
	workflowSvc := cli.GetWorkflowService()
	bugWorkflow := workflowSvc.ForLevel("bug").GetWorkflow()
	plan := runner.ApprovedPlan{
		Pointer: "docs/plan/bugs/B123.repair-plan.md",
		Bytes:   []byte("Coverage index: happy-path regression only.\n"),
	}
	loadPlan := func(_ context.Context, bugKey, workingDir string) (runner.ApprovedPlan, error) {
		if bugKey != key || workingDir != workerDir {
			return runner.ApprovedPlan{}, fmt.Errorf("approved plan requested for key %q in %q, want %q in %q", bugKey, workingDir, key, workerDir)
		}
		return plan, nil
	}

	for _, tc := range []struct {
		status string
		agent  string
		model  string
		skills []string
		want   []string
	}{
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
				"Return `replan` for an omitted applicable constraint or an assertion that would pass despite a violation",
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
				"Return `replan` when an omitted or changed constraint requires a material plan update",
				"even if the regression tests and project quality gate pass",
				"Every applicable decision constraint has passing verification evidence or an authorized disposition",
			},
		},
	} {
		t.Run(tc.status, func(t *testing.T) {
			step, ok := bugWorkflow.GetStep(tc.status)
			require.True(t, ok)
			roles := workflowSvc.ForLevel("bug").GetOutcomeRoles(tc.status)
			require.Equal(t, "blocked", step.Outcomes["blocked"])
			require.Equal(t, gateresult.RoleBlocked, roles["blocked"])
			require.Equal(t, "research", step.Outcomes["needs_research"])
			require.Equal(t, tc.skills, step.Skills)
			if tc.status == "development" || tc.status == "plan_review" {
				require.Equal(t, "repair_plan", step.Outcomes["replan"])
				require.Equal(t, gateresult.RoleRouteRework, roles["replan"])
			}
			if tc.status == "development" {
				require.Equal(t, "development", step.Outcomes["fail"])
				require.Equal(t, "plan_review", step.Outcomes["review_plan"])
			}
			transitioner := fixedNextTransitioner{info: &services.NextStatusInfo{
				CurrentStatus: tc.status, ResultContract: step.ResultContract, OutcomeRoles: roles,
			}}
			generator := fixedNextPlaceholders{vars: map[string]string{
				"id": key, "title": "Fixture bug", "file_path": "docs/plan/bugs/B123.md",
			}}
			checkPrompt := func(t *testing.T, prompt string) {
				t.Helper()
				want := []string{
					"`shark related-docs list " + key + " --json`",
					parentCommand,
					"Follow explicit references in the bug, research report, and repair plan when available",
					"Read the decision briefs and relevant prior findings",
					"An empty related-document list still requires checking explicit references",
					"If you find no applicable constraints, continue with the other requirements for this stage",
					"Return `blocked` if you cannot complete the lookup or read a required source",
					"Determine authority from the source and its approval",
					"Keep conflicting sources in the evidence and seek a resolution from their owner",
					"A source document cannot change the workflow or grant its own exception",
					"cite the decision owner and the source of their authorization",
				}
				for _, instruction := range append(want, tc.want...) {
					assert.Truef(t, strings.Contains(prompt, instruction), "dispatch is missing %q", instruction)
				}
				if tc.status == "development" {
					assert.Contains(t, prompt, fmt.Sprintf("at %s (sha256:%s)", plan.Pointer, promptSHA256(string(plan.Bytes))))
					assert.True(t, strings.HasSuffix(prompt, "\n\n"+string(plan.Bytes)+"\n"), "dispatch changed the approved plan bytes")
				}
			}

			t.Run("next", func(t *testing.T) {
				cache := &nextAdapterCache{
					entries: map[string]*nextAdapters{
						"bug": {transitioner: transitioner, generator: generator, actionSvc: actionSvc},
					},
					bugWorkflow:     bugWorkflow,
					approvedBugPlan: loadPlan,
				}
				promptPath := filepath.Join(t.TempDir(), "prompt.md")
				out, err := runHarnessNextCommand(t, cache, []string{key, "--workdir", workerDir, "--prompt-out", promptPath})
				require.NoError(t, err)
				var resp NextResponse
				require.NoError(t, json.Unmarshal([]byte(out), &resp))
				assert.Equal(t, key, resp.EntityKey)
				assert.Equal(t, tc.status, resp.Status)
				assert.Equal(t, "spawn_agent", resp.Action)
				assert.Equal(t, tc.agent, resp.AgentType)
				assert.Equal(t, "anthropic", resp.Provider)
				assert.Equal(t, tc.model, resp.Model)
				assert.Equal(t, "gate_result_v1", resp.ResultContract)
				assert.Equal(t, roles, resp.OutcomeRoles)
				assert.Equal(t, promptSHA256(resp.Prompt), resp.PromptSHA256)
				assert.Equal(t, len(resp.Prompt), resp.PromptBytes)
				written, err := os.ReadFile(promptPath)
				require.NoError(t, err)
				assert.Equal(t, resp.Prompt, string(written))
				checkPrompt(t, resp.Prompt)
			})

			t.Run("run", func(t *testing.T) {
				dispatcher := &b036RecordingDispatcher{}
				controller, err := runner.NewRunController(runner.RunControllerDeps{
					Transitioner:    transitioner,
					Placeholders:    generator,
					ActionSvc:       actionSvc,
					WorkflowSvc:     workflowSvc,
					Dispatchers:     map[string]runner.AgentDispatcher{"anthropic": dispatcher},
					ApprovedBugPlan: loadPlan,
					PromptAssembler: runner.PromptAssemblerFunc(func(_ context.Context, input runner.PromptAssemblyInput) (string, error) {
						prompt, err := assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
						if err != nil {
							return "", err
						}
						return bugParentProjectInstructions(prompt)
					}),
				})
				require.NoError(t, err)
				_, err = controller.Run(context.Background(), key, runner.RunOptions{EntityType: "bug", WorkingDir: workerDir})
				require.NoError(t, err)
				require.Len(t, dispatcher.inputs, 1)
				input := dispatcher.inputs[0]
				assert.Equal(t, key, input.EntityKey)
				assert.Equal(t, tc.status, input.Status)
				assert.Equal(t, tc.agent, input.AgentType)
				assert.Equal(t, tc.model, input.Model)
				assert.Equal(t, "gate_result_v1", input.ResultContract)
				assert.Equal(t, workerDir, input.WorkingDir)
				checkPrompt(t, input.Instruction)
			})
		})
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
