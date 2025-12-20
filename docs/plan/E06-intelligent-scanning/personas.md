# User Personas

**Epic**: [Intelligent Documentation Scanning](./epic.md)

---

## Overview

This document defines the primary user personas for the intelligent scanning epic.

The intelligent scanning system serves three distinct personas with different needs: Technical Leads migrating existing projects, AI Agents performing incremental syncs during development, and Product Managers working across multiple documentation styles. Each persona requires different levels of flexibility, performance, and control.

---

## Primary Personas

### Persona 1: Technical Lead (Existing Project Migration)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Technical Lead or Engineering Manager migrating existing multi-epic project to shark
- **Experience Level**: 5+ years in role, high technical proficiency, experienced with CLI tools and Git
- **Key Characteristics**:
  - Manages 3-5 concurrent epics with 10-30 features each
  - Has existing markdown documentation that evolved organically over 6-18 months
  - Documentation uses varied naming conventions (team changed standards mid-project)
  - Limited time available for migration work (wants "works out of the box")
  - Risk-averse: needs validation and rollback capabilities

**Goals Related to This Epic**:
1. Import 100-300 existing markdown files into shark database without manual restructuring
2. Validate that all important documents were discovered and correctly categorized
3. Preserve existing git history and file locations (no mass file moves)
4. Establish shark as source of truth while maintaining markdown files for context

**Pain Points This Epic Addresses**:
- Current sync assumes rigid E##-epic-slug naming; many existing epics use descriptive names (identity, notifications, tech-debt)
- Feature files have varied names (some use PRD_F##, others use descriptive names, some just prd.md)
- Task files scattered across features with inconsistent numbering schemes
- No way to validate what was imported vs. what was missed
- Fear of data loss or incorrect categorization during bulk import

**Success Looks Like**:
The Technical Lead runs `shark init --scan-existing` and receives a detailed report showing 250 of 265 files successfully imported, with clear warnings about 15 files that couldn't be categorized. They review the warnings, adjust 3 files to match whitelisted patterns, re-run the scan, and have a fully populated shark database in under 10 minutes. They can now query project state programmatically without restructuring any documentation.

---

### Persona 2: AI Agent (Incremental Development Sync)

**Reference**: Defined for this epic (extends E04 AI Agent persona)

**Profile**:
- **Role/Title**: Autonomous code generation and task execution agent (Claude Code)
- **Experience Level**: Stateless between sessions, relies on explicit project state, optimizes for token efficiency
- **Key Characteristics**:
  - Creates new task files during feature development
  - Updates existing files with progress notes and status changes
  - Works within feature folders (docs/plan/{epic}/{feature}/)
  - Cannot manually restructure or validate file organization
  - Needs fast sync to maintain database consistency between sessions

**Goals Related to This Epic**:
1. Automatically detect and sync new task files created during work session
2. Update database when task file frontmatter changes (status, description)
3. Complete sync in <2 seconds to minimize session overhead
4. Receive clear errors if created file doesn't match expected patterns

**Pain Points This Epic Addresses**:
- Current sync is slow (rescans entire docs/plan tree every time)
- Agent-generated files may have slight variations from ideal format (needs flexible parsing)
- No automatic detection of new files (must explicitly run sync command)
- Sync failures are opaque (agent doesn't understand what went wrong)

**Success Looks Like**:
The AI Agent creates 3 new task files in a feature's tasks/ folder, updates frontmatter on 2 existing files, and runs `shark sync` before ending the session. The sync completes in 1.2 seconds, reports 5 files modified, and confirms database is up-to-date. The agent knows the next agent session will have accurate project state without reading file contents.

---

### Persona 3: Product Manager (Multi-Style Documentation)

**Reference**: Defined for this epic

**Profile**:
- **Role/Title**: Product Manager or Technical shark managing projects across multiple teams
- **Experience Level**: 3-5 years in role, moderate technical proficiency, comfortable with markdown and Git
- **Key Characteristics**:
  - Works with documentation from different teams/time periods with varied conventions
  - Needs to track progress across epics that don't follow rigid naming (tech-debt, security, bugs)
  - Values flexibility over strict standardization
  - Uses shark CLI to query and report project status
  - Occasionally hand-edits markdown files directly (bypassing CLI)

**Goals Related to This Epic**:
1. Support special epic types (tech-debt, bugs, change-cards) that don't use E## numbering
2. Import features that use team-specific naming conventions
3. Keep database synchronized with manual markdown edits
4. Generate accurate reports across mixed documentation styles

**Pain Points This Epic Addresses**:
- Cannot track tech-debt or bug epics that don't use E## format
- Different teams use different feature file patterns (prd.md vs. PRD_F##-name.md)
- Manual edits to markdown files become out of sync with database
- Reporting tools fail when some epics/features don't match expected format

**Success Looks Like**:
The Product Manager configures shark to whitelist "tech-debt" and "bugs" as valid epic types, imports documentation from three different team conventions, and runs `shark sync --incremental` daily to catch manual edits. They can now run `shark epic list` and see all epics (E01-identity, E02-notifications, tech-debt, bugs) with accurate progress calculations. Weekly status reports combine data from standardized and special epic types seamlessly.

---

## Secondary Personas

- **New Project Lead**: Uses shark on greenfield project with standardized naming from day one. Benefits from validation warnings when files don't match conventions, guiding them toward best practices.
- **External Contributor**: Occasional contributor who adds task files without understanding full shark conventions. Benefits from flexible parsing that accepts reasonable variations while still providing feedback.

---

## Persona Validation Notes

These personas are based on:
- Direct experience with E04 development (current sync limitations discovered during dogfooding)
- Analysis of existing shark usage patterns (internal/sync/engine.go rigid parsing)
- Common project documentation structures observed in open-source projects
- Requirements gathered from the initial epic request

**Confidence Level**: High for Technical Lead and AI Agent personas (validated through E04 usage). Medium for Product Manager persona (inferred from common multi-team project patterns).

**Assumptions Requiring Validation**:
- Are special epic types (tech-debt, bugs, change-cards) common enough to warrant first-class support vs. configuration-only?
- What percentage of existing projects could be imported with 90% success rate using proposed pattern matching?
- Is 5-second sync for 100 files fast enough, or do users expect sub-second performance?

---

*See also*: [User Journeys](./user-journeys.md)
