---
# Extracted scaffolding from workflows/implement-frontend.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field design_path
shark context get $TASK_KEY --field test_plan_path
shark related-docs list --task=$TASK_KEY --filter="component|ui|state"

# gate
Components match design specifications.
UI renders correctly on all target browsers.
State management follows architectural patterns.

# mutate
shark context set $TASK_KEY --field frontend_implementation_status --value "complete"
shark note add $TASK_KEY --type implementation --content "Frontend implementation: [components, pages, styling complete]"

# advance
shark status advance $TASK_KEY
