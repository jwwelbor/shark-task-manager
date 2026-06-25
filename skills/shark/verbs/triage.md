# /shark triage — Quick-capture & classify

Capture a discovered work item and route it to the correct shark entity (task,
feature, bug, tech-debt, change-card, idea, or note) under the right parent.
This is **capture-and-classify**, not create-and-elaborate.

Usage: `/shark triage "short description of the thing to track"`

## Procedure

1. Resolve the content bundle root (SKILL.md → *Content bundle resolution*).
2. If `<bundle>/skills/triage/SKILL.md` exists, read and follow it.
3. **If it does not exist** (the bundle triage skill is not yet shipped in this
   project), print a concise unavailable message and stop:
   > `/shark triage` is not yet available in this project's content bundle
   > (`<bundle>/skills/triage/SKILL.md`). For now, capture the item directly with
   > `shark create <type> …` or `shark create note <parent-key> "…"`, or run the
   > standalone `/triage` command if installed. Bundle triage ships via
   > `shark upgrade` once added to canonical (`internal/sharkdata/default_data/`).

   Do not improvise a full classification workflow inline — keep degradation honest.

## Notes

- Dedup before creating: enumerate `shark list <type>` for every candidate
  classification, not just a keyword search.
- On a cloud DB, never assign keys — shark does; capture the key from the response.
