# Workflow Configuration Examples

This directory contains complete, working workflow configuration examples for Shark Task Manager. These examples demonstrate how to configure orchestrator actions for different project types and complexity levels.

## Overview

Shark supports customizable workflow configurations through `.sharkconfig.json`. The `status_metadata` section defines task statuses and their associated orchestrator actions, enabling workflow-driven agent spawning without hardcoded orchestrator logic.

Each example includes:
- **Status Flow**: Valid state transitions between statuses
- **Status Metadata**: Description, colors, phases, and orchestrator actions for each status
- **Special Statuses**: Start and terminal states for new tasks

## Example Workflows

### 1. Simple Workflow (`simple.json`)

**Use This For**:
- Small teams with simple project structures
- Straightforward task lifecycle: todo → in_progress → review → done
- Projects without complex refinement phases

**Statuses** (5 total):
- `todo` - Initial state, no action
- `in_progress` - Developer working on implementation
- `review` - Code review by reviewer agent
- `done` - Task completed (terminal)
- `blocked` - Waiting on dependency (can return to todo)

**Key Features**:
- Minimal complexity: only essential statuses
- Linear primary flow
- Clear, simple agent instructions
- Fast time-to-value

**Agent Types**:
- `developer` - Implementation (skills: implementation, testing)
- `reviewer` - Code review (skills: code-review, quality)

**When to Use**:
- Quick prototypes or MVPs
- Small projects with 1-5 developers
- Teams without formal QA or architecture review

**Example Commands**:
```bash
# Use simple workflow for your project
cp simple.json ../.sharkconfig.json

# Start implementing a task
shark task start E01-F01-001

# Verify workflow configuration
shark workflow validate-actions --config=simple.json
```

---

### 2. WormwoodGM Workflow (`wormwoodgm.json`)

**Use This For**:
- AI-driven development with AI agent orchestrators
- Projects with business analysis and technical architecture phases
- Organizations using specification-writing and collaborative design

**Statuses** (16 total):
- **Planning Phase** (Draft & Business Analysis)
  - `draft` - Wait for triage
  - `ready_for_refinement_ba` - Spawn business analyst
  - `in_refinement_ba` - BA working

- **Architecture Phase** (Technical Design)
  - `ready_for_refinement_tech` - Spawn architect
  - `in_refinement_tech` - Architect designing

- **Implementation Phase**
  - `ready_for_development` - Spawn developer
  - `in_development` - Developer implementing

- **Code Review Phase**
  - `ready_for_code_review` - Spawn reviewer
  - `in_code_review` - Reviewer working

- **QA Phase**
  - `ready_for_qa` - Spawn QA engineer
  - `in_qa` - QA testing

- **Approval Phase**
  - `ready_for_approval` - Spawn tech lead
  - `in_approval` - Tech lead approving

- **Terminal States**
  - `completed` - Archive
  - `blocked` - Pause
  - `cancelled` - Archive

**Key Features**:
- Multi-stage refinement (BA → Architect → Developer)
- Clear separation of concerns
- Comprehensive quality gates
- AI agent skill sets aligned with task types
- Template instructions guide agent work

**Agent Types with Skills**:
- `business-analyst` - (specification-writing, shark-task-management, research)
- `architect` - (architecture, database-design, api-design)
- `developer` - (test-driven-development, implementation, shark-task-management)
- `reviewer` - (code-review, quality)
- `qa-engineer` - (testing, validation)
- `tech-lead` - (quality, approval)

**When to Use**:
- WormwoodGM development workflows
- Projects with 5-50 AI agents handling different roles
- Organizations valuing structured refinement and design phases
- Teams using test-driven development (TDD)

**Example Commands**:
```bash
# Use WormwoodGM workflow for your project
cp wormwoodgm.json ../.sharkconfig.json

# View all orchestrator actions in the workflow
shark workflow show-actions --config=wormwoodgm.json --json

# Check what action would occur for a status
shark config get-status-action ready_for_development --config=wormwoodgm.json

# Validate the complete workflow configuration
shark workflow validate-actions --config=wormwoodgm.json --strict
```

**Workflow Diagram**:
```
draft → ready_for_refinement_ba → in_refinement_ba →
  ready_for_refinement_tech → in_refinement_tech →
    ready_for_development → in_development →
      ready_for_code_review → in_code_review →
        ready_for_qa → in_qa →
          ready_for_approval → in_approval →
            completed
```

---

### 3. Enterprise Workflow (`enterprise.json`)

**Use This For**:
- Large organizations with compliance and security requirements
- Projects requiring audit trails and management approval
- Industries with regulatory requirements (healthcare, finance, etc.)

**Statuses** (22 total):
- **Requirement Phase** (Planning)
  - `draft` - Wait for manual triage
  - `ready_for_refinement` - Spawn business analyst
  - `in_refinement` - BA working

- **Security Phase** (Compliance Review)
  - `ready_for_security_review` - Spawn security engineer
  - `in_security_review` - Security assessment

- **Architecture Phase** (Design)
  - `ready_for_architecture_review` - Spawn architect
  - `in_architecture_review` - Architecture design

- **Implementation Phase**
  - `ready_for_development` - Spawn developer
  - `in_development` - Developer implementing

- **Review Phases**
  - `ready_for_code_review` - Spawn senior developer
  - `in_code_review` - Code review

- **Quality & Compliance**
  - `ready_for_qa` - Spawn QA engineer
  - `in_qa` - QA testing
  - `ready_for_compliance_check` - Spawn compliance officer
  - `in_compliance_check` - Compliance verification

- **Approval Phase**
  - `ready_for_approval` - Wait for manual approval
  - `in_approval` - Management approval
  - `approved` - Ready for deployment

- **Terminal States**
  - `completed` - Deploy and archive
  - `rejected` - Archive (approval failed)
  - `blocked` - Pause
  - `cancelled` - Archive

**Key Features**:
- Security and compliance gates
- Human approval decision points (`wait_for_triage`)
- Audit trail emphasis (archive includes documentation)
- Multi-level review (peer + senior developer)
- Comprehensive validation stages
- Corporate governance structure

**Agent Types with Skills**:
- `business-analyst` - (specification-writing, research, shark-task-management)
- `security-engineer` - (security, compliance)
- `architect` - (architecture, database-design, api-design)
- `developer` - (test-driven-development, implementation, shark-task-management)
- `senior-developer` - (code-review, quality)
- `qa-engineer` - (testing, validation)
- `compliance-officer` - (compliance, audit)

**When to Use**:
- Enterprise organizations (100+ engineers)
- Regulatory environments (HIPAA, SOC2, PCI-DSS)
- High-security projects
- Organizations with formal change management
- Projects requiring audit trails and compliance documentation

**Example Commands**:
```bash
# Use enterprise workflow for your project
cp enterprise.json ../.sharkconfig.json

# Validate strict compliance (all statuses must have actions)
shark workflow validate-actions --config=enterprise.json --strict

# Show all actions grouped by phase
shark workflow show-actions --config=enterprise.json

# Check specific status configuration
shark config get-status-action ready_for_compliance_check --config=enterprise.json --json
```

**Approval Gates**:
- Draft triage (manual decision to proceed)
- Security review (compliance assessment)
- Management approval (final authorization)
- Compliance check (audit verification)

---

## Common Configuration Patterns

### Orchestrator Action Types

All examples use four action types:

#### 1. **spawn_agent** - Launch an AI agent
```json
"orchestrator_action": {
  "action": "spawn_agent",
  "agent_type": "developer",
  "skills": ["implementation", "testing"],
  "instruction_template": "Implement task {task_id}..."
}
```

Used for: Implementation, review, testing, design phases
Required fields: `agent_type`, `skills`, `instruction_template`

#### 2. **pause** - Wait, do not spawn agent
```json
"orchestrator_action": {
  "action": "pause",
  "instruction_template": "Task {task_id} is blocked..."
}
```

Used for: Blocked status, dependencies
Required fields: `instruction_template`

#### 3. **wait_for_triage** - Human decision needed
```json
"orchestrator_action": {
  "action": "wait_for_triage",
  "instruction_template": "Task {task_id} awaiting triage..."
}
```

Used for: Manual review/approval, initial intake
Required fields: `instruction_template`

#### 4. **archive** - Task complete, no further action
```json
"orchestrator_action": {
  "action": "archive",
  "instruction_template": "Task {task_id} is completed..."
}
```

Used for: Terminal states (completed, cancelled, rejected)
Required fields: `instruction_template`

### Workflow Phases

All statuses are assigned to a workflow phase for display and organization:

- **planning** - Requirements, design, architecture
- **development** - Implementation work
- **review** - Code and peer review
- **qa** - Quality assurance and testing
- **approval** - Management/compliance approval
- **done** - Terminal states (completed, cancelled)
- **any** - Special statuses (blocked, on_hold)

### Template Variables

Instructions use `{task_id}` placeholder, which is replaced with the actual task ID:

```
Input template:  "Implement task {task_id} using TDD approach"
Task T-E07-F01-001 gets: "Implement task T-E07-F01-001 using TDD approach"
```

Future variables (roadmap):
- `{epic_id}` - Epic identifier
- `{feature_id}` - Feature identifier
- `{task_title}` - Full task title
- `{priority}` - Priority level

---

## Using These Examples

### Option 1: Use an Example as-is

```bash
# Copy simple workflow to your project
cp simple.json /path/to/project/.sharkconfig.json

# Verify it works
shark init --non-interactive
shark task list
```

### Option 2: Customize an Example

```bash
# Start with WormwoodGM workflow
cp wormwoodgm.json /path/to/project/.sharkconfig.json

# Edit to add custom statuses, agent types, or skills
vim .sharkconfig.json

# Validate your changes
shark workflow validate-actions --strict

# Test a status action
shark config get-status-action ready_for_development --task=T-E07-F01-001
```

### Option 3: Create a New Workflow

Combine patterns from multiple examples to create a workflow suited to your organization:

```bash
# Start with template
cat simple.json > my-workflow.json

# Add security phase from enterprise.json
vim my-workflow.json

# Add BA refinement from wormwoodgm.json
vim my-workflow.json

# Validate
shark workflow validate-actions --config=my-workflow.json --strict

# Deploy
cp my-workflow.json /path/to/project/.sharkconfig.json
```

---

## Validation & Testing

### Validate an Example

```bash
# Basic validation (warns on missing actions)
shark workflow validate-actions --config=simple.json

# Strict validation (fails on any issues)
shark workflow validate-actions --config=simple.json --strict

# JSON output for automation
shark workflow validate-actions --config=simple.json --json
```

### Load an Example in Your Project

```bash
# Use example as config
shark --config=docs/examples/workflows/simple.json task list

# View all actions in JSON format
shark --config=docs/examples/workflows/wormwoodgm.json workflow show-actions --json
```

### Test a Single Status Action

```bash
# View action for a status
shark config get-status-action in_development \
  --config=docs/examples/workflows/simple.json

# Populate template with task ID
shark config get-status-action ready_for_refinement_ba \
  --config=docs/examples/workflows/wormwoodgm.json \
  --task=T-E07-F01-001
```

---

## Comparison Table

| Feature | Simple | WormwoodGM | Enterprise |
|---------|--------|-----------|-----------|
| **Total Statuses** | 5 | 16 | 22 |
| **Refinement Phases** | 0 | 2 (BA + Tech) | 1 (BA only) |
| **Security Review** | ❌ | ❌ | ✅ |
| **Compliance Check** | ❌ | ❌ | ✅ |
| **Manual Approval Gates** | ❌ | ❌ | 2 (draft, ready_for_approval) |
| **Agent Types** | 2 | 6 | 7 |
| **Best For** | Small teams | AI-driven dev | Enterprise/regulated |
| **Complexity** | Low | Medium-High | High |
| **Time to Configure** | <5 min | 15-30 min | 30-60 min |

---

## Best Practices

### 1. Status Naming Convention

Use consistent naming patterns:
- `ready_for_<role>` - Status is queued for an agent
- `in_<role>` - Agent actively working
- `<action>` - Terminal or special states (completed, blocked, cancelled)

### 2. Agent Skills

Define skills that match actual agent capabilities:
- `specification-writing` - Write clear requirements
- `implementation` - Code a feature
- `code-review` - Review code quality
- `testing` - Execute test plans
- `architecture` - Design systems
- `compliance` - Verify regulatory requirements

### 3. Instruction Templates

Write clear, actionable instructions for agents:
- Start with action verb ("Implement", "Review", "Validate")
- Include task ID for context: `{task_id}`
- Specify what success looks like
- Reference related artifacts

### 4. Workflow Organization

- **Linear flows** (simple): 4-6 statuses
- **Staged flows** (wormwoodgm): 12-18 statuses with phases
- **Complex flows** (enterprise): 18+ statuses with approval gates

### 5. Testing Your Workflow

1. Validate configuration: `shark workflow validate-actions --strict`
2. Test status transitions: Create a test task and move it through workflow
3. Verify actions: `shark config get-status-action <status> --task=<test-task>`
4. Check completeness: Ensure all actionable statuses have actions defined

---

## Advanced: Creating Custom Workflows

### Key Decisions

Before creating a custom workflow, answer these questions:

1. **How many phases?**
   - Simple: 1-2 phases (dev + review)
   - Medium: 3-4 phases (plan + dev + review + qa)
   - Complex: 5+ phases (plan + security + design + dev + review + qa + approval)

2. **What agents are involved?**
   - Developers only (simple)
   - Developers + reviewers + QA (medium)
   - Full team: analysts, architects, devs, reviewers, QA, compliance (complex)

3. **What approval gates are needed?**
   - None (simple, pull request-based)
   - Automated (TDD enforcement)
   - Manual (management sign-off)

4. **What compliance needs exist?**
   - None (internal projects)
   - Basic (general best practices)
   - Advanced (regulatory requirements)

### Example: Custom "Startup" Workflow

```bash
# Copy simple workflow as starting point
cp simple.json startup.json

# Edit startup.json:
# 1. Add security review status
# 2. Add QA status
# 3. Remove reviewer status (use pull requests instead)
# 4. Add startup-specific statuses: "product_approved", "metrics_tracked"

# Validate
shark workflow validate-actions --config=startup.json --strict

# Deploy
cp startup.json /path/to/project/.sharkconfig.json
```

---

## Troubleshooting

### Configuration Won't Validate

**Error**: `Action "invalid_action" not supported`

**Solution**: Check action type is one of: `spawn_agent`, `pause`, `wait_for_triage`, `archive`

**Error**: `Missing required field: agent_type for spawn_agent action`

**Solution**: `spawn_agent` requires `agent_type` and `skills` fields

**Error**: `Status 'todo' in status_flow not found in status_metadata`

**Solution**: Every status in `status_flow` must have a corresponding entry in `status_metadata`

### Workflow Not Loading

**Error**: `Failed to parse JSON`

**Solution**: Validate JSON syntax: `jq . your-config.json`

**Error**: `Invalid phase: "planning"`

**Solution**: Use valid phases: `planning`, `development`, `review`, `qa`, `approval`, `done`, `any`

---

## Integration with Shark CLI

These examples are designed to work seamlessly with Shark CLI commands:

```bash
# Validate example
shark workflow validate-actions --config=simple.json

# Show all actions
shark workflow show-actions --config=simple.json --json

# Check specific action
shark config get-status-action in_progress --config=simple.json --task=T-E07-F01-001

# Use as project config
cp simple.json /path/to/project/.sharkconfig.json
shark task list  # Uses the workflow configuration
```

---

## Next Steps

1. **Choose your workflow**: Start with Simple, WormwoodGM, or Enterprise
2. **Copy to your project**: `cp <example>.json /path/to/project/.sharkconfig.json`
3. **Validate**: `shark workflow validate-actions --strict`
4. **Test**: Create a task and move it through your workflow
5. **Customize**: Modify based on your team's needs
6. **Document**: Add your workflow to team documentation

---

## Questions & Feedback

For questions about these examples or to suggest improvements:
- Check the [Feature PRD](../../plan/E07-enhancements/E07-F21-add-actions-to-status-transition/feature.md)
- Review the [Workflow Configuration Design](../../plan/E07-enhancements/E07-F21-add-actions-to-status-transition/shark-workflow-config-design.md)
- Run `shark workflow --help` for CLI reference

---

*Last Updated: 2026-01-15*
*Examples created for Shark Task Manager E07-F21 Feature*
