# E06: Intelligent Documentation Scanning

**Status**: Draft
**Created**: 2025-12-17
**Epic Key**: E06-intelligent-scanning

---

## Quick Links

- **[Epic Overview](./epic.md)** - Problem statement, solution approach, and business value
- **[User Personas](./personas.md)** - Technical Lead, AI Agent, Product Manager personas
- **[User Journeys](./user-journeys.md)** - 4 detailed workflows covering import, sync, and conflict resolution
- **[Requirements](./requirements.md)** - Complete functional and non-functional requirements catalog
- **[Success Metrics](./success-metrics.md)** - KPIs and measurement framework
- **[Scope Boundaries](./scope.md)** - Out of scope features and future epic candidates

---

## Executive Summary

Intelligent scanning enables shark to adapt to existing documentation patterns rather than forcing rigid restructuring. Key capabilities:

1. **Multi-Strategy Epic Discovery**: Parse epic-index.md (precedence) or folder structure (fallback)
2. **Regex Pattern Matching**: Configurable patterns with named capture groups for epic/feature/task discovery
3. **Pattern Presets**: Ship with defaults supporting E##-slug, tech-debt, bugs, change-cards, and more
4. **Flexible Sync / Standardized Generation**: Accept pattern variations during sync, generate new items with standard format
5. **Incremental Sync**: Process only changed files via timestamp tracking
6. **Conflict Resolution**: Automatic resolution with configurable strategies

**Target Impact**:
- 90% import success rate (vs. 40% with rigid scanning)
- <5 second incremental sync for 100 files
- Zero data loss during import with detailed conflict reporting

---

## Key Requirements Snapshot

### Must Have
- Epic-index.md parsing with folder structure fallback (REQ-F-001, REQ-F-002)
- Configurable regex pattern matching with named capture groups (REQ-F-003)
- Feature file pattern matching (prd.md, PRD_F##-*.md, {feature-slug}.md) (REQ-F-005)
- Task file pattern matching (T-E##-F##-###, numbered prefixes, .prp suffix) (REQ-F-007)
- Incremental sync with modification tracking (REQ-F-009)
- Conflict detection and resolution strategies (REQ-F-004, REQ-F-010)
- Configurable documentation root (REQ-F-011)
- Validation strictness levels with pattern validation (strict, balanced, permissive) (REQ-F-012)

### Should Have
- Pattern validation on config load (REQ-F-013)
- Generation format configuration (REQ-F-014)
- Pattern preset library for common styles (REQ-F-015)
- Detailed scan reports with actionable warnings (REQ-F-016)
- Import validation command (`shark validate`) (REQ-F-017)
- Fuzzy task key generation for PRP files (REQ-F-018)

### Could Have
- Multi-root documentation support (REQ-F-020)
- Git-based change detection (REQ-F-021)

---

## Success Metrics Summary

| Metric | Target | Timeline |
|--------|--------|----------|
| Import Success Rate | 90% | First 10 imports |
| Incremental Sync (100 files) | <5 seconds | 30 days |
| Conflict Resolution Accuracy | 95% | 30 days |
| Special Epic Type Adoption | 40% of projects | 90 days |
| Parse Error Rate | <5% | 30 days |

---

## Out of Scope

Explicitly excluded from this epic:
- Bidirectional sync (database → markdown)
- Real-time file watching
- Machine learning for pattern recognition
- Import from external systems (Jira, Linear, GitHub Issues)
- Interactive conflict resolution UI
- Multi-project/workspace support

See [scope.md](./scope.md) for details and future epic candidates.

---

## Configuration Example

Example `.sharkconfig.json` with regex patterns:

```json
{
  "docs_root": "docs/plan",
  "validation_level": "balanced",

  "patterns": {
    "epic": {
      "folder": "(?P<epic_id>E\\d{2})-(?P<epic_slug>[a-z0-9-]+)|(?P<epic_id>tech-debt|bugs|change-cards)",
      "file": "epic\\.md",
      "generation": {
        "format": "E{number:02d}-{slug}",
        "file": "epic.md"
      }
    },

    "feature": {
      "folder": "(?P<epic_id>E\\d{2})-(?P<feature_id>F\\d{2})-(?P<feature_slug>[a-z0-9-]+)",
      "file": "prd\\.md|PRD_(?P<feature_id>F\\d{2})-[a-z0-9-]+\\.md",
      "generation": {
        "format": "E{epic:02d}-F{number:02d}-{slug}",
        "file": "prd.md"
      }
    },

    "task": {
      "file": "(?P<task_id>T-(?P<epic_id>E\\d{2})-(?P<feature_id>F\\d{2})-(?P<number>\\d{3}))(-(?P<slug>[a-z0-9-]+))?\\.md|(?P<number>\\d{3})-[a-z0-9-]+\\.md",
      "generation": {
        "format": "T-E{epic:02d}-F{feature:02d}-{number:03d}.md"
      }
    }
  }
}
```

**Key Features**:
- Named capture groups extract components: `epic_id`, `feature_id`, `task_id`, `slug`, `number`
- Alternation (`|`) supports multiple patterns: standard E##-slug OR special types (tech-debt, bugs)
- Generation formats ensure new items follow standard conventions
- Flexible sync accepts variations, standardized generation maintains consistency

---

## User Journeys Overview

1. **Initial Project Import**: Technical Lead imports 200+ existing files with mixed conventions
2. **Incremental Sync**: AI Agent syncs 5 new + 2 modified files after work session
3. **Adding Custom Epic Pattern**: Product Manager configures pattern to recognize "tech-debt" epic mid-project
4. **Handling Conflicts**: Technical Lead resolves epic-index.md vs. folder structure conflicts

See [user-journeys.md](./user-journeys.md) for detailed step-by-step workflows.

---

## Next Steps

1. Review and approve epic documentation
2. Generate feature PRDs for implementation phases
3. Create tasks with dependencies and sequencing
4. Begin implementation with E06-F01 (Epic Discovery & Import)

---

## Related Documentation

- **E04: Task Management CLI - Core Functionality** - Foundation epic providing database schema and CLI infrastructure
- **E04-F07: Initialization & Synchronization** - Current sync implementation this epic improves upon
- **.sharkconfig.json** - Project configuration file format (to be documented in feature PRDs)

---

*For questions or clarifications, see [requirements.md](./requirements.md) for detailed acceptance criteria.*
