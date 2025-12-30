# User Personas

**Epic**: [Configurable Status Workflow System](./epic.md)

---

## Overview

This epic primarily serves **AI development agents** and secondarily serves **human development teams**. The personas reflect the multi-agent collaboration model that Shark enables.

---

## Primary Personas

### Persona 1: Business Analyst Agent

**Reference**: Defined for this epic (AI persona)

**Profile**:
- **Role/Title**: AI agent specializing in requirements analysis and task refinement
- **Experience Level**: Autonomous agent with domain knowledge in product requirements
- **Key Characteristics**:
  - Queries tasks in "planning" workflow phase
  - Needs to distinguish "draft" tasks from "ready for refinement" tasks
  - Cannot efficiently filter through 150+ mixed-status tasks
  - Requires clear handoff points to development agents

**Goals Related to This Epic**:
1. Query only tasks that need business analysis (status: `ready_for_refinement`)
2. Transition tasks to `in_refinement` to claim ownership and prevent duplicate work
3. Move completed specifications to `ready_for_development` for developer agents
4. Avoid wasting cycles on tasks that are already being worked by other agents

**Pain Points This Epic Addresses**:
- Currently no status distinction between "needs spec" and "ready to code" (both are just "todo")
- Cannot query by agent type or workflow phase - must manually scan all tasks
- No atomic claim mechanism - two agents might start refining the same task
- No backward transition if requirements change mid-development

**Success Looks Like**:
Business analyst agent runs `shark task next --agent=business-analyst` and immediately receives the highest-priority unrefined task, claims it atomically by moving to `in_refinement`, completes analysis, and hands off to developers with confidence that the status reflects actual state.

---

### Persona 2: Developer Agent

**Reference**: Defined for this epic (AI persona)

**Profile**:
- **Role/Title**: AI agent specializing in code implementation and TDD workflows
- **Experience Level**: Autonomous agent with programming capabilities
- **Key Characteristics**:
  - Only works on fully-specified tasks (no ambiguity tolerance)
  - Queries tasks in "development" workflow phase
  - Needs clear handoff from business analysts
  - Must be able to send tasks back to refinement if specs are incomplete

**Goals Related to This Epic**:
1. Query only tasks with completed specifications (status: `ready_for_development`)
2. Transition to `in_development` to signal work has started
3. Move completed implementation to `ready_for_review` for code review agents
4. Send incomplete specs back to `ready_for_refinement` without manual coordination

**Pain Points This Epic Addresses**:
- Currently no way to filter for "fully specified" vs "draft" tasks
- Cannot programmatically query by workflow phase
- No backward transition to request spec improvements
- Hardcoded workflow assumes linear progress (no rework loops)

**Success Looks Like**:
Developer agent queries `shark task list --status=ready_for_development --agent=developer`, receives only tasks with complete specifications, implements features, and either advances to review or sends back to refinement with clear status signals visible to all agents.

---

### Persona 3: QA Agent

**Reference**: Defined for this epic (AI persona)

**Profile**:
- **Role/Title**: AI agent specializing in quality assurance and testing
- **Experience Level**: Autonomous agent with testing capabilities
- **Key Characteristics**:
  - Works only on code-reviewed tasks (post-implementation)
  - Queries tasks in "qa" workflow phase
  - Needs ability to reject implementations that fail tests
  - Requires backward transition authority to send tasks back to development

**Goals Related to This Epic**:
1. Query tasks that passed code review (status: `ready_for_qa`)
2. Transition to `in_qa` to claim testing work
3. Either approve (move to `ready_for_approval`) or reject (move back to `in_development`)
4. Record rejection reason for developers without out-of-band communication

**Pain Points This Epic Addresses**:
- Currently no QA-specific status (everything is just "ready for review")
- No backward transition from QA to development when bugs are found
- Cannot query by QA workflow phase
- No audit trail of QA → Dev rework loops

**Success Looks Like**:
QA agent finds 30% of tasks have bugs, transitions them to `in_development` with notes "Safari login failure", developers see the rejected tasks in their queue automatically, fix issues, and resubmit through the workflow - all without manual project management overhead.

---

### Persona 4: Tech Lead Agent

**Reference**: Defined for this epic (AI persona)

**Profile**:
- **Role/Title**: AI agent specializing in code review and architectural oversight
- **Experience Level**: Autonomous agent with advanced code analysis capabilities
- **Key Characteristics**:
  - Reviews code after development, before QA
  - Queries tasks in "review" workflow phase
  - Needs authority to send tasks back to refinement if architecture is wrong
  - Requires visibility into workflow bottlenecks

**Goals Related to This Epic**:
1. Query tasks awaiting code review (status: `ready_for_review`)
2. Approve well-implemented tasks (advance to `ready_for_qa`)
3. Request specification improvements for architectural issues (back to `ready_for_refinement`)
4. Request code fixes for implementation issues (back to `in_development`)

**Pain Points This Epic Addresses**:
- Cannot distinguish between "needs review" and "in review" (both show as "ready for review")
- No multi-path rejection (can only send back to dev, not to refinement)
- Cannot query review queue by workflow phase
- No analytics on review bottlenecks

**Success Looks Like**:
Tech lead agent processes review queue, sends 10% of tasks back to refinement for arch changes, sends 20% back to development for bug fixes, and approves 70% to QA - all with clear status signals and audit trail showing why tasks were rejected.

---

## Secondary Personas

### Human Project Manager
Needs visibility into workflow bottlenecks (which phase has the most tasks?) and progress tracking. Benefits from status metadata (colored output, phase grouping) and workflow analytics (future enhancement).

### Human Developer
Occasionally needs to override workflow for hotfixes or exceptions. Benefits from `--force` flag and clear error messages when attempting invalid transitions.

---

## Persona Validation Notes

These AI agent personas are derived from the multi-agent SDLC workflows documented in E10 (Advanced Task Intelligence) and align with Shark's strategic vision as an AI-native development platform. Human personas are based on observed Shark usage patterns. Confidence level: **High** for AI agents (directly supports E10 features), **Medium** for human users (need real-world usage data after launch).

---

*See also*: [User Journeys](./user-journeys.md)
