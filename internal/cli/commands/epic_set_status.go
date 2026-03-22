package commands

import "github.com/jwwelbor/shark-task-manager/internal/cli"

func init() {
	epicCmd.AddCommand(makeSetStatusCmd("epic", func() entityTransitioner {
		return cli.GetEpicService()
	}))
}
