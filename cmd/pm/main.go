package main

import (
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	_ "github.com/jwwelbor/shark-task-manager/internal/cli/commands" // Import command packages for side effects
)

func main() {
	if err := cli.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
