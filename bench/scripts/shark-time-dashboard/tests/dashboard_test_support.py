"""Shared fixtures for dashboard unit tests."""

import importlib.util
import sqlite3
from collections.abc import Generator
from pathlib import Path
from typing import Any

import pytest

DASHBOARD_SCHEMA = """
create table features (id integer primary key, key text);
create table tasks (id integer primary key, key text);
create table epics (id integer primary key, key text);
create table tech_debts (id integer primary key, key text);
create table change_cards (id integer primary key, key text);
create table ideas (id integer primary key, key text);
create table questions (id integer primary key, key text);
create table sprints (id integer primary key, key text);
create table entity_history (
    entity_type text, entity_id integer, from_status text,
    to_status text, changed_at text, notes text
);
create table entity_relationships (
    relationship_type text, from_entity_type text,
    to_entity_type text, to_entity_id integer
);
create table bugs (
    id integer primary key, key text,
    linked_entity_type text, linked_entity_key text
);
create table work_sessions (
    entity_type text, entity_key text, started_at text,
    ended_at text, outcome text
);
"""


def create_dashboard_schema(connection: sqlite3.Connection) -> None:
    """Create the common dashboard fixture schema without fixture data."""
    connection.executescript(DASHBOARD_SCHEMA)


@pytest.fixture
def dashboard(monkeypatch: pytest.MonkeyPatch) -> Any:
    path = (
        Path(__file__).resolve().parents[4]
        / "bench"
        / "scripts"
        / "shark-time-dashboard"
        / "generate_dashboard.py"
    )
    monkeypatch.syspath_prepend(str(path.parent))
    spec = importlib.util.spec_from_file_location("shark_time_dashboard", path)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


@pytest.fixture
def dashboard_db() -> Generator[sqlite3.Connection, None, None]:
    connection = sqlite3.connect(":memory:")
    connection.row_factory = sqlite3.Row
    create_dashboard_schema(connection)
    connection.executemany(
        "insert into features(id, key) values (?, ?)", [(1, "F01"), (2, "F02")]
    )
    connection.executemany(
        "insert into entity_history values (?, ?, ?, ?, ?, ?)",
        [
            (
                "feature",
                1,
                "draft",
                "active",
                "2026-01-01T00:00:00+00:00",
                None,
            ),
            (
                "feature",
                1,
                "active",
                "code_review",
                "2026-01-01T01:00:00+00:00",
                None,
            ),
            (
                "feature",
                1,
                "code_review",
                "active",
                "2026-01-01T02:00:00+00:00",
                "code review defect",
            ),
            (
                "feature",
                1,
                "active",
                "completed",
                "2026-01-01T03:00:00+00:00",
                None,
            ),
            (
                "feature",
                2,
                "draft",
                "active",
                "2026-01-01T00:00:00+00:00",
                None,
            ),
            (
                "feature",
                2,
                "active",
                "completed",
                "2026-01-06T00:00:00+00:00",
                None,
            ),
        ],
    )
    connection.executemany(
        "insert into work_sessions values (?, ?, ?, ?, ?)",
        [
            (
                "feature",
                "F01",
                "2026-01-01T00:30:00Z",
                "2026-01-01T01:00:00Z",
                "pass",
            ),
            (
                "feature",
                "F02",
                "2026-01-01T00:00:00Z",
                "2026-01-02T00:00:00Z",
                "stale",
            ),
        ],
    )
    connection.execute(
        "insert into entity_relationships values (?, ?, ?, ?)",
        ("spawned_from", "tech_debt", "feature", 1),
    )
    connection.execute(
        "insert into bugs values (?, ?, ?, ?)", (1, "B01", "feature", "F01")
    )
    connection.commit()
    try:
        yield connection
    finally:
        connection.close()
