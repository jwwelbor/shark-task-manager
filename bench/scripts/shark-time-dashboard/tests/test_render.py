"""HTML escaping and rendering coverage for the generated dashboard."""

import sqlite3
from collections import Counter
from typing import Any

import pytest
from dashboard_test_support import create_dashboard_schema


@pytest.mark.unit
def test_render_escapes_untrusted_values_and_preserves_computed_output(
    dashboard: Any,
) -> None:
    payload = '</script><script>alert("dashboard-xss")</script>'
    cell_payload = "</td><script>alert('cell-xss')</script>"
    rendered = dashboard.render(
        phase_seconds={("feature", payload): 3600.0},
        wallclock={("task", payload): {"total_hours": 1.0, "n": 1}},
        rework_counts=Counter({"Code review kickback": 2, "QA kickback": 1}),
        cat_counts=Counter({"Code review defect": 1, "Schema/validation gap": 3}),
        n_notes=1,
        quality_rows=[
            {
                "feature": payload,
                "rejections": 1,
                "tech_debt_spawned": 0,
                "bugs": 1,
            },
            {
                "feature": "F01",
                "rejections": 1,
                "tech_debt_spawned": 1,
                "bugs": 0,
            },
            {
                "feature": cell_payload,
                "rejections": 1,
                "tech_debt_spawned": 0,
                "bugs": 0,
            },
            {"feature": "F02", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
        ],
        correlation=0.5,
        excluded=[
            {
                "entity_type": "task",
                "entity_key": payload,
                "started_at": "2026-09-27T12:00:00+00:00",
                "duration_hours": 13.0,
                "outcome": payload,
            }
        ],
        db_path="/private/developer/<db>&.db",
        excluded_intervals=[
            {
                "entity_type": "feature",
                "entity_id": 4,
                "status": "completed",
                "started_at": "2026-09-27T12:00:00+00:00",
                "duration_hours": 24.0,
            }
        ],
    )

    assert '<script>alert("dashboard-xss")</script>' not in rendered
    assert "<td><script>" not in rendered
    assert (
        "&lt;/td&gt;&lt;script&gt;alert(&#x27;cell-xss&#x27;)&lt;/script&gt;"
        in rendered
    )
    assert "&lt;/script&gt;&lt;script&gt;" in rendered
    assert r"\u003c/script\u003e" in rendered
    assert "feature:" in rendered
    assert "Code review kickback" in rendered
    assert rendered.index("Code review kickback") < rendered.index("QA kickback")
    assert "data: [1.0]" in rendered
    assert "data: [2, 1]" in rendered
    assert "F01</td><td>1</td><td>1</td><td>0</td>" in rendered
    assert "F02</td><td>0</td><td>0</td><td>0</td>" in rendered
    assert "13.0" in rendered
    assert "24.0" in rendered
    assert "Pearson r = <b>0.500</b>" in rendered
    assert "status interval" in rendered
    assert "&lt;db&gt;&amp;.db" in rendered
    assert "/private/developer" not in rendered
    assert (
        "https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js" in rendered
    )
    assert (
        'integrity="sha384-vsrfeLOOY6KuIYKDlmVH5UiBmgIdB1oEf7p01YgWHuqmOHfZr374+odEv96n9tNC"'
        in rendered
    )
    assert 'crossorigin="anonymous"' in rendered


@pytest.mark.unit
def test_render_labels_top_15_charts_and_retains_all_exclusions(dashboard: Any) -> None:
    many_phases = {("feature", f"phase-{index}"): index * 3600 for index in range(16)}
    many_statuses = {
        ("feature", f"status-{index}"): {
            "total_hours": float(index),
            "n": 1,
            "avg_hours": float(index),
        }
        for index in range(16)
    }
    rendered = dashboard.render(
        phase_seconds=many_phases,
        wallclock=many_statuses,
        rework_counts=Counter(),
        cat_counts=Counter(),
        n_notes=0,
        quality_rows=[],
        correlation=None,
        excluded=[
            {
                "entity_type": "task",
                "entity_key": f"session-{index}",
                "started_at": "2026-09-27T12:00:00+00:00",
                "duration_hours": 13.0,
                "outcome": "stale",
            }
            for index in range(51)
        ],
        db_path=":memory:",
        excluded_intervals=[
            {
                "entity_type": "feature",
                "entity_id": index,
                "status": "active",
                "started_at": "2026-09-27T12:00:00+00:00",
                "duration_hours": 13.0,
            }
            for index in range(51)
        ],
    )

    assert '"feature:phase-15"' in rendered
    assert '"feature:phase-0"' not in rendered
    assert rendered.count('"feature:phase-') == 15
    assert '"feature:status-15"' in rendered
    assert '"feature:status-0"' not in rendered
    assert rendered.count('"feature:status-') == 15
    assert "top 15 workflow phases" in rendered
    assert "top 15 statuses" in rendered
    assert "session-0" in rendered
    assert "session-50" in rendered
    assert ">0</td>" in rendered
    assert ">50</td>" in rendered


@pytest.mark.unit
def test_render_orders_equal_chart_values_deterministically(dashboard: Any) -> None:
    rendered = dashboard.render(
        phase_seconds={("feature", "zeta"): 3600.0, ("feature", "alpha"): 3600.0},
        wallclock={
            ("feature", "zeta"): {"total_hours": 1.0, "n": 1},
            ("feature", "alpha"): {"total_hours": 1.0, "n": 1},
        },
        rework_counts=Counter({"zeta": 1, "alpha": 1}),
        cat_counts=Counter({"zeta": 1, "alpha": 1}),
        n_notes=0,
        quality_rows=[],
        correlation=None,
        excluded=[],
        db_path=":memory:",
    )

    assert rendered.count('"feature:alpha", "feature:zeta"') == 2
    assert rendered.count('"alpha", "zeta"') == 2


@pytest.mark.unit
def test_empty_feature_database_reports_unavailable_correlation(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert rows == []
        assert correlation is None
        rendered = dashboard.render(
            phase_seconds={},
            wallclock={},
            rework_counts=Counter(),
            cat_counts=Counter(),
            n_notes=0,
            quality_rows=rows,
            correlation=correlation,
            excluded=[],
            db_path=":memory:",
        )
        assert "Pearson r = <b>n/a</b> across 0 features" in rendered
        assert "no features or no variation" in rendered
    finally:
        connection.close()


@pytest.mark.unit
@pytest.mark.parametrize(
    ("correlation", "expected_note"),
    [
        (0.1, "negligible/no linear relationship"),
        (0.15, "positive relationship"),
        (-0.5, "negative relationship"),
        (0.5, "positive relationship"),
    ],
)
def test_render_explains_nonpositive_and_negligible_correlations(
    dashboard: Any, correlation: float, expected_note: str
) -> None:
    rendered = dashboard.render(
        phase_seconds={},
        wallclock={},
        rework_counts=Counter(),
        cat_counts=Counter(),
        n_notes=0,
        quality_rows=[
            {"feature": "F01", "rejections": 1, "tech_debt_spawned": 0, "bugs": 0}
        ],
        correlation=correlation,
        excluded=[],
        db_path=":memory:",
    )

    assert expected_note in rendered


@pytest.mark.unit
def test_render_aggregates_excluded_sessions_from_raw_duration(
    dashboard: Any,
) -> None:
    excluded = [
        {
            "entity_type": "task",
            "entity_key": f"T{index}",
            "started_at": "2026-09-27T12:00:00+00:00",
            "duration_seconds": 1788.0,
            "duration_hours": 0.5,
            "outcome": "stale",
        }
        for index in range(3)
    ]

    rendered = dashboard.render(
        phase_seconds={},
        wallclock={},
        rework_counts=Counter(),
        cat_counts=Counter(),
        n_notes=0,
        quality_rows=[],
        correlation=None,
        excluded=excluded,
        db_path=":memory:",
        excluded_intervals=[],
    )

    assert "3</div>outlier sessions excluded (1h)" in rendered
