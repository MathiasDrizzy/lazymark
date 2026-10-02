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

## Notes folder

- Open lazymark with `--dir <folder>` to use a folder for that run only. Changing a
  setting during that run does **not** make the folder your default.
- Choose the folder from Settings (`Notes folder`): a folder picker opens, `Enter`
  goes into a folder, `..` goes up and `s` picks the current one. The tree reloads
  at once and the choice is saved as your default.

Notes are plain markdown files. lazymark only rewrites the line you change (toggling
a task) or appends to the end of the note (pasting an image), and it refuses to write
if the note was modified by another program since it loaded it.

What lazymark reads from your notes:

- **Tags**: any `#word`. They fill the Categories panel.
- **Tasks**: lines that start with `- [ ] something` or `- [x] something` (also with `*`). Checkboxes inside an indented list are not picked up yet.
- **Kanban column**: a task is *In Progress* when its line contains `#doing`, `#wip`,
  `#progreso` or `#in-progress`; `[x]` is *Done*; everything else is *To Do*.
- **Images**: `![alt](assets/picture.png)` shows the picture in the preview on
  terminals that support the Kitty graphics protocol.

## Minimal example

```json
{
  "theme": "tokyo-night",
  "editor": "code --wait",
  "popup_background": "none",
  "task_scope": "tag:work"
}
```

## All settings

| Key | Values | Default | What it does |
|---|---|---|---|
| `notes_dir` | a folder path | `~/Documents/notes` | The notes folder. |
| `editor` | a command, with arguments if you want | `$EDITOR`, else `micro`, `vim` or `nano` | Opened with `Enter` or `e`. A path with spaces works. |
| `theme` | `catppuccin-mocha`, `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `tokyo-night`, `gruvbox-dark`, `nord`, `dracula`, `one-dark`, `rose-pine`, `kanagawa`, `everforest-dark`, `solarized-dark`, `solarized-light` | `catppuccin-mocha` | Colors of the whole interface, including the markdown preview and its code blocks. Changes live in Settings. |
| `language` | `auto`, `en`, `es` | `auto` | `auto` follows `LANG`, `LC_ALL` and `LC_MESSAGES`: Spanish if they mention `es`, otherwise English. |
| `screen_background` | `theme`, `terminal` | `theme` | `theme` paints the whole screen with the theme's base color (panels, gaps, bottom bar, popups, Kanban and preview). `terminal` leaves your terminal's background, so a translucent terminal stays translucent. Changes live in Settings ("Screen background"). |
| `popup_background` | `none`, `theme` | `none` | `none` leaves the terminal background behind popups, so a translucent terminal stays translucent. `theme` paints the theme's base color. Every popup uses the theme palette either way. |
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
| `lazymark task list [--json] [--pending]` | List the tasks found in your notes. |
| `lazymark task toggle --path <note> --line <n>` | Toggle one task. |
| `lazymark note list [--json]` | List the notes. |
| `lazymark note get <path>` | Print a note. |
| `lazymark paste [--no-newline] [<note.md>]` | Save the image you have copied (a screenshot or an image file) in the note's `assets/` folder and print `![](assets/…)`. Without a note it uses `$LAZYMARK_NOTE`. Exits with an error and prints nothing if there is no image. |
| `lazymark editor-plugins install\|uninstall [micro\|vim\|nano]` | Add or remove the plugins that paste images from micro, vim and nano. See [editor-plugins.md](editor-plugins.md). |
| `lazymark mcp` | Start an MCP server over stdio (`list_notes`, `read_note`, `list_tasks`, `toggle_task`, `get_kanban`). |

`task`, `note` and `mcp` accept `--dir <folder>`.
