# B086 definitive tracker refusal replay

This is the approved post-pair row 29 control. It reuses the independently verified R14 A/B pair as the prerequisite and compares two fresh cases with the same matching Feature A grant, packet, applied history, seed database, launcher and normalized host instructions. The sole injected fault is the tracker returning a definitive task-create refusal in `case-c`.

The matching case (`grant-a`, run `a99f01a4-a6be-4ede-a5ad-7b1eaa2b3f9c`) creates one task under E01-F01, reads back its type, parent, scope and fingerprint, and records Defer. The refusal case (`case-c`, run `b72122fc-3f72-4acc-bcb4-03e10b0ee042`) attempts one task create, the recorder rejects it before forwarding, the database remains empty, no disposition is recorded, and the acting host sends the parent an escalation with the fingerprint, operation, refusal reason, target feature and requested action. The independent verifier returns `observed` for both cases and the refusal boundary.

The prior verified wrong-grant run remains in `prior-pair-verification.json`, `prior-pair-input.json` and `prior-pair-runs.json`. The suite output in `fixture-suite.log` records 21 passing tests. The acting host used `gpt-6.2-astra` through Codex CLI 0.172.0 and the existing ChatGPT login. No separate reviewer/provider, CI or merge calls were made in these host cases.
