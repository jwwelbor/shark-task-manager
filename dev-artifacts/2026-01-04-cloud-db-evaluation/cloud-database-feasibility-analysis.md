# Cloud Database Support Feasibility Analysis

**Idea Reference:** I-2026-01-03-02 - "add cloud db support"
**Date:** 2026-01-04
**Prepared By:** ProductManager Agent

## Executive Summary

This analysis evaluates the feasibility and customer experience of adding cloud database support to Shark Task Manager to enable multiple workstations to share the same task database. The analysis covers cloud-native SQLite options, traditional cloud databases (PostgreSQL/MySQL), and hybrid approaches.

**Key Findings:**
- **SQLite-first options (Turso, LiteFS, SQLite Cloud)** provide the easiest migration path with minimal code changes
- **Turso libSQL** offers the best combination of free tier, performance, and SQLite compatibility
- **Migration complexity** varies significantly: SQLite cloud options require minimal changes, while PostgreSQL/MySQL require substantial refactoring
- **Customer experience** is critical: setup complexity must be minimal for CLI tool users

---

## Current State Analysis

### Shark's Database Architecture

**Current Setup:**
- **Database Engine:** SQLite 3 with local file (`shark-tasks.db`)
- **Configuration:** WAL mode for concurrency, 64MB cache, memory-mapped I/O
- **File Size:** ~704KB (current production database)
- **Schema:** 10+ tables (epics, features, tasks, task_history, task_notes, etc.)
- **Features Used:**
  - Foreign key constraints with CASCADE DELETE
  - Auto-update triggers for timestamps
  - 10+ indexes for query performance
  - PRAGMA settings for optimal local performance
  - FTS5 virtual tables for full-text search (optional)

**Key SQLite-Specific Features in Use:**
```sql
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;
PRAGMA temp_store = MEMORY;
PRAGMA mmap_size = 30000000000;
```

**Current Concurrency Model:**
- Single-writer via WAL mode
- Multiple readers don't block
- 5-second busy timeout for lock contention
- Auto-detect project root (can run from any subdirectory)

### User Need

**Primary Use Case:** Developer using multiple workstations (desktop, laptop, etc.) wants to:
- Access the same task database from any machine
- See real-time updates from other machines
- Avoid manual database file synchronization (e.g., Git, Dropbox)

**Secondary Needs:**
- Collaboration: Multiple team members sharing tasks
- Backup/Recovery: Automatic cloud backups
- Access Control: Optional user authentication (future)

---

## Cloud Database Options Evaluated

### Category 1: SQLite Cloud-Native Options

These options maintain SQLite compatibility while adding cloud sync/distribution.

#### 1. Turso (libSQL) - **RECOMMENDED**

**Overview:**
- Distributed database built on libSQL (SQLite fork)
- Edge replication with multi-region support
- Native similarity search for AI apps (bonus feature)

**Pricing (2026):**
- **Free Tier:** 500M rows read/month, 10M rows write/month, 5GB storage, 500 active databases
- **Developer Plan:** $4.99/month - 2.5B rows read, 1B rows write, 9GB storage
- No credit card required for free tier

**Technical Details:**
- **Concurrency:** Rust-based SQLite with improved concurrent performance
- **API:** HTTP API + native SQLite protocol
- **Latency:** Sub-100ms for edge locations
- **Replication:** Multi-region with eventual consistency
- **Offline Support:** Can sync on-demand

**Migration Complexity:** ⭐⭐⭐⭐⭐ (Low)
- Minimal code changes required
- SQLite syntax fully compatible
- Add libSQL Go driver
- Update connection string

**Customer Experience:** ⭐⭐⭐⭐⭐ (Excellent)
- **Setup:** 5 minutes (create account, get connection URL)
- **Configuration:** Single environment variable or config flag
- **Authentication:** Simple API token
- **Offline:** Works offline, syncs when online

**Pros:**
- 100% SQLite compatible (minimal migration)
- Generous free tier (perfect for individual developers)
- Native Go support via libSQL driver
- Built-in encryption at rest
- Dr. Richard Hipp (SQLite creator) involved

**Cons:**
- Relatively new service (may lack enterprise maturity)
- Requires internet connection for real-time sync
- Free tier has monthly active database limits

---

#### 2. LiteFS (Fly.io)

**Overview:**
- FUSE-based distributed SQLite replication
- Open-source, not locked to Fly.io
- Primary/replica architecture

**Pricing (2026):**
- **LiteFS:** Free (open-source)
- **LiteFS Cloud (Backups):** $5/month for up to 10GB
- **Fly.io Infrastructure:**
  - Volumes: $0.15/GB/month
  - Snapshots: $0.08/GB/month (first 10GB free)
  - Compute: Variable based on machine type

**Technical Details:**
- **Concurrency:** Single primary writer, multiple read replicas
- **Replication:** Uses Raft consensus for strong consistency
- **Write Model:** All writes go to primary, then propagate to replicas
- **Latency:** Dependent on primary node location

**Migration Complexity:** ⭐⭐⭐⭐ (Low-Medium)
- No code changes (transparent FUSE filesystem)
- Requires Fly.io infrastructure setup
- Need to handle primary election
- Application must route writes to primary

**Customer Experience:** ⭐⭐⭐ (Good)
- **Setup:** 15-30 minutes (Fly.io account, deploy config)
- **Configuration:** Requires understanding of primary/replica model
- **Authentication:** Fly.io auth
- **Complexity:** Must understand distributed systems concepts

**Pros:**
- True SQLite (no fork, 100% compatible)
- Open-source (can self-host)
- Strong consistency via Raft
- Works with existing SQLite files

**Cons:**
- Requires Fly.io infrastructure (more complex setup)
- Single-writer limitation (SQLite constraint)
- Must route writes to primary node
- Infrastructure costs beyond LiteFS itself
- Steeper learning curve for non-DevOps users

---

#### 3. SQLite Cloud

**Overview:**
- Distributed, high-concurrency cloud database
- Built on 100% open-source SQLite
- Dr. Richard Hipp involved

**Pricing (2026):**
- **Free Tier:** Available ("Start Free" plan)
- Paid tiers available (specific pricing not disclosed in search)

**Technical Details:**
- **Concurrency:** Enhanced multi-user write support
- **Consistency:** Raft consensus for fault-tolerance
- **Features:** Pub/Sub for real-time updates, offline sync
- **Write Performance:** Only waits for commit message write (fast)

**Migration Complexity:** ⭐⭐⭐⭐ (Low-Medium)
- SQLite compatible
- Need to integrate SQLite Cloud SDK
- Handle real-time pub/sub (optional)

**Customer Experience:** ⭐⭐⭐⭐ (Good)
- **Setup:** 10 minutes (account, connection)
- **Configuration:** Connection string update
- **Features:** Real-time updates via Pub/Sub

**Pros:**
- SQLite compatible
- Multi-user concurrent writes (improved over vanilla SQLite)
- Real-time Pub/Sub capabilities
- Offline synchronization

**Cons:**
- Less transparent pricing (compared to Turso)
- Smaller community/ecosystem (newer service)
- Limited documentation availability

---

### Category 2: Traditional Cloud Databases (PostgreSQL/MySQL)

These require migrating away from SQLite to a different database engine.

#### 4. Supabase (PostgreSQL)

**Overview:**
- PostgreSQL-based Firebase alternative
- Built-in auth, real-time, storage
- Generous free tier

**Pricing (2026):**
- **Free Tier:** 500MB database, 50K MAU, 1GB file storage
  - 2 projects, limited backups, auto-pause on inactivity
- **Pro Plan:** $25/month - 8GB database, 100K MAU, daily backups
- **Team Plan:** $599/month - SOC2, SSO, custom domains

**Technical Details:**
- **Engine:** PostgreSQL 15+
- **Concurrency:** 10,000+ concurrent connections (paid tiers)
- **API:** Auto-generated REST API, GraphQL, real-time subscriptions
- **Latency:** 20-50ms for indexed queries
- **Connection Pooling:** Built-in via pgBouncer

**Migration Complexity:** ⭐⭐ (High)
- **Schema Conversion:** SQLite → PostgreSQL
  - AUTOINCREMENT → SERIAL
  - Type affinity → strict typing
  - PRAGMA statements → PostgreSQL config
- **Query Rewrite:**
  - Prepared statements: `?` → `$1, $2, $3`
  - Function differences (e.g., `ILIKE`, `array_agg`)
- **Code Changes:**
  - Replace `mattn/go-sqlite3` with `lib/pq` or `pgx`
  - Update repository layer for PostgreSQL-specific features
  - Rewrite migrations
  - Test all queries

**Customer Experience:** ⭐⭐⭐ (Fair)
- **Setup:** 10 minutes (account, database creation, connection)
- **Configuration:** Database URL, credentials
- **Learning Curve:** Must understand PostgreSQL vs SQLite differences
- **Complexity:** Higher for users unfamiliar with PostgreSQL

**Pros:**
- Free tier sufficient for individual developers
- Excellent concurrency (true multi-writer)
- Auto-generated APIs (bonus features)
- Real-time subscriptions
- Built-in authentication/authorization

**Cons:**
- **Major code refactoring required**
- Breaking change for existing Shark users
- Increased complexity vs SQLite
- Auto-pause on free tier (can cause cold starts)
- No local offline mode

---

#### 5. PlanetScale (MySQL)

**Overview:**
- MySQL-compatible serverless database
- Built on Vitess (YouTube's database tech)

**Pricing (2026):**
- **Free Tier:** DISCONTINUED (removed March 2024)
- **Minimum Plan:** $39/month
- **Scaler Plan:** Discontinued (moved to predictable pricing)

**Migration Complexity:** ⭐⭐ (High)
- Same challenges as PostgreSQL migration
- MySQL-specific syntax differences

**Customer Experience:** ⭐ (Poor for small projects)
- **Cost:** No free tier makes it unsuitable for individual developers
- **Setup:** 15 minutes (account, database, connection)

**Recommendation:** **NOT SUITABLE** - No free tier, high minimum cost

---

#### 6. AWS RDS / Aurora Serverless

**Overview:**
- Managed PostgreSQL/MySQL (RDS) or Aurora (proprietary)
- Serverless v2 for variable workloads

**Pricing (2026):**
- **Aurora Serverless v2 Minimum:**
  - $43-45/month for 0.5 ACU minimum (us-east-1)
  - Plus storage: $0.10/GB-month
  - Plus I/O: $0.20 per million requests
- **Standard RDS:** Starts ~$15/month (db.t4g.micro)
- **Free Tier:** 750 hours/month for 12 months (standard RDS only, not Aurora)

**Migration Complexity:** ⭐⭐ (High)
- Same PostgreSQL/MySQL migration challenges
- AWS-specific setup (VPC, security groups, IAM)

**Customer Experience:** ⭐⭐ (Poor for small projects)
- **Setup:** 30-45 minutes (AWS account, VPC, RDS setup, security)
- **Cost:** High minimum ($15-45/month)
- **Complexity:** AWS infrastructure knowledge required

**Recommendation:** **NOT SUITABLE** - Too expensive and complex for small databases

---

#### 7. Azure Database / Google Cloud SQL

**Overview:**
- Managed PostgreSQL/MySQL on Azure/GCP
- Similar to AWS RDS

**Pricing (2026):**

**Azure:**
- **Free Tier:** 100K vCore seconds/month, 32GB max, serverless (12 months)
- **Serverless:** Auto-scales, pay per second
- **MySQL/PostgreSQL Flexible Server:** ~$15-30/month minimum

**Google Cloud SQL:**
- **Free Tier:** $300 credits for 90 days (new customers)
- **db-f1-micro:** ~$10-15/month (shared CPU, 614MB RAM)
- **Pricing:** $30.11/vCPU, $5.11/GB memory/month

**Migration Complexity:** ⭐⭐ (High)
- Same PostgreSQL/MySQL challenges

**Customer Experience:** ⭐⭐ (Poor for small projects)
- **Setup:** 20-30 minutes (account, database, firewall)
- **Cost:** Minimum $10-15/month
- **Complexity:** Cloud provider knowledge required

**Recommendation:** **NOT SUITABLE** - Cost and complexity too high for task management CLI

---

## Comparison Summary Table

| Option | Monthly Cost | Migration Effort | Concurrency | Setup Time | Offline Support | Recommendation |
|--------|-------------|------------------|-------------|------------|-----------------|----------------|
| **Turso (libSQL)** | Free - $5 | ⭐⭐⭐⭐⭐ Low | High (Rust-based) | 5 min | Yes | ✅ **BEST** |
| **LiteFS** | $5-20 | ⭐⭐⭐⭐ Low | Medium (single writer) | 30 min | Yes | ✅ Good |
| **SQLite Cloud** | Free - ? | ⭐⭐⭐⭐ Low-Med | High | 10 min | Yes | ✅ Good |
| **Supabase** | Free - $25 | ⭐⭐ High | Very High | 10 min | No | ⚠️ If switching DBs |
| **PlanetScale** | $39+ | ⭐⭐ High | Very High | 15 min | No | ❌ Too expensive |
| **AWS Aurora** | $45+ | ⭐⭐ High | Very High | 45 min | No | ❌ Too expensive |
| **Azure/GCP SQL** | $15+ | ⭐⭐ High | Very High | 30 min | No | ❌ Too expensive |

**Legend:**
- ⭐⭐⭐⭐⭐ = Minimal changes (days)
- ⭐⭐⭐⭐ = Some changes (1-2 weeks)
- ⭐⭐ = Major refactor (3-4 weeks)

---

## Customer Experience Assessment

### Setup Flow Comparison

#### **Turso (Recommended)**
```bash
# Step 1: Install Turso CLI
curl -sSfL https://get.tur.so/install.sh | bash

# Step 2: Create database
turso db create shark-tasks

# Step 3: Get connection URL
turso db show shark-tasks --url

# Step 4: Configure Shark
export SHARK_DB_URL="libsql://shark-tasks-username.turso.io"
export SHARK_DB_TOKEN="eyJ..."

# Step 5: Run shark (existing commands work)
shark task list
```

**Time to First Query:** ~5 minutes
**Configuration Complexity:** Low (2 environment variables)
**User Knowledge Required:** Minimal (copy-paste commands)

---

#### **LiteFS**
```bash
# Step 1: Install Fly CLI
brew install flyctl

# Step 2: Create Fly.io app
fly launch --name shark-tasks

# Step 3: Configure litefs.yml
# (requires understanding of primary/replica concepts)

# Step 4: Deploy
fly deploy

# Step 5: Configure Shark to use Fly instance
export SHARK_DB_URL="https://shark-tasks.fly.dev/db"
```

**Time to First Query:** ~30 minutes
**Configuration Complexity:** Medium (YAML config, primary election)
**User Knowledge Required:** DevOps familiarity, distributed systems concepts

---

#### **Supabase (PostgreSQL)**
```bash
# Step 1: Create Supabase project (web UI)
# https://app.supabase.com

# Step 2: Migrate SQLite schema to PostgreSQL
# (manual schema conversion, data migration)

# Step 3: Rewrite Shark queries for PostgreSQL syntax
# (significant code changes)

# Step 4: Configure connection
export DATABASE_URL="postgresql://postgres:password@db.xxx.supabase.co:5432/postgres"

# Step 5: Rebuild and test Shark
```

**Time to First Query:** ~3-4 weeks (includes code changes)
**Configuration Complexity:** High (schema migration, query rewrites)
**User Knowledge Required:** PostgreSQL expertise, database migration experience

---

### User Personas & Fit

#### **Persona 1: Solo Developer (Primary User)**
- Uses Shark on 2-3 machines (work desktop, home laptop, maybe a VM)
- Budget: Free or <$5/month
- Technical Level: Comfortable with CLI, not necessarily DevOps expert
- Offline Needs: Occasional (travel, coffee shops)

**Best Fit:** Turso (free tier, easy setup, offline support)
**Second Choice:** SQLite Cloud (if Turso unavailable)

---

#### **Persona 2: Small Team (2-5 developers)**
- Shared task database for team coordination
- Budget: $5-25/month
- Technical Level: Mixed (some DevOps, some pure developers)
- Concurrent Access: 2-5 users simultaneously

**Best Fit:** Turso ($5/month developer plan if free tier insufficient)
**Second Choice:** Supabase (if team wants PostgreSQL features like auth)

---

#### **Persona 3: Enterprise Team**
- Large org with 10+ users
- Budget: $50-500/month
- Technical Level: Dedicated DevOps team
- Requirements: SSO, audit logs, compliance (SOC2, HIPAA)

**Best Fit:** Supabase Team plan ($599/month) OR custom enterprise solution
**Not Recommended:** SQLite-based solutions (may not meet enterprise compliance)

---

## Migration Strategy

### Phase 1: Add Cloud Backend Support (Turso) - **RECOMMENDED FIRST PHASE**

**Goal:** Enable optional cloud database without breaking local SQLite usage.

**Implementation Plan:**

1. **Database Abstraction Layer (1-2 days)**
   ```go
   // internal/db/connection.go
   type DatabaseConfig struct {
       Type     string // "sqlite" or "libsql"
       Path     string // file path for sqlite, URL for libsql
       Token    string // auth token for libsql
   }

   func InitDB(config DatabaseConfig) (*sql.DB, error) {
       switch config.Type {
       case "libsql":
           return initLibSQL(config)
       case "sqlite":
           return initSQLite(config)
       default:
           return initSQLite(config) // default to local
       }
   }
   ```

2. **Add libSQL Driver (1 day)**
   ```bash
   go get github.com/tursodatabase/libsql-client-go
   ```

3. **Configuration Enhancement (1 day)**
   ```json
   // .sharkconfig.json
   {
       "database": {
           "type": "libsql", // or "sqlite"
           "url": "libsql://shark-tasks-username.turso.io",
           "token": "${SHARK_DB_TOKEN}" // read from env var
       }
   }
   ```

4. **CLI Commands for Cloud Setup (2 days)**
   ```bash
   # New commands
   shark cloud login        # Authenticate with Turso
   shark cloud init         # Create cloud database
   shark cloud sync         # Push local data to cloud
   shark cloud status       # Check cloud connection
   shark cloud switch local # Switch back to local mode
   shark cloud switch cloud # Switch to cloud mode
   ```

5. **Migration Utility (2 days)**
   ```bash
   # Export local database to cloud
   shark export-to-cloud --source=./shark-tasks.db --target=libsql://...

   # Import cloud database to local
   shark import-from-cloud --source=libsql://... --target=./shark-tasks.db
   ```

6. **Documentation (1 day)**
   - User guide for cloud setup
   - Troubleshooting guide
   - FAQ (cost, privacy, performance)

**Total Effort:** ~1-2 weeks
**Risk Level:** Low (backward compatible, opt-in feature)

---

### Phase 2: Enhanced Multi-User Features (Optional)

Only implement if users request collaborative features:

1. **Real-time Updates** (via Turso's change notifications)
   - Notify CLI when remote tasks change
   - Auto-refresh local cache

2. **User Authentication**
   - Associate tasks with user IDs
   - Track who created/completed tasks

3. **Conflict Resolution**
   - Handle concurrent edits to same task
   - Merge strategies for description updates

**Effort:** 2-3 weeks
**When:** Only if Phase 1 adoption is high

---

### Phase 3: Alternative Backend Support (PostgreSQL) - **FUTURE**

Only if enterprise customers request it:

1. **Schema Translation Layer**
   - Map SQLite schema to PostgreSQL
   - Handle type differences

2. **Query Abstraction**
   - Repository pattern with interface
   - Separate implementations for SQLite vs PostgreSQL

3. **Feature Parity**
   - Ensure all features work on both backends

**Effort:** 4-6 weeks
**When:** Customer demand justifies investment

---

## Recommendation

### Primary Recommendation: **Implement Turso (libSQL) Support**

**Rationale:**
1. **Minimal Migration Risk:** SQLite-compatible, requires minimal code changes
2. **Excellent Free Tier:** 500M rows read, 10M rows write, 5GB storage (perfect for individual users)
3. **Easy Setup:** 5-minute setup vs 30+ minutes for alternatives
4. **Offline Support:** Can work offline, sync when online (critical for CLI tool)
5. **Future-Proof:** Can add PostgreSQL support later if needed
6. **Cost-Effective:** Free for most users, $5/month for power users

**Implementation Approach:**
- **Opt-in Feature:** Keep local SQLite as default, cloud as optional
- **Backward Compatible:** Existing users unaffected
- **Progressive Enhancement:** Add cloud features incrementally

---

### Alternative Recommendation: **LiteFS** (If Fly.io Infrastructure Acceptable)

**When to Choose LiteFS:**
- User already uses Fly.io
- User wants 100% standard SQLite (no fork)
- User willing to manage infrastructure
- User needs strong consistency guarantees (Raft)

**Tradeoffs:**
- Higher setup complexity
- Infrastructure costs beyond database
- Single-writer limitation

---

### **NOT Recommended:**

1. **PostgreSQL/MySQL Options** (Supabase, AWS, Azure, GCP)
   - Too much migration effort (3-4 weeks minimum)
   - Higher cost ($15-25/month minimum)
   - Breaks SQLite compatibility
   - No offline support
   - **Only consider if:** Enterprise customer with specific compliance needs

2. **PlanetScale**
   - No free tier ($39/month minimum)
   - Not cost-effective for task management use case

---

## Risk Assessment

### Turso Implementation Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Turso service shutdown | Low | High | Abstract DB layer, enable easy backend swap |
| Free tier changes | Medium | Medium | Document paid plan costs, make opt-in |
| Performance degradation | Low | Medium | Local caching, fallback to local DB |
| User adoption low | Medium | Low | Make optional, market to multi-machine users |
| Authentication issues | Low | Medium | Clear error messages, troubleshooting docs |
| Data loss/corruption | Very Low | Very High | Automatic backups, export utility |

### Mitigation Strategies

1. **Service Risk:** Abstract database interface, allow swapping backends
2. **Cost Risk:** Clear pricing documentation, usage monitoring
3. **Performance Risk:** Benchmark before release, implement local caching
4. **Adoption Risk:** Target specific user personas, gather feedback early
5. **Data Risk:** Built-in backup/restore, export to local SQLite

---

## Next Steps

### If Approved for Development:

1. **Create Epic:** "E09 - Cloud Database Support"
2. **Features:**
   - E09-F01: Database abstraction layer
   - E09-F02: Turso libSQL integration
   - E09-F03: Cloud CLI commands (login, init, sync)
   - E09-F04: Documentation and user guides
   - E09-F05: Migration utilities (local ↔ cloud)

3. **Prototype:** Build POC with Turso (1 week)
4. **User Testing:** Beta test with 5-10 early adopters
5. **Launch:** Phased rollout with opt-in

### If NOT Approved:

Document alternative approaches:
- **Manual Sync:** Users sync `shark-tasks.db` via Git/Dropbox
- **Network Mount:** Users mount cloud storage as local filesystem
- **SSH Tunnel:** Remote access to single database instance

---

## Appendix: Technical Research Sources

### Turso
- [Turso Database Pricing](https://turso.tech/pricing)
- [Turso - Databases Everywhere](https://turso.tech/)
- [Turso Cloud Debuts the New Developer Plan](https://turso.tech/blog/turso-cloud-debuts-the-new-developer-plan)
- [Database Freedom Day - Unlimited Databases Are Here](https://turso.tech/blog/unlimited-databases-are-here)

### LiteFS
- [LiteFS - Distributed SQLite · Fly Docs](https://fly.io/docs/litefs/)
- [Introducing LiteFS · The Fly Blog](https://fly.io/blog/introducing-litefs/)
- [LiteFS Cloud: Distributed SQLite with Managed Backups · The Fly Blog](https://fly.io/blog/litefs-cloud/)
- [Pricing · Fly](https://fly.io/pricing/)

### SQLite Cloud
- [SQLite AI Pricing](https://www.sqlite.ai/pricing)
- [How (and why) we brought SQLite to the Cloud](https://blog.sqlite.ai/how-and-why-we-brought-sqlite-to-the-cloud)

### Supabase
- [Pricing & Fees | Supabase](https://supabase.com/pricing)
- [Supabase Review 2026: We Tested the Firebase Alternative](https://hackceleration.com/supabase-review/)
- [The Complete Guide to Supabase Pricing Models and Cost Optimization](https://flexprice.io/blog/supabase-pricing-breakdown)

### AWS
- [Amazon Aurora Pricing](https://aws.amazon.com/rds/aurora/pricing/)
- [Managed Relational Database - Amazon RDS Pricing](https://aws.amazon.com/rds/pricing/)
- [RDS vs Aurora: A Detailed Pricing Comparison | Vantage](https://www.vantage.sh/blog/aws-rds-vs-aurora-pricing-in-depth)

### Azure
- [Pricing - Azure SQL Database Single Database | Microsoft Azure](https://azure.microsoft.com/en-us/pricing/details/azure-sql-database/single/)
- [Deploy for Free - Azure SQL Database | Microsoft Learn](https://learn.microsoft.com/en-us/azure/azure-sql/database/free-offer?view=azuresql)
- [Pricing - Azure Database for PostgreSQL Flexible Server](https://azure.microsoft.com/en-us/pricing/details/postgresql/flexible-server/)

### Google Cloud
- [Cloud SQL pricing | Google Cloud](https://cloud.google.com/sql/docs/postgres/pricing)
- [Google Cloud SQL Pricing - Cost Guide & Comparison](https://www.pump.co/blog/google-cloud-sql-pricing)
- [Understanding Google Cloud SQL Pricing](https://www.bytebase.com/blog/understanding-google-cloud-sql-pricing/)

### PlanetScale
- [Pricing and plans — PlanetScale](https://planetscale.com/pricing)
- [11 Planetscale alternatives with free tiers - LogRocket Blog](https://blog.logrocket.com/11-planetscale-alternatives-free-tiers/)
- [No More Free Tier on PlanetScale, Here Are Free Alternatives](https://www.codu.co/articles/no-more-free-tier-on-planetscale-here-are-free-alternatives-q4wzqcu9)

### Migration Challenges
- [Database Migration: SQLite to PostgreSQL](https://www.bytebase.com/blog/database-migration-sqlite-to-postgresql/)
- [Database Migration: Transitioning from SQLite to PostgreSQL on AWS](https://medium.com/@kagegreo/database-migration-transitioning-from-sqlite-to-postgresql-on-aws-e84f0b79430e)
- [How to Perform Database Migrations using Go Migrate](https://www.bomberbot.com/golang/how-to-perform-database-migrations-using-go-migrate-a-comprehensive-guide/)

---

## Conclusion

Cloud database support is **feasible and recommended** for Shark Task Manager, with **Turso (libSQL)** as the optimal choice. The implementation can be done incrementally as an opt-in feature, preserving backward compatibility while enabling multi-workstation workflows.

**Estimated Development Time:** 1-2 weeks for Phase 1 (Turso integration)
**Estimated Cost to Users:** Free for most users, $5/month for power users
**Risk Level:** Low (backward compatible, well-defined scope)

The primary value proposition is **eliminating manual database synchronization** for developers using multiple machines, which directly addresses the user need stated in idea I-2026-01-03-02.
