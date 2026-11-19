package commands

import (
	"context"
	"os"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnifiedChangeCreate_ExplicitSizeSatisfiesRequiredSize checks the
// unified `shark create change` handlers with the production
// size_required_for: ["change"] contract. Both registered command forms must
// forward an explicit size to the service.
func TestUnifiedChangeCreate_ExplicitSizeSatisfiesRequiredSize(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		size string
		want int
	}{
		{name: "change XS size", cmd: createChangeCmd, size: "XS", want: 1},
		{name: "change numeric size from B082 reproduction", cmd: createChangeCmd, size: "3", want: 3},
		{name: "change-card label size", cmd: createChangeCardCmd, size: "XXL", want: 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUnifiedChangeCreateExplicitSize(t, tt.cmd, tt.size, tt.want)
		})
	}
}

func assertUnifiedChangeCreateExplicitSize(t *testing.T, cmd *cobra.Command, size string, want int) {
	t.Helper()
	withEmptyStdin(t)
	withBackgroundCommandContext(t, cmd)
	var capturedInput services.CreateChangeCardInput
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
			if input.Size == nil {
				return nil, false, &services.SizeRequiredError{EntityType: string(models.EntityTypeChange)}
			}
			capturedInput = input
			return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, false, nil
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)
	cmd.SetContext(context.Background())
	flag := cmd.Flags().Lookup("size")
	originalSize := flag.Value.String()
	originalSizeChanged := flag.Changed
	t.Cleanup(func() {
		assert.NoError(t, cmd.Flags().Set("size", originalSize), "restore --size")
		flag.Changed = originalSizeChanged
	})
	require.NoError(t, cmd.Flags().Set("size", size), "set --size")
	require.NoError(t, runChangeCardCreate(cmd, []string{"Explicit-size change"}))
	require.NotNil(t, capturedInput.Size, "explicit --size should reach the change-card service")
	assert.Equal(t, want, *capturedInput.Size, "service input Size")
}

// TestUnifiedChangeCreate_MissingSizeStillRejectedWhenRequired verifies that
// forwarding an optional flag does not bypass change-card size enforcement.
func TestUnifiedChangeCreate_MissingSizeStillRejectedWhenRequired(t *testing.T) {
	withEmptyStdin(t)
	withBackgroundCommandContext(t, createChangeCmd)

	called := false
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
			called = true
			assert.Nil(t, input.Size, "service input Size when --size is omitted")
			return nil, false, &services.SizeRequiredError{EntityType: string(models.EntityTypeChange)}
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := createChangeCmd
	cmd.SetContext(context.Background())
	flag := cmd.Flags().Lookup("size")
	originalSize := flag.Value.String()
	originalSizeChanged := flag.Changed
	t.Cleanup(func() {
		assert.NoError(t, cmd.Flags().Set("size", originalSize), "restore --size")
		flag.Changed = originalSizeChanged
	})
	require.NoError(t, cmd.Flags().Set("size", ""), "clear --size")

	err := runChangeCardCreate(cmd, []string{"Missing-size change"})
	require.Error(t, err, "service should reject missing size when change is required")
	var sizeErr *services.SizeRequiredError
	require.ErrorAs(t, err, &sizeErr)
	assert.True(t, called, "required-size service should be called")
}

func TestUnifiedChangeCreate_InvalidSizeRejectedBeforeService(t *testing.T) {
	for _, bad := range []string{"clear", "4", "XXXL"} {
		t.Run(bad, func(t *testing.T) {
			withEmptyStdin(t)
			withBackgroundCommandContext(t, createChangeCmd)

			called := false
			stub := &MockChangeCardService{
				CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
					called = true
					return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, false, nil
				},
			}
			withChangeCardSvcOverrideForSize(t, stub)

			cmd := createChangeCmd
			cmd.SetContext(context.Background())
			flag := cmd.Flags().Lookup("size")
			originalSize := flag.Value.String()
			originalSizeChanged := flag.Changed
			t.Cleanup(func() {
				assert.NoError(t, cmd.Flags().Set("size", originalSize), "restore --size")
				flag.Changed = originalSizeChanged
			})
			require.NoError(t, cmd.Flags().Set("size", bad), "set invalid --size")

			err := runChangeCardCreate(cmd, []string{"Invalid-size change"})
			require.ErrorIs(t, err, models.ErrInvalidSize)
			assert.False(t, called, "service should not be called for invalid --size")
		})
	}
}

func TestUnifiedChangeCreate_SizeFlagReadErrorIsReturned(t *testing.T) {
	withEmptyStdin(t)

	called := false
	withChangeCardSvcOverrideForSize(t, &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
			called = true
			return &models.ChangeCard{BaseEntity: models.BaseEntity{Key: "CC-001", Title: input.Title}}, false, nil
		},
	})

	cmd := &cobra.Command{}
	withBackgroundCommandContext(t, cmd)
	cmd.Flags().StringSlice("tag", nil, "")
	err := runChangeCardCreate(cmd, []string{"Invalid-size-flag change"})
	require.Error(t, err, "expected --size flag lookup error")
	assert.False(t, called, "service should not be called when --size cannot be read")
}

func TestUnifiedChangeCreate_TagsReachService(t *testing.T) {
	for _, cmd := range []*cobra.Command{createChangeCmd, createChangeCardCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			withEmptyStdin(t)
			withBackgroundCommandContext(t, cmd)

			var capturedInput services.CreateChangeCardInput
			withChangeCardSvcOverrideForSize(t, &MockChangeCardService{
				CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
					capturedInput = input
					return &models.ChangeCard{BaseEntity: models.BaseEntity{ID: 1, Key: "CC-001", Title: input.Title}}, false, nil
				},
			})

			cmd.SetContext(context.Background())
			sizeFlag := cmd.Flags().Lookup("size")
			tagFlag := cmd.Flags().Lookup("tag")
			originalSize := sizeFlag.Value.String()
			originalSizeChanged := sizeFlag.Changed
			tagValue, ok := tagFlag.Value.(pflag.SliceValue)
			require.Truef(t, ok, "--tag value has type %T, want pflag.SliceValue", tagFlag.Value)
			originalTags := append([]string(nil), tagValue.GetSlice()...)
			originalTagChanged := tagFlag.Changed
			t.Cleanup(func() {
				assert.NoError(t, cmd.Flags().Set("size", originalSize), "restore --size")
				sizeFlag.Changed = originalSizeChanged
				assert.NoError(t, tagValue.Replace(originalTags), "restore --tag")
				tagFlag.Changed = originalTagChanged
			})
			require.NoError(t, cmd.Flags().Set("size", ""), "clear --size")
			require.NoError(t, cmd.Flags().Set("tag", "ops,review"), "set --tag")

			require.NoError(t, runChangeCardCreate(cmd, []string{"Tagged change"}))
			assert.Equal(t, []string{"ops", "review"}, capturedInput.Tags, "service input Tags")
		})
	}
}

// withEmptyStdin gives direct create-handler tests a pipe that immediately
// reaches EOF when ResolveContentInput checks for piped content.
func withEmptyStdin(t *testing.T) {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err, "create stdin pipe")
	if err := writer.Close(); err != nil {
		assert.NoError(t, reader.Close(), "close stdin pipe reader after writer close failure")
		require.NoError(t, err, "close stdin pipe writer")
	}

	originalStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = originalStdin
		assert.NoError(t, reader.Close(), "close stdin pipe reader")
	})
}

func withBackgroundCommandContext(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	originalContext := cmd.Context()
	cmd.SetContext(context.Background())
	t.Cleanup(func() { cmd.SetContext(originalContext) })
}
