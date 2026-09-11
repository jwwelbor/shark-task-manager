#!/usr/bin/env python3
"""bench/scripts/lib/i05_validation

The SINGLE owner of the I-05 typed-consumer-edge shape check: an artifact's
`consumers[]` entry is valid only if it is an object containing exactly
`consuming_stage`, `edge_kind`, and `observed_at`, each a non-empty string,
with `edge_kind` drawn from the I-05 edge_kind vocabulary
(bench/evidence/i05-schema.yaml's `edge_kind` list).

Before this module existed, this exact check was independently reimplemented
in four scripts (evaluate-lifecycle.sh, aggregate-lifecycle.sh,
verify-lifecycle-run.sh, verify-stage-evidence.sh) with no shared base --
already showing message-wording drift across the four copies (review finding
F3, code-review-2026-08-20T2138-E40-F10.md's precedent for this exact defect
class in lib/retain_pair's docstring). validate_typed_consumer is the single
place that decides validity; callers keep their own error-reporting shape
(raise, a `fail()` callback, or boolean accumulation) and their own existing
message wording, keyed off the returned reason code.
"""

REQUIRED_CONSUMER_FIELDS = frozenset({"consuming_stage", "edge_kind", "observed_at"})


def validate_typed_consumer(consumer, known_edge_kinds):
    """Return the first failing check's short reason code, or None if
    consumer is a valid typed-consumer-edge entry.

    Reason codes (checked in this fixed order, matching every pre-existing
    call site's own check ordering): "shape" (not a dict, or not exactly
    consuming_stage/edge_kind/observed_at), "fields" (some field is not a
    non-empty string), "edge_kind" (edge_kind is not in known_edge_kinds).
    """
    if not isinstance(consumer, dict) or set(consumer) != REQUIRED_CONSUMER_FIELDS:
        return "shape"
    if not all(isinstance(consumer[field], str) and consumer[field].strip() for field in REQUIRED_CONSUMER_FIELDS):
        return "fields"
    if consumer["edge_kind"] not in known_edge_kinds:
        return "edge_kind"
    return None
