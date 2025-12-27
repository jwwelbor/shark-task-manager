# User Personas

**Epic**: [Advanced Task Intelligence & Context Management](./epic.md)

---

## Overview

This epic serves three primary user personas with distinct but overlapping needs around task context, completion tracking, and knowledge discovery.

---

## Primary Personas

### Persona 1: AI Development Agent (Claude Code)

**Reference**: Defined for this epic (represents autonomous AI agents executing development work)

**Profile**:
- **Role/Title**: AI-powered development agent performing backend, frontend, and test implementation
- **Experience Level**: Expert technical knowledge but limited session continuity (must resume work across conversation boundaries)
- **Key Characteristics**:
  - Executes complex multi-step tasks requiring decisions, research, and iteration
  - Works asynchronously with frequent pause/resume cycles as conversations end
  - Benefits from structured context to avoid re-analyzing codebases on resume
  - Produces detailed implementation artifacts (code, tests, documentation)

**Goals Related to This Epic**:
1. Record implementation decisions and rationale during task execution
2. Capture completion metadata (files modified, tests added, verification status)
3. Resume paused tasks with full context of what was done and what remains
4. Discover related tasks and dependencies before starting work
5. Track acceptance criteria progress to ensure nothing is missed

**Pain Points This Epic Addresses**:
- **Context Loss on Resume**: When resuming a paused task, agents must re-read files and rediscover decisions already made
- **Invisible Completion Details**: Current system doesn't capture what files were modified, tests added, or verification performed
- **Dependency Blindness**: No way to discover that task X depends on or relates to task Y
- **Incomplete Verification**: No structured tracking of acceptance criteria, leading to missed requirements

**Success Looks Like**:
AI agents can pause work at any point, resume hours or days later with complete context (including decisions made, blockers encountered, and remaining steps), verify all acceptance criteria are met, and discover related work without manual searching.

---

### Persona 2: Human Developer (Technical Lead)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Senior developer or tech lead managing development workflow
- **Experience Level**: 5+ years development experience, responsible for code quality and task review
- **Key Characteristics**:
  - Reviews task completions from AI agents and human developers
  - Needs to quickly understand what was done and verify quality
  - Troubleshoots blockers and investigates related implementations
  - Plans task dependencies and sequences work

**Goals Related to This Epic**:
1. Quickly understand what was accomplished in a completed task
2. Verify acceptance criteria were met before approving tasks
3. Search for tasks that modified specific files or implemented similar features
4. Understand task dependencies and relationships to plan work sequences
5. Identify patterns in blockers or implementation approaches

**Pain Points This Epic Addresses**:
- **Opaque Completions**: Can't easily see what files were changed, tests added, or verification performed
- **Manual Criteria Checking**: Must manually compare implementation to acceptance criteria
- **Difficult Discovery**: No way to search "which tasks modified useTheme.ts" or "find tasks similar to this one"
- **Hidden Dependencies**: Task relationships are implicit, making planning difficult

**Success Looks Like**:
Tech leads can review a completed task and immediately see all files modified, tests added, verification status, and criteria met. They can search for related work by file, technology, or pattern. They can visualize task dependencies before sequencing work.

---

### Persona 3: Product Manager

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Product manager overseeing feature development and release planning
- **Experience Level**: 3+ years product management, focuses on delivery and quality metrics
- **Key Characteristics**:
  - Tracks feature completion and identifies blockers
  - Measures team velocity and quality metrics
  - Needs visibility into what's blocking progress
  - Reports on acceptance criteria completion for stakeholder confidence

**Goals Related to This Epic**:
1. Track acceptance criteria progress across all tasks in a feature
2. Identify blocked tasks and understand what's causing delays
3. Measure work session patterns to improve estimates
4. Report on verification status (how many tasks are verified vs. unverified)
5. Understand task relationships to manage scope and dependencies

**Pain Points This Epic Addresses**:
- **No Criteria Visibility**: Can't see "15/20 acceptance criteria met" for a feature
- **Hidden Blockers**: Blockers are buried in task history notes, not categorized
- **No Time Metrics**: Can't measure actual time spent vs. estimates
- **Relationship Opacity**: Can't see that 5 tasks are blocked waiting on 1 upstream task

**Success Looks Like**:
Product managers can view a feature dashboard showing acceptance criteria progress, identify all blocked tasks with categorized blocker types, analyze work session patterns for better estimation, and visualize task dependency graphs to manage scope and sequencing.

---

## Secondary Personas

- **QA Engineer**: Benefits from acceptance criteria tracking and verification status for targeted testing
- **Future AI Agents**: As more specialized agents join the workflow (frontend-agent, backend-agent), they'll use relationships and notes to coordinate work

---

## Persona Validation Notes

These personas are derived from real usage patterns observed during E13 (Dark Mode implementation) where AI agents encountered context loss on resume, tech leads struggled to verify completions, and product managers lacked visibility into criteria progress. The enhancement document at `/docs/plan/E07-enhancements/claude-enhancements.md` contains specific examples from actual tasks (T-E13-F05-002, T-E13-F05-003, T-E13-F02-004) that motivated these features.

---

*See also*: [User Journeys](./user-journeys.md)
