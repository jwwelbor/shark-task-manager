# Shared Consultation Request Resolution

Both `/shark-rider consult` and `/shark-rider consult-agent` use this procedure
before applying their mode-specific behavior. The caller handles bare help
requests first and passes the remaining arguments here.

## 1. Parse the request

- First token → `<agent>` candidate.
- Remaining tokens → `<referent>` (may be empty).

## 2. Resolve the agent

Run `shark agent list --json`.

- If the list is empty, say no Shark agents are installed, suggest
  `shark admin install-shark-data`, and **STOP**.
- Prefer an exact case-insensitive name match.
- Otherwise find fuzzy matches where `<agent>` appears in an agent name or
  description. Use a unique match. If multiple match, list them and ask the user
  to clarify, then **STOP**.
- If no agent matches, say `No agent matching "<agent>" found.`, print
  `shark agent list --all`, and **STOP**.

## 3. Load the persona

Run `shark agent get <resolved-agent>` and read the returned definition fully.
If the command fails or returns no persona definition, say:
`Agent "<resolved-agent>" was listed but its content could not be loaded.`
Print `shark agent list --all` and **STOP** before the caller adopts or dispatches
the persona.

## 4. Resolve the referent

| Referent form | Resolution |
|---|---|
| Path-shaped file token (contains `/` or a Windows-style path separator, or ends in a known file suffix such as `.md`, `.go`, `.json`, `.yaml`, `.yml`, `.toml`, or `.txt`) | Remove sentence-final punctuation from the candidate token only. Resolve relative paths against the active project workspace root. A period elsewhere in ordinary prose is not enough to identify a path. If the sentence includes both a path and a question, read the path and preserve the question. |
| Shark entity key accepted by `shark get` | Fetch the key with `shark get <key>` and read its content; preserve any remaining prose as the question. |
| `"this"` or a clear reference to the current artifact | Use the most recent unambiguous artifact visible in the conversation and state which file it is. |
| Empty or ambiguous prose | Ask: `What would you like to discuss with <agent>?` and **STOP**. |
| Clear prose topic | Use as-is; no file read is needed. |

Before reading a user-requested path referent, canonicalize both the active
project workspace root and the existing target after following symlinks, then
verify containment with a component-aware relative-path check (not a
string-prefix comparison). Absolute paths are allowed only when their canonical
target remains inside the workspace root. If the target is outside that root or
cannot be safely canonicalized, do not read it; ask the user to paste the
content or provide an in-workspace copy. This boundary applies to user-referred
files only; documented host-local Shark Rider skill/context reads in Section 5
remain available through their specified local paths.

If a requested file, entity, or in-context artifact cannot be identified
unambiguously, do not guess; ask the user and **STOP**. For a clear prose topic,
pass it through unchanged. Return the loaded persona and resolved context to the
caller without adopting the persona or launching a subagent; the caller owns
that mode-specific step.

## 5. Load relevant context

Read persona-referenced context only when it is relevant. Get Shark-bundled
skills through `shark skill get <name> [relative-path]`; read host-local Shark
Rider sub-skills from their local paths. Do not read bundled Shark persona or
skill content directly from `shark-data/`. If required content is unavailable,
report it as unavailable and continue only if the consultation remains useful.
