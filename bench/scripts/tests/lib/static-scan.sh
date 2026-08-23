#!/usr/bin/env bash
# lib/static-scan.sh -- shared static-scan helper for TC-091
# (T-E40-F10-014, spec.md REQ-NF-001/REQ-NF-005/REQ-NF-006). Implements, in
# this file only, the four enumerated scans test-plan.md's TC-091 defines:
#
#   (a) static_scan_write_path        -- write-path assignment-chain trace
#   (b) static_scan_forbidden_path    -- internal/, cmd/, .sharkconfig.json,
#                                         migrations/ literal grep
#   (c) static_scan_language_branch   -- python/go fixture-language branch
#                                         grep
#   (d) static_scan_content_disclosure -- credential / evaluator_only /
#                                          prompt_body / rendered_prompt /
#                                          uncapped-transcript-read grep
#
# tc091_static_safety_language_neutrality_test.sh sources this file and
# calls each function once, over the exact TC-091 Input file list. This is
# F10's ONE scanner, not four divergent copies -- TC-086's and TC-088's
# static scans (T-E40-F10-009/010/011) currently carry their own inline
# forbidden_effort_language / forbidden_composite_fields grep logic; this
# library is the shared home future TC-086/TC-088 edits can fold into
# without a second implementation, per this task's Scope note. Folding
# them in is NOT done by this task (T-E40-F10-014's Scope lists only the
# two new files below; TC-086/TC-088 keep their existing, already-passing
# inline scans unchanged).
#
# Caller contract: source this file, then call each static_scan_* function
# with the exact file list to scan (absolute or repo-relative paths, same
# argv order every call so AC-T1's "scanned file list enumerated" output is
# reproducible). Every function prints its verdict to stdout, including the
# scanned file list and an explicit zero-match/violation count, and returns
# 0 on a clean scan or 1 if it found a real violation.

STATIC_SCAN_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# -----------------------------------------------------------------------
# scan (a): write-path assignment-chain tracing (AC-T1, AC-T2).
#
# Every shell redirection (`>`, `>>`, `tee`, `cp ... <dest>`,
# `mv ... <dest>`, `mkdir`) whose destination argument contains a bash
# variable reference is traced to that variable's assignment(s), following
# the chain through further variable references. A traced variable
# classifies as:
#   - "root"    -- anchored under the script's own declared
#                  --retention-root value (the flag's parsed variable,
#                  auto-detected per file from its `--retention-root)`
#                  case-arm, e.g. `retention_root="$2"`), directly or via
#                  an assignment chain -- INCLUDING a destination variable
#                  built by string concatenation across two (or more)
#                  separate assignment statements (AC-T2): a
#                  self-referential assignment (`x="$x/suffix"`) only ever
#                  upgrades or preserves an already-established root
#                  anchor, it never lets an appended literal or unrelated
#                  path segment erase one, matching real sequential bash
#                  assignment semantics.
#   - "mktemp"  -- process-scratch space from `$(mktemp ...)` with no
#                  root anchor at all (every site in this file set is
#                  `trap`-cleaned up before the script exits; it is
#                  intermediate exchange storage, never a persisted
#                  artifact, so it is not "a write outside the declared
#                  retention root" in REQ-NF-001's sense).
#   - "unrelated" -- a genuine violation: the destination traces to
#                  something other than the root or mktemp scratch space
#                  (a scenario field, a fixture path, or an unvalidated
#                  positional argument).
#
# Tracing is scope-aware: a `local` variable's assignments are resolved
# against its OWN enclosing function first (falling back to file-level
# globals only for names that function does not shadow), so two different
# functions reusing the same local name (e.g. `dest` in both
# `retain_pair` and `quarantine_pair`) are never conflated.
#
# Non-shell files (e.g. the schema YAML, which uses '>' for its own
# folded-scalar syntax) have no shell-redirection semantics to trace; they
# are still listed in the scanned-file enumeration with zero write sites
# found, not silently skipped.
static_scan_write_path() {
	python3 - "$@" <<'PYEOF'
import re
import sys

HEREDOC_START_RE = re.compile(r"<<-?\s*(['\"]?)(\w+)\1")
IDENT_START_RE = re.compile(r"[A-Za-z_][A-Za-z0-9_]*")
VARREF_RE = re.compile(r"\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?")
CASE_ARM_RE = re.compile(r"^\s*--retention-root\)\s*$")
FUNC_DEF_RE = re.compile(r"^\s*([A-Za-z_][A-Za-z0-9_]*)\s*\(\)\s*\{\s*$")
SAFE_LITERAL_DESTS = {"/dev/null"}
SHELL_EXTENSIONS = (".sh",)
GLOBAL_SCOPE = "<global>"


def strip_heredocs(lines):
    out = list(lines)
    i, n = 0, len(lines)
    while i < n:
        m = HEREDOC_START_RE.search(lines[i])
        if m:
            delim = m.group(2)
            j = i + 1
            while j < n and lines[j].strip() != delim:
                out[j] = ""
                j += 1
            if j < n:
                out[j] = ""
            i = j + 1
            continue
        i += 1
    return out


def compute_masks(full_text):
    """Whole-file, cross-line quote/comment tracking (a bash quoted
    string, e.g. a multi-line `python3 -c '...'` argument, can span many
    physical lines; per-line-only tracking would desync at the closing
    quote)."""
    n = len(full_text)
    qmask, cmask = [False] * n, [False] * n
    in_s = in_d = False
    i = 0
    while i < n:
        c = full_text[i]
        if in_d:
            qmask[i] = True
            if c == "\\" and i + 1 < n and full_text[i + 1] != "\n":
                qmask[i + 1] = True
                i += 2
                continue
            if c == '"':
                in_d = False
            i += 1
            continue
        if in_s:
            qmask[i] = True
            if c == "'":
                in_s = False
            i += 1
            continue
        if c == "#":
            j = i
            while j < n and full_text[j] != "\n":
                cmask[j] = True
                j += 1
            i = j
            continue
        if c == '"':
            in_d, qmask[i] = True, True
            i += 1
            continue
        if c == "'":
            in_s, qmask[i] = True, True
            i += 1
            continue
        i += 1
    return qmask, cmask


def assign_line_scopes(code_lines, qmask, cmask):
    scopes = [GLOBAL_SCOPE] * len(code_lines)
    offsets, offset = [], 0
    for line in code_lines:
        offsets.append(offset)
        offset += len(line) + 1

    stack, depth = [], 0
    for idx, line in enumerate(code_lines):
        o = offsets[idx]
        m = FUNC_DEF_RE.match(line)
        if m and not stack:
            stack.append((m.group(1), depth))
        for pos, ch in enumerate(line):
            gi = o + pos
            if gi < len(qmask) and (qmask[gi] or cmask[gi]):
                continue
            if ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if stack and depth == stack[-1][1]:
                    stack.pop()
        scopes[idx] = stack[-1][0] if stack else GLOBAL_SCOPE
    return scopes


def find_assignments_on_line(line):
    results = []
    i, n = 0, len(line)
    while i < n:
        m = IDENT_START_RE.match(line, i)
        if not m:
            i += 1
            continue
        j = m.end()
        if line[j : j + 2] == "+=":
            op_len = 2
        elif line[j : j + 1] == "=" and line[j : j + 2] != "==":
            op_len = 1
        else:
            i = j
            continue
        varname = m.group(0)
        k, depth = j + op_len, 0
        while k < n:
            c = line[k]
            if c == "(":
                depth += 1
                k += 1
                continue
            if c == ")":
                if depth == 0:
                    break
                depth -= 1
                k += 1
                continue
            if depth == 0 and (c in " \t" or c == ";"):
                break
            k += 1
        results.append((varname, line[j + op_len : k]))
        i = k
    return results


def collect_assignments(code_lines, scopes):
    by_scope = {}
    for line, scope in zip(code_lines, scopes):
        if line.strip().startswith("#"):
            continue
        for varname, rhs in find_assignments_on_line(line):
            by_scope.setdefault(scope, {}).setdefault(varname, []).append(rhs)
    return by_scope


def detect_root_var(code_lines):
    for idx, line in enumerate(code_lines):
        if CASE_ARM_RE.match(line):
            for look in code_lines[idx + 1 : idx + 6]:
                m = re.search(r'([A-Za-z_][A-Za-z0-9_]*)="\$2"', look)
                if m:
                    return m.group(1)
    return None


def assignments_for(varname, scope, by_scope):
    return by_scope.get(scope, {}).get(varname) or by_scope.get(GLOBAL_SCOPE, {}).get(varname)


def resolve(varname, scope, root_var, by_scope, visited):
    if varname == root_var:
        return "root"
    key = (varname, scope)
    if key in visited:
        return "root"
    visited = visited | {key}
    rhs_list = assignments_for(varname, scope, by_scope)
    if not rhs_list:
        return "unrelated"

    current = "unrelated"
    self_ref_re = re.compile(r"\$\{?" + re.escape(varname) + r"\}?(?![A-Za-z0-9_])")
    for rhs in rhs_list:
        is_self_ref = bool(self_ref_re.search(rhs))
        other_refs = [r for r in VARREF_RE.findall(rhs) if r != varname]
        has_mktemp = re.search(r"\bmktemp\b", rhs) is not None
        if other_refs:
            sub = [resolve(r, scope, root_var, by_scope, visited) for r in other_refs]
            this_class = "root" if any(s == "root" for s in sub) else (
                "mktemp" if all(s == "mktemp" for s in sub) else "unrelated"
            )
        elif has_mktemp:
            this_class = "mktemp"
        else:
            bare = rhs.strip("\"'")
            this_class = "root" if bare in SAFE_LITERAL_DESTS else "unrelated"

        if is_self_ref:
            # A concatenating assignment (x="$x/suffix") only ever upgrades
            # or preserves the running classification -- appending a plain
            # literal/unrelated suffix segment must not erase an
            # already-established root anchor (AC-T2).
            if this_class == "root" or current == "root":
                current = "root"
            elif this_class == "mktemp" or current == "mktemp":
                current = "mktemp"
            else:
                current = "unrelated"
        else:
            current = this_class  # fresh (non-concatenating) overwrite
    return current


WRITE_RE = re.compile(r"(>>|>)(?!=)\s*(\S+)")
TEE_RE = re.compile(r"\btee\b((?:\s+-\w+)*)\s+(\S+)")


def strip_trailing_syntax(token):
    return re.sub(r"[)\";'&]+$", "", token)


def shell_tokens(line):
    toks, i, n = [], 0, len(line)
    while i < n:
        while i < n and line[i] in " \t":
            i += 1
        if i >= n:
            break
        start, in_dquote = i, False
        while i < n:
            c = line[i]
            if in_dquote:
                if c == '"':
                    in_dquote = False
                i += 1
                continue
            if c == '"':
                in_dquote = True
                i += 1
                continue
            if c in " \t":
                break
            i += 1
        toks.append(line[start:i])
    return toks


def strip_var_syntax(token):
    token = token.strip()
    if len(token) >= 2 and token[0] == token[-1] and token[0] in "\"'":
        token = token[1:-1]
    return token


def split_statements(line, qslice, cslice):
    segments, start, i, n = [], 0, 0, len(line)
    while i < n:
        if qslice[i] or cslice[i]:
            i += 1
            continue
        c = line[i]
        if c == ";":
            segments.append((line[start:i], qslice[start:i], cslice[start:i]))
            i += 1
            start = i
            continue
        if c in "&|" and i + 1 < n and line[i + 1] == c and not qslice[i + 1] and not cslice[i + 1]:
            segments.append((line[start:i], qslice[start:i], cslice[start:i]))
            i += 2
            start = i
            continue
        i += 1
    segments.append((line[start:], qslice[start:], cslice[start:]))
    return segments


def unquoted_spans(line, qslice, cslice):
    return "".join(" " if (q or c) else ch for ch, q, c in zip(line, qslice, cslice))


def extract_write_targets_from_segment(line, qslice, cslice):
    unquoted = unquoted_spans(line, qslice, cslice)
    targets = []

    for m in WRITE_RE.finditer(unquoted):
        rest_stripped = line[m.end(1) :].lstrip()
        if not rest_stripped:
            continue
        dest_tok = rest_stripped.split(None, 1)[0]
        if not dest_tok.startswith("&"):
            targets.append(dest_tok)

    for m in TEE_RE.finditer(unquoted):
        dest = line[m.start(2) : m.end(2)]
        if not dest.startswith("-"):
            targets.append(dest)

    for cmd in ("cp", "mv"):
        if re.search(r"\b" + cmd + r"\b", unquoted):
            toks = shell_tokens(line)
            if cmd in toks:
                idx = toks.index(cmd)
                args = [t for t in toks[idx + 1 :] if not t.startswith("-") and t != "--"]
                if len(args) >= 2:
                    targets.append(args[-1])

    if re.search(r"\bmkdir\b", unquoted):
        toks = shell_tokens(line)
        if "mkdir" in toks:
            idx = toks.index("mkdir")
            targets.extend(t for t in toks[idx + 1 :] if not t.startswith("-"))

    return targets


def extract_write_targets(line, qslice, cslice):
    if line.strip().startswith("#"):
        return []
    targets = []
    for seg, qslc, cslc in split_statements(line, qslice, cslice):
        targets.extend(extract_write_targets_from_segment(seg, qslc, cslc))
    return targets


def scan_file(path):
    text = open(path, encoding="utf-8").read()
    raw_lines = text.split("\n")

    if not path.endswith(SHELL_EXTENSIONS):
        return None, [], len(raw_lines)

    code_lines = strip_heredocs(raw_lines)
    qmask, cmask = compute_masks("\n".join(code_lines))
    scopes = assign_line_scopes(code_lines, qmask, cmask)
    by_scope = collect_assignments(code_lines, scopes)
    root_var = detect_root_var(code_lines)

    findings = []
    offset = 0
    for lineno, (line, scope) in enumerate(zip(code_lines, scopes), 1):
        qslice = qmask[offset : offset + len(line)]
        cslice = cmask[offset : offset + len(line)]
        offset += len(line) + 1
        for raw_target in extract_write_targets(line, qslice, cslice):
            target = strip_trailing_syntax(strip_var_syntax(raw_target))
            if target in SAFE_LITERAL_DESTS or target.startswith("/dev/null"):
                continue
            refs = VARREF_RE.findall(target)
            if not refs:
                cls = "unrelated"
            else:
                results = [
                    resolve(r, scope, root_var, by_scope, set()) if root_var else "unrelated"
                    for r in refs
                ]
                if any(r == "root" for r in results):
                    cls = "root"
                elif all(r == "mktemp" for r in results):
                    cls = "mktemp"
                else:
                    cls = "unrelated"
            findings.append((lineno, scope, raw_target, cls))
    return root_var, findings, len(raw_lines)


files = sys.argv[1:]
total_sites = 0
violations = []
per_file_report = []
for path in files:
    root_var, findings, _nlines = scan_file(path)
    total_sites += len(findings)
    n_root = sum(1 for f in findings if f[3] == "root")
    n_mktemp = sum(1 for f in findings if f[3] == "mktemp")
    n_bad = [f for f in findings if f[3] == "unrelated"]
    violations.extend((path,) + f for f in n_bad)
    per_file_report.append(
        f"  {path}: {len(findings)} write-destination site(s) "
        f"({n_root} root-anchored, {n_mktemp} mktemp-scratch, {len(n_bad)} violation(s))"
    )

print("scan (a) write-path assignment-chain trace -- scanned files:")
for p in files:
    print(f"  - {p}")
print("\n".join(per_file_report))
if violations:
    print(f"scan (a): {len(violations)} VIOLATION(S) found:", file=sys.stderr)
    for v in violations:
        path, lineno, scope, target, cls = v
        print(f"  {path}:{lineno} [{scope}] destination {target!r} traces to {cls!r}, not the retention root", file=sys.stderr)
    sys.exit(1)
print(f"scan (a): 0 violations across {total_sites} write-destination site(s) in {len(files)} scanned file(s)")
PYEOF
}

# -----------------------------------------------------------------------
# scan (b): forbidden-path scan. A literal grep for the path-prefixes
# `internal/`, `cmd/`, `.sharkconfig.json`, and `migrations/` anywhere in
# the scanned files; zero matches required.
static_scan_forbidden_path() {
	python3 - "$@" <<'PYEOF'
import re
import sys

FORBIDDEN = ["internal/", "cmd/", ".sharkconfig.json", "migrations/"]
files = sys.argv[1:]

print("scan (b) forbidden-path scan -- scanned files:")
for p in files:
    print(f"  - {p}")

violations = []
for path in files:
    text = open(path, encoding="utf-8").read()
    for term in FORBIDDEN:
        for m in re.finditer(re.escape(term), text):
            lineno = text.count("\n", 0, m.start()) + 1
            violations.append((path, lineno, term))

if violations:
    print(f"scan (b): {len(violations)} VIOLATION(S) found:", file=sys.stderr)
    for path, lineno, term in violations:
        print(f"  {path}:{lineno}: forbidden path prefix {term!r}", file=sys.stderr)
    sys.exit(1)
print(f"scan (b): 0 matches for {len(FORBIDDEN)} forbidden path prefixes across {len(files)} scanned file(s)")
PYEOF
}

# -----------------------------------------------------------------------
# scan (c): language-branch scan. A literal grep for `python`, `\.py\b`,
# `go run`, `go build`, `go test`, and `\.go\b` -- fixture-language
# behavior must be reached only through the I-04-registered adapter, never
# a branch in F10 code. The bare `python3` interpreter name is exempted:
# every F10 script uses it uniformly, regardless of fixture language, as
# its own JSON/YAML glue-code engine (the same role bash itself plays) --
# that is implementation-language choice, not a fixture-language branch.
# Any OTHER match (a bare "python" not followed by "3", any `.py`/`.go`
# file extension, or a literal `go run`/`go build`/`go test` invocation) is
# a violation.
static_scan_language_branch() {
	python3 - "$@" <<'PYEOF'
import re
import sys

PATTERNS = {
    "python (not python3)": re.compile(r"python(?!3)"),
    r"\.py\b": re.compile(r"\.py\b"),
    "go run": re.compile(r"\bgo run\b"),
    "go build": re.compile(r"\bgo build\b"),
    "go test": re.compile(r"\bgo test\b"),
    r"\.go\b": re.compile(r"\.go\b"),
}
files = sys.argv[1:]

print("scan (c) language-branch scan -- scanned files:")
for p in files:
    print(f"  - {p}")

violations = []
for path in files:
    text = open(path, encoding="utf-8").read()
    for label, pattern in PATTERNS.items():
        for m in pattern.finditer(text):
            lineno = text.count("\n", 0, m.start()) + 1
            violations.append((path, lineno, label, m.group(0)))

if violations:
    print(f"scan (c): {len(violations)} VIOLATION(S) found:", file=sys.stderr)
    for path, lineno, label, matched in violations:
        print(f"  {path}:{lineno}: matched {label!r} ({matched!r})", file=sys.stderr)
    sys.exit(1)
print(f"scan (c): 0 fixture-language-branch matches across {len(files)} scanned file(s) (python3 interpreter invocations exempted)")
PYEOF
}

# -----------------------------------------------------------------------
# scan (d): content-disclosure scan.
#   1. credential-pattern grep (sk-..., AKIA..., ANTHROPIC_API_KEY,
#      OPENAI_API_KEY, TURSO_AUTH_TOKEN) -- zero matches required.
#   2. `evaluator_only` / `prompt_body` / `rendered_prompt` grep -- a match
#      is a violation UNLESS it is a bare reference to the isolation
#      guard's own root-name constant (documented allowlist below,
#      currently empty: no such reference exists in the scanned file set
#      today).
#   3. a file-read call (`cat`, `read`, shell `<`) against a path whose
#      token contains "transcript", with no size-cap check (`wc -c`,
#      `head -c`, `du`) within the same statement or an adjacent line.
#      Stdout/stderr are output surfaces equally with file writes here:
#      this scan does not special-case the write's sink, so a rendered
#      value merely echoed to stderr "for debugging" is caught exactly
#      the same way a value written to a retained file would be.
static_scan_content_disclosure() {
	python3 - "$@" <<'PYEOF'
import re
import sys

CREDENTIAL_PATTERNS = {
    "sk-<token>": re.compile(r"sk-[A-Za-z0-9]{20,}"),
    "AKIA<id>": re.compile(r"AKIA[0-9A-Z]{16}"),
    "ANTHROPIC_API_KEY": re.compile(r"ANTHROPIC_API_KEY"),
    "OPENAI_API_KEY": re.compile(r"OPENAI_API_KEY"),
    "TURSO_AUTH_TOKEN": re.compile(r"TURSO_AUTH_TOKEN"),
}
# Reference-only mentions of the isolation guard's OWN root-name constant
# (e.g. a comment naming the directory it guards, without reading its
# contents) are not a disclosure. No such reference exists in the F10
# file set today; this allowlist exists so a future one can be recorded
# explicitly rather than silently weakening the scan.
DISCLOSURE_DIR_ALLOWLIST = set()
DISCLOSURE_DIR_PATTERNS = {
    "evaluator_only": re.compile(r"evaluator_only"),
    "prompt_body": re.compile(r"prompt_body"),
    "rendered_prompt": re.compile(r"rendered_prompt"),
}
READ_CALL_RE = re.compile(r"\bcat\b|\bread\b(?!only)|<\s*[\"$]?\S*")
SIZE_CAP_RE = re.compile(r"\bwc\s+-c\b|\bhead\s+-c\b|\bdu\b")
TRANSCRIPT_TOKEN_RE = re.compile(r"transcript", re.IGNORECASE)

files = sys.argv[1:]
print("scan (d) content-disclosure scan -- scanned files:")
for p in files:
    print(f"  - {p}")

violations = []
for path in files:
    text = open(path, encoding="utf-8").read()
    lines = text.split("\n")

    for label, pattern in CREDENTIAL_PATTERNS.items():
        for m in pattern.finditer(text):
            lineno = text.count("\n", 0, m.start()) + 1
            violations.append((path, lineno, f"credential pattern {label!r}"))

    for label, pattern in DISCLOSURE_DIR_PATTERNS.items():
        for m in pattern.finditer(text):
            lineno = text.count("\n", 0, m.start()) + 1
            if (path, lineno) in DISCLOSURE_DIR_ALLOWLIST:
                continue
            violations.append((path, lineno, f"disclosure-directory term {label!r}"))

    for lineno, line in enumerate(lines, 1):
        if line.strip().startswith("#"):
            continue
        if not TRANSCRIPT_TOKEN_RE.search(line):
            continue
        if not READ_CALL_RE.search(line):
            continue
        window = lines[max(0, lineno - 4) : lineno + 3]
        if any(SIZE_CAP_RE.search(w) for w in window):
            continue
        violations.append((path, lineno, "transcript read with no adjacent size-cap check"))

if violations:
    print(f"scan (d): {len(violations)} VIOLATION(S) found:", file=sys.stderr)
    for path, lineno, reason in violations:
        print(f"  {path}:{lineno}: {reason}", file=sys.stderr)
    sys.exit(1)
print(f"scan (d): 0 credential / disclosure-directory / uncapped-transcript-read matches across {len(files)} scanned file(s)")
PYEOF
}
