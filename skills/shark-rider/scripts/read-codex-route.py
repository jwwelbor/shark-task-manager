"""Read one selected Codex CLI configuration without submitting a prompt."""
import argparse
import json
from pathlib import Path
import signal
import subprocess
import sys


class ConfigReadError(ValueError):
    pass


def read_timed_out(signum, frame):
    raise ConfigReadError("selected CLI configuration read timed out")


def read_reply(server, request):
    server.stdin.write(json.dumps(request) + "\n")
    server.stdin.flush()
    for line in server.stdout:
        reply = json.loads(line)
        if not isinstance(reply, dict):
            raise ConfigReadError("selected CLI returned an invalid response")
        if reply.get("id") == request["id"]:
            if not isinstance(reply.get("result"), dict):
                raise ConfigReadError("selected CLI refused " + request["method"])
            return reply["result"]
    raise ConfigReadError("selected CLI closed before answering " + request["method"])


def read_config(args, request):
    command = [args.cli]
    if args.profile:
        command.extend(["--profile", args.profile])
    command.extend(["app-server", "--config", "model_provider=" + args.provider])
    if request["model"] and not args.mapping:
        command.extend(["--config", "model=" + request["model"]])
    if request.get("effort"):
        command.extend(["--config", "model_reasoning_effort=" + request["effort"]])
    with subprocess.Popen(command, cwd=args.cwd, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                          stderr=subprocess.DEVNULL, text=True, encoding="utf-8") as server:
        try:
            read_reply(server, {"id": 1, "method": "initialize", "params": {
                "clientInfo": {"name": "shark-rider", "version": "1"}}})
            server.stdin.write('{"method":"initialized"}\n')
            result = read_reply(server, {"id": 2, "method": "config/read", "params": {
                "cwd": args.cwd, "includeLayers": False}})
            config = result.get("config")
            if not isinstance(config, dict):
                raise ConfigReadError("config/read returned no configuration")
            return {k: config[k] for k in ("model_provider", "model", "model_reasoning_effort") if k in config}
        finally:
            server.kill()


def select(args, request, record):
    if not Path(args.cli).is_absolute() or not Path(args.cwd).is_absolute():
        raise ConfigReadError("selection requires an absolute executable and work directory")
    fields = read_config(args, request)
    record["readback"] = fields
    if fields.get("model_provider") != args.provider or not args.provider:
        raise ConfigReadError("provider missing or conflicting")
    record["selected_provider"] = args.provider
    model = fields.get("model")
    if not isinstance(model, str) or not model.strip() or model != model.strip() or not model.isprintable():
        raise ConfigReadError("missing or invalid concrete model")
    if not args.mapping and model != (request["model"] or args.default_model):
        raise ConfigReadError("effective model differs from the pin or documented default")
    record["selected_model"] = model
    if request.get("effort") and request["effort"] != fields.get("model_reasoning_effort"):
        raise ConfigReadError("explicit effort missing or conflicting")
    if not args.observable:
        raise ConfigReadError("producing-worker observation unavailable")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for field in ("cli", "cwd", "request", "provider", "evidence-ref"):
        parser.add_argument("--" + field, required=True)
    for field in ("profile", "mapping", "default-model", "sandbox"):
        parser.add_argument("--" + field, default="")
    parser.add_argument("--observable", action="store_true")
    args = parser.parse_args()
    request = json.loads(Path(args.request).read_text())
    record = dict(phase="preflight", requested_model=request["model"], requested_provider=request["provider"],
                  host_worker_class="codex-cli", evidence_ref=args.evidence_ref,
                  executable=args.cli, workdir=args.cwd, profile=args.profile, sandbox=args.sandbox)
    if args.mapping:
        record["mapping_ref"] = args.mapping
    if request.get("effort"):
        record["effort"] = request["effort"]
    try:
        # Rider's timeout sends SIGTERM while the reader awaits the selected CLI.
        signal.signal(signal.SIGTERM, read_timed_out)
        select(args, request, record)
    except (ConfigReadError, OSError, UnicodeError, json.JSONDecodeError) as error:
        record["route_status"] = "unverified"
        record["reason"] = str(error)[:200]
        print(json.dumps(record))
        return 1
    print(json.dumps(record))
    return 0


if __name__ == "__main__":
    sys.exit(main())
