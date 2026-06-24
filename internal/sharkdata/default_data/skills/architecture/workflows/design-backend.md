---
inputs:
  - feature_prd_path: absolute path to the feature PRD markdown
  - research_report_path: absolute path to the research report (00-research-report.md)
  - interface_contracts: list of DTO/interface contracts defined by coordinator (optional)
  - data_design_path: absolute path to 03-data-design.md (optional, if db design exists)
  - architecture_doc_path: absolute path where 02-architecture.md should be written
  - backend_design_path: absolute path where 04-backend-design.md should be written
  - api_patterns_path: absolute path to api-patterns.md context
  - integration_patterns_path: absolute path to integration-patterns.md context
  - architecture_template_path: absolute path to architecture-doc.md template
  - api_spec_template_path: absolute path to api-spec-doc.md template
outputs:
  - architecture_doc: structured markdown written to architecture_doc_path (system architecture)
  - backend_design_doc: structured markdown written to backend_design_path (interface specification)
  - open_questions: list of unresolved decisions / assumptions / risks needing user input
  - design_decisions: structured list of {decision, rationale, alternatives, trade_offs}
---

# Backend Architecture Design Workflow (craft)

This workflow guides you through creating comprehensive backend architecture documentation for a feature. It produces two documents: system architecture (02-architecture.md) and detailed backend interface specification (04-backend-design.md).

## CRITICAL: No Implementation Code

This workflow produces DESIGN SPECIFICATIONS, not code. Every section must use prose, tables, and Mermaid diagrams — NEVER programming language code blocks.

- NEVER include Go, Python, TypeScript, SQL, or any language code blocks
- Describe method signatures in prose: "Method X accepts parameters A (string) and B (context), returns C and error"
- Describe data structures as tables with columns: Field, Type, Required, Description
- Describe logic flows as numbered prose steps or Mermaid sequence diagrams
- The ONLY acceptable code fences are ```mermaid for diagrams
- If you catch yourself writing a code block, STOP and rewrite it as prose or a table
- Target 150-200 lines per document. Over 250 lines means you are over-specifying.
- Use standard file names: 02-architecture.md and 04-backend-design.md (not custom names)

The implementing developer agent decides HOW to code it. You describe WHAT the interfaces are, WHY they exist, and what CONTRACTS they must satisfy.

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

### Create the system architecture document

Use the template referenced by `architecture_template_path`.

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

### Create the backend interface document

Use the template referenced by `api_spec_template_path`.

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
- Apply patterns from `api_patterns_path`

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
- Apply patterns from `api_patterns_path`

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
- Apply patterns from `integration_patterns_path`

#### Error Handling
- Error codes and meanings
- Error response format (standardized structure)
- Recovery strategies

#### Pagination (if applicable)
- Pattern type (page-based, cursor-based, offset-based)
- Parameters with defaults and limits
- Apply patterns from `api_patterns_path`

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

### API Patterns (`api_patterns_path`)
- RESTful resource design
- Error handling standards
- Pagination approaches
- Versioning strategies
- Request validation
- Response formatting

### Integration Patterns (`integration_patterns_path`)
- Service communication (REST, gRPC, events)
- Message queue patterns
- Event-driven architecture
- API gateway patterns

## Step 5: Ensure Cross-Domain Alignment

### Align with Data Design
If `data_design_path` is provided:
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
- [ ] System architecture document is 150-200 lines
- [ ] Backend design document is 150-200 lines
- [ ] All DTOs are fully specified
- [ ] All interfaces have complete specifications
- [ ] All integration points are documented

### NO CODE Constraint (MANDATORY — document fails review if any of these are violated)
- [ ] No Python/Node.js/Go/Rust/Java code blocks anywhere in the document
- [ ] No SQL statements or DDL
- [ ] No class, struct, interface, or function definitions in code syntax
- [ ] No code fences except ```mermaid for diagrams
- [ ] All interfaces described in prose and tables only
- [ ] All data structures described as field/type/description tables
- [ ] Method signatures described in natural language, not code syntax

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

## Step 7: Write Documents

Write the system architecture markdown to `architecture_doc_path`.
Write the backend interface specification markdown to `backend_design_path`.

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

## MANDATORY: Interactive Review of Open Questions

After generating the document(s), you MUST surface any open questions, unresolved decisions, concerns, or assumptions to the user **before proceeding to the next workflow step**. Do NOT silently move on.

### Process

1. **Scan** the completed document for:
   - Open questions or items needing stakeholder input
   - Unresolved design decisions with trade-offs
   - Assumptions that need validation
   - Risks or concerns that require discussion
   - Items marked as "TBD", "TODO", or "deferred"
   - Anything in the "Open Questions & Decisions Required" section

2. **Present** a structured summary to the user:
   - List each item clearly with its context
   - For decisions, present options with trade-offs
   - For assumptions, state what you assumed and ask for confirmation
   - For risks, explain impact and ask for guidance

3. **Walk through** each item interactively:
   - Discuss one item at a time
   - Record the resolution in the document
   - Update the document with decisions made

4. **Only proceed** when all items are resolved or explicitly deferred by the user.

If there are no open questions, confirm this explicitly: "All design decisions are resolved. No open questions remain."

---

## Output Requirements

Upon completion, you will have:
1. **System architecture document** at `architecture_doc_path` — diagrams, components, flows
2. **Backend interface specification** at `backend_design_path` — complete interface specification
3. Both documents following templates exactly
4. All sections complete and comprehensive
5. NO implementation code, only design specifications
6. Cross-domain alignment verified
7. All open questions reviewed interactively and resolved
