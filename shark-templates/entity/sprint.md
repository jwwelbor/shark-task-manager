---
sprint_key: {{.SprintKey}}
name: {{.Name}}
goal: {{.Goal}}
start_date: {{.StartDate}}
end_date: {{.EndDate}}
status: {{.Status}}
---

# {{.Name}}

**Sprint**: {{.SprintKey}} | **{{.StartDate}} → {{.EndDate}}**

---

## Goal

{{.Goal}}

---

## Assigned Work

| Key | Type | Title | Size | Agent | Status |
|-----|------|-------|------|-------|--------|
| _run `shark sprint backlog {{.SprintKey}}` to see current assignments_ | | | | | |

---

## Capacity

_run `shark sprint capacity {{.SprintKey}}` to see configured capacity_

---

## Notes

<!-- Add sprint notes, blockers, or retrospective items here -->

*Created*: {{.Date}}
