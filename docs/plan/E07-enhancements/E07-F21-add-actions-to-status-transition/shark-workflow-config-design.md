# Shark Workflow Configuration Design

**Version:** 1.0
**Date:** 2026-01-13
**Status:** Architecture Design
**Author:** Architect Agent

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Problem Statement](#2-problem-statement)
3. [Design Principles](#3-design-principles)
4. [Workflow Configuration Schema](#4-workflow-configuration-schema)
5. [Storage Location](#5-storage-location)
6. [Configuration Examples](#6-configuration-examples)
7. [Orchestrator Integration](#7-orchestrator-integration)
8. [Migration Path](#8-migration-path)
9. [Benefits Analysis](#9-benefits-analysis)
10. [Implementation Guidance](#10-implementation-guidance)

---

## 1. Executive Summary

This document specifies a **declarative workflow configuration system** for Shark that externalizes workflow-specific knowledge from the orchestrator. The configuration defines:

- **Agent mapping**: Which agent type handles each workflow status
- **Stage instructions**: Prompts and context for agents at each stage
- **Transition rules**: Valid status transitions and their prerequisites
- **Constraints**: Per-stage concurrency limits, timeouts, and requirements
- **Context propagation**: What information flows between stages

**Key Benefits:**
- Workflow knowledge lives with the workflow definition (DRY principle)
- Workflows are data-driven and modifiable without code changes
- Multiple workflows supported without orchestrator changes
- Clear contract between Shark and orchestrator
- Easier testing and validation of workflow rules

**Recommendation:** Store workflow configs in **Shark's config directory** (`~/.config/shark/workflows/`) as YAML files, loaded by Shark CLI and queryable via new shark commands.

---

## 2. Problem Statement

### 2.1 Current State

The orchestrator currently has **hardcoded workflow knowledge**:

```yaml
# Hardcoded in orchestrator
status_to_agent_mapping:
  ready_for_refinement_ba: business-analyst
  ready_for_refinement_tech: architect
  ready_for_development: developer
  ready_for_code_review: tech-lead
  ready_for_qa: qa
  ready_for_approval: product-manager
```

**Problems with this approach:**

1. **Tight Coupling**: Orchestrator must know workflow-specific details
2. **Code Changes Required**: Adding new workflows requires orchestrator modification
3. **No Single Source of Truth**: Workflow definition is split between Shark (status values) and orchestrator (agent mapping)
4. **Hard to Validate**: Can't validate that a workflow config is complete/correct
5. **Testing Difficulty**: Can't test workflow logic in isolation
6. **Documentation Drift**: Workflow docs (workflow-custom.md) separate from implementation

### 2.2 Desired State

Workflow configuration should:

1. **Live in Shark**: Workflow knowledge stored with task state
2. **Be Declarative**: YAML/JSON files defining workflow behavior
3. **Be Queryable**: `shark workflow show wormwoodGM` displays config
4. **Be Validated**: Shark validates workflow completeness and correctness
5. **Be Versioned**: Track workflow definition changes over time
6. **Support Multiple Workflows**: Different projects can use different workflows

---

## 3. Design Principles

All workflow configuration design follows:

1. **Appropriate**: Use proven configuration patterns (YAML, declarative)
2. **Proven**: Based on workflow systems like GitHub Actions, Argo, Tekton
3. **Simple**: No unnecessary complexity, clear structure, easy to understand

Additional principles:

- **Declarative over Imperative**: Describe WHAT, not HOW
- **Self-Documenting**: Config should be readable as documentation
- **Fail Fast**: Validate config on load, reject invalid workflows
- **Extensible**: Easy to add new fields without breaking existing configs
- **Backward Compatible**: Support generic workflow as fallback

---

## 4. Workflow Configuration Schema

### 4.1 Top-Level Structure

```yaml
# Workflow configuration file: wormwoodGM.workflow.yml
workflow:
  name: wormwoodGM
  version: "1.0"
  description: "WormwoodGM custom workflow with BA → Arch → Dev → Review → QA → Approval stages"

  metadata:
    author: "Product Team"
    created: "2026-01-10"
    updated: "2026-01-13"
    project: "wormwoodGM"

  # Entry points: How tasks enter this workflow
  entry_points:
    - status: draft
      description: "Tasks created manually or via epic breakdown"
    - status: ready_for_development
      description: "Tasks imported from external design tools"

  # Terminal states: Where workflow ends
  terminal_states:
    - status: completed
      description: "Successfully delivered and approved"
    - status: cancelled
      description: "Abandoned or deprioritized"

  # Special states: Cross-phase states
  special_states:
    - status: blocked
      description: "Waiting on external dependency"
      allow_from: any  # Can transition to blocked from any state
    - status: on_hold
      description: "Paused by product decision"
      allow_from: any

  # Workflow stages: Linear progression
  stages: [...]  # See section 4.2

  # Status definitions: All valid statuses
  statuses: [...]  # See section 4.3

  # Transition rules: Valid status changes
  transitions: [...]  # See section 4.4

  # Global constraints
  constraints:
    max_concurrent_tasks: 100
    max_per_agent_type: {...}  # See section 4.5
    max_task_duration: "24h"
```

### 4.2 Stage Definitions

Each stage represents a phase in the workflow with specific agent responsibilities.

```yaml
stages:
  - name: business_analysis
    description: "Business requirements and acceptance criteria refinement"
    phase: planning
    color: cyan

    statuses:
      - ready_for_refinement_ba  # Queue status
      - in_refinement_ba         # Active status

    agent_type: business-analyst

    # Agent instructions for this stage
    agent_instructions:
      base_prompt: |
        You are the Business Analyst for this task.
        Your goal is to refine business requirements and define acceptance criteria.

        Review the task description and user stories.
        Clarify requirements with stakeholders if needed.
        Document clear acceptance criteria.
        Ensure requirements are testable and complete.

      skills_required:
        - specification-writing
        - shark-task-management
        - research

      context_to_include:
        - task_description
        - feature_context
        - epic_context
        - related_tasks
        - user_stories

      success_criteria:
        - "Acceptance criteria are documented"
        - "Business requirements are clear"
        - "Edge cases are identified"

      artifacts_to_create:
        - "F01-requirements.md"
        - "F02-acceptance-criteria.md"

    # Transition outcomes from this stage
    transitions:
      on_success:
        - to: ready_for_refinement_tech
          condition: "Needs technical design"
          default: true
        - to: ready_for_development
          condition: "No technical design needed (simple implementation)"

      on_failure:
        - to: draft
          condition: "Requirements unclear, needs rework"

      on_block:
        - to: blocked
          condition: "External dependency or decision needed"

    # Stage-specific constraints
    constraints:
      max_concurrent: 2
      timeout: "4h"
      retry_limit: 2

    # Progress tracking
    progress_indicators:
      - field: current_step
        values:
          - "Reviewing task requirements"
          - "Defining acceptance criteria"
          - "Documenting edge cases"
          - "Finalizing business requirements"

      - field: completion_percentage
        calculation: "completed_steps / total_steps"

  - name: technical_design
    description: "Technical architecture and API contract design"
    phase: planning
    color: blue

    statuses:
      - ready_for_refinement_tech
      - in_refinement_tech

    agent_type: architect

    agent_instructions:
      base_prompt: |
        You are the Architect for this task.
        Your goal is to design the technical solution.

        Review business requirements from the BA stage.
        Design API contracts, data models, and system flows.
        Document technical decisions and rationale.
        Ensure design is Appropriate, Proven, and Simple.

      skills_required:
        - architecture
        - specification-writing
        - shark-task-management

      context_to_include:
        - task_description
        - business_requirements  # From previous stage
        - acceptance_criteria    # From previous stage
        - technical_constraints
        - existing_architecture

      success_criteria:
        - "API contracts are defined"
        - "Data models are designed"
        - "Technical decisions are documented"

      artifacts_to_create:
        - "T01-api-contracts.md"
        - "T03-data-models.md"
        - "T06-system-flows.md"

    transitions:
      on_success:
        - to: ready_for_development
          default: true

      on_failure:
        - to: ready_for_refinement_ba
          condition: "Business requirements gaps found"
        - to: draft
          condition: "Major issues, complete rework needed"

      on_block:
        - to: blocked
          condition: "External dependency or decision needed"

    constraints:
      max_concurrent: 2
      timeout: "6h"
      retry_limit: 2

  - name: development
    description: "Code implementation following TDD"
    phase: development
    color: yellow

    statuses:
      - ready_for_development
      - in_development

    agent_type: developer

    agent_instructions:
      base_prompt: |
        You are the Developer for this task.
        Your goal is to implement the feature following TDD practices.

        Review technical specifications from the Architect.
        Write tests first, then implement to pass tests.
        Follow project coding standards and best practices.
        Document implementation decisions in shark notes.

      skills_required:
        - test-driven-development
        - implementation
        - shark-task-management

      context_to_include:
        - task_description
        - business_requirements
        - acceptance_criteria
        - api_contracts
        - data_models
        - coding_standards

      success_criteria:
        - "All tests passing"
        - "Implementation complete"
        - "Code follows standards"

      artifacts_to_create:
        - "Source code files"
        - "Test files"
        - "Implementation notes in shark"

    transitions:
      on_success:
        - to: ready_for_code_review
          default: true

      on_failure:
        - to: ready_for_refinement_ba
          condition: "Business requirements unclear"
        - to: ready_for_refinement_tech
          condition: "Technical design gaps found"

      on_block:
        - to: blocked
          condition: "External dependency needed"

    constraints:
      max_concurrent: 5
      timeout: "8h"
      retry_limit: 3

  - name: code_review
    description: "Code quality and standards review"
    phase: review
    color: magenta

    statuses:
      - ready_for_code_review
      - in_code_review

    agent_type: tech-lead

    agent_instructions:
      base_prompt: |
        You are the Tech Lead reviewing this code.
        Your goal is to ensure code quality and adherence to standards.

        Review implementation against technical specifications.
        Check for code quality, security, and performance issues.
        Verify tests are comprehensive and passing.
        Provide constructive feedback if changes needed.

      skills_required:
        - quality
        - shark-task-management

      context_to_include:
        - task_description
        - technical_specifications
        - implementation_code
        - test_results
        - coding_standards

      success_criteria:
        - "Code meets quality standards"
        - "Tests are comprehensive"
        - "No security vulnerabilities"

      artifacts_to_create:
        - "Code review notes in shark"

    transitions:
      on_success:
        - to: ready_for_qa
          default: true

      on_failure:
        - to: in_development
          condition: "Code issues found, needs fixes"
        - to: ready_for_refinement_tech
          condition: "Architectural issues found"

      on_block:
        - to: blocked
          condition: "Need architect consultation"

    constraints:
      max_concurrent: 2
      timeout: "2h"
      retry_limit: 1

  - name: quality_assurance
    description: "Functional testing and QA validation"
    phase: qa
    color: green

    statuses:
      - ready_for_qa
      - in_qa

    agent_type: qa

    agent_instructions:
      base_prompt: |
        You are the QA Engineer for this task.
        Your goal is to validate the implementation against acceptance criteria.

        Review acceptance criteria from BA stage.
        Execute comprehensive test plan.
        Verify edge cases and error handling.
        Document test results and any issues found.

      skills_required:
        - quality
        - shark-task-management

      context_to_include:
        - task_description
        - acceptance_criteria
        - test_plan
        - implementation_code

      success_criteria:
        - "All acceptance criteria met"
        - "No critical bugs found"
        - "Edge cases handled correctly"

      artifacts_to_create:
        - "Test results in shark"
        - "Bug reports (if any)"

    transitions:
      on_success:
        - to: ready_for_approval
          default: true

      on_failure:
        - to: in_development
          condition: "Bugs found, needs fixes"
        - to: ready_for_refinement_ba
          condition: "Requirements mismatch"

      on_block:
        - to: blocked
          condition: "Cannot test due to external dependency"

    constraints:
      max_concurrent: 2
      timeout: "4h"
      retry_limit: 2

  - name: approval
    description: "Final product approval"
    phase: approval
    color: purple

    statuses:
      - ready_for_approval
      - in_approval

    agent_type: product-manager

    agent_instructions:
      base_prompt: |
        You are the Product Manager approving this task.
        Your goal is to verify the implementation meets business needs.

        Review implementation against original requirements.
        Verify acceptance criteria are satisfied.
        Check that solution delivers business value.
        Approve for production or request changes.

      skills_required:
        - shark-task-management

      context_to_include:
        - task_description
        - business_requirements
        - acceptance_criteria
        - qa_results

      success_criteria:
        - "Meets business requirements"
        - "Delivers expected value"
        - "Ready for production"

      artifacts_to_create:
        - "Approval decision in shark"

    transitions:
      on_success:
        - to: completed
          default: true

      on_failure:
        - to: ready_for_qa
          condition: "Minor issues found"
        - to: ready_for_development
          condition: "Major rework needed"
        - to: ready_for_refinement_ba
          condition: "Requirements mismatch"

      on_block:
        - to: on_hold
          condition: "Strategic pause"

    constraints:
      max_concurrent: 1
      timeout: "2h"
      retry_limit: 1
```

### 4.3 Status Definitions

Complete catalog of all workflow statuses.

```yaml
statuses:
  # Entry point
  - name: draft
    description: "Task created, awaiting triage"
    phase: planning
    color: gray
    is_entry_point: true
    requires_agent: false

  # Business Analysis stage
  - name: ready_for_refinement_ba
    description: "Queued for business analysis"
    phase: planning
    color: cyan
    is_queue: true
    agent_type: business-analyst
    stage: business_analysis

  - name: in_refinement_ba
    description: "Business analyst actively refining requirements"
    phase: planning
    color: cyan
    is_active: true
    agent_type: business-analyst
    stage: business_analysis

  # Technical Design stage
  - name: ready_for_refinement_tech
    description: "Queued for technical design"
    phase: planning
    color: blue
    is_queue: true
    agent_type: architect
    stage: technical_design

  - name: in_refinement_tech
    description: "Architect actively designing solution"
    phase: planning
    color: blue
    is_active: true
    agent_type: architect
    stage: technical_design

  # Development stage
  - name: ready_for_development
    description: "Queued for implementation"
    phase: development
    color: yellow
    is_queue: true
    agent_type: developer
    stage: development

  - name: in_development
    description: "Developer actively implementing"
    phase: development
    color: yellow
    is_active: true
    agent_type: developer
    stage: development

  # Code Review stage
  - name: ready_for_code_review
    description: "Queued for code review"
    phase: review
    color: magenta
    is_queue: true
    agent_type: tech-lead
    stage: code_review

  - name: in_code_review
    description: "Tech lead actively reviewing code"
    phase: review
    color: magenta
    is_active: true
    agent_type: tech-lead
    stage: code_review

  # QA stage
  - name: ready_for_qa
    description: "Queued for quality assurance"
    phase: qa
    color: green
    is_queue: true
    agent_type: qa
    stage: quality_assurance

  - name: in_qa
    description: "QA engineer actively testing"
    phase: qa
    color: green
    is_active: true
    agent_type: qa
    stage: quality_assurance

  # Approval stage
  - name: ready_for_approval
    description: "Queued for final approval"
    phase: approval
    color: purple
    is_queue: true
    agent_type: product-manager
    stage: approval

  - name: in_approval
    description: "Product manager actively reviewing"
    phase: approval
    color: purple
    is_active: true
    agent_type: product-manager
    stage: approval

  # Terminal states
  - name: completed
    description: "Successfully delivered and approved"
    phase: done
    color: white
    is_terminal: true

  - name: cancelled
    description: "Abandoned or deprioritized"
    phase: done
    color: gray
    is_terminal: true

  # Special states
  - name: blocked
    description: "Waiting on external dependency"
    phase: any
    color: red
    is_special: true
    allow_from_any: true

  - name: on_hold
    description: "Temporarily paused"
    phase: any
    color: orange
    is_special: true
    allow_from_any: true
```

### 4.4 Transition Rules

Define valid status transitions and their conditions.

```yaml
transitions:
  # From draft
  - from: draft
    to: ready_for_refinement_ba
    description: "Triage to BA for requirements"
    authorized_by: [product-manager, business-analyst]

  - from: draft
    to: ready_for_refinement_tech
    description: "Triage to architect (skip BA)"
    authorized_by: [product-manager]
    conditions:
      - "Business requirements already clear"

  - from: draft
    to: cancelled
    description: "Task not needed"
    authorized_by: [product-manager, client]

  - from: draft
    to: on_hold
    description: "Deprioritize task"
    authorized_by: [product-manager]

  # From ready_for_refinement_ba
  - from: ready_for_refinement_ba
    to: in_refinement_ba
    description: "BA picks up task"
    authorized_by: [business-analyst]
    automatic: true  # Can be auto-assigned by orchestrator

  - from: ready_for_refinement_ba
    to: cancelled
    description: "Task no longer needed"
    authorized_by: [product-manager]

  - from: ready_for_refinement_ba
    to: on_hold
    description: "Pause refinement"
    authorized_by: [product-manager]

  # From in_refinement_ba
  - from: in_refinement_ba
    to: ready_for_refinement_tech
    description: "BA complete, needs technical design"
    authorized_by: [business-analyst]
    requires:
      - "Business requirements documented"
      - "Acceptance criteria defined"

  - from: in_refinement_ba
    to: ready_for_development
    description: "BA complete, skip technical design"
    authorized_by: [business-analyst]
    requires:
      - "Business requirements documented"
      - "Acceptance criteria defined"
    conditions:
      - "Simple implementation, no design needed"

  - from: in_refinement_ba
    to: draft
    description: "Requirements unclear, needs rework"
    authorized_by: [business-analyst]

  - from: in_refinement_ba
    to: blocked
    description: "External dependency needed"
    authorized_by: [business-analyst]
    requires:
      - "Blocker reason documented"

  # From ready_for_refinement_tech
  - from: ready_for_refinement_tech
    to: in_refinement_tech
    description: "Architect picks up task"
    authorized_by: [architect]
    automatic: true

  # From in_refinement_tech
  - from: in_refinement_tech
    to: ready_for_development
    description: "Technical design complete"
    authorized_by: [architect]
    requires:
      - "API contracts defined"
      - "Data models designed"
      - "Technical decisions documented"

  - from: in_refinement_tech
    to: ready_for_refinement_ba
    description: "Business requirements gaps found"
    authorized_by: [architect]

  - from: in_refinement_tech
    to: draft
    description: "Major issues, complete rework"
    authorized_by: [architect]

  - from: in_refinement_tech
    to: blocked
    description: "External dependency needed"
    authorized_by: [architect]
    requires:
      - "Blocker reason documented"

  # From ready_for_development
  - from: ready_for_development
    to: in_development
    description: "Developer picks up task"
    authorized_by: [developer]
    automatic: true

  - from: ready_for_development
    to: ready_for_refinement_ba
    description: "Business requirements unclear"
    authorized_by: [developer]

  - from: ready_for_development
    to: ready_for_refinement_tech
    description: "Technical design unclear"
    authorized_by: [developer]

  # From in_development
  - from: in_development
    to: ready_for_code_review
    description: "Implementation complete"
    authorized_by: [developer]
    requires:
      - "All tests passing"
      - "Implementation complete"
      - "Code follows standards"

  - from: in_development
    to: ready_for_refinement_ba
    description: "Business requirements gaps found"
    authorized_by: [developer]

  - from: in_development
    to: ready_for_refinement_tech
    description: "Technical design gaps found"
    authorized_by: [developer]

  - from: in_development
    to: blocked
    description: "External dependency needed"
    authorized_by: [developer]
    requires:
      - "Blocker reason documented"

  # From ready_for_code_review
  - from: ready_for_code_review
    to: in_code_review
    description: "Tech lead picks up review"
    authorized_by: [tech-lead]
    automatic: true

  - from: ready_for_code_review
    to: in_development
    description: "Developer withdraws for fixes"
    authorized_by: [developer]

  # From in_code_review
  - from: in_code_review
    to: ready_for_qa
    description: "Code review passes"
    authorized_by: [tech-lead]
    requires:
      - "Code meets quality standards"
      - "No security issues"

  - from: in_code_review
    to: in_development
    description: "Code review fails, needs fixes"
    authorized_by: [tech-lead]
    requires:
      - "Issues documented in shark notes"

  - from: in_code_review
    to: ready_for_refinement_ba
    description: "Requirements issues found"
    authorized_by: [tech-lead]

  - from: in_code_review
    to: ready_for_refinement_tech
    description: "Architectural issues found"
    authorized_by: [tech-lead]

  # From ready_for_qa
  - from: ready_for_qa
    to: in_qa
    description: "QA picks up testing"
    authorized_by: [qa]
    automatic: true

  # From in_qa
  - from: in_qa
    to: ready_for_approval
    description: "QA passes"
    authorized_by: [qa]
    requires:
      - "All acceptance criteria met"
      - "No critical bugs"
      - "Test results documented"

  - from: in_qa
    to: in_development
    description: "QA fails, bugs found"
    authorized_by: [qa]
    requires:
      - "Bugs documented in shark notes"

  - from: in_qa
    to: ready_for_refinement_ba
    description: "Requirements mismatch"
    authorized_by: [qa]

  - from: in_qa
    to: ready_for_refinement_tech
    description: "Design mismatch"
    authorized_by: [qa]

  - from: in_qa
    to: blocked
    description: "Cannot test due to external dependency"
    authorized_by: [qa]
    requires:
      - "Blocker reason documented"

  # From ready_for_approval
  - from: ready_for_approval
    to: in_approval
    description: "PM/Client picks up approval"
    authorized_by: [product-manager, client]
    automatic: true

  # From in_approval
  - from: in_approval
    to: completed
    description: "Approved for production"
    authorized_by: [product-manager, client]
    requires:
      - "Meets business requirements"
      - "Approval documented"

  - from: in_approval
    to: ready_for_qa
    description: "Minor issues found"
    authorized_by: [product-manager, client]

  - from: in_approval
    to: ready_for_development
    description: "Major rework needed"
    authorized_by: [product-manager, client]

  - from: in_approval
    to: ready_for_refinement_ba
    description: "Requirements mismatch"
    authorized_by: [product-manager, client]

  - from: in_approval
    to: on_hold
    description: "Strategic pause"
    authorized_by: [product-manager]

  # From blocked (special state)
  - from: blocked
    to: ready_for_refinement_ba
    description: "Blocker resolved, needs BA"
    authorized_by: [product-manager]
    requires:
      - "Blocker resolution documented"

  - from: blocked
    to: ready_for_refinement_tech
    description: "Blocker resolved, needs architect"
    authorized_by: [product-manager]
    requires:
      - "Blocker resolution documented"

  - from: blocked
    to: ready_for_development
    description: "Blocker resolved, ready for dev"
    authorized_by: [product-manager]
    requires:
      - "Blocker resolution documented"

  - from: blocked
    to: cancelled
    description: "Blocker unresolvable"
    authorized_by: [product-manager]

  # From on_hold (special state)
  - from: on_hold
    to: ready_for_refinement_ba
    description: "Resume planning"
    authorized_by: [product-manager]

  - from: on_hold
    to: ready_for_refinement_tech
    description: "Resume planning"
    authorized_by: [product-manager]

  - from: on_hold
    to: ready_for_development
    description: "Resume development"
    authorized_by: [product-manager]

  - from: on_hold
    to: cancelled
    description: "Deprioritized permanently"
    authorized_by: [product-manager]
```

### 4.5 Constraints

Global and per-stage constraints.

```yaml
constraints:
  # Global workflow constraints
  global:
    max_concurrent_tasks: 100
    max_task_duration: "24h"
    enable_auto_block_on_timeout: true
    enable_stale_task_detection: true
    stale_threshold: "2h"  # No updates for 2h = stale

  # Per-agent-type concurrency limits
  per_agent_type:
    business-analyst:
      max_concurrent: 2
      max_queue_size: 20
      priority_boost_after: "12h"  # Boost priority if queued >12h

    architect:
      max_concurrent: 2
      max_queue_size: 15
      priority_boost_after: "12h"

    developer:
      max_concurrent: 5
      max_queue_size: 50
      priority_boost_after: "24h"

    tech-lead:
      max_concurrent: 2
      max_queue_size: 10
      priority_boost_after: "6h"

    qa:
      max_concurrent: 2
      max_queue_size: 15
      priority_boost_after: "6h"

    product-manager:
      max_concurrent: 1
      max_queue_size: 10
      priority_boost_after: "6h"

  # Resource constraints
  resources:
    max_api_cost_per_hour: 50.00  # USD
    max_tokens_per_task: 1000000
    enable_cost_tracking: true
    cost_limit_action: pause  # or 'block' or 'alert'

  # Retry constraints
  retries:
    max_retries: 3
    backoff_strategy: exponential
    initial_backoff: "5m"
    max_backoff: "1h"
    retry_on_errors:
      - rate_limit
      - timeout
      - transient_error
    escalate_on_errors:
      - authentication_error
      - permission_error
      - invalid_request
```

---

## 5. Storage Location

### 5.1 Recommended Location

**Store workflow configs in Shark's config directory:**

```
~/.config/shark/
├── config.yml              # Shark CLI config
├── workflows/
│   ├── generic.workflow.yml        # Default/fallback workflow
│   ├── wormwoodGM.workflow.yml     # WormwoodGM custom workflow
│   ├── simple-dev.workflow.yml     # Simplified dev workflow
│   └── enterprise.workflow.yml     # Enterprise approval workflow
└── db/
    └── shark.db            # SQLite database
```

**Rationale:**

1. **Standard Location**: Follows XDG Base Directory Specification
2. **User-Scoped**: Each user can have custom workflows
3. **Easy Discovery**: `shark workflow list` can scan this directory
4. **Version Control**: Users can track workflow definitions in git
5. **Validation**: Shark CLI validates on load

### 5.2 Alternative Locations Considered

| Location | Pros | Cons | Verdict |
|----------|------|------|---------|
| **Shark config dir** (chosen) | Standard location, easy discovery, version control | User must manage files | ✅ **Recommended** |
| **Shark database** | Single source of truth, versioned, queryable | Harder to edit, no git tracking | ⚠️ Consider for future |
| **Project directory** | Per-project workflows, git-tracked | Discovery complexity, orchestrator needs project path | ❌ Too complex |
| **Orchestrator config** | Close to consumer | Violates separation of concerns | ❌ Rejected |

### 5.3 Database Schema Extension (Future)

For advanced use cases, store workflow configs in Shark database:

```sql
CREATE TABLE workflow_definitions (
    workflow_name TEXT PRIMARY KEY,
    version TEXT NOT NULL,
    config_yaml TEXT NOT NULL,  -- Full YAML config
    config_hash TEXT NOT NULL,  -- SHA256 for change detection
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1
);

CREATE TABLE workflow_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_name TEXT NOT NULL,
    version TEXT NOT NULL,
    config_yaml TEXT NOT NULL,
    changed_by TEXT,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    change_reason TEXT
);

-- Link tasks to workflow version
ALTER TABLE tasks ADD COLUMN workflow_name TEXT DEFAULT 'generic';
ALTER TABLE tasks ADD COLUMN workflow_version TEXT DEFAULT '1.0';
```

**Benefits of database storage:**
- Version history tracking
- Atomic updates
- Queryable via SQL
- Can enforce workflow-per-epic or workflow-per-feature

**Implementation:** Phase 2 enhancement after file-based system proven.

---

## 6. Configuration Examples

### 6.1 Complete WormwoodGM Workflow

See section 4 for the complete WormwoodGM workflow configuration.

### 6.2 Generic Workflow (Fallback)

Simplified workflow for generic task management:

```yaml
workflow:
  name: generic
  version: "1.0"
  description: "Simple todo → in_progress → completed workflow"

  metadata:
    author: "Shark CLI"
    created: "2026-01-13"

  entry_points:
    - status: todo

  terminal_states:
    - status: completed
    - status: cancelled

  special_states:
    - status: blocked
      allow_from: any

  stages:
    - name: execution
      description: "Task execution"
      phase: development
      color: yellow

      statuses:
        - todo
        - in_progress

      agent_type: developer

      agent_instructions:
        base_prompt: |
          You are implementing this task.
          Review the task description and acceptance criteria.
          Implement the solution following best practices.
          Update shark with progress and completion.

        skills_required:
          - implementation
          - shark-task-management

        context_to_include:
          - task_description
          - acceptance_criteria

        success_criteria:
          - "Task requirements met"
          - "Implementation complete"

      transitions:
        on_success:
          - to: completed

        on_block:
          - to: blocked

      constraints:
        max_concurrent: 10
        timeout: "8h"

  statuses:
    - name: todo
      description: "Ready to start"
      is_queue: true
      agent_type: developer

    - name: in_progress
      description: "Being worked on"
      is_active: true
      agent_type: developer

    - name: completed
      description: "Finished"
      is_terminal: true

    - name: cancelled
      description: "Not needed"
      is_terminal: true

    - name: blocked
      description: "Blocked"
      is_special: true

  transitions:
    - from: todo
      to: in_progress
      authorized_by: [developer]
      automatic: true

    - from: in_progress
      to: completed
      authorized_by: [developer]

    - from: in_progress
      to: blocked
      authorized_by: [developer]

    - from: blocked
      to: todo
      authorized_by: [product-manager]

    - from: todo
      to: cancelled
      authorized_by: [product-manager]

  constraints:
    global:
      max_concurrent_tasks: 50

    per_agent_type:
      developer:
        max_concurrent: 10
```

### 6.3 Simplified Development Workflow

For projects that don't need full BA/Arch stages:

```yaml
workflow:
  name: simple-dev
  version: "1.0"
  description: "Design → Dev → Review → QA → Deploy"

  stages:
    - name: design
      agent_type: developer
      statuses: [ready_for_design, in_design]
      # ... (similar structure to wormwoodGM)

    - name: development
      agent_type: developer
      statuses: [ready_for_development, in_development]
      # ...

    - name: review
      agent_type: tech-lead
      statuses: [ready_for_review, in_review]
      # ...

    - name: qa
      agent_type: qa
      statuses: [ready_for_qa, in_qa]
      # ...

    - name: deploy
      agent_type: devops
      statuses: [ready_for_deploy, deploying]
      # ...

  # ... rest of config
```

---

## 7. Orchestrator Integration

### 7.1 How Orchestrator Consumes Workflow Config

The orchestrator queries Shark for workflow configuration:

```go
// Orchestrator initialization
func (o *Orchestrator) Initialize() error {
    // Get workflow name from config or task metadata
    workflowName := o.config.WorkflowName // e.g., "wormwoodGM"

    // Query shark for workflow config
    cmd := exec.Command("shark", "workflow", "show", workflowName, "--json")
    output, err := cmd.Output()
    if err != nil {
        return fmt.Errorf("failed to load workflow: %w", err)
    }

    // Parse workflow config
    var workflow WorkflowConfig
    err = json.Unmarshal(output, &workflow)
    if err != nil {
        return fmt.Errorf("invalid workflow config: %w", err)
    }

    // Validate workflow completeness
    if err := workflow.Validate(); err != nil {
        return fmt.Errorf("workflow validation failed: %w", err)
    }

    // Cache workflow config
    o.workflow = &workflow

    return nil
}
```

### 7.2 Task Selection Using Workflow Config

```go
func (ts *TaskSelector) SelectTasks() ([]*Task, error) {
    var selectedTasks []*Task

    // Iterate through workflow stages
    for _, stage := range ts.orchestrator.workflow.Stages {
        // Get queue status for this stage
        queueStatus := stage.GetQueueStatus() // e.g., "ready_for_development"
        agentType := stage.AgentType           // e.g., "developer"

        // Check concurrency limit for this agent type
        maxConcurrent := ts.orchestrator.workflow.GetAgentLimit(agentType)
        currentCount := ts.CountActiveAgents(agentType)

        if currentCount >= maxConcurrent {
            continue // Skip this stage, at capacity
        }

        // Query shark for tasks in queue
        tasks, err := ts.QueryTasks(queueStatus)
        if err != nil {
            return nil, err
        }

        // Filter by dependencies and priority
        availableTasks := ts.FilterByDependencies(tasks)
        prioritizedTasks := ts.PrioritizeTasks(availableTasks)

        // Select up to limit
        slotsAvailable := maxConcurrent - currentCount
        tasksToSpawn := prioritizedTasks[:min(len(prioritizedTasks), slotsAvailable)]

        selectedTasks = append(selectedTasks, tasksToSpawn...)
    }

    return selectedTasks, nil
}
```

### 7.3 Agent Spawning with Stage Instructions

```go
func (as *AgentSpawner) SpawnAgent(task *Task) error {
    // Determine stage from task status
    stage := as.orchestrator.workflow.GetStageForStatus(task.Status)
    if stage == nil {
        return fmt.Errorf("no stage found for status: %s", task.Status)
    }

    // Get agent instructions for this stage
    instructions := stage.AgentInstructions

    // Build agent context
    context := as.BuildAgentContext(task, instructions)

    // Create agent prompt
    prompt := fmt.Sprintf(`%s

Task ID: %s
Task Title: %s

Context:
%s

Success Criteria:
%s

Artifacts to Create:
%s

Skills to Use:
%s
`,
        instructions.BasePrompt,
        task.ID,
        task.Title,
        context.Render(),
        strings.Join(instructions.SuccessCriteria, "\n- "),
        strings.Join(instructions.ArtifactsToCreate, "\n- "),
        strings.Join(instructions.SkillsRequired, ", "),
    )

    // Spawn agent with constructed prompt
    agent := NewAgent(task.ID, stage.AgentType, prompt)
    return agent.Start()
}
```

### 7.4 Caching Strategy

**Recommendation: Load on startup, cache in memory, hot reload on change**

```go
type Orchestrator struct {
    workflow       *WorkflowConfig
    workflowMtime  time.Time
    config         *OrchestratorConfig
    // ... other fields
}

func (o *Orchestrator) CheckWorkflowChanges() error {
    // Check if workflow file changed
    workflowPath := o.GetWorkflowPath()
    stat, err := os.Stat(workflowPath)
    if err != nil {
        return err
    }

    if stat.ModTime().After(o.workflowMtime) {
        log.Info("Workflow config changed, reloading...")

        // Reload workflow
        if err := o.LoadWorkflow(); err != nil {
            log.Error("Failed to reload workflow: %v", err)
            // Continue with cached version
            return err
        }

        o.workflowMtime = stat.ModTime()
        log.Info("Workflow config reloaded successfully")
    }

    return nil
}

// Call in main scheduler loop
func (s *Scheduler) Run() {
    ticker := time.NewTicker(s.pollInterval)
    defer ticker.Stop()

    configCheckTicker := time.NewTicker(1 * time.Minute)
    defer configCheckTicker.Stop()

    for {
        select {
        case <-ticker.C:
            // Regular scheduling cycle
            s.orchestrator.ScheduleTasks()

        case <-configCheckTicker.C:
            // Check for workflow config changes
            s.orchestrator.CheckWorkflowChanges()

        case <-s.stopChan:
            return
        }
    }
}
```

**Benefits:**
- Fast access (in-memory cache)
- Hot reload without restart
- No database query overhead
- Config file changes reflected within 1 minute

### 7.5 Validation and Error Handling

```go
func (w *WorkflowConfig) Validate() error {
    var errors []string

    // Validate all statuses are defined
    definedStatuses := make(map[string]bool)
    for _, status := range w.Statuses {
        definedStatuses[status.Name] = true
    }

    // Validate transitions reference valid statuses
    for _, transition := range w.Transitions {
        if !definedStatuses[transition.From] {
            errors = append(errors, fmt.Sprintf("transition references undefined status: %s", transition.From))
        }
        if !definedStatuses[transition.To] {
            errors = append(errors, fmt.Sprintf("transition references undefined status: %s", transition.To))
        }
    }

    // Validate each stage has valid statuses
    for _, stage := range w.Stages {
        for _, status := range stage.Statuses {
            if !definedStatuses[status] {
                errors = append(errors, fmt.Sprintf("stage %s references undefined status: %s", stage.Name, status))
            }
        }

        // Validate agent_type is set
        if stage.AgentType == "" {
            errors = append(errors, fmt.Sprintf("stage %s missing agent_type", stage.Name))
        }
    }

    // Validate at least one entry point
    if len(w.EntryPoints) == 0 {
        errors = append(errors, "workflow must have at least one entry point")
    }

    // Validate at least one terminal state
    if len(w.TerminalStates) == 0 {
        errors = append(errors, "workflow must have at least one terminal state")
    }

    if len(errors) > 0 {
        return fmt.Errorf("workflow validation failed:\n- %s", strings.Join(errors, "\n- "))
    }

    return nil
}
```

---

## 8. Migration Path

### 8.1 Phase 1: File-Based Workflow Config (Weeks 1-2)

**Goal:** Get workflow config out of orchestrator and into Shark config

**Steps:**

1. **Create workflow schema** (this document)
2. **Create `generic.workflow.yml`** (fallback)
3. **Create `wormwoodGM.workflow.yml`** (current workflow)
4. **Add shark CLI commands:**
   ```bash
   shark workflow list
   shark workflow show <name>
   shark workflow validate <name>
   shark workflow set-default <name>
   ```
5. **Update orchestrator to read workflow config**
6. **Remove hardcoded workflow mapping from orchestrator**
7. **Test with both workflows**

**Deliverables:**
- Workflow YAML files in `~/.config/shark/workflows/`
- Shark CLI workflow commands
- Updated orchestrator using workflow config
- Documentation updates

### 8.2 Phase 2: Per-Task Workflow Assignment (Weeks 3-4)

**Goal:** Allow different tasks to use different workflows

**Steps:**

1. **Add workflow columns to tasks table:**
   ```sql
   ALTER TABLE tasks ADD COLUMN workflow_name TEXT DEFAULT 'generic';
   ALTER TABLE tasks ADD COLUMN workflow_version TEXT DEFAULT '1.0';
   ```
2. **Update shark CLI to set workflow per task:**
   ```bash
   shark task create --workflow wormwoodGM
   shark task set-workflow T-E10-F05-001 wormwoodGM
   ```
3. **Update orchestrator to query task workflow:**
   ```go
   workflow := o.GetWorkflowForTask(task)
   ```
4. **Support mixed workflows in single orchestrator run**

**Deliverables:**
- Database schema update
- Per-task workflow commands
- Orchestrator support for multi-workflow
- Testing with mixed workflow tasks

### 8.3 Phase 3: Database-Backed Workflow (Future)

**Goal:** Store workflows in Shark database for versioning and history

**Steps:**

1. **Create workflow tables** (see section 5.3)
2. **Add workflow import command:**
   ```bash
   shark workflow import wormwoodGM.workflow.yml
   ```
3. **Add workflow versioning:**
   ```bash
   shark workflow update wormwoodGM --version 2.0
   shark workflow history wormwoodGM
   shark workflow rollback wormwoodGM --to-version 1.0
   ```
4. **Migrate existing file-based workflows to database**
5. **Add workflow export command:**
   ```bash
   shark workflow export wormwoodGM > wormwoodGM.workflow.yml
   ```

**Deliverables:**
- Workflow database schema
- Import/export commands
- Version history tracking
- Rollback capabilities

### 8.4 Phase 4: Advanced Features (Future)

**Potential enhancements:**

1. **Workflow Templates:**
   ```bash
   shark workflow create-from-template agile-scrum
   shark workflow create-from-template kanban
   ```

2. **Workflow Visualization:**
   ```bash
   shark workflow diagram wormwoodGM > workflow.mermaid
   shark workflow diagram wormwoodGM --format png > workflow.png
   ```

3. **Workflow Analytics:**
   ```bash
   shark workflow analytics wormwoodGM
   # Shows: average time per stage, bottlenecks, success rates
   ```

4. **Conditional Transitions:**
   ```yaml
   transitions:
     - from: in_development
       to: ready_for_code_review
       conditions:
         - test_coverage >= 80%
         - no_linting_errors
         - all_acceptance_criteria_met
   ```

5. **Dynamic Agent Instructions:**
   ```yaml
   agent_instructions:
     base_prompt_template: |
       You are {{agent_type}} working on {{task.epic}}.
       Priority: {{task.priority}}
       ...
   ```

---

## 9. Benefits Analysis

### 9.1 Compared to Hardcoded Orchestrator Mapping

| Benefit | Hardcoded | Workflow Config | Impact |
|---------|-----------|-----------------|--------|
| **Add New Workflow** | Change orchestrator code, rebuild, redeploy | Create YAML file, reload config | 🟢 10x faster |
| **Modify Stage Instructions** | Code change | Edit YAML | 🟢 No rebuild needed |
| **Understand Workflow** | Read Go code | Read YAML | 🟢 Self-documenting |
| **Validate Workflow** | Manual testing | `shark workflow validate` | 🟢 Automated |
| **Version Control** | Code commits | Workflow file commits | 🟡 Same (both git-tracked) |
| **Multiple Workflows** | If/else logic in code | Load different YAML | 🟢 Clean separation |
| **Test Workflow Logic** | Integration tests | Unit tests on YAML | 🟢 Easier testing |
| **Documentation Sync** | Docs separate from code | Config IS documentation | 🟢 No drift |
| **Onboarding New Devs** | Read orchestrator code | Read workflow YAML | 🟢 Clearer |
| **Client Customization** | Fork orchestrator | Provide custom YAML | 🟢 No fork needed |

### 9.2 Key Benefits

**1. Separation of Concerns**
- Orchestrator: execution engine (how to run agents)
- Shark: workflow engine (what work exists, what transitions are valid)
- Workflow config: workflow definition (stages, agents, instructions)

**2. Flexibility**
- Add new workflows without code changes
- Modify workflows without orchestrator rebuild
- Support multiple workflows in parallel
- Per-project workflow customization

**3. Maintainability**
- Workflow definition is declarative and readable
- Changes are localized to config files
- Validation catches errors early
- Self-documenting system

**4. Testability**
- Validate workflow structure automatically
- Test workflow transitions in isolation
- Mock workflows for orchestrator testing
- Version control tracks workflow changes

**5. Extensibility**
- Easy to add new stages
- Add new agent types without orchestrator changes
- Extend with custom fields (future)
- Plugin system for custom validators (future)

### 9.3 Concrete Examples

**Example 1: Add New Stage**

*Before (hardcoded):*
```go
// In orchestrator code
statusMapping := map[string]string{
    "ready_for_refinement_ba": "business-analyst",
    "ready_for_refinement_tech": "architect",
    "ready_for_development": "developer",
    // Want to add security review stage...
    // Must modify code, rebuild, test, deploy
}
```

*After (config):*
```yaml
# Just edit wormwoodGM.workflow.yml
stages:
  # ... existing stages ...
  - name: security_review
    agent_type: security-engineer
    statuses:
      - ready_for_security_review
      - in_security_review
    # ... rest of config
```
```bash
shark workflow validate wormwoodGM
# Orchestrator auto-reloads within 1 minute
```

**Example 2: Support New Project with Different Workflow**

*Before (hardcoded):*
```go
// Need to add if/else logic in orchestrator
if project == "wormwoodGM" {
    mapping = wormwoodGMMapping
} else if project == "acmeCorpEnterprise" {
    mapping = acmeCorpMapping
} else {
    mapping = genericMapping
}
```

*After (config):*
```bash
# Create new workflow config
cp ~/.config/shark/workflows/wormwoodGM.workflow.yml \
   ~/.config/shark/workflows/acmeCorp.workflow.yml

# Edit as needed
vim ~/.config/shark/workflows/acmeCorp.workflow.yml

# Assign to tasks
shark task create --workflow acmeCorp

# Orchestrator automatically handles it
```

**Example 3: Document Workflow for Stakeholders**

*Before:*
- Read orchestrator Go code
- Read workflow-custom.md (may be outdated)
- Infer from Shark status values
- Ask developers to explain

*After:*
```bash
shark workflow show wormwoodGM
# or
cat ~/.config/shark/workflows/wormwoodGM.workflow.yml
# Self-documenting, always accurate
```

### 9.4 ROI Calculation

**Time Saved:**

| Activity | Before (hours) | After (hours) | Savings |
|----------|----------------|---------------|---------|
| Add new workflow | 8 (code + test + deploy) | 2 (create YAML + validate) | 6h |
| Modify stage instructions | 4 (code change + test) | 0.5 (edit YAML) | 3.5h |
| Understand workflow | 2 (read code) | 0.5 (read YAML) | 1.5h |
| Validate workflow completeness | 4 (manual testing) | 0.1 (automated validation) | 3.9h |
| Create workflow documentation | 3 (write separate doc) | 0 (config is doc) | 3h |

**Over 1 year (4 new workflows, 20 modifications, 10 docs):**
- Before: `(4×8) + (20×4) + (10×3) = 142 hours`
- After: `(4×2) + (20×0.5) + (10×0) = 18 hours`
- **Savings: 124 hours** (~3 weeks of developer time)

**Plus intangible benefits:**
- Fewer bugs from hardcoded logic
- Faster onboarding
- Better documentation
- More maintainable system

---

## 10. Implementation Guidance

### 10.1 Immediate Actions (Week 1)

1. **Create workflow schema** (✅ This document)
2. **Create `~/.config/shark/workflows/` directory**
3. **Create `generic.workflow.yml`** (section 6.2)
4. **Create `wormwoodGM.workflow.yml`** (section 6.1)
5. **Validate YAML syntax:**
   ```bash
   yamllint ~/.config/shark/workflows/*.workflow.yml
   ```

### 10.2 Shark CLI Implementation (Week 2)

**Add workflow commands to Shark CLI:**

```bash
# List available workflows
shark workflow list
# Output:
# Available workflows:
#   generic (v1.0) - Simple todo → in_progress → completed
#   wormwoodGM (v1.0) - BA → Arch → Dev → Review → QA → Approval

# Show workflow details
shark workflow show wormwoodGM [--json]
# Output: Full workflow config

# Validate workflow
shark workflow validate wormwoodGM
# Output: ✅ Workflow is valid
#   - 6 stages defined
#   - 14 statuses defined
#   - 42 transitions defined
#   - All transitions reference valid statuses
#   - All stages have agent types
#   - Entry points and terminal states defined

# Set default workflow
shark workflow set-default wormwoodGM
# Stored in ~/.config/shark/config.yml

# Get workflow for task
shark task get T-E10-F05-001 --show-workflow
# Shows which workflow this task uses

# Diagram workflow (future)
shark workflow diagram wormwoodGM
# Output: Mermaid diagram
```

**Implementation in Shark CLI (Go):**

```go
// shark/cmd/workflow.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "shark/internal/workflow"
)

var workflowCmd = &cobra.Command{
    Use:   "workflow",
    Short: "Manage workflow configurations",
}

var workflowListCmd = &cobra.Command{
    Use:   "list",
    Short: "List available workflows",
    RunE: func(cmd *cobra.Command, args []string) error {
        wm := workflow.NewManager()
        workflows, err := wm.List()
        if err != nil {
            return err
        }

        for _, w := range workflows {
            fmt.Printf("%s (v%s) - %s\n", w.Name, w.Version, w.Description)
        }
        return nil
    },
}

var workflowShowCmd = &cobra.Command{
    Use:   "show [workflow-name]",
    Short: "Show workflow configuration",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        wm := workflow.NewManager()
        w, err := wm.Load(args[0])
        if err != nil {
            return err
        }

        jsonOutput, _ := cmd.Flags().GetBool("json")
        if jsonOutput {
            return w.PrintJSON()
        }
        return w.PrintYAML()
    },
}

var workflowValidateCmd = &cobra.Command{
    Use:   "validate [workflow-name]",
    Short: "Validate workflow configuration",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        wm := workflow.NewManager()
        w, err := wm.Load(args[0])
        if err != nil {
            return err
        }

        if err := w.Validate(); err != nil {
            fmt.Printf("❌ Workflow validation failed:\n%v\n", err)
            return err
        }

        fmt.Printf("✅ Workflow '%s' is valid\n", w.Name)
        fmt.Printf("  - %d stages defined\n", len(w.Stages))
        fmt.Printf("  - %d statuses defined\n", len(w.Statuses))
        fmt.Printf("  - %d transitions defined\n", len(w.Transitions))
        return nil
    },
}

func init() {
    rootCmd.AddCommand(workflowCmd)
    workflowCmd.AddCommand(workflowListCmd)
    workflowCmd.AddCommand(workflowShowCmd)
    workflowCmd.AddCommand(workflowValidateCmd)

    workflowShowCmd.Flags().Bool("json", false, "Output as JSON")
}
```

### 10.3 Orchestrator Integration (Weeks 3-4)

**Update orchestrator to consume workflow config:**

```go
// orchestrator/pkg/workflow/loader.go
package workflow

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "time"
)

type Loader struct {
    workflowName  string
    workflow      *Config
    lastLoaded    time.Time
    cacheDuration time.Duration
}

func NewLoader(workflowName string) *Loader {
    return &Loader{
        workflowName:  workflowName,
        cacheDuration: 1 * time.Minute, // Reload check interval
    }
}

func (l *Loader) Load() (*Config, error) {
    // Check if cached version is fresh
    if l.workflow != nil && time.Since(l.lastLoaded) < l.cacheDuration {
        return l.workflow, nil
    }

    // Query shark CLI for workflow config
    cmd := exec.Command("shark", "workflow", "show", l.workflowName, "--json")
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to load workflow %s: %w", l.workflowName, err)
    }

    // Parse workflow config
    var config Config
    if err := json.Unmarshal(output, &config); err != nil {
        return nil, fmt.Errorf("invalid workflow config: %w", err)
    }

    // Validate
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("workflow validation failed: %w", err)
    }

    // Cache
    l.workflow = &config
    l.lastLoaded = time.Now()

    return &config, nil
}

func (l *Loader) GetStageForStatus(status string) *Stage {
    if l.workflow == nil {
        return nil
    }

    for _, stage := range l.workflow.Stages {
        for _, s := range stage.Statuses {
            if s == status {
                return &stage
            }
        }
    }
    return nil
}

func (l *Loader) GetAgentLimit(agentType string) int {
    if l.workflow == nil {
        return 0
    }

    if limit, ok := l.workflow.Constraints.PerAgentType[agentType]; ok {
        return limit.MaxConcurrent
    }
    return 0
}
```

**Update TaskSelector:**

```go
// orchestrator/pkg/selector/selector.go
func (ts *TaskSelector) SelectTasks() ([]*Task, error) {
    workflow, err := ts.workflowLoader.Load()
    if err != nil {
        return nil, err
    }

    var selectedTasks []*Task

    // Iterate workflow stages
    for _, stage := range workflow.Stages {
        queueStatus := stage.GetQueueStatus()
        agentType := stage.AgentType

        // Check concurrency limit
        maxConcurrent := workflow.GetAgentLimit(agentType)
        currentCount := ts.CountActiveAgents(agentType)

        if currentCount >= maxConcurrent {
            continue
        }

        // Query shark for tasks
        tasks, err := ts.QueryTasksByStatus(queueStatus)
        if err != nil {
            return nil, err
        }

        // Filter and prioritize
        availableTasks := ts.FilterByDependencies(tasks)
        prioritizedTasks := ts.PrioritizeTasks(availableTasks)

        // Select tasks
        slotsAvailable := maxConcurrent - currentCount
        tasksToSpawn := prioritizedTasks[:min(len(prioritizedTasks), slotsAvailable)]

        selectedTasks = append(selectedTasks, tasksToSpawn...)
    }

    return selectedTasks, nil
}
```

### 10.4 Testing Strategy

**1. Workflow Config Validation Tests:**

```go
func TestWorkflowValidation(t *testing.T) {
    tests := []struct{
        name        string
        config      string
        expectError bool
        errorMsg    string
    }{
        {
            name: "valid_workflow",
            config: `
workflow:
  name: test
  statuses:
    - name: todo
    - name: done
  transitions:
    - from: todo
      to: done
`,
            expectError: false,
        },
        {
            name: "invalid_transition_reference",
            config: `
workflow:
  name: test
  statuses:
    - name: todo
  transitions:
    - from: todo
      to: invalid_status
`,
            expectError: true,
            errorMsg: "undefined status",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var config WorkflowConfig
            err := yaml.Unmarshal([]byte(tt.config), &config)
            require.NoError(t, err)

            err = config.Validate()
            if tt.expectError {
                require.Error(t, err)
                require.Contains(t, err.Error(), tt.errorMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

**2. Integration Tests:**

```go
func TestOrchestratorWithWorkflowConfig(t *testing.T) {
    // Setup test workflow
    testWorkflow := createTestWorkflow()
    writeWorkflowFile(testWorkflow)

    // Initialize orchestrator
    orch := NewOrchestrator("test-workflow")
    err := orch.Initialize()
    require.NoError(t, err)

    // Create test tasks
    createTestTask("T-001", "ready_for_development")
    createTestTask("T-002", "ready_for_code_review")

    // Run task selection
    tasks, err := orch.SelectTasks()
    require.NoError(t, err)

    // Verify correct agent types assigned
    assert.Equal(t, "developer", tasks[0].AgentType)
    assert.Equal(t, "tech-lead", tasks[1].AgentType)
}
```

**3. Workflow Migration Tests:**

```bash
# Test migration from hardcoded to config
./test-migration.sh
# 1. Run orchestrator with hardcoded workflow
# 2. Create equivalent workflow config
# 3. Run orchestrator with config
# 4. Verify identical behavior
```

### 10.5 Documentation Updates

**Update documentation to reference workflow config:**

1. **Orchestrator README:**
   - Remove hardcoded workflow section
   - Add "Workflow Configuration" section
   - Link to workflow config files
   - Document how to create custom workflows

2. **Shark CLI Documentation:**
   - Add workflow commands section
   - Document workflow config schema
   - Provide workflow examples
   - Troubleshooting guide

3. **Agent Documentation:**
   - Update agent role descriptions
   - Reference workflow config for stage instructions
   - Document how agents receive instructions

4. **Migration Guide:**
   - Step-by-step migration from hardcoded
   - Backward compatibility notes
   - Rollback procedure

### 10.6 Rollout Plan

**Phase 1: Preparation (Week 1)**
- ✅ Create workflow schema (this doc)
- Create workflow YAML files
- Validate YAML syntax
- Review with team

**Phase 2: Shark CLI (Week 2)**
- Implement workflow commands
- Add validation logic
- Write tests
- Update documentation

**Phase 3: Orchestrator Integration (Weeks 3-4)**
- Update orchestrator to read workflow config
- Remove hardcoded workflow mapping
- Add hot reload support
- Integration testing

**Phase 4: Validation (Week 5)**
- End-to-end testing with real workflows
- Performance testing
- Validate hot reload works
- Document any issues

**Phase 5: Deployment (Week 6)**
- Deploy updated Shark CLI
- Deploy updated orchestrator
- Migrate existing tasks (if needed)
- Monitor for issues

**Rollback Plan:**
- Keep hardcoded workflow as fallback in orchestrator v1
- If workflow config fails to load, use hardcoded
- Log warning and continue with fallback
- Fix config and reload

---

## Conclusion

This workflow configuration design provides:

1. **Clear Separation of Concerns**: Orchestrator executes, Shark manages state, workflow config defines behavior
2. **Flexibility**: Multiple workflows, easy customization, no code changes
3. **Maintainability**: Declarative config, self-documenting, version controlled
4. **Extensibility**: Easy to add stages, agent types, and features
5. **Testability**: Automated validation, isolated testing, clear contracts

**Next Steps:**

1. Review this design with team
2. Create workflow YAML files (section 6)
3. Implement Shark CLI workflow commands (section 10.2)
4. Update orchestrator to consume workflow config (section 10.3)
5. Test and deploy (section 10.6)

The configuration-driven approach aligns with our design principles (Appropriate, Proven, Simple) and provides a solid foundation for multi-workflow support without coupling the orchestrator to specific workflow implementations.

---

## References

- [AI Agent Orchestrator Design](/home/jwwelbor/.claude/docs/architecture/ai-agent-orchestrator-design.md)
- [Orchestration Approach Analysis](/home/jwwelbor/.claude/docs/architecture/orchestration-approach-analysis.md)
- [WormwoodGM Workflow Status Map](/home/jwwelbor/.claude/docs/workflow-custom.md)
- [Shark Task Management CLI](~/.claude/skills/shark-task-management/SKILL.md)
- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions)
- [Argo Workflows](https://argoproj.github.io/argo-workflows/)
- [Tekton Pipelines](https://tekton.dev/docs/pipelines/)
