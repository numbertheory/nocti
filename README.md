# Nocti

Nocti is a CLI tool for note-taking and knowledge management. It allows you to initialize projects and manage resources like notebooks, todo lists, and calendars directly from your terminal.

## Features

- **Project Initialization**: Quickly set up a new Nocti project with a local configuration.
- **Resource Management**: Create and track different types of resources:
  - Notebooks
  - Todo Lists
  - Calendars
- **Local Storage**: All data is stored locally in a `.nocti/nocti.json` file within your project directory.
- **Unique ID Generation**: Automatic generation of unique 6-character hex IDs for all resources.

## Project Structure

- `main.go`: The entry point of the application.
- `cmd/`: Contains the CLI command definitions (built with Cobra).
  - `root.go`: The base command and versioning.
  - `init.go`: Logic for `nocti init`.
  - `new.go`: Logic for `nocti new` and resource subcommands.
- `tests/`: Unit tests for the application logic.
- `Makefile`: Build and test automation.

## Getting Started

### Prerequisites

- [Go](https://go.dev/doc/install) (version 1.26.2 or higher)

### Installation

To build and install the `nocti` binary to your local bin directory (`~/.local/bin`):

```bash
make install
```

Ensure `~/.local/bin` is in your `PATH`.

### Building Locally

To simply build the binary in the `build/` directory:

```bash
make build
```

### Running Tests

To run the unit tests:

```bash
make test
```

## Usage

### Initialize a Project

Create a new Nocti project in the current directory:

```bash
nocti init
```
Or specify a name with a flag:
```bash
nocti init --project "my-notes"
```

### Create Resources

You can create resources interactively:

```bash
nocti new
```

Or use specific subcommands with names:

```bash
nocti new notebook --name "Daily Journal"
nocti new todo --name "Project Tasks"
nocti new calendar --name "Work Schedule"
```

## Development

- **Clean build artifacts**: `make clean`
- **Show help**: `make help` or `nocti --help`
