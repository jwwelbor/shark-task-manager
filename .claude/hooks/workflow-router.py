#!/usr/bin/env python3
"""
Hook Name: workflow-router.py
Event: Stop
Purpose: Route to next agent when current agent completes

This hook fires when an agent session ends. It:
1. Checks if workflow is active
2. Verifies current node is complete
3. Consults workflow CSV to find next node
4. Updates state for transition
5. Provides handoff instructions for next agent
"""

import json
import sys
import csv
from pathlib import Path
from datetime import datetime


# Map workflow graph names to CSV files
GRAPH_CSV_MAP = {
    'PDLC': '01-pdlc.csv',
    'Feature-Refinement': '02-feature-refinement.csv',
    'Story-Elaboration-Subgraph': '03-story-elaboration.csv',
    'Prototyping-Subgraph': '04-prototyping.csv',
    'Tech-Spec-Subgraph': '05-tech-spec.csv',
    'Development-Subgraph': '06-development.csv',
    'Infrastructure-Setup': '07-infrastructure.csv',
    'Software-Development-Lifecycle': '08-release.csv'
}


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


def load_workflow_csv(graph_name):
    """Load workflow CSV definition for given graph."""
    project_root = Path.cwd()
    csv_filename = GRAPH_CSV_MAP.get(graph_name)

    if not csv_filename:
        print(f"ERROR: Unknown workflow graph: {graph_name}", file=sys.stderr)
        return None

    csv_path = project_root / 'docs' / 'plan' / 'E01-SDLC-Workflow' / 'csv' / csv_filename

    if not csv_path.exists():
        print(f"ERROR: Workflow CSV not found: {csv_path}", file=sys.stderr)
        return None

    try:
        with open(csv_path) as f:
            reader = csv.DictReader(f)
            nodes = {row['node_name']: row for row in reader}
        return nodes
    except Exception as e:
        print(f"ERROR: Failed to read workflow CSV: {e}", file=sys.stderr)
        return None


def check_node_complete(state):
    """Check if current node has all required artifacts created."""
    current_node = state.get('current_workflow', {}).get('current_node')
    if not current_node:
        return False

    # Check if any pending artifacts for this node remain
    pending = [
        a for a in state.get('pending_artifacts', [])
        if a.get('expected_from') == current_node and a.get('status') != 'created'
    ]

    return len(pending) == 0


def record_node_completion(state):
    """Record current node as completed in history."""
    current_node = state['current_workflow']['current_node']
    current_agent = state['current_workflow']['current_agent']

    # Get artifacts produced by this node
    artifacts_produced = [
        a['artifact_name'] for a in state.get('pending_artifacts', [])
        if a.get('expected_from') == current_node and a.get('status') == 'created'
    ]

    completion_record = {
        'node_name': current_node,
        'agent': current_agent,
        'completed_at': datetime.now().isoformat(),
        'artifacts_produced': artifacts_produced
    }

    state['completed_nodes'].append(completion_record)
    print(f"[workflow-router] Recorded completion of node: {current_node}", file=sys.stderr)


def determine_next_node(state, workflow_nodes):
    """Determine the next node based on current position and CSV definition."""
    current_node = state['current_workflow']['current_node']

    if current_node not in workflow_nodes:
        print(f"ERROR: Current node '{current_node}' not found in workflow CSV", file=sys.stderr)
        return None

    current_node_def = workflow_nodes[current_node]
    next_nodes_value = current_node_def.get('next_nodes', '').strip()

    if not next_nodes_value:
        print(f"ERROR: No next_nodes defined for {current_node}", file=sys.stderr)
        return None

    # Handle terminal node
    if next_nodes_value == '__end__':
        return '__end__'

    # Handle parallel nodes (pipe-separated)
    if '|' in next_nodes_value:
        next_nodes = [n.strip() for n in next_nodes_value.split('|')]
        print(f"[workflow-router] Parallel next nodes detected: {next_nodes}", file=sys.stderr)
        # For now, just return the first one
        # TODO: Implement parallel execution
        return next_nodes[0]

    return next_nodes_value


def prepare_next_node_state(state, next_node_name, workflow_nodes):
    """Update state for next node."""
    if next_node_name == '__end__':
        # Workflow complete
        state['current_workflow']['status'] = 'completed'
        state['current_workflow']['current_node'] = '__end__'
        state['current_workflow']['current_agent'] = None

        # Check if we're in a subgraph
        if state.get('subgraph_stack'):
            print(f"[workflow-router] Subgraph complete, should return to parent", file=sys.stderr)
            # TODO: Handle subgraph return
        else:
            state['workflow_context']['completed_at'] = datetime.now().isoformat()
            print(f"[workflow-router] Workflow completed!", file=sys.stderr)

        return

    # Check if next is a subgraph
    if next_node_name.endswith('-Subgraph') or next_node_name.endswith('-Workflow') or next_node_name.endswith('-Setup'):
        print(f"[workflow-router] Next step is subgraph: {next_node_name}", file=sys.stderr)
        # TODO: Handle subgraph launch
        state['current_workflow']['status'] = 'waiting_subgraph'
        state['current_workflow']['pending_subgraph'] = next_node_name
        return

    # Normal next node
    next_node_def = workflow_nodes.get(next_node_name)
    if not next_node_def:
        print(f"ERROR: Next node '{next_node_name}' not found in workflow CSV", file=sys.stderr)
        return

    next_agent = next_node_def.get('agent_type')
    next_outputs = next_node_def.get('outputs', '').split('|') if next_node_def.get('outputs') else []
    next_inputs = next_node_def.get('inputs', '').split('|') if next_node_def.get('inputs') else []

    # Update current workflow
    state['current_workflow']['current_node'] = next_node_name
    state['current_workflow']['current_agent'] = next_agent
    state['current_workflow']['updated_at'] = datetime.now().isoformat()

    # Update pending artifacts for next node
    state['pending_artifacts'] = [
        {
            'artifact_name': output.strip(),
            'required_by': next_node_def.get('next_nodes', 'unknown'),
            'expected_from': next_node_name,
            'status': 'pending'
        }
        for output in next_outputs if output.strip()
    ]

    print(f"[workflow-router] Transitioned to node: {next_node_name} (agent: {next_agent})", file=sys.stderr)


def generate_handoff_message(state, next_node_name, workflow_nodes):
    """Generate handoff message for next agent."""
    if next_node_name == '__end__':
        return "\n=== WORKFLOW COMPLETE ===\nAll nodes in this workflow have been executed.\n"

    if state['current_workflow'].get('status') == 'waiting_subgraph':
        pending_subgraph = state['current_workflow'].get('pending_subgraph')
        return f"\n=== SUBGRAPH LAUNCH REQUIRED ===\nWorkflow requires launching subgraph: {pending_subgraph}\n"

    next_node_def = workflow_nodes.get(next_node_name)
    if not next_node_def:
        return "\n=== ERROR ===\nNext node definition not found.\n"

    next_agent = next_node_def.get('agent_type')
    description = next_node_def.get('description')
    outputs = next_node_def.get('outputs', '').split('|')
    inputs = next_node_def.get('inputs', '').split('|')

    message = f"""
=== WORKFLOW HANDOFF ===

Next Agent: {next_agent}
Current Node: {next_node_name}

Task: {description}

Required Outputs:
{chr(10).join(f"  - {o.strip()}" for o in outputs if o.strip())}

Available Inputs:
{chr(10).join(f"  - {i.strip()}" for i in inputs if i.strip())}

To proceed: Launch the {next_agent} agent with context about node {next_node_name}.
The agent should create the required outputs, which will trigger the next workflow step.

=========================
"""
    return message


def main():
    """Main hook execution."""
    try:
        # Read event data from stdin
        event_data = json.load(sys.stdin)

        print(f"[workflow-router] Agent session ending, checking workflow state", file=sys.stderr)

        # Load workflow state
        state = load_state()
        if not state:
            print("[workflow-router] No workflow state found, exiting", file=sys.stderr)
            return

        # Check if workflow is active
        workflow_status = state.get('current_workflow', {}).get('status')
        if workflow_status not in ['active', 'waiting', 'waiting_approval']:
            print(f"[workflow-router] Workflow status is '{workflow_status}', no routing needed", file=sys.stderr)
            return

        # Check if current node is complete
        if not check_node_complete(state):
            print(f"[workflow-router] Current node incomplete, not routing", file=sys.stderr)
            print(f"[workflow-router] Pending artifacts:", file=sys.stderr)
            for art in state.get('pending_artifacts', []):
                if art.get('status') != 'created':
                    print(f"  - {art['artifact_name']} ({art['status']})", file=sys.stderr)
            return

        # Record completion
        record_node_completion(state)

        # Load workflow CSV
        graph_name = state['current_workflow']['graph_name']
        workflow_nodes = load_workflow_csv(graph_name)
        if not workflow_nodes:
            return

        # Determine next node
        next_node_name = determine_next_node(state, workflow_nodes)
        if not next_node_name:
            return

        # Update state for next node
        prepare_next_node_state(state, next_node_name, workflow_nodes)

        # Save state
        if not save_state(state):
            return

        # Generate handoff message
        handoff_msg = generate_handoff_message(state, next_node_name, workflow_nodes)
        print(handoff_msg, file=sys.stderr)

        # Output to Claude
        print("\n" + handoff_msg)

    except json.JSONDecodeError as e:
        print(f"[workflow-router] ERROR: Failed to parse event data: {e}", file=sys.stderr)
    except Exception as e:
        print(f"[workflow-router] ERROR: Unexpected error: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)


if __name__ == '__main__':
    main()
