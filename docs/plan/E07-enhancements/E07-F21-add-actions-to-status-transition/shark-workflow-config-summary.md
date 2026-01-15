# Shark Workflow Configuration - Executive Summary

**Document:** [Complete Design](/home/jwwelbor/.claude/docs/architecture/shark-workflow-config-design.md)
**Date:** 2026-01-13
**Author:** Architect Agent

---

## Problem

The orchestrator currently has **hardcoded workflow knowledge**:

```yaml
# Hardcoded in orchestrator code
status_to_agent_mapping:
  ready_for_refinement_ba: business-analyst
  ready_for_refinement_tech: architect
  ready_for_development: developer
  # ...
```

This violates separation of concerns and makes adding new workflows difficult.

---

## Solution

**Store workflow configuration in Shark as declarative YAML files.**

### Configuration Structure

```yaml
workflow:
  name: wormwoodGM
  version: "1.0"

  # Define all workflow stages
  stages:
    - name: business_analysis
      agent_type: business-analyst
      statuses: [ready_for_refinement_ba, in_refinement_ba]

      # Agent instructions for this stage
      agent_instructions:
        base_prompt: "You are the BA. Review requirements..."
        skills_required: [specification-writing, shark-task-management]
        context_to_include: [task_description, feature_context]
        success_criteria: ["Requirements documented", "Criteria defined"]
        artifacts_to_create: ["F01-requirements.md"]

      # Valid transitions from this stage
      transitions:
        on_success:
          - to: ready_for_refinement_tech  # Default
          - to: ready_for_development      # Skip tech design
        on_failure:
          - to: draft                      # Needs rework

      # Stage constraints
      constraints:
        max_concurrent: 2
        timeout: "4h"

    # More stages: technical_design, development, code_review, qa, approval
    # ...

  # All valid statuses
  statuses:
    - name: ready_for_refinement_ba
      agent_type: business-analyst
      is_queue: true
    # ...

  # Transition rules
  transitions:
    - from: draft
      to: ready_for_refinement_ba
      authorized_by: [product-manager]
    # ...

  # Constraints
  constraints:
    per_agent_type:
      business-analyst:
        max_concurrent: 2
      developer:
        max_concurrent: 5
      # ...
```

### Storage Location

```
~/.config/shark/workflows/
├── generic.workflow.yml       # Default workflow
├── wormwoodGM.workflow.yml    # Custom workflow
└── simple-dev.workflow.yml    # Other workflows
```

### Shark CLI Commands

```bash
# List workflows
shark workflow list

# Show workflow
shark workflow show wormwoodGM [--json]

# Validate workflow
shark workflow validate wormwoodGM

# Set default
shark workflow set-default wormwoodGM
```

### Orchestrator Integration

```go
// Load workflow on startup
workflow, err := loader.Load("wormwoodGM")

// Select tasks using workflow config
for _, stage := range workflow.Stages {
    queueStatus := stage.GetQueueStatus()
    agentType := stage.AgentType
    maxConcurrent := stage.Constraints.MaxConcurrent

    tasks := queryTasks(queueStatus)
    // Spawn agents with stage instructions
    spawnAgents(tasks, stage.AgentInstructions)
}

// Hot reload (check every 1 minute)
if workflowFileChanged() {
    workflow.Reload()
}
```

---

## What Information Does Orchestrator Need?

For each workflow stage:

1. **Agent Type**: Which agent handles this stage (`business-analyst`, `developer`, etc.)

2. **Status Mapping**:
   - Queue status: `ready_for_development`
   - Active status: `in_development`

3. **Agent Instructions**:
   - Base prompt for the agent
   - Required skills to invoke
   - Context to include (task description, prior stage outputs, etc.)
   - Success criteria to validate
   - Artifacts to create

4. **Transition Rules**:
   - Valid next statuses on success/failure/block
   - Prerequisites for transitions
   - Who can authorize transitions

5. **Constraints**:
   - Max concurrent tasks for this agent type
   - Timeout duration
   - Retry limits

6. **Context Propagation**:
   - What information flows from previous stages
   - Which artifacts from prior stages to include

---

## Configuration Schema Highlights

### Stage Definition

```yaml
stages:
  - name: development
    description: "Code implementation"
    agent_type: developer

    statuses:
      - ready_for_development  # Queue
      - in_development         # Active

    agent_instructions:
      base_prompt: |
        You are the Developer. Implement following TDD.
        Review specs from architect. Write tests first.

      skills_required:
        - test-driven-development
        - implementation
        - shark-task-management

      context_to_include:
        - task_description
        - business_requirements    # From BA stage
        - api_contracts           # From architect stage
        - data_models             # From architect stage
        - coding_standards

      success_criteria:
        - "All tests passing"
        - "Implementation complete"
        - "Code follows standards"

      artifacts_to_create:
        - "Source code files"
        - "Test files"

    transitions:
      on_success:
        - to: ready_for_code_review  # Default
      on_failure:
        - to: ready_for_refinement_ba   # Requirements gaps
        - to: ready_for_refinement_tech # Design gaps
      on_block:
        - to: blocked  # External dependency

    constraints:
      max_concurrent: 5
      timeout: "8h"
      retry_limit: 3
```

### Status Definition

```yaml
statuses:
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
```

### Transition Definition

```yaml
transitions:
  - from: in_development
    to: ready_for_code_review
    description: "Implementation complete"
    authorized_by: [developer]
    requires:
      - "All tests passing"
      - "Implementation complete"
      - "Code follows standards"

  - from: in_development
    to: ready_for_refinement_tech
    description: "Technical design gaps found"
    authorized_by: [developer]
    # No automatic transition - developer explicitly sends back
```

### Constraints

```yaml
constraints:
  global:
    max_concurrent_tasks: 100
    max_task_duration: "24h"
    enable_auto_block_on_timeout: true

  per_agent_type:
    business-analyst:
      max_concurrent: 2
      max_queue_size: 20
      priority_boost_after: "12h"

    developer:
      max_concurrent: 5
      max_queue_size: 50
      priority_boost_after: "24h"

    # ... other agent types

  resources:
    max_api_cost_per_hour: 50.00  # USD
    max_tokens_per_task: 1000000
```

---

## Benefits vs Hardcoded Approach

| Benefit | Impact |
|---------|--------|
| **Add new workflow** | 10x faster (YAML file vs code change) |
| **Modify stage instructions** | No rebuild/redeploy needed |
| **Understand workflow** | Self-documenting YAML |
| **Validate workflow** | Automated (`shark workflow validate`) |
| **Support multiple workflows** | Clean separation, no if/else in code |
| **Test workflow logic** | Unit tests on YAML vs integration tests |
| **Documentation** | Config IS documentation (no drift) |
| **Client customization** | Provide custom YAML (no fork needed) |

### Concrete Example: Add Security Review Stage

**Before (hardcoded):**
1. Modify orchestrator code
2. Add status mapping
3. Add agent instructions
4. Rebuild orchestrator
5. Test
6. Deploy

**After (config):**
1. Edit `wormwoodGM.workflow.yml`:
   ```yaml
   stages:
     - name: security_review
       agent_type: security-engineer
       statuses: [ready_for_security_review, in_security_review]
       # ... config
   ```
2. Validate: `shark workflow validate wormwoodGM`
3. Orchestrator auto-reloads within 1 minute

**Time savings: 6+ hours**

---

## Implementation Plan

### Phase 1: File-Based Config (Weeks 1-2)

1. Create workflow schema ✅ (this document)
2. Create `generic.workflow.yml` and `wormwoodGM.workflow.yml`
3. Add shark CLI commands:
   - `shark workflow list`
   - `shark workflow show <name>`
   - `shark workflow validate <name>`
4. Update orchestrator to read workflow config
5. Remove hardcoded mapping

### Phase 2: Per-Task Workflow (Weeks 3-4)

1. Add workflow columns to tasks table
2. Support `shark task create --workflow wormwoodGM`
3. Orchestrator handles mixed workflows

### Phase 3: Database-Backed (Future)

1. Store workflows in Shark DB
2. Add versioning and history
3. Import/export commands

### Phase 4: Advanced Features (Future)

- Workflow templates
- Workflow visualization (generate Mermaid diagrams)
- Workflow analytics
- Conditional transitions
- Dynamic agent instructions

---

## Integration with Orchestrator

### How Orchestrator Consumes Config

```go
// On startup
workflow := loadWorkflow("wormwoodGM")
workflow.Validate()

// Task selection loop (every 30s)
for _, stage := range workflow.Stages {
    // Get queue status and agent type from config
    queueStatus := stage.Statuses[0]  // e.g., "ready_for_development"
    agentType := stage.AgentType      // e.g., "developer"

    // Check concurrency limit from config
    maxConcurrent := workflow.GetAgentLimit(agentType)  // e.g., 5
    currentCount := countActiveAgents(agentType)

    if currentCount >= maxConcurrent {
        continue  // Skip, at capacity
    }

    // Query tasks
    tasks := shark.QueryTasks(queueStatus)

    // Filter by dependencies
    availableTasks := filterByDependencies(tasks)

    // Spawn agents with stage instructions from config
    for _, task := range availableTasks {
        spawnAgent(task, stage.AgentInstructions)
    }
}

// Hot reload check (every 1 minute)
if workflowFileChanged() {
    workflow.Reload()
    log.Info("Workflow config reloaded")
}
```

### Agent Spawning with Instructions

```go
func spawnAgent(task *Task, instructions AgentInstructions) {
    // Build context from config
    context := buildContext(task, instructions.ContextToInclude)

    // Construct prompt from config
    prompt := fmt.Sprintf(`%s

Task: %s
%s

Success Criteria:
%s

Artifacts to Create:
%s

Skills to Use: %s
`,
        instructions.BasePrompt,
        task.Title,
        context.Render(),
        strings.Join(instructions.SuccessCriteria, "\n- "),
        strings.Join(instructions.ArtifactsToCreate, "\n- "),
        strings.Join(instructions.SkillsRequired, ", "),
    )

    // Spawn agent
    agent := NewAgent(task.ID, agentType, prompt)
    agent.Start()
}
```

---

## Validation Example

```bash
$ shark workflow validate wormwoodGM

✅ Workflow 'wormwoodGM' is valid
  - 6 stages defined
  - 14 statuses defined
  - 42 transitions defined
  - All transitions reference valid statuses
  - All stages have agent types
  - Entry points: draft, ready_for_development
  - Terminal states: completed, cancelled
  - Special states: blocked, on_hold
  - Per-agent limits configured for 6 agent types
```

---

## Example Workflow Queries

```bash
# List all workflows
$ shark workflow list
Available workflows:
  generic (v1.0) - Simple todo → in_progress → completed
  wormwoodGM (v1.0) - BA → Arch → Dev → Review → QA → Approval
  simple-dev (v1.0) - Design → Dev → Review → QA → Deploy

# Show workflow details
$ shark workflow show wormwoodGM
Workflow: wormwoodGM (v1.0)
Description: WormwoodGM custom workflow with BA → Arch → Dev → Review → QA → Approval

Stages:
  1. business_analysis (business-analyst)
     - ready_for_refinement_ba → in_refinement_ba
     - Max concurrent: 2

  2. technical_design (architect)
     - ready_for_refinement_tech → in_refinement_tech
     - Max concurrent: 2

  3. development (developer)
     - ready_for_development → in_development
     - Max concurrent: 5

  # ... more stages

# Get workflow as JSON (for orchestrator)
$ shark workflow show wormwoodGM --json
{"workflow": {"name": "wormwoodGM", "version": "1.0", ...}}
```

---

## Migration Path

### Current State
```go
// Hardcoded in orchestrator
statusMapping := map[string]string{
    "ready_for_refinement_ba": "business-analyst",
    "ready_for_development": "developer",
    // ...
}
```

### Future State
```yaml
# ~/.config/shark/workflows/wormwoodGM.workflow.yml
stages:
  - name: business_analysis
    agent_type: business-analyst
    statuses: [ready_for_refinement_ba, in_refinement_ba]

  - name: development
    agent_type: developer
    statuses: [ready_for_development, in_development]
  # ...
```

```go
// Orchestrator reads from config
workflow := loader.Load("wormwoodGM")
stage := workflow.GetStageForStatus("ready_for_development")
agentType := stage.AgentType  // "developer"
```

---

## Key Takeaways

1. **Workflow config lives with Shark** (not orchestrator)
2. **Declarative YAML** defines stages, agents, instructions, transitions
3. **Hot reload** - no restart needed for workflow changes
4. **Self-documenting** - config is the documentation
5. **Multiple workflows** supported without code changes
6. **Validated automatically** before use
7. **Orchestrator consumes via `shark workflow show --json`**

---

## Next Steps

1. ✅ Review design (this document)
2. Create workflow YAML files (`generic.workflow.yml`, `wormwoodGM.workflow.yml`)
3. Implement Shark CLI workflow commands
4. Update orchestrator to consume workflow config
5. Test and deploy

---

**Full Design:** [shark-workflow-config-design.md](/home/jwwelbor/.claude/docs/architecture/shark-workflow-config-design.md)
