package commands

func init() {
	featureCmd.AddCommand(makeNoteCmd("feature"))
	featureCmd.AddCommand(makeNotesCmd("feature"))
}
