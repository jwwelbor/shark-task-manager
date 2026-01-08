# Bug Tracker CLI - Usage Examples

This document provides practical examples of using the bug tracker CLI in various scenarios.

---

## Table of Contents

1. [Quick Start Examples](#quick-start-examples)
2. [AI Agent Integration](#ai-agent-integration)
3. [Human Developer Workflows](#human-developer-workflows)
4. [QA Engineer Workflows](#qa-engineer-workflows)
5. [Product Manager Workflows](#product-manager-workflows)
6. [Advanced Filtering](#advanced-filtering)
7. [Conversion Workflows](#conversion-workflows)
8. [File Attachment Scenarios](#file-attachment-scenarios)

---

## Quick Start Examples

### Create a Simple Bug

```bash
shark bug create "Login button not working"
```

Output:
```
Created bug B-2026-01-04-01: Login button not working
```

### Create a Bug with Severity

```bash
shark bug create "Database connection fails" --severity=critical
```

### List All Active Bugs

```bash
shark bug list
```

Output:
```
Key              Title                      Severity  Status    Category  Detected
B-2026-01-04-01  Login button not working   medium    new       -         2026-01-04
B-2026-01-04-02  Database connection fails  critical  new       -         2026-01-04
```

### Get Bug Details

```bash
shark bug get B-2026-01-04-01
```

---

## AI Agent Integration

### Automated Test Agent Reports Bug

```bash
#!/bin/bash
# Script: automated-test-agent.sh
# Purpose: AI agent running automated tests

# Detect bug during test execution
if test_fails; then
  # Create bug report with full context
  BUG_KEY=$(shark bug create "User authentication test failure" \
    --severity=high \
    --priority=8 \
    --category=backend \
    --description="Authentication endpoint returns 500 instead of 401 for invalid credentials" \
    --steps="1. Send POST to /api/auth/login with invalid credentials\n2. Observe response code" \
    --expected="HTTP 401 Unauthorized with error message" \
    --actual="HTTP 500 Internal Server Error" \
    --error="AttributeError: 'NoneType' object has no attribute 'check_password'" \
    --environment=test \
    --os="$(uname -a)" \
    --version="$(git describe --tags)" \
    --reporter=automated-test-agent \
    --reporter-type=ai_agent \
    --epic=E07 \
    --feature=E07-F01 \
    --json | jq -r '.key')

  echo "Bug reported: $BUG_KEY"

  # Log to test results
  echo "$BUG_KEY: User authentication test failure" >> test-failures.log
fi
```

### Performance Monitoring Agent

```bash
#!/bin/bash
# Script: performance-monitor-agent.sh
# Purpose: AI agent monitoring performance metrics

# Detect slow query
QUERY_TIME=$(measure_query_time)

if [ "$QUERY_TIME" -gt 5000 ]; then
  # Save query execution plan to file
  cat > /tmp/slow-query-plan.txt <<EOF
Query: SELECT * FROM users WHERE email LIKE '%@example.com' ORDER BY created_at DESC
Execution Time: ${QUERY_TIME}ms
Execution Plan:
$(get_query_plan)
EOF

  # Create bug with file attachment
  shark bug create "User search query exceeds 5s threshold" \
    --severity=high \
    --priority=7 \
    --category=performance \
    --description="User search query takes ${QUERY_TIME}ms, exceeding 5s SLA" \
    --file=/tmp/slow-query-plan.txt \
    --environment=production \
    --reporter=performance-monitor-agent \
    --reporter-type=ai_agent \
    --epic=E07 \
    --feature=E07-F05
fi
```

### Backend API Agent Detects Error Pattern

```bash
#!/bin/bash
# Script: api-error-monitor.sh
# Purpose: AI agent monitoring API error logs

# Detect error spike
ERROR_COUNT=$(grep "500 Internal Server Error" /var/log/api/error.log | wc -l)

if [ "$ERROR_COUNT" -gt 100 ]; then
  # Extract error samples
  cat > /tmp/api-error-samples.txt <<EOF
Error spike detected: ${ERROR_COUNT} occurrences in last hour

Sample errors:
$(grep "500 Internal Server Error" /var/log/api/error.log | head -10)

Error pattern:
$(grep "500 Internal Server Error" /var/log/api/error.log | awk '{print $5}' | sort | uniq -c | sort -rn)
EOF

  # Create critical bug
  shark bug create "API error rate spike: ${ERROR_COUNT} 500 errors/hour" \
    --severity=critical \
    --priority=10 \
    --category=backend \
    --file=/tmp/api-error-samples.txt \
    --environment=production \
    --os="$(uname -a)" \
    --version="$(cat /app/VERSION)" \
    --reporter=api-error-monitor \
    --reporter-type=ai_agent \
    --json
fi
```

---

## Human Developer Workflows

### Developer Creates Bug During Development

```bash
# Scenario: Developer discovers bug while working on feature

# Create bug with detailed context
shark bug create "User profile image not saving" \
  --severity=medium \
  --priority=6 \
  --category=backend \
  --description="Profile image upload succeeds but image URL not saved to database" \
  --steps="1. Login as user\n2. Navigate to profile settings\n3. Upload new profile image\n4. Save settings\n5. Refresh page\n6. Image reverts to default" \
  --expected="Uploaded image persists after refresh" \
  --actual="Image URL not saved, reverts to default" \
  --error="No error in logs, but database UPDATE returns 0 rows affected" \
  --environment=development \
  --task=T-E07-F03-012 \
  --reporter=john-dev

# Output: Created bug B-2026-01-04-05: User profile image not saving
```

### Developer Investigates Existing Bug

```bash
# List critical bugs
shark bug list --severity=critical

# Get details
shark bug get B-2026-01-04-02

# Reproduce locally
# ... (developer reproduces bug)

# Confirm bug
shark bug confirm B-2026-01-04-02 \
  --notes="Reproduced locally. Root cause: connection pool not releasing connections after query timeout."

# Convert to task for tracking
shark bug convert task B-2026-01-04-02 --epic=E07 --feature=E07-F20

# Output: Bug B-2026-01-04-02 converted to task T-E07-F20-018

# Start working on fix
shark task start T-E07-F20-018
```

### Developer Fixes Bug and Updates Status

```bash
# After implementing fix
shark bug resolve B-2026-01-04-02 \
  --resolution="Fixed connection pool leak in database/pool.py:145. Added proper cleanup in finally block."

# Or update via task completion (if converted)
shark task complete T-E07-F20-018 \
  --notes="Fixed connection pool leak. Added unit test to prevent regression."
```

---

## QA Engineer Workflows

### QA Triages Incoming Bugs

```bash
# List new bugs
shark bug list --status=new

# Review each bug
shark bug get B-2026-01-04-03

# Attempt to reproduce
# ... (QA reproduction steps)

# If reproduced, confirm
shark bug confirm B-2026-01-04-03 \
  --notes="Reproduced on staging. Also occurs on Chrome 120, Firefox 115. Edge unaffected."

# Update with additional details
shark bug update B-2026-01-04-03 \
  --description="Login button unresponsive on Chrome/Firefox mobile browsers. Edge works correctly." \
  --steps="1. Open site on Chrome mobile (Android 14)\n2. Navigate to /login\n3. Tap login button\n4. Observe no response\n5. Switch to Edge\n6. Login works normally"

# If cannot reproduce, mark as duplicate or wont_fix
shark bug update B-2026-01-04-04 --status=duplicate \
  --notes="Duplicate of B-2026-01-03-12"
```

### QA Verifies Resolved Bugs

```bash
# List resolved bugs awaiting verification
shark bug list --status=resolved

# Get bug details
shark bug get B-2026-01-04-02

# Check resolution notes
# Resolution: Fixed connection pool leak in database/pool.py:145

# Run verification tests
# ... (QA testing in staging/production)

# If verified, close bug
shark bug close B-2026-01-04-02 \
  --notes="Verified in staging. Connection pool stable after 1 hour load test (500 req/s). Ready for production."

# If bug persists, reopen
shark bug reopen B-2026-01-04-02 \
  --notes="Bug still occurs under extreme load (1000+ req/s). Connection pool still exhausts after 2 minutes."
```

### QA Reports Bug from User Feedback

```bash
# User reported bug via support ticket
shark bug create "App crashes when viewing large PDF files" \
  --severity=high \
  --priority=8 \
  --category=frontend \
  --description="Mobile app crashes when opening PDF files larger than 50MB" \
  --steps="1. Download PDF file (>50MB)\n2. Open file in app\n3. Observe crash\n4. Restart app\n5. Attempt to open again\n6. Crash repeats" \
  --expected="PDF renders correctly regardless of file size" \
  --actual="App crashes with out-of-memory error" \
  --error="OutOfMemoryError: Failed to allocate 128MB for PDF rendering" \
  --environment=production \
  --os="Android 14, Samsung Galaxy S23" \
  --version="v2.1.5" \
  --reporter=qa-team \
  --epic=E05
```

---

## Product Manager Workflows

### PM Reviews Bug Dashboard

```bash
# Get bug summary by severity
shark bug list --json | jq 'group_by(.severity) | map({severity: .[0].severity, count: length})'

# Output:
# [
#   {"severity": "critical", "count": 2},
#   {"severity": "high", "count": 8},
#   {"severity": "medium", "count": 15},
#   {"severity": "low", "count": 12}
# ]

# List critical bugs
shark bug list --severity=critical

# Review specific epic bugs
shark bug list --epic=E07
```

### PM Prioritizes Bugs for Sprint

```bash
# Review high-priority bugs
shark bug list --severity=high --json | jq '.[] | {key, title, category, detected_at}'

# Decide to convert bug to feature for architectural fix
shark bug get B-2026-01-04-08

# Bug requires major refactoring, convert to feature
shark bug convert feature B-2026-01-04-08 --epic=E07

# Output: Bug B-2026-01-04-08 converted to feature E07-F25

# Plan feature for next sprint
shark feature update E07-F25 --execution-order=1
```

### PM Marks Bug as Won't Fix

```bash
# Review low-priority bug
shark bug get B-2026-01-04-15

# Decide not to fix (minor cosmetic issue, low user impact)
shark bug update B-2026-01-04-15 --status=wont_fix \
  --resolution="Low priority cosmetic issue. Does not affect functionality. Will address in future UI redesign."
```

---

## Advanced Filtering

### Filter by Multiple Criteria

```bash
# Critical bugs in production backend
shark bug list --severity=critical --category=backend --environment=production

# High-priority bugs in specific feature
shark bug list --severity=high --feature=E07-F20

# Bugs reported by AI agents
shark bug list --json | jq '.[] | select(.reporter_type == "ai_agent")'

# Bugs created today
TODAY=$(date +%Y-%m-%d)
shark bug list --json | jq --arg today "$TODAY" '.[] | select(.detected_at | startswith($today))'
```

### Complex Queries with jq

```bash
# Bugs by category
shark bug list --json | jq 'group_by(.category) | map({category: .[0].category, count: length}) | sort_by(.count) | reverse'

# Average time to resolution
shark bug list --status=resolved --json | \
  jq '[.[] | select(.resolved_at != null) |
      (((.resolved_at | fromdateiso8601) - (.detected_at | fromdateiso8601)) / 3600)] |
      add / length'

# Bugs with no category
shark bug list --json | jq '.[] | select(.category == null) | {key, title}'

# Top 10 oldest unresolved critical bugs
shark bug list --severity=critical --json | \
  jq 'sort_by(.detected_at) | .[0:10] | .[] | {key, title, detected_at}'
```

---

## Conversion Workflows

### Convert Bug to Task

```bash
# Standard conversion
shark bug convert task B-2026-01-04-10 --epic=E07 --feature=E07-F15

# Output: Bug B-2026-01-04-10 converted to task T-E07-F15-022

# Verify conversion
shark bug get B-2026-01-04-10 | grep "Converted to"
# Converted to: task T-E07-F15-022

# Work on task
shark task start T-E07-F15-022
shark task complete T-E07-F15-022 --notes="Fixed bug by adding input validation"
```

### Convert Bug to Feature (Major Refactoring)

```bash
# Bug reveals need for architectural change
shark bug get B-2026-01-04-12

# Convert to feature
shark bug convert feature B-2026-01-04-12 --epic=E07

# Output: Bug B-2026-01-04-12 converted to feature E07-F30

# Plan feature
shark feature update E07-F30 --description="Complete rewrite of authentication system to fix security vulnerabilities"
```

### Convert Bug to Epic (Systemic Issue)

```bash
# Bug reveals systemic problem requiring new epic
shark bug get B-2026-01-04-20

# Convert to epic
shark bug convert epic B-2026-01-04-20

# Output: Bug B-2026-01-04-20 converted to epic E15

# Plan epic
shark epic update E15 --description="System-wide security audit and remediation"
```

---

## File Attachment Scenarios

### Large Stack Trace

```bash
# Capture full stack trace from crash
python app.py 2> /tmp/crash-stacktrace.txt

# Create bug with stack trace file
shark bug create "Application crashes on concurrent requests" \
  --severity=critical \
  --category=backend \
  --description="App crashes when handling 50+ concurrent requests" \
  --file=/tmp/crash-stacktrace.txt \
  --environment=production
```

### Multiple Log Files

```bash
# Create comprehensive bug report with multiple log references
cat > /tmp/bug-report.md <<'EOF'
# Multi-System Failure Analysis

## Timeline
- 10:30:00 - First error in application log
- 10:30:15 - Database connection pool exhausted
- 10:30:45 - Load balancer health check fails
- 10:31:00 - System recovery initiated

## Log Files

### Application Log
File: /var/log/app/error.log
Lines: 1234-1890

```
[10:30:00] ERROR: Connection pool exhausted
[10:30:01] ERROR: Failed to acquire connection after 5s timeout
[10:30:02] ERROR: Request handler crashed: NoneType object has no attribute 'execute'
```

### Database Log
File: /var/log/postgres/postgresql.log
Lines: 456-789

```
[10:30:00] FATAL: too many connections for role "app_user"
[10:30:15] LOG: connection received: host=10.0.1.5 port=52341
[10:30:15] FATAL: remaining connection slots are reserved for non-replication superuser connections
```

### Load Balancer Log
File: /var/log/nginx/error.log
Lines: 9876-9999

```
[10:30:45] [error] 1234#0: upstream timed out (110: Connection timed out)
[10:30:50] [error] 1234#0: no live upstreams while connecting to upstream
```

## Reproduction Steps
1. Start application with default connection pool (size=10)
2. Send 50 concurrent requests
3. Observe connection pool exhaustion
4. System fails to recover

## Root Cause Analysis
Connection pool too small for production load. Connections not released after errors.
EOF

shark bug create "Multi-system cascade failure under load" \
  --severity=critical \
  --priority=10 \
  --category=backend \
  --file=/tmp/bug-report.md \
  --environment=production
```

### Screenshot References

```bash
# UI bug with visual evidence
cat > /tmp/ui-bug-details.md <<'EOF'
# UI Button Rendering Issue

## Visual Evidence
- **Before**: screenshots/2026-01-04/button-broken.png
- **Expected**: screenshots/2026-01-04/button-expected.png
- **Browser Console**: screenshots/2026-01-04/console-errors.png

## Browser Details
- Chrome 120.0.6099.109
- Viewport: 1920x1080
- Device Pixel Ratio: 2

## Console Errors
```
Uncaught TypeError: Cannot read property 'addEventListener' of null
    at button-handler.js:45
    at HTMLDocument.<anonymous> (main.js:12)
```

## Reproduction
1. Open https://app.example.com/dashboard
2. Resize window to < 768px (mobile breakpoint)
3. Click hamburger menu
4. Observe button not rendering
EOF

shark bug create "Mobile menu button not rendering below 768px" \
  --severity=medium \
  --category=frontend \
  --file=/tmp/ui-bug-details.md \
  --environment=production
```

### Performance Profiling Data

```bash
# Run profiler and save output
python -m cProfile -o /tmp/profile.stats slow_function.py

# Create human-readable report
python -c "import pstats; p = pstats.Stats('/tmp/profile.stats'); p.sort_stats('cumulative').print_stats(50)" > /tmp/profile-report.txt

# Create bug with profiling data
shark bug create "User search function takes 8+ seconds" \
  --severity=high \
  --priority=8 \
  --category=performance \
  --description="User search query extremely slow with large dataset (100k+ users)" \
  --file=/tmp/profile-report.txt \
  --environment=production \
  --feature=E07-F05
```

---

## Script Integration Examples

### CI/CD Pipeline Integration

```bash
#!/bin/bash
# .github/workflows/test-and-report-bugs.sh

set -e

echo "Running test suite..."
if ! pytest --json-report --json-report-file=test-results.json; then
  echo "Tests failed, creating bug reports..."

  # Parse test failures and create bugs
  jq -c '.tests[] | select(.outcome == "failed")' test-results.json | while read test; do
    TEST_NAME=$(echo "$test" | jq -r '.nodeid')
    ERROR_MSG=$(echo "$test" | jq -r '.call.longrepr')

    # Create bug for each test failure
    shark bug create "Test failure: $TEST_NAME" \
      --severity=high \
      --category=test \
      --description="Automated test failed in CI/CD pipeline" \
      --error="$ERROR_MSG" \
      --environment=test \
      --reporter=ci-cd-pipeline \
      --reporter-type=ai_agent \
      --json
  done

  exit 1
fi

echo "All tests passed"
```

### Monitoring Integration

```bash
#!/bin/bash
# /usr/local/bin/monitor-and-report.sh

# Monitor application metrics and create bugs for anomalies

# Check error rate
ERROR_RATE=$(curl -s http://localhost:9090/metrics | grep error_rate | awk '{print $2}')

if (( $(echo "$ERROR_RATE > 0.05" | bc -l) )); then
  shark bug create "Error rate spike: ${ERROR_RATE}% (threshold: 5%)" \
    --severity=critical \
    --category=backend \
    --environment=production \
    --reporter=monitoring-agent \
    --reporter-type=ai_agent
fi

# Check response time
AVG_RESPONSE_TIME=$(curl -s http://localhost:9090/metrics | grep avg_response_time | awk '{print $2}')

if (( $(echo "$AVG_RESPONSE_TIME > 1000" | bc -l) )); then
  shark bug create "Response time degradation: ${AVG_RESPONSE_TIME}ms (threshold: 1000ms)" \
    --severity=high \
    --category=performance \
    --environment=production \
    --reporter=monitoring-agent \
    --reporter-type=ai_agent
fi
```

---

**End of Examples**
