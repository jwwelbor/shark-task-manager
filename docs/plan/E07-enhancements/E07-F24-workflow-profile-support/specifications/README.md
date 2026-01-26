# Workflow Profile Support - Documentation

**Feature**: E07-F24
**Status**: Specification Complete
**Location**: `docs/plan/E07-enhancements/E07-F24-workflow-profile-support/`

## Overview

This feature adds workflow profile management to the `shark init` command, enabling users to switch between "basic" and "advanced" workflow configurations without modifying the database.

## Documentation Files

### 1. Business Requirements
**File**: `specifications/business-requirements.md` (416 lines)

Contains:
- 4 user stories with acceptance criteria
- Workflow profile specifications (basic: 5 statuses, advanced: 19 statuses)
- Command syntax and examples
- Error handling requirements
- Success metrics

### 2. Technical Design
**File**: `technical-design.md` (1,587 lines)

Contains:
- Architecture overview with component diagrams
- 4 component designs (ProfileRegistry, ProfileService, ConfigMerger, CLI Command)
- 6 data structures with code examples
- File-by-file implementation breakdown (6 new files, 2 modified)
- 6-phase implementation plan (13-19 hours total)
- Testing strategy (21+ unit tests, 8+ integration tests)
- Risk assessment with mitigation strategies
- Performance considerations and benchmarks

### 3. Placeholder Feature File
**File**: `feature.md` (placeholder)

Needs to be updated with:
- Requirements summary
- Design overview (link to technical-design.md)
- Test plan summary

## Quick Reference

### Command Syntax
```bash
# Apply basic profile (5 statuses)
shark init update --workflow=basic

# Apply advanced profile (19 statuses with full SDLC)
shark init update --workflow=advanced

# Merge defaults (preserve customizations)
shark init update

# Preview changes without applying
shark init update --workflow=advanced --dry-run

# Force overwrite all settings
shark init update --workflow=basic --force
```

### Workflow Profiles

**Basic Profile** (5 statuses):
- todo, in_progress, ready_for_review, completed, blocked
- Simple linear flow
- Suitable for solo developers

**Advanced Profile** (19 statuses):
- Multi-stage workflow: draft → refinement (BA/Tech) → development → code review → QA → approval → completed
- Agent type assignments (ba, tech_lead, developer, qa, product_owner)
- Progress weights (0.0 to 1.0)
- Status flow enforcement
- Special status groups (_start_, _complete_, _blocked_)
- Suitable for multi-agent teams with formal process

## Implementation Status

- [x] Business requirements documented
- [x] Technical design completed
- [ ] Phase 1: Data structures & profile registry
- [ ] Phase 2: Config merger
- [ ] Phase 3: Profile service
- [ ] Phase 4: CLI command
- [ ] Phase 5: Documentation
- [ ] Phase 6: Integration testing

## Next Steps

1. Review business requirements and technical design
2. Create implementation tasks based on 6-phase plan
3. Assign to developer agent
4. Begin Phase 1 implementation

## Notes

### Recovery History
- **2026-01-25**: Original specifications created by business-analyst and architect agents
- **2026-01-25**: Accidentally deleted during cleanup (bug investigation)
- **2026-01-25**: Recreated in correct location (`E07-enhancements` directory)

The specifications were recreated with the same level of detail and completeness as the originals.
