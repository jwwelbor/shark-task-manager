# Retained evidence: 2026-09-04 full-family operator root

This registers the failed 2026-09-04 capture as feature evidence for
E40-F11 (REQ-F-001, AC-F11-02). It is **not** a baseline to retry or
promote -- `feature.md` §Observed evidence describes what it shows and
why. The digests below are verbatim from that section.

- **Registry entry:** `registry_id: 2026-09-04-e40-full-family`, in
  `bench/retention-registry.yaml`.
- **Classification:** `incomplete`.
- **Root path:** `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family`
  (matches `feature.md` §Observed evidence's "Retained operator root").
- **Whole-tree manifest:**
  `bench/retention-manifests/2026-09-04-e40-full-family.sha256`
  (1060 files, 145292529 bytes; recomputed and verified byte-identical by
  `bench/scripts/verify-retention-manifest.sh 2026-09-04-e40-full-family`
  at registration time, per AC-F11-01b).
- **Preflight result:**
  `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family/preflight/baseline/preflight-result.json`
- **Preflight result SHA-256:**
  `4c49f8e4839ffaf2371526a3f35fdc3e1afad3796bdcff414ed9dcb03fe02111`
  (matches `feature.md` §Observed evidence's "Preflight result SHA-256"
  verbatim).

## Immutability

`bench/scripts/lib/e40_benchmark.py`'s `ensure_external_operator_root`
rejects every one of the nine operator subcommands (`setup`, `preflight`,
`prepare-replay`, `pilot`, `baseline`, `variant`, `validate-variant`,
`compare`, `demo`, including `--retry-incomplete` on any of them) against
this root, unconditionally -- regardless of the recorded `status` or
`candidate_identity.head` (AC-F11-01). No F11 validation run writes into
it; every F11 run uses a new external operator root.

See `bench/scripts/tests/tc099_retained_root_immutability_test.sh` for the
full seam x condition decision-table proof (spec.md §7.3.1) and the
5-mutation manifest-invariance sweep (AC-F11-01b).
