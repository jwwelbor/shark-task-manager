# Feature: Turso Integration

**Feature Key:** E13-F02
**Epic:** E13 - Cloud Database Support
**Status:** Draft
**Execution Order:** 2

## Overview

Integrate Turso (libSQL) as a cloud database backend for Shark Task Manager, enabling real-time multi-workstation sync while maintaining offline functionality through embedded replicas.

## Goal

### Problem

Developers working across multiple machines (work desktop, home laptop, cloud VM) currently face:
- Manual database sync via Git/Dropbox (error-prone, conflicts on binary SQLite files)
- Stale data when forgetting to sync
- No real-time updates across machines
- Cannot work simultaneously on multiple devices

Traditional cloud databases (PostgreSQL, MySQL) would require:
- 3-4 weeks of migration work (query syntax, schema changes)
- Always-online requirement (no offline mode)
- Higher cost ($15-45/month minimum)
- Breaking changes to existing codebase

### Solution

Integrate Turso (libSQL), a distributed SQLite platform that provides:
- **SQLite-compatible:** Existing queries work without modification (zero breaking changes)
- **Embedded replicas:** Local SQLite replica syncs with cloud (works offline)
- **Generous free tier:** 500M rows read, 10M rows write, 5GB storage (sufficient for 90% of users)
- **5-minute setup:** Create database, export env vars, start using
- **Real-time sync:** Changes propagate automatically across workstations

### Impact

**For Users:**
- Work seamlessly across 2-3 machines without manual sync
- Offline capability preserved (airplane, coffee shop, no wifi)
- Zero monthly cost for typical usage (< 10M writes/month)
- No behavior changes to existing commands

**For Development:**
- Minimal code changes (1-2 weeks vs 3-4 weeks for PostgreSQL)
- Backward compatible (opt-in feature)
- Future-proof (abstraction layer allows other backends later)

## User Personas

### Persona 1: Multi-Machine Developer

**Profile:**
- **Role:** Software developer or DevOps engineer
- **Experience Level:** Comfortable with CLI tools, Git, environment variables
- **Setup:** Work desktop (primary), home laptop (secondary), occasional cloud VM
- **Key Characteristics:**
  - Works on tasks throughout the day across different locations
  - Needs up-to-date task status regardless of machine
  - Occasionally works offline (travel, coffee shops)

**Goals Related to This Feature:**
1. See latest task status on any machine without manual sync
2. Work offline and have changes sync automatically when back online
3. Avoid Git merge conflicts on database files
4. Zero-friction setup (< 10 minutes total)

**Pain Points This Feature Addresses:**
- Forgetting to commit/push database changes before leaving work
- Stale task list when switching machines
- Binary SQLite conflicts in Git
- Manual sync overhead

**Success Looks Like:**
Completes a task on work desktop, switches to laptop at home, immediately sees task marked complete without any manual sync steps. Works on plane with no wifi, changes auto-sync when landing.

### Persona 2: Cost-Conscious Developer

**Profile:**
- **Role:** Independent developer, student, or hobbyist
- **Budget:** Prefers free tools, willing to pay $5/month for real value
- **Key Characteristics:**
  - Evaluates free tiers carefully
  - Avoids vendor lock-in
  - Values simplicity over enterprise features

**Goals Related to This Feature:**
1. Use cloud sync without monthly fees
2. Understand pricing clearly (no surprises)
3. Easy opt-out if switching back to local
4. Data export capability (avoid lock-in)

**Pain Points This Feature Addresses:**
- AWS/Azure pricing confusion ($45+ minimum)
- Hidden costs in "free" tiers
- No offline mode (pay even when not syncing)

**Success Looks Like:**
Enables cloud sync, uses it for 6 months on free tier, exports data back to local SQLite if needed. Clear pricing page shows $0/month for current usage.

## User Stories

### Must-Have Stories

**Story 1:** As a developer, I want to connect Shark to a Turso cloud database so that my tasks sync across multiple workstations automatically.

**Acceptance Criteria:**
- [ ] Turso connection established with URL and auth token
- [ ] All existing tasks visible in cloud database
- [ ] New tasks created locally appear in cloud immediately
- [ ] Connection errors show clear, actionable messages

**Story 2:** As a developer, I want to work offline with embedded replicas so that I can use Shark on planes, trains, and areas with no internet.

**Acceptance Criteria:**
- [ ] Embedded replica configured automatically during setup
- [ ] All commands work offline (reads from local replica)
- [ ] Writes saved to local replica while offline
- [ ] Automatic sync to cloud when connection restored
- [ ] No data loss during offline period

**Story 3:** As a developer, I want authentication to be secure and persistent so that I don't need to re-enter credentials every time.

**Acceptance Criteria:**
- [ ] Auth token stored securely (environment variable or token file)
- [ ] Token persists across CLI sessions
- [ ] Invalid token shows clear error message
- [ ] Token never logged or displayed in plain text

**Story 4:** As a developer, I want connection pooling and retry logic so that temporary network issues don't cause command failures.

**Acceptance Criteria:**
- [ ] Connection pool maintains 5-10 active connections
- [ ] Failed requests retry 3 times with exponential backoff
- [ ] Timeout after 10 seconds with clear error
- [ ] Connection health check on startup

### Should-Have Stories

**Story 5:** As a developer, I want to monitor my Turso usage so that I know if I'm approaching free tier limits.

**Acceptance Criteria:**
- [ ] `shark cloud status` shows read/write counts
- [ ] Warning when approaching 80% of free tier
- [ ] Link to Turso dashboard for detailed usage

### Could-Have Stories

**Story 6:** As a developer, I want read replicas in multiple regions so that commands are fast regardless of my location.

**Acceptance Criteria:**
- [ ] Turso automatically routes reads to nearest replica
- [ ] Query latency < 100ms from US, EU, Asia
- [ ] Replication lag < 5 seconds

## Requirements

### Functional Requirements

**REQ-F-001: libSQL Driver Integration**
- **Description:** Add libSQL Go driver as dependency and implement connection logic
- **User Story:** Links to Story 1
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] `go.mod` includes `github.com/tursodatabase/libsql-client-go`
  - [ ] Driver registered in database abstraction layer
  - [ ] Connection established with URL: `libsql://db-name.turso.io`
  - [ ] Auth token passed via query parameter or header

**REQ-F-002: Connection Pooling**
- **Description:** Implement connection pooling for efficient resource usage
- **User Story:** Links to Story 4
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Pool size configurable (default: 10 connections)
  - [ ] Idle connections closed after 5 minutes
  - [ ] Connection reuse across requests
  - [ ] Pool metrics available for debugging

**REQ-F-003: Embedded Replica Support**
- **Description:** Configure local embedded replica for offline mode
- **User Story:** Links to Story 2
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Local replica file at `~/.shark/turso-replica.db`
  - [ ] Replica syncs with cloud on startup
  - [ ] Reads use local replica (fast!)
  - [ ] Writes sync to cloud in background
  - [ ] Offline writes queued and synced when online

**REQ-F-004: Authentication Token Management**
- **Description:** Securely store and retrieve Turso auth tokens
- **User Story:** Links to Story 3
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Token read from `TURSO_AUTH_TOKEN` env var (first priority)
  - [ ] Token read from `~/.shark/turso-token` file (second priority)
  - [ ] Token file has `chmod 600` permissions (enforced)
  - [ ] Invalid token returns clear error with fix instructions
  - [ ] Token never logged or displayed in output

**REQ-F-005: Retry Logic and Error Handling**
- **Description:** Implement robust retry logic for transient network failures
- **User Story:** Links to Story 4
- **Priority:** Must-Have
- **Acceptance Criteria:**
  - [ ] Failed requests retry 3 times
  - [ ] Exponential backoff: 1s, 2s, 4s
  - [ ] Timeout after 10 seconds total
  - [ ] Clear error messages distinguish network vs auth vs server errors

### Non-Functional Requirements

**Performance:**
- **REQ-NF-001:** Embedded replica reads complete in < 50ms (same as local SQLite)
- **Measurement:** Benchmark `shark task list` with 1000+ tasks
- **Target:** 95th percentile < 50ms
- **Justification:** Must not degrade user experience vs local SQLite

**Reliability:**
- **REQ-NF-002:** Cloud sync success rate > 99.5%
- **Measurement:** Track sync failures over 30 days
- **Target:** < 0.5% failure rate (excluding user's internet outages)
- **Justification:** Users must trust cloud database

**Offline Capability:**
- **REQ-NF-003:** All read operations work 100% offline
- **Measurement:** Disable network, run all read commands
- **Target:** 0 failures when offline
- **Justification:** Critical for CLI tool used during travel

**Security:**
- **REQ-NF-004:** Auth tokens stored with file permissions 600 or in env vars only
- **Measurement:** Code review + security scan
- **Compliance:** OWASP A02:2021 (Cryptographic Failures)
- **Risk Mitigation:** Prevents token theft from filesystem

## Technical Design

### Architecture: Embedded Replica Pattern

```
┌─────────────────────────────────────────────────┐
│                 Shark CLI                        │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│          Turso Driver (libSQL)                   │
│  ┌──────────────────────────────────────────┐   │
│  │   Embedded Replica (~/.shark/replica.db) │   │
│  │                                           │   │
│  │   ┌─────────────┐    ┌─────────────┐    │   │
│  │   │  Local Read │    │ Local Write │    │   │
│  │   │  (instant)  │    │   (queue)   │    │   │
│  │   └─────────────┘    └──────┬──────┘    │   │
│  └──────────────────────────────┼───────────┘   │
│                                 │                │
│                                 ▼                │
│         ┌────────────────────────────┐          │
│         │   Background Sync Thread   │          │
│         │  (syncs to cloud when online)         │
│         └────────────┬───────────────┘          │
└──────────────────────┼──────────────────────────┘
                       │
                       ▼ (HTTPS/WSS)
        ┌──────────────────────────────┐
        │    Turso Cloud (Primary)     │
        │   libsql://db.turso.io       │
        └──────────────────────────────┘
```

### Connection String Format

```
libsql://[database-name].[org-name].turso.io?authToken=[token]
```

**Example:**
```bash
export SHARK_DB_URL="libsql://shark-tasks-jwwelbor.turso.io"
export TURSO_AUTH_TOKEN="eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9..."
```

### Code Structure

**File: `internal/db/drivers/turso/turso.go`**
```go
package turso

import (
    "context"
    "database/sql"
    _ "github.com/tursodatabase/libsql-client-go"
    "github.com/yourorg/shark/internal/db"
)

type TursoDriver struct {
    conn     *sql.DB
    url      string
    token    string
    syncFile string // Path to embedded replica
}

func init() {
    // Register with database abstraction layer
    db.RegisterDriver("turso", func() db.Database {
        return &TursoDriver{}
    })
}

func (d *TursoDriver) Connect(ctx context.Context, dsn string) error {
    // Parse libsql:// URL
    // Get auth token from env or file
    // Enable embedded replica if configured
    // Open connection with libSQL driver

    connStr := fmt.Sprintf("%s?authToken=%s", dsn, d.token)
    if d.syncFile != "" {
        connStr += fmt.Sprintf("&syncUrl=%s&syncInterval=30", dsn)
    }

    conn, err := sql.Open("libsql", connStr)
    if err != nil {
        return fmt.Errorf("failed to connect to Turso: %w", err)
    }

    d.conn = conn
    return d.Ping(ctx)
}

func (d *TursoDriver) Query(ctx context.Context, query string, args ...interface{}) (db.Rows, error) {
    // Reads from embedded replica (fast, works offline)
    return d.conn.QueryContext(ctx, query, args...)
}

func (d *TursoDriver) Exec(ctx context.Context, query string, args ...interface{}) (db.Result, error) {
    // Writes to embedded replica, syncs to cloud in background
    return d.conn.ExecContext(ctx, query, args...)
}
```

### Embedded Replica Configuration

**Auto-sync settings:**
- Sync interval: 30 seconds (configurable)
- Batch size: 1000 operations
- Retry failed syncs: 3 times with exponential backoff
- Conflict resolution: Last-write-wins (cloud timestamp)

**Storage:**
- Replica file: `~/.shark/turso-replica.db`
- Sync metadata: `~/.shark/turso-sync.json` (tracks last sync timestamp)
- Token file: `~/.shark/turso-token` (chmod 600)

### Authentication Flow

```
1. Check TURSO_AUTH_TOKEN env var
   ├─ Found → Use it
   └─ Not found → Check ~/.shark/turso-token file
      ├─ Found → Read token (verify chmod 600)
      └─ Not found → Error: "Run 'shark cloud login' to authenticate"

2. Validate token format
   ├─ JWT format → OK
   └─ Invalid → Error: "Invalid token format"

3. Test connection with token
   ├─ Success → Proceed
   └─ Failure → Error: "Authentication failed. Check token validity"
```

## Tasks

- **T-E13-F02-001:** Add libSQL Go driver dependency to go.mod (Priority: 9)
- **T-E13-F02-002:** Implement Turso database driver with connection pooling (Priority: 8)
- **T-E13-F02-003:** Add embedded replica support for offline mode (Priority: 7)
- **T-E13-F02-004:** Implement authentication token management (Priority: 7)
- **T-E13-F02-005:** Write integration tests for Turso connectivity (Priority: 6)

## Dependencies

- **F01 (Database Abstraction Layer):** Must be completed first - provides interface for Turso driver
- **External:** Turso service (turso.tech) availability and API stability
- **External:** libSQL Go driver maintenance (github.com/tursodatabase/libsql-client-go)

## Success Metrics

**Functional:**
- [ ] All existing tests pass with Turso backend
- [ ] Offline mode works (all read commands succeed without network)
- [ ] Multi-workstation sync verified (2+ machines see same data)
- [ ] Zero data loss in 100+ test scenarios

**Performance:**
- [ ] Embedded replica reads < 50ms (95th percentile)
- [ ] Cloud sync latency < 300ms (95th percentile)
- [ ] Offline→Online sync completes < 5 seconds for 100 tasks

**Reliability:**
- [ ] Sync success rate > 99.5% (30-day measurement)
- [ ] Connection retry logic handles transient failures
- [ ] No auth token leakage (security audit pass)

**Cost:**
- [ ] Free tier sufficient for 90% of test users (< 10M writes/month)
- [ ] Clear usage monitoring in `shark cloud status`

## Out of Scope

### Explicitly Excluded

1. **Multi-Master Replication**
   - **Why:** Turso handles this internally; we don't need to implement it
   - **Future:** Already supported by Turso, may enable if needed

2. **Custom Conflict Resolution UI**
   - **Why:** Last-write-wins sufficient for task management (not collaborative editing)
   - **Future:** Only if users report conflicts in practice
   - **Workaround:** Manual resolution via `shark task update`

3. **Alternative Cloud Providers (Supabase, AWS)**
   - **Why:** Turso is SQLite-compatible, others require major migration
   - **Future:** Abstraction layer makes this possible later
   - **Workaround:** Database export to SQL, manual migration

4. **Custom Sync Intervals**
   - **Why:** 30-second default is optimal for task management
   - **Future:** Add config option if users request it
   - **Workaround:** Use `shark cloud sync` for immediate manual sync

### Alternative Approaches Rejected

**Alternative 1: Supabase (PostgreSQL)**
- **Description:** Use Supabase as cloud backend
- **Why Rejected:**
  - Requires 3-4 weeks migration (query syntax, schema changes)
  - No offline support
  - No SQLite compatibility
- **Trade-off:** Better for multi-user collaboration, but not needed for v1

**Alternative 2: Self-Hosted libSQL**
- **Description:** Run libSQL server on user's infrastructure
- **Why Rejected:**
  - Defeats "zero-friction" goal (requires server management)
  - Users can already use network-mounted SQLite for this
- **Trade-off:** More control, but higher complexity

**Alternative 3: CRDTs for Conflict Resolution**
- **Description:** Use Conflict-free Replicated Data Types
- **Why Rejected:**
  - Overkill for task management (not collaborative editing)
  - Adds complexity with minimal benefit
- **Trade-off:** Handles conflicts better, but tasks rarely conflict

## Risks & Mitigation

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|-----------|
| Turso service outage | Low | High | Embedded replicas allow offline work; cloud syncs when restored |
| Free tier reduction | Medium | Medium | Document pricing clearly; abstraction layer allows backend swap |
| libSQL driver bugs | Medium | High | Pin driver version; contribute fixes upstream; fallback to local |
| Auth token leakage | Low | Critical | Enforce file permissions; never log tokens; env var recommended |
| Network latency issues | Medium | Low | Embedded replicas make reads instant; async background sync |

## Security Considerations

**Authentication:**
- Tokens stored in environment variables (preferred) or `~/.shark/turso-token` (chmod 600)
- Never commit tokens to version control (add to `.gitignore`)
- Never log or display tokens in output

**Data Protection:**
- All connections use HTTPS/WSS (encrypted in transit)
- Turso encrypts data at rest (AES-256)
- Local embedded replica inherits SQLite file permissions

**Audit Trail:**
- Turso provides audit logs (who accessed what, when)
- Local operations logged to `~/.shark/sync.log` (optional)

## Compliance & Regulations

- **GDPR:** Data stored in Turso EU regions if needed (configurable)
- **OWASP Top 10:** Addressed A02 (Cryptographic Failures) via secure token storage
- **Data Portability:** Export to local SQLite ensures no vendor lock-in

---

*Last Updated:* 2026-01-04
*Dependencies:* F01 (Database Abstraction Layer)
