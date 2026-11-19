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

### Step 0 — Handle help

If there are **no args** or the only arg is `help`, print the help response by
composing the mode-specific opening below, the shared role-selection guide at
`skills/shark-rider/context/consult-role-guide.md` verbatim, and the
mode-specific closing below. **STOP** without resolving or adopting an agent
persona:

```text
Consult a Shark specialist when a decision needs role-specific judgment. Use
`/shark-rider consult <agent-name> <question, entity, or artifact>` to continue
the current conversation in the specialist's voice. Include the relevant context
and ask a clear question. The consultation is read-only by default.
```

After the shared guide, append this closing:

```text
For one-shot delegated advice, use
`/shark-rider consult-agent <agent-name> <question, entity, or artifact>`.
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

### Step 2 — Announce and adopt the persona

Print a brief announcement:

```text
Consulting <agent-name> about <referent-summary>...
```

Adopt the persona **inline** for the rest of the consultation:

- Respond in the persona's voice to the requested referent.
- Keep using that voice on follow-up turns until the user ends the consultation
  or invokes another verb.
- Do not spawn a subagent. Use `/shark-rider consult-agent` when the user asks
  for a separate, one-shot delegated opinion.

---

### Step 3 — Read-only by default

The persona does **not** mutate shark state (no `shark create`, `shark status
set`, `shark delete`, etc.) unless the user explicitly asks for it. If a
mutation seems warranted, describe what it would do and ask whether to proceed.

---

## Notes

- The consultation is **turn-by-turn in this session**. Use `consult-agent` for a separate, one-shot host subagent request.
- To end a consultation and return to normal mode, the user can invoke any other
  `/shark-rider` verb or simply say "done" / "thanks".
