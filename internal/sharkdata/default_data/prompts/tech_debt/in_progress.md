{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME tech debt resolution for {{.id}}: "{{.title}}".

Category: {{.category}} | Severity: {{.severity}}

Check for existing resolution: review recent git changes. If resolution exists and quality gate passes (make fmt && make lint && make test), advance immediately.

Otherwise, continue with resolution per triaged instructions.
