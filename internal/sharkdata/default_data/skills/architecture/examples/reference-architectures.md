# Reference Architectures

This document provides reference architecture examples demonstrating best practices and common patterns. These serve as starting points for feature architecture design.

## Three-Tier Web Application

### Overview
Classic web application with presentation, application, and data layers.

### Architecture

```
User → Load Balancer → Web Servers → Application Servers → Database
                                    ↓
                                  Cache
```

### Components
- **Presentation**: Web servers serving HTML/CSS/JS
- **Application**: Business logic, API endpoints
- **Data**: Relational database, cache layer

### Patterns Used
- Load balancing for high availability
- Session persistence (sticky sessions or external store)
- Database connection pooling
- Read replicas for scaling reads
- Cache for frequently accessed data

### When to Use
- Traditional web applications
- CRUD applications
- Admin panels
- Content management systems

## Microservices Architecture

### Overview
Distributed system with independent, loosely coupled services.

### Architecture

```
API Gateway → Service A → Database A
           → Service B → Database B
           → Service C → Message Queue → Worker Services
```

### Components
- **API Gateway**: Single entry point, routing, authentication
- **Microservices**: Independent services owning their data
- **Message Queue**: Asynchronous communication
- **Service Discovery**: Dynamic service location

### Patterns Used
- Database per service
- Event-driven communication
- Circuit breakers
- Saga pattern for distributed transactions
- API gateway pattern
- Service mesh (optional)

### When to Use
- Large, complex applications
- Multiple teams working independently
- Need independent scaling per service
- Polyglot requirements (different languages/frameworks)

### Trade-offs
- Pro: Independent deployment, scaling, technology choice
- Con: Complexity, distributed transactions, testing

## Serverless Event-Driven Architecture

### Overview
Event-driven system using serverless functions and managed services.

### Architecture

```
Client → API Gateway → Lambda Functions → DynamoDB
                    ↓
                  S3 Events → Lambda → Process Data
                    ↓
                  SNS/SQS → Lambda → Database
```

### Components
- **API Gateway**: HTTP endpoints to Lambda
- **Lambda Functions**: Stateless compute
- **Event Sources**: S3, DynamoDB Streams, SNS/SQS
- **Managed Services**: DynamoDB, S3, etc.

### Patterns Used
- Event sourcing
- CQRS (separate read/write)
- Asynchronous processing
- Stateless functions

### When to Use
- Variable/unpredictable traffic
- Cost optimization (pay per use)
- Quick prototyping
- Event-driven workflows

### Trade-offs
- Pro: Auto-scaling, pay-per-use, no server management
- Con: Cold starts, vendor lock-in, debugging complexity

## Real-Time Data Pipeline

### Overview
Streaming data processing for analytics and real-time features.

### Architecture

```
Data Sources → Kafka → Stream Processor → Real-Time DB
                    ↓
                  Batch Processor → Data Warehouse → BI Tools
```

### Components
- **Message Broker**: Kafka, Kinesis for streaming
- **Stream Processing**: Flink, Spark Streaming for real-time
- **Batch Processing**: Spark, Airflow for historical
- **Data Stores**: OLTP for operations, OLAP for analytics

### Patterns Used
- Lambda architecture (batch + stream)
- Kappa architecture (stream only)
- Event sourcing
- Change data capture

### When to Use
- Real-time analytics dashboards
- Fraud detection
- Recommendation engines
- IoT data processing

## Mobile Backend Architecture

### Overview
Backend optimized for mobile clients with offline support.

### Architecture

```
Mobile App → CDN (Static Assets)
          → API Gateway → BFF (Mobile) → Microservices
                       ↓
                     Push Notifications Service
```

### Components
- **CDN**: Asset delivery, edge caching
- **BFF**: Backend for Frontend optimized for mobile
- **Push Service**: FCM, APNs for notifications
- **Sync Service**: Offline data synchronization

### Patterns Used
- Backend for Frontend (BFF)
- Optimistic updates
- Offline-first design
- Delta sync

### When to Use
- Mobile applications
- Need offline support
- Optimize for mobile bandwidth
- Push notifications

## Multi-Tenant SaaS Architecture

### Overview
Single codebase serving multiple customers (tenants).

### Architecture (Shared DB, Shared Schema)

```
Tenant A Users ↘
Tenant B Users → Load Balancer → App Servers → Shared Database (tenant_id column)
Tenant C Users ↗
```

### Components
- **Tenant Isolation**: Via tenant_id column in all tables
- **Authentication**: Multi-tenant aware
- **Data Access**: Always filtered by tenant_id

### Patterns Used
- Row-level security
- Tenant context propagation
- Shared database with tenant_id

### When to Use
- SaaS applications
- Many small-to-medium tenants
- Cost efficiency priority

### Alternative: Separate DB Per Tenant
- Better isolation
- Higher cost
- Easier to scale individual tenants

## API-First Architecture

### Overview
APIs as first-class citizens, consumed by multiple clients.

### Architecture

```
Web App    ↘
Mobile App → API Gateway → API Services → Database
3rd Party  ↗
```

### Components
- **API Gateway**: Versioning, rate limiting, auth
- **API Services**: RESTful or GraphQL
- **Documentation**: OpenAPI/Swagger
- **SDK/Client Libraries**: For consumers

### Patterns Used
- Contract-first design (OpenAPI spec first)
- API versioning
- Rate limiting
- SDK generation

### When to Use
- Multiple client types
- Third-party integrations
- Mobile + web applications
- API as a product

## Command Query Responsibility Segregation (CQRS)

### Overview
Separate read and write models for optimized performance.

### Architecture

```
Commands → Write Model → Events → Event Store
                                ↓
                              Read Models (multiple projections)
                                ↓
Queries → Read Model 1 (optimized for dashboard)
       → Read Model 2 (optimized for search)
```

### Components
- **Command Handler**: Validates and executes commands
- **Event Store**: Immutable event log
- **Projections**: Build read models from events
- **Read Models**: Denormalized, optimized for queries

### Patterns Used
- Event sourcing
- Eventual consistency
- Materialized views

### When to Use
- Different read and write patterns
- High read scalability needed
- Audit trail required
- Complex domain

### Trade-offs
- Pro: Optimized reads and writes independently
- Con: Complexity, eventual consistency

## Hexagonal Architecture (Ports and Adapters)

### Overview
Business logic independent of external concerns.

### Architecture

```
External Systems ← Adapters → Ports → Core Domain Logic → Ports → Adapters → External Systems
(Database, APIs)                      (Business Rules)                      (UI, CLI)
```

### Components
- **Core Domain**: Business logic, framework-agnostic
- **Ports**: Interfaces for communication
- **Adapters**: Implementations for ports (DB, API, UI)

### Patterns Used
- Dependency inversion
- Interface segregation
- Repository pattern

### When to Use
- Complex domain logic
- Need to swap implementations (DB, frameworks)
- Testability is critical
- Long-lived applications

## Choosing the Right Architecture

| Architecture | Complexity | Scalability | Cost | Best For |
|--------------|-----------|-------------|------|----------|
| Three-Tier | Low | Medium | Low | Traditional web apps, MVPs |
| Microservices | High | High | High | Large complex apps, multiple teams |
| Serverless | Medium | High | Variable | Variable traffic, event-driven |
| Real-Time Pipeline | High | High | High | Analytics, IoT, real-time data |
| Mobile Backend | Medium | High | Medium | Mobile apps, offline support |
| Multi-Tenant SaaS | Medium | Medium | Low | SaaS products, many tenants |
| API-First | Low | High | Medium | API products, multiple clients |
| CQRS | High | Very High | High | Different read/write patterns |
| Hexagonal | Medium | Medium | Medium | Complex domains, testability |

## Key Takeaways

1. **Start Simple**: Begin with simpler architectures (three-tier), evolve as needed
2. **Match Requirements**: Choose based on actual requirements, not buzzwords
3. **Consider Trade-offs**: Every architecture has trade-offs (complexity vs. scalability, cost vs. performance)
4. **Evolve Incrementally**: Migrate from simpler to complex as the system grows
5. **Team Capabilities**: Choose architectures your team can support
6. **Proven Patterns**: Use established patterns, don't reinvent
7. **Document Decisions**: Record why you chose this architecture
