# Feature PRD Template Structure

This document defines the complete structure for Feature-level PRDs. A feature PRD is a single comprehensive document.

## PRD Structure

```markdown
# {Feature Name}

**Feature Key**: E##-F##-{feature-slug}

---

## Epic

- **Epic PRD**: [Epic Name](/docs/plan/{epic-key}/epic.md)
- **Epic Architecture**: [Architecture](/docs/plan/{epic-key}/architecture.md) _(if available)_

---

## Goal

### Problem
{Describe the user problem or business need in 3-5 sentences. Be specific about who experiences this problem and why it matters.}

### Solution
{Explain how this feature solves the problem. Focus on the "what" not the "how."}

### Impact
{Define expected outcomes with specific, measurable metrics.}

**Examples**:
- Reduce user onboarding time by 40%
- Increase feature adoption to 60% of active users within 3 months

---

## User Personas

### Persona 1: {Persona Name/Role}

**Profile**:
- **Role/Title**: {e.g., "Marketing Manager at mid-size B2B SaaS company"}
- **Experience Level**: {e.g., "3-5 years in role, moderate technical proficiency"}
- **Key Characteristics**:
  - {Characteristic 1}
  - {Characteristic 2}

**Goals Related to This Feature**:
1. {Specific goal 1}
2. {Specific goal 2}

**Pain Points This Feature Addresses**:
- {Pain point 1}
- {Pain point 2}

**Success Looks Like**:
{2-3 sentences describing success from this persona's perspective}

---

{Repeat for additional personas as needed}

---

## User Stories

### Must-Have Stories

**Story 1**: As a {user persona}, I want to {perform an action} so that I can {achieve a benefit}.

**Acceptance Criteria**:
- [ ] {Specific testable criterion 1}
- [ ] {Specific testable criterion 2}
- [ ] {Specific testable criterion 3}

**Story 2**: As a {user persona}, I want to {perform an action} so that I can {achieve a benefit}.

**Acceptance Criteria**:
- [ ] {Specific testable criterion 1}
- [ ] {Specific testable criterion 2}

---

### Should-Have Stories

{Follow same format for important but not critical stories}

---

### Could-Have Stories

{Follow same format for nice-to-have stories}

---

### Edge Case & Error Stories

**Error Story 1**: As a {user persona}, when {error condition}, I want to {see/receive} so that I can {recover/understand}.

**Acceptance Criteria**:
- [ ] {How error is presented}
- [ ] {How user can recover}

---

## Requirements

### Functional Requirements

**Category: {e.g., Core Functionality}**

1. **REQ-F-001**: {Requirement Title}
   - **Description**: {Clear, specific, testable requirement statement}
   - **User Story**: Links to Story {#}
   - **Priority**: [Must-Have | Should-Have | Could-Have]
   - **Acceptance Criteria**:
     - [ ] {Specific criterion 1}
     - [ ] {Specific criterion 2}

2. **REQ-F-002**: {Requirement Title}
   {Follow same format}

**Category: {e.g., Data Management}**

{Continue with additional functional requirements}

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: {Performance Requirement}
   - **Description**: {Specific performance target}
   - **Measurement**: {How it will be measured}
   - **Target**: {Quantitative threshold, e.g., "Page load < 2 seconds on 3G"}
   - **Justification**: {Why this matters}

**Security**

1. **REQ-NF-010**: {Security Requirement}
   - **Description**: {Specific security control}
   - **Implementation**: {High-level approach}
   - **Compliance**: {Relevant standards: OWASP, SOC2, etc.}
   - **Risk Mitigation**: {What threat this addresses}

**Accessibility**

1. **REQ-NF-020**: {Accessibility Requirement}
   - **Description**: {Specific WCAG criterion}
   - **Standard**: {WCAG 2.1 Level AA, etc.}
   - **Testing**: {How compliance will be verified}

**Additional Categories** (as relevant):
- Data Privacy & Compliance (GDPR, data retention)
- Reliability & Availability (uptime targets, error rates)
- Usability (learning curve, UX standards)
- Compatibility (browsers, devices, integrations)
- Scalability (concurrent users, data volume)
- Maintainability (code quality, documentation)

---

## Acceptance Criteria

### Feature-Level Acceptance

**Given/When/Then Format**:

**Scenario 1: {Primary Use Case}**
- **Given** {initial context/state}
- **When** {user action is performed}
- **Then** {expected outcome}
- **And** {additional outcome}

**Scenario 2: {Error Handling}**
- **Given** {error precondition}
- **When** {action that triggers error}
- **Then** {error is handled gracefully}
- **And** {user can recover}

---

## Out of Scope

### Explicitly Excluded

1. **{Feature/Capability}**
   - **Why**: {Reasoning - complexity, dependencies, prioritization}
   - **Future**: {Will this be addressed later? If so, when/why?}
   - **Workaround**: {How users can accomplish this currently, if applicable}

2. **{Feature/Capability}**
   {Follow same format}

---

### Alternative Approaches Rejected

**Alternative 1: {Approach Name}**
- **Description**: {Brief overview}
- **Why Rejected**: {Reasoning}

---

### Implementation Details

**Note**: Implementation approach should be handled by architecture/design agents in response to this PRD. This PRD focuses on WHAT needs to be built, not HOW to build it (beyond necessary NFRs and integration requirements).

---

## Success Metrics

### Primary Metrics

1. **{Metric Name}**
   - **What**: {What data point is tracked}
   - **Target**: {Specific goal}
   - **Timeline**: {When to achieve}
   - **Measurement**: {How to measure}

2. **{Metric Name}**
   {Follow same format}

---

### Secondary Metrics

- **{Metric}**: {Brief description and target}

---

## Dependencies & Integrations

### Dependencies

- **{System/Feature/Service}**: {Description of dependency}

### Integration Requirements

- **{External System}**: {What data/functionality is exchanged}

---

## Compliance & Security Considerations

{If applicable, note specific requirements}:
- **Regulatory**: {GDPR, HIPAA, SOC2, etc.}
- **Data Protection**: {Encryption, access controls}
- **Audit**: {Logging, audit trail requirements}

---

*Last Updated*: {YYYY-MM-DD}
```

## Quality Checklist

Before finalizing, verify:
- [ ] Problem statement is specific and user-focused
- [ ] User personas have clear goals and pain points
- [ ] User stories use proper format: "As a... I want... so that..."
- [ ] All must-have stories have acceptance criteria
- [ ] Functional requirements are testable and implementation-agnostic
- [ ] Non-functional requirements include specific, measurable targets
- [ ] Out of scope section prevents ambiguity
- [ ] Success metrics are measurable and time-bound
- [ ] Links to epic documentation are correct
- [ ] No vague language (e.g., "fast", "user-friendly" without specifics)
