{{template "_resume_preamble" .}}
{{template "advance_preamble" .}}

Research codebase for epic {{.id}}: "{{.title}}".

Check for existing research report in epic directory. If report exists with all sections populated and file paths cited, advance immediately.

---

BROWNFIELD-FIRST RESEARCH — identify what already exists before proposing anything new.

Load skills from `shark-data/skills/research/`:
  - `workflows/brownfield-analysis.md`
  - `workflows/analyze-codebase.md`

READ:
(1) Epic PRD for scope and goals
(2) CLAUDE.md for project architecture, patterns, and conventions
(3) Existing codebase: grep for related functionality, services, models, tests

PRODUCE research report:
(1) Existing implementations relevant to this epic (with file paths and line numbers)
(2) Patterns and conventions that must be followed
(3) Integration points (services, repositories, CLI commands, database tables)
(4) What can be EXTENDED vs what needs NEW code
(5) Technical risks and feasibility assessment
(6) Recommended implementation approach (extend-first, minimize new code)

CRITICAL: The #1 goal is to AVOID duplicating existing functionality. Every recommendation must explain why existing code cannot be extended.

EXIT GATE:
- All existing related code identified with file paths
- Extension-vs-new analysis for every component
- Feasibility confirmed or risks flagged
- Actionable for architect
