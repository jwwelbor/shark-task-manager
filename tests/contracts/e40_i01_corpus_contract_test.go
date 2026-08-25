// Package contracts verifies the I-01 corpus and oracle contract shared
// between E40-F01 (producer) and E40-F02 (consumer). Per spec.md ADR-F01-05,
// this is the one deliberate Go addition E40-F01 makes to the shark module,
// and E40-F02 must reuse TestTC001_I01CorpusAndOracleContract verbatim
// rather than write a second reader.
package contracts

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// e40I01SupportedSchemaVersion is the corpus.yaml schema_version this
// validator understands (AC-002).
const e40I01SupportedSchemaVersion = "1.0"

type e40CorpusToolchain struct {
	GoVersion            string `yaml:"go_version" json:"go_version"`
	GolangciLintVersion  string `yaml:"golangci_lint_version" json:"golangci_lint_version"`
	GOOS                 string `yaml:"goos" json:"goos"`
	GOARCH               string `yaml:"goarch" json:"goarch"`
	GolangciConfigSHA256 string `yaml:"golangci_config_sha256" json:"golangci_config_sha256"`
}

type e40CorpusFixture struct {
	SubmodulePath string             `yaml:"submodule_path"`
	BaseSHA       string             `yaml:"base_sha"`
	Toolchain     e40CorpusToolchain `yaml:"toolchain"`
}

type e40CorpusP2PSet struct {
	Packages     []string `yaml:"packages"`
	RunSelector  string   `yaml:"run_selector"`
	ExcludeTests []string `yaml:"exclude_tests"`
}

type e40CorpusF2P struct {
	Paths     []string `yaml:"paths"`
	TestNames []string `yaml:"test_names"`
}

type e40CorpusItem struct {
	ID                 string       `yaml:"id"`
	Type               string       `yaml:"type"`
	PromptPath         string       `yaml:"prompt_path"`
	SeedPath           string       `yaml:"seed_path"`
	F2P                e40CorpusF2P `yaml:"f2p"`
	P2PSet             string       `yaml:"p2p_set"`
	ReferencePatchPath string       `yaml:"reference_patch_path"`
	FixtureBaseSHA     string       `yaml:"fixture_base_sha"`
}

type e40CorpusManifest struct {
	SchemaVersion string                     `yaml:"schema_version"`
	Fixture       e40CorpusFixture           `yaml:"fixture"`
	P2PSets       map[string]e40CorpusP2PSet `yaml:"p2p_sets"`
	Items         []e40CorpusItem            `yaml:"items"`
	NegativeItems []e40CorpusItem            `yaml:"negative_items"`
}

// e40SeedSpec is REQ-F-002's "entity seed spec": the title/description every
// item carries and the severity a bug item additionally carries, stored in
// the file corpus.yaml's seed_path points at.
type e40SeedSpec struct {
	Type        string `yaml:"type"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Severity    string `yaml:"severity"`
}

// e40LedgerDoc mirrors the toolchain block REQ-F-010 requires both the
// base-SHA test ledger and lint ledger to carry. Entries are not decoded:
// AC-013's cross-encoding check is about the toolchain block only.
type e40LedgerDoc struct {
	Toolchain e40CorpusToolchain `json:"toolchain"`
}

// e40BrokenManifestYAML is a deliberately malformed corpus manifest used
// only by TestTC001_I01CorpusAndOracleContract's negative subtests, never by
// the real-file assertion. %s is substituted with the item's `type` value
// and p2p_set reference so each subtest can target one violated field.
const e40BrokenManifestYAML = `
schema_version: %q
fixture:
  submodule_path: bench/fixture-repo
  base_sha: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
  toolchain:
    go_version: "go1.26.0"
    golangci_lint_version: "v2.9.0"
    goos: "linux"
    goarch: "amd64"
    golangci_config_sha256: "sha"
p2p_sets:
  default:
    packages: ["./..."]
    run_selector: ""
    exclude_tests: []
items:
  - id: broken-item
    type: %q
    prompt_path: items/broken-item/prompt.md
    seed_path: items/broken-item/seed.yaml
    f2p:
      paths: ["items/broken-item/testdata/f2p/broken_test.go"]
      test_names: ["pkg::TestBroken"]
    p2p_set: %q
    reference_patch_path: items/broken-item/reference.patch
    fixture_base_sha: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
negative_items: []
`

// e40BrokenToolchainManifestYAML is otherwise well-formed but omits every
// fixture.toolchain field, used only by TestTC001_I01CorpusAndOracleContract's
// toolchain-field-inventory negative subtest.
const e40BrokenToolchainManifestYAML = `
schema_version: "1.0"
fixture:
  submodule_path: bench/fixture-repo
  base_sha: "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
  toolchain:
    go_version: ""
    golangci_lint_version: ""
    goos: ""
    goarch: ""
    golangci_config_sha256: ""
p2p_sets:
  default:
    packages: ["./..."]
    run_selector: ""
    exclude_tests: []
items: []
negative_items: []
`

// TC-001 reads the real committed bench/corpus/corpus.yaml, item directories,
// and ledgers via os.ReadFile plus real YAML/JSON unmarshal (test-plan.md
// Caller-Path Contracts) and proves the REQ-F-002 field inventory (AC-002)
// and the base-SHA cross-encoding agreement (AC-013). E40-F02 reuses this
// exact function name rather than writing a second reader.
func TestTC001_I01CorpusAndOracleContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	corpusRoot := filepath.Join(repoRoot, "bench", "corpus")
	manifestPath := filepath.Join(corpusRoot, "corpus.yaml")

	t.Run("committed_manifest_and_oracle_shape", func(t *testing.T) {
		manifest := e40ReadManifest(t, manifestPath)
		if len(manifest.Items) == 0 {
			t.Fatal("corpus.yaml items is empty")
		}
		if len(manifest.NegativeItems) == 0 {
			t.Fatal("corpus.yaml negative_items is empty")
		}
		if errs := e40ValidateManifest(manifest, corpusRoot); len(errs) > 0 {
			t.Fatalf("corpus.yaml violates the REQ-F-002 field inventory:\n%s", strings.Join(errs, "\n"))
		}
		e40ValidateCrossEncoding(t, manifest, corpusRoot)
	})

	// AC-T3 (test-plan.md negative case): a manifest item whose p2p_set is
	// absent from p2p_sets must be rejected with that field named. This
	// proves the validator would actually catch corpus.yaml drift rather
	// than vacuously passing.
	t.Run("unresolved_p2p_set_is_rejected", func(t *testing.T) {
		manifest := e40ReadBrokenManifest(t, e40I01SupportedSchemaVersion, "task", "nonexistent_set")
		errs := e40ValidateManifest(manifest, t.TempDir())
		if !e40ContainsErrorMatching(errs, "p2p_set", "not found in p2p_sets") {
			t.Fatalf("expected an unresolved p2p_set error, got: %v", errs)
		}
	})

	// A wrongly typed `type` field must be rejected with that field named.
	t.Run("wrong_typed_item_type_is_rejected", func(t *testing.T) {
		manifest := e40ReadBrokenManifest(t, e40I01SupportedSchemaVersion, "epic", "default")
		errs := e40ValidateManifest(manifest, t.TempDir())
		if !e40ContainsErrorMatching(errs, "type", `want "task" or "bug"`) {
			t.Fatalf(`expected a type-not-task-or-bug error, got: %v`, errs)
		}
	})

	// An unsupported schema_version must be rejected with that field named.
	t.Run("unsupported_schema_version_is_rejected", func(t *testing.T) {
		manifest := e40ReadBrokenManifest(t, "9.9", "task", "default")
		errs := e40ValidateManifest(manifest, t.TempDir())
		if !e40ContainsErrorMatching(errs, "schema_version") {
			t.Fatalf("expected a schema_version error, got: %v", errs)
		}
	})

	// REQ-F-010 requires the manifest to carry the toolchain values. Without
	// this check, a missing fixture.toolchain block would compare as an
	// equal zero-value struct against equally-missing ledger toolchain
	// blocks in e40ValidateCrossEncoding and pass vacuously.
	t.Run("blank_toolchain_fields_are_rejected", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "corpus.yaml")
		if err := os.WriteFile(path, []byte(e40BrokenToolchainManifestYAML), 0o644); err != nil {
			t.Fatalf("write broken toolchain manifest fixture: %v", err)
		}
		manifest := e40ReadManifest(t, path)
		errs := e40ValidateManifest(manifest, t.TempDir())
		for _, field := range []string{"go_version", "golangci_lint_version", "goos", "goarch", "golangci_config_sha256"} {
			if !e40ContainsErrorMatching(errs, "fixture.toolchain", field, "is empty") {
				t.Errorf("expected a fixture.toolchain %s empty error, got: %v", field, errs)
			}
		}
	})
}

// e40ReadManifest reads and parses the real committed corpus.yaml at path.
func e40ReadManifest(t *testing.T, path string) *e40CorpusManifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus manifest %s: %v", path, err)
	}
	var manifest e40CorpusManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse corpus manifest %s: %v", path, err)
	}
	return &manifest
}

// e40ReadBrokenManifest writes a deliberately malformed manifest to a temp
// file and reads it back through the same os.ReadFile + yaml.Unmarshal path
// as e40ReadManifest, so the negative subtests exercise the real parser
// rather than a hand-built struct.
func e40ReadBrokenManifest(t *testing.T, schemaVersion, itemType, p2pSet string) *e40CorpusManifest {
	t.Helper()
	content := fmt.Sprintf(e40BrokenManifestYAML, schemaVersion, itemType, p2pSet)
	path := filepath.Join(t.TempDir(), "corpus.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write broken manifest fixture: %v", err)
	}
	return e40ReadManifest(t, path)
}

func e40ContainsErrorMatching(errs []string, substrings ...string) bool {
	for _, e := range errs {
		matched := true
		for _, s := range substrings {
			if !strings.Contains(e, s) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// e40ValidateManifest applies the REQ-F-002/AC-002 field inventory to a
// parsed corpus manifest, returning one description per violation.
// corpusRoot is the directory item-relative paths (prompt_path, seed_path,
// f2p.paths, reference_patch_path) resolve against.
func e40ValidateManifest(manifest *e40CorpusManifest, corpusRoot string) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Sprintf(format, args...))
	}

	if manifest.SchemaVersion != e40I01SupportedSchemaVersion {
		addf("schema_version = %q, want %q", manifest.SchemaVersion, e40I01SupportedSchemaVersion)
	}
	errs = append(errs, e40ValidateToolchain("fixture.toolchain", manifest.Fixture.Toolchain)...)
	if len(manifest.P2PSets) == 0 {
		addf("p2p_sets is empty")
	}
	for name, set := range manifest.P2PSets {
		if len(set.Packages) == 0 {
			addf("p2p_sets[%s]: packages is empty", name)
		}
	}

	for _, item := range manifest.Items {
		errs = append(errs, e40ValidateItem("items", item, manifest, corpusRoot)...)
	}
	for _, item := range manifest.NegativeItems {
		errs = append(errs, e40ValidateItem("negative_items", item, manifest, corpusRoot)...)
	}

	return errs
}

// e40ValidateItem checks one manifest item against the REQ-F-002 field
// inventory: id/type well-typed, prompt/seed/patch/F2P files exist, the
// seed spec itself is well-typed, p2p_set resolves, and fixture_base_sha
// agrees with the manifest's top-level pin (the corpus.yaml header comment's
// stated invariant).
func e40ValidateItem(group string, item e40CorpusItem, manifest *e40CorpusManifest, corpusRoot string) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		label := fmt.Sprintf("%s[%s]", group, item.ID)
		errs = append(errs, fmt.Sprintf(label+": "+format, args...))
	}

	if strings.TrimSpace(item.ID) == "" {
		addf("id is empty")
	}
	if item.Type != "task" && item.Type != "bug" {
		addf("type = %q, want \"task\" or \"bug\"", item.Type)
	}

	if strings.TrimSpace(item.PromptPath) == "" {
		addf("prompt_path is empty")
	} else if !e40FileExists(filepath.Join(corpusRoot, item.PromptPath)) {
		addf("prompt_path %q does not exist", item.PromptPath)
	}

	errs = append(errs, e40ValidateSeed(group, item, corpusRoot)...)

	if len(item.F2P.Paths) == 0 {
		addf("f2p.paths is empty")
	}
	for _, p := range item.F2P.Paths {
		if !e40FileExists(filepath.Join(corpusRoot, p)) {
			addf("f2p path %q does not exist", p)
		}
	}
	if len(item.F2P.TestNames) == 0 {
		addf("f2p.test_names is empty")
	}
	for _, name := range item.F2P.TestNames {
		if !strings.Contains(name, "::") {
			addf("f2p test name %q is not <package>::<test> shaped", name)
		}
	}

	if strings.TrimSpace(item.P2PSet) == "" {
		addf("p2p_set is empty")
	} else if _, ok := manifest.P2PSets[item.P2PSet]; !ok {
		addf("p2p_set %q not found in p2p_sets", item.P2PSet)
	}

	if strings.TrimSpace(item.ReferencePatchPath) == "" {
		addf("reference_patch_path is empty")
	} else if !e40FileExists(filepath.Join(corpusRoot, item.ReferencePatchPath)) {
		addf("reference_patch_path %q does not exist", item.ReferencePatchPath)
	}

	if strings.TrimSpace(item.FixtureBaseSHA) == "" {
		addf("fixture_base_sha is empty")
	} else if item.FixtureBaseSHA != manifest.Fixture.BaseSHA {
		addf("fixture_base_sha %q != fixture.base_sha %q", item.FixtureBaseSHA, manifest.Fixture.BaseSHA)
	}

	return errs
}

// e40ValidateSeed checks the entity seed spec (REQ-F-002: title, description;
// bugs add severity) that item.SeedPath references.
func e40ValidateSeed(group string, item e40CorpusItem, corpusRoot string) []string {
	var errs []string
	addf := func(format string, args ...interface{}) {
		label := fmt.Sprintf("%s[%s]", group, item.ID)
		errs = append(errs, fmt.Sprintf(label+": "+format, args...))
	}

	if strings.TrimSpace(item.SeedPath) == "" {
		addf("seed_path is empty")
		return errs
	}
	seedFullPath := filepath.Join(corpusRoot, item.SeedPath)
	if !e40FileExists(seedFullPath) {
		addf("seed_path %q does not exist", item.SeedPath)
		return errs
	}

	seedBytes, err := os.ReadFile(seedFullPath)
	if err != nil {
		addf("seed_path %q unreadable: %v", item.SeedPath, err)
		return errs
	}
	var seed e40SeedSpec
	if err := yaml.Unmarshal(seedBytes, &seed); err != nil {
		addf("seed_path %q invalid YAML: %v", item.SeedPath, err)
		return errs
	}
	if seed.Type != item.Type {
		addf("seed type = %q, want manifest type %q", seed.Type, item.Type)
	}
	if strings.TrimSpace(seed.Title) == "" {
		addf("seed title is empty")
	}
	if strings.TrimSpace(seed.Description) == "" {
		addf("seed description is empty")
	}
	if item.Type == "bug" && strings.TrimSpace(seed.Severity) == "" {
		addf("bug seed missing severity")
	}
	return errs
}

func e40FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// e40ValidateToolchain checks REQ-F-010's toolchain field inventory is
// present. Without this, a toolchain block missing entirely from
// corpus.yaml or a ledger would compare as an equal zero-value struct in
// e40ValidateCrossEncoding and pass vacuously.
func e40ValidateToolchain(label string, tc e40CorpusToolchain) []string {
	var errs []string
	fields := map[string]string{
		"go_version":             tc.GoVersion,
		"golangci_lint_version":  tc.GolangciLintVersion,
		"goos":                   tc.GOOS,
		"goarch":                 tc.GOARCH,
		"golangci_config_sha256": tc.GolangciConfigSHA256,
	}
	for _, name := range []string{"go_version", "golangci_lint_version", "goos", "goarch", "golangci_config_sha256"} {
		if strings.TrimSpace(fields[name]) == "" {
			errs = append(errs, fmt.Sprintf("%s: %s is empty", label, name))
		}
	}
	return errs
}

// e40ValidateCrossEncoding proves AC-013: corpus.yaml's fixture.base_sha
// equals the bench/corpus/ledgers/<base_sha>/ directory name, and both
// ledgers' toolchain blocks equal the manifest's toolchain pins. It reads
// bench/corpus/ledgers/ itself (not just the base_sha-derived path) so a
// stale sibling directory left over from a prior base_sha is caught rather
// than silently ignored — "equals the directory name" means the one and
// only ledgers directory, not merely a directory that happens to exist.
func e40ValidateCrossEncoding(t *testing.T, manifest *e40CorpusManifest, corpusRoot string) {
	t.Helper()
	baseSHA := manifest.Fixture.BaseSHA
	if strings.TrimSpace(baseSHA) == "" {
		t.Fatal("fixture.base_sha is empty")
	}
	ledgersRoot := filepath.Join(corpusRoot, "ledgers")
	entries, err := os.ReadDir(ledgersRoot)
	if err != nil {
		t.Fatalf("read ledgers directory %s: %v", ledgersRoot, err)
	}
	var dirNames []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}
	if len(dirNames) != 1 || dirNames[0] != baseSHA {
		t.Fatalf("bench/corpus/ledgers/ directories = %v, want exactly one directory named fixture.base_sha %q", dirNames, baseSHA)
	}
	ledgerDir := filepath.Join(ledgersRoot, baseSHA)

	want := manifest.Fixture.Toolchain
	testLedger := e40ReadLedger(t, filepath.Join(ledgerDir, "tests.json"))
	if testLedger.Toolchain != want {
		t.Errorf("bench/corpus/ledgers/%s/tests.json toolchain = %#v, want manifest toolchain %#v", baseSHA, testLedger.Toolchain, want)
	}
	lintLedger := e40ReadLedger(t, filepath.Join(ledgerDir, "lint.json"))
	if lintLedger.Toolchain != want {
		t.Errorf("bench/corpus/ledgers/%s/lint.json toolchain = %#v, want manifest toolchain %#v", baseSHA, lintLedger.Toolchain, want)
	}
}

func e40ReadLedger(t *testing.T, path string) e40LedgerDoc {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ledger %s: %v", path, err)
	}
	var doc e40LedgerDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse ledger %s: %v", path, err)
	}
	return doc
}

// TC-002 proves the fixture module and its held-back F2P tests stay
// invisible to the shark module's own `go list ./...` (AC-011's
// package-visibility half). It derives the expected package list from a live
// `go list ./...` invocation, never a hardcoded list, and asserts this
// unconditionally regardless of submodule state, since CI's
// actions/checkout@v4 never initializes bench/fixture-repo (task AC-T3). The
// stronger check — that the fixture's own go.mod is present at the pinned
// submodule path — only runs when the submodule is actually populated; it
// must never run, let alone fail, when it is gitlink-only.
func TestTC002_I01FixturePackageVisibilityContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	// The shark module's own path is derived live rather than hardcoded, so
	// a future rename of the fixture module (or of this module) cannot
	// silently defeat the guard below.
	moduleCmd := exec.Command("go", "list", "-buildvcs=false", "-m")
	moduleCmd.Dir = repoRoot
	moduleOutput, err := moduleCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -m failed: %v\n%s", err, moduleOutput)
	}
	ownModule := strings.TrimSpace(string(moduleOutput))
	if ownModule == "" {
		t.Fatal("go list -m returned an empty module path")
	}

	cmd := exec.Command("go", "list", "-buildvcs=false", "./...")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list ./... failed: %v\n%s", err, output)
	}
	packages := strings.Fields(string(output))
	if len(packages) == 0 {
		t.Fatal("go list ./... returned no packages")
	}
	for _, pkg := range packages {
		if pkg != ownModule && !strings.HasPrefix(pkg, ownModule+"/") {
			t.Errorf("go list ./... lists package %q outside the shark module %q, want no fixture or other foreign package", pkg, ownModule)
		}
		if strings.Contains(pkg, "testdata/f2p") {
			t.Errorf("go list ./... lists held-back-test package %q, want none", pkg)
		}
	}

	fixtureRepoPath := filepath.Join(repoRoot, "bench", "fixture-repo")
	entries, err := os.ReadDir(fixtureRepoPath)
	if err != nil || len(entries) == 0 {
		// Gitlink-only submodule: CI's normal state. The stronger go.mod
		// presence check below must never run or fail here.
		return
	}
	if _, err := os.Stat(filepath.Join(fixtureRepoPath, "go.mod")); err != nil {
		t.Fatalf("populated fixture submodule is missing its own go.mod at %s: %v", fixtureRepoPath, err)
	}
}
