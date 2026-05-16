---
title: "Templates"
icon: "description"
weight: 2
---

Templates allow you to create new pages with pre-filled content and dynamic metadata. They are specific to each resource (notebook, todo, or calendar) and are stored within the resource itself.

## Setup

To use templates in a resource, create a hidden directory named `.templates` at the root of that resource.

```bash
cd my-notebook
mkdir .templates
```

Any markdown (`.md`) files you place in this folder will automatically appear as options in the "New" dialog within `nocti list`.

## Creating from a Template

1.  Run `nocti list` inside your notebook.
2.  Press **n** to open the creation menu.
3.  Select your template from the list.
4.  Enter a name for the new file (or use the pre-filled one).

## Dynamic Content

Templates support placeholders that are automatically replaced when a new file is created.

### Built-in Variables

| Variable | Description |
| :--- | :--- |
| `{{NAME}}` | The name you provide for the new file (without extension). |
| `{{DATE}}` | The current date in `YYYY-MM-DD` format. |

### Frontmatter Variables

You can define custom metadata at the top of your template. Any key-value pair defined in the frontmatter can be used as a variable in the body.

The frontmatter section must end with a line containing at least three dashes (`---`).

**Example Template (`.templates/Meeting.md`):**
```markdown
Project: Nocti CLI
Attendees: @user1, @user2
---
# Meeting: {{NAME}}
Date: {{DATE}}
Project: {{Project}}
Attendees: {{Attendees}}

## Agenda
...
```

The frontmatter section itself is stripped from the final file.

## Advanced Naming

Templates can suggest or enforce specific naming conventions using the `filename` key in their frontmatter.

### Automated Date Naming
```markdown
filename: Log-{{DATE}}
---
# Daily Log for {{DATE}}
```
Selecting this template will pre-fill the naming prompt with `Log-2026-05-12`.

### Automated Incrementers
The `{{INC}}` variable automatically finds the next available number in the current directory.

```markdown
filename: Task-{{INC}}
---
# Task {{INC}}
```
If `Task-1.md` and `Task-2.md` already exist in the folder, selecting this template will pre-fill the naming prompt with `Task-3`.
