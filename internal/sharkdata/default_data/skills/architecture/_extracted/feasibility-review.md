---
# Extracted scaffolding from workflows/feasibility-review.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $EPIC_KEY --field proposed_architecture_path
shark related-docs list --epic=$EPIC_KEY --filter="feasibility|risk|constraint"
shark get $EPIC_KEY --field acceptance_criteria

# gate
Architecture documented with clear rationale.
Technical risks identified and quantified.
Resource and timeline feasibility validated.

# mutate
shark context set $EPIC_KEY --field feasibility_review_status --value "approved"
shark note add $EPIC_KEY --type decision --content "Feasibility approved: [architecture viable for proposed timeline]"

# advance
shark status advance $EPIC_KEY
