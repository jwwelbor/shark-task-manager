package sharkdata

import (
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// bundleRootedRef matches `skills/<name>/<path>.<ext>` references.
	bundleRootedRef = regexp.MustCompile(`(?:^|[^\w./-])(?:shark-data/)?(skills/[A-Za-z0-9_-]+/[A-Za-z0-9_./-]+\.(?:md|yaml|json|sh|py|ts))`)
	// skillRelativeRef matches a skill's own sibling references.
	skillRelativeRef = regexp.MustCompile(`(?:^|[^\w./<{-])((?:\.\./)*(?:workflows|context|references|assets|scripts)/[A-Za-z0-9_./-]+\.(?:md|yaml|json|sh|py|ts))`)
	includedSkill    = regexp.MustCompile(`\{\{\s*(?:include|augment)\s*:\s*(skills/[^}\s]+)`)
	// sharkInternalRef matches pointers into Shark's own source or design docs,
	// which a worker in another project cannot open.
	sharkInternalRef       = regexp.MustCompile(`(?:^|[^\w./{}-])((?:internal|cmd)/[a-z][\w/.-]*)|architecture\.md#i-0\d`)
	bundleForbiddenPhrases = []string{
		"repo tests use real DB",
		"ask your human partner",
		"human partner's permission",
		"via the Skill tool",
	}
)

func bundleExists(ref string) bool {
	_, err := ReadEmbedded(path.Clean(ref))
	return err == nil
}

func unresolvedRootedRefs(body string) []string {
	var missing []string
	for _, m := range bundleRootedRef.FindAllStringSubmatch(body, -1) {
		// shark-rider is installed in the host, not shipped in this bundle.
		if !strings.HasPrefix(m[1], "skills/shark-rider/") && !bundleExists(m[1]) {
			missing = append(missing, m[1])
		}
	}
	return missing
}

// unresolvedSkillRefs checks relative references from file, which lives in a
// skill. Each must resolve the way the include note tells a worker to read it:
// from the skill root, or for `../` paths from the file's own directory.
func unresolvedSkillRefs(file, body string) []string {
	parts := strings.Split(file, "/")
	if len(parts) < 3 || parts[0] != "skills" {
		return nil
	}
	skillRoot := path.Join(parts[0], parts[1])
	var missing []string
	for _, m := range skillRelativeRef.FindAllStringSubmatch(body, -1) {
		base := skillRoot
		if strings.HasPrefix(m[1], "../") {
			base = path.Dir(file)
		}
		target := path.Join(base, m[1])
		if !strings.HasPrefix(target, skillRoot+"/") || !bundleExists(target) {
			missing = append(missing, m[1])
		}
	}
	return missing
}

func walkBundleText(t *testing.T, visit func(file, body string)) {
	t.Helper()
	for _, root := range []string{"prompts", "agents", "skills"} {
		require.NoError(t, fs.WalkDir(embeddedFS, path.Join(embedRootDir, root), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || (!strings.HasSuffix(p, ".md") && !strings.HasSuffix(p, ".yaml")) {
				return err
			}
			file := strings.TrimPrefix(p, embedRootDir+"/")
			body, readErr := ReadEmbedded(file)
			if readErr != nil {
				return readErr
			}
			visit(file, string(body))
			return nil
		}))
	}
}

func TestEmbedded_BundleReferencesResolve(t *testing.T) {
	walkBundleText(t, func(file, body string) {
		assert.Empty(t, unresolvedRootedRefs(body), "%s names bundle files that do not exist", file)
		// assets/ holds worked example reports about Shark itself; no prompt or
		// workflow points a worker at them.
		if !strings.Contains(file, "/assets/") {
			assert.Empty(t, sharkInternalRef.FindAllString(body, -1), "%s points at Shark internals a worker cannot read", file)
			lower := strings.ToLower(body)
			for _, phrase := range bundleForbiddenPhrases {
				assert.NotContains(t, lower, strings.ToLower(phrase), "%s contains forbidden phrase %q", file, phrase)
			}
		}
	})
}

func TestEmbedded_InlinedSkillRelativeReferencesResolve(t *testing.T) {
	included := map[string]bool{}
	walkBundleText(t, func(_, body string) {
		for _, m := range includedSkill.FindAllStringSubmatch(body, -1) {
			included[m[1]] = true
		}
	})
	require.NotEmpty(t, included)
	for file := range included {
		body, err := ReadEmbedded(file)
		require.NoError(t, err, file)
		assert.Empty(t, unresolvedSkillRefs(file, string(body)), "%s has dead relative references", file)
	}
}

// Counterfactuals: each reference check must be able to fail.
func TestBundleReferenceChecks_RejectBadInput(t *testing.T) {
	assert.NotEmpty(t, unresolvedRootedRefs("read `skills/quality/workflows/renamed.md`"))
	assert.NotEmpty(t, unresolvedRootedRefs("`shark-data/skills/nope/SKILL.md`"))
	assert.NotEmpty(t, unresolvedSkillRefs("skills/debugging/SKILL.md", "Read workflows/renamed.md"))
	assert.NotEmpty(t, unresolvedSkillRefs("skills/quality/workflows/review-code.md", "see `../context/renamed.md`"))
	assert.NotEmpty(t, sharkInternalRef.FindAllString("typed in `internal/gateresult`", -1))
}
