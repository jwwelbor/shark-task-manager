# shark-data — canonical Shark 2.0 layout

This directory ships with the shark binary via `//go:embed`. It is the
canonical default that `shark init` lays down at a project root.

The real content (workflows, prompts, skills, agents) lands in F4 of E02 —
this commit ships the embedding machinery only. The directory is therefore
mostly empty placeholders; future `shark upgrade` runs after F4 will
populate it with real defaults.

## Layout (target)

```
shark-data/
  prompts/                 # status prompts (.md, replaces shark-templates/*.tmpl)
  skills/                  # decoupled craft skills (output of F1)
  agents/                  # in-scope agent definitions
  workflow/                # per-entity workflow YAML
  overrides/               # local-only — shark upgrade never touches this
  README.md                # this file
```

## Override semantics

A file at `shark-data/overrides/<path>` **fully replaces** the default at
`shark-data/<path>` — never merges. See E02 follow-up idea I-2026-05-10-01
on override drift mitigation.
