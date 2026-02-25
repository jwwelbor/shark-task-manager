package commands

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

// TestRunList_FeatureKeyDispatch_DoesNotPanicOnNilContext verifies that
// dispatching `shark list E15-F03` (feature-key format) does NOT panic with a
// nil-pointer dereference.
//
// Root cause: runTaskListWithFlags called runTaskList(taskListCmd, ...) where
// taskListCmd is a package-level cobra.Command whose context field is nil
// (Cobra only sets it during its own Execute path).  The nil context was then
// passed through to sql.DB.QueryContext, which panics.
//
// The fix must propagate the calling command's context to taskListCmd before
// delegating, so that runTaskList receives a non-nil context.
func TestRunList_FeatureKeyDispatch_DoesNotPanicOnNilContext(t *testing.T) {
	// Arrange: a parent command that carries a real context (as Cobra would
	// set on the top-level command during Execute).
	parentCmd := &cobra.Command{Use: "list"}
	parentCmd.SetContext(context.Background())

	// Capture any panic so the test can report it as a failure instead of
	// crashing the entire test binary.
	var panicVal interface{}
	func() {
		defer func() {
			panicVal = recover()
		}()
		// Act: dispatch through the list → task path.
		// "E15" + "F03" is a valid (epic, feature) pair that routes to runTaskList.
		// We don't care whether the query finds rows; we only care that no panic
		// occurs (i.e., the context is non-nil when it reaches the DB layer).
		_ = runTaskListWithFlags(parentCmd, "E15", "F03", "", "", false)
	}()

	// Assert: no panic
	if panicVal != nil {
		t.Errorf("runTaskListWithFlags panicked with nil context: %v", panicVal)
	}
}
