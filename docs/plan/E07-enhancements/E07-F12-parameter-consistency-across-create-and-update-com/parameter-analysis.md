# Parameter Consistency Analysis

**Feature**: E07-F12 - Parameter Consistency Across Create and Update Commands
**Date**: 2026-01-01
**Purpose**: Document current parameter inconsistencies and design unified interface

---

## Executive Summary

This analysis identifies parameter inconsistencies between create and update commands across Epic, Feature, and Task entities. The goal is to ensure DRY (Don't Repeat Yourself) principles and provide a consistent, predictable CLI interface.

### Key Findings

1. **Epic Commands**: Missing `--order` and `--notes` flags in create command
2. **Feature Commands**: Missing `--notes` flag in create command
3. **Task Commands**: Mostly consistent, but could benefit from parameter grouping refactoring
4. **Common Pattern**: Update commands have more parameters than create commands (anti-pattern)

---

## Current Parameter Inventory

### Epic Commands

#### `epic create` Parameters
```bash
--title <string>           # Positional arg (required)
--description <string>     # Description
--priority <string>        # Priority: high, medium, low (default: medium)
--business-value <string>  # Business value: high, medium, low
--path <string>            # Custom folder base path
--filename <string>        # Custom file path (takes precedence over --path)
--key <string>             # Custom key (e.g., E00, bugs)
--force                    # Force reassignment if file claimed
```

#### `epic update` Parameters
```bash
<epic-key>                 # Positional arg (required)
--title <string>           # New title
--description <string>     # New description
--status <string>          # New status: draft, active, completed, archived
--priority <string>        # New priority: low, medium, high
--business-value <string>  # New business value
--key <string>             # New key (must be unique)
--filename <string>        # New file path
--path <string>            # New custom folder base path
--force                    # Force reassignment if file claimed
```

**Missing in create:**
- ❌ `--status` - Not needed (defaults to draft)
- ❌ `--order` - **MISSING** (would be useful for epic ordering)
- ❌ `--notes` - **MISSING** (would be useful for creation context)

**Available in create but not update:**
- ✅ None (good - create is subset of update)

---

### Feature Commands

#### `feature create` Parameters
```bash
<title>                    # Positional arg (required)
--epic <string>            # Epic key (required)
--description <string>     # Feature description
--execution-order <int>    # Execution order (0 = not set)
--path <string>            # Custom folder base path
--filename <string>        # Custom file path
--key <string>             # Custom key
--force                    # Force reassignment
```

#### `feature update` Parameters
```bash
<feature-key>              # Positional arg (required)
--title <string>           # New title
--description <string>     # New description
--status <string>          # New status: draft, active, completed, archived
--execution-order <int>    # New execution order (-1 = no change)
--key <string>             # New key
--filename <string>        # New file path
--path <string>            # New custom folder base path
--force                    # Force reassignment
```

**Missing in create:**
- ❌ `--status` - Not needed (defaults to draft)
- ❌ `--notes` - **MISSING** (would be useful for creation context)

**Available in create but not update:**
- ✅ `--epic` - Not applicable to update (epic association is immutable after creation)

---

### Task Commands

#### `task create` Parameters
```bash
<title>                    # Positional arg (required)
--epic <string>            # Epic key (required)
--feature <string>         # Feature key (required)
--agent <string>           # Agent type
--template <string>        # Custom template path
--description <string>     # Description
--priority <int>           # Priority (1=highest, 10=lowest, default=5)
--depends-on <string>      # Comma-separated dependency keys
--execution-order <int>    # Execution order (0 = not set)
--order <int>              # Alias for --execution-order
--key <string>             # Custom key
--filename <string>        # Custom filename path
--force                    # Force reassignment
```

#### `task update` Parameters
```bash
<task-key>                 # Positional arg (required)
--title <string>           # New title
--description <string>     # New description
--priority <int>           # New priority (-1 = no change)
--agent <string>           # New agent type
--key <string>             # New key
--filename <string>        # New file path
--depends-on <string>      # New dependencies
--order <int>              # New execution order (-1 = no change)
--status <string>          # New status (uses workflow validation)
--force                    # Force reassignment or bypass validation
```

**Missing in create:**
- ❌ `--status` - Not needed (defaults to todo)
- ❌ `--notes` - **COULD BE ADDED** (but less critical for tasks)

**Available in create but not update:**
- ✅ `--epic` - Not applicable (epic derived from feature, immutable)
- ✅ `--feature` - Not applicable (immutable after creation)
- ✅ `--template` - Not applicable to update
- ✅ `--execution-order` - **REDUNDANT** (duplicate of --order)

---

## Identified Inconsistencies

### 1. Epic Create Missing Parameters

**Issue**: Epic create lacks `--order` and `--notes` that would be useful

**Impact**:
- No way to set epic ordering at creation time
- No way to document creation context/rationale

**Recommendation**: Add optional flags
```bash
epic create "Title" --order <int> --notes <string>
```

---

### 2. Feature Create Missing Parameters

**Issue**: Feature create lacks `--notes` flag

**Impact**:
- No way to document why feature was created
- Missing context for later reference

**Recommendation**: Add optional flag
```bash
feature create --epic=E07 "Title" --notes <string>
```

---

### 3. Task Parameter Duplication

**Issue**: Both `--execution-order` and `--order` exist as aliases

**Impact**:
- Confusing for users (which one to use?)
- Code duplication and maintenance burden

**Recommendation**:
- Keep `--order` as primary flag (shorter, clearer)
- Deprecate `--execution-order` with warning
- Update documentation

---

### 4. Status Flag Asymmetry

**Issue**: Update commands have `--status` but create commands don't

**Impact**: None (acceptable - status has sensible defaults)

**Recommendation**:
- Keep current behavior (create defaults to draft/todo)
- Document this design decision
- **NOT A BUG** - this is intentional

---

## Proposed Changes

### Phase 1: Add Missing Parameters (Required)

#### Epic Create
```go
epicCreateCmd.Flags().Int("order", 0, "Epic ordering (optional, 0 = not set)")
epicCreateCmd.Flags().String("notes", "", "Creation notes (optional)")
```

#### Feature Create
```go
featureCreateCmd.Flags().String("notes", "", "Creation notes (optional)")
```

#### Epic Update (for consistency)
```go
epicUpdateCmd.Flags().Int("order", -1, "New epic order (-1 = no change)")
epicUpdateCmd.Flags().String("notes", "", "Update notes (optional)")
```

#### Feature Update (for consistency)
```go
featureUpdateCmd.Flags().String("notes", "", "Update notes (optional)")
```

---

### Phase 2: Refactor Parameter Handling (Optional)

Consider creating shared parameter definition modules:

```go
// internal/cli/params/common.go
package params

type CommonParams struct {
    Title       string
    Description string
    Priority    int
    Order       int
    Notes       string
    Force       bool
}

type FileParams struct {
    Filename string
    Path     string
}

// Register common params on a command
func RegisterCommonParams(cmd *cobra.Command, params *CommonParams) {
    cmd.Flags().StringVar(&params.Title, "title", "", "Title")
    cmd.Flags().StringVar(&params.Description, "description", "", "Description")
    // ... etc
}
```

**Benefits**:
- DRY: Define once, use everywhere
- Consistency: All commands use same param names
- Maintainability: Single source of truth

**Risks**:
- Refactoring complexity
- Potential backwards compatibility issues
- May be overkill for current scope

**Recommendation**: Phase 2 is OPTIONAL and should be considered separately

---

## Database Schema Impact

### Epic Table
```sql
ALTER TABLE epics ADD COLUMN order_index INTEGER DEFAULT NULL;
ALTER TABLE epics ADD COLUMN creation_notes TEXT DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_epics_order ON epics(order_index);
```

### Feature Table
```sql
ALTER TABLE features ADD COLUMN creation_notes TEXT DEFAULT NULL;
```

### Notes on Implementation
- `order_index` separate from `execution_order` (epic vs feature/task ordering)
- `creation_notes` stored but not used in workflows (historical context only)
- Indexes improve query performance for ordered epic lists

---

## Migration Strategy

### Step 1: Database Migration
1. Add new columns with migrations
2. Add indexes for performance
3. Test backward compatibility

### Step 2: CLI Flag Addition
1. Add flags to create commands
2. Add flags to update commands (for symmetry)
3. Wire flags to repository methods

### Step 3: Repository Updates
1. Add `CreateWithOrder` methods
2. Add `UpdateOrder` methods
3. Add `UpdateNotes` methods
4. Ensure transactions are atomic

### Step 4: Testing
1. Unit tests for new repository methods
2. Integration tests for CLI commands
3. Backward compatibility tests

### Step 5: Documentation
1. Update CLI reference
2. Update CLAUDE.md
3. Add migration guide

---

## Acceptance Criteria

### Epic Create
- [ ] `--order` flag available and functional
- [ ] `--notes` flag available and functional
- [ ] Order stored in database
- [ ] Notes stored in database
- [ ] Backward compatible (flags are optional)

### Epic Update
- [ ] `--order` flag available and functional
- [ ] `--notes` flag available and functional
- [ ] Can update order independently
- [ ] Can update notes independently

### Feature Create
- [ ] `--notes` flag available and functional
- [ ] Notes stored in database
- [ ] Backward compatible (flag is optional)

### Feature Update
- [ ] `--notes` flag available and functional
- [ ] Can update notes independently

### Testing
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Backward compatibility verified
- [ ] Database migration tested

### Documentation
- [ ] CLI reference updated
- [ ] CLAUDE.md updated
- [ ] Migration guide created

---

## Task Breakdown Estimate

| Task | Complexity | Estimate |
|------|-----------|----------|
| Database schema migration | M | 2-3 hours |
| Repository method updates | M | 2-3 hours |
| CLI flag additions (epic/feature) | S | 1-2 hours |
| Unit tests | M | 2-3 hours |
| Integration tests | M | 2-3 hours |
| Documentation updates | S | 1 hour |
| **Total** | **M** | **10-14 hours** |

**Complexity**: M (Medium) - Straightforward changes but touches multiple layers

---

## Risks & Mitigation

### Risk 1: Breaking Changes
**Probability**: Low
**Impact**: High
**Mitigation**: All new flags are optional; defaults preserve current behavior

### Risk 2: Database Migration Failures
**Probability**: Low
**Impact**: High
**Mitigation**: Test migrations on backup database first; use auto-migration system

### Risk 3: Inconsistent State
**Probability**: Medium
**Impact**: Medium
**Mitigation**: Use transactions; validate in repository layer

---

## Open Questions

1. **Should epic order be global or per-status?**
   - Recommendation: Global order (simpler, matches feature/task pattern)

2. **Should notes be append-only or replaceable?**
   - Recommendation: Replaceable for creation_notes (single field)
   - Use task history pattern if append-only notes needed later

3. **Should we add notes to task create?**
   - Recommendation: Not in this feature (task already has description)
   - Can revisit if user feedback indicates need

---

## References

- Feature PRD: `E07-F12-parameter-consistency-across-create-and-update-com/feature.md`
- Epic PRD: `E07-enhancements/epic.md`
- CLI Source: `internal/cli/commands/epic.go`, `feature.go`, `task.go`
- Repository Source: `internal/repository/epic_repository.go`, `feature_repository.go`

---

*Analysis completed*: 2026-01-01
