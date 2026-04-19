package runner

import (
	"errors"
	"strings"
)

// shellSafeChars is the set of ASCII bytes that can appear unquoted in a
// POSIX sh word. Any character outside this set forces single-quote wrapping.
//
// The set is deliberately conservative: it only contains bytes whose meaning
// is literal in every POSIX shell context (sh, bash, dash, zsh). Characters
// with ambiguous or context-dependent meaning (e.g. `!` in bash history
// expansion, `~` at start-of-word, `#` in comments) are excluded.
const shellSafeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789/-_.:=@+,"

// errShellQuoteNUL is returned by shellQuote/joinCommand when an argument
// contains a NUL byte (0x00). POSIX shells cannot represent NUL in any
// quoting context, and Go's os/exec (syscall/exec_unix.go) rejects argv
// elements containing NUL with EINVAL. Such input would never produce a
// runnable command, so the dispatch-time shell-equivalent contract cannot
// be honored. Callers must surface this condition (e.g. emit
// run.stage.error with phase=shell_quote) and skip Dispatch rather than
// silently logging a command that cannot run.
var errShellQuoteNUL = errors.New("argument contains NUL byte, cannot be represented in a POSIX shell")

// shellQuote returns a POSIX-safe representation of arg suitable for
// inclusion in a shell command line. Splitting the result with a POSIX
// shell (or shellSplitForTest) reproduces arg exactly — no character is
// expanded or interpreted.
//
// Rules:
//   - If arg contains a NUL byte → returns errShellQuoteNUL. NUL cannot
//     be represented in any POSIX shell quoting context, and os/exec
//     rejects NUL-bearing argv with EINVAL, so no shell-equivalent string
//     exists for such input.
//   - Empty string → "”" (preserves argument identity when a shell would
//     otherwise drop it).
//   - All bytes in shellSafeChars → returned unchanged (common-case
//     optimization: logged command strings stay readable).
//   - Otherwise → wrap in single quotes. Each embedded `'` is encoded as
//     `'\”` (close quote, backslash-escaped quote, reopen quote), which
//     is the canonical POSIX-safe idiom and works in every Bourne-family
//     shell.
//
// This function NEVER alters semantic meaning for argv construction — it
// only produces a shell string whose tokenization yields the same argv
// that exec.Command passes directly (argv-first, no shell).
func shellQuote(arg string) (string, error) {
	if strings.IndexByte(arg, 0) >= 0 {
		return "", errShellQuoteNUL
	}
	if arg == "" {
		return "''", nil
	}
	if isShellSafe(arg) {
		return arg, nil
	}
	// Wrap in single quotes, escaping embedded single quotes.
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'", nil
}

// isShellSafe reports whether every byte in s is in shellSafeChars.
func isShellSafe(s string) bool {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(shellSafeChars, rune(s[i])) {
			return false
		}
	}
	return true
}

// joinCommand builds a shell-equivalent command string for binary `bin`
// invoked with `args`. Each element is passed through shellQuote, so the
// output can be pasted into any POSIX shell to reproduce the same argv
// that exec.Command passes to the OS.
//
// Returns errShellQuoteNUL if any element of args (or bin itself) contains
// a NUL byte — the caller must surface this as a dispatch-time error and
// NOT invoke the subprocess, because os/exec would reject it with EINVAL
// anyway.
//
// Both ClaudeDispatcher and CodexDispatcher call this helper in BuildCommand
// AND in Dispatch (the cmdStr passed to execAndCapture) to guarantee the
// two construction sites cannot drift.
func joinCommand(bin string, args []string) (string, error) {
	// bin is always a well-known safe literal ("claude", "codex") but we
	// quote it anyway for consistency and defense-in-depth.
	parts := make([]string, 0, len(args)+1)
	qb, err := shellQuote(bin)
	if err != nil {
		return "", err
	}
	parts = append(parts, qb)
	for _, a := range args {
		qa, err := shellQuote(a)
		if err != nil {
			return "", err
		}
		parts = append(parts, qa)
	}
	return strings.Join(parts, " "), nil
}
