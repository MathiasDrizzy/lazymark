# Command line and MCP

lazymark has headless commands for scripts and agents. They read and write the same notes as the app, with the same rules: a path must be a `.md` note inside the notes folder (no `..`, no symlinks out of it), and a task is only rewritten on its own line.

All commands take `--dir <folder>` (default: the notes folder of your config) and `--json`. Flags and arguments can come in any order. `-h` prints the usage.

```
lazymark note list [--json]
lazymark note show <path> [--json]
lazymark note new  <title> [--folder <subfolder>] [--empty] [--template <name>] [--json]
lazymark daily [--json]
lazymark search <text> [--regex] [--case] [--limit <n>] [--json]
lazymark task list [--json] [--pending] [--column <id>] [--note <path>]
lazymark task toggle <id> [--json]
lazymark task move   <id> <column> [--json]
lazymark task due    <id> <YYYY-MM-DD|none> [--json]
lazymark task start  <id> <YYYY-MM-DD|none> [--json]
lazymark dates migrate --to dataview|emoji [--dry-run] [--json]
```

`note new --template <name>` fills the note from `templates/<name>.md` (see [templates.md](templates.md)); a template that does not exist exits with 3 and one that is not valid text (binary, UTF-16, over 256 KB) exits with 2; neither creates anything. An unknown `{{variable}}` stays as written, is reported on stderr (`warning: unknown variable: {{x}}`) and in a `warnings` array of the JSON, and does not change the exit code. `daily` creates today's note `journal/YYYY-MM-DD.md` from `templates/daily.md`, or opens it if it exists, and prints its path (`--json`: the note plus `"created": true|false`); it never modifies an existing one.

`note get <path>` and `task toggle --path <note> --line <n>` still work.

## Search

`lazymark search <text>` looks for the text in every note and prints `note:line: text` for each match, ordered by note and line. It is case-insensitive (accents count: `cafe` does not find `café`); `--case` makes it case-sensitive and `--regex` reads the text as a regular expression ([RE2 syntax](https://github.com/google/re2/wiki/Syntax)). `--limit <n>` caps the matches (500 by default; `truncated` tells you there were more). Several words without quotes are one search. No match is not an error: it prints nothing and exits with 0; an empty search, an invalid expression or a negative limit exit with 2 and touch nothing.

There is no index: the notes are scanned on every search, in parallel, skipping the ones larger than 2 MiB (`skipped` counts them), the hidden folders (`.trash`) and `assets/`, and symbolic links that leave the notes folder. A note linked from inside the folder is searched once. A search stops after 10 seconds: it returns what it found with `"timed_out": true` (a field that only appears when it happens; and a warning on stderr), and a regular expression that would be too costly (long or nested repetitions over Unicode classes, such as `([\p{L}\p{N}]{50}){20}x`; the cost is computed from the compiled program before running it) is refused at once with exit code 2, because those can take minutes on a big note. When a search is cut short, nothing keeps running in the background.

`--json` prints `{"query", "matches", "files", "skipped", "truncated"}`; each match is `{"note", "path", "title", "line", "text", "start", "end"}`, where `text` is the line trimmed around the match (with `…` if it is long, and without control characters) and `start`/`end` are the byte offsets of the match inside `text`.

In the app, `/` opens the same search with live results; `Enter` (or a second click) jumps to the note and the line.

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

`due` and `start` set (or, with `none`, remove) the due and start dates of a task. Setting replaces the first marker of that emoji in the line, even one with an invalid date, and drops any other marker of the same emoji, so a line never ends up with two; `none` removes all of them and tidies the spaces around. The `id` that `task due`, `task start`, `task move` and `task toggle` print (and put in `--json`) is read again after writing, so it is the task's current id: if the edit changed the text that the id is made of (for example by removing a repeated marker) it differs from the one you passed; an invalid date (not `YYYY-MM-DD`, or one that does not exist like `2026-02-30`) exits with 2 and touches nothing. The completion date is not set by hand: moving a task to the done column (or ticking it) adds the completion date (today) unless it already had one, and moving it out removes it. In the text output the dates follow the column, drawn with symbols (`▸` start, `◑` scheduled, `◷` due, `✓` completed, `+` created), never the emoji or the Dataview syntax of the file; `--json` has `start`, `due`, `completed` and, when the task has them, `scheduled` and `created`: `… (todo)  ▸ 2026-05-01 ◷ 2026-05-10 (overdue)`.

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
| `start` | string | start date `YYYY-MM-DD`, or `""` |
| `due` | string | due date, or `""` |
| `completed` | string | completion date, or `""` |
| `overdue` | bool | has a due date before today and is not done |
| `line` | number | 1-based |
| `note` | string | relative path |
| `note_title` | string | |
| `path` | string | absolute |

Notes are sorted by path, tasks by their order in the note.

## MCP server

`lazymark mcp [--dir <folder>]` serves MCP over stdio (newline-delimited JSON-RPC 2.0). It speaks both eras of the protocol on the same connection:

- **2026-07-28 (current):** no sessions and no `initialize` handshake. Every request carries its protocol version and client capabilities in `_meta` (`io.modelcontextprotocol/protocolVersion`, `io.modelcontextprotocol/clientCapabilities`), the server answers each one on its own (`resultType: "complete"` and its name in `_meta["io.modelcontextprotocol/serverInfo"]`) and implements `server/discover` (supported versions, capabilities, identity). A version it does not support gets `UnsupportedProtocolVersion` (`-32022`) with the list of the ones it does; a request with some but not all of the per-request fields (or with a field of the wrong type) gets `-32602`; an `id` that is `null`, an object or an array gets `-32600` (the specification requires a string or integer); `tools/list` carries `ttlMs` and `cacheScope` (`CacheableResult`); `ping`, which that revision removed, is answered with `-32601`; an unknown tool in `tools/call` is a JSON-RPC error `-32602` ("Unknown tool: …"), as the [tools specification](https://modelcontextprotocol.io/specification/2026-07-28/server/tools#error-handling) requires, while an invalid argument is a tool result with `isError: true`. The older versions that `server/discover` lists in `supportedVersions` are only valid through the `initialize` handshake: a request that carries `_meta` with one of them is rejected with `-32022`. See [Versioning and Compatibility](https://modelcontextprotocol.io/specification/2026-07-28/basic/versioning) and [stdio](https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio).
- **2025-11-25 and earlier (`2025-06-18`, `2025-03-26`, `2024-11-05`):** the `initialize` handshake; the server answers with the version the client asks for if it knows it, and with `2025-11-25` otherwise.

Which one is used depends on how the client opens (the same rule the specification gives for dual-era servers). Claude Code speaks 2026-07-28 with its v2 runtime, but asks stdio servers for it only when `MCP_PROTOCOL_NEGOTIATION=auto` is set, and otherwise connects as before ([Claude Code MCP documentation](https://code.claude.com/docs/en/mcp)); both paths work with lazymark. The tools are the commands above:

| Tool | Arguments | Same as |
|---|---|---|
| `list_notes` | | `note list` |
| `read_note` | `path` | `note show` |
| `create_note` | `title`, `folder?`, `empty?` | `note new` |
| `search_notes` | `query`, `regex?`, `case_sensitive?`, `limit?` | `search` |
| `list_tasks` | `pending_only?`, `column?`, `note_path?` | `task list` |
| `move_task` | `id`, `column` | `task move` |
| `set_task_date` | `id`, `field` (`start` or `due`), `date` (`YYYY-MM-DD` or `none`) | `task due` and `task start` |
| `toggle_task` | `id` (or `path` and `line`) | `task toggle` |
| `get_kanban` | | the board: columns in order, each with its cards |

An error comes back as a tool result with `isError: true` and the exit code in the text.

### Register it in Claude Code

```
claude mcp add --transport stdio lazymark -- lazymark mcp
```

Add `--scope user` to have it in all your projects, or `--scope project` to share it through `.mcp.json`. `claude mcp list` shows it and `/mcp` checks it inside Claude Code. The syntax is from the [Claude Code MCP documentation](https://code.claude.com/docs/en/mcp). Use `-- lazymark mcp --dir <folder>` for a notes folder other than the default.

## Dates

A task can carry up to five dates at the end of its line, in one of the two formats of [Obsidian Tasks](https://publish.obsidian.md/tasks/Reference/Task+Formats/About+Task+Formats) (plain markdown; GitHub shows them as text). lazymark **always reads both**, also mixed in one line:

| Field | Emoji format | Dataview format |
|---|---|---|
| start | `🛫 2026-05-01` | `[start:: 2026-05-01]` |
| due | `📅 2026-05-10` | `[due:: 2026-05-10]` |
| completed | `✅ 2026-05-09` | `[completion:: 2026-05-09]` |
| scheduled | `⏳ 2026-05-02` | `[scheduled:: 2026-05-02]` |
| created | `➕ 2026-04-30` | `[created:: 2026-04-30]` |

In the Dataview format parentheses (`(due:: 2026-05-10)`) and spaces around `::` are also read (field names are lowercase). Only a real calendar date counts; an invalid one stays in the text. If a field repeats, the first valid date is the one that counts. The id of a task does not change when its dates change. Obsidian Tasks reads and writes one format at a time ([its documentation](https://publish.obsidian.md/tasks/Reference/Task+Formats/About+Task+Formats): "Tasks only supports reading and writing one format at a time"): choose **Task format: Dataview** in its settings if you use the default format of lazymark.

**Which format lazymark writes** (`date_format` in the config, or Settings → Date format): `"dataview"` or `"emoji"`; by default Dataview, except in a vault that already has tasks with emoji dates and none with Dataview, where it writes emoji and says so once (`date_format_notice_shown` remembers it). A line you edit keeps the format its dates already have (a new field uses the format of the other dates of that line); the completion date added when you tick a task follows the same rule.

### `lazymark dates migrate`

```
lazymark dates migrate --to dataview|emoji [--dry-run] [--json] [--dir <folder>]
```

Moves the dates of every task to the chosen format. It is never automatic. It only touches task lines (not paragraphs or code blocks), keeps the rest of the line and the line endings, leaves invalid dates alone and writes each note atomically (a note that changed while migrating is not written: exit code 4). `--dry-run` prints the change (`-` before, `+` after) without writing anything; `--json` gives `{to, dry_run, notes, lines, changes: [{note, line, before, after}]}`. It is idempotent: a second run changes nothing. After migrating to Dataview, set **Task format: Dataview** in Obsidian Tasks.

Rules for edge cases: a field written as `(due:: …)` is kept with parentheses while you edit the line, but migrating normalises it to square brackets (what Obsidian Tasks writes), so `(due:: …)` → emoji → Dataview comes back as `[due:: …]` with the same dates. `[due:: 2026-05-10](https://…)` (a Markdown link), `[[due:: …]]` (a wikilink), `[due:: …][ref]` / `[due:: …][]` (reference links) and anything inside inline code are text, not fields. As in CommonMark, inline code needs a closing run of the same number of backticks; a lone backtick is plain text. If a field is repeated the first valid one wins; the others are also hidden from the task text, and editing that field replaces the first and removes the repeats. A value that is not a date (`[due:: tomorrow]`) is not read as a date, stays in the text, and editing that field replaces it (a line never ends up with two `due`). If a note cannot be written, the command lists the notes already migrated and exits with a non-zero code; running it again continues where it stopped.

`task start` and `task due` warn on stderr (exit code 0) when the start ends up after the due date.

## Kanban format

The column of a task is a tag at the end of its line: `- [ ] write report #kb/doing`. No tag means the first column; `[x]` is always the done column; a `#kb/…` tag that is not one of your columns counts as the first column and is left alone until you move the card. Only `[ ]` and `[x]` are ever written. The old `#doing`, `#wip`, `#progreso` and `#in-progress` tags are read as `doing` and replaced by `#kb/doing` only when you move that card.
