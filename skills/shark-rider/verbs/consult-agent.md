# /shark-rider consult-agent — Specialist subagent consultation

Consult a Shark agent persona through a bounded, read-only host subagent. The
parent remains the lead advisor: it provides context, reviews the response, and
synthesizes advice in this conversation. This is a local consultation recipe,
not keyed Shark workflow dispatch.

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

### Step 0 — Parse args

Split the args after the verb:
- First token → `<agent>` candidate (or empty).
- Remaining tokens → `<referent>` (may be empty).

If there are **no args** or the only arg is `help`, print this help response and
**STOP** without resolving or adopting an agent persona:

```text
Consult a Shark specialist when a decision needs role-specific judgment. Use
`/shark-rider consult-agent <agent-name> <question, entity, or artifact>` to get
one-shot, read-only advice from a host subagent. Include the relevant context
and ask for a concrete recommendation. The parent reviews and synthesizes the
response. Use `/shark-rider consult <agent> <question or artifact>` to continue
the current conversation in the persona voice.

Choose the role that matches the question:
- `tech-director`: epic-level coordination, cross-feature dependencies, blockers,
  conflicting project state, or UAT direction.
- `architect`: system design, APIs, data models, technical feasibility, or
  compliance.
- `product-manager`: feature readiness, delivery planning, or feature-level
  coordination.
- `qa`: test planning, defect assessment, execution, or quality validation.
- `cx-designer`: journey mapping or experience coherence across touchpoints.
- `ux-designer`: user research, interface flows, wireframes, or prototypes.

Consult only the role needed. If specialist advice conflicts or an issue remains
unresolved at the epic level, consult `tech-director`; involve the owner if it
is still unclear or owner approval is required.

If these roles do not fit, run `shark agent list --json` to list all available
Shark agents.
```

Then **STOP**. Do not treat `help` as an agent name.

---

### Step 1 — Resolve the agent

```bash
shark agent list --json
```

Walk the list:

1. **Exact name match** (case-insensitive) — use it.
2. **Fuzzy match** — find agents where `<agent>` appears as a substring in the
   name OR description. If exactly one match, use it. If multiple matches, list
   them and ask the user to clarify, then **STOP**.
3. **No match** — print:
   ```
   No agent matching "<agent>" found.

   Available agents:
   ```
   followed by `shark agent list --all` output, then **STOP**.

---

### Step 2 — Load the persona

```bash
shark agent get <resolved-agent>
```

This returns the agent's definition (name, description, role, instructions,
skills, etc.). Read it fully and use it as the specialist role instructions in
the subagent handoff. Do not set the host subagent type to the Shark agent name.

---

### Step 3 — Resolve the referent

Determine what to discuss:

| Referent form | Resolution |
|---|---|
| Explicit file path (contains `/` or `.`) | `Read <path>` |
| Shark entity key (`E##`, `E##-F##`, `E##-F##-###`, `B###`, `CC-###`) | Fetch the entity (`shark get` equivalent) and read its content |
| `"this"` | Use the most recent artifact visible in this conversation (state which one you are using) |
| Empty or ambiguous prose | Ask the user: "What would you like to discuss with <agent>?" then **STOP** |
| Clear prose topic | Use as-is; no file read needed |

If the referent is a file or entity, read its content before proceeding.

---

### Step 4 — Prepare the consultation context

Print a brief announcement (one line):

```
Consulting <agent-name> about <referent-summary> with a specialist subagent...
```

Read context referenced by the persona only when relevant to the question. Get
Shark-bundled skills through `shark skill get <name> [relative-path]`; read
host-local Shark Rider sub-skills from their local paths. Do not read bundled
Shark persona or skill content directly from `shark-data/`. If required content
is unavailable, say so and continue only if the consultation remains useful.

### Step 5 — Select and launch a suitable subagent

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

Give the subagent a self-contained, bounded handoff containing:

- The requested question or decision and the resolved referent.
- The Shark persona definition and relevant retrieved skill/context content.
- Only the project facts and artifact content needed to advise.
- A read-only scope: inspect and recommend, but do not edit files, mutate Shark
  state, launch other agents, or perform implementation.
- The expected response: a concrete recommendation, its reasoning and evidence,
  assumptions or uncertainty, and any follow-up needed from the parent.

After the subagent returns, inspect material claims against the supplied
context, resolve unsupported or conflicting advice, and answer in the parent's
own voice. Do not remain in character across later turns. A follow-up
consultation is a new bounded handoff using the relevant prior context.

If no suitable host subagent or model is available, say so and direct the user
to `/shark-rider consult <agent> <question or artifact>` for an inline consultation.

---

### Step 6 — Read-only by default

The subagent provides advice only. It does **not** mutate Shark state (no
`shark create`, `shark status set`, `shark delete`, etc.), edit files, or
advance workflow. If implementation or a mutation seems warranted, report the
recommendation; handle execution through the appropriate workflow only when
explicitly requested.

---

## Notes

- Each consultation is a bounded subagent request whose response is integrated
  into the current conversation; the parent remains responsible for the answer.
- Fuzzy matching checks both `name` and `description` fields from the agent list.
- To end a consultation and return to normal mode, the user can invoke any other
  `/shark-rider` verb or simply say "done" / "thanks".
