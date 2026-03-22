package commands

import "github.com/jwwelbor/shark-task-manager/internal/cli"

func init() {
	featureCmd.AddCommand(makeSetStatusCmd("feature", func() entityTransitioner {
		return cli.GetFeatureService()
	}))
}
