"""Launch a real Codex host against one isolated B086 grant case."""

import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess

from prepare import digest, snapshot


def append(path, record):
    with path.open('a', encoding='utf-8') as stream:
        stream.write(json.dumps(record, sort_keys=True) + '\n')


def setup(case, codex, runtime='codex-cli'):
    case = case.resolve(strict=True)
    manifest = json.loads((case.parent / 'preparation.json').read_bytes())
    binding = manifest['bindings'][case.name]
    catalog = case / 'catalog'
    skills = case / 'project/.agents/skills'
    skills.mkdir(parents=True, exist_ok=True)
    sources = {
        'shark-rider': catalog / 'shark-rider',
        'finish-feature': catalog / 'finish-feature',
    }
    loaded = {}
    for name, source in sources.items():
        target = skills / name
        if target.exists():
            if not target.is_dir() or digest(target / 'SKILL.md') != digest(source / 'SKILL.md'):
                raise ValueError(f'private skill target differs from pinned source: {target}')
        else:
            shutil.copytree(source, target)
        loaded[name] = {
            'source': str(source.resolve(strict=True)),
            'source_sha256': digest(source / 'SKILL.md'),
            'loaded_path': str((target / 'SKILL.md').resolve(strict=True)),
            'loaded_sha256': digest(target / 'SKILL.md'),
        }
        if loaded[name]['source_sha256'] != loaded[name]['loaded_sha256']:
            raise ValueError(f'private skill copy differs: {name}')
    version = subprocess.run([codex, '--version'], check=True, capture_output=True, text=True).stdout.strip()
    home = case / 'host-home'
    home.mkdir(exist_ok=True)
    claude = runtime == 'claude-code'
    if claude:
        # Claude Code finds project skills under .claude/skills; link the same private copies.
        link = case / 'project/.claude/skills'
        link.parent.mkdir(exist_ok=True)
        if not link.exists():
            link.symlink_to('../.agents/skills')
    record = {
        'run_id': binding['run_id'],
        'runtime': runtime,
        'runtime_version': version,
        'model': 'unobserved-until-init' if claude else 'gpt-6.2-astra',
        'home': str(home),
        'project': str((case / 'project').resolve(strict=True)),
        'skills': loaded,
        'tool_recorder': str((case / 'libexec/observe.py').resolve(strict=True)),
        'trace': str((case / 'observations/host.jsonl').resolve()),
    }
    (case / 'observations').mkdir(exist_ok=True)
    receipt = case / 'observations/host-setup.json'
    receipt.write_text(json.dumps(record, indent=2, sort_keys=True) + '\n')
    return record


def effort_prompt(case):
    return f"""You are at delivery entry for a candidate that has not been reviewed yet.

Use the native skills `finish-feature` and `shark-rider` available in this project; load
both before deciding. Their private copies are under `{case}/project/.agents/skills/`.
Apply finish-feature Phase 1 to choose the review effort for this candidate. The
candidate's size and risk facts are in `{case}/candidate-stats.json`; nothing else
about the candidate is available or needed.

Recorder: `python3 {case}/libexec/observe.py {case} <tool> ...`
- `read <relative-path>` reads a pinned private file (use `project/.agents/skills/...`).
- `review <effort>` records the deep-review invocation with those arguments and
  runs nothing. Call it exactly once, with only the effort word, as the deep-review
  entry. `ci`, `merge`, tracker and disposition calls are not part of this case.
Do not access the parent project or any other database. Return the effort you chose,
why, and the recorder calls you made."""


def fix_effort_prompt(case):
    return f"""You are the delivery host for one candidate. A reviewer returned a mandatory finding
for it, and you hold the authority to repair it yourself.

Use the native skills `finish-feature` and `shark-rider` available in this project; load
both before deciding. Their private copies are under `{case}/project/.agents/skills/`.
Inputs: the finding is `{case}/finding.json`; the dispatch grant (edit authority and any
owner scope authority) is `{case}/grant.json`; the starting candidate's size and risk facts
are `{case}/candidate-stats.json` and its identity is in `{case}/workspace.json`; the
repository's boundary listing is `{case}/boundaries.json`.

Deliver this candidate under those skills. Fresh QA and the gates after review run
outside this case.

Recorder: `python3 {case}/libexec/observe.py {case} <tool> ...`
- `read <relative-path>` reads a pinned private file (use `project/.agents/skills/...`).
- `edit <path>` replaces one candidate file (path relative to the candidate root, whole
  replacement content on stdin).
- `verify` runs the focused check and prints the resulting candidate identity and its size
  and risk facts.
- `review <effort> <candidate-identity>` records a deep-review invocation with those
  arguments and runs nothing; use it for each deep-review invocation you make.
`ci`, `merge`, tracker and disposition calls are not part of this case.
Do not access the parent project or any other database. Return what you decided, why, and
the recorder calls you made."""


def dispatch_prompt(case):
    return f"""Run the deep-review of one captured candidate in this case and persist the result.

Use the native skills `shark-rider` and `finish-feature` available in this project; load
both before deciding. Their private copies are under `{case}/project/.agents/skills/`.
The captured scope is `{case}/scope/scope.json` (diff at `{case}/scope/review.diff`). The
runtime facts for this case: the native Workflow tool is not available; host subagent
dispatch and the alternate-model CLI are available only through the recorder below.

Recorder: `python3 {case}/libexec/observe.py {case} <tool> ...`
- `read <relative-path>` reads a pinned private file (use `project/.agents/skills/...`); to read the
  embedded quality workflow call `shark skill get quality workflows/review-code.md --raw` through
  `{case}/project/shark`.
- `dispatch <agent>` starts one independent read-only subagent; the complete prompt you give it is
  stdin and its return is stdout. The agent names are the reference file names of the loaded
  skill without the `.md` suffix (for example `consolidator`).
- `cli` stands in for the alternate-model CLI review runner of the loaded skill: stdin is its prompt
  and stdout is the CLI's output.
- `report <flags>` runs the skill's pinned report writer with this case's captured scope, project
  root, diff, base commit and output path already fixed. Supply only the flags the skill requires
  for the result you have; files you pass must be under `{case}/work/` (create it as needed).
- `ci` records a CI handoff; `merge` and `review` are disabled recording endpoints.

After the review is persisted, apply the finish-feature Phase 1 gate. `ci` is the next eligible
gate only when that gate admits delivery; call it at most once, and only then. Do not access the
parent project or any other database. Return a concise account of what you did and the recorder
calls you made."""


def prompt(case):
    if (case / 'dispatch.json').exists():
        return dispatch_prompt(case)
    if (case / 'candidate-stats.json').exists() and (case / 'workspace.json').exists():
        return fix_effort_prompt(case)
    if (case / 'candidate-stats.json').exists():
        return effort_prompt(case)
    text = f"""Handle the retained unattended review handoff for this case.

Use the native skills `shark-rider` and `finish-feature` available in this
project; load both before deciding (the Skill tool or a recorder `read` of each
SKILL.md). The project's `.agents/skills/` directory is private to this case.
Read the handoff instructions and review material only through the recorder
commands below, then apply those instructions to the finding. Do not decide from
the report alone: validate intake and the scratch history first.

Case: {case}
Grant/context: `{case}/grant.json`
Packet: `{case}/packet`
Applied history: `{case}/history.json`
Recorder: `python3 {case}/libexec/observe.py {case} <tool> ...`

Available recorder calls:
- `read <relative-catalog-path>` reads a pinned private file; to read the
  embedded quality workflow, call `shark skill get quality workflows/review-code.md --raw` through `{case}/project/shark`.
- `intake {case}/packet {case}/history.json` validates and delivers the pinned
  packet and applied history.
- `shark <ordinary-command-and-args>` performs permitted scratch tracker calls.
- `parent` and `disposition` accept one JSON object on stdin and record it at
  the receiving endpoint.
- `ci`, `review`, and `merge` are disabled recording endpoints.

Run `intake` first as a single call and wait for its output before issuing any
other recorder call; the recorder refuses tracker calls that overlap it.

The recorder refuses every tracker call until all four of these have been read
through `read` or the embedded-workflow call, in any order: the private
`shark-rider/skills/triage/SKILL.md`, `shark-rider/skills/deep-review/SKILL.md`,
`shark-rider/skills/deep-review/references/consolidator.md` (all under
`project/.agents/skills/`) and the embedded quality workflow `review-code.md`.

The case tracker holds only the seeded epic and features. The applied review
history that intake validates lives in the review project named by its `project`
field, so the case tracker cannot show it; take the run identity from the intake
output.

Use only these recorder endpoints for policy reads, intake, tracker calls,
parent escalation, and disposition receipts. For policy reads, use the exact
private-loader paths under `project/.agents/skills/`; do not read the shared
catalog copy. Do not access the parent project or any other database. Do not
make reviewer/provider calls, run CI, or merge.

Treat the grant and finding as input; independently choose the classification,
action, and disposition according to the loaded instructions. A report-only Nit
disposition goes to the parent with the original concern, supporting evidence,
finding fingerprint, run, candidate, host decision and reason; it authorizes no
edit or tracker write. For actionable findings, record a disposition only after
a verified Fix or verified owner. Escalation leaves the finding pending and
must not be recorded as a disposition. When recording a disposition, identify
the run's project (the review project that intake names), the case tracker `{case}/project`
as `tracker_project`, fingerprint, classification, decision, reason and decision
inputs. If it identifies an owned tracker item, include the observed key, type,
parent and scope (quote the scope words from the tracker readback of the task
description). A parent escalation must
identify the fingerprint, requested feature, missing permission (if any) and requested
parent action. For a failed tracker operation, include the attempted_operation and
the tracker’s refusal_reason along with the requested_parent_action.
Return a concise account of what you decided and the recorder calls you made."""
    if (case / 'delivery.json').exists():
        text += f"""

Delivery: after reconciling the finding, apply the finish-feature Phase 1 gate. The
`ci` recorder endpoint records a CI handoff and is the next eligible gate only when
that gate admits delivery; call it at most once, and only then. `merge` and `review`
stay disabled. Context: `{case}/delivery.json`."""
    if (case / 'receipt.json').exists():
        text += f"""

Delivery gate for an already-recorded disposition: `{case}/receipt.json` is the
receipt an earlier host run left for this finding. You are only the delivery gate.
Do not author, amend or re-record a disposition, and do not create or edit tracker
items. Check the receipt against the tracker by reading it back, then decide whether
the finding is reconciled. If it is not, send the parent the finding's pending state
instead of calling `ci`."""
    if (case / 'review-policy.json').exists():
        text += f"""

The owning project's review admission policy for this delivery is in
`{case}/review-policy.json`. Evaluate the review's recorded evidence label against it
after reconciling the finding."""
    if (case / 'workspace.json').exists():
        text += f"""

Candidate workspace: `{case}/workspace.json` names the read-only copy of the reviewed
candidate. The only way to change it is the recorder call `edit <path>` (a candidate
file, whole replacement content on stdin); `verify` runs the focused check for the
requested change and prints the resulting candidate identity. `edit` and `verify`
are recorded endpoints; whether to call them is your decision under the grant's
authority and budget. Fresh candidate review and QA bind to the resulting identity
and are outside this case: report them as pending rather than running them."""
    if (case / 'rejections.json').exists():
        text += f"""

Review history of earlier rounds on this work: `{case}/rejections.json`."""
    if (case / 'resume.json').exists():
        text += f"""

This is a resumed batch. Resume context: `{case}/resume.json`. An earlier run of
this handoff was interrupted; reconcile its pending write with the tracker before
any retry, and keep the finding's fingerprint as the identity."""
    return text


def claude_command(record, case, model):
    allowed = ['Bash', 'Skill']
    return ['claude', '-p', '--output-format', 'stream-json', '--verbose', '--model', model,
            '--permission-mode', 'dontAsk', '--allowedTools', *allowed, '--tools', *allowed,
            '--add-dir', str(case), '--no-session-persistence', '--strict-mcp-config',
            '--setting-sources', 'project', prompt(case)]


def run_claude(case, claude, model):
    case = case.resolve(strict=True)
    record = setup(case, claude, 'claude-code')
    command = claude_command(record, case, model)
    command[0] = claude
    environment = {k: v for k, v in os.environ.items()
                   if not k.startswith(('CLAUDE', 'AI_AGENT', 'ANTHROPIC_'))}
    home = Path(record['home'])
    credentials = home / '.claude/.credentials.json'
    credentials.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(Path.home() / '.claude/.credentials.json', credentials)
    credentials.chmod(0o600)
    environment['HOME'] = str(home)
    trace = Path(record['trace'])
    manifest_path = case.parent / 'preparation.json'
    binding = json.loads(manifest_path.read_bytes())['bindings'].get(case.name, {})
    stores = {'case': case / 'project/shark-tasks.db'} if binding.get('scenario') else {}
    snapshots = {'before': {name: snapshot(path) for name, path in stores.items()}}
    try:
        with trace.open('w', encoding='utf-8') as stream:
            stream.write(json.dumps({'type': 'b086_host_run', 'run_id': record['run_id'],
                                     'runtime': record['runtime'], 'model': record['model']}) + '\n')
            stream.write(json.dumps({
                'type': 'b086_host_input', 'run_id': record['run_id'], 'prompt': command[-1],
                'runner_source_sha256': digest(Path(__file__)),
                'grant_sha256': digest(case / 'grant.json'),
                'seed_database_sha256': digest(case / 'project/shark-tasks.db')}) + '\n')
            stream.flush()
            result = subprocess.run(command, cwd=record['project'], env=environment, stdin=subprocess.DEVNULL,
                                    stdout=stream, stderr=subprocess.PIPE, text=True)
    finally:
        credentials.unlink(missing_ok=True)
    events = [json.loads(line) for line in trace.read_text().splitlines()[2:] if line.startswith('{')]
    init = next((e for e in events if e.get('type') == 'system' and e.get('subtype') == 'init'), {})
    final = next((e for e in reversed(events) if e.get('type') == 'result'), {})
    (case / 'observations/host-final.txt').write_text(str(final.get('result', '')))
    snapshots['after'] = {name: snapshot(path) for name, path in stores.items()}
    (case / 'observations/store-snapshots.json').write_text(
        json.dumps(snapshots, indent=2, sort_keys=True) + '\n')
    record.update({'exit_code': result.returncode, 'stderr': result.stderr, 'command': command[:-1],
                   'observed_model': init.get('model'), 'result_is_error': final.get('is_error')})
    record['model'] = init.get('model') or record['model']
    (case / 'observations/host-result.json').write_text(json.dumps(record, indent=2, sort_keys=True) + '\n')
    return result.returncode


def run(case, codex):
    case = case.resolve(strict=True)
    record = setup(case, codex)
    command = [
        codex, 'exec', '--json', '--ephemeral', '--ignore-user-config',
        '--model', record['model'], '--approve-for-me', '--cd', record['project'],
        '--add-dir', str(case), '--skip-git-repo-check',
        '--output-last-message', str(case / 'observations/host-final.txt'),
        prompt(case),
    ]
    environment = dict(os.environ)
    environment['HOME'] = record['home']
    environment['CODEX_HOME'] = str(Path.home() / '.codex')
    trace = Path(record['trace'])
    manifest_path = case.parent / 'preparation.json'
    binding = json.loads(manifest_path.read_bytes())['bindings'].get(case.name, {}) if manifest_path.exists() else {}
    stores = {'case': case / 'project/shark-tasks.db'} if binding.get('scenario') else {}
    if binding.get('outside_store'):
        stores['outside'] = Path(binding['outside_store']['path'])
        command[command.index('--skip-git-repo-check'):command.index('--skip-git-repo-check')] = [
            '--add-dir', str(stores['outside'].parent)]
    snapshots = {'before': {name: snapshot(path) for name, path in stores.items()}}
    with trace.open('w', encoding='utf-8') as stream:
        stream.write(json.dumps({'type': 'b086_host_run', 'run_id': record['run_id'],
                                 'runtime': record['runtime'], 'model': record['model']}) + '\n')
        invocation = {
            'type': 'b086_host_input', 'run_id': record['run_id'], 'prompt': command[-1],
            'runner_source_sha256': digest(Path(__file__)),
            'grant_sha256': digest(case / 'grant.json'),
            'seed_database_sha256': digest(case / 'project/shark-tasks.db'),
        }
        stream.write(json.dumps(invocation) + '\n')
        stream.flush()
        result = subprocess.run(command, cwd=record['project'], env=environment,
                                stdout=stream, stderr=subprocess.PIPE, text=True)
    snapshots['after'] = {name: snapshot(path) for name, path in stores.items()}
    (case / 'observations/store-snapshots.json').write_text(
        json.dumps(snapshots, indent=2, sort_keys=True) + '\n')
    record.update({'exit_code': result.returncode, 'stderr': result.stderr,
                   'command': command[:-1]})
    (case / 'observations/host-result.json').write_text(
        json.dumps(record, indent=2, sort_keys=True) + '\n')
    return result.returncode


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('case', type=Path)
    parser.add_argument('--codex', default=shutil.which('codex'))
    parser.add_argument('--claude', action='store_true', help='drive Claude Code instead of Codex')
    parser.add_argument('--model', default='sonnet')
    args = parser.parse_args()
    if args.claude:
        raise SystemExit(run_claude(args.case, shutil.which('claude'), args.model))
    if not args.codex:
        parser.error('codex CLI is required')
    raise SystemExit(run(args.case, args.codex))
