# Integration Patterns

This document contains common patterns for integrating services, systems, and external APIs.

## Service Communication Patterns

### Synchronous Communication

#### REST API

**Pattern**: HTTP-based request/response

**When to use**:
- Client needs immediate response
- Simple request/response model
- Widely supported, standardized

**Best Practices**:
- Idempotent operations (PUT, DELETE, GET)
- Proper HTTP methods and status codes
- Versioning strategy
- Timeouts and circuit breakers
- Retry logic with exponential backoff

#### gRPC

**Pattern**: HTTP/2-based RPC with Protocol Buffers

**When to use**:
- High-performance service-to-service communication
- Strongly-typed contracts
- Streaming (bi-directional, server, client)
- Internal microservices

**Pros**: Fast, efficient, type-safe, streaming
**Cons**: Less human-readable, requires proto definitions

### Asynchronous Communication

#### Message Queue

**Pattern**: Producer sends messages to queue, consumer processes

**Examples**: RabbitMQ, AWS SQS, Azure Queue

**When to use**:
- Decouple services
- Handle variable load (buffering)
- Guaranteed delivery
- Background jobs

**Queue Types**:
- **Point-to-Point**: One consumer per message
- **Work Queue**: Multiple consumers compete for messages

**Best Practices**:
- Idempotent message processing
- Dead-letter queue for failed messages
- Message TTL and retention
- Visibility timeout for processing time

#### Publish/Subscribe (Pub/Sub)

**Pattern**: Publishers send to topic, multiple subscribers receive

**Examples**: Redis Pub/Sub, AWS SNS, Google Pub/Sub, Kafka

**When to use**:
- Fan-out to multiple consumers
- Event notifications
- Real-time updates
- Loosely coupled services

**Pattern**:
```
Publisher → Topic → Subscriber A
                 → Subscriber B
                 → Subscriber C
```

**Best Practices**:
- Design events for extensibility
- Include event metadata (timestamp, id, version)
- Schema versioning for events
- At-least-once delivery (handle duplicates)

#### Event Streaming

**Pattern**: Ordered, replayable event log

**Examples**: Apache Kafka, AWS Kinesis, Azure Event Hubs

**When to use**:
- Event sourcing
- High-throughput data pipelines
- Multiple consumers need same data
- Replay events for debugging or new consumers

**Kafka Concepts**:
- **Topic**: Category of events
- **Partition**: Ordered log within topic
- **Consumer Group**: Load balancing across consumers
- **Offset**: Position in partition

**Best Practices**:
- Partition key for ordering guarantees
- Retention policy (time or size)
- Compaction for state snapshots
- Monitor consumer lag

## Integration Strategies

### API Gateway Pattern

**Pattern**: Single entry point for all client requests

**Responsibilities**:
- Routing to backend services
- Authentication and authorization
- Rate limiting
- Request/response transformation
- Caching
- Logging and monitoring
- Protocol translation (HTTP to gRPC)

**Examples**: Kong, AWS API Gateway, Azure API Management

**When to use**:
- Microservices architecture
- Multiple backend services
- Need centralized policies (auth, rate limiting)
- Public API for external clients

**Trade-offs**:
- Pro: Simplified client, centralized control
- Con: Single point of failure, added latency

### Backend for Frontend (BFF)

**Pattern**: Separate backend for each frontend type

```
Web App → Web BFF → Microservices
Mobile App → Mobile BFF → Microservices
```

**When to use**:
- Different frontends have different data needs
- Optimize payload per device type
- Different authentication per platform

**Best Practices**:
- Keep BFF thin (composition, not business logic)
- Owned by frontend team
- Share common logic in services

### Service Mesh

**Pattern**: Infrastructure layer for service-to-service communication

**Examples**: Istio, Linkerd, Consul Connect

**Features**:
- Service discovery
- Load balancing
- Mutual TLS
- Circuit breaking
- Observability (tracing, metrics)
- Traffic management (canary, A/B testing)

**When to use**:
- Large microservices deployments
- Need advanced traffic control
- Security (mTLS) without code changes
- Polyglot services

**Trade-offs**:
- Pro: Network concerns out of application code
- Con: Operational complexity, resource overhead

## Reliability Patterns

### Circuit Breaker

**Pattern**: Stop calling failing service, fail fast

**States**:
1. **Closed**: Normal operation, requests pass through
2. **Open**: Failure threshold exceeded, requests fail immediately
3. **Half-Open**: After timeout, allow test request

**Configuration**:
- Failure threshold (e.g., 50% errors in 10 seconds)
- Timeout (how long to wait before half-open)
- Success threshold (successful requests to close)

**When to use**:
- Calling external APIs
- Microservice communication
- Prevent cascading failures

**Libraries**: Hystrix (Java), resilience4j (Java), Polly (.NET), pybreaker (Python)

### Retry Pattern

**Pattern**: Retry failed requests with backoff

**Strategies**:
- **Immediate**: Retry right away (for transient errors)
- **Fixed Delay**: Wait fixed time between retries
- **Exponential Backoff**: Increase delay exponentially (1s, 2s, 4s, 8s)
- **Jitter**: Add randomness to backoff (avoid thundering herd)

**Best Practices**:
- Max retry attempts (e.g., 3)
- Only retry idempotent operations
- Exponential backoff with jitter
- Circuit breaker + retry (circuit breaker to stop retrying)

**Retry-able errors**:
- Network timeouts
- 429 Too Many Requests
- 500, 502, 503, 504
- Connection errors

**Don't retry**:
- 400 Bad Request
- 401 Unauthorized
- 403 Forbidden
- 404 Not Found

### Timeout Pattern

**Pattern**: Set maximum time for operation

**Levels**:
- Connection timeout (e.g., 5 seconds)
- Request timeout (e.g., 30 seconds)
- Total timeout (connection + request)

**Best Practices**:
- Always set timeouts (don't wait indefinitely)
- Reasonable defaults per operation type
- Different timeouts for different endpoints
- Log timeout occurrences

### Bulkhead Pattern

**Pattern**: Isolate resources to prevent cascading failures

**Example**: Separate connection pools per service

```
Service A: Connection Pool (10 connections)
Service B: Connection Pool (10 connections)
Service C: Connection Pool (10 connections)
```

If Service B is slow, it doesn't exhaust connections for A and C.

**When to use**:
- Multiple downstream dependencies
- Prevent one slow service from affecting others

### Graceful Degradation

**Pattern**: Provide reduced functionality when dependencies fail

**Examples**:
- Use cached data if API fails
- Show static content if database is down
- Disable non-critical features

**Best Practices**:
- Identify critical vs. non-critical functionality
- Default to safe fallback
- Inform users of degraded state

## Data Integration Patterns

### Database per Service

**Pattern**: Each microservice has its own database

**Pros**: Loose coupling, independent scaling, technology choice
**Cons**: Distributed transactions, data consistency challenges

**Best Practices**:
- Service owns its data exclusively
- Other services access via API only
- Eventual consistency via events

### Shared Database

**Pattern**: Multiple services access same database

**Pros**: Simple, ACID transactions
**Cons**: Tight coupling, schema changes affect all services

**When to use**: Monolithic or tightly coupled services

### Saga Pattern

**Pattern**: Distributed transactions via events

**Choreography-Based**:
- Each service publishes events
- Other services listen and react
- No central coordinator

**Orchestration-Based**:
- Central orchestrator coordinates
- Tells each service what to do
- Handles compensation logic

**Example - Order Saga**:
1. Create Order (orders service)
2. Reserve Inventory (inventory service)
3. Charge Payment (payment service)
4. If any fails, compensate (unreserve, refund)

**Best Practices**:
- Idempotent operations
- Compensation actions for rollback
- Timeout handling
- Monitor saga state

### Event Sourcing

**Pattern**: Store state changes as events, not current state

**Example**:
```
Events:
- OrderCreated(order_id, user_id, items)
- PaymentReceived(order_id, amount)
- OrderShipped(order_id, tracking_number)

Current State: Replay all events
```

**Pros**: Full audit trail, can reconstruct state, enables time travel
**Cons**: Complexity, eventual consistency, event schema evolution

**When to use**: Audit requirements, complex domain, need to replay events

### CQRS (Command Query Responsibility Segregation)

**Pattern**: Separate read and write models

**Write Model**: Handles commands, ensures consistency
**Read Model**: Optimized for queries, denormalized

**Often combined with Event Sourcing**:
- Commands create events
- Events update read models

**When to use**:
- Different read and write patterns
- High read scalability needed
- Complex queries

## External API Integration Patterns

### Adapter Pattern

**Pattern**: Wrap external API with internal interface

**Benefits**:
- Isolate external dependency
- Easy to swap providers
- Simplified testing (mock adapter)
- Transform external model to internal model

### Webhook Pattern

**Pattern**: External service calls your API when event occurs

**Flow**:
1. Register webhook URL with provider
2. Provider sends HTTP POST when event happens
3. Verify webhook signature
4. Process event asynchronously

**Best Practices**:
- Verify webhook signature (HMAC)
- Respond quickly (202 Accepted)
- Process asynchronously
- Handle duplicates (idempotency)
- Retry failed webhook processing

### Polling Pattern

**Pattern**: Periodically check for updates

**When to use**: No webhook support, simple integration

**Best Practices**:
- Exponential backoff if no changes
- Track last checked timestamp
- Batch processing

**Trade-offs**:
- Pro: Simple, reliable
- Con: Inefficient, delayed updates

## API Versioning Strategies

### URI Versioning

```
/api/v1/users
/api/v2/users
```

**Pros**: Clear, easy to route
**Cons**: URL changes

### Header Versioning

```
Accept: application/vnd.myapi.v2+json
```

**Pros**: Clean URLs
**Cons**: Less visible

### Content Negotiation

```
Accept: application/json; version=2
```

**Pros**: RESTful
**Cons**: Complex

**Best Practice**: URI versioning for major versions, maintain backward compatibility

## Data Synchronization Patterns

### Change Data Capture (CDC)

**Pattern**: Capture database changes and stream to other systems

**Tools**: Debezium, AWS DMS, Maxwell

**Use cases**:
- Replicate to data warehouse
- Update search index
- Cache invalidation
- Event-driven architecture

### ETL/ELT

**Extract, Transform, Load**: Traditional batch data integration

**When to use**: Periodic data sync, reporting, analytics

### Real-Time Replication

**Pattern**: Continuous data replication between databases

**Examples**: Logical replication (PostgreSQL), binlog replication (MySQL)

**Use cases**: Read replicas, disaster recovery, multi-region
