---
feature_key: E27-F02-viewer-api-endpoints-read-only-dashboard-data-laye
epic_key: E27
title: Viewer API Endpoints - Read-Only Dashboard Data Layer
description: Add a new internal/api/viewer package with ViewerHandler (7 read-only endpoints) and ViewerService, providing the JSON data layer that the SPA consumes for dashboard rollups, project hierarchy, entity details, file content, and status history.
---

# Viewer API Endpoints - Read-Only Dashboard Data Layer

**Feature Key**: E27-F02-viewer-api-endpoints-read-only-dashboard-data-laye

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Description

This feature introduces `internal/api/viewer/` (handler.go, service.go, types.go, handler_test.go) and `internal/services/viewer_service.go` — a read-only data layer that composes existing Epic/Feature/Task/Bug repositories to produce dashboard-shaped aggregates. The seven endpoints registered under `/api/v1/viewer/` are: `GET /summary` (entity-type counts with status breakdowns and colors), `GET /hierarchy` (full epic→feature tree with task counts for the sidebar), `GET /history/{key}` (status audit trail for any entity type), `GET /file/{key}` (raw markdown of the entity's spec file, path resolved from DB with project-root validation, never from user input), `GET /features/{key}/tasks` (filterable task list for a feature), `GET /recent-activity` (N most recent status changes across all types with optional entity_type/since filters), and `GET /workflow-meta` (full workflow definition serialized for the UI color/phase/transition tables). All endpoints are strictly read-only; the ViewerService interface and MockViewerServicer enable unit testing without a real database. Depends on E27-F01 (dbinit) to ensure the server DB connects correctly to both local SQLite and Turso.

---

## Dependencies

- **E27-F01** (DB Init Extraction) — server must support Turso before viewer endpoints are wired

## Integration Points

- `cmd/server/services.go` — ViewerService wired into ServiceContainer
- `cmd/server/main.go` — viewer routes registered on the existing ServeMux
- Existing repositories: EpicRepository, FeatureRepository, TaskRepository, BugRepository, ChangeCardRepository, TaskHistoryRepository, EntityNoteRepository

---

*Last Updated*: 2026-04-11

### Solution
[Explain how this feature solves the problem. Focus on the "what" not the "how."]

### Impact
[Define expected outcomes with specific, measurable metrics.]

**Examples**:
- Reduce user onboarding time by 40%
- Increase feature adoption to 60% of active users within 3 months

---

## User Personas

### Persona 1: [Persona Name/Role]

**Profile**:
- **Role/Title**: [e.g., "Marketing Manager at mid-size B2B SaaS company"]
- **Experience Level**: [e.g., "3-5 years in role, moderate technical proficiency"]
- **Key Characteristics**:
  - [Characteristic 1]
  - [Characteristic 2]

**Goals Related to This Feature**:
1. [Specific goal 1]
2. [Specific goal 2]

**Pain Points This Feature Addresses**:
- [Pain point 1]
- [Pain point 2]

**Success Looks Like**:
[2-3 sentences describing success from this persona's perspective]

---

## User Stories

### Must-Have Stories

**Story 1**: As a [user persona], I want to [perform an action] so that I can [achieve a benefit].

**Acceptance Criteria**:
- [ ] [Specific testable criterion 1]
- [ ] [Specific testable criterion 2]
- [ ] [Specific testable criterion 3]

---

### Should-Have Stories

[Follow same format for important but not critical stories]

---

### Could-Have Stories

[Follow same format for nice-to-have stories]

---

### Edge Case & Error Stories

**Error Story 1**: As a [user persona], when [error condition], I want to [see/receive] so that I can [recover/understand].

**Acceptance Criteria**:
- [ ] [How error is presented]
- [ ] [How user can recover]

---

## Requirements

### Functional Requirements

**Category: [e.g., Core Functionality]**

1. **REQ-F-001**: [Requirement Title]
   - **Description**: [Clear, specific, testable requirement statement]
   - **User Story**: Links to Story [#]
   - **Priority**: [Must-Have | Should-Have | Could-Have]
   - **Acceptance Criteria**:
     - [ ] [Specific criterion 1]
     - [ ] [Specific criterion 2]

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: [Performance Requirement]
   - **Description**: [Specific performance target]
   - **Measurement**: [How it will be measured]
   - **Target**: [Quantitative threshold, e.g., "Page load < 2 seconds on 3G"]
   - **Justification**: [Why this matters]

**Security**

1. **REQ-NF-010**: [Security Requirement]
   - **Description**: [Specific security control]
   - **Implementation**: [High-level approach]
   - **Compliance**: [Relevant standards: OWASP, SOC2, etc.]
   - **Risk Mitigation**: [What threat this addresses]

**Accessibility**

1. **REQ-NF-020**: [Accessibility Requirement]
   - **Description**: [Specific WCAG criterion]
   - **Standard**: [WCAG 2.1 Level AA, etc.]
   - **Testing**: [How compliance will be verified]

---

## Acceptance Criteria

### Feature-Level Acceptance

**Given/When/Then Format**:

**Scenario 1: [Primary Use Case]**
- **Given** [initial context/state]
- **When** [user action is performed]
- **Then** [expected outcome]
- **And** [additional outcome]

**Scenario 2: [Error Handling]**
- **Given** [error precondition]
- **When** [action that triggers error]
- **Then** [error is handled gracefully]
- **And** [user can recover]

---

## Out of Scope

### Explicitly Excluded

1. **[Feature/Capability]**
   - **Why**: [Reasoning - complexity, dependencies, prioritization]
   - **Future**: [Will this be addressed later? If so, when/why?]
   - **Workaround**: [How users can accomplish this currently, if applicable]

---

### Alternative Approaches Rejected

**Alternative 1: [Approach Name]**
- **Description**: [Brief overview]
- **Why Rejected**: [Reasoning]

---

## Success Metrics

### Primary Metrics

1. **[Metric Name]**
   - **What**: [What data point is tracked]
   - **Target**: [Specific goal]
   - **Timeline**: [When to achieve]
   - **Measurement**: [How to measure]

---

### Secondary Metrics

- **[Metric]**: [Brief description and target]

---

## Dependencies & Integrations

### Dependencies

- **[System/Feature/Service]**: [Description of dependency]

### Integration Requirements

- **[External System]**: [What data/functionality is exchanged]

---

## Compliance & Security Considerations

[If applicable, note specific requirements]:
- **Regulatory**: [GDPR, HIPAA, SOC2, etc.]
- **Data Protection**: [Encryption, access controls]
- **Audit**: [Logging, audit trail requirements]

---

*Last Updated*: 2026-04-11
