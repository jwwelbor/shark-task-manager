{{define "_register_doc"}}# Register a produced document with shark

After a workflow creates or updates a document, register it as a related-doc for the parent entity so subsequent workflows (and shark related-docs queries) can find it.

```bash
shark related-docs add "{{.doc_friendly_name}}" "${DOC_PATH}" \
  {{- if eq .doc_parent "epic" }}
  --epic=$EPIC_ID
  {{- else if eq .doc_parent "feature" }}
  --feature=$FEATURE_ID
  {{- else if eq .doc_parent "task" }}
  --task=$TASK_ID
  {{- end }}
```

Variables:
- `doc_friendly_name` — human-readable name (e.g., "Backend Architecture", "Feature Test Plan").
- `doc_parent` — `"epic"`, `"feature"`, or `"task"` — which entity the doc belongs to.
- `DOC_PATH` — bash variable in scope (path to the registered doc).

For multi-document registration (e.g., write-task creates several specs), call this partial in a loop with each `(doc_friendly_name, DOC_PATH)` pair.
{{end}}
