---
# Extracted scaffolding from workflows/design-database.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field data_model_requirements
shark related-docs list --feature=$FEATURE_KEY --filter="schema|database|migration"
shark context get $FEATURE_KEY --field performance_requirements

# gate
Data model aligns with feature requirements.
Normalization and indexing decisions documented.
Migration strategy defined for existing data.

# mutate
shark context set $FEATURE_KEY --field database_design_status --value "reviewed"
shark note add $FEATURE_KEY --type design --content "Database schema: [tables, indexes, constraints defined]"

# advance
shark status advance $FEATURE_KEY
