"""Shared final-predicate operand lookup for admission and evaluation."""


def named_ids_for(kind, predicate):
    if kind == "f2p_p2p":
        return list(predicate.get("f2p_test_ids") or [])
    if kind in ("acceptance_tests", "task_acceptance_tests"):
        return list(predicate.get("acceptance_test_ids") or [])
    if kind in ("child_oracles_union", "descendant_oracles_union"):
        return list(predicate.get("integration_test_ids") or []) + list(predicate.get("child_oracles") or [])
    return []
