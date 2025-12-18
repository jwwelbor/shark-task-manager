# Feature: Import Reporting & Validation

**Feature Key**: E06-F05-import-reporting-validation

---

## Epic

- [Epic PRD](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/epic.md)
- [Epic Requirements](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/requirements.md)

---

## Goal

### Problem

When Technical Leads import 200+ existing markdown files using the intelligent scanning system, they face a critical visibility gap. The scanner processes hundreds of files through multiple stages (discovery, pattern matching, parsing, validation, import), but without detailed reporting, users cannot:

- **Verify completeness**: Which files were successfully imported vs. skipped? Were all epics/features/tasks discovered?
- **Diagnose failures**: Why was a specific file skipped? What pattern mismatch or validation error occurred?
- **Trust results**: Are all database file paths valid? Are there orphaned records (features without epics, tasks without features)?
- **Fix issues efficiently**: What specific changes are needed to make skipped files importable?
- **Integrate programmatically**: Can AI agents parse scan results to verify success without human intervention?

Without comprehensive reporting and validation, users must manually inspect hundreds of files post-import to verify correctness, defeating the purpose of automated scanning. Errors discovered weeks later (broken file paths, missing relationships) erode trust in the system and require manual database cleanup.

### Solution

Implement a robust reporting and validation infrastructure that provides visibility into every stage of the scanning process and verifies database integrity post-import. The system will:

1. **Detailed Scan Reports**: Generate comprehensive reports showing files scanned, matched, and skipped with breakdown by type (epics, features, tasks, related docs). Each skipped file includes file path, reason for skip (pattern mismatch, validation failure, parse error), and suggested fix.

2. **Actionable Error Messages**: Provide context-rich errors with file path, line number (for parse errors), specific issue description, and concrete suggested fix. Follow REQ-NF-006 format: "Cannot parse frontmatter in {file_path}:{line}: {specific_error}. Suggestion: {fix}."

3. **Import Validation Command**: Implement `shark validate` command to verify database integrity after import. Check file path existence, epic/feature/task relationships, detect orphaned records, identify broken references, and suggest corrective actions.

4. **JSON Output Format**: Support `--output=json` flag for all scan and validation commands with documented schema. Enable AI agents and scripts to parse results programmatically for automated verification and error handling.

5. **Dry-Run Mode**: Implement `--dry-run` flag that executes entire scan workflow (discovery, parsing, validation) but skips database commit. Generate identical reports as real run to provide preview without database changes.

This infrastructure ensures users have complete visibility and trust in the scanning process while enabling programmatic integration for AI agents.

### Impact

- **Import Confidence**: Users can verify 100% of scanned files were processed correctly before committing to database, reducing post-import cleanup time from hours to minutes
- **Debugging Efficiency**: Reduce time to diagnose and fix scan failures from 30+ minutes (manual file inspection) to <5 minutes (read error message, apply suggested fix)
- **Database Integrity**: Detect and prevent orphaned records, broken file paths, and invalid relationships before they cause downstream issues in task queries and reports
- **Agent Integration**: Enable AI agents to verify scan success programmatically via JSON output, reducing manual verification overhead by 90%
- **User Trust**: Provide transparency into scanning process through detailed reports, increasing user confidence in automated import from 60% to 95%

---

## User Personas

### Primary Persona: Technical Lead (Existing Project Migration)

**Profile**:
- Technical Lead migrating existing multi-epic project to shark
- Importing 200-300 existing markdown files with varied conventions
- Needs to verify import completeness before trusting database as source of truth
- Risk-averse: must validate results before committing to new workflow
- Limited time for debugging: wants clear, actionable error messages

**Pain Points Addressed**:
- Cannot verify which files were imported without manually checking database vs. filesystem
- Scan errors are opaque: "import failed" messages don't explain why or how to fix
- No way to preview import results before committing to database
- Cannot detect orphaned records or broken relationships until queries fail weeks later
- Manual verification of 200+ files is impractical and error-prone

**Success Scenario**:
Technical Lead runs `shark scan --dry-run` and receives detailed report showing 245 of 265 files matched, with 20 skipped files listed with specific reasons (pattern mismatch, missing frontmatter, invalid epic key). They review warnings, fix 3 files with suggested changes, re-run dry-run to verify all 248 files will import, then execute real scan with confidence. Post-import, they run `shark validate` to confirm all file paths exist and no orphaned records present. Total time: 12 minutes instead of 2+ hours manual verification.

### Secondary Persona: AI Agent (Incremental Development Sync)

**Profile**:
- Autonomous code generation agent creating task files during development
- Runs `shark sync --incremental` at end of work session
- Needs to verify sync succeeded before ending session (next agent needs accurate state)
- Cannot interpret human-readable error messages: requires structured JSON output
- Optimizes for token efficiency: wants programmatic success/failure without reading verbose logs

**Pain Points Addressed**:
- Current sync provides human-readable output that's difficult to parse programmatically
- No structured way to detect which files failed to sync and why
- Cannot verify sync success without reading entire database (token-expensive)
- Error messages designed for humans, not machines (requires complex regex parsing)

**Success Scenario**:
AI Agent creates 3 new task files, runs `shark sync --incremental --output=json`, receives structured response: `{"status": "success", "scanned": 3, "imported": 3, "skipped": 0, "errors": []}`. Agent confirms sync succeeded, logs result, ends session. If sync had failures, JSON output provides file paths and error codes enabling programmatic error handling. Total tokens consumed: <500 vs. 5,000+ for verbose logs.

### Tertiary Persona: Product Manager (Multi-Style Documentation)

**Profile**:
- Product Manager tracking progress across epics with varied documentation styles
- Runs periodic scans to import new documentation from multiple teams
- Needs to verify special epic types (tech-debt, bugs) were correctly imported
- Uses scan reports to identify documentation quality issues across teams
- Shares validation results with stakeholders to demonstrate project tracking accuracy

**Pain Points Addressed**:
- No visibility into which special epic types were recognized vs. skipped
- Cannot generate reports showing documentation coverage (which features lack PRDs, which epics lack tasks)
- Manual verification of cross-team documentation is time-consuming
- No way to demonstrate to stakeholders that database accurately reflects reality

**Success Scenario**:
Product Manager runs `shark scan --execute` and receives report showing 5 standard epics (E01-E05) and 2 special epics (tech-debt, bugs) imported with breakdown of features and tasks per epic. Validation command confirms all 47 features have parent epics and all 183 tasks have parent features. They export JSON report to generate dashboard showing documentation coverage: 92% of features have PRDs, 78% have architecture docs, 100% have at least one task. Total time to generate coverage report: 3 minutes vs. 45+ minutes manual analysis.

---

## User Stories

### Must-Have Stories

**Story 1: Detailed Scan Report**
As a Technical Lead, I want to see a comprehensive report after running scan so that I can verify which files were imported and which were skipped with specific reasons.

**Acceptance Criteria**:
- Report shows total files scanned, matched, and skipped
- Breakdown by type: epics, features, tasks, related docs
- Each skipped file lists: file path, skip reason (pattern mismatch, validation failure, parse error), suggested fix
- Report includes scan metadata: timestamp, duration, validation level used, patterns applied
- Human-readable format for CLI output with clear visual hierarchy

**Story 2: Actionable Parse Errors**
As a Technical Lead, I want parse errors to include file path, line number, and suggested fix so that I can resolve issues without trial-and-error debugging.

**Acceptance Criteria**:
- Parse errors follow format: "Cannot parse {component} in {file_path}:{line}: {specific_error}. Suggestion: {fix}"
- File path is absolute and clickable in most terminal emulators
- Line number points to exact location of parse failure
- Suggested fix is concrete and actionable (example: "Add '---' on line 8 to close frontmatter block")
- Errors grouped by type (frontmatter errors, heading errors, metadata errors) for easier scanning

**Story 3: Import Validation Command**
As a Technical Lead, I want to run `shark validate` after import so that I can verify database integrity before relying on it as source of truth.

**Acceptance Criteria**:
- `shark validate` checks all file_path entries point to existing files
- Validates epic/feature/task foreign key relationships
- Detects orphaned records: features without epics, tasks without features
- Reports broken references with specific IDs and file paths
- Suggests corrective actions: re-scan, manual fix, or delete orphaned record
- Exit code 0 for success, non-zero for validation failures

**Story 4: JSON Output for Programmatic Processing**
As an AI Agent, I want `--output=json` flag so that I can parse scan results programmatically and verify success without reading verbose logs.

**Acceptance Criteria**:
- All scan commands support `--output=json` flag
- JSON schema is documented and versioned
- Output includes: status (success/failure), counts (scanned/matched/skipped), errors array with structured error objects
- Error objects include: file_path, error_type, line_number (if applicable), message, suggested_fix
- Valid JSON even when scan fails (no truncated output)

**Story 5: Dry-Run Mode**
As a Technical Lead, I want `--dry-run` flag so that I can preview import results without committing changes to database.

**Acceptance Criteria**:
- `--dry-run` executes full scan workflow: discovery, pattern matching, parsing, validation
- Skips database transaction commit (rolls back changes)
- Generates identical report as real run (same warnings, errors, counts)
- Clearly indicates dry-run mode in output: "DRY RUN: No database changes made"
- Can be combined with `--output=json` for programmatic preview

### Should-Have Stories

**Story 6: File Path Consistency Checks**
As a Technical Lead, I want validation to detect when database file_path doesn't point to actual file so that I can fix stale references before queries fail.

**Acceptance Criteria**:
- Validation iterates all epics, features, tasks and checks file_path exists on filesystem
- Reports missing files with: entity type, entity key, stale file_path
- Distinguishes between: file moved, file deleted, path typo
- Suggests corrective actions: update file_path if file moved, delete record if file deleted, re-scan if path typo
- Provides summary: "Found 3 broken file paths: 2 missing files, 1 incorrect path"

**Story 7: Orphaned Record Detection**
As a Technical Lead, I want validation to detect orphaned records so that I can maintain referential integrity and prevent query failures.

**Acceptance Criteria**:
- Detect features where epic_key doesn't match any epic in database
- Detect tasks where feature_key doesn't match any feature in database
- Report orphaned records with: entity type, entity key, missing parent key
- Suggest corrective actions: create missing parent, reassign to different parent, delete orphaned record
- Provides summary: "Found 2 orphaned features, 5 orphaned tasks"

**Story 8: Scan Progress Indicators**
As a Technical Lead, I want progress indicators during long scans so that I know the system is working and can estimate completion time.

**Acceptance Criteria**:
- Display progress for scans processing 50+ files
- Show: current phase (discovering, parsing, validating, importing), files processed, estimated time remaining
- Update progress without excessive output (every 10 files or 1 second, whichever is less frequent)
- Suppress progress in `--output=json` mode (only final JSON output)
- Disable progress with `--quiet` flag for scripting

**Story 9: Warning Severity Levels**
As a Technical Lead, I want warnings categorized by severity so that I can prioritize which issues to fix first.

**Acceptance Criteria**:
- Warnings have severity: ERROR (blocks import), WARNING (imports but may have issues), INFO (informational only)
- ERROR: pattern mismatch, validation failure, parse error (file skipped)
- WARNING: missing optional metadata, auto-generated key, inferred relationship (file imported)
- INFO: file matched, relationship established, key extracted (file imported successfully)
- Report summary shows counts by severity: "3 errors, 12 warnings, 245 info"

### Could-Have Stories

**Story 10: Validation Auto-Fix Mode**
As a Technical Lead, I want validation to optionally auto-fix simple issues so that I don't have to manually correct obvious problems.

**Acceptance Criteria**:
- `shark validate --fix` mode auto-corrects simple issues
- Auto-fix: update file_path if file moved (detected via filename match), delete orphaned records if parent missing, regenerate keys for invalid keys
- Requires confirmation before applying fixes (interactive prompt)
- Logs all auto-fixes applied: "Updated 3 file paths, deleted 2 orphaned tasks"
- Dry-run mode: `shark validate --fix --dry-run` shows what would be fixed

**Story 11: Diff Output for Dry-Run**
As a Technical Lead, I want dry-run to show diff of database changes so that I can see exactly what would be added/updated/deleted.

**Acceptance Criteria**:
- `--dry-run --diff` shows database changes: inserts (green), updates (yellow), deletes (red)
- Diff format: entity type, key, changed fields (old value → new value)
- Grouped by operation type for easy scanning
- Can be combined with `--output=json` for programmatic diff parsing

---

## Requirements

### Functional Requirements

#### Report Generation

**FR-001: Scan Result Summary**
The system must generate a scan result summary showing total files scanned, matched, and skipped with breakdown by entity type (epics, features, tasks, related docs).

**FR-002: Skipped File Details**
For each skipped file, the system must report: absolute file path, skip reason (pattern mismatch, validation failure, parse error, file too large), and suggested fix.

**FR-003: Scan Metadata**
The scan report must include metadata: scan start timestamp, scan duration, validation level used, patterns applied (epic/feature/task patterns from config), documentation root scanned.

**FR-004: Report Phases**
The scan report must reflect scan workflow phases: discovery phase (files found), pattern matching phase (files matched to entity types), parsing phase (metadata extracted), validation phase (integrity checks), import phase (database operations).

#### Error Messages

**FR-005: Structured Error Format**
All errors and warnings must follow structured format: `{severity}: Cannot {operation} in {file_path}:{line}: {specific_error}. Suggestion: {fix}`.

**FR-006: Absolute File Paths**
All file paths in errors, warnings, and reports must be absolute paths (not relative) to enable terminal emulator clickable links and unambiguous file identification.

**FR-007: Line Number Precision**
Parse errors must include line number pointing to exact location of failure. For frontmatter errors, line number points to invalid line. For markdown errors, line number points to problematic heading or section.

**FR-008: Actionable Suggestions**
Each error must include concrete, actionable suggested fix. Examples: "Add '---' on line 8 to close frontmatter", "Rename folder to match pattern: E##-epic-slug", "Add 'title:' field to frontmatter".

**FR-009: Error Grouping**
Errors must be grouped by type for efficient scanning: frontmatter parse errors, pattern mismatch errors, validation errors, file access errors.

#### Validation Command

**FR-010: File Path Existence Checks**
The `shark validate` command must check all file_path values in epics, features, and tasks tables point to existing files on filesystem. Report missing files with entity type, key, and stale path.

**FR-011: Relationship Integrity Checks**
The `shark validate` command must validate foreign key relationships: all features reference existing epics (epic_key), all tasks reference existing features (feature_key). Report orphaned records with missing parent keys.

**FR-012: Broken Reference Detection**
The `shark validate` command must detect broken references: epic_key or feature_key values that don't match any entity in database. Distinguish between missing parent (orphan) and invalid key format (typo).

**FR-013: Validation Summary Report**
The `shark validate` command must generate summary report: total entities validated, broken file paths found, orphaned records found, broken references found, exit code (0 for success, 1 for validation failures).

**FR-014: Corrective Action Suggestions**
For each validation failure, the system must suggest corrective action: "Re-scan to update file paths", "Delete orphaned record: shark task delete {key}", "Fix epic_key typo in {file_path}".

#### JSON Output

**FR-015: JSON Schema**
JSON output must follow documented schema with versioned structure. Schema includes: version field, status (success/failure), metadata object (timestamp, duration, config), counts object (scanned, matched, skipped), entities object (epics, features, tasks, related_docs with counts), warnings array, errors array.

**FR-016: Structured Error Objects**
Each error in JSON errors array must be structured object with fields: severity (ERROR/WARNING/INFO), error_type (pattern_mismatch, validation_failure, parse_error, file_access_error), file_path (absolute), line_number (optional), message (human-readable), suggested_fix (actionable text).

**FR-017: Valid JSON Output**
JSON output must be valid JSON even when scan fails. No truncated output, no partial JSON. If scan crashes mid-process, output error JSON with available information.

**FR-018: JSON Validation Output**
The `shark validate --output=json` command must output JSON schema with: validation_checks object (file_paths, relationships, references), failures array (structured failure objects), summary object (total_checked, failures_found, orphans_found).

#### Dry-Run Mode

**FR-019: Full Workflow Execution**
Dry-run mode (`--dry-run`) must execute complete scan workflow: file discovery, pattern matching, metadata parsing, validation checks. Only database transaction commit is skipped.

**FR-020: Transaction Rollback**
In dry-run mode, the system must wrap all database operations in transaction and rollback at end. No changes persist to database.

**FR-021: Identical Report Generation**
Dry-run mode must generate identical report as real scan: same warnings, errors, counts, suggestions. Only difference is dry-run indicator in output.

**FR-022: Dry-Run Indicator**
Dry-run output must clearly indicate no database changes made. CLI output includes header: "DRY RUN MODE: No database changes will be committed". JSON output includes field: "dry_run": true.

**FR-023: Combinable Flags**
Dry-run mode must be combinable with other flags: `--dry-run --output=json`, `--dry-run --verbose`, `--dry-run --detect-conflicts`.

### Non-Functional Requirements

#### Performance

**NFR-001: Report Generation Overhead**
Report generation must add <5% overhead to scan time. For 100-file scan completing in 2 seconds, report generation must add <100ms.

**NFR-002: Validation Performance**
The `shark validate` command must complete in <1 second for databases with 1,000 entities (epics + features + tasks). File existence checks are majority of time.

**NFR-003: JSON Serialization**
JSON output serialization must complete in <50ms for reports containing 1,000+ error objects. Use efficient JSON library (encoding/json in Go).

#### Reliability

**NFR-004: Error Recovery (REQ-NF-005)**
Individual file parse failures must not halt entire scan. Wrap file parsing in error handler, log error, continue with next file. Scan completes even if 50% of files fail.

**NFR-005: Graceful Degradation**
If report generation fails (filesystem error writing report file), scan must still complete successfully. Log report generation failure, output summary to stderr.

**NFR-006: Validation Safety**
The `shark validate` command must be read-only. No database modifications. No file modifications. Exit with error if attempting unsafe operations.

#### Usability

**NFR-007: Actionable Error Messages (REQ-NF-006)**
All errors and warnings must include: file path, line number (if applicable), specific issue description, suggested fix. Users should fix issues in <5 minutes without external documentation.

**NFR-008: Dry-Run Mode (REQ-NF-007)**
All scan commands must support `--dry-run` flag for preview without database changes. Dry-run mode generates full report as if committed.

**NFR-009: Terminal Output Formatting**
CLI output must use clear visual hierarchy: section headers, indented lists, color coding (errors red, warnings yellow, success green). Respect `--no-color` flag for CI environments.

**NFR-010: Progress Feedback**
For scans processing 50+ files, provide progress indicators showing current phase and files processed. Update progress without excessive output (max 1 update per second).

#### Accessibility

**NFR-011: Color-Blind Friendly**
Terminal output must not rely solely on color to convey information. Use symbols: ✗ for errors, ⚠ for warnings, ✓ for success. Colors are enhancement, not requirement.

**NFR-012: Screen Reader Compatible**
JSON output schema must be self-documenting with clear field names. CLI output must use clear text labels, not ASCII art or Unicode box-drawing characters (unless `--fancy` flag).

#### Maintainability

**NFR-013: Extensible Report Format**
Report generation code must support adding new report sections without modifying existing code. Use plugin or template pattern for report sections.

**NFR-014: JSON Schema Versioning**
JSON output schema must include version field. Breaking changes require version bump. Code must support parsing multiple schema versions for backward compatibility.

**NFR-015: Test Coverage**
Report generation and validation code must have >85% unit test coverage. Test all error types, all validation checks, all output formats (CLI, JSON).

---

## Acceptance Criteria

### Scan Report Acceptance Criteria

**Given** a scan that processes 100 files (80 matched, 20 skipped),
**When** the scan completes,
**Then** the report must show:
- Total files scanned: 100
- Breakdown by type: 10 epics, 35 features, 35 tasks, 0 related docs matched
- 20 skipped files with file paths, skip reasons, and suggested fixes
- Scan metadata: timestamp, duration, validation level, patterns used
- Clear visual hierarchy with section headers and indented lists

**Given** a file with invalid frontmatter (missing closing '---'),
**When** the file is parsed during scan,
**Then** the error must include:
- File path: absolute path to file
- Line number: exact line where closing '---' should be
- Specific error: "Missing closing '---' for frontmatter block"
- Suggested fix: "Add '---' on line {N} to close frontmatter"

### Validation Command Acceptance Criteria

**Given** a database with 3 tasks referencing non-existent features,
**When** `shark validate` is executed,
**Then** the validation must:
- Detect all 3 orphaned tasks
- Report task keys and missing feature keys
- Suggest corrective actions: "Create missing feature or delete orphaned task"
- Exit with code 1 (failure)
- Display summary: "Found 3 orphaned tasks"

**Given** a database with 5 epics where file_path points to moved files,
**When** `shark validate` is executed,
**Then** the validation must:
- Check all 5 file paths against filesystem
- Detect all 5 missing files
- Report epic keys and stale file paths
- Suggest corrective action: "Re-scan to update file paths"
- Exit with code 1 (failure)

### JSON Output Acceptance Criteria

**Given** a scan executed with `--output=json`,
**When** the scan completes (success or failure),
**Then** the output must:
- Be valid JSON (parseable by standard JSON parsers)
- Include version field (e.g., "schema_version": "1.0")
- Include status field: "success" or "failure"
- Include counts object with scanned, matched, skipped counts
- Include errors array with structured error objects
- Include all fields from documented schema

**Given** a scan that fails mid-process (crash or interrupt),
**When** JSON output is generated,
**Then** the output must:
- Still be valid JSON (no truncation)
- Include status: "failure"
- Include error object describing failure
- Include partial results if available (files processed before crash)

### Dry-Run Mode Acceptance Criteria

**Given** a scan executed with `--dry-run`,
**When** the scan completes,
**Then** the system must:
- Execute full workflow: discovery, pattern matching, parsing, validation
- Generate complete report with all warnings, errors, counts
- Display "DRY RUN MODE: No database changes will be committed" header
- Rollback database transaction (no changes persist)
- Allow re-running scan without conflicts

**Given** a dry-run scan that would import 50 new files,
**When** dry-run completes and real scan is executed,
**Then** the real scan must:
- Import all 50 files successfully
- Generate identical warnings and errors as dry-run
- Commit database transaction (changes persist)
- Match dry-run counts exactly (50 imported)

### Error Message Acceptance Criteria

**Given** any error or warning during scan,
**When** the error is reported,
**Then** the message must include:
- Severity level: ERROR, WARNING, or INFO
- File path: absolute path (clickable in terminal)
- Line number: if applicable (parse errors)
- Specific error description: what went wrong
- Suggested fix: concrete action to resolve issue

**Given** 20 errors of mixed types (parse errors, validation errors, pattern mismatches),
**When** errors are displayed,
**Then** the errors must be:
- Grouped by error type (all parse errors together, all validation errors together)
- Sorted within groups (alphabetically by file path)
- Displayed with clear separators between groups
- Summarized at end: "20 total errors: 8 parse errors, 7 validation errors, 5 pattern mismatches"

---

## Out of Scope

### Explicitly NOT Included

**1. Interactive Error Resolution UI**
Providing an interactive TUI (Text User Interface) or GUI for stepping through errors, previewing fixes, and applying corrections is explicitly out of scope. This feature provides clear error messages and suggested fixes; users apply fixes manually.

**Rationale**: Interactive UIs add significant complexity (state management, input handling, screen rendering). Clear error messages with actionable suggestions provide 90% of value with 10% of complexity.

**Workaround**: Users read error report, apply suggested fixes to files, re-run scan to verify fixes.

**2. Validation Auto-Fix Implementation**
While Story 10 describes auto-fix mode as "could-have," implementing `shark validate --fix` is explicitly deferred to future iteration. This feature will only detect and report issues, not automatically correct them.

**Rationale**: Auto-fix requires sophisticated logic to avoid data corruption (what if auto-fix guesses wrong parent epic?). Detection and reporting provides immediate value; auto-fix can be added incrementally.

**Future Consideration**: E06-F06 (Validation Auto-Fix) could implement safe auto-corrections with user confirmation and rollback.

**3. Report Export Formats Beyond JSON**
Supporting report export in formats like HTML, PDF, Markdown, CSV, or XML is out of scope. Only CLI text output and JSON output are supported.

**Rationale**: JSON covers programmatic use cases; CLI text covers human readability. Additional formats add maintenance burden without clear use case.

**Workaround**: Users can pipe JSON output to external tools (jq, custom scripts) to convert to desired format.

**4. Real-Time Scan Monitoring Dashboard**
Providing a web-based dashboard or live-updating TUI for monitoring scan progress in real-time is out of scope. Progress indicators are text-based CLI output only.

**Rationale**: Real-time dashboards require web server or TUI framework, adding significant complexity for marginal benefit (scans complete in seconds).

**Workaround**: Users monitor CLI progress indicators or use `--quiet` mode for scripting.

**5. Historical Scan Report Archive**
Automatically storing historical scan reports for comparison over time (e.g., "show diff between today's scan and last week's scan") is out of scope.

**Rationale**: Report archiving requires storage management, retention policies, and comparison logic. Users can manually save reports if needed.

**Workaround**: Users redirect scan output to files: `shark scan > scan-$(date +%Y%m%d).log` for manual archiving.

**6. Validation Scheduled Jobs**
Automatically running validation checks on schedule (e.g., nightly validation cron job) with email/Slack notifications is out of scope.

**Rationale**: Scheduling and notifications are orthogonal concerns better handled by external tools (cron, GitHub Actions, CI/CD pipelines).

**Workaround**: Users configure cron jobs or CI pipelines to run `shark validate --output=json` and parse results for alerting.

**7. Machine-Readable Error Codes**
Providing error codes (e.g., ERR_001_PARSE_FRONTMATTER) for programmatic error classification beyond error_type field in JSON is out of scope.

**Rationale**: JSON error_type field (pattern_mismatch, validation_failure, parse_error) provides sufficient categorization for programmatic handling. Numeric error codes add maintenance burden.

**Future Consideration**: If error type taxonomy becomes insufficient, error codes can be added without breaking changes.

**8. Performance Profiling and Bottleneck Reports**
Providing detailed performance breakdowns (time spent in discovery vs. parsing vs. validation) and bottleneck identification is out of scope.

**Rationale**: Scan performance optimization is separate concern from import correctness. Performance profiling can be added if scans become slow.

**Workaround**: Users can use external profiling tools (Go pprof, time command) to analyze scan performance.

**9. Validation Rule Customization**
Allowing users to configure custom validation rules (e.g., "require all features to have architecture.md file") is out of scope. Validation checks are hardcoded: file existence, relationship integrity, orphaned record detection.

**Rationale**: Custom validation rules require rule engine, DSL, or plugin system. Hardcoded checks cover core integrity concerns.

**Future Consideration**: E06-F07 (Custom Validation Rules) could add rule engine if demand warrants.

**10. Diff Output Implementation**
While Story 11 describes diff output for dry-run as "could-have," implementing `--dry-run --diff` showing database changes (inserts, updates, deletes) is explicitly deferred.

**Rationale**: Diff output requires tracking all database operations, comparing current vs. new state, and formatting changes. Dry-run with standard report provides sufficient preview.

**Future Consideration**: Can be added in future iteration if users need granular change visibility.

---

## Report Format Examples

### CLI Text Output Example

```
Shark Scan Report
=================
Scan completed at 2025-12-17 14:32:05
Duration: 2.3 seconds
Validation level: balanced
Documentation root: /home/user/project/docs/plan

Summary
-------
Total files scanned: 265
  ✓ Matched: 245
  ✗ Skipped: 20

Breakdown by Type
-----------------
Epics:
  ✓ Matched: 12 (10 standard + 2 special types)
  ✗ Skipped: 1

Features:
  ✓ Matched: 87
  ✗ Skipped: 5

Tasks:
  ✓ Matched: 146
  ✗ Skipped: 14

Related Docs:
  ✓ Matched: 0
  ✗ Skipped: 0

Errors and Warnings
-------------------

Parse Errors (3):
  ERROR: Cannot parse frontmatter in /home/user/project/docs/plan/E04-task-mgmt/E04-F07-sync/tasks/T-E04-F07-003.md:5
    Missing closing '---' for frontmatter block
    Suggestion: Add '---' on line 8 to close frontmatter

  ERROR: Cannot parse frontmatter in /home/user/project/docs/plan/E05-reporting/prd.md:12
    Invalid YAML syntax: unexpected character '#' at line 12
    Suggestion: Check YAML formatting, ensure proper indentation and quoting

  ERROR: Cannot extract title from /home/user/project/docs/plan/bugs/tasks/027-fix-login.md
    No title field in frontmatter, no H1 heading found, filename not descriptive
    Suggestion: Add 'title: Fix login validation bug' to frontmatter

Pattern Mismatch Warnings (14):
  WARNING: File does not match any pattern: /home/user/project/docs/plan/E04-task-mgmt/notes.md
    No epic, feature, or task pattern matched
    Suggestion: Rename to match pattern or move to non-tracked location

  WARNING: File does not match any pattern: /home/user/project/docs/plan/tech-debt/random-ideas.txt
    File extension .txt not supported (only .md)
    Suggestion: Convert to markdown (.md) or exclude from docs/plan

  ... (12 more pattern mismatch warnings)

Validation Warnings (3):
  WARNING: Auto-generated epic title for /home/user/project/docs/plan/bugs/
    No epic.md found, generated title from folder name: "Auto: Bugs"
    Suggestion: Create epic.md with title: field for custom epic name

  ... (2 more validation warnings)

Scan Complete
-------------
Successfully imported 245 items:
  - 12 epics
  - 87 features
  - 146 tasks

Run 'shark validate' to verify database integrity.
```

### JSON Output Example

```json
{
  "schema_version": "1.0",
  "status": "success",
  "dry_run": false,
  "metadata": {
    "timestamp": "2025-12-17T14:32:05Z",
    "duration_seconds": 2.3,
    "validation_level": "balanced",
    "documentation_root": "/home/user/project/docs/plan",
    "patterns": {
      "epic_folder": "(?P<epic_id>E\\d{2})-(?P<epic_slug>[a-z0-9-]+)|(?P<epic_id>tech-debt|bugs|change-cards)",
      "feature_folder": "E(?P<epic_num>\\d{2})-F(?P<feature_num>\\d{2})-(?P<feature_slug>[a-z0-9-]+)",
      "task_file": "T-E(?P<epic_num>\\d{2})-F(?P<feature_num>\\d{2})-(?P<number>\\d{3})\\.md"
    }
  },
  "counts": {
    "scanned": 265,
    "matched": 245,
    "skipped": 20
  },
  "entities": {
    "epics": {
      "matched": 12,
      "skipped": 1
    },
    "features": {
      "matched": 87,
      "skipped": 5
    },
    "tasks": {
      "matched": 146,
      "skipped": 14
    },
    "related_docs": {
      "matched": 0,
      "skipped": 0
    }
  },
  "errors": [
    {
      "severity": "ERROR",
      "error_type": "parse_error",
      "file_path": "/home/user/project/docs/plan/E04-task-mgmt/E04-F07-sync/tasks/T-E04-F07-003.md",
      "line_number": 5,
      "message": "Cannot parse frontmatter: Missing closing '---' for frontmatter block",
      "suggested_fix": "Add '---' on line 8 to close frontmatter"
    },
    {
      "severity": "ERROR",
      "error_type": "parse_error",
      "file_path": "/home/user/project/docs/plan/E05-reporting/prd.md",
      "line_number": 12,
      "message": "Cannot parse frontmatter: Invalid YAML syntax: unexpected character '#' at line 12",
      "suggested_fix": "Check YAML formatting, ensure proper indentation and quoting"
    },
    {
      "severity": "ERROR",
      "error_type": "parse_error",
      "file_path": "/home/user/project/docs/plan/bugs/tasks/027-fix-login.md",
      "line_number": null,
      "message": "Cannot extract title: No title field in frontmatter, no H1 heading found, filename not descriptive",
      "suggested_fix": "Add 'title: Fix login validation bug' to frontmatter"
    }
  ],
  "warnings": [
    {
      "severity": "WARNING",
      "error_type": "pattern_mismatch",
      "file_path": "/home/user/project/docs/plan/E04-task-mgmt/notes.md",
      "line_number": null,
      "message": "File does not match any pattern: No epic, feature, or task pattern matched",
      "suggested_fix": "Rename to match pattern or move to non-tracked location"
    },
    {
      "severity": "WARNING",
      "error_type": "validation_warning",
      "file_path": "/home/user/project/docs/plan/bugs/",
      "line_number": null,
      "message": "Auto-generated epic title: No epic.md found, generated title from folder name: 'Auto: Bugs'",
      "suggested_fix": "Create epic.md with title: field for custom epic name"
    }
  ],
  "summary": {
    "imported": 245,
    "errors": 3,
    "warnings": 17
  }
}
```

### Validation Output Example (CLI)

```
Shark Validation Report
=======================
Validation completed at 2025-12-17 14:35:12
Database: /home/user/project/.shark/project.db

File Path Existence Checks
---------------------------
Checking 245 file paths...

✗ Broken file paths found: 3

Epic file paths:
  ✗ E04-task-mgmt-cli-core
    Path: /home/user/project/docs/plan/E04-task-mgmt-cli/epic.md
    Issue: File not found (may have been moved or deleted)
    Suggestion: Update path to /home/user/project/docs/plan/E04-task-mgmt-cli-core/epic.md or re-scan

Feature file paths:
  ✗ E04-F07-initialization-sync
    Path: /home/user/project/docs/plan/E04-task-mgmt-cli-core/E04-F07-sync/prd.md
    Issue: File not found (may have been moved or deleted)
    Suggestion: Update path to correct location or re-scan

Task file paths:
  ✗ T-E04-F07-003
    Path: /home/user/project/docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/tasks/T-E04-F07-003.md
    Issue: File not found (may have been moved or deleted)
    Suggestion: Update path or delete stale task record

Relationship Integrity Checks
------------------------------
Checking epic/feature/task relationships...

✗ Orphaned records found: 2

Orphaned features (no parent epic):
  ✗ E09-F01-oauth-integration
    Missing parent epic: E09-identity-platform
    Suggestion: Create epic E09-identity-platform or reassign feature to existing epic

Orphaned tasks (no parent feature):
  ✗ T-E04-F10-001
    Missing parent feature: E04-F10-distribution
    Suggestion: Create feature E04-F10-distribution or delete orphaned task

Validation Summary
------------------
Total entities validated: 245
  - Epics: 12
  - Features: 87
  - Tasks: 146

Issues found: 5
  - Broken file paths: 3
  - Orphaned records: 2

✗ VALIDATION FAILED

Run 'shark scan --incremental' to update file paths.
Run 'shark epic create E09-identity-platform' to create missing parent epic.
```

### Validation Output Example (JSON)

```json
{
  "schema_version": "1.0",
  "status": "failure",
  "metadata": {
    "timestamp": "2025-12-17T14:35:12Z",
    "database_path": "/home/user/project/.shark/project.db"
  },
  "validation_checks": {
    "file_paths": {
      "total_checked": 245,
      "broken_paths": 3
    },
    "relationships": {
      "total_checked": 233,
      "orphaned_features": 1,
      "orphaned_tasks": 1
    }
  },
  "failures": [
    {
      "check_type": "file_path_existence",
      "entity_type": "epic",
      "entity_key": "E04-task-mgmt-cli-core",
      "file_path": "/home/user/project/docs/plan/E04-task-mgmt-cli/epic.md",
      "issue": "File not found (may have been moved or deleted)",
      "suggested_fix": "Update path to /home/user/project/docs/plan/E04-task-mgmt-cli-core/epic.md or re-scan"
    },
    {
      "check_type": "file_path_existence",
      "entity_type": "feature",
      "entity_key": "E04-F07-initialization-sync",
      "file_path": "/home/user/project/docs/plan/E04-task-mgmt-cli-core/E04-F07-sync/prd.md",
      "issue": "File not found (may have been moved or deleted)",
      "suggested_fix": "Update path to correct location or re-scan"
    },
    {
      "check_type": "relationship_integrity",
      "entity_type": "feature",
      "entity_key": "E09-F01-oauth-integration",
      "missing_parent_type": "epic",
      "missing_parent_key": "E09-identity-platform",
      "issue": "Orphaned feature: parent epic does not exist",
      "suggested_fix": "Create epic E09-identity-platform or reassign feature to existing epic"
    },
    {
      "check_type": "relationship_integrity",
      "entity_type": "task",
      "entity_key": "T-E04-F10-001",
      "missing_parent_type": "feature",
      "missing_parent_key": "E04-F10-distribution",
      "issue": "Orphaned task: parent feature does not exist",
      "suggested_fix": "Create feature E04-F10-distribution or delete orphaned task"
    }
  ],
  "summary": {
    "total_validated": 245,
    "total_issues": 5,
    "broken_file_paths": 3,
    "orphaned_records": 2
  }
}
```

---

## CLI Output Design Considerations

### Visual Hierarchy

1. **Section Headers**: Use uppercase, underlined with '=' or '-' for clear section delineation
2. **Indentation**: Use 2-space indentation for nested items (errors under error type groups)
3. **Symbols**: Use ✓ (success), ✗ (failure), ⚠ (warning) for quick visual scanning
4. **Color Coding**: Red for errors, yellow for warnings, green for success (with --no-color fallback)
5. **Grouping**: Group related items (all parse errors together, all validation warnings together)

### Information Density

- **Summary First**: Show high-level summary before detailed breakdowns (users may not need details)
- **Progressive Disclosure**: Summary → Breakdown by Type → Detailed Errors → Corrective Actions
- **Truncation**: For reports with 100+ errors, show first 20 with "... (80 more errors)" and suggest `--output=json` for full list

### Actionability

- **Next Steps**: End report with clear next steps: "Run 'shark validate' to verify integrity"
- **Copy-Paste Ready**: Suggested fixes should be copy-pasteable commands when applicable
- **File Paths**: Use absolute paths that are clickable in modern terminal emulators (iTerm2, VS Code terminal)

### Machine Readability

- **JSON Schema**: Document JSON schema with field descriptions, types, example values
- **Stable Fields**: Never remove or rename JSON fields (additive changes only, use versioning for breaking changes)
- **Error Taxonomy**: Use consistent error_type values: pattern_mismatch, validation_failure, parse_error, file_access_error

---

*Last Updated*: 2025-12-17
