---
# Extracted scaffolding from workflows/generate-standards.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $TASK_KEY --field coding_standard_template_path
shark related-docs list --task=$TASK_KEY --filter="standard|guideline|convention"

# gate
Coding standards document complete and approved.
Standards align with project architecture.
Standards are actionable and measurable.

# mutate
shark context set $TASK_KEY --field standards_generation_status --value "complete"
shark note add $TASK_KEY --type quality --content "Coding standards: [generated and documented]"

# advance
shark status advance $TASK_KEY
