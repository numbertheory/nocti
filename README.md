# Nocti

Nocti is a specialized CLI tool designed for developers and power users who want to manage their notes, tasks, and schedules directly from the terminal. 

Built on a foundation of local-first principles, Nocti organizes your knowledge into a hierarchy of directories, using simple JSON files for metadata and standard text formats (Markdown and TXT) for content.

## Key Concepts

*   **Projects**: A project is the root container for all your work, initialized with a central registry.
*   **Resources**: The building blocks of your organization. Nocti currently supports three types:
    *   **Notebooks**: For long-form notes and documentation.
    *   **Todo Lists**: For task management.
    *   **Calendars**: For scheduling.
*   **Hierarchy**: Resources can be nested, allowing you to create complex structures that reflect your mental model (e.g., a "Project Tasks" todo list inside a "Work" notebook).

## Detailed Documentation

For full command references and advanced usage, please see the following guides:

*   [`nocti init`](docs/init.md): Setting up your project.
*   [`nocti new`](docs/new.md): Creating and nesting resources.
*   [`nocti list`](docs/list.md): Exploring your notebook content.

## Installation

### Prerequisites
- [Go](https://go.dev/doc/install) (version 1.26.2 or higher)

### Quick Start
To build and install the `nocti` binary to your local bin directory:
```bash
make install
```
Ensure `~/.local/bin` is in your `PATH`.

## Project Structure

- `cmd/`: CLI command definitions (Cobra).
- `docs/`: Detailed command documentation.
- `tests/`: Automated test suite.
- `.nocti/`: Hidden project registry (created on `init`).
