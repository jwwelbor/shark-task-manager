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
3. After the workflow completes, suggest one Shark-native next step:
   - If product docs or a clear product vision are still missing, suggest
     `/shark product-design` for the full D01-D14 flow or
     `/shark vision "idea"` for a quick captured initiative.
   - If product direction already exists, suggest `/shark run <key>` when there
     is a concrete epic, feature, or task ready to drive. If the next initiative
     is not tracked yet, suggest `/shark vision "next idea"`.

## Notes

- The workflow detects brownfield vs greenfield and generates `docs/architecture/*`.
- Pass any extra arguments (target path, scope) straight through to the workflow.
