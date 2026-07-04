{{define "_resume_preamble"}}{{if eq .is_resume "true"}}RESUME CONTEXT: This entity is already in an active workflow status.

Before starting, check `shark claims`, recent notes, and existing artifacts for work already underway. If partial work exists, continue it instead of restarting; if nothing useful exists, follow the instructions below from the beginning.

Do not advance status just because code or docs exist; review, QA, and approval phases still require their own explicit evidence.

{{end}}{{end}}
