# Clean agent_fixture_checkout fixture

Committed stand-in for an agent-visible fixture checkout root that carries
no evaluator-only material. Used by verify-evidence-roots.sh's offline test
paths (paired with ../scratch-project) as the positive control: the guard
must exit 0 ("CLEAN") against this pair and ../../package/package.yaml.
