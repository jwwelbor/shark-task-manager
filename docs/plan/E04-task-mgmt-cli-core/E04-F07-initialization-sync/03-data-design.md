# Data Design: Initialization & Synchronization

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F07-initialization-sync
**Date**: 2025-12-14
**Author**: db-admin

## Overview

This document defines the data structures, formats, and persistence patterns for the Initialization & Synchronization feature. Unlike typical features, E04-F07 does not introduce new database tables but instead works with existing tables from E04-F01 and manages configuration and template files on the filesystem.

---

## Data Structures

### 1. Configuration File (.pmconfig.json)

**Location**: `./.pmconfig.json` (project root)

**Format**: JSON

**Schema**:
```json
{
  "default_epic": "E04",                    // Optional: Default epic key
  "default_agent": "general-purpose",        // Optional: Default agent type
  "color_enabled": true,                     // Boolean: Enable colored output
  "json_output": false                       // Boolean: JSON output mode
}
```

**Field Specifications**:

| Field | Type | Required | Default | Constraints | Description |
|-------|------|----------|---------|-------------|-------------|
| `default_epic` | string\|null | No | null | Must match `E\d{2}` pattern | Default epic for task creation |
| `default_agent` | string\|null | No | null | Must be valid agent type enum | Default agent for task creation |
| `color_enabled` | boolean | No | true | N/A | Enable ANSI color codes in output |
| `json_output` | boolean | No | false | N/A | Output all results as JSON |

**Validation Rules**:
1. File must be valid JSON
2. If `default_epic` provided, must exist in database
3. If `default_agent` provided, must be in: `general-purpose`, `backend-developer`, `frontend-developer`, `devops-engineer`, `api-developer`
4. Unknown fields ignored (forward compatibility)

**Example**:
```json
{
  "default_epic": "E04",
  "default_agent": "backend-developer",
  "color_enabled": true,
  "json_output": false
}
```

---

### 2. Task Frontmatter Format

**Purpose**: Define task metadata in markdown files for sync parsing

**Format**: YAML frontmatter between `---` delimiters

**Schema**:
```yaml
---
task_key: T-E04-F01-001
status: todo
feature: /path/to/feature
created: 2025-12-14
assigned_agent: general-purpose
dependencies: []
estimated_time: 4 hours
file_path: /path/to/task.md
---
```

**Field Specifications**:

| Field | Type | Required | Format | Sync Behavior |
|-------|------|----------|--------|---------------|
| `task_key` | string | Yes | `T-E\d{2}-F\d{2}-\d{3}` | Must match filename |
| `status` | string | Yes | Enum | Synced to DB, validated against folder location |
| `feature` | string | Yes | Path to feature dir | Extracted to feature_id FK |
| `created` | string | Yes | YYYY-MM-DD | Parsed to ISO 8601 timestamp |
| `assigned_agent` | string | Yes | Enum | Synced to DB |
| `dependencies` | array | No | Array of task keys | Synced to DB as JSON |
| `estimated_time` | string | No | "X hours" | Informational only |
| `file_path` | string | No | Absolute path | Updated during file moves |

**Status Enum Values**:
- `todo`
- `in_progress`
- `ready_for_review`
- `completed`
- `archived`

**Agent Type Enum Values**:
- `general-purpose`
- `backend-developer`
- `frontend-developer`
- `devops-engineer`
- `api-developer`

**Validation Rules**:
1. `task_key` must match filename (e.g., file `T-E04-F01-001.md` requires `task_key: T-E04-F01-001`)
2. `status` must match folder location or sync will move file
3. `feature` path must reference existing feature directory
4. `dependencies` must be array of valid task keys (circular deps allowed for now)
5. Invalid YAML causes file skip with warning

**Example**:
```yaml
---
task_key: T-E04-F01-003
status: in_progress
feature: /home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema
created: 2025-12-14
assigned_agent: backend-developer
dependencies: ["T-E04-F01-001", "T-E04-F01-002"]
estimated_time: 12 hours
file_path: /home/jwwelbor/.claude/docs/tasks/active/T-E04-F01-003.md
---

# PRP: Repository Layer Implementation

...
```

---

### 3. Template Files

**Location**: `./templates/` (project directory)

**Source**: `pm/templates/` (package resources)

#### 3.1 task.md Template

**Purpose**: Default task/PRP template

**Content**:
```markdown
---
task_key: {TASK_KEY}
status: todo
feature: {FEATURE_PATH}
created: {CREATED_DATE}
assigned_agent: {AGENT_TYPE}
dependencies: []
estimated_time: X hours
file_path: {FILE_PATH}
---

# PRP: {TITLE}

## Goal

{Single sentence describing what will be built and why}

## Success Criteria

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] All validation gates pass

## Implementation Guidance

### Overview

{High-level description of component}

### Key Requirements

- Requirement 1 - See [Design Doc](../path/to/doc.md#section)
- Requirement 2 - See [Design Doc](../path/to/doc.md#section)

### Files to Create/Modify

- `path/to/file.py` - Description

### Integration Points

- Component A: Description
- Component B: Description

## Validation Gates

- **Type Safety**: mypy passes
- **Unit Tests**: Coverage >80%
- **Integration Tests**: End-to-end flow works

## Context & Resources

- [Architecture](../path/to/architecture.md)
- [Data Design](../path/to/data-design.md)

## Notes for Agent

- Note 1
- Note 2
```

**Placeholders**:
- `{TASK_KEY}` - Generated task key
- `{FEATURE_PATH}` - Path to feature directory
- `{CREATED_DATE}` - Current date (YYYY-MM-DD)
- `{AGENT_TYPE}` - Assigned agent type
- `{FILE_PATH}` - Absolute path to task file
- `{TITLE}` - Task title from CLI

---

#### 3.2 epic.md Template

**Purpose**: Epic PRD template

**Content**:
```markdown
# Epic: {EPIC_TITLE}

**Epic Key**: {EPIC_KEY}
**Status**: {STATUS}
**Created**: {CREATED_DATE}

## Vision

{High-level epic vision and business value}

## Goals

1. Goal 1
2. Goal 2
3. Goal 3

## Features

| Feature | Status | Description |
|---------|--------|-------------|
| F01 | Planned | Feature 1 description |
| F02 | Planned | Feature 2 description |

## Success Metrics

- Metric 1: Target value
- Metric 2: Target value

## Timeline

- **Start**: {START_DATE}
- **End**: {END_DATE}
- **Duration**: X weeks

## Dependencies

- Dependency 1
- Dependency 2

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Risk 1 | High | Mitigation strategy |

## Out of Scope

- Item 1
- Item 2
```

---

#### 3.3 feature.md Template

**Purpose**: Feature PRD template

**Content**:
```markdown
# Feature: {FEATURE_TITLE}

**Feature Key**: {FEATURE_KEY}
**Epic**: {EPIC_KEY}
**Status**: {STATUS}
**Created**: {CREATED_DATE}

## Goal

### Problem

{Problem statement}

### Solution

{Solution description}

### Impact

- Impact 1
- Impact 2

## User Stories

### Must-Have

**Story 1**: As a user, I want to... so that...

### Should-Have

**Story 2**: As a user, I want to... so that...

## Requirements

### Functional Requirements

1. Requirement 1
2. Requirement 2

### Non-Functional Requirements

- Performance: Target
- Security: Requirements
- Reliability: Requirements

## Acceptance Criteria

**Given** initial condition
**When** action occurs
**Then** expected outcome

## Out of Scope

1. Item 1
2. Item 2
```

---

## Data Mapping: File → Database

### Task Frontmatter → Database Task Table

| Frontmatter Field | Database Column | Transformation | Notes |
|-------------------|-----------------|----------------|-------|
| `task_key` | `key` (TEXT) | Direct copy | Must be unique |
| `status` | `status` (TEXT) | Direct copy | Validated against enum |
| `feature` (path) | `feature_id` (INTEGER FK) | Extract feature key from path, query ID | E.g., path ends with `E04-F01-database-schema` → query feature with key `E04-F01` |
| `created` | `created_at` (TIMESTAMP) | Parse YYYY-MM-DD to ISO 8601 | Append `T00:00:00Z` |
| `assigned_agent` | `agent_type` (TEXT) | Direct copy | Validated against enum |
| `dependencies` | `depends_on` (TEXT JSON) | Serialize array to JSON | E.g., `["T-E04-F01-001"]` → `'["T-E04-F01-001"]'` |
| N/A (from filename) | `file_path` (TEXT) | Absolute path to file | Updated during moves |
| N/A | `title` (TEXT) | Extracted from first `#` heading | Parse markdown body |
| N/A | `description` (TEXT) | Extracted from "## Goal" section | Parse markdown body |
| N/A | `priority` (TEXT) | Default "medium" | Not in frontmatter |
| N/A | `progress_pct` (INTEGER) | Default 0 | Not in frontmatter |
| N/A | `updated_at` (TIMESTAMP) | File modification time | From filesystem |

**Feature Key Extraction Logic**:
```python
def extract_feature_key(feature_path: str) -> str:
    """
    Extract feature key from path like:
    /home/user/.claude/docs/plan/E04-task-mgmt-cli-core/E04-F01-database-schema
    → E04-F01
    """
    dir_name = Path(feature_path).name  # "E04-F01-database-schema"
    match = re.match(r'(E\d{2}-F\d{2})', dir_name)
    if match:
        return match.group(1)
    raise ValueError(f"Invalid feature path: {feature_path}")
```

---

## File System Structure

### Initialization Creates

```
project-root/
├── project.db                      # SQLite database (E04-F01)
├── .pmconfig.json                  # Configuration file (E04-F07)
├── docs/
│   └── tasks/
│       ├── todo/                   # Status: todo
│       ├── active/                 # Status: in_progress
│       ├── ready-for-review/       # Status: ready_for_review
│       ├── completed/              # Status: completed
│       └── archived/               # Status: archived
└── templates/
    ├── task.md                     # Task template
    ├── epic.md                     # Epic template
    └── feature.md                  # Feature template
```

---

### Folder → Status Mapping

| Folder | Expected Status | Sync Behavior |
|--------|----------------|---------------|
| `docs/tasks/todo/` | `todo` | File-wins: move file if status != "todo" |
| `docs/tasks/active/` | `in_progress` | File-wins: move file if status != "in_progress" |
| `docs/tasks/ready-for-review/` | `ready_for_review` | File-wins: move file if status != "ready_for_review" |
| `docs/tasks/completed/` | `completed` | File-wins: move file if status != "completed" |
| `docs/tasks/archived/` | `archived` | File-wins: move file if status != "archived" |

**Conflict Example**:
- File location: `docs/tasks/todo/T-E04-F01-001.md`
- Frontmatter status: `in_progress`
- **File-Wins Strategy**: Move file to `docs/tasks/active/T-E04-F01-001.md`, update DB status to `in_progress`
- **Database-Wins Strategy**: Update frontmatter to `status: todo`, keep file in `todo/`

---

## Sync Data Structures

### SyncReport (Output)

**Purpose**: Report sync results to user

**Format**: Python dataclass → JSON

```python
@dataclass
class SyncReport:
    scanned: int                     # Total files scanned
    imported: int                    # New tasks created
    updated: int                     # Existing tasks updated
    conflicts: int                   # Conflicts resolved
    skipped: int                     # Files skipped (invalid)
    warnings: List[str]              # Warning messages
    errors: List[str]                # Error messages
    dry_run: bool                    # Was this a dry-run?

    def to_json(self) -> str:
        """Serialize to JSON"""
        return json.dumps(asdict(self), indent=2)

    def to_text(self) -> str:
        """Format as human-readable text"""
        return f"""
Sync completed:
  Files scanned: {self.scanned}
  New tasks imported: {self.imported}
  Existing tasks updated: {self.updated}
  Conflicts resolved: {self.conflicts}
  Warnings: {len(self.warnings)}
  Errors: {len(self.errors)}
        """
```

**JSON Example**:
```json
{
  "scanned": 47,
  "imported": 5,
  "updated": 3,
  "conflicts": 2,
  "skipped": 1,
  "warnings": [
    "Warning: Invalid frontmatter in todo/broken.md, skipping"
  ],
  "errors": [],
  "dry_run": false
}
```

---

### InitResult (Output)

**Purpose**: Report init results to user

**Format**: Python dataclass → text output

```python
@dataclass
class InitResult:
    database_created: bool
    folders_created: bool
    config_created: bool
    templates_installed: bool
    warnings: List[str]

    def to_text(self) -> str:
        """Format as human-readable text"""
        lines = ["PM CLI initialized successfully!", ""]
        if self.database_created:
            lines.append("✓ Database created: project.db")
        else:
            lines.append("- Database already exists (skipped)")

        if self.folders_created:
            lines.append("✓ Folder structure created")
        else:
            lines.append("- Folders already exist (skipped)")

        if self.config_created:
            lines.append("✓ Config file created: .pmconfig.json")
        else:
            lines.append("- Config file already exists (skipped)")

        if self.templates_installed:
            lines.append("✓ Templates installed to templates/")
        else:
            lines.append("- Templates already exist (skipped)")

        if self.warnings:
            lines.append("")
            lines.append("Warnings:")
            for warning in self.warnings:
                lines.append(f"  - {warning}")

        lines.append("")
        lines.append("Next steps:")
        lines.append("  1. Edit .pmconfig.json to set default epic and agent")
        lines.append("  2. Create tasks with: pm task create ...")
        lines.append("  3. Import existing tasks with: pm sync")

        return "\n".join(lines)
```

---

## Conflict Detection Data

### Conflict Record

**Purpose**: Track detected conflicts during sync

```python
@dataclass
class Conflict:
    task_key: str
    field: str                       # "status", "priority", "title", etc.
    file_value: Any                  # Value from file frontmatter
    db_value: Any                    # Value from database
    file_path: Path
    resolution: str                  # "file-wins", "database-wins", "newer-wins"

    def describe(self) -> str:
        """Human-readable conflict description"""
        return f"""
Conflict detected in {self.task_key}:
  Field: {self.field}
  Database: {self.db_value}
  File: {self.file_value}
  Resolution: {self.resolution} ({"file" if self.resolution == "file-wins" else "database"} updated)
        """
```

**Example**:
```
Conflict detected in T-E04-F01-003:
  Field: status
  Database: in_progress
  File: todo
  Resolution: file-wins (database updated to "todo")
```

---

## Data Validation Rules

### Frontmatter Validation

1. **YAML Structure**:
   - Must be valid YAML between `---` delimiters
   - Must be at start of file
   - Invalid YAML → skip file, log warning

2. **Required Fields**:
   - `task_key` - always required
   - `status` - always required
   - `feature` - always required
   - Missing required field → skip file, log warning

3. **Key Format**:
   - Must match `T-E\d{2}-F\d{2}-\d{3}` pattern
   - Must match filename
   - Mismatch → skip file, log warning

4. **Feature Path**:
   - Must be absolute path
   - Must end with `E\d{2}-F\d{2}-{feature-slug}` pattern
   - Feature must exist in DB (or `--create-missing` flag)
   - Invalid → skip file, log warning

5. **Dependencies**:
   - Must be array (YAML list)
   - Each element must match task key format
   - Invalid format → skip file, log warning

---

### Config Validation

1. **JSON Structure**:
   - Must be valid JSON
   - Invalid JSON → error on read, use defaults

2. **Field Types**:
   - `default_epic`: string or null
   - `default_agent`: string or null
   - `color_enabled`: boolean
   - `json_output`: boolean
   - Wrong type → error, use defaults

3. **Epic Key Validation**:
   - If provided, must match `E\d{2}` pattern
   - If provided, must exist in database
   - Invalid → warning, treat as null

4. **Agent Type Validation**:
   - If provided, must be in enum
   - Invalid → warning, treat as null

---

## Performance Considerations

### Bulk Data Loading

**Sync Performance**: Use batch operations for database writes

```python
# Good: Bulk insert
new_tasks = [task1, task2, task3, ...]
session.bulk_insert_mappings(Task, new_tasks)
session.commit()

# Bad: Individual inserts
for task_data in new_tasks:
    task = Task(**task_data)
    session.add(task)
    session.commit()  # Commits per task!
```

**Target**: 100 files in <10 seconds
- File scanning: ~2s
- YAML parsing: ~1s (10ms per file)
- DB operations: ~5s (bulk inserts)
- File moves: ~2s
- Total: ~10s

---

### File Scanning

**Strategy**: Use `Path.rglob()` for recursive scanning

```python
def scan_folders(base_path: Path, folder_filter: Optional[str]) -> List[Path]:
    """Scan for *.md files recursively"""
    folders = ['todo', 'active', 'ready-for-review', 'completed', 'archived']
    if folder_filter:
        folders = [f for f in folders if f == folder_filter]

    files = []
    for folder in folders:
        folder_path = base_path / 'docs' / 'tasks' / folder
        files.extend(folder_path.rglob('*.md'))

    return files
```

**Target**: Scan 1000 files in <1 second

---

## Data Persistence Patterns

### Transactional Sync

**Pattern**: All DB changes in single transaction

```python
with session_factory.get_session() as session:
    try:
        # All DB operations
        for action in actions:
            if action.type == ActionType.CREATE:
                session.add(Task(**action.data))
            elif action.type == ActionType.UPDATE:
                task = session.get(Task, action.task_id)
                for field, value in action.changes.items():
                    setattr(task, field, value)

        # Commit all at once
        session.commit()

    except Exception as e:
        session.rollback()
        raise SyncError(f"Sync failed: {e}")
```

**Rationale**: Atomicity - either all changes succeed or none do.

---

### Incremental Config Updates

**Pattern**: Read, modify, write config

```python
def update_config(updates: Dict[str, Any]) -> None:
    """Update specific config fields"""
    config = read_config()  # Read existing
    config.update(updates)  # Merge changes
    write_config(config)    # Write back
```

**Atomic Write**: Use temp file + rename

```python
def write_config(config: Dict[str, Any]) -> None:
    """Atomically write config file"""
    temp_path = Path('.pmconfig.json.tmp')
    final_path = Path('.pmconfig.json')

    # Write to temp file
    with temp_path.open('w') as f:
        json.dump(config, f, indent=2)

    # Atomic rename
    temp_path.rename(final_path)
```

---

## Summary

This data design defines:
- **Configuration Format**: `.pmconfig.json` structure and validation
- **Frontmatter Schema**: Task metadata format for sync
- **Template Structure**: Reusable templates for tasks, epics, features
- **Data Mapping**: File → database transformation logic
- **Sync Data Structures**: Report formats and conflict tracking
- **Validation Rules**: Comprehensive input validation
- **Performance Patterns**: Bulk operations and atomic writes

**Key Principles**:
1. **Validation First**: Validate all inputs before processing
2. **Fail Fast**: Skip invalid files, don't halt entire sync
3. **Atomicity**: Use transactions and atomic file writes
4. **Performance**: Batch operations for large datasets
5. **Extensibility**: Forward-compatible config, ignore unknown fields
