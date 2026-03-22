package commands

func init() {
	featureCmd.AddCommand(makeContextCmd("feature"))
}
