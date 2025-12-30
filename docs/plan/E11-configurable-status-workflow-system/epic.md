---
epic_key: E11
title: Configurable Status Workflow System
description: Transform Shark's task status system from hardcoded linear progression to flexible, configuration-driven workflow engine supporting multi-agent collaboration and complex development processes
---

# Configurable Status Workflow System

**Epic Key**: E11

---

## Goal

### Problem
Currently, Shark uses a hardcoded, linear task status progression (`todo → in_progress → ready_for_review → completed`, with `blocked` as a side branch). This rigid model prevents multi-agent AI workflows where different agent types (business analysts, developers, QA, tech leads) need distinct workflow phases. AI agents cannot self-organize around workflow states, teams cannot adapt Shark to their existing processes (Kanban, GitFlow, custom workflows), and there's no support for backward transitions (e.g., QA finding bugs and sending tasks back to development). This limitation blocks Shark's evolution as an AI-native development tool and prevents adoption by teams with established workflows.

### Solution
Transform the task status system into a **configuration-driven workflow engine** where status transitions are defined in `.sharkconfig.json`. The new system will validate transitions against the configured workflow, support bidirectional state changes, enable agent-targeted queries by workflow phase, and provide migration tooling to transition existing projects safely. Teams can define their own statuses and valid transitions while Shark enforces the rules, with a `--force` flag for exceptional cases.

### Impact
- **Enable multi-agent coordination**: AI agents can query tasks by workflow phase (planning, development, review, QA) reducing time spent filtering through irrelevant tasks by 60%
- **Increase workflow flexibility**: Teams can adapt Shark to match their process (Kanban, Scrum, GitFlow) rather than changing their process to match Shark
- **Improve quality gates**: Explicit QA and approval stages reduce defect leakage to production by ~30%
- **Support complex workflows**: Backward transitions (QA → Dev, Approval → QA) enable real-world development processes

---

## Business Value

**Rating**: High

This feature directly enables Shark's strategic vision as an **AI-native development platform**. By supporting multi-agent workflows, Shark differentiates from traditional task managers that assume single-threaded, human-driven processes. The configuration-driven approach positions Shark for enterprise adoption where teams require workflow customization, and establishes the foundation for future workflow automation and analytics capabilities.

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

**Primary Users**: AI Development Agents (Business Analyst, Developer, QA, Tech Lead), Human Development Teams, Project Managers

**Key Features**:
- Configuration-driven workflow definition in `.sharkconfig.json`
- Bidirectional status transitions (forward, backward, lateral)
- Agent-targeted task queries by workflow phase and status metadata
- Workflow validation with `--force` escape hatch for emergencies
- Backward compatible default workflow for existing projects

**Success Criteria**:
- 40% of new projects define custom workflows within 30 days
- <5% of status updates require `--force` flag (indicates workflow matches actual process)
- Workflow validation adds <20ms overhead to status updates (P95)
- Backward compatible: existing projects work unchanged with default workflow

**Timeline**:
- Phase 1 (Core Infrastructure): Weeks 1-2
- Phase 2 (CLI Commands): Weeks 3-4
- Phase 3 (Agent UX): Week 5
- Phase 4 (Visualization): Week 6 (optional)

---

*Last Updated*: 2025-12-29
