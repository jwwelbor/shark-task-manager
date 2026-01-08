# Turso Prototype - Proof of Concept

**Created:** 2026-01-05
**Epic:** E13 (Cloud Database Support)
**Purpose:** Validate Turso (libSQL) before full implementation

## Objective

Build a quick proof-of-concept to validate:
1. libSQL Go driver works with Turso cloud database
2. Basic CRUD operations function correctly
3. Embedded replica support for offline mode
4. Performance is acceptable vs local SQLite
5. Error handling and edge cases

## Prerequisites

### Turso Account Setup
User has created a Turso account. Next steps:

1. **Install Turso CLI** (if not already installed):
   ```bash
   # macOS/Linux
   curl -sSfL https://get.tur.so/install.sh | bash

   # Verify installation
   turso --version
   ```

2. **Authenticate**:
   ```bash
   turso auth login
   ```

3. **Create test database**:
   ```bash
   turso db create shark-tasks-prototype
   ```

4. **Get connection details**:
   ```bash
   # Get database URL
   turso db show shark-tasks-prototype --url

   # Create authentication token
   turso db tokens create shark-tasks-prototype
   ```

5. **Save credentials** (for prototype use):
   ```bash
   export TURSO_DATABASE_URL="libsql://shark-tasks-prototype-[org].turso.io"
   export TURSO_AUTH_TOKEN="eyJhbGc..."
   ```

## Prototype Structure

```
prototype/
├── main.go              # Main test program
├── go.mod               # Module with libSQL dependency
├── test_results.txt     # Output and benchmarks
└── .env.example         # Example environment variables
```

## Test Plan

### Phase 1: Basic Connection (15 min)
- [ ] Add libSQL dependency to go.mod
- [ ] Create simple connection test
- [ ] Verify connection to Turso cloud database
- [ ] Handle authentication errors gracefully

### Phase 2: CRUD Operations (30 min)
- [ ] Create table (tasks schema subset)
- [ ] Insert test records
- [ ] Query records
- [ ] Update records
- [ ] Delete records
- [ ] Verify data integrity

### Phase 3: Embedded Replica (30 min)
- [ ] Enable embedded replica mode
- [ ] Test read operations (should be instant)
- [ ] Test write operations (background sync)
- [ ] Test offline mode (disconnect network)
- [ ] Verify data syncs when reconnected

### Phase 4: Performance Benchmarks (15 min)
- [ ] Measure read latency (embedded replica)
- [ ] Measure write latency (cloud sync)
- [ ] Compare to local SQLite performance
- [ ] Test with 1,000 record insert
- [ ] Test with complex queries

### Phase 5: Edge Cases (15 min)
- [ ] Invalid credentials
- [ ] Network timeout
- [ ] Concurrent writes
- [ ] Large data sets
- [ ] Schema migrations

## Success Criteria

**Must Pass:**
- Connection to Turso succeeds with valid credentials
- CRUD operations work without errors
- Embedded replica provides fast reads (<50ms)
- Offline mode works (embedded replica)
- Zero data loss in basic testing

**Performance Targets:**
- Embedded replica reads: <50ms (same as local SQLite)
- Cloud write sync: <300ms
- 1,000 record insert: <5 seconds

**If Fails:**
- Document failure reason
- Consider alternative backends (PostgreSQL, Supabase)
- Reassess E13 implementation strategy

## Findings

_To be filled during prototype implementation_

### Connection Test
- Status: ⏳ Pending
- Issues: None yet
- Notes: TBD

### CRUD Operations
- Status: ⏳ Pending
- Issues: None yet
- Notes: TBD

### Embedded Replica
- Status: ⏳ Pending
- Issues: None yet
- Notes: TBD

### Performance
- Status: ⏳ Pending
- Local SQLite baseline: TBD
- Turso embedded replica: TBD
- Turso cloud sync: TBD
- Notes: TBD

### Edge Cases
- Status: ⏳ Pending
- Issues: None yet
- Notes: TBD

## Decision

Based on prototype results:
- [ ] ✅ Proceed with full E13 implementation (Turso)
- [ ] ⚠️ Proceed with modifications (document changes needed)
- [ ] ❌ Do not proceed (choose alternative backend)

## Next Steps After Prototype

If successful:
1. Review findings with user
2. Decide on implementation approach (full vs MVP)
3. Start F01 (database abstraction layer)
4. Implement F02 (Turso integration) using prototype learnings

## References

- **Turso Docs:** https://docs.turso.tech/
- **libSQL Go Driver:** https://github.com/tursodatabase/libsql-client-go
- **Embedded Replicas:** https://docs.turso.tech/features/embedded-replicas
- **E13 Epic PRD:** `docs/plan/E13-add-cloud-db-support/epic.md`
- **F02 Feature PRD:** `docs/plan/E13-add-cloud-db-support/E13-F02-turso-integration/feature.md`
