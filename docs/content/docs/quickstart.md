---
title: "Quickstart"
weight: 1
---

# Quickstart Guide

This guide will help you install Nocti and get started with your first project.

## Installation

### Download Binaries
You can download the latest pre-compiled binaries for your system from the [GitHub Releases](https://github.com/numbertheory/nocti/releases) page.

### Build from Source
If you have [Go](https://go.dev/doc/install) (1.26.2+) installed, you can build Nocti manually:

#### Linux and macOS
```bash
git clone https://github.com/numbertheory/nocti.git
cd nocti
make install
```
*Note: This installs the binary to `~/.local/bin`. Ensure this directory is in your `PATH`.*

#### Windows (PowerShell)
```powershell
git clone https://github.com/numbertheory/nocti.git
cd nocti
go build -o nocti.exe main.go
# Move nocti.exe to a directory in your PATH
```

---

## Your First Project

### 1. Initialize a Project
Create a folder for your notes and tasks, then initialize it:
```bash
mkdir my-knowledge-base
cd my-knowledge-base
nocti init
```
This creates a `.nocti/` directory to store your project's configuration and registry.

### 2. The Interactive UI
Most of your time in Nocti will be spent in the interactive list view. Open it by running:
```bash
nocti list
```
*Tip: You can just run `nocti` with no arguments to trigger the list view automatically.*

---

## Working with Resources

Nocti uses three main types of resources. You can create any of them by pressing `n` inside the interactive view.

### 📓 Notebooks
For long-form writing, documentation, and organized notes.
- **Create**: Press `n` -> `Notebook`.
- **Use**: Enter the notebook to create standard Markdown (`.md`) or text (`.txt`) files.
- **Organization**: You can nest notebooks inside other notebooks to create a deep hierarchy.

### 📅 Calendars
For scheduling and daily logs.
- **Create**: Press `n` -> `Calendar`.
- **Overview**: When you enter a calendar, you see a list of days. 
- **Events**: Select a day and press `n` -> `Event` to create a note for that specific date.

### 📝 Todo Lists
For task management and progress tracking.
- **Create**: Press `n` -> `Todo`.
- **Tasks**: Inside a Todo resource, press `n` -> `Todo List` to create a task file from a template.
- **Tracking**: Use `- [ ]` and `- [x]` in your markdown. Nocti will automatically show your progress (e.g., `(2/5)`) in the file list!

---

## Basic Navigation Tips
- **Arrows**: Navigate the list.
  - In the Calendar view, use Ctrl+Up/Down to skip ahead by seven days.
- **Enter**: Enter a resource or edit a file in your default editor.
- **q / Esc**: Go back to the parent resource or exit.
- **Tab**: Switch focus between the file list and the preview pane.
- **Ctrl+H**: Show the help menu for more shortcuts.
