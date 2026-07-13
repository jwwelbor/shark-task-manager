# /shark-rider project product-design — Product design (D01–D14)

This is the owning Rider adapter for the bundled `product-design` methodology.
Bootstrap loads and follows this procedure in the same coordination run instead
of emitting a nested host-level dispatch.

## Procedure

1. Read the current `docs/product/progress.md` and derive the selected scope,
   current estate, initiative posture, and next artifact from actual files.
2. Read and follow the bundle through this adapter:
   `shark skill get product-design`.
   If retrieval fails, report the unavailable bundle and stop. Do not fall back
   to a copied procedure.
3. Before producing an artifact, mark its checklist item `[~]` in the derived
   view when the progress file is writable. After each D01–D14 artifact is
   written or materially revised, checkpoint immediately.
4. At each checkpoint, inspect the artifact and its evidence dependencies,
   preserve the Decision Log and Cross-Epic Integration Map, regenerate derived
   setup groups, and append only decisions, approvals, reasons, or deferrals.
5. On interruption, leave completed artifacts `[x]`, the active artifact `[~]`,
   and the next required artifact visible. Re-entry resumes from evidence; it
   does not trust a manually changed checkbox or restart completed work.
6. Return a `CHILD ACTION RESULT` to the bootstrap coordinator with the
   artifacts, checkpoint state, next missing artifact, and any D04 feedback.

## D04 stack feedback

The bundle returns a structured stack-feedback signal. This adapter owns the
response:

- `feasible as described`: still run the architecture integration gate and
  clear provisional metadata after reconciliation, even when no stack choice
  changes.
- `feasible with changes` or `not feasible` in greenfield: return a
  `stack_feedback` signal requesting bootstrap reconciliation against D01, D04,
  and D07. The already-running bootstrap coordinator owns that transition.
- `feasible with changes` or `not feasible` in brownfield: classify the gap by
  the initiative posture and return a deferred-remediation signal only for
  genuinely deferred work or a constraint note. The current stack is evidence,
  not automatically a universal hard constraint.

Never let the bundle invoke the CLI or retrieve another skill. This adapter is
the only place in this action that owns the `shark skill get product-design`
integration.

## Direct invocation

Direct `/shark-rider project product-design [D0X|scope]` uses the same
checkpoint contract as bootstrap. If `docs/product/progress.md` is missing,
seed it from `file_templates/progress.md` before starting and report any
read-only or malformed-file warning.
