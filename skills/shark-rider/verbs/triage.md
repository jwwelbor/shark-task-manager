# /shark-rider triage — Quick-capture & classify

Capture a discovered work item and route it to the correct shark entity (task,
feature, bug, tech-debt, change-card, idea, or note) under the right parent.
This is **capture-and-classify**, not create-and-elaborate.

Usage: `/shark-rider triage "short description of the thing to track"`

## Procedure

1. Read `skills/triage/SKILL.md` (under this shark skill's directory) and follow
   its procedure, passing any remaining arguments through as that skill's
   arguments.

## Notes

- Dedup before creating: enumerate `shark list <type>` for every candidate
  classification, not just a keyword search.
- On a cloud DB, never assign keys — shark does; capture the key from the response.
