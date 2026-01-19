# File Path Organization

All entity creation commands support custom file paths via the `--file` flag.

## Default File Path Behavior

### Epics
- Default: `docs/plan/{epic-key}-{slug}/epic.md`
- Example: `docs/plan/E07-user-management-system/epic.md`

### Features
- Default: `docs/plan/{epic-key}-{epic-slug}/{feature-key}-{feature-slug}/feature.md`
- Example: `docs/plan/E07-user-management-system/E07-F01-authentication/feature.md`

### Tasks
- Default: `docs/plan/{epic-key}-{epic-slug}/{feature-key}-{feature-slug}/tasks/{task-key}.md`
- Example: `docs/plan/E07-user-management-system/E07-F01-authentication/tasks/T-E07-F01-001.md`

## Custom File Path Examples

```bash
# Epic with custom path
shark epic create "Q1 Roadmap" --file="docs/roadmap/2025-q1/epic.md"

# Feature with custom path
shark feature create --epic=E01 "User Growth" --file="docs/roadmap/features/growth.md"

# Task with custom path
shark task create --epic=E07 --feature=F01 "Migrate auth" --file="docs/migration/auth.md"
```

## File Path Rules

1. **Must be relative to project root**
2. **Must include .md extension**
3. **Parent directories created automatically**
4. **Use `--force` to reassign existing files**

## Organization Strategies

### By Timeline
```
docs/
└── roadmap/
    ├── 2025-q1/
    │   ├── epic.md
    │   └── features/
    └── 2025-q2/
        ├── epic.md
        └── features/
```

### By Domain
```
docs/
└── plan/
    ├── backend/
    │   ├── api/
    │   └── database/
    └── frontend/
        ├── components/
        └── pages/
```

### Mixed Approach
```
docs/
├── plan/           # Default shark structure
│   └── E07-*/
└── migration/      # Custom organization
    └── legacy-auth/
```

## Related Documentation

- [Epic Commands](epic-commands.md) - Epic file paths
- [Feature Commands](feature-commands.md) - Feature file paths
- [Task Commands](task-commands.md) - Task file paths
- [Sync Commands](sync-commands.md) - Sync file system with database
