# shark-data — canonical Shark 2.0 layout

This directory ships with the shark binary via `//go:embed`. It is the
canonical default that `shark init` lays down at a project root.

The embedded bundle contains the default workflows, prompts, markdown file templates, skills, and agents
that Shark can serve directly from the binary or
materialize to disk for local customization.

## Layout (target)

```
shark-data/
  prompts/                 # status prompts (.md)
    <entity>/              # entity-owned prompts (bug, change, epic, feature, sprint, task, tech_debt)
    _shared/               # dispatchable prompts reused across entities (see manifest)
    _partials/             # non-standalone {{template}} fragments
  skills/                  # decoupled craft skills (output of F1)
  agents/                  # in-scope agent definitions
  workflow/                # per-entity workflow YAML
  file_templates/          # markdown skeletons for created entity files
  manifest.yaml            # declarative structure for the bundle validators
  overrides/               # local-only — shark upgrade never touches this
  README.md                # this file
```

## Manifest

`manifest.yaml` is the declarative source of truth the bundle validators
(`shark admin validate-data`) consult to tell intentional structure from drift:
the prompt namespaces, the cross-entity `_shared/` prompt allowlist, and each
skill's normalized identity slug + ownership. It declares no runtime behavior.
Update it when adding an entity, a shared prompt, or a skill.

## Override semantics

A file at `shark-data/overrides/<path>` **fully replaces** the default at
`shark-data/<path>` — never merges. See E02 follow-up idea I-2026-05-10-01
on override drift mitigation.

Created-file skeleton customizations use the same rule. For example,
`shark-data/overrides/file_templates/task.md` replaces the default
`shark-data/file_templates/task.md` and is preserved by `shark admin upgrade`.
