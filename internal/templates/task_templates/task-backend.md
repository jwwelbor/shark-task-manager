{{/* Backend Agent Task Template */}}
---
key: {{.Key}}
title: {{.Title}}
epic: {{.Epic}}
feature: {{.Feature}}
agent: backend
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

## API Endpoints

### Endpoints to Implement

- [ ] Endpoint routes defined
- [ ] Request/response schemas documented
- [ ] Authentication/authorization requirements identified

## Data Models

### Database Schema

- [ ] Tables/collections identified
- [ ] Relationships defined
- [ ] Indexes planned
- [ ] Migrations prepared

### Business Logic

- [ ] Domain models defined
- [ ] Validation rules documented
- [ ] Business rules implemented

## Implementation Details

### Service Layer

- [ ] Service methods defined
- [ ] Transaction boundaries identified
- [ ] External service integrations documented

### Error Handling

- [ ] Error types defined
- [ ] Error responses standardized
- [ ] Logging strategy implemented

## Acceptance Criteria

- [ ] All endpoints return correct responses
- [ ] Input validation working
- [ ] Error handling complete
- [ ] Database operations atomic
- [ ] Performance targets met

## Testing Requirements

### Unit Tests

- [ ] Service layer tests
- [ ] Business logic tests
- [ ] Validation tests
- [ ] Error handling tests

### Integration Tests

- [ ] Database integration tests
- [ ] API endpoint tests
- [ ] External service integration tests

## Notes

- Follow REST/GraphQL conventions
- Implement proper logging and monitoring
- Document all assumptions and trade-offs
