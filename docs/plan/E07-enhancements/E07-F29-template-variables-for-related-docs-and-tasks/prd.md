# Template Variables for Related Docs, Tasks, Features, and Epics

**Feature Key**: E07-F29-template-variables-for-related-docs-and-tasks

---

## Epic

- **Epic PRD**: [E07 - Enhancements](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

AI agents and human developers receive orchestrator action instructions through the `instruction_template` system, but these templates currently cannot reference related documentation, dependent tasks, related features, or related epics. This forces template authors to either hardcode document paths (which breaks when files move) or omit contextual references entirely, leaving agents without crucial background information. When an entity (task/feature/epic) has associated architecture documents, dependent tasks, related features, or related epics, agents must manually discover these relationships instead of having them provided automatically in their instructions.

### Solution

Extend the existing template variable system (`{id}`, `{title}`, `{status}`, etc.) to include `{related_docs}`, `{related_tasks}`, `{related_features}`, and `{related_epics}` placeholders. When populated, these variables insert comma-separated file paths (for documents) or entity keys (for tasks/features/epics) directly into instruction templates. This leverages the existing document relationship system (E07-F05) and extends the context data structure to include feature and epic relationships, dynamically providing agents with relevant context without manual template updates.

### Impact

**Expected Outcomes**:
- **Reduce agent context discovery time by 60%**: Agents receive document paths automatically instead of searching
- **Increase instruction template reusability to 90%+**: Templates work across tasks without hardcoded paths
- **Enable context-aware workflows**: Agents can read specifications before implementation automatically
- **Improve task handoff quality**: Related tasks and documents explicitly referenced in instructions

**Measurable Metrics**:
- Number of templates using `{related_docs}` or `{related_tasks}` > 50% within 3 months
- Average instruction quality score (measured by agent success rate) increases by 25%
- Zero broken template references due to moved files

---

## User Personas

### Persona 1: AI Agent / Orchestrator

**Profile**:
- **Role/Title**: AI Agent spawned by orchestrator to complete tasks
- **Experience Level**: Fully automated, relies on instruction templates for task context
- **Key Characteristics**:
  - Needs explicit file paths to read documentation
  - Cannot navigate file system or discover documents independently
  - Requires structured, machine-readable instructions
  - Benefits from related task references to understand dependencies

**Goals Related to This Feature**:
1. Receive all relevant document paths in initial instruction
2. Know which tasks are dependencies or related work
3. Avoid manual context discovery through file system exploration

**Pain Points This Feature Addresses**:
- Instructions lack document references, forcing manual prompting for paths
- No visibility into related tasks without explicit handoff notes
- Template instructions are generic and lack task-specific context

**Success Looks Like**:
Receives instruction like: "Implement task E07-F29-001. Review related documentation: `docs/spec/template-system.md,docs/arch/placeholder-factory.md`. Related tasks: `E07-F05-001,E10-F05-002`." Agent reads docs automatically before starting work.

---

### Persona 2: Workflow Template Author / Technical Lead

**Profile**:
- **Role/Title**: Technical lead configuring orchestrator workflows in `.sharkconfig.json`
- **Experience Level**: Advanced understanding of shark configuration and workflow design
- **Key Characteristics**:
  - Writes `instruction_template` strings for status transitions
  - Wants reusable templates across features and epics
  - Needs to provide agents with contextual information dynamically

**Goals Related to This Feature**:
1. Write generic templates that work for any task
2. Automatically inject task-specific document references
3. Reduce template maintenance burden

**Pain Points This Feature Addresses**:
- Must hardcode document paths, breaking when files move
- Cannot reference task-specific relationships dynamically
- Templates become stale when documentation structure changes
- Copying templates requires updating all path references

**Success Looks Like**:
Writes template once: `"Read {related_docs} before implementing {id}"`. Template works for every task automatically, with correct document paths inserted at runtime.

---

### Persona 3: Developer / QA Engineer Linking Documents

**Profile**:
- **Role/Title**: Developer or QA engineer creating documentation and linking it to work items
- **Experience Level**: Moderate CLI proficiency, familiar with `shark related-docs` commands
- **Key Characteristics**:
  - Creates design docs, test plans, API specs
  - Links documents to tasks/features using `shark related-docs add`
  - Wants documents to be discoverable by agents and team members

**Goals Related to This Feature**:
1. Ensure linked documents are automatically provided to agents
2. See documents referenced in orchestrator instructions
3. Track which documents are being used in workflows

**Pain Points This Feature Addresses**:
- No way to verify if linked documents are being provided to agents
- Unclear whether document linking actually affects agent behavior
- Duplicate effort: linking docs AND updating templates manually

**Success Looks Like**:
Links API spec to task via `shark related-docs add "API Spec" docs/api/spec.md --task=E07-F29-001`. Next status transition automatically includes spec path in agent instructions.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI agent, I want to receive related document paths in my instruction template so that I can read specifications before implementing a task.

**Acceptance Criteria**:
- [ ] `{related_docs}` placeholder populates with comma-separated file paths
- [ ] Documents fetched from junction tables (`task_documents`, `feature_documents`, `epic_documents`)
- [ ] Empty string returned if no related documents exist
- [ ] File paths are absolute or project-relative (consistent with entity file paths)

---

**Story 2**: As an AI agent, I want to receive related task keys in my instruction template so that I can understand dependencies and coordinate with other work.

**Acceptance Criteria**:
- [ ] `{related_tasks}` placeholder populates with comma-separated task keys
- [ ] Task keys fetched from `ContextData.RelatedTasks` JSON field
- [ ] Empty string returned if task has no `context_data` or `RelatedTasks` is empty
- [ ] Graceful handling of malformed JSON (logs warning, returns empty string)

---

**Story 3**: As a workflow template author, I want to use `{related_docs}` and `{related_tasks}` in instruction templates so that agents receive dynamic context.

**Acceptance Criteria**:
- [ ] Template example: `"Work on {id}. Related docs: {related_docs}. Related tasks: {related_tasks}"`
- [ ] Placeholders replaced at template population time (when orchestrator action generated)
- [ ] Works for task-level, feature-level, and epic-level templates
- [ ] Backward compatible: existing templates without new placeholders continue working

---

**Story 4**: As a developer linking documents, I want documents to automatically appear in agent instructions when linked via `shark related-docs add`.

**Acceptance Criteria**:
- [ ] Linking document to task/feature/epic makes it available for `{related_docs}`
- [ ] Unlinking document removes it from next placeholder population
- [ ] No manual template updates required
- [ ] Document paths reflect current file locations (dynamic lookup)

---

**Story 4a**: As an AI agent working on a feature, I want to receive related feature keys in my instruction template so that I can understand cross-feature dependencies and coordinate work.

**Acceptance Criteria**:
- [ ] `{related_features}` placeholder populates with comma-separated feature keys
- [ ] Feature keys fetched from feature's `ContextData.RelatedFeatures` JSON array
- [ ] Empty string returned if feature has no `context_data` or `RelatedFeatures` is empty
- [ ] Works for both feature-level and epic-level instruction templates
- [ ] Supports cross-epic feature relationships (e.g., E01-F01 relates to E02-F05)

---

**Story 4b**: As an AI agent working on an epic, I want to receive related epic keys in my instruction template so that I can understand epic-level dependencies and context.

**Acceptance Criteria**:
- [ ] `{related_epics}` placeholder populates with comma-separated epic keys
- [ ] Epic keys fetched from epic's `ContextData.RelatedEpics` JSON array
- [ ] Empty string returned if epic has no `context_data` or `RelatedEpics` is empty
- [ ] Works for epic-level instruction templates
- [ ] Enables referencing prerequisite or parallel epics in instructions

---

### Should-Have Stories

**Story 5**: As a workflow template author, I want to customize the formatting of related docs/tasks so that instructions are readable.

**Acceptance Criteria**:
- [ ] Template can wrap placeholder: `"Docs:\n{related_docs}"` formats as `"Docs:\ndoc1.md,doc2.md"`
- [ ] Empty values don't create awkward formatting (e.g., "Docs:\n" when no docs)
- [ ] Template author can add conditional-like formatting (e.g., prefix "Related Docs: " only if non-empty)

---

**Story 6**: As a project manager, I want to see which templates are using relational placeholders so that I can verify workflow configuration.

**Acceptance Criteria**:
- [ ] `shark config get status_metadata.*.orchestrator_action.instruction_template` shows templates
- [ ] Documentation lists available placeholders (`{related_docs}`, `{related_tasks}`)
- [ ] Example templates provided in docs/guides/

---

### Could-Have Stories

**Story 7**: As a QA engineer, I want to link test plans to features and have them automatically referenced in QA phase instructions.

**Acceptance Criteria**:
- [ ] QA status templates can use `{related_docs}` for test plans
- [ ] Feature-level documents (like test plans) available in feature placeholder context
- [ ] Documentation includes example: QA workflow using related docs

---

### Edge Case & Error Stories

**Error Story 1**: As an AI agent, when a task has no related documents or tasks, I want empty placeholders so that template population doesn't fail.

**Acceptance Criteria**:
- [ ] `{related_docs}` replaced with empty string `""` when no documents linked
- [ ] `{related_tasks}` replaced with empty string `""` when no context data or empty array
- [ ] Template remains valid: `"Docs: {related_docs}"` becomes `"Docs: "` (not an error)
- [ ] No error logs for empty relational data (normal case)

---

**Error Story 2**: As a system, when task context_data JSON is malformed, I want graceful handling so that template population succeeds.

**Acceptance Criteria**:
- [ ] Malformed JSON in `task.context_data` logs warning but doesn't fail template population
- [ ] `{related_tasks}` returns empty string on parse failure
- [ ] Error logged to system logs for debugging: `"Failed to parse context_data for task E07-F29-001: invalid JSON"`
- [ ] Template population completes successfully despite parse error

---

**Error Story 3**: As a template author, when I reference `{related_docs}` but the entity has 50+ documents, I want reasonable handling of large lists.

**Acceptance Criteria**:
- [ ] All related documents included (no arbitrary truncation in MVP)
- [ ] Future consideration: Add truncation config if performance degrades
- [ ] Documentation warns about performance implications of linking excessive documents
- [ ] Recommendation: Use feature/epic level docs for broad context, task-level for specific references

---

## Requirements

### Functional Requirements

**Category: Template Variable Extension**

1. **REQ-F-001**: Related Docs Placeholder
   - **Description**: Add `{related_docs}` placeholder that populates with comma-separated file paths of all documents linked to the entity (task/feature/epic)
   - **User Story**: Links to Story 1, 3, 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Placeholder factory functions (`TaskPlaceholders`, `FeaturePlaceholders`, `EpicPlaceholders`) extended to include `related_docs` key
     - [ ] Value fetched via `DocumentRepository.ListForTask/Feature/Epic()`
     - [ ] Paths extracted from `*models.Document` list and joined with commas
     - [ ] Empty string if no documents linked

2. **REQ-F-002**: Related Tasks Placeholder
   - **Description**: Add `{related_tasks}` placeholder that populates with comma-separated task keys from context data
   - **User Story**: Links to Story 2, 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `TaskPlaceholders` extended to include `related_tasks` key
     - [ ] Value extracted from `task.context_data` JSON field's `RelatedTasks` array
     - [ ] Task keys joined with commas
     - [ ] Empty string if no context data or empty array
     - [ ] Graceful handling of JSON parse errors

2a. **REQ-F-002a**: Related Features Placeholder
   - **Description**: Add `{related_features}` placeholder that populates with comma-separated feature keys from relationship table
   - **User Story**: Links to Story 4a
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `FeaturePlaceholders` and `EpicPlaceholders` extended to include `related_features` key
     - [ ] Value queried from `feature_relationships` table (all relationship types)
     - [ ] Feature keys extracted and joined with commas (e.g., "E01-F01,E02-F05")
     - [ ] Empty string if no relationships exist
     - [ ] Includes both outbound (`from_feature_id`) and inbound (`to_feature_id`) relationships
     - [ ] Fallback to context data if table query fails (backward compatibility)

2b. **REQ-F-002b**: Related Epics Placeholder
   - **Description**: Add `{related_epics}` placeholder that populates with comma-separated epic keys from relationship table
   - **User Story**: Links to Story 4b
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `EpicPlaceholders` extended to include `related_epics` key
     - [ ] Value queried from `epic_relationships` table (all relationship types)
     - [ ] Epic keys extracted and joined with commas (e.g., "E01,E03,E07")
     - [ ] Empty string if no relationships exist
     - [ ] Includes both outbound and inbound relationships
     - [ ] Fallback to context data if table query fails (backward compatibility)

3. **REQ-F-003**: Entity-Level Placeholder Support
   - **Description**: All three entity types (task, feature, epic) support `{related_docs}` placeholder
   - **User Story**: Links to Story 3, 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `TaskPlaceholdersWithRelated(task, docRepo, ctx)` implemented
     - [ ] `FeaturePlaceholdersWithRelated(feature, docRepo, ctx)` implemented
     - [ ] `EpicPlaceholdersWithRelated(epic, docRepo, ctx)` implemented
     - [ ] Consistent behavior across all entity types

4. **REQ-F-004**: Repository Integration
   - **Description**: Orchestrator action generation in repositories uses extended placeholder functions
   - **User Story**: Links to Story 1, 2, 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `TaskRepository.GetOrchestratorActionForTask()` calls `TaskPlaceholdersWithRelated`
     - [ ] `EpicService` transition methods use `EpicPlaceholdersWithRelated`
     - [ ] `FeatureService` transition methods use `FeaturePlaceholdersWithRelated`
     - [ ] Document repository injected or passed to placeholder factory

**Category: Database Schema**

5. **REQ-F-005**: Feature Relationships Table
   - **Description**: Create `feature_relationships` table for typed feature-to-feature relationships
   - **User Story**: Links to Story 4a
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Table created in `internal/db/db.go` schema
     - [ ] Columns: `id`, `from_feature_id`, `to_feature_id`, `relationship_type`, `created_at`
     - [ ] CHECK constraint on `relationship_type` enum (7 types: depends_on, blocks, related_to, follows, spawned_from, duplicates, references)
     - [ ] FOREIGN KEY constraints on both feature_id columns with CASCADE delete
     - [ ] UNIQUE constraint on (from_feature_id, to_feature_id, relationship_type)
     - [ ] Indexes on from_feature_id, to_feature_id, relationship_type
     - [ ] Auto-migration adds table if not exists (backward compatible)

6. **REQ-F-006**: Epic Relationships Table
   - **Description**: Create `epic_relationships` table for typed epic-to-epic relationships
   - **User Story**: Links to Story 4b
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Table created in `internal/db/db.go` schema
     - [ ] Columns: `id`, `from_epic_id`, `to_epic_id`, `relationship_type`, `created_at`
     - [ ] CHECK constraint on `relationship_type` enum (same 7 types as features)
     - [ ] FOREIGN KEY constraints on both epic_id columns with CASCADE delete
     - [ ] UNIQUE constraint on (from_epic_id, to_epic_id, relationship_type)
     - [ ] Indexes on from_epic_id, to_epic_id, relationship_type
     - [ ] Auto-migration adds table if not exists (backward compatible)

**Category: Repository Layer**

7. **REQ-F-007**: Feature Relationship Repository
   - **Description**: Repository methods for managing feature relationships
   - **User Story**: Links to Story 4a
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `FeatureRelationshipRepository` created in `internal/repository/`
     - [ ] `ListRelatedFeatures(ctx, featureID) ([]string, error)` - returns feature keys
     - [ ] `AddRelationship(ctx, fromID, toID, relType)` - creates relationship
     - [ ] `RemoveRelationship(ctx, fromID, toID, relType)` - deletes relationship
     - [ ] `GetRelationshipsByType(ctx, featureID, relType)` - filtered query
     - [ ] All methods handle both directions (from and to)

8. **REQ-F-008**: Epic Relationship Repository
   - **Description**: Repository methods for managing epic relationships
   - **User Story**: Links to Story 4b
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `EpicRelationshipRepository` created in `internal/repository/`
     - [ ] `ListRelatedEpics(ctx, epicID) ([]string, error)` - returns epic keys
     - [ ] `AddRelationship(ctx, fromID, toID, relType)` - creates relationship
     - [ ] `RemoveRelationship(ctx, fromID, toID, relType)` - deletes relationship
     - [ ] `GetRelationshipsByType(ctx, epicID, relType)` - filtered query
     - [ ] All methods handle both directions (from and to)

**Category: Data Retrieval**

9. **REQ-F-009**: Document Path Formatting
   - **Description**: Helper function to extract file paths from document list and format as CSV
   - **User Story**: Links to Story 1, 4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `formatDocPathsAsCSV([]*models.Document) string` implemented
     - [ ] Returns empty string for empty/nil input
     - [ ] Extracts `doc.FilePath` from each document
     - [ ] Joins with comma separator: `"docs/a.md,docs/b.md,docs/c.md"`

10. **REQ-F-010**: Context Data Parsing
   - **Description**: Helper function to parse RelatedTasks from task context data JSON
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `extractRelatedTasksFromContext(*string) string` implemented
     - [ ] Returns empty string if `context_data` is nil or empty
     - [ ] Parses JSON using `models.FromJSON()`
     - [ ] Extracts `ContextData.RelatedTasks` array
     - [ ] Joins task keys with comma separator
     - [ ] Logs warning (not error) if JSON parse fails

**Category: Backward Compatibility**

11. **REQ-F-011**: Existing Template Compatibility
   - **Description**: Templates without new placeholders continue working unchanged
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Existing placeholders (`{id}`, `{title}`, `{status}`, etc.) still work
     - [ ] Templates with no `{related_docs}` or `{related_tasks}` unaffected
     - [ ] No breaking changes to `PopulateTemplate()` function
     - [ ] All existing tests pass without modification

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Placeholder Population Latency
   - **Description**: Adding relational placeholders must not significantly degrade orchestrator action response time
   - **Measurement**: `shark task get --json` response time with 10 related docs
   - **Target**: < 50ms overhead compared to baseline (without related docs)
   - **Justification**: Orchestrator actions are called frequently; latency impacts user experience

2. **REQ-NF-002**: Query Efficiency
   - **Description**: Document lookup must use indexed queries to avoid performance degradation
   - **Measurement**: Existing indexes on junction tables (`idx_task_documents_task_id`, etc.)
   - **Target**: Single query per entity (no N+1 pattern)
   - **Justification**: Prevents database bottleneck when listing many tasks

**Code Quality**

3. **REQ-NF-003**: Test Coverage
   - **Description**: New placeholder functionality must have comprehensive test coverage
   - **Measurement**: Code coverage report for `template_helpers.go` and integration tests
   - **Target**: ≥ 80% coverage for new code
   - **Justification**: Template system is critical infrastructure; bugs affect all orchestrator workflows

4. **REQ-NF-004**: Error Handling
   - **Description**: All failure modes must be handled gracefully without breaking template population
   - **Implementation**: Try-catch patterns, default to empty string on errors
   - **Compliance**: No panics, all errors logged for debugging
   - **Risk Mitigation**: Prevents orchestrator action failures due to malformed data

**Maintainability**

5. **REQ-NF-005**: Separation of Concerns
   - **Description**: Placeholder logic isolated in `template_helpers.go`, not scattered across repositories
   - **Implementation**: Centralized placeholder factory pattern
   - **Compliance**: Single responsibility principle
   - **Risk Mitigation**: Easier to test, modify, and extend in future

**Documentation**

6. **REQ-NF-006**: Template Variable Reference
   - **Description**: Comprehensive documentation of available placeholders and usage examples
   - **Implementation**: Add section to `docs/guides/template-system.md` (create if needed)
   - **Compliance**: Lists all placeholders with descriptions, data sources, and examples
   - **Risk Mitigation**: Template authors need clear guidance to use feature effectively

---

## Acceptance Criteria

### Feature-Level Acceptance

**Given/When/Then Format**:

**Scenario 1: Task with Related Docs**
- **Given** task E07-F29-001 has 2 related documents (`docs/spec.md`, `docs/design.md`)
- **And** status metadata for `ready_for_development` has instruction template: `"Implement {id}. Read: {related_docs}"`
- **When** task transitions to `ready_for_development`
- **Then** orchestrator action instruction is: `"Implement E07-F29-001. Read: docs/spec.md,docs/design.md"`
- **And** `{related_docs}` placeholder replaced with correct comma-separated paths

---

**Scenario 2: Task with Related Tasks in Context**
- **Given** task E07-F29-002 has `context_data` JSON: `{"related_tasks": ["E07-F05-001", "E10-F05-002"]}`
- **And** instruction template: `"Work on {id}. Dependencies: {related_tasks}"`
- **When** orchestrator action generated for task
- **Then** instruction is: `"Work on E07-F29-002. Dependencies: E07-F05-001,E10-F05-002"`
- **And** task keys correctly extracted from context data

---

**Scenario 3: Task with No Related Docs or Tasks**
- **Given** task E07-F29-003 has no related documents
- **And** task has no context data
- **And** instruction template: `"Task {id}. Docs: {related_docs}. Tasks: {related_tasks}"`
- **When** orchestrator action generated
- **Then** instruction is: `"Task E07-F29-003. Docs: . Tasks: "`
- **And** empty strings replace placeholders (no error)

---

**Scenario 4: Feature-Level Template with Related Docs**
- **Given** feature E07-F29 has 3 related documents (architecture, PRD, test plan)
- **And** feature status template uses: `"Review feature {id} documentation: {related_docs}"`
- **When** feature transitions to `ready_for_refinement_tech`
- **Then** orchestrator action includes all 3 document paths in instruction
- **And** feature-level placeholder function used (`FeaturePlaceholdersWithRelated`)

---

**Scenario 5: Malformed Context Data Handling**
- **Given** task E07-F29-004 has `context_data` = `"{invalid json"`
- **And** instruction template: `"Related: {related_tasks}"`
- **When** orchestrator action generated
- **Then** `{related_tasks}` replaced with empty string
- **And** warning logged: `"Failed to parse context_data for task E07-F29-004"`
- **And** template population succeeds (no error)

---

**Scenario 6: Feature with Related Features**
- **Given** feature E07-F29 has `context_data` JSON: `{"related_features": ["E07-F05", "E07-F21", "E10-F05"]}`
- **And** feature status template for `ready_for_research` uses: `"Research {id}. Related features: {related_features}"`
- **When** feature transitions to `ready_for_research`
- **Then** orchestrator action instruction is: `"Research E07-F29. Related features: E07-F05,E07-F21,E10-F05"`
- **And** feature keys correctly extracted from context data
- **And** cross-epic feature references work (E10-F05 from different epic)

---

**Scenario 7: Epic with Related Epics**
- **Given** epic E07 has `context_data` JSON: `{"related_epics": ["E01", "E05"]}`
- **And** epic status template uses: `"Work on epic {id}. Prerequisite epics: {related_epics}"`
- **When** epic orchestrator action generated
- **Then** instruction is: `"Work on epic E07. Prerequisite epics: E01,E05"`
- **And** epic keys correctly extracted from context data
- **And** provides context about dependent/related epics

---

## Out of Scope

### Explicitly Excluded

1. **Conditional Placeholder Logic**
   - **Why**: Current template system is simple string replacement, not a full templating engine (e.g., Jinja2, Handlebars)
   - **Future**: Could add conditional placeholders in later feature (e.g., `{{#if related_docs}}Docs: {related_docs}{{/if}}`)
   - **Workaround**: Template authors format around potentially empty values

2. **Automatic Document Discovery**
   - **Why**: Documents must be explicitly linked via `shark related-docs add`; no filesystem scanning
   - **Future**: Possible enhancement to auto-discover docs in feature/epic directories
   - **Workaround**: Use `shark related-docs add` to link relevant documents

3. **Advanced Relationship Analytics**
   - **Why**: Basic relationship storage/retrieval only; no dependency graph visualization, impact analysis, or cycle detection
   - **Future**: Build analytics features on top of relationship tables (e.g., "what features block this epic?", circular dependency detection)
   - **Workaround**: Manual SQL queries against relationship tables for basic analysis

4. **CLI Commands for Relationship Management**
   - **Why**: No user-facing CLI commands for adding/removing feature/epic relationships in MVP
   - **Future**: Add commands like `shark feature relate E07-F29 E07-F05 --type=depends_on` and `shark epic relate E07 E05 --type=follows`
   - **Workaround**: Relationships can be added via SQL or context_data JSON for MVP; templates will display them

5. **Document Content Inlining**
   - **Why**: Placeholders provide paths only, not document contents
   - **Future**: Could add `{related_docs_content}` placeholder that inlines file contents (expensive)
   - **Workaround**: Agents use paths to read documents via Read tool

6. **Placeholder Truncation/Pagination**
   - **Why**: All related docs/tasks included in MVP, regardless of count
   - **Future**: Add config option for max items if performance issues arise
   - **Workaround**: Limit number of documents linked at task level

---

### Alternative Approaches Rejected

**Alternative 1: Template Engine Upgrade (Handlebars/Jinja2)**
- **Description**: Replace simple string replacement with full template engine supporting conditionals, loops, filters
- **Why Rejected**: Massive complexity increase, breaks existing templates, overkill for current needs
- **Trade-off**: Simpler system vs more powerful features

**Alternative 2: Nested Placeholder Maps**
- **Description**: Return structured data instead of CSV: `{related_docs: [{title, path}, ...]}`
- **Why Rejected**: Requires JSON serialization in templates, harder to use in string instructions
- **Trade-off**: Simplicity vs structure

**Alternative 3: Create `task_relationships` Table**
- **Description**: Formal many-to-many table for task dependencies instead of JSON context field
- **Why Rejected**: Over-engineering for MVP; context data approach already works
- **Trade-off**: Queryability vs implementation speed (can add later if needed)

---

## Success Metrics

### Primary Metrics

1. **Template Placeholder Adoption**
   - **What**: Percentage of instruction templates using `{related_docs}` or `{related_tasks}`
   - **Target**: ≥ 50% of active templates use at least one relational placeholder
   - **Timeline**: 3 months post-release
   - **Measurement**: Scan `.sharkconfig.json` for placeholder usage in `instruction_template` fields

2. **Agent Context Discovery Time Reduction**
   - **What**: Average time agents spend discovering documents/dependencies before starting task
   - **Target**: 60% reduction (e.g., 5 minutes → 2 minutes)
   - **Timeline**: 1 month post-release
   - **Measurement**: Survey of agent execution logs, time from task assignment to first implementation action

3. **Broken Template Reference Rate**
   - **What**: Incidents of template references to moved/deleted files
   - **Target**: 0 broken references (dynamic lookup prevents)
   - **Timeline**: Ongoing
   - **Measurement**: Zero error logs related to hardcoded paths in templates

---

### Secondary Metrics

- **Document Linkage Quality**: Average number of related docs per task increases to ≥ 2 (shows feature utility)
- **Template Reusability**: Number of times generic templates reused across features (≥ 10 reuses per template)
- **Template Maintenance Effort**: Time spent updating templates when docs move (target: 0 hours)

---

## Dependencies & Integrations

### Dependencies

- **E07-F05 (Related Documents)**: **CRITICAL** - Provides `documents` table, junction tables, and `DocumentRepository`
- **E07-F21 (Orchestrator Actions)**: **CRITICAL** - Defines `instruction_template` system and `PopulateTemplate()` function
- **E10-F05 (Work Sessions/Context)**: **IMPORTANT** - Provides `ContextData.RelatedTasks` field in JSON context
- **E07-F28 (Orchestration Action on Get)**: **MINOR** - Displays populated orchestrator actions (beneficiary of feature)

### Integration Requirements

- **Template System** (`internal/config/template_helpers.go`): Extend placeholder factory functions
- **Document Repository** (`internal/repository/document_repository.go`): Use existing `ListForTask/Feature/Epic()` methods
- **Repository Layer** (`internal/repository/task_repository.go`, etc.): Inject document repository for placeholder population
- **Service Layer** (`internal/services/epic_service.go`, `feature_service.go`): Update orchestrator action generation calls

---

## Compliance & Security Considerations

**None** - Feature operates entirely on internal data structures (document paths, task keys). No external systems, user data, or compliance requirements.

**Security Note**: Document file paths are project-relative and already exposed via `shark related-docs list`. No new security surface area introduced.

---

## Open Questions & Assumptions

### Assumptions

1. **Document Path Format**: Assumes file paths in `documents.file_path` are project-relative (e.g., `docs/spec.md`, not `/home/user/project/docs/spec.md`)
   - **Validation**: Confirm with existing `shark related-docs` implementation
   - **Impact**: If absolute paths, agents may have access issues
   - **Resolution**: Research shows `DocumentRepository` stores paths as provided; recommend project-relative in docs

2. **Related Tasks Scope**: Assumes `ContextData.RelatedTasks` contains task keys only, not full task objects
   - **Validation**: Confirmed in research - array of strings (keys)
   - **Impact**: Placeholder provides keys for lookup, not full context
   - **Resolution**: Accepted - agents can fetch full task details via `shark task get` if needed

3. **Performance Impact**: Assumes fetching 10-20 related docs per task has negligible performance impact
   - **Validation**: Needs load testing with real data
   - **Impact**: If too slow, may need batch loading or caching
   - **Resolution**: Implement MVP, add batch loading only if performance issue observed

4. **Empty Value Handling**: Assumes template authors can handle empty placeholders gracefully
   - **Validation**: Document best practices in template guide
   - **Impact**: May result in awkward formatting like "Docs: " with trailing space
   - **Resolution**: Accept for MVP; future enhancement could add conditional logic

### Resolved Questions

**Q1: Should we support feature-level and epic-level relationship placeholders (`{related_features}`, `{related_epics}`)?**
- **Decision**: YES - Features and epics can have cross-entity relationships
- **Rationale**:
  - Features frequently depend on other features (e.g., Auth depends on User Management)
  - Features can be related across epics (e.g., API Gateway in E01 relates to Service Mesh in E02)
  - Epics often have prerequisite or parallel epics that provide context
  - Infrastructure already exists: both features and epics have `context_data` JSON column
- **Impact**:
  - Add `RelatedFeatures []string` to ContextData model
  - Add `RelatedEpics []string` to ContextData model
  - Extend `FeaturePlaceholders` and `EpicPlaceholders` with these fields
  - Note: `{related_tasks}` remains task-level only (tasks don't relate to features at template level)

**Q2: Should we create a formal `task_relationships` table instead of using context data?**
- **Decision**: NO for MVP - Use existing `ContextData.RelatedTasks` approach
- **Rationale**: Context data approach already exists and works; table can be added later if queryability needed
- **Impact**: Less queryable but faster to implement; sufficient for MVP

**Q3: Should we truncate lists if too many related items?**
- **Decision**: NO for MVP - Include all related docs/tasks/features/epics
- **Rationale**: Better to provide complete context; truncation can cause missing critical references
- **Impact**: Potentially long comma-separated lists; acceptable for text-based instructions
- **Future**: Add config option if performance becomes issue

**Q4: Should we create formal relationship tables (`feature_relationships`, `epic_relationships`) like tasks have?**
- **Decision**: YES - Create formal relationship tables with typed relationships
- **Rationale**:
  - Tasks already have `task_relationships` table with typed relationships (`depends_on`, `blocks`, `related_to`, `follows`, etc.)
  - Features and epics deserve the same level of relationship modeling
  - SQL-queryable relationships enable analytics, dependency graphs, and impact analysis
  - Typed relationships provide semantic meaning (dependency vs soft relationship)
  - Context data still used for backward compatibility and lightweight references
- **Design Decision**: Dual Approach (Tables + Context Data)
  - **Primary**: Formal `feature_relationships` and `epic_relationships` tables
  - **Secondary**: Context data arrays for lightweight references (backward compatible)
  - Follow proven `task_relationships` pattern exactly
- **Database Schema**:
  ```sql
  CREATE TABLE feature_relationships (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      from_feature_id INTEGER NOT NULL,
      to_feature_id INTEGER NOT NULL,
      relationship_type TEXT CHECK (relationship_type IN (
          'depends_on',    -- from_feature depends on to_feature completing
          'blocks',        -- from_feature blocks to_feature from proceeding
          'related_to',    -- Features share common code/concerns
          'follows',       -- from_feature naturally follows to_feature
          'spawned_from',  -- from_feature created from UAT/bugs in to_feature
          'duplicates',    -- Features represent duplicate work
          'references'     -- from_feature consults/uses output of to_feature
      )) NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (from_feature_id) REFERENCES features(id) ON DELETE CASCADE,
      FOREIGN KEY (to_feature_id) REFERENCES features(id) ON DELETE CASCADE,
      UNIQUE(from_feature_id, to_feature_id, relationship_type)
  );

  CREATE TABLE epic_relationships (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      from_epic_id INTEGER NOT NULL,
      to_epic_id INTEGER NOT NULL,
      relationship_type TEXT CHECK (relationship_type IN (
          'depends_on',    -- from_epic depends on to_epic
          'blocks',        -- from_epic blocks to_epic
          'related_to',    -- Epics share themes/objectives
          'follows',       -- from_epic follows to_epic (roadmap sequence)
          'spawned_from',  -- from_epic created from to_epic learnings
          'duplicates',    -- Epics represent duplicate initiatives
          'references'     -- from_epic references to_epic deliverables
      )) NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      FOREIGN KEY (from_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
      FOREIGN KEY (to_epic_id) REFERENCES epics(id) ON DELETE CASCADE,
      UNIQUE(from_epic_id, to_epic_id, relationship_type)
  );

  -- Indexes for query performance
  CREATE INDEX idx_feature_relationships_from ON feature_relationships(from_feature_id);
  CREATE INDEX idx_feature_relationships_to ON feature_relationships(to_feature_id);
  CREATE INDEX idx_feature_relationships_type ON feature_relationships(relationship_type);
  CREATE INDEX idx_epic_relationships_from ON epic_relationships(from_epic_id);
  CREATE INDEX idx_epic_relationships_to ON epic_relationships(to_epic_id);
  CREATE INDEX idx_epic_relationships_type ON epic_relationships(relationship_type);
  ```
- **Impact**:
  - Full relationship management with semantic types
  - SQL-queryable for analytics and dependency analysis
  - Enables future features: dependency graphs, impact analysis, roadmap visualization
  - Context data arrays remain as lightweight alternative
  - Migration path for existing context_data relationships

---

*Last Updated*: 2026-02-13
