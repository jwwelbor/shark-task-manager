# /shark-rider consult — In-persona agent consultation

Adopt a shark agent persona and consult it about any topic or artifact — inline,
in this conversation. No subagent is spawned; the persona speaks directly.

Usage:

```
/shark-rider consult                            # show role-selection help and stop
/shark-rider consult <agent>                    # consult agent; ask what to discuss
/shark-rider consult <agent> <referent>         # consult agent about a specific thing
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
`/shark-rider consult <agent-name> <question, entity, or artifact>` to continue
the current conversation in the specialist's voice. Include the relevant context
and ask a clear question. The consultation is read-only by default.

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
Shark agents. For one-shot delegated advice, use
`/shark-rider consult-agent <agent-name> <question, entity, or artifact>`.
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
skills, etc.). Read it fully — this is the voice you will adopt.

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

### Step 4 — Announce and adopt the persona

Print a brief announcement:

```text
Consulting <agent-name> about <referent-summary>...
```

Read context referenced by the persona only when relevant. Get Shark-bundled
skills through `shark skill get <name> [relative-path]`; read host-local
Shark Rider sub-skills from their local paths. Do not read bundled Shark persona
or skill content directly from `shark-data/`.

Adopt the persona **inline** for the rest of the consultation:

- Respond in the persona's voice to the requested referent.
- Keep using that voice on follow-up turns until the user ends the consultation
  or invokes another verb.
- Do not spawn a subagent. Use `/shark-rider consult-agent` when the user asks
  for a separate, one-shot delegated opinion.

---

### Step 5 — Read-only by default

The persona does **not** mutate shark state (no `shark create`, `shark status
set`, `shark delete`, etc.) unless the user explicitly asks for it. If a
mutation seems warranted, describe what it would do and ask whether to proceed.

---

## Notes

- The consultation is **turn-by-turn in this session**. Use `consult-agent` for a separate, one-shot host subagent request.
- Fuzzy matching checks both `name` and `description` fields from the agent list.
- Get Shark-bundled skills through `shark skill get`; read host-local Shark
  Rider sub-skills from their local paths.
- To end a consultation and return to normal mode, the user can invoke any other
  `/shark-rider` verb or simply say "done" / "thanks".
