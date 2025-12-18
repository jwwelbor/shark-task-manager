# E06-F03: Task File Recognition & Import

**Status**: POC Design Complete
**Last Updated**: 2025-12-17

---

## Overview

This feature enables configurable task file pattern matching through regex patterns defined in `.sharkconfig.json`, allowing shark to recognize multiple task file naming conventions beyond the rigid `T-E##-F##-###.md` format.

**Key Capabilities**:
- Configurable pattern matching via `.sharkconfig.json`
- Multi-source metadata extraction (frontmatter → filename → H1)
- Automatic task key generation for PRP files
- Path-based epic/feature inference

---

## Documentation

### Architecture Documents (POC Level)

This POC project has focused, minimal architecture documentation:

1. **[02-architecture.md](./02-architecture.md)** - System architecture and integration
   - Configuration schema
   - Component integration points
   - Data flow diagrams
   - Error handling strategy
   - Implementation phases

2. **[04-backend-design.md](./04-backend-design.md)** - Backend implementation specifications
   - Config package specifications
   - PatternRegistry enhancements
   - Metadata extraction algorithms
   - Error messages and warnings
   - Testing specifications

### Product Requirements

- **[prd.md](./prd.md)** - Complete Product Requirements Document
  - User personas and stories
  - Functional requirements
  - Acceptance criteria
  - Out of scope items

---

## POC Design Philosophy

This architecture follows POC principles:

**Minimal Scope**:
- Extend existing `patterns.go` and `sync/engine.go`
- No new architectural layers
- Focus on pattern matching and metadata extraction

**Practical Implementation**:
- Use Go's standard `regexp` package
- No external dependencies
- Leverage existing `KeyGenerator` and repositories

**Quick Validation**:
- Test configurable patterns hypothesis
- Measure import success rate improvement
- Gather user feedback on pattern configuration

**Low Risk**:
- Changes isolated to pattern matching logic
- Preserve existing transaction boundaries
- Backward compatible (no breaking changes)

---

## What's NOT Included

Since this is POC-level architecture, we intentionally excluded:

1. **Separate Database Design Doc** - Uses existing schema, no new tables
2. **Separate Frontend Design Doc** - CLI-only feature, no UI components
3. **Separate Security Design Doc** - Security considerations in architecture doc
4. **Performance Design Doc** - Performance specs in backend design doc
5. **API Contracts** - Extends existing sync engine, no new APIs
6. **Deployment Architecture** - CLI tool, no deployment changes

These would be created for production-level features but are overkill for POC.

---

## Implementation Phases

The architecture defines 5 implementation phases:

1. **Config & Pattern Validation** (Must Have)
   - Load patterns from `.sharkconfig.json`
   - Validate regex and capture groups

2. **Pattern Matching & Inference** (Must Have)
   - Match files against configured patterns
   - Infer epic/feature from path

3. **Metadata Extraction** (Must Have)
   - Extract title/description with fallbacks
   - Handle missing frontmatter gracefully

4. **Task Key Generation** (Must Have)
   - Generate keys for PRP files
   - Write keys to frontmatter

5. **Testing & Documentation** (Should Have)
   - Comprehensive test coverage
   - User documentation and examples

---

## Success Criteria

**POC Success Metrics**:
- Import success rate: 90% (up from 40%)
- Pattern support: 3+ common conventions
- Error clarity: 100% actionable messages
- Performance: <30s for 1000 files
- Compatibility: Zero breaking changes

---

## Next Steps

1. Review architecture documents
2. Validate design against requirements
3. Begin Phase 1 implementation (Config & Pattern Validation)
4. Create implementation tasks for each phase

---

## Related Documents

- [Epic: E06 Intelligent Scanning](../../epic.md)
- [System Design](../../../architecture/SYSTEM_DESIGN.md)
- [Existing Sync Engine](../../../../internal/sync/engine.go)
- [Existing Pattern Registry](../../../../internal/sync/patterns.go)

---

**Document Maintainer**: Architect Agent
**Review Frequency**: After each implementation phase
