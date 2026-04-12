---
feature_key: E27-F04-shark-web-cli-command-browser-launch-entry-point
epic_key: E27
title: shark web CLI Command - Browser Launch Entry Point
description: Add internal/cli/commands/web.go registering the `shark web [--port N] [--no-open]` Cobra command — a thin wrapper that starts the viewer server in-process, picks a free port starting at 7777, prints the URL, and opens the browser with xdg-open (Linux) or open (macOS).
---

# shark web CLI Command - Browser Launch Entry Point

**Feature Key**: E27-F04-shark-web-cli-command-browser-launch-entry-point
**Execution Order**: 4

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md)

---

## Description

This feature adds `internal/cli/commands/web.go`, a thin Cobra command that registers `shark web` as the user-facing entry point for the status viewer. The command accepts `--port N` (default 7777, auto-increments to find a free port if in use) and `--no-open` (skip browser launch). It starts the viewer server **in-process** — not as a shell-out to a separate binary — by calling the same `startServer(addr, db)` function assembled in E27-F05. After the server is listening, it prints the URL to stdout and invokes `xdg-open` on Linux or `open` on macOS to open the browser. The command follows the standard thin-wrapper pattern from `internal/cli/commands/`: argument parsing, service call (server startup), output formatting (URL print). Relies on E27-F05 (server wiring) providing the callable `startServer` entry point, which itself depends on E27-F01 (dbinit) for DB initialization. No new Go dependencies are introduced.

---

## Dependencies

- **E27-F05** (Server Wiring) — must expose a callable `startServer` function for the CLI to invoke

## Integration Points

- `internal/cli/root.go` — command auto-registers via `init()`
- Inherits project root auto-detection and `cli.GetDB` conventions from the CLI framework

---

*Last Updated*: 2026-04-11

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

*Last Updated*: 2026-04-11
