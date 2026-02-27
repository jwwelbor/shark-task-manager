package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	_ "github.com/jwwelbor/shark-task-manager/internal/cli/commands" // Import command packages for side effects
)

// Build-time variables set via -ldflags
var (
	Version   = "dev"
	BuildDate = ""
	GitCommit = ""
)

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

		// In JSON mode, output structured error to stdout
		if cli.GlobalConfig.JSON {
			cli.ErrorJSON(cli.CLIError{
				Code:    cli.ErrCodeCommandError,
				Message: err.Error(),
			})
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}
