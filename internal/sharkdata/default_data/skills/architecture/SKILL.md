---
name: architecture
description: Authoritative source for all architecture design workflows (System, Backend, Frontend, Database, Security). Provides standardized patterns, templates, and design frameworks for consistent architecture documentation across all projects.
version: 1.0.0
created: 2025-12-09
inputs:
  - feature_prd_path: absolute path to the feature PRD (provides scope and requirements)
  - epic_prd_path: absolute path to the epic PRD (optional, provides broader context)
  - task_spec_path: absolute path to the task spec (optional, specific architecture task)
  - existing_architecture_path: absolute path to existing architecture documentation (optional)
  - constraints_doc_path: absolute path to constraints and limitations document (optional)
outputs:
  - selected_workflow: one of {design-system, design-backend, design-frontend, design-database, design-security, design-compliance, feasibility-review, infra-requirements}
  - architecture_documents: list of paths to created/modified architecture documents
  - design_decisions: structured record of key design choices and rationale
  - trade-offs: documented trade-offs with analysis of alternatives
  - risks_and_mitigations: identified architectural risks and mitigation strategies
---

# Architecture Skill

This skill provides software architecture capabilities including system design, backend architecture, frontend architecture, data modeling, and security design. It delivers standardized workflows, patterns, and templates for creating comprehensive architecture documentation.

For task-level architecture work, also refer to `specification-writing/workflows/refine-task-requirements.md` for the architect-specific workflow.

## Workflow Selection

Based on what the user needs, invoke the appropriate workflow:

### System Architecture
**When**: User needs high-level system architecture design across all components
**Invoke**: `workflows/design-system.md`
**Output**: System architecture document (02-architecture.md)
**Used by**: the `architect` agent (system design)

### Backend Architecture
**When**: User needs API, service layer, or backend component design
**Invoke**: `workflows/design-backend.md`
**Output**: Backend design documents (02-architecture.md, 04-backend-design.md)
**Used by**: the `architect` agent (backend design)

### Frontend Architecture
**When**: User needs frontend component, state management, or UI architecture design
**Invoke**: `workflows/design-frontend.md`
**Output**: Frontend design document (05-frontend-design.md)
**Used by**: the `architect` agent (frontend design)
**References**: frontend-design skill for UI component aesthetics

### Database Architecture
**When**: User needs data modeling, schema design, or persistence strategy
**Invoke**: `workflows/design-database.md`
**Output**: Data design document (03-data-design.md)
**Used by**: the `architect` agent (database design)

### Security Architecture
**When**: User needs security, authentication, authorization, or compliance design
**Invoke**: `workflows/design-security.md`
**Output**: Security design document (06-security-design.md)
**Used by**: the `architect` agent (security design)

## Pattern Resources

Common architectural patterns are documented in `context/patterns/`:

- `api-patterns.md` - REST API design, GraphQL patterns, versioning
- `data-patterns.md` - Data modeling, schema design, migration strategies
- `security-patterns.md` - Authentication, authorization, RLS, encryption
- `integration-patterns.md` - Service integration, event-driven, messaging

## Template Resources

All workflows reference these templates for consistent document structure:

- `architecture-doc.md` - System architecture document template (02-architecture.md)
- `api-spec-doc.md` - API/Backend specification template (04-backend-design.md)
- `frontend-doc.md` - Frontend design template (05-frontend-design.md)
- `database-doc.md` - Data design template (03-data-design.md)
- `security-doc.md` - Security design template (06-security-design.md)

## Integration Points

### Depends On
- **specification-writing skill**: Uses document generation patterns and naming conventions
- **frontend-design skill**: Referenced by frontend workflow for UI component design

### Used By
- **`architect` agent**: Invokes these workflows for feature architecture — `design-system` for system design, `design-backend` for backend/API, `design-frontend` for frontend, `design-database` for data modeling, and `design-security` for security. A single feature may orchestrate several of them.

## Usage Pattern

Agents reference this skill using:

```markdown
## Your Process
1. Analyze feature requirements from PRD
2. Review project research report (00-research-report.md)
3. Invoke architecture skill workflow: architecture/workflows/design-{domain}.md
4. Apply patterns from: architecture/context/patterns/{pattern-type}.md
5. Use template: architecture/context/templates/{template-name}.md
6. Generate documentation at /docs/plan/{epic-key}/{feature-key}/
```

## Quality Standards

All architecture documents must:
- Be design specifications, NOT implementation code
- Use prose descriptions and Mermaid diagrams, not code blocks
- Follow the NO CODE principle (describe interfaces, don't implement them)
- Align with existing project patterns from research reports
- Include concrete specifications (field names, types, validation rules)
- Consider all WAF pillars: Security, Reliability, Performance, Cost, Operations
- Document trade-offs and design decisions explicitly
- Cross-reference related architecture documents

## Architecture Principles

1. **Contract-First**: Define interfaces before implementation
2. **Consistency**: Follow patterns from project research reports
3. **Completeness**: All required sections must be documented
4. **Clarity**: Specifications must be unambiguous and actionable
5. **Integration**: Backend, frontend, data, and security designs must align
6. **Scalability**: Consider growth and evolution from the start
7. **Security**: Security is not an afterthought, it's built into design

---

*For detailed usage instructions, see [README.md](./README.md)*
