# Architecture Design: Template Variables for Related Docs and Tasks

**Feature:** E07-F29
**Version:** 1.0
**Last Updated:** 2026-02-13

## Executive Summary

This document details the architecture for extending the template variable system to support `{related_docs}`, `{related_tasks}`, `{related_features}`, and `{related_epics}` placeholders. The system leverages existing document repository infrastructure (E07-F05) and introduces new relationship tables for features and epics, following the proven pattern established by `task_relationships`. The design ensures minimal changes to existing code while providing dynamic context to AI agents through orchestrator action templates.

---

## System Overview

### Component Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                 Orchestrator Action Layer                       │
│  (Generates instructions from templates)                       │
└────────────────────────┬───────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────────────────┐
│              Template Placeholder Factory                       │
│  - TaskPlaceholdersWithRelated()                               │
│  - FeaturePlaceholdersWithRelated()                            │
│  - EpicPlaceholdersWithRelated()                               │
└────────────────────────┬───────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┬──────────────────┐
        ▼                ▼                ▼                  ▼
┌───────────────┐ ┌──────────────┐ ┌────────────────┐ ┌────────────────┐
│  Document     │ │    Feature   │ │     Epic       │ │  ContextData   │
│  Repository   │ │  Relationship│ │  Relationship  │ │    Parser      │
│               │ │  Repository  │ │  Repository    │ │                │
└───────┬───────┘ └──────┬───────┘ └────────┬───────┘ └────────┬───────┘
        │                │                  │                  │
        ▼                ▼                  ▼                  ▼
┌────────────────────────────────────────────────────────────────┐
│                       Database Layer                            │
│  - documents + junction tables (task/feature/epic_documents)   │
│  - feature_relationships (new)                                 │
│  - epic_relationships (new)                                    │
│  - tasks.context_data (JSON, existing)                         │
└────────────────────────────────────────────────────────────────┘
```

**Key Architectural Decisions:**

1. **Dual Approach for Relationships**:
   - Tasks: Use existing `context_data` JSON field (lightweight, already exists)
   - Features/Epics: Create formal relationship tables (enables analytics, follows `task_relationships` pattern)

2. **Minimal Template System Changes**: Extend existing placeholder functions instead of rewriting template engine

3. **Backward Compatibility**: All new placeholders return empty strings when no relationships exist

4. **Repository Injection**: Document and relationship repositories injected via constructor, not global state

---

## Summary

This architecture achieves template variable extension through:

1. **Minimal Code Changes**: Extend existing placeholder functions, no template engine rewrite
2. **Proven Patterns**: Follow `task_relationships` table pattern for features/epics
3. **Graceful Degradation**: Empty strings on errors, never break template population
4. **Repository Injection**: Clean dependency management via constructor injection
5. **Backward Compatible**: All existing templates continue working unchanged
6. **Extensible**: JSON metadata and relationship tables support future analytics

**Key Design Decisions:**
- ✅ Use formal relationship tables for features/epics (enables analytics)
- ✅ Keep task relationships in context_data (simple, already works)
- ✅ Return empty strings on errors (graceful degradation)
- ✅ Inject repositories explicitly (no global state)
- ✅ Follow existing patterns (consistency with E07-F22, task_relationships)
