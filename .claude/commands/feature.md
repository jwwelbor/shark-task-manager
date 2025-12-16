---
description: Start feature refinement workflow from journey maps or epic
---

# Start Feature Refinement Workflow

Initiate the Feature Refinement workflow to elaborate features, create stories, design prototypes, and produce developer-ready packages.

## Usage

```bash
/feature
```

Or with a feature name:

```bash
/feature "User Authentication System"
```

## What This Does

This command:
1. Initializes the Feature-Refinement workflow state
2. Launches the Researcher agent for feature context gathering
3. Creates the foundational feature artifacts
4. Automatically progresses through the workflow:
   - Feature context research
   - Journey-to-feature decomposition
   - Experience and technical feasibility review
   - Scope approval
   - Parallel story elaboration and prototyping
   - Technical specification
   - Quality gates and stakeholder validation

## Workflow Integration

**Workflow Graph**: Feature-Refinement (02-feature-refinement.csv)
**Entry Node**: Feature_Context_Research
**First Agent**: Researcher
**Skills Used**: research, architecture, specification, design, quality

## Prerequisites

**Required artifacts** (should exist before running):
- D12-journey-maps.md - Customer journey maps from PDLC
- OR
- E01-epic.md - Epic-level requirements

**Optional context**:
- Existing architecture documentation
- Related feature implementations
- Design system or component library

## Implementation

The command:
1. Updates `/home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json` with Feature-Refinement workflow state
2. Sets current_node to "Feature_Context_Research"
3. Invokes the Researcher agent with context gathering task
4. Hooks automatically advance the workflow through all nodes

## Workflow Stages

### Stage 1: Research and Decomposition
- **Feature_Context_Research** (Researcher): Gather related features, architecture standards, prior decisions
- **Journey_To_Feature_Decomposition** (BusinessAnalyst): Break journey into discrete features
- **Experience_Alignment_Review** (CXDesigner): Validate features deliver intended experience
- **Technical_Feasibility_Review** (Architect): Assess viability, boundaries, and risks

### Stage 2: Scope Approval and Kickoff
- **Feature_Scope_Approval** (ProductManager): Confirm scope, priority, authorize elaboration
- **Story_And_Design_Start** (ProductManager): Kick off parallel story and design work

### Stage 3: Parallel Elaboration (Subgraphs)
- **Story-Elaboration-Subgraph**: Create detailed user stories
- **Prototyping-Subgraph**: Design wireframes and prototypes

### Stage 4: Technical Specification
- **Story_Design_Review** (ProductManager): Verify story and design alignment
- **Tech_Spec_Start** (TechLead): Kick off technical specification
- **Technical-Specification-Subgraph**: Create API contracts, data models, flow diagrams

### Stage 5: Quality Gates
- **Technical_Review** (Architect): Review specs for completeness and standards
- **Test_Criteria_Definition** (QA): Define test cases, edge cases, quality gates
- **Artifact_Review** (TechLead): Review completeness and developer-readiness

### Stage 6: Approval and Completion
- **Stakeholder_Validation** (Human): Final business approval
- **Feature_Refinement_Complete**: Terminal node, outputs developer-ready package

## Output Artifacts

### Context and Planning
- F01-feature-context.md
- F02-related-decisions.md
- F03-standards-reference.md
- F04-feature-list.md
- F05-initial-criteria.md
- F06-dependencies.md

### Review and Approval
- F07-experience-validation.md
- F08-journey-coherence.md
- F09-feasibility-assessment.md
- F10-technical-risks.md
- F11-boundaries.md
- F12-approved-scope.md
- F13-priority-matrix.md

### Elaboration
- F14-elaboration-kickoff.md
- S-refined-stories.md (from Story-Elaboration subgraph)
- P-prototypes/* (from Prototyping subgraph)
- F15-alignment-review.md
- F16-discrepancy-resolution.md

### Technical Specification
- F17-spec-kickoff.md
- T-api-contracts.md (from Tech-Spec subgraph)
- T-data-models.md
- T-flow-diagrams.md
- F18-spec-review.md
- F19-standards-compliance.md

### Quality and Readiness
- F20-test-criteria.md
- F21-edge-cases.md
- F22-quality-gates.md
- F23-readiness-review.md
- F24-dev-package.md
- F25-stakeholder-approval.md

### Final Output
- F-developer-ready-package/* (Complete package for development)

## Monitoring Progress

Check workflow status:
```bash
cat /home/jwwelbor/projects/ai-dev-team/docs/workflow/state.json
```

View created artifacts:
```bash
ls /home/jwwelbor/projects/ai-dev-team/docs/workflow/artifacts/
```

## Skills Used

**Primary Skills**:
- `research` - Feature context and related decisions
- `specification-writing` - Stories, PRDs, documentation
- `design` - Wireframes, prototypes, journey validation
- `architecture` - Feasibility, technical specs, reviews
- `quality` - Test criteria, reviews, validation gates
- `orchestration` - Workflow coordination and parallel execution

## Example Session

```
User: /feature "Multi-factor Authentication"

System: [Initializes Feature-Refinement workflow]
System: [Launches Researcher agent]

Researcher: I'll gather context for the Multi-factor Authentication feature...
[Searches codebase for existing auth patterns, security standards, related features]
[Creates F01-feature-context.md, F02-related-decisions.md, F03-standards-reference.md]

System: [artifact-watcher hook detects artifacts]
System: [workflow-router advances to Journey_To_Feature_Decomposition]
System: [BusinessAnalyst agent launches]

BusinessAnalyst: Based on the journey maps, I'll decompose this into discrete features...
[Creates F04-feature-list.md, F05-initial-criteria.md, F06-dependencies.md]

[Workflow continues automatically through all stages...]

ProductManager: [At Feature_Scope_Approval]
Reviewing feasibility assessment and feature list...
Approving scope for: SMS MFA, Authenticator App MFA, Backup Codes
[Creates F12-approved-scope.md, F13-priority-matrix.md]

ProductManager: [At Story_And_Design_Start]
Launching parallel story elaboration and prototyping workflows...
[Launches Story-Elaboration-Subgraph and Prototyping-Subgraph concurrently]

[Both subgraphs complete...]

ProductManager: [At Story_Design_Review]
Reviewing alignment between stories and prototypes...
[Creates F15-alignment-review.md]

[Workflow continues through technical specification, quality gates...]

TechLead: [At Artifact_Review]
Developer-ready package is complete. All artifacts verified.
[Creates F23-readiness-review.md, F24-dev-package.md]