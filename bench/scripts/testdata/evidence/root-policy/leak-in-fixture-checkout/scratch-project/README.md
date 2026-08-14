# Clean scratch_shark_project fixture

Committed stand-in for an agent-visible scratch Shark project root that
carries no evaluator-only material. Used by verify-evidence-roots.sh's
offline test paths (paired with ../fixture-checkout) as the positive
control: the guard must exit 0 ("CLEAN") against this pair and
../../package/package.yaml.
