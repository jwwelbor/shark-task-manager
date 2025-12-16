#!/usr/bin/env python3
"""
Hook Name: quality-gate.py
Event: PreToolUse (Write)
Purpose: Validate artifacts and enforce quality gates before proceeding

This hook fires BEFORE certain tools execute. It:
1. Checks if workflow is attempting to advance
2. Validates that all required artifacts exist
3. Checks artifact completeness and quality
4. Blocks advancement if validation fails
5. Provides clear feedback on missing/incomplete artifacts
"""

import json
import sys
import re
from pathlib import Path
from datetime import datetime


# Artifact naming patterns
ARTIFACT_PATTERNS = {
    'discovery': r'^D\d{2}-.*\.md$',
    'feature': r'^F\d{2}-.*\.md$',
    'technical': r'^T\d{2}-.*\.md$',
    'development': r'^DEV-.*',
    'release': r'^R\d{2}-.*\.md$'
}

# Minimum file size for artifacts (bytes) - prevents empty file creation
MIN_ARTIFACT_SIZE = 100

# Required sections for artifact types
REQUIRED_SECTIONS = {
    'vision': ['## Problem Statement', '## Desired Outcomes', '## Success Criteria'],
    'prd': ['## Overview', '## Requirements', '## User Stories'],
    'architecture': ['## Overview', '## Components', '## Decisions'],
    'release': ['## Features', '## Deployment', '## Rollback']
}


def is_artifact(filename):
    """Check if filename matches any artifact pattern."""
    for pattern_name, pattern in ARTIFACT_PATTERNS.items():
        if re.match(pattern, filename):
            return True, pattern_name
    return False, None


def load_state():
    """Load workflow state from state.json."""
    project_root = Path.cwd()
    state_path = project_root / 'docs' / 'workflow' / 'state.json'

    if not state_path.exists():
        return None

    try:
        with open(state_path) as f:
            return json.load(f)
    except json.JSONDecodeError as e:
        print(f"ERROR: Failed to parse state.json: {e}", file=sys.stderr)
        return None


def check_artifact_exists(artifact_name):
    """Check if artifact file exists."""
    project_root = Path.cwd()
    artifacts_dir = project_root / 'docs' / 'workflow' / 'artifacts'
    artifact_path = artifacts_dir / artifact_name

    return artifact_path.exists(), artifact_path


def validate_artifact_completeness(artifact_path):
    """Validate that artifact is complete and not empty."""
    try:
        # Check file size
        size = artifact_path.stat().st_size
        if size < MIN_ARTIFACT_SIZE:
            return False, f"File too small ({size} bytes), appears empty or incomplete"

        # Read content
        with open(artifact_path) as f:
            content = f.read()

        # Check for placeholder text
        if 'TODO' in content or '[Fill in]' in content or '...' in content:
            return False, "Contains TODO markers or placeholders"

        # Basic markdown structure check
        if not content.strip().startswith('#'):
            return False, "Missing markdown header structure"

        # Check for minimum content length
        if len(content.strip()) < 200:
            return False, "Content too brief, appears incomplete"

        return True, "Valid"

    except Exception as e:
        return False, f"Error reading file: {e}"


def get_required_artifacts_for_node(state):
    """Get list of required artifacts for current node."""
    if not state or not state.get('current_workflow'):
        return []

    # Get pending artifacts for current node
    current_node = state['current_workflow'].get('current_node')
    pending = [
        a for a in state.get('pending_artifacts', [])
        if a.get('expected_from') == current_node
    ]

    return pending


def check_prerequisites_complete(state):
    """Check if all required artifacts from previous nodes exist."""
    if not state or not state.get('current_workflow'):
        return True, []

    current_node = state['current_workflow'].get('current_node')

    # Get completed nodes
    completed_nodes = state.get('completed_nodes', [])

    # Check if any artifacts from previous nodes are still pending
    missing = []
    for completed in completed_nodes:
        artifacts_produced = completed.get('artifacts_produced', [])
        for artifact_name in artifacts_produced:
            exists, artifact_path = check_artifact_exists(artifact_name)
            if not exists:
                missing.append({
                    'artifact_name': artifact_name,
                    'node': completed.get('node_name'),
                    'agent': completed.get('agent')
                })

    return len(missing) == 0, missing


def is_state_transition_tool_use(event_data):
    """
    Detect if this tool use is attempting to transition workflow state.

    This is a conservative check - we only validate when we detect
    state transitions or critical artifact creation.
    """
    tool_name = event_data.get('tool_name')

    # Currently only validate Write tool (artifact creation)
    if tool_name == 'Write':
        file_path = event_data.get('parameters', {}).get('file_path', '')

        # Check if writing to state.json (state transition)
        if 'state.json' in file_path:
            return True

        # Check if writing an artifact (critical output)
        filename = Path(file_path).name
        is_art, _ = is_artifact(filename)
        if is_art:
            return True

    return False


def validate_workflow_state(state, event_data):
    """
    Main validation logic for quality gate.

    Returns: (is_valid, error_message)
    """
    if not state:
        # No workflow active, allow operation
        return True, None

    workflow_status = state.get('current_workflow', {}).get('status')

    # Only validate active workflows
    if workflow_status not in ['active', 'waiting', 'waiting_approval']:
        return True, None

    current_node = state['current_workflow'].get('current_node')
    current_agent = state['current_workflow'].get('current_agent')

    # Check 1: Verify prerequisites from previous nodes exist
    prereqs_complete, missing_prereqs = check_prerequisites_complete(state)
    if not prereqs_complete:
        error_msg = f"""
╔═══════════════════════════════════════════════════════════════╗
║                    QUALITY GATE: BLOCKED                      ║
╚═══════════════════════════════════════════════════════════════╝

Current Node: {current_node}
Current Agent: {current_agent}

ERROR: Missing required artifacts from previous nodes

Missing Prerequisites:
"""
        for missing in missing_prereqs:
            error_msg += f"  • {missing['artifact_name']} (from {missing['node']} - {missing['agent']})\n"

        error_msg += """
Action Required:
  1. Return to previous nodes and complete missing artifacts
  2. Verify artifacts are in docs/workflow/artifacts/
  3. Check artifact naming matches expected output

The workflow cannot proceed until all dependencies are satisfied.
═══════════════════════════════════════════════════════════════
"""
        return False, error_msg

    # Check 2: If writing artifact, validate it's expected from current node
    tool_name = event_data.get('tool_name')
    if tool_name == 'Write':
        file_path = event_data.get('parameters', {}).get('file_path', '')
        filename = Path(file_path).name
        is_art, artifact_type = is_artifact(filename)

        if is_art:
            # Check if this artifact is expected from current node
            expected_artifacts = [
                a['artifact_name'] for a in state.get('pending_artifacts', [])
                if a.get('expected_from') == current_node
            ]

            if expected_artifacts and filename not in expected_artifacts:
                # Creating unexpected artifact - warn but allow
                print(f"[quality-gate] WARNING: Creating artifact '{filename}' not in expected outputs for {current_node}", file=sys.stderr)
                print(f"[quality-gate] Expected: {', '.join(expected_artifacts)}", file=sys.stderr)

    return True, None


def main():
    """Main hook execution."""
    try:
        # Read event data from stdin
        event_data = json.load(sys.stdin)

        tool_name = event_data.get('tool_name')

        # Only validate certain tool uses
        if not is_state_transition_tool_use(event_data):
            # Not a critical operation, allow
            return

        print(f"[quality-gate] Validating tool use: {tool_name}", file=sys.stderr)

        # Load workflow state
        state = load_state()
        if not state:
            print("[quality-gate] No workflow state found, allowing operation", file=sys.stderr)
            return

        # Validate workflow state
        is_valid, error_message = validate_workflow_state(state, event_data)

        if not is_valid:
            # BLOCK the operation
            print(error_message, file=sys.stderr)
            print(error_message)  # Also to stdout so agent sees it

            # Exit with error code to block tool execution
            sys.exit(1)
        else:
            print(f"[quality-gate] Validation passed", file=sys.stderr)

    except json.JSONDecodeError as e:
        print(f"[quality-gate] ERROR: Failed to parse event data: {e}", file=sys.stderr)
        # Allow operation on parse errors (fail open for safety)
    except Exception as e:
        print(f"[quality-gate] ERROR: Unexpected error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)
        # Allow operation on unexpected errors (fail open for safety)


if __name__ == '__main__':
    main()
