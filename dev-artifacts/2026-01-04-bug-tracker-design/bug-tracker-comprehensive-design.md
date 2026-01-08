# Bug Tracker Feature - Comprehensive Design Document

**Date**: 2026-01-04
**Status**: Design Phase
**Target Epic**: TBD

---

## Executive Summary

This document defines a comprehensive bug tracking feature for Shark Task Manager, designed to support both AI agents and human users in reporting, tracking, and resolving bugs. The design follows shark's existing patterns (similar to the idea tracker) while introducing bug-specific fields and workflows.

---

## 1. Overview & Motivation

### 1.1 Purpose

The bug tracker enables:
- **AI agents** to programmatically report bugs encountered during development, testing, or operations
- **Human users** to submit bug reports through the CLI
- **Teams** to track bug resolution lifecycle from discovery to closure
- **Integration** with shark's existing task/feature/epic hierarchy

### 1.2 Core Principles

1. **CLI-First Design**: Both AI agents and humans use the same CLI commands
2. **Lightweight Capture**: Quick bug entry with minimal required fields
3. **Rich Context**: Support detailed technical information when needed
4. **Task Integration**: Bugs can be converted to tasks or linked to existing work
5. **Flexible Storage**: Inline storage for typical bugs, file storage for complex cases

---

## 2. Data Model

### 2.1 Database Schema

```sql
-- ============================================================================
-- Table: bugs
-- ============================================================================
CREATE TABLE IF NOT EXISTS bugs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,                          -- Format: B-YYYY-MM-DD-xx (e.g., B-2026-01-04-01)
    title TEXT NOT NULL,                               -- Short bug description
    description TEXT,                                  -- Detailed bug description

    -- Bug classification
    severity TEXT NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')) DEFAULT 'medium',
    priority INTEGER CHECK (priority >= 1 AND priority <= 10) DEFAULT 5,
    category TEXT,                                     -- e.g., 'backend', 'frontend', 'database', 'cli', 'performance'

    -- Technical details
    steps_to_reproduce TEXT,                           -- How to reproduce the bug
    expected_behavior TEXT,                            -- What should happen
    actual_behavior TEXT,                              -- What actually happens
    error_message TEXT,                                -- Error message or stack trace

    -- Environment information
    environment TEXT,                                  -- e.g., 'production', 'staging', 'development', 'test'
    os_info TEXT,                                      -- Operating system details
    version TEXT,                                      -- Software version where bug occurs

    -- Metadata
    reporter_type TEXT CHECK (reporter_type IN ('human', 'ai_agent')) DEFAULT 'human',
    reporter_id TEXT,                                  -- Username or agent ID
    detected_at TIMESTAMP NOT NULL,                    -- When bug was detected

    -- File references
    attachment_file TEXT,                              -- Path to file with additional details (logs, screenshots refs, etc.)
    related_docs TEXT,                                 -- JSON array of related document paths

    -- Relationships
    related_to_epic TEXT,                              -- Epic key if bug relates to specific epic
    related_to_feature TEXT,                           -- Feature key if bug relates to specific feature
    related_to_task TEXT,                              -- Task key if bug found in specific task
    dependencies TEXT,                                 -- JSON array of bug keys this depends on

    -- Status tracking
    status TEXT NOT NULL CHECK (status IN ('new', 'confirmed', 'in_progress', 'resolved', 'closed', 'wont_fix', 'duplicate')) DEFAULT 'new',
    resolution TEXT,                                   -- Resolution notes

    -- Conversion tracking
    converted_to_type TEXT CHECK (converted_to_type IN ('task', 'epic', 'feature')),
    converted_to_key TEXT,
    converted_at TIMESTAMP,

    -- Audit fields
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,                             -- When bug was marked resolved
    closed_at TIMESTAMP                                -- When bug was closed
);

-- Indexes for bugs
CREATE UNIQUE INDEX IF NOT EXISTS idx_bugs_key ON bugs(key);
CREATE INDEX IF NOT EXISTS idx_bugs_status ON bugs(status);
CREATE INDEX IF NOT EXISTS idx_bugs_severity ON bugs(severity);
CREATE INDEX IF NOT EXISTS idx_bugs_priority ON bugs(priority);
CREATE INDEX IF NOT EXISTS idx_bugs_detected_at ON bugs(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_bugs_category ON bugs(category);
CREATE INDEX IF NOT EXISTS idx_bugs_environment ON bugs(environment);
CREATE INDEX IF NOT EXISTS idx_bugs_related_epic ON bugs(related_to_epic);
CREATE INDEX IF NOT EXISTS idx_bugs_related_feature ON bugs(related_to_feature);
CREATE INDEX IF NOT EXISTS idx_bugs_related_task ON bugs(related_to_task);

-- Trigger to auto-update updated_at for bugs
CREATE TRIGGER IF NOT EXISTS bugs_updated_at
AFTER UPDATE ON bugs
FOR EACH ROW
BEGIN
    UPDATE bugs SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

-- Trigger to set resolved_at timestamp
CREATE TRIGGER IF NOT EXISTS bugs_resolved_at
AFTER UPDATE ON bugs
FOR EACH ROW
WHEN NEW.status = 'resolved' AND OLD.status != 'resolved'
BEGIN
    UPDATE bugs SET resolved_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Trigger to set closed_at timestamp
CREATE TRIGGER IF NOT EXISTS bugs_closed_at
AFTER UPDATE ON bugs
FOR EACH ROW
WHEN NEW.status = 'closed' AND OLD.status != 'closed'
BEGIN
    UPDATE bugs SET closed_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
```

### 2.2 Go Model

```go
package models

import (
    "fmt"
    "regexp"
    "time"
)

// BugSeverity represents the severity level of a bug
type BugSeverity string

const (
    BugSeverityCritical BugSeverity = "critical"
    BugSeverityHigh     BugSeverity = "high"
    BugSeverityMedium   BugSeverity = "medium"
    BugSeverityLow      BugSeverity = "low"
)

// BugStatus represents the status of a bug
type BugStatus string

const (
    BugStatusNew        BugStatus = "new"
    BugStatusConfirmed  BugStatus = "confirmed"
    BugStatusInProgress BugStatus = "in_progress"
    BugStatusResolved   BugStatus = "resolved"
    BugStatusClosed     BugStatus = "closed"
    BugStatusWontFix    BugStatus = "wont_fix"
    BugStatusDuplicate  BugStatus = "duplicate"
)

// ReporterType indicates who reported the bug
type ReporterType string

const (
    ReporterTypeHuman   ReporterType = "human"
    ReporterTypeAIAgent ReporterType = "ai_agent"
)

// Bug represents a bug report
type Bug struct {
    ID    int64  `json:"id" db:"id"`
    Key   string `json:"key" db:"key"` // Format: B-YYYY-MM-DD-xx
    Title string `json:"title" db:"title"`

    // Core bug information
    Description        *string `json:"description,omitempty" db:"description"`
    Severity           BugSeverity `json:"severity" db:"severity"`
    Priority           *int    `json:"priority,omitempty" db:"priority"`
    Category           *string `json:"category,omitempty" db:"category"`

    // Technical details
    StepsToReproduce   *string `json:"steps_to_reproduce,omitempty" db:"steps_to_reproduce"`
    ExpectedBehavior   *string `json:"expected_behavior,omitempty" db:"expected_behavior"`
    ActualBehavior     *string `json:"actual_behavior,omitempty" db:"actual_behavior"`
    ErrorMessage       *string `json:"error_message,omitempty" db:"error_message"`

    // Environment
    Environment        *string `json:"environment,omitempty" db:"environment"`
    OSInfo             *string `json:"os_info,omitempty" db:"os_info"`
    Version            *string `json:"version,omitempty" db:"version"`

    // Reporter information
    ReporterType       ReporterType `json:"reporter_type" db:"reporter_type"`
    ReporterID         *string `json:"reporter_id,omitempty" db:"reporter_id"`
    DetectedAt         time.Time `json:"detected_at" db:"detected_at"`

    // File references
    AttachmentFile     *string `json:"attachment_file,omitempty" db:"attachment_file"`
    RelatedDocs        *string `json:"related_docs,omitempty" db:"related_docs"` // JSON array

    // Relationships
    RelatedToEpic      *string `json:"related_to_epic,omitempty" db:"related_to_epic"`
    RelatedToFeature   *string `json:"related_to_feature,omitempty" db:"related_to_feature"`
    RelatedToTask      *string `json:"related_to_task,omitempty" db:"related_to_task"`
    Dependencies       *string `json:"dependencies,omitempty" db:"dependencies"` // JSON array

    // Status and resolution
    Status             BugStatus `json:"status" db:"status"`
    Resolution         *string `json:"resolution,omitempty" db:"resolution"`

    // Conversion tracking
    ConvertedToType    *string    `json:"converted_to_type,omitempty" db:"converted_to_type"`
    ConvertedToKey     *string    `json:"converted_to_key,omitempty" db:"converted_to_key"`
    ConvertedAt        *time.Time `json:"converted_at,omitempty" db:"converted_at"`

    // Audit timestamps
    CreatedAt          time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt          time.Time  `json:"updated_at" db:"updated_at"`
    ResolvedAt         *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
    ClosedAt           *time.Time `json:"closed_at,omitempty" db:"closed_at"`
}

// Validate validates the Bug fields
func (b *Bug) Validate() error {
    // Validate key format
    if err := ValidateBugKey(b.Key); err != nil {
        return err
    }

    // Validate title
    if b.Title == "" {
        return ErrEmptyTitle
    }

    // Validate severity
    if err := ValidateBugSeverity(string(b.Severity)); err != nil {
        return err
    }

    // Validate status
    if err := ValidateBugStatus(string(b.Status)); err != nil {
        return err
    }

    // Validate priority if provided
    if b.Priority != nil {
        if *b.Priority < 1 || *b.Priority > 10 {
            return ErrInvalidPriority
        }
    }

    // Validate reporter type
    if err := ValidateReporterType(string(b.ReporterType)); err != nil {
        return err
    }

    // Validate JSON arrays
    if b.RelatedDocs != nil {
        if err := ValidateJSONArray(*b.RelatedDocs); err != nil {
            return fmt.Errorf("invalid related_docs JSON: %w", err)
        }
    }

    if b.Dependencies != nil {
        if err := ValidateJSONArray(*b.Dependencies); err != nil {
            return fmt.Errorf("invalid dependencies JSON: %w", err)
        }
    }

    return nil
}

// ValidateBugKey validates the bug key format (B-YYYY-MM-DD-xx)
func ValidateBugKey(key string) error {
    if key == "" {
        return ErrEmptyKey
    }

    // Pattern: B-YYYY-MM-DD-xx where xx is 01-99
    pattern := `^B-\d{4}-\d{2}-\d{2}-\d{2}$`
    matched, err := regexp.MatchString(pattern, key)
    if err != nil {
        return fmt.Errorf("error validating bug key pattern: %w", err)
    }
    if !matched {
        return fmt.Errorf("invalid bug key format %q: must match B-YYYY-MM-DD-xx (e.g., B-2026-01-04-01)", key)
    }

    return nil
}

// ValidateBugSeverity validates the bug severity enum
func ValidateBugSeverity(severity string) error {
    validSeverities := map[string]bool{
        "critical": true,
        "high":     true,
        "medium":   true,
        "low":      true,
    }

    if !validSeverities[severity] {
        return fmt.Errorf("invalid bug severity %q: must be one of critical, high, medium, low", severity)
    }

    return nil
}

// ValidateBugStatus validates the bug status enum
func ValidateBugStatus(status string) error {
    validStatuses := map[string]bool{
        "new":         true,
        "confirmed":   true,
        "in_progress": true,
        "resolved":    true,
        "closed":      true,
        "wont_fix":    true,
        "duplicate":   true,
    }

    if !validStatuses[status] {
        return fmt.Errorf("invalid bug status %q: must be one of new, confirmed, in_progress, resolved, closed, wont_fix, duplicate", status)
    }

    return nil
}

// ValidateReporterType validates the reporter type enum
func ValidateReporterType(reporterType string) error {
    validTypes := map[string]bool{
        "human":    true,
        "ai_agent": true,
    }

    if !validTypes[reporterType] {
        return fmt.Errorf("invalid reporter type %q: must be one of human, ai_agent", reporterType)
    }

    return nil
}
```

---

## 3. CLI Interface Design

### 3.1 Command Structure

Following shark's established patterns (similar to idea tracker):

```
shark bug
├── list [--status=<status>] [--severity=<severity>] [--category=<category>] [--epic=<epic>] [--feature=<feature>] [--json]
├── get <bug-key> [--json]
├── create <title> [flags] [--json]
├── update <bug-key> [flags] [--json]
├── delete <bug-key> [--force] [--hard] [--json]
├── confirm <bug-key> [--notes="..."] [--json]
├── resolve <bug-key> [--resolution="..."] [--json]
├── close <bug-key> [--notes="..."] [--json]
├── reopen <bug-key> [--notes="..."] [--json]
└── convert
    ├── task <bug-key> --epic=<epic> --feature=<feature> [--json]
    ├── feature <bug-key> --epic=<epic> [--json]
    └── epic <bug-key> [--json]
```

### 3.2 Command Details

#### 3.2.1 `shark bug list`

**Purpose**: List bugs with flexible filtering

**Syntax**:
```bash
shark bug list [--status=<status>] [--severity=<severity>] [--category=<category>]
               [--epic=<epic>] [--feature=<feature>] [--environment=<env>] [--json]
```

**Flags**:
- `--status`: Filter by status (new, confirmed, in_progress, resolved, closed, wont_fix, duplicate)
- `--severity`: Filter by severity (critical, high, medium, low)
- `--category`: Filter by category (backend, frontend, database, cli, performance, etc.)
- `--epic`: Filter by related epic key
- `--feature`: Filter by related feature key
- `--environment`: Filter by environment (production, staging, development, test)
- `--json`: JSON output

**Default Behavior**: Lists all non-closed bugs, sorted by severity then detected_at DESC

**Examples**:
```bash
# List all active bugs
shark bug list

# List critical bugs
shark bug list --severity=critical

# List bugs in production
shark bug list --environment=production

# List bugs related to epic E07
shark bug list --epic=E07

# AI agent querying bugs
shark bug list --status=new --json
```

**Output (table)**:
```
Key              Title                      Severity  Status    Category  Detected
B-2026-01-04-01  Database connection fails  critical  new       database  2026-01-04
B-2026-01-04-02  UI button not responding   high      confirmed frontend  2026-01-04
```

**Output (JSON)**:
```json
[
  {
    "id": 1,
    "key": "B-2026-01-04-01",
    "title": "Database connection fails",
    "severity": "critical",
    "status": "new",
    "category": "database",
    "detected_at": "2026-01-04T10:30:00Z",
    ...
  }
]
```

---

#### 3.2.2 `shark bug get`

**Purpose**: Get detailed information about a specific bug

**Syntax**:
```bash
shark bug get <bug-key> [--json]
```

**Examples**:
```bash
# Get bug details
shark bug get B-2026-01-04-01

# AI agent retrieving bug details
shark bug get B-2026-01-04-01 --json
```

**Output (text)**:
```
Bug: B-2026-01-04-01
Title: Database connection fails
Status: new
Severity: critical
Priority: 9
Category: database

Description: Connection pool exhausted after 30 seconds

Steps to Reproduce:
1. Start server with 10 workers
2. Run concurrent load test
3. Observe connection pool exhaustion

Expected Behavior: Connections should be released back to pool
Actual Behavior: Connections leak, pool exhausts

Error Message:
sqlalchemy.exc.TimeoutError: QueuePool limit of size 5 overflow 10 reached

Environment: production
OS: Linux 6.6.87.2-microsoft-standard-WSL2
Version: v1.2.3

Reporter: ai_agent (backend-test-agent)
Detected: 2026-01-04 10:30:00
Created: 2026-01-04 10:30:15
Updated: 2026-01-04 10:30:15
```

---

#### 3.2.3 `shark bug create`

**Purpose**: Create a new bug report

**Syntax**:
```bash
shark bug create <title> [flags] [--json]
```

**Required Flags**: None (title is positional argument)

**Optional Flags**:
- `--description=<text>`: Detailed description
- `--severity=<severity>`: critical, high, medium (default), low
- `--priority=<1-10>`: Priority (default: 5)
- `--category=<category>`: backend, frontend, database, cli, etc.
- `--steps=<text>`: Steps to reproduce
- `--expected=<text>`: Expected behavior
- `--actual=<text>`: Actual behavior
- `--error=<text>`: Error message or stack trace
- `--environment=<env>`: production, staging, development, test
- `--os=<os-info>`: Operating system details
- `--version=<version>`: Software version
- `--reporter=<id>`: Reporter ID (human username or agent ID)
- `--reporter-type=<type>`: human (default), ai_agent
- `--file=<path>`: Attachment file with additional details
- `--related-docs=<path1,path2>`: Related document paths
- `--epic=<epic-key>`: Related epic
- `--feature=<feature-key>`: Related feature
- `--task=<task-key>`: Related task
- `--depends-on=<bug-key1,bug-key2>`: Dependencies

**Use of `--file` Flag**:
The `--file` flag should be used when:
- Stack traces are too large for `--error` flag (> 500 characters)
- Multiple log files need to be referenced
- Screenshot references need to be documented
- Complex reproduction steps require a separate document
- Performance profiling data needs to be attached

**Examples**:

**Simple bug (human user)**:
```bash
shark bug create "Login button not working" \
  --severity=high \
  --category=frontend \
  --description="Users cannot click the login button on mobile devices" \
  --steps="1. Open app on mobile\n2. Navigate to login\n3. Tap login button" \
  --expected="Login form should appear" \
  --actual="Nothing happens, button doesn't respond"
```

**Detailed bug (AI agent)**:
```bash
shark bug create "Database query timeout in user search" \
  --severity=critical \
  --priority=9 \
  --category=database \
  --description="User search query times out after 5 seconds" \
  --steps="1. Execute user search with 1000+ results\n2. Observe timeout" \
  --expected="Results returned within 1 second" \
  --actual="Query times out after 5 seconds" \
  --error="sqlalchemy.exc.OperationalError: (psycopg2.OperationalError) connection timeout" \
  --environment=production \
  --os="Linux 6.6.87.2" \
  --version="v1.2.3" \
  --reporter=backend-test-agent \
  --reporter-type=ai_agent \
  --epic=E07 \
  --feature=E07-F20
```

**Bug with large stack trace (use file)**:
```bash
# First, save stack trace to file
cat > /tmp/stack-trace.txt <<'EOF'
Traceback (most recent call last):
  File "main.py", line 245, in process_request
    result = handler.execute()
  ... (500+ lines of stack trace)
EOF

# Then create bug with file reference
shark bug create "Application crashes on startup" \
  --severity=critical \
  --category=backend \
  --file=/tmp/stack-trace.txt \
  --description="Application crashes immediately after startup" \
  --environment=production
```

**Output**:
```
Created bug B-2026-01-04-03: Database query timeout in user search
```

---

#### 3.2.4 `shark bug update`

**Purpose**: Update existing bug properties

**Syntax**:
```bash
shark bug update <bug-key> [flags] [--json]
```

**Flags**: Same as create, plus:
- `--status=<status>`: Update status
- `--title=<new-title>`: Update title

**Examples**:
```bash
# Update severity
shark bug update B-2026-01-04-01 --severity=high

# Add reproduction steps
shark bug update B-2026-01-04-01 --steps="1. Start server\n2. Run test suite\n3. Observe error"

# Change status to confirmed
shark bug update B-2026-01-04-01 --status=confirmed

# AI agent updating bug with resolution
shark bug update B-2026-01-04-01 \
  --status=resolved \
  --resolution="Fixed connection pool leak in database.py:234" \
  --json
```

---

#### 3.2.5 `shark bug confirm`

**Purpose**: Confirm a bug (shortcut for status=confirmed)

**Syntax**:
```bash
shark bug confirm <bug-key> [--notes="..."] [--json]
```

**Examples**:
```bash
shark bug confirm B-2026-01-04-01 --notes="Reproduced on my machine"
```

---

#### 3.2.6 `shark bug resolve`

**Purpose**: Mark bug as resolved

**Syntax**:
```bash
shark bug resolve <bug-key> [--resolution="..."] [--json]
```

**Examples**:
```bash
shark bug resolve B-2026-01-04-01 --resolution="Fixed in commit abc123"
```

---

#### 3.2.7 `shark bug close`

**Purpose**: Close a bug (after verification)

**Syntax**:
```bash
shark bug close <bug-key> [--notes="..."] [--json]
```

**Examples**:
```bash
shark bug close B-2026-01-04-01 --notes="Verified fix in production"
```

---

#### 3.2.8 `shark bug reopen`

**Purpose**: Reopen a resolved/closed bug

**Syntax**:
```bash
shark bug reopen <bug-key> [--notes="..."] [--json]
```

**Examples**:
```bash
shark bug reopen B-2026-01-04-01 --notes="Bug still occurs in edge case"
```

---

#### 3.2.9 `shark bug delete`

**Purpose**: Delete a bug (soft or hard delete)

**Syntax**:
```bash
shark bug delete <bug-key> [--force] [--hard] [--json]
```

**Flags**:
- `--force`: Skip confirmation prompt
- `--hard`: Permanent deletion (default is soft delete = status change to duplicate/wont_fix)

**Examples**:
```bash
# Soft delete (mark as wont_fix)
shark bug delete B-2026-01-04-01

# Hard delete (permanent)
shark bug delete B-2026-01-04-01 --hard --force
```

---

#### 3.2.10 `shark bug convert`

**Purpose**: Convert a bug to a task, feature, or epic

**Syntax**:
```bash
shark bug convert task <bug-key> --epic=<epic> --feature=<feature> [--json]
shark bug convert feature <bug-key> --epic=<epic> [--json]
shark bug convert epic <bug-key> [--json]
```

**Examples**:
```bash
# Convert bug to task
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20

# Convert bug to feature
shark bug convert feature B-2026-01-04-02 --epic=E07

# Convert bug to epic
shark bug convert epic B-2026-01-04-03
```

**Output**:
```
Bug B-2026-01-04-01 converted to task T-E07-F20-015
```

---

## 4. User Journeys

### 4.1 AI Agent Reports Bug During Testing

**Scenario**: Backend test agent discovers a database connection leak during automated testing.

**Journey**:
```bash
# 1. Agent detects bug during test execution
# 2. Agent creates bug report with full details
shark bug create "Database connection pool exhaustion" \
  --severity=critical \
  --priority=9 \
  --category=database \
  --description="Connection pool exhausted after 30 seconds of load testing" \
  --steps="1. Start server with 10 workers\n2. Run concurrent load test (100 req/s)\n3. Monitor connection pool\n4. Observe exhaustion after 30s" \
  --expected="Connections released back to pool after request completion" \
  --actual="Connections leak, pool size grows until exhausted" \
  --error="sqlalchemy.exc.TimeoutError: QueuePool limit of size 5 overflow 10 reached" \
  --environment=test \
  --os="Linux 6.6.87.2" \
  --version="v1.2.3" \
  --reporter=backend-test-agent \
  --reporter-type=ai_agent \
  --epic=E07 \
  --feature=E07-F20 \
  --json

# Output: {"key": "B-2026-01-04-01", "title": "Database connection pool exhaustion", ...}

# 3. Agent logs bug key for tracking
# 4. Agent continues test execution
```

---

### 4.2 Human Developer Investigates Bug

**Scenario**: Developer sees bug report and investigates.

**Journey**:
```bash
# 1. List critical bugs
shark bug list --severity=critical

# Output:
# Key              Title                              Severity  Status  Category  Detected
# B-2026-01-04-01  Database connection pool exhaustion  critical  new     database  2026-01-04

# 2. Get full bug details
shark bug get B-2026-01-04-01

# 3. Reproduce bug locally
# ... (developer investigation)

# 4. Confirm bug
shark bug confirm B-2026-01-04-01 --notes="Reproduced locally, issue is in connection cleanup logic"

# 5. Developer converts bug to task for tracking
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20

# Output: Bug B-2026-01-04-01 converted to task T-E07-F20-015

# 6. Developer works on task
shark task start T-E07-F20-015

# 7. After fix, developer marks task complete
shark task complete T-E07-F20-015 --notes="Fixed connection leak in database.py:234"

# 8. Bug is automatically marked as resolved via conversion link
```

---

### 4.3 QA Engineer Verifies Fix

**Scenario**: QA engineer verifies bug fix after deployment.

**Journey**:
```bash
# 1. List resolved bugs for verification
shark bug list --status=resolved

# Output:
# Key              Title                              Severity  Status    Category  Resolved
# B-2026-01-04-01  Database connection pool exhaustion  critical  resolved  database  2026-01-04

# 2. Get bug details
shark bug get B-2026-01-04-01

# 3. Review resolution
# Resolution: Fixed connection leak in database.py:234

# 4. Run verification tests
# ... (QA testing)

# 5a. If fix verified, close bug
shark bug close B-2026-01-04-01 --notes="Verified in production, connection pool stable"

# 5b. If bug still occurs, reopen
shark bug reopen B-2026-01-04-01 --notes="Bug still occurs under high load (200+ req/s)"
```

---

### 4.4 Product Manager Reviews Bug Backlog

**Scenario**: PM reviews bugs to prioritize for next sprint.

**Journey**:
```bash
# 1. List all active bugs
shark bug list --json | jq 'group_by(.severity) | map({severity: .[0].severity, count: length})'

# Output:
# [
#   {"severity": "critical", "count": 2},
#   {"severity": "high", "count": 5},
#   {"severity": "medium", "count": 12},
#   {"severity": "low", "count": 8}
# ]

# 2. Review critical bugs
shark bug list --severity=critical

# 3. Check bugs in specific epic
shark bug list --epic=E07

# 4. Decide to convert bug to feature for major refactoring
shark bug convert feature B-2026-01-04-05 --epic=E07

# Output: Bug B-2026-01-04-05 converted to feature E07-F25
```

---

## 5. File Storage vs Inline Storage

### 5.1 When to Use `--file` Flag

Use `--file` when:

1. **Large Stack Traces**: Error messages exceed 500 characters
   ```bash
   shark bug create "App crashes on startup" \
     --file=/tmp/crash-stacktrace.txt
   ```

2. **Multiple Log Files**: Bug requires multiple log file references
   ```bash
   # Create attachment file with log references
   cat > /tmp/bug-logs.md <<EOF
   ## Log Files
   - Application log: /var/log/app/error.log (lines 1234-1890)
   - Database log: /var/log/postgres/error.log (lines 456-789)
   - System log: /var/log/syslog (lines 9876-9999)

   ## Stack Trace
   [... full stack trace ...]
   EOF

   shark bug create "Multi-system failure" --file=/tmp/bug-logs.md
   ```

3. **Screenshot References**: Bug involves UI issues with screenshots
   ```bash
   cat > /tmp/ui-bug-details.md <<EOF
   ## Screenshots
   - Before: screenshots/bug-before.png
   - After: screenshots/bug-after.png
   - Expected: screenshots/expected-ui.png

   ## Browser Console Errors
   [... console output ...]
   EOF

   shark bug create "UI rendering issue" --file=/tmp/ui-bug-details.md
   ```

4. **Performance Profiling Data**: Bug includes profiling output
   ```bash
   shark bug create "Slow query performance" \
     --file=/tmp/profiling-report.txt \
     --category=performance
   ```

### 5.2 When to Use Inline Fields

Use inline flags (`--description`, `--steps`, `--error`, etc.) when:

1. **Concise Bug Reports**: All details fit comfortably in CLI flags
2. **Quick AI Agent Reports**: Automated bug detection with structured data
3. **Simple Reproduction Steps**: 3-5 step reproduction process
4. **Short Error Messages**: Error text under 500 characters

---

## 6. Integration with Tasks/Features/Epics

### 6.1 Relationship Types

**1. Bug relates to existing work** (via `--epic`, `--feature`, `--task`):
```bash
# Bug found in specific task
shark bug create "Test fails in E07-F20-001" \
  --task=T-E07-F20-001 \
  --severity=high

# Bug affects entire feature
shark bug create "Feature E07-F20 breaks on mobile" \
  --feature=E07-F20 \
  --severity=critical

# Bug impacts epic scope
shark bug create "E07 authentication design flaw" \
  --epic=E07 \
  --severity=high
```

**2. Bug converts to task** (bug becomes work item):
```bash
# Convert bug to task for tracking
shark bug convert task B-2026-01-04-01 --epic=E07 --feature=E07-F20

# Result:
# - Bug status changes to 'converted'
# - Task T-E07-F20-015 created with bug details
# - Bug tracks conversion: converted_to_type='task', converted_to_key='T-E07-F20-015'
```

**3. Bug converts to feature** (bug requires feature-level work):
```bash
# Convert bug to feature (major refactoring needed)
shark bug convert feature B-2026-01-04-02 --epic=E07

# Result:
# - Bug status changes to 'converted'
# - Feature E07-F25 created
# - Bug tracks conversion
```

**4. Bug converts to epic** (bug reveals systemic issue):
```bash
# Convert bug to epic (architectural change needed)
shark bug convert epic B-2026-01-04-03

# Result:
# - Bug status changes to 'converted'
# - Epic E15 created
# - Bug tracks conversion
```

### 6.2 Conversion Workflow

**Conversion Process**:
1. Validate bug exists and is not already converted
2. Create target entity (epic/feature/task) with bug details:
   - Title copied from bug
   - Description includes bug details + reproduction steps
   - Priority mapped from bug priority
3. Mark bug as converted with tracking fields
4. Output confirms conversion

**Reverse Link** (Task → Bug):
When a task is created from a bug, the task description includes:
```markdown
## Related Bug
This task resolves bug B-2026-01-04-01

[Original bug details copied here]
```

---

## 7. Repository Layer

### 7.1 Interface

```go
package repository

// BugRepository handles CRUD operations for bugs
type BugRepository struct {
    db *DB
}

// BugFilter represents filtering options for listing bugs
type BugFilter struct {
    Status         *models.BugStatus
    Severity       *models.BugSeverity
    Category       *string
    Environment    *string
    RelatedToEpic  *string
    RelatedToFeature *string
    RelatedToTask  *string
}

// Methods
func (r *BugRepository) Create(ctx context.Context, bug *models.Bug) error
func (r *BugRepository) GetByID(ctx context.Context, id int64) (*models.Bug, error)
func (r *BugRepository) GetByKey(ctx context.Context, key string) (*models.Bug, error)
func (r *BugRepository) List(ctx context.Context, filter *BugFilter) ([]*models.Bug, error)
func (r *BugRepository) Update(ctx context.Context, bug *models.Bug) error
func (r *BugRepository) Delete(ctx context.Context, id int64) error
func (r *BugRepository) MarkAsConverted(ctx context.Context, bugID int64, convertedToType, convertedToKey string) error
func (r *BugRepository) GetNextSequenceForDate(ctx context.Context, dateStr string) (int, error)
func (r *BugRepository) UpdateStatus(ctx context.Context, bugID int64, status models.BugStatus, notes *string) error
```

---

## 8. Testing Strategy

### 8.1 Repository Tests (Use Real DB)

**File**: `internal/repository/bug_repository_test.go`

Test coverage:
- Create bug with all fields
- Create bug with minimal fields
- Get bug by ID
- Get bug by key
- List bugs with filters (status, severity, category, relationships)
- Update bug fields
- Delete bug (soft and hard)
- Mark as converted
- Generate next sequence for date
- Update status with timestamp triggers

**Pattern**:
```go
func TestBugRepository_Create(t *testing.T) {
    ctx := context.Background()
    database := test.GetTestDB()
    db := NewDB(database)
    repo := NewBugRepository(db)

    // Clean up existing test data
    _, _ = database.ExecContext(ctx, "DELETE FROM bugs WHERE key LIKE 'B-TEST-%'")

    // Create bug
    bug := &models.Bug{...}
    err := repo.Create(ctx, bug)
    assert.NoError(t, err)

    // Cleanup
    defer database.ExecContext(ctx, "DELETE FROM bugs WHERE id = ?", bug.ID)
}
```

### 8.2 CLI Command Tests (Use Mocks)

**File**: `internal/cli/commands/bug_test.go`

Test coverage:
- Create bug with various flag combinations
- List bugs with filters
- Get bug details
- Update bug fields
- Confirm/resolve/close/reopen bug
- Delete bug with confirmation
- Convert bug to task/feature/epic
- JSON output formatting
- Error handling (invalid keys, missing bugs, validation errors)

**Pattern**:
```go
type MockBugRepository struct {
    CreateFunc func(ctx context.Context, bug *models.Bug) error
    GetByKeyFunc func(ctx context.Context, key string) (*models.Bug, error)
    // ... other methods
}

func TestBugCreateCommand(t *testing.T) {
    mockRepo := &MockBugRepository{
        CreateFunc: func(ctx context.Context, bug *models.Bug) error {
            bug.ID = 123
            return nil
        },
    }

    // Test command execution with mock
    // Verify correct arguments passed to repo
    // Verify output format
}
```

---

## 9. Migration Plan

### 9.1 Database Migration

**File**: `internal/db/db.go` (in `runMigrations()`)

```go
func runMigrations(db *sql.DB) error {
    // ... existing migrations ...

    // Migration: Add bugs table
    if err := addBugsTable(db); err != nil {
        return err
    }

    return nil
}

func addBugsTable(db *sql.DB) error {
    // Check if bugs table exists
    var tableName string
    err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='bugs'").Scan(&tableName)
    if err == nil {
        // Table exists, skip migration
        return nil
    }

    // Create bugs table (full schema from section 2.1)
    _, err = db.Exec(`CREATE TABLE IF NOT EXISTS bugs (...)`)
    return err
}
```

### 9.2 Implementation Phases

**Phase 1: Core Infrastructure**
- Database schema creation
- Bug model with validation
- Bug repository with CRUD operations
- Repository tests

**Phase 2: Basic CLI Commands**
- `bug create` with essential flags
- `bug list` with basic filtering
- `bug get` for details
- `bug update` for modifications
- CLI command tests with mocks

**Phase 3: Status Management**
- `bug confirm`
- `bug resolve`
- `bug close`
- `bug reopen`
- `bug delete`
- Status transition tests

**Phase 4: Advanced Features**
- `bug convert` (task, feature, epic)
- Conversion tracking
- Integration with task/feature/epic repositories
- Conversion tests

**Phase 5: Documentation & Polish**
- CLI help text
- Usage examples
- Error message refinement
- Performance optimization

---

## 10. Future Enhancements

### 10.1 Bug Analytics

Commands to analyze bug patterns:
```bash
# Bug metrics
shark bug stats --epic=E07
shark bug stats --category=backend --group-by=severity

# Time-to-resolution analysis
shark bug metrics --metric=resolution-time --group-by=severity
```

### 10.2 Bug Templates

Pre-defined templates for common bug types:
```bash
shark bug create --template=crash "App crashes on startup"
shark bug create --template=performance "Slow query in user search"
```

### 10.3 Bug Attachments

Support for binary file attachments (screenshots, logs):
```bash
shark bug attach B-2026-01-04-01 screenshot.png
shark bug attach B-2026-01-04-01 error.log --type=log
```

### 10.4 Bug Notifications

Alert stakeholders when bugs are created/resolved:
```bash
shark bug notify B-2026-01-04-01 --to=team@example.com
```

---

## 11. Summary

This design provides a comprehensive bug tracking feature for Shark Task Manager that:

1. **Follows Shark Patterns**: Uses established conventions from idea tracker
2. **Supports Both User Types**: CLI-first design works for AI agents and humans
3. **Flexible Storage**: Inline storage for typical bugs, file references for complex cases
4. **Rich Context**: Captures technical details, environment info, and relationships
5. **Task Integration**: Converts bugs to tasks/features/epics for workflow integration
6. **Test Coverage**: Clear testing strategy with mocks for CLI, real DB for repositories

**Key Differentiators from Idea Tracker**:
- Bug-specific fields: severity, steps_to_reproduce, expected/actual behavior, error messages
- Environment tracking: os_info, version, environment
- Reporter tracking: reporter_type, reporter_id for AI agent attribution
- Status workflow: new → confirmed → in_progress → resolved → closed
- Attachment support: `--file` flag for large stack traces, logs, screenshots

**Recommended Epic**: E08 (Enhancements) or create new epic E10 (Bug Tracking System)

---

## Appendix A: Key Format

**Bug Key Format**: `B-YYYY-MM-DD-xx`

**Examples**:
- `B-2026-01-04-01` - First bug on January 4, 2026
- `B-2026-01-04-15` - Fifteenth bug on January 4, 2026
- `B-2026-12-31-99` - 99th bug on December 31, 2026 (max per day)

**Pattern**: `^B-\d{4}-\d{2}-\d{2}-\d{2}$`

**Sequence**: Auto-incremented per day, max 99 bugs per day

---

## Appendix B: Severity vs Priority

**Severity** (Impact on system):
- **critical**: System down, data loss, security breach
- **high**: Major functionality broken, workarounds difficult
- **medium**: Functionality impaired, workarounds available
- **low**: Minor issue, cosmetic problem

**Priority** (Urgency to fix, 1-10 scale):
- **9-10**: Drop everything, fix immediately
- **7-8**: Fix in current sprint
- **5-6**: Fix in next sprint
- **3-4**: Fix when capacity allows
- **1-2**: Backlog, fix if time permits

**Relationship**:
- High severity often means high priority, but not always
- Low severity bug in critical path may have high priority
- High severity bug with workaround may have medium priority

---

## Appendix C: Status Lifecycle

```
┌─────┐
│ new │ ────> [Created]
└──┬──┘
   │
   v
┌───────────┐
│ confirmed │ ────> [Reproduced/Verified]
└──┬────────┘
   │
   v
┌─────────────┐
│ in_progress │ ────> [Being Fixed]
└──┬──────────┘
   │
   v
┌──────────┐
│ resolved │ ────> [Fix Deployed]
└──┬───────┘
   │
   v
┌────────┐
│ closed │ ────> [Fix Verified]
└────────┘

Alternative Paths:
- Any status → wont_fix (won't be fixed)
- Any status → duplicate (duplicate of another bug)
- closed/resolved → new (reopened)
```

---

**End of Document**
