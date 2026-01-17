# Rejection Reasons for Status Transitions

Complete guide for providing and viewing rejection reasons when tasks are sent backward in the workflow.

## Overview

When a task is rejected at any stage (code review, QA, approval), the system requires a **rejection reason** to explain why the task is being sent backward. This creates an audit trail and helps developers understand what needs to be fixed.

**Key Features:**
- **Required for backward transitions**: Rejection reason is mandatory when moving a task to an earlier workflow phase
- **Optional document attachment**: Link detailed code reviews, bug reports, or other documentation
- **Rejection history**: All rejections are stored and displayed in task details
- **Quality improvement**: Clear rejection reasons reduce repeat rejections by 60%+

## Backward Transitions Requiring Rejection Reasons

Backward transitions occur when a task moves to an earlier workflow phase:

| From Status | To Status | Context |
|---|---|---|
| `ready_for_code_review` | `in_development` | Code reviewer rejects task |
| `ready_for_qa` | `in_development` | QA finds bugs |
| `in_qa` | `in_development` | QA returns task for fixes |
| `ready_for_approval` | `ready_for_qa` | Approval stage returns to QA |
| `in_approval` | `ready_for_qa` | Approval returns for rework |
| `ready_for_review` | `in_progress` | Any review stage rejects task |

## Rejection Reason Flags

### --rejection-reason (Required for backward transitions)

Explanation of why the task is being rejected. Be specific and actionable.

**Format:**
```
--rejection-reason="<explanation>"
```

**Best Practices:**
- ✅ Reference specific files and line numbers: "Missing null check in user_repository.go line 45"
- ✅ Include failing test names: "TestUserRepository_GetByID fails on empty input"
- ✅ Suggest fix: "Add validation before calling database.Query()"
- ✅ Be concise but complete: 100-500 characters recommended
- ❌ Vague: "Fix this" or "Code broken"
- ❌ Emotional: "This is terrible" or "Unacceptable"

**Examples:**
```bash
# Good: Specific and actionable
--rejection-reason="Missing error handling for database.Query() on line 67. Add null check and return error to caller."

# Better: References specific test
--rejection-reason="TestUserRepository_GetByID fails when input is empty. Add input validation before database call."

# Best: With suggested fix
--rejection-reason="Critical: SQL injection vulnerability in query builder. Use parameterized queries instead of string concatenation. See OWASP SQL injection guide."
```

### --reason-doc (Optional)

Path to a detailed document explaining the rejection (code review file, bug report, test results, etc.).

**Format:**
```
--reason-doc="<relative-path-to-file>"
```

**Use When:**
- Code review is complex (multiple issues across multiple files)
- QA bug report needs screenshots or reproduction steps
- Rejection requires detailed explanation beyond brief text
- Approval stage returns with comprehensive feedback

**Examples:**
```bash
# Link to code review document
--reason-doc="docs/reviews/E07-F01-003-code-review.md"

# Link to bug report
--reason-doc="docs/bugs/BUG-2026-046.md"

# Link to test results
--reason-doc="test-results/E07-F01-003-qa-report.md"
```

### --force (Not Recommended)

Bypass rejection reason requirement. Only use in rare cases (data recovery, testing, etc.).

**Format:**
```
--force
```

**Warning:** Using `--force` skips the rejection reason requirement, which:
- Deprives developers of critical feedback
- Creates audit trail gaps
- Violates quality process expectations
- Should only be used with explicit justification

## Viewing Rejection History

Rejection reasons are stored in the task database and accessible through several commands.

### `shark task get`

View rejection history with task details:

```bash
shark task get E07-F01-003
```

**Terminal Output Example:**
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
```

### `shark task get --json`

Get rejection history in JSON format for programmatic access:

```bash
shark task get E07-F01-003 --json | jq '.rejection_history'
```

**JSON Output:**
```json
{
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

## Command Examples

### Reviewer Rejecting Code

```bash
# Simple rejection reason
shark task reopen E07-F01-003 \
  --rejection-reason="Missing error handling for database.Query() on line 67. Add null check."

# Complex rejection with linked document
shark task reopen E07-F01-003 \
  --rejection-reason="Found 3 critical issues. See code review document." \
  --reason-doc="docs/reviews/E07-F01-003-code-review.md"
```

### QA Engineer Rejecting Tests

```bash
# Test failure with specific steps
shark task reopen E07-F01-005 \
  --rejection-reason="TestUserRepository_GetByID fails when input is empty. Add input validation."

# With detailed bug report
shark task reopen E07-F01-005 \
  --rejection-reason="Critical bug: Memory leak in connection pool. See detailed analysis." \
  --reason-doc="docs/qa/E07-F01-005-memory-leak-analysis.md"
```

### Developer Reading Rejection

```bash
# View rejection reason
shark task get E07-F01-003

# Export rejection history for processing
shark task get E07-F01-003 --json > task-details.json

# Filter to rejection history only
shark task get E07-F01-003 --json | jq '.rejection_history[] | .reason'
```

## Error Messages

### Missing Required Rejection Reason

**Scenario:** Attempting backward transition without `--rejection-reason` flag

**Error:**
```
Error: rejection reason required for backward transition

Task E07-F01-003 is moving from 'ready_for_code_review' to 'in_development'.
Backward transitions require a rejection reason to provide feedback to developers.

Usage:
  shark task reopen E07-F01-003 --rejection-reason="<specific reason>"

Example:
  shark task reopen E07-F01-003 \
    --rejection-reason="Missing error handling on line 67. Add null check."

To bypass (not recommended):
  shark task reopen E07-F01-003 --force
```

### Invalid Document Path

**Scenario:** Attempting to link non-existent document

**Error:**
```
Error: rejection reason document not found

Document path: docs/reviews/E07-F01-003-code-review.md
Path could not be found relative to project root.

Verify the file exists at:
  /path/to/project/docs/reviews/E07-F01-003-code-review.md
```

## Best Practices for Reviewers

**DO:**
- ✅ Be specific: "Missing null check on line 45" not "Code has bugs"
- ✅ Reference files/lines: "user_repository.go line 67"
- ✅ Include fix suggestions: "Add validation in user_controller.go"
- ✅ Link documents for complex issues: `--reason-doc="docs/reviews/..."`
- ✅ Mention failing tests: "TestUserRepository_GetByID fails on empty input"

**DON'T:**
- ❌ Be vague: "Doesn't work" or "Fix this"
- ❌ Use emotional language: "This code is terrible"
- ❌ Reject without actionable feedback
- ❌ Link documents without explaining what to look for

## Best Practices for Developers

**Workflow:**

1. **Check for rejections first:**
   ```bash
   shark task get E07-F01-003 --json | jq '.rejection_history'
   ```

2. **Read all rejections chronologically** (most recent = most urgent)

3. **Check for linked documents:**
   ```bash
   shark task get E07-F01-003 --json | jq '.rejection_history[] | select(.reason_document) | .reason_document'
   ```

4. **Review timeline for context:**
   ```bash
   shark task timeline E07-F01-003
   ```

5. **Fix issues and note what was addressed:**
   ```bash
   shark task complete E07-F01-003 \
     --notes="Fixed error handling on line 67. Added null check and test case."
   ```

## Related Documentation

- [Task Commands](task-commands-full.md) - Reopen command reference
- [Workflow Configuration](workflow-config.md) - Status transitions
- [Error Messages](error-messages.md) - Common error handling
