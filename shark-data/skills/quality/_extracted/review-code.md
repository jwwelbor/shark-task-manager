---
# Extracted scaffolding from workflows/review-code.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field implementation_paths
shark context get $TASK_KEY --field architecture_path
shark related-docs list --task=$TASK_KEY --filter="review|rubric|standard"

# gate
Code follows project standards.
Architecture decisions respected.
No critical issues found in review.

# mutate
shark context set $TASK_KEY --field code_review_status --value "approved"
shark note add $TASK_KEY --type quality --content "Code review: [approved, meets standards]"

# advance
shark status advance $TASK_KEY
