---
# Extracted scaffolding from workflows/design-security.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field security_requirements
shark related-docs list --feature=$FEATURE_KEY --filter="security|threat-model|authentication"
shark context get $FEATURE_KEY --field data_sensitivity_level

# gate
Threat model complete and reviewed.
Security controls mapped to requirements.
Authentication and authorization architecture defined.

# mutate
shark context set $FEATURE_KEY --field security_design_status --value "reviewed"
shark note add $FEATURE_KEY --type design --content "Security architecture: [threat model, controls defined]"

# advance
shark status advance $FEATURE_KEY
