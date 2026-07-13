package commands

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	cli "github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config/action"
	"github.com/jwwelbor/shark-task-manager/internal/repository"
	"github.com/jwwelbor/shark-task-manager/internal/runner"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tc010ActionService struct {
	populated *action.PopulatedAction
	err       error
	calls     int
}

func (s *tc010ActionService) GetStatusAction(context.Context, string) (*action.OrchestratorAction, error) {
	return nil, s.err
}

func (s *tc010ActionService) GetStatusActionPopulated(context.Context, string, map[string]string) (*action.PopulatedAction, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.populated, nil
}

func (s *tc010ActionService) GetAllActions(context.Context) (map[string]*action.OrchestratorAction, error) {
	return nil, s.err
}

func (s *tc010ActionService) ValidateActions(context.Context) (*action.ValidationResult, error) {
	return nil, s.err
}

func (s *tc010ActionService) Reload(context.Context) error { return s.err }

func (s *tc010ActionService) ForEntity(string) action.ActionService { return s }

type tc010Transitioner struct {
	info *services.NextStatusInfo
}

func (t tc010Transitioner) TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error) {
	return nil, errors.New("transition not expected in TC-010 dispatch-path regression")
}

func (t tc010Transitioner) GetNextStatus(context.Context, string) (*services.NextStatusInfo, error) {
	return t.info, nil
}

type tc010PlaceholderGenerator struct{}

func (tc010PlaceholderGenerator) GeneratePlaceholders(context.Context, string) (map[string]string, error) {
	return map[string]string{"key": "T-E38-F01-001"}, nil
}

func TestTC010_CobraNextAndRunUseSharedDispatchResolver(t *testing.T) {
	// TC-010 caller-path contract: invoke the actual Cobra root with production
	// argv. The only replaced seams are the lower service adapters and action
	// service; resolveNext, runRun, and the shared resolver remain real code.
	projectDir := t.TempDir()
	configPath := filepath.Join(projectDir, ".sharkconfig.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}\n"), 0o644))

	origActionService := getDispatchActionService
	origNextTransitioner := nextBuildTransitioner
	origNextPlaceholderGenerator := nextBuildPlaceholderGenerator
	origRunTransitioner := buildRunTransitioner
	origRunPlaceholderGenerator := buildRunPlaceholderGenerator
	dbInitCalls := 0
	restoreDBInitializer := cli.SetDBInitializerForTest(func(context.Context) (*repository.DB, error) {
		dbInitCalls++
		return nil, errors.New("TC-010 must not initialize SQLite")
	})
	t.Cleanup(func() {
		getDispatchActionService = origActionService
		nextBuildTransitioner = origNextTransitioner
		nextBuildPlaceholderGenerator = origNextPlaceholderGenerator
		buildRunTransitioner = origRunTransitioner
		buildRunPlaceholderGenerator = origRunPlaceholderGenerator
		cli.RootCmd.SetArgs(nil)
		resetTC010RootState(t)
		cli.ResetServices()
		cli.ResetWorkflowService()
		cli.ResetDB()
	})
	t.Cleanup(restoreDBInitializer)
	resetTC010RootState(t)

	transitioner := tc010Transitioner{info: &services.NextStatusInfo{
		EntityKey:     "T-E38-F01-001",
		CurrentStatus: "awaiting_approval",
	}}
	placeholderGenerator := tc010PlaceholderGenerator{}
	nextBuildTransitioner = func(context.Context, string) (runner.EntityTransitioner, error) {
		return transitioner, nil
	}
	nextBuildPlaceholderGenerator = func(context.Context, string) runner.PlaceholderGenerator {
		return placeholderGenerator
	}
	buildRunTransitioner = func(context.Context, string) (runner.EntityTransitioner, error) {
		return transitioner, nil
	}
	buildRunPlaceholderGenerator = func(context.Context, string) runner.PlaceholderGenerator {
		return placeholderGenerator
	}

	t.Run("next invokes production resolver path", func(t *testing.T) {
		actionService := &tc010ActionService{populated: &action.PopulatedAction{
			Action: action.ActionPause,
		}}
		getDispatchActionService = func(context.Context) (action.ActionService, error) {
			return actionService, nil
		}
		stdout, stderr, execErr := executeTC010Root(t, configPath, "next", "T-E38-F01-001", "--json")
		require.NoError(t, execErr, "stderr: %s", stderr)
		assert.Equal(t, 1, actionService.calls, "next must reach the shared dispatch resolver through the action seam")
		var response struct {
			Action string `json:"action"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal([]byte(stdout), &response))
		assert.Equal(t, "pause", response.Action)
		assert.Equal(t, "awaiting_approval", response.Status)
	})

	t.Run("run propagates shared resolver failure", func(t *testing.T) {
		providerErr := errors.New("provider configuration unavailable")
		actionService := &tc010ActionService{err: providerErr}
		getDispatchActionService = func(context.Context) (action.ActionService, error) {
			return actionService, nil
		}
		_, stderr, execErr := executeTC010Root(t, configPath, "run", "T-E38-F01-001", "--dry-run", "--json")
		require.Error(t, execErr)
		assert.Equal(t, 1, actionService.calls, "run must reach the shared dispatch resolver through the action seam")
		assert.Contains(t, execErr.Error(), "failed to resolve dispatch step")
		assert.Contains(t, execErr.Error(), "populate action")
		assert.NotContains(t, stderr, "panic")
	})
	assert.Zero(t, dbInitCalls, "TC-010 Cobra paths must not fall back to SQLite initialization")
}

func executeTC010Root(t *testing.T, configPath string, args ...string) (stdout, stderr string, execErr error) {
	t.Helper()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = wOut, wErr
	defer func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	}()

	cli.RootCmd.SetArgs(append([]string{"--config", configPath}, args...))
	execErr = cli.RootCmd.Execute()
	require.NoError(t, wOut.Close())
	require.NoError(t, wErr.Close())
	outBytes, readErr := io.ReadAll(rOut)
	require.NoError(t, readErr)
	errBytes, readErr := io.ReadAll(rErr)
	require.NoError(t, readErr)
	_ = rOut.Close()
	_ = rErr.Close()
	return string(outBytes), string(errBytes), execErr
}

func resetTC010RootState(t *testing.T) {
	t.Helper()
	flags := cli.RootCmd.PersistentFlags()
	for name, value := range map[string]string{
		"config": "", "field": "", "json": "false",
		"no-color": "false", "verbose": "false",
	} {
		require.NoError(t, flags.Set(name, value))
	}
	require.NoError(t, flags.Set("db", flags.Lookup("db").DefValue))
	cli.GlobalConfig.ConfigFile = ""
	cli.GlobalConfig.DBPath = flags.Lookup("db").DefValue
	cli.GlobalConfig.Field = ""
	cli.GlobalConfig.JSON = false
	cli.GlobalConfig.NoColor = false
	cli.GlobalConfig.Verbose = false
}
