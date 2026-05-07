---
feature_key: E19-F01-sprint-database-schema-core-entity-foundation
epic_key: E19
title: Sprint Database Schema & Core Entity Foundation
description: Introduces the sprint as a first-class database entity by adding three new tables (sprints, sprint_assignments, sprint_capacity), all required indexes and triggers, a schema migration (version bump to 7), and the S### key format to the key parsing service. This is the foundational layer that all other E19 features depend on; nothing in E19 can be built until the sprint data model and polymorphic entity_type+entity_id assignment pattern are in place. The sprint_assignments table uses a polymorphic (entity_type, entity_id) pattern — mirroring entity_notes — so that tasks (E##-F##-###), bugs (B###), change-cards (CC-###), AND tech-debt (TD-###) can all be assigned to sprints. A partial unique index on (entity_type, entity_id) WHERE removed_at IS NULL enforces one-active-sprint-per-entity at the database level. Integration points include internal/db/db.go (migration), internal/keys/service.go (S### key detection), and the existing entity_notes polymorphic pattern which the sprint_assignments table mirrors.
---

# Sprint Database Schema & Core Entity Foundation

**Feature Key**: E19-F01-sprint-database-schema-core-entity-foundation

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem
Shark has no sprint entity in its database. All subsequent E19 features (lifecycle commands, assignment, analytics, planning) depend on three tables that do not yet exist: sprints, sprint_assignments, and sprint_capacity. Without the schema, no sprint data can be persisted and no sprint commands can function.

### Solution
Add the three sprint tables via an idempotent migration, extend the key parsing service to recognize S### keys, and define the polymorphic sprint_assignments pattern (entity_type + entity_id) that allows tasks, bugs, change-cards, AND tech-debt items to be assigned to sprints. entity_type accepts values: `task`, `bug`, `change_card`, `tech_debt`.

### Impact
Unblocks all other E19 features. Zero impact on existing commands or data — all changes are additive new tables.

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

*Last Updated*: 2026-05-05
