---
inputs:
  - feature_prd_path: absolute path to the feature PRD or feature list (F04-feature-list.md)
  - experience_validation_path: absolute path to experience validation document (F07-experience-validation.md), optional
  - research_report_path: absolute path to the research report (00-research-report.md), optional
  - feasibility_assessment_path: absolute path where the feasibility assessment should be written
  - technical_risks_path: absolute path where technical risks document should be written
  - boundaries_path: absolute path where system boundaries document should be written
  - spec_review_path: absolute path where spec completeness review should be written (Technical_Review mode)
  - standards_compliance_path: absolute path where standards compliance verification should be written (Technical_Review mode)
  - mode: enum {technical_feasibility, technical_review} — which review variant to run
outputs:
  - feasibility_assessment_doc: structured markdown written to feasibility_assessment_path
  - technical_risks_doc: structured markdown written to technical_risks_path
  - boundaries_doc: structured markdown written to boundaries_path
  - spec_review_doc: structured markdown written to spec_review_path (mode=technical_review only)
  - standards_compliance_doc: structured markdown written to standards_compliance_path (mode=technical_review only)
  - viability: enum {viable, viable_with_concerns, not_viable}
  - complexity: enum {simple, standard, complex}
  - risks: list of {category, description, likelihood, impact, mitigation}
---

# Technical Feasibility Review Workflow (craft)

**When**: Feature is entering refinement and needs technical viability assessment.

## Process

### 1. Assess Technical Viability

- Can this be built with the current technology stack?
- What are the technical challenges and unknowns?
- Are there proven patterns for this type of solution?
- What is the complexity level (simple, standard, complex)?

### 2. Identify System Boundaries

- What is in scope for this system vs. external systems?
- Where are the integration points?
- What new services or components are needed?
- What existing services can be extended?

### 3. Flag Technical Risks

- Performance concerns (latency, throughput, resource usage)
- Scalability issues (data growth, user growth, load patterns)
- Security vulnerabilities (auth, input validation, data exposure)
- Data consistency challenges (eventual consistency, transactions)
- Integration complexity (third-party APIs, legacy systems)

### 4. Document Constraints and Recommend Approach

- Document hard constraints (budget, timeline, stack limitations)
- Recommend technical approach or alternatives with trade-offs
- Identify what needs further investigation

## Output

Write three documents (always) and two more in `technical_review` mode:

- `feasibility_assessment_path` — Technical viability assessment (viability verdict, complexity tier, rationale)
- `technical_risks_path` — Identified risks and mitigations (table of category/description/likelihood/impact/mitigation)
- `boundaries_path` — System boundaries and integration points (in-scope vs. external, integration points, new vs. extended components)

When `mode == technical_review` (spec completeness check), additionally write:

- `spec_review_path` — Specification completeness review (which spec sections are complete, missing, or ambiguous)
- `standards_compliance_path` — Standards compliance verification (alignment with project conventions, naming, patterns)
