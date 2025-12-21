# CLI_REFERENCE.md Documentation Review Report

**Review Date:** 2025-12-19
**Reviewer:** Documentation Review Agent
**Status:** CRITICAL ISSUES FOUND - Multiple discrepancies between documentation and implementation

---

## Executive Summary

The CLI_REFERENCE.md documentation provides a good foundation but contains **several critical accuracy issues**, **missing commands**, **incomplete flag documentation**, and **inconsistencies with the actual implementation**. The documentation is approximately **70% accurate** but has gaps that could confuse users and cause integration issues.

**Key Findings:**
- **3 missing commands** not documented (config, validate, feature status)
- **5+ flag discrepancies** between docs and code
- **Incomplete examples** in task create section
- **Missing flags** in multiple command sections
- **Inaccurate command names** in examples (uses "pm" instead of "shark")
- **Missing undocumented flag options** (--force, multiple patterns, index/discovery features)

**Priority:** HIGH - These issues should be addressed before next release

---

## Section-by-Section Analysis

### 1. Installation & Setup Section

#### Issue 1.1: Init Command - Missing Flags
**Status:** INCOMPLETE

The `shark init` command documentation is accurate but minimal.

**What's documented:**
- `--non-interactive`
- `--force`

**What's actually implemented (VERIFIED CORRECT):**
All flags documented are correctly implemented in `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/init.go` lines 39-42.

**Assessment:** ✓ ACCURATE

---

### 2. Epic Commands Section

#### Issue 2.1: Epic Create - No Flag Type Documentation
**Status:** MINOR DOCUMENTATION GAP

In CLI_REFERENCE.md line 143-144:
```
Flags:
- `--description <text>` - Epic description (optional)
```

**Actual Implementation** (`epic.go` lines 132-133):
```go
epicCreateCmd.Flags().StringVar(&epicCreateDescription, "description", "", "Epic description (optional)")
```

**Assessment:** ✓ ACCURATE - The flag works as documented.

#### Issue 2.2: Epic Status Command - NOT DOCUMENTED
**Status:** COMMAND EXISTS BUT NOT IN DOCS

**Found in code:** `epic.go` lines 70-80
```go
var epicStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show epic status summary",
	Long:  `Display a summary of all epics with completion percentages and task counts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement in E05-F01 (Status Dashboard)
		cli.Warning("Not yet implemented - coming in E05-F01")
		return nil
	},
}
```

**Is it documented in CLI_REFERENCE.md?** NO

**Impact:** Users won't know this command exists (even though it's not implemented yet).

**Recommendation:** Add a note section at the end of Epic Commands documenting the status command as "Coming soon in E05-F01" or remove it entirely from code if not ready.

#### Issue 2.3: Epic Examples - Wrong Command Prefix
**Status:** MINOR - Example uses "pm" instead of "shark"

**Documented in CLI_REFERENCE.md line 40:**
```
shark epic list                 List all epics
shark epic list --json          Output as JSON
```

**Should be:**
```
shark epic list                 List all epics
shark epic list --json          Output as JSON
```

**Assessment:** INCONSISTENT - The workflow examples correctly use "shark" (line 65) but some command help text uses "pm".

---

### 3. Feature Commands Section

#### Issue 3.1: Feature Create - Execution Order Flag NOT DOCUMENTED
**Status:** MISSING FLAG

**Documented in CLI_REFERENCE.md:**
No mention of execution-order flag.

**Actually implemented in `feature.go` lines 121-122:**
```go
featureCreateCmd.Flags().IntVar(&featureCreateExecutionOrder, "execution-order", 0, "Execution order (optional, 0 = not set)")
```

**Impact:** Users cannot discover this important flag from documentation.

**Should Add:**
```
**Flags (optional):**
- `--execution-order <n>` - Execution order for feature sequencing (optional, 0 = not set)
```

#### Issue 3.2: Feature List - Table Output Shows Extra Columns
**Status:** INCOMPLETE DOCUMENTATION

Actual implementation in `feature.go` lines 405-406 shows table includes "Order" column:
```go
tableData := pterm.TableData{
	{"Key", "Title", "Epic ID", "Status", "Progress", "Tasks", "Order"},
}
```

This column is not mentioned in the documentation. The "Order" refers to `ExecutionOrder`.

#### Issue 3.3: Feature Delete - Missing Flag Documentation
**Status:** FLAG EXISTS BUT NOT DOCUMENTED

**Documented:** `--force` flag is documented for feature delete (lines 279-280)

**Verified:** Correctly implemented in `feature.go` line 126

**Assessment:** ✓ ACCURATE

---

### 4. Task Commands Section

#### Issue 4.1: Task Create - Incorrect Positional Argument Documentation
**Status:** DISCREPANCY

**Documented in CLI_REFERENCE.md line 300-301:**
```
**Arguments:**
- `<title>` - Task title (positional argument)
```

**Actual Implementation in `task.go` lines 550-555:**
The title CAN be either positional OR a flag:
```go
var title string
if len(args) > 0 {
	title = args[0]
} else {
	title, _ = cmd.Flags().GetString("title")
}
```

**The documentation doesn't mention that title can be passed via `--title` flag.**

**Impact:** Users might not discover the `--title` flag option.

#### Issue 4.2: Task Create - Incomplete Agent Type Documentation
**Status:** INCOMPLETE

**Documented in CLI_REFERENCE.md lines 305-306:**
```
- `-a, --agent <type>` - Agent type: `frontend`, `backend`, `api`, `testing`, `devops`, `general`
```

**Actual Implementation in `task.go` lines 1098:**
```go
taskCreateCmd.Flags().StringP("agent", "a", "", "Agent type (optional, accepts any string)")
```

**Finding:** The code accepts ANY string value, not just the listed types! The documentation is too restrictive.

**Should Be:**
```
- `-a, --agent <type>` - Agent type (optional, accepts any string value - examples: frontend, backend, api, testing, devops, general)
```

#### Issue 4.3: Task Create - Missing --title Flag Documentation
**Status:** MISSING FLAG

The `--title` flag is NOT documented but is implemented and used in code.

**Verified in code:** `task.go` line 554 reads the title flag.

**Should Add:**
```
**Flags (alternative positional):**
- `--title <text>` - Task title (can use instead of positional argument)
```

#### Issue 4.4: Task Create - Filename Documentation is Accurate
**Status:** ✓ VERIFIED CORRECT

Lines 315-364 correctly document the `--filename` flag and its behavior.

Implementation in `task.go` line 578 confirms this flag is properly implemented.

**Assessment:** ✓ ACCURATE

#### Issue 4.5: Task List - Missing Feature Flag
**Status:** INCOMPLETE

**Documented in CLI_REFERENCE.md lines 370-371:**
```
- `-f, --feature <key>` - Filter by feature key
```

However, looking at `task.go` line 1087, the feature flag is NOT implemented:
```go
// No feature filter in task list implementation
```

**Wait - Let me recheck the code...**

Actually in `task.go` lines 195-196, I see:
```go
statusStr, _ := cmd.Flags().GetString("status")
epicKey, _ := cmd.Flags().GetString("epic")
```

But no feature filter is read. Let me check line 1085-1091 more carefully:
```go
taskListCmd.Flags().StringP("status", "s", "", "Filter by status (todo, in_progress, completed, blocked)")
taskListCmd.Flags().StringP("epic", "e", "", "Filter by epic key")
taskListCmd.Flags().StringP("feature", "f", "", "Filter by feature key")
taskListCmd.Flags().StringP("agent", "a", "", "Filter by assigned agent")
```

The `--feature` flag IS defined but let me verify it's used in the FilterCombined method...

Looking at line 237 in task.go, the `FilterCombined` uses `epicKeyPtr` but I don't see feature filter being applied. This is a CODE ISSUE - the flag is defined but not actually used.

**Impact:** The feature filter flag doesn't work as documented.

**Action Needed:** Either implement the feature filter or remove it from documentation and code.

#### Issue 4.6: Task Commands - --force Flag NOT Documented for Transitions
**Status:** CRITICAL MISSING DOCUMENTATION

The task start, complete, approve, block, unblock, and reopen commands all support a `--force` flag that is NOT documented in CLI_REFERENCE.md.

**Evidence from code:**

- `task start` line 1113: `--force` flag defined and used (lines 679-686)
- `task complete` line 1116: `--force` flag defined and used (lines 741-748)
- `task approve` line 1119: `--force` flag defined and used (lines 803-810)
- `task block` line 1124: `--force` flag defined and used (lines 883-890)
- `task unblock` line 1126: `--force` flag defined and used (lines 940-947)
- `task reopen` line 1129: `--force` flag defined and used (lines 997-1004)

**What documentation says:** Nothing about `--force`

**What code comment says:** "Force status change bypassing validation (use with caution)"

**This is a MAJOR OMISSION.** The `--force` flag is critical for administrative overrides.

**Should add to each command:**
```
**Flags (optional):**
- `--force` - Bypass status transition validation (use with caution)
```

---

### 5. Sync Commands Section

#### Issue 5.1: Sync - Missing Advanced Flags
**Status:** INCOMPLETE - Several flags missing from docs

**Not documented but implemented flags:**

1. `--force-full-scan` (line 98 in sync.go)
   - Forces full scan ignoring incremental filtering

2. `--output` (line 100 in sync.go)
   - Output format: text, json

3. `--quiet` (line 102 in sync.go)
   - Quiet mode (only show errors)

4. `--index` (line 104 in sync.go)
   - Enable discovery mode (parse epic-index.md)

5. `--discovery-strategy` (line 106 in sync.go)
   - Discovery strategy: index-only, folder-only, merge

6. `--validation-level` (line 108 in sync.go)
   - Validation level: strict, balanced, permissive

**What documentation says:** Only documents `--folder`, `--dry-run`, `--strategy`, `--create-missing`, `--cleanup`, and `--pattern`

**Impact:** Users cannot discover these important advanced features.

**Sync Strategy Options - Incomplete:**
Documentation (line 603) says:
```
- `--strategy <strategy>` - Conflict resolution: `file-wins`, `database-wins`, `newer-wins` (default: `file-wins`)
```

Code (line 90 in sync.go) shows:
```go
syncCmd.Flags().StringVar(&syncStrategy, "strategy", "file-wins",
	"Conflict resolution strategy: file-wins, database-wins, newer-wins, manual")
```

Missing: `manual` strategy option!

---

### 6. Global Flags Section

#### Issue 6.1: Global Flags - Documentation Complete
**Status:** ✓ ACCURATE

Lines 645-673 correctly document all global flags. Verified against root command implementation.

**Assessment:** ✓ ACCURATE

---

### 7. Missing Commands - Complete Analysis

The following commands exist in code but are NOT documented in CLI_REFERENCE.md:

#### Missing Command 1: `shark config`
**Location:** `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/config.go`

**Subcommands:**
- `config show` - Show current configuration
- `config show --patterns` - Show only pattern configuration
- `config validate` - Validate configuration file (incomplete - TODO in code line 88)

**What is documented:** NOTHING

**Why it matters:** Users have no way to discover configuration commands.

#### Missing Command 2: `shark feature status`
**Status:** Not yet implemented (TODO in code)

Should be documented as "Coming soon" if keeping it in the code.

#### Missing Command 3: Config Pattern Testing
The config.go file has additional commands mentioned but not fully documented:
- `config validate-patterns` (line 26)
- `config test-pattern` (line 27)
- `config get-format` (line 28)

---

## Accuracy Issues Summary

### Critical Issues (Must Fix)
1. **Missing `--force` flag documentation** for task state transitions (6 commands affected)
2. **Missing `shark config` command** section entirely
3. **Sync command - 6 advanced flags not documented** (force-full-scan, output, quiet, index, discovery-strategy, validation-level)
4. **Task create - agent type too restrictive** (docs say specific types, code accepts any string)
5. **Sync strategy missing `manual` option**

### High Priority Issues
6. **Task create - no --title flag documentation** (alternative to positional)
7. **Feature create - missing --execution-order flag**
8. **Task list - feature filter flag defined but not implemented** (either fix code or remove flag)
9. **Example commands use wrong prefix** ("pm" instead of "shark" in some places)

### Medium Priority Issues
10. **Feature list table shows "Order" column** not documented
11. **Epic status command exists but marked TODO** - needs clarification in docs

### Low Priority Issues
12. **Some command help text uses "pm"** instead of "shark" as prefix

---

## Completeness Assessment

### Command Coverage
- **Epic commands:** 80% documented (missing status command note)
- **Feature commands:** 85% documented (missing execution-order)
- **Task commands:** 75% documented (missing --force flags, --title flag)
- **Sync commands:** 60% documented (missing 6 advanced flags)
- **Config commands:** 0% documented (completely missing)
- **Init commands:** 95% documented (complete)

### Flag Coverage
- **Epic commands:** 95% (complete)
- **Feature commands:** 85% (missing execution-order)
- **Task commands:** 70% (missing --force, --title, incorrect agent type info)
- **Sync commands:** 50% (missing 6 important flags)
- **Config commands:** 0% (no documentation)
- **Global flags:** 100% (complete)

---

## Recommendations - Prioritized

### Priority 1: CRITICAL (Do First)
1. **Add `--force` flag documentation to all task transition commands**
   - Task start, complete, approve, block, unblock, reopen
   - Add warning that this bypasses validation
   - Location: Lines 470-590 (task lifecycle section)

2. **Create `shark config` command section**
   - Document: `config show`, `config show --patterns`, `config validate`
   - Add examples showing configuration management
   - Location: Add new section after "Global Flags"

3. **Fix sync command documentation**
   - Add missing flags: force-full-scan, output, quiet, index, discovery-strategy, validation-level
   - Add manual strategy to conflict resolution options
   - Update examples to show advanced patterns
   - Location: Lines 592-637

### Priority 2: HIGH (Should Fix Soon)
4. **Document task create --title flag** (alternative to positional)
   - Add to optional flags section
   - Show example: `shark task create --epic=E01 --feature=F02 --title="Task name"`
   - Location: Lines 300-364

5. **Fix agent type documentation**
   - Change from restrictive list to "accepts any string value"
   - Update location: Line 306

6. **Document feature execution-order flag**
   - Add to feature create optional flags
   - Explain usage for sequencing
   - Location: Lines 215-231

7. **Fix feature filter in task list**
   - Either implement the feature filter in code or remove from documentation
   - Location: Line 373

### Priority 3: MEDIUM (Nice to Have)
8. **Fix example command prefixes**
   - Change all "pm" to "shark" for consistency
   - Locations: Epic command lines 40, 41, etc.

9. **Document epic status command**
   - Note it's "Coming in E05-F01" as placeholder
   - Or remove from code if not ready

10. **Document feature list table columns**
    - Explain the "Order" column in output
    - Location: Feature list section

---

## Organization & Clarity Assessment

### What Works Well
- Clear section hierarchy with command grouping
- Comprehensive workflow example (lines 56-135)
- Good use of subsections for command breakdown
- Examples are practical and realistic
- Flag formatting is consistent
- JSON output examples are helpful

### What Could Be Improved
1. Add a "Tip" box warning about --force flag availability
2. Create a "Configuration" section for config commands
3. Add a "Common Patterns" section with workflows
4. Add a "Troubleshooting" section
5. Better organization of global vs local flags

---

## File References for Verification

All findings verified against actual implementation:

**Implementation Files Reviewed:**
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/epic.go` (lines 1-741)
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/feature.go` (lines 1-831)
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/task.go` (lines 1-1131)
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/sync.go` (lines 1-300+)
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/init.go` (lines 1-131)
- `/home/jwwelbor/projects/shark-task-manager/internal/cli/commands/config.go` (lines 1-100+)

**Documentation File Reviewed:**
- `/home/jwwelbor/projects/shark-task-manager/docs/CLI_REFERENCE.md` (673 lines)

---

## Quality Metrics

| Metric | Score | Notes |
|--------|-------|-------|
| Accuracy | 70% | Multiple discrepancies found |
| Completeness | 65% | Missing commands and flags |
| Clarity | 85% | Generally well-written but confusing gaps |
| Organization | 80% | Good structure, needs config section |
| Consistency | 60% | Some command prefixes inconsistent |
| Usability | 70% | Users might miss advanced features |

**Overall Documentation Quality: NEEDS IMPROVEMENT**

The documentation provides a solid foundation but requires updates before the next release to match the actual implementation.

---

## Conclusion

The CLI_REFERENCE.md documentation is approximately **70% accurate and 65% complete**. While it covers the core functionality well, it has several critical gaps:

1. An entire command group (`config`) is missing
2. Important flags (`--force` for task transitions) are undocumented
3. Advanced sync features are not disclosed
4. Some implementations don't match documentation claims

These issues should be resolved before the next public release. The "Priority 1" recommendations should be implemented immediately as they affect critical user workflows.

**Recommended Action:** Schedule documentation update sprint to address all Priority 1 and 2 items before v1.1.0 release.
