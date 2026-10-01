"""Render the Shark dashboard as an escaped HTML report with CDN charts."""

from __future__ import annotations

import collections
import html
import json
from datetime import UTC, datetime
from pathlib import Path
from typing import TypedDict

from dashboard_analysis import ABSOLUTE_CAP_SECONDS, IQR_MULTIPLIER
from dashboard_types import (
    DURATION_DECIMAL_PLACES,
    SECONDS_PER_HOUR,
    ExcludedInterval,
    ExcludedSession,
    QualityRow,
    WallclockSummary,
)

CHART_ITEM_LIMIT = 15
CORRELATION_NEGLIGIBLE_THRESHOLD = 0.15


class TemplateContext(TypedDict):
    phase_labels: str
    phase_values: str
    wallclock_labels: str
    wallclock_values: str
    rework_labels: str
    rework_values: str
    defect_labels: str
    defect_values: str
    generated_at: str
    db_path: str
    total_session_hours: float
    excluded_sessions_count: int
    excluded_sessions_hours: float
    total_rework: int
    correlation_short: str
    correlation_long: str
    n_notes: int
    quality_rows: str
    n_features: int
    correlation_note: str
    excluded_rows: str
    iqr_mult: float
    abs_cap: int
    chart_item_limit: int


class ChartPayload(TypedDict):
    phase_labels: str
    phase_values: str
    wallclock_labels: str
    wallclock_values: str
    rework_labels: str
    rework_values: str
    defect_labels: str
    defect_values: str


HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Shark Time &amp; Quality Dashboard</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js"
  integrity="sha384-vsrfeLOOY6KuIYKDlmVH5UiBmgIdB1oEf7p01YgWHuqmOHfZr374+odEv96n9tNC"
  crossorigin="anonymous"></script>
<style>
  body {{ font-family: -apple-system, Segoe UI, Arial, sans-serif; margin: 2rem; background:#0f1117; color:#e6e6e6; }}
  h1 {{ font-size: 1.6rem; }} h2 {{ margin-top: 2.5rem; border-bottom: 1px solid #333; padding-bottom: .3rem; }}
  .meta {{ color:#999; font-size:.85rem; }}
  .grid {{ display:grid; grid-template-columns: 1fr 1fr; gap:2rem; }}
  .card {{ background:#1a1d27; border-radius:10px; padding:1rem 1.5rem; box-shadow:0 1px 4px rgba(0,0,0,.4); }}
  table {{ border-collapse: collapse; width:100%; font-size:.85rem; }}
  th, td {{ text-align:left; padding:.35rem .6rem; border-bottom:1px solid #2a2d3a; }}
  th {{ color:#9ab; }}
  .kpi {{ display:flex; gap:1.5rem; flex-wrap:wrap; }}
  .kpi div {{ background:#1a1d27; border-radius:10px; padding:1rem 1.5rem; min-width:180px; }}
  .kpi .num {{ font-size:1.8rem; font-weight:700; }}
  .warn {{ color:#e2b93b; }}
  canvas {{ max-height: 380px; }}
</style>
</head>
<body>
<h1>&#128267; Shark Time &amp; Quality Dashboard</h1>
<div class="meta">Generated {generated_at} from {db_path} &mdash; external Chart.js CDN; re-run <code>uv run python scripts/shark-time-dashboard/generate_dashboard.py</code> to refresh.</div>

<div class="kpi">
  <div><div class="num">{total_session_hours:.0f}h</div>kept agent work-session time</div>
  <div><div class="num warn">{excluded_sessions_count}</div>outlier sessions excluded ({excluded_sessions_hours:.0f}h)</div>
  <div><div class="num">{total_rework}</div>rework/rejection events</div>
  <div><div class="num">{correlation_short}</div>rejection&harr;quality-issue correlation (r)</div>
</div>

<h2>Where agent work-session time goes (top {chart_item_limit} workflow phases)</h2>
<div class="card"><canvas id="phaseChart"></canvas></div>

<h2>Wall-clock time-in-status (top {chart_item_limit} statuses; completed intervals, outlier-trimmed)</h2>
<div class="card"><canvas id="wallclockChart"></canvas></div>

<h2>Rework &amp; rejection-loop frequency</h2>
<div class="grid">
  <div class="card"><canvas id="reworkChart"></canvas></div>
  <div class="card"><canvas id="defectClassChart"></canvas></div>
</div>

<h2>Quality correlation: does review kickback frequency predict downstream bugs/tech-debt?</h2>
<p>Pearson r = <b>{correlation_long}</b> across {n_features} features (rejections vs. linked bugs + tech-debt spawned from that feature).
Interpretation: {correlation_note}</p>
<table>
<tr><th>Feature</th><th>Rejection loops</th><th>Tech-debt spawned</th><th>Bugs linked</th></tr>
{quality_rows}
</table>

<h2 class="warn">Outliers excluded from work-session and status-interval time (IQR &times; {iqr_mult}, capped at {abs_cap}h)</h2>
<p class="meta">These records were left out of the headline charts because their wall-clock duration is implausible for continuous agent work or status time. They are shown here for transparency, not silently dropped.</p>
<table>
<tr><th>Kind</th><th>Entity type</th><th>Entity</th><th>Started</th><th>Duration (h)</th><th>Status/outcome</th></tr>
{excluded_rows}
</table>

<script>
new Chart(document.getElementById('phaseChart'), {{
  type: 'bar',
  data: {{ labels: {phase_labels}, datasets: [{{ label: 'Hours (session time, outlier-excluded)', data: {phase_values}, backgroundColor: '#5b8def' }}] }},
  options: {{ indexAxis: 'y', plugins: {{ legend: {{ display:false }} }} }}
}});
new Chart(document.getElementById('wallclockChart'), {{
  type: 'bar',
  data: {{ labels: {wallclock_labels}, datasets: [{{ label: 'Hours (calendar time-in-status)', data: {wallclock_values}, backgroundColor: '#e28743' }}] }},
  options: {{ indexAxis: 'y', plugins: {{ legend: {{ display:false }} }} }}
}});
new Chart(document.getElementById('reworkChart'), {{
  type: 'bar',
  data: {{ labels: {rework_labels}, datasets: [{{ label: 'Occurrences', data: {rework_values}, backgroundColor: '#d9534f' }}] }},
  options: {{ indexAxis: 'y', plugins: {{ legend: {{ display:false }}, title: {{ display:true, text:'Rework/rejection loops by type', color:'#ccc' }} }} }}
}});
new Chart(document.getElementById('defectClassChart'), {{
  type: 'doughnut',
  data: {{ labels: {defect_labels}, datasets: [{{ data: {defect_values}, backgroundColor: ['#5b8def','#e28743','#d9534f','#5cb85c','#9b59b6','#f0ad4e','#1abc9c','#e74c3c','#95a5a6'] }}] }},
  options: {{ plugins: {{ title: {{ display:true, text:'Rework-note defect classes ({n_notes} notes analyzed)', color:'#ccc' }} }} }}
}});
</script>
</body>
</html>
"""


def _json_for_script(value: object) -> str:
    """Serialize chart data without allowing values to close the script tag."""
    return (
        json.dumps(value, ensure_ascii=False)
        .replace("&", "\\u0026")
        .replace("<", "\\u003c")
        .replace(">", "\\u003e")
        .replace("\u2028", "\\u2028")
        .replace("\u2029", "\\u2029")
    )


def _html_text(value: object) -> str:
    return html.escape(str(value), quote=True)


def _quality_rows_html(rows: list[QualityRow]) -> str:
    return "\n".join(
        f"<tr><td>{_html_text(row['feature'])}</td><td>{_html_text(row['rejections'])}</td>"
        f"<td>{_html_text(row['tech_debt_spawned'])}</td><td>{_html_text(row['bugs'])}</td></tr>"
        for row in rows
    )


def _excluded_rows_html(
    sessions: list[ExcludedSession], intervals: list[ExcludedInterval]
) -> str:
    def session_duration_hours(item: ExcludedSession) -> float:
        duration_seconds = item.get(
            "duration_seconds", item["duration_hours"] * SECONDS_PER_HOUR
        )
        return round(
            float(duration_seconds) / SECONDS_PER_HOUR, DURATION_DECIMAL_PLACES
        )

    rows = [
        (
            "work session",
            item["entity_type"],
            item["entity_key"],
            item["started_at"],
            session_duration_hours(item),
            item.get("outcome") or "-",
        )
        for item in sessions
    ] + [
        (
            "status interval",
            item["entity_type"],
            item["entity_id"],
            item["started_at"],
            item["duration_hours"],
            item["status"] or "-",
        )
        for item in intervals
    ]
    return (
        "\n".join(
            "<tr>" + "".join(f"<td>{_html_text(value)}</td>" for value in row) + "</tr>"
            for row in rows
        )
        or "<tr><td colspan='6'>None excluded</td></tr>"
    )


def _correlation_display(
    correlation: float | None, feature_count: int
) -> tuple[str, str, str]:
    if correlation is None:
        return (
            "n/a",
            "n/a",
            "not available because this database has no features or no variation in one series.",
        )
    short = f"{correlation:.2f}"
    long = f"{correlation:.3f}"
    if abs(correlation) < CORRELATION_NEGLIGIBLE_THRESHOLD:
        note = (
            "negligible/no linear relationship in this data; the result is descriptive and may "
            f"be underpowered across {feature_count} features."
        )
    elif correlation > 0:
        note = "positive relationship — more kickbacks associates with more downstream issues."
    else:
        note = "negative relationship — more kickbacks associates with fewer downstream issues."
    return short, long, note


def _chart_payload(
    phase_seconds: dict[tuple[str, str], float],
    wallclock: dict[tuple[str, str], WallclockSummary],
    rework_counts: collections.Counter[str],
    cat_counts: collections.Counter[str],
) -> ChartPayload:
    phase_items = sorted(phase_seconds.items(), key=lambda item: (-item[1], item[0]))[
        :CHART_ITEM_LIMIT
    ]
    wallclock_items = sorted(
        wallclock.items(),
        key=lambda item: (-float(item[1]["total_hours"]), item[0]),
    )[:CHART_ITEM_LIMIT]
    rework_items = sorted(rework_counts.items(), key=lambda item: (-item[1], item[0]))
    defect_items = sorted(cat_counts.items(), key=lambda item: (-item[1], item[0]))
    return {
        "phase_labels": _json_for_script(
            [f"{et}:{phase}" for (et, phase), _ in phase_items]
        ),
        "phase_values": _json_for_script(
            [
                round(
                    seconds / SECONDS_PER_HOUR,
                    DURATION_DECIMAL_PLACES,
                )
                for _, seconds in phase_items
            ]
        ),
        "wallclock_labels": _json_for_script(
            [f"{et}:{status}" for (et, status), _ in wallclock_items]
        ),
        "wallclock_values": _json_for_script(
            [
                round(float(values["total_hours"]), DURATION_DECIMAL_PLACES)
                for _, values in wallclock_items
            ]
        ),
        "rework_labels": _json_for_script([label for label, _ in rework_items]),
        "rework_values": _json_for_script([count for _, count in rework_items]),
        "defect_labels": _json_for_script([label for label, _ in defect_items]),
        "defect_values": _json_for_script([count for _, count in defect_items]),
    }


def _template_context(
    *,
    phase_seconds: dict[tuple[str, str], float],
    wallclock: dict[tuple[str, str], WallclockSummary],
    rework_counts: collections.Counter[str],
    cat_counts: collections.Counter[str],
    n_notes: int,
    quality_rows: list[QualityRow],
    correlation: float | None,
    excluded: list[ExcludedSession],
    db_path: str,
    excluded_intervals: list[ExcludedInterval],
) -> TemplateContext:
    correlation_short, correlation_long, correlation_note = _correlation_display(
        correlation, len(quality_rows)
    )
    return {
        **_chart_payload(phase_seconds, wallclock, rework_counts, cat_counts),
        "generated_at": _html_text(datetime.now(UTC).isoformat(timespec="seconds")),
        "db_path": _html_text(Path(db_path).name),
        "total_session_hours": sum(phase_seconds.values()) / SECONDS_PER_HOUR,
        "excluded_sessions_count": len(excluded),
        "excluded_sessions_hours": sum(
            float(
                item.get("duration_seconds", item["duration_hours"] * SECONDS_PER_HOUR)
            )
            for item in excluded
        )
        / SECONDS_PER_HOUR,
        "total_rework": sum(rework_counts.values()),
        "correlation_short": correlation_short,
        "correlation_long": correlation_long,
        "n_notes": n_notes,
        "quality_rows": _quality_rows_html(quality_rows),
        "n_features": len(quality_rows),
        "correlation_note": correlation_note,
        "excluded_rows": _excluded_rows_html(excluded, excluded_intervals),
        "iqr_mult": IQR_MULTIPLIER,
        "abs_cap": int(ABSOLUTE_CAP_SECONDS / SECONDS_PER_HOUR),
        "chart_item_limit": CHART_ITEM_LIMIT,
    }


def render(
    *,
    phase_seconds: dict[tuple[str, str], float],
    wallclock: dict[tuple[str, str], WallclockSummary],
    rework_counts: collections.Counter[str],
    cat_counts: collections.Counter[str],
    n_notes: int,
    quality_rows: list[QualityRow],
    correlation: float | None,
    excluded: list[ExcludedSession],
    db_path: str,
    excluded_intervals: list[ExcludedInterval] | None = None,
) -> str:
    """Render analysis results into an escaped HTML dashboard."""
    return HTML_TEMPLATE.format(
        **_template_context(
            phase_seconds=phase_seconds,
            wallclock=wallclock,
            rework_counts=rework_counts,
            cat_counts=cat_counts,
            n_notes=n_notes,
            quality_rows=quality_rows,
            correlation=correlation,
            excluded=excluded,
            db_path=db_path,
            excluded_intervals=excluded_intervals or [],
        )
    )
