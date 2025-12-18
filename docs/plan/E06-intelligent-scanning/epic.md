# Epic: Intelligent Documentation Scanning

**Epic Key**: E06-intelligent-scanning

---

## Goal

### Problem

When trying to use shark on existing project documentation, the current sync process fails to accurately discover and import epics, features, and tasks. The system assumes a rigid naming convention (E##-epic-slug) and file structure, making it incompatible with real-world documentation that may use:
- Different epic naming patterns (tech-debt, bugs, change-cards instead of EXX)
- Varied file structures (epic-index.md vs. folder-only organization)
- Multiple feature file naming conventions (prd.md, PRD_F##-name.md, feature-slug.md)
- Diverse task patterns (numbered prefixes, prp suffix, tasks in prps/ subfolder)
- Non-standard document organization (architecture docs, related files scattered across feature folders)

This rigid approach forces developers to either manually restructure all documentation to match shark's expectations or manually import items one-by-one. Neither solution scales for projects with hundreds of existing markdown files across multiple epics.

### Solution

Implement an intelligent scanning system that adapts to existing documentation patterns rather than forcing documentation to conform to rigid conventions. The system will:

1. **Multi-Strategy Epic Discovery**:
   - Parse epic-index.md (if present) as primary source of truth with explicit epic/feature links
   - Fall back to folder structure analysis when no index exists
   - Use configurable regex patterns to match epic/feature/task naming conventions
   - Merge information from both sources with precedence rules

2. **Regex-Based Pattern Matching**:
   - Configure regex patterns in .sharkconfig.json for epic/feature/task folders and files
   - Use named capture groups to extract components (epic_id, feature_id, task_id, slugs, numbers)
   - Support multiple patterns per entity type (try in order, first match wins)
   - Ship with comprehensive defaults matching standard conventions (E##-slug, tech-debt, bugs, etc.)
   - Generate new items using standardized format while accepting pattern variations during sync
   - Identify and catalog related documents at epic/feature levels (architecture docs, design specs)

3. **Incremental Sync with Conflict Resolution**:
   - Detect file modifications via timestamp tracking
   - Update only changed items to minimize sync time
   - Apply balanced validation (strict for IDs, flexible for metadata)
   - Preserve manual database edits when file content hasn't changed

4. **Configurable Root Directory**:
   - Support configurable documentation root (default: docs/plan)
   - Allow per-project .sharkconfig.json with custom patterns and paths
   - Enable multiple documentation trees in a single project

### Impact

- **Onboarding Time**: Reduce setup time for existing projects from hours (manual import) to minutes (intelligent scan)
- **Documentation Flexibility**: Support 90% of existing documentation patterns without requiring restructuring
- **Sync Performance**: Achieve <5 second incremental syncs for 100+ file changes (vs. full rescan)
- **Data Accuracy**: Maintain 100% consistency between database and filesystem with bidirectional conflict resolution
- **Developer Experience**: Enable "works out of the box" experience for projects with varied documentation styles

---

## Business Value

**Rating**: High

The intelligent scanning system is critical for shark's adoption beyond greenfield projects. Most real-world teams have existing documentation structures that evolved organically over months or years. Requiring restructuring creates a massive adoption barrier.

**Direct Impact**:
- Enables shark usage on existing projects with 100+ markdown files
- Eliminates 2-4 hours of manual migration work per project
- Supports incremental adoption (scan subset of docs, expand gradually)
- Reduces sync errors by adapting to reality rather than enforcing idealized structure

**Strategic Value**:
- Positions shark as "works with your workflow" vs. "forces new workflow"
- Enables migration path from other project management tools
- Provides foundation for import adapters (Jira, Linear, GitHub Issues)
- Demonstrates AI-friendly design (pattern recognition, flexible parsing)

**Risk Mitigation**:
- Prevents data loss during import (detailed conflict reports)
- Validates imported data with configurable strictness levels
- Supports rollback via dry-run mode and transaction safety
- Maintains git-friendly markdown as source of truth

---

## Epic Components

This epic is documented across multiple interconnected files:

- **[User Personas](./personas.md)** - Target user profiles and characteristics
- **[User Journeys](./user-journeys.md)** - High-level workflows and interaction patterns
- **[Requirements](./requirements.md)** - Functional and non-functional requirements
- **[Success Metrics](./success-metrics.md)** - KPIs and measurement framework
- **[Scope Boundaries](./scope.md)** - Out of scope items and future considerations

---

## Quick Reference

**Primary Users**:
- Technical Leads (existing project migration)
- AI Agents (incremental sync during development)
- Product Managers (multi-documentation-style projects)

**Key Features**:
- Epic-index.md parsing with precedence over folder structure
- Regex pattern matching with named capture groups for epic/feature/task discovery
- Configurable patterns in .sharkconfig.json (defaults support E##-slug, tech-debt, bugs, etc.)
- Standardized generation format with flexible sync acceptance
- Incremental sync with modification tracking
- Configurable documentation root and validation levels

**Success Criteria**:
- Successfully import 90% of existing markdown files without manual intervention
- Complete incremental sync of 100+ files in <5 seconds
- Zero data loss during import with detailed conflict reporting

**Timeline**: Foundation for E04-F07 (Initialization & Synchronization) improvements

---

*Last Updated*: 2025-12-17
