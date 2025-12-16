---
agent: implementation-orchestrator
description: Launch the API development agent to resolve api / backend code issues found during code-review 
---
review the outstanding issues in {{arguments}} 

confirm with user if all issues should be resolved or only high priority items. After confirmation of scope, resolve all agreed upon issues. When possible, launch parallel agents to resolve the issues more quickly.

Update the status of the issues in the code-review-frontend and code-review-backend.md documents. 

# Completion checklist
[ ] - Agree upon scope
[ ] - Completed each issue in scope (if not launch a new agent task to resolve and check again)
[ ] - updated code-review*.md files to indicate status.
[ ] - any items deemed too big for a quick fix are put in a TODO.md file and noted in code-review*.md