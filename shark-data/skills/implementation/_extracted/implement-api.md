---
# Extracted scaffolding from workflows/implement-api.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field api_spec_path
shark context get $TASK_KEY --field test_plan_path
shark related-docs list --task=$TASK_KEY --filter="api|contract|endpoint"

# gate
API contract validated and approved.
Test plan covers all endpoints and error cases.
Implementation matches specification exactly.

# mutate
shark context set $TASK_KEY --field implementation_status --value "api_complete"
shark note add $TASK_KEY --type implementation --content "API implementation: [endpoints, routes, handlers complete]"

# advance
shark status advance $TASK_KEY
