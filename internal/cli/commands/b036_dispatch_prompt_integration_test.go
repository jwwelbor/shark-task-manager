package commands

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestNext_RendersRepresentativeDispatchPromptsFromWorkflowIndexBundle(t *testing.T) {
	projectDir := t.TempDir()
	dbPath := filepath.Join(projectDir, "shark-tasks.db")

	fixture := testutil.WriteWorkflowIndexFixture(t)
	writeB036Config(t, projectDir, fixture)

	t.Cleanup(func() {
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetDB()
		resetB036RootState(t)
		config.ClearWorkflowCache()
		resetB036TemplateState()
	})

	runCLI := func(args ...string) string { return runB036CLI(t, projectDir, dbPath, args...) }

	runCLI("admin", "init", "--non-interactive", "--force")
	writeB036Config(t, projectDir, fixture)

	runCLI("epic", "create", "Epic prompt coverage")
	runCLI("feature", "create", "E01", "Feature prompt coverage")
	runCLI("task", "create", "E01", "F01", "Task prompt coverage")
	runCLI("bug", "create", "Bug prompt coverage")
	runCLI("bug", "create", "Bug review prompt coverage")
	runCLI("bug", "create", "Bug QA prompt coverage")
	runCLI("change", "create", "Change prompt coverage")
	runCLI("change", "create", "Change review prompt coverage")
	runCLI("td", "create", "Tech debt prompt coverage")
	runCLI("td", "create", "Tech debt review prompt coverage")

	runCLI("status", "set", "E01", "assessment", "--force", "--reason", "test setup")
	runCLI("status", "set", "E01-F01", "assessment", "--force", "--reason", "test setup")
	runCLI("status", "set", "E01-F01-001", "development", "--force", "--reason", "test setup")
	runCLI("status", "set", "B001", "development", "--force", "--reason", "test setup")
	runCLI("status", "set", "B002", "code_review", "--force", "--reason", "test setup")
	runCLI("status", "set", "B003", "qa", "--force", "--reason", "test setup")
	runCLI("status", "set", "CC-001", "development", "--force", "--reason", "test setup")
	runCLI("status", "set", "CC-002", "code_review", "--force", "--reason", "test setup")
	runCLI("status", "set", "TD-001", "in_progress", "--force", "--reason", "test setup")
	runCLI("status", "set", "TD-002", "code_review", "--force", "--reason", "test setup")

	cases := []struct {
		key       string
		want      string
		agentWant string
		notWant   string
	}{
		{key: "E01", want: "ROUTE FIXTURE EPIC ASSESSMENT", agentWant: "ROUTE FIXTURE AGENT RESEARCHER", notWant: "\"instruction\":\"\""},
		{key: "E01-F01", want: "ROUTE FIXTURE FEATURE ASSESSMENT", agentWant: "ROUTE FIXTURE AGENT PRODUCT MANAGER", notWant: "\"instruction\":\"\""},
		{key: "E01-F01-001", want: "ROUTE FIXTURE TASK DEVELOPMENT", agentWant: "ROUTE FIXTURE AGENT DEVELOPER", notWant: "\"instruction\":\"\""},
		{key: "B001", want: "ROUTE FIXTURE BUG DEVELOPMENT", agentWant: "ROUTE FIXTURE AGENT DEVELOPER", notWant: "\"instruction\":\"\""},
		{key: "B002", want: "ROUTE FIXTURE BUG CODE REVIEW", agentWant: "ROUTE FIXTURE AGENT REVIEWER", notWant: "\"instruction\":\"\""},
		{key: "B003", want: "ROUTE FIXTURE BUG QA", agentWant: "ROUTE FIXTURE AGENT QA", notWant: "\"instruction\":\"\""},
		{key: "CC-001", want: "ROUTE FIXTURE CHANGE DEVELOPMENT", agentWant: "ROUTE FIXTURE AGENT DEVELOPER", notWant: "\"instruction\":\"\""},
		{key: "CC-002", want: "ROUTE FIXTURE CHANGE CODE REVIEW", agentWant: "ROUTE FIXTURE AGENT REVIEWER", notWant: "\"instruction\":\"\""},
		{key: "TD-001", want: "ROUTE FIXTURE TECH DEBT IN PROGRESS", agentWant: "ROUTE FIXTURE AGENT DEVELOPER", notWant: "\"instruction\":\"\""},
		{key: "TD-002", want: "ROUTE FIXTURE TECH DEBT CODE REVIEW", agentWant: "ROUTE FIXTURE AGENT REVIEWER", notWant: "\"instruction\":\"\""},
	}

	for _, tc := range cases {
		out := runCLI("next", tc.key, "--json")
		if tc.key == "E01" {
			wantTemplateDir := fixture.ExpectedPromptsDir
			if got := templates.GetTemplateDirName(); got != wantTemplateDir {
				t.Fatalf("configured template dir = %q, want %q", got, wantTemplateDir)
			}
			rendered, err := templates.GetOrchestratorEngine().Render("epic/assessment.tmpl", map[string]string{
				"id":        "E01",
				"title":     "Epic prompt coverage",
				"file_path": "docs/plan/epic.md",
			})
			if err != nil {
				t.Fatalf("direct renderer check failed: %v", err)
			}
			if !strings.Contains(rendered, "ROUTE FIXTURE EPIC ASSESSMENT") {
				t.Fatalf("direct renderer check missing assessment instructions:\n%s", rendered)
			}
		}

		var resp struct {
			Prompt string `json:"prompt"`
		}
		if err := json.Unmarshal([]byte(out), &resp); err != nil {
			t.Fatalf("parse next output for %s: %v\nbody:\n%s", tc.key, err, out)
		}
		if !strings.Contains(resp.Prompt, tc.want) {
			t.Fatalf("prompt for %s missing %q\nprompt:\n%s", tc.key, tc.want, resp.Prompt)
		}
		if !strings.Contains(resp.Prompt, tc.agentWant) {
			t.Fatalf("prompt for %s missing configured agent body %q\nprompt:\n%s", tc.key, tc.agentWant, resp.Prompt)
		}
		if strings.Contains(out, tc.notWant) {
			t.Fatalf("next output for %s still contains empty instruction payload\nbody:\n%s", tc.key, out)
		}
	}
}

// TestRunController_RendersRepresentativeDispatchPromptsFromWorkflowIndexBundle
// exercises the same assembled prompt boundary that shark run uses. It uses
// the real temporary-project database, workflow index, action population, and
// template engine; the recording dispatcher is the only seam, preventing the
// test from launching an external agent while preserving the exact prompt that
// would be dispatched.
func TestRunController_RendersRepresentativeDispatchPromptsFromWorkflowIndexBundle(t *testing.T) {
	projectDir := t.TempDir()
	dbPath := filepath.Join(projectDir, "shark-tasks.db")
	fixture := testutil.WriteWorkflowIndexFixture(t)

	t.Cleanup(func() {
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetDB()
		resetB036RootState(t)
		config.ClearWorkflowCache()
		resetB036TemplateState()
	})
	writeB036Config(t, projectDir, fixture)
	setupB036Project(t, projectDir, dbPath, fixture)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir %s: %v", projectDir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("restore working directory %s: %v", origWd, err)
		}
	})
	// Cobra closes its DB after each setup command. Rebuild the production
	// service graph for the direct RunController entrypoint rather than
	// retaining a setup command's service that points at that closed DB.
	cli.ResetServices()
	cli.ResetWorkflowService()
	cli.ResetDB()
	config.ClearWorkflowCache()
	resetB036TemplateState()
	cli.GlobalConfig.ConfigFile = filepath.Join(projectDir, ".sharkconfig.json")
	cli.GlobalConfig.DBPath = dbPath
	// The next test above reaches Cobra's root initialization, which applies
	// this same resolved bundle configuration. The controller test starts at
	// the production run-controller seam so its recording dispatcher can avoid
	// launching an external agent; configure the renderer with that resolved
	// project setting before constructing the real action service.
	templates.SetConfiguredTemplateDir(fixture.ExpectedPromptsDir)
	templates.SetConfiguredSharkDataPath(fixture.BundleRoot)

	ctx := context.Background()
	actionSvcRoot, err := cli.GetActionService(ctx)
	if err != nil {
		t.Fatalf("GetActionService: %v", err)
	}
	workflowSvc := cli.GetWorkflowService()

	cases := []struct {
		key         string
		entityType  string
		instruction string
		agentBody   string
	}{
		{key: "E01", entityType: "epic", instruction: "ROUTE FIXTURE EPIC ASSESSMENT", agentBody: "ROUTE FIXTURE AGENT RESEARCHER"},
		{key: "E01-F01", entityType: "feature", instruction: "ROUTE FIXTURE FEATURE ASSESSMENT", agentBody: "ROUTE FIXTURE AGENT PRODUCT MANAGER"},
		{key: "T-E01-F01-001", entityType: "task", instruction: "ROUTE FIXTURE TASK DEVELOPMENT", agentBody: "ROUTE FIXTURE AGENT DEVELOPER"},
		{key: "B001", entityType: "bug", instruction: "ROUTE FIXTURE BUG DEVELOPMENT", agentBody: "ROUTE FIXTURE AGENT DEVELOPER"},
		{key: "CC-001", entityType: "change", instruction: "ROUTE FIXTURE CHANGE DEVELOPMENT", agentBody: "ROUTE FIXTURE AGENT DEVELOPER"},
		{key: "TD-001", entityType: "tech_debt", instruction: "ROUTE FIXTURE TECH DEBT IN PROGRESS", agentBody: "ROUTE FIXTURE AGENT DEVELOPER"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			dispatcher := &b036RecordingDispatcher{}
			transitioner, err := buildTransitioner(ctx, tc.entityType)
			if err != nil {
				t.Fatalf("buildTransitioner(%s): %v", tc.entityType, err)
			}
			controller, err := runner.NewRunController(runner.RunControllerDeps{
				Transitioner: transitioner,
				Placeholders: buildPlaceholderGenerator(ctx, tc.entityType),
				ActionSvc:    narrowActionServiceForEntity(actionSvcRoot, tc.entityType),
				WorkflowSvc:  workflowSvc,
				Dispatchers:  map[string]runner.AgentDispatcher{"": dispatcher},
				PromptAssembler: runner.PromptAssemblerFunc(func(ctx context.Context, input runner.PromptAssemblyInput) (string, error) {
					return assembleDispatchPrompt(input.Instruction, input.AgentType, input.Vars)
				}),
			})
			if err != nil {
				t.Fatalf("NewRunController: %v", err)
			}

			if _, err := controller.Run(ctx, tc.key, runner.RunOptions{EntityType: tc.entityType, WorkingDir: projectDir}); err != nil {
				t.Fatalf("RunController.Run(%s): %v", tc.key, err)
			}
			if len(dispatcher.inputs) != 1 {
				t.Fatalf("dispatch count for %s = %d, want 1", tc.key, len(dispatcher.inputs))
			}
			prompt := dispatcher.inputs[0].Instruction
			if !strings.Contains(prompt, tc.instruction) {
				t.Fatalf("run prompt for %s missing status instruction %q\nprompt:\n%s", tc.key, tc.instruction, prompt)
			}
			if !strings.Contains(prompt, tc.agentBody) {
				t.Fatalf("run prompt for %s missing configured agent body %q\nprompt:\n%s", tc.key, tc.agentBody, prompt)
			}
		})
	}
}

type b036RecordingDispatcher struct {
	inputs []runner.DispatchInput
}

func (d *b036RecordingDispatcher) Dispatch(_ context.Context, input runner.DispatchInput) (*runner.DispatchResult, error) {
	d.inputs = append(d.inputs, input)
	return nil, b036DispatchStop("stop after recording B036 dispatch prompt")
}

func (d *b036RecordingDispatcher) Name() string { return "b036-recording" }

func (d *b036RecordingDispatcher) BuildCommand(runner.DispatchInput) (string, error) {
	return "b036-recording-dispatch", nil
}

type b036DispatchStop string

func (e b036DispatchStop) Error() string { return string(e) }

var _ runner.AgentDispatcher = (*b036RecordingDispatcher)(nil)

func setupB036Project(t *testing.T, projectDir, dbPath string, fixture testutil.WorkflowIndexFixture) {
	t.Helper()

	// This setup deliberately reaches the Cobra command surface so the
	// temporary database and workflow-index bundle use the same configuration
	// initialization as `shark next` and `shark run`.
	runB036CLI(t, projectDir, dbPath, "admin", "init", "--non-interactive", "--force")
	writeB036Config(t, projectDir, fixture)
	runB036CLI(t, projectDir, dbPath, "epic", "create", "Epic prompt coverage")
	runB036CLI(t, projectDir, dbPath, "feature", "create", "E01", "Feature prompt coverage")
	runB036CLI(t, projectDir, dbPath, "task", "create", "E01", "F01", "Task prompt coverage")
	runB036CLI(t, projectDir, dbPath, "bug", "create", "Bug prompt coverage")
	runB036CLI(t, projectDir, dbPath, "change", "create", "Change prompt coverage")
	runB036CLI(t, projectDir, dbPath, "td", "create", "Tech debt prompt coverage")
	runB036CLI(t, projectDir, dbPath, "status", "set", "E01", "assessment", "--force", "--reason", "test setup")
	runB036CLI(t, projectDir, dbPath, "status", "set", "E01-F01", "assessment", "--force", "--reason", "test setup")
	runB036CLI(t, projectDir, dbPath, "status", "set", "E01-F01-001", "development", "--force", "--reason", "test setup")
	runB036CLI(t, projectDir, dbPath, "status", "set", "B001", "development", "--force", "--reason", "test setup")
	runB036CLI(t, projectDir, dbPath, "status", "set", "CC-001", "development", "--force", "--reason", "test setup")
	runB036CLI(t, projectDir, dbPath, "status", "set", "TD-001", "in_progress", "--force", "--reason", "test setup")
}

func runB036CLI(t *testing.T, projectDir, dbPath string, args ...string) string {
	t.Helper()

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("chdir %s: %v", projectDir, err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Errorf("restore working directory %s: %v", origWd, err)
		}
	}()

	cli.ResetServices()
	cli.ResetWorkflowService()
	cli.ResetDB()
	config.ClearWorkflowCache()
	resetB036TemplateState()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("stderr pipe: %v", err)
	}
	os.Stdout = wOut
	os.Stderr = wErr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	cli.RootCmd.SetArgs(append([]string{"--config", filepath.Join(projectDir, ".sharkconfig.json"), "--db", dbPath}, args...))
	execErr := cli.RootCmd.Execute()

	if err := wOut.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := wErr.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}
	outBytes, readOutErr := io.ReadAll(rOut)
	errBytes, readErrErr := io.ReadAll(rErr)
	if err := rOut.Close(); err != nil {
		t.Errorf("close stdout reader: %v", err)
	}
	if err := rErr.Close(); err != nil {
		t.Errorf("close stderr reader: %v", err)
	}
	if readOutErr != nil {
		t.Fatalf("read shark stdout: %v", readOutErr)
	}
	if readErrErr != nil {
		t.Fatalf("read shark stderr: %v", readErrErr)
	}
	if execErr != nil {
		t.Fatalf("shark %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), execErr, string(outBytes), string(errBytes))
	}
	if strings.Contains(string(errBytes), "agent body inline skipped") {
		t.Fatalf("shark %s unexpectedly skipped fixture agent body:\n%s", strings.Join(args, " "), string(errBytes))
	}
	return string(outBytes)
}

func resetB036TemplateState() {
	// The renderer retains these process-global settings across engine resets.
	// Restore canonical defaults before t.TempDir cleanup removes a fixture.
	templates.SetConfiguredTemplateDir("")
	templates.SetConfiguredSharkDataPath(config.DefaultSharkDataPath)
	templates.ResetOrchestratorEngine()
}

func resetB036RootState(t *testing.T) {
	t.Helper()

	cli.RootCmd.SetArgs(nil)
	flags := cli.RootCmd.PersistentFlags()
	for name, value := range map[string]string{
		"config":   "",
		"db":       "shark-tasks.db",
		"field":    "",
		"json":     "false",
		"no-color": "false",
		"verbose":  "false",
	} {
		if err := flags.Set(name, value); err != nil {
			t.Fatalf("reset root flag %s: %v", name, err)
		}
	}
	cli.GlobalConfig.ConfigFile = ""
	cli.GlobalConfig.DBPath = "shark-tasks.db"
	cli.GlobalConfig.Field = ""
	cli.GlobalConfig.JSON = false
	cli.GlobalConfig.NoColor = false
	cli.GlobalConfig.Verbose = false
}

func writeB036Config(t *testing.T, projectDir string, fixture testutil.WorkflowIndexFixture) {
	t.Helper()

	body := `{
  "color_enabled": false,
  "interactive_mode": false,
  "require_rejection_reason": false,
  "workflow_config": "` + fixture.WorkflowIndexPath + `",
  "shark_data_path": "` + fixture.BundleRoot + `"
}
`
	if err := os.WriteFile(filepath.Join(projectDir, ".sharkconfig.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write .sharkconfig.json: %v", err)
	}
}
