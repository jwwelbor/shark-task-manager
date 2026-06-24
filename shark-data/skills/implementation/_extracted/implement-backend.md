---
# Extracted scaffolding from workflows/implement-backend.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field architecture_path
shark context get $TASK_KEY --field test_plan_path
shark related-docs list --task=$TASK_KEY --filter="service|business logic|pattern"

# gate
Service interfaces defined and contracts validated.
Business logic tested with comprehensive test suite.
Error handling implemented for all edge cases.

# mutate
shark context set $TASK_KEY --field backend_implementation_status --value "complete"
shark note add $TASK_KEY --type implementation --content "Backend implementation: [services, repositories, business logic complete]"

# advance
shark status advance $TASK_KEY
