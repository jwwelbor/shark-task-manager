"""CLI safety and read-only database coverage for the generated dashboard."""

import sqlite3
import sys
from collections.abc import Callable
from pathlib import Path
from typing import Any, cast

import pytest


@pytest.mark.unit
def test_cli_rejects_output_path_that_would_overwrite_database(
    dashboard: Any, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    database = tmp_path / "shark-tasks.db"
    with sqlite3.connect(database) as connection:
        connection.execute("create table sentinel (value text)")
        connection.execute("insert into sentinel values ('preserve me')")
    monkeypatch.setattr(
        sys,
        "argv",
        ["generate_dashboard.py", "--db", str(database), "--out", str(database)],
    )

    with pytest.raises(SystemExit):
        dashboard.main()

    symlink_target = tmp_path / "existing-report.html"
    symlink_target.write_text("preserve me", encoding="utf-8")
    symlink = tmp_path / "report-link.html"
    symlink.symlink_to(symlink_target)
    monkeypatch.setattr(
        sys,
        "argv",
        ["generate_dashboard.py", "--db", str(database), "--out", str(symlink)],
    )
    with pytest.raises(SystemExit):
        dashboard.main()
    assert symlink_target.read_text(encoding="utf-8") == "preserve me"

    output_directory = tmp_path / "report-directory"
    output_directory.mkdir()
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "generate_dashboard.py",
            "--db",
            str(database),
            "--out",
            str(output_directory),
        ],
    )
    with pytest.raises(SystemExit):
        dashboard.main()

    unrelated_target = tmp_path / "unrelated-target.txt"
    unrelated_target.write_text("keep this file", encoding="utf-8")
    unrelated_hardlink = tmp_path / "hardlinked-report.html"
    unrelated_hardlink.hardlink_to(unrelated_target)
    monkeypatch.setattr(
        sys,
        "argv",
        [
            "generate_dashboard.py",
            "--db",
            str(database),
            "--out",
            str(unrelated_hardlink),
        ],
    )
    with pytest.raises(SystemExit):
        dashboard.main()
    assert unrelated_target.read_text(encoding="utf-8") == "keep this file"

    with sqlite3.connect(database) as connection:
        assert (
            connection.execute("select value from sentinel").fetchone()[0]
            == "preserve me"
        )

    alias = tmp_path / "database-alias.db"
    alias.hardlink_to(database)
    monkeypatch.setattr(
        sys,
        "argv",
        ["generate_dashboard.py", "--db", str(database), "--out", str(alias)],
    )
    with pytest.raises(SystemExit):
        dashboard.main()


@pytest.mark.unit
def test_cli_generates_report_from_uri_safe_read_only_database(
    dashboard: Any,
    dashboard_db: sqlite3.Connection,
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    database = tmp_path / "shark tasks ? #.db"
    output = tmp_path / "dashboard.html"
    with sqlite3.connect(database) as target:
        dashboard_db.backup(target)
    original_connect = cast(
        Callable[..., sqlite3.Connection], dashboard.sqlite3.connect
    )
    connect_calls: list[tuple[tuple[Any, ...], dict[str, Any]]] = []

    def capture_connect(*args: Any, **kwargs: Any) -> sqlite3.Connection:
        connect_calls.append((args, kwargs))
        return original_connect(*args, **kwargs)

    monkeypatch.setattr(dashboard.sqlite3, "connect", capture_connect)
    monkeypatch.setattr(
        sys,
        "argv",
        ["generate_dashboard.py", "--db", str(database), "--out", str(output)],
    )

    dashboard.main()

    assert output.exists()
    report = output.read_text(encoding="utf-8")
    assert "Shark Time &amp; Quality Dashboard" in report
    assert "feature:active" in report
    assert "1</div>outlier sessions excluded (24h)" in report
    assert "1</div>rework/rejection events" in report
    assert "1.000</b> across 2 features" in report
    assert "F01</td><td>1</td><td>1</td><td>1</td>" in report
    assert connect_calls == [
        ((f"{database.resolve().as_uri()}?mode=ro",), {"uri": True})
    ]
    with sqlite3.connect(database) as connection:
        assert connection.execute("select count(*) from features").fetchone()[0] == 2


@pytest.mark.unit
def test_atomic_report_write_preserves_previous_report_on_replace_failure(
    dashboard: Any, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    output = tmp_path / "dashboard.html"
    output.write_text("previous report", encoding="utf-8")

    def fail_replace(*args: Any, **kwargs: Any) -> None:
        raise OSError("simulated replacement failure")

    monkeypatch.setattr(dashboard.os, "replace", fail_replace)
    with pytest.raises(OSError, match="simulated replacement failure"):
        dashboard._write_text_safely(output, "new report")

    assert output.read_text(encoding="utf-8") == "previous report"


@pytest.mark.unit
def test_cli_rejects_malformed_database_without_replacing_previous_report(
    dashboard: Any,
    dashboard_db: sqlite3.Connection,
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    database = tmp_path / "malformed.db"
    output = tmp_path / "dashboard.html"
    output.write_text("previous report", encoding="utf-8")
    with sqlite3.connect(database) as target:
        dashboard_db.backup(target)
        target.execute(
            "update entity_history set changed_at='not-a-timestamp' "
            "where entity_type='feature' and entity_id=1"
        )
        target.commit()

    monkeypatch.setattr(
        sys,
        "argv",
        ["generate_dashboard.py", "--db", str(database), "--out", str(output)],
    )

    with pytest.raises(ValueError, match="Unparseable Shark timestamp"):
        dashboard.main()

    assert output.read_text(encoding="utf-8") == "previous report"
