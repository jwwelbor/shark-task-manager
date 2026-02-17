# E07-F30: Template Engine - Tactical Research Report

**Date**: 2026-02-14
**Researcher**: AI Agent (Researcher)
**Epic**: E07 - Enhancements
**Feature**: E07-F30 - External Template Engine for Orchestrator Instructions

---

## Executive Summary

Analyzed codebase integration points for external template engine feature. Found well-established template infrastructure (`internal/templates/`) for task markdown rendering, but orchestrator actions use simple string replacement (`PopulateTemplate`). Integration points identified: service layer (`resolveAction` methods), config layer (`OrchestratorAction.PopulateTemplate`), and template helpers (`*PlaceholdersWithRelated`). Recommendation: **Extend existing system** rather than new implementation. File: `internal/templates/renderer.go` already uses Go `text/template` with custom functions - proven pattern to replicate for orchestrator actions.

---

## 1. Codebase Patterns Relevant to This Feature

### 1.1 Existing Template Infrastructure

**Location**: `/home/jwwel/projects/shark-task-manager/internal/templates/`

Shark already has a sophisticated template rendering system for task markdown files:

**Files**:
- `internal/templates/renderer.go` - Template execution with `text/template`
- `internal/templates/loader.go` - Template loading from embedded files and filesystem
- `internal/templates/renderer_test.go` - Test suite

**Key Pattern (Renderer)**:
```go
// Renderer uses Go's text/template with custom functions
type Renderer struct {
    loader *Loader
}

func (r *Renderer) Render(agentType string, data TemplateData) (string, error) {
    tmplContent, err := r.loader.LoadTemplate(agentType)
    tmpl, err := template.New("task").Funcs(templateFuncs()).Parse(tmplContent)
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute template: %w", err)
    }
    return buf.String(), nil
}

// Custom template functions
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "join": func(items []string, sep string) string { ... },
        "quote": func(items []string) []string { ... },
        "isEmpty": func(s string) bool { ... },
        "formatTime": func(t time.Time) string { ... },
    }
}
```

**Evidence**: This exact pattern can be replicated for orchestrator action templates. The infrastructure for loading, parsing, and executing Go templates already exists and is battle-tested.

### 1.2 Current Template Population (String Replacement)

**Location**: `/home/jwwel/projects/shark-task-manager/internal/config/orchestrator_action.go`

**Current Implementation**:
```go
// OrchestratorAction.PopulateTemplate (line 136-151)
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    if len(vars) == 0 {
        return oa.InstructionTemplate
    }

    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }

    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

**Limitations**:
- No conditionals (can't hide empty sections)
- No loops or complex logic
- No shared partials
- Embedded as JSON strings in `.sharkconfig.json`

**Extension Point**: This method is the **primary integration point**. Add detection: if `InstructionTemplate` ends with `.tmpl`, use template engine; otherwise use legacy string replacement.

---

## 2. Existing Implementations with File Paths

### 2.1 Template Rendering (Task Markdown Files)

**File**: `/home/jwwel/projects/shark-task-manager/internal/templates/renderer.go`

**Usage Pattern**:
- Template loader reads from embedded files (`//go:embed task_templates/*.md`)
- Falls back to filesystem if embedded not found
- Supports agent-specific templates with fallback to general template

**Relevance**: Same dual-loading approach (embedded + filesystem) can work for orchestrator templates.

### 2.2 Template Variable Population

**File**: `/home/jwwel/projects/shark-task-manager/internal/config/template_helpers.go`

**Functions** (lines 16-377):
- `TaskPlaceholders(task)` → `map[string]string` (basic fields)
- `TaskPlaceholdersWithRelated(ctx, task, docRepo, taskRelRepo)` → `map[string]string` (with relationships)
- `FeaturePlaceholders(feature)` → `map[string]string`
- `FeaturePlaceholdersWithRelated(ctx, feature, docRepo, relRepo)` → `map[string]string`
- `EpicPlaceholders(epic)` → `map[string]string`
- `EpicPlaceholdersWithRelated(epic, docRepo, relRepo, ctx)` → `map[string]string`

**Evidence**: These functions build `map[string]string` for simple placeholder replacement. For Go templates, these maps can be converted to structs or passed directly as template data.

**Extension Needed**: Add metadata fields to placeholder maps:
- `complexity_tier` (from feature/task metadata)
- `related_docs` (already present via `*WithRelated` functions)
- `related_tasks`, `related_features`, `related_epics` (already present)

### 2.3 Action Resolution (Service Layer)

**Files**:
- `/home/jwwel/projects/shark-task-manager/internal/services/epic_service.go` (lines 202-238)
- `/home/jwwel/projects/shark-task-manager/internal/services/feature_service.go` (lines 210-238)
- `/home/jwwel/projects/shark-task-manager/internal/services/display_service.go` (lines 294, similar pattern)

**Pattern** (`resolveAction` method):
```go
// EpicService.resolveAction (line 206)
func (s *EpicService) resolveAction(ctx context.Context, epic *models.Epic, status string) *config.PopulatedAction {
    wf := s.workflowSvc.GetWorkflow()
    if wf == nil || wf.StatusMetadata == nil {
        return nil
    }

    metadata, exists := wf.StatusMetadata[status]
    if !exists || metadata.OrchestratorAction == nil {
        return nil
    }

    action := metadata.OrchestratorAction

    // Get placeholders (with relationships if repos available)
    var vars map[string]string
    if s.docRepo != nil && s.relRepo != nil {
        vars = config.EpicPlaceholdersWithRelated(epic, s.docRepo, s.relRepo, ctx)
    } else {
        vars = config.EpicPlaceholders(epic)
    }

    // INTEGRATION POINT: Populate template
    instruction := action.PopulateTemplate(vars)

    return &config.PopulatedAction{
        Action:      action.Action,
        AgentType:   action.AgentType,
        Skills:      action.Skills,
        Instruction: instruction,
    }
}
```

**Identical pattern in FeatureService (line 213), DisplayService.**

**Integration Point**: The `action.PopulateTemplate(vars)` call (lines 230, 233) is where template engine integration happens. This is called from:
- `TransitionStatus()` methods (returns `TransitionResult` with action)
- `GetAvailableTransitions()` methods (wraps each transition with action)
- Display services for `get` commands

---

## 3. Integration Points (Services, APIs, Tables)

### 3.1 Service Layer Integration

**Primary Call Sites** (where orchestrator actions are resolved):

1. **EpicService.TransitionStatus()** (epic_service.go:148)
   - Calls `s.resolveAction(ctx, epic, targetStatus)`
   - Returns `TransitionResult` with populated action

2. **FeatureService.TransitionStatus()** (feature_service.go:156)
   - Calls `s.resolveAction(ctx, feature, targetStatus)`
   - Returns `TransitionResult` with populated action

3. **DisplayService.ResolveEpicAction()** (display_service.go:294)
   - Called when displaying epic details (`shark epic get E07`)

4. **DisplayService.ResolveFeatureAction()** (display_service.go:similar pattern)
   - Called when displaying feature details (`shark feature get E07-F01`)

**Shared Pattern**: All services use same `resolveAction` private method pattern.

### 3.2 Config Layer Integration

**File**: `/home/jwwel/projects/shark-task-manager/internal/config/orchestrator_action.go`

**Struct**:
```go
type OrchestratorAction struct {
    Action              string   `json:"action"`
    AgentType           string   `json:"agent_type,omitempty"`
    Skills              []string `json:"skills,omitempty"`
    InstructionTemplate string   `json:"instruction_template"` // <-- INTEGRATION POINT
}
```

**Method to Extend** (line 140):
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string
```

**Extension Strategy**:
```go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    // Detection: if template ends with .tmpl, use template engine
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        return templateEngine.Render(oa.InstructionTemplate, vars)
    }

    // Otherwise, use legacy string replacement
    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }
    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

### 3.3 Configuration File Structure

**File**: `.sharkconfig.json`

**Current Structure**:
```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["tdd", "implementation"],
        "instruction_template": "Launch developer for {task_id}..."
      }
    }
  }
}
```

**Proposed Structure (backward compatible)**:
```json
{
  "template_directory": "templates",  // Optional, defaults to "templates"
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["tdd", "implementation"],
        "instruction_template": "task/ready_for_development.tmpl"  // .tmpl extension triggers template engine
      }
    }
  }
}
```

**No schema changes needed** - `instruction_template` field works for both inline strings and filenames.

### 3.4 No Database Changes Required

**Evidence**: Orchestrator actions are config-driven only. No database tables store instruction templates.

**Tables Used** (read-only for template population):
- `tasks` - task fields for placeholders
- `features` - feature fields for placeholders
- `epics` - epic fields for placeholders
- `documents` - related docs via foreign keys
- `task_relationships`, `feature_relationships`, `epic_relationships` - related entities

**No Migrations Needed**: All integration is code-only.

---

## 4. Extension vs New Code Analysis

### 4.1 Extend Existing Components

**Recommendation: EXTEND** these components:

1. **Existing: `internal/templates/renderer.go`**
   - **Extend**: Create `OrchestratorRenderer` struct similar to `Renderer`
   - **Reuse**: Template loading pattern, `text/template` usage, custom functions
   - **Justification**: Proven pattern already in production for task markdown

2. **Existing: `internal/templates/loader.go`**
   - **Extend**: Add `LoadOrchestratorTemplate(filename)` method
   - **Reuse**: Embedded file loading, filesystem fallback
   - **Justification**: Same dual-loading strategy needed

3. **Existing: `internal/config/orchestrator_action.go`**
   - **Extend**: Modify `PopulateTemplate()` to detect `.tmpl` suffix
   - **Reuse**: Validation, action types, config structure
   - **Justification**: Maintains backward compatibility with inline templates

4. **Existing: `internal/config/template_helpers.go`**
   - **Extend**: Add complexity tier and metadata to placeholder functions
   - **Reuse**: All existing placeholder logic
   - **Justification**: No changes needed to signature, just add fields to returned map

### 4.2 New Code Required

**New Components to Create**:

1. **New: `internal/templates/orchestrator_renderer.go`**
   - Struct: `OrchestratorRenderer` (similar to `Renderer`)
   - Methods:
     - `RenderTask(templateName, taskData)` → `string, error`
     - `RenderFeature(templateName, featureData)` → `string, error`
     - `RenderEpic(templateName, epicData)` → `string, error`
   - Custom functions:
     - `eq` (equality check for conditionals)
     - `ne` (not equal)
     - `and`, `or` (boolean logic)
     - `join` (CSV formatting)
     - `isEmpty` (check for empty strings)

2. **New: `templates/` directory structure**
   ```
   templates/
   ├── epic/
   │   ├── ready_for_research.tmpl
   │   └── ready_for_refinement_ba.tmpl
   ├── feature/
   │   ├── ready_for_refinement_ba.tmpl
   │   └── ready_for_test_planning.tmpl
   ├── task/
   │   ├── ready_for_development.tmpl
   │   └── ready_for_code_review.tmpl
   └── partials/
       ├── _tdd_process.tmpl
       ├── _exit_gate.tmpl
       └── _read_section.tmpl
   ```

3. **New: Template validation command**
   - `shark config validate --templates` (check syntax of all .tmpl files)
   - Precompile all templates at startup, report errors

4. **New: Migration helper (optional)**
   - `shark config migrate-templates` (extract inline templates to files)
   - Creates `.tmpl` files from inline strings
   - Updates `.sharkconfig.json` to reference files

### 4.3 Reuse vs Rewrite Decision Matrix

| Component | Decision | Rationale |
|-----------|----------|-----------|
| Template parsing | **Reuse** Go stdlib `text/template` | Already used in `renderer.go`, proven |
| Template loading | **Extend** existing `Loader` | Same dual-loading pattern needed |
| Placeholder building | **Extend** existing `*Placeholders` functions | Just add new fields to map |
| Action resolution | **Extend** existing `resolveAction` methods | Single line change to call engine |
| Config schema | **Reuse** existing `OrchestratorAction` | No schema changes needed |
| Validation | **Extend** existing `Validate()` method | Add template syntax checks |
| Custom functions | **New** for orchestrator templates | Different functions needed (eq, ne, conditionals) |
| File structure | **New** `templates/` directory | Organizational, not code |

**Summary**: 70% extension, 30% new code.

---

## 5. Inter-Feature Technical Dependency Map

### 5.1 Dependencies ON This Feature (Blocking)

**E07-F29: Template Variables for Related Docs and Tasks** (COMPLETED)
- **Why Needed**: Provides `{related_docs}`, `{related_tasks}`, `{related_features}`, `{related_epics}` variables
- **Integration Point**: `*PlaceholdersWithRelated()` functions in `internal/config/template_helpers.go`
- **Evidence**: Lines 174-217 (FeaturePlaceholdersWithRelated), 245-300 (TaskPlaceholdersWithRelated), 318-361 (EpicPlaceholdersWithRelated)
- **Status**: ✅ SATISFIED - E07-F29 merged on 2026-02-14
- **Impact**: Can reference related entities in templates immediately

**No other blocking dependencies identified.**

### 5.2 Sibling Features (Potential Interactions)

**E07-F21: Add Actions to Status Transition** (ACTIVE, 78% complete)
- **Interaction**: Introduces orchestrator actions on status transitions
- **Shared Code**: `internal/config/orchestrator_action.go`, service layer `resolveAction` methods
- **Risk**: Low - E07-F21 establishes patterns E07-F30 extends
- **Coordination**: E07-F30 makes E07-F21 actions more maintainable via external templates

**E07-F22: Rejection Reason for Status Transitions** (ACTIVE, 79% complete)
- **Interaction**: Adds `rejection_reason` field to transitions
- **Shared Code**: Service layer transition methods
- **Risk**: None - orthogonal features
- **Benefit**: Rejection reasons can be included in template variables if needed

**E07-F26: Centralized Workflow Service Authority** (ACTIVE, 92% complete)
- **Interaction**: Consolidates workflow validation
- **Shared Code**: `workflow.Service`, status validation
- **Risk**: None - E07-F30 doesn't change validation logic
- **Benefit**: Workflow service provides config for template resolution

**E07-F16: Workflow Config Integration for Status Display** (DRAFT, 51% complete)
- **Interaction**: Reads status metadata from config
- **Shared Code**: `config.WorkflowConfig`, `StatusMetadata`
- **Risk**: None - both features read same config

**E07-F27: Embedded JSON Workflow Profiles** (COMPLETED if referenced in config)
- **Interaction**: Provides default workflow profiles
- **Shared Code**: `internal/init/profiles.go`
- **Risk**: None - profiles include orchestrator actions to externalize

### 5.3 Dependencies BY This Feature (Enables)

**Future: Template Testing Framework**
- Once E07-F30 completes, can add `shark config test-templates` command
- Render templates with mock data, verify output

**Future: Template Library/Sharing**
- External `.tmpl` files can be version controlled separately
- Teams can share template collections across projects

**Future: Hot Reload in Dev Mode**
- With external files, can watch `templates/` directory
- Reload templates on change without restart

---

## 6. Implementation Approach Recommendations

### 6.1 Recommended Phasing (4 Phases)

**Phase 1: Foundation (MVP)**
- **Goal**: Engine works, zero breaking changes
- **Tasks**:
  1. Create `internal/templates/orchestrator_renderer.go` with Go template support
  2. Add `.tmpl` detection to `OrchestratorAction.PopulateTemplate()`
  3. Create `templates/` directory structure
  4. Write unit tests for renderer
  5. Add template precompilation at startup (catch syntax errors early)
- **Deliverable**: Engine ready, 100% backward compatible
- **Testing**: Existing tests still pass, new template tests pass
- **Duration**: 2-3 days

**Phase 2: High-Value Templates (12 Templates)**
- **Goal**: Prove value with most complex templates
- **Templates to Convert**:
  - Task execution (5): `ready_for_development`, `ready_for_code_review`, `ready_for_qa`, `ready_for_refinement_ba`, `ready_for_refinement_tech`
  - Feature planning (4): `ready_for_research`, `ready_for_refinement_ba`, `ready_for_refinement_tech`, `ready_for_test_planning`
  - Epic strategic (3): `ready_for_research`, `ready_for_feasibility_review_ba`, `ready_for_feasibility_review_tech`
- **Partials to Create**:
  - `partials/_tdd_process.tmpl` (used by 5 task templates)
  - `partials/_exit_gate.tmpl` (used by all templates)
  - `partials/_read_section.tmpl` (used by 12+ templates)
- **Deliverable**: 12 external templates, immediate readability wins
- **Testing**: Integration tests with real workflow config
- **Duration**: 3-4 days

**Phase 3: Full Migration (50+ Templates)**
- **Goal**: Externalize all 62 templates
- **Tasks**:
  1. Convert remaining 50 templates to external files
  2. Expand partial library to 6-10 partials
  3. Add `shark config validate --templates` command
  4. Create optional migration tool: `shark config migrate-templates`
  5. Update docs with template authoring guide
  6. Add deprecation notice for inline templates (warning, not error)
- **Deliverable**: Complete external template system
- **Testing**: Full workflow regression tests
- **Duration**: 4-5 days

**Phase 4: Advanced Features (Future)**
- Template inheritance (base templates with overrides)
- Template testing framework (`shark config test-templates`)
- Hot reload in dev mode (watch templates/, reload on change)
- Template linting (enforce conventions, max complexity)
- Template library (shareable across projects)

### 6.2 Technical Implementation Details

**Backward Compatible Detection**:
```go
// In orchestrator_action.go
func (oa *OrchestratorAction) PopulateTemplate(vars map[string]string) string {
    // If value ends with .tmpl, use template engine
    if strings.HasSuffix(oa.InstructionTemplate, ".tmpl") {
        engine := templates.GetOrchestratorEngine()  // Singleton
        return engine.Render(oa.InstructionTemplate, vars)
    }

    // Otherwise, use legacy string replacement
    replacements := make([]string, 0, 2*len(vars))
    for key, value := range vars {
        replacements = append(replacements, "{"+key+"}", value)
    }
    return strings.NewReplacer(replacements...).Replace(oa.InstructionTemplate)
}
```

**Template Engine Singleton**:
```go
// In internal/templates/orchestrator_engine.go
var (
    engineOnce     sync.Once
    engineInstance *OrchestratorRenderer
)

func GetOrchestratorEngine() *OrchestratorRenderer {
    engineOnce.Do(func() {
        engineInstance, _ = NewOrchestratorRenderer("templates")
    })
    return engineInstance
}
```

**Custom Template Functions**:
```go
func orchestratorFuncs() template.FuncMap {
    return template.FuncMap{
        // Conditionals
        "eq":      func(a, b interface{}) bool { return a == b },
        "ne":      func(a, b interface{}) bool { return a != b },
        "and":     func(a, b bool) bool { return a && b },
        "or":      func(a, b bool) bool { return a || b },

        // String helpers
        "join":    strings.Join,
        "isEmpty": func(s string) bool { return strings.TrimSpace(s) == "" },
        "trim":    strings.TrimSpace,

        // Complexity tier helpers
        "isSimple":   func(tier string) bool { return tier == "SIMPLE" },
        "isStandard": func(tier string) bool { return tier == "STANDARD" },
        "isComplex":  func(tier string) bool { return tier == "COMPLEX" },
    }
}
```

### 6.3 Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|------------|------------|
| Breaking existing inline templates | High | Low | Filename detection (`.tmpl` suffix) maintains backward compatibility |
| Template syntax errors at runtime | Medium | Medium | Precompile templates at startup, fail fast with helpful errors |
| Performance degradation | Low | Low | Cache compiled templates, benchmark against string replacement |
| Template complexity explosion | Medium | Medium | Lint templates, enforce max complexity limits |
| Partial template conflicts | Low | Low | Use `_` prefix convention for partials, namespace by entity type |

**Critical Mitigation**: Comprehensive regression testing with existing workflow configs before Phase 3 deployment.

### 6.4 Success Metrics

**Quantitative**:
- All existing tests pass (0 regressions)
- 95%+ test coverage for new template engine code
- Template rendering performance < 5ms per action
- All 62 templates externalized by Phase 3 completion
- Zero inline template warnings after 30-day deprecation period

**Qualitative**:
- Template authors report improved readability
- Git diffs show line-by-line changes (not 500-char single-line changes)
- New templates authored 2x faster with partials
- Onboarding time reduced (no more JSON escaping nightmare)

---

## 7. Open Questions for BA and Architect

### For Business Analyst:

1. **User Stories**: Should migration tool (`shark config migrate-templates`) be part of MVP or Phase 3?
   - **Context**: Helps users convert inline templates to files automatically
   - **Trade-off**: MVP without it means manual conversion; with it adds 2-3 days

2. **Deprecation Timeline**: How long should inline templates be supported before deprecation?
   - **Recommendation**: Show warnings immediately, remove support in 6 months
   - **Alternative**: Support forever (backward compatibility forever)

3. **Partial Template Naming**: Enforce `_partial_name.tmpl` convention or allow any name?
   - **Recommendation**: Require `_` prefix to prevent accidental inclusion
   - **Alternative**: Directory-based (`templates/partials/`)

### For Architect:

1. **Template Caching Strategy**: Global singleton vs per-request parsing?
   - **Current Thinking**: Singleton with precompiled templates (best performance)
   - **Alternative**: Per-request parsing (simpler, slower)
   - **Decision Needed**: Accept performance trade-off or optimize early?

2. **Error Handling**: Fail fast on template errors or graceful degradation?
   - **Recommendation**: Fail fast at startup (precompile all templates)
   - **Alternative**: Warn and fall back to inline template
   - **Decision Needed**: Strictness vs resilience trade-off

3. **Template Validation Timing**: Startup vs lazy validation?
   - **Recommendation**: Validate all templates at `shark` startup
   - **Alternative**: Validate only when template is first used
   - **Decision Needed**: Fast startup vs early error detection

4. **Integration with Workflow Service**: Should `workflow.Service` own template engine?
   - **Current Thinking**: Template engine is independent, called by config layer
   - **Alternative**: `workflow.Service` wraps template engine
   - **Decision Needed**: Layering and ownership

5. **Template Testing Strategy**: Unit tests vs integration tests?
   - **Recommendation**: Both (unit tests for functions, integration for full workflows)
   - **Decision Needed**: Test coverage requirements, mocking strategy

---

## 8. Related Artifacts

**Sibling Feature Documentation**:
- E07-F29 Feature PRD: `/home/jwwel/projects/shark-task-manager/docs/plan/E07-enhancements/E07-F29-template-variables-for-related-docs-and-tasks/prd.md`
- E07-F21 Technical Design: `/home/jwwel/projects/shark-task-manager/docs/plan/E07-enhancements/E07-F21-add-actions-to-status-transition/technical-design.md`

**Codebase References**:
- Template Renderer: `/home/jwwel/projects/shark-task-manager/internal/templates/renderer.go`
- Orchestrator Action: `/home/jwwel/projects/shark-task-manager/internal/config/orchestrator_action.go`
- Template Helpers: `/home/jwwel/projects/shark-task-manager/internal/config/template_helpers.go`
- Epic Service: `/home/jwwel/projects/shark-task-manager/internal/services/epic_service.go` (lines 202-238)
- Feature Service: `/home/jwwel/projects/shark-task-manager/internal/services/feature_service.go` (lines 210-238)

**Configuration Examples**:
- Current Config: `/home/jwwel/projects/shark-task-manager/.sharkconfig.json`
- Advanced Profile: `/home/jwwel/projects/shark-task-manager/internal/init/profiles/advanced.json`

**External Proposal**:
- Original Proposal: `/home/jwwel/.claude/docs/external-prompt-templates-proposal.md` (referenced in feature.md)
- Variable Guide: `/home/jwwel/.claude/docs/instruction-template-variable-usage-guide.md`

---

## Appendix A: File Path Summary

| Category | File Path | Purpose |
|----------|-----------|---------|
| **Integration Points** | | |
| Service Layer | `internal/services/epic_service.go:206` | `resolveAction()` method |
| Service Layer | `internal/services/feature_service.go:213` | `resolveAction()` method |
| Config Layer | `internal/config/orchestrator_action.go:140` | `PopulateTemplate()` method |
| Template Helpers | `internal/config/template_helpers.go:174-377` | Placeholder building functions |
| **Existing Infrastructure** | | |
| Task Renderer | `internal/templates/renderer.go:38-58` | Template execution pattern to replicate |
| Template Loader | `internal/templates/loader.go:34-70` | Loading strategy to extend |
| **Extension Targets** | | |
| Orchestrator Renderer | `internal/templates/orchestrator_renderer.go` | NEW - to create |
| Template Directory | `templates/epic/*.tmpl` | NEW - directory structure |
| Template Directory | `templates/feature/*.tmpl` | NEW - directory structure |
| Template Directory | `templates/task/*.tmpl` | NEW - directory structure |
| Template Directory | `templates/partials/*.tmpl` | NEW - shared partials |

---

## Appendix B: Integration Sequence Diagram

```
User runs: shark task next-status E07-F01-001

1. CLI Command Layer
   └─> task.go:runTaskNextStatus()

2. Service Layer
   ├─> Get database via cli.GetDB()
   ├─> Repository: task_repository.GetByKey(E07-F01-001)
   ├─> Repository: task_repository.UpdateStatus(taskID, nextStatus)
   └─> Service: resolveAction(ctx, task, nextStatus)
       │
       ├─> Get workflow config
       ├─> Check if status has orchestrator_action
       ├─> Build placeholder map: TaskPlaceholdersWithRelated(task, docRepo, relRepo)
       └─> action.PopulateTemplate(vars)  <-- INTEGRATION POINT
           │
           ├─> Detect: does InstructionTemplate end with ".tmpl"?
           │   ├─> YES: Call template engine
           │   │   ├─> Load template file
           │   │   ├─> Parse with text/template
           │   │   ├─> Execute with vars as data
           │   │   └─> Return rendered string
           │   │
           │   └─> NO: Use legacy string replacement
           │       ├─> Build replacements array
           │       └─> strings.NewReplacer().Replace()
           │
           └─> Return populated instruction

3. CLI Output
   └─> Display TransitionResult with orchestrator_action
```

---

**END OF REPORT**

**Next Steps**:
1. Business Analyst reviews for user story creation
2. Architect reviews for technical design approval
3. Developer estimates Phase 1 effort
4. Create tasks for Phase 1 implementation

**Feature Status**: Ready for BA refinement and technical design.

---

**Researcher**: AI Agent (Researcher Role)
**Date**: 2026-02-14
**Version**: 1.0 (Initial Tactical Research)
