# Sizing Guide

Sizing in Shark is used to estimate effort and track velocity. Shark uses a canonical Fibonacci set for numeric sizes, which can also be represented as T-shirt labels.

## Size Scales

Shark supports two primary development models: **Human-Centric** and **AI-Native**. The interpretation of "size" differs significantly between them.

### AI-Native Sizing (Autonomous Agents)

For AI coding agents, sizes are measured in **Active Cycle Time** (time spent in `in_development` or `in_review` statuses). AI agents maintain deep focus but are bound by context windows and token limits.

| Label | Points | AI Active Time | Workflow Guidance |
| :--- | :--- | :--- | :--- |
| **XS** | 1 | < 5 mins | Atomic updates: documentation, single-line fixes, config tweaks. |
| **S** | 2 | 5 - 20 mins | Simple logic: adding a struct field, writing a single unit test. |
| **M** | 3 | 20 - 60 mins | Standard implementation: service/repository methods, feature logic. |
| **L** | 5 | 1 - 2 hours | Complex work: multi-file features, large test suites, integrations. |
| **XL** | 8 | 2 - 5 hours | Major refactoring or foundation for new modules. |
| **XXL** | 13 | > 5 hours | **CRITICAL RISK.** High chance of context drift. **MUST BE DECOMPOSED.** |

### Human-Centric Sizing

For human developers, sizes typically map to traditional agile "days of effort."

| Label | Points | Human Effort | Guidance |
| :--- | :--- | :--- | :--- |
| **XS** | 1 | < 4 hours | Quick task, minimal overhead. |
| **S** | 2 | 4 - 8 hours | Solid 1-day task. |
| **M** | 3 | 1 - 2 days | Standard task size. |
| **L** | 5 | 2 - 4 days | Complex task; watch for blockers. |
| **XL** | 8 | 4 - 7 days | Large effort; decomposition recommended. |
| **XXL** | 13 | > 7 days | **MUST BE DECOMPOSED.** |

---

## Using the `--size` Flag

The `--size` flag is supported on `create` and `update` commands for tasks, features, and epics.

### Accepted Values

You can provide either the numeric Fibonacci value or the T-shirt label (case-insensitive):
- **Numeric**: `1`, `2`, `3`, `5`, `8`, `13`
- **Labels**: `XS`, `S`, `M`, `L`, `XL`, `XXL`

### Examples

```bash
# Create a small task
shark task create E07 F01 "Fix typo in README" --size=XS

# Update an existing feature size using numeric points
shark feature update E07-F01 --size=3

# Clear a size estimate
shark task update T-E07-F01-001 --size=clear
```

---

## Sizing and Sprint Planning

Sizes are critical for effective sprint planning and agentic execution:
1. **Velocity**: Run `shark sprint velocity` to see your team's historical capacity.
2. **Readiness**: `shark sprint readiness` checks if your sprint is over-capacity or has too many "unsized" items.
3. **Decomposition**: Any entity sized **XXL (13)** should be flagged for breakdown into smaller, more manageable units.

---

## Analytics and Calibration

Shark automatically tracks active time for every task based on status transitions. You can calibrate these guidelines for your team by running:

```bash
shark sprint summary <key> --detailed
```

Look at the **Cycle Time by Phase** and **Size Distribution** sections to see if your "M" tasks are actually taking "L" amounts of time. This data should be used in **Agentic Retrospectives** to refine planning prompts.
