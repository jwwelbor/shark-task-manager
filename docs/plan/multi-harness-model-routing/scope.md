# E41 scope boundaries

The [epic PRD](./epic.md#scope) sets the release boundary. Use these cases to decide whether a proposed feature belongs in E41.

| Area | Included | Excluded |
|---|---|---|
| Workflow authoring | Three declared complexity levels, named exceptions and opt-in execution profiles | Inferring complexity from prompt text or agent name, redesigning E35 outcomes |
| Supported workers | Qualifying the three initial managed provider bindings for specific executable versions and modes | Arbitrary command templates, additional providers or bring-your-own-key bindings in the initial support claim |
| Review | Policy checks against the exact candidate, including fallback authors and multiple contributors | Replacing the existing gate system or changing its authorized manual disposition |
| Recovery | Bounded alternates, protected result application and one eligible final Rider inline attempt | Replaying uncertain mutations, switching credentials to escape a denial, or inline execution in controlled E40 runs |
| Migration and comparison | Compatibility lane, rollback at a run boundary, frozen E40 profiles and retained evidence | Fleet-wide forced migration, changing E40 calibration or publication criteria |
| Interface | Workflow files, CLI diagnostics, structured output and Rider instructions | Graphical configuration or monitoring UI. Wireframes are unnecessary for these CLI and file-based interactions. |

## Discovery

Initialize an advisory inventory of installed executable paths and versions during `shark admin init` when it is missing, or lazily on the first routing lookup if setup did not create it. `shark admin harness refresh` explicitly reruns the local inventory probe. Refresh writes a new snapshot only after a successful probe and retains the last known snapshot as stale if refresh fails. This inventory does not claim model compatibility, account entitlement or current service availability; strict routing rechecks those separately at launch. Authenticated canaries still require separate approval. Package-manager installation hooks and automatic account/model probes are out of scope. [Validation and evidence](./02-architecture.md#validation-and-evidence)
