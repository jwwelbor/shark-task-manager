"""Errors shared by Codex command, catalog, and app-server helpers."""


class CatalogError(RuntimeError):
    """The selected Codex CLI could not provide a complete model catalog."""
