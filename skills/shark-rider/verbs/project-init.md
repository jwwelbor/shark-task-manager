# /shark-rider project-init — Compatibility alias

`project-init` is a deprecated compatibility alias for the progress-driven
project bootstrap flow. Keep this route thin so the canonical procedure has
one owner.

1. Print:

   > `/shark-rider project-init` is deprecated; use `/shark-rider project bootstrap`.

2. Load and follow `skills/shark-rider/verbs/project.md` with the `bootstrap`
   activity and pass the remaining arguments unchanged.

Do not create a second initialization workflow here. The canonical bootstrap
procedure owns Shark validation, progress checkpoints, child Rider actions, and
the architecture handoff.
