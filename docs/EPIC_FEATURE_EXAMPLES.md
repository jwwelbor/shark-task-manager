# Epic & Feature Query Examples

Real-world examples and scenarios for using epic and feature query commands.

## Table of Contents

- [Basic Examples](#basic-examples)
- [Developer Workflows](#developer-workflows)
- [AI Agent Workflows](#ai-agent-workflows)
- [Reporting Scenarios](#reporting-scenarios)
- [Troubleshooting Scenarios](#troubleshooting-scenarios)
- [Advanced JQ Patterns](#advanced-jq-patterns)
- [Shell Script Integration](#shell-script-integration)

## Basic Examples

### Example 1: List All Epics

View all epics in your project with their progress.

```bash
pm epic list
```

**Output:**

```
┌─────┬──────────────────────────────────────┬───────────┬──────────┬──────────┐
│ Key │ Title                                │ Status    │ Progress │ Priority │
├─────┼──────────────────────────────────────┼───────────┼──────────┼──────────┤
│ E04 │ Task Management CLI Core             │ active    │    65.3% │ high     │
│ E05 │ Task Management CLI Capabilities     │ draft     │     0.0% │ medium   │
│ E06 │ Advanced Workflow Automation         │ draft     │     0.0% │ low      │
└─────┴──────────────────────────────────────┴───────────┴──────────┴──────────┘
```

**Interpretation:**
- E04 is actively being worked on (65.3% complete)
- E05 and E06 are in draft stage (not started)
- E04 has high priority, should be completed first

### Example 2: Get Epic Details

Drill into a specific epic to see its features.

```bash
pm epic get E04
```

**Output:**

```
Epic: E04 - Task Management CLI Core
Status: active
Priority: high
Business Value: High developer productivity
Overall Progress: 65.3%

Description:
Core CLI functionality for task management including database operations,
CLI framework, task lifecycle, and epic/feature queries.

Features:
┌───────────┬──────────────────────────────────────┬───────────┬──────────┬────────┐
│ Key       │ Title                                │ Status    │ Progress │ Tasks  │
├───────────┼──────────────────────────────────────┼───────────┼──────────┼────────┤
│ E04-F01   │ Database Foundation                  │ completed │   100.0% │     6  │
│ E04-F02   │ CLI Infrastructure                   │ completed │   100.0% │     6  │
│ E04-F03   │ Task Lifecycle Operations            │ completed │   100.0% │     4  │
│ E04-F04   │ Epic & Feature Queries               │ active    │    33.3% │     6  │
│ E04-F05   │ File Path Management                 │ draft     │     0.0% │     6  │
│ E04-F06   │ Task Creation & Templating           │ draft     │     0.0% │     4  │
│ E04-F07   │ Database Initialization & Sync       │ draft     │     0.0% │     7  │
└───────────┴──────────────────────────────────────┴───────────┴──────────┴────────┘

Total Features: 7
Total Tasks: 39
```

**Interpretation:**
- 3 features complete (F01, F02, F03)
- 1 feature in progress (F04 at 33.3%)
- 3 features not started (F05, F06, F07)
- Total of 39 tasks across all features

### Example 3: List All Features

See all features across all epics.

```bash
pm feature list
```

**Output:**

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F01   │ Database Foundation                  │ E04  │ completed │   100.0% │     6 │
│ E04-F02   │ CLI Infrastructure                   │ E04  │ completed │   100.0% │     6 │
│ E04-F03   │ Task Lifecycle Operations            │ E04  │ completed │   100.0% │     4 │
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
│ E04-F05   │ File Path Management                 │ E04  │ draft     │     0.0% │     6 │
│ E05-F01   │ Status Dashboard                     │ E05  │ draft     │     0.0% │     5 │
│ E05-F02   │ Dependency Management                │ E05  │ draft     │     0.0% │     4 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

### Example 4: Filter Features by Epic

See only features in a specific epic.

```bash
pm feature list --epic=E04
```

**Output:**

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F01   │ Database Foundation                  │ E04  │ completed │   100.0% │     6 │
│ E04-F02   │ CLI Infrastructure                   │ E04  │ completed │   100.0% │     6 │
│ E04-F03   │ Task Lifecycle Operations            │ E04  │ completed │   100.0% │     4 │
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
│ E04-F05   │ File Path Management                 │ E04  │ draft     │     0.0% │     6 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

### Example 5: Filter Features by Status

See only active features.

```bash
pm feature list --status=active
```

**Output:**

```
┌───────────┬──────────────────────────────────────┬──────┬───────────┬──────────┬───────┐
│ Key       │ Title                                │ Epic │ Status    │ Progress │ Tasks │
├───────────┼──────────────────────────────────────┼──────┼───────────┼──────────┼───────┤
│ E04-F04   │ Epic & Feature Queries               │ E04  │ active    │    33.3% │     6 │
└───────────┴──────────────────────────────────────┴──────┴───────────┴──────────┴───────┘
```

**Interpretation:**
- Only one feature is currently active
- This is the feature that needs attention

### Example 6: Get Feature Details

See detailed information about a specific feature.

```bash
pm feature get E04-F04
```

**Output:**

```
Feature: E04-F04 - Epic & Feature Queries
Epic: E04 - Task Management CLI Core
Status: active
Progress: 33.3%

Description:
Implement CLI commands for querying epics and features with automatic
progress calculation built on E04-F01 (Database) and E04-F02 (CLI Framework).

Task Status Breakdown:
  Completed: 2
  Todo: 4
  In Progress: 0
  Blocked: 0

Tasks:
┌────────────────┬──────────────────────────────────────┬───────────┬──────────┬──────────┐
│ Key            │ Title                                │ Status    │ Priority │ Agent    │
├────────────────┼──────────────────────────────────────┼───────────┼──────────┼──────────┤
│ T-E04-F04-001  │ Progress Calculation Service         │ completed │ high     │ backend  │
│ T-E04-F04-002  │ Epic Query Commands                  │ todo      │ high     │ backend  │
│ T-E04-F04-003  │ Feature Query Commands               │ todo      │ high     │ backend  │
│ T-E04-F04-004  │ Unit Tests - Progress Calculation    │ completed │ medium   │ qa       │
│ T-E04-F04-005  │ Integration Tests - CLI Commands     │ todo      │ medium   │ qa       │
│ T-E04-F04-006  │ Documentation                        │ todo      │ low      │ docs     │
└────────────────┴──────────────────────────────────────┴───────────┴──────────┴──────────┘

Total Tasks: 6
```

**Interpretation:**
- 2 tasks completed (T-001, T-004)
- 4 tasks remaining (T-002, T-003, T-005, T-006)
- Next logical task: T-002 (Epic Query Commands)

## Developer Workflows

### Workflow 1: Daily Standup Preparation

Prepare for daily standup by checking your progress.

```bash
# Step 1: Check what features are active
pm feature list --status=active

# Step 2: Get details on your current feature
pm feature get E04-F04

# Step 3: See which tasks you completed
pm task list --feature=E04-F04 --status=completed

# Step 4: See what's next
pm task list --feature=E04-F04 --status=todo
```

**Standup Summary:**
- "Yesterday: Completed progress calculation service and unit tests"
- "Today: Working on epic query commands (T-E04-F04-002)"
- "Blockers: None"
- "Progress: Feature E04-F04 is 33.3% complete"

### Workflow 2: Start of Sprint Planning

Understand project status before sprint planning.

```bash
# Step 1: See all epics
pm epic list

# Step 2: Drill into active epic
pm epic get E04

# Step 3: Identify incomplete features
pm feature list --epic=E04 --status=draft

# Step 4: Check feature details to estimate work
pm feature get E04-F05
```

**Planning Insights:**
- E04 is 65.3% complete
- 3 features remaining (F05, F06, F07)
- F05 has 6 tasks (estimate: 1 sprint)
- F06 has 4 tasks (estimate: 1 sprint)
- F07 has 7 tasks (estimate: 1-2 sprints)

### Workflow 3: End of Day Summary

Generate a summary of work completed today.

```bash
# Step 1: Check feature progress
pm feature get E04-F04

# Step 2: Compare with morning snapshot
# (assuming you saved morning progress)
# Morning: 16.7% (1/6 tasks)
# Evening: 33.3% (2/6 tasks)
# Completed: 1 task today

# Step 3: Generate report
echo "Daily Summary - $(date +%Y-%m-%d)"
echo "Feature: E04-F04 - Epic & Feature Queries"
echo "Progress: 16.7% → 33.3%"
echo "Tasks Completed: 1"
echo "Tasks Remaining: 4"
```

### Workflow 4: Identifying Blocked Work

Find features that might be blocked.

```bash
# Step 1: List active features
pm feature list --status=active

# Step 2: Get details on each active feature
pm feature get E04-F04

# Step 3: Check for blocked tasks
pm task list --feature=E04-F04 --status=blocked

# Step 4: If no blocked tasks, feature is healthy
# If blocked tasks exist, investigate dependencies
```

### Workflow 5: Prioritizing Next Feature

Decide which feature to work on next.

```bash
# Step 1: List draft features in current epic
pm feature list --epic=E04 --status=draft

# Step 2: Get details on each candidate
pm feature get E04-F05
pm feature get E04-F06
pm feature get E04-F07

# Step 3: Consider:
# - Task count (complexity)
# - Dependencies (which features depend on this)
# - Priority (from PRD)
# - Estimated time

# Decision: Start F05 (File Path Management)
# Reason: Needed by F06 and F07, moderate complexity
```

## AI Agent Workflows

### Workflow 1: Agent Selects Next Epic to Work On

AI agent determines which epic to focus on.

```bash
#!/bin/bash

# Step 1: Get all epics with progress
EPICS=$(pm epic list --json)

# Step 2: Find active epic with lowest progress
ACTIVE_EPIC=$(echo "$EPICS" | jq -r '.results[] | select(.status == "active") | select(.priority == "high") | .key' | head -1)

echo "Selected Epic: $ACTIVE_EPIC"

# Step 3: Get epic details
pm epic get "$ACTIVE_EPIC" --json
```

**Agent Logic:**
1. Query all epics
2. Filter to active + high priority
3. Select epic with lowest progress
4. Get full details for context

### Workflow 2: Agent Selects Feature Within Epic

AI agent chooses which feature to work on.

```bash
#!/bin/bash

EPIC="E04"

# Step 1: Get all features in epic
FEATURES=$(pm feature list --epic="$EPIC" --json)

# Step 2: Find incomplete active features
ACTIVE_FEATURE=$(echo "$FEATURES" | jq -r '.results[] | select(.status == "active") | select(.progress_pct < 100) | .key' | head -1)

# Step 3: If no active features, find draft features
if [ -z "$ACTIVE_FEATURE" ]; then
  ACTIVE_FEATURE=$(echo "$FEATURES" | jq -r '.results[] | select(.status == "draft") | .key' | head -1)
fi

echo "Selected Feature: $ACTIVE_FEATURE"

# Step 4: Get feature details with tasks
pm feature get "$ACTIVE_FEATURE" --json
```

**Agent Logic:**
1. Get all features in epic
2. Prefer incomplete active features
3. Fall back to draft features if none active
4. Get full feature details for task selection

### Workflow 3: Agent Reports Progress to User

AI agent generates a progress report.

```bash
#!/bin/bash

# Step 1: Get epic progress
EPIC_DATA=$(pm epic get E04 --json)
EPIC_PROGRESS=$(echo "$EPIC_DATA" | jq -r '.progress_pct')

# Step 2: Count features by status
TOTAL_FEATURES=$(echo "$EPIC_DATA" | jq '.features | length')
COMPLETED_FEATURES=$(echo "$EPIC_DATA" | jq '[.features[] | select(.progress_pct == 100)] | length')
ACTIVE_FEATURES=$(echo "$EPIC_DATA" | jq '[.features[] | select(.status == "active")] | length')

# Step 3: Generate report
cat <<EOF
Epic E04 Progress Report
========================
Overall Progress: $EPIC_PROGRESS%
Features: $COMPLETED_FEATURES/$TOTAL_FEATURES completed
Active Features: $ACTIVE_FEATURES

Recent Activity:
- Completed progress calculation service
- Completed unit tests for progress calculation
- Started epic query commands (in progress)

Next Steps:
- Complete epic query commands
- Implement feature query commands
- Write integration tests
EOF
```

### Workflow 4: Agent Validates Task Completion

AI agent checks if a task can be marked complete.

```bash
#!/bin/bash

FEATURE="E04-F04"

# Step 1: Get feature details
FEATURE_DATA=$(pm feature get "$FEATURE" --json)

# Step 2: Check task breakdown
COMPLETED=$(echo "$FEATURE_DATA" | jq '.task_breakdown.completed')
TODO=$(echo "$FEATURE_DATA" | jq '.task_breakdown.todo')
IN_PROGRESS=$(echo "$FEATURE_DATA" | jq '.task_breakdown.in_progress')

# Step 3: Determine if feature is complete
if [ "$TODO" -eq 0 ] && [ "$IN_PROGRESS" -eq 0 ]; then
  echo "Feature is complete! Ready to mark as completed."
else
  echo "Feature incomplete. $TODO tasks todo, $IN_PROGRESS in progress."
fi
```

### Workflow 5: Agent Discovers Project Context

AI agent learns about project structure before starting work.

```bash
#!/bin/bash

# Step 1: Discover all epics
echo "=== PROJECT EPICS ==="
pm epic list --json | jq -r '.results[] | "\(.key): \(.title) (\(.progress_pct)%)"'

# Step 2: Get active epic details
ACTIVE_EPIC=$(pm epic list --json | jq -r '.results[] | select(.status == "active") | .key' | head -1)
echo ""
echo "=== ACTIVE EPIC: $ACTIVE_EPIC ==="
pm epic get "$ACTIVE_EPIC" --json | jq -r '.features[] | "\(.key): \(.title) (\(.progress_pct)%)"'

# Step 3: Find current work
CURRENT_FEATURE=$(pm feature list --status=active --json | jq -r '.results[0].key')
echo ""
echo "=== CURRENT FEATURE: $CURRENT_FEATURE ==="
pm feature get "$CURRENT_FEATURE" --json | jq -r '.tasks[] | "\(.key): \(.title) [\(.status)]"'

# Step 4: Identify next task
NEXT_TASK=$(pm feature get "$CURRENT_FEATURE" --json | jq -r '.tasks[] | select(.status == "todo") | .key' | head -1)
echo ""
echo "=== NEXT TASK TO WORK ON ==="
echo "$NEXT_TASK"
```

## Reporting Scenarios

### Scenario 1: Weekly Status Report for Manager

Generate a weekly progress report.

```bash
#!/bin/bash

# Create report file
REPORT="weekly-report-$(date +%Y-%m-%d).md"

cat > "$REPORT" <<EOF
# Weekly Progress Report - $(date +%Y-%m-%d)

## Epic Progress

EOF

# Add epic progress
pm epic list --json | jq -r '.results[] | "- **\(.key)**: \(.title) - \(.progress_pct)% complete"' >> "$REPORT"

cat >> "$REPORT" <<EOF

## Active Features

EOF

# Add active features
pm feature list --status=active --json | jq -r '.results[] | "- **\(.key)**: \(.title) - \(.progress_pct)% complete (\(.task_count) tasks)"' >> "$REPORT"

cat >> "$REPORT" <<EOF

## Completed This Week

EOF

# Add recently completed features (would need task history for actual dates)
pm feature list --status=completed --json | jq -r '.results[] | "- \(.key): \(.title)"' | tail -3 >> "$REPORT"

echo "Report generated: $REPORT"
cat "$REPORT"
```

**Output Report:**

```markdown
# Weekly Progress Report - 2025-12-15

## Epic Progress

- **E04**: Task Management CLI Core - 65.3% complete
- **E05**: Task Management CLI Capabilities - 0.0% complete

## Active Features

- **E04-F04**: Epic & Feature Queries - 33.3% complete (6 tasks)

## Completed This Week

- E04-F01: Database Foundation
- E04-F02: CLI Infrastructure
- E04-F03: Task Lifecycle Operations
```

### Scenario 2: Executive Dashboard Summary

Generate a high-level executive summary.

```bash
#!/bin/bash

# Get metrics
TOTAL_EPICS=$(pm epic list --json | jq '.count')
ACTIVE_EPICS=$(pm epic list --json | jq '[.results[] | select(.status == "active")] | length')
COMPLETED_EPICS=$(pm epic list --json | jq '[.results[] | select(.status == "completed")] | length')

TOTAL_FEATURES=$(pm feature list --json | jq '.count')
COMPLETED_FEATURES=$(pm feature list --json | jq '[.results[] | select(.progress_pct == 100)] | length')

# Calculate overall progress
OVERALL_PROGRESS=$(pm epic list --json | jq '[.results[].progress_pct] | add / length')

# Generate summary
cat <<EOF
Executive Summary
=================

Project Health: $(if (( $(echo "$OVERALL_PROGRESS > 50" | bc -l) )); then echo "On Track"; else echo "Needs Attention"; fi)

Overall Progress: ${OVERALL_PROGRESS}%

Epics:
- Total: $TOTAL_EPICS
- Active: $ACTIVE_EPICS
- Completed: $COMPLETED_EPICS

Features:
- Total: $TOTAL_FEATURES
- Completed: $COMPLETED_FEATURES ($((COMPLETED_FEATURES * 100 / TOTAL_FEATURES))%)

Key Highlights:
$(pm epic list --json | jq -r '.results[] | select(.status == "active") | "- \(.title): \(.progress_pct)% complete"')

Risks:
$(pm feature list --status=active --json | jq -r '.results[] | select(.progress_pct == 0) | "- \(.title) - not started"')
EOF
```

### Scenario 3: Burndown Chart Data

Extract data for creating a burndown chart.

```bash
#!/bin/bash

# Get total tasks across all active epics
ACTIVE_EPICS=$(pm epic list --json | jq -r '.results[] | select(.status == "active") | .key')

echo "Epic,Total Tasks,Completed Tasks,Remaining Tasks"

for EPIC in $ACTIVE_EPICS; do
  EPIC_DATA=$(pm epic get "$EPIC" --json)
  TOTAL_TASKS=$(echo "$EPIC_DATA" | jq '[.features[].task_count] | add')

  # Calculate completed tasks (features at 100% * their task count)
  COMPLETED_TASKS=$(echo "$EPIC_DATA" | jq '[.features[] | select(.progress_pct == 100) | .task_count] | add // 0')

  REMAINING=$((TOTAL_TASKS - COMPLETED_TASKS))

  echo "$EPIC,$TOTAL_TASKS,$COMPLETED_TASKS,$REMAINING"
done
```

**Output:**

```
Epic,Total Tasks,Completed Tasks,Remaining Tasks
E04,39,16,23
```

## Troubleshooting Scenarios

### Scenario 1: Progress Seems Incorrect

Investigate why progress percentage seems wrong.

```bash
#!/bin/bash

FEATURE="E04-F04"

echo "Investigating feature: $FEATURE"
echo ""

# Get feature data
FEATURE_DATA=$(pm feature get "$FEATURE" --json)

# Extract task breakdown
echo "Task Breakdown:"
echo "$FEATURE_DATA" | jq '.task_breakdown'
echo ""

# List all tasks with status
echo "Task Status:"
pm task list --feature="$FEATURE" --json | jq -r '.results[] | "\(.key): \(.status)"'
echo ""

# Calculate expected progress
TOTAL_TASKS=$(pm task list --feature="$FEATURE" --json | jq '.count')
COMPLETED_TASKS=$(pm task list --feature="$FEATURE" --status=completed --json | jq '.count')
EXPECTED_PROGRESS=$(echo "scale=1; $COMPLETED_TASKS * 100 / $TOTAL_TASKS" | bc)

ACTUAL_PROGRESS=$(echo "$FEATURE_DATA" | jq '.progress_pct')

echo "Expected Progress: ${EXPECTED_PROGRESS}%"
echo "Actual Progress: ${ACTUAL_PROGRESS}%"

if [ "$EXPECTED_PROGRESS" != "$ACTUAL_PROGRESS" ]; then
  echo "WARNING: Progress mismatch!"
else
  echo "Progress is correct."
fi
```

### Scenario 2: Find Features with No Progress

Identify features that haven't been started.

```bash
#!/bin/bash

echo "Features with 0% Progress:"
echo ""

pm feature list --json | jq -r '.results[] | select(.progress_pct == 0) | "\(.key): \(.title) (\(.status))"'

echo ""
echo "Action Items:"
echo "1. Review draft features and prioritize"
echo "2. Check if active features are blocked"
echo "3. Consider starting next feature in backlog"
```

### Scenario 3: Database Health Check

Verify database integrity and progress calculations.

```bash
#!/bin/bash

echo "Database Health Check"
echo "===================="
echo ""

# Check epic count
EPIC_COUNT=$(pm epic list --json | jq '.count')
echo "Epics: $EPIC_COUNT"

# Check feature count
FEATURE_COUNT=$(pm feature list --json | jq '.count')
echo "Features: $FEATURE_COUNT"

# Check for orphaned features (features without epic)
# (Would need direct DB query)

# Check for features with invalid progress
INVALID_PROGRESS=$(pm feature list --json | jq '[.results[] | select(.progress_pct > 100 or .progress_pct < 0)] | length')
echo "Features with invalid progress: $INVALID_PROGRESS"

if [ "$INVALID_PROGRESS" -gt 0 ]; then
  echo "ERROR: Invalid progress values found!"
  pm feature list --json | jq '.results[] | select(.progress_pct > 100 or .progress_pct < 0)'
fi

echo ""
echo "Health check complete."
```

## Advanced JQ Patterns

### Pattern 1: Sort Epics by Progress (Lowest First)

```bash
pm epic list --json | jq '.results | sort_by(.progress_pct)'
```

### Pattern 2: Find High-Priority Epics Under 50% Complete

```bash
pm epic list --json | jq '.results[] | select(.priority == "high" and .progress_pct < 50)'
```

### Pattern 3: Get Feature Keys for Active Epic

```bash
pm epic get E04 --json | jq -r '.features[].key'
```

### Pattern 4: Count Tasks by Status Across All Features

```bash
pm feature list --json | jq '[.results[]] | group_by(.status) | map({status: .[0].status, count: length})'
```

### Pattern 5: Calculate Total Task Count Across Project

```bash
pm feature list --json | jq '[.results[].task_count] | add'
```

### Pattern 6: Find Features with Most Tasks

```bash
pm feature list --json | jq '.results | sort_by(.task_count) | reverse | .[0:3]'
```

### Pattern 7: Generate CSV of Feature Progress

```bash
echo "Key,Title,Progress,Tasks"
pm feature list --json | jq -r '.results[] | "\(.key),\(.title),\(.progress_pct),\(.task_count)"'
```

## Shell Script Integration

### Script 1: Automated Progress Tracker

Save progress snapshots over time.

```bash
#!/bin/bash
# save-progress.sh

SNAPSHOT_DIR="progress-snapshots"
mkdir -p "$SNAPSHOT_DIR"

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
SNAPSHOT_FILE="$SNAPSHOT_DIR/progress-$TIMESTAMP.json"

# Save epic progress
pm epic list --json > "$SNAPSHOT_FILE"

echo "Progress snapshot saved: $SNAPSHOT_FILE"

# Optional: Compare with previous snapshot
LATEST_SNAPSHOT=$(ls -t "$SNAPSHOT_DIR"/*.json | head -1)
PREVIOUS_SNAPSHOT=$(ls -t "$SNAPSHOT_DIR"/*.json | head -2 | tail -1)

if [ "$LATEST_SNAPSHOT" != "$PREVIOUS_SNAPSHOT" ]; then
  echo ""
  echo "Progress Changes:"
  diff <(jq -r '.results[] | "\(.key): \(.progress_pct)%"' "$PREVIOUS_SNAPSHOT") \
       <(jq -r '.results[] | "\(.key): \(.progress_pct)%"' "$LATEST_SNAPSHOT")
fi
```

### Script 2: Feature Completion Notifier

Send notification when feature reaches 100%.

```bash
#!/bin/bash
# check-feature-completion.sh

FEATURES=$(pm feature list --status=active --json)

# Check each active feature
echo "$FEATURES" | jq -r '.results[] | select(.progress_pct == 100) | .key' | while read FEATURE; do
  echo "🎉 Feature $FEATURE is 100% complete!"

  # Send notification (example with desktop notification)
  notify-send "Feature Complete" "Feature $FEATURE is ready for review"

  # Or send email, Slack message, etc.
done
```

### Script 3: Sprint Planning Assistant

Help plan next sprint based on remaining work.

```bash
#!/bin/bash
# sprint-planner.sh

EPIC="E04"
VELOCITY=10  # tasks per sprint

echo "Sprint Planning for Epic $EPIC"
echo "Velocity: $VELOCITY tasks/sprint"
echo ""

# Get incomplete features
INCOMPLETE_FEATURES=$(pm feature list --epic="$EPIC" --json | jq -r '.results[] | select(.progress_pct < 100)')

echo "Incomplete Features:"
echo "$INCOMPLETE_FEATURES" | jq -r '"\(.key): \(.title) - \(.task_count) tasks remaining"'

TOTAL_REMAINING=$(echo "$INCOMPLETE_FEATURES" | jq '[.task_count] | add')
SPRINTS_NEEDED=$(echo "scale=0; ($TOTAL_REMAINING + $VELOCITY - 1) / $VELOCITY" | bc)

echo ""
echo "Total Remaining Tasks: $TOTAL_REMAINING"
echo "Estimated Sprints Needed: $SPRINTS_NEEDED"
```

---

**Examples Version:** 1.0.0
**Last Updated:** 2025-12-15
