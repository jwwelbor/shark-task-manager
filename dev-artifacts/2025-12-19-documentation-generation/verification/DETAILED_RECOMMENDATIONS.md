# CLI_REFERENCE.md - Detailed Line-by-Line Recommendations

## How to Use This Document

Each recommendation includes:
- **Location:** Where in CLI_REFERENCE.md the change should be made
- **Current Text:** What's there now
- **Recommended Change:** What should replace it
- **Justification:** Why this change is needed
- **Implementation Level:** Code file that validates this change

---

## CRITICAL RECOMMENDATIONS

### Recommendation 1: Add Missing `shark config` Section

**Location:** Insert new section after "Global Flags" section (after line 673)

**Insert This Content:**

```markdown
## Configuration Commands

### `shark config show`

Display current configuration settings and file location.

**Flags:**
- `--patterns` - Show only pattern configuration

**Examples:**

```bash
# Show all configuration
shark config show

# Show pattern configuration only
shark config show --patterns

# JSON output
shark config show --json
```

### `shark config validate`

Validate the configuration file for errors.

**Examples:**

```bash
# Validate configuration
shark config validate
```

**Coming Soon:**
- `config validate-patterns` - Validate all patterns in configuration
- `config test-pattern` - Test a pattern against a string
- `config get-format` - Get generation format for entity type
```

**Justification:** The config command group exists in code but users have no way to discover it. This makes configuration management invisible.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/config.go` lines 18-29

---

### Recommendation 2: Add `--force` Flag to All Task State Transition Commands

**Location 1: `shark task start` command (after line 489)**

**Current Text:**
```markdown
### `shark task start <task-key>`

Start working on a task (todo → in_progress).

**Validations:**
- Current status must be `todo`
- Warns if dependencies incomplete

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task start <task-key>`

Start working on a task (todo → in_progress).

**Validations:**
- Current status must be `todo`
- Warns if dependencies incomplete

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Start task
shark task start T-E04-F06-001

# With agent identifier
shark task start T-E04-F06-001 --agent="ai-agent-001"

# Force start from non-todo status (admin override)
shark task start T-E04-F06-001 --force
```
```

**Location 2: `shark task complete` command (after line 510)**

**Current Text:**
```markdown
### `shark task complete <task-key>`

Mark task ready for review (in_progress → ready_for_review).

**Validations:**
- Current status must be `in_progress`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Completion notes

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task complete <task-key>`

Mark task ready for review (in_progress → ready_for_review).

**Validations:**
- Current status must be `in_progress`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Completion notes
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Mark ready for review
shark task complete T-E04-F06-001

# With notes
shark task complete T-E04-F06-001 --notes="All tests passing"

# Force complete from non-in_progress status (admin override)
shark task complete T-E04-F06-001 --force --notes="Force completed"
```
```

**Location 3: `shark task approve` command (after line 531)**

**Current Text:**
```markdown
### `shark task approve <task-key>`

Approve and complete task (ready_for_review → completed).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Approval notes

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task approve <task-key>`

Approve and complete task (ready_for_review → completed).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Approval notes
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Approve task
shark task approve T-E04-F06-001

# With approval notes
shark task approve T-E04-F06-001 --agent="reviewer-001" --notes="LGTM"

# Force approve from non-ready_for_review status (admin override)
shark task approve T-E04-F06-001 --force
```
```

**Location 4: `shark task reopen` command (after line 552)**

**Current Text:**
```markdown
### `shark task reopen <task-key>`

Reopen for rework (ready_for_review → in_progress).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Rework notes

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task reopen <task-key>`

Reopen for rework (ready_for_review → in_progress).

**Validations:**
- Current status must be `ready_for_review`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `-n, --notes <text>` - Rework notes
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Reopen task
shark task reopen T-E04-F06-001

# With rework reason
shark task reopen T-E04-F06-001 --notes="Need to add error handling"

# Force reopen from non-ready_for_review status (admin override)
shark task reopen T-E04-F06-001 --force
```
```

**Location 5: `shark task block` command (after line 573)**

**Current Text:**
```markdown
### `shark task block <task-key>`

Block a task (todo/in_progress → blocked).

**Validations:**
- Current status must be `todo` or `in_progress`

**Flags:**
- `-r, --reason <text>` - Reason for blocking (REQUIRED)
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task block <task-key>`

Block a task (todo/in_progress → blocked).

**Validations:**
- Current status must be `todo` or `in_progress`

**Flags:**
- `-r, --reason <text>` - Reason for blocking (REQUIRED)
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Block task with reason
shark task block T-E04-F06-001 --reason="Waiting for API design approval"

# Short form
shark task block T-E04-F06-001 -r "Missing credentials"

# Force block from any status (admin override)
shark task block T-E04-F06-001 --force -r "Admin blocked"
```
```

**Location 6: `shark task unblock` command (after line 590)**

**Current Text:**
```markdown
### `shark task unblock <task-key>`

Unblock a task (blocked → todo).

**Validations:**
- Current status must be `blocked`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)

**Examples:**
```

**Recommended Change:**
```markdown
### `shark task unblock <task-key>`

Unblock a task (blocked → todo).

**Validations:**
- Current status must be `blocked`

**Flags:**
- `--agent <identifier>` - Agent identifier (defaults to `$USER`)
- `--force` - Force transition from any status (bypasses validation, use with caution)

**Examples:**
```bash
# Unblock task
shark task unblock T-E04-F06-001

# Force unblock from any status (admin override)
shark task unblock T-E04-F06-001 --force
```
```

**Justification:** The `--force` flag is critical for administrative workflows and bypassing blocked state machines, but it's completely undocumented. This is a significant usability and discoverability issue.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` lines 1113-1129

---

### Recommendation 3: Expand Sync Command Documentation

**Location:** Replace entire Sync Commands section (lines 592-637)

**Current Content:**
```markdown
## Sync Commands

### `shark sync`

Synchronize task files with database bidirectionally.

**Important:** Status is managed EXCLUSIVELY in the database and is NOT synced from files.

**Flags:**
- `--folder <path>` - Sync specific folder (default: `docs/plan`)
- `--dry-run` - Preview changes without applying
- `--strategy <strategy>` - Conflict resolution: `file-wins`, `database-wins`, `newer-wins` (default: `file-wins`)
- `--create-missing` - Auto-create missing epics/features from files
- `--cleanup` - Delete orphaned database tasks (files deleted)
- `--pattern <type>` - File patterns to scan: `task`, `prp` (can specify multiple, default: `task`)

**Examples:**

```bash
# Sync all (task files only)
shark sync

# Preview changes
shark sync --dry-run

# Sync PRP files only
shark sync --pattern=prp

# Sync both task and PRP files
shark sync --pattern=task --pattern=prp

# Sync specific folder
shark sync --folder=docs/plan/E04-task-mgmt-cli-core

# Database overrides files
shark sync --strategy=database-wins

# Create missing epics/features
shark sync --create-missing

# Delete orphaned tasks
shark sync --cleanup

# JSON output
shark sync --json
```

**Use cases:**
- After `git pull` or `git checkout`
- After manual file edits
- To clean up after deleting files
- To import tasks from file system
```

**Recommended Change:**
```markdown
## Sync Commands

### `shark sync`

Synchronize task files with database bidirectionally.

**Important:** Status is managed EXCLUSIVELY in the database and is NOT synced from files.

**Flags:**

**Core Flags:**
- `--folder <path>` - Sync specific folder (default: `docs/plan`)
- `--dry-run` - Preview changes without applying them

**Conflict Resolution:**
- `--strategy <strategy>` - Conflict resolution strategy: `file-wins`, `database-wins`, `newer-wins`, `manual` (default: `file-wins`)
  - `file-wins`: File changes override database
  - `database-wins`: Database changes override files
  - `newer-wins`: Whichever was modified most recently wins
  - `manual`: Prompt for each conflict

**File Management:**
- `--pattern <type>` - File patterns to scan: `task`, `prp` (can specify multiple, default: `task`)
- `--create-missing` - Auto-create missing epics/features from files
- `--cleanup` - Delete orphaned database tasks (files deleted)

**Discovery & Indexing:**
- `--index` - Enable discovery mode (parse epic-index.md)
- `--discovery-strategy <strategy>` - Discovery strategy: `index-only`, `folder-only`, `merge` (default: `merge`)
- `--validation-level <level>` - Validation strictness: `strict`, `balanced`, `permissive` (default: `balanced`)

**Performance & Output:**
- `--force-full-scan` - Force full scan ignoring incremental filtering
- `--output <format>` - Output format: `text`, `json` (default: `text`)
- `--quiet` - Quiet mode (only show errors)

**Examples:**

```bash
# Basic operations
shark sync                           # Sync all (task pattern)
shark sync --dry-run                 # Preview changes
shark sync --json                    # JSON output

# File patterns
shark sync --pattern=prp             # Sync PRP files only
shark sync --pattern=task --pattern=prp  # Sync both types

# Conflict resolution
shark sync --strategy=database-wins  # Database overrides files
shark sync --strategy=manual         # Prompt for each conflict
shark sync --strategy=newer-wins     # Whichever is newer wins

# Folder-specific
shark sync --folder=docs/plan/E04    # Sync specific epic folder
shark sync --folder=docs/plan/E04/E04-F06  # Sync specific feature

# Advanced discovery
shark sync --index                   # Enable epic-index.md discovery
shark sync --index --discovery-strategy=index-only  # Index-only mode
shark sync --index --discovery-strategy=merge --validation-level=permissive

# Cleanup operations
shark sync --create-missing          # Auto-create missing structures
shark sync --cleanup                 # Delete orphaned tasks
shark sync --force-full-scan         # Force full scan (skip incremental)

# Quiet mode for scripting
shark sync --quiet                   # Only output errors
shark sync --output=json --quiet     # JSON with minimal output
```

**Use Cases:**
- After `git pull` or `git checkout` - Use `shark sync` to sync changes
- After manual file edits - Use `shark sync --dry-run` first to preview
- To clean up after deleting files - Use `shark sync --cleanup`
- To import tasks from file system - Use `shark sync --create-missing`
- For advanced workflows - Use `--index` with discovery strategies
- For CI/automation - Use `--quiet --output=json`
```

**Justification:** The sync command has six important flags that are completely undocumented, making powerful features invisible to users. The manual strategy option is also missing.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/sync.go` lines 83-110

---

### Recommendation 4: Fix Agent Type Documentation

**Location:** Line 306 in task create section

**Current Text:**
```
- `-a, --agent <type>` - Agent type: `frontend`, `backend`, `api`, `testing`, `devops`, `general`
```

**Recommended Change:**
```
- `-a, --agent <type>` - Agent type (optional, accepts any string value - examples: `frontend`, `backend`, `api`, `testing`, `devops`, `general`)
```

**Justification:** The code accepts ANY string value for agent type, not just the listed options. The current documentation is misleading.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` line 1098

---

## HIGH PRIORITY RECOMMENDATIONS

### Recommendation 5: Document `--title` Flag for Task Create

**Location:** Insert into Task Create section (after line 308, before Dependencies)

**Insert This Content:**

```markdown
**Alternative to Positional Argument:**
- `--title <text>` - Task title (alternative to positional argument)

Note: You can provide the title either as a positional argument or via the `--title` flag.
```

**Update Examples Section to Include:**

```bash
# Using --title flag instead of positional
shark task create --title="Build login form" --epic=E01 --feature=F02 --agent=frontend

# Combining positional and flag styles
shark task create "User validation" --epic=E01 --feature=F02 --title="Will use positional title"
```

**Justification:** The `--title` flag option is not documented but is implemented, so users won't discover this alternative syntax.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` lines 550-555

---

### Recommendation 6: Document `--execution-order` Flag for Feature Create

**Location:** Feature Create section (after line 221)

**Current Text:**
```
**Flags:**
- `--epic <key>` - Epic key (required, e.g., `E01`)
- `--description <text>` - Feature description (optional)
```

**Recommended Change:**
```
**Flags (required):**
- `--epic <key>` - Epic key (required, e.g., `E01`)

**Flags (optional):**
- `--description <text>` - Feature description
- `--execution-order <n>` - Execution order for feature sequencing (0 = not set)
```

**Update Examples Section to Include:**

```bash
# With execution order
shark feature create --epic=E01 "Authentication" --execution-order=1

# Features can be ordered for sequential execution
shark feature create --epic=E01 "Database Setup" --execution-order=1
shark feature create --epic=E01 "API Endpoints" --execution-order=2
shark feature create --epic=E01 "Testing" --execution-order=3
```

**Justification:** The execution-order flag is implemented and important for feature sequencing, but users cannot discover it from documentation.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go` lines 121-122

---

### Recommendation 7: Fix Task List Feature Filter

**Location:** Task List Flags section (line 373)

**Current Text:**
```
- `-f, --feature <key>` - Filter by feature key
```

**Action Required:** VERIFY if feature filter is actually implemented in the task list command.

**What to Do:**
1. If implemented: Verify it works correctly
2. If NOT implemented: Remove this flag from documentation AND from code

**Code Status:** Flag is defined in code (task.go line 1087) but appears NOT to be used in the FilterCombined method.

**Recommendation:** Either implement the feature filter properly or remove it from both code and documentation. As-is, it creates confusion.

**Code Reference:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` lines 1085-1091

---

## MEDIUM PRIORITY RECOMMENDATIONS

### Recommendation 8: Fix Example Command Prefixes

**Locations to Fix:**
1. epic.go help text (lines 40, 41) - Change `pm` to `shark`
2. feature.go help text (line 34, 35) - Change `pm` to `shark`
3. task.go help text (line 26, 27) - Change `pm` to `shark`

**Why:** Consistency with the main documentation and actual command name.

---

### Recommendation 9: Document Feature List "Order" Column

**Location:** Feature List section (after line 259)

**Insert This Content:**

```markdown
The table output includes an "Order" column showing the `execution_order` value if set (displays "-" if not set). This can be used to sequence feature work within an epic.
```

**Justification:** The "Order" column appears in output but users won't know what it means.

---

### Recommendation 10: Clarify Epic Status Command Status

**Location:** Insert after Epic Commands section (after line 209)

**Insert This Content:**

```markdown
### `shark epic status` (Coming Soon)

Show epic status summary with completion percentages and task counts.

**Status:** This command is planned for E05-F01 (Status Dashboard) and is not yet implemented.

**Example:**
```bash
# Will display summary of all epics when implemented
shark epic status
```
```

**Justification:** The command exists in code but is marked TODO, creating confusion.

---

## Summary Table of All Changes

| Priority | Recommendation | Effort | Impact | Type |
|---|---|---|---|---|
| 1 | Add config command section | 1-2 hrs | HIGH | New Content |
| 1 | Add --force flags (6 commands) | 1 hr | HIGH | New Content |
| 1 | Expand sync command docs | 1-2 hrs | HIGH | Enhanced Content |
| 1 | Fix agent type docs | 15 mins | HIGH | Correction |
| 2 | Document --title flag | 30 mins | MEDIUM | New Content |
| 2 | Document --execution-order | 30 mins | MEDIUM | New Content |
| 2 | Fix task list feature filter | 1 hr | MEDIUM | Code/Docs Fix |
| 3 | Fix example prefixes | 15 mins | LOW | Consistency |
| 3 | Document "Order" column | 15 mins | LOW | Clarification |
| 3 | Clarify epic status command | 15 mins | LOW | Clarification |

**Total Estimated Effort:** 5-7 hours

**Recommended Completion:** Before v1.1.0 release

---

## Implementation Checklist

- [ ] Add config command section
- [ ] Add --force flag to task start
- [ ] Add --force flag to task complete
- [ ] Add --force flag to task approve
- [ ] Add --force flag to task block
- [ ] Add --force flag to task unblock
- [ ] Add --force flag to task reopen
- [ ] Expand sync command documentation with all flags
- [ ] Fix agent type to indicate "any string"
- [ ] Document task create --title flag
- [ ] Document feature create --execution-order flag
- [ ] Verify/fix task list feature filter
- [ ] Fix example command prefixes (pm → shark)
- [ ] Document feature list "Order" column
- [ ] Clarify epic status command status
- [ ] Final review and testing

---

## Testing After Updates

After implementing these changes:

1. **Cross-reference check:** Run `shark --help` for each command and verify docs match
2. **Flag validation:** Test each documented flag actually works
3. **Example verification:** Run each example command and verify output
4. **Link validation:** Ensure all references are accurate
5. **Consistency check:** Verify terminology is consistent throughout

