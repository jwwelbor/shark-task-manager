---
# Extracted scaffolding from SKILL.md
# Documents TDD cycle orchestration if applicable
---

# fetch
shark context get $TASK_KEY --field test_approach
shark context get $TASK_KEY --field implementation_status
shark related-docs list --task=$TASK_KEY --filter="test|tdd|cycle"

# gate
TDD cycle defined if applicable: RED-GREEN-REFACTOR
Test assertions are business-focused, not implementation-focused.

# mutate
shark context set $TASK_KEY --field tdd_status --value "in_progress"
shark context set $TASK_KEY --field current_cycle --value "[RED|GREEN|REFACTOR]"
shark note add $TASK_KEY --type implementation --content "TDD: [cycle progress]"

# advance
shark status advance $TASK_KEY
