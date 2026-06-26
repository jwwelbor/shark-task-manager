{{define "_resume_preamble"}}{{if eq .is_resume "true"}}RESUME CONTEXT: This entity is already in an active workflow status. A previous agent session may have been interrupted.

BEFORE STARTING WORK:
1. Check for existing work artifacts (docs, code, reports) that a previous agent may have produced
2. If work is PARTIAL → continue from where it left off, do not redo completed sections
3. If NO work found → proceed with full instructions below

IMPORTANT: Do NOT skip to advancing status unless the specific instructions below explicitly say you can (e.g., a report with a PASS verdict exists). The presence of code or implementation files alone is NOT sufficient — review/QA/approval phases require their own reports.

{{end}}{{end}}
