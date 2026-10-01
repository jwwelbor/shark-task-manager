"""Load Shark history and calculate time and rework metrics."""

from __future__ import annotations

import bisect
import collections
import logging
import re
import sqlite3
import statistics
from datetime import UTC, datetime, timedelta

from dashboard_types import (
    DURATION_DECIMAL_PLACES,
    SECONDS_PER_HOUR,
    ExcludedInterval,
    ExcludedSession,
    WallclockSummary,
)

ABSOLUTE_CAP_SECONDS = 12 * SECONDS_PER_HOUR
IQR_MULTIPLIER = 3.0
IQR_MIN_SAMPLE_SIZE = 4
IQR_QUARTILE_COUNT = 4
ENTITY_QUERIES = (
    ("feature", "features", "select id, key from features order by id"),
    ("task", "tasks", "select id, key from tasks order by id"),
    ("epic", "epics", "select id, key from epics order by id"),
    ("bug", "bugs", "select id, key from bugs order by id"),
    ("tech_debt", "tech_debts", "select id, key from tech_debts order by id"),
    ("change", "change_cards", "select id, key from change_cards order by id"),
    ("idea", "ideas", "select id, key from ideas order by id"),
    ("question", "questions", "select id, key from questions order by id"),
    ("sprint", "sprints", "select id, key from sprints order by id"),
)
REQUIRED_TABLES = tuple(table for _, table, _ in ENTITY_QUERIES) + (
    "entity_history",
    "entity_relationships",
    "work_sessions",
)
GO_TIMESTAMP_RE = re.compile(
    r"^(?P<base>\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?)"
    r"(?:\s+(?P<offset>[+-]\d{4})\s+\S+)?$"
)
TIMESTAMP_PREFIX_RE = re.compile(r"^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?")

HistoryEntry = (
    tuple[list[datetime], list[str | None]]
    | tuple[list[datetime], list[str | None], str | None]
)
History = dict[tuple[str, int], HistoryEntry]
Interval = tuple[float, datetime, int]
logger = logging.getLogger(__name__)


def parse_ts(ts: str | None) -> datetime | None:
    """Parse a Shark timestamp and normalize it to UTC.

    Offsetless database timestamps are interpreted as UTC. Non-empty malformed
    values raise so a dashboard cannot silently publish incomplete metrics.
    """
    if ts is None or ts == "":
        return None
    if not isinstance(ts, str):
        raise ValueError(  # noqa: TRY004 - malformed timestamps share one failure contract
            f"Unparseable Shark timestamp: {ts!r}"
        )

    value = ts.strip()
    if not value:
        logger.error("Unparseable Shark timestamp", extra={"timestamp": repr(ts)})
        raise ValueError(f"Unparseable Shark timestamp: {ts!r}")
    value = re.sub(r"\s+m=[+-]\d+(?:\.\d+)?$", "", value)
    if not value:
        logger.error("Unparseable Shark timestamp", extra={"timestamp": repr(ts)})
        raise ValueError(f"Unparseable Shark timestamp: {ts!r}")
    if not TIMESTAMP_PREFIX_RE.match(value):
        logger.error("Unparseable Shark timestamp", extra={"timestamp": repr(ts)})
        raise ValueError(f"Unparseable Shark timestamp: {ts!r}")
    match = GO_TIMESTAMP_RE.fullmatch(value)
    try:
        if match and match.group("offset"):
            offset = match.group("offset")
            parsed = datetime.fromisoformat(
                f"{match.group('base')}{offset[:3]}:{offset[3:]}"
            )
        else:
            parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError as exc:
        logger.error("Unparseable Shark timestamp", extra={"timestamp": repr(ts)})
        raise ValueError(f"Unparseable Shark timestamp: {ts!r}") from exc

    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)


def iqr_bounds(values: list[float]) -> float | None:
    """Return the conservative upper IQR threshold for at least four values."""
    if len(values) < IQR_MIN_SAMPLE_SIZE:
        return None
    quartiles = statistics.quantiles(values, n=IQR_QUARTILE_COUNT)
    return quartiles[2] + IQR_MULTIPLIER * (quartiles[2] - quartiles[0])


def load_key_maps(conn: sqlite3.Connection) -> dict[str, dict[str, int]]:
    """Load supported entity keys into direct key-to-ID lookup maps."""
    available = {
        row[0]
        for row in conn.execute("select name from sqlite_master where type='table'")
    }
    missing = sorted(set(REQUIRED_TABLES) - available)
    if missing:
        raise RuntimeError(
            "Dashboard database is missing required tables: " + ", ".join(missing)
        )
    maps: dict[str, dict[str, int]] = {}
    for entity_type, _, query in ENTITY_QUERIES:
        maps[entity_type] = {row["key"]: row["id"] for row in conn.execute(query)}
    return maps


def load_history_intervals(conn: sqlite3.Connection) -> History:
    """Load entity-history timestamps grouped for binary-search status lookup."""
    raw: collections.defaultdict[
        tuple[str, int], list[tuple[datetime, int, str | None, str | None]]
    ] = collections.defaultdict(list)
    columns = {row[1] for row in conn.execute("pragma table_info(entity_history)")}
    from_status_column = "from_status" if "from_status" in columns else "null"
    for row in conn.execute(
        "select rowid as history_id, entity_type, entity_id, "
        f"{from_status_column} as from_status, to_status, changed_at "
        "from entity_history"
    ):
        timestamp = parse_ts(row["changed_at"])
        if timestamp is None:
            continue
        raw[(row["entity_type"], row["entity_id"])].append(
            (
                timestamp,
                row["history_id"],
                row["to_status"],
                row["from_status"],
            )
        )
    history: History = {}
    for key, raw_events in raw.items():
        events = sorted(raw_events, key=lambda event: (event[0], event[1]))
        history[key] = (
            [timestamp for timestamp, _, _, _ in events],
            [status for _, _, status, _ in events],
            events[0][3],
        )
    return history


def status_at(
    history: History,
    entity_type: str,
    entity_id: int,
    timestamp: datetime | None,
) -> str | None:
    """Return the status in force at a timestamp, or ``None`` before history."""
    events = history.get((entity_type, entity_id))
    if not events or timestamp is None:
        return None
    timestamps, statuses, *initial_status = events
    index = bisect.bisect_right(timestamps, timestamp) - 1
    if index < 0:
        return initial_status[0] if initial_status else None
    return statuses[index]


def _read_closed_sessions(
    conn: sqlite3.Connection,
) -> tuple[
    list[tuple[str, str | None, int | None, datetime, float, str | None]],
    dict[str, list[float]],
]:
    sessions = []
    durations: collections.defaultdict[str, list[float]] = collections.defaultdict(list)
    columns = {row[1] for row in conn.execute("pragma table_info(work_sessions)")}
    task_id_column = "task_id" if "task_id" in columns else "null"
    for row in conn.execute(
        "select entity_type, entity_key, "
        f"{task_id_column} as task_id, started_at, ended_at, outcome "
        "from work_sessions where ended_at is not null "
        "order by entity_type, entity_key, started_at"
    ):
        started, ended = parse_ts(row["started_at"]), parse_ts(row["ended_at"])
        if started is None or ended is None:
            continue
        duration = (ended - started).total_seconds()
        if duration <= 0:
            raise ValueError(
                "Closed work-session must end after it starts: "
                f"{row['entity_type']}:{row['entity_key']}"
            )
        session = (
            row["entity_type"],
            row["entity_key"],
            row["task_id"],
            started,
            duration,
            row["outcome"],
        )
        sessions.append(session)
        durations[row["entity_type"]].append(duration)
    return sessions, durations


def _duration_cutoff(values: list[float]) -> float:
    """Return an IQR cutoff bounded by the absolute twelve-hour cap."""
    return min(iqr_bounds(values) or ABSOLUTE_CAP_SECONDS, ABSOLUTE_CAP_SECONDS)


def _attribute_session_time(
    phase_seconds: collections.defaultdict[tuple[str, str], float],
    history: History,
    entity_type: str,
    entity_id: int | None,
    started: datetime,
    duration: float,
) -> None:
    ended = started + timedelta(seconds=duration)
    phase = (
        status_at(history, entity_type, entity_id, started)
        if entity_id is not None
        else None
    ) or "unknown"
    cursor = started
    if entity_id is not None and (entity_type, entity_id) in history:
        timestamps, statuses, *_ = history[(entity_type, entity_id)]
        first_after_start = bisect.bisect_right(timestamps, started)
        for index in range(first_after_start, len(timestamps)):
            boundary = timestamps[index]
            if boundary >= ended:
                break
            phase_seconds[(entity_type, phase)] += (boundary - cursor).total_seconds()
            cursor = boundary
            phase = statuses[index] or "unknown"
    phase_seconds[(entity_type, phase)] += (ended - cursor).total_seconds()


def analyze_work_sessions(
    conn: sqlite3.Connection,
    history: History,
    key_maps: dict[str, dict[str, int]],
) -> tuple[dict[tuple[str, str], float], list[ExcludedSession]]:
    """Attribute kept work-session time to workflow phases and list exclusions."""
    sessions, durations = _read_closed_sessions(conn)
    cutoffs = {
        entity_type: _duration_cutoff(values)
        for entity_type, values in durations.items()
    }
    phase_seconds: collections.defaultdict[tuple[str, str], float] = (
        collections.defaultdict(float)
    )
    excluded: list[ExcludedSession] = []
    for entity_type, entity_key, task_id, started, duration, outcome in sessions:
        if duration > cutoffs.get(entity_type, ABSOLUTE_CAP_SECONDS):
            display_key = entity_key or (
                f"task:{task_id}" if task_id is not None else "unknown"
            )
            excluded.append(
                {
                    "entity_type": entity_type,
                    "entity_key": display_key,
                    "started_at": started.isoformat(),
                    "duration_seconds": duration,
                    "duration_hours": round(
                        duration / SECONDS_PER_HOUR, DURATION_DECIMAL_PLACES
                    ),
                    "outcome": outcome,
                }
            )
            continue
        entity_id = (
            key_maps.get(entity_type, {}).get(entity_key)
            if entity_key is not None
            else None
        )
        if entity_id is None and entity_type == "task":
            entity_id = task_id
        _attribute_session_time(
            phase_seconds, history, entity_type, entity_id, started, duration
        )
    excluded.sort(
        key=lambda item: (
            -float(item["duration_seconds"]),
            str(item["entity_type"]),
            str(item["entity_key"]),
        )
    )
    return dict(phase_seconds), excluded


def _build_intervals(
    history: History,
) -> dict[tuple[str, str | None], list[Interval]]:
    intervals: collections.defaultdict[tuple[str, str | None], list[Interval]] = (
        collections.defaultdict(list)
    )
    for (entity_type, entity_id), (timestamps, statuses, *_) in history.items():
        for index in range(len(timestamps) - 1):
            started, ended = timestamps[index : index + 2]
            status = statuses[index]
            duration = (ended - started).total_seconds()
            if duration > 0 and status is not None:
                intervals[(entity_type, status)].append((duration, started, entity_id))
    return dict(intervals)


def _summarize_intervals(
    intervals: dict[tuple[str, str | None], list[Interval]],
) -> tuple[dict[tuple[str, str], WallclockSummary], int, float, list[ExcludedInterval]]:
    result: dict[tuple[str, str], WallclockSummary] = {}
    excluded: list[ExcludedInterval] = []
    excluded_hours = 0.0
    for (entity_type, status), records in intervals.items():
        cutoff = _duration_cutoff([record[0] for record in records])
        kept = [record[0] for record in records if record[0] <= cutoff]
        for duration, started, entity_id in records:
            if duration > cutoff:
                excluded.append(
                    {
                        "entity_type": entity_type,
                        "entity_id": entity_id,
                        "status": status,
                        "started_at": started.isoformat(),
                        "duration_hours": round(
                            duration / SECONDS_PER_HOUR, DURATION_DECIMAL_PLACES
                        ),
                    }
                )
                excluded_hours += duration / SECONDS_PER_HOUR
        if kept:
            total_hours = sum(kept) / SECONDS_PER_HOUR
            result[(entity_type, status or "unknown")] = {
                "total_hours": total_hours,
                "n": len(kept),
                "avg_hours": total_hours / len(kept),
            }
    excluded.sort(
        key=lambda item: (
            -float(item["duration_hours"]),
            str(item["entity_type"]),
            int(item["entity_id"]),
            str(item["status"]),
        )
    )
    return result, len(excluded), excluded_hours, excluded


def analyze_wallclock_status_time(
    conn: sqlite3.Connection,
    history: History | None = None,
) -> tuple[dict[tuple[str, str], WallclockSummary], int, float, list[ExcludedInterval]]:
    """Calculate capped time-in-status metrics and interval exclusions."""
    if history is None:
        history = load_history_intervals(conn)
    return _summarize_intervals(_build_intervals(history))
