#!/usr/bin/env python3
import json
import sys

if sys.argv[1:] == ["--version"]:
    print("codex-fixture/1")
    raise SystemExit(0)

for line in sys.stdin:
    request = json.loads(line)
    if "id" not in request:
        continue
    if request.get("method") == "initialize":
        result = {"userAgent": "codex-fixture/1"}
    elif request.get("method") == "model/list":
        if not request["params"].get("cursor"):
            result = {
                "data": [
                    {
                        "model": "synthetic-first",
                        "displayName": "First",
                        "description": "first fixture",
                        "hidden": False,
                        "isDefault": True,
                        "defaultReasoningEffort": "low",
                        "supportedReasoningEfforts": [{"reasoningEffort": "low"}],
                    },
                    {
                        "model": "synthetic-hidden",
                        "displayName": "Hidden",
                        "description": "hidden fixture",
                        "hidden": True,
                        "isDefault": False,
                        "defaultReasoningEffort": "none",
                        "supportedReasoningEfforts": [{"reasoningEffort": "none"}],
                    },
                ],
                "nextCursor": "page-two",
            }
        else:
            result = {
                "data": [{
                    "model": "synthetic-second",
                    "displayName": "Second",
                    "description": "second fixture",
                    "hidden": False,
                    "isDefault": False,
                    "defaultReasoningEffort": "high",
                    "supportedReasoningEfforts": [{"reasoningEffort": "high"}],
                }],
                "nextCursor": None,
            }
    else:
        continue
    print(json.dumps({"id": request["id"], "result": result}), flush=True)
