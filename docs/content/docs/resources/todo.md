---
title: "Todo Lists"
icon: "checklist"
weight: 3
---

Todo lists in Nocti allow you to manage tasks and track progress directly from the file list. A Todo resource is a specialized container that organizes markdown-based task lists and other sub-resources.

## Creating a Todo Resource

You can create a new Todo resource using the CLI:

```bash
nocti new todo --name "Project Tasks"
```

Or within the interactive `list` view by pressing `n` and selecting **Todo**.

## Managing Tasks

Within a Todo resource, you can create **Todo Lists**. These are standard Markdown files that utilize checkboxes for task tracking.

### Todo List Template
When you create a new Todo List via the interactive menu, Nocti uses a template (located in `templates/todo_template.md`) to pre-populate the file with a header and sample tasks:

```markdown
# My Tasks To Do List

- [ ] Sample Task 1
- [ ] Sample Task 2
- [ ] Sample Task 3
```

## Task Tracking

Nocti automatically scans your markdown files for GitHub-flavored checkboxes:
- `- [ ]` represents an incomplete task.
- `- [x]`, `- [X]`, or any other character inside the brackets (e.g., `- [/-]`) represents a completed task.

### Progress Indicators
The interactive file list provides real-time feedback on your progress:

- **In Progress**: If a file contains both complete and incomplete tasks, a progress ratio is appended to the name, such as `My Tasks (2/5)`.
- **Completed**: When all tasks in a file are finished, the ratio is removed, and the empty checkbox glyph `` is replaced with a bold checkmark `󰸞`.
- **Not Started**: If no tasks are completed, the file is shown with the standard empty checkbox glyph and no ratio.

## Context-Sensitive Creation

The creation menu (triggered by `n`) is context-sensitive. When you are inside or have selected a Todo resource, the options are tailored for task management:
1. **Todo List**: Creates a new task-based markdown file using the template.
2. **Notebook**: Creates a nested Notebook resource.
3. **Calendar**: Creates a nested Calendar resource.
4. **Todo**: Creates another nested Todo resource.

If you select a nested Notebook or Calendar, the menu will automatically switch to the options appropriate for those resource types.
