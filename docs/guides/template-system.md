# Template System and Multi-Level Workflow Configuration

Shark's orchestrator action system uses a template engine to generate context-rich instructions for AI agents. Templates support placeholder substitution from entity fields, enabling instructions that include task titles, file paths, statuses, and other metadata.

## Template Syntax

Templates use `{placeholder_name}` syntax. Each placeholder is replaced with the corresponding entity field value at runtime.

```
"instruction_template": "Implement {id} - {title} at {file_path}"
```

After substitution for a task with key `T-E07-F01-001`, title `JWT Validation`, and file path `docs/plan/task.md`:

```
"instruction": "Implement T-E07-F01-001 - JWT Validation at docs/plan/task.md"
```

Placeholders not found in the entity's field map are left unchanged in the output. This is intentional -- it allows templates to be validated visually and prevents silent data loss.

## Available Placeholders

Placeholders are populated from entity fields. Each entity level (task, feature, epic) provides a different set of available placeholders.

### Task Placeholders

| Placeholder | Source | Always Present | Example |
|-------------|--------|----------------|---------|
| `{id}` | `task.Key` | Yes | `T-E07-F01-001` |
| `{task_id}` | `task.Key` | Yes | `T-E07-F01-001` |
| `{epic_id}` | `task.Key` | Yes | `T-E07-F01-001` |
| `{feature_id}` | `task.Key` | Yes | `T-E07-F01-001` |
| `{title}` | `task.Title` | Yes | `Implement JWT validation` |
| `{status}` | `task.Status` | Yes | `ready_for_development` |
| `{priority}` | `task.Priority` | Yes | `5` |
| `{created_at}` | `task.CreatedAt` | Yes | `2026-02-10T10:00:00Z` |
| `{updated_at}` | `task.UpdatedAt` | Yes | `2026-02-10T12:30:00Z` |
| `{slug}` | `task.Slug` | Yes | `implement-jwt-validation` |
| `{file_path}` | `task.FilePath` | No | `docs/plan/E07/E07-F01/tasks/T-E07-F01-001.md` |
| `{agent_type}` | `task.AgentType` | No | `developer` |
| `{description}` | `task.Description` | No | `Detailed task description` |
| `{execution_order}` | `task.ExecutionOrder` | No | `3` |
| `{blocked_reason}` | `task.BlockedReason` | No | `Waiting on API spec` |
| `{depends_on}` | `task.DependsOn` | No | `T-E07-F01-001` |
| `{completion_notes}` | `task.CompletionNotes` | No | `All tests passing` |
| `{files_changed}` | `task.FilesChanged` | No | `internal/auth/jwt.go` |

**Note:** `{id}`, `{task_id}`, `{epic_id}`, and `{feature_id}` all resolve to the task key for backward compatibility. Templates written with `{task_id}` continue to work without modification.

### Feature Placeholders

| Placeholder | Source | Always Present | Example |
|-------------|--------|----------------|---------|
| `{id}` | `feature.Key` | Yes | `E07-F01` |
| `{feature_id}` | `feature.Key` | Yes | `E07-F01` |
| `{title}` | `feature.Title` | Yes | `User Authentication` |
| `{status}` | `feature.Status` | Yes | `active` |
| `{created_at}` | `feature.CreatedAt` | Yes | `2026-02-10T10:00:00Z` |
| `{updated_at}` | `feature.UpdatedAt` | Yes | `2026-02-10T12:30:00Z` |
| `{slug}` | `feature.Slug` | Yes | `user-authentication` |
| `{description}` | `feature.Description` | No | `Feature description` |
| `{file_path}` | `feature.FilePath` | No | `docs/plan/E07/E07-F01/feature.md` |
| `{execution_order}` | `feature.ExecutionOrder` | No | `2` |

### Epic Placeholders

| Placeholder | Source | Always Present | Example |
|-------------|--------|----------------|---------|
| `{id}` | `epic.Key` | Yes | `E07` |
| `{epic_id}` | `epic.Key` | Yes | `E07` |
| `{title}` | `epic.Title` | Yes | `User Management` |
| `{status}` | `epic.Status` | Yes | `active` |
| `{priority}` | `epic.Priority` | Yes | `high` |
| `{created_at}` | `epic.CreatedAt` | Yes | `2026-02-10T10:00:00Z` |
| `{updated_at}` | `epic.UpdatedAt` | Yes | `2026-02-10T12:30:00Z` |
| `{slug}` | `epic.Slug` | Yes | `user-management` |
| `{description}` | `epic.Description` | No | `Epic description` |
| `{file_path}` | `epic.FilePath` | No | `docs/plan/E07-user-management/epic.md` |
| `{business_value}` | `epic.BusinessValue` | No | `high` |

### Optional Placeholders

Placeholders marked "No" under "Always Present" are only available when the corresponding field has a value. If the field is nil/empty, the placeholder is omitted from the replacement map and left unchanged in the output.

For example, if a task has no `file_path` set:
```
Template:  "Work on {id} at {file_path}"
Output:    "Work on T-E07-F01-001 at {file_path}"
```

To avoid unreplaced placeholders in output, either ensure the field is populated or avoid using optional placeholders in templates for entities that may not have them set.

## Multi-Level Workflow Configuration

Shark supports orchestrator actions at three entity levels: **tasks** (default), **epics**, and **features**. Each level has its own workflow configuration section in `.sharkconfig.json`.

### Configuration Structure

```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["implementation"],
        "instruction_template": "Implement task {id} - {title}"
      }
    }
  },

  "epic_workflow": {
    "status_metadata": {
      "ready_for_research": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research epic {id}: {title}. Review {file_path}."
        }
      }
    }
  },

  "feature_workflow": {
    "status_metadata": {
      "ready_for_refinement_tech": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "architect",
          "skills": ["architecture"],
          "instruction_template": "Design architecture for feature {id}: {title}"
        }
      }
    }
  }
}
```

### How It Works

| Config Section | Entity Level | Placeholder Source | Resolved By |
|----------------|-------------|-------------------|-------------|
| `status_metadata` | Tasks | `config.TaskPlaceholders(task)` | `task_repository.go` |
| `epic_workflow.status_metadata` | Epics | `config.EpicPlaceholders(epic)` | `epic_service.go` |
| `feature_workflow.status_metadata` | Features | `config.FeaturePlaceholders(feature)` | `feature_service.go` |

When an entity transitions to a status that has an `orchestrator_action` configured, the system:

1. Looks up the `orchestrator_action` from the appropriate workflow config section
2. Calls the entity-specific placeholder factory (`TaskPlaceholders`, `FeaturePlaceholders`, or `EpicPlaceholders`)
3. Passes the placeholder map to `PopulateTemplate(vars)` which replaces all `{key}` patterns
4. Returns the populated instruction in the JSON response

### Task-Level Actions (Default)

Task actions are configured in the top-level `status_metadata`. This is the original and most common configuration:

```json
{
  "status_metadata": {
    "ready_for_development": {
      "color": "blue",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation"],
        "instruction_template": "Implement task {id}. Title: {title}. See spec at {file_path}."
      }
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "tech-lead",
        "skills": ["quality", "code-review"],
        "instruction_template": "Review code for task {id}: {title}."
      }
    },
    "blocked": {
      "color": "red",
      "phase": "any",
      "orchestrator_action": {
        "action": "pause",
        "instruction_template": "Task {id} is blocked: {blocked_reason}. Wait for resolution."
      }
    }
  }
}
```

### Epic-Level Actions

Epic actions are configured under `epic_workflow.status_metadata`:

```json
{
  "epic_workflow": {
    "status_metadata": {
      "ready_for_research": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research market and feasibility for epic {id}: {title}. Review the epic document at {file_path}. Report findings."
        }
      },
      "ready_for_planning": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing"],
          "instruction_template": "Create feature breakdown for epic {id}: {title}. Priority: {priority}."
        }
      },
      "completed": {
        "orchestrator_action": {
          "action": "archive",
          "instruction_template": "Epic {id} ({title}) is completed. No further action needed."
        }
      }
    }
  }
}
```

### Feature-Level Actions

Feature actions are configured under `feature_workflow.status_metadata`:

```json
{
  "feature_workflow": {
    "status_metadata": {
      "ready_for_refinement_tech": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "architect",
          "skills": ["architecture"],
          "instruction_template": "Design technical architecture for feature {id}: {title}. Reference {file_path}."
        }
      },
      "ready_for_task_generation": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing"],
          "instruction_template": "Generate implementation tasks for feature {id}: {title}."
        }
      }
    }
  }
}
```

## Template Writing Guide

### Effective Templates

Good templates give agents enough context to work independently:

```
"Implement task {id} - {title}. Reference the task specification at {file_path}. Follow TDD practices."
```

This provides:
- **Identity**: `{id}` tells the agent which task to work on
- **Context**: `{title}` gives a human-readable summary
- **Location**: `{file_path}` points to detailed specs
- **Instructions**: Static text guides the agent's approach

### Template Patterns by Action Type

**spawn_agent templates** should include the task identity, what to do, and where to find specs:

```json
{
  "action": "spawn_agent",
  "agent_type": "developer",
  "skills": ["implementation"],
  "instruction_template": "Implement task {id}: {title}. Read spec at {file_path}. Write tests first."
}
```

**pause templates** should explain what's blocking and what to wait for:

```json
{
  "action": "pause",
  "instruction_template": "Task {id} is blocked: {blocked_reason}. Do not spawn agents. Wait for resolution."
}
```

**wait_for_triage templates** should explain what needs human attention:

```json
{
  "action": "wait_for_triage",
  "instruction_template": "Task {id} ({title}) needs triage. Priority: {priority}. Assign to appropriate team."
}
```

**archive templates** can be minimal:

```json
{
  "action": "archive",
  "instruction_template": "Task {id} is completed. No action needed."
}
```

### Backward Compatibility

Templates using the original `{task_id}` placeholder continue to work. The task placeholder factory maps `{task_id}` to the task key, same as `{id}`.

**Old template (still works):**
```
"instruction_template": "Implement task {task_id}. Follow TDD."
```

**New template (richer context):**
```
"instruction_template": "Implement task {id}: {title}. Spec at {file_path}. Follow TDD."
```

Both produce valid output. There is no need to migrate existing templates unless you want richer context.

## Validation

### CLI Validation

Use `shark workflow validate-actions` to check all orchestrator actions:

```bash
# Validate all actions across all levels
shark workflow validate-actions

# JSON output
shark workflow validate-actions --json
```

The validator checks:
- Action type is valid (`spawn_agent`, `pause`, `wait_for_triage`, `archive`)
- `instruction_template` is present and non-empty
- `agent_type` is set for `spawn_agent` actions
- `skills` array is non-empty for `spawn_agent` actions
- Template syntax is well-formed (no unclosed braces)
- Template is under 2000 characters

### Viewing Actions

Use `shark workflow show-actions` to view configured actions:

```bash
# Show all configured actions
shark workflow show-actions

# JSON output with full detail
shark workflow show-actions --json

# Filter by level
shark workflow show-actions --level=task
shark workflow show-actions --level=epic
shark workflow show-actions --level=feature
```

### Template Syntax Warnings

The validator produces warnings (not errors) for:
- Templates with no placeholders at all
- Malformed placeholders (unclosed braces)
- Templates exceeding 2000 characters

Custom placeholders are valid -- since the placeholder map is dynamic, any `{word}` pattern is potentially valid depending on the entity type and its populated fields.

## How Templates Are Resolved at Runtime

### Task Resolution Path

```
CLI command (e.g., shark task next-status E07-F01-001)
  -> task_repository.getOrchestratorAction()
    -> config.TaskPlaceholders(task)        // builds map from task fields
    -> action.PopulateTemplate(vars)         // replaces {key} -> value
    -> returns populated instruction string
```

### Epic Resolution Path

```
CLI command (e.g., shark epic update E07 --status=active)
  -> epic_service.resolveAction(epic, status)
    -> config.EpicPlaceholders(epic)         // builds map from epic fields
    -> action.PopulateTemplate(vars)          // replaces {key} -> value
    -> returns populated instruction string
```

### Feature Resolution Path

```
CLI command (e.g., shark feature update E07-F01 --status=active)
  -> feature_service.resolveAction(feature, status)
    -> config.FeaturePlaceholders(feature)   // builds map from feature fields
    -> action.PopulateTemplate(vars)          // replaces {key} -> value
    -> returns populated instruction string
```

## Complete Configuration Example

Here is a complete `.sharkconfig.json` excerpt showing all three entity levels with orchestrator actions:

```json
{
  "status_metadata": {
    "draft": {
      "color": "gray",
      "phase": "planning",
      "orchestrator_action": {
        "action": "wait_for_triage",
        "instruction_template": "Task {id} is in draft. Advance with next-status to begin planning."
      }
    },
    "ready_for_refinement_ba": {
      "color": "cyan",
      "phase": "planning",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "business-analyst",
        "skills": ["specification-writing"],
        "instruction_template": "Refine requirements for task {id}: {title}. Read spec at {file_path}."
      }
    },
    "ready_for_development": {
      "color": "blue",
      "phase": "development",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": ["test-driven-development", "implementation"],
        "instruction_template": "Implement task {id}: {title}. Spec at {file_path}. Agent: {agent_type}. Priority: {priority}."
      }
    },
    "ready_for_code_review": {
      "color": "magenta",
      "phase": "review",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "tech-lead",
        "skills": ["quality"],
        "instruction_template": "Review code for task {id}: {title}. Check quality and test coverage."
      }
    },
    "ready_for_qa": {
      "color": "green",
      "phase": "qa",
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "qa",
        "skills": ["quality"],
        "instruction_template": "Test task {id}: {title}. Validate acceptance criteria from {file_path}."
      }
    },
    "blocked": {
      "color": "red",
      "phase": "any",
      "orchestrator_action": {
        "action": "pause",
        "instruction_template": "Task {id} is blocked: {blocked_reason}. Wait for resolution."
      }
    },
    "completed": {
      "color": "white",
      "phase": "done",
      "orchestrator_action": {
        "action": "archive",
        "instruction_template": "Task {id} completed. No action needed."
      }
    }
  },

  "epic_workflow": {
    "status_metadata": {
      "ready_for_research": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "researcher",
          "skills": ["discovery", "research"],
          "instruction_template": "Research epic {id}: {title}. Priority: {priority}. See {file_path}."
        }
      },
      "ready_for_planning": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing"],
          "instruction_template": "Plan features for epic {id}: {title}."
        }
      }
    }
  },

  "feature_workflow": {
    "status_metadata": {
      "ready_for_refinement_tech": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "architect",
          "skills": ["architecture"],
          "instruction_template": "Design architecture for feature {id}: {title}. Spec: {file_path}."
        }
      },
      "ready_for_task_generation": {
        "orchestrator_action": {
          "action": "spawn_agent",
          "agent_type": "product-manager",
          "skills": ["specification-writing"],
          "instruction_template": "Generate tasks for feature {id}: {title}."
        }
      }
    }
  }
}
```

## Related Documentation

- [Orchestrator Actions API](../cli-reference/orchestrator-actions.md) - JSON response format and integration guide
- [Workflow Configuration](../cli-reference/workflow-config.md) - Status flow and metadata configuration
- [Workflow Profiles](workflow-profiles.md) - Predefined workflow profiles (basic, advanced)
- [Best Practices](../cli-reference/best-practices.md) - AI agent orchestration best practices
