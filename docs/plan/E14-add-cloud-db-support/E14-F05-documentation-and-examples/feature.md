# Feature: Documentation and Examples

**Feature Key:** E14-F05
**Epic:** E14 - Cloud Database Support
**Status:** Draft
**Execution Order:** 5

## Overview

Create comprehensive documentation and practical examples for cloud database support, enabling users to quickly understand, adopt, and troubleshoot Turso cloud sync features.

## Goal

### Problem

Without proper documentation, users will struggle to:
- Understand what cloud sync is and why they need it
- Set up Turso for the first time (account creation, database setup, token management)
- Migrate from local to cloud without data loss
- Troubleshoot common issues (connection failures, sync conflicts, auth errors)
- Understand best practices (when to use local vs cloud, backup strategies, cost management)

Poor documentation leads to:
- Low adoption (users don't try features they don't understand)
- High support burden (same questions answered repeatedly)
- User frustration (trial-and-error instead of guided success)
- Negative perception (powerful feature seems complicated)

### Solution

Create multi-layered documentation:
1. **Quick Start Guide:** 5-minute setup for impatient users
2. **Migration Guide:** Step-by-step local → cloud transition
3. **Troubleshooting Guide:** Common issues and solutions
4. **Example Workflows:** Real-world multi-workstation scenarios
5. **CLI Reference Update:** All cloud commands documented
6. **CLAUDE.md Update:** AI agent instructions for cloud features

### Impact

**For Users:**
- 5-minute quick start (vs 30-minute manual exploration)
- Self-service troubleshooting (fewer support requests)
- Confidence to adopt cloud features (clear, tested examples)
- Understanding of costs and limits (no surprises)

**For Support:**
- Link to docs instead of explaining repeatedly
- Standardized answers (docs are single source of truth)
- Fewer support tickets (users solve own problems)

## User Stories

### Must-Have Stories

**Story 1:** As a new user, I want a quick start guide so that I can enable cloud sync in 5 minutes without reading extensive documentation.

**Acceptance Criteria:**
- [ ] Single page, < 500 words
- [ ] Step-by-step numbered instructions
- [ ] Copy-paste commands that work as-is
- [ ] Expected output shown for each step
- [ ] Verification step at end
- [ ] Links to detailed docs for more info

**Story 2:** As an existing user, I want a migration guide so that I can safely move my local database to cloud without data loss.

**Acceptance Criteria:**
- [ ] Addresses common concerns (will I lose data? can I rollback?)
- [ ] Pre-migration checklist
- [ ] Step-by-step migration process
- [ ] Post-migration verification
- [ ] Rollback instructions if needed
- [ ] Estimated time for each step

**Story 3:** As a user experiencing issues, I want a troubleshooting guide so that I can fix common problems without contacting support.

**Acceptance Criteria:**
- [ ] Organized by symptom ("I see error X" or "Cloud sync isn't working")
- [ ] Each issue has: Symptoms, Cause, Solution
- [ ] Diagnostic commands to gather info
- [ ] When to contact support (escalation path)
- [ ] Searchable (good keywords)

**Story 4:** As a multi-machine user, I want example workflows so that I can understand how cloud sync works in practice.

**Acceptance Criteria:**
- [ ] Scenario 1: Work desktop + home laptop
- [ ] Scenario 2: Offline work (airplane) with auto-sync
- [ ] Scenario 3: Emergency rollback to local
- [ ] Each scenario shows actual commands and outputs
- [ ] Explains what's happening at each step

**Story 5:** As a developer, I want updated CLI reference so that I know all available cloud commands and their options.

**Acceptance Criteria:**
- [ ] All `shark cloud` commands documented
- [ ] Command syntax, flags, examples
- [ ] Exit codes and error messages
- [ ] JSON output format documented
- [ ] Integrated into existing docs/CLI_REFERENCE.md

**Story 6:** As an AI agent (Claude), I want updated CLAUDE.md so that I can correctly advise users on cloud features.

**Acceptance Criteria:**
- [ ] Cloud database section added to CLAUDE.md
- [ ] Explains when to recommend local vs cloud
- [ ] Common setup commands
- [ ] Troubleshooting steps
- [ ] Security best practices (token management)

## Requirements

### Functional Requirements

**REQ-F-001: Quick Start Guide**
- **Description:** Single-page guide for 5-minute cloud setup
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] File: `docs/CLOUD_QUICK_START.md`
  - [ ] Length: < 500 words
  - [ ] Format: Numbered steps with commands
  - [ ] Includes verification step
  - [ ] Tested on fresh account (works as written)

**REQ-F-002: Migration Guide**
- **Description:** Comprehensive guide for local → cloud transition
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] File: `docs/CLOUD_MIGRATION.md`
  - [ ] Sections: Pre-flight check, Migration, Verification, Rollback
  - [ ] Addresses data loss concerns
  - [ ] Backup strategy explained
  - [ ] Multi-workstation setup instructions

**REQ-F-003: Troubleshooting Guide**
- **Description:** Issue-solution mapping for common problems
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] File: `docs/CLOUD_TROUBLESHOOTING.md`
  - [ ] Minimum 10 common issues covered
  - [ ] Format: Problem → Diagnosis → Solution
  - [ ] Includes diagnostic commands
  - [ ] Contact support escalation path

**REQ-F-004: Example Workflows**
- **Description:** Real-world scenarios with step-by-step walkthroughs
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] File: `docs/CLOUD_WORKFLOWS.md`
  - [ ] Minimum 3 scenarios
  - [ ] Actual commands and output shown
  - [ ] Explains rationale for each step
  - [ ] Screenshots optional but helpful

**REQ-F-005: CLI Reference Update**
- **Description:** Document all cloud commands in main CLI reference
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Update `docs/CLI_REFERENCE.md`
  - [ ] New section: "Cloud Database Commands"
  - [ ] All `shark cloud` commands documented
  - [ ] Consistent format with existing docs
  - [ ] Links to detailed guides

**REQ-F-006: CLAUDE.md Update**
- **Description:** Add cloud database instructions for AI agents
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Update `CLAUDE.md`
  - [ ] New section: "Cloud Database Support"
  - [ ] When to recommend local vs cloud
  - [ ] Common setup commands
  - [ ] Security best practices
  - [ ] Troubleshooting reference

### Non-Functional Requirements

**Readability:**
- **REQ-NF-001:** Docs use consistent formatting (headings, code blocks, lists)
- **Measurement:** Markdown linter + style guide compliance
- **Target:** 100% compliant with project style guide
- **Justification:** Professional, maintainable documentation

**Accuracy:**
- **REQ-NF-002:** All commands tested and verified before publishing
- **Measurement:** Manual testing on fresh account
- **Target:** 100% of commands work as documented
- **Justification:** Broken docs erode trust

**Discoverability:**
- **REQ-NF-003:** Docs linked from README and help output
- **Measurement:** Check `README.md` and `shark --help`
- **Target:** Cloud docs visible in both places
- **Justification:** Users can't use docs they can't find

## Technical Design

### Documentation Structure

```
docs/
├── README.md                     (update: link to cloud docs)
├── CLI_REFERENCE.md              (update: add cloud commands)
├── CLOUD_QUICK_START.md          (new)
├── CLOUD_MIGRATION.md            (new)
├── CLOUD_TROUBLESHOOTING.md      (new)
├── CLOUD_WORKFLOWS.md            (new)
└── CLAUDE.md                     (update: cloud section)
```

### Quick Start Guide Outline

```markdown
# Cloud Sync Quick Start

Set up Turso cloud sync in 5 minutes.

## Prerequisites
- Shark CLI installed
- Existing local database with tasks

## Steps

### 1. Create Turso Account
\`\`\`bash
# Visit: https://turso.tech/signup
# Or use Turso CLI: turso auth signup
\`\`\`

### 2. Initialize Cloud Database
\`\`\`bash
shark cloud init
\`\`\`

Expected output:
\`\`\`
✓ Connected to Turso
✓ Database created: shark-tasks-yourname
✓ Exported 42 tasks to cloud
✓ Configuration updated

Next: Add to your shell profile:
  export SHARK_DB_URL="libsql://shark-tasks-yourname.turso.io"
\`\`\`

### 3. Verify Cloud Sync
\`\`\`bash
shark task list
shark cloud status
\`\`\`

### 4. Set Up Second Workstation
\`\`\`bash
shark cloud login
# Enter URL and token from step 2
\`\`\`

## Next Steps
- [Migration Guide](CLOUD_MIGRATION.md) - Detailed migration
- [Troubleshooting](CLOUD_TROUBLESHOOTING.md) - Fix issues
- [Workflows](CLOUD_WORKFLOWS.md) - Real-world examples
```

### Troubleshooting Guide Outline

```markdown
# Cloud Sync Troubleshooting

## Connection Issues

### "ERROR: Cannot connect to Turso"

**Symptoms:**
- `shark cloud init` fails with connection error
- `shark cloud status` shows "Offline"

**Diagnosis:**
\`\`\`bash
# Check internet connection
ping turso.tech

# Check auth token
echo $TURSO_AUTH_TOKEN

# Verify database URL
shark cloud status --json | jq .database_url
\`\`\`

**Solution:**
1. Verify internet connection
2. Check auth token is valid (not expired)
3. Test connection: `curl -I https://turso.tech`
4. If still failing, regenerate token at https://turso.tech/tokens

...
```

## Tasks

- **T-E14-F05-005:** Update CLI reference with cloud commands (Priority: 8)
- **T-E14-F05-001:** Write quick start guide for Turso setup (Priority: 7)
- **T-E14-F05-002:** Write migration guide for local to cloud transition (Priority: 7)
- **T-E14-F05-003:** Write troubleshooting guide for common issues (Priority: 6)
- **T-E14-F05-004:** Create example workflows for multi-workstation setup (Priority: 6)
- **T-E14-F05-006:** Update CLAUDE.md with cloud database instructions (Priority: 5)

## Dependencies

- **F01-F04:** Must be implemented first (can't document non-existent features)
- **Testing:** Commands must be tested before documenting

## Success Metrics

**Adoption:**
- [ ] 80% of users who enable cloud reference docs (telemetry: link clicks)
- [ ] 50% reduction in cloud-related support tickets after docs published

**Quality:**
- [ ] 90% of quick start guide users succeed on first try (user testing)
- [ ] Zero inaccurate commands (all tested before publishing)
- [ ] Positive feedback score > 4.5/5 (user survey)

**Discoverability:**
- [ ] Cloud docs linked from README
- [ ] Cloud docs visible in `shark --help`
- [ ] Search engines index cloud docs (SEO)

## Out of Scope

### Explicitly Excluded

1. **Video Tutorials**
   - **Why:** Text docs sufficient for v1, videos require maintenance
   - **Future:** Could add if users request it
   - **Workaround:** Written examples with outputs

2. **Interactive Tutorials**
   - **Why:** Complex to build and maintain
   - **Future:** Could add CLI wizard mode
   - **Workaround:** Step-by-step written guides

3. **Translations (Non-English)**
   - **Why:** English-first for v1, limited audience for translations
   - **Future:** Community translations if demand exists
   - **Workaround:** English docs + machine translation

4. **API Documentation (Programmatic Access)**
   - **Why:** CLI focus for v1, no public API yet
   - **Future:** If users build tools on top of Shark
   - **Workaround:** Users can inspect CLI source code

## Documentation Standards

**Style Guide:**
- Use present tense ("Run `shark task list`" not "You should run...")
- Use second person ("You can..." not "The user can...")
- Code blocks use sh/bash syntax highlighting
- Commands show expected output
- Use emojis sparingly (✓, ✗, ⚠️ only)

**Command Examples:**
- Show full command (not abbreviated)
- Include expected output
- Explain non-obvious flags
- Show both success and error cases

**Links:**
- Use relative links for internal docs
- Use absolute HTTPS links for external sites
- Verify all links before publishing

---

*Last Updated:* 2026-01-04
*Dependencies:* F01, F02, F03, F04 (must be implemented first)
