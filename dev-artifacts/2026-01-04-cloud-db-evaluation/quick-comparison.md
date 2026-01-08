# Cloud Database Quick Comparison

**For:** Shark Task Manager cloud database support
**Date:** 2026-01-04

## The Question
"I use Shark on multiple workstations (desktop, laptop). How can I share the same task database without manually copying files?"

## The Answer: Add Cloud Database Support

---

## Top 3 Realistic Options

### 🥇 Turso (libSQL) - **RECOMMENDED**

**What it is:** SQLite in the cloud with edge replication
**Cost:** FREE (up to 500M rows read, 10M rows write, 5GB storage)
**Setup time:** 5 minutes
**Code changes:** Minimal (add driver, update connection)

**Pros:**
- ✅ 100% SQLite compatible (no query changes)
- ✅ Generous free tier (perfect for individuals)
- ✅ Works offline, syncs when online
- ✅ 5-minute setup (easiest option)
- ✅ Native Go support

**Cons:**
- ⚠️ Relatively new service (less battle-tested)
- ⚠️ Free tier has 500 active database limit

**Best for:** Individual developers, small teams (2-5 people), anyone who wants simplest setup

---

### 🥈 LiteFS (Fly.io)

**What it is:** Distributed SQLite replication (open-source)
**Cost:** $5-20/month (open-source + Fly.io infrastructure)
**Setup time:** 30 minutes
**Code changes:** Minimal (transparent FUSE layer)

**Pros:**
- ✅ 100% standard SQLite (no fork)
- ✅ Open-source (can self-host)
- ✅ Strong consistency (Raft consensus)
- ✅ Works offline

**Cons:**
- ⚠️ Requires Fly.io infrastructure setup
- ⚠️ Must understand primary/replica model
- ⚠️ Single-writer limitation
- ⚠️ Infrastructure costs beyond LiteFS

**Best for:** Users already on Fly.io, users who want pure SQLite, users comfortable with DevOps

---

### 🥉 Supabase (PostgreSQL)

**What it is:** PostgreSQL-based Firebase alternative
**Cost:** FREE (500MB database, 50K MAU) or $25/month (8GB, 100K MAU)
**Setup time:** 10 minutes (but 3-4 weeks for code migration)
**Code changes:** MAJOR (complete schema + query rewrite)

**Pros:**
- ✅ Free tier available
- ✅ Excellent concurrency (true multi-writer)
- ✅ Auto-generated REST APIs
- ✅ Built-in auth/real-time features

**Cons:**
- ❌ NOT SQLite compatible (requires full migration)
- ❌ 3-4 weeks of code refactoring
- ❌ No offline support
- ❌ Breaking change for existing users
- ❌ Auto-pause on free tier (cold starts)

**Best for:** Teams willing to migrate away from SQLite, teams needing enterprise PostgreSQL features

---

## Options NOT Recommended

| Option | Why Not? |
|--------|----------|
| **PlanetScale** | No free tier ($39/month minimum) - too expensive for task management |
| **AWS Aurora** | $45+/month minimum - overkill for small databases |
| **Azure SQL** | $15+/month minimum - unnecessary complexity |
| **Google Cloud SQL** | $10+/month minimum - setup complexity not worth it |

---

## Side-by-Side Comparison

|  | Turso | LiteFS | Supabase | AWS/Azure/GCP |
|--|-------|--------|----------|---------------|
| **Monthly Cost** | Free - $5 | $5 - $20 | Free - $25 | $15 - $45 |
| **Free Tier** | ✅ Generous | ⚠️ OSS only | ✅ Good | ❌ Limited |
| **Setup Time** | ⚙️ 5 min | ⚙️⚙️ 30 min | ⚙️ 10 min* | ⚙️⚙️⚙️ 45 min |
| **Code Changes** | ✏️ Minimal | ✏️ Minimal | ✏️✏️✏️ Major | ✏️✏️✏️ Major |
| **SQLite Compatible** | ✅ Yes | ✅ Yes | ❌ No (PostgreSQL) | ❌ No |
| **Offline Support** | ✅ Yes | ✅ Yes | ❌ No | ❌ No |
| **Concurrent Writers** | ✅ High | ⚠️ Single | ✅ Very High | ✅ Very High |
| **Dev Time** | 1-2 weeks | 1-2 weeks | 3-4 weeks | 3-4 weeks |

*Setup is 10 min, but code migration takes 3-4 weeks

---

## Customer Experience Comparison

### Turso Setup Flow
```bash
# Total time: 5 minutes

# 1. Create account + database (web UI)
# Visit https://turso.tech

# 2. Get connection details
turso db show shark-tasks --url

# 3. Configure Shark
export SHARK_DB_URL="libsql://shark-tasks-yourname.turso.io"
export SHARK_DB_TOKEN="eyJ..."

# 4. Use Shark normally
shark task list  # Works immediately!
```

### LiteFS Setup Flow
```bash
# Total time: 30 minutes

# 1. Install Fly CLI
brew install flyctl

# 2. Create Fly app
fly launch --name shark-tasks

# 3. Configure litefs.yml
# (requires understanding primary/replica concepts)

# 4. Deploy
fly deploy

# 5. Configure Shark
export SHARK_DB_URL="https://shark-tasks.fly.dev/db"
```

### Supabase Setup Flow
```bash
# Total time: 3-4 weeks (includes migration)

# 1. Create project (5 minutes)
# Visit https://app.supabase.com

# 2. Migrate schema (1-2 days)
# Convert SQLite schema to PostgreSQL
# Handle type differences, constraints, indexes

# 3. Rewrite queries (1-2 weeks)
# Change all prepared statement syntax
# Update repository layer for PostgreSQL
# Rewrite all SQL queries for compatibility

# 4. Test everything (3-5 days)
# Regression test all features
# Fix edge cases and bugs

# 5. Deploy
export DATABASE_URL="postgresql://..."
```

---

## Real-World Use Cases

### Use Case 1: Solo Developer with 2 Machines
**User:** "I work on desktop at office, laptop at home. I want same tasks everywhere."

**Best Solution:** ✅ **Turso Free Tier**
- Setup once (5 min)
- Zero monthly cost
- Works offline (airplane, coffee shop)
- Syncs automatically when online

---

### Use Case 2: Small Team (3-5 developers)
**User:** "Our team wants shared tasks, real-time updates, simple setup."

**Best Solution:** ✅ **Turso Developer Plan ($5/month)**
- Team shares one database
- Real-time sync across team
- Still SQLite-compatible
- If outgrow: upgrade to $25 Supabase

---

### Use Case 3: Enterprise Team (10+ users)
**User:** "We need SSO, audit logs, SOC2 compliance, dedicated support."

**Best Solution:** ⚠️ **Supabase Team ($599/month)** OR **Custom Enterprise**
- Not Turso (may lack enterprise compliance)
- Worth the PostgreSQL migration for enterprise features
- Dedicated support, SLAs, compliance

---

## Implementation Roadmap (Turso)

### Phase 1: Core Cloud Support (1-2 weeks)
- [ ] Database abstraction layer
- [ ] Add libSQL driver
- [ ] Update config to support cloud URLs
- [ ] CLI commands: `shark cloud init`, `shark cloud sync`
- [ ] Export/import utilities
- [ ] Documentation

**Risk:** Low (opt-in, backward compatible)
**User Value:** High (solves multi-workstation pain)

### Phase 2: Enhanced Features (2-3 weeks, optional)
- [ ] Real-time change notifications
- [ ] User authentication
- [ ] Conflict resolution UI

**When:** Only if Phase 1 adoption high

### Phase 3: Alternative Backends (4-6 weeks, optional)
- [ ] PostgreSQL support (if enterprise demand)
- [ ] Schema translation layer
- [ ] Feature parity across backends

**When:** Customer demand justifies investment

---

## Recommendation Summary

### ✅ RECOMMENDED: Implement Turso (libSQL) Support

**Why:**
1. Lowest migration risk (SQLite-compatible)
2. Best free tier for target users
3. Fastest setup (5 minutes)
4. Offline support (critical for CLI)
5. Low development effort (1-2 weeks)

**Implementation:**
- Make it **opt-in** (existing users unaffected)
- Keep local SQLite as **default**
- Add cloud as **optional enhancement**

**Target Users:**
- Individual developers with multiple machines (PRIMARY)
- Small teams (2-5 people) wanting shared tasks (SECONDARY)

**Estimated Cost:**
- Development: 1-2 weeks
- User cost: FREE (free tier) or $5/month (power users)

**Next Steps:**
1. Get approval to proceed
2. Create Epic E09: "Cloud Database Support"
3. Build 1-week prototype with Turso
4. Beta test with 5-10 early adopters
5. Phased rollout (opt-in only)

---

## FAQ

**Q: Will this break my existing local database?**
A: No. Cloud support is opt-in. Default behavior remains local SQLite.

**Q: What if Turso shuts down or changes pricing?**
A: We'll abstract the database layer to allow swapping backends. You can export to local SQLite anytime.

**Q: How much will this cost me?**
A: Free for most individual users (500M rows read/month). Power users might pay $5/month.

**Q: Can I use this offline?**
A: Yes! Turso supports offline mode with automatic sync when connection restored.

**Q: What about PostgreSQL? I want "real" database.**
A: We can add PostgreSQL later if there's demand. Start with Turso (easier), migrate later if needed.

**Q: Do I have to use cloud? Can I stay local-only?**
A: Yes! Local SQLite remains the default. Cloud is completely optional.

---

## Decision

**Should we implement cloud database support via Turso?**

- ✅ **YES** → Proceed with Epic E09 (1-2 week implementation)
- ❌ **NO** → Document manual sync alternatives (Git, Dropbox, network mount)
- ⏸️ **DEFER** → Monitor user demand, revisit Q2 2026

**ProductManager Recommendation:** ✅ **PROCEED**
- Low risk (opt-in, backward compatible)
- High user value (solves real pain point)
- Minimal effort (1-2 weeks)
- Free for target users
- Future-proof (can add PostgreSQL later if needed)
