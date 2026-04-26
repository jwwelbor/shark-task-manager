---
feature_key: E07-F42-add-size-field-to-all-entities-numeric-with-t-shir
epic_key: E07
title: Add Size field to all entities (numeric with t-shirt mapping)
description: 
---

# Add Size field to all entities (numeric with t-shirt mapping)

**Feature Key**: E07-F42-add-size-field-to-all-entities-numeric-with-t-shir

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Triage Notes (2026-04-25)

Captured via `/triage`. Elaborate via `/run` or `/prd` when picked up.

**What**: Add a first-class `Size` field to all 7 entity types (Epic, Feature, Task, Bug, ChangeCard, TechDebt, Idea) on the shared `BaseEntity` (`internal/models/entity.go:36`). Today there's no size concept — only `Priority` (1–10) and `ExecutionOrder`. The closest existing thing is `complexity_tier` stored in the free-form `Metadata` map and surfaced only as a template variable (`internal/config/template/helpers.go:376-388`); it has no schema, no validation, no CLI flag, no rollups.

**Why**: `.claude/rules/development-workflows.md` already prescribes t-shirt sizes (XS–XXL) or story points (1, 2, 3, 5, 8, 13) as guidance, including a "L/XL/XXL must be broken down" rule. None of it is enforceable or queryable today. Making size a typed field unlocks filtering, rollups, and (eventually) workflow gates.

**Decision (user, 2026-04-25)**:
- **Store internally as a number** (Fibonacci: 1, 2, 3, 5, 8, 13).
- **Mapping layer** translates to/from t-shirt labels (XS=1, S=2, M=3, L=5, XL=8, XXL=13).
- **Both forms accepted** on input; both forms presentable on output.

**Scope sketch** (not a spec — for breakdown later):
- Add `Size *int` to `BaseEntity` (nullable, progressive adoption).
- New `internal/models/size.go` with the canonical numeric values, label mapping, and `ParseSize(string) (int, error)` accepting `"3"`, `"M"`, `"m"`, etc.
- `ValidateSize` allowlist in `internal/models/validation.go`.
- DB migration adding `size INTEGER` column to entity tables (epics, features, tasks, bugs, change_cards, tech_debt, ideas) — bump `CurrentSchemaVersion` per `.claude/rules/database-critical.md`.
- `--size` flag on all `create` and `update` commands (~14 commands), accepting either form.
- Repository SELECT/INSERT/UPDATE statements include `size` everywhere.
- Template placeholder in `internal/config/template/helpers.go` (both numeric and label forms).
- `complexity_tier` Metadata read kept as a fallback for one release, then deprecated.

**Out of scope (deferred follow-ups)**:
- Workflow gate enforcement (e.g., "block transition to `ready_for_development` if size > M"). Configurable in `.sharkconfig.json` later.
- Rollups / size-based analytics.
- Whether to require sizing per entity type (similar to `tag_required_for` from E28).

**Reference precedent**: This feature mirrors E28 (Entity Tagging) in shape — cross-cutting attribute, vocabulary, applied to all entities. Worth reading E28 implementation patterns when this is picked up.

---

## Goal

### Problem
[Describe the user problem or business need in 3-5 sentences. Be specific about who experiences this problem and why it matters.]

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

*Last Updated*: 2026-04-25
