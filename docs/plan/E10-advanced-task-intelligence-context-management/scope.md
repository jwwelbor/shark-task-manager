# Scope Boundaries

**Epic**: [Advanced Task Intelligence & Context Management](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in this epic to prevent scope creep and set clear expectations.

---

## Out of Scope

### Explicitly Excluded Features

**1. Estimates vs. Actuals Tracking**
- **Why It's Out of Scope**: Enhancement #9 in source document is lower priority; requires estimation feature that doesn't exist yet
- **Future Consideration**: Potential E11 epic for "Task Estimation & Analytics"
- **Workaround**: Work sessions (Phase 3) provide actual duration data, but no comparison to estimates

**2. Review/Approval Workflow with Multiple Reviewers**
- **Why It's Out of Scope**: Enhancement #10 suggests reviewer assignment and review_status fields, but single-reviewer workflow (current) is sufficient
- **Future Consideration**: Multi-reviewer support could be added if teams grow beyond single tech lead
- **Workaround**: Current approve/reopen commands support single reviewer adequately

**3. Task Tags/Labels**
- **Why It's Out of Scope**: Enhancement #7 (task tags) is nice-to-have but relationships + note types provide similar categorization
- **Future Consideration**: Could add if note types prove insufficient for categorization needs
- **Workaround**: Use note types (decision, blocker, etc.) and relationships for categorization

**4. Visual Dependency Graph Rendering**
- **Why It's Out of Scope**: `shark task graph` command mentioned in requirements would require external graphviz/visualization library
- **Future Consideration**: Could output DOT format for external rendering tools
- **Workaround**: `shark task deps` shows textual dependency tree sufficient for most use cases

**5. Real-Time Collaboration / Concurrent Editing**
- **Why It's Out of Scope**: Shark is single-user tool with SQLite backend; multi-user collaboration requires architecture change
- **Future Consideration**: Potential epic for team/collaborative features with PostgreSQL backend
- **Workaround**: Single developer/agent per project; use git for team coordination

**6. AI-Generated Note Summaries**
- **Why It's Out of Scope**: While AI agents create notes, auto-summarization of note history is out of scope
- **Future Consideration**: Could add `shark task summary` that uses LLM to summarize notes
- **Workaround**: `shark task timeline` provides chronological view; humans/agents read manually

**7. Mobile App or Web UI**
- **Why It's Out of Scope**: Shark is CLI-focused tool; web UI is separate effort
- **Future Consideration**: Separate epic for web dashboard
- **Workaround**: Use CLI with `--json` flag for machine-readable output; could integrate with other tools

**8. Automatic Note Generation from Git Commits**
- **Why It's Out of Scope**: While valuable, git integration is complex and out of scope
- **Future Consideration**: Potential E12 epic for "Git Integration"
- **Workaround**: Agents manually record notes during implementation

**9. Note Attachments / File Uploads**
- **Why It's Out of Scope**: Notes are text-only; file attachments require blob storage
- **Future Consideration**: Could support file URLs/paths in notes
- **Workaround**: Store file paths as reference notes with full paths

**10. Bulk Operations on Notes/Criteria**
- **Why It's Out of Scope**: No requirement for batch operations like "delete all notes" or "mark all criteria complete"
- **Future Consideration**: Could add if requested by users
- **Workaround**: Individual commands for each operation

---

## Edge Cases & Scenarios Not Covered

**1. Circular Dependencies**
- **Impact**: Low - task relationships could create cycles (A depends on B depends on A)
- **Rationale**: Detection is complex; relies on user discipline to avoid cycles
- **Mitigation**: Documentation warning against circular dependencies; future enhancement could detect

**2. Very Large Note Counts (>1000 notes per task)**
- **Impact**: Low - performance degradation possible with excessive notes
- **Rationale**: Unusual scenario; most tasks have <50 notes
- **Mitigation**: Timeline pagination not implemented in Phase 1; could add if needed

**3. Multi-Project Note Search**
- **Impact**: Low - search is scoped to current project's database
- **Rationale**: Each project has separate shark-tasks.db; cross-project search requires new architecture
- **Mitigation**: Users must search each project separately

**4. Note Edit/Delete Functionality**
- **Impact**: Medium - notes are append-only; no edit/delete commands
- **Rationale**: Audit trail integrity more important than correction ability
- **Mitigation**: Add correction note if information was wrong; original note remains for audit

**5. Advanced Search Operators (AND/OR/NOT)**
- **Impact**: Medium - search is simple substring match, not boolean query
- **Rationale**: Complex query parsing is significant scope addition
- **Mitigation**: Simple search is sufficient for most use cases; can do multiple searches

**6. Relationship Strength/Weight**
- **Impact**: Low - all relationships are equal weight (no "strongly depends" vs. "weakly depends")
- **Rationale**: Added complexity for unclear benefit
- **Mitigation**: Use note attached to relationship if context needed

**7. Note Formatting/Markdown Rendering**
- **Impact**: Low - notes are plain text, no markdown rendering in CLI output
- **Rationale**: CLI output is line-based; markdown would complicate formatting
- **Mitigation**: Use markdown in note content; will render if viewed in external tool with --json

**8. Historical Context Data Versioning**
- **Impact**: Low - context_data is overwritten, not versioned
- **Rationale**: Current context is what matters; historical context in notes
- **Mitigation**: Record context changes as notes if history needed

---

## Alternative Approaches Considered But Rejected

### Alternative 1: External Knowledge Base Integration

**Description**: Instead of task_notes table, integrate with external tools like Notion, Confluence, or Obsidian

**Pros**:
- Leverage existing rich-text editors and collaboration features
- No need to build note management UI
- Users can access notes outside CLI

**Cons**:
- Requires API integration with multiple platforms
- Breaks offline-first SQLite architecture
- Adds external dependencies and API keys
- Complicates setup and reduces portability

**Decision Rationale**: Shark's value is self-contained, portable SQLite database; external integration violates this core principle

---

### Alternative 2: Graph Database for Relationships

**Description**: Use Neo4j or other graph database instead of SQLite task_relationships table

**Pros**:
- Native graph query language (Cypher)
- Optimized for relationship traversal
- Built-in graph algorithms (shortest path, etc.)

**Cons**:
- Requires separate database server (violates single-file principle)
- Adds complexity to setup and deployment
- Overkill for expected graph size (<1000 nodes/project)

**Decision Rationale**: SQLite with task_relationships table is sufficient; graph database is premature optimization

---

### Alternative 3: Full-Text Search Engine (Elasticsearch)

**Description**: Index tasks/notes in Elasticsearch for advanced search

**Pros**:
- Fast full-text search with ranking
- Advanced query syntax (fuzzy matching, proximity search)
- Scalable to millions of tasks

**Cons**:
- External service required
- Significant complexity increase
- Shark projects won't approach scale needing Elasticsearch

**Decision Rationale**: SQLite FTS5 extension (if used) or simple LIKE queries sufficient for project scale

---

### Alternative 4: Append-Only Event Log Instead of Mutable Tables

**Description**: Store all task changes (notes, status, metadata) as immutable events in append-only log

**Pros**:
- Perfect audit trail
- Time-travel queries (see task state at any point)
- Event sourcing enables replay

**Cons**:
- Query complexity increases significantly
- Materialized views or projections needed for current state
- More complex to implement and maintain

**Decision Rationale**: Current task_history + task_notes provides sufficient audit trail; full event sourcing is over-engineering

---

## Future Epic Candidates

Features that are natural follow-ons to this epic:

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| **E11: Task Estimation & Analytics** | Medium | Depends on E10 (needs work sessions and completion data) |
| **E12: Git Integration** | Medium | Independent but enhanced by E10 (notes + commits correlation) |
| **E13: Multi-User Collaboration** | Low | Requires architecture change (PostgreSQL backend) |
| **E14: Web Dashboard** | Medium | Independent but benefits from E10 (visualize relationships, notes) |
| **E15: AI-Powered Task Insights** | Low | Depends on E10 (needs note corpus for summarization/insights) |

---

## Phasing Strategy

This epic is explicitly broken into 3 phases to manage scope:

**Phase 1 (Must Have)**:
- Features 1 & 2: Task Notes + Completion Metadata
- **Rationale**: Core value proposition; enables AI agent workflows
- **Timeline**: 4-6 weeks

**Phase 2 (High Value)**:
- Features 3 & 4: Relationships + Acceptance Criteria + Search
- **Rationale**: Enhances discovery and verification; builds on Phase 1 data
- **Timeline**: 4-6 weeks post-Phase 1

**Phase 3 (Nice to Have)**:
- Feature 5: Work Sessions + Context Resume
- **Rationale**: Analytics and time tracking; valuable but not critical
- **Timeline**: 2-4 weeks post-Phase 2

**Total Epic Timeline**: 10-16 weeks across 3 releases

Each phase is independently valuable; if Phase 3 is deprioritized, Phases 1-2 still deliver complete workflows.

---

## Boundary Clarifications

### What "Completion Metadata" Includes vs. Excludes

**Included**:
- Files created (array of paths)
- Files modified (array of paths)
- Test status (e.g., "16/16 passing")
- Verification status (verified/not_verified/failed)
- Agent ID (who completed it)
- Completion summary (free text)

**Excluded**:
- Line-level diff data (use git for this)
- Screenshot/video attachments
- Performance metrics (load time, bundle size, etc.)
- Code coverage percentage (use external tools)

### What "Search" Includes vs. Excludes

**Included**:
- Full-text search across task titles, descriptions, notes content, completion_metadata
- File-based search (find tasks that modified specific files)
- Filter by epic, feature, status, agent
- Filter by note type

**Excluded**:
- Regular expression search
- Fuzzy matching / typo tolerance
- Proximity search ("decision within 5 words of singleton")
- Saved searches or search history

### What "Relationships" Includes vs. Excludes

**Included**:
- 7 relationship types (depends_on, blocks, related_to, follows, spawned_from, duplicates, references)
- Bidirectional queries (what blocks X, what does X block)
- Textual dependency tree output

**Excluded**:
- Visual graph rendering (DOT format output is stretch goal)
- Transitive dependency resolution ("what are all upstream blockers")
- Relationship strength/confidence scores
- Automatic relationship inference from code analysis

---

*See also*: [Requirements](./requirements.md), [Success Metrics](./success-metrics.md)
