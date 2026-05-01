package fileops

import "testing"

func TestReplaceBodyAfterFrontmatter_EmptyBody_ReturnsRenderedUnchanged(t *testing.T) {
	rendered := "---\nkey: value\n---\n\n# Title\n\nbody text\n"
	got := ReplaceBodyAfterFrontmatter(rendered, "")
	if got != rendered {
		t.Errorf("expected unchanged rendered string, got %q", got)
	}
}

func TestReplaceBodyAfterFrontmatter_WithFrontmatter_ReplacesBodyOnly(t *testing.T) {
	rendered := "---\nkey: value\nstatus: todo\n---\n\n# Old Title\n\nold body\n"
	newBody := "## New Body\n\nfresh content"
	got := ReplaceBodyAfterFrontmatter(rendered, newBody)

	wantPrefix := "---\nkey: value\nstatus: todo\n---\n"
	if got[:len(wantPrefix)] != wantPrefix {
		t.Errorf("frontmatter not preserved; got prefix %q", got[:len(wantPrefix)])
	}
	wantSuffix := "## New Body\n\nfresh content\n"
	if got[len(wantPrefix):] != wantSuffix {
		t.Errorf("body region wrong; got %q", got[len(wantPrefix):])
	}
}

func TestReplaceBodyAfterFrontmatter_NoFrontmatter_ReturnsBodyAlone(t *testing.T) {
	rendered := "# Plain Markdown\n\nno frontmatter here\n"
	newBody := "fresh"
	got := ReplaceBodyAfterFrontmatter(rendered, newBody)
	if got != "fresh\n" {
		t.Errorf("expected %q, got %q", "fresh\n", got)
	}
}

func TestReplaceBodyAfterFrontmatter_TrailingNewlineNormalized(t *testing.T) {
	rendered := "---\nk: v\n---\nold\n"
	// Body without trailing newline — function must add exactly one.
	got := ReplaceBodyAfterFrontmatter(rendered, "no trailing newline")
	want := "---\nk: v\n---\nno trailing newline\n"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	// Body with extra trailing newlines — function must normalize to one.
	got = ReplaceBodyAfterFrontmatter(rendered, "many\n\n\n")
	want = "---\nk: v\n---\nmany\n"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}
