---
# Extracted scaffolding from workflows/debug-devops.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field deployment_logs_path
shark context get $TASK_KEY --field infrastructure_status
shark related-docs list --task=$TASK_KEY --filter="deployment|infrastructure|devops"

# gate
Infrastructure issue isolated from application code.
Deployment logs reviewed and root cause identified.
Reproducible in staging environment.

# mutate
shark context set $TASK_KEY --field devops_debug_status --value "in_progress"
shark note add $TASK_KEY --type blocker --content "Infrastructure issue: [root cause identified]"

# advance
shark status advance $TASK_KEY
