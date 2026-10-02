# Shark time & quality dashboard

Re-runnable analysis of `shark-tasks.db` answering: where does agent time
actually go, where does rework happen, and does rework frequency predict
downstream quality problems?

## Run it

```
uv run python generate_dashboard.py --db /path/to/shark-tasks.db --out dashboard.html
```

Defaults: `--db shark-tasks.db` (cwd), `--out dashboard.html` (script dir).
Open `dashboard.html` in a browser (needs internet for the Chart.js CDN).

## Data sources

- `entity_history`: status transitions (`from_status`/`to_status`/`notes`) —
  used for wall-clock time-in-status and rework/rejection-loop detection.
- `work_sessions`: `started_at`/`ended_at` per agent session — used as the
  best available proxy for actual agent work time (token/duration-level
  usage in `agent_usage_events` is currently unpopulated).
- `entity_relationships` (`spawned_from`) + `bugs.linked_entity_*`: used to
  correlate a feature's rejection-loop count with downstream tech-debt/bugs.

## Outlier handling

Both work-session durations and status-interval durations are cleaned before
aggregation using an IQR rule (`Q3 + 3*IQR`, capped absolutely at 12h per
session) — sessions/intervals beyond that are excluded from the headline
charts as almost certainly "left open"/stale rather than real work, but are
listed in the report's "Outliers excluded" table rather than silently
dropped.

## Known limitation

The rejection→quality correlation (r ≈ 0.03) is likely underpowered: most
features have zero linked bugs/tech-debt, which more plausibly reflects
inconsistent post-hoc linking discipline than genuinely defect-free
features. Treat the correlation as inconclusive, not as evidence that review
gates don't matter.
