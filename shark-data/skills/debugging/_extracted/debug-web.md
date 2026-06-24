---
# Extracted scaffolding from workflows/debug-web.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field http_logs
shark context get $TASK_KEY --field network_error_details
shark related-docs list --task=$TASK_KEY --filter="api|network|http"

# gate
Network issue isolated from application logic.
HTTP error codes and responses documented.
Request/response payloads captured for analysis.

# mutate
shark context set $TASK_KEY --field web_debug_status --value "in_progress"
shark note add $TASK_KEY --type blocker --content "Web/API issue: [endpoint, status code, root cause identified]"

# advance
shark status advance $TASK_KEY
