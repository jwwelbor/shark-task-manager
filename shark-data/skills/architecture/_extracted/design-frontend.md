---
# Extracted scaffolding from workflows/design-frontend.md
# Tagged blocks document shark-specific plumbing
---

# fetch
shark context get $FEATURE_KEY --field ui_requirements
shark related-docs list --feature=$FEATURE_KEY --filter="wireframe|design|ux"
shark context get $FEATURE_KEY --field accessibility_requirements

# gate
UI design aligns with system design system.
Component architecture defined and approved.
Accessibility requirements met (WCAG compliance).

# mutate
shark context set $FEATURE_KEY --field frontend_design_status --value "approved"
shark note add $FEATURE_KEY --type design --content "Frontend component architecture: [components, state management defined]"

# advance
shark status advance $FEATURE_KEY
