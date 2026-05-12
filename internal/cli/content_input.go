package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// ResolveContentInput returns body content to pre-populate a newly created
// entity's markdown file. Resolution precedence:
//
//  1. piped stdin (when stdin is not a TTY) — the entire stream is read
//  2. the --content flag value
//  3. "" (caller should keep the default placeholder body)
//
// The function deliberately tolerates a missing --content flag: commands that
// have not registered the flag will fall through to the empty string. Callers
// pass the returned string as the Body field on the relevant create-input
// DTO; an empty string is interpreted by services as "no override".
func ResolveContentInput(cmd *cobra.Command) (string, error) {
	if fi, err := os.Stdin.Stat(); err == nil {
		// Only consume stdin implicitly when it is a pipe or a regular file
		// (the legitimate `echo … | shark create` or `shark create < file.md`
		// shapes). A bare "not a TTY" check is too broad: process wrappers
		// such as Claude Code connect stdin to a Unix-domain socket that
		// never sends EOF, which would make io.ReadAll block forever.
		mode := fi.Mode()
		if mode&os.ModeNamedPipe != 0 || mode.IsRegular() {
			b, readErr := io.ReadAll(os.Stdin)
			if readErr != nil {
				return "", fmt.Errorf("read stdin: %w", readErr)
			}
			if len(b) > 0 {
				return string(b), nil
			}
		}
	}
	if cmd != nil {
		if v, _ := cmd.Flags().GetString("content"); v != "" {
			return v, nil
		}
	}
	return "", nil
}
