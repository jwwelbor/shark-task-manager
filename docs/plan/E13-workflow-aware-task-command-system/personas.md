# User Personas

**Epic**: [Workflow-Aware Task Command System](./epic.md)

---

## Overview

This document defines the primary user personas for the workflow-aware task command system. These personas represent distinct usage patterns ranging from fully automated AI orchestration to human workflow customization.

The phase-aware command model must serve both AI agents (programmatic, workflow-agnostic) and humans (interactive, context-aware) while supporting arbitrary workflow configurations.

---

## Primary Personas

### Persona 1: Atlas (AI Orchestrator Agent)

**Reference**: Defined for this epic (AI system persona)

**Profile**:
- **Role/Title**: "Autonomous AI task orchestrator managing multi-agent development team"
- **Technical Proficiency**: Programmatic API consumer, executes shark CLI commands via subprocess
- **Key Characteristics**:
  - Runs continuously in background (systemd service)
  - Polls shark database every 30 seconds for ready tasks
  - Makes decisions based on workflow config, dependencies, agent availability
  - No user interface - operates purely through shark CLI JSON output
  - Manages multiple concurrent AI agents (Claude, GPT-4, local LLMs)

**Goals Related to This Epic**:
1. Query tasks by workflow phase and agent type without hardcoded status assumptions
2. Claim tasks for agents, track work sessions automatically
3. Advance tasks to next workflow phase based on agent completion
4. Handle task rejection/rework with clear backward transitions
5. Adapt to any workflow configuration without code changes

**Pain Points This Epic Addresses**:
- **Hardcoded workflow assumptions**: Current `start` command assumes `in_progress` status, but workflow has `in_development`, `in_qa`, etc.
- **Ambiguous completion semantics**: Can't distinguish "developer finished coding" from "QA approved" from "task complete"
- **No phase-based queries**: Can't query "ready_for_development" tasks for backend agents vs. "ready_for_qa" for QA agents
- **Brittle agent code**: Every workflow change requires rewriting agent logic

**Success Looks Like**:
Atlas polls shark database, finds task T-E07-F20-001 with status `ready_for_development`. It queries `shark task next --agent=backend --json`, gets the task, calls `shark task claim T-E07-F20-001 --agent=backend` (transitions to `in_development`), spawns Claude agent. Agent completes work, Atlas calls `shark task finish T-E07-F20-001` (transitions to `ready_for_code_review`). Next cycle, Atlas assigns to tech-lead agent. All transitions follow workflow config with zero hardcoded statuses.

---

### Persona 2: Sarah (Product Manager / Scrum Master)

**Reference**: Adapted from /docs/personas/product-manager.md (if exists), otherwise defined here

**Profile**:
- **Role/Title**: "Product Manager at software company using shark for task management"
- **Experience Level**: "5+ years in product, moderate CLI proficiency, daily shark user"
- **Key Characteristics**:
  - Manages sprint planning and work assignment
  - Reviews task completion across workflow phases
  - Monitors team velocity and blockers
  - Creates epics, features, and tasks manually
  - Prefers categorized commands for faster discovery

**Goals Related to This Epic**:
1. Quickly assign work to team members based on current workflow phase
2. Understand which tasks are ready for which agent type/role
3. Track task progress across all workflow phases (not just development)
4. Approve final deliverables before marking completed

**Pain Points This Epic Addresses**:
- **Too many similar commands**: Confused by overlap between `complete`, `approve`, `next-status` - which one to use when?
- **Flat command list**: Scanning 25 alphabetically-sorted commands is cognitively taxing
- **Workflow mismatch**: Team uses custom "refinement → development → code review → QA → approval" workflow, but commands assume different flow
- **No feature-level queries**: Has to manually check tasks in each feature to assign work

**Success Looks Like**:
Sarah runs `shark feature next E07 --agent=backend` and gets F20 as next feature needing backend work. She runs `shark task next E07 F20 --agent=backend` to see which task is ready. Task is in `ready_for_development`, so she assigns it to developer. Later, developer finishes and runs `shark task finish T-E07-F20-001`, moving it to `ready_for_code_review`. Sarah approves as final reviewer with same `finish` command (context-dependent), advancing to `completed`. Clear semantics at each phase.

---

### Persona 3: Dev (Software Developer)

**Reference**: Adapted from /docs/personas/developer.md (if exists)

**Profile**:
- **Role/Title**: "Backend developer implementing features using shark task tracking"
- **Experience Level**: "3+ years development, comfortable with CLI, uses shark daily"
- **Key Characteristics**:
  - Picks up tasks from ready queue
  - Works on assigned tasks until complete
  - Sometimes realizes task isn't ready (missing specs)
  - Adds progress notes during implementation
  - Expects clear task handoff to next phase

**Goals Related to This Epic**:
1. Claim a task and start work session with single command
2. Mark task done when implementation complete (hand to QA/reviewer)
3. Send task back if requirements unclear
4. Understand which phase task is in and what comes next

**Pain Points This Epic Addresses**:
- **Confusing "complete" semantics**: Unsure if `complete` means "done with code" or "fully finished"
- **Can't reject task easily**: `reopen` implies QA rejection, not "this isn't ready for development"
- **No session tracking**: Doesn't know when they started task for time tracking
- **Status transitions unclear**: What happens after running `complete`? Goes to review? QA? Completed?

**Success Looks Like**:
Dev runs `shark task claim T-E07-F20-001 --agent=backend` to start work. Command transitions from `ready_for_development` to `in_development` and logs start time. Dev implements feature, adds notes. Runs `shark task finish T-E07-F20-001 --notes="API endpoints complete"` when done. System reads workflow config, sees next phase is `code_review`, transitions to `ready_for_code_review`. Clear handoff to next agent. If Dev realizes acceptance criteria are missing, runs `shark task reject T-E07-F20-001 --reason="Missing AC" --to=in_refinement` to send back to BA.

---

## Secondary Personas

- **Workflow Administrator (Alex)**: Defines custom workflow configurations in `.sharkconfig.json`. Needs commands to work with any workflow without code changes. This epic enables arbitrary workflow phases without breaking commands.

- **QA Engineer (Tessa)**: Similar to Dev, but works in QA phase. Uses same `claim → finish → reject` pattern but in different workflow phase. Benefits from consistent command semantics.

- **Tech Lead (Marcus)**: Conducts code reviews. Claims tasks from `ready_for_code_review`, approves or rejects back to development. Needs clear rejection semantics with reasons.

---

## Persona Validation Notes

**Data Sources**:
- AI Orchestrator Architecture document (Atlas persona)
- Workflow configuration analysis (real .sharkconfig.json)
- Command usage patterns from idea I-2026-01-09-02
- User feedback on command confusion

**Confidence Level**: High - personas directly derived from actual system architecture and user pain points

**Assumptions to Validate**:
- Do most teams use custom workflows or stick with defaults?
- What percentage of tasks are human-assigned vs. AI-orchestrated?
- How frequently do tasks get rejected backward through workflow?

---

*See also*: [User Journeys](./user-journeys.md)
