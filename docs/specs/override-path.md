/feature create a new feature in the E07 epic

  let's update the epic and feature to store an optional path field. If it's there, in either case,
  if the value is present then it will be used as the base for the child items.

  e.g.
  docs/plan/E01-Default-epic-naming/E01-F01-default-feature-name/tasks/T-E01-F01-001-default-task.md

  could become
  docs/plan/special-epic/E01-F01-default-feature-name/tasks/T-E01-F01-001-default-task.md
  or
  docs/plan/E01-default-epic/phase-1/F01-custom-feature-name/tasks/T-E01-F01-001-default-task.md

  the rule is that when creating a new epic or feature, it looks up the entry, and if a
  custom_folder_path is set, then it uses that as the base and creates the new item with the
  custom_folder_path as the base. If custom_folder_path is not set, then it uses the default logic.

  the custom_path would be set via a cli parameter `--path`