# /shark vision — Idea → epic → workflow

Turn a one-line idea into a shark epic and kick off its workflow. Delegates spec
authoring to the **content bundle's** epic-writing workflow.

Usage: `/shark vision "one-line idea"`

## Procedure

1. Read and follow:
   ```
   shark skill get specification-writing workflows/write-epic.md
   ```
   passing the idea text as input. That workflow creates the epic in shark and
   authors its initial spec.
2. If the workflow command fails because the content is absent, degrade: create
   the epic directly so the idea is still captured:
   ```bash
   shark create epic "<idea>"            # cloud DB assigns the key; capture it from the response
   ```
   then note that the full epic-authoring workflow is unavailable in this bundle
   (`shark skill get specification-writing workflows/write-epic.md` failed) and
   stop.
3. After the epic exists, offer to drive it: `/shark run <epic-key>`.

## Notes

- On a cloud (Turso) database, **never** specify the epic key — shark assigns it;
  read the assigned key back from the create response.
