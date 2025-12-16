# Backend Architecture Design Workflow

This workflow guides you through creating comprehensive backend architecture documentation for a feature. It produces two documents: system architecture (02-architecture.md) and detailed backend interface specification (04-backend-design.md).

## Prerequisites

Before starting this workflow, ensure you have:
1. Feature PRD at `/docs/plan/{epic-key}/{feature-key}/prd.md`
2. Research report at `/docs/plan/{epic-key}/{feature-key}/00-research-report.md`
3. Interface contracts (if defined by feature-architect)
4. Data design (03-data-design.md) if created by db-admin

## Step 1: Analyze Requirements

### Read the PRD
- Identify what backend interfaces the feature requires (API, library, CLI, service)
- Extract functional requirements
- Note non-functional requirements (performance, scalability, reliability)
- Identify integration points with other systems

### Read the Research Report
- Understand existing backend patterns (framework, architecture style)
- Note naming conventions (snake_case, PascalCase, etc.)
- Identify existing similar interfaces to extend or reference
- Review technology stack in use

### Review Interface Contracts (if provided)
- Understand the DTOs and interfaces defined by coordinator
- Note required request/response structures
- Identify integration requirements with frontend/other services

## Step 2: Define System Architecture

### Create 02-architecture.md

Use the template from `context/templates/architecture-doc.md`

**Key sections to complete:**

#### Architecture Overview
- Brief 2-3 sentence description of backend scope
- Key design decisions with rationale
- Primary architectural patterns being used

#### System Architecture Diagram
Create a Mermaid diagram showing:
- Client layer (web, CLI, API consumers)
- Application layer (services, workers)
- Data layer (database, cache, files)
- External services (third-party APIs)
- All interactions between layers

#### Component Details
For each major backend component:
- **Purpose**: What it does
- **Responsibilities**: Key functions it performs
- **Dependencies**: What it needs
- **Interfaces**: What it exposes and consumes

#### Data Flow
Create Mermaid sequence diagrams for:
- Primary user flows
- Key backend operations
- Integration flows with external services

#### Integration Points
Document:
- Internal service integrations (REST, gRPC, events, function calls)
- External service integrations (APIs, with fallback strategies)

#### Technology Stack
Table showing:
- Layer (API, Service, Data, etc.)
- Technology choice
- Justification for the choice

#### Technical Risks & Mitigations
Identify:
- Potential scalability bottlenecks
- Integration risks
- Performance concerns
- Mitigation strategies for each

## Step 3: Design Backend Interface

### Create 04-backend-design.md

Use the template from `context/templates/api-spec-doc.md`

The format of this document depends on what type of interface the feature provides:
- **API**: Focus on endpoints, DTOs, request/response specs
- **Library**: Focus on functions, classes, method signatures
- **CLI**: Focus on commands, arguments, options
- **Service**: Focus on messages, events, contracts

**Key sections to complete:**

#### Interface Overview
- Brief description of what interfaces this feature provides
- Interface type (API / Library / CLI / Service / Mixed)

#### Codebase Analysis
- Existing related interfaces found in research
- Naming patterns to follow
- Decision: extend existing code or create new?

#### DTO / Data Structures
For each data structure:
- **Purpose**: What it represents and when it's used
- **Fields**: Complete table with field, type, required, validation, description
- Apply patterns from `context/patterns/api-patterns.md`

#### Interface Specifications

**For API endpoints:**
- Method and path
- Purpose and use case
- Authentication and authorization requirements
- Parameters (path, query, body)
- Request body DTO reference
- Response DTO and status code
- Processing steps (logic flow)
- Error responses (all possible error conditions)
- Apply patterns from `context/patterns/api-patterns.md`

**For Library interfaces:**
- Module and function name
- Purpose
- Signature (parameters with types, defaults)
- Return type and description
- Exceptions/errors raised
- Behavior description

**For CLI commands:**
- Command name and subcommands
- Purpose
- Usage syntax
- Arguments and options
- Output format
- Exit codes

**For Service contracts:**
- Event/message name
- Trigger condition
- Payload structure
- Consumers
- Apply patterns from `context/patterns/integration-patterns.md`

#### Error Handling
- Error codes and meanings
- Error response format (standardized structure)
- Recovery strategies

#### Pagination (if applicable)
- Pattern type (page-based, cursor-based, offset-based)
- Parameters with defaults and limits
- Apply patterns from `context/patterns/api-patterns.md`

#### Rate Limiting (if applicable)
- Per-interface limits
- Time windows
- Throttling behavior

#### Versioning
- Strategy (URL-based, header-based)
- Current version
- Deprecation policy

## Step 4: Apply Architecture Patterns

Review and apply relevant patterns from:

### API Patterns (`context/patterns/api-patterns.md`)
- RESTful resource design
- Error handling standards
- Pagination approaches
- Versioning strategies
- Request validation
- Response formatting

### Integration Patterns (`context/patterns/integration-patterns.md`)
- Service communication (REST, gRPC, events)
- Message queue patterns
- Event-driven architecture
- API gateway patterns

## Step 5: Ensure Cross-Domain Alignment

### Align with Data Design
If 03-data-design.md exists:
- Ensure DTOs map cleanly to/from database entities
- Document any transformations needed
- Verify field types are compatible

### Align with Interface Contracts
If coordinator provided contracts:
- Ensure all DTOs match the contract specifications
- Verify request/response structures align
- Maintain naming consistency

### Consider Security
Note security requirements for later security-architect review:
- Authentication requirements
- Authorization rules
- Data protection needs
- Input validation requirements

## Step 6: Quality Checklist

Before finalizing, verify:

### Completeness
- [ ] All required template sections are filled
- [ ] 02-architecture.md is 150-200 lines
- [ ] 04-backend-design.md is 150-200 lines
- [ ] All DTOs are fully specified
- [ ] All interfaces have complete specifications
- [ ] All integration points are documented

### NO CODE Constraint
- [ ] No Python/Node.js/Go code blocks
- [ ] No SQL statements
- [ ] No class or function definitions
- [ ] Only prose descriptions and specifications
- [ ] Mermaid diagrams for visualization only

### Consistency
- [ ] Follows patterns from research report
- [ ] Naming matches project conventions
- [ ] DTOs align with interface contracts
- [ ] Backend design aligns with data design

### Clarity
- [ ] Every interface has exact specifications
- [ ] All fields have types and validation rules
- [ ] Error conditions are documented
- [ ] Processing logic is described clearly

### Integration
- [ ] Cross-references to related documents
- [ ] Integration points clearly specified
- [ ] Dependencies documented
- [ ] Contracts align across domains

## Step 7: Create Documents

### File Locations
- Create `/docs/plan/{epic-key}/{feature-key}/02-architecture.md`
- Create `/docs/plan/{epic-key}/{feature-key}/04-backend-design.md`

### Review
- Verify both files are complete
- Check all cross-references are valid
- Ensure diagrams render correctly
- Confirm no code blocks exist

## Common Patterns to Apply

### RESTful API Design
- Use resource-oriented URLs (nouns, not verbs)
- HTTP methods for operations (GET, POST, PUT, PATCH, DELETE)
- Consistent response formats
- Standard error structures

### Interface Naming
- Follow project conventions from research report
- Be consistent within the feature
- Use descriptive, unambiguous names

### Error Handling
- Comprehensive error codes
- Helpful error messages
- Consistent error structure
- Recovery guidance

### Documentation
- Every public interface fully specified
- All edge cases documented
- Integration requirements clear
- Dependencies explicit

## Output Requirements

Upon completion, you will have:
1. **02-architecture.md** - System architecture with diagrams, components, flows
2. **04-backend-design.md** - Complete backend interface specification
3. Both documents following templates exactly
4. All sections complete and comprehensive
5. NO implementation code, only design specifications
6. Cross-domain alignment verified
