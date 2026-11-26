package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An inlined skill loses its location, so the worker must be told which skill
// its relative paths belong to and how to read them.
func TestIncludeResolver_SkillIncludeCarriesReadingNote(t *testing.T) {
	resolver := NewIncludeResolverWithEmbed("")

	root, err := resolver.Resolve("{{include: skills/debugging/SKILL.md}}")
	require.NoError(t, err)
	assert.Contains(t, root, "`shark skill get debugging <path>`")
	assert.NotContains(t, root, "A path starting `../`")

	nested, err := resolver.Resolve("{{include: skills/quality/workflows/review-code.md}}")
	require.NoError(t, err)
	assert.Contains(t, nested, "A path starting `../` is relative to `workflows/` (for example, read `../context/foo.md` with `shark skill get quality context/foo.md`).")

	augmented, err := resolver.Resolve("{{augment: skills/debugging/SKILL.md}}")
	require.NoError(t, err)
	assert.Contains(t, augmented, "`shark skill get debugging <path>`")
}

// A body with no sibling references has nothing to explain and is inlined
// byte for byte, which the override-replaces-default contract relies on.
func TestBundledSkillNote_OnlyForBodiesWithRelativePaths(t *testing.T) {
	assert.Empty(t, bundledSkillNote("skills/tdd/SKILL.md", "Write a failing test first."))
	assert.Empty(t, bundledSkillNote("skills/tdd/SKILL.md", "See skills/other/workflows/x.md"))
	assert.Empty(t, bundledSkillNote("skills/SKILL.md", "See references/debugging.md"))
	assert.NotEmpty(t, bundledSkillNote("skills/tdd/SKILL.md", "See references/debugging.md"))
	assert.NotEmpty(t, bundledSkillNote("skills/tdd/workflows/run.md", "See ../foo.md"))
	assert.Empty(t, bundledSkillNote("prompts/_partials/x.md", "See references/debugging.md"))
	assert.Empty(t, bundledSkillNote("skills/tdd/schema.yaml", "# see context/x.md"), "a note would corrupt non-markdown content")
}

func TestIncludeResolver_PartialIncludeHasNoSkillNote(t *testing.T) {
	out, err := NewIncludeResolverWithEmbed("").Resolve("{{include: prompts/_partials/_sizing.md}}")
	require.NoError(t, err)
	assert.NotContains(t, out, "shark skill get")
}
