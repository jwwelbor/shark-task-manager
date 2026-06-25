# /shark amend — Apply a spec change and rewind to the right phase

Usage:
```
/shark amend <key> "what changed"
/shark amend <key> "what changed" --rewind-to=<status>   # skip assessment, force target
```

Works for any entity. Edits the entity's spec, records a requirement note, and
moves the entity back to the earliest phase that must re-verify the change — all
**driven by the active workflow YAML, with no hardcoded status names**.

## Procedure

### 1. Discover the spec file
```bash
shark get <key> --json --field file_path
```
If empty, fall back to the conventional `docs/plan/...` location for the entity.

### 2. Assess the change
Classify the change as one of:
- **coverage** — adds/changes acceptance criteria or tests but not the design
  (re-verify from test planning / decomposition onward).
- **architectural** — changes requirements, design, or research assumptions
  (re-verify from specification/research onward).

`--rewind-to=<status>` skips this assessment and uses the given target directly.

### 3. Edit the spec
Add a dated amendment entry to the spec file describing the changed acceptance
criterion / requirement.

### 4. Record the change
```bash
shark create note <key> "Amended: <summary>" --type=requirement
```
(`requirement` is a valid note type — see `context/notes-context-docs.md`.)

### 5. Resolve the rewind target from the workflow YAML
Build the `status → phase` and `phase → order` maps from the entity's active
workflow file (include each step's `aliases:`). Define intent sets by **phase name**:

- architectural-intent phases: `specification, design, research, investigation, refinement`
- coverage-intent phases: `test_planning, decomposition`

Pick the target step:
- **architectural** → the earliest step whose phase is in the architectural set;
  **fallback** if none exist → the workflow's `start:` / planning step.
- **coverage** → the earliest step whose phase is in the coverage set;
  **fallback** if none → the architectural target → the start step.

Per-entity reality (active feature workflow has both; epic has design only;
task/bug/change have neither):
- If the entity's workflow has **no** matching phase, fall back to `draft`
  (`start:`), and tell the user that re-verification will happen through that
  step's own gate rather than a dedicated spec/test phase.

### 6. Move the entity
Write a short delta into entity context, then:
```bash
shark status set <key> <target> --force --reason="amendment: <summary>"
```

## Notes
- Never hardcode `ready_for_*` / `in_*` — every status comes from the YAML maps.
- On a cloud DB this only changes status/notes/context; it does not reassign keys.
