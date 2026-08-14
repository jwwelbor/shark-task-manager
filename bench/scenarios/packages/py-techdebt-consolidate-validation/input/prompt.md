# Tech debt: task-title validation duplicated across three modules

`taskmanager/validation.py` already has a `non_empty_title` helper that
validates and cleans a task title (strips whitespace, rejects an empty or
whitespace-only value). Three other modules don't use it -- each
re-implements the exact same check inline instead:

- `taskmanager/bulk_import.py`'s `import_titles`
- `taskmanager/cli.py`'s `parse_title_arg`
- `taskmanager/reminders.py`'s `format_reminder`

`bulk_import.py`'s copy is bad enough that our linter (`ruff`) actually
flags it: the module still imports `non_empty_title` from
`taskmanager.validation`, and then immediately shadows that import with its
own local `non_empty_title` function, which ruff reports as
`redefined-while-unused` (rule `F811`).

Consolidate all three call sites onto the shared
`taskmanager.validation.non_empty_title` validator, and delete the inline
duplicate logic (including the shadowing redefinition in `bulk_import.py`).
Every module's public behavior -- what it accepts, what it rejects, and
what it returns -- must stay exactly the same; this is a pure clean-up, not
a feature change.
