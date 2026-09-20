#!/usr/bin/env python3
"""Thin operator orchestration for the E40 lifecycle benchmark.

This module prepares isolated inputs and durable identities, then delegates
execution, retention, evaluation, aggregation, and lifecycle reporting to the
existing F01-F10 scripts. It deliberately contains no workflow engine or
execution evaluator.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import math
import os
import re
import secrets
import shutil
import stat
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, NamedTuple

try:
    import yaml
except ImportError as exc:  # pragma: no cover - repository setup guarantees it
    print(f"e40-benchmark: PyYAML is required: {exc}", file=sys.stderr)
    raise SystemExit(2)


LIB_DIR = Path(__file__).resolve().parent
SCRIPTS_DIR = LIB_DIR.parent
BENCH_DIR = SCRIPTS_DIR.parent
REPO_ROOT = BENCH_DIR.parent
DIGEST_PATH_BIN = LIB_DIR / "digest_path"
DEFAULT_SCENARIO_INDEX = BENCH_DIR / "scenarios" / "scenarios.yaml"
DEFAULT_SCHEMA = BENCH_DIR / "reports" / "lifecycle-baseline-schema.yaml"
DEFAULT_I05_SCHEMA = BENCH_DIR / "evidence" / "i05-schema.yaml"
DEFAULT_I07_SCHEMA = BENCH_DIR / "runs" / "i07-schema.yaml"
DEFAULT_PROFILES_DIR = BENCH_DIR / "profiles"
DEFAULT_PROFILE_SCHEMA = DEFAULT_PROFILES_DIR / "schema.yaml"
VALID_ROUTING_EFFORTS = frozenset(
    {"none", "minimal", "low", "medium", "high", "xhigh", "max", "unavailable"}
)
VALID_ROUTING_PROVIDERS = frozenset(
    {"copilot", "github-copilot", "github_copilot", "github", "claude", "anthropic", "codex", "openai"}
)
RETENTION_REGISTRY_PATH = BENCH_DIR / "retention-registry.yaml"
CHECKOUT_SCENARIO_FIXTURE_BIN = SCRIPTS_DIR / "checkout-scenario-fixture.sh"
FIXTURE_BASE_SHA_PATTERN = re.compile(r"^[0-9a-f]{40}$")
SAFE_ID = re.compile(r"^[A-Za-z0-9][A-Za-z0-9._-]*$")
CHANGE_AXES = {"prompt", "workflow", "provider", "model", "effort", "policy"}
UNAVAILABLE = "unavailable"
# AC-F11-06a: the four operator-approved resource ceilings, named once so
# every projection/comparison site (S3's allowlist, S6's expected-authority
# and expected-ceilings blocks) stays in sync -- a ceiling present in one
# but not another is exactly the "accepted but silently dropped" defect
# this tuple exists to prevent.
RESOURCE_CEILING_NAMES = (
    "max_cost_usd",
    "max_wall_clock_seconds",
    "max_generated_tasks",
    "max_provider_calls",
)


class OperatorError(RuntimeError):
    """A bounded operator-input, identity, or publication-boundary failure."""

    def __init__(self, message: str, exit_code: int = 2):
        super().__init__(message)
        self.exit_code = exit_code


class StagingDirectory(NamedTuple):
    path: Path
    parent_path: Path
    name: str
    parent_descriptor: int
    directory_descriptor: int


class OperatorLock(NamedTuple):
    path: Path
    root_path: Path
    name: str
    root_descriptor: int
    lock_descriptor: int
    owner_pid: int


def utc_now() -> str:
    return (
        datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")
    )


def canonical_bytes(value: Any) -> bytes:
    return json.dumps(
        value, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode("utf-8")


def canonical_digest(value: Any) -> str:
    return hashlib.sha256(canonical_bytes(value)).hexdigest()


def file_digest(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def path_digest(path: Path, label: str) -> str:
    if not path.exists():
        raise OperatorError(f"{label} does not exist: {path}")
    process = subprocess.run(
        [str(DIGEST_PATH_BIN), str(path)], text=True, capture_output=True, check=False
    )
    if process.returncode != 0:
        raise OperatorError(
            f"{label} is not digestible (missing, special, or symlinked): {path}"
        )
    digest = process.stdout.strip()
    if not re.fullmatch(r"[0-9a-f]{64}", digest):
        raise OperatorError(f"{label} returned a malformed digest: {path}")
    return digest


def workflow_routing_identity(root: Path) -> dict[str, Any]:
    path_digest(root, "workflow routing bundle")
    routes: list[dict[str, Any]] = []

    def visit(value: Any, source: str, pointer: list[str]) -> None:
        if isinstance(value, dict):
            if any(key in value for key in ("provider", "model", "effort")):
                routes.append(
                    {
                        "source": source,
                        "pointer": "/" + "/".join(pointer),
                        "provider": value.get("provider", UNAVAILABLE),
                        "model": value.get("model", UNAVAILABLE),
                        "effort": value.get("effort", UNAVAILABLE),
                    }
                )
            for key, child in value.items():
                visit(child, source, [*pointer, str(key)])
        elif isinstance(value, list):
            for index, child in enumerate(value):
                visit(child, source, [*pointer, str(index)])

    if root.is_file():
        paths = [root]
        base_dir = root.parent
    else:
        paths = sorted((*root.rglob("*.yaml"), *root.rglob("*.yml")))
        base_dir = root
    for path in paths:
        document = load_yaml(path, "workflow routing file")
        visit(document, path.relative_to(base_dir).as_posix(), [])
    routes.sort(key=lambda row: (row["source"], row["pointer"]))
    if not routes:
        raise OperatorError(
            f"workflow bundle contains no provider/model/effort routes: {root}"
        )
    result = {
        "routes": routes,
        "provider_digest": canonical_digest(
            [(row["source"], row["pointer"], row["provider"]) for row in routes]
        ),
        "model_digest": canonical_digest(
            [(row["source"], row["pointer"], row["model"]) for row in routes]
        ),
        "effort_digest": canonical_digest(
            [(row["source"], row["pointer"], row["effort"]) for row in routes]
        ),
    }
    result["routing_digest"] = canonical_digest(result)
    return result


def load_routing_profile(path: Path) -> dict[str, Any]:
    """Load and parse a YAML model routing profile."""
    return load_yaml(path, "model routing profile")


def validate_routing_profile_schema(
    profile: dict[str, Any], schema_path: Path | None = None
) -> list[str]:
    """Validate a model routing profile dictionary against schema.yaml rules (AC-F13-13).
    Returns a list of error strings; an empty list indicates the profile is valid.
    """
    errors: list[str] = []
    if not isinstance(profile, dict):
        return ["profile must be a YAML mapping/dictionary"]

    schema = None
    resolved_schema_path = schema_path or DEFAULT_PROFILE_SCHEMA
    if resolved_schema_path.exists():
        try:
            schema = load_yaml(resolved_schema_path, "profile schema")
        except (OSError, yaml.YAMLError, OperatorError):
            # Fall back to built-in default vocabulary when the external schema
            # cannot be read, parsed, or loaded as a dictionary.
            schema = None

    valid_efforts = (
        set(schema.get("valid_efforts", []))
        if schema and isinstance(schema.get("valid_efforts"), list)
        else set(VALID_ROUTING_EFFORTS)
    )
    valid_providers = (
        set(schema.get("valid_providers", []))
        if schema and isinstance(schema.get("valid_providers"), list)
        else set(VALID_ROUTING_PROVIDERS)
    )

    schema_version = profile.get("schema_version")
    if schema_version != "1.0":
        errors.append(f"expected schema_version '1.0', got {schema_version!r}")

    profile_id = profile.get("profile_id")
    if not isinstance(profile_id, str) or not profile_id.strip():
        errors.append("profile_id must be a non-empty string")

    description = profile.get("description")
    if not isinstance(description, str) or not description.strip():
        errors.append("description must be a non-empty string")

    provider = profile.get("provider")
    if not isinstance(provider, str) or not provider.strip():
        errors.append("provider must be a non-empty string")
    elif valid_providers and provider.lower() not in valid_providers:
        errors.append(
            f"unsupported provider {provider!r}; expected one of {sorted(valid_providers)}"
        )

    routes = profile.get("routes")
    if not isinstance(routes, dict) or not routes:
        errors.append("routes must be a non-empty dictionary of step definitions")
    else:
        for step_name, route in routes.items():
            if not isinstance(route, dict):
                errors.append(f"routes[{step_name!r}] must be a dictionary")
                continue
            model = route.get("model")
            if not isinstance(model, str) or not model.strip():
                errors.append(f"routes[{step_name!r}].model must be a non-empty string")
            effort = route.get("effort")
            if effort is not None:
                if not isinstance(effort, str) or effort.lower() not in valid_efforts:
                    errors.append(
                        f"routes[{step_name!r}].effort {effort!r} is invalid; expected one of {sorted(valid_efforts)}"
                    )

    return errors


def apply_routing_profile(
    profile: dict[str, Any], workflow_dir: Path
) -> list[Path]:
    """Apply a model routing profile to workflow YAML files in workflow_dir.
    Updates steps matching route keys with the profile's provider, model, and effort.
    Returns the list of modified file paths.
    """
    errors = validate_routing_profile_schema(profile)
    if errors:
        raise OperatorError(
            f"cannot apply invalid routing profile: {'; '.join(errors)}"
        )
    routes = profile.get("routes") or {}
    provider = profile.get("provider")

    paths = (
        [workflow_dir]
        if workflow_dir.is_file()
        else sorted((*workflow_dir.rglob("*.yaml"), *workflow_dir.rglob("*.yml")))
    )
    modified_paths: list[Path] = []

    for path in paths:
        document = load_yaml(path, "workflow file")
        modified = False

        # Route-based workflow (Shark 2.x steps block)
        steps = document.get("steps")
        if isinstance(steps, dict):
            for step_name, step_data in steps.items():
                if isinstance(step_data, dict) and step_name in routes:
                    route = routes[step_name]
                    if provider:
                        step_data["provider"] = provider
                    if "model" in route:
                        step_data["model"] = route["model"]
                    if "effort" in route:
                        step_data["effort"] = route["effort"]
                    modified = True

        # Legacy status_metadata fallback
        status_meta = document.get("status_metadata")
        if isinstance(status_meta, dict):
            for step_name, step_data in status_meta.items():
                if isinstance(step_data, dict) and step_name in routes:
                    route = routes[step_name]
                    if provider:
                        step_data["provider"] = provider
                    if "model" in route:
                        step_data["model"] = route["model"]
                    if "effort" in route:
                        step_data["effort"] = route["effort"]
                    modified = True

        if modified:
            write_yaml(path, document)
            modified_paths.append(path)

    return modified_paths


def load_yaml(path: Path, label: str) -> dict[str, Any]:
    try:
        value = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
    except (OSError, yaml.YAMLError) as exc:
        raise OperatorError(f"cannot read {label} {path}: {exc}") from exc
    if not isinstance(value, dict):
        raise OperatorError(f"{label} must be a YAML mapping: {path}")
    return value


def load_bytes_no_follow(
    path: Path, label: str, *, invalid_exit_code: int = 2
) -> bytes:
    path = lexical_absolute_path(path)
    try:
        parent_descriptor = open_real_directory_fd(
            path.parent, f"parent directory for {label}", create=False
        )
    except OperatorError as exc:
        raise OperatorError(str(exc), invalid_exit_code) from exc
    descriptor: int | None = None
    try:
        try:
            descriptor = os.open(
                path.name,
                os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0),
                dir_fd=parent_descriptor,
            )
        except OSError as exc:
            raise OperatorError(
                f"{label} must be a retained non-symlink regular file: {path}: {exc}",
                invalid_exit_code,
            ) from exc
        descriptor_status = os.fstat(descriptor)
        if not stat.S_ISREG(descriptor_status.st_mode):
            raise OperatorError(
                f"{label} must be a retained regular file: {path}",
                invalid_exit_code,
            )
        chunks: list[bytes] = []
        while True:
            chunk = os.read(descriptor, 1024 * 1024)
            if not chunk:
                break
            chunks.append(chunk)
        final_descriptor_status = os.fstat(descriptor)
        stable_fields = (
            "st_dev",
            "st_ino",
            "st_mode",
            "st_size",
            "st_mtime_ns",
            "st_ctime_ns",
        )
        if any(
            getattr(descriptor_status, field) != getattr(final_descriptor_status, field)
            for field in stable_fields
        ):
            raise OperatorError(
                f"{label} changed while reading: {path}", invalid_exit_code
            )
        try:
            path_status = os.stat(
                path.name, dir_fd=parent_descriptor, follow_symlinks=False
            )
        except OSError as exc:
            raise OperatorError(
                f"{label} path changed while reading: {path}: {exc}",
                invalid_exit_code,
            ) from exc
        if not stat.S_ISREG(path_status.st_mode) or (
            path_status.st_dev,
            path_status.st_ino,
        ) != (final_descriptor_status.st_dev, final_descriptor_status.st_ino):
            raise OperatorError(
                f"{label} path changed while reading: {path}",
                invalid_exit_code,
            )
        assert_directory_fd_matches_path(
            parent_descriptor, path.parent, f"parent directory for {label}"
        )
        encoded = b"".join(chunks)
        if len(encoded) != final_descriptor_status.st_size:
            raise OperatorError(
                f"{label} changed while reading: {path}", invalid_exit_code
            )
        return encoded
    finally:
        if descriptor is not None:
            os.close(descriptor)
        os.close(parent_descriptor)


def load_json_no_follow(
    path: Path, label: str, *, invalid_exit_code: int = 2
) -> tuple[dict[str, Any], bytes]:
    encoded = load_bytes_no_follow(path, label, invalid_exit_code=invalid_exit_code)
    try:
        value = json.loads(encoded)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise OperatorError(
            f"cannot read {label} {path}: {exc}", invalid_exit_code
        ) from exc
    if not isinstance(value, dict):
        raise OperatorError(f"{label} must be a JSON object: {path}", invalid_exit_code)
    return value, encoded


def load_bytes_at(
    parent_descriptor: int,
    name: str,
    label: str,
    *,
    invalid_exit_code: int = 2,
) -> bytes:
    if not name or "/" in name or name in {".", ".."}:
        raise OperatorError(
            f"{label} has an invalid descriptor-relative name: {name!r}",
            invalid_exit_code,
        )
    descriptor: int | None = None
    try:
        try:
            descriptor = os.open(
                name,
                os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0),
                dir_fd=parent_descriptor,
            )
        except OSError as exc:
            raise OperatorError(
                f"{label} must be a retained non-symlink regular file: {name}: {exc}",
                invalid_exit_code,
            ) from exc
        initial = os.fstat(descriptor)
        if not stat.S_ISREG(initial.st_mode):
            raise OperatorError(
                f"{label} must be a retained regular file: {name}", invalid_exit_code
            )
        chunks: list[bytes] = []
        while True:
            chunk = os.read(descriptor, 1024 * 1024)
            if not chunk:
                break
            chunks.append(chunk)
        final = os.fstat(descriptor)
        stable_fields = (
            "st_dev",
            "st_ino",
            "st_mode",
            "st_size",
            "st_mtime_ns",
            "st_ctime_ns",
        )
        if any(
            getattr(initial, field) != getattr(final, field) for field in stable_fields
        ):
            raise OperatorError(
                f"{label} changed while reading: {name}", invalid_exit_code
            )
        path_status = os.stat(name, dir_fd=parent_descriptor, follow_symlinks=False)
        if not stat.S_ISREG(path_status.st_mode) or (
            path_status.st_dev,
            path_status.st_ino,
        ) != (final.st_dev, final.st_ino):
            raise OperatorError(
                f"{label} path changed while reading: {name}", invalid_exit_code
            )
        encoded = b"".join(chunks)
        if len(encoded) != final.st_size:
            raise OperatorError(
                f"{label} changed while reading: {name}", invalid_exit_code
            )
        return encoded
    finally:
        if descriptor is not None:
            os.close(descriptor)


def load_json_at(
    parent_descriptor: int,
    name: str,
    label: str,
    *,
    invalid_exit_code: int = 2,
) -> tuple[dict[str, Any], bytes]:
    encoded = load_bytes_at(
        parent_descriptor, name, label, invalid_exit_code=invalid_exit_code
    )
    try:
        value = json.loads(encoded)
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise OperatorError(
            f"cannot read {label} {name}: {exc}", invalid_exit_code
        ) from exc
    if not isinstance(value, dict):
        raise OperatorError(f"{label} must be a JSON object: {name}", invalid_exit_code)
    return value, encoded


def assert_directory_fd_matches_path(descriptor: int, path: Path, label: str) -> None:
    try:
        path_status = os.stat(path, follow_symlinks=False)
        descriptor_status = os.fstat(descriptor)
    except OSError as exc:
        raise OperatorError(f"{label} changed after validation: {path}: {exc}") from exc
    if not stat.S_ISDIR(path_status.st_mode) or (
        path_status.st_dev,
        path_status.st_ino,
    ) != (descriptor_status.st_dev, descriptor_status.st_ino):
        raise OperatorError(f"{label} changed after validation: {path}")


def open_real_directory_fd(path: Path, label: str, *, create: bool) -> int:
    absolute = lexical_absolute_path(path)
    flags = os.O_RDONLY | os.O_DIRECTORY | getattr(os, "O_NOFOLLOW", 0)
    try:
        descriptor = os.open(absolute.anchor, flags)
    except OSError as exc:
        raise OperatorError(f"cannot open filesystem root for {label}: {exc}") from exc
    try:
        for component in absolute.parts[1:]:
            if create:
                try:
                    os.mkdir(component, dir_fd=descriptor)
                except FileExistsError:
                    pass
                except OSError as exc:
                    raise OperatorError(
                        f"{label} could not create a real directory component "
                        f"{component!r}: {exc}"
                    ) from exc
            try:
                child = os.open(component, flags, dir_fd=descriptor)
            except OSError as exc:
                raise OperatorError(
                    f"{label} contains a symlinked directory or non-directory "
                    f"component {component!r}: {exc}"
                ) from exc
            os.close(descriptor)
            descriptor = child
        assert_directory_fd_matches_path(descriptor, absolute, label)
        return descriptor
    except BaseException:
        os.close(descriptor)
        raise


def ensure_real_directory(path: Path, label: str) -> Path:
    absolute = lexical_absolute_path(path)
    descriptor = open_real_directory_fd(absolute, label, create=True)
    os.close(descriptor)
    return absolute


def require_existing_real_directory(path: Path, label: str) -> Path:
    absolute = lexical_absolute_path(path)
    descriptor = open_real_directory_fd(absolute, label, create=False)
    os.close(descriptor)
    return absolute


def open_child_directory_fd(parent_descriptor: int, name: str, label: str) -> int:
    try:
        return os.open(
            name,
            os.O_RDONLY | os.O_DIRECTORY | getattr(os, "O_NOFOLLOW", 0),
            dir_fd=parent_descriptor,
        )
    except OSError as exc:
        raise OperatorError(
            f"{label} is not a real child directory {name!r}: {exc}"
        ) from exc


def secure_mkdtemp(parent: Path, prefix: str, label: str) -> StagingDirectory:
    if "/" in prefix or not prefix:
        raise OperatorError(f"{label} has an invalid staging prefix: {prefix!r}")
    parent_path = lexical_absolute_path(parent)
    parent_descriptor = open_real_directory_fd(parent_path, label, create=True)
    directory_descriptor: int | None = None
    name: str | None = None
    try:
        assert_directory_fd_matches_path(parent_descriptor, parent_path, label)
        for _attempt in range(128):
            candidate = f"{prefix}{secrets.token_hex(8)}"
            try:
                os.mkdir(candidate, 0o700, dir_fd=parent_descriptor)
            except FileExistsError:
                continue
            except OSError as exc:
                raise OperatorError(
                    f"cannot create {label} under {parent_path}: {exc}"
                ) from exc
            name = candidate
            directory_descriptor = open_child_directory_fd(
                parent_descriptor, candidate, label
            )
            break
        if name is None or directory_descriptor is None:
            raise OperatorError(f"cannot allocate {label} under {parent_path}")
        path = parent_path / name
        assert_directory_fd_matches_path(parent_descriptor, parent_path, label)
        assert_directory_fd_matches_path(directory_descriptor, path, label)
        return StagingDirectory(
            path=path,
            parent_path=parent_path,
            name=name,
            parent_descriptor=parent_descriptor,
            directory_descriptor=directory_descriptor,
        )
    except BaseException:
        if directory_descriptor is not None:
            os.close(directory_descriptor)
        os.close(parent_descriptor)
        raise


def verify_staging_directory(staging: StagingDirectory, label: str) -> None:
    assert_directory_fd_matches_path(
        staging.parent_descriptor, staging.parent_path, f"{label} parent"
    )
    assert_directory_fd_matches_path(staging.directory_descriptor, staging.path, label)


def publish_staging_directory(
    staging: StagingDirectory,
    destination: Path,
    label: str,
    *,
    replace_empty: bool,
) -> None:
    destination = lexical_absolute_path(destination)
    if destination.parent != staging.parent_path:
        raise OperatorError(
            f"{label} destination must share its validated staging parent: {destination}"
        )
    verify_staging_directory(staging, label)
    try:
        destination_status = os.stat(
            destination.name,
            dir_fd=staging.parent_descriptor,
            follow_symlinks=False,
        )
    except FileNotFoundError:
        destination_status = None
    except OSError as exc:
        raise OperatorError(
            f"cannot inspect {label} destination {destination}: {exc}"
        ) from exc
    if destination_status is not None:
        if not replace_empty or not stat.S_ISDIR(destination_status.st_mode):
            raise OperatorError(f"{label} destination already exists: {destination}")
        destination_descriptor = open_child_directory_fd(
            staging.parent_descriptor, destination.name, f"{label} destination"
        )
        try:
            if os.listdir(destination_descriptor):
                raise OperatorError(f"{label} destination is not empty: {destination}")
            assert_directory_fd_matches_path(
                destination_descriptor, destination, f"{label} destination"
            )
        finally:
            os.close(destination_descriptor)
        os.rmdir(destination.name, dir_fd=staging.parent_descriptor)
    verify_staging_directory(staging, label)
    try:
        os.rename(
            staging.name,
            destination.name,
            src_dir_fd=staging.parent_descriptor,
            dst_dir_fd=staging.parent_descriptor,
        )
        os.fsync(staging.parent_descriptor)
        assert_directory_fd_matches_path(
            staging.directory_descriptor, destination, label
        )
    finally:
        os.close(staging.directory_descriptor)
        os.close(staging.parent_descriptor)


def atomic_write(path: Path, data: bytes, *, overwrite: bool = True) -> None:
    path = lexical_absolute_path(path)
    parent_label = f"parent directory for {path}"
    parent_descriptor = open_real_directory_fd(path.parent, parent_label, create=True)
    temporary_name: str | None = None
    try:
        try:
            destination_status = os.stat(
                path.name, dir_fd=parent_descriptor, follow_symlinks=False
            )
        except FileNotFoundError:
            destination_status = None
        except OSError as exc:
            raise OperatorError(f"cannot inspect destination {path}: {exc}") from exc
        if destination_status is not None:
            if not stat.S_ISREG(destination_status.st_mode):
                raise OperatorError(
                    f"refusing to write through non-file destination: {path}"
                )
            if not overwrite:
                raise OperatorError(f"refusing to overwrite existing artifact: {path}")

        temporary_descriptor: int | None = None
        for _attempt in range(128):
            candidate = f".{path.name}.{os.getpid()}.{secrets.token_hex(8)}"
            try:
                temporary_descriptor = os.open(
                    candidate,
                    os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0),
                    0o600,
                    dir_fd=parent_descriptor,
                )
            except FileExistsError:
                continue
            except OSError as exc:
                raise OperatorError(
                    f"cannot create temporary artifact beside {path}: {exc}"
                ) from exc
            temporary_name = candidate
            break
        if temporary_descriptor is None or temporary_name is None:
            raise OperatorError(f"cannot allocate temporary artifact beside {path}")

        with os.fdopen(temporary_descriptor, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        assert_directory_fd_matches_path(parent_descriptor, path.parent, parent_label)
        os.replace(
            temporary_name,
            path.name,
            src_dir_fd=parent_descriptor,
            dst_dir_fd=parent_descriptor,
        )
        temporary_name = None
        os.fsync(parent_descriptor)
        assert_directory_fd_matches_path(parent_descriptor, path.parent, parent_label)
    except BaseException:
        if temporary_name is not None:
            try:
                os.unlink(temporary_name, dir_fd=parent_descriptor)
            except FileNotFoundError:
                pass
        raise
    finally:
        os.close(parent_descriptor)


def atomic_write_at(
    parent_descriptor: int,
    name: str,
    data: bytes,
    *,
    label: str,
    overwrite: bool = True,
) -> None:
    if not name or "/" in name or name in {".", ".."}:
        raise OperatorError(
            f"{label} has an invalid descriptor-relative name: {name!r}"
        )
    temporary_name: str | None = None
    try:
        try:
            destination_status = os.stat(
                name, dir_fd=parent_descriptor, follow_symlinks=False
            )
        except FileNotFoundError:
            destination_status = None
        except OSError as exc:
            raise OperatorError(f"cannot inspect {label} {name}: {exc}") from exc
        if destination_status is not None:
            if not stat.S_ISREG(destination_status.st_mode):
                raise OperatorError(
                    f"refusing to write through non-file {label}: {name}"
                )
            if not overwrite:
                raise OperatorError(f"refusing to overwrite existing {label}: {name}")
        temporary_descriptor: int | None = None
        for _attempt in range(128):
            candidate = f".{name}.{os.getpid()}.{secrets.token_hex(8)}"
            try:
                temporary_descriptor = os.open(
                    candidate,
                    os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0),
                    0o600,
                    dir_fd=parent_descriptor,
                )
            except FileExistsError:
                continue
            except OSError as exc:
                raise OperatorError(f"cannot create temporary {label}: {exc}") from exc
            temporary_name = candidate
            break
        if temporary_descriptor is None or temporary_name is None:
            raise OperatorError(f"cannot allocate temporary {label}")
        with os.fdopen(temporary_descriptor, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(
            temporary_name,
            name,
            src_dir_fd=parent_descriptor,
            dst_dir_fd=parent_descriptor,
        )
        temporary_name = None
        os.fsync(parent_descriptor)
    except BaseException:
        if temporary_name is not None:
            try:
                os.unlink(temporary_name, dir_fd=parent_descriptor)
            except FileNotFoundError:
                pass
        raise


def write_json_at(
    parent_descriptor: int,
    name: str,
    value: Any,
    *,
    label: str,
    overwrite: bool = True,
) -> None:
    atomic_write_at(
        parent_descriptor,
        name,
        (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n").encode(
            "utf-8"
        ),
        label=label,
        overwrite=overwrite,
    )


def write_json(path: Path, value: Any, *, overwrite: bool = True) -> None:
    atomic_write(
        path,
        (json.dumps(value, indent=2, sort_keys=True, ensure_ascii=False) + "\n").encode(
            "utf-8"
        ),
        overwrite=overwrite,
    )


def write_yaml(path: Path, value: Any, *, overwrite: bool = True) -> None:
    data = yaml.safe_dump(value, sort_keys=False, allow_unicode=True).encode("utf-8")
    atomic_write(path, data, overwrite=overwrite)


def lexical_absolute_path(
    value: str | os.PathLike[str], base: Path | None = None
) -> Path:
    """Return a normalized absolute path without following symlinks."""
    path = Path(value)
    if not path.is_absolute():
        path = (base if base is not None else Path.cwd()) / path
    return Path(os.path.abspath(path))


def resolve_path(value: str | os.PathLike[str], base: Path) -> Path:
    return lexical_absolute_path(value, base)


def run_process(
    command: list[str],
    *,
    cwd: Path | None = None,
    env: dict[str, str] | None = None,
    input_text: str | None = None,
    pass_fds: tuple[int, ...] = (),
) -> subprocess.CompletedProcess[str]:
    try:
        return subprocess.run(
            command,
            cwd=cwd,
            env=env,
            input=input_text,
            text=True,
            capture_output=True,
            check=False,
            pass_fds=pass_fds,
        )
    except OSError as exc:
        raise OperatorError(f"cannot execute {' '.join(command)}: {exc}") from exc


def require_safe_id(value: str, label: str) -> str:
    if not SAFE_ID.fullmatch(value):
        raise OperatorError(f"{label} must match {SAFE_ID.pattern}, got {value!r}")
    return value


def require_positive(value: Any, label: str, integer: bool = False) -> int | float:
    # AC-F11-06b: harden against non-finite and lossy inputs the plain
    # int()/float() conversion below would otherwise accept silently.
    # Finiteness is checked via a float parse BEFORE any int() truncation,
    # so a fractional or non-finite value is caught before its information
    # is thrown away -- int(2.5) or int(float("nan")) would otherwise
    # either silently truncate or raise the wrong (generic) error.
    if isinstance(value, bool):
        raise OperatorError(f"{label} must be strictly positive")
    try:
        as_float = float(value)
    except OverflowError:
        # An int literal too large for a float to represent at all (e.g.
        # 10**400) is unbounded in the same sense "inf" is.
        raise OperatorError(f"{label} must be a finite number") from None
    except (TypeError, ValueError) as exc:
        raise OperatorError(f"{label} must be numeric and strictly positive") from exc
    if math.isnan(as_float) or math.isinf(as_float):
        raise OperatorError(f"{label} must be a finite number")
    if not integer:
        parsed: int | float = as_float
    elif not as_float.is_integer():
        raise OperatorError(f"{label} must be a whole number, got {value}")
    else:
        try:
            # int(value) on the ORIGINAL value (not as_float) so a large
            # exact integer (e.g. 10**30) keeps its exact magnitude rather
            # than round-tripping through float precision loss.
            parsed = int(value)
        except (TypeError, ValueError) as exc:
            # Reached only when as_float is finite and whole but int()
            # itself still refuses the original value -- e.g. the decimal-
            # point string "5.0", which int() has always rejected. Preserve
            # that existing rejection rather than silently widening
            # acceptance to a shape the committed-data sweep never covered.
            raise OperatorError(
                f"{label} must be numeric and strictly positive"
            ) from exc
    if parsed <= 0:
        raise OperatorError(f"{label} must be strictly positive")
    return parsed


def normalized_ceiling(value: Any) -> Any:
    try:
        text = str(value)
        return float(value) if "." in text or "e" in text.lower() else int(value)
    except (TypeError, ValueError, OverflowError):
        return value


def batch_authority_projection(batch: dict[str, Any]) -> dict[str, Any]:
    raw_ceilings = batch.get("ceilings") or {}
    retained_root = batch.get("retention_root")
    return {
        "phase": batch.get("phase"),
        "batch_id": batch.get("batch_id"),
        "mode": batch.get("mode"),
        "retention_root": str(lexical_absolute_path(retained_root))
        if isinstance(retained_root, str) and Path(retained_root).is_absolute()
        else retained_root,
        "batch_policy_digest": batch.get("batch_policy_digest"),
        "ceilings": {
            key: normalized_ceiling(raw_ceilings.get(key))
            for key in RESOURCE_CEILING_NAMES
        },
        "acknowledgement_ref": batch.get("acknowledgement_ref"),
        "min_reps": batch.get("min_reps"),
    }


def config_context(config_path: Path) -> tuple[dict[str, Any], Path]:
    config_path = config_path.resolve()
    config = load_yaml(config_path, "benchmark config")
    if str(config.get("schema_version")) != "1.0":
        raise OperatorError(
            f"unsupported benchmark config schema_version {config.get('schema_version')!r}: {config_path}"
        )
    return config, config_path.parent


def config_path_value(config: dict[str, Any], base: Path, key: str) -> Path:
    raw = config.get(key)
    if not isinstance(raw, str) or not raw:
        raise OperatorError(f"benchmark config is missing {key}")
    return resolve_path(raw, base)


def retention_registry_entries() -> list[dict[str, Any]]:
    """Load `bench/retention-registry.yaml`'s `entries[]` (AC-F11-01a).

    The registry is a committed data file, not source: it holds the ONLY
    copy of any operator-specific absolute path this repository carries.
    Missing file -> no entries (a fresh checkout has frozen nothing yet).
    """
    if not RETENTION_REGISTRY_PATH.is_file():
        return []
    registry = load_yaml(RETENTION_REGISTRY_PATH, "retention registry")
    entries = registry.get("entries")
    if entries is None:
        return []
    if not isinstance(entries, list):
        raise OperatorError(
            f"retention registry entries must be a list: {RETENTION_REGISTRY_PATH}"
        )
    for entry in entries:
        if not isinstance(entry, dict):
            raise OperatorError(
                f"retention registry entry must be a mapping: {RETENTION_REGISTRY_PATH}"
            )
        for field in ("registry_id", "root_path", "classification", "tree_manifest_path"):
            value = entry.get(field)
            if not isinstance(value, str) or not value:
                raise OperatorError(
                    f"retention registry entry is missing {field}: {RETENTION_REGISTRY_PATH}"
                )
    return entries


def retention_registry_entry(registry_id: str) -> dict[str, Any]:
    for entry in retention_registry_entries():
        if entry["registry_id"] == registry_id:
            return entry
    raise OperatorError(
        f"retention registry has no entry for registry_id {registry_id!r}: "
        f"{RETENTION_REGISTRY_PATH}"
    )


def retention_registry_lookup(path: Path) -> dict[str, Any] | None:
    """Match a candidate path against every registered retention root.

    Returns the matching entry, or None if the path is not (and is not
    nested inside) any registered root. AC-F11-01a: an entry whose
    `root_path` resolves inside REPO_ROOT is a validation error -- the
    registry must never smuggle a live-repo path back into scope. An entry
    whose `root_path` does not exist on the current host is a no-match, not
    a rejection -- retention roots are machine-local operator evidence.
    """
    candidate = path.resolve()
    for entry in retention_registry_entries():
        raw_root = entry["root_path"]
        root_path = Path(raw_root)
        if not root_path.is_absolute():
            raise OperatorError(
                f"retention registry root_path must be absolute: {raw_root}"
            )
        resolved_root = root_path.resolve()
        if resolved_root == REPO_ROOT or REPO_ROOT in resolved_root.parents:
            raise OperatorError(
                f"retention registry entry {entry['registry_id']!r} root_path "
                f"resolves inside the repository: {root_path}"
            )
        if not resolved_root.exists():
            continue
        if candidate == resolved_root or resolved_root in candidate.parents:
            return entry
    return None


def _tree_manifest_entries(text: str) -> dict[str, str]:
    entries: dict[str, str] = {}
    for line in text.splitlines():
        if line.startswith("# files="):
            continue
        relative_path, separator, value = line.partition(" ")
        if relative_path and separator:
            entries[relative_path] = value
    return entries


def diff_tree_manifests(
    recorded_text: str, recomputed_text: str
) -> list[dict[str, str]]:
    """Name every relative path whose entry differs between two manifests.

    AC-F11-01b: a single changed, added, removed, or renamed file anywhere
    in the tree fails the check with the differing path named. A rename
    surfaces as one removed path plus one added path -- both are named.
    """
    recorded = _tree_manifest_entries(recorded_text)
    recomputed = _tree_manifest_entries(recomputed_text)
    differences: list[dict[str, str]] = []
    for relative_path in sorted(set(recorded) | set(recomputed)):
        before = recorded.get(relative_path)
        after = recomputed.get(relative_path)
        if before == after:
            continue
        if before is None:
            differences.append({"path": relative_path, "change": "added"})
        elif after is None:
            differences.append({"path": relative_path, "change": "removed"})
        else:
            differences.append({"path": relative_path, "change": "modified"})
    return differences


def compute_tree_manifest(root: Path) -> str:
    """Whole-tree digest manifest of `root` (AC-F11-01b).

    Every regular file contributes a sorted `<relative-path> <sha256>`
    line. Symlinks are recorded by their link TARGET text, never followed
    (a followed symlink would hash content outside the retained tree, or
    silently vanish if dangling). A trailing `# files=<n> bytes=<n>` footer
    names the total entry count and the total byte size of regular-file
    content.
    """
    root = root.resolve()
    lines: list[str] = []
    file_count = 0
    total_bytes = 0
    for entry in root.rglob("*"):
        relative_path = entry.relative_to(root).as_posix()
        if entry.is_symlink():
            lines.append(f"{relative_path} symlink:{os.readlink(entry)}")
            file_count += 1
            continue
        if entry.is_dir():
            continue
        if not entry.is_file():
            raise OperatorError(
                f"retention root contains an unsupported filesystem entry: {entry}"
            )
        lines.append(f"{relative_path} {file_digest(entry)}")
        file_count += 1
        total_bytes += entry.stat().st_size
    lines.sort()
    lines.append(f"# files={file_count} bytes={total_bytes}")
    return "\n".join(lines) + "\n"


def verify_tree_manifest(
    registry_id: str, *, root_override: Path | None = None
) -> dict[str, Any]:
    """Recompute a registered root's tree manifest and diff it against the
    manifest captured at registration time (AC-F11-01b).

    `root_override` lets a caller (the `tc099` mutation sweep) recompute
    against a disposable COPY of the root while still diffing against the
    real registry entry's recorded manifest -- the registered root itself
    is never touched by this function.

    Returns `{"status": "match"|"mismatch"|"not_present_on_host", ...}`.
    `not_present_on_host` is distinct from both `match` and `mismatch`
    (AC-F11-01a): a registry entry whose root does not exist on this host
    cannot be verified here, one way or the other.
    """
    entry = retention_registry_entry(registry_id)
    root_path = Path(entry["root_path"])
    if not root_path.is_absolute():
        raise OperatorError(f"retention registry root_path must be absolute: {root_path}")
    resolved_root = root_path.resolve()
    if resolved_root == REPO_ROOT or REPO_ROOT in resolved_root.parents:
        raise OperatorError(
            f"retention registry entry {registry_id!r} root_path resolves "
            f"inside the repository: {root_path}"
        )
    verify_root = root_override.resolve() if root_override is not None else resolved_root
    if root_override is None and not resolved_root.exists():
        return {
            "status": "not_present_on_host",
            "registry_id": registry_id,
            "root_path": str(root_path),
        }
    manifest_path = REPO_ROOT / entry["tree_manifest_path"]
    if not manifest_path.is_file():
        raise OperatorError(f"registered tree manifest is missing: {manifest_path}")
    recorded_text = manifest_path.read_text(encoding="utf-8")
    recomputed_text = compute_tree_manifest(verify_root)
    if recomputed_text == recorded_text:
        return {"status": "match", "registry_id": registry_id, "root_path": str(root_path)}
    return {
        "status": "mismatch",
        "registry_id": registry_id,
        "root_path": str(root_path),
        "differences": diff_tree_manifests(recorded_text, recomputed_text),
    }


def ensure_external_operator_root(path: Path) -> None:
    candidate = path.resolve()
    if candidate == REPO_ROOT or REPO_ROOT in candidate.parents:
        raise OperatorError(
            f"operator root must be outside the live repository checkout: {candidate}"
        )
    if candidate == Path(candidate.anchor):
        raise OperatorError(f"operator root is too broad: {candidate}")
    registered = retention_registry_lookup(candidate)
    if registered is not None:
        raise OperatorError(
            "operator root is frozen as retained evidence "
            f"(registry_id={registered['registry_id']}, "
            f"root_path={registered['root_path']}): {candidate}"
        )


def fixture_checkout_head(checkout_dir: Path) -> str:
    process = subprocess.run(
        ["git", "-C", str(checkout_dir), "rev-parse", "HEAD"],
        capture_output=True,
        text=True,
        check=False,
    )
    if process.returncode != 0:
        raise OperatorError(
            f"cannot resolve fixture checkout HEAD: {checkout_dir}: {process.stderr.strip()}"
        )
    return process.stdout.strip()


def verify_fixture_checkout_binding(checkout_dir: Path, base_sha: str) -> None:
    """AC-F11-03: the checkout's resolved HEAD must equal the package's
    admitted fixture.base_sha before any collector process starts. Raises
    OperatorError naming both SHAs on mismatch -- never silently proceeds."""
    actual = fixture_checkout_head(checkout_dir)
    if actual != base_sha:
        raise OperatorError(
            "admitted fixture checkout HEAD mismatch: "
            f"checkout={checkout_dir} expected fixture.base_sha={base_sha} got={actual}"
        )


def mark_tree_read_only(root: Path) -> None:
    """Clears write permission on every regular file under `root` (ADR-F11-03:
    the checkout is marked read-only once verified), so no in-place content
    edit -- by a collector, an isolation-guard bug, or an operator typo --
    can silently succeed against a checkout other scenarios in the same
    operator run are relying on. Directory write bits are deliberately left
    untouched: unlinking a file only requires write permission on its
    PARENT directory, not the file itself, so a caller tearing down an
    operator run with a plain `rm -rf` still works. A content edit is
    blocked by permission; a directory-level tamper (rename, add, remove)
    is instead caught the next time this checkout is reused, by the HEAD
    re-verification in admitted_fixture_checkout(). Symlinks are skipped --
    chmod on a symlink follows it and would affect its target, not the
    link itself."""
    for dirpath, _dirnames, filenames in os.walk(root):
        for name in filenames:
            entry = os.path.join(dirpath, name)
            if os.path.islink(entry):
                continue
            mode = os.stat(entry, follow_symlinks=False).st_mode
            os.chmod(entry, mode & ~(stat.S_IWUSR | stat.S_IWGRP | stat.S_IWOTH))


def admitted_fixture_checkout(fixture_id: str, base_sha: str, cache_root: Path) -> Path:
    """REQ-F-002/ADR-F11-03: one immutable checkout per (fixture_id,
    base_sha) per operator run, shared across scenarios that admit the
    same pair.

    Creates or reuses `<cache_root>/<fixture_id>/<base_sha>/`, delegating
    the actual clone/checkout to checkout-scenario-fixture.sh (REQ-NF-006:
    that script's <fixture_id> <base_sha> <dest_dir> interface is frozen).
    Verifies the resulting checkout's HEAD against base_sha, marks the tree
    read-only, and returns its absolute path. Never touches the repository
    submodule -- checkout-scenario-fixture.sh only ever clones FROM it into
    a caller-supplied, brand-new dest_dir.
    """
    require_safe_id(fixture_id, "fixture_id")
    if not FIXTURE_BASE_SHA_PATTERN.fullmatch(base_sha):
        raise OperatorError(
            f"fixture base_sha must be a 40-character commit SHA: {base_sha!r}"
        )
    checkout_dir = (cache_root / fixture_id / base_sha).resolve()
    if checkout_dir.is_dir():
        # Reuse: checkout-scenario-fixture.sh itself refuses a pre-existing
        # dest_dir, so a second (fixture_id, base_sha) request within the
        # same operator run re-verifies the existing checkout rather than
        # re-cloning it.
        verify_fixture_checkout_binding(checkout_dir, base_sha)
        return checkout_dir
    checkout_dir.parent.mkdir(parents=True, exist_ok=True)
    process = run_process(
        [str(CHECKOUT_SCENARIO_FIXTURE_BIN), fixture_id, base_sha, str(checkout_dir)]
    )
    if process.returncode != 0:
        raise OperatorError(
            f"admitted fixture checkout failed for {fixture_id}@{base_sha}: "
            f"{process.stderr.strip() or process.stdout.strip()}"
        )
    verify_fixture_checkout_binding(checkout_dir, base_sha)
    mark_tree_read_only(checkout_dir)
    return checkout_dir


def isolated_environment() -> dict[str, str]:
    environment = os.environ.copy()
    environment["SHARK_DB_BACKEND"] = "sqlite"
    for name in (
        "SHARK_DB_URL",
        "SHARK_AUTH_TOKEN",
        "SHARK_AUTH_TOKEN_FILE",
        "SHARK_BIN",
        "LIFECYCLE_ADAPTER",
        "LIFECYCLE_ADAPTER_PATH",
        "LIFECYCLE_PROVIDER_COMMAND",
        "RUN_LIFECYCLE_BIN",
        "EVALUATE_LIFECYCLE_BIN",
        "ENTITY_HISTORY_EXPORT_BIN",
        "LIFECYCLE_SCHEMA",
        "I05_SCHEMA",
        "I07_SCHEMA",
        "E40_RUN_LIFECYCLE_BATCH_BIN",
        "E40_AGGREGATE_LIFECYCLE_BIN",
        "E40_REPORT_LIFECYCLE_BIN",
        "E40_VERIFY_RETENTION_ROOT_BIN",
    ):
        environment.pop(name, None)
    return environment


def aggregation_environment() -> dict[str, str]:
    environment = isolated_environment()
    environment["LIFECYCLE_SCHEMA"] = str(DEFAULT_SCHEMA)
    environment["I05_SCHEMA"] = str(DEFAULT_I05_SCHEMA)
    environment["I07_SCHEMA"] = str(DEFAULT_I07_SCHEMA)
    return environment


def parse_json_stdout(
    process: subprocess.CompletedProcess[str], label: str
) -> dict[str, Any]:
    if process.returncode != 0:
        detail = process.stderr.strip() or process.stdout.strip()
        raise OperatorError(f"{label} failed ({process.returncode}): {detail}", 1)
    try:
        value = json.loads(process.stdout)
    except json.JSONDecodeError as exc:
        raise OperatorError(f"{label} returned non-JSON output: {exc}") from exc
    if not isinstance(value, dict):
        raise OperatorError(f"{label} returned a non-object JSON value")
    return value


def shark_json(
    binary: Path, scratch: Path, arguments: list[str], label: str
) -> dict[str, Any]:
    process = run_process(
        [str(binary), *arguments, "--json"], cwd=scratch, env=isolated_environment()
    )
    return parse_json_stdout(process, label)


def scenario_packages(index_path: Path) -> list[tuple[Path, dict[str, Any]]]:
    index = load_yaml(index_path, "scenario index")
    rows = index.get("scenarios")
    if not isinstance(rows, list) or not rows:
        raise OperatorError(f"scenario index contains no scenarios: {index_path}")
    packages: list[tuple[Path, dict[str, Any]]] = []
    for raw in rows:
        if not isinstance(raw, str) or not raw:
            raise OperatorError(
                f"scenario index has an invalid package reference: {raw!r}"
            )
        package_path = (index_path.parent / raw / "package.yaml").resolve()
        package = load_yaml(package_path, "scenario package")
        packages.append((package_path, package))
    return packages


def selected_matrix(
    config: dict[str, Any],
    config_base: Path,
    selected: list[str] | None,
    reps: int,
    cache_root: Path | None = None,
) -> list[dict[str, Any]]:
    index_path = resolve_path(
        str(config.get("scenario_index") or DEFAULT_SCENARIO_INDEX), config_base
    )
    selected_set = set(selected or [])
    rows: list[dict[str, Any]] = []
    for package_path, package in scenario_packages(index_path):
        scenario_id = str(package.get("scenario_id") or "")
        require_safe_id(scenario_id, "scenario_id")
        if selected_set and scenario_id not in selected_set:
            continue
        fixture = package.get("fixture") or {}
        adapter = package.get("adapter") or {}
        resource_policy = package.get("resource_policy") or {}
        fixture_id = fixture.get("fixture_id")
        fixture_base_sha = fixture.get("base_sha")
        fixture_checkout = None
        # REQ-F-002/AC-F11-05: bound to the admitted fixture.base_sha via an
        # immutable checkout, never the submodule's incidental working-tree
        # HEAD -- computed only when a cache_root is supplied (preflight and
        # provider-backed execution; setup has no operator run yet to scope
        # the cache to).
        if cache_root is not None and fixture_id and fixture_base_sha:
            fixture_checkout = str(
                admitted_fixture_checkout(
                    str(fixture_id), str(fixture_base_sha), cache_root
                )
            )
        rows.append(
            {
                "scenario_id": scenario_id,
                "scenario_version": str(package.get("scenario_version")),
                "family": str(package.get("entity_family")),
                "package_path": str(package_path),
                "package_digest": path_digest(
                    package_path.parent, f"scenario package {scenario_id}"
                ),
                "fixture_id": str(fixture_id),
                "fixture_base_sha": str(fixture_base_sha),
                "fixture_checkout": fixture_checkout,
                "adapter_id": str(adapter.get("name")),
                "adapter_version": str(adapter.get("version")),
                "toolchain_identity": package.get("toolchain_identity"),
                "resource_policy": resource_policy,
                "resource_policy_digest": canonical_digest(resource_policy),
                "reps": reps,
            }
        )
    unresolved = selected_set - {row["scenario_id"] for row in rows}
    if unresolved:
        raise OperatorError(f"selected scenarios did not resolve: {sorted(unresolved)}")
    if not rows:
        raise OperatorError("scenario selection resolved to zero admitted scenarios")
    return sorted(rows, key=lambda row: row["scenario_id"])


def candidate_identity(repo_root: Path) -> dict[str, str]:
    if not (repo_root / ".git").exists():
        raise OperatorError(f"candidate_root is not a git checkout: {repo_root}")

    def git_bytes(arguments: list[str]) -> bytes:
        process = subprocess.run(
            ["git", *arguments], cwd=repo_root, capture_output=True, check=False
        )
        if process.returncode != 0:
            raise OperatorError(
                f"cannot derive candidate identity with git {' '.join(arguments)}: "
                f"{process.stderr.decode(errors='replace').strip()}"
            )
        return process.stdout

    base_commit = git_bytes(["merge-base", "HEAD", "main"]).decode().strip()
    if not re.fullmatch(r"[0-9a-f]{40}", base_commit):
        raise OperatorError(
            "candidate identity base_commit is not a 40-character commit"
        )
    tracked_paths = git_bytes(["diff", "--name-only", "-z", "main...HEAD"])
    binary_diff = git_bytes(["diff", "--binary", "main...HEAD"])
    dirty_manifest = git_bytes(["status", "--porcelain=v1", "--untracked-files=all"])
    working_tree_diff = git_bytes(["diff", "--binary", "HEAD"])
    untracked_paths = git_bytes(["ls-files", "--others", "--exclude-standard", "-z"])
    untracked_material = bytearray()
    for relative in sorted(filter(None, untracked_paths.decode().split("\0"))):
        path = repo_root / relative
        untracked_material.extend(relative.encode("utf-8"))
        untracked_material.append(0)
        if path.is_symlink():
            untracked_material.extend(b"symlink\0")
            untracked_material.extend(os.readlink(path).encode("utf-8"))
        elif path.is_file():
            untracked_material.extend(b"file\0")
            untracked_material.extend(path.read_bytes())
        else:
            untracked_material.extend(b"non-file\0")
        untracked_material.append(0)
    tests = git_bytes(["ls-files", "-z", "--", "*_test.go", "**/*_test.go"])
    test_material = bytearray()
    for relative in sorted(filter(None, tests.decode().split("\0"))):
        test_material.extend(relative.encode("utf-8"))
        test_material.append(0)
        test_material.extend((repo_root / relative).read_bytes())
        test_material.append(0)
    components = {
        "base_commit": base_commit,
        "tree_digest": hashlib.sha256(
            git_bytes(["rev-parse", "HEAD^{tree}"])
        ).hexdigest(),
        "binary_diff_digest": hashlib.sha256(binary_diff).hexdigest(),
        "changed_path_digest": hashlib.sha256(tracked_paths).hexdigest(),
        "dirty_untracked_manifest": hashlib.sha256(dirty_manifest).hexdigest(),
        "working_tree_diff_digest": hashlib.sha256(working_tree_diff).hexdigest(),
        "untracked_content_digest": hashlib.sha256(
            bytes(untracked_material)
        ).hexdigest(),
        "test_suite_digest": hashlib.sha256(bytes(test_material)).hexdigest(),
    }
    components["identity_digest"] = canonical_digest(components)
    return components


def seed_scenario_root(
    scratch_binary: Path, scratch: Path, scenario_id: str, family: str
) -> str:
    """Seed a dedicated hierarchy for one scenario_id (REQ-F-012) and return
    its root_key -- the entity the lifecycle harness operates against.

    Every level is a fresh entity created just for this scenario_id and
    captured from Shark's own JSON response (AC-F11-40); no epic or feature
    created here is ever reused by another scenario_id (AC-F11-39): epic
    scenarios get a dedicated target epic; feature scenarios get a dedicated
    host epic plus target feature; task scenarios get a dedicated host epic,
    host feature, and target task; bug/change_card/tech_debt scenarios each
    get a dedicated standalone root.
    """
    label = f"E40 {scenario_id}"
    description = "Isolated E40 lifecycle benchmark root"

    def create(
        entity: str, positionals: list[str], flags: list[str], note: str
    ) -> dict[str, Any]:
        return shark_json(
            scratch_binary,
            scratch,
            ["create", entity, *positionals, *flags],
            f"seed {note} for scenario {scenario_id}",
        )

    if family == "epic":
        epic = create("epic", [f"{label} epic root"], ["--size", "S"], "epic root")
        return str(epic.get("key") or "")

    if family in ("feature", "task"):
        epic = create("epic", [f"{label} host epic"], ["--size", "S"], "host epic")
        epic_key = str(epic.get("key") or "")
        if family == "task":
            feature = create(
                "feature",
                [epic_key, f"{label} host feature"],
                ["--description", description, "--size", "S"],
                "host feature",
            )
            feature_key = str(feature.get("key") or "")
            task = create(
                "task",
                [feature_key, f"{label} target task"],
                ["--size", "S"],
                "target task",
            )
            return str(task.get("key") or "")
        feature = create(
            "feature",
            [epic_key, f"{label} target feature"],
            ["--description", description, "--size", "S"],
            "target feature",
        )
        return str(feature.get("key") or "")

    if family == "bug":
        bug = create(
            "bug",
            [f"{label} bug root"],
            ["--description", description, "--severity", "medium", "--size", "S"],
            "bug root",
        )
        return str(bug.get("key") or "")

    if family == "change_card":
        change = create(
            "change",
            [f"{label} change root"],
            ["--description", description, "--size", "S"],
            "change-card root",
        )
        return str(change.get("key") or "")

    if family == "tech_debt":
        tech_debt = create(
            "tech-debt",
            [f"{label} tech-debt root"],
            [
                "--description",
                description,
                "--category",
                "code-quality",
                "--severity",
                "medium",
                "--size",
                "S",
            ],
            "tech-debt root",
        )
        return str(tech_debt.get("key") or "")

    raise OperatorError(
        f"scenario {scenario_id} has unsupported entity_family for root seeding: {family!r}"
    )


def setup_config(
    operator_root: Path,
    scratch: Path,
    scenario_index_path: Path,
    scenario_roots: dict[str, Any],
) -> dict[str, Any]:
    return {
        "schema_version": "1.0",
        "operator_root": str(operator_root),
        "run_store": str(operator_root / "runs"),
        "comparison_store": str(operator_root / "comparisons"),
        "scenario_index": str(scenario_index_path),
        "candidate_root": str(REPO_ROOT),
        "scratch_template": str(scratch),
        "shark_binary": str(scratch / "shark"),
        "content_root": str(scratch / "shark-data"),
        "prompt_root": str(scratch / "shark-data" / "prompts"),
        "workflow_root": str(scratch / "shark-data" / "workflow"),
        "policy_root": str(scratch / "shark-data" / "workflow"),
        "runtime": {
            "lifecycle_adapter": None,
            "provider_command": None,
            "missing_real_runtime_inputs": [
                "runtime.lifecycle_adapter: executable accepting one run-lifecycle request on stdin",
            ],
        },
        "resource_policy": {
            "max_cost_usd": 5.0,
            "max_wall_clock_seconds": 900,
            "max_generated_tasks": 20,
            "max_provider_calls": 30,
        },
        "repetitions": 3,
        "scenario_roots": scenario_roots,
        "profile": {
            "name": "baseline",
            "role": "baseline",
            "change_axes": [],
            "baseline_definition_digest": None,
        },
    }


def cmd_setup(args: argparse.Namespace) -> int:
    operator_root = lexical_absolute_path(args.out)
    ensure_external_operator_root(operator_root)
    ensure_real_directory(operator_root.parent, "operator-root parent")
    if operator_root.is_symlink():
        raise OperatorError(f"operator root must not be a symlink: {operator_root}")
    lock = acquire_lock(operator_root.parent, f".{operator_root.name}.e40-setup.lock")
    try:
        return _cmd_setup_locked(args)
    finally:
        release_lock(lock)


def _cmd_setup_locked(args: argparse.Namespace) -> int:
    operator_root = lexical_absolute_path(args.out)
    ensure_external_operator_root(operator_root)
    if operator_root.is_symlink():
        raise OperatorError(f"operator root must not be a symlink: {operator_root}")
    config_path = (
        lexical_absolute_path(args.config_out)
        if args.config_out
        else operator_root / "e40-demo.yaml"
    )
    ensure_external_operator_root(config_path)
    if args.config_out:
        ensure_real_directory(config_path.parent, "benchmark config parent")
    if config_path.is_symlink():
        raise OperatorError(f"benchmark config must not be a symlink: {config_path}", 4)
    if config_path.exists():
        config = load_yaml(config_path, "existing benchmark config")
        if (
            resolve_path(str(config.get("operator_root")), config_path.parent)
            != operator_root
        ):
            raise OperatorError(
                f"existing config belongs to a different operator root: {config_path}"
            )
        if getattr(args, "scenario_index", None):
            # AC-F11-38/39: --scenario-index must not be silently accepted
            # and then dropped on the idempotent re-run path -- a caller
            # requesting a different index than the one this operator root
            # was actually seeded from gets a named refusal, never a quiet
            # "already_prepared" that keeps the old index in place.
            requested_scenario_index = lexical_absolute_path(args.scenario_index)
            existing_scenario_index = resolve_path(
                str(config.get("scenario_index") or DEFAULT_SCENARIO_INDEX),
                config_path.parent,
            )
            if requested_scenario_index != existing_scenario_index:
                raise OperatorError(
                    "existing config was prepared with a different scenario_index: "
                    f"requested {requested_scenario_index}, existing {existing_scenario_index}"
                )
        setup_result_path = operator_root / "setup-result.json"
        scratch_binary = operator_root / "scratch-template" / "shark"
        content_root = operator_root / "scratch-template" / "shark-data"
        if (
            not setup_result_path.is_file()
            or not scratch_binary.is_file()
            or not content_root.is_dir()
        ):
            raise OperatorError(
                f"existing config has no complete isolated setup: {operator_root}", 4
            )
        setup_result, _setup_bytes = load_json_no_follow(
            setup_result_path, "existing setup result", invalid_exit_code=4
        )
        recorded_binary = (setup_result.get("shark_binary") or {}).get("sha256")
        if recorded_binary != path_digest(
            scratch_binary, "existing scratch Shark binary"
        ):
            raise OperatorError(
                f"existing scratch Shark binary identity changed: {scratch_binary}", 4
            )
        if setup_result.get("content_bundle_digest") != path_digest(
            content_root, "existing installed Shark-data bundle"
        ):
            raise OperatorError(
                f"existing installed Shark-data identity changed: {content_root}", 4
            )
        print(
            json.dumps(
                {"status": "already_prepared", "config": str(config_path)},
                sort_keys=True,
            )
        )
        return 0
    if operator_root.exists() and any(operator_root.iterdir()):
        raise OperatorError(
            f"operator root already exists and is not empty: {operator_root}"
        )
    ensure_real_directory(operator_root.parent, "operator-root parent")
    staging_handle = secure_mkdtemp(
        operator_root.parent,
        f".{operator_root.name}.",
        "operator setup staging directory",
    )
    staging = staging_handle.path
    verify_staging_directory(staging_handle, "operator setup staging directory")
    scratch = staging / "scratch-template"
    ensure_real_directory(scratch, "operator scratch root")
    atomic_write(
        scratch / ".e40-scratch-root",
        b"isolated Shark benchmark scratch project; never use the live repository database\n",
        overwrite=False,
    )
    for command, label in (
        (["git", "init", "--initial-branch=main"], "initialize scratch Git repository"),
        (
            ["git", "config", "user.name", "E40 Benchmark"],
            "configure scratch Git user name",
        ),
        (
            ["git", "config", "user.email", "e40-benchmark@invalid.example"],
            "configure scratch Git user email",
        ),
        (["git", "add", ".e40-scratch-root"], "stage scratch identity marker"),
        (
            ["git", "commit", "-m", "Initialize isolated E40 scratch project"],
            "commit scratch identity marker",
        ),
    ):
        completed = run_process(command, cwd=scratch, env=isolated_environment())
        if completed.returncode != 0:
            raise OperatorError(
                f"{label} failed: {completed.stderr.strip() or completed.stdout.strip()}",
                1,
            )
    binary = REPO_ROOT / "bin" / "shark"
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise OperatorError(
            f"Shark binary is missing; run 'make shark' first: {binary}"
        )
    environment = isolated_environment()
    init = run_process(
        [
            str(binary),
            "admin",
            "init",
            "--non-interactive",
            "--db",
            str(scratch / "shark-tasks.db"),
        ],
        cwd=scratch,
        env=environment,
    )
    if init.returncode != 0:
        raise OperatorError(
            f"isolated Shark initialization failed: {init.stderr.strip()}", 1
        )
    install = run_process(
        [str(binary), "admin", "install-shark-data"], cwd=scratch, env=environment
    )
    if install.returncode != 0:
        raise OperatorError(
            f"isolated Shark-data installation failed: {install.stderr.strip()}", 1
        )
    shutil.copy2(binary, scratch / "shark")
    scratch_binary = scratch / "shark"

    # REQ-F-012/AC-F11-38/39: key setup roots by scenario_id, not by
    # entity_family, so same-family scenarios can never collide. The
    # scenario index is injectable (--scenario-index, defaulting to
    # bench/scenarios/scenarios.yaml) so a caller can seed a dedicated set
    # of packages through this production CLI rather than editing harness
    # internals.
    scenario_index_path = (
        lexical_absolute_path(args.scenario_index)
        if getattr(args, "scenario_index", None)
        else DEFAULT_SCENARIO_INDEX
    )
    final_root = operator_root
    final_scratch = final_root / "scratch-template"
    scenario_roots: dict[str, Any] = {}
    root_keys: dict[str, str] = {}
    for _package_path, package in scenario_packages(scenario_index_path):
        scenario_id = require_safe_id(
            str(package.get("scenario_id") or ""), "scenario_id"
        )
        family = str(package.get("entity_family") or "")
        root_key = seed_scenario_root(scratch_binary, scratch, scenario_id, family)
        scenario_roots[scenario_id] = {
            "root_key": root_key,
            "scratch_root": str(final_scratch),
            "i05_bundle_dir": None,
            "replay_result": None,
        }
        root_keys[scenario_id] = root_key
    config = setup_config(final_root, final_scratch, scenario_index_path, scenario_roots)
    staging_config = (
        staging / config_path.name
        if config_path.parent == operator_root
        else staging / "e40-demo.yaml"
    )
    write_yaml(staging_config, config, overwrite=False)
    version_process = run_process(
        [str(scratch_binary), "--version"], env=isolated_environment()
    )
    if version_process.returncode != 0:
        raise OperatorError(
            "cannot record the setup-owned Shark version: "
            + (version_process.stderr.strip() or version_process.stdout.strip()),
            1,
        )
    binary_version = (
        version_process.stdout or version_process.stderr
    ).strip() or UNAVAILABLE
    setup_record = {
        "schema_version": "1.0",
        "created_at": utc_now(),
        "operator_root": str(final_root),
        "scratch_root": str(final_scratch),
        "live_repository_database_touched": False,
        "shark_binary": {
            "path": str(final_scratch / "shark"),
            "sha256": path_digest(scratch_binary, "scratch Shark binary"),
            "version": binary_version,
        },
        "content_bundle_digest": path_digest(
            scratch / "shark-data", "installed Shark-data bundle"
        ),
        "prompt_bundle_digest": path_digest(
            scratch / "shark-data" / "prompts", "installed prompt bundle"
        ),
        "workflow_bundle_digest": path_digest(
            scratch / "shark-data" / "workflow", "installed workflow bundle"
        ),
        "enabled_gate_identity": {
            "source": "installed_workflow_bundle",
            "digest": path_digest(
                scratch / "shark-data" / "workflow",
                "installed enabled-gate workflow bundle",
            ),
        },
        "provider_routing_identity": workflow_routing_identity(
            scratch / "shark-data" / "workflow"
        ),
        "scenario_matrix": selected_matrix(
            config, operator_root, None, config["repetitions"]
        ),
        "root_keys": root_keys,
        "provider_ready": False,
        "missing_real_runtime_inputs": config["runtime"]["missing_real_runtime_inputs"],
    }
    write_json(staging / "setup-result.json", setup_record, overwrite=False)
    publish_staging_directory(
        staging_handle,
        operator_root,
        "operator setup staging directory",
        replace_empty=True,
    )
    actual_config = operator_root / staging_config.name
    if config_path != actual_config:
        if config_path.exists():
            raise OperatorError(
                f"external config destination already exists: {config_path}"
            )
        ensure_real_directory(config_path.parent, "external config parent")
        atomic_write(config_path, actual_config.read_bytes(), overwrite=False)
    print(
        json.dumps(
            {
                "status": "prepared_offline",
                "config": str(config_path),
                "operator_root": str(operator_root),
                "provider_ready": False,
                "missing_real_runtime_inputs": config["runtime"][
                    "missing_real_runtime_inputs"
                ],
            },
            sort_keys=True,
        )
    )
    return 0


def overlay_directory(source: Path, destination: Path, label: str) -> None:
    if not source.is_dir():
        raise OperatorError(f"{label} must be a directory: {source}")
    path_digest(source, label)
    ensure_real_directory(destination, label)
    shutil.copytree(source, destination, dirs_exist_ok=True, symlinks=False)


def requested_variant_overrides(args: argparse.Namespace) -> dict[str, Any]:
    result: dict[str, Any] = {
        "prompt": None,
        "workflow": None,
        "policy": None,
        "allow_identical_control": bool(args.allow_identical),
    }
    for key, label in (
        ("prompt", "variant prompt root"),
        ("workflow", "variant workflow root"),
        ("policy", "variant policy root"),
    ):
        raw = getattr(args, f"{key}_root")
        if raw:
            path = Path(raw).resolve()
            if not path.is_dir():
                raise OperatorError(f"{label} must be a directory: {path}")
            result[key] = {"path": str(path), "digest": path_digest(path, label)}
    return result


def cmd_validate_variant(args: argparse.Namespace) -> int:
    baseline_config_path = Path(args.config).resolve()
    baseline, config_base = config_context(baseline_config_path)
    operator_root = config_path_value(baseline, config_base, "operator_root")
    ensure_external_operator_root(operator_root)
    name = require_safe_id(args.variant_name, "variant-name")
    variants_root = operator_root / "variants"
    ensure_real_directory(variants_root, "variant store")
    lock = acquire_lock(variants_root, f".{name}.e40-variant.lock")
    try:
        return _cmd_validate_variant_locked(args)
    finally:
        release_lock(lock)


def _cmd_validate_variant_locked(args: argparse.Namespace) -> int:
    baseline_config_path = Path(args.config).resolve()
    baseline, config_base = config_context(baseline_config_path)
    operator_root = config_path_value(baseline, config_base, "operator_root")
    ensure_external_operator_root(operator_root)
    name = require_safe_id(args.variant_name, "variant-name")
    variant_root = operator_root / "variants" / name
    requested_overrides = requested_variant_overrides(args)
    if variant_root.exists():
        definition = variant_root / "variant-definition.json"
        if definition.is_file():
            existing, _definition_bytes = load_json_no_follow(
                definition, "existing variant definition", invalid_exit_code=4
            )
            expected_baseline = path_digest(baseline_config_path, "baseline config")
            if (
                existing.get("baseline_config_digest") == expected_baseline
                and existing.get("requested_overrides") == requested_overrides
            ):
                print(
                    json.dumps(
                        {"status": "already_validated", "definition": str(definition)},
                        sort_keys=True,
                    )
                )
                return 0
            raise OperatorError(
                f"variant name already belongs to different immutable inputs: {variant_root}",
                4,
            )
        raise OperatorError(
            f"variant root exists without a complete definition: {variant_root}"
        )
    baseline_scratch = config_path_value(baseline, config_base, "scratch_template")
    staging_parent = variant_root.parent
    ensure_real_directory(staging_parent, "variant staging parent")
    staging_handle = secure_mkdtemp(
        staging_parent, f".{name}.", "variant staging directory"
    )
    staging = staging_handle.path
    verify_staging_directory(staging_handle, "variant staging directory")
    variant_scratch = staging / "scratch-template"
    shutil.copytree(baseline_scratch, variant_scratch, symlinks=False)
    variant = copy.deepcopy(baseline)
    variant["operator_root"] = str(operator_root)
    variant["scratch_template"] = str(variant_root / "scratch-template")
    variant["shark_binary"] = str(variant_root / "scratch-template" / "shark")
    variant["content_root"] = str(variant_root / "scratch-template" / "shark-data")
    variant["prompt_root"] = str(
        variant_root / "scratch-template" / "shark-data" / "prompts"
    )
    variant["workflow_root"] = str(
        variant_root / "scratch-template" / "shark-data" / "workflow"
    )
    roots = variant.get("scenario_roots") or {}
    for entry in roots.values():
        if isinstance(entry, dict):
            entry["scratch_root"] = str(variant_root / "scratch-template")
    policy_destination = variant_root / "policy-bundle"
    if args.prompt_root:
        overlay_directory(
            Path(args.prompt_root).resolve(),
            variant_scratch / "shark-data" / "prompts",
            "variant prompt root",
        )
    if args.workflow_root:
        overlay_directory(
            Path(args.workflow_root).resolve(),
            variant_scratch / "shark-data" / "workflow",
            "variant workflow root",
        )
    if args.policy_root:
        overlay_directory(
            Path(args.policy_root).resolve(),
            staging / "policy-bundle",
            "variant policy root",
        )
        variant["policy_root"] = str(policy_destination)
    else:
        variant["policy_root"] = str(
            variant_root / "scratch-template" / "shark-data" / "workflow"
        )
    baseline_digests = {
        "prompt": path_digest(
            config_path_value(baseline, config_base, "prompt_root"),
            "baseline prompt root",
        ),
        "workflow": path_digest(
            config_path_value(baseline, config_base, "workflow_root"),
            "baseline workflow root",
        ),
        "policy": path_digest(
            config_path_value(baseline, config_base, "policy_root"),
            "baseline policy root",
        ),
    }
    variant_digests = {
        "prompt": path_digest(
            variant_scratch / "shark-data" / "prompts", "variant prompt root"
        ),
        "workflow": path_digest(
            variant_scratch / "shark-data" / "workflow", "variant workflow root"
        ),
        "policy": path_digest(staging / "policy-bundle", "variant policy root")
        if args.policy_root
        else path_digest(
            variant_scratch / "shark-data" / "workflow", "variant policy root"
        ),
    }
    for axis, supplied in (
        ("prompt", bool(args.prompt_root)),
        ("workflow", bool(args.workflow_root)),
        ("policy", bool(args.policy_root)),
    ):
        if (
            supplied
            and baseline_digests[axis] == variant_digests[axis]
            and not args.allow_identical
        ):
            raise OperatorError(
                f"the supplied {axis} override leaves the baseline {axis} digest unchanged"
            )
    axes = [
        axis
        for axis in ("prompt", "workflow", "policy")
        if baseline_digests[axis] != variant_digests[axis]
    ]
    baseline_routing = workflow_routing_identity(
        config_path_value(baseline, config_base, "workflow_root")
    )
    variant_routing = workflow_routing_identity(
        variant_scratch / "shark-data" / "workflow"
    )
    for axis in ("provider", "model", "effort"):
        if variant_routing[f"{axis}_digest"] != baseline_routing[f"{axis}_digest"]:
            axes.append(axis)
    if not axes and not args.allow_identical:
        raise OperatorError(
            "variant identity is identical to the baseline; pass --allow-identical only for an explicit control rerun"
        )
    variant["profile"] = {
        "name": name,
        "role": "variant",
        "change_axes": axes,
        "baseline_definition_digest": path_digest(
            baseline_config_path, "baseline config"
        ),
    }
    variant_config_staging = staging / "variant-config.yaml"
    write_yaml(variant_config_staging, variant, overwrite=False)
    definition = {
        "schema_version": "1.0",
        "variant_name": name,
        "created_at": utc_now(),
        "baseline_config": str(baseline_config_path),
        "baseline_config_digest": path_digest(baseline_config_path, "baseline config"),
        "change_axes": axes,
        "allow_identical_control": bool(args.allow_identical),
        "requested_overrides": requested_overrides,
        "baseline_digests": baseline_digests,
        "variant_digests": variant_digests,
        "baseline_provider_routing": baseline_routing,
        "variant_provider_routing": variant_routing,
        "variant_config": str(variant_root / "variant-config.yaml"),
    }
    definition["definition_digest"] = canonical_digest(definition)
    write_json(staging / "variant-definition.json", definition, overwrite=False)
    publish_staging_directory(
        staging_handle,
        variant_root,
        "variant staging directory",
        replace_empty=False,
    )
    print(
        json.dumps(
            {
                "status": "validated",
                "definition": str(variant_root / "variant-definition.json"),
                "config": str(variant_root / "variant-config.yaml"),
                "change_axes": axes,
            },
            sort_keys=True,
        )
    )
    return 0


# T-E40-F11-010 (spec.md REQ-F-004, §7.3.3): the seven independently
# falsifiable conditions of the fail-closed preflight gate. `PREFLIGHT_*`
# document the labels used in `blockers[]`/`diagnostics[]` entries so the
# gate's own code and its tests share one vocabulary with the spec.
PREFLIGHT_BLOCKED_EXIT = 1

# AC-F11-09/§7.3.3 P3: the only terminal outcome `run-lifecycle.sh --mode
# resolve-route` (or a live run) may reach that counts as a genuine success
# for gating purposes. Every other STOP_OUTCOMES value -- including `pause`
# and `unresolved_gate`, which resolve_route reports as `resolution:
# resolved` because tracing itself did not fail -- is a real blocker here:
# this is exactly the X-13 requirement that `unresolved_gate` never reads as
# provider-ready (spec.md Notes for Agent, T-E40-F11-010).
SUCCESS_TERMINAL_STATES = frozenset({"complete"})

# AC-F11-10/§7.1a row 10: the diagnostics[] type vocabulary is closed to
# exactly these five entries.
DIAGNOSTIC_TYPES = frozenset(
    {
        "dry_run_limitation",
        "missing_runtime_input",
        "unverified_fixture_checkout",
        "missing_ledger_record",
        "ceiling_exceeded",
    }
)


def replay_requirement_status(
    package_path: Path, package: dict[str, Any], operator_root: Path, scenario_id: str
) -> tuple[bool, str | None]:
    """AC-F11-09 P6/AC-F11-12: a scenario whose prelude has at least one
    D01-D05 stage that is `applicable` and not already answered by the
    package's own shipped `replay_reference` bundle needs a genuinely
    prepared, verified replay result (T-E40-F11-006's `prepare-replay`)
    before it can be reported provider-ready. Ordinary preflight never
    prepares one itself (spec.md Notes for Agent: "Ordinary preflight only
    validates an existing replay") -- it looks for the most recent attempt
    under `<operator_root>/replay/<scenario_id>/` and re-runs
    `verify-replay-result.sh` against it, exactly as `prepare-replay` did
    when it was created."""
    prelude = (package.get("stage_matrix") or {}).get("prelude") or {}
    if not isinstance(prelude, dict):
        prelude = {}
    already_replayed = replay_bundle_stage_ids(package_path, package)
    needs_replay = any(
        isinstance(entry, dict)
        and entry.get("applicable")
        and stage not in already_replayed
        for stage, entry in prelude.items()
    )
    if not needs_replay:
        return True, None

    replay_root = operator_root / "replay" / scenario_id
    if not replay_root.is_dir():
        return False, (
            f"scenario {scenario_id} requires a prepared replay result but "
            f"none exists under {replay_root} (run prepare-replay first)"
        )
    replay_reference = package.get("replay_reference")
    bundle_path = (
        (package_path.parent / str(replay_reference)).resolve()
        if replay_reference
        else None
    )
    attempts = sorted(
        (path for path in replay_root.iterdir() if path.is_dir()),
        key=lambda path: path.name,
        reverse=True,
    )
    verify_bin = SCRIPTS_DIR / "verify-replay-result.sh"
    for attempt_dir in attempts:
        result_path = attempt_dir / "result.json"
        if not result_path.is_file():
            continue
        if bundle_path is None:
            return True, None
        if not bundle_path.is_file():
            continue
        process = run_process([str(verify_bin), str(result_path), str(bundle_path)])
        if process.returncode == 0:
            return True, None
    return False, (
        f"scenario {scenario_id} has no prepared replay result under "
        f"{replay_root} that verifies against its replay_reference bundle"
    )


def evaluator_collection_status(
    package_path: Path, fixture_checkout: Path | None
) -> tuple[bool, str | None]:
    """AC-F11-09 P7 (evaluator half): re-run the REQ-F-010/F-011 isolation
    guard (`verify-evidence-roots.sh`) at preflight time, before any agent
    has touched the fixture. With no live dispatch yet, the "scratch
    project" argument the guard requires is the pristine admitted fixture
    checkout itself -- the same admission-time shape `admit-scenario.sh`
    exercises -- so this re-verifies that no evaluator-only material has
    leaked into the agent-visible fixture since admission, at zero
    provider cost."""
    # verify-evidence-roots.sh resolves each declared evaluator_only path
    # (e.g. "evaluator/reference.patch") against evaluator_root itself, so
    # evaluator_root is the PACKAGE root (package_path.parent), not its
    # "evaluator" subdirectory -- passing the subdirectory double-joins the
    # declared path's own leading "evaluator/" segment.
    evaluator_root = package_path.parent.resolve()
    if not (evaluator_root / "evaluator").is_dir():
        return True, None
    if fixture_checkout is None:
        return False, (
            "evaluator collection requires an admitted fixture checkout, "
            "but none is available"
        )
    guard = SCRIPTS_DIR / "verify-evidence-roots.sh"
    process = run_process(
        [
            str(guard),
            str(package_path),
            str(fixture_checkout),
            str(fixture_checkout),
            str(evaluator_root),
        ]
    )
    if process.returncode != 0:
        detail = (process.stderr or process.stdout).strip()
        return False, f"evaluator collection isolation guard rejected the roots: {detail}"
    return True, None


def runtime_readiness(
    config: dict[str, Any], config_base: Path, matrix: list[dict[str, Any]]
) -> tuple[bool, list[dict[str, Any]]]:
    """AC-F11-09 P4/P5: global provider plumbing (the adapter itself) and
    per-scenario I-05 evidence input. Returns `(ready, blockers)` where
    each blocker is a `{"requirement", "scenario_id", "cause", "detail"}`
    mapping -- `requirement` is `"P4"` (global) or `"P5"` (per-scenario) so
    a caller can classify and report each finding by its own cause without
    re-deriving it, and `ready` is `not blockers` (P4 and P5 combined),
    matching this function's pre-existing meaning and strictness for
    `execute_profile`'s own spend gate (pilot/baseline/variant) --
    unchanged by T-E40-F11-010. `runtime.provider_command` is checked
    separately, by `preflight_provider_command_finding`, called only from
    `_cmd_preflight_locked`: folding it in here would tighten
    `execute_profile`'s existing spend gate too, which is out of this
    task's scope and breaks scaffolds (e.g. tc094's symlinked-run_store
    case) that configure an adapter but not a provider_command and rely on
    reaching a later check. P6/P7 (replay requirement, fixture/evaluator
    isolation) are likewise `_cmd_preflight_locked`'s own additional
    findings, not folded in here."""
    blockers: list[dict[str, Any]] = []
    runtime = runtime_config(config)
    adapter = runtime.get("lifecycle_adapter")
    if not isinstance(adapter, str) or not adapter:
        blockers.append(
            {
                "requirement": "P4",
                "scenario_id": None,
                "cause": "provider_not_ready",
                "detail": "runtime.lifecycle_adapter is not configured",
            }
        )
    else:
        adapter_path = resolve_path(adapter, config_base)
        if not adapter_path.is_file() or not os.access(adapter_path, os.X_OK):
            blockers.append(
                {
                    "requirement": "P4",
                    "scenario_id": None,
                    "cause": "provider_not_ready",
                    "detail": f"runtime.lifecycle_adapter is not executable: {adapter_path}",
                }
            )

    roots = config.get("scenario_roots") or {}
    for row in matrix:
        scenario_id = row["scenario_id"]
        entry = roots.get(scenario_id) if isinstance(roots, dict) else None
        i05 = entry.get("i05_bundle_dir") if isinstance(entry, dict) else None
        if not isinstance(i05, str) or not i05:
            blockers.append(
                {
                    "requirement": "P5",
                    "scenario_id": scenario_id,
                    "cause": "missing_runtime_input",
                    "detail": f"scenario_roots.{scenario_id}.i05_bundle_dir is not configured",
                }
            )
        else:
            i05_path = resolve_path(i05, config_base)
            if i05_path.is_symlink() or not i05_path.is_dir():
                blockers.append(
                    {
                        "requirement": "P5",
                        "scenario_id": scenario_id,
                        "cause": "missing_runtime_input",
                        "detail": (
                            f"scenario_roots.{scenario_id}.i05_bundle_dir is not a "
                            f"real directory: {i05_path}"
                        ),
                    }
                )

    return not blockers, blockers


ROUTE_AWARE_LIFECYCLE_ADAPTERS = frozenset({"lifecycle-worker-adapter.sh"})


def is_route_aware_lifecycle_adapter(adapter: str) -> bool:
    """Return True if the configured lifecycle_adapter is known to resolve
    route providers without requiring runtime.provider_command."""
    return Path(adapter).name in ROUTE_AWARE_LIFECYCLE_ADAPTERS


def preflight_provider_command_finding(config: dict[str, Any]) -> dict[str, Any] | None:
    """AC-F11-09 P4 (preflight-only half) and AC-F13-10:
    When runtime.lifecycle_adapter is configured with a route-aware adapter
    (such as the shipped `lifecycle-worker-adapter.sh`), it resolves route
    providers internally and does not require runtime.provider_command.
    When a generic custom adapter is configured without runtime.provider_command,
    preflight returns a P4 blocker. An unconfigured adapter is reported by
    `runtime_readiness`."""
    runtime = runtime_config(config)
    adapter = runtime.get("lifecycle_adapter")
    if (
        isinstance(adapter, str)
        and adapter
        and not is_route_aware_lifecycle_adapter(adapter)
        and not runtime.get("provider_command")
    ):
        return {
            "requirement": "P4",
            "scenario_id": None,
            "cause": "provider_not_ready",
            "detail": "runtime.provider_command is not configured",
        }
    return None


def preflight_replay_and_evidence_findings(
    matrix: list[dict[str, Any]], operator_root: Path
) -> list[dict[str, Any]]:
    """AC-F11-09 P6/P7: replay requirement and fixture-checkout/evaluator-
    collection isolation. Preflight-only (never folded into
    `runtime_readiness`, which `execute_profile`'s pre-existing spend gate
    also calls) -- P6/P7 involve real subprocess work
    (`verify-replay-result.sh`, `verify-evidence-roots.sh`) that only
    `preflight`'s own zero-spend gate is specified to run (spec.md §7.7's
    permission table: "preflight (evaluator collection): allow")."""
    findings: list[dict[str, Any]] = []
    for row in matrix:
        scenario_id = row["scenario_id"]
        package_path = Path(row["package_path"])
        package = load_yaml(package_path, "scenario package")

        replay_ok, replay_reason = replay_requirement_status(
            package_path, package, operator_root, scenario_id
        )
        if not replay_ok:
            findings.append(
                {
                    "requirement": "P6",
                    "scenario_id": scenario_id,
                    "cause": "missing_replay",
                    "detail": replay_reason,
                }
            )

        fixture_checkout = row.get("fixture_checkout")
        fixture_required = row.get("fixture_id") not in (None, "", "None")
        if fixture_required and not fixture_checkout:
            findings.append(
                {
                    "requirement": "P7",
                    "scenario_id": scenario_id,
                    "cause": "fixture_identity_mismatch",
                    "detail": f"no admitted fixture checkout resolved for scenario {scenario_id}",
                }
            )
        elif fixture_checkout and not Path(fixture_checkout).is_dir():
            findings.append(
                {
                    "requirement": "P7",
                    "scenario_id": scenario_id,
                    "cause": "fixture_identity_mismatch",
                    "detail": f"admitted fixture checkout is not a real directory: {fixture_checkout}",
                }
            )
        else:
            evaluator_ok, evaluator_reason = evaluator_collection_status(
                package_path, Path(fixture_checkout) if fixture_checkout else None
            )
            if not evaluator_ok:
                findings.append(
                    {
                        "requirement": "P7",
                        "scenario_id": scenario_id,
                        "cause": "evaluator_collection_failure",
                        "detail": evaluator_reason,
                    }
                )

    return findings


def runtime_config(config: dict[str, Any]) -> dict[str, Any]:
    runtime = config.get("runtime")
    if runtime is None:
        return {}
    if not isinstance(runtime, dict):
        raise OperatorError("benchmark config runtime must be a YAML mapping")
    return runtime


def execution_input_identity(
    config: dict[str, Any], config_base: Path, matrix: list[dict[str, Any]]
) -> dict[str, Any]:
    runtime = config.get("runtime") or {}
    if not isinstance(runtime, dict):
        raise OperatorError("benchmark config runtime must be a mapping")
    adapter = runtime.get("lifecycle_adapter")
    if isinstance(adapter, str) and adapter:
        adapter_path = resolve_path(adapter, config_base)
        adapter_identity = {
            "path": str(adapter_path),
            "digest": path_digest(adapter_path, "runtime lifecycle adapter")
            if adapter_path.exists()
            else UNAVAILABLE,
        }
    else:
        adapter_identity = {"path": UNAVAILABLE, "digest": UNAVAILABLE}

    provider_command = runtime.get("provider_command")
    try:
        provider_command_digest = canonical_digest(provider_command)
    except (TypeError, ValueError) as exc:
        raise OperatorError(
            "runtime.provider_command must contain JSON-compatible values"
        ) from exc

    roots = config.get("scenario_roots") or {}
    bundles: list[dict[str, Any]] = []
    replay_results: list[dict[str, Any]] = []
    for row in matrix:
        entry = roots.get(row["scenario_id"]) if isinstance(roots, dict) else None
        raw = entry.get("i05_bundle_dir") if isinstance(entry, dict) else None
        if isinstance(raw, str) and raw:
            bundle_path = resolve_path(raw, config_base)
            digest = (
                path_digest(bundle_path, f"{row['scenario_id']} I-05 evidence bundle")
                if bundle_path.exists()
                else UNAVAILABLE
            )
            bundles.append(
                {
                    "scenario_id": row["scenario_id"],
                    "path": str(bundle_path),
                    "digest": digest,
                }
            )
        else:
            bundles.append(
                {
                    "scenario_id": row["scenario_id"],
                    "path": UNAVAILABLE,
                    "digest": UNAVAILABLE,
                }
            )
        raw_replay = entry.get("replay_result") if isinstance(entry, dict) else None
        if isinstance(raw_replay, str) and raw_replay:
            replay_path = resolve_path(raw_replay, config_base)
            replay_results.append({
                "scenario_id": row["scenario_id"],
                "path": str(replay_path),
                "digest": path_digest(replay_path, f"{row['scenario_id']} I-06 replay result")
                if replay_path.exists()
                else UNAVAILABLE,
            })
        else:
            replay_results.append({
                "scenario_id": row["scenario_id"],
                "path": UNAVAILABLE,
                "digest": UNAVAILABLE,
            })
    result = {
        "lifecycle_adapter": adapter_identity,
        "provider_command_digest": provider_command_digest,
        "i05_bundles": bundles,
        "i05_bundle_set_digest": canonical_digest(bundles),
        "i06_replay_results": replay_results,
        "i06_replay_set_digest": canonical_digest(replay_results),
    }
    result["execution_input_digest"] = canonical_digest(result)
    return result


def trusted_execution_scratch_root(
    config: dict[str, Any], config_base: Path, matrix: list[dict[str, Any]]
) -> Path:
    operator_root = config_path_value(config, config_base, "operator_root")
    scratch_template = config_path_value(config, config_base, "scratch_template")
    ensure_external_operator_root(operator_root)
    ensure_external_operator_root(scratch_template)
    require_existing_real_directory(operator_root, "operator root")
    require_existing_real_directory(scratch_template, "benchmark scratch template")
    if (
        operator_root != scratch_template
        and operator_root not in scratch_template.parents
    ):
        raise OperatorError(
            f"benchmark scratch template must remain inside its operator root: {scratch_template}"
        )

    setup_result_path = operator_root / "setup-result.json"
    if setup_result_path.is_symlink() or not setup_result_path.is_file():
        raise OperatorError(
            f"operator root has no regular setup authority: {setup_result_path}"
        )
    setup_result, _setup_bytes = load_json_no_follow(
        setup_result_path, "operator setup authority"
    )
    profile = config.get("profile") or {}
    role = profile.get("role") if isinstance(profile, dict) else None
    if role == "baseline":
        recorded_scratch = setup_result.get("scratch_root")
        if (
            not isinstance(recorded_scratch, str)
            or lexical_absolute_path(recorded_scratch) != scratch_template
        ):
            raise OperatorError(
                "baseline scratch template does not match setup-recorded authority: "
                f"{scratch_template}"
            )
    elif role == "variant":
        name = require_safe_id(str(profile.get("name") or ""), "variant profile name")
        variant_root = operator_root / "variants" / name
        expected_scratch = variant_root / "scratch-template"
        if scratch_template != expected_scratch:
            raise OperatorError(
                "variant scratch template is not the validated derivative: "
                f"expected {expected_scratch}, got {scratch_template}"
            )
        definition_path = variant_root / "variant-definition.json"
        if definition_path.is_symlink() or not definition_path.is_file():
            raise OperatorError(
                f"variant scratch has no regular validation authority: {definition_path}"
            )
        definition, _definition_bytes = load_json_no_follow(
            definition_path, "variant validation authority"
        )
        recorded_digest = definition.get("definition_digest")
        recomputed_digest = canonical_digest(
            {
                key: value
                for key, value in definition.items()
                if key != "definition_digest"
            }
        )
        if recorded_digest != recomputed_digest:
            raise OperatorError(
                f"variant validation authority digest changed: {definition_path}"
            )
        if definition.get("variant_name") != name or definition.get(
            "variant_config"
        ) != str(variant_root / "variant-config.yaml"):
            raise OperatorError(
                f"variant validation authority does not bind {variant_root}"
            )
    else:
        raise OperatorError(f"unsupported benchmark profile role: {role!r}")

    roots = config.get("scenario_roots") or {}
    for row in matrix:
        entry = roots.get(row["scenario_id"]) if isinstance(roots, dict) else None
        raw_scratch = (
            entry.get("scratch_root") if isinstance(entry, dict) else None
        ) or config.get("scratch_template")
        if not isinstance(raw_scratch, str) or not raw_scratch:
            raise OperatorError(
                f"scenario_roots.{row['scenario_id']}.scratch_root is missing"
            )
        scenario_scratch = resolve_path(raw_scratch, config_base)
        ensure_external_operator_root(scenario_scratch)
        require_existing_real_directory(
            scenario_scratch,
            f"scenario_roots.{row['scenario_id']}.scratch_root",
        )
        if scenario_scratch != scratch_template:
            raise OperatorError(
                f"scenario_roots.{row['scenario_id']}.scratch_root must equal the "
                f"trusted scratch template {scratch_template}, got {scenario_scratch}"
            )
    return scratch_template


def materialize_batch_policy(
    config: dict[str, Any],
    config_base: Path,
    matrix: list[dict[str, Any]],
    destination: Path,
    *,
    parent_descriptor: int | None = None,
) -> dict[str, Any]:
    trusted_scratch_root = trusted_execution_scratch_root(config, config_base, matrix)
    roots = config.get("scenario_roots") or {}
    policy: dict[str, Any] = {
        "schema_version": "1.0",
        "min_reps": matrix[0]["reps"],
        "scenario_index": str(
            resolve_path(
                str(config.get("scenario_index") or DEFAULT_SCENARIO_INDEX), config_base
            )
        ),
        "scenarios": {},
    }
    for row in matrix:
        entry = roots.get(row["scenario_id"]) if isinstance(roots, dict) else None
        if not isinstance(entry, dict):
            raise OperatorError(
                f"benchmark config is missing scenario_roots.{row['scenario_id']}"
            )
        root_key = entry.get("root_key")
        if not isinstance(root_key, str) or not root_key:
            raise OperatorError(
                f"scenario_roots.{row['scenario_id']}.root_key is missing"
            )
        policy_entry: dict[str, Any] = {
            "reps": row["reps"],
            "root_key": root_key,
            "scratch_root": str(trusted_scratch_root),
        }
        i05 = entry.get("i05_bundle_dir")
        if isinstance(i05, str) and i05:
            policy_entry["i05_bundle_dir"] = str(resolve_path(i05, config_base))
        replay_result = entry.get("replay_result")
        if isinstance(replay_result, str) and replay_result:
            policy_entry["replay_result"] = str(resolve_path(replay_result, config_base))
        policy["scenarios"][row["scenario_id"]] = policy_entry
    encoded = yaml.safe_dump(policy, sort_keys=False, allow_unicode=True).encode(
        "utf-8"
    )
    if parent_descriptor is not None:
        try:
            os.stat(destination.name, dir_fd=parent_descriptor, follow_symlinks=False)
        except FileNotFoundError:
            atomic_write_at(
                parent_descriptor,
                destination.name,
                encoded,
                label="prepared batch policy",
                overwrite=False,
            )
        else:
            if (
                load_bytes_at(
                    parent_descriptor,
                    destination.name,
                    "prepared batch policy",
                    invalid_exit_code=4,
                )
                != encoded
            ):
                raise OperatorError(
                    f"prepared batch policy identity changed; refusing overwrite: {destination}",
                    4,
                )
    elif destination.exists():
        if load_bytes_no_follow(destination, "prepared batch policy") != encoded:
            raise OperatorError(
                f"prepared batch policy identity changed; refusing overwrite: {destination}"
            )
    else:
        atomic_write(destination, encoded, overwrite=False)
    return policy


def trusted_shark_identity(config: dict[str, Any], config_base: Path) -> dict[str, str]:
    operator_root = config_path_value(config, config_base, "operator_root")
    scratch_root = config_path_value(config, config_base, "scratch_template")
    configured_binary = config_path_value(config, config_base, "shark_binary")
    expected_binary = (scratch_root / "shark").resolve()
    if configured_binary != expected_binary:
        raise OperatorError(
            "benchmark config shark_binary must be the setup-owned scratch binary: "
            f"expected {expected_binary}, got {configured_binary}"
        )
    if operator_root != scratch_root and operator_root not in scratch_root.parents:
        raise OperatorError(
            f"benchmark scratch root must remain inside its operator root: {scratch_root}"
        )
    setup_result, _setup_bytes = load_json_no_follow(
        operator_root / "setup-result.json", "operator setup result"
    )
    recorded = setup_result.get("shark_binary") or {}
    recorded_digest = recorded.get("sha256") if isinstance(recorded, dict) else None
    current_digest = path_digest(configured_binary, "setup-owned scratch Shark binary")
    if recorded_digest != current_digest:
        raise OperatorError(
            f"setup-owned scratch Shark binary identity changed: {configured_binary}"
        )
    version = recorded.get("version") if isinstance(recorded, dict) else None
    return {
        "path": str(configured_binary),
        "sha256": current_digest,
        "version": version if isinstance(version, str) and version else UNAVAILABLE,
    }


def preflight_environment(config: dict[str, Any], config_base: Path) -> dict[str, str]:
    environment = isolated_environment()
    environment["SHARK_BIN"] = trusted_shark_identity(config, config_base)["path"]
    environment["RUN_LIFECYCLE_BIN"] = str(SCRIPTS_DIR / "run-lifecycle.sh")
    environment["EVALUATE_LIFECYCLE_BIN"] = str(SCRIPTS_DIR / "evaluate-lifecycle.sh")
    environment["ENTITY_HISTORY_EXPORT_BIN"] = str(
        SCRIPTS_DIR / "export-entity-history.sh"
    )
    environment["LIFECYCLE_SCHEMA"] = str(DEFAULT_SCHEMA)
    return environment


def execution_environment(config: dict[str, Any], config_base: Path) -> dict[str, str]:
    environment = isolated_environment()
    environment["SHARK_BIN"] = trusted_shark_identity(config, config_base)["path"]
    runtime = runtime_config(config)
    if (
        isinstance(runtime.get("lifecycle_adapter"), str)
        and runtime["lifecycle_adapter"]
    ):
        environment["LIFECYCLE_ADAPTER"] = str(
            resolve_path(runtime["lifecycle_adapter"], config_base)
        )
    if runtime.get("provider_command"):
        environment["LIFECYCLE_PROVIDER_COMMAND"] = json.dumps(
            runtime["provider_command"]
        )
    return environment


def profile_name(config: dict[str, Any]) -> str:
    profile = config.get("profile") or {}
    return require_safe_id(str(profile.get("name") or "baseline"), "profile name")


def cmd_preflight(args: argparse.Namespace) -> int:
    config_path = Path(args.config).resolve()
    config, config_base = config_context(config_path)
    operator_root = config_path_value(config, config_base, "operator_root")
    output_root = (
        lexical_absolute_path(args.out)
        if args.out
        else operator_root / "preflight" / profile_name(config)
    )
    ensure_external_operator_root(output_root)
    ensure_real_directory(output_root, "preflight output root")
    lock = acquire_lock(output_root)
    try:
        return _cmd_preflight_locked(args)
    finally:
        release_lock(lock)


def evaluate_ledger_conditions(
    matrix: list[dict[str, Any]], ledger_records: dict[str, dict[str, Any]]
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    """AC-F11-09/AC-F11-11 P1/P2/P3, pure over an already-parsed ledger: a
    small, directly testable function so P1 (`missing_ledger_record`) has a
    seam reachable without depending on `run-lifecycle-batch.sh` itself
    ever actually omitting a record for a selected scenario -- which, by
    inspection of its own `emit_ledger_record` call sites, it never does on
    any of its current code paths. Returns `(blockers, diagnostics)`."""
    blockers: list[dict[str, Any]] = []
    diagnostics: list[dict[str, Any]] = []

    for row in matrix:
        scenario_id = row["scenario_id"]
        record = ledger_records.get(scenario_id)
        if record is None:
            detail = f"resolution-ledger.jsonl has no record for scenario {scenario_id}"
            blockers.append(
                {
                    "requirement": "P1",
                    "scenario_id": scenario_id,
                    "cause": "missing_ledger_record",
                    "detail": detail,
                }
            )
            diagnostics.append(
                {
                    "type": "missing_ledger_record",
                    "spend_eligible": False,
                    "scenario_id": scenario_id,
                    "detail": detail,
                }
            )
            continue

        resolution = record.get("resolution")
        cause_class = record.get("cause_class")
        if resolution != "resolved":
            # AC-F11-11: `resolution: skipped` (unconfigured_root /
            # missing_scratch_root) and `resolution: failed` (route_defect,
            # or the residual caller-path/output-contract class with
            # cause_class None) are both hard blockers here -- never a
            # silent omission.
            blockers.append(
                {
                    "requirement": "P2",
                    "scenario_id": scenario_id,
                    "cause": cause_class or "route_resolution_incomplete",
                    "detail": record.get("reason")
                    or f"scenario {scenario_id} did not resolve (resolution={resolution!r})",
                }
            )
            continue

        # P3 only applies once a scenario actually resolved -- an
        # unresolved scenario's terminal is not a fresh, independent
        # falsification of the same condition.
        terminal = record.get("terminal")
        if terminal not in SUCCESS_TERMINAL_STATES:
            blockers.append(
                {
                    "requirement": "P3",
                    "scenario_id": scenario_id,
                    "cause": "terminal_not_in_success_set",
                    "detail": (
                        f"scenario {scenario_id} resolved to terminal {terminal!r}, "
                        f"outside the declared success set {sorted(SUCCESS_TERMINAL_STATES)}"
                    ),
                }
            )

        if cause_class == "artifact_deferred":
            # AC-F11-10/AC-F11-14: an absent live-only artifact never fails
            # route resolution and never blocks `pass` -- reported as an
            # informational, spend-eligible diagnostic instead.
            diagnostics.append(
                {
                    "type": "dry_run_limitation",
                    "spend_eligible": True,
                    "scenario_id": scenario_id,
                    "detail": record.get("reason")
                    or "a live-worker-only artifact was deferred during route resolution",
                }
            )

    return blockers, diagnostics


def _cmd_preflight_locked(args: argparse.Namespace) -> int:
    config_path = Path(args.config).resolve()
    config, config_base = config_context(config_path)
    reps = int(
        require_positive(
            args.reps or config.get("repetitions", 1), "reps", integer=True
        )
    )
    selected = [args.scenario] if args.scenario else None
    operator_root = config_path_value(config, config_base, "operator_root")
    ensure_external_operator_root(operator_root)
    matrix = selected_matrix(
        config, config_base, selected, reps, operator_root / "fixture-checkouts"
    )
    output_root = (
        lexical_absolute_path(args.out)
        if args.out
        else operator_root / "preflight" / profile_name(config)
    )
    ensure_external_operator_root(output_root)
    ensure_real_directory(output_root, "preflight output root")
    batch_path = output_root / "batch-policy.yaml"
    materialize_batch_policy(config, config_base, matrix, batch_path)
    preview_bin = SCRIPTS_DIR / "run-lifecycle-batch.sh"
    command = [
        str(preview_bin),
        "--batch",
        str(batch_path),
        "--retention-root",
        str(output_root / "retention-preview"),
        "--mode",
        "preview",
        "--reps",
        str(reps),
    ]
    if args.scenario:
        command.extend(["--scenarios", args.scenario])
    process = run_process(
        command, cwd=REPO_ROOT, env=preflight_environment(config, config_base)
    )
    preview_text = process.stdout + process.stderr
    atomic_write(output_root / "preview.txt", preview_text.encode("utf-8"))

    # AC-F11-09/AC-F11-11/§2.3.2: the fail-closed gate consumes the
    # structured `resolution-ledger.jsonl` `run-lifecycle-batch.sh --mode
    # preview` wrote (T-E40-F11-007) -- never a regex scrape of the preview's
    # human-readable stdout/stderr, which cannot see a "skipped" record at
    # all (that was the exact 2026-09-04 defect class, AC-F11-11).
    ledger_path = output_root / "retention-preview" / "resolution-ledger.jsonl"
    ledger_records: dict[str, dict[str, Any]] = {}
    if ledger_path.is_file():
        for line in ledger_path.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line:
                continue
            record = json.loads(line)
            ledger_records[record["scenario_id"]] = record

    # §7.3.3 P1/P2/P3: every selected scenario must be present, resolved,
    # and terminate inside the declared success set.
    blockers, diagnostics = evaluate_ledger_conditions(matrix, ledger_records)

    # §7.3.3 P4/P5/P6/P7: provider plumbing, I-05 evidence input, replay
    # requirement, and fixture/evaluator isolation.
    ready, runtime_blockers = runtime_readiness(config, config_base, matrix)
    provider_command_finding = preflight_provider_command_finding(config)
    if provider_command_finding is not None:
        runtime_blockers = runtime_blockers + [provider_command_finding]
    runtime_blockers = runtime_blockers + preflight_replay_and_evidence_findings(
        matrix, operator_root
    )
    blockers.extend(runtime_blockers)
    for blocker in runtime_blockers:
        if blocker["requirement"] == "P5":
            diagnostics.append(
                {
                    "type": "missing_runtime_input",
                    "spend_eligible": False,
                    "scenario_id": blocker["scenario_id"],
                    "detail": blocker["detail"],
                }
            )
        elif blocker["cause"] == "fixture_identity_mismatch":
            diagnostics.append(
                {
                    "type": "unverified_fixture_checkout",
                    "spend_eligible": False,
                    "scenario_id": blocker["scenario_id"],
                    "detail": blocker["detail"],
                }
            )
    missing_real_runtime_inputs = [
        blocker["detail"] for blocker in runtime_blockers if blocker["requirement"] == "P5"
    ]

    if process.returncode != 0:
        status = "preview_failed"
    elif blockers:
        status = "blocked"
    else:
        status = "pass"

    configured_resources = config.get("resource_policy") or {}
    planned_resources = {
        "max_cost_usd": require_positive(
            configured_resources.get("max_cost_usd"), "resource_policy.max_cost_usd"
        ),
        "max_wall_clock_seconds": require_positive(
            configured_resources.get("max_wall_clock_seconds"),
            "resource_policy.max_wall_clock_seconds",
        ),
        "max_generated_tasks": require_positive(
            configured_resources.get("max_generated_tasks"),
            "resource_policy.max_generated_tasks",
            integer=True,
        ),
        "max_provider_calls": require_positive(
            configured_resources.get("max_provider_calls"),
            "resource_policy.max_provider_calls",
            integer=True,
        ),
        "repetitions": reps,
    }
    # REQ-F-006/§2.3.3: the conservative, evidence-backed provider-call plan
    # -- derived (and admission-rejected on a package/policy mismatch)
    # before any spend can be approved against it, independent of the
    # ledger-based blockers computed above.
    call_plan = derive_call_plan(config, config_base, matrix, planned_resources)
    result = {
        "schema_version": "1.0",
        # AC-F11-10: narrowed to exactly these three values --
        # `pass_with_dry_run_limitations` is no longer producible.
        "status": status,
        "provider_calls": 0,
        "live_database_mutations": 0,
        "profile": profile_name(config),
        "config": str(config_path),
        "config_digest": path_digest(config_path, "benchmark config"),
        "batch_policy": str(batch_path),
        "batch_policy_digest": path_digest(batch_path, "prepared batch policy"),
        "scenario_matrix": matrix,
        "planned_identity": identity_profile(
            config, config_base, matrix, planned_resources
        ),
        "call_plan": call_plan,
        "provider_ready": ready,
        "missing_real_runtime_inputs": missing_real_runtime_inputs,
        # AC-F11-09: every combination that is not `pass` names its
        # requirement, scenario, and cause here -- never a silent omission.
        "blockers": blockers,
        # AC-F11-10: the closed, typed vocabulary that replaces the single
        # `dry_run_limitation` string field.
        "diagnostics": diagnostics,
        "resolution_ledger": [
            ledger_records[row["scenario_id"]]
            for row in matrix
            if row["scenario_id"] in ledger_records
        ],
        "resolution_ledger_path": str(ledger_path),
        "preview_exit_code": process.returncode,
        "preview_output": str(output_root / "preview.txt"),
    }
    # AC-F11-10/§7.1a row 10: diagnostics[] is closed to exactly 5 named
    # types, and a `spend_eligible: false` diagnostic never coexists with
    # `status == "pass"` -- both are structural invariants of this
    # function's own construction above, asserted here so a future edit
    # that violates either fails loudly instead of silently shipping.
    for entry in diagnostics:
        if entry["type"] not in DIAGNOSTIC_TYPES:
            raise OperatorError(
                f"internal error: diagnostics[] type {entry['type']!r} is outside "
                f"the closed vocabulary {sorted(DIAGNOSTIC_TYPES)}"
            )
        if status == "pass" and not entry["spend_eligible"]:
            raise OperatorError(
                "internal error: a non-spend-eligible diagnostic coexists with "
                f"status == \"pass\": {entry!r}"
            )
    write_json(output_root / "preflight-result.json", result)
    if preview_text:
        sys.stdout.write(preview_text)
    print(json.dumps(result, sort_keys=True))
    if status == "preview_failed":
        return process.returncode
    if status == "blocked":
        return PREFLIGHT_BLOCKED_EXIT
    return 0


# REQ-F-003/AC-F11-06..08: prepare-replay's own refusal exit codes, kept
# distinct from require_positive's default OperatorError exit (2, shared
# with argparse) and from a genuine dispatch/verification failure (1) --
# spec.md §7.3.2 C1b/C1c requires the no-acknowledgement preview refusal to
# be distinguishable from an OperatorError, and re-using execute_profile's
# own "spend not yet authorized" exit (3, see its own
# "--acknowledge-provider-spend" check) keeps one vocabulary across both
# seams rather than inventing a second.
PREPARE_REPLAY_PREVIEW_EXIT = 3


def replay_bundle_stage_ids(package_path: Path, package: dict[str, Any]) -> set[str]:
    """§2.3.6/§2.3.3a: the set of D01-D05 stage ids that already carry at
    least one versioned response in the package's own replay_reference
    bundle -- read the same way run-prelude.sh's check_consistency (and
    replay-answer.sh's own stage lookup) do, never re-derived. A missing or
    unreadable replay_reference means "nothing is replayable yet", which is
    exactly the empty set, not an error: an all-non-applicable package
    legitimately carries no replay_reference at all (REQ-F-014)."""
    replay_reference = package.get("replay_reference")
    if not replay_reference or not str(replay_reference).strip():
        return set()
    bundle_path = (package_path.parent / str(replay_reference)).resolve()
    if not bundle_path.is_file():
        return set()
    try:
        bundle = json.loads(bundle_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return set()
    entries = bundle.get("entries") if isinstance(bundle, dict) else None
    if not isinstance(entries, list):
        return set()
    return {
        entry.get("stage")
        for entry in entries
        if isinstance(entry, dict) and entry.get("stage")
    }



# REQ-F-006/§2.3.3a (ADR-F11-12): family -> workflow YAML filename under
# shark-data/workflow/. Kebab-case filenames follow the canonical Shark 2.0
# layout (tech-debt.yaml, change.yaml), which does not match the
# package.yaml `entity_family` spelling (tech_debt, change_card) --
# internal/config/workflow/yaml_loader.go's own `yamlEntityFiles` table is
# the source of truth this mirrors, never re-invented.
FAMILY_WORKFLOW_FILENAMES = {
    "epic": "epic.yaml",
    "feature": "feature.yaml",
    "task": "task.yaml",
    "bug": "bug.yaml",
    "change_card": "change.yaml",
    "tech_debt": "tech-debt.yaml",
}

# AC-F11-18: every derived call_plan limit is evidenced by exactly these
# eight named fields.
CALL_PLAN_EVIDENCE_FIELDS = (
    "limit",
    "package_path",
    "package_digest",
    "workflow_file",
    "workflow_routing_digest",
    "model",
    "provider",
    "resource_policy_digest",
)


def family_workflow_path(workflow_root: Path, family: str) -> Path:
    filename = FAMILY_WORKFLOW_FILENAMES.get(family)
    if not filename:
        raise OperatorError(
            f"call_plan: no workflow file mapping for entity_family {family!r}"
        )
    path = workflow_root / filename
    if not path.is_file():
        raise OperatorError(
            f"call_plan: workflow file for family {family!r} does not exist: {path}"
        )
    return path


def family_workflow_steps(workflow_root: Path, family: str) -> dict[str, Any]:
    """§2.3.3a: the `steps:` mapping of `family`'s own resolved workflow
    file -- read from disk on every call, never cached or hardcoded, so a
    mutated workflow file changes `calls_per_entity` on the very next read
    (spec.md §7.1 row 17's negative case)."""
    path = family_workflow_path(workflow_root, family)
    document = load_yaml(path, f"{family} workflow file")
    steps = document.get("steps")
    if not isinstance(steps, dict):
        raise OperatorError(f"call_plan: {path} declares no steps: mapping")
    return steps


def calls_per_entity(workflow_root: Path, family: str) -> int:
    """§2.3.3a: the number of steps in `family`'s resolved workflow whose
    `action` is `spawn_agent` -- read from the workflow file identified by
    `call_plan.evidence[].workflow_file`, never a harness literal table.
    This matters because the benchmark compares candidate configurations
    whose workflows differ; a hardcoded count would silently misprice a
    variant."""
    return len(family_spawn_agent_step_names(workflow_root, family))


def family_gate_step_names(workflow_root: Path, family: str) -> set[str]:
    """The subset of `family`'s own steps carrying `result_contract:
    gate_result_v1` -- the only statically-named review-gate set available
    before any live run has produced a `workflow_policy.enabled_gates` list
    (that field is populated by run-lifecycle.sh's own dispatch, §2.5's
    `make_record`, and does not exist yet at preflight time)."""
    steps = family_workflow_steps(workflow_root, family)
    return {
        name
        for name, step in steps.items()
        if isinstance(step, dict) and step.get("result_contract") == "gate_result_v1"
    }


def family_spawn_agent_step_names(workflow_root: Path, family: str) -> set[str]:
    steps = family_workflow_steps(workflow_root, family)
    return {
        name
        for name, step in steps.items()
        if isinstance(step, dict) and step.get("action") == "spawn_agent"
    }


def aggregate_models_and_providers(
    workflow_root: Path, families: "list[str]"
) -> tuple[str, str]:
    """AC-F11-18's `model`/`provider` evidence fields: the distinct
    `model`/`provider` values named by the given families' own spawn_agent
    steps, joined deterministically when more than one is in play (an
    epic's own steps alone span haiku/sonnet/opus/gpt-5.6-terra) -- never a
    single hardcoded literal."""
    models: set[str] = set()
    providers: set[str] = set()
    for family in families:
        steps = family_workflow_steps(workflow_root, family)
        for step in steps.values():
            if isinstance(step, dict) and step.get("action") == "spawn_agent":
                models.add(str(step.get("model") or UNAVAILABLE))
                providers.add(str(step.get("provider") or UNAVAILABLE))
    return "+".join(sorted(models)) or UNAVAILABLE, "+".join(sorted(providers)) or UNAVAILABLE


def fixed_prelude_call_count(package_path: Path, package: dict[str, Any]) -> int:
    """§2.3.3a: D01-D05 prelude stages that are `applicable` and carry no
    versioned response in the package's replay bundle -- the same
    applicable-and-unreplayed set `cmd_prepare_replay`'s own
    `proposed_calls` computes, reused here rather than re-derived a second
    way. A package passing AC-F11-04's replay requirement always yields 0;
    a non-zero value here is itself a preflight inconsistency."""
    prelude = (package.get("stage_matrix") or {}).get("prelude") or {}
    if not isinstance(prelude, dict):
        return 0
    already_replayed = replay_bundle_stage_ids(package_path, package)
    return sum(
        1
        for stage, entry in prelude.items()
        if isinstance(entry, dict)
        and entry.get("applicable")
        and stage not in already_replayed
    )


def review_gate_call_count(workflow_root: Path, root_family: str) -> int:
    """§2.3.3a: `enabled_gates(workflow_policy_identity) \\
    spawn_agent_steps(root_family)` -- counting only review gates not
    already priced by `fixed_root_lifecycle_calls`, so no gate is
    double-counted. Statically (no live run has happened yet at preflight
    time), the only enabled-gate set this benchmark can name is
    `root_family`'s own `result_contract: gate_result_v1` steps, which are
    already members of `spawn_agent_steps(root_family)` by construction --
    the set difference is computed honestly rather than asserted as the
    literal 0 the worked example reports."""
    gates = family_gate_step_names(workflow_root, root_family)
    spawn_steps = family_spawn_agent_step_names(workflow_root, root_family)
    return len(gates - spawn_steps)


def bounded_descendant_call_count(
    workflow_root: Path,
    package_path: Path,
    package: dict[str, Any],
    resource_policy: dict[str, Any],
) -> tuple[int, list[str]]:
    """§2.3.3a: SUM(d.max * calls_per_entity(d.family)) over
    `expected_entity_graph.descendants[]`, summation not maximum -- or the
    `max_generated_tasks` fallback when the block is absent (AC-F11-25a).
    Returns (count, families-consulted). `resource_policy.max_generated_tasks`
    is always required (§7.3.4 B5/B6): its absence is a package/config
    defect, not a silent zero. When both sources are present and the
    graph's own task maxima exceed the ceiling, that is a package defect
    and admission rejects it (§2.3.3a "Both sources present") rather than
    one value silently overriding the other."""
    graph = package.get("expected_entity_graph")
    raw_max_generated_tasks = (
        resource_policy.get("max_generated_tasks")
        if isinstance(resource_policy, dict)
        else None
    )
    if raw_max_generated_tasks is None:
        raise OperatorError(
            f"scenario package {package_path}: resource_policy.max_generated_tasks "
            "is required to derive call_plan.bounded_descendant_calls"
        )
    max_generated_tasks = int(raw_max_generated_tasks)

    if isinstance(graph, dict) and graph.get("descendants"):
        descendants = [d for d in graph["descendants"] if isinstance(d, dict)]
        task_max = sum(
            int(d.get("max") or 0) for d in descendants if d.get("family") == "task"
        )
        if task_max > max_generated_tasks:
            raise OperatorError(
                f"scenario package {package_path}: expected_entity_graph declares up "
                f"to {task_max} task descendants, exceeding "
                f"resource_policy.max_generated_tasks={max_generated_tasks}"
            )
        total = sum(
            int(d.get("max") or 0)
            * calls_per_entity(workflow_root, str(d.get("family") or ""))
            for d in descendants
        )
        families = sorted({str(d.get("family")) for d in descendants if d.get("family")})
        return total, families

    # AC-F11-25a fallback: the block is absent, so calls_per_entity cannot be
    # resolved per family. max_generated_tasks counts generated tasks
    # specifically, so the fallback is bound to the "task" family.
    return max_generated_tasks * calls_per_entity(workflow_root, "task"), ["task"]


# AC-F11-17/AC-F11-25a: only `bounded_descendant_calls` is a ceiling
# ("labelled bound: true") -- the other three are exact counts derived from
# the package's own applicable-and-unreplayed prelude set, the root
# family's fixed step count, and the enabled-gate set difference
# (§2.3.3a), never presented as a bound.
CALL_PLAN_BOUNDED_LIMITS = frozenset({"bounded_descendant_calls", "worst_case_total_calls"})


def build_call_plan_evidence_entry(**fields: Any) -> dict[str, Any]:
    """AC-F11-18: every derived limit names all eight fields, each
    non-blank -- a limit whose evidence cannot be fully populated is a
    preflight defect, not a partially-evidenced ceiling. Also carries
    `bound` (AC-F11-17/-25a: "labelled bound: true"/"bound: false"),
    additive to the eight required fields, never a substitute for one of
    them."""
    entry = {name: fields.get(name) for name in CALL_PLAN_EVIDENCE_FIELDS}
    for name in CALL_PLAN_EVIDENCE_FIELDS:
        value = entry[name]
        if value is None or (isinstance(value, str) and not value.strip()):
            raise OperatorError(
                f"call_plan.evidence: field {name!r} is required and must be "
                f"non-blank (limit={fields.get('limit')!r})"
            )
    entry["bound"] = fields.get("limit") in CALL_PLAN_BOUNDED_LIMITS
    return entry


def validate_call_plan_schema(call_plan: dict[str, Any]) -> None:
    """AC-F11-16: `call_plan` names four per-scenario integers and four
    aggregate totals (plus the fourth-ceiling `max_provider_calls`,
    AC-F11-06a) -- a schema silently missing any one of them is rejected
    rather than shipped thin."""
    for scenario_id, entry in call_plan.get("per_scenario", {}).items():
        for field in (
            "fixed_prelude_calls",
            "fixed_root_lifecycle_calls",
            "review_gate_calls",
            "bounded_descendant_calls",
            "worst_case_total_calls",
        ):
            if field not in entry:
                raise OperatorError(
                    f"call_plan.per_scenario[{scenario_id!r}] is missing required "
                    f"field {field!r}"
                )
    totals = call_plan.get("totals", {})
    for field in (
        "worst_case_total_calls",
        "max_cost_usd",
        "max_wall_clock_seconds",
        "max_generated_tasks",
        "max_provider_calls",
    ):
        if field not in totals:
            raise OperatorError(f"call_plan.totals is missing required field {field!r}")


def derive_call_plan(
    config: dict[str, Any],
    config_base: Path,
    matrix: list[dict[str, Any]],
    resources: dict[str, Any],
) -> dict[str, Any]:
    """REQ-F-006/§2.3.3: builds `preflight-result.json.call_plan` -- the
    conservative, evidence-backed provider-call plan an operator approves
    spend against, per scenario and in aggregate (AC-F11-16)."""
    workflow_root = config_path_value(config, config_base, "workflow_root")
    routing_digest = workflow_routing_identity(workflow_root)["routing_digest"]
    per_scenario: dict[str, Any] = {}
    evidence: list[dict[str, Any]] = []
    worst_case_total = 0

    for row in matrix:
        package_path = Path(row["package_path"])
        package = load_yaml(package_path, "scenario package")
        family = str(row["family"])
        resource_policy = row.get("resource_policy") or {}
        package_digest = row["package_digest"]
        resource_policy_digest = row["resource_policy_digest"]

        fixed_prelude_calls = fixed_prelude_call_count(package_path, package)
        fixed_root_lifecycle_calls = calls_per_entity(workflow_root, family)
        review_gate_calls = review_gate_call_count(workflow_root, family)
        bounded_descendant_calls, descendant_families = bounded_descendant_call_count(
            workflow_root, package_path, package, resource_policy
        )
        scenario_worst_case = (
            fixed_prelude_calls
            + fixed_root_lifecycle_calls
            + review_gate_calls
            + bounded_descendant_calls
        )
        worst_case_total += scenario_worst_case

        per_scenario[row["scenario_id"]] = {
            "fixed_prelude_calls": fixed_prelude_calls,
            "fixed_root_lifecycle_calls": fixed_root_lifecycle_calls,
            "review_gate_calls": review_gate_calls,
            "bounded_descendant_calls": bounded_descendant_calls,
            "worst_case_total_calls": scenario_worst_case,
        }

        root_model, root_provider = aggregate_models_and_providers(
            workflow_root, [family]
        )
        descendant_model, descendant_provider = aggregate_models_and_providers(
            workflow_root, descendant_families
        )
        common = {
            "package_path": str(package_path),
            "package_digest": package_digest,
            "workflow_file": str(family_workflow_path(workflow_root, family)),
            "workflow_routing_digest": routing_digest,
            "resource_policy_digest": resource_policy_digest,
        }
        evidence.append(
            build_call_plan_evidence_entry(
                limit="fixed_prelude_calls",
                model=root_model,
                provider=root_provider,
                **common,
            )
        )
        evidence.append(
            build_call_plan_evidence_entry(
                limit="fixed_root_lifecycle_calls",
                model=root_model,
                provider=root_provider,
                **common,
            )
        )
        evidence.append(
            build_call_plan_evidence_entry(
                limit="review_gate_calls",
                model=root_model,
                provider=root_provider,
                **common,
            )
        )
        evidence.append(
            build_call_plan_evidence_entry(
                limit="bounded_descendant_calls",
                model=descendant_model,
                provider=descendant_provider,
                **common,
            )
        )
        # worst_case_total_calls is itself a ceiling (it sums a bounded
        # term), not an exact count -- AC-F11-17's "no field in call_plan
        # presents a bounded figure as an exact count" applies to it too,
        # so it is evidenced and labelled bound: true like
        # bounded_descendant_calls, never left unlabelled.
        evidence.append(
            build_call_plan_evidence_entry(
                limit="worst_case_total_calls",
                model=root_model,
                provider=root_provider,
                **common,
            )
        )

    call_plan = {
        "per_scenario": per_scenario,
        "totals": {
            "worst_case_total_calls": worst_case_total,
            "max_cost_usd": resources["max_cost_usd"],
            "max_wall_clock_seconds": resources["max_wall_clock_seconds"],
            "max_generated_tasks": resources["max_generated_tasks"],
            "max_provider_calls": resources["max_provider_calls"],
        },
        "evidence": evidence,
    }
    validate_call_plan_schema(call_plan)
    return call_plan


def cmd_prepare_replay(args: argparse.Namespace) -> int:
    config_path = Path(args.config).resolve()
    config, config_base = config_context(config_path)
    operator_root = config_path_value(config, config_base, "operator_root")
    # ADR-F11-10: prepare-replay inherits the retention-root guard by using
    # the same choke point every other operator seam resolves through --
    # never a private root check of its own.
    ensure_external_operator_root(operator_root)
    scenario_id = require_safe_id(args.scenario, "scenario")
    # cache_root=None: prepare-replay operates on the I-04 prelude only
    # (the X-10 product-design wrapper), which is not fixture-bound, so no
    # admitted-fixture checkout is resolved or cloned here (that is
    # preflight's and execute_profile's own concern for the fixture-bound
    # collector stages).
    matrix = selected_matrix(config, config_base, [scenario_id], 1, None)
    row = matrix[0]
    package_path = Path(row["package_path"])
    package = load_yaml(package_path, "scenario package")
    prelude = (package.get("stage_matrix") or {}).get("prelude") or {}
    if not isinstance(prelude, dict):
        raise OperatorError(f"scenario package has no stage_matrix.prelude: {package_path}")
    already_replayed = replay_bundle_stage_ids(package_path, package)
    # §2.3.6/§2.3.3a: reuses the fixed_prelude_calls term verbatim -- the
    # applicable-and-unreplayed D01-D05 stage set, sorted for a
    # deterministic preview rather than dict/set iteration order.
    proposed_calls = sorted(
        stage
        for stage, entry in prelude.items()
        if isinstance(entry, dict)
        and entry.get("applicable")
        and stage not in already_replayed
    )
    preview_dir = (
        lexical_absolute_path(args.out)
        if args.out
        else operator_root / "replay-preview" / scenario_id
    )
    ensure_external_operator_root(preview_dir)
    ensure_real_directory(preview_dir, "prepare-replay preview directory")
    preview = {
        "schema_version": "1.0",
        "scenario_id": scenario_id,
        "package_path": str(package_path),
        # Honest, not fabricated: run-prelude.sh's dispatch has no
        # configurable model/effort axis today (PROVIDER_BIN_NAME is the
        # only fixed identity), so those two are reported null rather than
        # invented.
        "provider": "claude",
        "model": None,
        "effort": None,
        "proposed_provider_calls": proposed_calls,
        "proposed_provider_call_count": len(proposed_calls),
        "resource_ceilings_required": list(RESOURCE_CEILING_NAMES),
    }
    write_json(preview_dir / "prepare-replay-preview.json", preview)

    if not args.acknowledge_provider_spend:
        # AC-F11-06: preview only, zero provider calls, writes nothing
        # outside preview_dir, exits non-zero -- distinct from both
        # argparse's exit 2 and a handler-required OperatorError (also 2),
        # per §7.3.2 C1c.
        print(json.dumps(preview, sort_keys=True))
        return PREPARE_REPLAY_PREVIEW_EXIT

    # AC-F11-06a/§7.2: the four ceilings are parser-optional but
    # handler-required once acknowledged -- validated with the same
    # require_positive() hardening (AC-F11-06b) every other seam uses, and
    # labelled resource_policy.<field> (not the execution seams' bare
    # "--max-*" labels) so a missing/invalid ceiling here reads the same
    # way preflight's own resource resolution does (§7.3.2 C1b).
    resources = {
        "max_cost_usd": require_positive(
            args.max_cost_usd, "resource_policy.max_cost_usd"
        ),
        "max_wall_clock_seconds": require_positive(
            args.max_wall_clock_seconds, "resource_policy.max_wall_clock_seconds"
        ),
        "max_generated_tasks": require_positive(
            args.max_generated_tasks,
            "resource_policy.max_generated_tasks",
            integer=True,
        ),
        "max_provider_calls": require_positive(
            args.max_provider_calls,
            "resource_policy.max_provider_calls",
            integer=True,
        ),
    }
    # §2.3.6: "[proposed_provider_calls] is the quantity this subcommand
    # spends against, and --max-provider-calls bounds it" -- validating the
    # ceiling's own shape (above) is not the same as enforcing it. Checked
    # before any dispatch is attempted (fail closed), so an operator-
    # approved ceiling can never be silently exceeded by this seam.
    if len(proposed_calls) > resources["max_provider_calls"]:
        raise OperatorError(
            f"prepare-replay: {len(proposed_calls)} proposed provider call(s) for "
            f"scenario {scenario_id} exceed resource_policy.max_provider_calls="
            f"{resources['max_provider_calls']}",
            3,
        )

    # AC-F11-07: retained under <operator_root>/replay/<scenario_id>/<attempt_id>/.
    replay_root = operator_root / "replay" / scenario_id
    ensure_external_operator_root(replay_root)
    ensure_real_directory(replay_root, "prepare-replay retention root")
    lock = acquire_lock(replay_root, ".e40-prepare-replay.lock")
    try:
        staging_handle = secure_mkdtemp(
            replay_root, "attempt-", "prepare-replay attempt directory"
        )
        verify_staging_directory(staging_handle, "prepare-replay attempt directory")
        attempt_dir = staging_handle.path
        artifact_root = attempt_dir / "artifact-root"
        ensure_real_directory(artifact_root, "prepare-replay artifact root")

        limits_path = attempt_dir / "limits.json"
        write_json(limits_path, resources, overwrite=False)

        result_path = attempt_dir / "result.json"
        transcript_path = attempt_dir / "transcript.txt"
        usage_path = attempt_dir / "usage.json"
        validation_path = attempt_dir / "validation.json"

        run_prelude_bin = SCRIPTS_DIR / "run-prelude.sh"
        started = time.monotonic()
        process = run_process(
            [
                str(run_prelude_bin),
                "--package",
                str(package_path),
                "--result-out",
                str(result_path),
                "--artifact-root",
                str(artifact_root),
            ],
            cwd=REPO_ROOT,
            env=isolated_environment(),
        )
        wall_clock_seconds = time.monotonic() - started
        transcript = process.stdout + process.stderr
        atomic_write(transcript_path, transcript.encode("utf-8"), overwrite=False)
        # AC-F11-07/§7.7: whether a dispatch was actually ATTEMPTED --
        # mirrors run-prelude.sh's own main() condition (any prelude stage
        # applicable), never proposed_calls: proposed_calls is a
        # preview-time estimate of calls still needed against the bundle,
        # but run-prelude.sh dispatches once whenever any stage is
        # applicable, whether or not that stage already carries a
        # versioned response (REQ-F-013's short-circuit is the only thing
        # that skips dispatch). Reporting proposed_calls here instead would
        # under-report a real dispatch against an already-fully-replayed
        # package. Real and measured, never the hardcoded provider_calls
        # literal preflight prints elsewhere.
        any_applicable_stage = any(
            isinstance(entry, dict) and entry.get("applicable")
            for entry in prelude.values()
        )
        usage = {
            "schema_version": "1.0",
            "wall_clock_seconds": wall_clock_seconds,
            "provider_calls": 1 if any_applicable_stage else 0,
            # Not tracked by this offline-capable flow: no live-provider
            # cost accounting exists yet, so this is left an honest null
            # rather than a fabricated figure (mirrors run-prelude.sh's own
            # write_complete_result placeholder discipline).
            "cost_usd": None,
        }
        write_json(usage_path, usage, overwrite=False)

        if process.returncode != 0 or not result_path.is_file():
            raise OperatorError(
                "prepare-replay: genuine replay producer failed for scenario "
                f"{scenario_id} (exit {process.returncode}): {transcript.strip()}",
                1,
            )

        replay_reference = package.get("replay_reference")
        if replay_reference:
            bundle_path = (package_path.parent / str(replay_reference)).resolve()
            verify_bin = SCRIPTS_DIR / "verify-replay-result.sh"
            verify_process = run_process(
                [str(verify_bin), str(result_path), str(bundle_path)]
            )
            validation_text = verify_process.stdout or verify_process.stderr
            atomic_write(
                validation_path, validation_text.encode("utf-8"), overwrite=False
            )
            if verify_process.returncode != 0:
                # AC-F11-07 negative case: a result failing verification
                # prints no path, on stdout or otherwise.
                raise OperatorError(
                    "prepare-replay: replay result failed verify-replay-result.sh "
                    f"for scenario {scenario_id}: {verify_process.stderr.strip()}",
                    1,
                )
        else:
            # No applicable prelude stage carries a replay_reference at all
            # (REQ-F-014) -- nothing for verify-replay-result.sh to reconcile.
            write_json(
                validation_path,
                {"scenario_id": scenario_id, "skipped": "no_replay_reference"},
                overwrite=False,
            )

        result_absolute = str(result_path.resolve())
    finally:
        release_lock(lock)

    print(result_absolute)
    return 0


def identity_profile(
    config: dict[str, Any],
    config_base: Path,
    matrix: list[dict[str, Any]],
    resources: dict[str, Any],
) -> dict[str, Any]:
    binary_identity = trusted_shark_identity(config, config_base)
    candidate_root = config_path_value(config, config_base, "candidate_root")
    scenario_set = [
        {
            key: row[key]
            for key in (
                "scenario_id",
                "scenario_version",
                "family",
                "package_digest",
                "fixture_id",
                "fixture_base_sha",
                "adapter_id",
                "adapter_version",
                "resource_policy_digest",
                "reps",
            )
        }
        for row in matrix
    ]
    workflow_root = config_path_value(config, config_base, "workflow_root")
    identity = {
        "scenario_set_digest": canonical_digest(scenario_set),
        "fixture_set_digest": canonical_digest(
            [(row["fixture_id"], row["fixture_base_sha"]) for row in matrix]
        ),
        "adapter_set_digest": canonical_digest(
            [
                (row["adapter_id"], row["adapter_version"], row["toolchain_identity"])
                for row in matrix
            ]
        ),
        "shark_binary": {
            "path": binary_identity["path"],
            "sha256": binary_identity["sha256"],
            "version": binary_identity["version"],
        },
        "content_bundle_digest": path_digest(
            config_path_value(config, config_base, "content_root"),
            "installed Shark-data bundle",
        ),
        "prompt_bundle_digest": path_digest(
            config_path_value(config, config_base, "prompt_root"), "prompt bundle"
        ),
        "workflow_bundle_digest": path_digest(
            config_path_value(config, config_base, "workflow_root"), "workflow bundle"
        ),
        "policy_bundle_digest": path_digest(
            config_path_value(config, config_base, "policy_root"), "policy bundle"
        ),
        "enabled_gate_identity": {
            "source": "workflow_bundle",
            "digest": path_digest(
                config_path_value(config, config_base, "workflow_root"),
                "enabled-gate workflow bundle",
            ),
        },
        "provider_routing": workflow_routing_identity(workflow_root),
        "execution_inputs": execution_input_identity(config, config_base, matrix),
        "resource_policy_digest": canonical_digest(resources),
        "candidate": candidate_identity(candidate_root),
    }
    identity["identity_digest"] = canonical_digest(identity)
    return identity


def read_lock_owner(lock_descriptor: int) -> str:
    try:
        descriptor = os.open(
            "pid",
            os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0),
            dir_fd=lock_descriptor,
        )
    except OSError:
        return "unknown"
    try:
        if not stat.S_ISREG(os.fstat(descriptor).st_mode):
            return "unknown"
        chunks: list[bytes] = []
        try:
            while True:
                chunk = os.read(descriptor, 1024)
                if not chunk:
                    break
                chunks.append(chunk)
                if sum(map(len, chunks)) > 64:
                    return "unknown"
        except OSError:
            return "unknown"
        return b"".join(chunks).decode("ascii", errors="replace").strip()
    finally:
        os.close(descriptor)


def open_lock_directory_fd(root_descriptor: int, name: str, label: str) -> int:
    return open_child_directory_fd(root_descriptor, name, label)


def create_and_open_lock_directory(root_descriptor: int, name: str, label: str) -> int:
    try:
        os.mkdir(name, 0o700, dir_fd=root_descriptor)
    except OSError as exc:
        raise OperatorError(f"cannot create {label}: {exc}", 4) from exc
    return open_lock_directory_fd(root_descriptor, name, label)


def write_lock_owner(lock_descriptor: int, owner_pid: int, label: str) -> None:
    try:
        descriptor = os.open(
            "pid",
            os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0),
            0o600,
            dir_fd=lock_descriptor,
        )
    except OSError as exc:
        raise OperatorError(f"cannot write {label} owner: {exc}", 4) from exc
    try:
        data = str(owner_pid).encode("ascii")
        written = 0
        while written < len(data):
            written += os.write(descriptor, data[written:])
        os.fsync(descriptor)
    finally:
        os.close(descriptor)


def acquire_lock(root: Path, name: str = ".e40-operator.lock") -> OperatorLock:
    if "/" in name or not name:
        raise OperatorError(f"operator lock has an invalid name: {name!r}", 4)
    root_path = lexical_absolute_path(root)
    lock_path = root_path / name
    root_descriptor = open_real_directory_fd(
        root_path, "operator lock root", create=True
    )
    lock_descriptor: int | None = None
    try:
        try:
            os.mkdir(name, 0o700, dir_fd=root_descriptor)
        except FileExistsError:
            lock_descriptor = open_lock_directory_fd(
                root_descriptor, name, "operator lock"
            )
            assert_directory_fd_matches_path(
                lock_descriptor, lock_path, "operator lock"
            )
            owner = read_lock_owner(lock_descriptor)
            if not owner.isdecimal():
                raise OperatorError(f"operator root has an unowned lock: {root}", 4)
            try:
                os.kill(int(owner), 0)
            except ProcessLookupError:
                entries = os.listdir(lock_descriptor)
                if entries != ["pid"]:
                    raise OperatorError(
                        f"stale operator lock contains unexpected entries: {lock_path}",
                        4,
                    )
                os.unlink("pid", dir_fd=lock_descriptor)
                os.fsync(lock_descriptor)
                assert_directory_fd_matches_path(
                    lock_descriptor, lock_path, "stale operator lock"
                )
                os.rmdir(name, dir_fd=root_descriptor)
                os.close(lock_descriptor)
                lock_descriptor = create_and_open_lock_directory(
                    root_descriptor, name, "operator lock"
                )
            except PermissionError:
                raise OperatorError(
                    f"operator root is locked by process {owner}: {root}", 4
                )
            else:
                raise OperatorError(
                    f"operator root is locked by process {owner}: {root}", 4
                )
        else:
            lock_descriptor = open_lock_directory_fd(
                root_descriptor, name, "operator lock"
            )

        assert_directory_fd_matches_path(
            root_descriptor, root_path, "operator lock root"
        )
        assert_directory_fd_matches_path(lock_descriptor, lock_path, "operator lock")
        owner_pid = os.getpid()
        write_lock_owner(lock_descriptor, owner_pid, "operator lock")
        os.fsync(lock_descriptor)
        os.fsync(root_descriptor)
        assert_directory_fd_matches_path(
            root_descriptor, root_path, "operator lock root"
        )
        assert_directory_fd_matches_path(lock_descriptor, lock_path, "operator lock")
        return OperatorLock(
            path=lock_path,
            root_path=root_path,
            name=name,
            root_descriptor=root_descriptor,
            lock_descriptor=lock_descriptor,
            owner_pid=owner_pid,
        )
    except BaseException:
        if lock_descriptor is not None:
            os.close(lock_descriptor)
        os.close(root_descriptor)
        raise


def release_lock(lock: OperatorLock) -> None:
    try:
        assert_directory_fd_matches_path(
            lock.root_descriptor, lock.root_path, "operator lock root"
        )
        assert_directory_fd_matches_path(
            lock.lock_descriptor, lock.path, "operator lock"
        )
        owner = read_lock_owner(lock.lock_descriptor)
        if owner != str(lock.owner_pid):
            raise OperatorError(
                f"operator lock ownership changed before release: {lock.path}", 4
            )
        entries = os.listdir(lock.lock_descriptor)
        if entries != ["pid"]:
            raise OperatorError(
                f"operator lock contains unexpected entries: {lock.path}", 4
            )
        os.unlink("pid", dir_fd=lock.lock_descriptor)
        os.fsync(lock.lock_descriptor)
        assert_directory_fd_matches_path(
            lock.lock_descriptor, lock.path, "operator lock"
        )
        os.rmdir(lock.name, dir_fd=lock.root_descriptor)
        os.fsync(lock.root_descriptor)
        assert_directory_fd_matches_path(
            lock.root_descriptor, lock.root_path, "operator lock root"
        )
    finally:
        os.close(lock.lock_descriptor)
        os.close(lock.root_descriptor)


def manifest_signature(manifest: dict[str, Any]) -> str:
    projection = {
        key: value
        for key, value in manifest.items()
        if key not in {"created_at", "command_arguments", "manifest_digest"}
    }
    return canonical_digest(projection)


def load_retained_manifest(
    path: Path, label: str, *, invalid_exit_code: int
) -> dict[str, Any]:
    manifest, _encoded = load_json_no_follow(
        path, label, invalid_exit_code=invalid_exit_code
    )
    return manifest


def require_valid_manifest_digest(
    manifest: dict[str, Any], path: Path, *, invalid_exit_code: int
) -> None:
    recorded = manifest.get("manifest_digest")
    recomputed = canonical_digest(
        {key: value for key, value in manifest.items() if key != "manifest_digest"}
    )
    if not isinstance(recorded, str) or recorded != recomputed:
        raise OperatorError(
            f"benchmark manifest digest is missing or changed: {path}",
            invalid_exit_code,
        )


def ensure_child_directory_at(parent_descriptor: int, name: str, label: str) -> int:
    if not name or "/" in name or name in {".", ".."}:
        raise OperatorError(f"{label} has an invalid child name: {name!r}")
    try:
        os.mkdir(name, 0o700, dir_fd=parent_descriptor)
        os.fsync(parent_descriptor)
    except FileExistsError:
        pass
    except OSError as exc:
        raise OperatorError(f"cannot create {label} {name!r}: {exc}") from exc
    return open_child_directory_fd(parent_descriptor, name, label)


def append_attempt(
    run_descriptor: int, run_root: Path, attempt: dict[str, Any]
) -> tuple[Path, bytes]:
    attempts_descriptor = ensure_child_directory_at(
        run_descriptor, "operator-attempts", "operator-attempts directory"
    )
    try:
        number = 1
        entries = set(os.listdir(attempts_descriptor))
        while f"{number:04d}.json" in entries:
            number += 1
        name = f"{number:04d}.json"
        encoded = (
            json.dumps(attempt, indent=2, sort_keys=True, ensure_ascii=False) + "\n"
        ).encode("utf-8")
        atomic_write_at(
            attempts_descriptor,
            name,
            encoded,
            label="operator attempt",
            overwrite=False,
        )
        return run_root / "operator-attempts" / name, encoded
    finally:
        os.close(attempts_descriptor)


def capture_batch_authority(
    run_descriptor: int,
    run_root: Path,
    manifest: dict[str, Any],
    attempt_path: Path,
    attempt: dict[str, Any],
    attempt_bytes: bytes,
    batch_policy_name: str,
) -> Path:
    batch, _batch_bytes = load_json_at(
        run_descriptor, "batch.json", "retained batch identity"
    )
    authority = batch_authority_projection(batch)
    batch_id = authority.get("batch_id")
    if not isinstance(batch_id, str):
        raise OperatorError("retained batch identity has no string batch_id")
    require_safe_id(batch_id, "retained batch_id")
    resources = manifest.get("resource_policy") or {}
    expected_authority = {
        "phase": manifest.get("phase"),
        "batch_id": batch_id,
        "mode": manifest.get("run_mode"),
        "retention_root": str(run_root),
        "batch_policy_digest": hashlib.sha256(
            load_bytes_at(run_descriptor, batch_policy_name, "operator batch policy")
        ).hexdigest(),
        "ceilings": {key: resources.get(key) for key in RESOURCE_CEILING_NAMES},
        "acknowledgement_ref": {
            "flag": "--acknowledge-provider-spend",
            "present": True,
        },
        "min_reps": resources.get("repetitions"),
    }
    if authority != expected_authority:
        raise OperatorError(
            "retained batch authority does not match the acknowledged operator manifest: "
            + json.dumps(
                {"expected": expected_authority, "observed": authority}, sort_keys=True
            )
        )
    attempt_relative = attempt_path.relative_to(run_root).as_posix()
    command = attempt.get("command") or []
    if not isinstance(command, list) or "--acknowledge-provider-spend" not in command:
        raise OperatorError(
            f"operator attempt does not retain provider-spend acknowledgement: {attempt_path}"
        )
    record = {
        "schema_version": "1.0",
        "manifest_digest": manifest.get("manifest_digest"),
        "attempt_path": attempt_relative,
        "attempt_digest": hashlib.sha256(attempt_bytes).hexdigest(),
        "batch_authority": authority,
    }
    record["authority_digest"] = canonical_digest(record)
    authorities_descriptor = ensure_child_directory_at(
        run_descriptor, "batch-authorities", "batch-authorities directory"
    )
    authorities = run_root / "batch-authorities"
    authority_path = authorities / f"{batch_id}.json"
    try:
        try:
            os.stat(
                authority_path.name,
                dir_fd=authorities_descriptor,
                follow_symlinks=False,
            )
        except FileNotFoundError:
            write_json_at(
                authorities_descriptor,
                authority_path.name,
                record,
                label="batch authority",
                overwrite=False,
            )
        else:
            existing, _encoded = load_json_at(
                authorities_descriptor,
                authority_path.name,
                "existing batch authority",
                invalid_exit_code=4,
            )
            if existing != record:
                raise OperatorError(
                    f"batch_id already belongs to a different immutable authority: {authority_path}",
                    4,
                )
        return authority_path
    finally:
        os.close(authorities_descriptor)


def aggregate_and_report(run_descriptor: int, run_root: Path) -> dict[str, Any]:
    aggregate_bin = SCRIPTS_DIR / "aggregate-lifecycle.sh"
    report_bin = SCRIPTS_DIR / "report-lifecycle.sh"
    verify_bin = SCRIPTS_DIR / "verify-retention-root.sh"
    assert_directory_fd_matches_path(run_descriptor, run_root, "operator run root")
    aggregate = run_process(
        [str(aggregate_bin), "--retention-root", str(run_root)],
        cwd=REPO_ROOT,
        env=aggregation_environment(),
    )
    assert_directory_fd_matches_path(run_descriptor, run_root, "operator run root")
    if aggregate.returncode != 0:
        return {
            "aggregate_exit_code": aggregate.returncode,
            "aggregate_error": aggregate.stderr.strip() or aggregate.stdout.strip(),
        }
    aggregate_path = run_root / "aggregate.json"
    atomic_write_at(
        run_descriptor,
        aggregate_path.name,
        aggregate.stdout.encode("utf-8"),
        label="retained aggregate",
    )
    # AC-F11-45 gate G-11: aggregate-lifecycle.sh exits 0 whether or not its
    # own `invalid[]` array is non-empty (a non-empty `invalid[]` is a
    # reporting finding, not a usage error) -- so publication eligibility
    # below must read this count explicitly rather than inferring it from
    # aggregate_exit_code, which cannot distinguish the two cases.
    try:
        aggregate_invalid_count = len(json.loads(aggregate.stdout).get("invalid") or [])
    except (json.JSONDecodeError, AttributeError):
        aggregate_invalid_count = None
    reports = run_root / "reports"
    reports_descriptor = ensure_child_directory_at(
        run_descriptor, reports.name, "retained reports directory"
    )
    report_results: dict[str, Any] = {}
    try:
        for view, filename in (
            ("headline", "headline.md"),
            ("stage_diagnostic", "stage-diagnostic.md"),
        ):
            assert_directory_fd_matches_path(
                run_descriptor, run_root, "operator run root"
            )
            rendered = run_process(
                [
                    str(report_bin),
                    "--aggregate",
                    str(aggregate_path),
                    "--view",
                    view,
                ],
                cwd=REPO_ROOT,
                env=aggregation_environment(),
            )
            assert_directory_fd_matches_path(
                run_descriptor, run_root, "operator run root"
            )
            report_results[f"{view}_exit_code"] = rendered.returncode
            if rendered.returncode == 0:
                atomic_write_at(
                    reports_descriptor,
                    filename,
                    rendered.stdout.encode("utf-8"),
                    label=f"retained {view} report",
                )
            else:
                report_results[f"{view}_error"] = (
                    rendered.stderr.strip() or rendered.stdout.strip()
                )
    finally:
        os.close(reports_descriptor)
    assert_directory_fd_matches_path(run_descriptor, run_root, "operator run root")
    verified = run_process(
        [
            str(verify_bin),
            "--retention-root",
            str(run_root),
            "--schema",
            str(DEFAULT_SCHEMA),
        ],
        cwd=REPO_ROOT,
        env=aggregation_environment(),
    )
    assert_directory_fd_matches_path(run_descriptor, run_root, "operator run root")
    atomic_write_at(
        run_descriptor,
        "retention-verification.jsonl",
        verified.stdout.encode("utf-8"),
        label="retention verification output",
    )
    if verified.stderr:
        atomic_write_at(
            run_descriptor,
            "retention-verification.stderr",
            verified.stderr.encode("utf-8"),
            label="retention verification diagnostics",
        )
    return {
        "aggregate_exit_code": 0,
        "aggregate": str(aggregate_path),
        "aggregate_invalid_count": aggregate_invalid_count,
        "reports": report_results,
        "retention_verification_exit_code": verified.returncode,
    }


def execute_profile(args: argparse.Namespace, *, role: str, run_mode: str) -> int:
    config_path = Path(args.config).resolve()
    config, config_base = config_context(config_path)
    profile = config.get("profile") or {}
    configured_role = str(profile.get("role") or "baseline")
    if role == "variant" and configured_role != "variant":
        raise OperatorError(
            "variant execution requires a variant-config.yaml from validate-variant"
        )
    if role != "variant" and configured_role == "variant":
        raise OperatorError("baseline/pilot execution cannot use a variant profile")
    run_id = require_safe_id(args.run_id, "run-id")
    reps = int(
        require_positive(
            args.reps or config.get("repetitions", 1), "reps", integer=True
        )
    )
    selected = [args.scenario] if args.scenario else None
    operator_root = config_path_value(config, config_base, "operator_root")
    # REQ-F-001: reject a registered (retained) operator root BEFORE
    # admitted_fixture_checkout() (inside selected_matrix) can write a
    # fixture checkout under it -- ensure_external_operator_root is the one
    # choke point (ADR-F11-10), and it must run before anything is written,
    # not only before run_root is created below.
    ensure_external_operator_root(operator_root)
    matrix = selected_matrix(
        config, config_base, selected, reps, operator_root / "fixture-checkouts"
    )
    if not args.acknowledge_provider_spend:
        raise OperatorError(
            "provider-backed execution requires --acknowledge-provider-spend", 3
        )
    ready, blockers = runtime_readiness(config, config_base, matrix)
    if not ready:
        raise OperatorError(
            "provider-backed execution is not ready; run preflight and supply: "
            + "; ".join(blocker["detail"] for blocker in blockers),
            3,
        )
    resources = {
        "max_cost_usd": require_positive(args.max_cost_usd, "max-cost-usd"),
        "max_wall_clock_seconds": require_positive(
            args.max_wall_clock_seconds, "max-wall-clock-seconds"
        ),
        "max_generated_tasks": require_positive(
            args.max_generated_tasks, "max-generated-tasks", integer=True
        ),
        "max_provider_calls": require_positive(
            args.max_provider_calls, "max-provider-calls", integer=True
        ),
        "repetitions": reps,
    }
    run_store = resolve_path(
        str(config.get("run_store") or operator_root / "runs"), config_base
    )
    run_root = lexical_absolute_path(args.out) if args.out else run_store / run_id
    ensure_external_operator_root(run_root)
    ensure_real_directory(run_root, "operator run root")
    lock = acquire_lock(run_root)
    try:
        identity = identity_profile(config, config_base, matrix, resources)
        change_axes = profile.get("change_axes") or []
        if not isinstance(change_axes, list) or any(
            axis not in CHANGE_AXES for axis in change_axes
        ):
            raise OperatorError(
                f"profile.change_axes contains an unsupported axis: {change_axes!r}"
            )
        manifest = {
            "schema_version": "1.0",
            "phase": "lifecycle_v2",
            "role": role,
            "run_mode": run_mode,
            "run_id": run_id,
            "created_at": utc_now(),
            "profile": profile_name(config),
            "identity": identity,
            "scenario_matrix": matrix,
            "resource_policy": resources,
            "comparison_boundary": {
                "allowed_change_axes": change_axes,
                "baseline_definition_digest": profile.get("baseline_definition_digest"),
            },
            "retention_root": str(run_root),
            "command_arguments": sys.argv[1:],
            "config": str(config_path),
            "config_digest": path_digest(config_path, "benchmark config"),
        }
        manifest["manifest_digest"] = canonical_digest(
            {key: value for key, value in manifest.items() if key != "manifest_digest"}
        )
        manifest_path = run_root / (
            "pilot-benchmark-manifest.json"
            if run_mode == "pilot"
            else "benchmark-manifest.json"
        )
        try:
            os.stat(
                manifest_path.name,
                dir_fd=lock.root_descriptor,
                follow_symlinks=False,
            )
        except FileNotFoundError:
            write_json_at(
                lock.root_descriptor,
                manifest_path.name,
                manifest,
                label="benchmark manifest",
                overwrite=False,
            )
        else:
            existing, _manifest_bytes = load_json_at(
                lock.root_descriptor,
                manifest_path.name,
                "existing benchmark manifest",
                invalid_exit_code=4,
            )
            require_valid_manifest_digest(existing, manifest_path, invalid_exit_code=4)
            if manifest_signature(existing) != manifest_signature(manifest):
                raise OperatorError(
                    f"run-id already belongs to a different immutable identity: {manifest_path}",
                    4,
                )
        batch_path = run_root / (
            "pilot-batch-policy.yaml" if run_mode == "pilot" else "batch-policy.yaml"
        )
        materialize_batch_policy(
            config,
            config_base,
            matrix,
            batch_path,
            parent_descriptor=lock.root_descriptor,
        )
        driver_bin = SCRIPTS_DIR / "run-lifecycle-batch.sh"
        descriptor_root = f"/proc/self/fd/{lock.root_descriptor}"
        command = [
            str(driver_bin),
            "--batch",
            f"{descriptor_root}/{batch_path.name}",
            "--retention-root",
            str(run_root),
            "--retention-root-fd",
            str(lock.root_descriptor),
            "--mode",
            run_mode,
            "--max-cost-usd",
            str(resources["max_cost_usd"]),
            "--max-wall-clock-seconds",
            str(resources["max_wall_clock_seconds"]),
            "--max-generated-tasks",
            str(resources["max_generated_tasks"]),
            "--max-provider-calls",
            str(resources["max_provider_calls"]),
            "--reps",
            str(reps),
        ]
        if args.acknowledge_provider_spend:
            command.append("--acknowledge-provider-spend")
        if args.scenario:
            command.extend(["--scenarios", args.scenario])
        if args.retry_incomplete:
            command.append("--reclaim-incomplete")
        started = utc_now()
        assert_directory_fd_matches_path(
            lock.root_descriptor, run_root, "operator run root"
        )
        process = run_process(
            command,
            cwd=REPO_ROOT,
            env=execution_environment(config, config_base),
            pass_fds=(lock.root_descriptor,),
        )
        assert_directory_fd_matches_path(
            lock.root_descriptor, run_root, "operator run root"
        )
        attempt = {
            "started_at": started,
            "completed_at": utc_now(),
            "command": command,
            "driver_exit_code": process.returncode,
            "stdout": process.stdout,
            "stderr": process.stderr,
        }
        attempt_path, attempt_bytes = append_attempt(
            lock.root_descriptor, run_root, attempt
        )
        derived: dict[str, Any] = {}
        authority_path: Path | None = None
        try:
            batch_status = os.stat(
                "batch.json", dir_fd=lock.root_descriptor, follow_symlinks=False
            )
        except FileNotFoundError:
            batch_status = None
        if (
            process.returncode in {0, 4}
            and batch_status is not None
            and stat.S_ISREG(batch_status.st_mode)
        ):
            authority_path = capture_batch_authority(
                lock.root_descriptor,
                run_root,
                manifest,
                attempt_path,
                attempt,
                attempt_bytes,
                batch_path.name,
            )
        try:
            scenarios_status = os.stat(
                "scenarios", dir_fd=lock.root_descriptor, follow_symlinks=False
            )
        except FileNotFoundError:
            scenarios_status = None
        if (
            process.returncode in {0, 4}
            and scenarios_status is not None
            and stat.S_ISDIR(scenarios_status.st_mode)
        ):
            derived = aggregate_and_report(lock.root_descriptor, run_root)
        if authority_path is not None:
            derived["batch_authority"] = str(authority_path)
        result = {
            "schema_version": "1.0",
            "run_id": run_id,
            "role": role,
            "run_mode": run_mode,
            "status": "completed"
            if process.returncode == 0
            else "incomplete_or_refused",
            "driver_exit_code": process.returncode,
            "manifest": str(manifest_path),
            "attempt": str(attempt_path),
            "retention_root": str(run_root),
            "derived": derived,
            "publication_eligible": bool(
                process.returncode == 0
                and derived.get("aggregate_exit_code") == 0
                and derived.get("aggregate_invalid_count") == 0
                and derived.get("retention_verification_exit_code") == 0
                and authority_path is not None
                and run_mode == "baseline"
            ),
        }
        assert_directory_fd_matches_path(
            lock.root_descriptor, run_root, "operator run root"
        )
        write_json_at(
            lock.root_descriptor,
            "operator-result.json",
            result,
            label="operator result",
        )
        if process.stdout:
            sys.stdout.write(process.stdout)
        if process.stderr:
            sys.stderr.write(process.stderr)
        print(json.dumps(result, sort_keys=True))
        if process.returncode != 0:
            return process.returncode
        if derived and derived.get("aggregate_exit_code") not in {None, 0}:
            return 1
        return 0
    finally:
        release_lock(lock)


def cmd_pilot(args: argparse.Namespace) -> int:
    resolve_named_variant(args)
    profile = (
        load_yaml(Path(args.config).resolve(), "benchmark config").get("profile") or {}
    )
    role = "variant" if profile.get("role") == "variant" else "baseline"
    return execute_profile(args, role=role, run_mode="pilot")


def cmd_baseline(args: argparse.Namespace) -> int:
    return execute_profile(args, role="baseline", run_mode="baseline")


def cmd_variant(args: argparse.Namespace) -> int:
    resolve_named_variant(args)
    return execute_profile(args, role="variant", run_mode="baseline")


def resolve_named_variant(args: argparse.Namespace) -> None:
    if args.variant_name:
        baseline_config_path = Path(args.config).resolve()
        baseline, config_base = config_context(baseline_config_path)
        operator_root = config_path_value(baseline, config_base, "operator_root")
        variant_config = (
            operator_root
            / "variants"
            / require_safe_id(args.variant_name, "variant-name")
            / "variant-config.yaml"
        )
        if not variant_config.is_file():
            raise OperatorError(
                f"validated variant does not exist; run validate-variant first: {variant_config}"
            )
        args.config = str(variant_config)


def nested(value: dict[str, Any], path: str, default: Any = None) -> Any:
    current: Any = value
    for part in path.split("."):
        if not isinstance(current, dict) or part not in current:
            return default
        current = current[part]
    return current


def comparison_boundary(
    baseline: dict[str, Any], variant: dict[str, Any]
) -> tuple[list[dict[str, Any]], list[str]]:
    reasons: list[dict[str, Any]] = []
    for side, manifest, expected_role in (
        ("baseline", baseline, "baseline"),
        ("variant", variant, "variant"),
    ):
        for path in (
            "manifest_digest",
            "config_digest",
            "identity.identity_digest",
            "identity.scenario_set_digest",
            "identity.fixture_set_digest",
            "identity.adapter_set_digest",
            "identity.shark_binary.sha256",
            "identity.content_bundle_digest",
            "identity.prompt_bundle_digest",
            "identity.workflow_bundle_digest",
            "identity.policy_bundle_digest",
            "identity.provider_routing.routing_digest",
            "identity.provider_routing.provider_digest",
            "identity.provider_routing.model_digest",
            "identity.provider_routing.effort_digest",
            "identity.execution_inputs.lifecycle_adapter.digest",
            "identity.execution_inputs.provider_command_digest",
            "identity.execution_inputs.i05_bundle_set_digest",
            "identity.execution_inputs.i06_replay_set_digest",
            "identity.execution_inputs.execution_input_digest",
            "identity.resource_policy_digest",
            "identity.candidate.identity_digest",
        ):
            value = nested(manifest, path)
            if not isinstance(value, str) or not re.fullmatch(r"[0-9a-f]{64}", value):
                reasons.append(
                    {
                        "path": f"{side}/{path}",
                        "reason": "missing_or_malformed_identity",
                    }
                )
        if manifest.get("phase") != "lifecycle_v2":
            reasons.append({"path": f"{side}/phase", "reason": "wrong_benchmark_phase"})
        if manifest.get("role") != expected_role:
            reasons.append(
                {
                    "path": f"{side}/role",
                    "reason": "wrong_comparison_role",
                    "expected": expected_role,
                    "observed": manifest.get("role"),
                }
            )
        if manifest.get("run_mode") != "baseline":
            reasons.append(
                {"path": f"{side}/run_mode", "reason": "pilot_cannot_enter_comparison"}
            )
        recorded_digest = manifest.get("manifest_digest")
        recomputed_digest = canonical_digest(
            {key: value for key, value in manifest.items() if key != "manifest_digest"}
        )
        if isinstance(recorded_digest, str) and recorded_digest != recomputed_digest:
            reasons.append(
                {
                    "path": f"{side}/manifest_digest",
                    "reason": "manifest_digest_mismatch",
                }
            )
    allowed = nested(variant, "comparison_boundary.allowed_change_axes", [])
    if not isinstance(allowed, list) or any(
        axis not in CHANGE_AXES for axis in allowed
    ):
        reasons.append(
            {
                "path": "variant/comparison_boundary/allowed_change_axes",
                "reason": "unsupported_change_axis",
            }
        )
        allowed = []
    baseline_definition = nested(
        variant, "comparison_boundary.baseline_definition_digest"
    )
    if baseline_definition != baseline.get("config_digest"):
        reasons.append(
            {
                "path": "variant/comparison_boundary/baseline_definition_digest",
                "reason": "variant_not_linked_to_baseline_definition",
                "expected": baseline.get("config_digest"),
                "observed": baseline_definition,
            }
        )
    always_equal = [
        "phase",
        "identity.scenario_set_digest",
        "identity.fixture_set_digest",
        "identity.adapter_set_digest",
        "identity.execution_inputs.execution_input_digest",
        "identity.shark_binary.sha256",
        "identity.resource_policy_digest",
        "identity.candidate.base_commit",
        "identity.candidate.tree_digest",
        "identity.candidate.binary_diff_digest",
        "identity.candidate.changed_path_digest",
        "identity.candidate.dirty_untracked_manifest",
        "identity.candidate.working_tree_diff_digest",
        "identity.candidate.untracked_content_digest",
        "identity.candidate.test_suite_digest",
        "identity.candidate.identity_digest",
        "scenario_matrix",
        "resource_policy",
    ]
    for path in always_equal:
        left, right = nested(baseline, path), nested(variant, path)
        if left != right:
            reasons.append(
                {
                    "path": path,
                    "reason": "comparison_boundary_mismatch",
                    "baseline": left,
                    "variant": right,
                }
            )
    axis_fields = {
        "prompt": ["identity.prompt_bundle_digest", "identity.content_bundle_digest"],
        "policy": ["identity.policy_bundle_digest"],
        "provider": [
            "identity.provider_routing.provider_digest",
            "identity.provider_routing.routing_digest",
        ],
        "model": [
            "identity.provider_routing.model_digest",
            "identity.provider_routing.routing_digest",
        ],
        "effort": [
            "identity.provider_routing.effort_digest",
            "identity.provider_routing.routing_digest",
        ],
        "workflow": [
            "identity.workflow_bundle_digest",
            "identity.enabled_gate_identity",
            "identity.content_bundle_digest",
            "identity.provider_routing.routing_digest",
        ],
    }
    field_axis: dict[str, set[str]] = {}
    for axis, paths in axis_fields.items():
        for path in paths:
            field_axis.setdefault(path, set()).add(axis)
    changes: list[dict[str, Any]] = []
    for path, axes in sorted(field_axis.items()):
        left, right = nested(baseline, path), nested(variant, path)
        if left == right:
            continue
        authorized = bool(set(allowed) & axes)
        changes.append(
            {
                "path": path,
                "baseline": left,
                "variant": right,
                "authorized": authorized,
                "axes": sorted(axes),
            }
        )
        if not authorized:
            reasons.append(
                {
                    "path": path,
                    "reason": "undeclared_identity_change",
                    "baseline": left,
                    "variant": right,
                }
            )
    return reasons, allowed


def oracle_result(value: Any) -> str:
    if isinstance(value, dict):
        return str(value.get("observed_result") or value.get("result") or "unknown")
    return str(value or "unknown")


def scenario_quality(
    aggregate: dict[str, Any],
) -> dict[tuple[str, int], dict[str, Any]]:
    result: dict[tuple[str, int], dict[str, Any]] = {}
    rows = nested(aggregate, "quality.by_scenario", [])
    if not isinstance(rows, list):
        return result
    per_id_counter: dict[str, int] = {}
    for row in rows:
        if not isinstance(row, dict):
            continue
        scenario_id = str(row.get("scenario_id"))
        rep = row.get("rep")
        if not isinstance(rep, int):
            per_id_counter[scenario_id] = per_id_counter.get(scenario_id, 0) + 1
            rep = per_id_counter[scenario_id]
        result[(scenario_id, rep)] = row
    return result


def aggregate_binding_reasons(
    aggregate: dict[str, Any], manifest: dict[str, Any], root: Path, side: str
) -> list[dict[str, Any]]:
    """Bind a stored aggregate to the exact retained inputs and run manifest."""
    reasons: list[dict[str, Any]] = []

    def mismatch(path: str, reason: str, expected: Any, observed: Any) -> None:
        if expected != observed:
            reasons.append(
                {
                    "path": f"{side}/{path}",
                    "reason": reason,
                    "expected": expected,
                    "observed": observed,
                }
            )

    aggregate_path = root / "aggregate.json"
    if aggregate_path.is_symlink():
        reasons.append(
            {
                "path": f"{side}/aggregate.json",
                "reason": "aggregate_path_is_symlink",
            }
        )
    regenerated = run_process(
        [str(SCRIPTS_DIR / "aggregate-lifecycle.sh"), "--retention-root", str(root)],
        cwd=REPO_ROOT,
        env=aggregation_environment(),
    )
    if regenerated.returncode != 0:
        reasons.append(
            {
                "path": f"{side}/aggregate.json",
                "reason": "retention_reaggregation_failed",
                "detail": regenerated.stderr.strip() or regenerated.stdout.strip(),
            }
        )
    else:
        try:
            regenerated_value = json.loads(regenerated.stdout)
        except json.JSONDecodeError as exc:
            reasons.append(
                {
                    "path": f"{side}/aggregate.json",
                    "reason": "retention_reaggregation_malformed",
                    "detail": str(exc),
                }
            )
        else:
            mismatch(
                "aggregate.json",
                "aggregate_detached_from_retention_root",
                canonical_digest(regenerated_value),
                canonical_digest(aggregate),
            )

    identity = aggregate.get("identity")
    if not isinstance(identity, dict):
        return reasons
    mismatch(
        "aggregate/identity/phase",
        "aggregate_manifest_phase_mismatch",
        manifest.get("phase"),
        identity.get("phase"),
    )

    batch_path = root / "batch.json"
    try:
        batch, _batch_bytes = load_json_no_follow(
            batch_path,
            f"{side} retained batch identity",
            invalid_exit_code=5,
        )
    except OperatorError as exc:
        reasons.append(
            {
                "path": f"{side}/batch.json",
                "reason": "invalid_batch_identity",
                "detail": str(exc),
            }
        )
        return reasons
    for field in ("batch_id", "batch_policy_digest", "acknowledgement_ref", "min_reps"):
        mismatch(
            f"aggregate/identity/{field}",
            "aggregate_batch_identity_mismatch",
            batch.get(field),
            identity.get(field),
        )
    normalized_batch_ceilings = batch_authority_projection(batch)["ceilings"]
    mismatch(
        "aggregate/identity/ceilings",
        "aggregate_batch_identity_mismatch",
        normalized_batch_ceilings,
        identity.get("ceilings"),
    )
    mismatch(
        "batch/phase",
        "batch_manifest_phase_mismatch",
        manifest.get("phase"),
        batch.get("phase"),
    )
    mismatch(
        "batch/mode",
        "batch_manifest_mode_mismatch",
        manifest.get("run_mode"),
        batch.get("mode"),
    )
    retained_root = batch.get("retention_root")
    try:
        if not isinstance(retained_root, str) or not Path(retained_root).is_absolute():
            raise OperatorError("recorded retention root must be an absolute path")
        observed_root = str(
            require_existing_real_directory(
                Path(retained_root), f"{side} recorded retention root"
            )
        )
    except OperatorError as exc:
        observed_root = None
        reasons.append(
            {
                "path": f"{side}/batch/retention_root",
                "reason": "batch_retention_root_not_real",
                "detail": str(exc),
            }
        )
    mismatch(
        "batch/retention_root",
        "batch_retention_root_mismatch",
        str(lexical_absolute_path(root)),
        observed_root,
    )

    batch_policy = root / "batch-policy.yaml"
    if batch_policy.is_symlink() or not batch_policy.is_file():
        reasons.append(
            {
                "path": f"{side}/batch-policy.yaml",
                "reason": "missing_or_symlinked_batch_policy",
            }
        )
    else:
        mismatch(
            "aggregate/identity/batch_policy_digest",
            "batch_policy_digest_mismatch",
            path_digest(batch_policy, f"{side} retained batch policy"),
            identity.get("batch_policy_digest"),
        )

    resources = manifest.get("resource_policy") or {}
    expected_ceilings = {key: resources.get(key) for key in RESOURCE_CEILING_NAMES}
    observed_ceilings = dict(identity.get("ceilings") or {})
    # T-E40-F11-003 (AC-F11-06a): a legacy batch that never declared
    # max_provider_calls has no identity claim to defend on that axis --
    # drop it from both sides rather than force agreement with whatever
    # the run manifest's resource_policy.max_provider_calls happens to
    # record (which can differ from a batch that predates the field).
    if observed_ceilings.get("max_provider_calls") is None:
        expected_ceilings.pop("max_provider_calls", None)
        observed_ceilings.pop("max_provider_calls", None)
    mismatch(
        "aggregate/identity/ceilings",
        "aggregate_manifest_ceiling_mismatch",
        expected_ceilings,
        observed_ceilings,
    )
    mismatch(
        "aggregate/identity/min_reps",
        "aggregate_manifest_repetitions_mismatch",
        resources.get("repetitions"),
        identity.get("min_reps"),
    )

    batch_id = batch.get("batch_id")
    if not isinstance(batch_id, str) or not SAFE_ID.fullmatch(batch_id):
        reasons.append(
            {
                "path": f"{side}/batch-authorities",
                "reason": "invalid_batch_authority_id",
                "observed": batch_id,
            }
        )
        return reasons
    authorities = root / "batch-authorities"
    authority_path = authorities / f"{batch_id}.json"
    try:
        authority_record, _authority_bytes = load_json_no_follow(
            authority_path,
            f"{side} batch authority",
            invalid_exit_code=5,
        )
    except OperatorError as exc:
        reasons.append(
            {
                "path": f"{side}/batch-authorities/{batch_id}.json",
                "reason": "invalid_batch_authority",
                "detail": str(exc),
            }
        )
        return reasons
    recorded_authority_digest = authority_record.get("authority_digest")
    recomputed_authority_digest = canonical_digest(
        {
            key: value
            for key, value in authority_record.items()
            if key != "authority_digest"
        }
    )
    mismatch(
        f"batch-authorities/{batch_id}.json/authority_digest",
        "batch_authority_digest_mismatch",
        recomputed_authority_digest,
        recorded_authority_digest,
    )
    mismatch(
        f"batch-authorities/{batch_id}.json/manifest_digest",
        "batch_authority_manifest_mismatch",
        manifest.get("manifest_digest"),
        authority_record.get("manifest_digest"),
    )
    mismatch(
        f"batch-authorities/{batch_id}.json/batch_authority",
        "batch_authority_mismatch",
        batch_authority_projection(batch),
        authority_record.get("batch_authority"),
    )
    attempt_relative = authority_record.get("attempt_path")
    if not isinstance(attempt_relative, str):
        reasons.append(
            {
                "path": f"{side}/batch-authorities/{batch_id}.json/attempt_path",
                "reason": "missing_batch_authority_attempt",
            }
        )
        return reasons
    relative_attempt = Path(attempt_relative)
    if (
        relative_attempt.is_absolute()
        or relative_attempt.as_posix() != attempt_relative
        or any(part in {"", ".", ".."} for part in attempt_relative.split("/"))
    ):
        reasons.append(
            {
                "path": f"{side}/batch-authorities/{batch_id}.json/attempt_path",
                "reason": "invalid_batch_authority_attempt_path",
                "observed": attempt_relative,
            }
        )
        return reasons
    attempt_path = lexical_absolute_path(root / relative_attempt)
    try:
        attempt, attempt_bytes = load_json_no_follow(
            attempt_path,
            f"{side} batch authority attempt",
            invalid_exit_code=5,
        )
        attempt_digest = hashlib.sha256(attempt_bytes).hexdigest()
    except OperatorError as exc:
        reasons.append(
            {
                "path": f"{side}/batch-authorities/{batch_id}.json/attempt_path",
                "reason": "invalid_batch_authority_attempt",
                "detail": str(exc),
            }
        )
        return reasons
    mismatch(
        f"batch-authorities/{batch_id}.json/attempt_digest",
        "batch_authority_attempt_digest_mismatch",
        attempt_digest,
        authority_record.get("attempt_digest"),
    )
    command = attempt.get("command") or []
    if not isinstance(command, list) or "--acknowledge-provider-spend" not in command:
        reasons.append(
            {
                "path": f"{side}/batch-authorities/{batch_id}.json/attempt_path",
                "reason": "batch_authority_attempt_missing_acknowledgement",
            }
        )
    return reasons


def aggregate_validation(
    aggregate: dict[str, Any], manifest: dict[str, Any], side: str
) -> list[dict[str, Any]]:
    reasons: list[dict[str, Any]] = []
    for field in (
        "identity",
        "scenarios",
        "time",
        "cost",
        "quality",
        "review_value",
        "artifact_use",
        "noise_bands",
        "invalid",
    ):
        if field not in aggregate:
            reasons.append(
                {
                    "path": f"{side}/aggregate/{field}",
                    "reason": "missing_aggregate_block",
                }
            )
    invalid = aggregate.get("invalid")
    if not isinstance(invalid, list):
        reasons.append(
            {"path": f"{side}/aggregate/invalid", "reason": "malformed_invalid_block"}
        )
    elif invalid:
        for row in invalid:
            reasons.append(
                {
                    "path": f"{side}/aggregate/invalid",
                    "reason": "retained_invalid_run",
                    "detail": row,
                }
            )
    scenarios = aggregate.get("scenarios")
    observed_pairs: set[tuple[str, int]] = set()
    if isinstance(scenarios, list) and scenarios:
        for row in scenarios:
            if not isinstance(row, dict):
                reasons.append(
                    {
                        "path": f"{side}/aggregate/scenarios",
                        "reason": "malformed_scenario",
                    }
                )
                continue
            scenario_id, rep = row.get("scenario_id"), row.get("rep")
            if isinstance(scenario_id, str) and isinstance(rep, int):
                observed_pairs.add((scenario_id, rep))
            else:
                reasons.append(
                    {
                        "path": f"{side}/aggregate/scenarios",
                        "reason": "malformed_scenario_identity",
                    }
                )
            eligibility = row.get("eligibility") or {}
            if (
                eligibility.get("aggregate_eligible") is not True
                or eligibility.get("publication_eligible") is not True
            ):
                reasons.append(
                    {
                        "path": f"{side}/aggregate/scenarios/{row.get('scenario_id')}/{row.get('rep')}",
                        "reason": "scenario_ineligible",
                        "detail": eligibility.get("invalidity_reasons") or [],
                    }
                )
    else:
        reasons.append(
            {"path": f"{side}/aggregate/scenarios", "reason": "empty_scenario_block"}
        )
    expected_pairs: set[tuple[str, int]] = set()
    manifest_matrix = manifest.get("scenario_matrix")
    if not isinstance(manifest_matrix, list) or not manifest_matrix:
        reasons.append(
            {
                "path": f"{side}/manifest/scenario_matrix",
                "reason": "missing_scenario_matrix",
            }
        )
    else:
        for row in manifest_matrix:
            if not isinstance(row, dict) or not isinstance(row.get("scenario_id"), str):
                reasons.append(
                    {
                        "path": f"{side}/manifest/scenario_matrix",
                        "reason": "malformed_scenario_matrix",
                    }
                )
                continue
            try:
                reps = int(row.get("reps"))
            except (TypeError, ValueError):
                reps = 0
            if reps <= 0:
                reasons.append(
                    {
                        "path": f"{side}/manifest/scenario_matrix",
                        "reason": "malformed_scenario_matrix",
                    }
                )
                continue
            expected_pairs.update(
                (row["scenario_id"], rep) for rep in range(1, reps + 1)
            )
    if observed_pairs != expected_pairs:
        reasons.append(
            {
                "path": f"{side}/aggregate/scenarios",
                "reason": "manifest_aggregate_matrix_mismatch",
                "expected": sorted(expected_pairs),
                "observed": sorted(observed_pairs),
            }
        )
    quality = scenario_quality(aggregate)
    if set(quality) != observed_pairs:
        reasons.append(
            {
                "path": f"{side}/aggregate/quality/by_scenario",
                "reason": "quality_scenario_matrix_mismatch",
            }
        )
    for pair, row in quality.items():
        result = oracle_result(row.get("execution_oracle"))
        if result != "pass":
            reasons.append(
                {
                    "path": f"{side}/aggregate/quality/{pair[0]}/{pair[1]}/execution_oracle",
                    "reason": "failed_or_missing_execution_oracle",
                    "observed": result,
                }
            )
    bands = aggregate.get("noise_bands")
    if not isinstance(bands, list) or not bands:
        reasons.append(
            {"path": f"{side}/aggregate/noise_bands", "reason": "empty_noise_bands"}
        )
    wall = nested(aggregate, "time.lifecycle_wall_seconds")
    if isinstance(wall, (int, float)) and not isinstance(wall, bool):
        for partition in ("stage_category", "interval_category", "share_partition"):
            block = nested(aggregate, f"time.{partition}")
            if not isinstance(block, dict) or any(
                not isinstance(value, (int, float)) or isinstance(value, bool)
                for value in block.values()
            ):
                reasons.append(
                    {
                        "path": f"{side}/aggregate/time/{partition}",
                        "reason": "malformed_time_partition",
                    }
                )
                continue
            if not math.isclose(sum(block.values()), wall, rel_tol=0.0, abs_tol=1e-5):
                reasons.append(
                    {
                        "path": f"{side}/aggregate/time/{partition}",
                        "reason": "time_reconciliation_failed",
                        "expected": wall,
                        "observed": round(sum(block.values()), 6),
                    }
                )
    else:
        reasons.append(
            {
                "path": f"{side}/aggregate/time/lifecycle_wall_seconds",
                "reason": "missing_wall_time",
            }
        )
    return reasons


def numeric_delta(left: Any, right: Any) -> float | str:
    if (
        isinstance(left, (int, float))
        and not isinstance(left, bool)
        and isinstance(right, (int, float))
        and not isinstance(right, bool)
    ):
        return round(float(right) - float(left), 6)
    return UNAVAILABLE


def band_map(aggregate: dict[str, Any]) -> dict[tuple[str, str], dict[str, Any]]:
    result: dict[tuple[str, str], dict[str, Any]] = {}
    for band in aggregate.get("noise_bands") or []:
        if isinstance(band, dict):
            result[(str(band.get("scenario_id")), str(band.get("metric")))] = band
    return result


def classify_band(
    baseline_band: dict[str, Any] | None,
    variant_band: dict[str, Any] | None,
    *,
    lower_is_better: bool,
) -> dict[str, Any]:
    if not isinstance(baseline_band, dict) or not isinstance(variant_band, dict):
        return {
            "classification": "indeterminate",
            "reason": "noise_band_missing",
            "delta": UNAVAILABLE,
        }
    if baseline_band.get("insufficient_reps") or variant_band.get("insufficient_reps"):
        return {
            "classification": "indeterminate",
            "reason": "insufficient_reps",
            "delta": UNAVAILABLE,
        }
    left, right = baseline_band.get("median"), variant_band.get("median")
    delta = numeric_delta(left, right)
    if delta == UNAVAILABLE:
        return {
            "classification": "indeterminate",
            "reason": "metric_unavailable",
            "delta": UNAVAILABLE,
        }
    if delta == 0:
        classification = "unchanged"
    else:
        interval = baseline_band.get("acceptance_interval") or {}
        lower, upper = interval.get("lower_bound"), interval.get("upper_bound")
        if (
            isinstance(lower, (int, float))
            and isinstance(upper, (int, float))
            and lower <= right <= upper
        ):
            classification = "no_detectable_effect"
        else:
            improvement = right < left if lower_is_better else right > left
            classification = "better" if improvement else "worse"
    return {
        "classification": classification,
        "delta": delta,
        "baseline_median": left,
        "variant_median": right,
        "baseline_acceptance_interval": baseline_band.get("acceptance_interval"),
        "derivation_rule": baseline_band.get("derivation_rule"),
    }


def pass_rate(aggregate: dict[str, Any]) -> tuple[int, int, float | str]:
    quality = scenario_quality(aggregate)
    if not quality:
        return 0, 0, UNAVAILABLE
    passed = sum(
        1
        for row in quality.values()
        if oracle_result(row.get("execution_oracle")) == "pass"
    )
    total = len(quality)
    return passed, total, round(passed / total, 6)


def partition_delta(left: Any, right: Any) -> dict[str, Any]:
    if not isinstance(left, dict) or not isinstance(right, dict):
        return {"status": UNAVAILABLE, "reason": "upstream_metric_missing"}
    keys = sorted(set(left) | set(right))
    return {key: numeric_delta(left.get(key), right.get(key)) for key in keys}


def review_totals(aggregate: dict[str, Any]) -> tuple[dict[str, float], bool]:
    totals: dict[str, float] = {}
    truth_available = True
    gates = nested(aggregate, "review_value.gates", [])
    if not isinstance(gates, list):
        return totals, False
    gate_seen = False
    for gate in gates:
        if not isinstance(gate, dict):
            continue
        gate_seen = True
        truth_available = truth_available and gate.get("truth_set_available") is True
        counts = gate.get("counts") or {}
        if isinstance(counts, dict):
            for key, value in counts.items():
                if isinstance(value, (int, float)) and not isinstance(value, bool):
                    totals[key] = totals.get(key, 0.0) + float(value)
    return totals, truth_available and gate_seen


def review_gate_comparison(
    baseline: dict[str, Any], variant: dict[str, Any]
) -> list[dict[str, Any]]:
    def gate_map(aggregate: dict[str, Any]) -> dict[str, dict[str, Any]]:
        return {
            str(row.get("gate_id")): row
            for row in nested(aggregate, "review_value.gates", [])
            if isinstance(row, dict) and row.get("gate_id")
        }

    left, right = gate_map(baseline), gate_map(variant)
    rows: list[dict[str, Any]] = []
    for gate_id in sorted(set(left) | set(right)):
        left_gate, right_gate = left.get(gate_id, {}), right.get(gate_id, {})
        left_counts, right_counts = (
            left_gate.get("counts") or {},
            right_gate.get("counts") or {},
        )
        rows.append(
            {
                "gate_id": gate_id,
                "baseline_state": left_gate.get("state", UNAVAILABLE),
                "variant_state": right_gate.get("state", UNAVAILABLE),
                "finding_counts": {
                    key: {
                        "baseline": left_counts.get(key, UNAVAILABLE),
                        "variant": right_counts.get(key, UNAVAILABLE),
                        "delta": numeric_delta(
                            left_counts.get(key), right_counts.get(key)
                        ),
                    }
                    for key in (
                        "emitted",
                        "normalized_unique",
                        "duplicate",
                        "recurrent",
                        "confirmed",
                        "unconfirmed",
                        "downstream_escape",
                    )
                },
                "elapsed_seconds": {
                    "baseline": left_gate.get("elapsed_seconds", UNAVAILABLE),
                    "variant": right_gate.get("elapsed_seconds", UNAVAILABLE),
                    "delta": numeric_delta(
                        left_gate.get("elapsed_seconds"),
                        right_gate.get("elapsed_seconds"),
                    ),
                },
                "provider_cost_usd": {
                    "baseline": left_gate.get("provider_cost_usd", UNAVAILABLE),
                    "variant": right_gate.get("provider_cost_usd", UNAVAILABLE),
                    "delta": numeric_delta(
                        left_gate.get("provider_cost_usd"),
                        right_gate.get("provider_cost_usd"),
                    ),
                },
                "resolution_cost_usd": {
                    "baseline": left_gate.get("resolution_cost_usd", UNAVAILABLE),
                    "variant": right_gate.get("resolution_cost_usd", UNAVAILABLE),
                    "delta": numeric_delta(
                        left_gate.get("resolution_cost_usd"),
                        right_gate.get("resolution_cost_usd"),
                    ),
                },
            }
        )
    return rows


def comparison_document(
    baseline_manifest: dict[str, Any],
    variant_manifest: dict[str, Any],
    baseline_aggregate: dict[str, Any],
    variant_aggregate: dict[str, Any],
    baseline_root: Path | None = None,
    variant_root: Path | None = None,
    binding_reasons: list[dict[str, Any]] | None = None,
) -> dict[str, Any]:
    reasons, allowed = comparison_boundary(baseline_manifest, variant_manifest)
    reasons.extend(binding_reasons or [])
    for side, manifest, root in (
        ("baseline", baseline_manifest, baseline_root),
        ("variant", variant_manifest, variant_root),
    ):
        if root is not None:
            retained = manifest.get("retention_root")
            try:
                if not isinstance(retained, str) or not Path(retained).is_absolute():
                    raise OperatorError(
                        "recorded retention root must be an absolute path"
                    )
                retained_path = require_existing_real_directory(
                    Path(retained), f"{side} manifest retention root"
                )
            except OperatorError as exc:
                reasons.append(
                    {
                        "path": f"{side}/retention_root",
                        "reason": "retention_root_not_real",
                        "detail": str(exc),
                    }
                )
                retained_path = None
            if retained_path != lexical_absolute_path(root):
                reasons.append(
                    {
                        "path": f"{side}/retention_root",
                        "reason": "retention_root_identity_mismatch",
                        "expected": str(lexical_absolute_path(root)),
                        "observed": retained,
                    }
                )
    reasons.extend(
        aggregate_validation(baseline_aggregate, baseline_manifest, "baseline")
    )
    reasons.extend(aggregate_validation(variant_aggregate, variant_manifest, "variant"))
    baseline_scenarios = {
        (str(row.get("scenario_id")), int(row.get("rep"))): row
        for row in baseline_aggregate.get("scenarios") or []
        if isinstance(row, dict) and isinstance(row.get("rep"), int)
    }
    variant_scenarios = {
        (str(row.get("scenario_id")), int(row.get("rep"))): row
        for row in variant_aggregate.get("scenarios") or []
        if isinstance(row, dict) and isinstance(row.get("rep"), int)
    }
    if set(baseline_scenarios) != set(variant_scenarios):
        reasons.append(
            {
                "path": "aggregate/scenario_matrix",
                "reason": "paired_scenario_matrix_mismatch",
                "baseline": sorted(baseline_scenarios),
                "variant": sorted(variant_scenarios),
            }
        )
    baseline_bands, variant_bands = (
        band_map(baseline_aggregate),
        band_map(variant_aggregate),
    )
    scenario_ids = sorted(
        {
            scenario_id
            for scenario_id, _rep in set(baseline_scenarios) | set(variant_scenarios)
        }
    )
    per_scenario: list[dict[str, Any]] = []
    for scenario_id in scenario_ids:
        metrics = {
            "elapsed_time": classify_band(
                baseline_bands.get((scenario_id, "elapsed_time")),
                variant_bands.get((scenario_id, "elapsed_time")),
                lower_is_better=True,
            ),
            "provider_cost": classify_band(
                baseline_bands.get((scenario_id, "provider_cost")),
                variant_bands.get((scenario_id, "provider_cost")),
                lower_is_better=True,
            ),
            "confirmed_findings": classify_band(
                baseline_bands.get((scenario_id, "confirmed_findings")),
                variant_bands.get((scenario_id, "confirmed_findings")),
                lower_is_better=False,
            ),
        }
        baseline_quality = scenario_quality(baseline_aggregate)
        variant_quality = scenario_quality(variant_aggregate)
        reps = sorted(
            rep
            for sid, rep in set(baseline_scenarios) | set(variant_scenarios)
            if sid == scenario_id
        )
        oracle_pairs = []
        for rep in reps:
            left = oracle_result(
                (baseline_quality.get((scenario_id, rep)) or {}).get("execution_oracle")
            )
            right = oracle_result(
                (variant_quality.get((scenario_id, rep)) or {}).get("execution_oracle")
            )
            oracle_pairs.append(
                {
                    "rep": rep,
                    "baseline": left,
                    "variant": right,
                    "classification": "unchanged"
                    if left == right
                    else ("better" if right == "pass" else "worse"),
                }
            )
        per_scenario.append(
            {
                "scenario_id": scenario_id,
                "oracle_pairs": oracle_pairs,
                "metrics": metrics,
            }
        )
    baseline_passed, baseline_total, baseline_rate = pass_rate(baseline_aggregate)
    variant_passed, variant_total, variant_rate = pass_rate(variant_aggregate)
    quality_delta = numeric_delta(baseline_rate, variant_rate)
    quality_regression = quality_delta != UNAVAILABLE and quality_delta < 0
    baseline_review, baseline_truth = review_totals(baseline_aggregate)
    variant_review, variant_truth = review_totals(variant_aggregate)
    insufficient = any(
        metric.get("classification") == "indeterminate"
        and metric.get("reason") == "insufficient_reps"
        for scenario in per_scenario
        for metric in scenario["metrics"].values()
    )
    if insufficient:
        reasons.append({"path": "aggregate/noise_bands", "reason": "insufficient_reps"})
    dimensions = {
        "correctness": {
            "baseline": {
                "passed": baseline_passed,
                "total": baseline_total,
                "pass_rate": baseline_rate,
            },
            "variant": {
                "passed": variant_passed,
                "total": variant_total,
                "pass_rate": variant_rate,
            },
            "delta": quality_delta,
            "classification": "worse"
            if quality_regression
            else (
                "unchanged"
                if quality_delta == 0
                else ("better" if quality_delta != UNAVAILABLE else "indeterminate")
            ),
        },
        "invalid_and_incomplete": {
            "baseline_count": len(baseline_aggregate.get("invalid") or []),
            "variant_count": len(variant_aggregate.get("invalid") or []),
        },
        "lifecycle_time": {
            "baseline_seconds": nested(
                baseline_aggregate, "time.lifecycle_wall_seconds", UNAVAILABLE
            ),
            "variant_seconds": nested(
                variant_aggregate, "time.lifecycle_wall_seconds", UNAVAILABLE
            ),
            "delta_seconds": numeric_delta(
                nested(baseline_aggregate, "time.lifecycle_wall_seconds"),
                nested(variant_aggregate, "time.lifecycle_wall_seconds"),
            ),
            "classification": "indeterminate",
            "reason": "aggregate_noise_band_not_published; use per-scenario bands",
            "stage_category_deltas": partition_delta(
                nested(baseline_aggregate, "time.stage_category"),
                nested(variant_aggregate, "time.stage_category"),
            ),
            "interval_category_deltas": partition_delta(
                nested(baseline_aggregate, "time.interval_category"),
                nested(variant_aggregate, "time.interval_category"),
            ),
            "share_partition_deltas": partition_delta(
                nested(baseline_aggregate, "time.share_partition"),
                nested(variant_aggregate, "time.share_partition"),
            ),
        },
        "provider_cost": {
            "baseline_usd": nested(
                baseline_aggregate,
                "cost.ceiling_consumption.observed_cost_usd",
                UNAVAILABLE,
            ),
            "variant_usd": nested(
                variant_aggregate,
                "cost.ceiling_consumption.observed_cost_usd",
                UNAVAILABLE,
            ),
            "delta_usd": numeric_delta(
                nested(
                    baseline_aggregate, "cost.ceiling_consumption.observed_cost_usd"
                ),
                nested(variant_aggregate, "cost.ceiling_consumption.observed_cost_usd"),
            ),
            "classification": "indeterminate",
            "reason": "aggregate_noise_band_not_published; use per-scenario bands",
            "stage_category_deltas": partition_delta(
                nested(baseline_aggregate, "cost.stage_category"),
                nested(variant_aggregate, "cost.stage_category"),
            ),
            "interval_category_deltas": partition_delta(
                nested(baseline_aggregate, "cost.interval_category"),
                nested(variant_aggregate, "cost.interval_category"),
            ),
            "share_partition_deltas": partition_delta(
                nested(baseline_aggregate, "cost.share_partition"),
                nested(variant_aggregate, "cost.share_partition"),
            ),
        },
        "tokens": {
            "input": {
                "status": UNAVAILABLE,
                "reason": "I-08/F10 aggregate exposes no input-token rollup",
            },
            "output": {
                "status": UNAVAILABLE,
                "reason": "I-08/F10 aggregate exposes no output-token rollup",
            },
        },
        "generated_loc": {
            "status": UNAVAILABLE,
            "reason": "lifecycle-v2 aggregate exposes candidate digests, not LOC payloads",
        },
        "changed_paths": {
            "status": UNAVAILABLE,
            "reason": "lifecycle-v2 aggregate exposes changed-path digests, not path payloads",
        },
        "review_findings": {
            "baseline": baseline_review,
            "variant": variant_review,
            "deltas": partition_delta(baseline_review, variant_review),
            "truth_set_available": baseline_truth and variant_truth,
            "precision": UNAVAILABLE
            if not (baseline_truth and variant_truth)
            else "reported_per_gate",
            "recall": UNAVAILABLE
            if not (baseline_truth and variant_truth)
            else "reported_per_gate",
            "gates": review_gate_comparison(baseline_aggregate, variant_aggregate),
        },
        "artifact_use": {
            key: {
                "baseline": nested(
                    baseline_aggregate, f"artifact_use.{key}", UNAVAILABLE
                ),
                "variant": nested(
                    variant_aggregate, f"artifact_use.{key}", UNAVAILABLE
                ),
                "delta": numeric_delta(
                    nested(baseline_aggregate, f"artifact_use.{key}"),
                    nested(variant_aggregate, f"artifact_use.{key}"),
                ),
            }
            for key in (
                "produced_count",
                "consumed_count",
                "reused_count",
                "orphan_count",
            )
        },
        "replayed_interaction_proxy": {
            "label": "replayed proxy; not observed human minutes",
            "deltas": partition_delta(
                nested(baseline_aggregate, "artifact_use.replayed_interaction_proxy"),
                nested(variant_aggregate, "artifact_use.replayed_interaction_proxy"),
            ),
        },
    }
    boundary_valid = not any(
        reason["reason"]
        in {
            "comparison_boundary_mismatch",
            "undeclared_identity_change",
            "unsupported_change_axis",
            "paired_scenario_matrix_mismatch",
            "missing_aggregate_block",
            "missing_or_malformed_identity",
            "manifest_digest_mismatch",
            "wrong_benchmark_phase",
            "wrong_comparison_role",
            "pilot_cannot_enter_comparison",
            "variant_not_linked_to_baseline_definition",
            "retention_root_identity_mismatch",
            "malformed_scenario",
            "malformed_scenario_identity",
            "empty_scenario_block",
            "manifest_aggregate_matrix_mismatch",
            "missing_scenario_matrix",
            "malformed_scenario_matrix",
            "quality_scenario_matrix_mismatch",
            "empty_noise_bands",
            "malformed_invalid_block",
            "malformed_time_partition",
            "time_reconciliation_failed",
            "missing_wall_time",
        }
        for reason in reasons
    )
    publication_eligible = boundary_valid and not reasons
    return {
        "schema_version": "1.0",
        "phase": "lifecycle_v2",
        "created_at": utc_now(),
        "baseline_manifest_digest": baseline_manifest.get("manifest_digest")
        or canonical_digest(baseline_manifest),
        "variant_manifest_digest": variant_manifest.get("manifest_digest")
        or canonical_digest(variant_manifest),
        "baseline_aggregate_digest": canonical_digest(baseline_aggregate),
        "variant_aggregate_digest": canonical_digest(variant_aggregate),
        "retained_raw_evidence": {
            "baseline": baseline_manifest.get("retention_root"),
            "variant": variant_manifest.get("retention_root"),
        },
        "boundary": {
            "valid": boundary_valid,
            "authorized_change_axes": allowed,
            "reasons": reasons,
        },
        "publication_eligible": publication_eligible,
        "quality_dominates_interpretation": True,
        "quality_regression": quality_regression,
        "interpretation": "variant_not_better_due_to_quality_regression"
        if quality_regression
        else ("invalid" if not publication_eligible else "separate_dimensions_only"),
        "dimensions": dimensions,
        "scenarios": per_scenario,
        "exclusions": reasons,
    }


def render_comparison(document: dict[str, Any]) -> str:
    lines = [
        "# E40 baseline versus variant comparison",
        "",
        f"- Publication eligible: `{str(document['publication_eligible']).lower()}`",
        f"- Comparison boundary valid: `{str(document['boundary']['valid']).lower()}`",
        f"- Authorized change axes: `{', '.join(document['boundary']['authorized_change_axes']) or 'none (control rerun)'}`",
        "- Quality dominates interpretation: `true`",
        f"- Interpretation: `{document['interpretation']}`",
        f"- Baseline retained evidence: `{document['retained_raw_evidence']['baseline']}`",
        f"- Variant retained evidence: `{document['retained_raw_evidence']['variant']}`",
        "",
        "## Quality and validity",
        "",
    ]
    correctness = document["dimensions"]["correctness"]
    lines.extend(
        [
            "| Dimension | Baseline | Variant | Delta | Result |",
            "|---|---:|---:|---:|---|",
            f"| Held-back oracle pass rate | {correctness['baseline']['pass_rate']} | {correctness['variant']['pass_rate']} | {correctness['delta']} | {correctness['classification']} |",
            f"| Invalid/incomplete runs | {document['dimensions']['invalid_and_incomplete']['baseline_count']} | {document['dimensions']['invalid_and_incomplete']['variant_count']} | - | {'invalid' if document['exclusions'] else 'unchanged'} |",
            "",
            "## Paired held-back oracle results",
            "",
            "| Scenario | Repetition | Baseline | Variant | Result |",
            "|---|---:|---|---|---|",
        ]
    )
    for scenario in document["scenarios"]:
        for pair in scenario["oracle_pairs"]:
            lines.append(
                f"| {scenario['scenario_id']} | {pair['rep']} | {pair['baseline']} | "
                f"{pair['variant']} | {pair['classification']} |"
            )
    lines.extend(
        [
            "",
            "## Per-scenario published noise-band results",
            "",
            "| Scenario | Metric | Baseline median | Variant median | Delta | Result |",
            "|---|---|---:|---:|---:|---|",
        ]
    )
    for scenario in document["scenarios"]:
        for metric_name, metric in scenario["metrics"].items():
            lines.append(
                f"| {scenario['scenario_id']} | {metric_name} | {metric.get('baseline_median', UNAVAILABLE)} | "
                f"{metric.get('variant_median', UNAVAILABLE)} | {metric.get('delta', UNAVAILABLE)} | "
                f"{metric['classification']} |"
            )
    lines.extend(["", "## Separate aggregate dimensions", ""])
    for label, key, unit in (
        ("Lifecycle wall time", "lifecycle_time", "seconds"),
        ("Provider cost", "provider_cost", "USD"),
    ):
        block = document["dimensions"][key]
        baseline_key = "baseline_seconds" if key == "lifecycle_time" else "baseline_usd"
        variant_key = "variant_seconds" if key == "lifecycle_time" else "variant_usd"
        delta_key = "delta_seconds" if key == "lifecycle_time" else "delta_usd"
        lines.append(
            f"- {label}: baseline {block[baseline_key]} {unit}; variant {block[variant_key]} {unit}; "
            f"delta {block[delta_key]} {unit}; aggregate classification `{block['classification']}` "
            f"({block['reason']})."
        )
    time_dimensions = document["dimensions"]["lifecycle_time"]
    lines.extend(
        [
            "",
            "## Lifecycle-time partition deltas",
            "",
            "| Partition | Category | Variant minus baseline (seconds) |",
            "|---|---|---:|",
        ]
    )
    for partition in ("stage_category", "interval_category", "share_partition"):
        values = time_dimensions[f"{partition}_deltas"]
        if isinstance(values, dict):
            for category, delta in values.items():
                lines.append(f"| {partition} | {category} | {delta} |")
    lines.extend(
        [
            f"- Input tokens: `{document['dimensions']['tokens']['input']['status']}` — {document['dimensions']['tokens']['input']['reason']}.",
            f"- Output tokens: `{document['dimensions']['tokens']['output']['status']}` — {document['dimensions']['tokens']['output']['reason']}.",
            f"- Generated LOC: `{document['dimensions']['generated_loc']['status']}` — {document['dimensions']['generated_loc']['reason']}.",
            f"- Changed paths: `{document['dimensions']['changed_paths']['status']}` — {document['dimensions']['changed_paths']['reason']}.",
            "- Replayed interaction values are proxies, not observed human minutes.",
            "",
            "## Artifact-use deltas",
            "",
            "| Measure | Baseline | Variant | Delta |",
            "|---|---:|---:|---:|",
        ]
    )
    for measure, values in document["dimensions"]["artifact_use"].items():
        lines.append(
            f"| {measure} | {values['baseline']} | {values['variant']} | {values['delta']} |"
        )
    proxy = document["dimensions"]["replayed_interaction_proxy"]
    lines.extend(
        [
            "",
            f"Replayed-interaction label: `{proxy['label']}`.",
            "",
            "| Replayed proxy measure | Variant minus baseline |",
            "|---|---:|",
        ]
    )
    for measure, delta in proxy["deltas"].items():
        if measure != "label":
            lines.append(f"| {measure} | {delta} |")
    lines.extend(
        [
            "",
            "## Review-gate findings and time",
            "",
            "| Gate | Baseline state | Variant state | Emitted | Unique | Duplicate | Recurrent | Confirmed | Unconfirmed | Escape | Time | Provider cost | Resolution cost |",
            "|---|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|",
        ]
    )
    for gate in document["dimensions"]["review_findings"]["gates"]:
        counts = gate["finding_counts"]
        lines.append(
            f"| {gate['gate_id']} | {gate['baseline_state']} | {gate['variant_state']} | "
            f"{counts['emitted']['delta']} | {counts['normalized_unique']['delta']} | "
            f"{counts['duplicate']['delta']} | {counts['recurrent']['delta']} | "
            f"{counts['confirmed']['delta']} | {counts['unconfirmed']['delta']} | "
            f"{counts['downstream_escape']['delta']} | {gate['elapsed_seconds']['delta']} | "
            f"{gate['provider_cost_usd']['delta']} | {gate['resolution_cost_usd']['delta']} |"
        )
    lines.extend(
        [
            "",
            "## Invalid or excluded evidence",
            "",
        ]
    )
    if document["exclusions"]:
        for reason in document["exclusions"]:
            detail = {
                key: value
                for key, value in reason.items()
                if key not in {"reason", "path"}
            }
            suffix = f" — `{json.dumps(detail, sort_keys=True)}`" if detail else ""
            lines.append(
                f"- `{reason.get('reason')}` at `{reason.get('path')}`{suffix}"
            )
    else:
        lines.append("- None.")
    lines.append("")
    return "\n".join(lines)


def resolve_run_reference(reference: str, config: str | None) -> Path:
    candidate = Path(reference)
    if candidate.exists() or candidate.is_symlink():
        resolved_candidate = lexical_absolute_path(candidate)
        if not resolved_candidate.is_dir():
            raise OperatorError(
                f"run reference is not a directory: {resolved_candidate}"
            )
        ensure_real_directory(resolved_candidate, "run reference")
        return resolved_candidate
    if not config:
        raise OperatorError(
            f"run reference is not a path; --config is required to resolve id {reference!r}"
        )
    config_value, config_base = config_context(Path(config).resolve())
    operator_root = config_path_value(config_value, config_base, "operator_root")
    run_store = resolve_path(
        str(config_value.get("run_store") or operator_root / "runs"), config_base
    )
    resolved = run_store / require_safe_id(reference, "run reference")
    if not resolved.is_dir():
        raise OperatorError(f"run id did not resolve: {reference!r} -> {resolved}")
    ensure_real_directory(resolved, "configured run reference")
    return resolved


def cmd_compare(args: argparse.Namespace) -> int:
    baseline_root = resolve_run_reference(args.baseline, args.config)
    variant_root = resolve_run_reference(args.variant, args.config)
    baseline_manifest = load_retained_manifest(
        baseline_root / "benchmark-manifest.json",
        "baseline manifest",
        invalid_exit_code=5,
    )
    variant_manifest = load_retained_manifest(
        variant_root / "benchmark-manifest.json",
        "variant manifest",
        invalid_exit_code=5,
    )
    baseline_aggregate, _baseline_aggregate_bytes = load_json_no_follow(
        baseline_root / "aggregate.json",
        "baseline aggregate",
        invalid_exit_code=5,
    )
    variant_aggregate, _variant_aggregate_bytes = load_json_no_follow(
        variant_root / "aggregate.json",
        "variant aggregate",
        invalid_exit_code=5,
    )
    binding_reasons = [
        *aggregate_binding_reasons(
            baseline_aggregate, baseline_manifest, baseline_root, "baseline"
        ),
        *aggregate_binding_reasons(
            variant_aggregate, variant_manifest, variant_root, "variant"
        ),
    ]
    unsafe_authority_reasons = {
        "invalid_batch_identity",
        "invalid_batch_authority",
        "invalid_batch_authority_attempt",
        "invalid_batch_authority_attempt_path",
        "missing_or_symlinked_batch_policy",
    }
    if any(
        reason.get("reason") in unsafe_authority_reasons for reason in binding_reasons
    ):
        print(
            json.dumps(
                {
                    "status": "invalid",
                    "comparison": str(
                        lexical_absolute_path(args.out) / "comparison.json"
                    ),
                    "publication_eligible": False,
                    "reasons": binding_reasons,
                },
                sort_keys=True,
            )
        )
        return 5
    output_root = lexical_absolute_path(args.out)
    ensure_external_operator_root(output_root)
    ensure_real_directory(output_root, "comparison output root")
    result_path = output_root / "comparison.json"
    report_path = output_root / "comparison.md"
    document = comparison_document(
        baseline_manifest,
        variant_manifest,
        baseline_aggregate,
        variant_aggregate,
        baseline_root,
        variant_root,
        binding_reasons,
    )
    document["comparison_id"] = (
        "comparison-"
        + canonical_digest(
            {
                "baseline": document["baseline_manifest_digest"],
                "variant": document["variant_manifest_digest"],
                "baseline_aggregate": document["baseline_aggregate_digest"],
                "variant_aggregate": document["variant_aggregate_digest"],
            }
        )[:20]
    )
    completion_path = output_root / "comparison.complete.json"
    lock = acquire_lock(output_root)
    try:

        def invalid_cache(reason: str) -> int:
            print(
                json.dumps(
                    {
                        "status": "invalid",
                        "comparison": str(result_path),
                        "publication_eligible": False,
                        "reasons": [{"path": "comparison-cache", "reason": reason}],
                    },
                    sort_keys=True,
                )
            )
            return 5

        for path in (result_path, report_path, completion_path):
            if path.is_symlink():
                return invalid_cache("symlinked_comparison_cache")
            if path.exists() and not path.is_file():
                return invalid_cache("non_file_comparison_cache")

        result_exists = result_path.is_file()
        report_exists = report_path.is_file()
        completion_exists = completion_path.is_file()
        try:
            existing = (
                load_json_no_follow(
                    result_path,
                    "existing comparison",
                    invalid_exit_code=5,
                )[0]
                if result_exists
                else None
            )
        except OperatorError:
            return invalid_cache("malformed_comparison_cache")
        expected = copy.deepcopy(document)
        if existing is not None:
            expected["created_at"] = existing.get("created_at")
        expected_report = render_comparison(expected).encode("utf-8")
        expected_completion = {
            "schema_version": "1.0",
            "comparison_id": expected["comparison_id"],
            "comparison_digest": canonical_digest(expected),
            "report_sha256": hashlib.sha256(expected_report).hexdigest(),
        }
        result_matches = existing is None or canonical_digest(
            existing
        ) == canonical_digest(expected)
        try:
            retained_report = (
                load_bytes_no_follow(
                    report_path,
                    "existing comparison report",
                    invalid_exit_code=5,
                )
                if report_exists
                else None
            )
        except OperatorError:
            return invalid_cache("malformed_comparison_cache")
        report_matches = retained_report is None or retained_report == expected_report
        if not result_matches or not report_matches:
            return invalid_cache("cached_comparison_integrity_mismatch")

        if completion_exists:
            if not result_exists or not report_exists:
                return invalid_cache("incomplete_comparison_cache")
            try:
                completion, _completion_bytes = load_json_no_follow(
                    completion_path,
                    "comparison completion marker",
                    invalid_exit_code=5,
                )
            except OperatorError:
                return invalid_cache("malformed_comparison_completion")
            if completion != expected_completion:
                return invalid_cache("comparison_completion_mismatch")
            print(
                json.dumps(
                    {"status": "already_compared", "comparison": str(result_path)},
                    sort_keys=True,
                )
            )
            return 0 if expected.get("publication_eligible") else 5

        recovered = result_exists or report_exists
        if not result_exists:
            write_json(result_path, expected, overwrite=False)
        if not report_exists:
            atomic_write(report_path, expected_report, overwrite=False)
        write_json(completion_path, expected_completion, overwrite=False)
        print(
            json.dumps(
                {
                    "status": "recovered"
                    if recovered
                    else (
                        "published" if document["publication_eligible"] else "invalid"
                    ),
                    "comparison": str(result_path),
                    "report": str(report_path),
                    "completion": str(completion_path),
                    "publication_eligible": expected["publication_eligible"],
                    "quality_regression": expected["quality_regression"],
                    "reasons": expected["boundary"]["reasons"],
                },
                sort_keys=True,
            )
        )
        return 0 if expected["publication_eligible"] else 5
    finally:
        release_lock(lock)


def cmd_demo(args: argparse.Namespace) -> int:
    setup_args = argparse.Namespace(
        out=args.out, config_out=None, scenario_index=None
    )
    setup_status = cmd_setup(setup_args)
    if setup_status != 0:
        return setup_status
    config = str(lexical_absolute_path(args.out) / "e40-demo.yaml")
    preflight_args = argparse.Namespace(
        config=config, out=None, reps=args.reps, scenario=args.scenario
    )
    return cmd_preflight(preflight_args)


def add_execution_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--config", required=True)
    parser.add_argument("--run-id", required=True)
    parser.add_argument(
        "--out", help="explicit retention root; defaults to config run_store/run-id"
    )
    parser.add_argument("--reps", type=int)
    parser.add_argument("--scenario", help="focus one admitted scenario")
    parser.add_argument("--acknowledge-provider-spend", action="store_true")
    parser.add_argument("--max-cost-usd", required=True)
    parser.add_argument("--max-wall-clock-seconds", required=True)
    parser.add_argument("--max-generated-tasks", required=True)
    # AC-F11-06a seam S1: the fourth ceiling, required=True so pilot,
    # baseline, and variant all inherit it (unlike S2's prepare-replay,
    # which is parser-optional/handler-required -- T-E40-F11-006).
    parser.add_argument("--max-provider-calls", required=True)
    parser.add_argument("--retry-incomplete", action="store_true")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="e40-benchmark.sh",
        description="Prepare, preview, run, resume, and compare E40 lifecycle benchmarks safely.",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    setup = subparsers.add_parser(
        "setup", help="prepare isolated scratch roots and a demo config"
    )
    setup.add_argument(
        "--out",
        required=True,
        help="explicit operator root outside the live repository",
    )
    setup.add_argument("--config-out")
    setup.add_argument(
        "--scenario-index",
        help=(
            "scenario-index YAML to seed dedicated roots from; defaults to "
            "bench/scenarios/scenarios.yaml (REQ-F-012)"
        ),
    )
    setup.set_defaults(handler=cmd_setup)

    preflight = subparsers.add_parser(
        "preflight", aliases=["preview"], help="zero-provider validation"
    )
    preflight.add_argument("--config", required=True)
    preflight.add_argument("--out")
    preflight.add_argument("--reps", type=int)
    preflight.add_argument("--scenario")
    preflight.set_defaults(handler=cmd_preflight)

    validate = subparsers.add_parser(
        "validate-variant", help="copy and digest a controlled variant"
    )
    validate.add_argument("--config", required=True)
    validate.add_argument("--variant-name", required=True)
    validate.add_argument("--prompt-root")
    validate.add_argument("--workflow-root")
    validate.add_argument("--policy-root")
    validate.add_argument("--allow-identical", action="store_true")
    validate.set_defaults(handler=cmd_validate_variant)

    pilot = subparsers.add_parser(
        "pilot", help="run a bounded pilot through the existing F10 driver"
    )
    add_execution_arguments(pilot)
    pilot.add_argument(
        "--variant-name",
        help="resolve operator_root/variants/<name>/variant-config.yaml from the baseline config",
    )
    pilot.set_defaults(handler=cmd_pilot)

    baseline = subparsers.add_parser("baseline", help="run/resume an attested baseline")
    add_execution_arguments(baseline)
    baseline.set_defaults(handler=cmd_baseline)

    variant = subparsers.add_parser("variant", help="run/resume an attested variant")
    add_execution_arguments(variant)
    variant.add_argument(
        "--variant-name",
        help="resolve operator_root/variants/<name>/variant-config.yaml from the baseline config",
    )
    variant.set_defaults(handler=cmd_variant)

    compare = subparsers.add_parser(
        "compare", help="validate a paired boundary and render deltas"
    )
    compare.add_argument("--baseline", required=True, help="retention root or run id")
    compare.add_argument("--variant", required=True, help="retention root or run id")
    compare.add_argument("--out", required=True)
    compare.add_argument("--config", help="needed only when resolving run ids")
    compare.set_defaults(handler=cmd_compare)

    demo = subparsers.add_parser(
        "demo", help="prepare isolated inputs and run zero-provider preflight"
    )
    demo.add_argument("--out", required=True)
    demo.add_argument("--reps", type=int, default=1)
    demo.add_argument("--scenario")
    demo.set_defaults(handler=cmd_demo)

    prepare_replay = subparsers.add_parser(
        "prepare-replay",
        help="preview, or spend-gated run, the I-06 replay preparation for one scenario",
    )
    prepare_replay.add_argument("--config", required=True)
    prepare_replay.add_argument("--scenario", required=True)
    prepare_replay.add_argument("--out")
    prepare_replay.add_argument("--acknowledge-provider-spend", action="store_true")
    # AC-F11-06a S2: same four flag names as add_execution_arguments, but
    # parser-optional -- mandatory only in the handler, once
    # --acknowledge-provider-spend is present (§7.2). required=True here
    # would make the ceiling-free preview AC-F11-06 requires die at
    # argparse before the handler ever ran.
    prepare_replay.add_argument("--max-cost-usd")
    prepare_replay.add_argument("--max-wall-clock-seconds")
    prepare_replay.add_argument("--max-generated-tasks")
    prepare_replay.add_argument("--max-provider-calls")
    prepare_replay.set_defaults(handler=cmd_prepare_replay)
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return int(args.handler(args))
    except OperatorError as exc:
        print(f"e40-benchmark: {exc}", file=sys.stderr)
        return exc.exit_code
    except KeyboardInterrupt:
        print("e40-benchmark: interrupted", file=sys.stderr)
        return 130


if __name__ == "__main__":
    raise SystemExit(main())
