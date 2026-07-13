# Specialized Documentation

## Purpose

Document domain-specific technical details that do not fit the general analysis areas. This
adapts entirely to what the project contains — document what is present, skip what is not.

## Database documentation (`specialized/database/`)

If the project uses databases, create one file per database technology (e.g.
`oracle-schema.md`, `mongodb-collections.md`, `postgresql-schema.md`) covering:

- **Schema** — tables/collections, relationships, indexes
- **Stored procedures / functions** — what they do, when they are called
- **Migration history** — how the schema has evolved
- **Data access patterns** — how the code reads and writes data

## Infrastructure documentation (`specialized/infrastructure/`)

- **CI/CD pipeline** — build stages, deployment steps, environment promotion
- **Container orchestration** — orchestration manifests, container configs, scaling policies
- **Environment configuration** — every environment, its purpose, how it differs
- **Monitoring & observability** — what is monitored, alerting rules, dashboards

## Other specialized areas

Adapt to what you find:

- **Message queues** — topics, queues, consumer groups, message schemas
- **Caching** — cache layers, invalidation strategies, TTLs
- **Search** — search indexes and query patterns
- **Machine learning** — models, training pipelines, feature stores
- **Third-party integrations** — vendor APIs, webhooks, OAuth flows
