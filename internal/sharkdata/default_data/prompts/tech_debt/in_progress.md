{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

RESUME tech debt resolution for {{.id}}: "{{.title}}".

Category: {{.category}} | Severity: {{.severity}}

Check for existing resolution: review recent git changes. If resolution exists and the project quality gate passes (see `docs/architecture/tech-stack.md` **Quality Gate** section, or infer from the repo), advance immediately.

Otherwise, continue with resolution per triaged instructions.
