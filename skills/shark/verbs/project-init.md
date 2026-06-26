# /shark project-init — Bootstrap architecture docs

Delegates to the **content bundle's** project-init workflow. No external
`~/.claude` skill is used.

## Procedure

1. Read and follow:
   ```
   shark skill get research workflows/project-init.md
   ```
2. If that command fails because the workflow content is unavailable, print:
   > `project-init` content is not available in this project's bundle
   > (`shark skill get research workflows/project-init.md` failed). Check the
   > installed shark version or `shark_data_path`.

   Then stop — do not fall back to a hardcoded procedure.

## Notes

- The workflow detects brownfield vs greenfield and generates `docs/architecture/*`.
- Pass any extra arguments (target path, scope) straight through to the workflow.
