# Interactive Mode Configuration

Controls whether commands prompt for user input when multiple options are available.

## Configuration Field

- **Name:** `interactive_mode`
- **Type:** Boolean
- **Default:** `false` (non-interactive)
- **Purpose:** Controls interactive prompts in status transition commands

## Default Behavior (Non-Interactive)

- Commands automatically select the first valid option from workflow configuration
- Ideal for agent/automation workflows
- Prints clear message showing which option was selected
- Never blocks waiting for user input

## Interactive Mode (Opt-In)

- Commands display interactive prompts for user selection
- Requires manual input when multiple options available
- Suitable for human users who want explicit control
- Enable by setting `interactive_mode: true` in `.sharkconfig.json`

## Configuration Example

```json
{
  "interactive_mode": false,
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"],
    "in_qa": ["ready_for_approval", "in_development"]
  },
  "status_metadata": {
    "in_qa": {
      "color": "yellow",
      "phase": "qa",
      "progress_weight": 80
    }
  }
}
```

## Usage Examples

### Non-Interactive Mode (Default)

```bash
$ shark task next-status E07-F23-006
ℹ Auto-selected next status: in_qa (from 2 options)
✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa
```

### Interactive Mode (When Enabled)

```bash
# .sharkconfig.json: { "interactive_mode": true }
$ shark task next-status E07-F23-006
Task: T-E07-F23-006
Current status: ready_for_qa

Available transitions:
  1) in_qa
  2) on_hold

Enter selection [1-2]: 1
✅ Task T-E07-F23-006 transitioned: ready_for_qa → in_qa
```

## When to Use Each Mode

| Use Case | Recommended Mode | Reason |
|----------|------------------|--------|
| AI Agent workflows | Non-interactive (default) | Agents can't provide interactive input |
| CI/CD pipelines | Non-interactive (default) | Automation requires predictable behavior |
| Scripts/batch operations | Non-interactive (default) | Background processes need non-blocking execution |
| Human users (manual) | Interactive (opt-in) | Explicit control over status transitions |
| Development/debugging | Interactive (opt-in) | Review options before selecting |

## Auto-Selection Logic

When `interactive_mode` is `false` (default) and multiple transitions are available, the command automatically selects the first transition defined in the workflow configuration:

```json
{
  "status_flow": {
    "ready_for_qa": ["in_qa", "on_hold"]
    //               ^^^^^^^^  <- This is auto-selected (non-interactive mode)
  }
}
```

The order in `status_flow` determines selection priority. Place the most common/preferred transition first.

## Configuration Impact

| Config Setting | Multiple Transitions | Single Transition |
|----------------|---------------------|-------------------|
| `interactive_mode: false` (default) | Auto-selects first option | Auto-selects only option |
| `interactive_mode: true` | Shows interactive prompt | Auto-selects only option |
| `--status` flag provided | Uses specified status | Uses specified status |

## Use Cases

- **Agent/Automation Workflows:** Use default non-interactive mode
- **CI/CD Pipelines:** Use default non-interactive mode
- **Human Manual Operations:** Enable interactive mode in config
- **Explicit Control:** Use `--status` flag to specify exact transition

## Related Documentation

- [Workflow Configuration](workflow-config.md) - Configure status flows
- [Task Commands](task-commands.md) - Task status transition commands
- [Configuration](configuration.md) - General configuration commands
