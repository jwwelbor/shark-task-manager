---
feature_key: E38-F13-distribute-and-install-shark-rider-from-the-shark
epic_key: E38
title: Distribute and install Shark Rider from the Shark CLI
description: Package a version-matched Shark Rider distribution with Shark and add explicit administrative commands under 'shark admin rider' for safe install, status, and uninstall operations at user or project scope. Support Claude Code and Codex first; keep Rider architecturally separate from project shark-data, and gate Copilot or other host claims on validated adapter conformance in E38-F10. Preserve idempotent updates, owned-file cleanup, host-specific destinations, and accurate onboarding documentation.
---

# Distribute and install Shark Rider from the Shark CLI

**Feature Key**: E38-F13-distribute-and-install-shark-rider-from-the-shark

## Triage breadcrumb

Shark currently installs the executable without installing the host-local
`skills/shark-rider/` tree. Release-binary users therefore do not receive the
skill that provides `/shark-rider` requests, host-side orchestration, and local
Rider workflows. The README can only offer source-checkout symlinks until Shark
provides a supported installation path.

Package a version-matched Rider distribution with Shark while keeping it
architecturally separate from the project `shark-data` content bundle. Add the
infrequent lifecycle operations under `shark admin rider`, including explicit
install, status, and uninstall operations with user and project scopes.
Installation must be safe to repeat, update only Shark-owned files, preserve
unrelated host configuration, and report the installed Rider version.

Support Claude Code and Codex first. E38-F09 provides the completed
provider-neutral coordination and host-adapter contract. E38-F10 retains
ownership of GitHub Copilot, Google Antigravity, and cross-provider adapter
conformance; discovering the skill directory alone must not be presented as
verified host support.

## Agreed command surface

Keep Rider administration below `shark admin` so infrequent setup commands do
not expand Shark's top-level command list:

```bash
shark admin rider install --host claude --scope user
shark admin rider install --host codex --scope user
shark admin rider install --host codex --scope project
shark admin rider status
shark admin rider uninstall --host codex --scope user
```

Do not add a top-level `shark setup` command. Do not add a separate Rider
upgrade command: `install` must install an absent Rider and update an outdated
Shark-managed installation. `shark admin init` may detect a missing Rider and
print the appropriate install command, but it must not change an agent host's
configuration automatically.

The commands should support `--dry-run` for a change preview and `--json` for
automation. Host and scope must be explicit in non-interactive use. Interactive
host detection may recommend a command, but it must not silently install Rider
into every detected host.

## Agreed distribution boundaries

- Ship canonical, version-matched Rider assets with the Shark release so binary
  users do not need a source checkout or network access after installing Shark.
- Keep Rider separate from the project `shark-data` bundle. `shark-data` supplies
  workflow-time prompts, roles, skills, and templates; Rider integrates those
  capabilities with an agent host.
- Install release assets as Shark-managed files. Reserve source-checkout
  symlinks for contributors who want live skill updates.
- Write files atomically, preserve unrelated configuration, record the installed
  Rider and Shark versions, and make repeated installation safe.
- Uninstall only files recorded as Shark-owned. Never remove or replace an
  unrelated skill, instruction file, hook, or host setting.

## Agreed host support

Start with these validated destinations:

| Host | User scope | Project scope |
| --- | --- | --- |
| Claude Code | `~/.claude/skills/shark-rider/` | `.claude/skills/shark-rider/` |
| Codex | `~/.agents/skills/shark-rider/` | `.agents/skills/shark-rider/` |

Use a shared, provider-neutral Rider core with small host-specific adapters or
generated variants. Remove Claude-only assumptions such as `$CLAUDE_SID`,
Claude `Agent` terminology, and hard-coded installation paths from shared
instructions before presenting the same package as portable.

Treat GitHub Copilot and other Agent Skills hosts as discoverable but
unsupported until E38-F10 captures their orchestration capabilities and passes
adapter conformance. Do not infer runtime support from a compatible skill
directory alone.

## Installer precedent

Reuse Graphify's useful installer patterns: package host-specific assets,
support user and project scopes, record versions, stage atomic updates, and
provide scoped uninstall. Do not copy its host destination map without current
verification; Shark's Codex target is the shared `.agents/skills` location.

After these commands ship, replace the README's temporary source-symlink steps
with `shark admin rider install` examples. Keep the manual path documented only
for contributors.

Observed surfaces include `Makefile`, `internal/sharkdata/embed.go`,
`skills/shark-rider/`, and the installation section in `README.md`.
