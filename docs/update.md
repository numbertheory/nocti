# `nocti update`

The `update` command synchronizes the central `.nocti/nocti.json` registry with the actual resource directories present in the project.

## Usage

```bash
nocti update
```

## Description

Running `nocti update` scans the immediate subdirectories of the project root for hidden `.nocti.json` files. For every resource found, it reads the metadata and rebuilds the resource lists (`notebooks`, `todos`, `calendars`) in the main configuration file.

This is useful if:
*   Resources were manually moved or deleted.
*   The main `nocti.json` file became corrupted or out of sync.
*   You are importing an existing set of resource folders into a new Nocti project.

### Constraints

*   **Project Root Only**: This command must be run from the project root directory (where the `.nocti/` folder resides). It will fail if run inside an individual resource directory.
*   **Immediate Children**: Only directories directly inside the project root are scanned. Nested sub-resources are currently updated via their parent's metadata but are registered in the main config when created.

## Examples

Rebuild the project registry:
```bash
nocti update
```
