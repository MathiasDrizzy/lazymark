# Command line and MCP

lazymark has headless commands for scripts and agents. They read and write the same notes as the app, with the same rules: a path must be a `.md` note inside the notes folder (no `..`, no symlinks out of it), and a task is only rewritten on its own line.

All commands take `--dir <folder>` (default: the notes folder of your config) and `--json`. Flags and arguments can come in any order. `-h` prints the usage.

```
lazymark note list [--json]
lazymark note show <path> [--json]
lazymark note new  <title> [--folder <subfolder>] [--empty] [--json]
lazymark task list [--json] [--pending] [--column <id>] [--note <path>]
lazymark task toggle <id> [--json]
lazymark task move   <id> <column> [--json]
```

`note get <path>` and `task toggle --path <note> --line <n>` still work.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Done |
| 1 | The command was valid but failed (read or write error) |
| 2 | Invalid arguments, unknown column, or a path outside the notes folder. Nothing is touched, not even the notes folder |
| 3 | The note or the task does not exist |
| 4 | The note changed on disk while the command ran. Nothing is written; run it again |

Errors go to stderr; stdout stays empty.

The text output (without `--json`) drops control characters from the notes (escape sequences, BEL, `\r`), so a note cannot write to your terminal. `--json` escapes them and keeps the exact text.

## Task ids

A task id is `<note path relative to the notes folder>#<8 hex>`, and `.2`, `.3`… for the second and later tasks with the same text in the same note: `projects/plan.md#16de6420`. The hash is of the task text without its Kanban tag, lowercased, so the id survives editing or inserting other lines, moving the task to another column and ticking it. It changes if you edit the task's own text.

`<column>` is a column id (`todo`, `doing`, `done`, or your own) or its visible title.

## JSON schema

The fields below are stable: they are only ever added to. See `internal/cli/testdata/golden/` for complete examples, which the tests compare byte by byte.

`note list` → array of notes, `note new` → one note:

| Field | Type | |
|---|---|---|
| `id` | string | file name |
| `title` | string | |
| `path` | string | absolute |
| `tags` | string[] | categories (`#kb/…` is not one) |
| `tasks_count` | number | |
| `mod_time` | string | RFC 3339 |

`note show` → a note with `content` (string) added.

`task list` → array of tasks, `task move` and `task toggle` → the task after the change:

| Field | Type | |
|---|---|---|
| `id` | string | see above |
| `text` | string | without the Kanban tag |
| `column` | string | column id |
| `done` | bool | |
| `line` | number | 1-based |
| `note` | string | relative path |
| `note_title` | string | |
| `path` | string | absolute |

Notes are sorted by path, tasks by their order in the note.

## MCP server

`lazymark mcp [--dir <folder>]` serves MCP over stdio (JSON-RPC 2.0, protocol `2024-11-05`). The tools are the commands above:

| Tool | Arguments | Same as |
|---|---|---|
| `list_notes` | | `note list` |
| `read_note` | `path` | `note show` |
| `create_note` | `title`, `folder?`, `empty?` | `note new` |
| `list_tasks` | `pending_only?`, `column?`, `note_path?` | `task list` |
| `move_task` | `id`, `column` | `task move` |
| `toggle_task` | `id` (or `path` and `line`) | `task toggle` |
| `get_kanban` | | the board: columns in order, each with its cards |

An error comes back as a tool result with `isError: true` and the exit code in the text.

### Register it in Claude Code

```
claude mcp add --transport stdio lazymark -- lazymark mcp
```

Add `--scope user` to have it in all your projects, or `--scope project` to share it through `.mcp.json`. `claude mcp list` shows it and `/mcp` checks it inside Claude Code. The syntax is from the [Claude Code MCP documentation](https://code.claude.com/docs/en/mcp). Use `-- lazymark mcp --dir <folder>` for a notes folder other than the default.

## Kanban format

The column of a task is a tag at the end of its line: `- [ ] write report #kb/doing`. No tag means the first column; `[x]` is always the done column; a `#kb/…` tag that is not one of your columns counts as the first column and is left alone until you move the card. Only `[ ]` and `[x]` are ever written. The old `#doing`, `#wip`, `#progreso` and `#in-progress` tags are read as `doing` and replaced by `#kb/doing` only when you move that card.
