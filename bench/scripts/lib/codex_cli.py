"""Codex CLI command parsing shared by catalog discovery and dispatch."""

from __future__ import annotations


CODEX_GLOBAL_OPTIONS_WITH_VALUE = {
    "-a", "--ask-for-approval", "--add-dir", "--config", "--disable", "--enable",
    "-i", "--image", "--local-provider", "-m", "--model", "--remote",
    "--remote-auth-token-env", "-p", "--profile", "-s", "--sandbox", "-c",
    "-C", "--cd",
}
CODEX_EXEC_COMMANDS = {"exec", "e"}


def _validate_command(command: list[str]) -> None:
    if (
        not isinstance(command, list)
        or not command
        or any(not isinstance(part, str) or not part for part in command)
    ):
        raise ValueError("Codex CLI command must be a non-empty argv array")


def _next_option_index(command: list[str], index: int) -> int | None:
    """Return the next token index, or None when the token is positional."""
    argument = command[index]
    if argument in CODEX_GLOBAL_OPTIONS_WITH_VALUE:
        return index + 2
    if argument.startswith("--") and "=" in argument:
        return index + 1
    if argument.startswith("-"):
        return index + 1
    return None


def exec_subcommand_index(command: list[str]) -> int | None:
    """Return the route subcommand position while skipping global option values."""
    _validate_command(command)
    index = 1
    while index < len(command):
        argument = command[index]
        if argument == "--":
            return None
        if argument in CODEX_EXEC_COMMANDS:
            return index
        next_index = _next_option_index(command, index)
        if next_index is None:
            return None
        index = next_index
    return None


def _profile_at(command: list[str], index: int) -> tuple[bool, str | None]:
    argument = command[index]
    if argument in {"-p", "--profile"}:
        if index + 1 >= len(command):
            raise ValueError("Codex CLI profile option requires a name")
        return True, command[index + 1]
    if argument.startswith("--profile="):
        profile = argument.partition("=")[2]
        if not profile:
            raise ValueError("Codex CLI profile option requires a name")
        return True, profile
    if argument.startswith("-p") and len(argument) > 2:
        return True, argument[2:]
    return False, None


def _unique_profile(profiles: list[str]) -> str | None:
    unique_profiles = set(profiles)
    if len(unique_profiles) > 1:
        raise ValueError("Codex CLI command selects multiple profiles")
    return profiles[0] if profiles else None


def selected_profile(command: list[str]) -> str | None:
    """Return the single selected `--profile` value from a validated argv."""
    _validate_command(command)
    profiles = []
    index = 1
    while index < len(command):
        argument = command[index]
        if argument == "--":
            break
        is_profile, profile = _profile_at(command, index)
        if is_profile:
            assert profile is not None
            profiles.append(profile)
            index += 2 if argument in {"-p", "--profile"} else 1
            continue
        index = _next_option_index(command, index) or index + 1
    return _unique_profile(profiles)
