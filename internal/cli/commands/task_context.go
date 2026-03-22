package commands

func init() {
	taskCmd.AddCommand(makeContextCmd("task"))
}
