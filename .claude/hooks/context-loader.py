#!/usr/bin/env python3
"""
Hook Name: context-loader.py
Event: SessionStart
Purpose: Load workflow context when agent session begins

This hook fires when an agent session starts. It:
1. Loads current workflow state
2. Identifies if agent should be aware of workflow context
3. Provides relevant artifacts and history
4. Informs agent of their current position in workflow
"""

import json
import sys
from pathlib import Path
from datetime import datetime


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


def get_artifact_paths(artifact_names):
    """Get full paths to artifact files."""
    project_root = Path.cwd()
    artifacts_dir = project_root / 'docs' / 'workflow' / 'artifacts'

    paths = []
    for name in artifact_names:
        artifact_path = artifacts_dir / name
        if artifact_path.exists():
            paths.append(str(artifact_path))

    return paths


def get_current_node_inputs(state):
    """Identify input artifacts for current node."""
    # This would typically come from the CSV definition
    # For now, we'll look for recently created artifacts
    completed_nodes = state.get('completed_nodes', [])

    if not completed_nodes:
        return []

    # Get artifacts from the most recent completed node
    last_node = completed_nodes[-1]
    return last_node.get('artifacts_produced', [])


def generate_context_message(state):
    """Generate context message for agent."""
    if not state:
        return None

    workflow_status = state.get('current_workflow', {}).get('status')

    # Only provide context if workflow is active
    if workflow_status not in ['active', 'waiting', 'waiting_approval', 'waiting_subgraph']:
        return None

    current_workflow = state.get('current_workflow', {})
    graph_name = current_workflow.get('graph_name')
    current_node = current_workflow.get('current_node')
    current_agent = current_workflow.get('current_agent')

    if not all([graph_name, current_node, current_agent]):
        return None

    # Get workflow context
    workflow_context = state.get('workflow_context', {})
    triggered_by = workflow_context.get('triggered_by', 'unknown')
    started_at = workflow_context.get('started_at', 'unknown')

    # Get pending artifacts
    pending_artifacts = state.get('pending_artifacts', [])
    current_node_artifacts = [
        a for a in pending_artifacts
        if a.get('expected_from') == current_node
    ]

    # Get available inputs
    input_artifact_names = get_current_node_inputs(state)
    input_artifact_paths = get_artifact_paths(input_artifact_names)

    # Get workflow history
    completed_nodes = state.get('completed_nodes', [])
    num_completed = len(completed_nodes)

    # Build context message
    message = f"""
╔═══════════════════════════════════════════════════════════════╗
║                    WORKFLOW CONTEXT LOADED                    ║
╚═══════════════════════════════════════════════════════════════╝

Current Workflow: {graph_name}
Current Node: {current_node}
Assigned Agent: {current_agent}
Workflow Status: {workflow_status}

Triggered By: {triggered_by}
Started: {started_at}
Nodes Completed: {num_completed}
"""

    if current_node_artifacts:
        message += f"\nExpected Outputs from This Node:\n"
        for artifact in current_node_artifacts:
            status = artifact.get('status', 'pending')
            artifact_name = artifact.get('artifact_name')
            message += f"  • {artifact_name} [{status}]\n"

    if input_artifact_paths:
        message += f"\nAvailable Input Artifacts:\n"
        for path in input_artifact_paths:
            message += f"  • {path}\n"

    if completed_nodes:
        message += f"\nRecent Workflow History:\n"
        for node in completed_nodes[-3:]:  # Show last 3 nodes
            node_name = node.get('node_name')
            agent = node.get('agent')
            artifacts = node.get('artifacts_produced', [])
            message += f"  • {node_name} ({agent})"
            if artifacts:
                message += f" → {', '.join(artifacts)}"
            message += "\n"

    # Check if in subgraph
    subgraph_stack = state.get('subgraph_stack', [])
    if subgraph_stack:
        current_subgraph = subgraph_stack[-1]
        parent_graph = current_subgraph.get('parent_graph')
        message += f"\n⚠ Note: Currently in subgraph, will return to {parent_graph} when complete\n"

    message += "\n" + "═" * 65 + "\n"

    return message


def generate_workflow_guidance(state):
    """Generate specific guidance for current workflow state."""
    if not state:
        return None

    workflow_status = state.get('current_workflow', {}).get('status')
    current_node = state.get('current_workflow', {}).get('current_node')

    guidance = None

    if workflow_status == 'waiting_approval':
        guidance = """
⏸ Workflow is awaiting human approval.
  This node requires human checkpoint before proceeding.
  Complete your task and await approval to continue.
"""

    elif workflow_status == 'waiting_subgraph':
        pending_subgraph = state.get('current_workflow', {}).get('pending_subgraph')
        guidance = f"""
⏸ Workflow requires subgraph invocation.
  Launch the {pending_subgraph} workflow to proceed.
  The main workflow will resume when the subgraph completes.
"""

    elif workflow_status == 'active':
        pending = state.get('pending_artifacts', [])
        node_pending = [a for a in pending if a.get('expected_from') == current_node]

        if node_pending:
            guidance = f"""
✓ Workflow is active. Your task:
  Create the required artifacts for node '{current_node}'.
  When complete, the workflow will automatically advance.
"""

    return guidance


def main():
    """Main hook execution."""
    try:
        # Read event data from stdin
        event_data = json.load(sys.stdin)

        print(f"[context-loader] Session starting, loading workflow context", file=sys.stderr)

        # Load workflow state
        state = load_state()
        if not state:
            print("[context-loader] No workflow state found", file=sys.stderr)
            return

        # Generate context message
        context_msg = generate_context_message(state)
        if context_msg:
            # Output to stderr (appears in logs)
            print(context_msg, file=sys.stderr)

            # Also output to stdout (visible to agent)
            print(context_msg)

            # Generate specific guidance
            guidance = generate_workflow_guidance(state)
            if guidance:
                print(guidance, file=sys.stderr)
                print(guidance)

        else:
            print("[context-loader] Workflow not active, no context to load", file=sys.stderr)

    except json.JSONDecodeError as e:
        print(f"[context-loader] ERROR: Failed to parse event data: {e}", file=sys.stderr)
    except Exception as e:
        print(f"[context-loader] ERROR: Unexpected error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)


if __name__ == '__main__':
    main()
