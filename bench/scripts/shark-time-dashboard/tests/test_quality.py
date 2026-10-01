"""Rework and quality-correlation coverage for the generated dashboard."""

import sqlite3
from collections import Counter
from typing import Any

import pytest
from dashboard_test_support import create_dashboard_schema


@pytest.mark.unit
def test_rework_counts_cover_every_configured_transition(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    expected_transitions = {
        ("feature", "task_review", "task_generation"): (
            "Task review kickback",
            "schema validation gap",
        ),
        ("feature", "test_planning", "task_generation"): (
            "Test plan sent back for task re-gen",
            "missing test coverage",
        ),
        ("feature", "code_review", "active"): (
            "Code review kickback",
            "code review defect; systematic defect class sweep",
        ),
        ("feature", "qa", "active"): ("QA kickback", "UAT red-team failure"),
        ("feature", "approval", "code_review"): (
            "Approval kickback to code review",
            "scope integration contract issue",
        ),
        ("feature", "completed", "approval"): (
            "Reopened after completion",
            "release outcome: fail",
        ),
        ("task", "completed", "development"): (
            "Task reverted after completion (UAT/review fail)",
            "task structure dependency issue",
        ),
        ("task", "development", "blocked"): (
            "Task blocked mid-development",
            "unguarded error crash",
        ),
    }
    rows = [
        (*transition, note) for transition, (_, note) in expected_transitions.items()
    ]
    rows.append(("feature", "active", "completed", "not a configured transition"))
    connection.executemany(
        "insert into entity_history (entity_type, from_status, to_status, notes) "
        "values (?, ?, ?, ?)",
        rows,
    )

    try:
        counts, categories, note_count = dashboard.analyze_rework(connection)
        assert counts == Counter(
            {label: 1 for label, _ in expected_transitions.values()}
        )
        assert categories == Counter(
            {
                "Schema/validation gap": 1,
                "Missing test coverage": 1,
                "Code review defect": 1,
                "Systematic/defect-class sweep": 1,
                "UAT/red-team failure": 1,
                "Scope/integration contract issue": 1,
                "Release/verification gate fail": 1,
                "Task structure/DAG issue": 1,
                "Unguarded error/crash": 1,
            }
        )
        assert note_count == len(expected_transitions)
    finally:
        connection.close()


@pytest.mark.unit
def test_quality_correlation_is_nonperfect_and_constant_series_is_unavailable(
    dashboard: Any,
) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    connection.executemany(
        "insert into features values (?, ?)", [(1, "F01"), (2, "F02"), (3, "F03")]
    )
    connection.executemany(
        "insert into entity_history (entity_type, entity_id, from_status, to_status) "
        "values ('feature', ?, ?, ?)",
        [
            (2, "code_review", "active"),
            (3, "code_review", "active"),
            (3, "qa", "active"),
        ],
    )
    connection.execute(
        "insert into bugs (linked_entity_type, linked_entity_key) "
        "values ('feature', 'F02')"
    )

    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert rows == [
            {"feature": "F03", "rejections": 2, "tech_debt_spawned": 0, "bugs": 0},
            {"feature": "F02", "rejections": 1, "tech_debt_spawned": 0, "bugs": 1},
            {"feature": "F01", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
        ]
        assert correlation == pytest.approx(0.0)
    finally:
        connection.close()

    constant = sqlite3.connect(":memory:")
    constant.row_factory = sqlite3.Row
    create_dashboard_schema(constant)
    constant.executemany("insert into features values (?, ?)", [(2, "F02"), (1, "F01")])
    try:
        constant_rows, constant_correlation = dashboard.analyze_quality_correlation(
            constant
        )
        assert constant_rows == [
            {"feature": "F01", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
            {"feature": "F02", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
        ]
        assert constant_correlation is None
    finally:
        constant.close()


@pytest.mark.unit
def test_quality_correlation_preserves_multiplicity_and_filters_unrelated_records(
    dashboard: Any,
) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    connection.executemany(
        "insert into features values (?, ?)", [(1, "F01"), (2, "F02")]
    )
    connection.executemany(
        "insert into entity_history (entity_type, entity_id, from_status, to_status) "
        "values (?, ?, ?, ?)",
        [
            ("feature", 1, "code_review", "active"),
            ("feature", 1, "code_review", "active"),
            ("feature", 2, "active", "completed"),
            ("task", 1, "code_review", "active"),
        ],
    )
    connection.executemany(
        "insert into entity_relationships values (?, ?, ?, ?)",
        [
            ("spawned_from", "tech_debt", "feature", 1),
            ("spawned_from", "tech_debt", "feature", 1),
            ("spawned_from", "tech_debt", "task", 1),
        ],
    )
    connection.executemany(
        "insert into bugs (key, linked_entity_type, linked_entity_key) values (?, ?, ?)",
        [
            ("B01", "feature", "F01"),
            ("B02", "feature", "F01"),
            ("B03", "task", "F01"),
            ("B04", "feature", "unknown"),
        ],
    )

    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert rows == [
            {"feature": "F01", "rejections": 2, "tech_debt_spawned": 2, "bugs": 2},
            {"feature": "F02", "rejections": 0, "tech_debt_spawned": 0, "bugs": 0},
        ]
        assert correlation == pytest.approx(1.0)
    finally:
        connection.close()


@pytest.mark.unit
def test_quality_correlation_preserves_negative_relationships(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    connection.executemany(
        "insert into features values (?, ?)",
        [(1, "F01"), (2, "F02"), (3, "F03")],
    )
    connection.executemany(
        "insert into entity_history (entity_type, entity_id, from_status, to_status) "
        "values ('feature', ?, 'code_review', 'active')",
        [(2,), (3,), (3,)],
    )
    connection.executemany(
        "insert into bugs (key, linked_entity_type, linked_entity_key) values (?, 'feature', ?)",
        [("B01", "F01"), ("B02", "F01"), ("B03", "F02")],
    )

    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert rows == [
            {"feature": "F03", "rejections": 2, "tech_debt_spawned": 0, "bugs": 0},
            {"feature": "F02", "rejections": 1, "tech_debt_spawned": 0, "bugs": 1},
            {"feature": "F01", "rejections": 0, "tech_debt_spawned": 0, "bugs": 2},
        ]
        assert correlation == pytest.approx(-1.0)
    finally:
        connection.close()


@pytest.mark.unit
def test_quality_correlation_reads_normalized_bug_relationships(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.executescript("""
        create table features (id integer primary key, key text);
        create table entity_relationships (
            from_entity_type text, from_entity_id integer,
            to_entity_type text, to_entity_id integer, relationship_type text
        );
        create table bugs (id integer primary key, linked_entity_type text, linked_entity_key text);
        create table entity_history (
            entity_type text, entity_id integer, from_status text, to_status text, notes text
        );
        """)
    connection.execute("insert into features values (1, 'F01')")
    connection.executemany(
        "insert into entity_relationships values ('bug', ?, 'feature', 1, ?)",
        [(101, "linked_to"), (102, "related_to")],
    )

    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert rows == [
            {"feature": "F01", "rejections": 0, "tech_debt_spawned": 0, "bugs": 2}
        ]
        assert correlation is None
    finally:
        connection.close()


@pytest.mark.unit
def test_quality_correlation_counts_every_feature_rejection_transition(
    dashboard: Any,
) -> None:
    transitions = [
        ("task_review", "task_generation"),
        ("test_planning", "task_generation"),
        ("code_review", "active"),
        ("qa", "active"),
        ("approval", "code_review"),
        ("completed", "approval"),
    ]
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    connection.executemany(
        "insert into features values (?, ?)",
        [(index, f"F{index:02d}") for index in range(1, len(transitions) + 2)],
    )
    connection.executemany(
        "insert into entity_history (entity_type, entity_id, from_status, to_status) "
        "values ('feature', ?, ?, ?)",
        [
            (index, from_status, to_status)
            for index, (from_status, to_status) in enumerate(transitions, 1)
        ],
    )
    connection.execute(
        "insert into entity_history (entity_type, entity_id, from_status, to_status) "
        "values ('feature', ?, 'active', 'completed')",
        (len(transitions) + 1,),
    )

    try:
        rows, correlation = dashboard.analyze_quality_correlation(connection)
        assert [row["rejections"] for row in rows] == [1] * len(transitions) + [0]
        assert rows[-1]["feature"] == f"F{len(transitions) + 1:02d}"
        assert correlation is None
    finally:
        connection.close()


@pytest.mark.unit
def test_missing_required_dashboard_table_fails_loudly(dashboard: Any) -> None:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    connection.execute("create table features (id integer primary key, key text)")
    try:
        with pytest.raises(RuntimeError, match="missing required tables"):
            dashboard.load_key_maps(connection)
    finally:
        connection.close()
