---
entity_key: E38-F07
entity_type: feature
recipe: universal
rigor: standard
categories:
  - workflow_operations
  - documentation
source_set:
  - docs/plan/E38-shark-attack-team-orchestration/E38-F07-rider-execution-and-escalation/feature.md
  - skills/shark-rider/SKILL.md
  - skills/shark-rider/verbs/run.md
  - internal/sharkdata/default_data/skills/shark-attack/SKILL.md
  - internal/sharkdata/default_data/skills/shark-attack/workflows/escalate.md
related_work: true
---

# Research report

## Scope

E38-F07 adds a role-aware execution and escalation procedure around the existing Shark Rider dispatch loop.

## Capability map

| Capability | Source | Decision |
| --- | --- | --- |
| Parent-owned `next → claim → dispatch → advance → release` loop | `skills/shark-rider/SKILL.md` and `skills/shark-rider/verbs/run.md` | REUSE |
| Role roster, messages, escalation, and resume procedures | `internal/sharkdata/default_data/skills/shark-attack/` | EXTEND |
| Durable council coordination surface | E38-F04 feature brief | REUSE |
| Claim-time role-aware selection | E38-F06 feature brief | REUSE |
| New provider runtime, aggregate state, or team CLI | E38-F07 feature brief | NEW: explicitly out of scope |

## Ubiquitous vocabulary

- Rider: the parent procedure that owns Shark state transitions and leases.
- Worker: a dispatched craft agent that returns a semantic outcome only.
- Chair: the role that resolves escalations or routes them to product or human review.
- Escalation: a durable question with evidence, decision needed, and next owner.

## Findings

The existing Rider procedure already preserves prompt fidelity and keeps claims and transitions parent-owned. Shark Attack already supplies the role and escalation protocols. The feature should compose those surfaces, not recreate dispatch state, claims, or resume storage.

## Decisions

Use `shark next <key> --json` as the sole dispatch source. Keep the worker unable to claim, heartbeat, release, or advance. Store escalation and handoff context in the existing council artifacts. Reuse E38-F04 and E38-F06 contracts rather than defining parallel equivalents.

## Sources

- `skills/shark-rider/SKILL.md`
- `skills/shark-rider/verbs/run.md`
- `internal/sharkdata/default_data/skills/shark-attack/SKILL.md`
- `internal/sharkdata/default_data/skills/shark-attack/workflows/escalate.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F04-shark-attack-skill-and-role-protocol/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F06-role-aware-pull-and-claim/feature.md`
