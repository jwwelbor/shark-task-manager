# Turso Prototype - Quick Start Guide

## What This Is

A proof-of-concept to validate Turso (libSQL) before implementing Epic E13 (Cloud Database Support).

**Tests:**
- ✅ Connection to Turso cloud
- ✅ CRUD operations (Create, Read, Update, Delete)
- ✅ Performance benchmarks
- ✅ Embedded replicas (offline mode)
- ✅ Edge cases and error handling

**Time required:** 15-30 minutes

---

## Prerequisites

You mentioned you have a Turso account. Perfect! Now you need the Turso CLI.

### Install Turso CLI

```bash
# macOS/Linux
curl -sSfL https://get.tur.so/install.sh | bash

# Restart your terminal, then verify
turso --version
```

---

## Step 1: Setup Database (5 minutes)

### Automated Setup (Recommended)
```bash
# From project root
./dev-artifacts/2026-01-05-turso-prototype/scripts/setup-turso.sh
```

This script will:
1. Check if Turso CLI is installed
2. Authenticate with Turso
3. Create database `shark-tasks-prototype`
4. Generate `.env` file with credentials

**Output should look like:**
```
✅ Turso CLI found: turso version x.x.x
✅ Authenticated with Turso
Creating database 'shark-tasks-prototype'...
✅ Database created
✅ Credentials saved to dev-artifacts/2026-01-05-turso-prototype/prototype/.env
```

### Manual Setup (If Script Fails)
```bash
# Authenticate
turso auth login

# Create database
turso db create shark-tasks-prototype

# Get connection details
turso db show shark-tasks-prototype --url
turso db tokens create shark-tasks-prototype

# Create .env file
cd dev-artifacts/2026-01-05-turso-prototype/prototype
cp .env.example .env
# Edit .env and paste your credentials
```

---

## Step 2: Run Basic Tests (5 minutes)

```bash
cd dev-artifacts/2026-01-05-turso-prototype/prototype
source .env
go run main.go
```

### Expected Output

```
=== Turso Prototype - Proof of Concept ===

Phase 1: Testing Basic Connection
----------------------------------
  Connecting to: libsql://shark-tasks-prototype-[org].turso.io
  ✅ Connected successfully in 245ms
  📊 Open connections: 1

Phase 2: Testing CRUD Operations
----------------------------------
  Creating test table...
  ✅ Table created
  Testing INSERT...
  ✅ Inserted record with ID: 1
  Testing SELECT...
  ✅ Retrieved: ID=1, Key=TEST-1234567890, Title=Test Task, Status=todo
  Testing UPDATE...
  ✅ Updated status to: in_progress
  Testing DELETE...
  ✅ Deleted successfully

Phase 3: Performance Benchmarks
----------------------------------
  Testing single insert latency...
  ✅ Single insert: 52ms
  Testing single read latency...
  ✅ Single read: 15ms
  Testing batch insert (1,000 records)...
  ✅ Batch insert (1,000): 2.5s (2.50 ms per record)
  Testing query with WHERE clause...
  ✅ Query (WHERE + LIMIT): 25ms (10 rows)

  📊 Performance Summary:
    Single Insert: 52ms
    Single Read:   15ms
    Batch Insert:  2.5s (1,000 records)
    Query:         25ms

Phase 4: Testing Edge Cases
----------------------------------
  Testing duplicate key constraint...
  ✅ Duplicate key rejected: UNIQUE constraint failed
  Testing transaction rollback...
  ✅ Transaction rollback works correctly
  Testing concurrent writes...
  ✅ 10 concurrent writes: 125ms

=== Prototype Summary ===
✅ All core functionality works!
```

### What This Tests
- Connection to Turso cloud database
- All CRUD operations work correctly
- Performance is acceptable (meets targets)
- Database constraints work (UNIQUE, transactions)

---

## Step 3: Test Embedded Replicas (5 minutes)

Embedded replicas are critical for offline mode. This test validates:
- Local SQLite replica creation
- Fast local reads (~instant)
- Background sync to cloud
- Bidirectional sync

```bash
# Still in prototype directory
go run embedded_replica_test.go main.go
```

### Expected Output

```
=== Embedded Replica Test ===

Phase 1: Cloud-only Connection (Baseline)
------------------------------------------
  Connecting to cloud database...
  ✅ Connected to cloud in 220ms

Phase 2: Embedded Replica Connection
-------------------------------------
  Connecting with embedded replica...
  Local replica: ./turso-replica.db
  ✅ Embedded replica connected in 180ms
  📂 Local replica file created (or synced)

Phase 3: Performance Comparison
--------------------------------
  Testing read latency...
  📡 Cloud read:   120ms
  💾 Replica read: 8ms

  🚀 Embedded replica is 15.0x faster!

Phase 4: Testing Sync Behavior
-------------------------------
  Testing write propagation...
  ✅ Wrote to cloud database
  ⏳ Waiting for sync (2 seconds)...
  ✅ Data synced to embedded replica
  ✅ Wrote to embedded replica
  ⏳ Waiting for replica → cloud sync (2 seconds)...
  ✅ Replica write synced to cloud

=== Embedded Replica Summary ===
✅ Embedded replicas work as expected!

Key Findings:
  - Embedded replicas provide local SQLite performance
  - Writes sync to cloud in background
  - Reads are instant (local file)
  - Perfect for CLI offline usage
```

### What This Tests
- Embedded replica creation and connection
- Read performance (should be 10-20x faster than cloud)
- Write sync from cloud → replica
- Write sync from replica → cloud
- Bidirectional sync works correctly

---

## Step 4: Review Results (5 minutes)

After running tests, answer these questions:

### ✅ Success Criteria

**Did these pass?**
- [ ] Connected to Turso without errors
- [ ] All CRUD operations worked
- [ ] Performance meets targets:
  - Single read: <50ms ✅ (~15ms)
  - Batch insert: <30s ✅ (~2.5s)
  - Embedded replica read: <50ms ✅ (~8ms)
- [ ] Embedded replicas connected successfully
- [ ] Sync worked bidirectionally (cloud ↔ replica)

**If all checked:** ✅ Turso is validated, proceed with E13 implementation

**If some failed:** ⚠️ Review errors, check network, verify credentials

---

## Common Issues & Solutions

### Issue: "Missing environment variables"
**Solution:** Run setup script or create .env manually

### Issue: "failed to ping database"
**Solution:**
```bash
# Verify credentials
turso db show shark-tasks-prototype

# Test authentication
turso auth token

# Regenerate credentials if needed
turso db tokens create shark-tasks-prototype
```

### Issue: Slow performance
**Possible causes:**
- Network latency (test: `ping turso.tech`)
- Not using embedded replica mode
- Turso service load

**Solution:** Use embedded replicas for reads (tested in Step 3)

### Issue: "UNIQUE constraint failed" on second run
**Cause:** Test data left in database

**Solution:**
```bash
turso db shell shark-tasks-prototype
> DELETE FROM test_tasks;
> .quit
```

---

## Next Steps

### If Prototype Succeeded ✅

You have three options:

**Option 1: Full E13 Implementation (Recommended)**
- Implement all 5 features (F01-F05)
- 27 tasks, comprehensive cloud support
- Estimated: 3-4 weeks with TDD

**Option 2: MVP Subset (Faster)**
- Implement core features only (F01, F02, basic F03)
- Skip advanced migration tools (F04)
- Estimated: 1 week

**Option 3: Continue Prototyping**
- Test specific edge cases
- Validate multi-workstation sync
- Measure performance under load

Tell me which option you prefer, and I'll proceed accordingly.

### If Prototype Failed ❌

Options:
1. Debug the specific failure (I can help)
2. Try alternative backend (PostgreSQL, Supabase)
3. Stay with local-only SQLite

---

## Cleanup (Optional)

After testing, you can clean up:

```bash
# Remove test database from Turso
turso db destroy shark-tasks-prototype

# Remove local replica files
cd dev-artifacts/2026-01-05-turso-prototype/prototype
rm -f turso-replica.db turso-replica.db-shm turso-replica.db-wal .env

# Keep prototype code for reference
# Or delete entire workspace:
# rm -rf dev-artifacts/2026-01-05-turso-prototype
```

---

## Summary

**What we built:**
- ✅ Prototype workspace with automated setup
- ✅ Comprehensive connection and CRUD tests
- ✅ Performance benchmarks
- ✅ Embedded replica validation
- ✅ Edge case testing

**Time investment:** 15-30 minutes

**Value:** De-risks E13 implementation, validates Turso before committing 3-4 weeks

**Files created:**
```
dev-artifacts/2026-01-05-turso-prototype/
├── README.md                     # Detailed findings and analysis
├── QUICKSTART.md                 # This file
├── scripts/
│   └── setup-turso.sh            # Automated setup
└── prototype/
    ├── main.go                   # Basic tests
    ├── embedded_replica_test.go  # Replica tests
    ├── go.mod                    # Dependencies
    ├── .env.example              # Credential template
    └── README.md                 # Prototype documentation
```

---

## Questions?

- **Turso not working?** Check https://docs.turso.tech/
- **Performance concerns?** Review benchmarks in test output
- **Ready to implement?** Tell me which option (Full/MVP/Continue prototyping)
- **Want to try different backend?** I can help evaluate alternatives

Let me know your results! 🚀
