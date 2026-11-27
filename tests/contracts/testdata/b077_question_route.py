"""C18/C27: execute the installed Question procedure against isolated Shark state."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest


class QuestionRoute(unittest.TestCase):
    def setUp(self):
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        self.project = Path(tmp.name)
        self.binary = self.project / "shark"
        self.binary.symlink_to(os.environ["B077_SHARK"])
        subprocess.run(["git", "init", "-q"], cwd=self.project, check=True)
        self.shark("admin", "init", "--non-interactive")
        self.shark("admin", "install-shark-data")
        self.procedure = self.shark("skill", "get", "shark-attack", "workflows/route-question.md")
        self.shark("question", "create", "Choose a result", "--summary", "Clarify scope", "--requester", "asker")
        self.shark("question", "configure-workflow", "Q001", "--resolution-owner", "owner",
                   "--responder", "alice", "--responder", "bob", "--responder", "carol")

    def shark(self, *args):
        result = subprocess.run([self.binary, *args], cwd=self.project, text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        return result.stdout

    def assignment(self, responder):
        request = self.shark("next", "Q001", "--json")
        self.assertEqual(json.loads(request)["current_responder"], responder)
        (self.project / "request.json").write_text(request)

    def claim(self, responder, session):
        self.shark("claim", "Q001", "--by", responder, "--session", session)

    def respond(self, responder, session):
        self.shark("question", "respond", "Q001", "--responder", responder, "--session", session,
                   "--summary", responder + " answer", "--evidence-pointer", "notes/" + responder)

    def release(self, session):
        result = self.shark("release", "Q001", "--session", session, "--json")
        self.assertTrue(json.loads(result)["released"])

    def run_section(self, section, session, receiver, **inputs):
        start = self.procedure.split("## " + section + "\n", 1)[1].split("\n## ", 1)[0]
        commands = re.findall(r"^```bash\n(.*?)\n```$", start, re.MULTILINE | re.DOTALL)
        env = dict(os.environ, PATH=str(self.project) + os.pathsep + os.environ["PATH"],
                   REQUEST_FILE="request.json", SID=session, **inputs)
        return subprocess.run(["bash", "-c", "\n".join([*commands, receiver])], env=env,
                              capture_output=True, text=True, cwd=self.project)

    def submit(self, session):
        return self.run_section("Route", session, "jq -j .prompt request.json > received")

    def test_completed_responder_cannot_receive_stale_prompt(self):
        self.claim("alice", "first")
        self.respond("alice", "first")
        self.release("first")
        self.assignment("bob")
        self.claim("bob", "other")
        self.respond("bob", "other")
        self.release("other")
        # A's ordinary claim succeeds, although B has already answered and
        # the status remains answering. Only the focused read distinguishes it.
        self.claim("bob", "stale")
        self.assertNotEqual(self.submit("stale").returncode, 0)
        self.assertFalse((self.project / "received").exists())
        self.release("stale")
        self.assignment("carol")
        self.claim("carol", "last")
        result = self.submit("last")
        self.assertEqual(result.returncode, 0, result.stderr)
        expected = json.loads((self.project / "request.json").read_text())["prompt"]
        self.assertEqual((self.project / "received").read_bytes(), expected.encode())

    def finish_responses(self):
        for responder in ("alice", "bob", "carol"):
            self.assignment(responder)
            self.claim(responder, responder)
            self.respond(responder, responder)
            if responder != "carol":
                self.release(responder)

    def resolve(self, kind, pointer):
        return self.run_section("Resolve", "carol", 'cat "$ANSWER_FILE"',
                                RESOLUTION_OWNER="owner", RESOLUTION_KIND=kind,
                                RESOLUTION_POINTER=pointer, ANSWER_FILE="answer.json")

    def test_resolve_and_read_final_answer(self):
        self.finish_responses()
        result = self.resolve("no_lasting_consequence", "")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout), {"summary": "carol answer", "evidence_pointer": "notes/carol"})
        full = json.loads(self.shark("question", "full", "Q001", "--actor", "owner", "--json"))
        self.assertEqual(full["status"], "resolved")
        self.assertEqual([r["responder"] for r in full["responses"]], ["alice", "bob", "carol"])
        self.assertFalse(json.loads(self.shark("claims", "--json")))

    def test_refused_destination_stops_before_release_or_read(self):
        self.finish_responses()
        result = self.resolve("local_clarification", "note:999999")
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse((self.project / "answer.json").exists())
        claims = json.loads(self.shark("claims", "--json"))
        self.assertEqual([c["session_id"] for c in claims], ["carol"])
        full = json.loads(self.shark("question", "full", "Q001", "--actor", "owner", "--json"))
        self.assertEqual(full["status"], "ready_for_resolution")
        self.assertEqual(len(full["responses"]), 3)
        self.assertFalse(full.get("resolution_pointer"))

    def test_owner_read_refusal_stops_before_delivery(self):
        self.finish_responses()
        # Change only the read actor at the receiving Shark command. Resolution
        # still succeeds, and the caller already knows Carol's correct answer.
        self.binary.unlink()
        self.binary.write_text("""#!/bin/sh
if [ "$1 $2" = 'question full' ]; then
    exec "$B077_SHARK" question full "$3" --actor carol --json
fi
exec "$B077_SHARK" "$@"
""")
        self.binary.chmod(0o700)
        result = self.resolve("no_lasting_consequence", "")
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout, "")
        self.assertEqual((self.project / "answer.json").read_bytes(), b"")
        persisted = subprocess.run([os.environ["B077_SHARK"], "question", "full", "Q001",
                                    "--actor", "owner", "--json"], cwd=self.project,
                                   check=True, capture_output=True)
        self.assertEqual(json.loads(persisted.stdout)["status"], "resolved")


if __name__ == "__main__":
    unittest.main()
