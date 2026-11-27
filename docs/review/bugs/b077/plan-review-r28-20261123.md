# B077 Repair Plan Revision 28 Review

plan_path: docs/plan/bugs/B077.repair-plan.md
plan_sha256: sha256:12dc190659d0fd434c4a26eace943b67d9007932b4aa8ee169b96cf6334c0ac8

**Reviewer:** Codex independent reviewer
**Candidate:** `8bc0b769119ad11e7ce9f268c01822433a39bca1` on `fix/B077-model-discovery-20261123`.
**Review baseline:** `bbf490c6ec2bd597d6f3e6e4cdd93e68ca54fa50`, independently resolved and confirmed as an ancestor. The plan's immediate predecessor is committed revision 27 (`HEAD^`); its plan metadata declares the same code review baseline.
**Canonical prompt:** SHA-256 `91c35014136649b9d48234460e3376e70af9673885d5e9f7be907530ee5f0fee` (13,590 bytes).

## Review

Revision 28 follows Tech Director decision 4273 and Architect decision 4011. It limits the remaining work to discovering the model from the same selected Codex CLI executable used for invocation, explicitly selecting that model at preflight/dispatch, and checking worker-correlated observed provider/model identity before accepting the result. Its required evidence calls for a recording CLI proving discovery and execution use one executable, captured argv carrying the discovered selector, and rejection before launch/result acceptance when selection or correlated identity is missing, mismatched, or unverified. It prohibits hardcoded IDs, ambient defaults, inference from persona/effort, and silent fallback.

The successor-specific paragraph explicitly supersedes revision 27's fixed Codex model table for this follow-up while preserving it as historical context for delivered route-attestation work. It distinguishes the requested route, discovered selection, executable identity, dispatch selector, worker correlation, and observed identity. The scope explicitly excludes B083's E40 runtime/harness discovery. This matches decision 4273 and does not broaden the B083 plan.

I checked the complete plan and its coverage index, registered plan path, accepted route/rejection policy, referenced Architect decision 4011, and decision 4273. The original declared baseline resolves and is an ancestor; the immediate plan predecessor is revision 27. The plan preserves baseline identity through development, review and QA and calls for full baseline-to-candidate manifest verification. The existing broad route, lease, cost, effort and outcome obligations remain documented; no cost/model accounting behavior is replaced by the successor scope.

No unresolved plan-review findings remain.
