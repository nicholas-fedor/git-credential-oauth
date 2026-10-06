# docgen

CLI documentation generator for git-credential-oauth.

## Overview

docgen introspects the Cobra command tree at runtime and emits Hugo-compatible Markdown files with frontmatter. It is the single source of truth for the site's CLI reference at `docs/content/cli-reference/`.

The tree comes from `cmd.DocRoot`, which builds the same command tree the binary runs, without the process dependencies. No command is executed, so the generated surface cannot drift from the shipped one.

## Directory Structure

```text
tools/docgen/
    main.go           Entry point and CLI flag parsing
    models.go         Data structures for template rendering
    extractor.go      Cobra command tree introspection
    generator.go      Orchestration: output directory, recursive generation
    renderer.go       Go text/template rendering with Hugo helpers
    templates/
        command.tmpl  Single command page template
        index.tmpl    Root index page template
```

## Architecture

docgen uses three main components:

- **DocExtractor** (`extractor.go`) — Walks the `cobra.Command` tree, builds `CommandDoc` structures with titles, descriptions, flags, examples, and subcommand index data.
- **DocGenerator** (`generator.go`) — Creates the output directory, extracts the root doc, renders the root `_index.md`, then recursively renders each top-level section and its subcommands as directories of `_index.md` files.
- **TemplateRenderer** (`renderer.go`) — Loads `text/template` files and writes rendered Markdown to disk with `0o600` permissions.

Generated files include Hugo frontmatter (`title`, `description`, `type: docs`) so they are immediately consumable by the Hugo site.

## What Is Documented

Cobra only reports a command that is neither hidden nor deprecated, so `store` and `erase` are omitted. They exist for the git credential protocol, and documenting them would imply a user-facing surface that does not exist.

## Usage

```bash
go run ./tools/docgen -out ./docs/content/cli-reference
```

or via Taskfile:

```bash
task docs
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-out` | `./docs/content/cli-reference` | Output directory for generated Markdown |

## Output

Running docgen produces a directory tree of `_index.md` files:

```text
docs/content/cli-reference/
    _index.md                  Root index (CLI Reference)
    capability/_index.md
    configure/_index.md
    get/_index.md
    unconfigure/_index.md
    version/_index.md
```

## Inline Generation

docgen can also be triggered via `go:generate`:

```go
//go:generate go run github.com/nicholas-fedor/git-credential-oauth/tools/docgen -out ../../docs/content/cli-reference
```

## Dependencies

- `github.com/spf13/cobra` — Command tree introspection
- `github.com/spf13/pflag` — Flag metadata extraction
- `golang.org/x/text/cases` — Title-casing command names for display

## Notes

- docgen is part of the main module (no separate `go.mod`).
- Templates are resolved relative to `renderer.go`, so the output directory is the only thing the working directory affects.
- Run `task docs` after any CLI change that affects command descriptions, flags, or subcommands, and commit the regenerated pages.