# Ops-as-Entities Convention

Recurring operational work — deploy runs, infrastructure changes, devops tasks,
on-call remediations — belongs as **shark entities** (tasks or change-cards),
not as checklist items inside the `/shark-rider project` namespace.

---

## The Convention

When operational work recurs or has a meaningful history, model it as a shark
entity:

- **Tasks** (`E##-F##-###`) for operational work that is part of a feature or
  epic (e.g., "Deploy v2 to staging," "Rotate prod credentials after release").
- **Change-cards** (`CC-###`) for standalone operational changes that are not
  tied to a feature (e.g., "Rotate AWS IAM keys," "Update TLS certificate,"
  "Terraform drift remediation").
- **An "Ops" epic** (e.g., `E99`) as an optional home for groups of recurring
  operational tasks that don't belong to any product epic.

---

## Why Not a Checklist Item?

The `/shark-rider project` checklist is for **pre-epic, one-time, human-driven
activities** that produce a durable artifact (a design doc, a decision record, a
validated assumption). It is not a general-purpose to-do list.

A checklist item cannot model recurrence. If a deploy happens 200 times, you
cannot have 200 checkboxes in a project bootstrap checklist — but you can have
200 shark tasks, each with its own status history, completion timestamp, and
assignee. Shark entities give you:

- **History** — who ran it, when, and with what outcome
- **Status transitions** — track in-progress vs. completed vs. blocked runs
- **Queryability** — `shark list E99` or `shark list --status=blocked` to find
  stuck ops work
- **Notes and context** — attach runbooks, post-mortems, and blockers to the
  entity

---

## `/shark-rider project` Namespace Membership Rule

An activity belongs in the `/shark-rider project` namespace only if it meets **all
four criteria**:

| Criterion | Description |
|-----------|-------------|
| **Pre-epic** | It must happen before any epic can be started |
| **One-time** | It is done once per project, not repeated |
| **Human-driven** | It requires deliberate human judgment, not routine execution |
| **Produces a durable doc** | The output is a document or decision that other work references |

If the activity fails any of these criteria — particularly if it recurs or
doesn't produce a durable artifact — it belongs as a shark entity instead.

---

## Concrete Examples

### Do This / Not That

| Situation | Wrong | Right |
|-----------|-------|-------|
| Daily deploy to staging | Checklist item "Deploy to staging" | `shark task create E05 F02 "Deploy build #312 to staging"` |
| Monthly cert rotation | Checklist item "Rotate TLS cert" | `shark create change "Rotate TLS cert — July 2026" --tag ops` |
| Terraform drift fix | Note in a doc | `shark create change "Remediate Terraform drift in prod VPC"` |
| Infrastructure audit (one-time, produces report) | Shark task | `/shark-rider project` checklist item — produces a durable audit report |
| "Set up CI pipeline" (one-time, pre-epic) | Recurring ops task | `/shark-rider project` checklist item — done once, produces a CI config artifact |
| On-call incident response | Jira only | `shark create change "Incident CC-042: redis OOM — 2026-06-29"` |

### Grouping Recurring Ops Work

```bash
# Create an Ops epic as a home for recurring work
shark create epic "Ops & Infra" --priority=2

# Group change-cards under it (or leave them standalone)
shark task create E99 F01 "Deploy release 3.4.1 to prod"
shark task create E99 F01 "Rotate prod DB credentials"

# Standalone change-cards for one-off operational changes
shark create change "Upgrade k8s node pool to 1.30"
```

---

## Summary

- **Recurring or repeatable work** → shark task or change-card
- **One-time, pre-epic, human-driven, produces a durable doc** → `/shark-rider project` checklist
- **Ambiguous?** Default to a shark entity — history and queryability are almost
  always more valuable than a checkbox

---

## Related Documentation

- [Route-Based Workflow Guide](route-based-workflow.md) — status transitions and outcome routing
- [CLI Reference — Change Commands](../cli-reference/README.md) — `shark create change`, `shark change get`
- [CLI Reference — Task Commands](../cli-reference/README.md) — full task lifecycle
