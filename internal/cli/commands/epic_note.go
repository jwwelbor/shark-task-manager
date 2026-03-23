package commands

func init() {
	epicCmd.AddCommand(makeNoteCmd("epic"))
	epicCmd.AddCommand(makeNotesCmd("epic"))
}
