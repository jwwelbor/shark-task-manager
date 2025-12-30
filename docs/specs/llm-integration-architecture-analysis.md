# LLM Integration Architecture Analysis: Shark vs Shrimp

**Date:** 2025-12-29
**Author:** Architect Agent
**Status:** Architectural Recommendation

---

## Executive Summary

After comprehensive analysis of both `shark-task-manager` (Go-based CLI/API) and `shrimp-coctail` (TypeScript MCP server), this document provides architectural recommendations for integrating LLM capabilities into Shark. The analysis reveals fundamentally different approaches: Shark is a **task management database**, while Shrimp is an **LLM prompt orchestration system**. These are complementary, not competing architectures.

**Key Recommendation:** Shark should **NOT** become Shrimp. Instead, Shark should add **minimal, optional LLM hooks** via configuration while maintaining its core strength as a robust task database and CLI tool.

---

## 1. Architectural Comparison

### 1.1 Shrimp-Coctail: LLM Prompt Orchestrator

**Core Architecture:**
- **Model Context Protocol (MCP) Server**: Exposes tools to LLM clients (Claude Desktop, Cursor IDE)
- **Prompt Template Engine**: Handlebars-based templates with environment variable overrides
- **State-Driven Workflows**: Guides LLM through structured thinking (plan → analyze → execute → verify)
- **Multi-Project Support**: Isolated task namespaces with plan management
- **JSON-Based Storage**: File-based persistence in `DATA_DIR`

**Key Technical Patterns:**
```typescript
// Tool Definition (MCP SDK)
export const executeTaskSchema = z.object({
  taskId: z.string().regex(UUID_V4_REGEX),
  projectName: z.string()
});

// Prompt Generation
export function getExecuteTaskPrompt(params: ExecuteTaskPromptParams): string {
  const template = loadPromptFromTemplate("executeTask/index.md");
  return generatePrompt(template, {
    name: task.name,
    description: task.description,
    complexityAssessment,
    relatedFilesSummary,
    dependencyTasks
  });
}

// State Transition Hook
await updateTaskStatus(taskId, TaskStatus.IN_PROGRESS, projectId);
const prompt = getExecuteTaskPrompt({ task, complexityAssessment, ... });
return { content: [{ type: "text", text: prompt }] };
```

**LLM Integration Strategy:**
1. **Guided Workflows**: Each tool call returns structured prompts that guide LLM thinking
2. **Chain-of-Thought**: `process_thought`, `analyze_task`, `reflect_task` tools enforce deliberate reasoning
3. **Context Injection**: Automatically injects dependency summaries, related files, complexity assessments
4. **Template Customization**: Environment variables allow per-tool prompt overrides
5. **Research Mode**: Dedicated workflow for systematic technical investigation

**Dependencies:**
- `@modelcontextprotocol/sdk`: MCP server implementation
- `handlebars`: Template rendering
- `zod`: Schema validation for tool parameters
- Node.js/TypeScript ecosystem

---

### 1.2 Shark Task Manager: Task Database & CLI

**Core Architecture:**
- **Go CLI + HTTP API**: Native binary with Cobra command framework
- **SQLite Database**: ACID-compliant task storage with WAL mode
- **Repository Pattern**: Clean separation of data access layer
- **File-Database Sync**: Bidirectional markdown ↔ database synchronization
- **Auto-Detect Project Root**: Walks directory tree to find `.sharkconfig.json`

**Key Technical Patterns:**
```go
// Repository Layer
func (r *TaskRepository) UpdateTaskStatus(ctx context.Context, taskID string, status string) error {
    tx, err := r.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    _, err = tx.ExecContext(ctx,
        "UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
        status, taskID)

    return tx.Commit()
}

// CLI Command
var startCmd = &cobra.Command{
    Use:   "start <task-key>",
    Short: "Start a task (todo → in_progress)",
    RunE: func(cmd *cobra.Command, args []string) error {
        repo := repository.NewTaskRepository(db)
        err := repo.UpdateTaskStatus(ctx, taskID, "in_progress")
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(task)
        }
        return cli.Success("Task started")
    },
}
```

**Current LLM Support:**
- **JSON Output**: All commands support `--json` flag for machine-readable output
- **Dependency Tracking**: `task next` returns only unblocked tasks
- **Agent Type Filtering**: Tasks tagged with agent type (backend, frontend, etc.)
- **Audit Trail**: Complete task history with timestamps
- **Markdown Integration**: Task files with frontmatter metadata

**Dependencies:**
- Go 1.23.4+
- SQLite3 with CGO
- Cobra CLI framework
- Viper configuration

---

## 2. LLM Capability Comparison

| Capability | Shrimp-Coctail | Shark Task Manager |
|------------|----------------|-------------------|
| **Prompt Engineering** | ✅ Full template system | ❌ None |
| **Guided Workflows** | ✅ Chain-of-thought tools | ❌ None |
| **Status Transition Hooks** | ✅ Returns prompts on status change | ❌ None |
| **Context Injection** | ✅ Auto-loads related files, deps | ⚠️ Manual via JSON output |
| **Complexity Assessment** | ✅ Automated evaluation | ❌ None |
| **Research Mode** | ✅ Dedicated workflow | ❌ None |
| **Template Customization** | ✅ Env vars + file overrides | ❌ None |
| **Multi-Project** | ✅ Project + Plan namespaces | ⚠️ Single DB per repo |
| **Task Database** | ⚠️ JSON files only | ✅ SQLite with ACID |
| **CLI Performance** | ❌ Node.js overhead | ✅ Native Go binary |
| **File-Database Sync** | ❌ No filesystem integration | ✅ Bidirectional sync |
| **Dependency Resolution** | ⚠️ Basic validation | ✅ Foreign keys + triggers |

---

## 3. Architectural Feasibility Analysis

### 3.1 Should Shark Add LLM Integration?

**Arguments FOR:**
1. **Enhanced AI Workflows**: Shark already targets AI agents - adding prompts makes it more powerful
2. **Status Transition Context**: Currently, status changes return minimal output. Could return guidance.
3. **Complexity Assessment**: Shark could evaluate task size/complexity automatically
4. **Template-Driven Prompts**: Config-based prompts don't violate Shark's simplicity principle

**Arguments AGAINST:**
1. **Separation of Concerns**: Shark is a database, not a prompt engine. Keep it focused.
2. **Dependency Bloat**: Adding template engines (Handlebars, Go templates) increases complexity
3. **Performance Impact**: Template rendering on every command slows CLI responsiveness
4. **Maintenance Burden**: Prompt templates require ongoing curation and testing
5. **User Choice**: Some users want raw data tools, not opinionated LLM workflows

**Verdict:** **YES, but minimally and optionally**

---

### 3.2 Recommended Architecture: Config-Based Hooks

**Design Principle:** Add LLM support as **optional configuration hooks** without changing core behavior.

#### 3.2.1 Configuration Schema

Add to `.sharkconfig.json`:

```json
{
  "project_name": "My Project",
  "default_path": "docs/plan",

  "llm": {
    "enabled": false,
    "hooks": {
      "task_start": {
        "enabled": true,
        "template": "{{.TaskKey}} started. Context: {{.Description}}. Dependencies: {{range .Dependencies}}{{.Title}} ({{.Status}}){{end}}",
        "inject_related_files": true,
        "inject_dependencies": true,
        "complexity_assessment": true
      },
      "task_complete": {
        "enabled": true,
        "template": "Task {{.TaskKey}} ready for review. Verify: {{.VerificationCriteria}}"
      },
      "task_next": {
        "enabled": true,
        "template": "Next task: {{.TaskKey}} - {{.Title}}. Agent: {{.AgentType}}. Priority: {{.Priority}}."
      }
    },
    "templates_dir": "shark-templates/prompts"
  }
}
```

#### 3.2.2 Implementation Design

**File Structure:**
```
.sharkconfig.json               # Config with llm.hooks
shark-templates/
  prompts/
    task_start.tmpl             # Go text/template
    task_complete.tmpl
    task_next.tmpl
    complexity_assessment.tmpl
  epic.md                        # Existing templates
  feature.md
  task.md
```

**Code Architecture:**

```go
// internal/llm/hooks.go
package llm

import (
    "text/template"
    "github.com/jwwelbor/shark-task-manager/internal/models"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

type HookContext struct {
    TaskKey              string
    Title                string
    Description          string
    Status               string
    AgentType            string
    Priority             int
    Dependencies         []DependencyContext
    RelatedFiles         []string
    ComplexityLevel      string
    ComplexityMetrics    map[string]interface{}
    VerificationCriteria string
}

type LLMHooks struct {
    config   *config.LLMConfig
    templates map[string]*template.Template
}

func NewLLMHooks(cfg *config.Config) (*LLMHooks, error) {
    if !cfg.LLM.Enabled {
        return &LLMHooks{enabled: false}, nil
    }

    hooks := &LLMHooks{
        config:    &cfg.LLM,
        templates: make(map[string]*template.Template),
    }

    // Load templates from templates_dir
    for hookName, hookCfg := range cfg.LLM.Hooks {
        if !hookCfg.Enabled {
            continue
        }

        tmplPath := filepath.Join(cfg.LLM.TemplatesDir, hookName + ".tmpl")
        tmpl, err := template.ParseFiles(tmplPath)
        if err != nil {
            // Fallback to inline template from config
            tmpl, err = template.New(hookName).Parse(hookCfg.Template)
        }
        hooks.templates[hookName] = tmpl
    }

    return hooks, nil
}

func (h *LLMHooks) OnTaskStart(ctx context.Context, task *models.Task, deps []*models.Task) (string, error) {
    if !h.enabled() || h.templates["task_start"] == nil {
        return "", nil // No hook output
    }

    hookCtx := h.buildContext(task, deps)

    // Optionally inject related files
    if h.config.Hooks["task_start"].InjectRelatedFiles {
        hookCtx.RelatedFiles = h.loadRelatedFiles(task)
    }

    // Optionally assess complexity
    if h.config.Hooks["task_start"].ComplexityAssessment {
        hookCtx.ComplexityLevel, hookCtx.ComplexityMetrics = h.assessComplexity(task)
    }

    var buf bytes.Buffer
    err := h.templates["task_start"].Execute(&buf, hookCtx)
    return buf.String(), err
}
```

**CLI Integration:**

```go
// internal/cli/commands/task.go
var startCmd = &cobra.Command{
    Use:   "start <task-key>",
    Short: "Start a task (todo → in_progress)",
    RunE: func(cmd *cobra.Command, args []string) error {
        repo := repository.NewTaskRepository(db)
        task, err := repo.GetTaskByKey(ctx, args[0])

        // Update status (existing logic)
        err = repo.UpdateTaskStatus(ctx, task.ID, "in_progress")

        // NEW: LLM hook (optional)
        if cli.GlobalConfig.LLM.Enabled {
            deps, _ := repo.GetTaskDependencies(ctx, task.ID)
            prompt, _ := cli.LLMHooks.OnTaskStart(ctx, task, deps)

            if prompt != "" {
                if cli.GlobalConfig.JSON {
                    return cli.OutputJSON(map[string]interface{}{
                        "task": task,
                        "llm_context": prompt,
                    })
                }
                fmt.Println("\n--- LLM Context ---")
                fmt.Println(prompt)
                fmt.Println("--- End Context ---\n")
            }
        }

        // Standard output (existing logic)
        if cli.GlobalConfig.JSON {
            return cli.OutputJSON(task)
        }
        return cli.Success(fmt.Sprintf("Task %s started", task.Key))
    },
}
```

---

### 3.3 Complexity Assessment Implementation

**Algorithm (inspired by Shrimp):**

```go
// internal/llm/complexity.go
package llm

type ComplexityLevel string

const (
    ComplexityLow      ComplexityLevel = "low"
    ComplexityMedium   ComplexityLevel = "medium"
    ComplexityHigh     ComplexityLevel = "high"
    ComplexityVeryHigh ComplexityLevel = "very_high"
)

type ComplexityMetrics struct {
    DescriptionLength int
    DependenciesCount int
    NotesLength       int
    EstimatedHours    float64
}

func AssessComplexity(task *models.Task) (ComplexityLevel, ComplexityMetrics) {
    metrics := ComplexityMetrics{
        DescriptionLength: len(task.Description),
        DependenciesCount: len(task.Dependencies),
        NotesLength:       len(task.Notes),
    }

    // Scoring algorithm
    score := 0
    if metrics.DescriptionLength > 500 { score += 2 }
    if metrics.DescriptionLength > 1000 { score += 2 }
    if metrics.DependenciesCount > 3 { score += 2 }
    if metrics.DependenciesCount > 5 { score += 2 }
    if metrics.NotesLength > 300 { score += 1 }

    level := ComplexityLow
    if score >= 8 { level = ComplexityVeryHigh }
    else if score >= 5 { level = ComplexityHigh }
    else if score >= 3 { level = ComplexityMedium }

    return level, metrics
}
```

---

### 3.4 Alternative: Shark as MCP Server

**Design:** Make Shark expose an MCP server endpoint alongside the CLI.

**Pros:**
- Native integration with Claude Desktop, Cursor IDE
- Leverage existing Shark CLI commands as MCP tools
- No need to rewrite Shark - wrap existing functionality

**Cons:**
- Requires Node.js/TypeScript bridge or Go MCP SDK (immature)
- MCP SDK is TypeScript-first - Go support limited
- Adds deployment complexity (HTTP server always running)
- Duplicates Shrimp's functionality

**Verdict:** **Not recommended.** Shark's strength is as a CLI tool. Use Shrimp for MCP integration instead.

---

## 4. Recommended Implementation Roadmap

### Phase 1: Minimal Hooks (1-2 weeks)

**Goal:** Add optional LLM context output without changing core behavior.

**Tasks:**
1. Add `llm` section to `.sharkconfig.json` schema
2. Create `internal/llm/` package with:
   - `hooks.go`: Hook registration and execution
   - `complexity.go`: Task complexity assessment
   - `context.go`: Context building utilities
3. Add template support:
   - Use Go's `text/template` (no external deps)
   - Load templates from `shark-templates/prompts/`
   - Fallback to inline templates from config
4. Integrate hooks into CLI commands:
   - `task start`: Inject context, dependencies, complexity
   - `task complete`: Inject verification guidance
   - `task next`: Inject task recommendation reasoning
5. Add `--no-llm` flag to disable hooks per-command
6. Update documentation with LLM integration guide

**Acceptance Criteria:**
- All existing tests pass (no breaking changes)
- LLM hooks disabled by default (`llm.enabled: false`)
- CLI performance impact < 50ms when hooks enabled
- Zero impact when hooks disabled (no code execution)

---

### Phase 2: Enhanced Context (2-3 weeks)

**Goal:** Richer context injection for complex workflows.

**Tasks:**
1. **Related Files Loading:**
   - Parse task markdown frontmatter for `related-docs` field
   - Load file contents and inject into context
   - Add file size limits to prevent prompt bloat
2. **Dependency Context:**
   - Load completed dependency task summaries
   - Inject into prompt as "prerequisite knowledge"
   - Include dependency status indicators
3. **Complexity Recommendations:**
   - Generate actionable recommendations based on complexity level
   - Suggest task splitting for high complexity tasks
   - Warn about dependency complexity
4. **Template Variables:**
   - Add epic/feature metadata to context
   - Include project statistics (progress, velocity)
   - Add agent-specific guidance (frontend vs backend)

**Acceptance Criteria:**
- Context injection configurable per hook
- Related files loaded only when explicitly configured
- Complexity assessment cached (no re-computation)
- Template variables documented

---

### Phase 3: Advanced Features (3-4 weeks)

**Goal:** Workflow guidance and research mode integration.

**Tasks:**
1. **Workflow Prompts:**
   - Add `workflow` templates for common patterns (feature development, bugfix, refactor)
   - Integrate with task status transitions
   - Provide next-step guidance based on current state
2. **Research Mode Stub:**
   - Add `shark research <topic>` command
   - Provide structured prompts for LLM-driven research
   - Save research results to task notes
3. **Custom Prompt Scripts:**
   - Allow shell scripts in `shark-templates/prompts/`
   - Execute scripts to generate dynamic prompts
   - Pass task data as JSON via stdin
4. **Prompt Testing:**
   - Add `shark test-prompt <hook-name>` command
   - Dry-run hook with sample task data
   - Validate template rendering

**Acceptance Criteria:**
- Workflow prompts improve agent task completion rate
- Research mode provides structured investigation prompts
- Custom scripts sandboxed (no arbitrary code execution)
- Prompt testing catches template errors before use

---

## 5. Configuration Examples

### 5.1 Minimal Configuration (Disabled)

```json
{
  "project_name": "My Project",
  "llm": {
    "enabled": false
  }
}
```

**Behavior:** Zero LLM functionality. Shark behaves exactly as it does today.

---

### 5.2 Basic Hooks (Task Start Only)

```json
{
  "project_name": "My Project",
  "llm": {
    "enabled": true,
    "hooks": {
      "task_start": {
        "enabled": true,
        "template": "Task: {{.TaskKey}} - {{.Title}}\n\nDescription: {{.Description}}\n\nDependencies: {{range .Dependencies}}\n- {{.Title}} ({{.Status}}){{end}}\n\nPriority: {{.Priority}}"
      }
    }
  }
}
```

**Output:**
```bash
$ shark task start T-E04-F01-001 --json
{
  "task": { ... },
  "llm_context": "Task: T-E04-F01-001 - Implement user authentication\n\nDescription: Add JWT-based auth...\n\nDependencies:\n- Database schema migration (completed)\n\nPriority: 3"
}
```

---

### 5.3 Full Context Injection

```json
{
  "project_name": "My Project",
  "llm": {
    "enabled": true,
    "templates_dir": "shark-templates/prompts",
    "hooks": {
      "task_start": {
        "enabled": true,
        "inject_related_files": true,
        "inject_dependencies": true,
        "complexity_assessment": true,
        "max_file_size_kb": 50
      },
      "task_complete": {
        "enabled": true,
        "template": "Task {{.TaskKey}} completed. Review checklist:\n{{.VerificationCriteria}}"
      },
      "task_next": {
        "enabled": true,
        "inject_epic_context": true
      }
    }
  }
}
```

**Template File** (`shark-templates/prompts/task_start.tmpl`):

```
# Task Execution: {{.TaskKey}}

**Title:** {{.Title}}
**Agent:** {{.AgentType}}
**Complexity:** {{.ComplexityLevel}} ({{.ComplexityMetrics.DescriptionLength}} chars, {{.ComplexityMetrics.DependenciesCount}} deps)

## Description
{{.Description}}

## Dependencies
{{range .Dependencies}}
### {{.Title}} ({{.Status}})
{{if .Summary}}
{{.Summary}}
{{end}}
{{end}}

## Related Files
{{range .RelatedFiles}}
- {{.}}
{{end}}

## Recommendations
{{if eq .ComplexityLevel "very_high"}}
⚠️ **Warning:** This task is very complex. Consider splitting into smaller tasks.
{{else if eq .ComplexityLevel "high"}}
**Notice:** High complexity task. Plan implementation carefully.
{{end}}

Begin execution following project standards.
```

---

## 6. Comparison with Shrimp Integration

### 6.1 When to Use Shark + LLM Hooks

**Best For:**
- **Standalone CLI workflows**: Developers running `shark` commands directly
- **Git-integrated workflows**: Bidirectional sync with markdown task files
- **Performance-critical operations**: Native Go binary with SQLite
- **Strict data integrity**: ACID transactions, foreign keys, triggers
- **Single project focus**: One database per repository
- **Custom prompt control**: Team-specific templates via config

**Use Cases:**
- Backend developer using Shark CLI in VSCode terminal
- CI/CD pipeline querying task status via `shark task list --json`
- Project manager generating reports with `shark epic get E04 --json`
- AI agent with custom prompt templates in `.sharkconfig.json`

---

### 6.2 When to Use Shrimp-Coctail

**Best For:**
- **Claude Desktop / Cursor IDE integration**: MCP native support
- **Multi-project context switching**: Isolated namespaces per project
- **Guided LLM workflows**: Chain-of-thought, reflection, research mode
- **Plan management**: Track multiple execution plans per project
- **Prompt experimentation**: Environment variable overrides
- **Task memory**: Automatic backups to memory directory

**Use Cases:**
- Developer using Claude Desktop with MCP integration
- AI agent requiring structured thinking workflows
- Research-driven development with `research_mode` tool
- Project with multiple parallel feature branches (plan per branch)

---

### 6.3 Hybrid Approach: Shark + Shrimp

**Architecture:**

```
┌─────────────────────────────────────────────────────────┐
│                   Claude Desktop / Cursor               │
└─────────────────────────────────────────────────────────┘
                            │
                            │ MCP Protocol
                            ▼
┌─────────────────────────────────────────────────────────┐
│              Shrimp-Coctail MCP Server                  │
│  - Prompt orchestration                                 │
│  - Guided workflows (plan → analyze → execute)          │
│  - Multi-project + plan management                      │
└─────────────────────────────────────────────────────────┘
                            │
                            │ Shell Commands
                            ▼
┌─────────────────────────────────────────────────────────┐
│                 Shark Task Manager                      │
│  - SQLite task database                                 │
│  - File-database sync                                   │
│  - Dependency resolution                                │
│  - (Optional) LLM hooks for context                     │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  Git Repo     │
                    │  docs/plan/   │
                    └───────────────┘
```

**Integration Strategy:**

1. **Shrimp calls Shark commands**:
   ```typescript
   // In Shrimp's executeTask tool
   const taskDetails = await exec(`shark task get ${taskId} --json`);
   const prompt = getExecuteTaskPrompt({
       task: JSON.parse(taskDetails),
       // ... Shrimp-specific context
   });
   ```

2. **Shark provides data, Shrimp provides workflow**:
   - Shark: Task CRUD, status transitions, dependency resolution
   - Shrimp: Prompt generation, chain-of-thought, research mode

3. **Shark's LLM hooks used for Shark-native workflows**:
   - Direct CLI usage gets Shark's config-based prompts
   - Shrimp-mediated usage gets Shrimp's template-based prompts

---

## 7. Security & Performance Considerations

### 7.1 Security

**Template Injection Risk:**
- User-provided templates could execute arbitrary code
- **Mitigation:** Use Go's `text/template` (safe by default, no code execution)
- Validate template syntax before loading
- Document template variable escaping

**File Access Control:**
- Related files injection could expose sensitive data
- **Mitigation:** Add `max_file_size_kb` config limit
- Whitelist file extensions (`.md`, `.txt`, `.go`)
- Never inject `.env`, `secrets.yaml`, etc.

**Prompt Injection:**
- Malicious task descriptions could manipulate LLM behavior
- **Mitigation:** Escape template variables (automatic in `text/template`)
- Document sanitization best practices
- Add `--raw` flag to disable escaping if needed

---

### 7.2 Performance

**Template Rendering Overhead:**
- Initial concern: Template parsing on every command
- **Mitigation:** Cache parsed templates in `LLMHooks` struct
- Lazy-load templates (only parse when first used)
- Benchmark target: < 10ms overhead per command

**File Loading Overhead:**
- Loading related files could slow status transitions
- **Mitigation:** Make file injection opt-in per hook
- Implement file size limits (default 50KB)
- Use Go's efficient file I/O (no Node.js overhead)

**Complexity Assessment:**
- Calculating complexity on every task query is wasteful
- **Mitigation:** Cache complexity in task metadata
- Recalculate only when task changes
- Store in SQLite as JSON column

**Benchmark Targets (with LLM hooks enabled):**
- `shark task start`: < 100ms total (< 50ms overhead)
- `shark task next`: < 150ms total (< 50ms overhead)
- `shark task list`: < 200ms for 1000 tasks (< 50ms overhead)

---

## 8. Migration Path for Existing Users

**Problem:** Existing Shark users have no LLM config. Adding LLM features must not break their workflows.

**Solution: Default-Disabled, Opt-In Design**

### 8.1 Config Migration

**Existing config** (`.sharkconfig.json`):
```json
{
  "project_name": "My Project",
  "default_path": "docs/plan"
}
```

**After upgrade** (automatic):
```json
{
  "project_name": "My Project",
  "default_path": "docs/plan",
  "llm": {
    "enabled": false
  }
}
```

**Behavior:** No changes. LLM hooks never execute.

---

### 8.2 Opt-In Flow

1. **User runs new command:**
   ```bash
   $ shark llm init
   ```

2. **Shark generates default config:**
   ```
   ✓ Created shark-templates/prompts/
   ✓ Generated task_start.tmpl
   ✓ Generated task_complete.tmpl
   ✓ Generated task_next.tmpl
   ✓ Updated .sharkconfig.json (llm.enabled = true)

   LLM hooks enabled. Edit templates in shark-templates/prompts/
   ```

3. **User customizes templates:**
   ```bash
   $ vim shark-templates/prompts/task_start.tmpl
   ```

4. **User tests templates:**
   ```bash
   $ shark llm test task_start --task=T-E04-F01-001

   --- Rendered Template ---
   Task: T-E04-F01-001 - Implement auth
   Description: Add JWT tokens...
   --- End Template ---
   ```

5. **User disables specific hooks:**
   ```bash
   $ shark llm disable task_complete
   Disabled hook: task_complete
   ```

---

### 8.3 Backward Compatibility

**Guarantee:** All existing commands work identically with `llm.enabled: false` (default).

**Testing Strategy:**
1. Run full test suite with `llm.enabled: false` → must pass
2. Run full test suite with `llm.enabled: true` → must pass + additional context
3. Add integration tests for hook rendering
4. Add performance benchmarks for overhead measurement

---

## 9. Documentation Requirements

### 9.1 User-Facing Documentation

**New Files:**
- `docs/LLM_INTEGRATION.md`: Comprehensive LLM hooks guide
- `docs/TEMPLATE_REFERENCE.md`: Template variable documentation
- `docs/COMPLEXITY_ASSESSMENT.md`: How complexity is calculated

**Updated Files:**
- `README.md`: Add LLM integration section
- `docs/CLI_REFERENCE.md`: Document `shark llm` commands
- `.sharkconfig.json`: Add comments for LLM config

---

### 9.2 Developer Documentation

**New Files:**
- `internal/llm/README.md`: LLM package architecture
- `internal/llm/hooks_test.go`: Hook testing examples
- `dev-artifacts/2025-12-29-llm-integration/`: Implementation notes

**Updated Files:**
- `CLAUDE.md`: Add LLM integration patterns
- `TESTING.md`: Add hook testing guidelines

---

## 10. Success Criteria

### 10.1 Technical Metrics

- [ ] All existing tests pass with `llm.enabled: false`
- [ ] LLM hooks add < 50ms overhead per command
- [ ] Template rendering cached (< 1ms after first load)
- [ ] File injection respects size limits (default 50KB)
- [ ] Complexity assessment cached in database
- [ ] Zero performance impact when hooks disabled

---

### 10.2 User Experience Metrics

- [ ] LLM hooks disabled by default (zero breaking changes)
- [ ] `shark llm init` generates working templates
- [ ] Template syntax errors caught before execution
- [ ] `--no-llm` flag works for all commands
- [ ] Documentation explains all template variables
- [ ] Examples provided for common use cases

---

### 10.3 Integration Metrics

- [ ] Shark + Shrimp hybrid workflow documented
- [ ] Shrimp can call Shark via shell commands
- [ ] Shark's LLM context compatible with Shrimp's prompts
- [ ] No conflict between Shark hooks and Shrimp MCP tools

---

## 11. Risks & Mitigation

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **Performance degradation** | High | Medium | Benchmark all hooks, cache templates, lazy-load files |
| **Template injection attacks** | High | Low | Use `text/template` (safe), validate syntax, escape variables |
| **Config complexity** | Medium | High | Default-disabled, `shark llm init` wizard, clear docs |
| **Breaking changes** | High | Low | Comprehensive backward compatibility testing |
| **Prompt quality** | Medium | Medium | Provide tested default templates, document variables |
| **Maintenance burden** | Medium | High | Keep templates in-tree, version with Shark releases |

---

## 12. Conclusion

### 12.1 Recommendation Summary

**Adopt a minimal, config-based LLM hook system:**

1. **Add optional LLM hooks** via `.sharkconfig.json`
2. **Use Go's `text/template`** (no external dependencies)
3. **Default-disabled** (zero breaking changes)
4. **Inject context on status transitions** (start, complete, next)
5. **Add complexity assessment** algorithm
6. **Provide tested default templates**
7. **Document hybrid Shark + Shrimp workflows**

**Do NOT:**
- ❌ Rewrite Shark as an MCP server
- ❌ Add Node.js/TypeScript dependencies
- ❌ Make LLM integration mandatory
- ❌ Duplicate Shrimp's workflow orchestration

---

### 12.2 Why This Approach Works

1. **Preserves Shark's Strengths:**
   - Native Go performance
   - SQLite data integrity
   - File-database sync
   - CLI-first design

2. **Adds Value for AI Workflows:**
   - Contextual prompts on status transitions
   - Complexity-aware recommendations
   - Customizable per-team templates
   - Zero impact when disabled

3. **Complements Shrimp:**
   - Shark = task database + sync
   - Shrimp = prompt orchestration + MCP
   - Hybrid = best of both worlds

4. **Low Risk, High Reward:**
   - Minimal code changes (new `internal/llm/` package)
   - Backward compatible (default-disabled)
   - Easy to test (hook mocking, template validation)
   - Clear migration path (opt-in via `shark llm init`)

---

### 12.3 Next Steps

1. **Review this document** with stakeholders
2. **Prototype Phase 1** (minimal hooks)
3. **Benchmark performance** overhead
4. **Test with real AI workflows**
5. **Iterate on template quality**
6. **Document integration patterns**
7. **Release as optional feature** (v1.x.0)

---

## Appendix A: Template Variable Reference

**Available in all hooks:**
- `{{.TaskKey}}`: Task key (e.g., `T-E04-F01-001`)
- `{{.Title}}`: Task title
- `{{.Description}}`: Task description
- `{{.Status}}`: Current status (`todo`, `in_progress`, etc.)
- `{{.Priority}}`: Priority (1-10)
- `{{.AgentType}}`: Agent type (`backend`, `frontend`, etc.)

**Available with `inject_dependencies: true`:**
- `{{range .Dependencies}}`: Iterate over dependencies
  - `{{.TaskKey}}`: Dependency task key
  - `{{.Title}}`: Dependency title
  - `{{.Status}}`: Dependency status
  - `{{.Summary}}`: Dependency completion summary (if completed)

**Available with `complexity_assessment: true`:**
- `{{.ComplexityLevel}}`: `low`, `medium`, `high`, `very_high`
- `{{.ComplexityMetrics.DescriptionLength}}`: Description character count
- `{{.ComplexityMetrics.DependenciesCount}}`: Number of dependencies
- `{{.ComplexityMetrics.NotesLength}}`: Notes character count

**Available with `inject_related_files: true`:**
- `{{range .RelatedFiles}}`: Iterate over related file paths

**Available with `inject_epic_context: true`:**
- `{{.Epic.Key}}`: Epic key
- `{{.Epic.Title}}`: Epic title
- `{{.Epic.Progress}}`: Epic progress percentage
- `{{.Feature.Key}}`: Feature key
- `{{.Feature.Title}}`: Feature title

---

## Appendix B: Example Use Cases

### B.1 Backend Developer Using Shark CLI

**Scenario:** Developer working on authentication feature.

**Workflow:**
```bash
# Get next task
$ shark task next --agent=backend --json
{
  "task": {
    "key": "T-E04-F01-003",
    "title": "Implement JWT token generation",
    "description": "Create service method to generate JWT tokens...",
    "priority": 3,
    "agent_type": "backend"
  },
  "llm_context": "Task: T-E04-F01-003 - Implement JWT token generation\n\nDependencies:\n- Database schema migration (completed)\n- User model implementation (completed)\n\nRecommendation: Medium complexity task. Review JWT library docs before implementation."
}

# Start task (LLM hook injects context)
$ shark task start T-E04-F01-003
Task T-E04-F01-003 started

--- LLM Context ---
Task: T-E04-F01-003 - Implement JWT token generation

Description: Create service method to generate JWT tokens with user claims...

Dependencies:
- Database schema migration (completed)
  Summary: Added users table with auth fields
- User model implementation (completed)
  Summary: Created User model with password hashing

Related Files:
- internal/auth/service.go
- internal/models/user.go

Complexity: medium (350 chars, 2 deps)

Begin execution following project standards.
--- End Context ---

# ... developer implements feature ...

# Complete task
$ shark task complete T-E04-F01-003
Task T-E04-F01-003 ready for review

--- LLM Context ---
Task T-E04-F01-003 completed. Review checklist:
- JWT tokens generated correctly
- Token expiration works
- Unit tests pass
- Integration tests pass
--- End Context ---
```

**Benefits:**
- Developer sees dependency context without manual lookup
- Complexity assessment warns about potential issues
- Completion checklist ensures thorough review

---

### B.2 AI Agent Using Shark + Shrimp

**Scenario:** Claude Desktop agent working via Shrimp MCP.

**Workflow:**
1. **Agent calls Shrimp's `plan_task` tool:**
   ```
   User: "Add user authentication to the app"

   Shrimp → Calls plan_task("Add user authentication", project="MyApp")
   Shrimp → Returns structured planning prompt
   Claude → Analyzes requirements, breaks into tasks
   Claude → Calls split_tasks with task list
   ```

2. **Shrimp creates tasks in Shark:**
   ```typescript
   // Shrimp's split_tasks tool
   for (const taskData of taskList) {
       await exec(`shark task create \
           --epic=${epicKey} \
           --feature=${featureKey} \
           --title="${taskData.name}" \
           --description="${taskData.description}" \
           --agent=${taskData.agent} \
           --json`);
   }
   ```

3. **Agent executes tasks via Shrimp:**
   ```
   Claude → Calls execute_task(taskId, project="MyApp")
   Shrimp → Calls `shark task start ${taskId} --json`
   Shark → Returns task + LLM context (if enabled)
   Shrimp → Combines with Shrimp's template
   Shrimp → Returns final prompt to Claude
   Claude → Implements feature
   ```

4. **Task completion:**
   ```
   Claude → Calls verify_task(taskId)
   Shrimp → Calls `shark task complete ${taskId} --json`
   Shark → Returns completion context
   Shrimp → Validates and marks complete
   ```

**Benefits:**
- Shark provides data integrity and dependency resolution
- Shrimp provides workflow orchestration and MCP integration
- Shark's LLM hooks add extra context (optional)
- Both systems remain independent and maintainable

---

## Appendix C: Alternative Architectures Considered

### C.1 Shark as Full MCP Server

**Design:** Rewrite Shark to expose MCP protocol directly.

**Rejected Because:**
- Go MCP SDK immature (TypeScript SDK is canonical)
- Would require HTTP server running constantly
- Duplicates Shrimp's functionality
- Violates Shark's CLI-first design
- Increases deployment complexity

---

### C.2 Embedded Scripting Engine (Lua, JavaScript)

**Design:** Add Lua/JavaScript engine for dynamic prompt generation.

**Rejected Because:**
- Significant security risk (arbitrary code execution)
- Large dependency footprint (gopher-lua, goja)
- Performance overhead (VM initialization)
- Go's `text/template` sufficient for 95% of use cases

---

### C.3 External Prompt Service (HTTP API)

**Design:** POST task data to external service, receive prompt.

**Rejected Because:**
- Network latency on every command
- Requires service deployment and maintenance
- Privacy concerns (task data sent externally)
- Overkill for template rendering

---

## Appendix D: Implementation Checklist

### Phase 1: Minimal Hooks
- [ ] Design `.sharkconfig.json` LLM schema
- [ ] Create `internal/llm/` package structure
- [ ] Implement `hooks.go` with template loading
- [ ] Implement `complexity.go` with assessment algorithm
- [ ] Add hook calls to `task start` command
- [ ] Add hook calls to `task complete` command
- [ ] Add hook calls to `task next` command
- [ ] Add `--no-llm` flag to all commands
- [ ] Write unit tests for hook rendering
- [ ] Write integration tests for CLI output
- [ ] Benchmark performance overhead
- [ ] Document LLM integration in `docs/`

### Phase 2: Enhanced Context
- [ ] Parse task frontmatter for `related-docs`
- [ ] Implement file loading with size limits
- [ ] Add dependency summary injection
- [ ] Cache complexity assessments in SQLite
- [ ] Add epic/feature metadata to context
- [ ] Create default template files
- [ ] Add `shark llm init` command
- [ ] Add `shark llm test` command
- [ ] Update CLI reference documentation

### Phase 3: Advanced Features
- [ ] Design workflow template structure
- [ ] Implement `shark research` command
- [ ] Add custom script execution (sandboxed)
- [ ] Create example templates for common workflows
- [ ] Document Shark + Shrimp hybrid architecture
- [ ] Write migration guide for existing users
- [ ] Performance optimization (template caching)
- [ ] Security audit (template injection risks)

---

**End of Document**
