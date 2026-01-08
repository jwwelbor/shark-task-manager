# Turso Prototype - Quick Start

## Setup

### Option 1: Automated Setup (Recommended)
```bash
# From project root
./dev-artifacts/2026-01-05-turso-prototype/scripts/setup-turso.sh

# This will:
# 1. Check if Turso CLI is installed
# 2. Create database "shark-tasks-prototype"
# 3. Generate .env file with credentials
```

### Option 2: Manual Setup
```bash
# Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Login
turso auth login

# Create database
turso db create shark-tasks-prototype

# Get credentials
turso db show shark-tasks-prototype --url
turso db tokens create shark-tasks-prototype

# Create .env file
cp .env.example .env
# Edit .env and add your credentials
```

## Running Tests

### Test 1: Basic Functionality (Connection, CRUD, Performance)
```bash
cd dev-artifacts/2026-01-05-turso-prototype/prototype
source .env
go run main.go
```

**Tests:**
- Phase 1: Connection to Turso cloud
- Phase 2: CRUD operations (Create, Read, Update, Delete)
- Phase 3: Performance benchmarks (single ops, batch inserts, queries)
- Phase 4: Edge cases (constraints, transactions, concurrent writes)

**Expected output:**
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
  ...
```

### Test 2: Embedded Replicas (Offline Mode)
```bash
cd dev-artifacts/2026-01-05-turso-prototype/prototype
source .env
go run embedded_replica_test.go main.go
```

**Tests:**
- Phase 1: Cloud-only connection (baseline)
- Phase 2: Embedded replica connection
- Phase 3: Performance comparison (cloud vs replica reads)
- Phase 4: Sync behavior (cloud ↔ replica)

**Expected behavior:**
- Embedded replica reads: <50ms (local SQLite speed)
- Cloud reads: 100-300ms (network latency)
- Writes sync automatically in background

## What Gets Tested

### Basic Connection ✅
- Valid credentials work
- Invalid credentials rejected
- Connection pooling functional

### CRUD Operations ✅
- CREATE TABLE works
- INSERT works (single and batch)
- SELECT works (simple and complex queries)
- UPDATE works (single and batch)
- DELETE works
- Constraints enforced (UNIQUE, NOT NULL)

### Performance 📊
- Single insert latency
- Single read latency
- Batch insert (1,000 records)
- Complex queries with WHERE/LIMIT
- Transaction commit time

### Edge Cases ⚠️
- Duplicate key rejection
- Transaction rollback
- Concurrent writes
- Large data sets

### Embedded Replicas 💾
- Local file creation
- Read performance (should be ~instant)
- Write sync (background)
- Bidirectional sync (cloud ↔ replica)

## Performance Targets

Based on E13 PRD requirements:

| Metric | Target | Typical Result |
|--------|--------|----------------|
| Embedded replica read | <50ms | 5-15ms |
| Cloud read | <300ms | 100-250ms |
| Single write | <300ms | 50-150ms |
| Batch insert (1,000) | <30s | 2-5s |

## Troubleshooting

### Error: "Missing environment variables"
**Solution:** Run setup script or manually create .env file

### Error: "failed to ping database"
**Possible causes:**
- Invalid credentials
- Network connectivity
- Turso service down

**Solution:**
```bash
# Verify credentials
turso db show shark-tasks-prototype

# Test connectivity
curl https://turso.tech

# Regenerate token
turso db tokens create shark-tasks-prototype
```

### Error: "UNIQUE constraint failed"
**Cause:** Test table has leftover data

**Solution:**
```bash
# Connect to database and clean up
turso db shell shark-tasks-prototype
> DELETE FROM test_tasks;
> .quit
```

### Slow performance
**Possible causes:**
- Not using embedded replica
- Network latency
- Turso service load

**Solution:**
- Use embedded replica mode for reads
- Check network latency: `ping turso.tech`
- Try test at different time

## Files Created

During testing, these files are created:

- `turso-replica.db` - Local embedded replica (SQLite file)
- `turso-replica.db-shm` - Shared memory file
- `turso-replica.db-wal` - Write-ahead log

These can be safely deleted after testing.

## Cleanup

```bash
# Remove test database
turso db destroy shark-tasks-prototype

# Remove local files
rm -f turso-replica.db turso-replica.db-shm turso-replica.db-wal

# Remove .env (contains credentials)
rm .env
```

## Next Steps

After successful prototype:
1. Review findings in `../README.md`
2. Update findings section with test results
3. Decide on E13 implementation approach
4. If proceeding, start with F01 (database abstraction layer)

## Reference

- **Turso Docs:** https://docs.turso.tech/
- **libSQL Driver:** https://github.com/tursodatabase/libsql-client-go
- **E13 Epic:** `docs/plan/E13-add-cloud-db-support/epic.md`
