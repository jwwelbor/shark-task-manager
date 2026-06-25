# /shark project-init — Bootstrap architecture docs

Delegates to the **content bundle's** project-init workflow. No external
`~/.claude` skill is used.

## Procedure

1. Resolve the content bundle root (see SKILL.md → *Content bundle resolution*):
   project root → `.sharkconfig.json.shark_data_path` → default `<root>/shark-data`.
2. Read and follow:
   ```
   <bundle>/skills/research/workflows/project-init.md
   ```
3. If that file does not exist in the resolved bundle, print:
   > `project-init` content is not available in this project's bundle
   > (`<bundle>/skills/research/workflows/project-init.md`). Run `shark init` /
   > `shark upgrade` to materialize the bundle, or check `shark_data_path`.

   Then stop — do not fall back to a hardcoded procedure.

## Notes

- The workflow detects brownfield vs greenfield and generates `docs/architecture/*`.
- Pass any extra arguments (target path, scope) straight through to the workflow.
