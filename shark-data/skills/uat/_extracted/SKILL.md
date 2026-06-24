---
# Extracted scaffolding from SKILL.md
# Documents phase orchestration and state mutations
---

# fetch
shark context get $TASK_KEY --field phase_current
shark context get $TASK_KEY --field acceptance_criteria
shark context get $TASK_KEY --field user_stories

# gate
Current phase clear: Phase 1 (planning) | Phase 4 (UAT execution)
Acceptance criteria and user stories documented.

# mutate
shark context set $TASK_KEY --field uat_phase --value "[current phase]"
shark context set $TASK_KEY --field uat_status --value "in_progress"
shark note add $TASK_KEY --type uat --content "UAT: [phase findings]"

# advance
shark status advance $TASK_KEY
