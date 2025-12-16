# Data Modeling Patterns

This document contains common data modeling patterns, schema design best practices, and database architecture patterns.

## Relational Database Patterns

### Normalization

**First Normal Form (1NF)**:
- Each column contains atomic values (no lists or nested structures)
- Each row is unique
- No repeating groups

**Second Normal Form (2NF)**:
- Meets 1NF requirements
- No partial dependencies (all non-key columns depend on entire primary key)

**Third Normal Form (3NF)**:
- Meets 2NF requirements
- No transitive dependencies (non-key columns depend only on primary key)

**When to Normalize**:
- OLTP systems (write-heavy)
- Data integrity is critical
- Minimize data redundancy

**When to Denormalize**:
- Read-heavy workloads
- Performance critical
- Data warehouse / analytics
- Trade data redundancy for query speed

### Primary Key Patterns

**Auto-Increment Integer**:
```
id: SERIAL PRIMARY KEY (PostgreSQL)
id: AUTO_INCREMENT (MySQL)
```

**Pros**: Simple, small storage, ordered
**Cons**: Predictable, exposes record count, sharding complexity
**Use for**: Most tables, single-database systems

**UUID (v4 - Random)**:
```
id: UUID PRIMARY KEY DEFAULT gen_random_uuid()
```

**Pros**: Globally unique, no coordination needed, good for distributed systems
**Cons**: Larger storage (16 bytes), random (no natural ordering), index performance
**Use for**: Distributed systems, public-facing IDs, multi-database sync

**UUID (v7 - Time-Ordered)**:
```
id: UUID PRIMARY KEY (timestamp-based)
```

**Pros**: Globally unique, time-ordered (better index performance than v4)
**Cons**: Larger storage
**Use for**: Distributed systems needing ordering

**Natural Keys**:
```
email: VARCHAR(255) PRIMARY KEY
```

**Pros**: Meaningful, no lookup needed
**Cons**: Immutable requirement, potential size, updates are complex
**Use for**: Rare cases where truly immutable and unique

**Composite Keys**:
```
PRIMARY KEY (user_id, product_id)
```

**Use for**: Junction tables, multi-tenant data with tenant_id

### Foreign Key Patterns

**Basic Foreign Key**:
```
Table: orders
  user_id: INTEGER REFERENCES users(id)
```

**With Cascade Actions**:
```
user_id REFERENCES users(id) ON DELETE CASCADE
user_id REFERENCES users(id) ON DELETE SET NULL
user_id REFERENCES users(id) ON DELETE RESTRICT
```

**CASCADE**: Delete related records
**SET NULL**: Null out foreign key
**RESTRICT**: Prevent deletion if references exist

**When to use**:
- CASCADE: Parent-child where child is meaningless without parent
- SET NULL: Optional relationship
- RESTRICT: Prevent accidental data loss

### One-to-Many Relationship

**Pattern**: Foreign key on the "many" side

```
users table:
  id: PRIMARY KEY

orders table:
  id: PRIMARY KEY
  user_id: FOREIGN KEY REFERENCES users(id)
```

**Query pattern**:
- Get user's orders: `WHERE user_id = ?`
- Add index on `user_id` for performance

### Many-to-Many Relationship

**Pattern**: Junction table with two foreign keys

```
users table:
  id: PRIMARY KEY

roles table:
  id: PRIMARY KEY

user_roles table:
  user_id: FOREIGN KEY REFERENCES users(id)
  role_id: FOREIGN KEY REFERENCES roles(id)
  PRIMARY KEY (user_id, role_id)
```

**With additional attributes**:
```
user_roles table:
  user_id: FOREIGN KEY
  role_id: FOREIGN KEY
  granted_at: TIMESTAMP
  granted_by: INTEGER
  PRIMARY KEY (user_id, role_id)
```

### One-to-One Relationship

**Pattern**: Foreign key with unique constraint

```
users table:
  id: PRIMARY KEY

user_profiles table:
  id: PRIMARY KEY
  user_id: INTEGER UNIQUE REFERENCES users(id)
```

**Alternative**: Same table (if always present)

**When to separate**:
- Optional relationship
- Different access patterns
- Large text/binary data (performance)
- Security isolation

### Soft Delete Pattern

**Pattern**: Add `deleted_at` timestamp instead of deleting

```
users table:
  id: PRIMARY KEY
  name: VARCHAR
  deleted_at: TIMESTAMP NULL
```

**Queries**:
- Active records: `WHERE deleted_at IS NULL`
- Deleted records: `WHERE deleted_at IS NOT NULL`
- All records: No filter

**Pros**:
- Audit trail
- Can "undelete"
- Maintain referential integrity

**Cons**:
- Every query needs filter
- Unique constraints more complex
- Storage grows over time

**When to use**: Audit requirements, user data retention

### Temporal Data Pattern

**Pattern**: Track history with valid_from/valid_to

```
product_prices table:
  id: PRIMARY KEY
  product_id: FOREIGN KEY
  price: DECIMAL
  valid_from: TIMESTAMP
  valid_to: TIMESTAMP
  CONSTRAINT no_overlap CHECK (valid_from < valid_to)
```

**Queries**:
- Current price: `WHERE valid_from <= NOW() AND valid_to > NOW()`
- Price at date: `WHERE valid_from <= ? AND valid_to > ?`

**Use for**: Pricing history, employee positions, changing relationships

### Audit Trail Pattern

**Pattern**: Separate audit table tracking all changes

```
users table:
  id: PRIMARY KEY
  name: VARCHAR
  email: VARCHAR
  updated_at: TIMESTAMP

users_audit table:
  id: PRIMARY KEY
  user_id: FOREIGN KEY
  changed_by: INTEGER
  changed_at: TIMESTAMP
  field_name: VARCHAR
  old_value: TEXT
  new_value: TEXT
```

**Implementation**: Database triggers or application-level

**Use for**: Compliance, security, debugging

### Polymorphic Association Pattern

**Pattern**: A table relates to multiple other tables

```
comments table:
  id: PRIMARY KEY
  commentable_type: VARCHAR (e.g., "post" or "photo")
  commentable_id: INTEGER
  content: TEXT
```

**Pros**: Flexible, DRY
**Cons**: Can't use foreign keys, complex queries

**Alternative**: Separate junction tables (more normalized)
```
post_comments table:
  post_id: FOREIGN KEY
  comment_id: FOREIGN KEY

photo_comments table:
  photo_id: FOREIGN KEY
  comment_id: FOREIGN KEY
```

## Indexing Patterns

### B-Tree Index (Default)

**Pattern**: Standard index for most queries

```
CREATE INDEX idx_users_email ON users(email);
```

**Use for**:
- Equality comparisons: `WHERE email = ?`
- Range queries: `WHERE created_at > ?`
- Sorting: `ORDER BY created_at`
- Prefix searches: `WHERE name LIKE 'John%'`

### Composite Index

**Pattern**: Index on multiple columns

```
CREATE INDEX idx_users_status_created ON users(status, created_at);
```

**Column order matters**:
- Left-most prefix rule: Can use for (status) or (status, created_at), but not (created_at) alone
- Put most selective column first (or column in WHERE clause)

**Use for**: Queries filtering on multiple columns

### Unique Index

**Pattern**: Enforce uniqueness and provide fast lookup

```
CREATE UNIQUE INDEX idx_users_email ON users(email);
```

**Use for**: Natural keys, unique constraints

### Partial Index

**Pattern**: Index only subset of rows

```
CREATE INDEX idx_active_users ON users(email) WHERE status = 'active';
```

**Pros**: Smaller index, faster for specific queries
**Use for**: Frequently queried subsets

### Full-Text Search Index

**Pattern**: Index for text search (PostgreSQL GIN, MySQL FULLTEXT)

```
CREATE INDEX idx_posts_content_fts ON posts USING GIN(to_tsvector('english', content));
```

**Use for**: Text search, keyword matching

### Covering Index

**Pattern**: Index includes all columns needed for query

```
CREATE INDEX idx_users_lookup ON users(email) INCLUDE (name, status);
```

**Pros**: Index-only scan (no table lookup needed)
**Use for**: Performance-critical queries

## Schema Evolution Patterns

### Additive Changes (Safe)

**Safe operations**:
- Add new table
- Add new column (with default or NULL)
- Add new index
- Drop index

**Deploy without downtime**

### Non-Additive Changes (Risky)

**Risky operations**:
- Rename column
- Change column type
- Add NOT NULL constraint
- Drop column

**Migration strategy**:
1. Add new column
2. Dual-write to old and new
3. Backfill data
4. Switch reads to new column
5. Drop old column

### Versioned Schema Pattern

**Pattern**: Include version column for schema evolution

```
CREATE TABLE schema_version (
  version: INTEGER PRIMARY KEY
  applied_at: TIMESTAMP
);
```

**Use migration tools**: Alembic, Flyway, Liquibase, Prisma Migrate

## Multi-Tenancy Patterns

### Shared Database, Shared Schema

**Pattern**: All tenants in same tables, filtered by tenant_id

```
users table:
  id: PRIMARY KEY
  tenant_id: INTEGER
  name: VARCHAR

CREATE INDEX idx_users_tenant ON users(tenant_id);
```

**Pros**: Simple, cost-effective, easy maintenance
**Cons**: Security risk (query bugs expose data), noisy neighbors, hard to customize per tenant

### Shared Database, Separate Schema

**Pattern**: Each tenant has own schema in same database

```
Database: myapp
  Schema: tenant_1.users
  Schema: tenant_2.users
```

**Pros**: Data isolation, can customize schema per tenant
**Cons**: More complex, schema migrations across all tenants

### Separate Database Per Tenant

**Pattern**: Each tenant has dedicated database

**Pros**: Complete isolation, easy to backup/restore per tenant, custom performance tuning
**Cons**: High overhead, expensive, complex provisioning

**Choose based on**:
- Number of tenants (few → separate DB, many → shared)
- Isolation requirements
- Customization needs

## JSON/Document Patterns (in Relational DBs)

### JSON Column Pattern

**Pattern**: Store flexible data in JSON column (PostgreSQL JSONB, MySQL JSON)

```
users table:
  id: PRIMARY KEY
  name: VARCHAR
  preferences: JSONB
```

**Pros**: Flexible schema, no migrations for preference changes
**Cons**: Harder to query, no referential integrity, type validation at app level

**Use for**:
- User preferences
- Metadata
- Sparse fields
- Flexible attributes

**Indexing JSON**:
```
CREATE INDEX idx_users_prefs ON users USING GIN(preferences);
```

### Hybrid Pattern

**Pattern**: Core fields in columns, flexible data in JSON

```
products table:
  id: PRIMARY KEY
  name: VARCHAR
  price: DECIMAL
  category_id: INTEGER
  attributes: JSONB  -- Color, size, material vary by category
```

## Performance Patterns

### Read Replica Pattern

**Pattern**: Separate read and write databases

**Architecture**:
- Primary database: All writes
- Read replicas: All reads (replicated from primary)

**Pros**: Scale reads horizontally, offload reporting queries
**Cons**: Replication lag, eventual consistency

### Partitioning Pattern

**Pattern**: Split large table into smaller physical tables

**Horizontal Partitioning (Sharding)**:
```
users_2024_01 table
users_2024_02 table
users_2024_03 table
```

**Partitioning strategies**:
- Range (by date, ID)
- Hash (by ID modulo)
- List (by category, region)

**Pros**: Improve query performance, easier archival
**Cons**: Complex queries spanning partitions, maintenance overhead

### Materialized View Pattern

**Pattern**: Pre-compute and store query results

```
CREATE MATERIALIZED VIEW user_order_summary AS
SELECT user_id, COUNT(*) as order_count, SUM(total) as total_spent
FROM orders
GROUP BY user_id;
```

**Refresh**: Manual or scheduled

**Pros**: Fast reads for complex aggregations
**Cons**: Stale data, storage overhead, refresh cost

**Use for**: Dashboards, reports, analytics

## Choosing Database Type

### Relational (PostgreSQL, MySQL)
- Structured data with relationships
- ACID compliance required
- Complex queries, joins
- Data integrity critical

### Document (MongoDB, CouchDB)
- Semi-structured or evolving schema
- Hierarchical data
- Denormalized data
- No complex joins needed

### Key-Value (Redis, DynamoDB)
- Simple key-based lookups
- Caching
- Session storage
- High throughput, low latency

### Graph (Neo4j, Neptune)
- Complex relationships (social networks, recommendations)
- Traversal queries
- Multiple-degree connections

### Time-Series (InfluxDB, TimescaleDB)
- Time-stamped data (metrics, logs, IoT)
- High-volume writes
- Time-based queries and aggregations

### Vector (Weaviate, Pinecone, pgvector)
- Similarity search
- Embeddings (ML/AI)
- Semantic search
