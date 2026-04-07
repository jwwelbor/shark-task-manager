# UAT Plan: Add Tech-Debt Entity Type (E25)

**Date**: 2026-04-05

---

## 1. UAT Scenarios

Each scenario maps to a success criterion from the epic PRD.

### UAT-1: Tech-Debt CRUD Lifecycle (SC-1)

**Objective**: Verify all CRUD operations work correctly for tech-debt entities.

**Scenarios**:

1.1 **Create tech-debt item**
- Run `shark td create "Refactor database connection pooling" --category=architecture --severity=high`
- Verify: entity created with key TD-001 (or next available)
- Verify: status defaults to workflow initial status (identified)
- Verify: category is "architecture", severity is "high"
- Verify: CLI output shows key, title, and file path (SC-6)
- Verify: `shark td get TD-001` returns correct details
- Verify: markdown file exists at the displayed file path

1.2 **Create with defaults**
- Run `shark td create "Fix N+1 queries"`
- Verify: category defaults to "code-quality"
- Verify: severity defaults to "medium"
- Verify: effort_estimate is null/empty

1.3 **Create with all fields**
- Run `shark td create "Update stale dependencies" --category=dependency --severity=low --effort-estimate="2 hours" --description="Several npm packages are 3+ major versions behind"`
- Verify: all fields persisted correctly

1.4 **Read tech-debt item**
- Run `shark td get TD-001`
- Verify: output includes key, title, status, category, severity, effort_estimate, file_path, created_at, updated_at

1.5 **Update tech-debt item**
- Run `shark td update TD-001 --severity=critical`
- Verify: severity changed to critical
- Run `shark td update TD-001 --title="Refactor DB connection pooling (urgent)"`
- Verify: title updated
- Run `shark td update TD-001 --category=performance`
- Verify: category updated

1.6 **Delete tech-debt item**
- Run `shark td delete TD-001`
- Verify: entity removed from database
- Verify: `shark td list` no longer includes TD-001
- Verify: `shark td get TD-001` returns not-found error

1.7 **List tech-debt items**
- Create 3+ tech-debt items with different categories and severities
- Run `shark td list`
- Verify: all items listed
- Verify: output includes key, title, status, category, severity

### UAT-2: Tech-Debt Filtering (SC-1, SC-8)

**Objective**: Verify list filtering and JSON output.

2.1 **Filter by category**
- Run `shark td list --category=architecture`
- Verify: only architecture items returned

2.2 **Filter by severity**
- Run `shark td list --severity=critical`
- Verify: only critical items returned

2.3 **Filter by status**
- Run `shark td list --status=identified`
- Verify: only identified items returned

2.4 **JSON output**
- Run `shark td list --json`
- Verify: valid JSON array returned
- Verify: all fields present in each object

### UAT-3: Tech-Debt Workflow (SC-2)

**Objective**: Verify status transitions via workflow system.

3.1 **Advance status**
- Create TD-001 (status: identified)
- Run `shark status advance TD-001`
- Verify: status becomes "triaged"
- Run `shark status advance TD-001`
- Verify: status becomes "in_progress"
- Run `shark status advance TD-001`
- Verify: status becomes "resolved"

3.2 **Set status directly**
- Run `shark status set TD-001 wont_fix`
- Verify: status changed to "wont_fix"

3.3 **Invalid transition rejected**
- With TD-001 in "resolved" status
- Run `shark status advance TD-001`
- Verify: error returned indicating terminal state (no valid next status)

3.4 **Status options**
- Run `shark status options TD-001` (when in "triaged")
- Verify: shows valid next statuses (in_progress, wont_fix, cancelled)

3.5 **Triage command**
- Create TD-002 (status: identified)
- Run `shark td triage TD-002 --severity=high --category=testing`
- Verify: status advances and fields updated

### UAT-4: Core Command Auto-Detection (SC-4)

**Objective**: Verify TD-### keys are routed correctly by core commands.

4.1 **shark get**
- Run `shark get TD-001`
- Verify: returns tech-debt details (same as `shark td get TD-001`)

4.2 **shark get with --field**
- Run `shark get TD-001 --field status`
- Verify: returns only the status value as plain string

4.3 **shark get with --json**
- Run `shark get TD-001 --json`
- Verify: returns valid JSON with all tech-debt fields

4.4 **shark status**
- Run `shark status advance TD-001`
- Verify: advances tech-debt status (same as entity-specific path)

4.5 **shark delete**
- Run `shark delete TD-001`
- Verify: tech-debt entity deleted

4.6 **shark view**
- Run `shark view TD-001`
- Verify: displays tech-debt markdown file contents

4.7 **shark history**
- Run `shark history TD-001`
- Verify: shows status change history

4.8 **shark update**
- Run `shark update TD-001 --title="Updated title"`
- Verify: title updated

4.9 **Case insensitivity**
- Run `shark get td-001`
- Verify: resolves correctly (key normalization)

### UAT-5: Notes and Context (SC-3)

**Objective**: Verify notes and context support for tech-debt entities.

5.1 **Add note**
- Run `shark td note add TD-001 --content="Found during code review of auth module" --type=comment`
- Verify: note persisted

5.2 **List notes**
- Run `shark td notes TD-001`
- Verify: note appears with content, type, and timestamp

5.3 **Multiple note types**
- Add notes with types: comment, decision, blocker, solution
- Verify: all types accepted and persisted

5.4 **Set context field**
- Run `shark td context set TD-001 --field affected_area --value "internal/repository/"`
- Verify: field set

5.5 **Get context**
- Run `shark td context get TD-001`
- Verify: returns context data including affected_area field

5.6 **Clear context**
- Run `shark td context clear TD-001`
- Verify: context data cleared

### UAT-6: File Path Display on Entity Create (SC-6)

**Objective**: Verify all entity creation commands display file paths.

6.1 **Epic create**
- Run `shark epic create "Test Epic"`
- Verify: output includes file path (e.g., "docs/plan/E##-test-epic/epic.md")

6.2 **Feature create**
- Run `shark feature create E## "Test Feature"`
- Verify: output includes file path

6.3 **Task create**
- Run `shark task create E## F## "Test Task"`
- Verify: output includes file path

6.4 **Bug create**
- Run `shark bug create "Test Bug" --severity=low`
- Verify: output includes file path

6.5 **Change-card create**
- Run `shark change create "Test Change"`
- Verify: output includes file path

6.6 **Tech-debt create**
- Run `shark td create "Test Debt" --category=testing`
- Verify: output includes file path

6.7 **JSON output includes file path**
- Run `shark td create "Test" --json`
- Verify: JSON output includes `file_path` field

### UAT-7: Search Integration (SC-5)

**Objective**: Verify tech-debt items appear in search results.

7.1 **Search by title**
- Create TD-001 with title "Refactor database connection pooling"
- Run `shark search "database"`
- Verify: TD-001 appears in results

7.2 **Search by key**
- Run `shark search "TD-001"`
- Verify: TD-001 appears in results

7.3 **Search by description**
- Create TD-002 with description containing "authentication"
- Run `shark search "authentication"`
- Verify: TD-002 appears in results

7.4 **Search includes tech-debt alongside other entities**
- Create a task and a tech-debt item both containing "refactor"
- Run `shark search "refactor"`
- Verify: both the task and tech-debt item appear

7.5 **Analytics inclusion**
- Run `shark analytics`
- Verify: output includes tech-debt counts

### UAT-8: Migration Safety (SC-7)

**Objective**: Verify database migration does not affect existing data.

8.1 **Pre-migration snapshot**
- Record counts: `shark epic list --json | jq length`, same for feature, task, bug, change

8.2 **Run migration**
- Set `skip_migrations: false`
- Run any shark command
- Set `skip_migrations: true`

8.3 **Post-migration verification**
- Verify: `shark epic list` returns same results as before
- Verify: `shark task list` returns same results as before
- Verify: `shark bug list` returns same results as before
- Verify: `shark change list` returns same results as before
- Verify: `shark td list` returns empty list (no tech-debt items yet)

8.4 **Idempotent migration**
- Set `skip_migrations: false` and run again
- Verify: no errors (CREATE TABLE IF NOT EXISTS)

### UAT-9: JSON Output Consistency (SC-8)

**Objective**: Verify JSON output follows conventions of all other entity types.

9.1 **Get with --json**
- Run `shark td get TD-001 --json`
- Verify: valid JSON with fields: id, key, title, slug, description, status, category, severity, effort_estimate, context_data, file_path, created_at, updated_at

9.2 **Get with --field**
- Run `shark td get TD-001 --field status`
- Verify: returns plain status string
- Run `shark td get TD-001 --field category`
- Verify: returns plain category string

9.3 **List with --json**
- Run `shark td list --json`
- Verify: valid JSON array

---

## 2. Cross-Feature Integration Scenarios

### INT-1: Tech-Debt + Entity Relationships
- Create TD-001 and a task E07-F01-001
- Run `shark link TD-001 E07-F01-001 --type=related_to`
- Verify: relationship created
- Verify: `shark td get TD-001` shows relationship or related entities are queryable

### INT-2: Tech-Debt + Workflow Profile Switch
- Apply basic workflow: `shark admin init update --workflow=basic`
- Create TD-001
- Verify: tech-debt uses basic workflow statuses
- Apply advanced workflow: `shark admin init update --workflow=advanced`
- Verify: tech-debt uses advanced workflow statuses (or keeps its own dedicated workflow)

### INT-3: Tech-Debt + Entity History
- Create TD-001
- Advance status several times
- Run `shark history TD-001`
- Verify: complete status change history displayed

### INT-4: Tech-Debt + Global Status Commands
- Create TD-001
- Run `shark status TD-001`
- Verify: shows tech-debt status
- Run `shark status set TD-001 triaged`
- Verify: status updated
- Run `shark status options TD-001`
- Verify: shows valid transitions from "triaged"

---

## 3. Performance Considerations

- **Search query**: The UNION ALL query adds one more table scan. With an index on `tech_debts(key)` and `tech_debts(title)`, the impact is negligible for projects with < 1000 tech-debt items.
- **Key detection**: `IsTechDebtKey()` is O(1) regex match, added to a sequential check list. Adding one more check has negligible impact on command startup time.
- **Database migration**: CREATE TABLE is a one-time operation. No ongoing performance impact.

## 4. Security Considerations

- **Input validation**: All tech-debt fields go through the same validation pipeline as Bug/Change-Card: regex allowlist for keys, map allowlist for category/severity, TrimSpace for text fields, parameterized queries for all SQL.
- **SQL injection**: Repository uses parameterized queries exclusively (same pattern as all other repositories).
- **Key format**: `^TD-\d{3}$` regex is anchored and accepts only the exact expected format.
- **Category/severity validation**: `map[string]bool` allowlist prevents arbitrary values reaching the database.
- **Effort estimate**: Free-text field with length validation (max 100 characters) to prevent oversized inputs.

---

## 5. Exit Criteria Mapping

| Success Criterion | UAT Scenario(s) | Status |
|-------------------|-----------------|--------|
| SC-1: CRUD operations | UAT-1 (1.1-1.7), UAT-2 | Pending |
| SC-2: Workflow transitions | UAT-3 (3.1-3.5) | Pending |
| SC-3: Notes and context | UAT-5 (5.1-5.6) | Pending |
| SC-4: Core command auto-detection | UAT-4 (4.1-4.9) | Pending |
| SC-5: Search and analytics | UAT-7 (7.1-7.5) | Pending |
| SC-6: File path display on create | UAT-6 (6.1-6.7) | Pending |
| SC-7: Migration safety | UAT-8 (8.1-8.4) | Pending |
| SC-8: JSON output consistency | UAT-9 (9.1-9.3), UAT-2 (2.4) | Pending |

All success criteria have at least one UAT scenario. No orphaned requirements.

---

*UAT Plan by: Architect agent | Date: 2026-04-05*
