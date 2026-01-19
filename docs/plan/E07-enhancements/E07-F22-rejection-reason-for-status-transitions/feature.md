# Feature: Rejection Reason for Status Transitions

**Feature Key:** E07-F22
**Epic:** E07 - Shark Enhancements
**Status:** Draft
**Priority:** High
**Business Value:** 8/10

## Executive Summary

When a task is rejected by QA, code review, or approval stages and sent backward in the workflow, we need to capture WHY it was rejected. This helps developers understand what needs to be fixed and creates an audit trail of quality issues.

This feature adds a `--reason` flag to backward status transitions, storing rejection reasons as specialized task notes while maintaining backward compatibility with existing commands.

---

## Goal

### Problem

AI agents and developers working in Shark Task Manager currently have no way to capture rejection reasons when a task fails review and must be sent backward in the workflow. When a code reviewer, QA engineer, or approver rejects a task, the developer picking it up has no context about what went wrong.

Current issues:
- Developer receives rejected task but doesn't know why it was rejected
- Multiple rejection cycles waste time due to lack of clarity
- No audit trail of quality issues for process improvement
- Agents must maintain rejection context outside of Shark system

### Solution

Add `--reason` flag to status transition commands that:
- Captures WHY a task was rejected when sent backward in workflow
- Stores rejection as typed task note (`note_type: rejection`) linked to history
- Displays rejection history prominently in `shark task get` output
- Enables filtering and searching rejection reasons
- Optionally links detailed rejection documents for complex issues

### Impact

**Expected Outcomes:**
- Reduce time to fix rejected tasks by 40% (developers know exactly what to fix)
- Decrease repeat rejections by 60% (clearer feedback prevents same mistakes)
- Improve rejection reason quality to > 80% including specific file/line references
- Enable quality trend analysis (track most common rejection reasons)

---

## User Personas

### Persona 1: AI Code Reviewer Agent

**Profile:**
- **Role/Title**: Autonomous code reviewer agent
- **Experience Level**: Expert-level code analysis, automated quality checking
- **Key Characteristics**:
  - Analyzes code for bugs, style violations, missing error handling
  - Runs automated tests and linting tools
  - Must provide actionable feedback to developer agents

**Goals Related to This Feature:**
1. Provide clear, actionable rejection reasons referencing specific files/lines
2. Link to detailed review documents when issues are complex
3. Track quality trends to identify patterns in code issues

**Pain Points This Feature Addresses:**
- No way to record rejection reasons in Shark system (currently uses external logs)
- Developers don't see why code was rejected
- Can't link to detailed code review documents

**Success Looks Like:**
Reviewer agent rejects task with specific reason ("Missing error handling on line 67 in user_repository.go"), developer agent reads reason and fixes issue on first attempt.

### Persona 2: AI Developer Agent

**Profile:**
- **Role/Title**: Autonomous developer implementing tasks
- **Experience Level**: Follows specifications and coding standards
- **Key Characteristics**:
  - Implements features based on task specifications
  - Reads rejection feedback to fix issues
  - Resubmits tasks after addressing review comments

**Goals Related to This Feature:**
1. Understand why task was rejected before starting fix
2. Access detailed feedback including linked documents
3. Track rejection history to avoid repeat mistakes

**Pain Points This Feature Addresses:**
- Must guess why task was rejected (wastes time)
- No access to detailed review feedback
- Can't see if task has been rejected multiple times

**Success Looks Like:**
Developer agent picks up rejected task, reads rejection reason, accesses linked bug report, fixes all issues, and task passes review on next attempt.

### Persona 3: QA Engineer Agent

**Profile:**
- **Role/Title**: Quality assurance and testing agent
- **Experience Level**: Expert in test automation and quality verification
- **Key Characteristics**:
  - Runs test suites against implemented features
  - Documents bugs with detailed reproduction steps
  - Validates fixes before final approval

**Goals Related to This Feature:**
1. Document test failures with clear reproduction steps
2. Link bug reports when rejection reason requires detailed explanation
3. Track quality metrics (rejection rate, common failure patterns)

**Pain Points This Feature Addresses:**
- No way to attach detailed bug reports to rejection
- Rejection reasons limited to short text (complex bugs need documents)
- Can't track quality trends over time

**Success Looks Like:**
QA agent finds critical bug, creates detailed bug report with screenshots, rejects task with link to bug report, developer reads report and fixes issue correctly.

---

## User Stories

### Must-Have Stories

**Story 1**: As a code reviewer agent, I want to reject a task with a specific reason so that the developer knows exactly what to fix.

**Acceptance Criteria**:
- [ ] Can provide rejection reason when transitioning backward (e.g., `ready_for_code_review` → `in_development`)
- [ ] Reason is required for backward transitions (or `--force` to bypass)
- [ ] Reason is stored as task note with `note_type: rejection`
- [ ] Reason is linked to task history record for traceability

**Story 2**: As a developer agent, I want to see why my task was rejected so that I can fix the issues on first attempt.

**Acceptance Criteria**:
- [ ] `shark task get <task-key>` displays rejection history prominently
- [ ] Can see all past rejections chronologically
- [ ] Each rejection shows timestamp, rejector, from/to status, and reason
- [ ] JSON output includes complete `rejection_history` array

**Story 3**: As a QA agent, I want to link detailed bug reports when rejecting tasks so that developers have complete context.

**Acceptance Criteria**:
- [ ] Can provide `--reason-doc` flag pointing to document
- [ ] Document is automatically linked to task via task_documents
- [ ] Linked document appears in rejection history display
- [ ] Can access document with `shark task docs <task-key>`

---

### Should-Have Stories

**Story 4**: As a developer agent, I want to filter tasks by rejection status so that I can prioritize fixing rejected tasks.

**Acceptance Criteria**:
- [ ] `shark task list` shows rejection indicators (⚠ symbol)
- [ ] Can filter tasks with `--has-rejections` flag
- [ ] JSON output includes `rejection_count` and `last_rejection_at`

**Story 5**: As an orchestrator, I want to search rejection reasons so that I can identify quality trends.

**Acceptance Criteria**:
- [ ] Can search rejection notes with `shark task notes --type=rejection --search="<term>"`
- [ ] Can group rejections by agent or reason pattern
- [ ] Can filter rejections by epic or time period

**Story 6**: As a developer agent, I want to see rejections in the task timeline so that I understand the full task history.

**Acceptance Criteria**:
- [ ] `shark task timeline` highlights rejections with ⚠ symbol
- [ ] Rejection entries show reason inline (truncated if long)
- [ ] Rejection entries indicate linked documents if present

---

### Could-Have Stories

**Story 7**: As a product manager agent, I want to see rejection metrics so that I can identify process improvements.

**Acceptance Criteria**:
- [ ] `shark epic get <epic>` shows rejection rate statistics
- [ ] Can view most common rejection reasons
- [ ] Can see rejection trends over time

**Story 8**: As a reviewer agent, I want rejection reason templates so that I can provide consistent feedback.

**Acceptance Criteria**:
- [ ] Predefined rejection reason templates for common issues
- [ ] Can use template with `--reason-template=<name>`
- [ ] Can customize template with parameters

---

### Edge Case & Error Stories

**Error Story 1**: As a reviewer agent, when I try to send a task backward without providing a reason, I want to see a helpful error message so that I know what's required.

**Acceptance Criteria**:
- [ ] Error message explains reason is required for backward transitions
- [ ] Error shows example command with `--reason` flag
- [ ] Error mentions `--force` flag as bypass option

**Error Story 2**: As a QA agent, when I provide `--reason-doc` pointing to non-existent file, I want to see an error so that I can correct the path.

**Acceptance Criteria**:
- [ ] Error message indicates document not found
- [ ] Error shows the path that was attempted
- [ ] Suggestion to verify file exists at path

---

## Requirements

### Functional Requirements

**Category: Status Transition**

1. **REQ-F-001**: Backward Transition Detection
   - **Description**: System automatically detects when a status transition moves backward in workflow
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Compares workflow phases of current status vs new status
     - [ ] Returns true if new phase < current phase (backward)
     - [ ] Handles special statuses (blocked, on_hold) correctly

2. **REQ-F-002**: Rejection Reason Requirement
   - **Description**: Require rejection reason for backward transitions unless force flag used
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Backward transitions without `--reason` flag return error
     - [ ] Error message is clear and actionable
     - [ ] `--force` flag bypasses reason requirement
     - [ ] Forward transitions allow optional reason

3. **REQ-F-003**: Rejection Note Creation
   - **Description**: Store rejection reason as task note with metadata linking to history
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Creates task_note with `note_type: rejection`
     - [ ] Stores reason in `content` field
     - [ ] Stores rejector in `created_by` field
     - [ ] Metadata includes history_id, from_status, to_status
     - [ ] Created within same transaction as status update

**Category: Display & Query**

4. **REQ-F-004**: Rejection History Display
   - **Description**: Show rejection history in task get command
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `shark task get` shows rejection history section if rejections exist
     - [ ] Each rejection shows timestamp, rejector, transition, reason
     - [ ] Rejections ordered chronologically (most recent first)
     - [ ] Terminal output uses visual separators and colors
     - [ ] JSON output includes complete `rejection_history` array

5. **REQ-F-005**: Document Linking
   - **Description**: Link detailed documents to rejection reasons
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `--reason-doc` flag accepts file path
     - [ ] Validates document exists before accepting
     - [ ] Creates task_documents link with `link_type: rejection_reason`
     - [ ] Document path stored in rejection note metadata
     - [ ] Document appears in `shark task docs` output

6. **REQ-F-006**: Timeline Integration
   - **Description**: Display rejections in task timeline
   - **User Story**: Links to Story 6
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `shark task timeline` shows rejection events
     - [ ] Rejection events highlighted with ⚠ symbol
     - [ ] Inline reason display (truncated if > 80 chars)
     - [ ] Link to document shown if present

**Category: Filtering & Search**

7. **REQ-F-007**: Rejection Indicators
   - **Description**: Show rejection status in task list
   - **User Story**: Links to Story 4
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `shark task list` shows ⚠ symbol for tasks with rejections
     - [ ] Terminal output shows rejection count
     - [ ] JSON output includes `rejection_count` and `last_rejection_at`

8. **REQ-F-008**: Rejection Note Filtering
   - **Description**: Filter and search rejection notes
   - **User Story**: Links to Story 5
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] `shark task notes --type=rejection` filters to rejection notes only
     - [ ] Can search rejection content with `--search` flag
     - [ ] Can filter by time period
     - [ ] JSON output for programmatic analysis

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Rejection History Query Performance
   - **Description**: Loading rejection history must not degrade task get performance
   - **Measurement**: Time to execute `shark task get` with rejection history
   - **Target**: < 100ms for tasks with up to 10 rejections
   - **Justification**: Agents poll task status frequently; slow queries impair workflow

**Security**

1. **REQ-NF-010**: Rejection Reason Content Security
   - **Description**: Rejection reasons must be sanitized to prevent injection attacks
   - **Implementation**: Escape special characters, validate input length
   - **Compliance**: OWASP secure coding guidelines
   - **Risk Mitigation**: Prevents stored XSS if rejection reasons displayed in web UI

**Accessibility**

1. **REQ-NF-020**: Terminal Output Accessibility
   - **Description**: Rejection history display must be readable with screen readers
   - **Standard**: Accessible text formatting with `--no-color` flag
   - **Testing**: Test with `--no-color` flag to verify readability without ANSI colors

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Code Review Rejection**
- **Given** task is in `ready_for_code_review` status
- **When** reviewer executes `shark task update <task> --status=in_development --reason="Missing error handling"`
- **Then** task status updates to `in_development`
- **And** rejection note is created with reason
- **And** task history records status transition
- **And** rejection note links to history record

**Scenario 2: Backward Transition Without Reason**
- **Given** task is in `ready_for_qa` status
- **When** agent executes `shark task update <task> --status=in_development` (no reason)
- **Then** command fails with error
- **And** error message explains reason is required for backward transitions
- **And** error shows example command with `--reason` flag

**Scenario 3: Developer Views Rejection**
- **Given** task has been rejected once
- **When** developer executes `shark task get <task>`
- **Then** terminal output shows rejection history section
- **And** rejection shows timestamp, rejector, transition, and reason
- **And** rejection is visually highlighted

**Scenario 4: Multiple Rejections**
- **Given** task has been rejected 3 times
- **When** developer executes `shark task get <task> --json`
- **Then** JSON response includes `rejection_history` array with 3 entries
- **And** entries are ordered chronologically (most recent first)

**Scenario 5: Rejection with Document**
- **Given** QA agent has bug report at `docs/bugs/BUG-2026-046.md`
- **When** agent executes `shark task update <task> --status=in_development --reason="Tests fail" --reason-doc="docs/bugs/BUG-2026-046.md"`
- **Then** rejection note is created with reason
- **And** document is linked to task
- **And** rejection history shows document path
- **And** `shark task docs <task>` lists the bug report

---

## Out of Scope

### Explicitly Excluded

1. **Rejection Notifications (Webhooks/Events)**
   - **Why**: Adds significant complexity (event system, webhook infrastructure)
   - **Future**: Planned for E07-F23 - Rejection Notifications
   - **Workaround**: Agents poll for task updates with `shark task list --has-rejections`

2. **Rejection Reason Templates**
   - **Why**: Need to gather data on common rejection patterns first
   - **Future**: E07-F26 - Rejection Reason Templates
   - **Workaround**: Agents maintain templates externally, paste into `--reason` flag

3. **AI-Powered Rejection Reason Suggestions**
   - **Why**: Requires ML model training, out of scope for MVP
   - **Future**: E07-F25 - Auto-Reason Suggestions
   - **Workaround**: Manual reason entry

4. **Rejection Analytics Dashboard**
   - **Why**: Visualization requires separate UI work
   - **Future**: E07-F24 - Rejection Analytics Dashboard
   - **Workaround**: Use JSON output with external analytics tools

---

### Alternative Approaches Rejected

**Alternative 1: Store Rejection Reason in task_history.notes**
- **Description**: Use existing `notes` field in task_history instead of separate task_note
- **Why Rejected**:
  - Harder to query all rejections across tasks
  - Can't filter rejection notes separately from general notes
  - Mixes concerns (history vs feedback)
  - Can't link documents to rejection reason

**Alternative 2: Dedicated `shark task reject` Command**
- **Description**: Create separate command for rejecting tasks instead of using `shark task update`
- **Why Rejected**:
  - Adds command complexity (more commands to learn)
  - Still need `shark task update` for other transitions
  - Duplicates status transition logic
  - Harder to maintain consistency

**Alternative 3: Auto-Detect Rejection from Commit Messages**
- **Description**: Parse git commit messages for "rejected" keyword and extract reason
- **Why Rejected**:
  - Too fragile (depends on commit message format)
  - Doesn't work for non-git workflows
  - Can't control what's considered a "rejection"
  - Adds unnecessary complexity

---

## Success Metrics

### Primary Metrics

1. **Time to Fix Rejected Tasks**
   - **What**: Average time from task rejection to fix completion
   - **Target**: Reduce by 40% (from ~4 hours to ~2.5 hours)
   - **Timeline**: 2 months after deployment
   - **Measurement**: Query task_history for time between rejection and next ready_for_review

2. **Repeat Rejection Rate**
   - **What**: Percentage of tasks rejected multiple times for same issue
   - **Target**: Decrease by 60% (from 30% to 12%)
   - **Timeline**: 3 months after deployment
   - **Measurement**: Analyze rejection reasons for similarity

3. **Rejection Reason Quality**
   - **What**: Percentage of rejections including specific file/line references
   - **Target**: > 80% of rejections have actionable details
   - **Timeline**: 1 month after deployment
   - **Measurement**: Text analysis of rejection reasons for file paths/line numbers

---

### Secondary Metrics

- **Rejection Note Usage**: > 90% of backward transitions include reason
- **Document Link Usage**: > 30% of rejections link to detailed documents
- **Developer Satisfaction**: Positive feedback on rejection clarity from agents
- **Review Cycle Time**: Average review cycles per task decreases by 20%

---

## Dependencies & Integrations

### Dependencies

- **Task Notes System (E10-F01)**: Existing infrastructure for typed notes
- **Task Documents Linking (E10-F03)**: Existing system for linking documents to tasks
- **Task History (existing)**: Audit trail of status transitions
- **Workflow Config (existing)**: Status flow and phase definitions

### Integration Requirements

- **Backward Compatibility**: Existing status transition commands continue working
- **CLI Output**: Enhanced terminal and JSON output for task get/list/timeline
- **Repository Layer**: New methods for rejection note creation and querying

---

## Compliance & Security Considerations

**Data Protection:**
- Rejection reasons may contain sensitive information (security vulnerabilities)
- Implement input sanitization to prevent injection attacks
- Consider encryption for rejection reasons in cloud database deployments

**Audit Trail:**
- All rejections must be traceable (timestamp, rejector, reason)
- Rejection history must be immutable (no editing past rejections)
- Link rejections to task_history for complete audit trail

---

## Implementation Details

### CLI Syntax

**Recommended Command:**
```bash
shark task update <task-key> --status=<new-status> [--reason="..."] [--reason-doc="..."]
```

**Examples:**

Code review rejection:
```bash
shark task update E07-F01-003 --status=in_development \
  --reason="Missing error handling for database.Query() on line 67. Add null check."
```

QA rejection with document:
```bash
shark task update E07-F01-005 --status=in_development \
  --reason="Tests fail on empty user input. See bug report." \
  --reason-doc="docs/bugs/BUG-2026-046.md"
```

Force bypass:
```bash
shark task update E07-F01-007 --status=in_development --force
# No reason required with --force
```

### Terminal Output Mock

**With Rejections:**
```
Task: E07-F01-003
Title: Implement user authentication
Status: in_development
Priority: 5

⚠️  REJECTION HISTORY (2 rejections)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[2026-01-15 14:30] Rejected by reviewer-agent-001
ready_for_code_review → in_development

Reason:
Missing error handling for database.Query() on line 67.
Add null check and return error to caller.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[2026-01-16 10:15] Rejected by qa-agent-003
ready_for_qa → in_development

Reason:
Tests fail on edge case: empty user input. Add validation.

📄 Related Document: docs/bugs/BUG-2026-046.md

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Started: 2026-01-14 11:30
Updated: 2026-01-16 10:15
```

### JSON Output Mock

```json
{
  "task": {
    "key": "E07-F01-003",
    "status": "in_development",
    "title": "Implement user authentication"
  },
  "rejection_history": [
    {
      "id": 45,
      "timestamp": "2026-01-15T14:30:00Z",
      "from_status": "ready_for_code_review",
      "to_status": "in_development",
      "rejected_by": "reviewer-agent-001",
      "reason": "Missing error handling for database.Query() on line 67. Add null check.",
      "reason_document": null,
      "history_id": 234
    },
    {
      "id": 52,
      "timestamp": "2026-01-16T10:15:00Z",
      "from_status": "ready_for_qa",
      "to_status": "in_development",
      "rejected_by": "qa-agent-003",
      "reason": "Tests fail on edge case: empty user input. Add validation.",
      "reason_document": "docs/bugs/BUG-2026-046.md",
      "history_id": 241
    }
  ]
}
```

### Database Schema

**No new tables needed.** Use existing:

**task_notes** (add metadata column):
```sql
ALTER TABLE task_notes ADD COLUMN metadata TEXT; -- JSON string
CREATE INDEX idx_task_notes_type_task ON task_notes(note_type, task_id);
```

**Metadata structure for rejection notes:**
```json
{
  "history_id": 234,
  "from_status": "ready_for_code_review",
  "to_status": "in_development",
  "document_path": "docs/bugs/BUG-2026-046.md"
}
```

### Repository Layer

**New methods:**
```go
// TaskNoteRepository
func (r *TaskNoteRepository) CreateRejectionNote(
    ctx context.Context,
    taskID int64,
    historyID int64,
    fromStatus string,
    toStatus string,
    reason string,
    rejectedBy string,
    documentPath *string,
) (*models.TaskNote, error)

func (r *TaskNoteRepository) GetRejectionHistory(
    ctx context.Context,
    taskID int64,
) ([]*RejectionHistoryEntry, error)

// TaskRepository
func (r *TaskRepository) UpdateStatus(
    ctx context.Context,
    taskKey string,
    newStatus string,
    agent *string,
    notes *string,
    reason *string,        // NEW
    reasonDoc *string,     // NEW
    force bool,
) error
```

### Backward Transition Detection

```go
func isBackwardTransition(currentStatus, newStatus string, workflow *config.WorkflowConfig) bool {
    phaseOrder := map[string]int{
        "planning": 1,
        "development": 2,
        "review": 3,
        "qa": 4,
        "approval": 5,
        "done": 6,
        "any": 0,
    }

    currentPhase := workflow.StatusMetadata[currentStatus].Phase
    newPhase := workflow.StatusMetadata[newStatus].Phase

    currentOrder := phaseOrder[currentPhase]
    newOrder := phaseOrder[newPhase]

    return newOrder < currentOrder && newOrder > 0
}
```

---

## AI Agent Guidelines

### Providing Rejection Reasons (Reviewer/QA Agents)

**DO:**
- ✅ Be specific: "Missing null check on line 45" not "Code has bugs"
- ✅ Reference files/lines: "user_repository.go line 67"
- ✅ Include fix suggestions: "Add validation in user_controller.go"
- ✅ Link documents for complex issues: `--reason-doc="docs/bugs/BUG-123.md"`
- ✅ Mention failing tests: "TestUserRepository_GetByID fails on empty input"

**DON'T:**
- ❌ Be vague: "Doesn't work" or "Fix this"
- ❌ Use emotional language: "This code is terrible"
- ❌ Reject without actionable feedback

**Example:**
```bash
# Good
shark task update E07-F01-003 --status=in_development \
  --reason="Missing error handling for database.Query() on line 67. Add null check and return error to caller."

# Better (with document)
shark task update E07-F01-003 --status=in_development \
  --reason="Found 3 critical issues. See code review document." \
  --reason-doc="docs/reviews/E07-F01-003-review.md"
```

### Reading Rejection Reasons (Developer Agents)

**Workflow:**

1. Check for rejections first:
   ```bash
   shark task get E07-F01-003 --json | jq '.rejection_history'
   ```

2. Read all rejections chronologically (most recent = most urgent)

3. Check for linked documents:
   ```bash
   shark task docs E07-F01-003
   ```

4. Review timeline for context:
   ```bash
   shark task timeline E07-F01-003
   ```

5. Fix issues and note what was addressed:
   ```bash
   shark task update E07-F01-003 --status=ready_for_code_review \
     --notes="Fixed error handling on line 67. Added null check and test case."
   ```

---

## Related Features

### Dependencies
- **E10-F01**: Task notes system (existing)
- **E10-F03**: Task documents linking (existing)
- **E07-F21**: Task update command (existing)

### Future Enhancements
- **E07-F23**: Rejection notifications (webhook/event system)
- **E07-F24**: Rejection analytics dashboard
- **E07-F25**: AI-powered rejection reason suggestions
- **E07-F26**: Rejection reason templates

---

*Last Updated*: 2026-01-16
