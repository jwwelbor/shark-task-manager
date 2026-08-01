package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	_ "github.com/jwwelbor/shark-task-manager/internal/cli/commands" // Import command packages for side effects
	"github.com/jwwelbor/shark-task-manager/internal/services"
)

// Build-time variables set via -ldflags
var (
	Version   = "dev"
	BuildDate = ""
	GitCommit = ""
)

// extractExitCode inspects err for a leading "exit code N: " prefix produced
// by handleVocabularyErrorWithSnippet and handleEntityServiceError. When
// found it returns N; otherwise it returns the provided fallback.
//
// The prefix format is "exit code <N>: <original error>" where N is a decimal
// integer. This convention is used to thread the correct exit code through the
// cobra RunE chain without requiring every handler to call os.Exit directly.
//
// Spec reference: T-E28-F04-013 FR-3, REQ-F-016.
func extractExitCode(err error, fallback int) int {
	if err == nil {
		return 0
	}
	msg := err.Error()
	const prefix = "exit code "
	if !strings.HasPrefix(msg, prefix) {
		return fallback
	}
	rest := msg[len(prefix):]
	colonIdx := strings.Index(rest, ":")
	if colonIdx < 0 {
		return fallback
	}
	n, parseErr := strconv.Atoi(strings.TrimSpace(rest[:colonIdx]))
	if parseErr != nil {
		return fallback
	}
	return n
}

func main() {
	// Build a descriptive version string for dev builds
	version := Version
	if version == "dev" && (BuildDate != "" || GitCommit != "") {
		version = "dev"
		if GitCommit != "" {
			version += fmt.Sprintf(" (%s)", GitCommit)
		}
		if BuildDate != "" {
			version += fmt.Sprintf(" built %s", BuildDate)
		}
	}

	cli.SetVersion(version)

	if err := cli.RootCmd.Execute(); err != nil {
		// Check for FieldNotFoundError (exit code 4)
		var fieldErr *cli.FieldNotFoundError
		if errors.As(err, &fieldErr) {
			if cli.GlobalConfig.JSON {
				cli.ErrorJSON(cli.CLIError{
					Code:    cli.ErrCodeNotFound,
					Message: fieldErr.Error(),
				})
			} else {
				fmt.Fprintln(os.Stderr, "Error:", fieldErr.Error())
			}
			os.Exit(4)
		}

		// Extract embedded exit code (set by handleVocabularyErrorWithSnippet
		// and handleEntityServiceError via "exit code N: ..." prefix).
		exitCode := extractExitCode(err, 1)

		// In JSON mode, output structured error to stdout.
		if cli.GlobalConfig.JSON {
			var blocked *services.QuestionBlockedError
			if errors.As(err, &blocked) {
				cli.ErrorJSON(cli.CLIError{
					Code:          cli.ErrCodeQuestionBlocked,
					Message:       blocked.Error(),
					Entity:        string(blocked.CandidateType),
					EntityKey:     blocked.CandidateKey,
					QuestionBlock: blocked.QuestionBlock,
				})
				os.Exit(exitCode)
			}
			cli.ErrorJSON(cli.CLIError{
				Code:    cli.ErrCodeCommandError,
				Message: err.Error(),
			})
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(exitCode)
	}
}
