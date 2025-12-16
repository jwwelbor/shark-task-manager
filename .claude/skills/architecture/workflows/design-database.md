# Database Architecture Design Workflow

This workflow guides you through creating comprehensive data persistence design documentation for a feature. It produces the data design document (03-data-design.md) covering data models, schemas, query patterns, and persistence strategies.

## Prerequisites

Before starting this workflow, ensure you have:
1. Feature PRD at `/docs/plan/{epic-key}/{feature-key}/prd.md`
2. Research report at `/docs/plan/{epic-key}/{feature-key}/00-research-report.md`
3. Interface contracts (if defined by feature-architect)
4. Backend design (04-backend-design.md) to understand DTOs

## Step 1: Analyze Requirements

### Read the PRD
- Identify what data the feature needs to persist
- Extract data volume estimates
- Note data retention requirements
- Identify query patterns (read-heavy vs. write-heavy)
- Understand data relationships

### Read the Research Report
- Understand what persistence mechanisms the project uses
- Identify existing database technology (PostgreSQL, MongoDB, Redis, etc.)
- Note ORM/migration tools in use (SQLAlchemy, Prisma, TypeORM)
- Review existing data models to extend or reference
- Find naming conventions for tables/collections/fields

### Review Interface Contracts
- Understand DTOs that need to be persisted
- Identify data transformations needed
- Note validation requirements

### Review Backend Design
- Understand what data the API needs to store/retrieve
- Identify query patterns from endpoint specifications
- Note any caching requirements

## Step 2: Determine Persistence Mechanism

### Choose Storage Type

Based on project patterns and feature requirements:

**Relational Database** (PostgreSQL, MySQL, SQLite):
- Structured data with clear relationships
- ACID compliance required
- Complex queries needed
- Project already uses relational DB

**Document Store** (MongoDB, Firestore):
- Semi-structured or evolving schema
- Hierarchical data
- Horizontal scalability priority
- Project already uses document DB

**Key-Value Store** (Redis, DynamoDB):
- Simple lookups by key
- Caching layer
- Session storage
- High-speed access required

**File Storage** (Local filesystem, S3):
- Configuration data
- Small datasets
- Simple read/write patterns
- No complex queries needed

**Time-Series Database** (InfluxDB, TimescaleDB):
- Time-stamped data
- Metrics and monitoring
- High-volume writes

**Vector Database** (Weaviate, Pinecone, pgvector):
- Embeddings storage
- Semantic search
- ML/AI features

### Document Rationale
Explain why this mechanism fits:
- Project patterns (consistency)
- Feature requirements (functionality)
- Performance needs (scalability)
- Team expertise (maintainability)

## Step 3: Design Data Model

### For Relational Databases

#### Create Entity Relationship Diagram

Use Mermaid ERD to show:
- All entities (tables)
- Relationships (1:1, 1:N, N:M)
- Cardinality
- Key relationships

#### Define Each Entity (Table)

For each table, specify:

**Entity Name**:
- **Purpose**: What this entity represents

**Fields**:
| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | UUID/Serial | PK | Unique identifier |
| field_name | type | constraints | purpose |
| created_at | timestamp | NOT NULL | Creation timestamp |
| updated_at | timestamp | NOT NULL | Last update timestamp |

**Common field types**:
- UUID, Integer, Serial (IDs)
- String, Text, VARCHAR(n)
- Boolean
- Timestamp, Date, Time
- JSON/JSONB (for flexible data)
- Enum (for fixed sets)

**Common constraints**:
- PRIMARY KEY (PK)
- FOREIGN KEY (FK) references table(field)
- NOT NULL
- UNIQUE
- CHECK (validation)
- DEFAULT value

**Relationships**:
| From | To | Type | Description |
|------|-----|------|-------------|
| this_table | other_table | 1:N | relationship description |

Apply patterns from `context/patterns/data-patterns.md`

### For Document Stores

#### Define Collections

For each collection:
- **Purpose**: What documents represent
- **Document Structure**: Field names, types, nesting
- **Indexes**: Fields to index
- **Validation Rules**: Schema validation if supported

### For Key-Value Stores

- **Key Pattern**: How keys are structured
- **Value Type**: String, JSON, binary
- **TTL**: Time-to-live if applicable
- **Eviction Policy**: How old data is removed

### For File Storage

- **File Structure**: Directory organization
- **File Format**: JSON, CSV, YAML, etc.
- **Naming Convention**: File naming pattern
- **Access Pattern**: How files are read/written

## Step 4: Design Query Patterns

### Identify Common Operations

Create a table:
| Operation | Frequency | Pattern |
|-----------|-----------|---------|
| Fetch user by ID | High | Single record lookup |
| List user's items | Medium | Filtered query with pagination |
| Search items | Low | Full-text search |

**Frequency levels**:
- **High**: > 100 requests/second
- **Medium**: 10-100 requests/second
- **Low**: < 10 requests/second

### Document Query Patterns

For each operation:
- Describe the query logic (in prose, not SQL)
- Note filtering, sorting, pagination needs
- Identify potential bottlenecks
- Consider caching opportunities

## Step 5: Design Indexing Strategy

### For Relational Databases

Create indexes for:
- Primary keys (automatic)
- Foreign keys (for JOIN performance)
- Frequently filtered fields (WHERE clauses)
- Frequently sorted fields (ORDER BY clauses)
- Unique constraints

**Index table**:
| Index Name | Fields | Type | Purpose |
|------------|--------|------|---------|
| idx_users_email | email | btree, unique | Lookup by email |
| idx_items_user_id | user_id | btree | Filter items by user |
| idx_items_created | created_at | btree | Sort by creation date |

**Index types**:
- btree (default, general purpose)
- hash (equality comparisons)
- gin/gist (full-text, arrays, JSON)

### For Document Stores

- Single-field indexes
- Compound indexes
- Text indexes for search
- Geospatial indexes if applicable

## Step 6: Define Data Integrity

### Constraints
- Primary key constraints
- Foreign key constraints
- Unique constraints
- Check constraints (business rules)
- NOT NULL constraints

### Validation Rules
- Field-level validation (format, range, length)
- Record-level validation (cross-field checks)
- Business logic validation
- Application-level vs. database-level validation

Apply patterns from `context/patterns/data-patterns.md`

## Step 7: Plan Migration Strategy

### Initial Setup
- How to create the schema (migration tool)
- Initial data seeding if needed
- Baseline migration number

### Version Management
- Migration file naming convention
- How migrations are tracked (migration table)
- How to generate new migrations
- Dependencies between migrations

### Rollback Plan
- How to undo migrations
- Data preservation strategy
- Testing rollback procedures
- Backup requirements before migrations

Apply patterns from `context/patterns/data-patterns.md`

## Step 8: Plan Performance Optimization

### Expected Data Volume
- Initial volume estimate
- Growth rate projection
- Peak volume estimate
- Retention/archival strategy

### Optimization Strategies
- Indexing (covered in Step 5)
- Partitioning (for large tables)
- Caching (which queries, TTL)
- Denormalization (if needed for performance)
- Connection pooling configuration
- Query optimization approaches

## Step 9: Plan Backup & Recovery

Document:
- Backup frequency (hourly, daily, etc.)
- Backup retention policy
- Recovery time objective (RTO)
- Recovery point objective (RPO)
- Disaster recovery approach
- Point-in-time recovery capability

## Step 10: Map to Interface Contracts

### Create DTO Mapping Table

| DTO Field | Data Field | Transformation |
|-----------|------------|----------------|
| userId | user_id | Direct mapping |
| createdAt | created_at | Format as ISO-8601 |
| fullName | first_name + last_name | Concatenate with space |

Document:
- Which DTOs map to which entities
- Any data transformations needed
- Which fields are computed vs. stored
- Validation alignment (DTO vs. database)

## Step 11: Quality Checklist

Before finalizing, verify:

### Completeness
- [ ] Persistence mechanism selected and justified
- [ ] All entities/collections defined
- [ ] All fields have types and constraints
- [ ] Relationships documented
- [ ] Query patterns identified
- [ ] Indexes designed
- [ ] Migration strategy defined
- [ ] Performance considerations addressed
- [ ] Backup plan documented
- [ ] DTO mapping complete

### NO CODE Constraint
- [ ] No SQL CREATE statements
- [ ] No ORM model code (SQLAlchemy, Prisma, etc.)
- [ ] No migration code
- [ ] Only prose descriptions and specifications
- [ ] Mermaid ERD for visualization

### Consistency
- [ ] Follows project persistence patterns
- [ ] Naming matches conventions
- [ ] Constraints align with DTOs
- [ ] Aligns with backend design

### Data Integrity
- [ ] Primary keys defined
- [ ] Foreign keys for relationships
- [ ] Constraints prevent invalid data
- [ ] Validation rules comprehensive

### Performance
- [ ] Indexes support common queries
- [ ] Volume estimates realistic
- [ ] Scalability considered
- [ ] Caching strategy defined

## Step 12: Create Document

### File Location
Create `/docs/plan/{epic-key}/{feature-key}/03-data-design.md`

### Use Template
Follow `context/templates/database-doc.md` structure

### Review
- Verify file is complete
- Check all cross-references valid
- Ensure ERD renders correctly
- Confirm no code blocks exist
- Validate DTO mappings align

## Common Patterns to Apply

### Naming Conventions
- Tables: plural nouns (users, items, orders)
- Fields: snake_case for SQL, camelCase for NoSQL
- Timestamps: created_at, updated_at, deleted_at
- Foreign keys: {table}_id (user_id, order_id)

### Standard Fields
- `id`: Primary key (UUID or auto-increment)
- `created_at`: Creation timestamp
- `updated_at`: Last modification timestamp
- `deleted_at`: Soft delete timestamp (if using soft deletes)

### Relationship Patterns
- **One-to-Many**: Foreign key on "many" side
- **Many-to-Many**: Junction table with two foreign keys
- **One-to-One**: Foreign key with unique constraint

### Soft Deletes
- Add `deleted_at` timestamp field
- NULL = active, timestamp = deleted
- Filter deleted records in queries

## Output Requirements

Upon completion, you will have:
1. **03-data-design.md** - Complete data persistence design
2. Data model with entities/collections fully specified
3. Relationships documented and diagrammed
4. Query patterns identified
5. Indexing strategy defined
6. Migration strategy planned
7. DTO mapping documented
8. Document following template exactly
9. NO implementation code, only design specifications
