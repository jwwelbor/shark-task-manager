# /shark-rider project brownfield-analysis — Brownfield analysis

This is the owning Rider action for comprehensive analysis of an existing
codebase. Bootstrap dispatches this action only when the selected depth or risk
profile requires it; the lightweight baseline remains part of bootstrap.

Usage: `/shark-rider project brownfield-analysis [path] [scope]`

## Procedure

1. Read `docs/product/progress.md` and inspect the repository evidence. Confirm
   the selected `brownfield_depth`, initiative posture, and remaining analysis
   areas. Do not infer that the current stack is a permanent constraint.
2. Read and follow the host-local methodology at
   `skills/shark-rider/skills/brownfield-analysis/SKILL.md`, passing the target
   path and selected scope. The methodology owns analytical quality; this
   Rider action owns ordering, resumability, and progress checkpoints.
3. Before each selected analysis area, mark the area `[~]` in the derived
   progress view when writable. After each durable output or completed area,
   inspect the output and checkpoint `docs/product/progress.md` immediately.
4. Preserve the Decision Log and Cross-Epic Integration Map. Append only
   choices, approvals, reasons, or deferrals. Record hard, soft, and unresolved
   constraints with their evidence source and owner.
5. If interrupted, leave completed areas `[x]`, the active area `[~]`, and the
   remaining selected scope visible. Re-entry resumes from observed outputs,
   not from checklist text.
6. Return a `CHILD ACTION RESULT` to the bootstrap coordinator with the
   completed areas, durable outputs, checkpoint state, and next missing area.

## Direct invocation and recovery

Direct invocation follows the same checkpoint contract as bootstrap. If the
progress file is missing, seed it from `file_templates/progress.md`. If it is
malformed, preserve the original and report a non-destructive recovery path.
If it cannot be written, warn that checkpoints were not saved and continue
from observed evidence without claiming persistence.

The analysis action does not invoke the Shark CLI, retrieve another bundle, or
claim, advance, or release a Shark entity. Any CLI integration belongs to the
owning parent Rider action or the Shark workflow loop.
