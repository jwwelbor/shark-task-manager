package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
	testutil "github.com/jwwelbor/shark-task-manager/internal/test"
)

func TestNext_RendersRepresentativeDispatchPromptsFromWorkflowIndexBundle(t *testing.T) {
	projectDir := t.TempDir()
	dbPath := filepath.Join(projectDir, "shark-tasks.db")

	fixture := testutil.WriteWorkflowIndexFixture(t)
	writeB036Config(t, projectDir, fixture.WorkflowIndexPath)

	t.Cleanup(func() {
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetDB()
		resetB036RootState(t)
		config.ClearWorkflowCache()
		templates.ResetOrchestratorEngine()
	})

	runCLI := func(args ...string) string {
		t.Helper()

		origWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(projectDir); err != nil {
			t.Fatalf("chdir %s: %v", projectDir, err)
		}
		defer func() {
			_ = os.Chdir(origWd)
		}()

		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetDB()
		config.ClearWorkflowCache()
		templates.ResetOrchestratorEngine()

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

		cli.RootCmd.SetArgs(append([]string{"--config", filepath.Join(projectDir, ".sharkconfig.json"), "--db", dbPath}, args...))
		execErr := cli.RootCmd.Execute()

		_ = wOut.Close()
		_ = wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		outBytes, _ := io.ReadAll(rOut)
		errBytes, _ := io.ReadAll(rErr)
		_ = rOut.Close()
		_ = rErr.Close()

		if execErr != nil {
			t.Fatalf("shark %s failed: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), execErr, string(outBytes), string(errBytes))
		}
		if len(errBytes) > 0 {
			t.Logf("shark %s stderr:\n%s", strings.Join(args, " "), string(errBytes))
		}
		return string(outBytes)
	}

	runCLI("admin", "init", "--non-interactive", "--force")
	writeB036Config(t, projectDir, fixture.WorkflowIndexPath)

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
		key     string
		want    string
		notWant string
	}{
		{key: "E01", want: "ROUTE FIXTURE EPIC ASSESSMENT", notWant: "\"instruction\":\"\""},
		{key: "E01-F01", want: "ROUTE FIXTURE FEATURE ASSESSMENT", notWant: "\"instruction\":\"\""},
		{key: "E01-F01-001", want: "ROUTE FIXTURE TASK DEVELOPMENT", notWant: "\"instruction\":\"\""},
		{key: "B001", want: "ROUTE FIXTURE BUG DEVELOPMENT", notWant: "\"instruction\":\"\""},
		{key: "B002", want: "ROUTE FIXTURE BUG CODE REVIEW", notWant: "\"instruction\":\"\""},
		{key: "B003", want: "ROUTE FIXTURE BUG QA", notWant: "\"instruction\":\"\""},
		{key: "CC-001", want: "ROUTE FIXTURE CHANGE DEVELOPMENT", notWant: "\"instruction\":\"\""},
		{key: "CC-002", want: "ROUTE FIXTURE CHANGE CODE REVIEW", notWant: "\"instruction\":\"\""},
		{key: "TD-001", want: "ROUTE FIXTURE TECH DEBT IN PROGRESS", notWant: "\"instruction\":\"\""},
		{key: "TD-002", want: "ROUTE FIXTURE TECH DEBT CODE REVIEW", notWant: "\"instruction\":\"\""},
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
		if strings.Contains(out, tc.notWant) {
			t.Fatalf("next output for %s still contains empty instruction payload\nbody:\n%s", tc.key, out)
		}
	}
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

func writeB036Config(t *testing.T, projectDir, workflowIndex string) {
	t.Helper()

	body := `{
  "color_enabled": false,
  "interactive_mode": false,
  "require_rejection_reason": false,
  "workflow_config": "` + workflowIndex + `"
}
`
	if err := os.WriteFile(filepath.Join(projectDir, ".sharkconfig.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write .sharkconfig.json: %v", err)
	}
}
