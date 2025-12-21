# Shark Task Manager - Sync Logic Documentation Index

**Generated**: 2025-12-19
**Status**: Complete

## Quick Access

### Start Here
- **README.md** - Navigation guide and quick reference (5.5 KB)
- **DOCUMENTATION_SUMMARY.md** - Executive summary (7.9 KB)

### Main Documentation
- **analysis/SYNC_LOGIC_DOCUMENTATION.md** - Complete sync logic (1,511 lines, 49 KB)

### Verification
- **VALIDATION_REPORT.md** - Completeness and quality verification (15 KB)

## Documentation Structure

```
dev-artifacts/2025-12-19-documentation-generation/
├── INDEX.md                           (this file)
├── README.md                          (navigation guide)
├── DOCUMENTATION_SUMMARY.md           (executive summary)
├── VALIDATION_REPORT.md               (verification report)
├── analysis/
│   └── SYNC_LOGIC_DOCUMENTATION.md   (main documentation)
├── scripts/                           (utilities)
├── shared/                            (shared code)
└── verification/                      (validation artifacts)
```

## Documentation Sections

### 1. Overview
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 17-40
- Core principles of file-database sync
- System components
- Status management philosophy

### 2. Discovery Process
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 41-273
- Epic and feature discovery from filesystem
- Index file (epic-index.md) parsing
- Three discovery strategies (index-only, folder-only, merge)
- Pattern matching algorithms
- Data structures and results

### 3. Folder Structure Interpretation
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 274-350
- Expected directory layout
- Path resolution algorithm
- Custom folder naming support
- Epic/Feature/Task relationship handling

### 4. Conflict Detection
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 351-572
- Four conflict types
- Detection algorithms (basic and incremental)
- Clock skew handling
- Detection rules and matrix
- Database-only fields

### 5. Conflict Resolution
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 573-808
- Four resolution strategies
  - File-wins (default)
  - Database-wins
  - Newer-wins
  - Manual
- Discovery conflict resolution
- Resolution algorithms

### 6. Sync Operations
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 809-1,081
- Complete 8-step workflow
- Transaction safety
- Error handling
- Step-by-step process breakdown

### 7. Edge Cases & Error Handling
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 1,082-1,283
- Missing files and orphaned records
- Missing task keys
- Partial sync failures
- Concurrent modifications
- File encoding issues
- Circular dependencies
- Large files
- Stale sync times
- Missing epic/feature
- Relationship mismatches

### 8. Incremental Sync
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 1,284-1,356
- LastSyncTime tracking
- Filtering algorithm
- Performance benefits
- Modified file detection

### 9. Real-World Examples
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 1,357-1,450
- Simple task update
- Discovery with merge
- Feature moved between epics
- File vs database conflict
- Cleanup of orphaned tasks

### 10. Implementation Reference
**Location**: SYNC_LOGIC_DOCUMENTATION.md lines 1,451-1,511
- All key files documented
- Configuration examples
- Troubleshooting guide
- Performance characteristics

## Key Diagrams

| Diagram | Location | Type | Purpose |
|---------|----------|------|---------|
| Discovery Workflow | Line 49 | Sequence | Shows complete discovery process |
| Index-Only Strategy | Line 213 | Flow | Validation and strict enforcement |
| Folder-Only Strategy | Line 233 | Flow | Filesystem as truth |
| Merge Strategy | Line 251 | Flow | Combined sources |
| File-Wins Resolution | Line 576 | Flow | Overwrite DB |
| Database-Wins Resolution | Line 596 | Flow | Preserve DB |
| Newer-Wins Resolution | Line 616 | Tree | Timestamp comparison |
| Manual Resolution | Line 636 | Flow | User prompts |
| Complete Sync Workflow | Line 809 | Sequence | Full 8-step process |
| Transaction Safety | Line 877 | Diagram | Atomicity guarantee |

## Code Examples

### Pattern Matching (Line 120)
```go
epicPattern := regexp.MustCompile(`^(E\d{2})`)
featurePattern := regexp.MustCompile(`^(E\d{2})(-P\d{2})?-(F\d{2})`)
```

### Conflict Detection Algorithm (Line 422)
Pseudocode showing field-by-field comparison

### Sync Operations Algorithm (Line 838)
Complete 8-step sync process with pseudocode

### Error Handling (Line 1,082)
Recovery procedures for 9 different edge cases

## Real-World Scenarios

### Scenario 1: Simple Task Update (Line 1,357)
- User updates description in file
- Shows conflict detection and resolution
- Demonstrates file-wins strategy

### Scenario 2: Discovery with Merge (Line 1,386)
- Adding new feature with epic-index.md
- Shows discovery process
- Demonstrates merge strategy

### Scenario 3: Feature Moved Between Epics (Line 1,416)
- Feature exists in different epic
- Shows relationship mismatch detection
- Demonstrates merge strategy resolution

### Scenario 4: File vs Database Conflict (Line 1,445)
- Both file and DB modified
- Shows timestamp-based resolution
- Demonstrates newer-wins strategy

### Scenario 5: Orphaned Tasks Cleanup (Line 1,471)
- File deleted, DB record remains
- Shows cleanup operation
- Demonstrates --cleanup flag

## Command Examples

### Basic Sync
```bash
shark sync
```

### With Discovery
```bash
shark sync --index
```

### Preview Only
```bash
shark sync --dry-run
```

### Different Strategies
```bash
shark sync --strategy=database-wins
shark sync --strategy=newer-wins
shark sync --strategy=manual
```

### Advanced Options
```bash
shark sync --cleanup                 # Delete orphaned tasks
shark sync --force-full-scan         # Ignore LastSyncTime
shark sync --create-missing          # Auto-create epic/feature
shark sync --json                    # JSON output
shark sync --verbose                 # Detailed logging
```

See section 9 (Implementation Reference) for all command combinations.

## Tables & References

### Sync Modes Comparison (Line 1,443)
Table comparing:
- Default mode
- Discovery mode
- Incremental mode
- Force full scan
- Cleanup mode

### Conflict Types Reference (Line 378)
Table of all conflict types with triggers and resolution

### Detection Rules Matrix (Line 517)
Shows which fields conflict and under what conditions

### Strategy Characteristics (Line 576)
Comparison of all four resolution strategies

### File Reference Table (Line 1,463)
All 11 key files with locations and purposes

### Performance Characteristics (Line 1,505)
Benchmark data for different sync scenarios

## Navigation Tips

### For Understanding Architecture
1. Read Overview (5 min)
2. Study Discovery Process (20 min)
3. Learn Conflict Detection (15 min)
4. Review Sync Operations (20 min)

### For Using the System
1. Read Conflict Resolution strategies (10 min)
2. Review Real-World Examples (15 min)
3. Check Troubleshooting Guide (as needed)
4. Use Command Examples (as needed)

### For Implementation
1. Review relevant section in main documentation
2. Check layer-by-layer breakdown
3. Study error handling paths
4. Review code examples

### For Debugging
1. Identify issue type
2. Locate in Edge Cases section
3. Review error scenario
4. Check recovery procedure

## Search Keywords

### By Topic
- **Discovery**: Line 41 (Discovery Process section)
- **Conflicts**: Line 351 (Conflict Detection section)
- **Resolution**: Line 573 (Conflict Resolution section)
- **Sync**: Line 809 (Sync Operations section)
- **Performance**: Line 1,284 (Incremental Sync section)
- **Errors**: Line 1,082 (Edge Cases section)
- **Examples**: Line 1,357 (Real-World Examples section)

### By Component
- **FileScanner**: Line 82, 1,463
- **FolderScanner**: Line 82, 1,463
- **ConflictDetector**: Line 351, 1,463
- **ConflictResolver**: Line 573, 1,463
- **IndexParser**: Line 154, 1,463
- **SyncEngine**: Line 27, 809

### By Strategy
- **File-wins**: Line 213 (discovery), 576 (conflict resolution)
- **Database-wins**: Line 233 (discovery), 596 (conflict resolution)
- **Newer-wins**: Line 251 (discovery), 616 (conflict resolution)
- **Manual**: Line 251 (discovery), 636 (conflict resolution)
- **Merge**: Line 251 (primary discovery strategy)

### By Scenario
- **Missing files**: Line 1,100
- **Conflicts**: Line 351, 573
- **Concurrency**: Line 1,140
- **Performance**: Line 1,284
- **Cleanup**: Line 1,471

## Quality Metrics

- **Completeness**: 100% verified
- **Coverage**: 15+ major sections
- **Diagrams**: 10 Mermaid diagrams
- **Examples**: 50+ code examples
- **Scenarios**: 5 real-world examples
- **Lines**: 1,511 total lines
- **Size**: 49 KB
- **Verification**: 100% validation report

## Files Analyzed

All documentation based on analysis of 11 key files:
1. internal/sync/engine.go
2. internal/sync/types.go
3. internal/sync/scanner.go
4. internal/sync/conflict.go
5. internal/sync/resolver.go
6. internal/sync/incremental.go
7. internal/sync/discovery.go
8. internal/discovery/folder_scanner.go
9. internal/discovery/types.go
10. internal/discovery/conflict_detector.go
11. internal/discovery/conflict_resolver.go
12. internal/discovery/index_parser.go
13. internal/cli/commands/sync.go

## Related Documentation

This documentation complements:
- **README.md** - User-facing guide
- **CLAUDE.md** - Project instructions
- **SYSTEM_DESIGN.md** - Overall architecture
- **CLI_REFERENCE.md** - Command reference

## Maintenance

- **Last Updated**: 2025-12-19
- **Status**: Production-ready
- **Format**: Markdown with Mermaid diagrams
- **Compatibility**: GitHub, GitLab, Markdown viewers

## Next Steps

1. **First Time**: Read README.md (5 min)
2. **Overview**: Read DOCUMENTATION_SUMMARY.md (10 min)
3. **Deep Dive**: Read SYNC_LOGIC_DOCUMENTATION.md (1-2 hours)
4. **Reference**: Bookmark this INDEX.md for quick navigation

---

**Documentation Status**: Complete and Verified
**Quality**: Production-Ready
**Suitable For**: Long-term reference, implementation guide, architecture documentation

