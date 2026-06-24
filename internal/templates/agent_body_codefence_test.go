package templates

import "testing"

func TestFirstUnrenderedToken_SkipsCodeFences(t *testing.T) {
	body := "Real prompt line.\n" +
		"\n" +
		"```bash\n" +
		"git commit -m \"<subject>\"\n" +
		"echo <body>\n" +
		"```\n" +
		"\n" +
		"All rendered above.\n"
	if tok, found := FirstUnrenderedToken(body); found {
		t.Errorf("did not expect a token outside code fences, found %q", tok)
	}
}

func TestFirstUnrenderedToken_CatchesOutsideFences(t *testing.T) {
	body := "Real prompt with <task-id> still unrendered.\n" +
		"\n" +
		"```\n<should-skip>\n```\n"
	tok, found := FirstUnrenderedToken(body)
	if !found {
		t.Fatal("expected to find unrendered token outside fences")
	}
	if tok != "<task-id>" {
		t.Errorf("expected <task-id>, got %q", tok)
	}
}

func TestFirstUnrenderedToken_HandlesUnclosedFence(t *testing.T) {
	// An unclosed fence opens a code block that extends to end-of-string.
	// Tokens after the open fence should be skipped.
	body := "Header text.\n```bash\n<should-skip>\n"
	if tok, found := FirstUnrenderedToken(body); found {
		t.Errorf("unexpected match in unclosed fence: %q", tok)
	}
}
