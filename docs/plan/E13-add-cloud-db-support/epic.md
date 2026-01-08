# Epic E13: Cloud Database Support

**Epic Key:** E13
**Status:** Draft
**Priority:** Medium
**Business Value:** Medium

## Executive Summary

Add cloud database support to Shark Task Manager using Turso (libSQL), enabling developers to sync tasks across multiple workstations while maintaining offline capabilities.

**Problem:** Developers working on multiple machines (work desktop, home laptop, remote VM) must manually sync the SQLite database file via Git, Dropbox, or USB drives, leading to conflicts and stale data.

**Solution:** Integrate Turso (cloud SQLite) as an optional backend, allowing automatic real-time sync across devices while preserving offline functionality through embedded replicas.

**Impact:**
- Enable seamless multi-workstation workflows
- Zero cost for 90% of users (generous free tier)
- 5-minute setup time
- No breaking changes (opt-in feature)
- Works offline with automatic sync when online

## Background & Context

### Current State

Shark Task Manager uses local SQLite (`shark-tasks.db`) for all data storage:
- **Pros:** Fast, simple, no network dependency, works offline
- **Cons:** Single-machine only, manual sync required, no real-time collaboration

### User Pain Points

**Target User:** Developer with 2-3 workstations
- Work desktop (primary development machine)
- Home laptop (evening work, travel)
- Cloud VM or remote machine (occasionally)

**Current Workflow (Painful):**
```bash
# On work desktop
shark task complete T-E07-F20-001
git add shark-tasks.db
git commit -m "Update tasks"
git push

# At home (hours later)
git pull  # Might have conflicts!
# If forgot to commit: tasks are stale
```

**Problems:**
- Manual sync required (easy to forget)
- Git conflicts on binary SQLite file
- Stale data if sync is missed
- Can't work on multiple machines simultaneously

### Desired State

**With Cloud Support:**
```bash
# One-time setup (5 minutes)
shark cloud init
export SHARK_DB_URL="libsql://shark-tasks-username.turso.io"

# Work seamlessly across all machines
shark task complete T-E07-F20-001  # Syncs immediately
# Switch to laptop - tasks already updated!
```

## Strategic Objectives

### Primary Goals

1. **Enable Multi-Workstation Workflow:** Allow developers to work on any machine without manual sync
2. **Zero Friction Setup:** 5-minute cloud setup vs 30+ minutes for alternatives
3. **Preserve Offline Capability:** Critical for CLI tool - must work on planes, coffee shops, etc.
4. **No Breaking Changes:** Existing local-only users unaffected

### Success Metrics

**Adoption:**
- 20% of active users enable cloud sync within 3 months
- 5+ positive user testimonials about multi-machine workflow

**Technical:**
- Setup time < 5 minutes (measured via telemetry)
- 95%+ of commands complete in < 200ms (local replica speed)
- Zero data loss incidents

**Business:**
- Free tier sufficient for 90% of users
- $5/month average cost for power users

## Solution Overview

### Technology Choice: Turso (libSQL)

**Why Turso Won:**

| Criteria | Turso ⭐ | Supabase | AWS Aurora | LiteFS |
|----------|---------|----------|------------|--------|
| Free Tier | ✅ 500M rows | ✅ 500MB | ❌ Limited | ⚠️ OSS |
| Setup Time | ✅ 5 min | ⚠️ 10 min | ❌ 45 min | ⚠️ 30 min |
| SQLite Compatible | ✅ Yes | ❌ No (PG) | ❌ No (MySQL) | ✅ Yes |
| Offline Support | ✅ Replicas | ❌ No | ❌ No | ✅ Yes |
| Migration Effort | ✅ 1-2 weeks | ❌ 3-4 weeks | ❌ 3-4 weeks | ✅ 1-2 weeks |
| Monthly Cost | $0-5 | $0-25 | $45+ | $5-20 |

### Architecture

**Database Abstraction Layer:**
```
┌─────────────────┐
│  Shark CLI      │
└────────┬────────┘
         │
    ┌────┴─────┐
    │ DB Repo  │ (abstracted)
    └────┬─────┘
         │
    ┌────┴──────────┐
    │  DB Interface │
    └────┬──────────┘
         │
    ┌────┴────────────────┐
    │                     │
┌───┴──────┐     ┌───────┴────┐
│ SQLite   │     │   Turso    │
│ Driver   │     │ (libSQL)   │
└──────────┘     └────────────┘
```

**Embedded Replica for Offline:**
```
┌──────────────┐      Sync       ┌─────────────┐
│ Local Replica│ ◄────────────► │ Cloud Turso │
│  (SQLite)    │  (when online)  │  Database   │
└──────────────┘                 └─────────────┘
      ↑
      │ Fast reads/writes
      │ (works offline)
      ↓
┌──────────────┐
│  Shark CLI   │
└──────────────┘
```

### Configuration

**Priority Order:**
1. Command-line flag: `--db=libsql://...`
2. Environment variable: `SHARK_DB_URL`
3. Config file: `.sharkconfig.json`
4. Default: Local SQLite

**Example .sharkconfig.json:**
```json
{
  "database": {
    "backend": "turso",
    "url": "libsql://shark-tasks-username.turso.io",
    "embedded_replica": true
  }
}
```

## Features & Implementation

### F01: Database Abstraction Layer
Create interface to support multiple database backends without changing business logic.

**Key Components:**
- Database interface with CRUD operations
- Driver registry for backend selection
- Configuration-driven backend selection

### F02: Turso Integration
Implement libSQL driver with connection pooling, auth, and offline support.

**Key Components:**
- libSQL Go driver integration
- Authentication token management
- Embedded replica for offline mode
- Connection pooling and retry logic

### F03: Cloud CLI Commands
User-facing commands for cloud database management.

**New Commands:**
- `shark cloud init` - Create cloud database
- `shark cloud login` - Connect to existing cloud DB
- `shark cloud sync` - Manual sync trigger
- `shark cloud status` - Show connection state
- `shark cloud logout` - Clear credentials

### F04: Migration and Sync Tools
Tools for moving data between local and cloud databases.

**Key Components:**
- Export local → cloud
- Import cloud → local
- Bidirectional sync with conflict detection
- Conflict resolution strategies
- Backup/restore utilities

### F05: Documentation and Examples
Comprehensive user-facing documentation.

**Deliverables:**
- Quick start guide (Turso setup)
- Migration guide (local → cloud)
- Troubleshooting guide
- Multi-workstation workflow examples
- Updated CLI reference
- Updated CLAUDE.md

## User Journeys

### Journey 1: First-Time Cloud Setup

**Actor:** Developer with local Shark database

**Steps:**
1. Run `shark cloud init`
2. Prompted to create Turso account (or login)
3. Choose database name: `shark-tasks-username`
4. Export existing local data to cloud? (Y/n): Y
5. System exports data, updates config
6. Add `export SHARK_DB_URL=...` to shell profile
7. Verification: `shark task list` shows same tasks from cloud

**Duration:** 5 minutes
**Outcome:** Cloud database created, local data migrated, ready for multi-machine use

### Journey 2: Connecting Second Workstation

**Actor:** Developer setting up cloud sync on another machine

**Steps:**
1. Install Shark CLI on second machine
2. Run `shark cloud login`
3. Enter cloud database URL (from first machine)
4. Enter auth token (securely shared)
5. System tests connection, downloads data
6. Add `export SHARK_DB_URL=...` to shell profile
7. Verification: `shark task list` shows tasks from cloud

**Duration:** 2 minutes
**Outcome:** Second machine connected, all data synced

### Journey 3: Offline Work with Auto-Sync

**Actor:** Developer on airplane (no internet)

**Steps:**
1. Open laptop (no wifi available)
2. Run `shark task list` → Uses local embedded replica (fast!)
3. Work normally: `shark task start`, `shark task complete`
4. All changes saved to local replica
5. Later: Connect to wifi
6. Shark auto-syncs local changes to cloud (background)
7. Changes appear on other workstations automatically

**Outcome:** Seamless offline work, automatic sync when online

## Out of Scope

### Explicitly Excluded

1. **Multi-User Collaboration**
   - **Why:** Complexity too high for v1 (permissions, access control, conflict resolution)
   - **Future:** Consider for v2 if teams request it
   - **Workaround:** Each team member has separate database

2. **Real-Time Collaborative Editing**
   - **Why:** Not needed for task management (not like Google Docs)
   - **Future:** Unlikely - tasks are discrete units
   - **Workaround:** Use chat/Slack for coordination

3. **Self-Hosted Cloud Option**
   - **Why:** Adds infrastructure complexity, deviates from "zero-friction" goal
   - **Future:** Evaluate if enterprises request it
   - **Workaround:** Use local SQLite with network mount (existing capability)

4. **Migration to Other Cloud DBs (Supabase, AWS, etc.)**
   - **Why:** Turso is best fit for current architecture
   - **Future:** Database abstraction layer makes this possible later
   - **Workaround:** Export to SQL, manual migration if needed

### Alternative Approaches Rejected

**Alternative 1: Supabase (PostgreSQL)**
- **Why Rejected:** Requires 3-4 weeks migration (query syntax, schema conversion), no offline support
- **Trade-off:** Better for multi-user collaboration, but not needed for v1

**Alternative 2: AWS Aurora Serverless**
- **Why Rejected:** $45+/month minimum cost, 45-minute setup, no free tier
- **Trade-off:** Enterprise-grade, but overkill for individual developers

**Alternative 3: Git-Based Sync**
- **Why Rejected:** Binary SQLite file causes merge conflicts, manual process
- **Trade-off:** No cost, but defeats purpose (same problem we're solving)

## Dependencies & Constraints

### External Dependencies

- **Turso Service:** Relies on Turso.tech availability (SLA: 99.9%)
- **libSQL Go Driver:** https://github.com/tursodatabase/libsql-client-go
- **Internet Connection:** Required for cloud sync (optional with embedded replicas)

### Technical Constraints

- **SQLite Compatibility:** Must maintain existing schema and query patterns
- **Backward Compatibility:** Local-only mode must continue to work unchanged
- **Performance:** Cloud operations should be < 300ms (local replica: < 50ms)

### Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Turso service shutdown | High | Abstract DB layer, document PostgreSQL migration path |
| Free tier reduction | Medium | Make opt-in, document pricing clearly, provide local fallback |
| Low adoption | Low | Target multi-machine users specifically, gather feedback early |
| Data loss/corruption | High | Built-in backups, export utilities, local replicas as backup |
| Network dependency | Medium | Embedded replicas for offline support |

## Success Criteria

### Must-Have (MVP)

- [ ] Cloud database creates successfully via `shark cloud init`
- [ ] Data exports from local to cloud without errors
- [ ] Second workstation connects and sees same tasks
- [ ] Offline mode works (embedded replicas)
- [ ] All existing commands work with cloud backend
- [ ] Zero data loss in testing (100+ scenarios)
- [ ] Documentation covers all workflows

### Should-Have (Post-MVP)

- [ ] Auto-sync in background (no manual `shark cloud sync`)
- [ ] Conflict resolution UI for simultaneous edits
- [ ] Telemetry on adoption rates
- [ ] Cost estimator (based on usage)

### Could-Have (Future)

- [ ] Multi-user support (team databases)
- [ ] Read replicas in different regions
- [ ] Point-in-time recovery
- [ ] Database branching (dev/staging/prod)

## Timeline & Phasing

### Phase 1: Basic Cloud Support (1-2 weeks)

**Week 1:**
- Database abstraction layer
- Turso driver integration
- Basic CLI commands (init, login)

**Week 2:**
- Export/import utilities
- Embedded replica support
- Documentation
- Testing

### Phase 2: Enhanced Features (Future)

- Auto-sync optimization
- Conflict resolution UI
- Advanced telemetry
- Enterprise features (if needed)

## Related Documents

- **Feasibility Analysis:** `dev-artifacts/2026-01-04-cloud-db-evaluation/cloud-database-feasibility-analysis.md`
- **Executive Summary:** `dev-artifacts/2026-01-04-cloud-db-evaluation/executive-summary.md`
- **Quick Comparison:** `dev-artifacts/2026-01-04-cloud-db-evaluation/quick-comparison.md`
- **Configuration Design:** `dev-artifacts/2026-01-04-cloud-db-evaluation/configuration-design.md`

## Approvals

- [ ] Product Owner: _______________
- [ ] Tech Lead: _______________
- [ ] Architecture Review: _______________

---

*Last Updated:* 2026-01-04
*Created From:* Idea I-2026-01-03-02
