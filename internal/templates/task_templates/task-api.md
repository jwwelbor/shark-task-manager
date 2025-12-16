{{/* API Developer Agent Task Template */}}
---
key: {{.Key}}
title: {{.Title}}
epic: {{.Epic}}
feature: {{.Feature}}
agent: api
status: todo
priority: {{.Priority}}
{{- if .DependsOn}}
depends_on: [{{join (quote .DependsOn) ", "}}]
{{- end}}
created_at: {{formatTime .CreatedAt}}
---

# Task: {{.Title}}

## Goal

{{if not (isEmpty .Description)}}{{.Description}}{{else}}[Describe the goal of this task]{{end}}

## API Specification

### Endpoint Details

**Method:** [GET|POST|PUT|PATCH|DELETE]
**Path:** `/api/v1/resource`
**Description:** [Brief description]

### Request Schema

```json
{
  "field": "type"
}
```

### Response Schema

#### Success Response (200/201)

```json
{
  "data": {},
  "meta": {}
}
```

#### Error Responses

- **400 Bad Request:** Invalid input
- **401 Unauthorized:** Authentication required
- **403 Forbidden:** Insufficient permissions
- **404 Not Found:** Resource not found
- **500 Internal Server Error:** Server error

## Authentication & Authorization

- [ ] Authentication method documented
- [ ] Required permissions identified
- [ ] Token/session handling defined

## Implementation Checklist

- [ ] Route handler implemented
- [ ] Request validation added
- [ ] Response serialization implemented
- [ ] Error handling complete
- [ ] API documentation generated

## Testing Requirements

### API Tests

- [ ] Happy path tests
- [ ] Validation error tests
- [ ] Authentication tests
- [ ] Authorization tests
- [ ] Edge case tests

### Performance Tests

- [ ] Response time benchmarks
- [ ] Load testing results
- [ ] Rate limiting verified

## Acceptance Criteria

- [ ] API responds with correct status codes
- [ ] Request/response formats match specification
- [ ] Authentication/authorization working
- [ ] Error messages are clear and actionable
- [ ] API documentation complete

## Notes

- Follow API versioning strategy
- Implement rate limiting if needed
- Add appropriate caching headers
- Document breaking changes
