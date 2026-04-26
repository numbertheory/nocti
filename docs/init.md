# `nocti init`

The `init` command initializes a new Nocti project in the current working directory.

## Usage

```bash
nocti init [flags]
```

## Description

Running `nocti init` creates a hidden `.nocti/` directory containing a `nocti.json` file. This file serves as the central configuration and registry for all resources (notebooks, todo lists, calendars) created within the project.

You cannot run `nocti init` inside an existing Nocti resource directory.

## Flags

*   `-p, --project string`: The name of the Nocti project. If not provided, you will be prompted to enter one.

## Examples

Initialize a project with a specific name:
```bash
nocti init --project "My Research Project"
```

Initialize a project interactively:
```bash
nocti init
```
