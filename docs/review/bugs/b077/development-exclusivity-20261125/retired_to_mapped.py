"""Exercise a retired host handle followed by the receiving mapped Codex recorder."""
import hashlib
import json
import os
from pathlib import Path
import re
import sys
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "development-6c6345cf"))
from assignment_probe import fixture


case = fixture.RiderCLI()
try:
    case.setUp()
    case.shark("claim", "B001", "--by", "prior", "--session", "retired")
    release = json.loads(case.shark("release", "B001", "--session", "retired", "--json").stdout)
    assert release["released"] and release["entity_key"] == "B001"
    text = (fixture.RIDER / "verbs/run.md").read_text()
    section = text.split("**Exclusivity pre-flight.**", 1)[1].split("### Route attestation", 1)[0]
    guard, = re.findall(r"^```bash\n(.*?)\n```$", section, re.M | re.S)
    # Bash owns this specific background process and can wait for it by its original handle.
    recording_host = """
    sleep 60 &
    original_pid=$!
    host_stop() { [[ $1 == original-handle ]] && kill "$original_pid"; }
    host_confirm_exit() {
        [[ $1 == original-handle ]] || return 1
        local rc=0
        wait "$original_pid" || rc=$?
        [[ $rc == 143 ]]
    }
    """
    dispatch = "\n".join([recording_host, guard, fixture.commands()])
    with patch.dict(os.environ, RETIRED_KEY=release["entity_key"], RETIRED_SID="retired",
                    RETIRED_HANDLE="original-handle", HOST_HANDLE="original-handle", HOST_WORKER_SID="retired"), \
            patch.object(fixture, "commands", return_value=dispatch):
        calls, selection, rc = case.submit("haiku", dict(model_provider="openai", model="discovered-for-new-lease",
                                                         model_reasoning_effort="low"), mapped=True)
    assert rc == 0, case.stderr
    assert [c["method"] for c in calls] == ["read", "exec"]
    keyed = json.loads((case.directory / "current").read_bytes())
    received = calls[1]["prompt"].encode()
    receipt = json.loads((case.directory / "observation").read_bytes())
    assert receipt["route_status"] == "verified" and received == keyed["prompt"].encode()
    assert selection["selected_model"] == receipt["effective_model"] == "discovered-for-new-lease"
    history = json.loads(case.shark("status", "history", "B001", "--json").stdout)["history"]
    assert "researcher@openai/discovered-for-new-lease" in [h.get("agent") for h in history]
    print(json.dumps(dict(previous_release=release, authorized_host_operations=["stop original-handle",
                          "wait original-handle (exit 143)"], receiving_calls=[c["method"] for c in calls],
                          requested=[keyed["provider"], keyed["model"]], observed=[receipt["effective_provider"],
                          receipt["effective_model"]], thread=calls[1]["thread"], bytes=len(received),
                          sha256=hashlib.sha256(received).hexdigest(), parent_commands=(case.directory /
                          "counts").read_text().splitlines(), result_actor="researcher@openai/discovered-for-new-lease")))
finally:
    case.doCleanups()
