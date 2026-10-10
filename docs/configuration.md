# Configuration

lazymark works without any configuration. Almost everything can be changed from
**Settings** (press `,`) and is saved immediately; this page is the reference for
the file behind it.

## Where things live

| What | Where |
|---|---|
| Notes | `~/Documents/notes` by default (any folder works; see below) |
| Configuration | `config.json` in your user configuration folder (table below) |
| Images | an `assets/` folder next to the note that uses them |
| Trash | a hidden `.trash/` folder inside the notes folder; items are deleted for good after 20 days |

The configuration file is created the first time lazymark saves a setting:

| System | Path |
|---|---|
| macOS | `~/Library/Application Support/lazymark/config.json` |
| Linux | `$XDG_CONFIG_HOME/lazymark/config.json`, or `~/.config/lazymark/config.json` |
| Windows | `%AppData%\lazymark\config.json` |

You can edit the file by hand while lazymark is closed. A missing field keeps its
default, so a file with only the settings you care about is fine.


### If `config.json` has a mistake

lazymark never throws your configuration away because of a typo. Each key is read on its own: a value of the wrong type (for example `"trash_days": "20"`) goes back to its default, a warning says which one (on the status bar, and on stderr in the commands), and **the rest of the file is respected**. A key lazymark does not know (a typo, or an option from a newer version) is kept as it is when the file is saved. If the file is not valid JSON, or cannot be read, lazymark uses the defaults and, before it ever writes over it, keeps what was there as `config.json.bak` (`.bak.1`, `.bak.2`… if there is already one); if it cannot make that copy it does not write at all.

## Notes folder

- Open lazymark with `--dir <folder>` to use a folder for that run only. Changing a
  setting during that run does **not** make the folder your default.
- Choose the folder from Settings (`Notes folder`): a folder picker opens, `Enter`
  goes into a folder, `..` goes up and `s` picks the current one. The tree reloads
  at once and the choice is saved as your default.

Relative folders (including `.`) are resolved against the working directory when
the store is opened. Absolute paths and symlinked vault roots also work; note and
task IDs stay relative to the vault and do not depend on how its root was spelled.
For a home-relative command-line path, let the shell expand `~` (for example,
`lazymark --dir ~/notes`); a literal `~` is not expanded by storage.

Notes are plain markdown files. lazymark only rewrites the line you change (toggling
a task) or appends to the end of the note (pasting an image), and it refuses to write
if the note was modified by another program since it loaded it.

What lazymark reads from your notes:

- **Tags**: inline `#word` at line start or after whitespace. Code fences, inline
  code, link destinations and frontmatter are excluded; nested tags use their
  first segment as the category. See [tags.md](tags.md) for the full rules.
- **Tasks**: list items that start with a checkbox, `- [ ] something` or `- [x] something`. The bullet can be `-`, `*`, `+` or a number (`1.`, `1)`), and the item can be nested at any depth (spaces or tabs, also under an item with no checkbox). Moving, ticking or dating a nested task rewrites only its line and keeps its indentation, and its id does not change. Checkboxes inside a code block (fenced with ``` or ~~~, or indented 4 spaces outside a list) and inside a quote (`> - [ ]`) are not tasks.
- **Card order**: the cards of a column follow the order of the notes (the board lists the most recently modified note first, so a note you just edited or reordered goes to the top) and, inside a note, the order of its lines. `K`/`J` (or `Shift+↑`/`Shift+↓`) and a vertical drag reorder by swapping the two task lines of the same note (with their subtasks; only sibling tasks, the same indentation); across different notes the order is the notes' order and cannot be changed from the board.
- **Kanban column**: a tag at the end of the task line, `- [ ] task #kb/doing`. No tag is the first
  column and `[x]` is always the done column. The old `#doing`, `#wip`, `#progreso` and `#in-progress`
  tags are still read as *doing* and are replaced when you move the card. See [cli.md](cli.md).
- **Images**: `![alt](assets/picture.png)` shows the picture in the preview on
  terminals that support the Kitty graphics protocol.

## Minimal example

```json
{
  "theme": "tokyo-night",
  "editor": "code --wait",
  "popup_background": "theme",
  "task_scope": "tag:work"
}
```

## All settings

| Key | Values | Default | What it does |
|---|---|---|---|
| `notes_dir` | a folder path | `~/Documents/notes` | The notes folder. |
| `editor` | a command, with arguments if you want | `$EDITOR`, else `micro`, `vim` or `nano` | Opened with `Enter` or `e`. A path with spaces works. |
| `theme` | `catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, `nord`, `dracula`, `one-dark`, `rose-pine`, `kanagawa`, `everforest-dark`, `solarized-dark`, `solarized-light` | `catppuccin-mocha` | Colors of the whole interface, including the markdown preview and its code blocks. Changes live in Settings. |
| `kanban_columns` | a list of 2 to 6 columns: `{"id": "doing", "title": "Writing"}` or `{"id": "doing", "titles": {"en": "Doing", "es": "En curso"}}` | `todo`, `doing`, `done` | The Kanban columns, in order. `id` is lowercase letters, digits, `_` or `-` and goes in the `#kb/<id>` tag; the title is free, per language, or (if missing) the default one for `todo`/`doing`/`done`, else the id. The column called `done` (or the last) holds the finished tasks. An invalid list goes back to the default. |
| `language` | `auto`, `en`, `es`, `pt`, `fr`, `de`, `it`, `ja`, `zh` | `auto` | `auto` follows `LC_ALL`, `LC_MESSAGES` and `LANG` (the first one that is set, as POSIX does): `pt_BR` is Portuguese, `zh_CN` Chinese, and any other language English. `pt` is Brazilian Portuguese and `zh` Simplified Chinese; `pt-BR` and `zh-CN` are accepted. Changes live in Settings ("Language"). Which translations were reviewed by a native speaker: [i18n.md](i18n.md). |
| `screen_background` | `theme`, `terminal` | `theme` | `theme` paints the whole screen with the theme's base color (panels, gaps, bottom bar, popups, Kanban and preview). `terminal` leaves your terminal's background, so a translucent terminal stays translucent. Changes live in Settings ("Screen background"). |
| `date_format` | `"dataview"`, `"emoji"` | (not set: Dataview, or emoji in a vault that only has emoji dates) | The format in which task dates are written to the file; both are always read. Changes live in Settings ("Date format"). `date_format_notice_shown` records that the one-time notice about an emoji vault was shown. See [cli.md](cli.md#dates). |
| `max_note_mb` | a whole number, 1 to 4096 | `10` | The biggest note lazymark reads (list, preview, CLI and MCP). A bigger one is listed as "too large" and not loaded, so a huge file cannot fill the memory. |
| `mascot` | `true`, `false` | `true` | Shows the sleeping sloth, small, at the bottom right of the preview when there is nothing to show (empty notes folder, empty folder, empty note). Click it and it wakes up, waves, dances or spins, then goes back to sleep. Changes live in Settings ("Mascot"). |
| `date_colors` | `true`, `false` | `true` | Draws the dates of a task in the color of their state, taken from the theme's palette: overdue (error color, bold), due soon (warning, bold), on time (accent), started, not started and completed (success). `false` draws every date in a neutral light gray. In the Kanban cards, the Tasks panel, the preview, the Dates popup and `lazymark task list` (colors only on a terminal, never with `--json`, in a pipe or with `NO_COLOR`). Settings: "Date colors". |
| `due_soon_days` | a whole number, 0 to 30 | `0` | A due date counts as "due soon" from today until N days ahead. `0`: only the same day. Settings: "Due soon (days before)". |
| `date_color_names` | an object: `overdue`, `soon`, `ontime`, `started`, `notstarted`, `done`, each one of `error`, `warning`, `orange`, `success`, `accent`, `info`, `special`, `text`, `muted` | `error`, `warning`, `accent`, `info`, `muted`, `success` | The palette color of each state. The defaults are replaced by the next palette color that is readable on the theme's background when the default is too pale (for example the yellow of the light themes becomes orange); a color you choose is always respected. Settings: "Color: overdue", "Color: due soon", and so on. |
| `click_hint` | `true`, `false` | `true` | The very faint `click me!` above the mascot after about 20 seconds without touching anything (15 seconds, then again every 60 seconds while you stay away) (and never after your first click on it). `false` turns it off and keeps the mascot. Settings: "\"click me!\" hint". |
| `date_warnings` | `true`, `false` | `true` | The warning that the start date of a task is after its due date (Dates popup and `task start` / `task due`). It never blocks anything. Settings: "Start > due warning". |
| `date_format_notice` | `true`, `false` | `true` | The one-time notice that says dates are written with emojis because the vault already has them. `false` never shows it. Settings: "Date format notice". |
| `click_hint_idle_seconds`, `click_hint_show_seconds`, `click_hint_every_seconds` | whole numbers, 5 to 600 | `20`, `15`, `60` | The rhythm of the faint `click me!`: seconds without touching anything before it shows, how long it stays, and how often it comes back while you stay away. "Shown" must be less than "every" (otherwise the three go back to their defaults). Settings: `"click me!": idle / shown / every`. |
| `trash_days` | a whole number, 0 to 365 | `20` | How many days what you delete stays in the trash. Each item keeps the number of days it had when it was deleted: changing `trash_days` (in Settings or by editing the file) **never purges what is already in the trash**, and 0 does not empty it. `0` means **no trash** for what you delete from then on: it removes the note or folder for good, and lazymark always asks first (even with `confirm_delete` off, also for an empty folder or one with files that are not notes) and says it is permanent and how many files go. Settings: "Days in trash" (←/→ do not wrap from 365 to 0). |
| `daily_folder`, `daily_name`, `templates_folder` | a folder inside the notes folder; a name with `YYYY` (or `AAAA`), `MM` and `DD`; a folder | `journal`, `YYYY-MM-DD`, `templates` | Where the daily notes live, how they are named and where the templates are. Folders are relative to the notes folder: no `..`, no absolute path, no hidden or `assets` segment; an invalid value goes back to the default. Settings: "Daily notes folder", "Daily note name", "Templates folder" (type the value). |
| `kanban_tag` | a letter and up to 19 letters, digits, `-` or `_` | `kb` | The prefix of the board's column tag: `#kb/doing`. **Changing it does not migrate the notes you already have** (their `#kb/…` stop being board tags): run `lazymark kanban retag --from kb --to <new>` (see [cli.md](cli.md)). Settings: "Board tag". |
| `notes_sort`, `tasks_sort` | `name`, `modified`; `note`, `due` | `name`, `note` | The order of the Notes tree (`modified`: the most recent notes first; folders stay by name) and of the Tasks panel (`due`: pending tasks with a due date first, from the most overdue to the farthest, then those without a date, and the finished ones last). Settings: "Notes order", "Tasks order". |
| `date_glyphs` | an object with `start`, `due`, `done`, `scheduled`, `created`, each one a single character 1 or 2 cells wide | (not set: the Nerd Font or text glyphs) | Your own glyph for a date field, on screen and in the text output of the CLI; what is missing or invalid uses the default. Settings: "Date glyphs" (five glyphs separated by spaces, `-` for the default). |
| `nerd_font` | `true`, `false` | `true` | The dates of the tasks are written in the note as emoji (start, due, done: the Obsidian Tasks format) but drawn on screen with monochrome Nerd Font glyphs in the theme's colors (overdue in the error color, done in the success color, the rest dimmed), never as color emoji. With `false` they are drawn with text symbols (`▸` start, `◷` due, `✓` done) for terminals without a Nerd Font. |
| `kanban_cards` | `cards`, `compact` | `cards` | `cards` draws each Kanban task as a rectangle with its text (up to 2 lines), its note and its dates; `compact` is one row per task. The board also falls back to compact when the columns are too narrow or the terminal too short for cards. Changes live in Settings ("Cards"). |
| `popup_background` | `theme`, `terminal` | `theme` | The same two values as `screen_background`. `terminal` leaves your terminal's background behind popups, even when the screen is painted with the theme, so a translucent terminal stays translucent. `theme` paints the theme's base color, like the screen. Every popup uses the theme palette either way and covers all its cells. The old value `none` is read as `terminal`. |
| `task_scope` | `all`, `tag:<tag>`, `folder:<folder>` | `all` | Which notes feed the Tasks panel: all of them, the ones with a tag, or the ones inside a folder. |
| `hide_completed_tasks` | `true`, `false` | `false` | Hide finished tasks. Toggled with `H` in the Tasks panel. |
| `confirm_delete` | `true`, `false` | `true` | Ask before moving notes to the trash. Folders with content always ask. |
| `keybinding_mode` | `dual`, `lazy` | `dual` | `dual` (shown as `Lazy + Vim` in Settings) keeps the Vim-style `h` `j` `k` `l` `g` `G` next to the arrows; `lazy` turns them off. A `lazygit` value written by v0.1.0 is read as `lazy`. |
| `mouse_click` | `true`, `false` | `true` | Mouse support. `--no-mouse` turns it off for one run. |
| `sidebar_ratio` | `0.15` to `0.75` | `0.33` | Width of the left column. Drag the divider or press `[` and `]`. |
| `show_tasks_tab` | `true`, `false` | `true` | Show the Tasks panel. |
| `show_tags_tab` | `true`, `false` | `true` | Show the Categories panel. |
| `keybindings` | see below | | Rebind some actions. |

### Rebinding keys

The `keybindings` object accepts one key per action. Anything you leave out keeps its
default:

```json
{
  "keybindings": {
    "new_note": "n",
    "delete": "D",
    "quit": "Q"
  }
}
```

Available names: `new_note`, `new_folder`, `edit`, `delete`, `move`, `paste_image`,
`toggle_panel`, `settings`, `cheatsheet` and `quit`. Keys are written as lazymark
shows them (`a`, `F`, `ctrl+v`, `tab`, `?`). The in-app list (`?`) and
[keybindings.md](keybindings.md) describe the defaults.

## Command line

```text
lazymark [options] [command]

Options:
  --dir <folder>    notes folder for this run
  --theme <name>    color theme for this run
  --no-mouse        disable mouse input
  --version         print the version
```

Commands that run without the interface, for scripts and other tools:

| Command | What it does |
|---|---|
| `lazymark note list\|show\|new` | List the notes, print one, create one. |
| `lazymark search <text>` | Full-text search in all the notes (`--regex`, `--case`, `--limit`, `--json`). |
| `lazymark task list\|toggle\|move` | List the tasks, tick one, move one to a Kanban column. |
| `lazymark paste [--no-newline] [<note.md>]` | Save the image you have copied (a screenshot or an image file) in the note's `assets/` folder and print `![](assets/…)`. Without a note it uses `$LAZYMARK_NOTE`. Exits with an error and prints nothing if there is no image. |
| `lazymark editor-plugins install\|uninstall [micro\|vim\|nano]` | Add or remove the plugins that paste images from micro, vim and nano. See [editor-plugins.md](editor-plugins.md). |
| `lazymark mcp` | Start an MCP server over stdio with the same operations. |

`task`, `note` and `mcp` accept `--dir <folder>`. The arguments, the `--json` schema, the exit codes and how to register the MCP server in Claude Code are in [cli.md](cli.md).
