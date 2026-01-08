# Feature: Cloud CLI Commands

**Feature Key:** E13-F03
**Epic:** E13 - Cloud Database Support
**Status:** Draft
**Execution Order:** 3

## Overview

Provide user-facing CLI commands for managing cloud database connections, enabling developers to set up, authenticate, monitor, and control their Turso cloud database from the Shark CLI.

## Goal

### Problem

After integrating Turso at the code level (F01, F02), users need:
- **Easy setup:** Initialize cloud database without manually editing config files or understanding libSQL URLs
- **Authentication:** Securely provide auth tokens without exposing them in command history
- **Status visibility:** Check connection health, sync status, and usage metrics
- **Control:** Manual sync triggers, logout, troubleshooting

Without user-friendly commands, developers would need to:
- Manually create Turso databases via web UI
- Copy/paste connection strings into config files
- Set environment variables in shell profiles
- Debug connection issues without feedback
- Check Turso dashboard for usage (context switch)

### Solution

Implement `shark cloud` command group with subcommands:
- **`shark cloud init`:** Interactive setup wizard (creates DB, configures shark)
- **`shark cloud login`:** Connect to existing cloud DB (for second workstation)
- **`shark cloud sync`:** Manual sync trigger (push/pull/bidirectional)
- **`shark cloud status`:** Show connection state, sync status, usage stats
- **`shark cloud logout`:** Clear credentials and revert to local mode

### Impact

**For Users:**
- 5-minute setup (vs 30+ minutes manual config)
- Clear error messages with actionable fixes
- Visibility into cloud state (connected, offline, syncing)
- Self-service troubleshooting

**For Support:**
- Fewer support requests (clear error messages)
- `shark cloud status` output for debugging
- Standardized setup flow (less variability)

## User Personas

### Persona 1: First-Time Cloud User

**Profile:**
- **Role:** Developer trying cloud sync for the first time
- **Experience:** Familiar with Shark CLI, new to Turso
- **Key Characteristics:**
  - Wants step-by-step guidance
  - Prefers interactive prompts over reading docs
  - Concerned about accidentally breaking existing local setup

**Goals Related to This Feature:**
1. Set up cloud sync without breaking local database
2. Understand what each step does (not black box)
3. Easy rollback if something goes wrong
4. Clear confirmation that setup worked

**Pain Points This Feature Addresses:**
- Fear of misconfiguration
- Uncertainty about what commands to run
- No visibility into what's happening during setup

**Success Looks Like:**
Runs `shark cloud init`, answers 3-4 prompts, sees "Setup complete!" message, runs `shark task list` to verify, switches to second machine and runs `shark cloud login` to connect.

### Persona 2: Multi-Workstation Power User

**Profile:**
- **Role:** Developer managing 3+ machines
- **Experience:** Advanced CLI user, scripts workflows
- **Key Characteristics:**
  - Values automation and scriptability
  - Needs non-interactive mode for CI/CD
  - Wants fine-grained control (manual sync, status checks)

**Goals Related to This Feature:**
1. Script cloud setup in dotfile automation
2. Check sync status before critical operations
3. Force immediate sync when needed (not wait 30s)
4. Monitor usage to stay within free tier

**Pain Points This Feature Addresses:**
- Can't script interactive prompts
- No way to check if cloud is up-to-date
- Blind to approaching free tier limits

**Success Looks Like:**
Runs `shark cloud init --non-interactive --url=... --token=$TOKEN` in setup script. Runs `shark cloud sync --push` before important demo. Checks `shark cloud status` daily to monitor usage.

## User Stories

### Must-Have Stories

**Story 1:** As a first-time user, I want to run `shark cloud init` and answer simple prompts so that I can set up cloud sync in 5 minutes without reading docs.

**Acceptance Criteria:**
- [ ] Command prompts for Turso account (create or existing)
- [ ] Guides through database creation (or uses existing)
- [ ] Asks whether to export local data to cloud
- [ ] Updates `.sharkconfig.json` automatically
- [ ] Provides shell export command for `SHARK_DB_URL`
- [ ] Verifies connection at end

**Story 2:** As a multi-machine user, I want to run `shark cloud login` on my second workstation so that I can connect to my existing cloud database.

**Acceptance Criteria:**
- [ ] Prompts for database URL (or auto-detects from account)
- [ ] Prompts for auth token (with secure input - no echo)
- [ ] Validates connection before saving config
- [ ] Downloads existing data from cloud
- [ ] Provides confirmation message with next steps

**Story 3:** As a developer, I want to run `shark cloud status` so that I can see connection health, sync state, and usage metrics.

**Acceptance Criteria:**
- [ ] Shows connection state: Connected | Offline | Error
- [ ] Shows last sync time and next sync time
- [ ] Shows usage stats: reads, writes, storage (% of free tier)
- [ ] Shows embedded replica status (if enabled)
- [ ] Provides actionable errors if unhealthy

**Story 4:** As a developer, I want to run `shark cloud sync` so that I can immediately push/pull changes without waiting for auto-sync.

**Acceptance Criteria:**
- [ ] `shark cloud sync` does bidirectional sync (default)
- [ ] `shark cloud sync --push` pushes local to cloud
- [ ] `shark cloud sync --pull` pulls cloud to local
- [ ] Shows sync progress and conflict warnings
- [ ] Returns error if offline

**Story 5:** As a developer, I want to run `shark cloud logout` so that I can disconnect from cloud and revert to local-only mode.

**Acceptance Criteria:**
- [ ] Clears auth token from config/env
- [ ] Reverts database URL to local SQLite
- [ ] Optionally keeps local replica as backup
- [ ] Confirms action before destructive changes
- [ ] Provides clear success message

**Story 6:** As an automation engineer, I want `--non-interactive` mode for all commands so that I can script cloud setup in dotfiles.

**Acceptance Criteria:**
- [ ] All commands accept `--non-interactive` flag
- [ ] Required parameters passed as flags (no prompts)
- [ ] Errors exit with non-zero code
- [ ] JSON output mode for parsing

### Should-Have Stories

**Story 7:** As a developer, I want `shark cloud init` to detect if I already have a Turso database so that I don't accidentally create duplicates.

**Acceptance Criteria:**
- [ ] Lists existing databases in account
- [ ] Prompts: "Use existing DB or create new?"
- [ ] Validates existing DB is compatible (schema version)

## Requirements

### Functional Requirements

**REQ-F-001: `shark cloud init` Command**
- **Description:** Interactive setup wizard for cloud database initialization
- **User Story:** Links to Story 1
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Prompts: Account credentials, database name, export local data (Y/n)
  - [ ] Creates Turso database via API (or uses existing)
  - [ ] Exports local data if requested
  - [ ] Updates `.sharkconfig.json` with cloud settings
  - [ ] Prints shell export command for user's profile
  - [ ] Verifies connection at end
  - [ ] `--non-interactive` mode with flags

**REQ-F-002: `shark cloud login` Command**
- **Description:** Connect to existing cloud database from another machine
- **User Story:** Links to Story 2
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Prompts: Database URL, auth token (secure input)
  - [ ] Tests connection before saving config
  - [ ] Downloads data from cloud (optional: overwrite local or merge)
  - [ ] Updates `.sharkconfig.json`
  - [ ] Success message with verification steps

**REQ-F-003: `shark cloud status` Command**
- **Description:** Display cloud connection state and usage metrics
- **User Story:** Links to Story 3
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Connection state: Connected | Offline | Error
  - [ ] Last sync: timestamp and duration
  - [ ] Next sync: ETA (if auto-sync enabled)
  - [ ] Usage: reads, writes, storage (numbers + % of free tier)
  - [ ] Embedded replica: enabled/disabled, file size
  - [ ] Warnings: approaching limits, stale sync, errors
  - [ ] `--json` mode for scripting

**REQ-F-004: `shark cloud sync` Command**
- **Description:** Manual sync trigger with directional control
- **User Story:** Links to Story 4
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Default: bidirectional sync
  - [ ] `--push`: local → cloud only
  - [ ] `--pull`: cloud → local only
  - [ ] `--dry-run`: show what would change
  - [ ] Progress indicator for large syncs
  - [ ] Conflict warnings (if any)
  - [ ] Error if offline (cannot sync)

**REQ-F-005: `shark cloud logout` Command**
- **Description:** Disconnect from cloud and revert to local mode
- **User Story:** Links to Story 5
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Prompts for confirmation (destructive action)
  - [ ] Clears auth token from config
  - [ ] Reverts `database.url` to local SQLite path
  - [ ] Option: `--keep-replica` (preserve local data)
  - [ ] Success message confirming local mode

**REQ-F-006: Backend Flag for Existing Commands**
- **Description:** Add `--backend` flag to all commands for temporary override
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] `--backend=local` forces local SQLite (ignores config)
  - [ ] `--backend=cloud` forces Turso (ignores config)
  - [ ] `--backend=auto` uses config (default)
  - [ ] Works on all commands (task, epic, feature, sync, etc.)

### Non-Functional Requirements

**Usability:**
- **REQ-NF-001:** Setup completes in < 5 minutes for typical user
- **Measurement:** User testing with 10 participants
- **Target:** 90% complete within 5 minutes
- **Justification:** Competitive with manual setup (30 minutes)

**Security:**
- **REQ-NF-002:** Auth tokens never displayed in plain text or logged
- **Measurement:** Code review + output inspection
- **Compliance:** OWASP A02 (Cryptographic Failures)
- **Risk Mitigation:** Prevents token leakage via logs/screenshots

**Reliability:**
- **REQ-NF-003:** Commands fail gracefully with actionable errors
- **Measurement:** Error message quality review
- **Target:** 100% of errors include "what happened" and "how to fix"
- **Justification:** Reduces support burden

## Technical Design

### Command Structure

```
shark cloud
├── init       [--non-interactive] [--url URL] [--token TOKEN] [--export]
├── login      [--non-interactive] [--url URL] [--token TOKEN]
├── status     [--json]
├── sync       [--push | --pull] [--dry-run] [--json]
└── logout     [--keep-replica] [--force]
```

### `shark cloud status` Output Example

```bash
$ shark cloud status

Cloud Database Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

 Connection:    ✓ Connected
 Database:      libsql://shark-tasks-jwwelbor.turso.io
 Region:        US East (Virginia)
 Last Sync:     2 minutes ago (2026-01-04 15:32:10)
 Next Sync:     in 28 seconds (auto-sync enabled)

Usage (Current Month)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

 Reads:         2.3M / 500M  [░░░░░░░░░░░░░░░░░░░░] 0.5%
 Writes:        145K / 10M   [░░░░░░░░░░░░░░░░░░░░] 1.5%
 Storage:       42MB / 5GB   [░░░░░░░░░░░░░░░░░░░░] 0.8%

Embedded Replica
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

 Status:        ✓ Enabled
 Location:      ~/.shark/turso-replica.db
 Size:          38MB
 Last Updated:  2 minutes ago

✓ All systems healthy
```

## Tasks

- **T-E13-F03-001:** Implement 'shark cloud init' command for database setup (Priority: 8)
- **T-E13-F03-002:** Implement 'shark cloud login' command for authentication (Priority: 8)
- **T-E13-F03-003:** Implement 'shark cloud sync' command for manual sync (Priority: 7)
- **T-E13-F03-006:** Add cloud backend flag to existing commands (Priority: 7)
- **T-E13-F03-004:** Implement 'shark cloud status' command to show connection state (Priority: 6)
- **T-E13-F03-005:** Implement 'shark cloud logout' command to clear credentials (Priority: 5)

## Dependencies

- **F01 (Database Abstraction Layer):** Needed for backend switching
- **F02 (Turso Integration):** Needed for actual cloud connection
- **External:** Turso CLI or API for database creation (optional - can be manual)

## Success Metrics

**Usability:**
- [ ] 90% of users complete setup in < 5 minutes (user testing)
- [ ] Zero users manually edit config files (telemetry)
- [ ] `shark cloud status` used in 80%+ of support requests

**Reliability:**
- [ ] 100% of errors include actionable fix instructions
- [ ] Zero auth tokens leaked in logs/output (security audit)
- [ ] Init success rate > 95% (telemetry)

**Adoption:**
- [ ] 20% of users run `shark cloud init` within 30 days of upgrade
- [ ] 80% of cloud users enable embedded replicas (telemetry)

## Out of Scope

### Explicitly Excluded

1. **Turso Account Management**
   - **Why:** Turso provides web UI and CLI for this
   - **Future:** Could integrate if users request it
   - **Workaround:** Use turso.tech dashboard

2. **Database Backups via CLI**
   - **Why:** Turso provides automated backups
   - **Future:** Could add `shark cloud backup` for manual backups
   - **Workaround:** Use Turso dashboard or `shark export`

3. **Team/Organization Management**
   - **Why:** Out of scope for v1 (single-user focus)
   - **Future:** If multi-user collaboration is added
   - **Workaround:** Each user has own database

4. **Database Branching (dev/staging/prod)**
   - **Why:** Adds complexity, not needed for task management
   - **Future:** Could add if users request it
   - **Workaround:** Use separate databases manually

## Security Considerations

**Token Storage:**
- Environment variables preferred (ephemeral, process-isolated)
- File storage at `~/.shark/turso-token` with `chmod 600`
- Never in `.sharkconfig.json` (to avoid Git commits)
- Never logged or displayed in output

**Token Transmission:**
- HTTPS only for Turso API calls
- Token passed in Authorization header (not URL params)
- Secure input (no echo) when prompting user

**Validation:**
- Test connection before saving config
- Rollback on failure (don't leave invalid state)
- Clear error messages (avoid leaking token in errors)

---

*Last Updated:* 2026-01-04
*Dependencies:* F01, F02
