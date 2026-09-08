# Epic: task organization (tagging + filtered listing)

Our task manager only supports one flat list of tasks today. As the list
grows, it's getting hard to find related tasks or focus on one area of work
at a time. We want a small, coherent capability for organizing tasks by
topic.

This epic covers exactly two feature slices:

1. **Task tagging.** Let a task carry zero or more free-form tags: settable
   when the task is created, and attachable afterward to an existing task.
   Attaching a tag that's already present must not create a duplicate.

2. **Filtered listing.** Let callers list only the tasks that carry a given
   tag, in the same order they'd appear in the full list. Listing with no
   filter must keep behaving exactly as it does today.

Everything else about `TaskManager` (adding, completing, checking overdue
tasks) must keep working unchanged for tasks that are never tagged.
