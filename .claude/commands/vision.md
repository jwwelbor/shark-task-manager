---
description: Start the PDLC workflow from client vision input
---

# Start PDLC Workflow

Initiate the Product Development Lifecycle (PDLC) workflow by capturing the client's vision and desired outcomes.

## Usage

```bash
/vision
```

Or with an initial vision statement:

```bash
/vision "Build a mobile app for meal planning with AI recommendations"
```

## What This Does

This command:
1. Initializes the PDLC workflow state
2. Launches the Client agent to define product vision
3. Creates the foundational artifacts:
   - D01-vision-statement.md
   - D02-success-criteria.md
4. Automatically triggers the next workflow step (Ideation_Brainstorming)

## Workflow Integration

**Workflow Graph**: PDLC (01-pdlc.csv)
**Entry Node**: Product_Vision_Definition
**First Agent**: Client
**Skills Used**: discovery, brainstorming

## Implementation

The command:
1. Updates `/home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json` with PDLC workflow state
2. Sets current_node to "Product_Vision_Definition"
3. Invokes the Client agent with vision definition task
4. Hooks automatically advance the workflow when artifacts are created

## Subsequent Steps

After you complete the vision artifacts, the workflow will automatically:
1. Advance to Ideation_Brainstorming (ProductManager agent)
2. Continue through Market_And_Feasibility_Research (Researcher agent)
3. Progress through the full PDLC workflow graph

## Monitoring Progress

Check workflow status at any time:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json
```

View created artifacts:
```bash
ls /home/jwwelbor/projects/ai-dev-team/docs/workflow/artifacts/
```

## Skills

**Primary Skill**: `orchestration/workflows/start-workflow.md`
**Supporting Skills**:
- `discovery/` - Research and context gathering
- `brainstorming/` - Ideation and solution exploration
- `specification-writing/` - Document creation

## Example Session

```
User: /vision "Create a platform for freelance developers to find projects"

System: [Initializes PDLC workflow]
System: [Launches Client agent]

Client Agent: I'll help you define the product vision. Let me ask a few questions to understand the opportunity:

1. What specific problem are freelance developers facing that this platform will solve?
2. Who are the key stakeholders (freelancers, companies, platform admins)?
3. What does success look like in 6 months? In 1 year?

[User provides answers]

Client Agent: [Creates D01-vision-statement.md and D02-success-criteria.md]

System: [artifact-watcher hook detects artifacts]
System: [workflow-router advances to Ideation_Brainstorming]
System: [ProductManager agent launches]

ProductManager Agent: Based on the vision, let's brainstorm solution approaches...
```

## Related Commands

- `/feature` - Start feature refinement workflow
- `/develop` - Start development workflow
- `/release` - Start release workflow

## Prerequisites

- Workflow infrastructure initialized (state.json, hooks configured)
- Client agent available
- Orchestration skill installed

## Notes

- This is the entry point for the entire PDLC workflow
- The workflow is automated via hooks - agents hand off to each other
- You can interrupt and resume workflows by checking state.json
- All artifacts are versioned and tracked in workflow history
