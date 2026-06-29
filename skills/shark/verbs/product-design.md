# /shark product-design — Product design (D01–D14)

Delegates to the **content bundle's** product-design skill.

## Procedure

1. Read and follow:
   ```
   shark skill get product-design
   ```
2. If the command fails because the skill is absent, print a clear unavailable
   message naming `shark skill get product-design` and suggest checking the
   installed shark version or `shark_data_path`, then stop.
3. **Act on the D04 stack-feedback signal.** The skill *returns* a feasibility
   verdict; this verb (the host) is what calls shark in response. When D04's
   verdict is "feasible with changes" or "not feasible", read the **track** from
   `docs/architecture/bootstrap.md` and:
   - **Greenfield** — re-run `/shark project bootstrap` in reconcile mode so the
     provisional `tech-stack.md` is revised against the verdict (its Phase 3.5).
   - **Brownfield** — the stack is fixed, so file the gap instead of absorbing it:
     ```bash
     /shark triage "<gap from D04 — required change + driver>"   # classifies as tech-debt / constraint note
     ```
   If `docs/architecture/` is absent (no bootstrap run), skip this step.

## Notes

- The skill runs vision → discovery → UX/CX design → concept validation (D01–D14),
  either as a guided orchestrator or a single D-artifact on request.
- Pass extra arguments (e.g. a specific D-artifact like `D08`) through to the skill.
- The skill *reads* the bootstrap marker (`docs/architecture/bootstrap.md`)
  for the brownfield/greenfield **track** and the recorded `tech-stack.md` /
  `integration-map.md`, so D04 tests the proposal against the real stack — but it
  only **returns** the verdict. Calling shark in response is step 3's job, not the
  skill's.
