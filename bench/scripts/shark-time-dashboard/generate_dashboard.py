#!/usr/bin/env python3
"""
Shark time-allocation & rework-quality dashboard generator.

Reads shark-tasks.db (entity_history, work_sessions, entity_relationships,
bugs, tech_debts) and produces a single self-contained HTML dashboard
showing:
  - Where agent work-session time actually goes, by workflow phase
  - Wall-clock time-in-status (calendar time, for comparison)
  - Rework / rejection loop frequency (task_review kickbacks, code_review
    kickbacks, UAT reverts, etc.)
  - Rejection-note defect-class categorization (keyword-based)
  - Correlation between a feature's rejection-loop count and its downstream
    bug/tech-debt count

Outlier handling: work-session and interval durations are cleaned with an
IQR-based rule (bounded by an absolute 12h sanity cap) before aggregation.
Every excluded record is logged to the "Outliers excluded" section of the
report rather than silently dropped (see README for rationale).

Usage:
    uv run python generate_dashboard.py [--db PATH] [--out PATH]

Re-run any time; it always reflects the live shark-tasks.db.
"""
import argparse
import bisect
import collections
import html
import json
import re
import sqlite3
import statistics
import datetime
from pathlib import Path
from typing import Any


def parse_ts(ts: str | None) -> datetime.datetime | None:
    if not ts:
        return None
    ts = ts.split("+")[0].split(".")[0].replace("T", " ")
    try:
        return datetime.datetime.strptime(ts, "%Y-%m-%d %H:%M:%S")
    except ValueError:
        return None


REWORK_TRANSITIONS = {
    ("feature", "task_review", "task_generation"): "Task review kickback",
    (
        "feature",
        "test_planning",
        "task_generation",
    ): "Test plan sent back for task re-gen",
    ("feature", "code_review", "active"): "Code review kickback",
    ("feature", "qa", "active"): "QA kickback",
    ("feature", "approval", "code_review"): "Approval kickback to code review",
    ("feature", "completed", "approval"): "Reopened after completion",
    (
        "task",
        "completed",
        "development",
    ): "Task reverted after completion (UAT/review fail)",
    ("task", "development", "blocked"): "Task blocked mid-development",
}

DEFECT_CATEGORIES = {
    "Release/verification gate fail": r"release outcome: fail|verification gate",
    "Systematic/defect-class sweep": r"defect class|systematic",
    "Scope/integration contract issue": r"scope|Integration Contract|cross-epic|X-0\d|I-0\d",
    "UAT/red-team failure": r"UAT|red-team|red team",
    "Schema/validation gap": r"schema|validat",
    "Task structure/DAG issue": r"depend|DAG|task-body|task_generation",
    "Code review defect": r"code review|code_review",
    "Unguarded error/crash": r"KeyError|ZeroDivisionError|TypeError|ValueError|crash|unguarded",
    "Missing test coverage": r"test coverage|no automated test|untested|no test",
}

ABSOLUTE_CAP_SECONDS = (
    12 * 3600
)  # no legitimate single agent work-session should exceed this
IQR_MULTIPLIER = 3.0  # "far out" outlier rule, deliberately conservative


def iqr_bounds(values: list[float]) -> float | None:
    if len(values) < 4:
        return None
    q1, q3 = statistics.quantiles(values, n=4)[0], statistics.quantiles(values, n=4)[2]
    iqr = q3 - q1
    return q3 + IQR_MULTIPLIER * iqr


def load_key_maps(conn: sqlite3.Connection) -> dict[str, dict[int, str]]:
    maps = {}
    for etype, table in (
        ("feature", "features"),
        ("task", "tasks"),
        ("epic", "epics"),
        ("bug", "bugs"),
        ("tech_debt", "tech_debts"),
        ("change", "change_cards"),
    ):
        maps[etype] = {
            r["id"]: r["key"] for r in conn.execute(f"select id, key from {table}")
        }
    return maps


def load_history_intervals(
    conn: sqlite3.Connection,
) -> dict[tuple[str, int], list[tuple[datetime.datetime, str]]]:
    hist = collections.defaultdict(list)
    for r in conn.execute(
        "select entity_type, entity_id, to_status, changed_at "
        "from entity_history order by entity_type, entity_id, changed_at"
    ):
        t = parse_ts(r["changed_at"])
        if t:
            hist[(r["entity_type"], r["entity_id"])].append((t, r["to_status"]))
    return hist


def status_at(
    hist: dict[tuple[str, int], list[tuple[datetime.datetime, str]]],
    etype: str,
    eid: int,
    ts: datetime.datetime | None,
) -> str | None:
    lst = hist.get((etype, eid))
    if not lst or ts is None:
        return None
    times = [x[0] for x in lst]
    idx = bisect.bisect_right(times, ts) - 1
    return lst[0][1] if idx < 0 else lst[idx][1]


def analyze_work_sessions(
    conn: sqlite3.Connection,
    hist: dict[tuple[str, int], list[tuple[datetime.datetime, str]]],
    key_maps: dict[str, dict[int, str]],
) -> tuple[
    dict[tuple[str, str], float],
    dict[tuple[str, str], int],
    list[dict[str, Any]],
]:
    """Attribute work-session wall-clock duration to workflow phase, with
    outlier exclusion. Returns (phase_seconds, phase_count, excluded)."""
    durations_by_type = collections.defaultdict(list)
    raw_sessions = []
    for r in conn.execute(
        "select entity_type, entity_key, started_at, ended_at, outcome "
        "from work_sessions where ended_at is not null"
    ):
        t1, t2 = parse_ts(r["started_at"]), parse_ts(r["ended_at"])
        if not t1 or not t2:
            continue
        dur = (t2 - t1).total_seconds()
        if dur <= 0:
            continue
        raw_sessions.append((r["entity_type"], r["entity_key"], t1, dur, r["outcome"]))
        durations_by_type[r["entity_type"]].append(dur)

    bounds = {
        et: (iqr_bounds(v) or ABSOLUTE_CAP_SECONDS)
        for et, v in durations_by_type.items()
    }

    phase_seconds = collections.defaultdict(float)
    phase_count = collections.defaultdict(int)
    excluded = []
    for etype, ekey, t1, dur, outcome in raw_sessions:
        # exclude if beyond either the per-type IQR bound or the absolute sanity cap
        cutoff = min(bounds.get(etype, ABSOLUTE_CAP_SECONDS), ABSOLUTE_CAP_SECONDS)
        if dur > cutoff:
            excluded.append(
                {
                    "entity_type": etype,
                    "entity_key": ekey,
                    "started_at": t1.isoformat(),
                    "duration_hours": round(dur / 3600, 1),
                    "outcome": outcome,
                }
            )
            continue
        phase = None
        for eid_candidate, k in key_maps.get(etype, {}).items():
            if k == ekey:
                phase = status_at(hist, etype, eid_candidate, t1)
                break
        phase = phase or "unknown"
        phase_seconds[(etype, phase)] += dur
        phase_count[(etype, phase)] += 1

    excluded.sort(key=lambda x: -x["duration_hours"])
    return phase_seconds, phase_count, excluded


def analyze_wallclock_status_time(
    conn: sqlite3.Connection,
) -> tuple[dict[tuple[str, str], dict[str, float | int]], int, float]:
    """Time-in-status per entity type using consecutive entity_history rows,
    with IQR-based outlier trimming per (entity_type, status)."""
    rows = collections.defaultdict(list)
    for etype in ("feature", "task", "epic"):
        for r in conn.execute(
            "select entity_id, to_status, changed_at from entity_history "
            "where entity_type=? order by entity_id, changed_at",
            (etype,),
        ):
            rows[etype].append(
                (r["entity_id"], r["to_status"], parse_ts(r["changed_at"]))
            )

    intervals = collections.defaultdict(list)  # (etype,status) -> [durations]
    for etype, evs in rows.items():
        by_entity = collections.defaultdict(list)
        for eid, status, t in evs:
            by_entity[eid].append((status, t))
        for eid, seq in by_entity.items():
            for i in range(len(seq) - 1):
                status, t1 = seq[i]
                _, t2 = seq[i + 1]
                if t1 and t2:
                    dur = (t2 - t1).total_seconds()
                    if dur > 0:
                        intervals[(etype, status)].append(dur)

    excluded_count = 0
    excluded_hours = 0.0
    result = {}
    for key, durs in intervals.items():
        cutoff = iqr_bounds(durs) or float("inf")
        kept = [d for d in durs if d <= cutoff]
        removed = [d for d in durs if d > cutoff]
        excluded_count += len(removed)
        excluded_hours += sum(removed) / 3600
        if kept:
            result[key] = {
                "total_hours": sum(kept) / 3600,
                "n": len(kept),
                "avg_hours": (sum(kept) / len(kept)) / 3600,
            }
    return result, excluded_count, excluded_hours


def analyze_rework(
    conn: sqlite3.Connection,
) -> tuple[collections.Counter[str], collections.Counter[str], int]:
    counts = collections.Counter()
    notes_pool = []
    for (etype, fr, to), label in REWORK_TRANSITIONS.items():
        rows = conn.execute(
            "select notes from entity_history where entity_type=? and from_status=? and to_status=?",
            (etype, fr, to),
        ).fetchall()
        counts[label] = len(rows)
        notes_pool.extend(r["notes"] for r in rows if r["notes"])

    cat_counts = collections.Counter()
    for n in notes_pool:
        for cat, pat in DEFECT_CATEGORIES.items():
            if re.search(pat, n, re.IGNORECASE):
                cat_counts[cat] += 1
    return counts, cat_counts, len(notes_pool)


def analyze_quality_correlation(
    conn: sqlite3.Connection,
) -> tuple[list[dict[str, str | int]], float | None]:
    feat_key = {r["id"]: r["key"] for r in conn.execute("select id, key from features")}
    rejection_pairs = {
        ("task_review", "task_generation"),
        ("test_planning", "task_generation"),
        ("code_review", "active"),
        ("qa", "active"),
        ("approval", "code_review"),
        ("completed", "approval"),
    }
    rejections = collections.defaultdict(int)
    for r in conn.execute(
        "select entity_id, from_status, to_status from entity_history where entity_type='feature'"
    ):
        if (r["from_status"], r["to_status"]) in rejection_pairs:
            rejections[r["entity_id"]] += 1

    techdebt_spawn = collections.defaultdict(int)
    for r in conn.execute(
        "select to_entity_id from entity_relationships where relationship_type='spawned_from' "
        "and from_entity_type='tech_debt' and to_entity_type='feature'"
    ):
        techdebt_spawn[r["to_entity_id"]] += 1

    bug_count = collections.defaultdict(int)
    key_to_fid = {v: k for k, v in feat_key.items()}
    for r in conn.execute(
        "select linked_entity_key from bugs where linked_entity_type='feature'"
    ):
        fid = key_to_fid.get(r["linked_entity_key"])
        if fid:
            bug_count[fid] += 1

    rows = []
    for fid, key in feat_key.items():
        rows.append(
            {
                "feature": key,
                "rejections": rejections.get(fid, 0),
                "tech_debt_spawned": techdebt_spawn.get(fid, 0),
                "bugs": bug_count.get(fid, 0),
            }
        )
    if not rows:
        return rows, None

    xs = [r["rejections"] for r in rows]
    ys = [r["tech_debt_spawned"] + r["bugs"] for r in rows]
    n = len(xs)
    mx, my = sum(xs) / n, sum(ys) / n
    cov = sum((x - mx) * (y - my) for x, y in zip(xs, ys))
    sx = (sum((x - mx) ** 2 for x in xs)) ** 0.5
    sy = (sum((y - my) ** 2 for y in ys)) ** 0.5
    corr = cov / (sx * sy) if sx and sy else 0.0

    rows.sort(key=lambda r: -r["rejections"])
    return rows, corr


HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Shark Time &amp; Quality Dashboard</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
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
<div class="meta">Generated {generated_at} from {db_path} &mdash; re-run <code>python3 generate_dashboard.py</code> any time to refresh.</div>

<div class="kpi">
  <div><div class="num">{total_session_hours:.0f}h</div>kept agent work-session time</div>
  <div><div class="num warn">{excluded_sessions_count}</div>outlier sessions excluded ({excluded_sessions_hours:.0f}h)</div>
  <div><div class="num">{total_rework}</div>rework/rejection events</div>
  <div><div class="num">{correlation_short}</div>rejection&harr;quality-issue correlation (r)</div>
</div>

<h2>Where agent work-session time goes (by workflow phase)</h2>
<div class="card"><canvas id="phaseChart"></canvas></div>

<h2>Wall-clock time-in-status (calendar time, outlier-trimmed)</h2>
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

<h2 class="warn">Outliers excluded from work-session time (IQR &times; {iqr_mult}, capped at {abs_cap}h)</h2>
<p class="meta">These sessions were left out of the "time goes" chart above because their wall-clock duration is
implausible for continuous agent work (most likely: session left open / agent paused without closing it).
Shown here for transparency, not silently dropped.</p>
<table>
<tr><th>Entity type</th><th>Entity key</th><th>Started</th><th>Duration (h)</th><th>Outcome</th></tr>
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


def _json_for_script(value: Any) -> str:
    """Serialize chart data without allowing values to close the script tag."""
    return (
        json.dumps(value, ensure_ascii=False)
        .replace("&", "\\u0026")
        .replace("<", "\\u003c")
        .replace(">", "\\u003e")
        .replace("\u2028", "\\u2028")
        .replace("\u2029", "\\u2029")
    )


def _html_text(value: Any) -> str:
    return html.escape(str(value), quote=True)


def render(
    phase_seconds: dict[tuple[str, str], float],
    phase_count: dict[tuple[str, str], int],
    wallclock: dict[tuple[str, str], dict[str, float | int]],
    rework_counts: collections.Counter[str],
    cat_counts: collections.Counter[str],
    n_notes: int,
    quality_rows: list[dict[str, str | int]],
    correlation: float | None,
    excluded: list[dict[str, Any]],
    db_path: str,
) -> str:
    phase_items = sorted(phase_seconds.items(), key=lambda x: -x[1])[:15]
    phase_labels = _json_for_script([f"{et}:{ph}" for (et, ph), _ in phase_items])
    phase_values = _json_for_script([round(s / 3600, 1) for _, s in phase_items])

    wc_items = sorted(wallclock.items(), key=lambda x: -x[1]["total_hours"])[:15]
    wc_labels = _json_for_script([f"{et}:{st}" for (et, st), _ in wc_items])
    wc_values = _json_for_script([round(v["total_hours"], 1) for _, v in wc_items])

    rework_items = sorted(rework_counts.items(), key=lambda x: -x[1])
    rework_labels = _json_for_script([k for k, _ in rework_items])
    rework_values = _json_for_script([v for _, v in rework_items])

    cat_items = sorted(cat_counts.items(), key=lambda x: -x[1])
    defect_labels = _json_for_script([k for k, _ in cat_items])
    defect_values = _json_for_script([v for _, v in cat_items])

    quality_rows_html = "\n".join(
        f"<tr><td>{_html_text(r['feature'])}</td><td>{_html_text(r['rejections'])}</td>"
        f"<td>{_html_text(r['tech_debt_spawned'])}</td><td>{_html_text(r['bugs'])}</td></tr>"
        for r in quality_rows
        if r["rejections"] or r["tech_debt_spawned"] or r["bugs"]
    )
    excluded_rows_html = (
        "\n".join(
            f"<tr><td>{_html_text(e['entity_type'])}</td><td>{_html_text(e['entity_key'])}</td>"
            f"<td>{_html_text(e['started_at'])}</td><td>{_html_text(e['duration_hours'])}</td>"
            f"<td>{_html_text(e['outcome'])}</td></tr>"
            for e in excluded[:50]
        )
        or "<tr><td colspan='5'>None excluded</td></tr>"
    )

    total_session_hours = sum(phase_seconds.values()) / 3600
    excluded_hours = sum(e["duration_hours"] for e in excluded)
    total_rework = sum(rework_counts.values())

    if correlation is None:
        corr_note = "not available because this database has no features."
        correlation_short = correlation_long = "n/a"
    else:
        correlation_short = f"{correlation:.2f}"
        correlation_long = f"{correlation:.3f}"
        if abs(correlation) < 0.15:
            corr_note = (
                "negligible/no linear relationship in this data. Either review gates catch issues "
                "before they become tracked bugs/tech-debt (so kickbacks don't leak downstream), or "
                "bug/tech-debt linkage back to originating features is too sparse to detect a signal "
                "&mdash; most features currently show 0 linked bugs, which likely reflects incomplete "
                "linking discipline rather than true absence of issues."
            )
        elif correlation > 0:
            corr_note = "weak positive relationship &mdash; more kickbacks associates with more downstream issues."
        else:
            corr_note = "weak negative relationship &mdash; more kickbacks associates with fewer downstream issues."

    return HTML_TEMPLATE.format(
        generated_at=_html_text(datetime.datetime.now().isoformat(timespec="seconds")),
        db_path=_html_text(db_path),
        total_session_hours=total_session_hours,
        excluded_sessions_count=len(excluded),
        excluded_sessions_hours=excluded_hours,
        total_rework=total_rework,
        correlation_short=correlation_short,
        correlation_long=correlation_long,
        phase_labels=phase_labels,
        phase_values=phase_values,
        wallclock_labels=wc_labels,
        wallclock_values=wc_values,
        rework_labels=rework_labels,
        rework_values=rework_values,
        defect_labels=defect_labels,
        defect_values=defect_values,
        n_notes=n_notes,
        quality_rows=quality_rows_html,
        n_features=len(quality_rows),
        correlation_note=corr_note,
        excluded_rows=excluded_rows_html,
        iqr_mult=IQR_MULTIPLIER,
        abs_cap=int(ABSOLUTE_CAP_SECONDS / 3600),
    )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default="shark-tasks.db")
    ap.add_argument("--out", default=None)
    args = ap.parse_args()

    db_path = Path(args.db).resolve()
    out_path = (
        Path(args.out).resolve()
        if args.out
        else Path(__file__).parent / "dashboard.html"
    )

    conn = sqlite3.connect(str(db_path))
    conn.row_factory = sqlite3.Row

    key_maps = load_key_maps(conn)
    hist = load_history_intervals(conn)
    phase_seconds, phase_count, excluded = analyze_work_sessions(conn, hist, key_maps)
    wallclock, wc_excl_count, wc_excl_hours = analyze_wallclock_status_time(conn)
    rework_counts, cat_counts, n_notes = analyze_rework(conn)
    quality_rows, correlation = analyze_quality_correlation(conn)

    print(f"Kept session hours: {sum(phase_seconds.values())/3600:.1f}")
    print(
        f"Excluded sessions: {len(excluded)} ({sum(e['duration_hours'] for e in excluded):.1f}h)"
    )
    print(
        f"Wall-clock outliers excluded: {wc_excl_count} intervals ({wc_excl_hours:.1f}h)"
    )
    print(f"Rework events: {sum(rework_counts.values())}")
    correlation_label = "n/a" if correlation is None else f"{correlation:.3f}"
    print(
        f"Quality correlation r={correlation_label} across {len(quality_rows)} features"
    )

    html = render(
        phase_seconds,
        phase_count,
        wallclock,
        rework_counts,
        cat_counts,
        n_notes,
        quality_rows,
        correlation,
        excluded,
        str(db_path),
    )
    out_path.write_text(html)
    print(f"\nDashboard written to {out_path}")


if __name__ == "__main__":
    main()
