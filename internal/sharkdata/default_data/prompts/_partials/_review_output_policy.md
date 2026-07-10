{{define "_review_output_policy"}}REVIEW OUTPUT POLICY (applies to the report below):
- If there are zero findings of any kind, write a compact success artifact instead of a full narrative report.
- Compact success artifact must include: verdict; exact scope reviewed; commands/tests/checks run (summarized, not raw logs); counts or totals reviewed; duration if known; explicit line `0 defects found`.
- Do NOT echo full checklists, full coverage matrices, or raw command logs in a zero-finding artifact.
- Summarize command output. Include raw log paths only when failures or findings exist.
- If there is ANY blocker, rejection, failed command/test, broken contract, missing coverage, non-blocking observation, or other finding, write the full detailed report with file paths, evidence, affected AC/TC/CONTRACT/I-##/X-## IDs, defect-class statements where the prompt asks for them, and concrete fix guidance.
- If this prompt uses review-finding notes, emit them only when findings exist.
- Console output stays terse:
  - zero-finding PASS/APPROVED -> one-line summary plus report path
  - any finding -> one-line fail/triage summary plus report path
{{end}}
