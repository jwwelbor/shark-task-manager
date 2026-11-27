package commands

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTagServiceFactory(t *testing.T, factory func() tagServiceIface) {
	t.Helper()
	previous := tagServiceFactory
	tagServiceFactory = factory
	t.Cleanup(func() { tagServiceFactory = previous })
}

func TestResolveTagService(t *testing.T) {
	fallback, explicit := &mockTagService{}, &mockTagService{}
	calls := 0
	withTagServiceFactory(t, func() tagServiceIface {
		calls++
		return fallback
	})
	assert.Same(t, explicit, resolveTagService(explicit))
	assert.Zero(t, calls)
	for want := 1; want <= 2; want++ {
		assert.Same(t, fallback, resolveTagService(nil))
		assert.Equal(t, want, calls)
	}
}

func TestChangeCreateTagErrors(t *testing.T) {
	for _, path := range [][]string{{"change", "create"}, {"create", "change"}, {"create", "change-card"}} {
		cmd, rest, err := cli.RootCmd.Find(path)
		require.NoError(t, err)
		require.Empty(t, rest)
		for _, tc := range []struct {
			code string
			err  error
		}{
			{"unregistered_tag", &services.UnregisteredTagError{Name: "missing"}},
			{"tag_required", &services.TagRequiredError{EntityType: "change"}},
		} {
			for _, mode := range []string{"text", "json"} {
				t.Run(cmd.CommandPath()+"/"+tc.code+"/"+mode, func(t *testing.T) {
					checkChangeCreateTagError(t, cmd, tc.err, tc.code, mode == "json")
				})
			}
		}
	}
}

func checkChangeCreateTagError(t *testing.T, cmd *cobra.Command, injected error, code string, jsonMode bool) {
	t.Helper()
	withEmptyStdin(t)
	withBackgroundCommandContext(t, cmd)
	withJSONMode(t, jsonMode)
	if flag := cmd.Flags().Lookup("json"); flag != nil {
		previous := flag.Changed
		flag.Changed = jsonMode
		t.Cleanup(func() { flag.Changed = previous })
	}
	creates, lists := 0, 0
	tags := &mockTagService{listTagsFn: func(context.Context) ([]*models.Tag, error) {
		lists++
		return []*models.Tag{{Name: "voice"}, {Name: "auth"}}, nil
	}}
	withTagServiceFactory(t, func() tagServiceIface { return tags })
	withChangeCardSvcOverrideForSize(t, &MockChangeCardService{
		CreateChangeCardFunc: func(context.Context, services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
			creates++
			return nil, false, injected
		},
	})

	var err error
	stdout, output := captureStdoutAndStderr(t, func() { err = cmd.RunE(cmd, []string{"Tagged change"}) })
	assert.Empty(t, stdout)
	require.Equal(t, 1, creates)
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "exit code 3:"), "%v", err)
	assert.ErrorIs(t, err, injected)
	if code == "unregistered_tag" {
		var target *services.UnregisteredTagError
		assert.ErrorAs(t, err, &target)
	} else {
		var target *services.TagRequiredError
		assert.ErrorAs(t, err, &target)
	}
	checkTagDiagnostic(t, output, injected, code, jsonMode, lists)
}

func checkTagDiagnostic(t *testing.T, output string, injected error, code string, jsonMode bool, lists int) {
	t.Helper()
	if jsonMode {
		assert.Zero(t, lists)
		var obj map[string]string
		require.NoError(t, json.Unmarshal([]byte(output), &obj))
		assert.Equal(t, map[string]string{"error": code, "message": injected.Error()}, obj)
		return
	}
	assert.Equal(t, 1, lists)
	assert.Contains(t, output, injected.Error())
	assert.Contains(t, output, "Available tags:")
	assert.Contains(t, output, "voice")
	assert.Contains(t, output, "auth")
	if code == "unregistered_tag" {
		lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
		assert.Equal(t, "To add it: shark tags add missing", lines[len(lines)-1])
	} else {
		assert.NotContains(t, output, "To add it:")
	}
}
