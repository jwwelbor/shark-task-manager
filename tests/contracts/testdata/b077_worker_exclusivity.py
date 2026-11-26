"""C04: run Rider's preflight with real Shark claims and addressable recording workers."""
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from uuid import uuid4


RIDER = Path(__file__).resolve().parents[3] / "skills/shark-rider/verbs/run.md"


class WorkerExclusivity(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.root = Path(temporary.name)
        (self.root / "shark").symlink_to(os.environ["B077_SHARK"])
        self.env = dict(os.environ, PATH=str(self.root) + os.pathsep + os.environ["PATH"],
                        ENTITY_KEY="B001", CLAUDE_SID="contender", REQUEST_FILE="request")
        subprocess.run(["git", "init", "-q"], cwd=self.root, check=True)
        self.shark("admin", "init", "--non-interactive")
        self.shark("bug", "create", "Worker exclusivity")
        self.shark("status", "set", "B001", "research", "--force", "--reason", "fixture")
        for parent in ("request", "winner-request"):
            keyed = self.shark("next", "B001", "--json")
            self.assertEqual(json.loads(keyed)["action"], "spawn_agent")
            (self.root / parent).write_text(keyed)
        self.shark("claim", "B001", "--by", "winner", "--session", "original")
        self.workers = {}
        for handle in ("original-worker", "other-worker"):
            proc = subprocess.Popen(["cat"], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
            self.workers[handle] = proc
            self.addCleanup(proc.communicate, timeout=5)
        self.trace = []

    def shark(self, *args):
        return subprocess.check_output(["shark", *args], cwd=self.root, env=self.env, text=True)

    def release_original(self):
        reply = json.loads(self.shark("release", "B001", "--session", "original", "--json"))
        self.assertTrue(reply["released"])
        return {"RETIRED_KEY": "B001", "RETIRED_SID": "original", "RETIRED_HANDLE": "original-worker"}

    def run_contender(self, retirement, *, handle="original-worker", session="original", stop_result="exited"):
        run = RIDER.read_text()
        section = run.split("**Exclusivity pre-flight.**", 1)[1].split("### Route attestation", 1)[0]
        commands = re.findall(r"^```bash\n(.*?)\n```$", section, re.M | re.S)
        claim = run.split("Claim the concrete entity returned by Shark:", 1)[1].split("```bash\n", 1)[1].split("\n```", 1)[0]
        adapter = """
host_stop() { echo "stop $1"; read -r reply && [[ $reply == ok ]]; }
host_confirm_exit() { echo "confirm $1"; read -r reply && [[ $reply == ok ]]; }
"""
        guards = commands if handle else []  # run the published guard per host inventory entry
        script = "\n".join([adapter, *guards, claim, 'echo "submit $SID"'])
        env = dict(self.env, **retirement, HOST_HANDLE=handle, HOST_WORKER_SID=session)
        with subprocess.Popen(["bash", "-eu", "-c", script], cwd=self.root, env=env,
                              stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True) as parent:
            for line in parent.stdout:
                action, target = line.strip().split(" ", 1)
                self.trace.append((action, target))
                if action == "submit":
                    continue
                self.assertIn(action, ("confirm", "stop"))
                worker = self.workers[target]
                if action == "stop" and stop_result == "exited":
                    worker.terminate()
                    worker.wait(timeout=5)
                success = stop_result != "refused" if action == "stop" else worker.poll() is not None
                parent.stdin.write("ok\n" if success else "fail\n")
                parent.stdin.flush()
            return parent.wait(timeout=5)

    def assert_responds(self, handle):
        worker = self.workers[handle]
        self.assertIsNone(worker.poll())
        challenge = str(uuid4()) + "\n"
        worker.stdin.write(challenge)
        worker.stdin.flush()
        self.assertEqual(worker.stdout.readline(), challenge)

    def test_live_holder_survives_stale_retirement_reference(self):
        stale = {"RETIRED_KEY": "B001", "RETIRED_SID": "original", "RETIRED_HANDLE": "original-worker"}
        self.assertNotEqual(self.run_contender(stale), 0)
        self.assertEqual(self.trace, [])
        self.assertEqual(["original"], [c["session_id"] for c in json.loads(self.shark("claims", "--json"))])
        self.assert_responds("original-worker")

    def test_competing_claim_after_empty_inventory(self):
        self.assertNotEqual(self.run_contender({}, handle=""), 0)
        self.assertEqual(self.trace, [])
        self.assertEqual(["original"], [c["session_id"] for c in json.loads(self.shark("claims", "--json"))])
        self.assert_responds("original-worker")

    def test_only_retired_handle_is_stopped_before_submission(self):
        self.assertEqual(self.run_contender(self.release_original()), 0)
        self.assertEqual(self.trace[:2], [("stop", "original-worker"), ("confirm", "original-worker")])
        self.assertEqual([action for action, _ in self.trace], ["stop", "confirm", "submit"])
        self.assertIsNotNone(self.workers["original-worker"].poll())
        self.assert_responds("other-worker")

    def test_unattributed_worker_withholds_submission(self):
        retirement = self.release_original()
        for handle, session in (("other-worker", "original"), ("original-worker", ""), ("original-worker", "new")):
            with self.subTest(handle=handle, session=session):
                self.assertNotEqual(self.run_contender(retirement, handle=handle, session=session), 0)
                self.assertEqual(self.trace, [])
                self.assert_responds(handle)

    def test_unconfirmed_exit_withholds_submission(self):
        self.assertNotEqual(self.run_contender(self.release_original(), stop_result="still-alive"), 0)
        self.assertEqual(self.trace, [("stop", "original-worker"), ("confirm", "original-worker")])
        self.assert_responds("original-worker")

    def test_failed_stop_withholds_submission(self):
        self.assertNotEqual(self.run_contender(self.release_original(), stop_result="refused"), 0)
        self.assertEqual(self.trace, [("stop", "original-worker")])
        self.assert_responds("original-worker")


if __name__ == "__main__":
    unittest.main()
