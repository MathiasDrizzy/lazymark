# Agent guide

lazymark is a Go terminal app with CLI and MCP entry points over the same Markdown
notes. Start with [README.md](README.md) for the feature overview.

Read the documentation for the area you are changing:

- [Configuration and notes folders](docs/configuration.md): settings, defaults and folder selection.
- [Tags and categories](docs/tags.md): inline parsing, exclusions and category names.
- [CLI and MCP](docs/cli.md): commands, IDs, JSON schemas, dates and Kanban format.
- [Templates and daily notes](docs/templates.md), [links](docs/links.md), and [editor plugins](docs/editor-plugins.md).
- [Keybindings](docs/keybindings.md), [translations](docs/i18n.md), and [Windows checks](docs/windows.md).

Storage and Markdown parsing live in `internal/storage`; shared headless operations
in `internal/ops`; entry points in `internal/app`, `internal/cli` and `internal/mcp`.
Keep user-facing behavior documented in `docs/` and link to it from this guide.

CI checks Go formatting, `go vet ./...`, `go test -race -count=1 ./...` and builds
for macOS, Linux and Windows; see [.github/workflows/ci.yml](.github/workflows/ci.yml).
