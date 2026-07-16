---
name: research
description: Systematic codebase analysis and project understanding workflows. Use this skill to analyze project structure, discover patterns, trace dependencies, and understand existing implementations before development work.
domain: codebase-analysis
inputs:
  - research_objective: description of what needs to be researched (feature|architecture|pattern|dependency|scalability)
  - search_scope: paths or directories to search (optional, defaults to project root)
  - existing_analysis_path: path to previous analysis (optional, for building on prior work)
  - related_docs_paths: list of related documentation paths (optional)
  - constraints: specific constraints or limitations to consider (optional)
outputs:
  - selected_workflow: one of {understand-feature, brownfield-analysis, map-filesystem, find-patterns, trace-dependencies, bootstrap, greenfield-scaffold, tracing-knowledge-lineages, consult-related-work, analyze-codebase}
  - findings_report: structured markdown documenting discoveries
  - pattern_catalog: identified patterns with locations and usage frequency
  - dependency_graph: mapped dependencies and interactions
  - recommended_approach: evidence-based recommendations for next steps
  - knowledge_gaps: identified gaps in understanding
---

# Research Skill

You are a meticulous technical researcher using the research skill domain. This skill provides systematic approaches to understanding codebases, discovering patterns, and documenting project structure.

## When to Use This Skill

Use research workflows:
- **Before feature implementation** - Understand existing patterns and integration points
- **During architecture** - Analyze codebase structure and conventions
- **For refactoring** - Map dependencies and understand data flows
- **For onboarding** - Build mental model of project organization
- **For documentation** - Create reference materials for team/AI agents

## Available Workflows

### Project Bootstrap Workflows

1. **bootstrap** - `workflows/bootstrap.md`
   - Main orchestrator for `/shark-rider project bootstrap` command
   - Detects brownfield vs greenfield, routes to correct track, produces `docs/architecture/` foundation
   - Readiness 2 (idea needs refinement): creates only a `tech-stack.md` placeholder; remaining docs generated on reconcile pass after product-design
   - Use for: Project bootstrapping before `/vision`, after `git clone`, at project start
   - Output: up to 7 files in `docs/architecture/` depending on track and idea readiness (see output contract in workflow)

2. **brownfield-analysis** - `workflows/brownfield-analysis.md`
   - Reverse-engineer existing codebase into architecture documents
   - Use for: Brownfield track of bootstrap (discovering stack, patterns, integrations, architecture)
   - Output: `tech-stack.md`, `patterns-catalog.md`, `integration-map.md`, `architecture-overview.md`

3. **greenfield-scaffold** - `workflows/greenfield-scaffold.md`
   - Interactive stack selection and prescriptive foundation doc generation
   - Use for: Greenfield track of bootstrap (new projects without existing code)
   - Output: `tech-stack.md`, `architecture-overview.md`, `file-system.md`, `patterns-catalog.md`, `integration-map.md`

### Core Analysis Workflows

4. **analyze-codebase** - `workflows/analyze-codebase.md`
   - Comprehensive codebase analysis and pattern discovery
   - Use for: Feature planning, architecture decisions, refactoring prep
   - Output: Structured research report with findings and recommendations

5. **map-filesystem** - `workflows/map-filesystem.md`
   - Project structure mapping and directory analysis
   - Use for: Creating file-system.md documentation, understanding organization
   - Output: Complete filesystem reference document

6. **find-patterns** - `workflows/find-patterns.md`
   - Pattern discovery and naming convention analysis
   - Use for: Understanding how similar features are implemented
   - Output: Pattern catalog with examples

7. **trace-dependencies** - `workflows/trace-dependencies.md`
   - Dependency analysis and call graph mapping
   - Use for: Understanding module relationships, impact analysis
   - Output: Dependency map with integration points

8. **tracing-knowledge-lineages** - `workflows/tracing-knowledge-lineages.md`
   - Understanding why something was built.
   - Understand how ideas evolved over time to find old solutions for new problems and avoid repeating past failures
   - Use when questioning "why do we use X", before abandoning approaches, or evaluating "new" ideas that might be revivals

9. **understand-feature** - `workflows/understand-feature.md`
   - Feature-specific deep dive analysis
   - Use for: Extending existing features, understanding specific functionality
   - Output: Feature documentation with extension recommendations

10. **consult-related-work** - `workflows/consult-related-work.md`
    - Mechanically discover prior art (sibling features, related epics, ADRs) before starting any new feature, epic refinement, or task spec
    - Use for: MANDATORY first step at research/refinement routing points and Step 1 of epic-tech-plan / feature-tech-plan
    - Output: a Capability Map section in the entity's unified `research-report.md`, with REUSE / EXTEND / NEW decisions and link relationships for reused siblings
    - Prevents: Re-implementing capabilities a sibling feature already established (the F35→F30 duplication failure mode)

## Workflow Selection Guide

**For project bootstrapping** (before any development):
1. Run `bootstrap` — detects brownfield/greenfield, generates `docs/architecture/` foundation files
2. This replaces manually running `map-filesystem`, `find-patterns`, and `analyze-codebase` separately

**For new feature development**:
1. **First, always**: Run `consult-related-work` to enumerate sibling features in the same epic and read their architecture/ADRs — prevents re-implementing what a sibling already established
2. Use `analyze-codebase` to understand project conventions
3. Use `find-patterns` to locate similar existing features
4. Use `understand-feature` to deeply analyze related functionality flagged by Step 1
5. Use `trace-dependencies` to map integration points

**For project onboarding**:
1. Start with `bootstrap` if `docs/architecture/` doesn't exist
2. Use `map-filesystem` for structure, `analyze-codebase` for overview
3. Use `find-patterns` to learn project conventions

**For refactoring**:
1. Use `understand-feature` to map current implementation
2. Use `trace-dependencies` to identify impact scope
3. Use `find-patterns` to ensure consistency

**For documentation creation**:
1. Use `bootstrap` for comprehensive foundation (up to 7 docs depending on track)
2. Or individually: `map-filesystem`, `analyze-codebase`, `find-patterns`

## Context Files

The research skill provides analysis techniques, standards, and reference data:

- **analysis-patterns.md** - Common analysis approaches and methodologies
- **search-strategies.md** - Effective search techniques using Grep, Glob, Read
- **documentation-standards.md** - How to document research findings
- **brownfield-detection.md** - Detection algorithm for brownfield vs greenfield projects (priority-ordered checks, confidence scoring, edge cases)
- **stack-research-guide.md** - Authoritative sources per tech stack, domain→stack mapping, scale modifiers, team experience weighting, coding standards augmentation pattern

## Integration with Other Skills

**This skill supports**:
- `specification-writing` - Research informs PRD and epic creation
- `architecture` - Analysis guides architecture decisions
- `implementation` - Pattern discovery ensures consistency
- `quality` - Understanding codebase enables better reviews

**Used by agents**:
- researcher (primary user)
- architect (for context before design)
- developer (understanding existing APIs and components)

## Tools Required

Research workflows use:
- **Read** - Reading files and documentation
- **Grep** - Content search across codebase
- **Glob** - File pattern matching
- **Bash** - Directory listing and structure analysis
- **WebSearch** - Technology documentation lookup (optional)

## Output Standards

All research workflows should:
1. **Be systematic** - Follow structured exploration steps
2. **Be evidence-based** - Cite file paths and code examples
3. **Be actionable** - Provide clear recommendations
4. **Be documented** - Use templates from context/documentation-standards.md
5. **Be reproducible** - Others can verify findings

## Success Criteria

Research is complete when:
- Project structure is mapped and understood
- Coding conventions are documented with examples
- Similar features are identified and analyzed
- Integration points are clearly identified
- Recommendations are actionable and specific
- Findings are well-documented for future reference

## Workflow Invocation Pattern

```markdown
## Research Phase

1. Invoke research skill workflow:
   - Skill: `research/workflows/analyze-codebase.md`
   - Context: {what you're building}

2. Review research findings

3. Document key patterns and integration points

4. Proceed with design/implementation using research insights
```

## Next Steps

Select the appropriate workflow based on your research goal:
- Comprehensive analysis: Use `workflows/analyze-codebase.md`
- Structure mapping: Use `workflows/map-filesystem.md`
- Pattern discovery: Use `workflows/find-patterns.md`
- Dependency mapping: Use `workflows/trace-dependencies.md`
- Feature understanding: Use `workflows/understand-feature.md`

Refer to context files for analysis techniques and documentation standards.
