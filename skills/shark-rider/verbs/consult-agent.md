# /shark-rider consult-agent — Specialist subagent consultation

Consult a Shark agent persona through a bounded host subagent when the host can
enforce read-only permissions. The parent remains the lead advisor: it provides
context, reviews the response, and synthesizes advice in this conversation. If
the host cannot enforce read-only permissions, use inline `consult` instead.
This is a local consultation recipe, not keyed Shark workflow dispatch.

Usage:

```
/shark-rider consult-agent                            # show consultation help and stop
/shark-rider consult-agent help                       # show consultation help and stop
/shark-rider consult-agent <agent>                    # consult agent; ask what to discuss
/shark-rider consult-agent <agent> <referent>         # consult agent about a specific thing
```

`<agent>` is an agent name or a fuzzy keyword.
`<referent>` is anything: a file path, an entity key, "this" (most recent
artifact in context), or plain prose describing what to discuss.

---

## Procedure

### Step 0 — Handle help

If there are **no args** or the only arg is `help`, print the help response by
composing the mode-specific opening below with the shared role-selection guide
at `skills/shark-rider/context/consult-role-guide.md` verbatim. **STOP** without
resolving or adopting an agent persona:

```text
Consult a Shark specialist when a decision needs role-specific judgment. Use
`/shark-rider consult-agent <agent-name> <question, entity, or artifact>` to get
one-shot, read-only advice from a host subagent. Include the relevant context
and ask for a concrete recommendation. The parent reviews and synthesizes the
response. Use `/shark-rider consult <agent> <question or artifact>` to continue
the current conversation in the persona voice.
```

Then **STOP**. Do not treat `help` as an agent name.

---

### Step 1 — Resolve the request

For every other invocation, read and follow
`skills/shark-rider/context/consult-request-resolution.md` with the arguments
that remain after this verb. It parses the agent and referent, loads the persona,
and reads the requested context. If that procedure stops for any reason, stop
here too; do not adopt or dispatch a persona after a failed or incomplete
resolution.

---

### Step 2 — Prepare the consultation context

Print a brief announcement (one line):

```
Consulting <agent-name> about <referent-summary> with a specialist subagent...
```

### Step 3 — Select and launch a suitable subagent

Use the `$efficient-advisor` approach: the parent remains responsible for
framing the question, choosing a suitable contributor, judging the evidence,
and integrating the answer. For a concrete specialist consultation, launch a
host subagent using the strongest suitable model currently available. Astra,
Sol, Fable, and Opus are examples only; check the active host's supported
models and choose by task fit, context needs, availability, latency, and cost.
Do not assume a fixed provider, model ranking, or launcher syntax.

Use a host-safe general-purpose subagent type (for example, `general-purpose`
where the host requires a `subagent_type`). The Shark persona belongs in the
handoff prompt, not in the host subagent type. Use the host's native subagent
launcher and model selector; do not claim a subagent was launched unless it
actually started and returned a result.

Require a host-provided read-only sandbox or tool profile that prevents workspace
writes, Shark mutations, and nested dispatch. If the launcher cannot enforce all
three parts of that boundary, do not launch the subagent; explain that delegated
read-only advice is unavailable on this host and direct the user to inline
`consult` instead. A prompt instruction alone does not count as an enforced
read-only profile.

Give the subagent a self-contained, bounded handoff containing:

- The requested question or decision and the resolved referent.
- The Shark persona definition and relevant retrieved skill/context content.
- Only the project facts and artifact content needed to advise.
- A read-only scope: inspect and recommend, but do not edit files, mutate Shark
  state, launch other agents, or perform implementation. This handoff constraint
  supplements the required read-only sandbox or tool profile; it does not replace
  one.
- The expected response: a concrete recommendation, its reasoning and evidence,
  assumptions or uncertainty, and any follow-up needed from the parent.

After the subagent returns, inspect material claims against the supplied
context, resolve unsupported or conflicting advice, and answer in the parent's
own voice. Do not remain in character across later turns. A follow-up
consultation is a new bounded handoff using the relevant prior context.

If no suitable host subagent, model, or enforced read-only profile is available,
say so and direct the user to `/shark-rider consult <agent> <question or
artifact>` for an inline consultation.

---

### Step 4 — Read-only by default

The subagent provides advice only. It does **not** mutate Shark state (no
`shark create`, `shark status set`, `shark delete`, etc.), edit files, or
advance workflow. If implementation or a mutation seems warranted, report the
recommendation; handle execution through the appropriate workflow only when
explicitly requested.

---

## Notes

- Each consultation is a bounded subagent request whose response is integrated
  into the current conversation; the parent remains responsible for the answer.
- To end a consultation and return to normal mode, the user can invoke any other
  `/shark-rider` verb or simply say "done" / "thanks".
