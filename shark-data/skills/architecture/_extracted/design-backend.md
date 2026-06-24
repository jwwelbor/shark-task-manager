---
# Extracted scaffolding from workflows/design-backend.md
# Tagged blocks document shark-specific plumbing (state mutations, validations, context operations)
---

# fetch
shark context get $FEATURE_KEY --field existing_backend_architecture
shark related-docs list --feature=$FEATURE_KEY --filter="backend|api|service"
shark get $FEATURE_KEY --field acceptance_criteria

# gate
Design must align with system architecture decisions.
Proposed services must not duplicate existing implementations.
API contracts must be defined before implementation begins.

# mutate
shark context set $FEATURE_KEY --field backend_architecture_status --value "in_design"
shark note add $FEATURE_KEY --type decision --content "Backend architecture decision: [selected pattern]"
shark note add $FEATURE_KEY --type design --content "Service layer contract defined"

# advance
shark status advance $FEATURE_KEY
