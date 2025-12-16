# Design Validation Criteria

This document defines the complete validation criteria for feature design documents.

## Required Files

All features must have these design documents in `/docs/plan/{epic}/{feature}/`:

1. `README.md` - Navigation hub and overview
2. `prd.md` - Feature PRD
3. `00-research-report.md` - Project research findings
4. `01-api-contracts.md` - Shared DTOs and endpoint contracts
5. `02-architecture.md` - System design and integration
6. `03-database-design.md` - Schema and data model
7. `04-api-specification.md` - API endpoints and contracts
8. `05-frontend-design.md` - UI components and state
9. `06-security-performance.md` - Security and optimization
10. `07-implementation-phases.md` - Timeline and phases
11. `08-test-criteria.md` - Test specifications
12. `prps/README.md` - PRP placeholder

## Section Requirements

### README.md
- [ ] Contains "Feature Overview" section (3-5 sentences)
- [ ] Contains "Quick Links" section with links to all docs
- [ ] Contains "Implementation Status" table
- [ ] Contains "Key Dependencies" section
- [ ] Contains "Success Metrics" section
- [ ] Links to `./prd.md` (not `./01-feature-prd.md`)

### 00-research-report.md
- [ ] Contains "Executive Summary" section
- [ ] Contains "Project Structure" section
- [ ] Contains "Coding Standards" section with naming conventions
- [ ] Contains "Technology Stack" section (backend and frontend)
- [ ] Contains "Related Existing Features" section
- [ ] Contains "Integration Points" section
- [ ] Contains "Recommendations" section (Do's and Don'ts)

### 01-api-contracts.md
- [ ] Contains "Contract Purpose" section
- [ ] Contains "Data Transfer Objects (DTOs)" section with field specifications
- [ ] Contains "Endpoint Contracts" table
- [ ] Contains "Field Naming Standards" section
- [ ] DTO specifications include: field name, type, required/optional, validation rules
- [ ] No code implementation (only specifications)

### 02-architecture.md
- [ ] Contains "System Architecture Overview" with Mermaid diagram
- [ ] Contains "Technology Stack Selection" section
- [ ] Contains "Integration Points" section
- [ ] Contains "Deployment Architecture" section
- [ ] Contains "Technical Risks & Mitigations" section
- [ ] Document is 150-200 lines (±20%)

### 03-database-design.md
- [ ] Contains "Entity-Relationship Diagram" (Mermaid)
- [ ] Contains "Table Specifications" section (describes, no SQL code)
- [ ] Contains "Migration Strategy" section
- [ ] Contains "Query Optimization Patterns" section
- [ ] No SQL DDL statements present (should be descriptions only)
- [ ] Document is 100-150 lines (±20%)

### 04-api-specification.md
- [ ] Contains "Architectural Principles" section (SOLID, DRY, YAGNI)
- [ ] Contains "API Endpoints" section with endpoint details
- [ ] Contains "Request/Response Contracts" section (described, not code)
- [ ] Contains "Error Handling Strategy" section
- [ ] Contains "API Testing Approach" section
- [ ] No TypeScript interfaces or Python classes present
- [ ] Document is 150-200 lines (±20%)

### 05-frontend-design.md
- [ ] Contains "Component Hierarchy" (Mermaid diagram)
- [ ] Contains "Component Specifications" section
- [ ] Contains "State Management Architecture" section
- [ ] Contains "UX Patterns" section
- [ ] No Vue/React component code present
- [ ] Document is 150-200 lines (±20%)

### 06-security-performance.md
- [ ] Contains "Authentication & Authorization" section
- [ ] Contains "Security Measures" section (OWASP Top 10)
- [ ] Contains "Rate Limiting" section
- [ ] Contains "Caching Strategy" section
- [ ] Contains "Performance Optimization" section
- [ ] Contains "Monitoring & Observability" section
- [ ] Document is 100-150 lines (±20%)

### 07-implementation-phases.md
- [ ] Contains "Phase Breakdown" section with phase details
- [ ] Contains "Overall Timeline" section
- [ ] Contains "Resource Requirements" section
- [ ] Contains "Risk Management" section
- [ ] Document is 100-150 lines (±20%)

### 08-test-criteria.md
- [ ] Contains "Test Strategy Overview" section with test pyramid
- [ ] Contains "Acceptance Criteria as Tests" section (Given/When/Then format)
- [ ] Contains "Unit Test Specifications" section
- [ ] Contains "Integration Test Specifications" section
- [ ] Contains "End-to-End Test Specifications" section
- [ ] Contains "Contract Tests" section
- [ ] Contains "Performance Test Criteria" section
- [ ] Contains "Security Test Criteria" section
- [ ] Contains "Quality Gates" section
- [ ] No test code present (only specifications)
- [ ] Document is 150-200 lines (±20%)

### prps/README.md
- [ ] Indicates PRPs have NOT been generated yet
- [ ] References the prp-generator agent
- [ ] Lists available design documents
- [ ] Contains instructions for generating PRPs

## Anti-Pattern Detection

### Code Implementation (FAIL)
Files should contain descriptions and specifications, NOT code:
- ❌ SQL DDL statements
- ❌ Python classes or functions
- ❌ TypeScript interfaces
- ❌ JavaScript/Vue/React components
- ❌ YAML configuration
- ❌ Any executable code

### Placeholders (FAIL)
No placeholder content allowed:
- ❌ "TODO"
- ❌ "TBD"
- ❌ "[to be completed]"
- ❌ "[insert here]"
- ❌ Empty sections

### Missing Diagrams (FAIL)
Required Mermaid diagrams:
- ❌ Missing System Architecture diagram in 02-architecture.md
- ❌ Missing Entity-Relationship diagram in 03-database-design.md
- ❌ Missing Component Hierarchy diagram in 05-frontend-design.md

### Premature PRPs (FAIL)
PRPs should not exist yet:
- ❌ Any PRP files in prps/ folder except README.md

## Cross-Reference Requirements

- [ ] README.md links to all design documents
- [ ] Design docs use relative paths (`../` for parent, `./` for same folder)
- [ ] No broken internal links
- [ ] Consistent terminology across all files

## Length Guidelines

Files should be detailed but concise (±20% tolerance):

| Document | Target Lines | Min | Max |
|----------|--------------|-----|-----|
| 02-architecture.md | 150-200 | 120 | 240 |
| 03-database-design.md | 100-150 | 80 | 180 |
| 04-api-specification.md | 150-200 | 120 | 240 |
| 05-frontend-design.md | 150-200 | 120 | 240 |
| 06-security-performance.md | 100-150 | 80 | 180 |
| 07-implementation-phases.md | 100-150 | 80 | 180 |
| 08-test-criteria.md | 150-200 | 120 | 240 |

Files outside these ranges may indicate:
- **Too short**: Insufficient detail
- **Too long**: Too much detail or contains implementation code

## Pass/Fail Thresholds

### PASS ✅
- All files exist
- All required sections present
- No code implementation
- No placeholders
- All required diagrams present
- No premature PRPs
- Cross-references valid
- Lengths within range

### PASS WITH WARNINGS ⚠️
- Minor length violations (slightly outside ±20%)
- Optional sections missing
- Minor style inconsistencies
- Non-critical issues that don't block PRP generation

### FAIL ❌
- Missing required files
- Missing required sections
- Code implementation found
- Placeholders present (TODO, TBD)
- Missing required diagrams
- PRPs already created
- Broken cross-references
- Severe length violations (> ±30%)
