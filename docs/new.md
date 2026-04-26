# `nocti new`

The `new` command creates a new resource (notebook, todo list, or calendar) within a Nocti project.

## Usage

```bash
nocti new [command] [flags]
```

## Description

Resources are the primary way to organize information in Nocti. Each resource is created as a directory containing a hidden `.nocti.json` file with its metadata (ID, name, type, and parent information).

### Hierarchical Resources

If you run `nocti new` while inside another resource's directory, the new resource will be created as a child of the current one. This relationship is tracked in:
1.  The main `.nocti/nocti.json` file (via a `parent` key in the resource entry).
2.  The parent resource's local `.nocti.json` file (in a `resources` list).
3.  The child resource's local `.nocti.json` file (via a `parent` key).

## Subcommands

*   `notebook`: Create a new notebook.
*   `todo`: Create a new todo list.
*   `calendar`: Create a new calendar.

## Flags (Global for `new`)

*   `-n, --name string`: The name of the resource to create. If not provided, you will be prompted.
*   `-o, --overwrite`: Overwrite the hidden `.nocti.json` file if it already exists in the target directory.

## Examples

Create a notebook interactively:
```bash
nocti new
```

Create a todo list with a specific name:
```bash
nocti new todo --name "Shopping List"
```

Create a nested resource:
```bash
cd my-notebook
nocti new todo --name "Notebook Tasks"
```
