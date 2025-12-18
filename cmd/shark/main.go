package main

import (
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	_ "github.com/jwwelbor/shark-task-manager/internal/cli/commands" // Import command packages for side effects
)

// Version is set at build time via -ldflags
var Version = "dev"

func main() {
	// Set version in CLI before executing
	cli.SetVersion(Version)

	if err := cli.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
