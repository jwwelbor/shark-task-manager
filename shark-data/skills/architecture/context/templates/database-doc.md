# Database Design Document Template (03-data-design.md)

This template is extracted from the db-admin agent.

---

# Data Design: {Feature Name}

**Epic**: {epic-key}
**Feature**: {feature-key}
**Date**: {YYYY-MM-DD}
**Author**: db-admin (data architect)

## Persistence Overview

{Brief description of what data this feature handles and how it will be stored}

### Persistence Mechanism

**Primary Storage**: {database/files/cache/etc.}
**Rationale**: {why this mechanism fits the project and feature}

## Data Model

{Use the appropriate format for the storage mechanism}

### Entity Diagram (if relational)

```mermaid
erDiagram
    {entity relationships}
```

### Schema Definition

{Describe in PROSE, not code. Tables/collections/file structures as specifications.}

#### {EntityName}

**Purpose**: {what this entity represents}

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | UUID | PK | Unique identifier |
| {field} | {type} | {constraints} | {purpose} |
| created_at | timestamp | NOT NULL | Creation timestamp |
| updated_at | timestamp | NOT NULL | Last update timestamp |

### Relationships

| From | To | Type | Description |
|------|-----|------|-------------|
| {entity} | {entity} | 1:N | {description} |

## Query Patterns

{Expected read/write patterns that inform indexing and optimization}

| Operation | Frequency | Pattern |
|-----------|-----------|---------|
| {operation} | High/Medium/Low | {description} |

## Indexing Strategy

| Index | Fields | Type | Purpose |
|-------|--------|------|---------|
| {name} | {fields} | {btree/hash/etc.} | {what queries it supports} |

## Data Integrity

### Constraints
- {constraint 1}
- {constraint 2}

### Validation Rules
- {rule 1}
- {rule 2}

## Migration Strategy

{How to evolve the data model over time}

### Initial Setup
{Steps to create initial schema/structure}

### Version Management
{How schema versions are tracked and applied}

### Rollback Plan
{How to safely rollback changes}

## Performance Considerations

### Expected Data Volume
- Initial: {estimate}
- Growth: {rate}
- Peak: {estimate}

### Optimization Strategies
- {strategy 1}
- {strategy 2}

## Backup & Recovery

{Data protection approach}

## Integration with Interface Contracts

{How the data model maps to/from the DTOs defined in interface contracts}

| DTO Field | Data Field | Transformation |
|-----------|------------|----------------|
| {dto_field} | {data_field} | {if any} |
