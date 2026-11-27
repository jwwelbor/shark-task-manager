"""Preflight a keyed Rider request against the selected Codex executable."""
import argparse
from contextlib import contextmanager
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys


MAPPING = "B077-default-bug-anthropic-codex-v1"
POLICY = "codex-catalog-description-v1"
TUPLES = {
    "research": ("researcher", "haiku"),
    "repair_plan": ("architect", "sonnet"),
    "development": ("developer", "sonnet"),
    "qa": ("qa", "sonnet"),
    "plan_review": ("tech-lead", "opus"),
    "code_review": ("tech-lead", "opus"),
}
CAPABILITIES = {
    "fast and affordable model for easier tasks": {"haiku"},
    "legacy fast model for simpler tasks": {"haiku"},
    "fast, go-to model for coding and everyday work": {"haiku", "sonnet"},
    "legacy balanced model for agentic coding": {"sonnet"},
    "previous generation model for coding and everyday work": {"sonnet"},
    "legacy model for coding and everyday work": {"sonnet"},
    "previous generation model for complex coding and professional work": {"sonnet"},
    "our most capable model for complex coding and professional work": {"sonnet", "opus"},
    "efficient model that excels at visual understanding and computer use": set(),
}


class ConfigReadError(ValueError):
    pass


class Unsupported(ConfigReadError):
    pass


def read_timed_out(signum, frame):
    raise ConfigReadError("selected CLI configuration read timed out")


def read_reply(server, request):
    server.stdin.write(json.dumps(request) + "\n")
    server.stdin.flush()
    while line := server.stdout.readline(2**20 + 1):
        if len(line) > 2**20:
            raise ConfigReadError("selected CLI response exceeds the size limit")
        reply = json.loads(line)
        if not isinstance(reply, dict):
            raise ConfigReadError("selected CLI returned an invalid response")
        if reply.get("id") == request["id"]:
            if not isinstance(reply.get("result"), dict):
                raise ConfigReadError("selected CLI refused " + request["method"])
            return reply["result"]
    raise ConfigReadError("selected CLI closed before answering " + request["method"])


@contextmanager
def app_server(args, model, effort):
    command = [args.cli]
    if args.profile:
        command.extend(["--profile", args.profile])
    command.extend(["app-server", "--config", "model_provider=" + args.provider])
    if model:
        command.extend(["--config", "model=" + model])
    if effort:
        command.extend(["--config", "model_reasoning_effort=" + effort])
    with subprocess.Popen(command, cwd=args.cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                          stderr=subprocess.DEVNULL, text=True, encoding="utf-8") as server:
        try:
            read_reply(server, {"id": 1, "method": "initialize", "params": {
                "clientInfo": {"name": "shark-rider", "version": "1"}}})
            server.stdin.write('{"method":"initialized"}\n')
            yield server
        finally:
            server.kill()


def config_read(server, cwd):
    result = read_reply(server, {"id": 2, "method": "config/read", "params": {
        "cwd": cwd, "includeLayers": False}})
    config = result.get("config")
    if not isinstance(config, dict):
        raise ConfigReadError("config/read returned no configuration")
    fields = {}
    for name in ("model", "model_provider", "model_reasoning_effort"):
        if name in config:
            value = config[name]
            if value is not None and (not isinstance(value, str) or len(value) > 256):
                raise ConfigReadError("invalid configuration routing field")
            fields[name] = value
    return fields


def token(value):
    return isinstance(value, str) and 0 < len(value) <= 256 and value.isprintable() and value == value.strip()


def row_facts(row):
    description = row.get("description")
    if not isinstance(description, str) or len(description) > 1024:
        raise ConfigReadError("missing or invalid capability description")
    options = row.get("supportedReasoningEfforts")
    if not isinstance(options, list) or len(options) > 32:
        raise ConfigReadError("invalid supported efforts")
    efforts = set()
    for option in options:
        if not isinstance(option, dict) or not token(option.get("reasoningEffort")):
            raise ConfigReadError("invalid supported effort")
        efforts.add(option["reasoningEffort"])
    default = row.get("defaultReasoningEffort")
    if not token(default) or default not in efforts or type(row.get("isDefault")) is not bool:
        raise ConfigReadError("invalid catalog default")
    return dict(description=" ".join(description.casefold().split()).removesuffix("."),
                supported_efforts=sorted(efforts), default_effort=default, catalog_default=row["isDefault"])


def catalog(server, record):
    seen_cursors = set()
    rows = record["catalog"] = {}
    cursor = None
    count = 0
    for page in range(1, 65):
        response = read_reply(server, {"id": 2 + page, "method": "model/list", "params": {
            "cursor": cursor, "includeHidden": False, "modelProvider": record["selected_provider"]}})
        entries = response.get("data")
        if not isinstance(entries, list) or count + len(entries) > 512:
            raise ConfigReadError("invalid or excessive catalog entries")
        count += len(entries)
        for row in entries:
            if not isinstance(row, dict) or type(row.get("hidden")) is not bool:
                raise ConfigReadError("catalog row has unknown visibility")
            if row["hidden"]:
                continue
            model = row.get("model")
            if not token(model):
                raise ConfigReadError("catalog row has no concrete selector")
            facts = row_facts(row)
            if model in rows:
                previous = rows[model]["facts"]
                if facts != previous and "conflict" not in record:
                    record["conflict"] = dict(model=model, first=rows[model], second=dict(page=page, facts=facts))
            else:
                rows[model] = dict(page=page, facts=facts)
        cursor = response.get("nextCursor")
        if cursor is None:
            record["catalog_pages"] = page
            if "conflict" in record:
                raise ConfigReadError("conflicting visible catalog rows")
            return
        if not token(cursor) or cursor in seen_cursors:
            raise ConfigReadError("invalid or cyclic catalog cursor")
        seen_cursors.add(cursor)
    raise ConfigReadError("catalog page limit exceeded")


def choose(record, request, config):
    matches = []
    unknown = False
    for model, row in record["catalog"].items():
        facts = row["facts"]
        roles = CAPABILITIES.get(facts["description"])
        effort = request.get("effort") or config.get("model_reasoning_effort") or facts["default_effort"]
        if roles is None:
            row["comparison"] = "unknown capability"
            unknown = True
        elif request["model"] not in roles:
            row["comparison"] = "insufficient capability"
        elif effort not in facts["supported_efforts"]:
            row["comparison"] = "unsupported effort"
        else:
            row["comparison"] = "qualified"
            matches.append((model != config["model"], not facts["catalog_default"], model))
    if not matches:
        if unknown:
            raise ConfigReadError("catalog cannot establish capability")
        raise Unsupported("no suitable model: insufficient capability or unsupported effort")
    _, _, model = min(matches)
    row = record["catalog"][model]
    source = "requested" if request.get("effort") else "profile" if config.get("model_reasoning_effort") else "row default"
    effort = request.get("effort") or config.get("model_reasoning_effort") or row["facts"]["default_effort"]
    branch = "configured" if model == config["model"] else "alternative"
    record["comparison"] = dict(branch=branch, model=model, effort=effort, effort_source=source)
    return model, effort


def check_mapped_host(args, record):
    reason = args.claude_reason
    reasons = {"host_unavailable", "authentication_failed", "selector_unavailable",
               "observation_unavailable", "alias_unresolved", "adapter_failed"}
    auth = args.claude_auth if args.claude_auth in ("unavailable", "failed", "authenticated") else None
    eligibility = args.claude_eligibility if args.claude_eligibility in ("eligible", "unavailable", "unusable") else None
    record["exact_claude"] = dict(auth=auth, eligibility=eligibility, reason=reason if reason in reasons else None)
    if (eligibility not in ("unavailable", "unusable") or reason not in reasons or not auth
            or (reason == "authentication_failed" and auth != "failed")):
        raise ConfigReadError("exact Claude route has no established reason for substitution")
    version = subprocess.run([args.cli, "--version"], cwd=args.cwd, capture_output=True)
    checked = version.stdout.decode().strip().removeprefix("codex-cli ")
    if version.returncode or not token(checked) or checked != args.cli_version.removeprefix("codex-cli "):
        raise ConfigReadError("selected executable version differs from capability check")
    record["cli_version"] = checked
    profile = ["--profile", args.profile] if args.profile else []
    auth = subprocess.run([args.cli, *profile, "login", "status"], cwd=args.cwd,
                          stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    record["codex_auth"] = "failed" if auth.returncode else "authenticated"
    if auth.returncode:
        raise ConfigReadError("selected Codex CLI authentication failed")


def select(args, request, record):
    if not Path(args.cli).is_absolute() or not Path(args.cwd).is_absolute():
        raise ConfigReadError("selection requires an absolute executable and work directory")
    explicit = request.get("effort")
    if args.mapping:
        record["policy"] = POLICY
        if (args.mapping != MAPPING or args.provider != "openai" or request["provider"] != "anthropic"
                or request.get("entity_type") != "bug" or request.get("action") != "spawn_agent"
                or TUPLES.get(request.get("status")) != (request.get("agent_type"), request["model"])):
            raise Unsupported("request outside authorized fallback tuples")
        check_mapped_host(args, record)
    with app_server(args, None if args.mapping else request["model"], explicit) as server:
        fields = record["readback"] = config_read(server, args.cwd)
        if fields.get("model_provider") != args.provider or not args.provider:
            raise ConfigReadError("provider missing or conflicting")
        record["selected_provider"] = args.provider
        model = fields.get("model")
        if not token(model):
            raise ConfigReadError("missing or invalid concrete model")
        if explicit and fields.get("model_reasoning_effort") != explicit:
            raise ConfigReadError("explicit effort missing or conflicting")
        if args.mapping:
            catalog(server, record)
        elif model != (request["model"] or args.default_model):
            raise ConfigReadError("effective model differs from the pin or documented default")
    if args.mapping:
        model, effort = choose(record, request, fields)
        with app_server(args, model, explicit) as server:
            confirmed = record["confirmation"] = config_read(server, args.cwd)
        effective_effort = confirmed.get("model_reasoning_effort")
        if not explicit and not fields.get("model_reasoning_effort") and effective_effort is None:
            effective_effort = record["catalog"][model]["facts"]["default_effort"]
        if (confirmed.get("model_provider"), confirmed.get("model"), effective_effort) != (args.provider, model, effort):
            raise ConfigReadError("pinned confirmation differs from catalog selection")
        record["selected_effort"] = effort
        record["reason"] = f"{record['comparison']['branch']} model qualified for capability and effort"
    record["selected_model"] = model
    if not args.observable:
        raise ConfigReadError("producing-worker observation unavailable")


def publish(record):
    if record.get("mapping_ref"):
        # This file holds the bounded decision. The attempt and final receipt both
        # identify it, including when the reader refused before a claim.
        decision = {k: v for k, v in record.items() if k != "evidence_ref"}
        data = json.dumps(decision, sort_keys=True, separators=(",", ":")).encode()
        path = Path(record["evidence_ref"] + ".decision.json")
        with path.open("xb") as output:
            output.write(data)
        record.update(decision_ref=str(path), decision_sha256=hashlib.sha256(data).hexdigest())
    print(json.dumps(record))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for field in ("cli", "cwd", "request", "provider", "evidence-ref"):
        parser.add_argument("--" + field, required=True)
    for field in ("profile", "mapping", "default-model", "sandbox", "cli-version",
                  "claude-auth", "claude-eligibility", "claude-reason"):
        parser.add_argument("--" + field, default="")
    parser.add_argument("--observable", action="store_true")
    args = parser.parse_args()
    request = json.loads(Path(args.request).read_text())
    record = dict(phase="preflight", requested_model=request["model"], requested_provider=request["provider"],
                  host_worker_class="codex-cli", evidence_ref=args.evidence_ref,
                  executable=args.cli, workdir=args.cwd, profile=args.profile, sandbox=args.sandbox)
    if args.mapping:
        record["mapping_ref"] = args.mapping
        context = {key: os.environ.get(key) for key in ("CODEX_HOME", "HOME", "PATH")}
        record["environment_sha256"] = hashlib.sha256(json.dumps(context, sort_keys=True).encode()).hexdigest()
        record["request"] = {key: request[key] for key in
                             ("entity_type", "entity_key", "action", "status", "agent_type", "provider", "model", "effort")
                             if key in request}
    if request.get("effort"):
        record["effort"] = request["effort"]
    try:
        signal.signal(signal.SIGTERM, read_timed_out)
        select(args, request, record)
    except (ConfigReadError, OSError, UnicodeError, json.JSONDecodeError) as error:
        record["route_status"] = "unverified"
        if isinstance(error, Unsupported):
            record["route_status"] = "unsupported"
            record.pop("selected_provider", None)
        record["reason"] = str(error)[:200]
    try:
        publish(record)
    except OSError:
        record.update(route_status="unverified", reason="unable to save a fresh preflight decision")
        print(json.dumps(record))
        return 1
    return int("route_status" in record)


if __name__ == "__main__":
    sys.exit(main())
