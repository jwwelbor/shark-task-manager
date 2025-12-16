#!/usr/bin/env python3
"""
Hook Name: artifact-watcher.py
Event: PostToolUse (Write)
Purpose: Detect workflow artifact creation and update workflow state

This hook fires after the Write tool creates a file. It:
1. Checks if the file matches an artifact naming pattern
2. Updates workflow state with the new artifact
3. Marks pending artifacts as created
4. Optionally auto-advances workflow if all outputs complete
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


def save_state(state):
    """Save updated workflow state to state.json."""
    project_root = Path.cwd()
    state_path = project_root / 'docs' / 'workflow' / 'state.json'

    try:
        with open(state_path, 'w') as f:
            json.dump(state, f, indent=2)
        return True
    except Exception as e:
        print(f"ERROR: Failed to save state.json: {e}", file=sys.stderr)
        return False


def update_artifact_status(state, artifact_name):
    """Mark artifact as created in pending_artifacts list."""
    updated = False

    for artifact in state.get('pending_artifacts', []):
        if artifact['artifact_name'] == artifact_name:
            artifact['status'] = 'created'
            artifact['created_at'] = datetime.now().isoformat()
            updated = True
            break

    return updated


def check_node_completion(state):
    """Check if all pending artifacts for current node are created."""
    if not state.get('current_workflow'):
        return False

    current_node = state['current_workflow'].get('current_node')
    if not current_node:
        return False

    # Get all pending artifacts for current node
    node_artifacts = [
        a for a in state.get('pending_artifacts', [])
        if a.get('expected_from') == current_node
    ]

    if not node_artifacts:
        return False

    # Check if all are created
    all_created = all(a.get('status') == 'created' for a in node_artifacts)

    return all_created


def main():
    """Main hook execution."""
    try:
        # Read event data from stdin
        event_data = json.load(sys.stdin)

        tool_name = event_data.get('tool_name')
        tool_result = event_data.get('result', {})

        # Only process Write tool events
        if tool_name != 'Write':
            return

        # Get the file path that was written
        file_path = tool_result.get('file_path')
        if not file_path:
            return

        file_path = Path(file_path)
        filename = file_path.name

        # Check if this is an artifact
        is_art, artifact_type = is_artifact(filename)
        if not is_art:
            # Not an artifact, ignore
            return

        print(f"[artifact-watcher] Detected artifact: {filename} (type: {artifact_type})", file=sys.stderr)

        # Load workflow state
        state = load_state()
        if not state:
            print("[artifact-watcher] No workflow state found, skipping state update", file=sys.stderr)
            return

        # Check if workflow is active
        workflow_status = state.get('current_workflow', {}).get('status')
        if workflow_status not in ['active', 'waiting', 'waiting_approval']:
            print(f"[artifact-watcher] Workflow status is '{workflow_status}', not tracking artifacts", file=sys.stderr)
            return

        # Update artifact status
        was_pending = update_artifact_status(state, filename)

        if was_pending:
            print(f"[artifact-watcher] Marked {filename} as created in pending artifacts", file=sys.stderr)
        else:
            print(f"[artifact-watcher] Artifact {filename} was not in pending list (unexpected artifact)", file=sys.stderr)

        # Update workflow metadata
        state['workflow_context']['updated_at'] = datetime.now().isoformat()
        state['metadata']['last_modified_by'] = 'artifact-watcher-hook'

        # Check if current node is now complete
        node_complete = check_node_completion(state)
        if node_complete:
            current_node = state['current_workflow']['current_node']
            print(f"[artifact-watcher] All artifacts created for node: {current_node}", file=sys.stderr)
            print(f"[artifact-watcher] Node is ready for transition (workflow-router will handle on Stop)", file=sys.stderr)

        # Save updated state
        if save_state(state):
            print(f"[artifact-watcher] State updated successfully", file=sys.stderr)
        else:
            print(f"[artifact-watcher] Failed to save state", file=sys.stderr)

    except json.JSONDecodeError as e:
        print(f"[artifact-watcher] ERROR: Failed to parse event data: {e}", file=sys.stderr)
    except Exception as e:
        print(f"[artifact-watcher] ERROR: Unexpected error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)


if __name__ == '__main__':
    main()
