---
# Extracted scaffolding from workflows/implement-database.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field schema_path
shark context get $TASK_KEY --field data_model_path
shark related-docs list --task=$TASK_KEY --filter="migration|schema|index"

# gate
Schema matches data model specifications.
Indexes and constraints optimized for queries.
Migration tested against existing data.

# mutate
shark context set $TASK_KEY --field database_implementation_status --value "deployed"
shark note add $TASK_KEY --type implementation --content "Database: [schema, indexes, migrations complete]"

# advance
shark status advance $TASK_KEY
