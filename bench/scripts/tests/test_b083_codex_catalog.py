import json
import hashlib
import importlib.util
import os
from pathlib import Path
import shutil
import sys

import pytest

REPO_ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(REPO_ROOT))
from bench.scripts.lib import codex_model_catalog
from bench.scripts.lib.codex_model_catalog import (
    CatalogError,
    _collect_model_pages,
    _read_response,
    discover_models,
    resolve_route,
    validate_catalog_snapshot,
)


def test_b083_materializer_distinguishes_catalog_model_from_omission():
    script = REPO_ROOT / "bench" / "scripts" / "materialize-routing-profile.py"
    spec = importlib.util.spec_from_file_location("b083_route_materializer", script)
    assert spec is not None and spec.loader is not None
    materializer = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(materializer)
    catalog = {
        "models": [
            {
                "model": "synthetic-default",
                "is_default": True,
                "default_reasoning_effort": "medium",
                "supported_reasoning_efforts": ["medium"],
            },
            {
                "model": "unavailable",
                "is_default": False,
                "default_reasoning_effort": "low",
                "supported_reasoning_efforts": ["low", "unavailable"],
            },
        ]
    }

    explicit = materializer._resolve_route(
        "development",
        {"provider": "codex", "model": "unavailable", "effort": "unavailable"},
        catalog,
    )
    omitted = materializer._resolve_route(
        "review",
        {"provider": "codex", "model": None, "effort": None},
        catalog,
    )

    assert explicit["model"] == "unavailable"
    assert explicit["effort"] == "unavailable"
    assert omitted["model"] == "synthetic-default"


def test_b083_partial_json_line_respects_response_deadline(monkeypatch):
    class ControlledClock:
        def __init__(self):
            self.now = 100.0

        def monotonic(self):
            self.now += 0.001
            return self.now

    class FakeStdout:
        def __init__(self, descriptor):
            self.descriptor = descriptor

        def fileno(self):
            return self.descriptor

    class FakeProcess:
        def __init__(self, descriptor):
            self.stdout = FakeStdout(descriptor)

    class OneChunkSelector:
        def __init__(self):
            self.calls = 0
            self.timeouts = []

        def select(self, timeout):
            self.calls += 1
            self.timeouts.append(timeout)
            return [(None, None)] if self.calls == 1 else []

    clock = ControlledClock()
    monkeypatch.setattr(codex_model_catalog.time, "monotonic", clock.monotonic)
    read_fd, write_fd = os.pipe()
    os.write(write_fd, b'{"id":')
    process = FakeProcess(read_fd)
    selector = OneChunkSelector()
    try:
        with pytest.raises(CatalogError, match="did not complete"):
            _read_response(process, selector, request_id=1, timeout_seconds=0.1)
    finally:
        os.close(read_fd)
        os.close(write_fd)
    assert selector.calls == 2
    assert all(0 < timeout <= 0.1 for timeout in selector.timeouts)


@pytest.mark.parametrize(
    ("continuations", "expected_requests"),
    [
        ([""], 1),
        ([7], 1),
        (["repeat", "repeat"], 2),
    ],
)
def test_b083_catalog_rejects_invalid_or_repeated_continuation_cursors(
    continuations, expected_requests
):
    requests = []

    def request(method, params, *, timeout_seconds):
        assert method == "model/list"
        assert timeout_seconds > 0
        requests.append(params)
        cursor_index = len(requests) - 1
        cursor = continuations[min(cursor_index, len(continuations) - 1)]
        return {
            "data": [{
                "model": f"synthetic-page-{cursor_index}",
                "displayName": "Synthetic page",
                "description": "cursor fixture",
                "hidden": False,
                "isDefault": cursor_index == 0,
                "defaultReasoningEffort": "low",
                "supportedReasoningEfforts": [{"reasoningEffort": "low"}],
            }],
            "nextCursor": cursor,
        }

    with pytest.raises(CatalogError, match="invalid continuation cursor"):
        _collect_model_pages(request)

    assert len(requests) == expected_requests


def _synthetic_catalog_page(index, next_cursor):
    return {
        "data": [{
            "model": f"synthetic-page-{index}",
            "isDefault": index == 0,
            "defaultReasoningEffort": "low",
            "supportedReasoningEfforts": [{"reasoningEffort": "low"}],
        }],
        "nextCursor": next_cursor,
    }


def test_b083_catalog_enforces_page_limit(monkeypatch):
    monkeypatch.setattr(codex_model_catalog, "MAX_MODEL_LIST_PAGES", 2)
    requests = []

    def request(method, params, *, timeout_seconds):
        requests.append(params)
        return _synthetic_catalog_page(len(requests), f"cursor-{len(requests)}")

    with pytest.raises(CatalogError, match="exceeded the page limit"):
        _collect_model_pages(request)

    assert len(requests) == 2


def test_b083_catalog_enforces_entry_limit(monkeypatch):
    monkeypatch.setattr(codex_model_catalog, "MAX_MODEL_LIST_ENTRIES", 1)

    def request(method, params, *, timeout_seconds):
        return {"data": [{"model": "synthetic-one"}, {"model": "synthetic-two"}]}

    with pytest.raises(CatalogError, match="exceeded the entry limit"):
        _collect_model_pages(request)


def test_b083_catalog_enforces_total_discovery_deadline(monkeypatch):
    class ControlledClock:
        now = 10.0

        def monotonic(self):
            return self.now

    clock = ControlledClock()
    monkeypatch.setattr(codex_model_catalog.time, "monotonic", clock.monotonic)
    timeouts = []

    def request(method, params, *, timeout_seconds):
        timeouts.append(timeout_seconds)
        clock.now += 0.6
        return _synthetic_catalog_page(len(timeouts), f"cursor-{len(timeouts)}")

    with pytest.raises(CatalogError, match="exceeded its total time limit"):
        _collect_model_pages(request, deadline=11.0)

    assert timeouts == pytest.approx([1.0, 0.4])


def _fake_catalog_cli(tmp_path):
    executable = tmp_path / "codex-fixture"
    fixture = Path(__file__).resolve().parent / "testdata" / "b083_fake_codex_cli.py"
    shutil.copyfile(fixture, executable)
    executable.chmod(0o755)
    return executable


def test_b083_catalog_reads_all_pages_and_keeps_dynamic_metadata(tmp_path):
    executable = _fake_catalog_cli(tmp_path)
    catalog = discover_models([str(executable)])

    assert catalog["cli_version"] == "codex-fixture/1"
    assert [item["model"] for item in catalog["models"]] == [
        "synthetic-first",
        "synthetic-second",
    ]
    assert catalog["models"][1]["description"] == "second fixture"
    assert catalog["models"][1]["display_name"] == "Second"
    assert all(item["model"] != "synthetic-hidden" for item in catalog["models"])
    assert catalog["source"] == "codex-app-server-model-list"
    assert len(catalog["catalog_digest"]) == 64
    assert resolve_route("synthetic-second", "high", catalog) == {
        "model": "synthetic-second",
        "effort": "high",
    }
    with pytest.raises(CatalogError, match="does not contain pinned model"):
        resolve_route("synthetic-hidden", None, catalog)


@pytest.mark.parametrize(
    ("model", "effort", "expected"),
    [
        ("synthetic-second", "high", {"model": "synthetic-second", "effort": "high"}),
        ("synthetic-second", None, {"model": "synthetic-second", "effort": "high"}),
        (None, "low", {"model": "synthetic-first", "effort": "low"}),
        (None, None, {"model": "synthetic-first", "effort": "low"}),
    ],
)
def test_b083_route_resolves_all_model_effort_presence_combinations(
    model, effort, expected
):
    catalog = {
        "models": [
            {
                "model": "synthetic-first",
                "is_default": True,
                "default_reasoning_effort": "low",
                "supported_reasoning_efforts": ["low", "medium"],
            },
            {
                "model": "synthetic-second",
                "is_default": False,
                "default_reasoning_effort": "high",
                "supported_reasoning_efforts": ["high"],
            },
        ]
    }

    assert resolve_route(model, effort, catalog) == expected


def test_b083_route_rejects_effort_not_advertised_by_pinned_model():
    catalog = {
        "models": [{
            "model": "synthetic-second",
            "is_default": True,
            "default_reasoning_effort": "high",
            "supported_reasoning_efforts": ["high"],
        }]
    }

    with pytest.raises(CatalogError, match="does not advertise effort"):
        resolve_route("synthetic-second", "low", catalog)


def test_b083_route_never_replaces_a_missing_explicit_pin_with_default():
    catalog = {
        "models": [{
            "model": "synthetic-default",
            "is_default": True,
            "default_reasoning_effort": "low",
            "supported_reasoning_efforts": ["low"],
        }]
    }

    with pytest.raises(CatalogError, match="does not contain pinned model"):
        resolve_route("synthetic-missing", None, catalog)


def test_b083_route_rejects_an_ambiguous_default_model():
    catalog = {
        "models": [
            {"model": "synthetic-one", "is_default": True,
             "default_reasoning_effort": "low", "supported_reasoning_efforts": ["low"]},
            {"model": "synthetic-two", "is_default": True,
             "default_reasoning_effort": "high", "supported_reasoning_efforts": ["high"]},
        ]
    }

    with pytest.raises(CatalogError, match="one unique default model"):
        resolve_route(None, None, catalog)


def test_b083_route_rejects_catalog_without_a_default_model():
    catalog = {
        "models": [{
            "model": "synthetic-model",
            "is_default": False,
            "default_reasoning_effort": "low",
            "supported_reasoning_efforts": ["low"],
        }]
    }

    with pytest.raises(CatalogError, match="one unique default model"):
        resolve_route(None, None, catalog)


def test_b083_route_accepts_effort_advertised_by_runtime_catalog():
    catalog = {
        "models": [{
            "model": "synthetic-ultra-model",
            "is_default": True,
            "default_reasoning_effort": "ultra",
            "supported_reasoning_efforts": ["high", "ultra"],
        }]
    }

    assert resolve_route(None, "ultra", catalog) == {
        "model": "synthetic-ultra-model",
        "effort": "ultra",
    }


@pytest.mark.parametrize("catalog", [None, [], "malformed"])
def test_b083_route_rejects_non_object_catalog_container(catalog):
    with pytest.raises(CatalogError, match="catalog must be an object"):
        resolve_route(None, None, catalog)


def test_b083_catalog_rejects_malformed_later_page(tmp_path):
    executable = tmp_path / "codex-fixture"
    executable.write_text(
        """#!/usr/bin/env python3
import json, sys
if sys.argv[1:] == ['--version']:
    print('codex-fixture/1')
    raise SystemExit(0)
for line in sys.stdin:
    request = json.loads(line)
    if 'id' not in request:
        continue
    if request.get('method') == 'initialize':
        result = {'userAgent': 'codex-fixture/1'}
    elif request.get('method') == 'model/list' and not request['params'].get('cursor'):
        result = {'data': [{'model': 'synthetic-first', 'displayName': 'First',
            'description': 'first fixture', 'hidden': False, 'isDefault': True,
            'defaultReasoningEffort': 'low', 'supportedReasoningEfforts': [
                {'reasoningEffort': 'low'}]}], 'nextCursor': 'page-two'}
    else:
        result = {'data': 'malformed', 'nextCursor': None}
    print(json.dumps({'id': request['id'], 'result': result}), flush=True)
""",
        encoding="utf-8",
    )
    executable.chmod(0o755)

    with pytest.raises(CatalogError, match="malformed model data"):
        discover_models(
            [str(executable)],
            env={**os.environ},
        )
