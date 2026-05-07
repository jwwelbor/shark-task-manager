# Shark CLI Templates

This directory contains templates for creating new epics, features, and tasks.

## Available Templates

### epic.md

Template for creating new epic files. Contains:
- YAML frontmatter with epic metadata
- Structured sections for goal, business value, quick reference
- Placeholders for epic components (personas, journeys, requirements, metrics, scope)

### feature.md

Template for creating new feature PRD files. Contains:
- YAML frontmatter with feature metadata
- Structured sections for goal, user personas, user stories, requirements
- Acceptance criteria, success metrics, dependencies, and integrations

### task.md

Template for creating new task files. Contains:
- YAML frontmatter with task metadata
- Structured sections for goals, success criteria, implementation guidance
- Validation gates and testing requirements

## Usage

Templates are automatically copied to your project's `shark-templates/` directory when you run:

```bash
shark init
```

To create new entities using templates:

**Epic**:
```bash
shark create epic "User Authentication System" --size=XL
```

**Feature**:
```bash
shark create feature E01 "OAuth Login Integration" --size=L
```

**Task**:
```bash
shark create task E01 F01 "Build Login" --agent=backend --size=M
```

The creation commands will automatically populate the templates with the correct keys and metadata.

## Sizing

Every new task and feature should carry a `--size`. Epics typically score 8/XL or 13/XXL since they exist to be decomposed.

| Numeric | T-shirt | Effort | Use for |
|--------:|:-------:|:-------|:--------|
| 1  | XS  | < 1 hour       | One-line change, doc tweak |
| 2  | S   | a few hours    | Single file or small set |
| 3  | M   | ~1 day         | Cohesive change across a few files |
| 5  | L   | 2-3 days       | Multiple components; consider splitting |
| 8  | XL  | ~1 week        | Many components; **should be split** |
| 13 | XXL | > 1 week       | Too large; **must be split** before work |

The `--size` flag accepts either form (case-insensitive): `--size=3` and `--size=M` are equivalent.

To **require** size on creation for a given entity type, set `size_required_for` in `.sharkconfig.json`:

```json
{
  "size_required_for": ["task", "feature"]
}
```

When set, `shark create <type>` rejects calls that omit `--size` for the listed types (mirrors the existing `tag_required_for` mechanism).

## Customization

You can customize these templates to match your team's workflow and documentation standards. After running `shark init`, edit the templates in your project's `shark-templates/` directory.
