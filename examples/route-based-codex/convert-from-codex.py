#!/usr/bin/env python3
"""Convert a legacy status_flow/status_metadata workflow (codex) into the
route-based steps: schema (E35). Collapses ready_for_X / in_X pairs into a
single step, derives pass/fail/blocked outcomes by phase order, and carries
aliases for migration + compat.

Usage: convert.py <codex.json> <out_dir>
Emits <out_dir>/workflow/<entity>.yaml + <out_dir>/workflow.yaml (index).
"""
import json
import os
import sys

PHASE_ORDER = {
    "planning": 0, "triage": 1, "refinement": 1, "research": 2, "assessment": 2,
    "investigation": 2, "specification": 3, "design": 3, "decomposition": 4,
    "task_generation": 4, "test_planning": 5, "task_review": 5,
    "feature_review": 5, "resolution": 6, "development": 6, "review": 7,
    "code_review": 7, "qa": 8, "approval": 9, "execution": 9, "done": 10,
    "blocked": -1, "paused": -1, "any": -1, "": 99,
}

PARKING_STATUSES = {"blocked", "on_hold"}
# Extra live statuses (present in the DB) that the codex workflow does not
# define; mapped here to the nearest collapsed step so migration leaves no
# orphan. Approximate by design — documented in the guide.
ORPHAN_ALIASES = {
    "task": {
        "todo": "draft", "in_progress": "development",
        "ready_for_refinement_ba": "draft",
        "ready_for_qa": "development", "in_qa": "development",
        "ready_for_code_review": "development",
        "ready_for_approval": "development", "in_approval": "development",
    },
}

ENTITY_FILES = {
    "epic_workflow": ("epic", "epic.yaml"),
    "feature_workflow": ("feature", "feature.yaml"),
    "task_workflow": ("task", "task.yaml"),
    "bug_workflow": ("bug", "bug.yaml"),
    "change_workflow": ("change", "change.yaml"),
    "tech_debt_workflow": ("tech_debt", "tech-debt.yaml"),
}


def step_name(status):
    """Collapse a status to its route-based step name."""
    if status.startswith("ready_for_"):
        return status[len("ready_for_"):]
    if status.startswith("in_"):
        return status[len("in_"):]
    return status


def phase_of(meta, status):
    return (meta.get(status) or {}).get("phase", "")


def convert_entity(entity, wf):
    flow = wf["status_flow"]
    meta = wf.get("status_metadata", {})
    special = wf.get("special_statuses", {})
    complete = set(special.get("_complete_", []))
    start = special.get("_start_", ["draft"])[0]

    # Group source statuses by collapsed step name.
    groups = {}
    for status in flow:
        groups.setdefault(step_name(status), []).append(status)

    def primary(sources):
        for pref in ("ready_for_", "in_"):
            for s in sources:
                if s.startswith(pref):
                    return s
        return sources[0]

    def step_phase(name):
        # phase of the step's primary source
        p = primary(groups[name])
        return phase_of(meta, p)

    steps = {}
    for name, sources in groups.items():
        p = primary(sources)
        m = meta.get(p, {})
        st = {}
        if m.get("phase"):
            st["phase"] = m["phase"]
        if m.get("color"):
            st["color"] = m["color"]
        if m.get("display_token"):
            st["display_token"] = m["display_token"]
        if m.get("description"):
            st["description"] = m["description"]
        if m.get("progress_weight") is not None:
            st["progress_weight"] = m["progress_weight"]
        if m.get("responsibility"):
            st["responsibility"] = m["responsibility"]
        if m.get("is_planning"):
            st["is_planning"] = True
        if m.get("aggregates_from"):
            st["aggregates_from"] = m["aggregates_from"]
        if m.get("blocks_feature"):
            st["blocks_feature"] = True
        if m.get("exclude_from_progress"):
            st["exclude_from_progress"] = True
        if m.get("sprint_bucket") is not None:
            st["sprint_bucket"] = m["sprint_bucket"]

        oa = m.get("orchestrator_action") or {}
        if oa.get("action"):
            st["action"] = oa["action"]
        if oa.get("agent_type"):
            st["agent"] = oa["agent_type"]
        if oa.get("provider"):
            st["provider"] = oa["provider"]
        if oa.get("model"):
            st["model"] = oa["model"]
        if oa.get("skills"):
            st["skills"] = list(oa["skills"])
        if oa.get("instruction_template"):
            st["prompt"] = oa["instruction_template"]

        is_terminal = name in {step_name(c) for c in complete} and all(
            step_name(t) == name for s in sources for t in flow.get(s, [])
        ) or name in {step_name(c) for c in complete}
        is_parking = name in PARKING_STATUSES or m.get("phase") in {"blocked", "paused"}

        if is_terminal:
            st["terminal"] = True
        elif is_parking:
            st["parking"] = True
        else:
            # Derive outcomes from the union of collapsed targets.
            targets = []
            for s in sources:
                for t in flow.get(s, []):
                    tn = step_name(t)
                    if tn != name and tn not in targets:
                        targets.append(tn)
            cur = PHASE_ORDER.get(step_phase(name), 99)

            def is_cancel(t):
                tm = meta.get(primary(groups.get(t, [t])), {}) if t in groups else {}
                return t == "cancelled" or tm.get("exclude_from_progress")

            forward = [t for t in targets
                       if PHASE_ORDER.get(step_phase(t), 99) > cur
                       and t not in PARKING_STATUSES and not is_cancel(t)]
            backward = [t for t in targets if 0 <= PHASE_ORDER.get(step_phase(t), 99) < cur and t not in PARKING_STATUSES]
            outcomes = {}
            # pass: the NEAREST forward step (next phase), else the completion
            # step, else self.
            if forward:
                outcomes["pass"] = min(forward, key=lambda t: PHASE_ORDER.get(step_phase(t), 99))
            else:
                term = [t for t in targets if t in {step_name(c) for c in complete} and not is_cancel(t)]
                outcomes["pass"] = term[0] if term else name
            # fail: nearest-backward, else start.
            if backward:
                outcomes["fail"] = min(backward, key=lambda t: PHASE_ORDER.get(step_phase(t), 99))
            else:
                outcomes["fail"] = step_name(start)
            # blocked: a blocked step if present in this workflow.
            outcomes["blocked"] = "blocked" if "blocked" in groups else outcomes["fail"]
            if "on_hold" in groups and "on_hold" in targets:
                outcomes["on_hold"] = "on_hold"
            # Preserve every other original transition as an extra outcome keyed
            # by the target step name, so the route stays faithful and all
            # statuses (cancelled, wont_fix, …) remain reachable.
            assigned = set(outcomes.values())
            for t in targets:
                if t not in assigned and t != name:
                    outcomes[t] = t
                    assigned.add(t)
            st["outcomes"] = outcomes

        # aliases: source statuses whose name differs from the step.
        aliases = [s for s in sorted(sources) if s != name]
        if aliases:
            st["aliases"] = aliases
        steps[name] = st

    # Orphan live-status aliases (best effort).
    for old, target in ORPHAN_ALIASES.get(entity, {}).items():
        if target in steps and old not in steps:
            steps[target].setdefault("aliases", [])
            if old not in steps[target]["aliases"]:
                steps[target]["aliases"].append(old)
                steps[target]["aliases"].sort()

    return {"version": "1.0", "start": step_name(start), "steps": steps}


def emit_yaml(doc):
    """Minimal deterministic YAML emitter for the step doc shape."""
    lines = [f'version: "{doc["version"]}"', f'start: {doc["start"]}', "steps:"]
    for name in sorted(doc["steps"]):
        st = doc["steps"][name]
        lines.append(f"  {name}:")
        for key in ("phase", "color", "display_token", "description",
                    "progress_weight", "responsibility", "is_planning",
                    "aggregates_from", "blocks_feature", "exclude_from_progress",
                    "sprint_bucket", "action", "agent", "provider", "model",
                    "terminal", "parking"):
            if key in st:
                v = st[key]
                if isinstance(v, bool):
                    lines.append(f"    {key}: {'true' if v else 'false'}")
                elif isinstance(v, str):
                    lines.append(f"    {key}: {yaml_scalar(v)}")
                else:
                    lines.append(f"    {key}: {v}")
        if "skills" in st:
            lines.append(f"    skills: [{', '.join(st['skills'])}]")
        if "prompt" in st:
            lines.append(f"    prompt: {yaml_scalar(st['prompt'])}")
        if "outcomes" in st:
            lines.append("    outcomes:")
            for ok in sorted(st["outcomes"]):
                lines.append(f"      {ok}: {st['outcomes'][ok]}")
        if "aliases" in st:
            lines.append(f"    aliases: [{', '.join(st['aliases'])}]")
    return "\n".join(lines) + "\n"


def yaml_scalar(s):
    if any(c in s for c in ":#{}[],&*!|>'\"%@`") or s != s.strip():
        return '"' + s.replace('"', '\\"') + '"'
    return s


def main():
    codex_path, out_dir = sys.argv[1], sys.argv[2]
    d = json.load(open(codex_path))
    wf_dir = os.path.join(out_dir, "workflow")
    os.makedirs(wf_dir, exist_ok=True)
    index = {"entities": {}}
    for key, (entity, fname) in ENTITY_FILES.items():
        if key not in d:
            continue
        doc = convert_entity(entity, d[key])
        with open(os.path.join(wf_dir, fname), "w") as f:
            f.write(emit_yaml(doc))
        index_entity = "tech-debt" if entity == "tech_debt" else entity
        index["entities"][index_entity] = f"workflow/{fname}"
    # index file
    idx_lines = ["entities:"]
    for e in sorted(index["entities"]):
        idx_lines.append(f"  {e}: {index['entities'][e]}")
    with open(os.path.join(out_dir, "workflow.yaml"), "w") as f:
        f.write("\n".join(idx_lines) + "\n")
    print(f"wrote {len(index['entities'])} entity workflows + index to {out_dir}")


if __name__ == "__main__":
    main()
