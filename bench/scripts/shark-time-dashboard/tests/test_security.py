"""Compatibility marker for dashboard coverage split by production seam.

The concrete tests live in the adjacent analysis, quality, render, and CLI
modules. Keeping this path avoids dropping the recovery artifact's original
test entry point while preventing unrelated concerns from sharing one large
module.
"""
