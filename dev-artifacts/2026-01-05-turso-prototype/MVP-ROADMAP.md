# E13 MVP Implementation Roadmap

**Goal:** Migrate existing shark database to cloud and enable multi-workstation sync

**Timeline:** 1 week (vs 3-4 weeks for full implementation)

**Tasks:** 15 tasks (vs 27 for full)

---

## MVP Scope

### What You Get
- ✅ Migrate existing `shark-tasks.db` to Turso cloud
- ✅ Work from multiple workstations with same data
- ✅ Offline mode with embedded replicas (instant local performance)
- ✅ Manual sync when switching workstations
- ✅ Basic cloud setup and configuration

### What's Deferred to Full Implementation
- ❌ Automatic conflict resolution (manual fix if needed)
- ❌ Bidirectional auto-sync (use manual sync)
- ❌ Backup/restore utilities
- ❌ Advanced cloud monitoring
- ❌ Comprehensive troubleshooting docs

---

## MVP Feature Breakdown

### F01: Database Abstraction Layer (ALL 5 tasks)
**Why needed:** Foundation to support multiple backends (local SQLite + Turso)

- [P9] T-E13-F01-005: Add configuration fields for backend selection
- [P9] T-E13-F01-001: Define database interface with CRUD operations
- [P8] T-E13-F01-002: Refactor SQLite repository to implement interface
- [P7] T-E13-F01-003: Implement driver registry for backend selection
- [P6] T-E13-F01-004: Write unit tests for abstraction layer

**Status:** 5/5 tasks required for MVP

---

### F02: Turso Integration (ALL 5 tasks)
**Why needed:** The cloud database backend

- [P9] T-E13-F02-001: Add libSQL Go driver dependency to go.mod
- [P8] T-E13-F02-002: Implement Turso driver with connection pooling
- [P7] T-E13-F02-003: Add embedded replica support for offline mode
- [P7] T-E13-F02-004: Implement authentication token management
- [P6] T-E13-F02-005: Write integration tests for Turso connectivity

**Status:** 5/5 tasks required for MVP

---

### F03: Cloud CLI Commands (3 of 6 tasks)
**Why partial:** Need setup and basic sync, skip monitoring features

✅ **MVP Tasks:**
- [P8] T-E13-F03-001: Implement 'shark cloud init' command
- [P8] T-E13-F03-002: Implement 'shark cloud login' command
- [P7] T-E13-F03-003: Implement 'shark cloud sync' command

❌ **Deferred to Full:**
- [P7] T-E13-F03-006: Add cloud backend flag to existing commands
- [P6] T-E13-F03-004: Implement 'shark cloud status' command
- [P5] T-E13-F03-005: Implement 'shark cloud logout' command

**Status:** 3/6 tasks required for MVP

---

### F04: Migration and Sync Tools (2 of 5 tasks)
**Why partial:** Need one-time migration, skip advanced sync features

✅ **MVP Tasks:**
- [P8] T-E13-F04-001: Implement export local database to cloud functionality
- [P8] T-E13-F04-002: Implement import cloud database to local functionality

❌ **Deferred to Full:**
- [P9] T-E13-F04-003: Implement bidirectional sync with conflict detection
- [P7] T-E13-F04-004: Implement conflict resolution strategies
- [P6] T-E13-F04-005: Add backup and restore utilities

**Status:** 2/5 tasks required for MVP

---

### F05: Documentation and Examples (1 of 6 tasks)
**Why partial:** Just need quick start, skip comprehensive guides

✅ **MVP Tasks:**
- [P7] T-E13-F05-001: Write quick start guide for Turso setup

❌ **Deferred to Full:**
- [P8] T-E13-F05-005: Update CLI reference with cloud commands
- [P7] T-E13-F05-002: Write migration guide (local → cloud)
- [P6] T-E13-F05-003: Write troubleshooting guide
- [P6] T-E13-F05-004: Create example workflows for multi-workstation
- [P5] T-E13-F05-006: Update CLAUDE.md with cloud instructions

**Status:** 1/6 tasks required for MVP

---

## Execution Order (Sequential)

### Phase 1: Foundation (F01 - Database Abstraction)
**Timeline:** 2 days

Execute in order:
1. T-E13-F01-005: Add config fields (foundation)
2. T-E13-F01-001: Define interface (contracts)
3. T-E13-F01-002: Refactor SQLite to use interface (make existing code pluggable)
4. T-E13-F01-003: Implement driver registry (backend selection mechanism)
5. T-E13-F01-004: Write unit tests (verify abstraction works)

**Completion Criteria:**
- Existing shark commands still work with local SQLite
- Config supports `database.backend` setting
- Interface defined for new backends

---

### Phase 2: Turso Backend (F02 - Turso Integration)
**Timeline:** 2-3 days

Execute in order:
1. T-E13-F02-001: Add libSQL dependency (setup)
2. T-E13-F02-002: Implement Turso driver (cloud connection)
3. T-E13-F02-004: Implement auth token management (security)
4. T-E13-F02-003: Add embedded replica support (offline mode)
5. T-E13-F02-005: Write integration tests (verify Turso works)

**Completion Criteria:**
- Can connect to Turso cloud database
- Embedded replicas work (local + cloud sync)
- Auth tokens stored securely
- All tests pass

---

### Phase 3: Cloud Setup (F03 - Cloud CLI)
**Timeline:** 1 day

Execute in order:
1. T-E13-F03-001: Implement `shark cloud init` (create Turso DB, save config)
2. T-E13-F03-002: Implement `shark cloud login` (auth token input)
3. T-E13-F03-003: Implement `shark cloud sync` (manual push/pull)

**Completion Criteria:**
- `shark cloud init` creates Turso database
- `shark cloud login` saves credentials
- `shark cloud sync` works both directions

---

### Phase 4: Migration Tools (F04 - Export/Import)
**Timeline:** 1-2 days

Execute in order:
1. T-E13-F04-001: Implement export local → cloud (migrate existing data)
2. T-E13-F04-002: Implement import cloud → local (setup second workstation)

**Completion Criteria:**
- Can migrate existing `shark-tasks.db` to cloud without data loss
- Second workstation can pull cloud data to local
- All epics, features, tasks, history preserved

---

### Phase 5: Documentation (F05 - Quick Start)
**Timeline:** 0.5 day

Execute:
1. T-E13-F05-001: Write quick start guide for Turso setup

**Completion Criteria:**
- Guide covers: setup, migration, multi-workstation usage
- Takes <5 minutes to follow
- Includes troubleshooting for common issues

---

## MVP Usage Flow

### Initial Migration (First Time Setup)

```bash
# 1. Setup Turso database
shark cloud init
# Interactive wizard:
#   - Creates Turso database
#   - Saves credentials to config
#   - Configures embedded replica

# 2. Migrate existing data
shark export --to=cloud
# Copies all data from shark-tasks.db to Turso

# 3. Switch to cloud mode
shark config set database.backend turso

# 4. Verify migration
shark task list --json
shark epic list --json
```

### Second Workstation Setup

```bash
# 1. Install shark CLI
# 2. Login to cloud
shark cloud login
# Prompts for database URL and auth token

# 3. Import cloud data
shark import --from=cloud

# 4. Configure cloud mode
shark config set database.backend turso

# 5. Use shark normally
shark task list
shark task create E07 F01 "New task"
```

### Daily Workflow

```bash
# Workstation A
shark task create E07 F01 "Implement feature X"
shark cloud sync --push

# Switch to Workstation B
shark cloud sync --pull
shark task list  # See new task
shark task start E07-F01-001
shark cloud sync --push

# Back to Workstation A
shark cloud sync --pull
shark task list  # See updated status
```

---

## Success Criteria for MVP

### Must Work
- [ ] Migrate existing shark-tasks.db to cloud (zero data loss)
- [ ] Second workstation can pull cloud data
- [ ] Create/update/delete operations sync to cloud
- [ ] Offline mode works (embedded replicas)
- [ ] Manual sync between workstations works
- [ ] All 15 MVP tasks completed and approved

### Performance Targets
- [ ] Local reads <50ms (embedded replica)
- [ ] Cloud sync <5s for typical dataset (10-100 tasks)
- [ ] Setup time <10 minutes (init + migration)

### User Experience
- [ ] Setup wizard is self-explanatory
- [ ] Error messages include actionable fixes
- [ ] Quick start guide is accurate and complete

---

## Post-MVP Enhancements (Full Implementation)

**If MVP succeeds and you want more features:**

1. **Automatic Sync** (T-E13-F03-006)
   - Add `--cloud` flag to all commands
   - Auto-sync on every operation

2. **Conflict Resolution** (T-E13-F04-003, T-E13-F04-004)
   - Detect when same task edited on 2 machines
   - Smart merge strategies (last-write-wins, manual resolution)

3. **Monitoring** (T-E13-F03-004, T-E13-F03-005)
   - `shark cloud status` shows sync health
   - `shark cloud logout` clears credentials

4. **Safety Features** (T-E13-F04-005)
   - Backup before migration
   - Rollback on failure
   - Dry-run mode

5. **Documentation** (T-E13-F05-002 through T-E13-F05-006)
   - Migration guide
   - Troubleshooting guide
   - Example workflows
   - Updated CLAUDE.md

**Estimated:** 2-3 weeks additional work

---

## Risks & Mitigation

| Risk | MVP Impact | Mitigation |
|------|-----------|-----------|
| Data loss during migration | High | Export creates backup before migration; manual verification step |
| Sync conflicts (edit same task on 2 machines) | Medium | Manual fix required; document in quick start |
| Turso service issues | Medium | Embedded replica allows offline work; can revert to local SQLite |
| Auth token leaked | Low | Store in config file with restricted permissions |
| Performance degradation | Low | Embedded replicas ensure local speed; prototype validated performance |

---

## Implementation Strategy

### Recommended: Sequential with TDD

Use developer agent with TDD for each phase:

```bash
# Phase 1: Foundation
use developer agent with TDD to implement F01 tasks in order

# Phase 2: Turso
use developer agent with TDD to implement F02 tasks in order

# Phase 3: CLI
use developer agent with TDD to implement F03 MVP tasks

# Phase 4: Migration
use developer agent with TDD to implement F04 MVP tasks

# Phase 5: Docs
write quick start guide for F05
```

**Why TDD:**
- Prevents regressions in existing shark commands
- Ensures cloud mode doesn't break local mode
- Validates migration preserves all data
- Catches edge cases early

---

## Next Steps

1. **Review this roadmap** - Confirm MVP scope matches your needs
2. **Start Phase 1** - Begin F01 database abstraction
3. **Track progress** - Use shark to track task status
4. **Test incrementally** - Validate each phase before moving to next

**Ready to start?** Let me know if this MVP scope works for you, or if we should adjust.
