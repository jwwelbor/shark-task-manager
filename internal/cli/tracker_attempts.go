package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/trackerattempts"
)

// readOnlyLeaves are commands that never mutate the tracker. Everything else,
// including an unresolvable command line, is recorded as a write attempt.
var readOnlyLeaves = map[string]bool{
	"get": true, "list": true, "search": true, "view": true, "show": true,
	"progress": true, "analytics": true, "history": true, "transitions": true,
	"notes": true, "timeline": true, "deps": true, "resume": true, "claims": true,
	"blocked-by": true, "blocks": true, "sessions": true, "plan": true,
	"version": true, "help": true,
}

// RecordTrackerAttempt appends the invocation to the dispatching parent's
// ledger. It runs before cobra parses or validates anything, so a command that
// is rejected before it mutates is still observed. It does nothing outside a
// dispatched run.
func RecordTrackerAttempt(args []string) {
	path, runID := os.Getenv(trackerattempts.EnvLog), os.Getenv(trackerattempts.EnvRun)
	if path == "" || runID == "" {
		return
	}
	command, write := classifyTrackerAttempt(args)
	// A ledger that cannot be written must not hide the attempt from the
	// parent: fail the invocation rather than let it proceed unrecorded.
	if err := trackerattempts.Append(path, runID, command, args, write); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "shark: cannot record tracker attempt: %v\n", err)
		os.Exit(3)
	}
}

func classifyTrackerAttempt(args []string) (string, bool) {
	cmd, rest, err := RootCmd.Find(args)
	if err != nil || cmd == nil {
		return "unresolved", true
	}
	name := strings.TrimPrefix(cmd.CommandPath(), RootCmd.Name()+" ")
	if cmd == RootCmd {
		for _, arg := range args {
			if arg == "-h" || arg == "--help" || arg == "--version" {
				return "shark", false
			}
		}
		return "shark", true
	}
	if len(rest) > 0 && cmd.HasSubCommands() {
		// An unknown subcommand under a group is a failed attempt, not a read.
		return name, true
	}
	return name, !readOnlyLeaves[cmd.Name()]
}
