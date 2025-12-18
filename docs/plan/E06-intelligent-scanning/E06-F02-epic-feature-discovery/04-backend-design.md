# Backend Design: Epic & Feature Discovery Engine

**Feature**: E06-F02 Epic & Feature Discovery Engine
**Epic**: E06 Intelligent Documentation Scanning
**Status**: POC Implementation Specification
**Last Updated**: 2025-12-17

---

## Overview

This document provides detailed implementation specifications for the Epic & Feature Discovery Engine backend components. It describes concrete Go implementations, algorithms, and integration points with existing codebase.

### Design Constraints (POC)

- **Extend existing patterns**: Follow `internal/sync/` conventions
- **Minimal dependencies**: Use standard library + existing project dependencies
- **No new frameworks**: Work within Cobra CLI, SQLite, existing repository layer
- **Simple first**: Optimize for clarity over performance in POC

---

## Package Structure

```
internal/
├── discovery/                   # NEW package for epic/feature discovery
│   ├── orchestrator.go         # Main discovery coordinator
│   ├── index_parser.go         # Epic-index.md parser
│   ├── folder_scanner.go       # Folder structure scanner
│   ├── metadata_extractor.go  # Metadata extraction with fallbacks
│   ├── conflict_detector.go   # Conflict detection logic
│   ├── conflict_resolver.go   # Conflict resolution strategies
│   ├── types.go               # Shared types and constants
│   ├── patterns.go            # Pattern matching utilities
│   └── orchestrator_test.go   # Integration tests
├── repository/                 # EXISTING (extend as needed)
│   ├── epic_repository.go     # Add UpsertTx method
│   └── feature_repository.go  # Add UpsertTx method
└── cli/commands/              # EXISTING (extend)
    └── scan.go                # NEW command: shark scan
```

---

## Core Components

### 1. Discovery Orchestrator

**File**: `internal/discovery/orchestrator.go`

**Purpose**: Coordinate epic/feature discovery from all sources and manage database import

#### Type Definitions

```go
package discovery

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// DiscoveryOrchestrator coordinates epic/feature discovery
type DiscoveryOrchestrator struct {
    db              *sql.DB
    epicRepo        *repository.EpicRepository
    featureRepo     *repository.FeatureRepository
    indexParser     *IndexParser
    folderScanner   *FolderScanner
    metadataExtractor *MetadataExtractor
    conflictDetector *ConflictDetector
    conflictResolver *ConflictResolver
}

// NewDiscoveryOrchestrator creates a new orchestrator instance
func NewDiscoveryOrchestrator(db *sql.DB) *DiscoveryOrchestrator {
    repoDb := repository.NewDB(db)
    return &DiscoveryOrchestrator{
        db:                db,
        epicRepo:          repository.NewEpicRepository(repoDb),
        featureRepo:       repository.NewFeatureRepository(repoDb),
        indexParser:       NewIndexParser(),
        folderScanner:     NewFolderScanner(),
        metadataExtractor: NewMetadataExtractor(),
        conflictDetector:  NewConflictDetector(),
        conflictResolver:  NewConflictResolver(),
    }
}

// DiscoveryOptions configures discovery behavior
type DiscoveryOptions struct {
    DocsRoot        string           // Root documentation directory (e.g., "docs/plan")
    IndexPath       string           // Path to epic-index.md (optional, default: {DocsRoot}/epic-index.md)
    Strategy        ConflictStrategy // Conflict resolution strategy
    DryRun          bool             // Preview mode - no database changes
    ValidationLevel ValidationLevel  // Validation strictness
    Patterns        *PatternConfig   // Pattern overrides (optional, uses .sharkconfig.json if nil)
}

// DiscoveryReport contains results of discovery operation
type DiscoveryReport struct {
    FoldersScanned       int        `json:"folders_scanned"`
    FilesAnalyzed        int        `json:"files_analyzed"`
    EpicsDiscovered      int        `json:"epics_discovered"`
    EpicsFromIndex       int        `json:"epics_from_index"`
    EpicsFromFolders     int        `json:"epics_from_folders"`
    FeaturesDiscovered   int        `json:"features_discovered"`
    FeaturesFromIndex    int        `json:"features_from_index"`
    FeaturesFromFolders  int        `json:"features_from_folders"`
    RelatedDocsCataloged int        `json:"related_docs_cataloged"`
    ConflictsDetected    int        `json:"conflicts_detected"`
    Conflicts            []Conflict `json:"conflicts"`
    Warnings             []string   `json:"warnings"`
    Errors               []string   `json:"errors"`
}
```

#### Main Discovery Algorithm

```go
// Discover executes epic/feature discovery workflow
func (o *DiscoveryOrchestrator) Discover(ctx context.Context, opts DiscoveryOptions) (*DiscoveryReport, error) {
    report := &DiscoveryReport{
        Warnings:  []string{},
        Errors:    []string{},
        Conflicts: []Conflict{},
    }

    // Step 1: Parse epic-index.md (if exists)
    var indexEpics []IndexEpic
    var indexFeatures []IndexFeature
    if opts.IndexPath != "" {
        var err error
        indexEpics, indexFeatures, err = o.indexParser.Parse(opts.IndexPath)
        if err != nil {
            // Non-fatal: index parsing is optional
            report.Warnings = append(report.Warnings,
                fmt.Sprintf("Failed to parse index: %v (will use folder-only discovery)", err))
        } else {
            report.EpicsFromIndex = len(indexEpics)
            report.FeaturesFromIndex = len(indexFeatures)
        }
    }

    // Step 2: Scan folder structure
    folderEpics, folderFeatures, scanStats, err := o.folderScanner.Scan(opts.DocsRoot, opts.Patterns)
    if err != nil {
        return nil, fmt.Errorf("failed to scan folders: %w", err)
    }
    report.FoldersScanned = scanStats.FoldersScanned
    report.FilesAnalyzed = scanStats.FilesAnalyzed
    report.EpicsFromFolders = len(folderEpics)
    report.FeaturesFromFolders = len(folderFeatures)

    // Step 3: Extract metadata from epic.md, prd.md files
    enrichedIndexEpics, err := o.metadataExtractor.ExtractEpicMetadata(indexEpics, opts.DocsRoot)
    if err != nil {
        report.Warnings = append(report.Warnings, fmt.Sprintf("Metadata extraction warnings: %v", err))
    }
    enrichedFolderEpics, err := o.metadataExtractor.ExtractEpicMetadata(folderEpics, opts.DocsRoot)
    if err != nil {
        report.Warnings = append(report.Warnings, fmt.Sprintf("Metadata extraction warnings: %v", err))
    }

    enrichedIndexFeatures, err := o.metadataExtractor.ExtractFeatureMetadata(indexFeatures, opts.DocsRoot)
    if err != nil {
        report.Warnings = append(report.Warnings, fmt.Sprintf("Feature metadata warnings: %v", err))
    }
    enrichedFolderFeatures, err := o.metadataExtractor.ExtractFeatureMetadata(folderFeatures, opts.DocsRoot)
    if err != nil {
        report.Warnings = append(report.Warnings, fmt.Sprintf("Feature metadata warnings: %v", err))
    }

    // Step 4: Detect conflicts between index and folders
    conflicts := o.conflictDetector.Detect(enrichedIndexEpics, enrichedFolderEpics,
                                           enrichedIndexFeatures, enrichedFolderFeatures)
    report.Conflicts = conflicts
    report.ConflictsDetected = len(conflicts)

    // Step 5: Resolve conflicts using configured strategy
    resolvedEpics, resolvedFeatures, warnings, err := o.conflictResolver.Resolve(
        enrichedIndexEpics, enrichedFolderEpics,
        enrichedIndexFeatures, enrichedFolderFeatures,
        conflicts, opts.Strategy)
    if err != nil {
        return nil, fmt.Errorf("conflict resolution failed: %w", err)
    }
    report.Warnings = append(report.Warnings, warnings...)
    report.EpicsDiscovered = len(resolvedEpics)
    report.FeaturesDiscovered = len(resolvedFeatures)

    // Step 6: Catalog related documents
    relatedDocsCount := 0
    for i := range resolvedFeatures {
        relatedDocsCount += len(resolvedFeatures[i].RelatedDocs)
    }
    report.RelatedDocsCataloged = relatedDocsCount

    // Step 7: Execute database import (unless dry-run)
    if !opts.DryRun {
        if err := o.executeImport(ctx, resolvedEpics, resolvedFeatures); err != nil {
            return nil, fmt.Errorf("database import failed: %w", err)
        }
    }

    return report, nil
}
```

#### Database Import (Transactional)

```go
// executeImport performs transactional database import
func (o *DiscoveryOrchestrator) executeImport(
    ctx context.Context,
    epics []ResolvedEpic,
    features []ResolvedFeature,
) error {
    // Begin transaction
    tx, err := o.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback() // Auto-rollback if not committed

    // Import epics
    for _, epic := range epics {
        epicModel := &models.Epic{
            Key:         epic.Key,
            Title:       epic.Title,
            Description: epic.Description,
            Status:      "draft", // POC: auto-set to draft
            FilePath:    epic.FilePath,
        }
        if err := o.epicRepo.UpsertTx(ctx, tx, epicModel); err != nil {
            return fmt.Errorf("failed to upsert epic %s: %w", epic.Key, err)
        }
    }

    // Import features
    for _, feature := range features {
        relatedDocsJSON, _ := json.Marshal(feature.RelatedDocs)
        featureModel := &models.Feature{
            Key:         feature.Key,
            EpicKey:     feature.EpicKey,
            Title:       feature.Title,
            Description: feature.Description,
            Status:      "planning", // POC: auto-set to planning
            FilePath:    feature.FilePath,
            RelatedDocs: string(relatedDocsJSON),
        }
        if err := o.featureRepo.UpsertTx(ctx, tx, featureModel); err != nil {
            return fmt.Errorf("failed to upsert feature %s: %w", feature.Key, err)
        }
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }

    return nil
}
```

---

### 2. Index Parser

**File**: `internal/discovery/index_parser.go`

**Purpose**: Parse epic-index.md to extract explicit epic/feature structure

#### Type Definitions

```go
// IndexParser parses epic-index.md markdown file
type IndexParser struct {
    // No state needed for POC
}

func NewIndexParser() *IndexParser {
    return &IndexParser{}
}

// IndexEpic represents an epic discovered from index
type IndexEpic struct {
    Key   string
    Title string
    Path  string // Relative path from link
}

// IndexFeature represents a feature discovered from index
type IndexFeature struct {
    Key     string
    EpicKey string
    Title   string
    Path    string // Relative path from link
}
```

#### Parsing Algorithm

```go
// Parse reads epic-index.md and extracts epic/feature links
func (p *IndexParser) Parse(indexPath string) ([]IndexEpic, []IndexFeature, error) {
    // Read file
    content, err := os.ReadFile(indexPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read index: %w", err)
    }

    // Parse markdown links
    // Pattern: [Link Text](./path/to/folder/)
    linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
    matches := linkPattern.FindAllStringSubmatch(string(content), -1)

    epics := []IndexEpic{}
    features := []IndexFeature{}

    for _, match := range matches {
        linkText := match[1]
        linkPath := match[2]

        // Clean path (remove leading ./ and trailing /)
        cleanPath := strings.TrimPrefix(linkPath, "./")
        cleanPath = strings.TrimSuffix(cleanPath, "/")

        // Count path segments to determine epic vs feature
        segments := strings.Split(cleanPath, "/")

        if len(segments) == 1 {
            // Epic link: ./E04-epic-slug/
            epic, err := p.parseEpicLink(linkText, cleanPath)
            if err != nil {
                // Log warning, skip invalid link
                continue
            }
            epics = append(epics, epic)
        } else if len(segments) == 2 {
            // Feature link: ./E04-epic-slug/E04-F01-feature-slug/
            feature, err := p.parseFeatureLink(linkText, cleanPath)
            if err != nil {
                // Log warning, skip invalid link
                continue
            }
            features = append(features, feature)
        }
        // Ignore deeper paths (task links, document links)
    }

    return epics, features, nil
}

// parseEpicLink extracts epic metadata from link
func (p *IndexParser) parseEpicLink(linkText, path string) (IndexEpic, error) {
    // Extract epic key from path using patterns
    // Try standard pattern: E##-slug
    epicPattern := regexp.MustCompile(`^(E\d{2})-([a-z0-9-]+)$`)
    matches := epicPattern.FindStringSubmatch(path)
    if len(matches) == 3 {
        return IndexEpic{
            Key:   matches[1],  // E04
            Title: linkText,
            Path:  path,
        }, nil
    }

    // Try special type pattern: tech-debt, bugs, change-cards
    specialPattern := regexp.MustCompile(`^(tech-debt|bugs|change-cards)$`)
    matches = specialPattern.FindStringSubmatch(path)
    if len(matches) == 2 {
        return IndexEpic{
            Key:   matches[1],  // tech-debt
            Title: linkText,
            Path:  path,
        }, nil
    }

    return IndexEpic{}, fmt.Errorf("path does not match epic patterns: %s", path)
}

// parseFeatureLink extracts feature metadata from link
func (p *IndexParser) parseFeatureLink(linkText, path string) (IndexFeature, error) {
    // Path format: E04-epic-slug/E04-F01-feature-slug
    segments := strings.Split(path, "/")
    if len(segments) != 2 {
        return IndexFeature{}, fmt.Errorf("invalid feature path: %s", path)
    }

    epicFolder := segments[0]
    featureFolder := segments[1]

    // Extract epic key from epic folder
    epicPattern := regexp.MustCompile(`^(E\d{2})-`)
    epicMatches := epicPattern.FindStringSubmatch(epicFolder)
    if len(epicMatches) < 2 {
        return IndexFeature{}, fmt.Errorf("cannot extract epic key from: %s", epicFolder)
    }
    epicKey := epicMatches[1]

    // Extract feature key from feature folder
    // Pattern: E04-F01-feature-slug
    featurePattern := regexp.MustCompile(`^(E\d{2})-(F\d{2})-`)
    featureMatches := featurePattern.FindStringSubmatch(featureFolder)
    if len(featureMatches) < 3 {
        return IndexFeature{}, fmt.Errorf("cannot extract feature key from: %s", featureFolder)
    }
    featureKey := featureMatches[1] + "-" + featureMatches[2] // E04-F01

    return IndexFeature{
        Key:     featureKey,
        EpicKey: epicKey,
        Title:   linkText,
        Path:    path,
    }, nil
}
```

**Key Implementation Notes**:
- Hardcoded patterns for POC (E##-slug, tech-debt, bugs, change-cards)
- Post-POC: Load patterns from .sharkconfig.json
- Skip invalid links with warnings (don't fail entire parse)
- Simple path segment counting to distinguish epics from features

---

### 3. Folder Scanner

**File**: `internal/discovery/folder_scanner.go`

**Purpose**: Discover epics/features by scanning directory structure

#### Type Definitions

```go
// FolderScanner discovers epics/features from directory structure
type FolderScanner struct {
    patterns *PatternConfig
}

func NewFolderScanner() *FolderScanner {
    return &FolderScanner{
        patterns: DefaultPatternConfig(),
    }
}

// FolderEpic represents an epic discovered from folder
type FolderEpic struct {
    Key        string
    Slug       string
    Path       string
    EpicMdPath *string
}

// FolderFeature represents a feature discovered from folder
type FolderFeature struct {
    Key         string
    EpicKey     string
    Slug        string
    Path        string
    PrdPath     *string
    RelatedDocs []string
}

// ScanStats contains statistics from folder scan
type ScanStats struct {
    FoldersScanned int
    FilesAnalyzed  int
}
```

#### Scanning Algorithm

```go
// Scan walks directory tree and discovers epics/features
func (s *FolderScanner) Scan(docsRoot string, patterns *PatternConfig) (
    []FolderEpic, []FolderFeature, ScanStats, error) {

    if patterns != nil {
        s.patterns = patterns // Override default patterns
    }

    stats := ScanStats{}
    epics := []FolderEpic{}
    features := []FolderFeature{}

    // Walk docs root directory
    err := filepath.Walk(docsRoot, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Skip hidden directories
        if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
            return filepath.SkipDir
        }

        // Only process directories
        if !info.IsDir() {
            return nil
        }

        stats.FoldersScanned++

        // Try to match as epic folder
        if epic, matched := s.matchEpicFolder(path, info.Name()); matched {
            // Check if epic.md exists
            epicMdPath := filepath.Join(path, "epic.md")
            if _, err := os.Stat(epicMdPath); err == nil {
                epic.EpicMdPath = &epicMdPath
                stats.FilesAnalyzed++
            }
            epics = append(epics, epic)
            return nil // Don't recurse into epic folder yet
        }

        // Try to match as feature folder (within epic)
        // Check if parent is an epic folder
        parentDir := filepath.Dir(path)
        parentName := filepath.Base(parentDir)
        if _, isEpic := s.matchEpicFolder(parentDir, parentName); isEpic {
            if feature, matched := s.matchFeatureFolder(path, info.Name(), parentName); matched {
                // Find PRD file
                prdPath := s.findPrdFile(path)
                if prdPath != nil {
                    feature.PrdPath = prdPath
                    stats.FilesAnalyzed++
                }

                // Catalog related documents
                relatedDocs := s.catalogRelatedDocs(path)
                feature.RelatedDocs = relatedDocs
                stats.FilesAnalyzed += len(relatedDocs)

                features = append(features, feature)
            }
        }

        return nil
    })

    if err != nil {
        return nil, nil, stats, fmt.Errorf("walk failed: %w", err)
    }

    return epics, features, stats, nil
}
```

#### Pattern Matching

```go
// matchEpicFolder tries to match folder name against epic patterns
func (s *FolderScanner) matchEpicFolder(fullPath, folderName string) (FolderEpic, bool) {
    for _, pattern := range s.patterns.Epic.Folder {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(folderName)
        if matches == nil {
            continue
        }

        // Extract named groups
        names := re.SubexpNames()
        groups := make(map[string]string)
        for i, name := range names {
            if i > 0 && name != "" && i < len(matches) {
                groups[name] = matches[i]
            }
        }

        // Build epic from captured groups
        epic := FolderEpic{Path: fullPath}

        if epicId, ok := groups["epic_id"]; ok {
            epic.Key = epicId
        }
        if epicSlug, ok := groups["epic_slug"]; ok {
            epic.Slug = epicSlug
        }

        // Validate required fields
        if epic.Key == "" {
            continue // Pattern didn't capture required field
        }

        return epic, true
    }

    return FolderEpic{}, false
}

// matchFeatureFolder tries to match folder name against feature patterns
func (s *FolderScanner) matchFeatureFolder(fullPath, folderName, epicFolderName string) (FolderFeature, bool) {
    for _, pattern := range s.patterns.Feature.Folder {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(folderName)
        if matches == nil {
            continue
        }

        // Extract named groups
        names := re.SubexpNames()
        groups := make(map[string]string)
        for i, name := range names {
            if i > 0 && name != "" && i < len(matches) {
                groups[name] = matches[i]
            }
        }

        // Build feature from captured groups
        feature := FolderFeature{Path: fullPath}

        if featureId, ok := groups["feature_id"]; ok {
            // Build full feature key: E04-F01
            if epicId, ok := groups["epic_id"]; ok {
                feature.Key = epicId + "-" + featureId
                feature.EpicKey = epicId
            } else {
                // Infer epic from parent folder
                if parentEpic, matched := s.matchEpicFolder("", epicFolderName); matched {
                    feature.Key = parentEpic.Key + "-" + featureId
                    feature.EpicKey = parentEpic.Key
                }
            }
        }

        if featureSlug, ok := groups["feature_slug"]; ok {
            feature.Slug = featureSlug
        }

        // Validate required fields
        if feature.Key == "" || feature.EpicKey == "" {
            continue
        }

        return feature, true
    }

    return FolderFeature{}, false
}
```

#### Related Document Cataloging

```go
// findPrdFile searches for PRD file using feature file patterns
func (s *FolderScanner) findPrdFile(featurePath string) *string {
    entries, err := os.ReadDir(featurePath)
    if err != nil {
        return nil
    }

    for _, pattern := range s.patterns.Feature.File {
        re := regexp.MustCompile(pattern)
        for _, entry := range entries {
            if entry.IsDir() {
                continue
            }
            if re.MatchString(entry.Name()) {
                prdPath := filepath.Join(featurePath, entry.Name())
                return &prdPath
            }
        }
    }

    return nil
}

// catalogRelatedDocs finds all related documents in feature folder
func (s *FolderScanner) catalogRelatedDocs(featurePath string) []string {
    relatedDocs := []string{}

    entries, err := os.ReadDir(featurePath)
    if err != nil {
        return relatedDocs
    }

    for _, entry := range entries {
        // Skip directories (including tasks/, prps/)
        if entry.IsDir() {
            continue
        }

        // Only include .md files
        if !strings.HasSuffix(entry.Name(), ".md") {
            continue
        }

        // Exclude PRD file
        if entry.Name() == "prd.md" || strings.HasPrefix(entry.Name(), "PRD_") {
            continue
        }

        // Include numbered design docs, named docs, etc.
        relatedDocs = append(relatedDocs, filepath.Join(featurePath, entry.Name()))
    }

    return relatedDocs
}
```

---

### 4. Metadata Extractor

**File**: `internal/discovery/metadata_extractor.go`

**Purpose**: Extract titles/descriptions from epic.md, prd.md with fallback priority

#### Epic Metadata Extraction

```go
// MetadataExtractor enriches discovered epics/features with metadata
type MetadataExtractor struct {}

func NewMetadataExtractor() *MetadataExtractor {
    return &MetadataExtractor{}
}

// ExtractEpicMetadata enriches epic with title/description from multiple sources
func (e *MetadataExtractor) ExtractEpicMetadata(epics []FolderEpic, docsRoot string) ([]EnrichedEpic, error) {
    enriched := []EnrichedEpic{}

    for _, epic := range epics {
        enrichedEpic := EnrichedEpic{
            Key:  epic.Key,
            Path: epic.Path,
        }

        // Priority 1: Title from epic-index.md (if epic came from index, title already set)
        // (This is handled by caller merging index and folder data)

        // Priority 2: Title/description from epic.md frontmatter
        if epic.EpicMdPath != nil {
            title, desc, err := e.extractFrontmatter(*epic.EpicMdPath)
            if err == nil && title != "" {
                enrichedEpic.Title = title
                enrichedEpic.Description = desc
                enrichedEpic.FilePath = *epic.EpicMdPath
                enriched = append(enriched, enrichedEpic)
                continue
            }

            // Priority 3: Title from first H1 in epic.md
            title, err = e.extractFirstH1(*epic.EpicMdPath)
            if err == nil && title != "" {
                enrichedEpic.Title = title
                enrichedEpic.FilePath = *epic.EpicMdPath
                enriched = append(enriched, enrichedEpic)
                continue
            }
        }

        // Priority 4: Generate from folder name
        enrichedEpic.Title = e.generateTitleFromSlug(epic.Slug)
        enriched = append(enriched, enrichedEpic)
    }

    return enriched, nil
}

// extractFrontmatter parses YAML frontmatter from markdown file
func (e *MetadataExtractor) extractFrontmatter(filePath string) (string, *string, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return "", nil, err
    }

    // Parse frontmatter between --- delimiters
    lines := strings.Split(string(content), "\n")
    if len(lines) < 3 || lines[0] != "---" {
        return "", nil, fmt.Errorf("no frontmatter found")
    }

    // Find closing ---
    endIdx := -1
    for i := 1; i < len(lines); i++ {
        if lines[i] == "---" {
            endIdx = i
            break
        }
    }
    if endIdx == -1 {
        return "", nil, fmt.Errorf("frontmatter not closed")
    }

    // Parse YAML frontmatter (simple key: value parsing for POC)
    var title string
    var description *string

    for i := 1; i < endIdx; i++ {
        line := strings.TrimSpace(lines[i])
        if strings.HasPrefix(line, "title:") {
            title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
            title = strings.Trim(title, `"'`) // Remove quotes
        }
        if strings.HasPrefix(line, "description:") {
            desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
            desc = strings.Trim(desc, `"'`)
            description = &desc
        }
    }

    if title == "" {
        return "", nil, fmt.Errorf("no title in frontmatter")
    }

    return title, description, nil
}

// extractFirstH1 extracts first H1 heading from markdown file
func (e *MetadataExtractor) extractFirstH1(filePath string) (string, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return "", err
    }

    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "# ") {
            return strings.TrimPrefix(trimmed, "# "), nil
        }
    }

    return "", fmt.Errorf("no H1 heading found")
}

// generateTitleFromSlug converts slug to Title Case
func (e *MetadataExtractor) generateTitleFromSlug(slug string) string {
    // Replace hyphens with spaces
    title := strings.ReplaceAll(slug, "-", " ")

    // Title case each word
    words := strings.Fields(title)
    for i, word := range words {
        // Handle common abbreviations
        switch strings.ToLower(word) {
        case "cli":
            words[i] = "CLI"
        case "api":
            words[i] = "API"
        case "ui":
            words[i] = "UI"
        default:
            words[i] = strings.Title(strings.ToLower(word))
        }
    }

    // Prefix with "Auto:" for transparency
    return "Auto: " + strings.Join(words, " ")
}
```

---

### 5. Conflict Detector

**File**: `internal/discovery/conflict_detector.go`

**Purpose**: Detect conflicts between index and folder discoveries

#### Detection Algorithm

```go
// ConflictDetector identifies conflicts between index and folder discoveries
type ConflictDetector struct {}

func NewConflictDetector() *ConflictDetector {
    return &ConflictDetector{}
}

// Detect finds all conflicts between index and folder discoveries
func (d *ConflictDetector) Detect(
    indexEpics []EnrichedEpic,
    folderEpics []EnrichedEpic,
    indexFeatures []EnrichedFeature,
    folderFeatures []EnrichedFeature,
) []Conflict {
    conflicts := []Conflict{}

    // Build key sets for comparison
    indexEpicKeys := makeKeySet(indexEpics)
    folderEpicKeys := makeKeySet(folderEpics)
    indexFeatureKeys := makeKeySet(indexFeatures)
    folderFeatureKeys := makeKeySet(folderFeatures)

    // Detect epic conflicts
    // 1. In index but not in folders (broken reference)
    for key := range indexEpicKeys {
        if !folderEpicKeys[key] {
            conflicts = append(conflicts, Conflict{
                Type:       ConflictEpicIndexOnly,
                Key:        key,
                Path:       "", // TODO: Get path from indexEpics
                Resolution: "",
                Strategy:   "",
                Suggestion: fmt.Sprintf("Create folder for epic %s or remove from epic-index.md", key),
            })
        }
    }

    // 2. In folders but not in index (undocumented)
    for key := range folderEpicKeys {
        if !indexEpicKeys[key] {
            conflicts = append(conflicts, Conflict{
                Type:       ConflictEpicFolderOnly,
                Key:        key,
                Path:       "", // TODO: Get path from folderEpics
                Resolution: "",
                Strategy:   "",
                Suggestion: fmt.Sprintf("Add epic %s to epic-index.md or use merge/folder-precedence strategy", key),
            })
        }
    }

    // Detect feature conflicts (same logic)
    for key := range indexFeatureKeys {
        if !folderFeatureKeys[key] {
            conflicts = append(conflicts, Conflict{
                Type:       ConflictFeatureIndexOnly,
                Key:        key,
                Resolution: "",
                Strategy:   "",
                Suggestion: fmt.Sprintf("Create folder for feature %s or remove from epic-index.md", key),
            })
        }
    }

    for key := range folderFeatureKeys {
        if !indexFeatureKeys[key] {
            conflicts = append(conflicts, Conflict{
                Type:       ConflictFeatureFolderOnly,
                Key:        key,
                Resolution: "",
                Strategy:   "",
                Suggestion: fmt.Sprintf("Add feature %s to epic-index.md or use merge/folder-precedence strategy", key),
            })
        }
    }

    return conflicts
}

// makeKeySet builds a set of keys from epic/feature list
func makeKeySet(items interface{}) map[string]bool {
    keySet := make(map[string]bool)

    // Type assertion and key extraction (implementation depends on type)
    // For POC: use reflection or separate functions for epics vs features

    return keySet
}
```

---

### 6. Conflict Resolver

**File**: `internal/discovery/conflict_resolver.go`

**Purpose**: Apply conflict resolution strategy to determine final import list

#### Resolution Strategies

```go
// ConflictResolver applies resolution strategy to conflicts
type ConflictResolver struct {}

func NewConflictResolver() *ConflictResolver {
    return &ConflictResolver{}
}

// Resolve applies strategy and returns final epic/feature lists
func (r *ConflictResolver) Resolve(
    indexEpics []EnrichedEpic,
    folderEpics []EnrichedEpic,
    indexFeatures []EnrichedFeature,
    folderFeatures []EnrichedFeature,
    conflicts []Conflict,
    strategy ConflictStrategy,
) ([]ResolvedEpic, []ResolvedFeature, []string, error) {

    warnings := []string{}

    switch strategy {
    case ConflictStrategyIndexPrecedence:
        return r.resolveIndexPrecedence(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts)

    case ConflictStrategyFolderPrecedence:
        return r.resolveFolderPrecedence(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts)

    case ConflictStrategyMerge:
        return r.resolveMerge(indexEpics, folderEpics, indexFeatures, folderFeatures, conflicts)

    default:
        return nil, nil, warnings, fmt.Errorf("unknown strategy: %s", strategy)
    }
}

// resolveIndexPrecedence uses index as source of truth
func (r *ConflictResolver) resolveIndexPrecedence(
    indexEpics []EnrichedEpic,
    folderEpics []EnrichedEpic,
    indexFeatures []EnrichedFeature,
    folderFeatures []EnrichedFeature,
    conflicts []Conflict,
) ([]ResolvedEpic, []ResolvedFeature, []string, error) {

    warnings := []string{}

    // Only include epics from index
    epics := convertEnrichedToResolved(indexEpics)

    // Validate all index epics have matching folders
    for _, conflict := range conflicts {
        if conflict.Type == ConflictEpicIndexOnly {
            return nil, nil, warnings, fmt.Errorf("epic %s in index but folder missing (index-precedence requires folders)", conflict.Key)
        }
    }

    // Warn about folder-only epics (ignored)
    for _, conflict := range conflicts {
        if conflict.Type == ConflictEpicFolderOnly {
            warnings = append(warnings, fmt.Sprintf("Epic %s in folders but not in index (skipped)", conflict.Key))
        }
    }

    // Only include features from index
    features := convertEnrichedFeaturesToResolved(indexFeatures)

    return epics, features, warnings, nil
}

// resolveFolderPrecedence uses folder structure as source of truth
func (r *ConflictResolver) resolveFolderPrecedence(
    indexEpics []EnrichedEpic,
    folderEpics []EnrichedEpic,
    indexFeatures []EnrichedFeature,
    folderFeatures []EnrichedFeature,
    conflicts []Conflict,
) ([]ResolvedEpic, []ResolvedFeature, []string, error) {

    warnings := []string{}

    // Only include epics from folders
    epics := convertEnrichedToResolved(folderEpics)

    // Warn about index-only epics (ignored)
    for _, conflict := range conflicts {
        if conflict.Type == ConflictEpicIndexOnly {
            warnings = append(warnings, fmt.Sprintf("Epic %s in index but folder missing (skipped)", conflict.Key))
        }
    }

    // Only include features from folders
    features := convertEnrichedFeaturesToResolved(folderFeatures)

    return epics, features, warnings, nil
}

// resolveMerge merges both sources (index metadata wins on conflict)
func (r *ConflictResolver) resolveMerge(
    indexEpics []EnrichedEpic,
    folderEpics []EnrichedEpic,
    indexFeatures []EnrichedFeature,
    folderFeatures []EnrichedFeature,
    conflicts []Conflict,
) ([]ResolvedEpic, []ResolvedFeature, []string, error) {

    warnings := []string{}

    // Build merged epic list
    epicMap := make(map[string]ResolvedEpic)

    // Add folder epics first
    for _, epic := range folderEpics {
        epicMap[epic.Key] = ResolvedEpic{
            Key:         epic.Key,
            Title:       epic.Title,
            Description: epic.Description,
            FilePath:    epic.FilePath,
            Source:      "folder",
        }
    }

    // Overlay index epics (index metadata wins)
    for _, epic := range indexEpics {
        if existing, ok := epicMap[epic.Key]; ok {
            // Merge: use index title, keep folder file path
            existing.Title = epic.Title // Index wins
            if epic.FilePath != "" {
                existing.FilePath = epic.FilePath
            }
            existing.Source = "merged"
            epicMap[epic.Key] = existing
        } else {
            // Index-only epic (folder missing) - include with warning
            warnings = append(warnings, fmt.Sprintf("Epic %s in index but folder missing (included anyway)", epic.Key))
            epicMap[epic.Key] = ResolvedEpic{
                Key:         epic.Key,
                Title:       epic.Title,
                Description: epic.Description,
                Source:      "index-only",
            }
        }
    }

    // Convert map to slice
    epics := []ResolvedEpic{}
    for _, epic := range epicMap {
        epics = append(epics, epic)
    }

    // Similar merge logic for features
    features := []ResolvedFeature{} // TODO: Implement feature merge

    return epics, features, warnings, nil
}
```

---

## Repository Extensions

### Epic Repository: Add Upsert Transaction Method

**File**: `internal/repository/epic_repository.go` (extend existing)

```go
// UpsertTx inserts or updates epic within transaction
func (r *EpicRepository) UpsertTx(ctx context.Context, tx *sql.Tx, epic *models.Epic) error {
    query := `
        INSERT INTO epics (key, title, description, status, file_path, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        ON CONFLICT(key) DO UPDATE SET
            title = excluded.title,
            description = excluded.description,
            status = excluded.status,
            file_path = excluded.file_path,
            updated_at = CURRENT_TIMESTAMP
    `

    _, err := tx.ExecContext(ctx, query,
        epic.Key,
        epic.Title,
        epic.Description,
        epic.Status,
        epic.FilePath,
    )

    if err != nil {
        return fmt.Errorf("upsert failed: %w", err)
    }

    return nil
}
```

### Feature Repository: Add Upsert Transaction Method

**File**: `internal/repository/feature_repository.go` (extend existing)

```go
// UpsertTx inserts or updates feature within transaction
func (r *FeatureRepository) UpsertTx(ctx context.Context, tx *sql.Tx, feature *models.Feature) error {
    query := `
        INSERT INTO features (epic_key, key, title, description, status, file_path, related_docs, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        ON CONFLICT(key) DO UPDATE SET
            title = excluded.title,
            description = excluded.description,
            status = excluded.status,
            file_path = excluded.file_path,
            related_docs = excluded.related_docs,
            updated_at = CURRENT_TIMESTAMP
    `

    _, err := tx.ExecContext(ctx, query,
        feature.EpicKey,
        feature.Key,
        feature.Title,
        feature.Description,
        feature.Status,
        feature.FilePath,
        feature.RelatedDocs,
    )

    if err != nil {
        return fmt.Errorf("upsert failed: %w", err)
    }

    return nil
}
```

---

## CLI Integration

### New Command: shark scan

**File**: `internal/cli/commands/scan.go` (new file)

```go
package commands

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/jwwelbor/shark-task-manager/internal/discovery"
    "github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
    Use:   "scan",
    Short: "Discover and import epics/features from documentation",
    Long:  "Scans documentation directory to discover epics and features, imports into database",
    RunE:  runScan,
}

var (
    scanDocsRoot    string
    scanIndexPath   string
    scanStrategy    string
    scanDryRun      bool
    scanJSONOutput  bool
)

func init() {
    scanCmd.Flags().StringVar(&scanDocsRoot, "docs-root", "docs/plan", "Documentation root directory")
    scanCmd.Flags().StringVar(&scanIndexPath, "index", "", "Path to epic-index.md (default: {docs-root}/epic-index.md)")
    scanCmd.Flags().StringVar(&scanStrategy, "strategy", "index-precedence", "Conflict resolution strategy (index-precedence, folder-precedence, merge)")
    scanCmd.Flags().BoolVar(&scanDryRun, "dry-run", false, "Preview changes without modifying database")
    scanCmd.Flags().BoolVar(&scanJSONOutput, "json", false, "Output report as JSON")
}

func runScan(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Open database
    dbPath := "shark-tasks.db" // TODO: Get from config
    db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
    if err != nil {
        return fmt.Errorf("failed to open database: %w", err)
    }
    defer db.Close()

    // Create orchestrator
    orchestrator := discovery.NewDiscoveryOrchestrator(db)

    // Set index path default
    if scanIndexPath == "" {
        scanIndexPath = filepath.Join(scanDocsRoot, "epic-index.md")
    }

    // Run discovery
    opts := discovery.DiscoveryOptions{
        DocsRoot:  scanDocsRoot,
        IndexPath: scanIndexPath,
        Strategy:  discovery.ConflictStrategy(scanStrategy),
        DryRun:    scanDryRun,
    }

    report, err := orchestrator.Discover(ctx, opts)
    if err != nil {
        return fmt.Errorf("discovery failed: %w", err)
    }

    // Output report
    if scanJSONOutput {
        jsonBytes, _ := json.MarshalIndent(report, "", "  ")
        fmt.Println(string(jsonBytes))
    } else {
        printTextReport(report)
    }

    return nil
}

func printTextReport(report *discovery.DiscoveryReport) {
    fmt.Printf("Discovery Report\n")
    fmt.Printf("================\n\n")
    fmt.Printf("Folders scanned: %d\n", report.FoldersScanned)
    fmt.Printf("Files analyzed: %d\n\n", report.FilesAnalyzed)

    fmt.Printf("Epics discovered: %d (%d from index, %d from folders)\n",
        report.EpicsDiscovered, report.EpicsFromIndex, report.EpicsFromFolders)
    fmt.Printf("Features discovered: %d (%d from index, %d from folders)\n",
        report.FeaturesDiscovered, report.FeaturesFromIndex, report.FeaturesFromFolders)
    fmt.Printf("Related docs cataloged: %d\n\n", report.RelatedDocsCataloged)

    if len(report.Conflicts) > 0 {
        fmt.Printf("Conflicts detected: %d\n", len(report.Conflicts))
        for _, conflict := range report.Conflicts {
            fmt.Printf("  - %s: %s (%s)\n", conflict.Type, conflict.Key, conflict.Suggestion)
        }
        fmt.Println()
    }

    if len(report.Warnings) > 0 {
        fmt.Printf("Warnings:\n")
        for _, warning := range report.Warnings {
            fmt.Printf("  - %s\n", warning)
        }
        fmt.Println()
    }

    if len(report.Errors) > 0 {
        fmt.Printf("Errors:\n")
        for _, err := range report.Errors {
            fmt.Printf("  - %s\n", err)
        }
    }
}
```

---

## Testing Strategy

### Unit Tests

**Test Files**:
- `internal/discovery/index_parser_test.go`
- `internal/discovery/folder_scanner_test.go`
- `internal/discovery/metadata_extractor_test.go`
- `internal/discovery/conflict_detector_test.go`
- `internal/discovery/conflict_resolver_test.go`

**Key Test Cases**:
1. Index parsing with varied link formats
2. Pattern matching (epic/feature folder patterns)
3. Metadata extraction with frontmatter, H1, fallback
4. Conflict detection (index-only, folder-only)
5. Conflict resolution strategies

### Integration Tests

**Test File**: `internal/discovery/orchestrator_test.go`

**Test Scenarios**:
1. Full discovery with sample documentation structure
2. Dry-run mode (no database changes)
3. Transaction rollback on error
4. Index-precedence strategy
5. Folder-precedence strategy
6. Merge strategy

**Test Data Structure**:
```
testdata/
├── epic-index.md
├── E04-task-mgmt-cli-core/
│   ├── epic.md
│   └── E04-F07-initialization-sync/
│       ├── prd.md
│       └── 02-architecture.md
├── E05-advanced-querying/
│   └── epic.md
└── tech-debt/
    └── epic.md
```

---

## Performance Optimization (Post-POC)

### Identified Bottlenecks

1. **File I/O**: Reading epic.md, prd.md files sequentially
2. **Pattern Matching**: Repeated regex compilation
3. **Transaction Overhead**: Individual epic/feature inserts

### Optimization Strategies

**Phase 1 (Easy Wins)**:
- Compile regex patterns once at initialization
- Use prepared statements for database inserts
- Batch inserts (slice of epics/features in single statement)

**Phase 2 (Parallelization)**:
- Parallel file scanning (goroutines per epic folder)
- Parallel metadata extraction (goroutine pool)
- Buffered channels for result aggregation

**Phase 3 (Incremental Discovery)**:
- Track `last_discovery_time` in .sharkconfig.json
- Compare folder mtime against last discovery
- Skip unchanged folders (significant speedup for large projects)

---

## Error Handling Standards

### Error Categories & Handling

**Parse Errors** (non-fatal):
```go
if err := parseEpicMd(path); err != nil {
    warnings = append(warnings, fmt.Sprintf("Failed to parse %s: %v (using folder name as title)", path, err))
    // Continue with fallback
}
```

**Validation Errors** (strictness-dependent):
```go
if !isValidEpicKey(key) {
    if validationLevel == "strict" {
        return fmt.Errorf("invalid epic key format: %s", key)
    } else {
        warnings = append(warnings, fmt.Sprintf("Invalid epic key format: %s (continuing anyway)", key))
    }
}
```

**Database Errors** (always fatal):
```go
if err := tx.Commit(); err != nil {
    return fmt.Errorf("transaction commit failed: %w", err)
}
```

### Actionable Error Messages

All errors must include:
1. **What failed**: "Failed to parse epic-index.md"
2. **Where**: File path and line number (if applicable)
3. **Why**: "Missing closing --- for frontmatter"
4. **How to fix**: "Add --- on line 8 to close frontmatter"

---

## Security Considerations

### Path Traversal Protection

```go
func validatePath(path, docsRoot string) error {
    // Canonicalize paths
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }

    absRoot, err := filepath.Abs(docsRoot)
    if err != nil {
        return fmt.Errorf("invalid docs root: %w", err)
    }

    // Ensure path is within docs root
    if !strings.HasPrefix(absPath, absRoot) {
        return fmt.Errorf("path traversal detected: %s not within %s", path, docsRoot)
    }

    return nil
}
```

### SQL Injection Prevention

**Always use parameterized queries**:
```go
// ✅ SAFE
query := "INSERT INTO epics (key, title) VALUES (?, ?)"
tx.ExecContext(ctx, query, epic.Key, epic.Title)

// ❌ NEVER DO THIS
query := fmt.Sprintf("INSERT INTO epics (key, title) VALUES ('%s', '%s')", epic.Key, epic.Title)
```

---

## Deployment

### POC Deployment Steps

1. **Add discovery package**: Create `internal/discovery/` with all components
2. **Extend repositories**: Add `UpsertTx` methods to epic/feature repositories
3. **Add scan command**: Create `internal/cli/commands/scan.go`
4. **Register command**: Add `scanCmd` to root command
5. **Database migration**: Add `file_path`, `related_docs` columns if missing
6. **Test**: Run integration tests with sample data

### Database Migration

```sql
-- Run during shark init or first shark scan
ALTER TABLE epics ADD COLUMN file_path TEXT;
ALTER TABLE features ADD COLUMN related_docs TEXT;
```

---

## Future Enhancements

### Post-POC Roadmap

1. **Incremental Discovery** (high value):
   - Track last_discovery_time
   - Skip unchanged folders based on mtime
   - 10x speedup for large projects

2. **Interactive Conflict Resolution**:
   - Prompt user for each conflict
   - "Epic E04 found in folders but not in index. [A]dd to index, [I]gnore, [Q]uit?"

3. **Epic-Index.md Generation**:
   - Reverse operation: generate index from folder structure
   - `shark generate-index --output=docs/plan/epic-index.md`

4. **Pattern Validation CLI**:
   - Test patterns against sample paths
   - `shark test-pattern --epic-pattern='E\d{2}-.*' --path='E04-my-epic'`

5. **Multi-Root Support**:
   - Scan multiple documentation roots
   - Useful for projects with docs/ and archived-docs/

---

## References

- **Architecture**: [02-architecture.md](./02-architecture.md)
- **PRD**: [prd.md](./prd.md)
- **Epic**: [E06 Intelligent Scanning](../epic.md)
- **Existing Sync Engine**: `internal/sync/engine.go`
- **Go Standard Library**: filepath, os, regexp, database/sql
