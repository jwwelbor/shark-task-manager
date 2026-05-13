---
# Extracted scaffolding from workflows/design-system.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $EPIC_KEY --field system_scope
shark related-docs list --epic=$EPIC_KEY --filter="system|architecture|requirements"
shark context get $EPIC_KEY --field stakeholder_requirements

# gate
All stakeholder requirements documented.
System boundaries and external interfaces identified.
High-level deployment topology defined.

# mutate
shark context set $EPIC_KEY --field system_design_status --value "in_progress"
shark note add $EPIC_KEY --type design --content "System architecture: [high-level components and interactions defined]"

# advance
shark status advance $EPIC_KEY
