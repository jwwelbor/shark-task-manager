# Architecture Skill

**Version**: 1.0.0
**Created**: 2025-12-09
**Domain**: Architecture Design & Documentation

## Overview

The Architecture skill is the authoritative source for all technical design workflows in the Claude Code system. It consolidates architecture knowledge from multiple specialized domains (backend, frontend, database, security, system) into a unified framework that ensures consistent, comprehensive, and well-integrated architecture documentation.

## What This Skill Provides

### 1. Architecture Workflows

Five core workflows for different architecture domains:

- **design-system.md** - System-level architecture for complex multi-component systems
- **design-backend.md** - Backend services, APIs, and business logic design
- **design-frontend.md** - Frontend components, state management, and UX patterns
- **design-database.md** - Data modeling, schema design, and persistence strategies
- **design-security.md** - Security, authentication, authorization, and compliance

### 2. Architecture Patterns

Reusable patterns for common architectural challenges:

- **API Patterns** - REST, GraphQL, versioning, pagination, error handling
- **Data Patterns** - Normalization, relationships, migrations, indexing
- **Security Patterns** - Authentication, authorization, encryption, RLS
- **Integration Patterns** - Service communication, event-driven, messaging

### 3. Document Templates

Standardized templates for architecture documentation:

- **architecture-doc.md** - System architecture (02-architecture.md)
- **api-spec-doc.md** - API/Backend specification (04-backend-design.md)
- **frontend-doc.md** - Frontend design (05-frontend-design.md)
- **database-doc.md** - Data design (03-data-design.md)
- **security-doc.md** - Security design (06-security-design.md)

### 4. Reference Examples

Example architectures demonstrating best practices and common patterns.

## When to Use This Skill

### For Agents

Architect agents (backend-architect, frontend-architect, db-admin, security-architect, principal-architect) invoke this skill to:
- Access architecture workflow procedures
- Apply standard patterns to new features
- Use templates for consistent documentation
- Ensure designs align across domains

### For Users

Users can reference this skill when:
- Learning architecture patterns used in the project
- Understanding document structure expectations
- Reviewing architecture best practices
- Creating custom architecture workflows

## Workflow Invocation

### System Architecture

```markdown
Invoke: skills/architecture/workflows/design-system.md

Use Case: High-level system design across multiple components
Agent: principal-architect
Output: 02-architecture.md
```

### Backend Architecture

```markdown
Invoke: skills/architecture/workflows/design-backend.md

Use Case: API, service layer, business logic design
Agent: backend-architect
Output: 02-architecture.md, 04-backend-design.md
Patterns: api-patterns.md
Template: api-spec-doc.md
```

### Frontend Architecture

```markdown
Invoke: skills/architecture/workflows/design-frontend.md

Use Case: Component hierarchy, state management, UX patterns
Agent: frontend-architect
Output: 05-frontend-design.md
References: frontend-design skill for UI aesthetics
Template: frontend-doc.md
```

### Database Architecture

```markdown
Invoke: skills/architecture/workflows/design-database.md

Use Case: Data modeling, schema design, persistence strategy
Agent: db-admin
Output: 03-data-design.md
Patterns: data-patterns.md
Template: database-doc.md
```

### Security Architecture

```markdown
Invoke: skills/architecture/workflows/design-security.md

Use Case: Security, auth, authorization, compliance
Agent: security-architect
Output: 06-security-design.md
Patterns: security-patterns.md
Template: security-doc.md
```

## Directory Structure

```
architecture/
├── SKILL.md                        # Router for workflow selection
├── README.md                       # This file
├── workflows/
│   ├── design-system.md           # System-level architecture workflow
│   ├── design-backend.md          # Backend services & APIs workflow
│   ├── design-frontend.md         # Frontend architecture workflow
│   ├── design-database.md         # Data modeling & schema workflow
│   └── design-security.md         # Security architecture workflow
├── context/
│   ├── patterns/
│   │   ├── api-patterns.md        # REST, GraphQL, API design patterns
│   │   ├── data-patterns.md       # Data modeling patterns
│   │   ├── security-patterns.md   # Auth, RLS, encryption patterns
│   │   └── integration-patterns.md # Service integration approaches
│   └── templates/
│       ├── architecture-doc.md     # System architecture template
│       ├── database-doc.md         # Database design template
│       ├── api-spec-doc.md         # API specification template
│       ├── frontend-doc.md         # Frontend design template
│       └── security-doc.md         # Security design template
└── examples/
    └── reference-architectures.md  # Example architectures
```

## Key Principles

### 1. No Code in Design Documents

Architecture documents describe WHAT to build, not HOW to build it:
- Use prose descriptions, not code blocks
- Specify interfaces, not implementations
- Use Mermaid diagrams for visualization
- Define DTOs as tables (field, type, validation), not as code

### 2. Contract-First Design

Define interfaces before implementation:
- API endpoints specify exact request/response structures
- DTOs define all fields with types and constraints
- Components define props and events explicitly
- Services define function signatures and behaviors

### 3. Cross-Domain Consistency

All architecture domains must align:
- Backend DTOs match frontend component props
- Database schemas support backend interfaces
- Security policies apply across all layers
- Integration patterns are consistent

### 4. Pattern Reuse

Apply standard patterns before inventing new ones:
- Check pattern files for common solutions
- Reference existing project patterns from research reports
- Document new patterns when they emerge
- Share patterns across features

## Integration with Other Skills

### Specification Writing Skill

Architecture skill uses specification-writing for:
- Document naming conventions
- File organization standards
- Cross-referencing patterns

### Frontend Design Skill

Frontend architecture workflow references frontend-design skill for:
- UI component visual design
- Production-grade aesthetics
- Modern design patterns

### Quality Skill

Quality skill validates architecture documents:
- Checks all required sections exist
- Validates cross-references
- Ensures completeness

## Document Output Locations

All architecture documents are created in the feature's plan directory:

```
/docs/plan/{epic-key}/{feature-key}/
├── 02-architecture.md        # System architecture (system/backend workflows)
├── 03-data-design.md          # Data design (database workflow)
├── 04-backend-design.md       # Backend specification (backend workflow)
├── 05-frontend-design.md      # Frontend design (frontend workflow)
└── 06-security-design.md      # Security design (security workflow)
```

## Quality Standards

Every architecture document must:
1. Follow the template structure completely
2. Be 150-200 lines (target range)
3. Include all required sections
4. Use Mermaid diagrams for architecture visualization
5. Specify interfaces with complete detail
6. Document design decisions and trade-offs
7. Align with project conventions from research reports
8. Cross-reference related documents

## Validation

Architecture documents are validated by the quality skill using:
- Structural validation (all sections present)
- Content validation (specifications are complete)
- Cross-reference validation (links are valid)
- Pattern alignment (matches project conventions)

## Maintenance

### Updating Patterns

When new architecture patterns emerge:
1. Document in appropriate pattern file
2. Include: name, purpose, when to use, implementation, trade-offs
3. Add examples from real features
4. Update reference architectures if significant

### Updating Templates

When document structure changes:
1. Update template in context/templates/
2. Update corresponding workflow
3. Document change rationale
4. Update validation criteria in quality skill

### Adding Workflows

When new architecture domains emerge:
1. Create workflow in workflows/
2. Add to SKILL.md router
3. Create or reference appropriate patterns
4. Create template if needed
5. Update this README

## Example Usage

### Backend Architect Creating API Design

```markdown
## Your Process
1. Read PRD at /docs/plan/{epic}/{feature}/prd.md
2. Read research report at /docs/plan/{epic}/{feature}/00-research-report.md
3. Invoke architecture/workflows/design-backend.md
4. Apply patterns from architecture/context/patterns/api-patterns.md
5. Use template from architecture/context/templates/api-spec-doc.md
6. Create 02-architecture.md and 04-backend-design.md
7. Validate using quality skill
```

### DB Admin Creating Data Model

```markdown
## Your Process
1. Read PRD and interface contracts
2. Invoke architecture/workflows/design-database.md
3. Apply patterns from architecture/context/patterns/data-patterns.md
4. Use template from architecture/context/templates/database-doc.md
5. Create 03-data-design.md
6. Ensure DTOs align with interface contracts
```

## Extracted From

This skill consolidates architecture knowledge from:
- backend-architect agent (385 lines → workflow + patterns)
- frontend-architect agent (459 lines → workflow + patterns)
- db-admin agent (207 lines → workflow + patterns)
- security-architect agent (230 lines → workflow + patterns)
- principal-architect agent (177 lines → workflow + patterns)

## Version History

- **1.0.0** (2025-12-09): Initial extraction from architect agents as part of E01-F01-P03
