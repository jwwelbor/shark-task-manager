import importlib.util
import json
from pathlib import Path
import tempfile
import unittest


MODULE_PATH = Path(__file__).with_name("observe-codex-turn.py")
SPEC = importlib.util.spec_from_file_location("observe_codex_turn", MODULE_PATH)
observer = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(observer)


class ObserveCodexTurnTests(unittest.TestCase):
    def test_matching_completion_event_is_verified(self):
        self.assertEqual(
            self.observe("01a2118e-c843-7dc3-98f6-972e4507220e"), "verified",
        )

    def test_mismatched_completion_event_is_unverified(self):
        self.assertEqual(self.observe("turn-other"), "unverified")

    def test_completion_event_without_turn_id_uses_rollout_join(self):
        self.assertEqual(self.observe(None), "verified")

    def observe(self, completion_turn_id):
        worker_id = "01a2118e-c630-7bb0-b63b-587e49353dbe"
        turn_id = "01a2118e-c843-7dc3-98f6-972e4507220e"
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sessions = root / "sessions"
            sessions.mkdir()
            events = root / "events.jsonl"
            rollout = sessions / f"rollout-2026-11-25T00-00-00-{worker_id}.jsonl"
            completion_event = {"type": "turn.completed"}
            if completion_turn_id is not None:
                completion_event["turn_id"] = completion_turn_id
            self.write_jsonl(events, [
                {"type": "thread.started", "thread_id": worker_id},
                {"type": "turn.started", "turn_id": turn_id},
                completion_event,
            ])
            self.write_jsonl(rollout, [
                {"type": "session_meta", "payload": {
                    "id": worker_id, "model_provider": "openai",
                }},
                {"type": "turn_context", "payload": {
                    "turn_id": turn_id, "model": "gpt-6.2-astra", "effort": "high",
                }},
                {"type": "event_msg", "payload": {
                    "type": "task_complete", "turn_id": turn_id,
                }},
            ])
            selection = {
                "selected_provider": "openai",
                "selected_model": "gpt-6.2-astra",
                "effort": "high",
            }
            return observer.observe(
                events, root, selection, {}, rejected=False,
            )

    @staticmethod
    def write_jsonl(path, records):
        path.write_text("".join(json.dumps(record) + "\n" for record in records))


if __name__ == "__main__":
    unittest.main()
