---
# Extracted scaffolding from workflows/infra-requirements.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field infrastructure_requirements
shark related-docs list --feature=$FEATURE_KEY --filter="infrastructure|deployment|scaling"
shark context get $FEATURE_KEY --field performance_sla

# gate
Infrastructure requirements tied to feature requirements.
Scaling strategy defined for projected load.
Deployment topology matches architecture design.

# mutate
shark context set $FEATURE_KEY --field infra_requirements_status --value "finalized"
shark note add $FEATURE_KEY --type design --content "Infrastructure: [compute, storage, networking requirements defined]"

# advance
shark status advance $FEATURE_KEY
