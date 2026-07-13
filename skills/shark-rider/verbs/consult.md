# /shark-rider consult — In-persona agent consultation

Adopt a shark agent persona and consult it about any topic or artifact — inline,
in this conversation. No subagent is spawned; the persona speaks directly.

Usage:

```
/shark-rider consult                            # list available agents and stop
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

If **no args at all**: run `shark agent list` to display available agents, then
**STOP** with: "Use `/shark-rider consult <agent>` to start a consultation."

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
   followed by `shark agent list` output, then **STOP**.

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

Print a brief announcement (one line):

```
Consulting <agent-name> about <referent-summary>...
```

Then read any context files the persona's instructions reference (prompts,
skills listed in the agent definition) if they are relevant to the referent.
Use `Read` for local paths; do not run `shark skill get`.

Now **become the persona** for the remainder of the conversation:
- Speak in the persona's voice, applying its role, expertise, and instructions.
- Address the referent directly.
- Stay in character for follow-up turns until the user ends the consultation or
  invokes a different verb.

---

### Step 5 — Read-only by default

The persona does **not** mutate shark state (no `shark create`, `shark status
set`, `shark delete`, etc.) unless the user explicitly asks for it. If a
mutation seems warranted, the persona describes what it would do and asks
whether to proceed.

---

## Notes

- The consultation is **turn-by-turn in this session**, not a background agent.
- Fuzzy matching checks both `name` and `description` fields from the agent list.
- If the agent definition references skills, read them locally from the bundle
  path; never invoke `shark skill get`.
- To end a consultation and return to normal mode, the user can invoke any other
  `/shark-rider` verb or simply say "done" / "thanks".
