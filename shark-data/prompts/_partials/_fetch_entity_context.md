{{define "_fetch_entity_context"}}# Fetch entity context

Extract opaque IDs and resolved paths for the entity:

```bash
ENT_JSON=$(shark get {{.id}} --json)
TASK_ID=$(echo "$ENT_JSON" | jq -r '.task_id // empty')
FEATURE_ID=$(echo "$ENT_JSON" | jq -r '.feature_id // empty')
EPIC_ID=$(echo "$ENT_JSON" | jq -r '.epic_id // empty')
ENT_PATH=$(echo "$ENT_JSON" | jq -r '.path')
ENT_FILE=$(echo "$ENT_JSON" | jq -r '.filename')
ENT_SPEC_PATH="$ENT_PATH/$ENT_FILE"
ACCEPTANCE_CRITERIA=$(echo "$ENT_JSON" | jq -r '.acceptance_criteria // []')

# Parent context (when applicable — empty for top-level entities)
[ -n "$FEATURE_ID" ] && {
  FEATURE_JSON=$(shark get $FEATURE_ID --json)
  FEATURE_PATH=$(echo "$FEATURE_JSON" | jq -r '.path')
  FEATURE_FILE=$(echo "$FEATURE_JSON" | jq -r '.filename')
  FEATURE_PRD_PATH="$FEATURE_PATH/$FEATURE_FILE"
}
[ -n "$EPIC_ID" ] && {
  EPIC_JSON=$(shark get $EPIC_ID --json)
  EPIC_PATH=$(echo "$EPIC_JSON" | jq -r '.path')
  EPIC_FILE=$(echo "$EPIC_JSON" | jq -r '.filename')
  EPIC_PRD_PATH="$EPIC_PATH/$EPIC_FILE"
}
```

Use `$ENT_SPEC_PATH`, `$FEATURE_PRD_PATH`, `$EPIC_PRD_PATH` downstream.

**Important**: do NOT hardcode `prd.md`, `feature.md`, or any filename — shark determines them per entity.
{{end}}
