{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Implement change {{.id}}: "{{.title}}".

Check for existing implementation. If changes exist and quality gate passes, advance immediately.

---

READ:
(1) Change card description at {{.file_path}}
(2) CLAUDE.md for coding standards

IMPLEMENT the change with appropriate testing. Run the project's quality gate before advancing. Determine the commands from `docs/architecture/tech-stack.md` (the **Quality Gate** section), or `docs/architecture/coding-standards.md` if present. If neither exists, infer from the repo: a `Makefile` → its documented format/lint/test targets; `go.mod` → `gofmt`/`go vet ./...`/`go test ./...`; `package.json` → its format/lint/test scripts; `pyproject.toml` → the configured formatter/linter and `pytest`. Fix ALL failures before advancing. No exceptions.

EXIT GATE:
- Change implemented per description
- Tests pass
- Quality gate passes
