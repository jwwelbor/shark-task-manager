# B086 R14 private host replay

Two Codex CLI hosts ran from isolated case projects using `.agents/skills/` copies that matched the pinned private catalog. Both received the same frozen synthetic review bundle and applied scratch-review history. Host actions went through the per-case recorder.

- `host-runs.json` records run identities, skill digests, packet/history pins, calls and independent outcomes.
- `host-verification.json` is the verifier output for the final pair.
- `preparation.json` carries the pair and source pins.
- `host-r14-replay-evidence.tar.gz` contains inputs, private skill copies, tracker snapshots, recorder logs and Codex JSONL traces. It also preserves the earlier grant-B attempt that submitted a disposition after escalation; that attempt is excluded from the passing pair.

The acting host used `gpt-6.2-astra` through Codex CLI authenticated with the existing ChatGPT login. No separate reviewer or diagnostic-provider calls, CI, or merge calls were made. The host runner and action recorder are under `scripts/testdata/b086/`.
