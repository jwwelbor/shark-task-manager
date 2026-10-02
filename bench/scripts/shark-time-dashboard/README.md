# Shark time & quality dashboard

Re-runnable analysis of `shark-tasks.db` answering: where does agent time
actually go, where does rework happen, and does rework frequency predict
downstream quality problems? It produces an HTML report that uses a pinned
Chart.js CDN dependency.

## Run it

```
uv run python bench/scripts/shark-time-dashboard/generate_dashboard.py --db /path/to/shark-tasks.db --out bench/scripts/shark-time-dashboard/dashboard.html
```

Defaults: `--db shark-tasks.db` (cwd), `--out dashboard.html` (script dir).
An explicitly supplied relative `--out` path is relative to the current
working directory.
Open `dashboard.html` in a browser. Internet access is required for the pinned
Chart.js CDN dependency.

## Tests

The utility tests live with the utility and are intentionally outside WWGM's
application test suite:

```
uv run pytest bench/scripts/shark-time-dashboard/tests
```

The headline charts show the top 15 categories by total; the full exclusion
table and quality table remain in the report. Wall-clock metrics cover only
completed intervals between recorded status transitions.

## Data sources

- `entity_history`: status transitions (`from_status`/`to_status`/`notes`) —
  used for wall-clock time-in-status and rework/rejection-loop detection.
- `work_sessions`: `started_at`/`ended_at` per agent session — used as the
  best available proxy for actual agent work time (token/duration-level
  usage in `agent_usage_events` is currently unpopulated).
- `entity_relationships` (`spawned_from` and bug `linked_to`/historical
  `related_to`) plus legacy `bugs.linked_entity_*`: used to correlate a
  feature's rejection-loop count with downstream tech-debt/bugs.

## Outlier handling

Both work-session durations and status-interval durations are cleaned before
aggregation using an IQR rule (`Q3 + 3*IQR`, capped absolutely at 12h) —
sessions/intervals beyond that are excluded from the headline charts as almost
certainly "left open"/stale rather than real work, but are listed in the
report's "Outliers excluded" table rather than silently dropped. Timestamps are
normalized to UTC; malformed non-empty timestamps fail the report loudly.

## Known limitation

The rejection→quality correlation (r ≈ 0.03) is likely underpowered: most
features have zero linked bugs/tech-debt, which more plausibly reflects
inconsistent post-hoc linking discipline than genuinely defect-free
features. Treat the correlation as inconclusive, not as evidence that review
gates don't matter.
