package commands

import "github.com/jwwelbor/shark-task-manager/internal/cli"

func init() {
	featureCmd.AddCommand(makeNextStatusCmd("feature", func() nextStatusGetter {
		return cli.GetFeatureService()
	}))
}
