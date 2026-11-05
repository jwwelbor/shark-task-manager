# CC-044 owner decisions — 2026-11-04

The user authorized proceeding with the CX and architect recommendations, with these explicit refinements:

- Project documents under `docs/plan` may live in a parent checkout or worktree, but must be committed and not Git-ignored. New `docs/review` and `docs/reviews` artifacts are local evidence and may be copied to the parent like `dev-artifacts`; do not remove historical tracked reviews. Preserve the actual checkout identity when resolving committed documents.
- Document the project vocabulary, including `--workdir`. Preserve its agent-working-directory contract; support linked-worktree locations and fail before dispatch with actionable root details when the relationship cannot be established safely. The unrelated scratch-project/fixture-repository contract remains with E40/B083.
- Preserve Claude provider metadata and deliver worker outcomes through an owned transport boundary. Claude is unavailable locally; defer that boundary to TD-238 and do not infer model/cost or claim a successful provider run. CC-044's existing result and telemetry limitations remain explicit.
- The approved plan snapshot must be authoritative for downstream gates and workers, identified by digest. A material plan change requires a new revision and plan review. Project documents remain committed.
- Multiple outcomes may share a target when their admission meaning is the same. Reject the conflicting case where success and rework from the same bug gate share a destination requiring success evidence; do not reject multiple rework outcomes merely for sharing a target.
- Legacy bug research must expose a recognizable success route; for this bounded change require its existing `pass` outcome and fail incompatible configuration early. Do not infer arbitrary outcome names. An explicit success-role extension is separate work.

The user said "proceed with recommendations" after the prior prompts; the working interpretation also applies their displayed recommended options for the later plan-review report reference, real baseline commit, and evidence-backed blocker downgrade. These interpretations remain subject to correction. A report reference and an immutable plan snapshot are separate requirements: local report retention must not cause `docs/plan` project documents to be silently discarded.
