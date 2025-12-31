# Architecture Diagrams - Slug and Path Management

## Current Architecture (Problematic)

### Data Flow - Epic Creation

```
┌─────────────────────────────────────────────────────────────────┐
│ User: shark epic create "Task Management CLI Capabilities"     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Generate Key: E05 (from database sequence)                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Create Database Record:                                     │
│    key: "E05"                                                   │
│    title: "Task Management CLI Capabilities"                   │
│    slug: NULL ← ❌ NOT STORED                                   │
│    file_path: NULL (not computed yet)                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Generate Slug (on-the-fly):                                 │
│    slug.Generate(title)                                         │
│    → "task-management-cli-capabilities"                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Compute File Path (PathBuilder):                            │
│    docs/plan/E05-task-management-cli-capabilities/epic.md      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Create File:                                                 │
│    # Epic: Task Management CLI Capabilities                    │
│    **Epic Key**: E05-task-management-cli-capabilities          │
│                   ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑           │
│                   ❌ SLUGGED KEY IN FILE, NOT IN DATABASE       │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow - Discovery (Sync)

```
┌─────────────────────────────────────────────────────────────────┐
│ User: shark sync                                                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Scan Filesystem:                                             │
│    Find folders matching E##-* pattern                          │
│    Found: E05-task-management-cli-capabilities/                 │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Parse Folder Name:                                           │
│    Extract key: "E05"                                           │
│    Extract slug: "task-management-cli-capabilities"             │
│                   ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑           │
│                   ❌ SLUG FROM FILESYSTEM, NOT DATABASE         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Read File (epic.md):                                         │
│    Parse markdown to find:                                      │
│    **Epic Key**: E05-task-management-cli-capabilities           │
│                   ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑           │
│                   ❌ REQUIRES MARKDOWN PARSING                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Update Database:                                             │
│    title: slug from folder (becomes database title!)           │
│    file_path: path to epic.md                                  │
│              ↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑↑               │
│              ❌ FILESYSTEM IS SOURCE OF TRUTH                   │
└─────────────────────────────────────────────────────────────────┘
```

### Problem Summary

```
┌────────────────┐         ┌────────────────┐         ┌────────────────┐
│   Filesystem   │ ──────> │   Discovery    │ ──────> │    Database    │
│ (source of     │  reads  │   (syncs)      │ updates │  (secondary)   │
│  truth) ❌     │         │                │         │                │
└────────────────┘         └────────────────┘         └────────────────┘
       │                            │                         │
       │                            │                         │
       ▼                            ▼                         ▼
  Slug in folder              Slug becomes               Slug NOT stored
  E05-task-mgmt...            database title             (computed later)
```

---

## Proposed Architecture (Database-First)

### Data Flow - Epic Creation

```
┌─────────────────────────────────────────────────────────────────┐
│ User: shark epic create "Task Management CLI Capabilities"     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Generate Key: E05 (from database sequence)                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Generate Slug (ONE TIME):                                    │
│    slug := slug.Generate(title)                                 │
│    → "task-management-cli-capabilities"                         │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Create Database Record (FIRST):                             │
│    key: "E05"                                                   │
│    title: "Task Management CLI Capabilities"                   │
│    slug: "task-management-cli-capabilities" ← ✅ STORED         │
│    file_path: "docs/plan/E05-.../epic.md"                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Compute File Path (from database):                          │
│    PathResolver reads:                                          │
│    - key: "E05"                                                 │
│    - slug: "task-management-cli-capabilities"                   │
│    - custom_folder_path: NULL (use default)                    │
│    → docs/plan/E05-task-management-cli-capabilities/epic.md    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. Create File (SECOND, using database data):                  │
│    ---                                                          │
│    epic_key: E05                                                │
│    slug: task-management-cli-capabilities                       │
│    title: Task Management CLI Capabilities                     │
│    status: active                                               │
│    ---                                                          │
│    # Epic: Task Management CLI Capabilities                    │
│                                                                 │
│    ✅ FRONTMATTER FROM DATABASE, NO PARSING NEEDED              │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow - Discovery (Sync)

```
┌─────────────────────────────────────────────────────────────────┐
│ User: shark sync                                                │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Scan Filesystem:                                             │
│    Find .md files with YAML frontmatter                         │
│    Found: E05-task-management-cli-capabilities/epic.md          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Parse YAML Frontmatter (fast):                               │
│    epic_key: "E05"                                              │
│    slug: "task-management-cli-capabilities"                     │
│    title: "Task Management CLI Capabilities"                   │
│    ✅ STRUCTURED METADATA, NO MARKDOWN PARSING                  │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Validate Against Database:                                  │
│    DB record for E05:                                           │
│      slug: "task-management-cli-capabilities"                   │
│    File slug: "task-management-cli-capabilities"                │
│    ✅ MATCH - No conflict                                       │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. Update Database (if needed):                                │
│    Only update if title/description changed                    │
│    Slug remains immutable                                       │
│    ✅ DATABASE IS SOURCE OF TRUTH                               │
└─────────────────────────────────────────────────────────────────┘
```

### Solution Summary

```
┌────────────────┐         ┌────────────────┐         ┌────────────────┐
│    Database    │ ──────> │  PathResolver  │ ──────> │   Filesystem   │
│ (source of     │  reads  │  (computes)    │ writes  │  (reflects DB) │
│  truth) ✅     │         │                │         │                │
└────────────────┘         └────────────────┘         └────────────────┘
       │                            │                         │
       │                            │                         │
       ▼                            ▼                         ▼
  Slug stored at              Path computed                File contains
  creation time               from DB slug                 DB metadata
  (immutable)                 (no file reads)              (YAML frontmatter)
```

---

## Path Resolution Comparison

### Current (PathBuilder)

```
┌─────────────────────────────────────────────────────────────────┐
│ PathBuilder.ResolveTaskPath(                                    │
│     epicKey: "E05",                                             │
│     featureKey: "E05-F01",                                      │
│     taskKey: "T-E05-F01-001",                                   │
│     taskTitle: "Implement sync engine"  ← ❌ REQUIRES TITLE     │
│ )                                                               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Generate slug from title (on-the-fly):                      │
│    slug := slug.Generate("Implement sync engine")              │
│    → "implement-sync-engine"                                    │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Check custom_folder_path (requires DB lookup):              │
│    feature := featureRepo.GetByKey("E05-F01")                  │
│    epic := epicRepo.GetByKey("E05")                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. Build path from components:                                 │
│    docs/plan/E05/E05-F01/tasks/T-E05-F01-001-implement-...md   │
│    ❌ COMPLEX LOGIC, MULTIPLE STEPS                             │
└─────────────────────────────────────────────────────────────────┘
```

### Proposed (PathResolver)

```
┌─────────────────────────────────────────────────────────────────┐
│ PathResolver.ResolveTaskPath(                                   │
│     ctx: context,                                               │
│     taskKey: "T-E05-F01-001"                                    │
│ )                                                               │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 1. Read from database:                                          │
│    task := taskRepo.GetByKey("T-E05-F01-001")                  │
│    → file_path: "docs/plan/.../T-E05-F01-001-implement-...md"  │
│    ✅ SINGLE DB QUERY                                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. Return absolute path:                                        │
│    filepath.Join(projectRoot, task.FilePath)                   │
│    ✅ SIMPLE, FAST, DATABASE-FIRST                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Lookup Flexibility

### Current Behavior

```
shark epic get E05
    ↓
    ✅ Works (exact key match)

shark epic get E05-task-management-cli-capabilities
    ↓
    ❌ FAILS (no record with this key)
```

### Proposed Behavior

```
shark epic get E05
    ↓
    GetByKey("E05")
    ↓
    Try exact match: SELECT * FROM epics WHERE key = 'E05'
    ↓
    ✅ Found! Return epic

shark epic get E05-task-management-cli-capabilities
    ↓
    GetByKey("E05-task-management-cli-capabilities")
    ↓
    Try exact match: SELECT * FROM epics WHERE key = 'E05-...'
    ↓
    Not found, extract numeric key: "E05"
    ↓
    Try numeric match: SELECT * FROM epics WHERE key = 'E05'
    ↓
    ✅ Found! Return epic (same record)
```

---

## Migration Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ Phase 1: Add Slug Column                                        │
│                                                                 │
│ Before:                                                         │
│ ┌─────────────────────────────────────┐                        │
│ │ epics                               │                        │
│ │ - id: 1                              │                        │
│ │ - key: "E05"                         │                        │
│ │ - title: "Task Management CLI..."   │                        │
│ │ - file_path: "docs/plan/E05-.../..." │                        │
│ └─────────────────────────────────────┘                        │
│                                                                 │
│ After Migration:                                                │
│ ┌─────────────────────────────────────┐                        │
│ │ epics                               │                        │
│ │ - id: 1                              │                        │
│ │ - key: "E05"                         │                        │
│ │ - title: "Task Management CLI..."   │                        │
│ │ - slug: "task-management-cli-..."   │ ← ✅ NEW COLUMN        │
│ │ - file_path: "docs/plan/E05-.../..." │                        │
│ └─────────────────────────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Phase 2: Update Creation Logic                                  │
│                                                                 │
│ Old:                                                            │
│   1. Create DB record (no slug)                                │
│   2. Generate slug at path computation time                    │
│   3. Write file                                                │
│                                                                 │
│ New:                                                            │
│   1. Generate slug from title                                  │
│   2. Create DB record (with slug) ← ✅ SLUG STORED              │
│   3. Compute path from DB slug                                 │
│   4. Write file with frontmatter                               │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Phase 3: Implement PathResolver                                 │
│                                                                 │
│ Old:                                                            │
│   PathBuilder.ResolveTaskPath(keys, title, custom_path)        │
│   → Computes slug, queries DB, builds path                     │
│                                                                 │
│ New:                                                            │
│   PathResolver.ResolveTaskPath(ctx, taskKey)                   │
│   → Queries DB, returns file_path ← ✅ DATABASE-FIRST           │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ Phase 4: File Format Conversion (Optional)                      │
│                                                                 │
│ Old Format (epic.md):                                           │
│   # Epic: Task Management CLI                                  │
│   **Epic Key**: E05-task-management-cli-capabilities           │
│   **Status**: Active                                            │
│                                                                 │
│ New Format (epic.md):                                           │
│   ---                                                           │
│   epic_key: E05                                                 │
│   slug: task-management-cli-capabilities                        │
│   title: Task Management CLI Capabilities                      │
│   status: active                                                │
│   ---                                                           │
│   # Epic: Task Management CLI Capabilities                     │
│                                                                 │
│   ✅ YAML FRONTMATTER (consistent with tasks)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Performance Impact

### Path Resolution

```
Current (PathBuilder):
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Generate     │→ │ Query DB for │→ │ Query DB for │→ │ Build path   │
│ slug from    │  │ feature      │  │ epic         │  │ string       │
│ title (~0.3ms)│  │ custom_path  │  │ custom_path  │  │              │
└──────────────┘  │ (~0.3ms)     │  │ (~0.3ms)     │  │ (~0.1ms)     │
                  └──────────────┘  └──────────────┘  └──────────────┘
Total: ~1ms per call

Proposed (PathResolver):
┌──────────────┐  ┌──────────────┐
│ Query DB for │→ │ Return       │
│ task.file_path│  │ file_path    │
│ (~0.1ms)     │  │              │
└──────────────┘  └──────────────┘
Total: ~0.1ms per call

✅ 10x FASTER
```

### Discovery (Sync)

```
Current:
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Scan folders │→ │ Parse folder │→ │ Read epic.md │→ │ Parse        │
│              │  │ name for     │  │ file         │  │ markdown     │
│              │  │ key/slug     │  │              │  │ for metadata │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
Per file: ~5-10ms (markdown parsing)

Proposed:
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Scan files   │→ │ Parse YAML   │→ │ Validate     │
│ with .md     │  │ frontmatter  │  │ against DB   │
│              │  │              │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
Per file: ~2-3ms (YAML parsing)

✅ 2-3x FASTER
```

---

## Summary: Before & After

### Before (Current)
- ❌ Filesystem is source of truth
- ❌ Slugs computed on-the-fly
- ❌ Multiple database queries per path resolution
- ❌ Markdown parsing required for discovery
- ❌ Inconsistent file formats
- ❌ Slugged keys required for CLI

### After (Proposed)
- ✅ Database is source of truth
- ✅ Slugs stored at creation
- ✅ Single database query per path resolution
- ✅ YAML frontmatter for fast parsing
- ✅ Consistent patterns across all entities
- ✅ Numeric keys work (slugged keys optional)

**Result**: Cleaner architecture, better performance, better UX
