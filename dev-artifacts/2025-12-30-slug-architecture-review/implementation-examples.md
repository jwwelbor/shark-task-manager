# Implementation Examples - Slug Architecture Refactor

**Date**: 2025-12-30
**Purpose**: Code samples for implementing proposed architecture

---

## 1. Database Migration Script

### File: `internal/db/migrations/add_slug_columns.go`

```go
package migrations

import (
    "database/sql"
    "fmt"
)

// MigrationAddSlugColumns adds slug columns to epics, features, and tasks tables
func MigrationAddSlugColumns(db *sql.DB) error {
    // Add slug column to epics
    if err := addSlugColumnToEpics(db); err != nil {
        return fmt.Errorf("failed to add slug column to epics: %w", err)
    }

    // Add slug column to features
    if err := addSlugColumnToFeatures(db); err != nil {
        return fmt.Errorf("failed to add slug column to features: %w", err)
    }

    // Add slug column to tasks
    if err := addSlugColumnToTasks(db); err != nil {
        return fmt.Errorf("failed to add slug column to tasks: %w", err)
    }

    // Backfill slugs from file_path
    if err := backfillSlugs(db); err != nil {
        return fmt.Errorf("failed to backfill slugs: %w", err)
    }

    return nil
}

func addSlugColumnToEpics(db *sql.DB) error {
    // Check if column already exists
    var columnExists int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM pragma_table_info('epics') WHERE name = 'slug'
    `).Scan(&columnExists)
    if err != nil {
        return fmt.Errorf("failed to check column existence: %w", err)
    }

    if columnExists > 0 {
        return nil // Already exists
    }

    // Add column
    if _, err := db.Exec(`ALTER TABLE epics ADD COLUMN slug TEXT;`); err != nil {
        return fmt.Errorf("failed to add column: %w", err)
    }

    // Create index
    if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_epics_slug ON epics(slug);`); err != nil {
        return fmt.Errorf("failed to create index: %w", err)
    }

    return nil
}

func addSlugColumnToFeatures(db *sql.DB) error {
    // Check if column already exists
    var columnExists int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM pragma_table_info('features') WHERE name = 'slug'
    `).Scan(&columnExists)
    if err != nil {
        return fmt.Errorf("failed to check column existence: %w", err)
    }

    if columnExists > 0 {
        return nil // Already exists
    }

    // Add column
    if _, err := db.Exec(`ALTER TABLE features ADD COLUMN slug TEXT;`); err != nil {
        return fmt.Errorf("failed to add column: %w", err)
    }

    // Create index
    if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_features_slug ON features(slug);`); err != nil {
        return fmt.Errorf("failed to create index: %w", err)
    }

    return nil
}

func addSlugColumnToTasks(db *sql.DB) error {
    // Similar to epics/features
    // ... (implementation omitted for brevity)
    return nil
}

func backfillSlugs(db *sql.DB) error {
    // Backfill epics
    _, err := db.Exec(`
        UPDATE epics
        SET slug = (
            SELECT CASE
                WHEN file_path IS NOT NULL AND file_path LIKE '%/' || key || '-%'
                THEN substr(
                    file_path,
                    instr(file_path, '/' || key || '-') + length('/' || key || '-'),
                    instr(
                        substr(file_path, instr(file_path, '/' || key || '-')),
                        '/'
                    ) - 1
                )
                ELSE NULL
            END
        )
        WHERE file_path IS NOT NULL AND slug IS NULL;
    `)
    if err != nil {
        return fmt.Errorf("failed to backfill epic slugs: %w", err)
    }

    // Backfill features
    _, err = db.Exec(`
        UPDATE features
        SET slug = (
            SELECT CASE
                WHEN file_path IS NOT NULL AND file_path LIKE '%/' || key || '-%'
                THEN substr(
                    file_path,
                    instr(file_path, '/' || key || '-') + length('/' || key || '-'),
                    instr(
                        substr(file_path, instr(file_path, '/' || key || '-')),
                        '/'
                    ) - 1
                )
                ELSE NULL
            END
        )
        WHERE file_path IS NOT NULL AND slug IS NULL;
    `)
    if err != nil {
        return fmt.Errorf("failed to backfill feature slugs: %w", err)
    }

    // Backfill tasks
    _, err = db.Exec(`
        UPDATE tasks
        SET slug = (
            SELECT CASE
                WHEN file_path IS NOT NULL AND file_path LIKE '%/' || key || '-%'
                THEN REPLACE(
                    substr(
                        file_path,
                        instr(file_path, '/' || key || '-') + length('/' || key || '-')
                    ),
                    '.md',
                    ''
                )
                ELSE NULL
            END
        )
        WHERE file_path IS NOT NULL AND slug IS NULL;
    `)
    if err != nil {
        return fmt.Errorf("failed to backfill task slugs: %w", err)
    }

    return nil
}
```

---

## 2. PathResolver Implementation

### File: `internal/pathresolver/resolver.go`

```go
package pathresolver

import (
    "context"
    "fmt"
    "path/filepath"

    "github.com/jwwelbor/shark-task-manager/internal/repository"
)

// PathResolver provides database-driven path resolution for all entities
type PathResolver struct {
    epicRepo    *repository.EpicRepository
    featureRepo *repository.FeatureRepository
    taskRepo    *repository.TaskRepository
    projectRoot string
}

// NewPathResolver creates a new PathResolver
func NewPathResolver(
    epicRepo *repository.EpicRepository,
    featureRepo *repository.FeatureRepository,
    taskRepo *repository.TaskRepository,
    projectRoot string,
) *PathResolver {
    return &PathResolver{
        epicRepo:    epicRepo,
        featureRepo: featureRepo,
        taskRepo:    taskRepo,
        projectRoot: projectRoot,
    }
}

// ResolveEpicPath resolves epic file path from database
func (pr *PathResolver) ResolveEpicPath(ctx context.Context, epicKey string) (string, error) {
    epic, err := pr.epicRepo.GetByKey(ctx, epicKey)
    if err != nil {
        return "", fmt.Errorf("failed to get epic %s: %w", epicKey, err)
    }

    // Precedence 1: Explicit file_path in database
    if epic.FilePath != nil && *epic.FilePath != "" {
        return filepath.Join(pr.projectRoot, *epic.FilePath), nil
    }

    // Precedence 2: Compute from database fields
    if epic.Slug == nil || *epic.Slug == "" {
        return "", fmt.Errorf("epic %s has no slug or file_path", epicKey)
    }

    path := pr.ComputeEpicPath(epic.Key, *epic.Slug, epic.CustomFolderPath)
    return filepath.Join(pr.projectRoot, path), nil
}

// ResolveFeaturePath resolves feature file path from database
func (pr *PathResolver) ResolveFeaturePath(ctx context.Context, featureKey string) (string, error) {
    feature, err := pr.featureRepo.GetByKey(ctx, featureKey)
    if err != nil {
        return "", fmt.Errorf("failed to get feature %s: %w", featureKey, err)
    }

    // Precedence 1: Explicit file_path in database
    if feature.FilePath != nil && *feature.FilePath != "" {
        return filepath.Join(pr.projectRoot, *feature.FilePath), nil
    }

    // Precedence 2: Compute from database fields
    if feature.Slug == nil || *feature.Slug == "" {
        return "", fmt.Errorf("feature %s has no slug or file_path", featureKey)
    }

    // Get epic for custom_folder_path inheritance
    epic, err := pr.epicRepo.GetByID(ctx, feature.EpicID)
    if err != nil {
        return "", fmt.Errorf("failed to get epic for feature %s: %w", featureKey, err)
    }

    path := pr.ComputeFeaturePath(
        epic.Key,
        feature.Key,
        *feature.Slug,
        feature.CustomFolderPath,
        epic.CustomFolderPath,
    )
    return filepath.Join(pr.projectRoot, path), nil
}

// ResolveTaskPath resolves task file path from database
func (pr *PathResolver) ResolveTaskPath(ctx context.Context, taskKey string) (string, error) {
    task, err := pr.taskRepo.GetByKey(ctx, taskKey)
    if err != nil {
        return "", fmt.Errorf("failed to get task %s: %w", taskKey, err)
    }

    // Precedence 1: Explicit file_path in database
    if task.FilePath != nil && *task.FilePath != "" {
        return filepath.Join(pr.projectRoot, *task.FilePath), nil
    }

    // Precedence 2: Compute from database fields
    if task.Slug == nil || *task.Slug == "" {
        return "", fmt.Errorf("task %s has no slug or file_path", taskKey)
    }

    // Get feature and epic for custom_folder_path inheritance
    feature, err := pr.featureRepo.GetByID(ctx, task.FeatureID)
    if err != nil {
        return "", fmt.Errorf("failed to get feature for task %s: %w", taskKey, err)
    }

    epic, err := pr.epicRepo.GetByID(ctx, feature.EpicID)
    if err != nil {
        return "", fmt.Errorf("failed to get epic for task %s: %w", taskKey, err)
    }

    path := pr.ComputeTaskPath(
        epic.Key,
        feature.Key,
        task.Key,
        *task.Slug,
        feature.CustomFolderPath,
        epic.CustomFolderPath,
    )
    return filepath.Join(pr.projectRoot, path), nil
}

// ComputeEpicPath computes expected path from key, slug, and custom path
func (pr *PathResolver) ComputeEpicPath(key, slug string, customPath *string) string {
    var baseDir string

    if customPath != nil && *customPath != "" {
        baseDir = *customPath
    } else {
        baseDir = "docs/plan"
    }

    folderName := key + "-" + slug
    return filepath.Join(baseDir, folderName, "epic.md")
}

// ComputeFeaturePath computes expected path from key, slug, and custom paths
func (pr *PathResolver) ComputeFeaturePath(
    epicKey, featureKey, slug string,
    featureCustomPath, epicCustomPath *string,
) string {
    var baseDir string

    // Precedence: feature custom path > epic custom path > default
    if featureCustomPath != nil && *featureCustomPath != "" {
        baseDir = *featureCustomPath
    } else if epicCustomPath != nil && *epicCustomPath != "" {
        epicFolderName := epicKey  // Would need epic slug here for full path
        baseDir = filepath.Join(*epicCustomPath, epicFolderName)
    } else {
        baseDir = filepath.Join("docs/plan", epicKey)
    }

    folderName := featureKey + "-" + slug
    return filepath.Join(baseDir, folderName, "feature.md")
}

// ComputeTaskPath computes expected path from key, slug, and custom paths
func (pr *PathResolver) ComputeTaskPath(
    epicKey, featureKey, taskKey, slug string,
    featureCustomPath, epicCustomPath *string,
) string {
    var baseDir string

    // Precedence: feature custom path > epic custom path > default
    if featureCustomPath != nil && *featureCustomPath != "" {
        featureFolderName := featureKey  // Would need feature slug here
        baseDir = filepath.Join(*featureCustomPath, featureFolderName, "tasks")
    } else if epicCustomPath != nil && *epicCustomPath != "" {
        epicFolderName := epicKey  // Would need epic slug here
        featureFolderName := featureKey  // Would need feature slug here
        baseDir = filepath.Join(*epicCustomPath, epicFolderName, featureFolderName, "tasks")
    } else {
        baseDir = filepath.Join("docs/plan", epicKey, featureKey, "tasks")
    }

    filename := taskKey + "-" + slug + ".md"
    return filepath.Join(baseDir, filename)
}
```

---

## 3. Updated Epic Repository (Flexible Key Lookup)

### File: `internal/repository/epic_repository.go` (additions)

```go
// GetByKey retrieves an epic by key, supporting both numeric (E05) and slugged (E05-slug) formats
func (r *EpicRepository) GetByKey(ctx context.Context, keyOrSlug string) (*models.Epic, error) {
    // Try exact key match first
    epic, err := r.getByExactKey(ctx, keyOrSlug)
    if err == nil {
        return epic, nil
    }

    // If not found and input contains hyphen, try extracting numeric key
    if strings.Contains(keyOrSlug, "-") {
        numericKey := extractNumericEpicKey(keyOrSlug)
        if numericKey != keyOrSlug {
            epic, err := r.getByExactKey(ctx, numericKey)
            if err == nil {
                return epic, nil
            }
        }
    }

    return nil, sql.ErrNoRows
}

// getByExactKey retrieves an epic by exact key match (internal helper)
func (r *EpicRepository) getByExactKey(ctx context.Context, key string) (*models.Epic, error) {
    query := `
        SELECT id, key, title, description, status, priority, business_value,
               file_path, custom_folder_path, slug, created_at, updated_at
        FROM epics
        WHERE key = ?
    `

    epic := &models.Epic{}
    err := r.db.QueryRowContext(ctx, query, key).Scan(
        &epic.ID,
        &epic.Key,
        &epic.Title,
        &epic.Description,
        &epic.Status,
        &epic.Priority,
        &epic.BusinessValue,
        &epic.FilePath,
        &epic.CustomFolderPath,
        &epic.Slug,  // ← NEW FIELD
        &epic.CreatedAt,
        &epic.UpdatedAt,
    )

    if err != nil {
        return nil, err
    }

    return epic, nil
}

// extractNumericEpicKey extracts numeric key from slugged key
// "E05-some-slug" → "E05"
// "E05" → "E05" (unchanged)
func extractNumericEpicKey(keyOrSlug string) string {
    parts := strings.Split(keyOrSlug, "-")

    // Epic keys start with E## (first part only)
    if len(parts) > 0 && strings.HasPrefix(parts[0], "E") {
        return parts[0]
    }

    return keyOrSlug
}
```

---

## 4. Updated Epic Creation

### File: `internal/cli/commands/epic.go` (updated runEpicCreate)

```go
func runEpicCreate(cmd *cobra.Command, args []string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Get title from args
    epicTitle := args[0]

    // Get optional flags
    filename, _ := cmd.Flags().GetString("filename")
    force, _ := cmd.Flags().GetBool("force")

    // Get database connection
    dbPath, err := cli.GetDBPath()
    if err != nil {
        cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
        return fmt.Errorf("database path error")
    }

    database, err := db.InitDB(dbPath)
    if err != nil {
        cli.Error("Error: Database error. Run with --verbose for details.")
        if cli.GlobalConfig.Verbose {
            fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
        }
        os.Exit(2)
    }

    // Get project root
    projectRoot, err := os.Getwd()
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to get working directory: %s", err.Error()))
        os.Exit(1)
    }

    // Get repositories
    repoDb := repository.NewDB(database)
    epicRepo := repository.NewEpicRepository(repoDb)

    // Generate epic key (E01, E02, etc.)
    epicKey, err := generateNextEpicKey(ctx, epicRepo)
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to generate epic key: %s", err.Error()))
        os.Exit(1)
    }

    // ✅ NEW: Generate slug from title (ONE TIME)
    epicSlug := slug.Generate(epicTitle)
    if epicSlug == "" {
        cli.Error("Failed to generate valid slug from title")
        os.Exit(1)
    }

    // Validate and process custom path if provided
    var customFolderPath *string
    if epicCreatePath != "" {
        _, relPath, err := utils.ValidateFolderPath(epicCreatePath, projectRoot)
        if err != nil {
            cli.Error(fmt.Sprintf("Error: %v", err))
            os.Exit(1)
        }
        customFolderPath = &relPath
    }

    // ✅ NEW: Compute file path using PathResolver
    pathResolver := pathresolver.NewPathResolver(epicRepo, nil, nil, projectRoot)
    filePath := pathResolver.ComputeEpicPath(epicKey, epicSlug, customFolderPath)

    // Handle custom filename override
    if filename != "" {
        absPath, relPath, err := taskcreation.ValidateCustomFilename(filename, projectRoot)
        if err != nil {
            cli.Error(fmt.Sprintf("Error: %v", err))
            os.Exit(1)
        }
        filePath = relPath
    }

    // Check for file collision
    existingEpic, err := epicRepo.GetByFilePath(ctx, filePath)
    if err != nil && err != sql.ErrNoRows {
        cli.Error(fmt.Sprintf("Error checking file collision: %s", err.Error()))
        os.Exit(1)
    }

    if existingEpic != nil {
        if !force {
            cli.Error(fmt.Sprintf(
                "File '%s' is already claimed by epic %s ('%s'). Use --force to reassign",
                filePath, existingEpic.Key, existingEpic.Title,
            ))
            os.Exit(1)
        }

        // Force mode: clear file path from old epic
        if err := epicRepo.UpdateFilePath(ctx, existingEpic.Key, nil); err != nil {
            cli.Error(fmt.Sprintf("Failed to unassign file from %s: %s", existingEpic.Key, err.Error()))
            os.Exit(1)
        }
    }

    // ✅ Create database record (FIRST, with slug)
    now := time.Now().UTC()
    epic := &models.Epic{
        Key:              epicKey,
        Title:            epicTitle,
        Slug:             &epicSlug,  // ← STORE SLUG IN DATABASE
        Description:      epicCreateDescription,
        Status:           models.EpicStatusActive,
        Priority:         models.PriorityMedium,
        FilePath:         &filePath,
        CustomFolderPath: customFolderPath,
        CreatedAt:        now,
        UpdatedAt:        now,
    }

    err = epicRepo.Create(ctx, epic)
    if err != nil {
        cli.Error(fmt.Sprintf("Failed to create epic: %s", err.Error()))
        os.Exit(1)
    }

    // ✅ Create file (SECOND, using database data)
    fullFilePath := filepath.Join(projectRoot, filePath)
    if err := writeEpicFile(fullFilePath, epic); err != nil {
        // Rollback: delete database record
        epicRepo.Delete(ctx, epic.ID)
        cli.Error(fmt.Sprintf("Failed to write epic file: %s", err.Error()))
        os.Exit(1)
    }

    // Output success
    if cli.GlobalConfig.JSON {
        cli.OutputJSON(epic)
    } else {
        cli.Success(fmt.Sprintf("Epic %s created: %s", epic.Key, epic.Title))
        cli.Info(fmt.Sprintf("File: %s", filePath))
    }

    return nil
}

// writeEpicFile writes epic file with YAML frontmatter
func writeEpicFile(filePath string, epic *models.Epic) error {
    // Ensure directory exists
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Render template with frontmatter
    tmpl := `---
epic_key: {{ .Key }}
slug: {{ .Slug }}
title: {{ .Title }}
{{ if .Description }}description: {{ .Description }}{{ end }}
status: {{ .Status }}
priority: {{ .Priority }}
{{ if .BusinessValue }}business_value: {{ .BusinessValue }}{{ end }}
created_at: {{ .CreatedAt.Format "2006-01-02" }}
---

# Epic: {{ .Title }}

{{ if .Description }}
## Description

{{ .Description }}
{{ end }}

## Features

<!-- Features will be listed here -->
`

    t, err := template.New("epic").Parse(tmpl)
    if err != nil {
        return fmt.Errorf("failed to parse template: %w", err)
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, epic); err != nil {
        return fmt.Errorf("failed to execute template: %w", err)
    }

    // Write file
    if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
        return fmt.Errorf("failed to write file: %w", err)
    }

    return nil
}
```

---

## 5. Updated Discovery (Validate Slug)

### File: `internal/sync/discovery.go` (updated importDiscoveredEntities)

```go
func (e *SyncEngine) importDiscoveredEntities(ctx context.Context, epics []discovery.DiscoveredEpic, features []discovery.DiscoveredFeature) (int, int, []string, error) {
    tx, err := e.db.BeginTx(ctx, nil)
    if err != nil {
        return 0, 0, nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    epicsImported := 0
    featuresImported := 0
    warnings := []string{}

    // Import epics
    for _, epic := range epics {
        // Validate epic key format
        if err := models.ValidateEpicKey(epic.Key); err != nil {
            warnings = append(warnings, fmt.Sprintf("Skipping epic with invalid key format: %s", epic.Key))
            continue
        }

        // Check if epic already exists
        existingEpic, err := e.epicRepo.GetByKey(ctx, epic.Key)
        if err != nil && err != sql.ErrNoRows {
            return 0, 0, nil, fmt.Errorf("failed to check epic %s: %w", epic.Key, err)
        }

        if existingEpic != nil {
            // ✅ NEW: Validate slug matches database
            if epic.Slug != nil && existingEpic.Slug != nil && *epic.Slug != *existingEpic.Slug {
                warnings = append(warnings, fmt.Sprintf(
                    "Epic %s: File slug '%s' does not match database slug '%s'",
                    epic.Key, *epic.Slug, *existingEpic.Slug,
                ))
            }

            // Update if needed
            needsUpdate := false
            if epic.Title != "" && existingEpic.Title != epic.Title {
                existingEpic.Title = epic.Title
                needsUpdate = true
            }
            if epic.Description != nil && existingEpic.Description != epic.Description {
                existingEpic.Description = epic.Description
                needsUpdate = true
            }

            // ✅ NEW: Update slug if missing in database (backfill)
            if existingEpic.Slug == nil && epic.Slug != nil {
                existingEpic.Slug = epic.Slug
                needsUpdate = true
            }

            if needsUpdate {
                if err := e.epicRepo.Update(ctx, existingEpic); err != nil {
                    return 0, 0, nil, fmt.Errorf("failed to update epic %s: %w", epic.Key, err)
                }
            }
        } else {
            // ✅ NEW: Create new epic with slug
            newEpic := &models.Epic{
                Key:              epic.Key,
                Title:            epic.Title,
                Slug:             epic.Slug,  // ← STORE SLUG FROM FILE
                Status:           models.EpicStatusActive,
                Priority:         models.PriorityMedium,
                FilePath:         &epic.FilePath,
                CustomFolderPath: epic.CustomFolderPath,
            }
            if epic.Description != nil {
                newEpic.Description = epic.Description
            }
            if err := e.epicRepo.Create(ctx, newEpic); err != nil {
                return 0, 0, nil, fmt.Errorf("failed to create epic %s: %w", epic.Key, err)
            }
            epicsImported++
        }
    }

    // Similar for features...

    if err := tx.Commit(); err != nil {
        return 0, 0, nil, fmt.Errorf("failed to commit transaction: %w", err)
    }

    return epicsImported, featuresImported, warnings, nil
}
```

---

## 6. Updated Models

### File: `internal/models/epic.go` (updated Epic struct)

```go
package models

import (
    "time"
)

// Epic represents a high-level project epic
type Epic struct {
    ID               int64      `json:"id"`
    Key              string     `json:"key"`
    Title            string     `json:"title"`
    Slug             *string    `json:"slug,omitempty"`  // ← NEW FIELD
    Description      *string    `json:"description,omitempty"`
    Status           EpicStatus `json:"status"`
    Priority         Priority   `json:"priority"`
    BusinessValue    *string    `json:"business_value,omitempty"`
    FilePath         *string    `json:"file_path,omitempty"`
    CustomFolderPath *string    `json:"custom_folder_path,omitempty"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

// Similar updates for Feature and Task models
```

---

## 7. CLI Migration Command

### File: `internal/cli/commands/migrate.go`

```go
package commands

import (
    "context"
    "fmt"
    "os"

    "github.com/jwwelbor/shark-task-manager/internal/cli"
    "github.com/jwwelbor/shark-task-manager/internal/db"
    "github.com/jwwelbor/shark-task-manager/internal/db/migrations"
    "github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
    Use:   "migrate",
    Short: "Run database migrations",
    Long:  `Run database schema migrations for backward compatibility and upgrades.`,
}

var migrateAddSlugCmd = &cobra.Command{
    Use:   "add-slug-column",
    Short: "Add slug column to epics, features, and tasks",
    Long: `Add slug column to all entity tables and backfill from existing file_path values.
This migration is idempotent and safe to run multiple times.`,
    RunE: runMigrateAddSlug,
}

func init() {
    cli.RootCmd.AddCommand(migrateCmd)
    migrateCmd.AddCommand(migrateAddSlugCmd)
}

func runMigrateAddSlug(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Get database connection
    dbPath, err := cli.GetDBPath()
    if err != nil {
        cli.Error(fmt.Sprintf("Error: Failed to get database path: %v", err))
        return fmt.Errorf("database path error")
    }

    database, err := db.InitDB(dbPath)
    if err != nil {
        cli.Error("Error: Database error. Run with --verbose for details.")
        if cli.GlobalConfig.Verbose {
            fmt.Fprintf(os.Stderr, "Database error: %v\n", err)
        }
        os.Exit(2)
    }
    defer database.Close()

    // Run migration
    cli.Info("Running migration: add slug columns...")
    if err := migrations.MigrationAddSlugColumns(database); err != nil {
        cli.Error(fmt.Sprintf("Migration failed: %s", err.Error()))
        os.Exit(1)
    }

    cli.Success("Migration completed successfully")
    cli.Info("Next steps:")
    cli.Info("  1. Run 'shark epic list' to verify epics")
    cli.Info("  2. Run 'shark feature list' to verify features")
    cli.Info("  3. Run 'shark task list' to verify tasks")

    return nil
}
```

---

## 8. Testing Examples

### File: `internal/pathresolver/resolver_test.go`

```go
package pathresolver

import (
    "context"
    "testing"

    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/repository"
    "github.com/stretchr/testify/assert"
)

func TestResolveEpicPath(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    epicRepo := repository.NewEpicRepository(db)
    resolver := NewPathResolver(epicRepo, nil, nil, "/home/user/project")

    ctx := context.Background()

    // Create test epic with slug
    slug := "task-management-cli"
    epic := &models.Epic{
        Key:   "E05",
        Title: "Task Management CLI",
        Slug:  &slug,
    }
    err := epicRepo.Create(ctx, epic)
    assert.NoError(t, err)

    // Test path resolution
    path, err := resolver.ResolveEpicPath(ctx, "E05")
    assert.NoError(t, err)
    assert.Equal(t, "/home/user/project/docs/plan/E05-task-management-cli/epic.md", path)
}

func TestResolveEpicPath_WithCustomPath(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    epicRepo := repository.NewEpicRepository(db)
    resolver := NewPathResolver(epicRepo, nil, nil, "/home/user/project")

    ctx := context.Background()

    // Create test epic with custom folder path
    slug := "q1-roadmap"
    customPath := "docs/roadmap/2025"
    epic := &models.Epic{
        Key:              "E01",
        Title:            "Q1 Roadmap",
        Slug:             &slug,
        CustomFolderPath: &customPath,
    }
    err := epicRepo.Create(ctx, epic)
    assert.NoError(t, err)

    // Test path resolution
    path, err := resolver.ResolveEpicPath(ctx, "E01")
    assert.NoError(t, err)
    assert.Equal(t, "/home/user/project/docs/roadmap/2025/E01-q1-roadmap/epic.md", path)
}

// Similar tests for features and tasks...
```

---

## Summary

These implementation examples provide:

1. **Database Migration**: Add slug columns and backfill from existing data
2. **PathResolver**: Database-first path resolution (no file reads)
3. **Flexible Key Lookup**: Support both numeric and slugged keys
4. **Updated Creation**: Store slugs at creation time
5. **Discovery Validation**: Validate file slugs against database
6. **Updated Models**: Add slug field to all entities
7. **CLI Migration Command**: Run migrations from command line
8. **Testing**: Unit tests for path resolution

**Next Steps**:
1. Review and refine implementation
2. Create tasks for each component
3. Implement in phases (migration → storage → resolver → validation)
4. Test thoroughly at each phase
