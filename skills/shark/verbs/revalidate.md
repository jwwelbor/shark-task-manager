# /shark revalidate — Spec ↔ tasks ↔ status readiness audit

Usage: `/shark revalidate <key>`

Default behavior is a **self-contained inline audit** with zero dependency on any
external skill. Optional deep validators enrich it only when present in the bundle.

## 1. Inline audit (default — always runs)

Using only `shark`:
```bash
shark get <key> --json                 # spec/status/metadata
shark list <epic> <feature> --json     # child tasks (for features/epics)
shark get <key> --json --field file_path
```
Check:
- **AC coverage** — every acceptance criterion in the spec maps to at least one
  task (for features) or is addressed (for tasks).
- **Task sequencing** — execution order / dependencies are sane; no orphan tasks.
- **Status sanity** — the entity's status is consistent with its children
  (e.g. a feature isn't past `active` while tasks are still `draft`).

Emit a verdict: **READY** / **WARNINGS** / **NOT READY**, with the specific gaps.

## 2. Optional enrichment (gated by preflight)

**Before** invoking a deep validator, confirm the content can be retrieved:
- `shark skill get quality workflows/validate-design.md`
- `shark skill get quality workflows/validate-tasks.md`

For each command that succeeds, follow the returned validator instructions and
fold its findings into the verdict. If neither succeeds, run the inline audit
only and append:
> Deep validators unavailable in this bundle — inline audit only.

A missing validator is **never** a hard failure.

## Notes
- The inline audit must always produce a verdict, even with no bundle present.
