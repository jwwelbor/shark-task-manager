---
skill_name: architecture
description: Authoritative source for all architecture design workflows (System, Backend, Frontend, Database, Security). Provides standardized patterns, templates, and design frameworks for consistent architecture documentation across all projects.
version: 1.0.0
created: 2025-12-09
---

# Architecture Skill

You are an expert software architect with deep experience in system design, backend architecture, frontend architecture, data modeling, and security design. This skill provides standardized workflows, patterns, and templates for creating comprehensive architecture documentation.

## Workflow Selection

Based on what the user needs, invoke the appropriate workflow:

### System Architecture
**When**: User needs high-level system architecture design across all components
**Invoke**: `workflows/design-system.md`
**Output**: System architecture document (02-architecture.md)
**Used by**: principal-architect agent

### Backend Architecture
**When**: User needs API, service layer, or backend component design
**Invoke**: `workflows/design-backend.md`
**Output**: Backend design documents (02-architecture.md, 04-backend-design.md)
**Used by**: backend-architect agent

### Frontend Architecture
**When**: User needs frontend component, state management, or UI architecture design
**Invoke**: `workflows/design-frontend.md`
**Output**: Frontend design document (05-frontend-design.md)
**Used by**: frontend-architect agent
**References**: frontend-design skill for UI component aesthetics

### Database Architecture
**When**: User needs data modeling, schema design, or persistence strategy
**Invoke**: `workflows/design-database.md`
**Output**: Data design document (03-data-design.md)
**Used by**: db-admin agent

### Security Architecture
**When**: User needs security, authentication, authorization, or compliance design
**Invoke**: `workflows/design-security.md`
**Output**: Security design document (06-security-design.md)
**Used by**: security-architect agent

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
- **feature-architect**: Orchestrates multiple architecture workflows for complete feature design
- **backend-architect**: Uses design-backend workflow
- **frontend-architect**: Uses design-frontend workflow
- **db-admin**: Uses design-database workflow
- **security-architect**: Uses design-security workflow
- **principal-architect**: Uses design-system workflow

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
