"""Calculate rework classifications and feature-quality correlation."""

from __future__ import annotations

import collections
import re
import sqlite3

from dashboard_types import QualityRow

FEATURE_REJECTION_TRANSITIONS = {
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
}
TASK_REWORK_TRANSITIONS = {
    (
        "task",
        "completed",
        "development",
    ): "Task reverted after completion (UAT/review fail)",
    ("task", "development", "blocked"): "Task blocked mid-development",
}
REWORK_TRANSITIONS = {**FEATURE_REJECTION_TRANSITIONS, **TASK_REWORK_TRANSITIONS}
BUG_FEATURE_RELATIONSHIP_TYPES = ("linked_to", "related_to")


def analyze_rework(
    conn: sqlite3.Connection,
) -> tuple[collections.Counter[str], collections.Counter[str], int]:
    """Count configured rework transitions and classify their notes."""
    categories = {
        "Release/verification gate fail": r"release outcome: fail|verification gate",
        "Systematic/defect-class sweep": r"defect class|systematic",
        "Scope/integration contract issue": (
            r"scope|Integration Contract|cross-epic|X-0\d|I-0\d"
        ),
        "UAT/red-team failure": r"UAT|red-team|red team",
        "Schema/validation gap": r"schema|validat",
        "Task structure/DAG issue": r"depend|DAG|task-body|task_generation",
        "Code review defect": r"code review|code_review",
        "Unguarded error/crash": (
            r"KeyError|ZeroDivisionError|TypeError|ValueError|crash|unguarded"
        ),
        "Missing test coverage": r"test coverage|no automated test|untested|no test",
    }
    counts: collections.Counter[str] = collections.Counter()
    notes: list[str] = []
    for (entity_type, from_status, to_status), label in REWORK_TRANSITIONS.items():
        rows = conn.execute(
            "select notes from entity_history where entity_type=? "
            "and from_status=? and to_status=?",
            (entity_type, from_status, to_status),
        ).fetchall()
        counts[label] = len(rows)
        notes.extend(row["notes"] for row in rows if row["notes"])
    category_counts: collections.Counter[str] = collections.Counter()
    for note in notes:
        for category, pattern in categories.items():
            if re.search(pattern, note, re.IGNORECASE):
                category_counts[category] += 1
    return counts, category_counts, len(notes)


def _pearson(xs: list[int], ys: list[int]) -> float | None:
    if not xs or not ys:
        return None
    mean_x, mean_y = sum(xs) / len(xs), sum(ys) / len(ys)
    covariance = sum((x - mean_x) * (y - mean_y) for x, y in zip(xs, ys, strict=True))
    scale_x = sum((x - mean_x) ** 2 for x in xs) ** 0.5
    scale_y = sum((y - mean_y) ** 2 for y in ys) ** 0.5
    return covariance / (scale_x * scale_y) if scale_x and scale_y else None


def _feature_rejection_counts(
    conn: sqlite3.Connection, rejection_pairs: set[tuple[str, str]]
) -> collections.defaultdict[int, int]:
    counts: collections.defaultdict[int, int] = collections.defaultdict(int)
    for row in conn.execute(
        "select entity_id, from_status, to_status from entity_history "
        "where entity_type='feature' order by entity_id, rowid"
    ):
        if (row["from_status"], row["to_status"]) in rejection_pairs:
            counts[row["entity_id"]] += 1
    return counts


def _spawned_tech_debt_counts(
    conn: sqlite3.Connection,
) -> collections.defaultdict[int, int]:
    counts: collections.defaultdict[int, int] = collections.defaultdict(int)
    for row in conn.execute(
        "select to_entity_id from entity_relationships "
        "where relationship_type='spawned_from' and from_entity_type='tech_debt' "
        "and to_entity_type='feature' order by to_entity_id"
    ):
        counts[row["to_entity_id"]] += 1
    return counts


def _linked_bug_counts(
    conn: sqlite3.Connection, feature_keys: dict[int, str]
) -> collections.defaultdict[int, int]:
    key_to_feature_id = {key: entity_id for entity_id, key in feature_keys.items()}
    linked_pairs: set[tuple[int, int]] = set()
    relationship_columns = {
        row[1] for row in conn.execute("pragma table_info(entity_relationships)")
    }
    if {"from_entity_id", "to_entity_id"} <= relationship_columns:
        for row in conn.execute(
            "select from_entity_id, to_entity_id from entity_relationships "
            "where relationship_type in (?, ?) and from_entity_type='bug' "
            "and to_entity_type='feature'",
            BUG_FEATURE_RELATIONSHIP_TYPES,
        ):
            linked_pairs.add((row["from_entity_id"], row["to_entity_id"]))
    counts: collections.defaultdict[int, int] = collections.defaultdict(int)
    for row in conn.execute(
        "select id, linked_entity_key from bugs where linked_entity_type='feature' "
        "order by linked_entity_key"
    ):
        feature_id = key_to_feature_id.get(row["linked_entity_key"])
        if feature_id is not None:
            linked_pairs.add((row["id"], feature_id))
    for _, feature_id in linked_pairs:
        counts[feature_id] += 1
    return counts


def _quality_rows(
    feature_keys: dict[int, str],
    rejections: collections.defaultdict[int, int],
    tech_debt_spawned: collections.defaultdict[int, int],
    bugs: collections.defaultdict[int, int],
) -> list[QualityRow]:
    rows: list[QualityRow] = [
        {
            "feature": key,
            "rejections": rejections.get(feature_id, 0),
            "tech_debt_spawned": tech_debt_spawned.get(feature_id, 0),
            "bugs": bugs.get(feature_id, 0),
        }
        for feature_id, key in feature_keys.items()
    ]
    rows.sort(key=lambda row: (-int(row["rejections"]), str(row["feature"])))
    return rows


def analyze_quality_correlation(
    conn: sqlite3.Connection,
) -> tuple[list[QualityRow], float | None]:
    """Correlate feature rejection counts with linked downstream issues."""
    feature_keys = {
        row["id"]: row["key"]
        for row in conn.execute("select id, key from features order by id")
    }
    rejection_pairs = {
        (from_status, to_status)
        for _, from_status, to_status in FEATURE_REJECTION_TRANSITIONS
    }
    rejections = _feature_rejection_counts(conn, rejection_pairs)
    tech_debt_spawned = _spawned_tech_debt_counts(conn)
    bugs = _linked_bug_counts(conn, feature_keys)
    rows = _quality_rows(feature_keys, rejections, tech_debt_spawned, bugs)
    return rows, _pearson(
        [int(row["rejections"]) for row in rows],
        [int(row["tech_debt_spawned"]) + int(row["bugs"]) for row in rows],
    )
