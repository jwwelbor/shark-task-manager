"""Typed contracts shared by Shark dashboard analysis and rendering."""

from __future__ import annotations

import collections
from dataclasses import dataclass
from typing import TypedDict

SECONDS_PER_HOUR = 3600
DURATION_DECIMAL_PLACES = 1


class ExcludedSession(TypedDict):
    entity_type: str
    entity_key: str
    started_at: str
    duration_seconds: float
    duration_hours: float
    outcome: str | None


class ExcludedInterval(TypedDict):
    entity_type: str
    entity_id: int
    status: str | None
    started_at: str
    duration_hours: float


class WallclockSummary(TypedDict):
    total_hours: float
    n: int
    avg_hours: float


class QualityRow(TypedDict):
    feature: str
    rejections: int
    tech_debt_spawned: int
    bugs: int


@dataclass(frozen=True)
class DashboardMetrics:
    """Complete analysis result passed from the CLI to the renderer."""

    phase_seconds: dict[tuple[str, str], float]
    wallclock: dict[tuple[str, str], WallclockSummary]
    excluded_count: int
    excluded_hours: float
    excluded_intervals: list[ExcludedInterval]
    rework_counts: collections.Counter[str]
    category_counts: collections.Counter[str]
    note_count: int
    quality_rows: list[QualityRow]
    correlation: float | None
    excluded_sessions: list[ExcludedSession]
