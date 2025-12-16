# Epic Template Structure

This document defines the complete structure for Epic-level PRDs. An epic consists of 6 interconnected files.

## File 1: `epic.md` (Main Index)

**Purpose**: Serves as the navigable entry point and executive summary.

```markdown
# {Epic Name}

**Epic Key**: E##-{epic-slug}

---

## Goal

### Problem
{3-5 sentences describing the user problem or business need. Be specific about who experiences this problem and why it matters.}

### Solution
{Explain the high-level approach to solving the problem. Focus on the "what" and "why," not the "how."}

### Impact
{Define 2-4 expected outcomes with specific metrics where possible.}

**Example**:
- Increase user engagement by 25% within Q3
- Reduce support tickets related to X by 40%
- Improve feature discoverability, measured by 50% increase in feature adoption

---

## Business Value

**Rating**: [High | Medium | Low]

{2-3 sentence justification considering:
- Revenue impact (direct or indirect)
- User satisfaction and retention
- Competitive positioning
- Strategic alignment with company goals
- Risk mitigation}

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - Target user profiles and characteristics
- **[User Journeys](./user-journeys.md)** - High-level workflows and interaction patterns
- **[Requirements](./requirements.md)** - Functional and non-functional requirements
- **[Success Metrics](./success-metrics.md)** - KPIs and measurement framework
- **[Scope Boundaries](./scope.md)** - Out of scope items and future considerations

---

## Quick Reference

**Primary Users**: {List 1-3 key persona names/roles}

**Key Features**:
- {Bullet list of 3-5 most important capabilities}

**Success Criteria**:
- {Top 2-3 metrics that define success}

**Timeline**: {If applicable, note any critical dates or phases}

---

*Last Updated*: {YYYY-MM-DD}
```

## File 2: `personas.md`

**Purpose**: Detailed user personas that provide deep context.

```markdown
# User Personas

**Epic**: [{Epic Name}](./epic.md)

---

## Overview

This document defines the primary user personas for this epic.

{1-2 sentences on why these specific personas matter for this epic.}

---

## Primary Personas

### Persona 1: {Persona Name/Role}

**Reference**: {If exists in /docs/personas/, link to it. Otherwise, mark as "Defined for this epic"}

**Profile**:
- **Role/Title**: {e.g., "Marketing Manager at mid-size B2B SaaS company"}
- **Experience Level**: {e.g., "3-5 years in role, moderate technical proficiency"}
- **Key Characteristics**:
  - {Characteristic 1 relevant to this epic}
  - {Characteristic 2}
  - {Characteristic 3}

**Goals Related to This Epic**:
1. {Specific goal 1}
2. {Specific goal 2}
3. {Specific goal 3}

**Pain Points This Epic Addresses**:
- {Pain point 1 with context}
- {Pain point 2 with context}
- {Pain point 3 with context}

**Success Looks Like**:
{2-3 sentences describing what success means from this persona's perspective}

---

{Repeat for Persona 2 and Persona 3 as needed}

---

## Secondary Personas

{If applicable, briefly list 1-2 secondary personas who are indirectly affected}

- **{Persona Name}**: {1 sentence on their relationship to this epic}

---

## Persona Validation Notes

{2-3 sentences on:
- Data sources used (user research, interviews, analytics)
- Confidence level in persona accuracy
- Any assumptions that need validation}

---

*See also*: [User Journeys](./user-journeys.md)
```

## File 3: `user-journeys.md`

**Purpose**: Step-by-step user workflows.

```markdown
# User Journeys

**Epic**: [{Epic Name}](./epic.md)

---

## Overview

This document maps the key user workflows enabled or improved by this epic.

---

## Journey 1: {Journey Name}

**Persona**: {Primary persona name}

**Goal**: {What the user is trying to accomplish}

**Preconditions**:
- {Required state or access before starting}

### Happy Path

1. **{Step 1 Title}**
   - User action: {What the user does}
   - System response: {How the system responds}
   - Expected outcome: {What the user sees/experiences}

2. **{Step 2 Title}**
   - User action: {What the user does}
   - System response: {How the system responds}
   - Expected outcome: {What the user sees/experiences}

{Continue for 5-10 steps as needed}

**Success Outcome**: {How the user knows they've successfully completed the journey}

### Alternative Paths

**Alt Path A: {Scenario Name}**
- **Trigger**: {What causes this alternate path}
- **Branch Point**: After Step {#}
- **Flow**:
  1. {Alternate step 1}
  2. {Alternate step 2}
- **Outcome**: {How this path resolves}

### Critical Decision Points

- **Decision at Step {#}**: {What choice the user makes and why it matters}

---

{Repeat for Journey 2, Journey 3, etc.}

---

*See also*: [Requirements](./requirements.md)
```

## File 4: `requirements.md`

**Purpose**: Comprehensive catalog of functional and non-functional requirements.

```markdown
# Requirements

**Epic**: [{Epic Name}](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for this epic.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### {Category 1: e.g., Authentication & Authorization}

**REQ-F-001**: {Requirement Title}
- **Description**: {Clear, specific, testable requirement statement}
- **User Story**: As a {persona}, I want to {action} so that {benefit}
- **Acceptance Criteria**:
  - [ ] {Specific testable criterion 1}
  - [ ] {Specific testable criterion 2}
- **Related Journey**: {Link to specific journey and step}

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: {Performance Requirement}
- **Description**: {Specific performance target}
- **Measurement**: {How it will be measured}
- **Target**: {Quantitative threshold}
- **Justification**: {Why this target matters}

### Security

**REQ-NF-010**: {Security Requirement}
- **Description**: {Specific security control or standard}
- **Implementation**: {High-level approach}
- **Compliance**: {Any relevant standards}
- **Risk Mitigation**: {What threat this addresses}

### Accessibility

**REQ-NF-020**: {Accessibility Requirement}
- **Description**: {Specific WCAG criterion}
- **Standard**: {WCAG 2.1 Level AA, etc.}
- **Testing**: {How compliance will be verified}

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
```

## File 5: `success-metrics.md`

**Purpose**: Defines measurable KPIs and evaluation framework.

```markdown
# Success Metrics

**Epic**: [{Epic Name}](./epic.md)

---

## Overview

This document defines the Key Performance Indicators (KPIs) that will measure the success of this epic.

**Measurement Timeline**: {e.g., "Initial metrics at 2 weeks post-launch, full evaluation at 3 months"}

---

## Primary Success Metrics

### Metric 1: {Metric Name}

**Type**: [Leading | Lagging]

**What We're Measuring**:
{Clear description of what data point(s) will be tracked}

**How We'll Measure**:
- **Data Source**: {e.g., Google Analytics, internal database}
- **Calculation Method**: {Formula or query approach}
- **Measurement Frequency**: {Real-time, daily, weekly, monthly}

**Success Criteria**:
- **Baseline**: {Current state measurement}
- **Target**: {Specific goal}
- **Timeline**: {When target should be achieved}
- **Minimum Viable**: {Threshold for success}

**Relates To**:
- **Requirement(s)**: {Link to specific requirements}
- **User Journey**: {Which journey success this measures}
- **Business Value**: {How this ties to business value}

---

## Success Criteria Summary

The epic is considered **successful** if:

1. {Primary metric 1 meets or exceeds target}
2. {Primary metric 2 meets or exceeds target}
3. {No critical regressions in existing metrics}

---

*See also*: [Requirements](./requirements.md)
```

## File 6: `scope.md`

**Purpose**: Explicitly defines boundaries to prevent scope creep.

```markdown
# Scope Boundaries

**Epic**: [{Epic Name}](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in this epic.

---

## Out of Scope

### Explicitly Excluded Features

**1. {Feature/Capability Name}**
- **Why It's Out of Scope**: {Reasoning}
- **Future Consideration**: {Will this be addressed later?}
- **Workaround**: {If applicable, current solution}

---

### Edge Cases & Scenarios Not Covered

**1. {Edge Case Description}**
- **Impact**: {How significant is this limitation?}
- **Rationale**: {Why we're not addressing it now}
- **Mitigation**: {How we'll handle if it occurs}

---

## Alternative Approaches Considered But Rejected

**Alternative 1: {Approach Name}**
- **Description**: {Brief overview}
- **Pros**: {What made this appealing}
- **Cons**: {Why it was rejected}
- **Decision Rationale**: {Final reasoning}

---

## Future Epic Candidates

{Features/capabilities that are natural follow-ons}

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| {Epic idea 1} | [High\|Medium\|Low] | {Depends on this epic} |

---

*See also*: [Requirements](./requirements.md)
```
