package commands

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	t.Cleanup(func() {
		if err := cmd.Flags().Set("size", originalSize); err != nil {
			t.Errorf("restore --size: %v", err)
		}
	})
	if err := cmd.Flags().Set("size", size); err != nil {
		t.Fatalf("set --size: %v", err)
	}
	if err := runChangeCardCreate(cmd, []string{"Explicit-size change"}); err != nil {
		t.Fatalf("runChangeCardCreate() error = %v", err)
	}
	if capturedInput.Size == nil {
		t.Fatal("expected explicit --size to reach the change-card service")
	}
	if *capturedInput.Size != want {
		t.Errorf("service input Size = %d, want %d", *capturedInput.Size, want)
	}
}

// TestUnifiedChangeCreate_MissingSizeStillRejectedWhenRequired verifies that
// forwarding an optional flag does not bypass change-card size enforcement.
func TestUnifiedChangeCreate_MissingSizeStillRejectedWhenRequired(t *testing.T) {
	withEmptyStdin(t)

	called := false
	stub := &MockChangeCardService{
		CreateChangeCardFunc: func(_ context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, bool, error) {
			called = true
			if input.Size != nil {
				t.Errorf("service input Size = %d, want nil when --size is omitted", *input.Size)
			}
			return nil, false, &services.SizeRequiredError{EntityType: string(models.EntityTypeChange)}
		},
	}
	withChangeCardSvcOverrideForSize(t, stub)

	cmd := createChangeCmd
	cmd.SetContext(context.Background())
	flag := cmd.Flags().Lookup("size")
	originalSize := flag.Value.String()
	t.Cleanup(func() {
		if err := cmd.Flags().Set("size", originalSize); err != nil {
			t.Errorf("restore --size: %v", err)
		}
	})
	if err := cmd.Flags().Set("size", ""); err != nil {
		t.Fatalf("clear --size: %v", err)
	}

	err := runChangeCardCreate(cmd, []string{"Missing-size change"})
	if err == nil {
		t.Fatal("expected service to reject missing size when change is required")
	}
	var sizeErr *services.SizeRequiredError
	if !errors.As(err, &sizeErr) {
		t.Fatalf("error = %v, want *services.SizeRequiredError", err)
	}
	if !called {
		t.Fatal("expected required-size service to be called")
	}
}

func TestUnifiedChangeCreate_InvalidSizeRejectedBeforeService(t *testing.T) {
	for _, bad := range []string{"clear", "4", "XXXL"} {
		t.Run(bad, func(t *testing.T) {
			withEmptyStdin(t)

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
			t.Cleanup(func() {
				if err := cmd.Flags().Set("size", originalSize); err != nil {
					t.Errorf("restore --size: %v", err)
				}
			})
			if err := cmd.Flags().Set("size", bad); err != nil {
				t.Fatalf("set invalid --size: %v", err)
			}

			err := runChangeCardCreate(cmd, []string{"Invalid-size change"})
			if !errors.Is(err, models.ErrInvalidSize) {
				t.Fatalf("error = %v, want models.ErrInvalidSize", err)
			}
			if called {
				t.Fatal("service should not be called for invalid --size")
			}
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
	cmd.SetContext(context.Background())
	cmd.Flags().StringSlice("tag", nil, "")
	err := runChangeCardCreate(cmd, []string{"Invalid-size-flag change"})
	if err == nil {
		t.Fatal("expected --size flag lookup error")
	}
	if called {
		t.Fatal("service should not be called when --size cannot be read")
	}
}

func TestUnifiedChangeCreate_TagsReachService(t *testing.T) {
	for _, cmd := range []*cobra.Command{createChangeCmd, createChangeCardCmd} {
		t.Run(cmd.Name(), func(t *testing.T) {
			withEmptyStdin(t)

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
			tagValue, ok := tagFlag.Value.(pflag.SliceValue)
			if !ok {
				t.Fatalf("--tag value has type %T, want pflag.SliceValue", tagFlag.Value)
			}
			originalTags := append([]string(nil), tagValue.GetSlice()...)
			originalTagChanged := tagFlag.Changed
			t.Cleanup(func() {
				if err := cmd.Flags().Set("size", originalSize); err != nil {
					t.Errorf("restore --size: %v", err)
				}
				if err := tagValue.Replace(originalTags); err != nil {
					t.Errorf("restore --tag: %v", err)
				}
				tagFlag.Changed = originalTagChanged
			})
			if err := cmd.Flags().Set("size", ""); err != nil {
				t.Fatalf("clear --size: %v", err)
			}
			if err := cmd.Flags().Set("tag", "ops,review"); err != nil {
				t.Fatalf("set --tag: %v", err)
			}

			if err := runChangeCardCreate(cmd, []string{"Tagged change"}); err != nil {
				t.Fatalf("runChangeCardCreate() error = %v", err)
			}
			if !reflect.DeepEqual(capturedInput.Tags, []string{"ops", "review"}) {
				t.Errorf("service input Tags = %q, want [ops review]", capturedInput.Tags)
			}
		})
	}
}

// withEmptyStdin gives direct create-handler tests a pipe that immediately
// reaches EOF when ResolveContentInput checks for piped content.
func withEmptyStdin(t *testing.T) {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if err := writer.Close(); err != nil {
		if readerErr := reader.Close(); readerErr != nil {
			t.Errorf("close stdin pipe reader after writer close failure: %v", readerErr)
		}
		t.Fatalf("close stdin pipe writer: %v", err)
	}

	originalStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = originalStdin
		if err := reader.Close(); err != nil {
			t.Errorf("close stdin pipe reader: %v", err)
		}
	})
}
