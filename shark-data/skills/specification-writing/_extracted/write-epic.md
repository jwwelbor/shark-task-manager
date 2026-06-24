---
# Extracted scaffolding from workflows/write-epic.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $EPIC_KEY --field specification_template_path
shark related-docs list --epic=$EPIC_KEY --filter="spec|documentation|requirements"

# gate
Specification complete and internally consistent.
Acceptance criteria clearly defined.
Dependencies documented.

# mutate
shark context set $EPIC_KEY --field specification_status --value "complete"
shark note add $EPIC_KEY --type requirement --content "Specification: [write-epic complete]"

# advance
shark status advance $EPIC_KEY
