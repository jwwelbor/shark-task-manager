# User Personas

**Epic**: [Bug and Change-Card Management System](./epic.md)

---

## Overview

This epic serves three distinct user types who interact with bugs and change-cards at different stages of the lifecycle. Developers create and fix bugs; QA engineers triage, verify fixes, and manage quality; product owners approve change-cards and make prioritization decisions.

---

## Primary Personas

### Persona 1: Developer (AI Agent or Human)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Software developer or AI coding agent working on Shark-managed projects
- **Experience Level**: Familiar with Shark CLI commands; uses `shark task`, `shark status`, and `shark get` daily
- **Key Characteristics**:
  - Discovers bugs during development, code review, or while running tests
  - Wants to report bugs quickly without interrupting workflow (under 30 seconds)
  - Needs to see bugs assigned to them and understand reproduction steps before fixing
  - Proposes small enhancements (change-cards) when they notice improvement opportunities during development

**Goals Related to This Epic**:
1. Report bugs immediately when discovered, with enough context for triage, without leaving the terminal
2. View bugs assigned for fixing with clear reproduction steps, severity, and linked context
3. Propose small enhancements as change-cards when full epic/feature decomposition is overkill

**Pain Points This Epic Addresses**:
- Currently must track bugs in external tools or ad-hoc notes, losing context and traceability
- Small enhancement ideas get forgotten because there is no lightweight capture mechanism
- Cannot see the relationship between a bug and the feature/task where it was found

**Success Looks Like**:
A developer discovers a bug, runs `shark bug create "Description" --severity=high --link=E07-F01`, and the bug is immediately visible in dashboards, triageable by QA, and linked to the relevant feature. The entire interaction takes under 30 seconds.

---

### Persona 2: QA Engineer

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Quality assurance engineer or testing specialist responsible for verifying software quality
- **Experience Level**: Familiar with Shark CLI; focused on status workflows, test results, and defect management
- **Key Characteristics**:
  - Triages incoming bug reports to assign severity and routing
  - Verifies bug fixes before marking them resolved
  - Tracks bug metrics (open count, resolution rate, severity distribution) to assess quality trends
  - Needs to identify duplicate bug reports efficiently

**Goals Related to This Epic**:
1. Triage incoming bugs by setting severity, assigning to developers, and adding notes
2. Verify fixes by reviewing the bug context, testing the fix, and advancing the bug to resolved
3. Monitor bug metrics to identify quality trends and areas needing attention

**Pain Points This Epic Addresses**:
- No structured triage workflow -- bugs arrive through informal channels with inconsistent information
- Cannot track verification status separately from fix status (was the fix actually tested?)
- No way to see bug counts, resolution times, or severity distribution in Shark dashboards

**Success Looks Like**:
A QA engineer runs `shark bug list --status=reported` to see new bugs, triages each with `shark bug triage B001 --severity=medium`, and later verifies fixes by advancing bugs through the verification step. Bug metrics are visible in `shark analytics`.

---

### Persona 3: Product Owner

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Product owner or team lead responsible for prioritization and approval decisions
- **Experience Level**: Uses Shark primarily through `shark status`, `shark list`, and `shark analytics` for overview and decision-making
- **Key Characteristics**:
  - Approves or declines change-card proposals based on business value and capacity
  - Reviews the balance between planned work (features) and reactive work (bugs/change-cards)
  - Needs high-level visibility without diving into implementation details
  - Makes decisions about whether a change-card should be promoted to a full feature

**Goals Related to This Epic**:
1. Review and approve/decline proposed change-cards efficiently
2. Understand the ratio of planned work versus reactive work (bugs + change-cards) to make staffing and prioritization decisions
3. Identify when a change-card has grown in scope and should be promoted to a feature

**Pain Points This Epic Addresses**:
- No visibility into the volume of small enhancement requests across the project
- Cannot distinguish between feature work and bug-fix work in current dashboards
- Change requests arrive informally and are not tracked, making capacity planning difficult

**Success Looks Like**:
A product owner runs `shark change list --status=proposed` to review pending change-cards, approves high-value ones with `shark change approve C001`, and checks `shark analytics` to see the planned vs. reactive work balance before sprint planning.

---

## Secondary Personas

- **CI/CD Pipeline Agent**: Automated systems that may create bugs programmatically when test suites fail or monitoring detects regressions. Interacts exclusively through CLI with `--json` output mode.

---

## Persona Validation Notes

- These personas are derived from the existing Shark user base as documented in the E15/E17 epics and the advanced workflow profile, which already defines agent types including `developer`, `qa`, and `product_owner`.
- Confidence level is high for Developer and QA Engineer personas, as they map directly to existing agent types. Product Owner persona is moderate confidence, as Shark's current usage is primarily developer-focused.
- The CI/CD Pipeline Agent secondary persona is based on the E12 design's emphasis on automated bug reporting, which remains a valid use case.

---

*See also*: [User Journeys](./user-journeys.md)
