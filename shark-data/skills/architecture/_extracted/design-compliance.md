---
# Extracted scaffolding from workflows/design-compliance.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field regulatory_requirements
shark related-docs list --feature=$FEATURE_KEY --filter="compliance|security|privacy"
shark context get $FEATURE_KEY --field legal_review_status

# gate
All regulatory requirements identified before design begins.
Compliance documentation must map to technical controls.

# mutate
shark context set $FEATURE_KEY --field compliance_review_status --value "in_progress"
shark note add $FEATURE_KEY --type decision --content "Compliance architecture: [controls identified]"

# advance
shark status advance $FEATURE_KEY
