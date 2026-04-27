package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/auth/maintainer"
	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// tagServiceIface mirrors the public surface of *services.TagService. CLI
// tests inject a mock via this unexported interface; production uses the
// real service.
type tagServiceIface interface {
	ListTags(ctx context.Context) ([]*models.Tag, error)
	AddTag(ctx context.Context, name, providedPass string) (*models.Tag, error)
	RemoveTag(ctx context.Context, name string, force bool, providedPass string) error
	RenameTag(ctx context.Context, oldName, newName, providedPass string) (*models.Tag, error)
}

// ---------------------------------------------------------------------------
// Root command
// ---------------------------------------------------------------------------

// tagsCmd is the "shark tags" parent command.
var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage the tag vocabulary",
	Long: `Commands for managing the managed tag vocabulary.

Tags are registered names (lowercase, alphanumeric, hyphens) that can be
applied to epics, features, and tasks. The vocabulary is maintained by
maintainer-authorized commands.

Mutating commands (add, rm, rename) require the maintainer password either
via the --pass flag or a live authorization cache (run
'shark admin maintainer set-password' to configure the password).`,
}

func init() {
	tagsCmd.AddCommand(newTagsListCmd(nil))
	tagsCmd.AddCommand(newTagsAddCmd(nil))
	tagsCmd.AddCommand(newTagsRmCmd(nil))
	tagsCmd.AddCommand(newTagsRenameCmd(nil))

	cli.RootCmd.AddCommand(tagsCmd)
}

// ---------------------------------------------------------------------------
// List command
// ---------------------------------------------------------------------------

// newTagsListCmd constructs the "shark tags list" command. Passing nil uses
// the real cli.GetTagService(); tests pass a mock.
func newTagsListCmd(svc tagServiceIface) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered tags",
		Long:  `List all tags in the vocabulary, ordered alphabetically.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := resolveTagService(svc)

			tags, err := s.ListTags(cmd.Context())
			if err != nil {
				// Generic error — no typed translation needed for list
				writeTagsError(cmd, "db_error", err.Error())
				return fmt.Errorf("exit code 2: %w", err)
			}

			jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
			if jsonMode {
				type tagJSON struct {
					Name string `json:"name"`
				}
				items := make([]tagJSON, len(tags))
				for i, t := range tags {
					items[i] = tagJSON{Name: t.Name}
				}
				data, _ := json.Marshal(items)
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
				return nil
			}

			// Plain text
			if len(tags) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no tags registered)")
				return nil
			}
			for _, t := range tags {
				fmt.Fprintln(cmd.OutOrStdout(), t.Name)
			}
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Add command
// ---------------------------------------------------------------------------

// newTagsAddCmd constructs the "shark tags add <name>" command.
func newTagsAddCmd(svc tagServiceIface) *cobra.Command {
	var flagPass string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a tag to the vocabulary",
		Long: `Add a new tag name to the managed vocabulary.

The name is normalized (lowercased, trimmed) before creation. Tag names must
match ^[a-z0-9][a-z0-9-]{0,63}$ — lowercase letters, digits, and hyphens only.

Requires maintainer authorization via --pass or a live cache entry.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			s := resolveTagService(svc)

			tag, err := s.AddTag(cmd.Context(), name, flagPass)
			if err != nil {
				code, exitCode := tagsErrorCode(err)
				writeTagsError(cmd, code, err.Error())
				// For UnauthorizedError, optionally print hint on a second line
				var unauth *maintainer.UnauthorizedError
				if errors.As(err, &unauth) {
					if hint := unauth.UserHint(); hint != "" {
						jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
						if !jsonMode {
							fmt.Fprintln(cmd.ErrOrStderr(), hint)
						}
					}
				}
				return fmt.Errorf("exit code %d: %w", exitCode, err)
			}

			jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
			if jsonMode {
				type addJSON struct {
					Name string `json:"name"`
				}
				data, _ := json.Marshal(addJSON{Name: tag.Name})
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added tag %s\n", tag.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPass, "pass", "", "Maintainer password (optional if cache is live)")
	return cmd
}

// ---------------------------------------------------------------------------
// Rm command
// ---------------------------------------------------------------------------

// newTagsRmCmd constructs the "shark tags rm <name>" command.
func newTagsRmCmd(svc tagServiceIface) *cobra.Command {
	var (
		flagPass  string
		flagForce bool
	)

	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Remove a tag from the vocabulary",
		Long: `Remove a tag from the managed vocabulary.

If the tag is still attached to entities, the command fails with an error
message showing the usage count. Use --force to remove the tag and all its
entity associations.

Requires maintainer authorization via --pass or a live cache entry.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			s := resolveTagService(svc)

			err := s.RemoveTag(cmd.Context(), name, flagForce, flagPass)
			if err != nil {
				return handleVocabularyErrorWithSnippet(cmd, s, name, err)
			}

			jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
			if jsonMode {
				type rmJSON struct {
					Name    string `json:"name"`
					Removed bool   `json:"removed"`
				}
				data, _ := json.Marshal(rmJSON{Name: name, Removed: true})
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed tag %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPass, "pass", "", "Maintainer password (optional if cache is live)")
	cmd.Flags().BoolVar(&flagForce, "force", false, "Force removal even if tag is in use by entities")
	return cmd
}

// ---------------------------------------------------------------------------
// Rename command
// ---------------------------------------------------------------------------

// newTagsRenameCmd constructs the "shark tags rename <old> <new>" command.
func newTagsRenameCmd(svc tagServiceIface) *cobra.Command {
	var flagPass string

	cmd := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a tag in the vocabulary",
		Long: `Rename a tag in the managed vocabulary.

The rename is atomic — entity associations are preserved and point to the
renamed tag. If the new name already exists, the command fails with a
conflict error.

Requires maintainer authorization via --pass or a live cache entry.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			s := resolveTagService(svc)

			tag, err := s.RenameTag(cmd.Context(), oldName, newName, flagPass)
			if err != nil {
				return handleVocabularyErrorWithSnippet(cmd, s, oldName, err)
			}

			jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
			if jsonMode {
				type renameJSON struct {
					Old string `json:"old"`
					New string `json:"new"`
				}
				data, _ := json.Marshal(renameJSON{Old: oldName, New: tag.Name})
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", data)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Renamed %s to %s\n", oldName, tag.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPass, "pass", "", "Maintainer password (optional if cache is live)")
	return cmd
}

// tagsErrorCode maps a service error to (jsonCode, exitCode).
func tagsErrorCode(err error) (string, int) {
	var notFound *services.NotFoundError
	if errors.As(err, &notFound) {
		return "not_found", 1
	}

	var unauth *maintainer.UnauthorizedError
	if errors.As(err, &unauth) {
		return "unauthorized", 3
	}

	var conflict *services.ConflictError
	if errors.As(err, &conflict) {
		return "conflict", 3
	}

	var inUse *services.TagInUseError
	if errors.As(err, &inUse) {
		return "in_use", 3
	}

	var validation *services.ValidationError
	if errors.As(err, &validation) {
		return "validation", 3
	}

	var unregistered *services.UnregisteredTagError
	if errors.As(err, &unregistered) {
		return "unregistered_tag", 3
	}

	var required *services.TagRequiredError
	if errors.As(err, &required) {
		return "tag_required", 3
	}

	var unavailable *services.TagFilterUnavailableError
	if errors.As(err, &unavailable) {
		return "unavailable", 3
	}

	return "db_error", 2
}

// writeTagsError writes an error to stderr (JSON format when --json is set).
func writeTagsError(cmd *cobra.Command, code, message string) {
	jsonMode := cmd.Flags().Changed("json") || cli.GlobalConfig.JSON
	if jsonMode {
		type errJSON struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		data, _ := json.Marshal(errJSON{Error: code, Message: message})
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", data)
	} else {
		fmt.Fprintln(cmd.ErrOrStderr(), message)
	}
}

// resolveTagService returns the injected svc (for tests) or the real service
// (for production).
func resolveTagService(svc tagServiceIface) tagServiceIface {
	if svc != nil {
		return svc
	}
	return cli.GetTagService()
}

// Ensure *services.TagService satisfies tagServiceIface at compile time.
var _ tagServiceIface = (*services.TagService)(nil)
