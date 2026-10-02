"""Analysis and timestamp coverage for the generated dashboard."""

import datetime
import sqlite3
from typing import Any

import pytest


@pytest.mark.unit
@pytest.mark.parametrize("value", [None, ""])
def test_parse_ts_returns_none_for_empty_values(
    dashboard: Any, value: str | None
) -> None:
    assert dashboard.parse_ts(value) is None


@pytest.mark.unit
def test_parse_ts_normalizes_supported_offsets_and_rejects_bad_values(
    dashboard: Any,
) -> None:
    utc = datetime.UTC
    assert dashboard.parse_ts("2026-08-06T11:00:00Z") == datetime.datetime(
        2026, 8, 6, 11, tzinfo=utc
    )
    assert dashboard.parse_ts("2026-08-06T06:00:00-05:00") == datetime.datetime(
        2026, 8, 6, 11, tzinfo=utc
    )
    assert dashboard.parse_ts(
        "2026-08-06 06:00:00.123456 -0500 CDT m=+1.2"
    ) == datetime.datetime(2026, 8, 6, 11, 0, 0, 123456, tzinfo=utc)
    assert dashboard.parse_ts("2026-08-06 06:00:00") == datetime.datetime(
        2026, 8, 6, 6, tzinfo=utc
    )
    with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
        dashboard.parse_ts("not-a-timestamp")


@pytest.mark.unit
def test_parse_ts_rejects_non_string_values_instead_of_treating_them_as_missing(
    dashboard: Any,
) -> None:
    with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
        dashboard.parse_ts(0)


@pytest.mark.unit
@pytest.mark.parametrize("value", [" ", "m=+1.2", "2026-08-06", "2026-08-06X06:00:00"])
def test_parse_ts_rejects_nonempty_values_that_normalize_to_no_timestamp(
    dashboard: Any, value: str
) -> None:
    with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
        dashboard.parse_ts(value)


@pytest.mark.unit
def test_malformed_database_timestamps_fail_analysis_loudly(dashboard: Any) -> None:
    history_connection = sqlite3.connect(":memory:")
    history_connection.row_factory = sqlite3.Row
    history_connection.execute(
        "create table entity_history "
        "(entity_type text, entity_id integer, to_status text, changed_at text)"
    )
    history_connection.execute(
        "insert into entity_history values ('feature', 1, 'active', 'not-a-timestamp')"
    )
    try:
        with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
            dashboard.load_history_intervals(history_connection)
    finally:
        history_connection.close()

    session_connection = sqlite3.connect(":memory:")
    session_connection.row_factory = sqlite3.Row
    session_connection.execute(
        "create table work_sessions "
        "(entity_type text, entity_key text, started_at text, ended_at text, outcome text)"
    )
    session_connection.execute(
        "insert into work_sessions values ('feature', 'F01', 'not-a-timestamp', "
        "'2026-01-01T01:00:00Z', 'fail')"
    )
    try:
        with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
            dashboard.analyze_work_sessions(session_connection, {}, {})
    finally:
        session_connection.close()

    ended_connection = sqlite3.connect(":memory:")
    ended_connection.row_factory = sqlite3.Row
    ended_connection.execute(
        "create table work_sessions "
        "(entity_type text, entity_key text, started_at text, ended_at text, outcome text)"
    )
    ended_connection.execute(
        "insert into work_sessions values ('feature', 'F01', "
        "'2026-01-01T00:00:00Z', 'not-a-timestamp', 'fail')"
    )
    try:
        with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
            dashboard.analyze_work_sessions(ended_connection, {}, {})
    finally:
        ended_connection.close()


@pytest.mark.unit
def test_analyzers_cover_session_rework_correlation_and_status_cap(
    dashboard: Any, dashboard_db: sqlite3.Connection
) -> None:
    key_maps = dashboard.load_key_maps(dashboard_db)
    history = dashboard.load_history_intervals(dashboard_db)
    phase_seconds, excluded_sessions = dashboard.analyze_work_sessions(
        dashboard_db, history, key_maps
    )
    (
        wallclock,
        excluded_count,
        excluded_hours,
        excluded_intervals,
    ) = dashboard.analyze_wallclock_status_time(dashboard_db)
    rework, categories, note_count = dashboard.analyze_rework(dashboard_db)
    quality_rows, correlation = dashboard.analyze_quality_correlation(dashboard_db)

    assert phase_seconds == {("feature", "active"): 1800.0}
    assert excluded_sessions[0]["entity_key"] == "F02"
    assert wallclock["feature", "active"] == {
        "total_hours": 2.0,
        "n": 2,
        "avg_hours": 1.0,
    }
    assert excluded_count == 1
    assert excluded_hours == 120.0
    assert excluded_intervals == [
        {
            "entity_type": "feature",
            "entity_id": 2,
            "status": "active",
            "started_at": "2026-01-01T00:00:00+00:00",
            "duration_hours": 120.0,
        }
    ]
    assert rework["Code review kickback"] == 1
    assert categories["Code review defect"] == 1
    assert note_count == 1
    assert quality_rows == [
        {"feature": "F01", "rejections": 1, "tech_debt_spawned": 1, "bugs": 1},
        {"feature": "F02", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
    ]
    assert correlation == pytest.approx(1.0)


@pytest.mark.unit
def test_history_is_sorted_after_timezone_normalization(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute(
        "create table entity_history "
        "(entity_type text, entity_id integer, to_status text, changed_at text)"
    )
    connection.executemany(
        "insert into entity_history values (?, ?, ?, ?)",
        [
            ("feature", 1, "blocked", "2026-01-01T01:30:00-02:00"),
            ("feature", 1, "active", "2026-01-01T02:00:00+00:00"),
            ("feature", 1, "review", "2026-01-01T02:00:00+00:00"),
        ],
    )

    try:
        history = dashboard.load_history_intervals(connection)
        timestamps, statuses, *_ = history[("feature", 1)]
        assert timestamps == [
            datetime.datetime(2026, 1, 1, 2, tzinfo=datetime.UTC),
            datetime.datetime(2026, 1, 1, 2, tzinfo=datetime.UTC),
            datetime.datetime(2026, 1, 1, 3, 30, tzinfo=datetime.UTC),
        ]
        assert statuses == ["active", "review", "blocked"]
        assert (
            dashboard.status_at(
                history,
                "feature",
                1,
                datetime.datetime(2026, 1, 1, 2, tzinfo=datetime.UTC),
            )
            == "review"
        )
        assert (
            dashboard.status_at(
                history,
                "feature",
                1,
                datetime.datetime(2026, 1, 1, 2, 30, tzinfo=datetime.UTC),
            )
            == "review"
        )
        assert (
            dashboard.status_at(
                history,
                "feature",
                1,
                datetime.datetime(2026, 1, 1, 1, tzinfo=datetime.UTC),
            )
            is None
        )
    finally:
        connection.close()


@pytest.mark.unit
def test_work_session_time_is_split_at_status_transitions(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute(
        "create table work_sessions "
        "(entity_type text, entity_key text, started_at text, ended_at text, outcome text)"
    )
    connection.execute(
        "insert into work_sessions values (?, ?, ?, ?, ?)",
        (
            "feature",
            "F01",
            "2026-01-01T00:30:00Z",
            "2026-01-01T02:30:00Z",
            "pass",
        ),
    )
    history = {
        ("feature", 1): (
            [
                datetime.datetime(2026, 1, 1, 0, tzinfo=datetime.UTC),
                datetime.datetime(2026, 1, 1, 1, tzinfo=datetime.UTC),
                datetime.datetime(2026, 1, 1, 2, tzinfo=datetime.UTC),
            ],
            ["active", "code_review", "active"],
        )
    }

    try:
        phase_seconds, excluded = dashboard.analyze_work_sessions(
            connection, history, {"feature": {"F01": 1}}
        )
        assert excluded == []
        assert phase_seconds == {
            ("feature", "active"): 3600.0,
            ("feature", "code_review"): 3600.0,
        }
    finally:
        connection.close()


@pytest.mark.unit
def test_task_id_only_session_uses_task_history(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute(
        "create table work_sessions ("
        "entity_type text, entity_key text, task_id integer, "
        "started_at text, ended_at text, outcome text)"
    )
    connection.execute(
        "insert into work_sessions values (?, ?, ?, ?, ?, ?)",
        (
            "task",
            None,
            7,
            "2026-01-01T00:00:00Z",
            "2026-01-01T01:00:00Z",
            "pass",
        ),
    )
    history = {
        ("task", 7): (
            [
                datetime.datetime(2026, 1, 1, 0, tzinfo=datetime.UTC),
                datetime.datetime(2026, 1, 1, 1, tzinfo=datetime.UTC),
            ],
            ["development", "completed"],
        )
    }

    try:
        phase_seconds, excluded = dashboard.analyze_work_sessions(
            connection, history, {"task": {"T07": 7}}
        )
        assert excluded == []
        assert phase_seconds == {("task", "development"): 3600.0}
    finally:
        connection.close()


@pytest.mark.unit
def test_work_session_before_first_transition_uses_from_status(
    dashboard: Any,
) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.executescript("""
        create table work_sessions (
            entity_type text, entity_key text, started_at text,
            ended_at text, outcome text
        );
        create table entity_history (
            entity_type text, entity_id integer, from_status text,
            to_status text, changed_at text
        );
        """)
    connection.execute(
        "insert into work_sessions values ('feature', 'F01', "
        "'2026-01-01T00:00:00Z', '2026-01-01T01:00:00Z', 'pass')"
    )
    connection.execute(
        "insert into entity_history values ('feature', 1, 'active', 'code_review', "
        "'2026-01-01T01:00:00Z')"
    )

    try:
        history = dashboard.load_history_intervals(connection)
        phase_seconds, excluded = dashboard.analyze_work_sessions(
            connection, history, {"feature": {"F01": 1}}
        )
        assert excluded == []
        assert phase_seconds == {("feature", "active"): 3600.0}
    finally:
        connection.close()


@pytest.mark.unit
def test_work_session_boundaries_are_not_double_counted(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute(
        "create table work_sessions "
        "(entity_type text, entity_key text, started_at text, ended_at text, outcome text)"
    )
    connection.executemany(
        "insert into work_sessions values (?, ?, ?, ?, ?)",
        [
            ("feature", "F01", "2026-01-01T00:30:00Z", "2026-01-01T01:30:00Z", "pass"),
        ],
    )
    history = {
        ("feature", 1): (
            [
                datetime.datetime(2026, 1, 1, 0, tzinfo=datetime.UTC),
                datetime.datetime(2026, 1, 1, 1, tzinfo=datetime.UTC),
                datetime.datetime(2026, 1, 1, 2, tzinfo=datetime.UTC),
            ],
            ["active", "code_review", "active"],
        )
    }

    try:
        phase_seconds, excluded = dashboard.analyze_work_sessions(
            connection, history, {"feature": {"F01": 1}}
        )
        assert excluded == []
        assert phase_seconds == {
            ("feature", "active"): 1800.0,
            ("feature", "code_review"): 1800.0,
        }
    finally:
        connection.close()


@pytest.mark.unit
@pytest.mark.parametrize("ended_at", ["2026-01-01T00:00:00Z", "2026-01-01T01:00:00Z"])
def test_non_positive_closed_work_session_fails_loudly(
    dashboard: Any, ended_at: str
) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute(
        "create table work_sessions "
        "(entity_type text, entity_key text, started_at text, ended_at text, outcome text)"
    )
    connection.execute(
        "insert into work_sessions values (?, ?, ?, ?, ?)",
        ("feature", "F01", "2026-01-01T01:00:00Z", ended_at, "fail"),
    )

    try:
        with pytest.raises(ValueError, match="must end after it starts"):
            dashboard.analyze_work_sessions(connection, {}, {"feature": {"F01": 1}})
    finally:
        connection.close()


@pytest.mark.unit
def test_duration_filter_exercises_iqr_and_twelve_hour_boundary(dashboard: Any) -> None:
    normal_sample_count = 7
    outlier_entity_id = normal_sample_count + 1
    normal_duration_hours = 1
    feature_outlier_duration_hours = 10

    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.executescript("""
        create table work_sessions (
            entity_type text, entity_key text, started_at text,
            ended_at text, outcome text
        );
        """)
    history: dict[tuple[str, int], tuple[list[datetime.datetime], list[str]]] = {}
    key_maps: dict[str, dict[str, int]] = {"feature": {}, "task": {}}
    base = datetime.datetime(2026, 1, 1, tzinfo=datetime.UTC)
    for entity_id in range(1, outlier_entity_id + 1):
        started = base + datetime.timedelta(days=entity_id)
        duration = datetime.timedelta(
            hours=(
                normal_duration_hours
                if entity_id <= normal_sample_count
                else feature_outlier_duration_hours
            )
        )
        key = f"F{entity_id:02d}"
        key_maps["feature"][key] = entity_id
        history["feature", entity_id] = (
            [started, started + duration],
            ["active", "completed"],
        )
        connection.execute(
            "insert into work_sessions values (?, ?, ?, ?, ?)",
            (
                "feature",
                key,
                started.isoformat(),
                (started + duration).isoformat(),
                "pass",
            ),
        )
    task_hours = {20: 1, 21: 12, 22: 12, 23: 13, 24: 14}
    for entity_id in range(20, 25):
        started = base + datetime.timedelta(days=entity_id)
        duration = datetime.timedelta(
            hours=task_hours[entity_id], seconds=1 if entity_id == 22 else 0
        )
        key = f"T{entity_id}"
        key_maps["task"][key] = entity_id
        history["task", entity_id] = (
            [started, started + duration],
            ["active", "completed"],
        )
        connection.execute(
            "insert into work_sessions values (?, ?, ?, ?, ?)",
            (
                "task",
                key,
                started.isoformat(),
                (started + duration).isoformat(),
                "pass",
            ),
        )

    try:
        phase_seconds, excluded_sessions = dashboard.analyze_work_sessions(
            connection, history, key_maps
        )
        (
            wallclock,
            excluded_count,
            _,
            excluded_intervals,
        ) = dashboard.analyze_wallclock_status_time(connection, history)
        assert dashboard.iqr_bounds(
            [dashboard.SECONDS_PER_HOUR * normal_duration_hours] * normal_sample_count
            + [dashboard.SECONDS_PER_HOUR * feature_outlier_duration_hours]
        ) == pytest.approx(dashboard.SECONDS_PER_HOUR * normal_duration_hours)
        assert dashboard.iqr_bounds([]) is None
        assert dashboard.iqr_bounds([1.0]) is None
        assert dashboard.iqr_bounds([1.0, 2.0]) is None
        assert dashboard.iqr_bounds([1.0, 2.0, 3.0]) is None
        assert dashboard.iqr_bounds([1.0, 1.0, 1.0, 1.0]) == pytest.approx(1.0)
        assert (
            phase_seconds["feature", "active"]
            == normal_sample_count * dashboard.SECONDS_PER_HOUR
        )
        assert phase_seconds["task", "active"] == 13 * dashboard.SECONDS_PER_HOUR
        assert {item["entity_key"] for item in excluded_sessions} == {
            f"F{outlier_entity_id:02d}",
            "T22",
            "T23",
            "T24",
        }
        assert wallclock["feature", "active"]["total_hours"] == 7.0
        assert wallclock["task", "active"]["total_hours"] == 13.0
        assert excluded_count == 4
        assert {item["entity_id"] for item in excluded_intervals} == {
            outlier_entity_id,
            22,
            23,
            24,
        }
    finally:
        connection.close()
