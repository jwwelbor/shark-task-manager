# PM CLI Templates

This directory contains templates for creating new tasks and features.

## Available Templates

### task.md

Template for creating new task files. Contains:
- YAML frontmatter with task metadata
- Structured sections for goals, success criteria, implementation guidance
- Validation gates and testing requirements

## Usage

Templates are automatically copied to your project's `templates/` directory when you run:

```bash
pm init
```

To create a new task using a template:

```bash
pm task create --epic=E01 --feature=F02 --title="Build Login" --agent=backend
```

The task creation command will automatically populate the template with the correct task key and metadata.

## Customization

You can customize these templates to match your team's workflow and documentation standards. After running `pm init`, edit the templates in your project's `templates/` directory.
