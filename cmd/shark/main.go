package main

import (
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
		os.Exit(1)
	}
}
