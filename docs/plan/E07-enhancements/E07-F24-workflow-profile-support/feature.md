---
feature_key: E07-F24-workflow-profile-support
epic_key: E07
title: Workflow Profile Support
description: Add workflow profile management to shark init command with basic and advanced profiles
status: specification_complete
created: 2026-01-25
updated: 2026-01-25
---

# Feature: Workflow Profile Support

**Epic**: E07 - Enhancements
**Feature Key**: E07-F24
**Status**: Specification Complete

## Overview

Add `shark init update` command with workflow profile support, enabling users to switch between "basic" and "advanced" workflow configurations without modifying the database.

## Documentation

This feature has comprehensive documentation across multiple files:

### 📋 Business Requirements
**Location**: `specifications/business-requirements.md` (416 lines)

Complete business specification including:
- 4 user stories with acceptance criteria
- Workflow profile definitions (basic: 5 statuses, advanced: 19 statuses)
- Command syntax and examples
- Error handling requirements
- Success metrics

[→ Read Business Requirements](specifications/business-requirements.md)

### 🏗️ Technical Design
**Location**: `technical-design.md` (1,587 lines)

Comprehensive technical implementation plan including:
- Architecture overview with component diagrams
- 4 component designs (ProfileRegistry, ProfileService, ConfigMerger, CLI)
- 6 data structures with code examples
- 6-phase implementation plan (13-19 hours)
- Testing strategy (21+ unit tests, 8+ integration tests)
- Risk assessment and mitigation strategies

[→ Read Technical Design](technical-design.md)

### 📖 Quick Reference
**Location**: `specifications/README.md`

Summary document with:
- Quick command reference
- Implementation status tracker
- Links to all documentation

[→ Read Quick Reference](specifications/README.md)

---

## Quick Reference

### Commands

```bash
# Apply basic profile (5 statuses: todo → in_progress → ready_for_review → completed, blocked)
shark init update --workflow=basic

# Apply advanced profile (19 statuses with full SDLC pipeline)
shark init update --workflow=advanced

# Merge defaults only (preserve customizations)
shark init update

# Preview changes without applying
shark init update --workflow=advanced --dry-run

# Force overwrite all settings
shark init update --workflow=basic --force
```

### Workflow Profiles

**Basic Profile** (5 statuses):
- Simple linear workflow: todo → in_progress → ready_for_review → completed
- Plus: blocked (escape hatch)
- Suitable for: Solo developers, simple projects

**Advanced Profile** (19 statuses):
- Multi-stage pipeline: draft → refinement (BA/Tech) → development → code review → QA → approval → completed
- Includes: Agent types, progress weights, status flows, special groups
- Suitable for: Multi-agent teams, formal SDLC processes

---

## User Stories Summary

1. **US-1**: Apply basic workflow profile (solo developers)
2. **US-2**: Apply advanced workflow profile (team leads)
3. **US-3**: Merge with existing configuration (power users)
4. **US-4**: Handle partial configuration (repair scenarios)

**Full details**: See [specifications/business-requirements.md](specifications/business-requirements.md)

---

## Implementation Plan

### Phase 1: Data Structures & Profile Registry (2-3 hours)
- Create `internal/config/profiles.go`
- Define `WorkflowProfile` and `StatusMetadata` types
- Implement profile registry with basic and advanced profiles

### Phase 2: Config Merger (3-4 hours)
- Create `internal/config/config_merger.go`
- Implement smart merge algorithm
- Handle preserve/overwrite rules

### Phase 3: Profile Service (3-4 hours)
- Create `internal/config/profile_service.go`
- Orchestrate profile application
- Implement atomic config updates

### Phase 4: CLI Command (2-3 hours)
- Extend `internal/cli/commands/init.go`
- Add `update` subcommand with flags
- Implement user feedback and reporting

### Phase 5: Documentation (1-2 hours)
- Update CLI reference docs
- Add usage examples
- Update CLAUDE.md with new command

### Phase 6: Integration Testing (2-3 hours)
- Write end-to-end tests
- Test all profile transitions
- Verify merge behavior

**Total Estimate**: 13-19 hours

**Detailed breakdown**: See [technical-design.md](technical-design.md#implementation-plan)

---

## Success Criteria

- [x] Business requirements documented
- [x] Technical design completed
- [ ] Phase 1 implementation complete
- [ ] Phase 2 implementation complete
- [ ] Phase 3 implementation complete
- [ ] Phase 4 implementation complete
- [ ] Phase 5 documentation complete
- [ ] Phase 6 testing complete
- [ ] Feature deployed and tested

---

## Key Decisions

### Configuration Merge Strategy
- **Default behavior**: Add missing fields only (non-destructive)
- **With --workflow**: Overwrite workflow-specific fields (status_metadata, status_flow, special_statuses)
- **With --force**: Overwrite all except database, project_root, viewer
- **Always preserve**: Database connection, project settings, user preferences

### Profile Storage
- Profiles stored as code constants (not external files)
- Two profiles initially: basic, advanced
- Extensible design for future profiles

### File Operations
- Atomic writes using temp file + rename pattern (existing pattern)
- Automatic backup before --force operations
- Validation before write (fail-safe)

---

## Testing Strategy

### Unit Tests (21+ tests)
- Profile registry: 6 tests
- Config merger: 9 tests
- Profile service: 6 tests

### Integration Tests (8+ scenarios)
- Fresh install → basic profile
- Fresh install → advanced profile
- Basic → advanced transition
- Custom config preservation
- Error handling (missing file, invalid JSON)
- Dry-run mode
- Force mode
- Rollback on validation failure

**Detailed test plan**: See [technical-design.md](technical-design.md#testing-strategy)

---

## Related Documents

- **Epic**: [E07 - Enhancements](../../epic.md)
- **Business Requirements**: [specifications/business-requirements.md](specifications/business-requirements.md)
- **Technical Design**: [technical-design.md](technical-design.md)
- **Quick Reference**: [specifications/README.md](specifications/README.md)

---

## Notes

### Design Principles
- **Appropriate**: Leverages existing patterns, no database changes, natural fit
- **Proven**: Uses established config merge and atomic write patterns
- **Simple**: Pure config manipulation, no new dependencies, clear separation

### Performance Targets
- Command execution: <100ms
- File operations: <50ms
- Memory usage: ~200KB peak

### Risk Mitigation
- Automatic backups before destructive operations
- Validation before write (rollback on failure)
- Dry-run mode for preview
- Comprehensive error messages

---

*Last Updated*: 2026-01-25
*Documentation Version*: 1.0
*Total Documentation*: 2,003+ lines across 4 files
