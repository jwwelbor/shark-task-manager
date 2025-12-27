 High-Impact Additions

  1. Task Notes/Activity Log ⭐ Most Important

  -- New table
  CREATE TABLE task_notes (
      id INTEGER PRIMARY KEY,
      task_id INTEGER NOT NULL,
      note_type TEXT CHECK (note_type IN ('comment', 'decision', 'blocker', 'solution', 'reference')),
      content TEXT NOT NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (task_id) REFERENCES tasks(id)
  );

  Use cases:
  - "Decided to use composable pattern instead of provide/inject"
  - "Blocked by missing dark mode CSS variables - resolved by completing T-E13-F05-001"
  - "UAT found flash still occurring in Safari - fixed by adding -webkit- prefix"
  - "Related PR: #123"

  CLI:
  shark task note T-E13-F05-002 "Flash prevention working in Chrome/Firefox, Safari needs testing"
  shark task notes T-E13-F05-002  # View all notes

  2. Completion Metadata

  ALTER TABLE tasks ADD COLUMN completion_notes TEXT;
  ALTER TABLE tasks ADD COLUMN files_modified TEXT;  -- JSON array
  ALTER TABLE tasks ADD COLUMN verification_status TEXT CHECK (verification_status IN ('not_verified', 'verified', 'failed'));
  ALTER TABLE tasks ADD COLUMN agent_id TEXT;  -- Which agent completed it

  Use cases:
  - Record what files were changed
  - Link to agent execution
  - Track verification status
  - Store completion summary

  3. Task Context/Resume Info

  ALTER TABLE tasks ADD COLUMN context_data TEXT;  -- JSON blob

  Stored JSON:
  {
    "implementation_notes": "Used singleton pattern for theme state",
    "tests_added": ["useTheme.spec.ts"],
    "documentation": ["dev-artifacts/2025-12-26-useTheme-composable/"],
    "known_issues": [],
    "next_steps": "Integrate with ThemeToggle component",
    "acceptance_criteria": [
      {"criterion": "localStorage persistence", "status": "complete"},
      {"criterion": "16/16 tests passing", "status": "complete"}
    ]
  }

  CLI:
  shark task context set T-E13-F05-003 --field implementation_notes "Used singleton pattern"
  shark task context get T-E13-F05-003

  4. Better Task Relationships

  CREATE TABLE task_relationships (
      id INTEGER PRIMARY KEY,
      from_task_id INTEGER NOT NULL,
      to_task_id INTEGER NOT NULL,
      relationship_type TEXT CHECK (relationship_type IN ('depends_on', 'blocks', 'related_to', 'follows', 'spawned_from')),
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (from_task_id) REFERENCES tasks(id),
      FOREIGN KEY (to_task_id) REFERENCES tasks(id)
  );

  Use cases:
  - Track task spawned from UAT findings
  - Link related implementation tasks
  - Show dependency graph

  5. Search & Filter Enhancements

  # Search notes content
  shark task search "flash prevention"

  # Filter by multiple criteria
  shark task list --epic E13 --status completed --agent-type frontend

  # Find by file
  shark task find --file "useTheme.ts"

  # Show task history
  shark task history T-E13-F05-003

  Medium Priority

  6. Acceptance Criteria Tracking

  CREATE TABLE task_criteria (
      id INTEGER PRIMARY KEY,
      task_id INTEGER NOT NULL,
      criterion TEXT NOT NULL,
      status TEXT CHECK (status IN ('pending', 'complete', 'failed')),
      verified_at TIMESTAMP,
      FOREIGN KEY (task_id) REFERENCES tasks(id)
  );

  7. Task Tags/Labels

  CREATE TABLE task_tags (
      id INTEGER PRIMARY KEY,
      task_id INTEGER NOT NULL,
      tag TEXT NOT NULL,
      FOREIGN KEY (task_id) REFERENCES tasks(id),
      UNIQUE(task_id, tag)
  );

  Use cases:
  - Tag: needs-uat, breaking-change, tech-debt, quick-win
  - Filter: shark task list --tag needs-uat

  8. Work Session Tracking

  CREATE TABLE work_sessions (
      id INTEGER PRIMARY KEY,
      task_id INTEGER NOT NULL,
      agent_id TEXT,
      started_at TIMESTAMP NOT NULL,
      ended_at TIMESTAMP,
      outcome TEXT,  -- 'completed', 'paused', 'blocked'
      FOREIGN KEY (task_id) REFERENCES tasks(id)
  );

  Lower Priority

  9. Estimates vs Actuals

  ALTER TABLE tasks ADD COLUMN estimated_points INTEGER;
  ALTER TABLE tasks ADD COLUMN actual_duration_seconds INTEGER;

  10. Review/Approval Workflow

  ALTER TABLE tasks ADD COLUMN reviewer TEXT;
  ALTER TABLE tasks ADD COLUMN review_status TEXT CHECK (review_status IN ('pending', 'approved', 'changes_requested'));
  ALTER TABLE tasks ADD COLUMN reviewed_at TIMESTAMP;

  ---
  Recommendation: Start with Top 3

  Phase 1 (Immediate Value):
  1. Task notes/activity log - critical for context
  2. Completion metadata - track what was actually done
  3. Better search/filter - find relevant tasks faster

  Phase 2 (Enhanced Workflow):
  4. Task relationships - understand dependencies
  5. Acceptance criteria tracking - verify completeness
  6. Tags/labels - categorize and filter




---



 1. Task Notes/Activity Log ⭐ Critical

  What Exists: task_history.notes field (barely used)

  What's Missing: Rich, searchable notes separate from status changes

  Real E13 Examples:

  During T-E13-F05-002 (Flash Prevention):

  # What I'd want to record:
  shark task note add T-E13-F05-002 \
    --type decision \
    "Used IIFE pattern instead of module to avoid async loading delay"

  shark task note add T-E13-F05-002 \
    --type reference \
    "Similar pattern used in shadcn-vue theme switching: https://github.com/..."

  shark task note add T-E13-F05-002 \
    --type solution \
    "Safari flash fix: Moved script BEFORE viewport meta tag"

  During T-E13-F02-004 (useContentColor):

  shark task note add T-E13-F02-004 \
    --type implementation \
    "Created 3 composables: useContentColor (main), useContentBadgeColor, useContentCardColor"

  shark task note add T-E13-F02-004 \
    --type testing \
    "All 21 tests passing - covered gradient edge cases and fallback behavior"

  shark task note add T-E13-F02-004 \
    --type future \
    "TODO: Consider caching computed color values if performance becomes issue"

  Schema:
  CREATE TABLE task_notes (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      task_id INTEGER NOT NULL,
      note_type TEXT CHECK (note_type IN (
          'comment',      -- General observation
          'decision',     -- Why we chose X over Y
          'blocker',      -- What's blocking progress
          'solution',     -- How we solved a problem
          'reference',    -- External links, docs
          'implementation', -- What we actually built
          'testing',      -- Test results, coverage
          'future',       -- Future improvements
          'question'      -- Unanswered questions
      )),
      content TEXT NOT NULL,
      created_by TEXT,  -- 'claude' or 'jwwelbor'
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
  );

  CLI:
  # Add note
  shark task note add T-E13-F05-002 --type decision "Used singleton pattern for theme state"

  # List all notes
  shark task notes T-E13-F05-002

  # Filter by type
  shark task notes T-E13-F05-002 --type decision

  # Search across all notes
  shark notes search "singleton pattern"

  # Timeline view (status changes + notes)
  shark task timeline T-E13-F05-002

  Output Example:
  Task T-E13-F05-002: Implement Flash Prevention Script

  Timeline:
    2025-12-26 06:22  Created                          (jwwelbor)
    2025-12-26 14:15  [DECISION] Used IIFE pattern     (claude)
    2025-12-26 14:20  [IMPLEMENTATION] Added to index.html:9-35  (claude)
    2025-12-26 14:25  [TESTING] Verified in Chrome, Firefox      (claude)
    2025-12-26 14:30  [SOLUTION] Safari flash fix - moved script (claude)
    2025-12-26 17:45  Status: todo → ready_for_review  (jwwelbor)
    2025-12-26 17:45  Status: ready_for_review → completed (jwwelbor)

  ---
  2. Completion Metadata

  What Exists: Timestamps only (started_at, completed_at)

  What's Missing: What actually got done, where, and verification status

  Real E13 Example - T-E13-F05-003 (useTheme):

  {
    "files_created": [
      "frontend/src/composables/useTheme.ts",
      "frontend/src/composables/__tests__/useTheme.spec.ts"
    ],
    "files_modified": [
      "frontend/index.html"
    ],
    "lines_added": 178,
    "tests_added": 16,
    "test_status": "16/16 passing",
    "agent_execution_id": "a5ad46d",
    "verification_method": "automated_tests",
    "verification_status": "verified",
    "completion_summary": "Implemented singleton theme composable with localStorage persistence. Supports light/dark/system themes with media query listeners.",
    "documentation_artifacts": [
      "dev-artifacts/2025-12-26-useTheme-composable/verification-report.md",
      "dev-artifacts/2025-12-26-useTheme-composable/task-completion-summary.md"
    ]
  }

  Schema:
  ALTER TABLE tasks ADD COLUMN completion_metadata TEXT;  -- JSON blob
  ALTER TABLE tasks ADD COLUMN verification_status TEXT
    CHECK (verification_status IN ('not_verified', 'verified', 'failed', 'manual_required'));
  ALTER TABLE tasks ADD COLUMN verification_notes TEXT;

  CLI:
  # Record completion
  shark task complete T-E13-F05-003 \
    --files-created "frontend/src/composables/useTheme.ts" \
    --tests "16/16 passing" \
    --agent-id "a5ad46d" \
    --summary "Implemented singleton theme composable"

  # Show what was done
  shark task show T-E13-F05-003 --completion-details

  # Find all tasks that modified a specific file
  shark task find --modified-file "frontend/index.html"

  ---
  3. Enhanced Task Context/Resume Data

  What Exists: Basic description field

  What's Missing: Structured context for resuming work

  Real E13 Example - If T-E13-F05-004 (ThemeToggle) was paused:

  {
    "progress": {
      "completed_steps": [
        "Created base ThemeToggle.vue component",
        "Added icon switching logic"
      ],
      "current_step": "Implementing dropdown menu for 3 theme options",
      "remaining_steps": [
        "Add keyboard shortcuts (Ctrl+Shift+T)",
        "Add tests",
        "Update documentation"
      ]
    },
    "implementation_decisions": {
      "component_location": "frontend/src/components/ThemeToggle.vue",
      "uses_composable": "useTheme from @/composables/useTheme",
      "icon_library": "lucide-vue-next (Sun, Moon, Monitor icons)",
      "pattern": "Dropdown menu with 3 options, not just toggle"
    },
    "open_questions": [
      "Should theme toggle be in header or settings page?",
      "Do we need keyboard shortcut or just click?"
    ],
    "blockers": [],
    "acceptance_criteria_status": [
      {"criterion": "Toggle switches between light/dark", "status": "complete"},
      {"criterion": "Dropdown shows 3 options (light/dark/system)", "status": "in_progress"},
      {"criterion": "Icon reflects current theme", "status": "complete"},
      {"criterion": "Keyboard accessible", "status": "pending"},
      {"criterion": "Tests pass", "status": "pending"}
    ],
    "related_tasks": [
      "T-E13-F05-003",  // useTheme composable (completed)
      "T-E13-F05-001"   // Dark mode CSS (completed)
    ]
  }

  CLI:
  # Set context when pausing work
  shark task context T-E13-F05-004 \
    --current-step "Implementing dropdown menu" \
    --decision "component_location=frontend/src/components/ThemeToggle.vue" \
    --question "Should theme toggle be in header or settings?"

  # Resume work - get full context
  shark task resume T-E13-F05-004

  # Output shows:
  # - What's been done
  # - What's next
  # - Open questions
  # - Related completed tasks

  ---
  4. Better Task Relationships

  What Exists: depends_on TEXT field (comma-separated?)

  What's Missing: Bidirectional links, relationship types, dependency graph

  Real E13 Examples:

  Current State (limited):

  -- T-E13-F05-004 depends on T-E13-F05-003
  UPDATE tasks SET depends_on = 'T-E13-F05-003' WHERE key = 'T-E13-F05-004';

  What We Need:

  -- Multiple relationship types
  INSERT INTO task_relationships VALUES
    -- ThemeToggle depends on useTheme
    (NULL, 'T-E13-F05-004', 'T-E13-F05-003', 'depends_on', CURRENT_TIMESTAMP),

    -- ThemeToggle depends on dark mode CSS
    (NULL, 'T-E13-F05-004', 'T-E13-F05-001', 'depends_on', CURRENT_TIMESTAMP),

    -- Flash prevention related to useTheme (both use same localStorage key)
    (NULL, 'T-E13-F05-002', 'T-E13-F05-003', 'related_to', CURRENT_TIMESTAMP),

    -- Form components follow from Input focus states
    (NULL, 'T-E13-F03-005', 'T-E13-F01-004', 'follows', CURRENT_TIMESTAMP),

    -- Integration testing blocks deployment
    (NULL, 'T-E13-F05-007', 'DEPLOY-E13', 'blocks', CURRENT_TIMESTAMP);

  Relationship Types:
  - depends_on: Must complete X before starting Y
  - blocks: Y cannot proceed until X is done
  - related_to: Share common code/concerns
  - follows: Y naturally comes after X (sequence)
  - spawned_from: Y was created from UAT/bug in X
  - duplicates: Same work, merge
  - references: Consults/uses output of X

  CLI:
  # Add relationship
  shark task link T-E13-F05-004 --depends-on T-E13-F05-003

  # Show all dependencies
  shark task deps T-E13-F05-004

  # Dependency graph (visual)
  shark task graph T-E13-F05-004

  # Find what blocks a task
  shark task blocked-by T-E13-F05-004

  # Find what this task blocks
  shark task blocks T-E13-F05-003

  Output Example:
  $ shark task deps T-E13-F05-004

  T-E13-F05-004: Create ThemeToggle Component

  Dependencies (must complete first):
    ✓ T-E13-F05-003: Create useTheme() Composable (completed 2025-12-26)
    ✓ T-E13-F05-001: Define Dark Mode CSS Variables (completed 2025-12-26)

  Related Tasks:
    ✓ T-E13-F05-002: Flash Prevention Script (shares localStorage key)

  Blocks (waiting on this):
    ○ T-E13-F05-007: Dark Mode Integration Testing
    ○ DEPLOY-E13: Deploy E13 to staging

  Spawned Tasks:
    (none)

  ---
  5. Acceptance Criteria Tracking

  What Exists: Criteria in markdown files only

  What's Missing: Checkable, trackable criteria in database

  Real E13 Example - T-E13-F05-002:

  INSERT INTO task_criteria VALUES
    (NULL, 415, 'Script added to index.html in <head> section', 'complete', '2025-12-26 14:20', NULL),
    (NULL, 415, 'Script placed BEFORE CSS links', 'complete', '2025-12-26 14:20', NULL),
    (NULL, 415, 'Script reads wgm-theme-preference from localStorage', 'complete', '2025-12-26 14:20', NULL),
    (NULL, 415, 'Script detects system dark mode preference', 'complete', '2025-12-26 14:20', NULL),
    (NULL, 415, 'Script applies .dark class if dark mode active', 'complete', '2025-12-26 14:20', NULL),
    (NULL, 415, 'No flash on page load in dark mode', 'complete', '2025-12-26 14:25', 'Tested in Chrome'),
    (NULL, 415, 'No JavaScript errors in console', 'complete', '2025-12-26 14:25', NULL);

  Schema:
  CREATE TABLE task_criteria (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      task_id INTEGER NOT NULL,
      criterion TEXT NOT NULL,
      status TEXT CHECK (status IN ('pending', 'in_progress', 'complete', 'failed', 'na')) DEFAULT 'pending',
      verified_at TIMESTAMP,
      verification_notes TEXT,
      FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
  );

  CLI:
  # Auto-import from task markdown
  shark task criteria import T-E13-F05-002

  # Check off criteria
  shark task criteria check T-E13-F05-002 1 --note "Verified in index.html:9"

  # Show progress
  shark task criteria T-E13-F05-002

  # Output:
  # T-E13-F05-002: Implement Flash Prevention Script (7/7 criteria met)
  #   ✓ Script added to index.html in <head> section
  #   ✓ Script placed BEFORE CSS links
  #   ✓ Script reads wgm-theme-preference from localStorage
  #   ...

  # Fail a criterion
  shark task criteria fail T-E13-F05-002 6 --note "Flash still visible in Safari"

  # Find incomplete tasks
  shark task list --incomplete-criteria

  ---
  6. Enhanced Search

  What Exists: Basic key/title filtering

  What's Missing: Full-text search across notes, descriptions, files

  Real Use Cases:

  # Find all tasks that touched a file
  shark search --file "useTheme.ts"
  # → T-E13-F05-003: Create useTheme() Composable
  # → T-E13-F05-004: Create ThemeToggle Component (uses useTheme)

  # Find tasks by implementation detail
  shark search "singleton pattern"
  # → T-E13-F05-003: [DECISION] Used singleton pattern for theme state

  # Find tasks with specific issues
  shark search "Safari" --type notes
  # → T-E13-F05-002: [SOLUTION] Safari flash fix - moved script

  # Find by agent
  shark search --agent-id "a5ad46d"
  # → T-E13-F05-003: Create useTheme() Composable

  # Combinedidate search
  shark search "dark mode" --status completed --epic E13

  ---
  7. Work Session Tracking

  What Exists: started_at, completed_at

  What's Missing: Multiple work sessions, pause/resume, time tracking

  Real E13 Example:

  -- T-E13-F02-004 had 3 work sessions:
  INSERT INTO work_sessions VALUES
    -- Session 1: Initial implementation
    (NULL, 404, 'abcf742', '2025-12-26 10:00:00', '2025-12-26 11:30:00',
     'paused', 'Created base composable, 10/21 tests passing'),

    -- Session 2: Fixed edge cases
    (NULL, 404, 'abcf742', '2025-12-26 13:00:00', '2025-12-26 14:15:00',
     'paused', 'Added badge/card helpers, 18/21 tests passing'),

    -- Session 3: Final polish
    (NULL, 404, 'abcf742', '2025-12-26 15:00:00', '2025-12-26 15:45:00',
     'completed', '21/21 tests passing, documentation complete');

  Use Cases:
  - Track actual time spent (for estimation improvement)
  - Resume where you left off
  - See if task taking longer than expected
  - Identify tasks that get repeatedly paused (blockers)

  CLI:
  # Start session (auto-called by `shark task start`)
  shark task session start T-E13-F05-004 --agent a5ad46d

  # Pause with note
  shark task session pause T-E13-F05-004 --note "Waiting for design decision"

  # Resume
  shark task session resume T-E13-F05-004

  # Show sessions
  shark task sessions T-E13-F05-004

  # Analytics
  shark analytics --task-duration-avg --epic E13

  ---
  Priority Recommendation

  Based on our E13 workflow, implement in this order:

  Phase 1 (Must Have):

  1. Task Notes - Critical for context across sessions
  2. Completion Metadata - Know what was actually done
  3. Acceptance Criteria - Track progress, know when done

  Phase 2 (High Value):

  4. Enhanced Relationships - Understand dependencies
  5. Full-Text Search - Find relevant work quickly

  Phase 3 (Nice to Have):

  6. Work Sessions - Analytics and time tracking
  7. Task Tags - Additional categorization
