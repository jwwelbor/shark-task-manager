# API Design Patterns

This document contains common API design patterns and best practices for creating consistent, maintainable, and well-designed APIs.

## RESTful API Design Patterns

### Resource-Oriented URLs

**Pattern**: Use nouns for resources, not verbs

**Good**:
- `GET /api/users` - List users
- `GET /api/users/123` - Get specific user
- `POST /api/users` - Create user
- `PUT /api/users/123` - Update user
- `DELETE /api/users/123` - Delete user

**Bad**:
- `GET /api/getUsers`
- `POST /api/createUser`
- `POST /api/deleteUser/123`

**When to use**: All RESTful APIs

### HTTP Methods for CRUD

| Method | Purpose | Idempotent | Safe |
|--------|---------|------------|------|
| GET | Retrieve resource(s) | Yes | Yes |
| POST | Create new resource | No | No |
| PUT | Replace entire resource | Yes | No |
| PATCH | Partially update resource | No | No |
| DELETE | Remove resource | Yes | No |

**Idempotent**: Multiple identical requests have the same effect as a single request
**Safe**: Request doesn't modify server state

### Nested Resources

**Pattern**: Represent relationships in URL hierarchy

**Examples**:
- `GET /api/users/123/orders` - Get orders for user 123
- `POST /api/projects/456/tasks` - Create task in project 456
- `GET /api/organizations/789/users` - Get users in organization 789

**Trade-offs**:
- Pro: Clear relationship representation
- Con: URLs can get deep (limit to 2-3 levels)

**Alternative**: Use query parameters for filtering
- `GET /api/orders?user_id=123`

### Pagination

**Cursor-Based Pagination** (Recommended for large datasets):
```
GET /api/users?cursor=abc123&limit=20

Response:
{
  "data": [...],
  "pagination": {
    "next_cursor": "def456",
    "has_more": true
  }
}
```

**Pros**: Consistent results, handles real-time data well
**Cons**: Can't jump to arbitrary pages
**When to use**: Large datasets, real-time data, social feeds

**Page-Based Pagination**:
```
GET /api/users?page=2&per_page=20

Response:
{
  "data": [...],
  "pagination": {
    "page": 2,
    "per_page": 20,
    "total_pages": 50,
    "total_count": 1000
  }
}
```

**Pros**: Simple, can jump to any page
**Cons**: Inconsistent with real-time data, expensive to count total
**When to use**: Small datasets, static data, traditional interfaces

**Offset-Based Pagination**:
```
GET /api/users?offset=40&limit=20
```

**When to use**: Simple cases, known data size

### Filtering, Sorting, and Field Selection

**Filtering**:
```
GET /api/users?status=active&role=admin
GET /api/orders?created_after=2024-01-01&status=pending
```

**Sorting**:
```
GET /api/users?sort=created_at:desc
GET /api/products?sort=price:asc,name:asc
```

**Field Selection** (sparse fieldsets):
```
GET /api/users?fields=id,name,email
```

**Trade-offs**:
- Reduces payload size
- More complex API implementation
- Requires careful security (don't expose sensitive fields)

### Versioning Strategies

**URL Versioning** (Most Common):
```
GET /api/v1/users
GET /api/v2/users
```

**Pros**: Clear, easy to route
**Cons**: URL changes, multiple versions running

**Header Versioning**:
```
GET /api/users
Accept: application/vnd.myapi.v2+json
```

**Pros**: Cleaner URLs
**Cons**: Less visible, harder to test

**Query Parameter Versioning**:
```
GET /api/users?version=2
```

**Pros**: Simple
**Cons**: Not RESTful, easily missed

**Recommendation**: Use URL versioning for major versions, maintain backward compatibility when possible

### Error Responses

**Standard Error Structure**:
```
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input data",
    "details": [
      {
        "field": "email",
        "issue": "Invalid email format"
      }
    ],
    "request_id": "req_abc123"
  }
}
```

**HTTP Status Codes**:
- **200 OK**: Success
- **201 Created**: Resource created
- **204 No Content**: Success, no response body
- **400 Bad Request**: Invalid input
- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Authenticated but not authorized
- **404 Not Found**: Resource doesn't exist
- **409 Conflict**: Resource conflict (duplicate, version mismatch)
- **422 Unprocessable Entity**: Validation failed
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Server error (don't expose details)
- **503 Service Unavailable**: Temporary unavailability

**Best Practices**:
- Use appropriate status codes
- Include error code for programmatic handling
- Human-readable message
- Details for validation errors
- Request ID for debugging
- Don't expose stack traces or internal details in production

### Rate Limiting

**Rate Limit Headers**:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 1640995200
```

**Response when exceeded**:
```
HTTP 429 Too Many Requests
Retry-After: 60

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Try again in 60 seconds.",
    "retry_after": 60
  }
}
```

**Common Limits**:
- Per user: 100-1000 requests/hour
- Per IP: 60 requests/minute (unauthenticated)
- Per endpoint: Vary based on cost (e.g., search is more expensive)

### Idempotency Keys

**Pattern**: Client provides unique key for POST requests to prevent duplicates

```
POST /api/orders
Idempotency-Key: order_abc123xyz
```

**Implementation**:
- Store key with result
- If same key received again, return original result
- Key expires after 24 hours

**When to use**: Financial transactions, critical operations

### Bulk Operations

**Bulk Create**:
```
POST /api/users/bulk
{
  "users": [
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob", "email": "bob@example.com"}
  ]
}
```

**Bulk Update**:
```
PATCH /api/users/bulk
{
  "updates": [
    {"id": 1, "status": "active"},
    {"id": 2, "status": "inactive"}
  ]
}
```

**Response includes success/failure per item**:
```
{
  "results": [
    {"id": 1, "status": "success"},
    {"id": 2, "status": "error", "message": "User not found"}
  ]
}
```

**Trade-offs**:
- Pro: Reduces network round-trips
- Con: More complex error handling
- Limit: Cap bulk size (e.g., max 100 items)

## GraphQL Patterns

### Schema Design

**Type-First Approach**: Define types, queries, mutations clearly

**Query Example**:
```
type Query {
  user(id: ID!): User
  users(filter: UserFilter, limit: Int, cursor: String): UserConnection
}

type User {
  id: ID!
  name: String!
  email: String!
  orders: [Order!]!
}
```

**Mutation Example**:
```
type Mutation {
  createUser(input: CreateUserInput!): UserPayload!
  updateUser(id: ID!, input: UpdateUserInput!): UserPayload!
}

input CreateUserInput {
  name: String!
  email: String!
}

type UserPayload {
  user: User
  errors: [Error!]
}
```

### Pagination in GraphQL

**Relay Cursor Connections**:
```
type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

### Error Handling

**Field-Level Errors**:
```
type UserPayload {
  user: User
  errors: [Error!]
}

type Error {
  field: String
  message: String!
  code: String!
}
```

**GraphQL Errors** (for system errors):
```
{
  "errors": [
    {
      "message": "Database connection failed",
      "extensions": {
        "code": "DATABASE_ERROR"
      }
    }
  ],
  "data": null
}
```

## Response Format Patterns

### Envelope Pattern

**With Envelope**:
```
{
  "status": "success",
  "data": {
    "user": {...}
  },
  "meta": {
    "request_id": "req_123"
  }
}
```

**Pros**: Consistent structure, metadata support
**Cons**: Extra wrapper overhead

**Without Envelope** (Direct response):
```
{
  "id": 1,
  "name": "Alice"
}
```

**Pros**: Simpler, less overhead
**Cons**: Harder to add metadata

**Recommendation**: Use envelope for consistency, especially with pagination and errors

### HATEOAS (Hypermedia)

**Pattern**: Include links to related actions

```
{
  "id": 123,
  "name": "Alice",
  "status": "pending",
  "_links": {
    "self": "/api/users/123",
    "approve": "/api/users/123/approve",
    "reject": "/api/users/123/reject",
    "orders": "/api/users/123/orders"
  }
}
```

**When to use**: Complex workflows, state machines, API discoverability

## Authentication Patterns

### Bearer Token (JWT)

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Use for**: Stateless authentication, microservices

### API Key

```
X-API-Key: sk_live_abc123xyz
```

**Use for**: Service-to-service, third-party integrations

### OAuth 2.0

- Authorization Code flow (web apps)
- Client Credentials flow (service-to-service)
- Refresh tokens for long-lived access

## Best Practices Summary

1. **Be Consistent**: Choose patterns and stick to them across all endpoints
2. **Use Standards**: Follow HTTP semantics, RESTful conventions
3. **Version Your API**: Plan for evolution
4. **Document Everything**: Every endpoint, parameter, response
5. **Handle Errors Gracefully**: Clear, actionable error messages
6. **Implement Rate Limiting**: Protect your API from abuse
7. **Paginate Large Results**: Don't return unbounded data
8. **Use Appropriate Status Codes**: HTTP codes communicate meaning
9. **Validate Input**: Server-side validation is non-negotiable
10. **Think About Caching**: Use ETags, Cache-Control headers
