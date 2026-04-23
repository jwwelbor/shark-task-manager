// Package maintainer provides a reusable, tag-agnostic authorization gate
// for admin-level operations in the Shark task manager.
//
// The package exposes the Gate interface, which any admin command can adopt
// with the following ten-line pattern:
//
//	func runAdminPurge(cmd *cobra.Command, args []string) error {
//	    pass, _ := cmd.Flags().GetString("pass")
//	    gate := cli.GetMaintainerGate()
//	    if err := gate.Authorize(cmd.Context(), pass); err != nil {
//	        return err // *UnauthorizedError renders correctly via cli.Error
//	    }
//	    defer func() { _ = gate.RecordSuccess(cmd.Context()) }()
//	    // ... destructive op ...
//	    return nil
//	}
//
// The gate uses a sudo-style file-backed cache so that a sequence of related
// admin operations within the configured window (default 60 seconds) does not
// require re-entering the password on each command.
//
// Design constraints (see spec.md §2.5 and epic ADR-2 through ADR-6):
//   - No shark-domain imports (models, repository, services, workflow, cli).
//   - SHA-256 password hashing with crypto/subtle.ConstantTimeCompare.
//   - Cache file at $XDG_CACHE_HOME/shark/<project-hash>/maintainer.session.
//   - Atomic cache writes via temp-file + os.Rename.
//   - Clock injection via private clock interface for testability.
//
// Spec reference: spec.md REQ-F-001 through REQ-F-010, §2.2.
package maintainer
