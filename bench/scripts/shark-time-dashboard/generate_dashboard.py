#!/usr/bin/env python3
"""Generate the Shark time-and-quality dashboard HTML report.

The report is a generated operator artifact with an external Chart.js CDN
dependency. Analysis and rendering live in sibling modules so each boundary
can be tested without invoking the CLI.
"""

from __future__ import annotations

import argparse
import contextlib
import logging
import os
import secrets
import sqlite3
from pathlib import Path

from dashboard_analysis import (
    analyze_wallclock_status_time,
    analyze_work_sessions,
    iqr_bounds,
    load_history_intervals,
    load_key_maps,
    parse_ts,
    status_at,
)
from dashboard_quality import (
    REWORK_TRANSITIONS,
    analyze_quality_correlation,
    analyze_rework,
)
from dashboard_render import render
from dashboard_types import SECONDS_PER_HOUR, DashboardMetrics

logger = logging.getLogger(__name__)
REPORT_FILE_MODE = 0o644  # Owner writable; group and other users may read reports.
TEMP_NAME_ATTEMPTS = 8
TEMP_TOKEN_BYTES = 8


def _argument_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser()
    parser.add_argument("--db", default="shark-tasks.db")
    parser.add_argument("--out", default=None)
    return parser


def _absolute_path(value: str) -> Path:
    """Make a path absolute without resolving its final symlink."""
    return Path(value).expanduser().absolute()


def _validate_output_target(out_path: Path) -> None:
    """Reject output targets that could redirect or corrupt another file."""
    if out_path.is_symlink():
        raise ValueError("--out must not be a symlink")
    if any(parent.is_symlink() for parent in out_path.parents):
        raise ValueError("--out must not be inside a symlinked directory")
    if out_path.exists():
        if not out_path.is_file():
            raise ValueError("--out must name a regular file or a new file path")
        if out_path.stat().st_nlink > 1:
            raise ValueError("--out must not be a hard link to another file")


def _validate_output_path(db_path: Path, out_path: Path) -> None:
    """Reject output targets that could overwrite the source database."""
    _validate_output_target(out_path)
    if out_path.exists() and out_path.samefile(db_path):
        raise ValueError("--out must not overwrite the --db SQLite source file")


def _write_text_safely(out_path: Path, content: str) -> None:
    """Atomically replace a report without truncating the previous version."""
    _validate_output_target(out_path)
    directory_flags = os.O_RDONLY
    if hasattr(os, "O_DIRECTORY"):
        directory_flags |= os.O_DIRECTORY
    if hasattr(os, "O_NOFOLLOW"):
        directory_flags |= os.O_NOFOLLOW
    directory_descriptor = os.open(out_path.parent, directory_flags)
    descriptor = None
    temporary_name = None
    try:
        for _ in range(TEMP_NAME_ATTEMPTS):
            candidate = f".{out_path.name}.{secrets.token_hex(TEMP_TOKEN_BYTES)}.tmp"
            flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL
            if hasattr(os, "O_NOFOLLOW"):
                flags |= os.O_NOFOLLOW
            try:
                descriptor = os.open(
                    candidate,
                    flags,
                    REPORT_FILE_MODE,
                    dir_fd=directory_descriptor,
                )
            except FileExistsError:
                continue
            temporary_name = candidate
            break
        if descriptor is None or temporary_name is None:
            raise FileExistsError("Could not create a unique dashboard temp file")
        with os.fdopen(descriptor, "w", encoding="utf-8") as output:
            descriptor = None
            os.fchmod(output.fileno(), REPORT_FILE_MODE)
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        os.replace(
            temporary_name,
            out_path.name,
            src_dir_fd=directory_descriptor,
            dst_dir_fd=directory_descriptor,
        )
    except OSError:
        logger.exception(
            "Failed to write dashboard report", extra={"path": str(out_path)}
        )
        raise
    finally:
        if descriptor is not None:
            os.close(descriptor)
        with contextlib.suppress(FileNotFoundError):
            if temporary_name is not None:
                os.unlink(temporary_name, dir_fd=directory_descriptor)
        os.close(directory_descriptor)


def _load_metrics(db_path: Path) -> DashboardMetrics:
    read_only_uri = f"{db_path.as_uri()}?mode=ro"
    with contextlib.closing(sqlite3.connect(read_only_uri, uri=True)) as connection:
        connection.row_factory = sqlite3.Row
        connection.execute("begin")
        key_maps = load_key_maps(connection)
        history = load_history_intervals(connection)
        phase_seconds, excluded_sessions = analyze_work_sessions(
            connection, history, key_maps
        )
        (
            wallclock,
            excluded_count,
            excluded_hours,
            excluded_intervals,
        ) = analyze_wallclock_status_time(connection, history)
        rework_counts, category_counts, note_count = analyze_rework(connection)
        quality_rows, correlation = analyze_quality_correlation(connection)
        connection.commit()
    return DashboardMetrics(
        phase_seconds=phase_seconds,
        wallclock=wallclock,
        excluded_count=excluded_count,
        excluded_hours=excluded_hours,
        excluded_intervals=excluded_intervals,
        rework_counts=rework_counts,
        category_counts=category_counts,
        note_count=note_count,
        quality_rows=quality_rows,
        correlation=correlation,
        excluded_sessions=excluded_sessions,
    )


def _log_metrics(metrics: DashboardMetrics) -> None:
    logger.info(
        "Kept session hours: %.1f",
        sum(metrics.phase_seconds.values()) / SECONDS_PER_HOUR,
    )
    logger.info(
        "Excluded sessions: %d (%.1fh)",
        len(metrics.excluded_sessions),
        sum(item["duration_seconds"] for item in metrics.excluded_sessions)
        / SECONDS_PER_HOUR,
    )
    logger.info(
        "Wall-clock outliers excluded: %d intervals (%.1fh)",
        metrics.excluded_count,
        metrics.excluded_hours,
    )
    logger.info("Rework events: %d", sum(metrics.rework_counts.values()))
    correlation_label = (
        "n/a" if metrics.correlation is None else f"{metrics.correlation:.3f}"
    )
    logger.info(
        "Quality correlation r=%s across %d features",
        correlation_label,
        len(metrics.quality_rows),
    )


def main() -> None:
    """Read the database, calculate metrics, and write the HTML report."""
    parser = _argument_parser()
    args = parser.parse_args()

    db_path = Path(args.db).resolve()
    out_path = (
        _absolute_path(args.out)
        if args.out
        else Path(__file__).parent / "dashboard.html"
    )
    if not db_path.is_file():
        parser.error(f"--db must name an existing SQLite file: {db_path}")
    try:
        _validate_output_path(db_path, out_path)
    except ValueError as exc:
        parser.error(str(exc))

    metrics = _load_metrics(db_path)
    _log_metrics(metrics)
    _write_text_safely(
        out_path,
        render(
            phase_seconds=metrics.phase_seconds,
            wallclock=metrics.wallclock,
            rework_counts=metrics.rework_counts,
            cat_counts=metrics.category_counts,
            n_notes=metrics.note_count,
            quality_rows=metrics.quality_rows,
            correlation=metrics.correlation,
            excluded=metrics.excluded_sessions,
            db_path=str(db_path),
            excluded_intervals=metrics.excluded_intervals,
        ),
    )
    logger.info("Dashboard written to %s", out_path)


__all__ = [
    "REWORK_TRANSITIONS",
    "analyze_quality_correlation",
    "iqr_bounds",
    "main",
    "parse_ts",
    "render",
    "status_at",
]


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(message)s")
    main()
