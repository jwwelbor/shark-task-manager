"""Exercise C27 refusals and paginated eligibility against candidate Shark."""
import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import subprocess

root = Path(__file__).resolve().parents[5]
spec = importlib.util.spec_from_file_location("question_fixture", root / "tests/contracts/testdata/b077_question_route.py")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)


def probe(mode):
    case = module.QuestionRoute()
    try:
        case.setUp()
        project = case.project
        key = "Q001"
        if mode == "later_page":
            with sqlite3.connect(project / "shark-tasks.db") as db:
                for n in range(2, 102):
                    db.execute("""INSERT INTO questions (key,title,status,summary,requester,blocking,context_data)
                        SELECT ?,title,status,summary,requester,blocking,context_data FROM questions WHERE key='Q001'""",
                               (f"Q{n:03d}",))
            key = "Q101"
        request = case.shark("next", key, "--json")
        (project / "request.json").write_text(request)
        case.shark("claim", key, "--by", "alice", "--session", "original")
        if mode == "takeover":
            case.shark("claim", key, "--by", "alice", "--session", "successor", "--force")
        elif mode == "wrong_holder":
            case.shark("claim", key, "--by", "bob", "--session", "original", "--force")
        elif mode == "terminal":
            case.shark("question", "withdraw", key, "--resolution-owner", "owner", "--reason", "withdrawn")
        elif mode == "other_question_only":
            case.shark("question", "create", "Second question", "--summary", "Different scope", "--requester", "asker")
            case.shark("question", "configure-workflow", "Q002", "--resolution-owner", "owner", "--responder", "alice")
            case.respond("alice", "original")
        case.binary.unlink()
        case.binary.write_text("""#!/bin/sh
printf '%s\\n' "$*" >> commands
if [ "$1 $2" = 'question open-by-responder' ] && [ -f refuse ]; then exit 1; fi
exec "$B077_SHARK" "$@"
""")
        case.binary.chmod(0o700)
        if mode == "unavailable":
            (project / "refuse").touch()
        rc = case.submit("original").returncode
        calls = (project / "commands").read_text().splitlines()
        receiver = project / "received"
        assert receiver.exists() == (mode == "later_page")
        assert (rc == 0) == receiver.exists()
        if receiver.exists():
            assert receiver.read_bytes() == json.loads(request)["prompt"].encode()
            assert any("--offset 100" in c for c in calls)
        claims = json.loads(case.shark("claims", "--json"))
        assert len(claims) == 1
        expected_holder = "bob" if mode == "wrong_holder" else "alice"
        expected_session = "successor" if mode == "takeover" else "original"
        assert (claims[0]["claimed_by"], claims[0]["session_id"]) == (expected_holder, expected_session)
        return dict(case=mode, question=key, holder=expected_holder, session=expected_session,
                    exit_status=rc, submissions=int(receiver.exists()), commands=calls)
    finally:
        case.doCleanups()


if __name__ == "__main__":
    for mode in ["later_page", "takeover", "wrong_holder", "terminal", "other_question_only", "unavailable"]:
        print(json.dumps(probe(mode)), flush=True)
