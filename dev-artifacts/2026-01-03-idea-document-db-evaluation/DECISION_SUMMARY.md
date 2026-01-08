# Decision Summary: Document Database Migration

**Idea ID:** I-2026-01-03-03
**Decision Date:** 2026-01-03
**Architect:** Architect Agent
**Status:** ❌ REJECTED

---

## Quick Decision

**DO NOT migrate from SQLite to document database.**

---

## Why Not?

### 1. Wrong Tool for the Job
- **Shark is:** Local, single-user CLI tool
- **Document DBs are for:** Distributed, cloud-native, multi-user systems
- **Mismatch:** Using a distributed database for local storage is architectural overkill

### 2. Current SQLite is Excellent
✅ Perfect for embedded use (zero config, single file)
✅ Battle-tested (most deployed database, 20+ years)
✅ Fast (all queries <10ms)
✅ Reliable (ACID guarantees, WAL mode)
✅ Simple (no server process, easy backup)

### 3. Migration Would Be Painful
- 6-8 weeks of development time
- ~6,000 lines of code to rewrite
- 38 repository methods completely rewritten
- 150+ test files need updates
- High risk of data corruption
- High risk of performance regression

### 4. No Benefits
| What We'd Gain | Value for Shark |
|----------------|-----------------|
| Schema flexibility | LOW - Schema is stable |
| Nested documents | LOW - Already use JSON in SQLite |
| Horizontal scaling | NONE - Single-user tool |
| Cloud sync | NONE - Local-only |

| What We'd Lose | Impact |
|----------------|--------|
| Foreign key constraints | Data corruption risk |
| Automatic cascading | Manual cleanup code |
| Join performance | Slow queries |
| CHECK constraints | More validation bugs |
| Triggers | More boilerplate |

---

## What About Local Document DB Options?

Researched these options:

1. **Lungo** (MongoDB-compatible, Go)
   - Status: Experimental
   - Issue: Not battle-tested, small community
   - Verdict: Too risky for production

2. **PoloDB** (Document DB, Rust)
   - Status: Beta
   - Issue: Rust FFI/CGO complexity, not native Go
   - Verdict: Adds unnecessary complexity

3. **BadgerDB/BoltDB** (Key-Value, Go)
   - Status: Production-ready
   - Issue: Not document databases - still need manual relationship management
   - Verdict: No advantage over SQLite

**None of these are superior to SQLite for Shark's use case.**

---

## When WOULD Document DB Make Sense?

If Shark evolved to have these requirements:
- ✅ Multi-user collaboration with cloud sync
- ✅ Horizontal scaling (millions of tasks)
- ✅ User-defined schema flexibility
- ✅ Distributed deployment across servers

**Current Reality:** Shark has NONE of these requirements.

---

## Alternative Solutions

If there are specific pain points, solve them without migration:

| Pain Point | Solution (No Migration) |
|-----------|------------------------|
| Schema changes are hard | Already automatic via migrations |
| Want nested data | Already using JSON1 extension |
| Want better performance | Add more indexes (cheap in SQLite) |
| Want better queries | Already optimal with indexes |

---

## Architecture Principles Alignment

Our principles: **Appropriate, Proven, Simple**

**SQLite:**
- ✅ **Appropriate:** Perfect for local CLI tools
- ✅ **Proven:** 20+ years, billions of deployments
- ✅ **Simple:** Single file, zero config, no server

**Document DB:**
- ❌ **Appropriate:** Designed for distributed systems (overkill)
- ❌ **Proven:** Experimental options (Lungo, PoloDB)
- ❌ **Simple:** Requires reimplementing all relational logic

---

## Final Recommendation

1. **Archive this idea** - No further investigation needed
2. **Keep SQLite** - It's optimal for Shark's use case
3. **Focus on features** - Invest time in user value, not infrastructure rewrites

---

## References

Full technical evaluation: `I-2026-01-03-03-evaluation.md`

Key findings:
- 40% of repository methods rely on JOINs (very hard to migrate)
- 13 foreign key relationships enforcing data integrity
- 20+ indexes optimized for query patterns
- Migration would be XXL complexity, HIGH risk, LOW value

---

**Bottom Line:** SQLite is not the problem - it's the perfect solution for Shark Task Manager.
