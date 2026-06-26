Epic {{.id}} ("{{.title}}") is ACTIVE — features are in progress.

## Steps

1. List all features: {{template "list_json" .}}
2. For each feature (in execution order):
   - **completed / cancelled** → skip
   - **agent-owned or active step** → run `/run {feature-key}` to drive or resume it
   - **blocked** → skip, try next
   - **draft** → run `/run {feature-key}` to start it
3. When ALL features are completed or cancelled: shark status advance {{.id}}
4. If ALL remaining features are blocked: report to user and STOP
