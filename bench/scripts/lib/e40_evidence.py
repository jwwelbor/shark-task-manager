"""Small, dependency-free evidence serialization and digest primitives.

These functions intentionally have no CLI or validation policy: callers keep
their existing argument, output, and error contracts while sharing the exact
byte and JSONL representations used to identify retained evidence.
"""

import hashlib
import json


# CandidateIdentityFields is the single lifecycle comparison identity contract.
# Its order is stable because every producer and verifier hashes this exact map.
CANDIDATE_IDENTITY_FIELDS = (
    "base_commit",
    "tree_digest",
    "binary_diff_digest",
    "changed_path_digest",
    "dirty_untracked_manifest",
    "test_suite_digest",
    "scratch_content_digest",
)


def sha256_bytes(value):
    """Return the lowercase SHA-256 digest of bytes-like evidence content."""
    return hashlib.sha256(value).hexdigest()


def sha256_file(path):
    """Return the lowercase SHA-256 digest of a file without changing errors."""
    with open(path, "rb") as stream:
        return sha256_bytes(stream.read())


def canonical_digest(value):
    """Digest the compact, sorted, UTF-8 JSON representation of ``value``."""
    encoded = json.dumps(
        value, sort_keys=True, separators=(",", ":"), ensure_ascii=False
    ).encode("utf-8")
    return sha256_bytes(encoded)


def load_jsonl(path):
    """Load non-blank JSON Lines records, preserving caller parse failures."""
    with open(path, encoding="utf-8") as stream:
        return [json.loads(line) for line in stream if line.strip()]
