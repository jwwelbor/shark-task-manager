package runner

import (
	"errors"
	"strings"
	"testing"
)

// TestShellQuote_Empty verifies that an empty string is quoted as ” to
// preserve argument identity when passed through a POSIX shell.
func TestShellQuote_Empty(t *testing.T) {
	got, err := shellQuote("")
	if err != nil {
		t.Fatalf("shellQuote(%q) unexpected error = %v", "", err)
	}
	want := "''"
	if got != want {
		t.Errorf("shellQuote(%q) = %q, want %q", "", got, want)
	}
}

// TestShellQuote_SafeInputs verifies that inputs composed of only safe ASCII
// characters (alphanumerics + /-_.:=@+,) are returned unquoted. This keeps
// logged command strings readable in the common case.
func TestShellQuote_SafeInputs(t *testing.T) {
	safeInputs := []string{
		"plain",
		"PLAIN",
		"claude",
		"codex",
		"E07-F41-003",
		"1234567890",
		"--output-format",
		"json",
		"--max-turns",
		"--allowedTools",
		"Read,Write,Bash",
		"path/to/file",
		"some_word",
		"some-word",
		"file.ext",
		"key:value",
		"key=value",
		"user@host",
		"a+b",
		"a,b",
		"a/b/c",
	}
	for _, in := range safeInputs {
		got, err := shellQuote(in)
		if err != nil {
			t.Errorf("shellQuote(%q) unexpected error = %v", in, err)
			continue
		}
		if got != in {
			t.Errorf("shellQuote(%q) = %q, want unquoted %q", in, got, in)
		}
	}
}

// TestShellQuote_Spaces verifies that inputs containing spaces are wrapped
// in single quotes so shell tokenization preserves argument boundaries.
func TestShellQuote_Spaces(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"do work now", "'do work now'"},
		{" leading", "' leading'"},
		{"trailing ", "'trailing '"},
		{"multi  space", "'multi  space'"},
	}
	for _, c := range cases {
		got, err := shellQuote(c.in)
		if err != nil {
			t.Errorf("shellQuote(%q) unexpected error = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestShellQuote_SingleQuotes verifies POSIX-safe escaping of embedded
// single quotes. The canonical form is: close quote, escaped quote, reopen
// quote. e.g. it's -> 'it'\”s'.
func TestShellQuote_SingleQuotes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"it's", `'it'\''s'`},
		{"'", `''\'''`},
		{"'start", `''\''start'`},
		{"end'", `'end'\'''`},
		{"a'b'c", `'a'\''b'\''c'`},
	}
	for _, c := range cases {
		got, err := shellQuote(c.in)
		if err != nil {
			t.Errorf("shellQuote(%q) unexpected error = %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("shellQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestShellQuote_Metacharacters verifies that every common shell
// metacharacter forces single-quote wrapping. Inside single quotes every
// character is literal (except ' itself), so no further escaping is needed.
func TestShellQuote_Metacharacters(t *testing.T) {
	metaChars := []string{
		";", "&", "|", "$", "`", "*", "?", "(", ")", "[", "]", "{", "}",
		"#", "~", "!", "<", ">", "\\", "\"", "\t", "\n", " ",
	}
	for _, m := range metaChars {
		in := "a" + m + "b"
		got, err := shellQuote(in)
		if err != nil {
			t.Errorf("shellQuote(%q) unexpected error = %v", in, err)
			continue
		}
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("shellQuote(%q) = %q, want single-quoted output for metacharacter %q", in, got, m)
		}
	}
}

// TestShellQuote_NUL_RejectsWithError verifies that NUL bytes are rejected
// with the errShellQuoteNUL sentinel. os/exec's argv execution path rejects
// any argv element containing NUL with EINVAL before the subprocess is
// spawned; reporting it up-front via shellQuote lets the caller (controller
// dispatch) surface the condition as a structured run.stage.error with
// phase="shell_quote" instead of silently producing a command string that
// cannot be executed.
func TestShellQuote_NUL_RejectsWithError(t *testing.T) {
	cases := []string{
		"\x00",           // lone NUL
		"a\x00b",         // embedded NUL
		"\x00prefix",     // leading NUL
		"suffix\x00",     // trailing NUL
		"mid\x00dle\x00", // multiple NUL
	}
	for _, in := range cases {
		got, err := shellQuote(in)
		if err == nil {
			t.Errorf("shellQuote(%q) = %q, want error", in, got)
			continue
		}
		if !errors.Is(err, errShellQuoteNUL) {
			t.Errorf("shellQuote(%q) error = %v, want errShellQuoteNUL", in, err)
		}
		if got != "" {
			t.Errorf("shellQuote(%q) returned non-empty string %q alongside error", in, got)
		}
	}
}

// TestShellQuote_RoundTripViaShlex verifies that shell-tokenizing the output
// of shellQuote round-trips back to the original argv element. This is the
// definitive correctness test: if this passes, the output is truly
// "shell-equivalent".
func TestShellQuote_RoundTripViaShlex(t *testing.T) {
	inputs := []string{
		"",
		"plain",
		"do work now",
		"it's fine",
		"a;b&c|d",
		`$USER`,
		"back`tick`",
		"with\"double",
		"tab\there",
		"new\nline",
		"'",
		"''",
		"a'b'c",
		"  spaces  ",
		"mix: $foo 'bar' \"baz\"",
	}
	for _, in := range inputs {
		quoted, err := shellQuote(in)
		if err != nil {
			t.Errorf("shellQuote(%q) unexpected error = %v", in, err)
			continue
		}
		tokens, err := shellSplitForTest(quoted)
		if err != nil {
			t.Errorf("shellSplitForTest(%q) error = %v (quoted=%q)", in, err, quoted)
			continue
		}
		if len(tokens) != 1 {
			t.Errorf("shellQuote(%q) = %q, shellSplit produced %d tokens, want 1: %v",
				in, quoted, len(tokens), tokens)
			continue
		}
		if tokens[0] != in {
			t.Errorf("round-trip failed for %q: quoted=%q, split=%q", in, quoted, tokens[0])
		}
	}
}

// shellSplitForTest is a minimal POSIX-ish shell tokenizer sufficient to
// round-trip the output of shellQuote. It handles:
//   - whitespace separators
//   - single-quoted strings (literal, no escapes)
//   - backslash escapes outside quotes
//
// It does NOT handle double-quoted strings, variable expansion, command
// substitution, globbing, etc. — shellQuote never emits those constructs.
func shellSplitForTest(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inToken := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
			inToken = true
		case c == '\\':
			if i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
				inToken = true
			}
		case c == ' ' || c == '\t' || c == '\n':
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteByte(c)
			inToken = true
		}
	}

	if inSingle {
		return nil, errUnterminatedQuote
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}

// errUnterminatedQuote is a sentinel error for the test-only tokenizer.
var errUnterminatedQuote = &shellSplitError{msg: "unterminated single quote"}

type shellSplitError struct{ msg string }

func (e *shellSplitError) Error() string { return e.msg }
