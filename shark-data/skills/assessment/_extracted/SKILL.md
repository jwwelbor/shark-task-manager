---
# Extracted scaffolding from SKILL.md
# Documents the 4 mode discriminators and state mutations
---

# fetch
shark context get $TASK_KEY --field complexity_level
shark context get $TASK_KEY --field scope_boundaries
shark context get $TASK_KEY --field readiness_status

# gate
Mode discriminators clear: complexity_triage | scope_validation | readiness_check | effort_estimation
Current state documents which mode is in progress.

# mutate
shark context set $TASK_KEY --field assessment_mode --value "[selected mode]"
shark context set $TASK_KEY --field assessment_status --value "in_progress"
shark note add $TASK_KEY --type assessment --content "Assessment: [mode and findings]"

# advance
shark status advance $TASK_KEY
