package commands

import (
	"context"
	"fmt"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
)

// entityTagServiceIface is the narrow tag-service surface the entity-tag
// commands actually need. Tests inject a mock that implements just these
// three methods instead of the full tagServiceIface. Production code sees
// *services.TagService satisfy this via resolveEntityTagService.
//
// The interface is a strict subset of TagAttacher (services/bug_service.go),
// plus ListTags (used by handleVocabularyErrorWithSnippet for the SC-2
// vocabulary snippet). Adding this extra method in the CLI layer keeps the
// service-side TagAttacher minimal.
//
// Spec: E28-F04 spec §2.7 ("factory is deliberately entity-agnostic"),
// test plan §3.1 row "entityTagServiceIface".
type entityTagServiceIface interface {
	AttachMany(ctx context.Context, entityType models.EntityType, entityID int64, names []string) error
	DetachOne(ctx context.Context, entityType models.EntityType, entityID int64, name string) error
	tagServiceIface // ListTags, AddTag, RemoveTag, RenameTag — only ListTags is used
}

// EntityKeyResolver turns a user-supplied entity key (e.g., "B001") into
// the numeric entity ID that TagService.AttachMany / DetachOne need. The
// returned error is propagated to the CLI exit code (typically 1 for "not
// found"). Production callers wire this to an entity-service GetByKey
// method; tests use a deterministic stub.
//
// Spec: E28-F04 spec §2.7 ("resolveKey" helper signature).
type EntityKeyResolver func(ctx context.Context, key string) (int64, error)

// makeEntityTagCmd builds the `tag` parent command with `add` and `rm`
// sub-subcommands for a single entity type. Each entity command file
// (bug.go and, in T-006..T-010, task.go / feature.go / epic.go /
// change.go / idea.go) calls this factory once in init().
//
// Invocation shape (from the user's perspective):
//
//	shark <entity> tag add <key> <name>
//	shark <entity> tag rm  <key> <name>
//
// The factory is deliberately entity-agnostic: all type-specific behaviour
// is carried in entityType (passed through to the tag service) and
// resolveKey (used to look up the numeric ID before the tag-service call).
// The production service accessor is threaded in via svcOverride-or-default
// style so the test injects a *mockEntityTagService.
//
// When svcOverride is non-nil (tests), the factory uses it; otherwise it
// resolves through cli.GetTagService() lazily per invocation. This
// matches the pattern used by newTagsRmCmd / newTagsAddCmd in tags.go.
//
// Spec references:
//   - REQ-F-013: `tag add` / `tag rm` subcommands.
//   - REQ-F-014: `tag rm` calls DetachOne.
//   - REQ-F-015: SC-2 error shape via handleVocabularyErrorWithSnippet.
//   - REQ-F-016: exit codes via tagsErrorCode.
//   - spec §2.7: entity-agnostic factory signature.
func makeEntityTagCmd(
	entityType models.EntityType,
	resolveKey EntityKeyResolver,
	svcOverride entityTagServiceIface,
) *cobra.Command {
	entityName := string(entityType)

	tagCmd := &cobra.Command{
		Use:   "tag",
		Short: fmt.Sprintf("Attach or detach tags on %ss", entityName),
		Long: fmt.Sprintf(`Retroactively attach or detach registered tags on an existing %s.

Tags must already exist in the vocabulary — register new tags first with
'shark tags add <name>'. Attach is idempotent (re-running 'tag add' with the
same name is a no-op). Detach is idempotent (re-running 'tag rm' on an
unattached tag is a no-op, but the tag name must still be in the vocabulary).`, entityName),
	}

	addCmd := &cobra.Command{
		Use:   fmt.Sprintf("add <%s-key> <name>", entityName),
		Short: fmt.Sprintf("Attach a registered tag to %s", articleEntity(entityName)),
		Long: fmt.Sprintf(`Attach a registered tag to %s. The tag must exist in the
vocabulary; run 'shark tags list' to see available tags or
'shark tags add <name>' to register a new one.

Idempotency: re-running with the same arguments is a no-op (exit 0).`, articleEntity(entityName)),
		Args: cobra.ExactArgs(2),
		RunE: makeEntityTagAdd(entityType, resolveKey, svcOverride),
	}

	rmCmd := &cobra.Command{
		Use:   fmt.Sprintf("rm <%s-key> <name>", entityName),
		Short: fmt.Sprintf("Detach a registered tag from %s", articleEntity(entityName)),
		Long: fmt.Sprintf(`Detach a registered tag from %s. The tag name must exist in
the vocabulary (exit 1 otherwise). Removing a tag that is not currently
attached is a no-op (exit 0).`, articleEntity(entityName)),
		Args: cobra.ExactArgs(2),
		RunE: makeEntityTagRm(entityType, resolveKey, svcOverride),
	}

	tagCmd.AddCommand(addCmd)
	tagCmd.AddCommand(rmCmd)
	return tagCmd
}

// makeEntityTagAdd returns the RunE handler for `shark <entity> tag add`.
// Kept as a factory (rather than a closure inline above) so the returned
// function is testable in isolation and so the command definition in
// makeEntityTagCmd stays readable.
func makeEntityTagAdd(
	entityType models.EntityType,
	resolveKey EntityKeyResolver,
	svcOverride entityTagServiceIface,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key, name := args[0], args[1]

		// Resolve the entity first. If the entity key does not exist the
		// user sees a "not found" error before the tag service is even
		// touched (REQ-F-016 exit code 1, handled by the usual error
		// propagation through cobra's RunE).
		id, err := resolveKey(cmd.Context(), key)
		if err != nil {
			return fmt.Errorf("%s %q not found: %w", entityType, key, err)
		}

		svc := resolveEntityTagService(svcOverride)

		if attachErr := svc.AttachMany(cmd.Context(), entityType, id, []string{name}); attachErr != nil {
			// The vocabulary-snippet helper handles *UnregisteredTagError,
			// *NotFoundError, *ValidationError, and unauthorized errors.
			// Other errors (e.g., DB) fall through to an exit-2 error.
			return handleVocabularyErrorWithSnippet(cmd, svc, name, attachErr)
		}

		// Success output. JSON mode emits a minimal record; plain text uses
		// the project's standard Success formatter. The payload mirrors the
		// shape used by other tag-related success paths (e.g., tags add).
		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]any{
				"entity_type": string(entityType),
				"entity_key":  key,
				"tag":         name,
				"attached":    true,
			})
		}
		cli.Success(fmt.Sprintf("Attached tag %q to %s %s", name, entityType, key))
		return nil
	}
}

// makeEntityTagRm returns the RunE handler for `shark <entity> tag rm`.
func makeEntityTagRm(
	entityType models.EntityType,
	resolveKey EntityKeyResolver,
	svcOverride entityTagServiceIface,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		key, name := args[0], args[1]

		id, err := resolveKey(cmd.Context(), key)
		if err != nil {
			return fmt.Errorf("%s %q not found: %w", entityType, key, err)
		}

		svc := resolveEntityTagService(svcOverride)

		if detachErr := svc.DetachOne(cmd.Context(), entityType, id, name); detachErr != nil {
			// NotFoundError from DetachOne means the vocabulary does not
			// contain the name; feed through the shared helper so the
			// user gets the SC-2 snippet + remediation line.
			return handleVocabularyErrorWithSnippet(cmd, svc, name, detachErr)
		}

		if cli.GlobalConfig.JSON {
			return cli.OutputJSON(map[string]any{
				"entity_type": string(entityType),
				"entity_key":  key,
				"tag":         name,
				"detached":    true,
			})
		}
		cli.Success(fmt.Sprintf("Detached tag %q from %s %s", name, entityType, key))
		return nil
	}
}

// resolveEntityTagService returns the injected override (tests) or the
// real production tag service. *services.TagService satisfies
// entityTagServiceIface, so the concrete type is returned directly.
func resolveEntityTagService(svc entityTagServiceIface) entityTagServiceIface {
	if svc != nil {
		return svc
	}
	return cli.GetTagService()
}
