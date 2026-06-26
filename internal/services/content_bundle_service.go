package services

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/sharkdata"
	"github.com/jwwelbor/shark-task-manager/internal/templates"
)

// BundleContentKind identifies a supported shark-data content family.
type BundleContentKind string

const (
	// BundleContentKindSkill resolves files under skills/.
	BundleContentKindSkill BundleContentKind = "skill"
	// BundleContentKindAgent resolves files under agents/.
	BundleContentKindAgent BundleContentKind = "agent"
)

// BundleContentSource identifies which content layer supplied an entry.
type BundleContentSource string

const (
	BundleContentSourceOverride BundleContentSource = "override"
	BundleContentSourceDisk     BundleContentSource = "disk"
	BundleContentSourceEmbedded BundleContentSource = "embedded"
)

// BundleContentGetOptions controls content retrieval behavior.
type BundleContentGetOptions struct {
	Raw bool
}

// BundleContent is the retrieved bundle file plus source metadata.
type BundleContent struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Source   string `json:"source"`
	Content  string `json:"content"`
	Resolved bool   `json:"resolved"`
	Raw      bool   `json:"raw"`
}

// BundleContentEntry is one logical list item after source precedence is
// applied.
type BundleContentEntry struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

// BundleContentNotFoundError reports that no bundle layer contained the
// requested content.
type BundleContentNotFoundError struct {
	Kind BundleContentKind
	Name string
	Path string
}

func (e *BundleContentNotFoundError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s not found: %s", e.Kind, e.Name)
	}
	return fmt.Sprintf("%s not found: %s/%s", e.Kind, e.Name, e.Path)
}

// BundleContentService resolves shark-data bundle files from disk overrides,
// disk defaults, then embedded defaults.
type BundleContentService struct {
	dataRoot string
}

// NewBundleContentService constructs a content resolver for projectRoot using
// .sharkconfig.json's shark_data_path field.
func NewBundleContentService(projectRoot string) (*BundleContentService, error) {
	if strings.TrimSpace(projectRoot) == "" {
		projectRoot = "."
	}

	configBytes, err := os.ReadFile(filepath.Join(projectRoot, ".sharkconfig.json"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read .sharkconfig.json: %w", err)
	}

	dataRoot, err := config.ResolveSharkDataRoot(projectRoot, configBytes)
	if err != nil {
		return nil, fmt.Errorf("resolve shark data root: %w", err)
	}

	return &BundleContentService{dataRoot: dataRoot}, nil
}

// Get retrieves one bundle file. Non-raw reads resolve include/augment
// directives and strip top-level Markdown frontmatter.
func (s *BundleContentService) Get(ctx context.Context, kind BundleContentKind, name, relPath string, opts BundleContentGetOptions) (*BundleContent, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bundleKind, err := normalizeBundleContentKind(kind)
	if err != nil {
		return nil, err
	}
	if err := validateBundleContentName(name); err != nil {
		return nil, err
	}

	displayPath, err := defaultBundleContentPath(bundleKind, name, relPath)
	if err != nil {
		return nil, err
	}
	if err := validateBundleContentPath(displayPath); err != nil {
		return nil, err
	}

	candidates := s.lookupCandidates(bundleKind, name, displayPath)
	for _, candidate := range candidates {
		data, readErr := s.readCandidate(candidate)
		if errors.Is(readErr, fs.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return nil, readErr
		}

		content := string(data)
		resolved := false
		if !opts.Raw {
			resolver := templates.NewIncludeResolverWithEmbed(s.dataRoot)
			content, err = resolver.Resolve(content)
			if err != nil {
				return nil, fmt.Errorf("resolve includes for %s %s: %w", bundleKind, name, err)
			}
			content = stripBundleFrontmatter(content)
			resolved = true
		}

		return &BundleContent{
			Kind:     string(bundleKind),
			Name:     name,
			Path:     displayPath,
			Source:   string(candidate.source),
			Content:  content,
			Resolved: resolved,
			Raw:      opts.Raw,
		}, nil
	}

	return nil, &BundleContentNotFoundError{Kind: bundleKind, Name: name, Path: displayPath}
}

// List returns bundle content names sorted by name, with higher-precedence
// sources replacing lower-precedence entries of the same name.
func (s *BundleContentService) List(ctx context.Context, kind BundleContentKind) ([]BundleContentEntry, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bundleKind, err := normalizeBundleContentKind(kind)
	if err != nil {
		return nil, err
	}

	entries := map[string]BundleContentEntry{}
	for _, name := range s.embeddedNames(bundleKind) {
		entries[name] = BundleContentEntry{Name: name, Source: string(BundleContentSourceEmbedded)}
	}
	for _, name := range s.diskNames(bundleKind, false) {
		entries[name] = BundleContentEntry{Name: name, Source: string(BundleContentSourceDisk)}
	}
	for _, name := range s.diskNames(bundleKind, true) {
		entries[name] = BundleContentEntry{Name: name, Source: string(BundleContentSourceOverride)}
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]BundleContentEntry, 0, len(names))
	for _, name := range names {
		result = append(result, entries[name])
	}
	return result, nil
}

type bundleLookupCandidate struct {
	source BundleContentSource
	rel    string
}

func normalizeBundleContentKind(kind BundleContentKind) (BundleContentKind, error) {
	switch kind {
	case BundleContentKindSkill, "skills":
		return BundleContentKindSkill, nil
	case BundleContentKindAgent, "agents":
		return BundleContentKindAgent, nil
	default:
		return "", fmt.Errorf("unsupported bundle content kind: %q", kind)
	}
}

func validateBundleContentName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("bundle content name is required")
	}
	if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) || name == "." || name == ".." || strings.Contains(name, "..") {
		return fmt.Errorf("bundle content name must be relative and stay within its content kind: %q", name)
	}
	return nil
}

func defaultBundleContentPath(kind BundleContentKind, name, relPath string) (string, error) {
	if strings.TrimSpace(relPath) != "" {
		return filepath.ToSlash(relPath), nil
	}
	switch kind {
	case BundleContentKindSkill:
		return "SKILL.md", nil
	case BundleContentKindAgent:
		return name + ".md", nil
	default:
		return "", fmt.Errorf("unsupported bundle content kind: %q", kind)
	}
}

func validateBundleContentPath(relPath string) error {
	if strings.TrimSpace(relPath) == "" {
		return fmt.Errorf("bundle content path is required")
	}
	if filepath.IsAbs(relPath) || strings.HasPrefix(relPath, "/") {
		return fmt.Errorf("bundle content path must be relative and stay within its content root: %q", relPath)
	}

	cleaned := path.Clean(strings.ReplaceAll(relPath, "\\", "/"))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("bundle content path must be relative and stay within its content root: %q", relPath)
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return fmt.Errorf("bundle content path must be relative and stay within its content root: %q", relPath)
		}
	}
	return nil
}

func (s *BundleContentService) lookupCandidates(kind BundleContentKind, name, displayPath string) []bundleLookupCandidate {
	switch kind {
	case BundleContentKindSkill:
		itemPath := path.Join("skills", name, filepath.ToSlash(displayPath))
		return []bundleLookupCandidate{
			{source: BundleContentSourceOverride, rel: path.Join("overrides", itemPath)},
			{source: BundleContentSourceDisk, rel: itemPath},
			{source: BundleContentSourceEmbedded, rel: itemPath},
		}
	case BundleContentKindAgent:
		candidates := []bundleLookupCandidate{
			{source: BundleContentSourceOverride, rel: path.Join("overrides", "agents", name, filepath.ToSlash(displayPath))},
			{source: BundleContentSourceOverride, rel: path.Join("overrides", "agents", filepath.ToSlash(displayPath))},
			{source: BundleContentSourceDisk, rel: path.Join("agents", name, filepath.ToSlash(displayPath))},
			{source: BundleContentSourceDisk, rel: path.Join("agents", filepath.ToSlash(displayPath))},
			{source: BundleContentSourceEmbedded, rel: path.Join("agents", name, filepath.ToSlash(displayPath))},
			{source: BundleContentSourceEmbedded, rel: path.Join("agents", filepath.ToSlash(displayPath))},
		}
		return dedupeLookupCandidates(candidates)
	default:
		return nil
	}
}

func dedupeLookupCandidates(candidates []bundleLookupCandidate) []bundleLookupCandidate {
	seen := map[string]bool{}
	result := make([]bundleLookupCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := string(candidate.source) + ":" + candidate.rel
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	return result
}

func (s *BundleContentService) readCandidate(candidate bundleLookupCandidate) ([]byte, error) {
	switch candidate.source {
	case BundleContentSourceOverride, BundleContentSourceDisk:
		data, err := os.ReadFile(filepath.Join(s.dataRoot, filepath.FromSlash(candidate.rel)))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fs.ErrNotExist
			}
			return nil, fmt.Errorf("read bundle content %s: %w", candidate.rel, err)
		}
		return data, nil
	case BundleContentSourceEmbedded:
		data, err := sharkdata.ReadEmbedded(candidate.rel)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fs.ErrNotExist
			}
			return nil, fmt.Errorf("read embedded bundle content %s: %w", candidate.rel, err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported bundle content source: %q", candidate.source)
	}
}

func (s *BundleContentService) embeddedNames(kind BundleContentKind) []string {
	fsys, root := sharkdata.EmbeddedFS()
	switch kind {
	case BundleContentKindSkill:
		return embeddedDirectoryNames(fsys, path.Join(root, "skills"))
	case BundleContentKindAgent:
		return embeddedMarkdownNames(fsys, path.Join(root, "agents"))
	default:
		return nil
	}
}

func embeddedDirectoryNames(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipBundleListName(name) {
			continue
		}
		if entry.IsDir() {
			names = append(names, name)
		}
	}
	return names
}

func embeddedMarkdownNames(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipBundleListName(name) {
			continue
		}
		if entry.IsDir() {
			names = append(names, name)
			continue
		}
		if strings.HasSuffix(name, ".md") {
			names = append(names, strings.TrimSuffix(name, ".md"))
		}
	}
	return names
}

func (s *BundleContentService) diskNames(kind BundleContentKind, override bool) []string {
	dir := filepath.Join(s.dataRoot, dirForBundleKind(kind))
	if override {
		dir = filepath.Join(s.dataRoot, "overrides", dirForBundleKind(kind))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if shouldSkipBundleListName(name) {
			continue
		}
		switch kind {
		case BundleContentKindSkill:
			if entry.IsDir() {
				names = append(names, name)
			}
		case BundleContentKindAgent:
			if entry.IsDir() {
				names = append(names, name)
				continue
			}
			if strings.HasSuffix(name, ".md") {
				names = append(names, strings.TrimSuffix(name, ".md"))
			}
		}
	}
	return names
}

func shouldSkipBundleListName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func dirForBundleKind(kind BundleContentKind) string {
	switch kind {
	case BundleContentKindSkill:
		return "skills"
	case BundleContentKindAgent:
		return "agents"
	default:
		return ""
	}
}

func stripBundleFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return content
	}

	rest := content[strings.Index(content, "\n")+1:]
	lines := splitBundleLines(rest)
	for i, line := range lines {
		if line == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return content
}

func splitBundleLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}
