<div align="center">

<img src="assets/brand/lazymark-256.png" alt="lazymark mascot: a sloth hanging a pencil from its claws" width="128">

# lazymark

**Markdown notes, tasks and a Kanban board in your terminal, in the style of lazygit.**

Plain markdown files. Keyboard and mouse. Inline images.

[![CI](https://github.com/MathiasDrizzy/lazymark/actions/workflows/ci.yml/badge.svg)](https://github.com/MathiasDrizzy/lazymark/actions/workflows/ci.yml)
[![Go version](https://img.shields.io/github/go-mod/go-version/MathiasDrizzy/lazymark)](go.mod)
[![License: MIT](https://img.shields.io/github/license/MathiasDrizzy/lazymark)](LICENSE)

<img src="assets/readme/main.gif" alt="lazymark browsing a folder of notes, ticking a task, opening the key list, the Kanban board and the settings" width="100%">

[Install](#install) · [Quick start](#quick-start) · [Keys](#keys) · [Configuration](#configuration)

</div>

## Why lazymark

- **Your notes stay yours.** They are plain markdown files in one folder. Edit them with any editor; lazymark only rewrites the line you change and never overwrites a note that changed outside it.
- **Learn it by looking at it.** Panels, rounded borders and a bar of keys at the bottom, like lazygit. Press `?` for the keys of the panel you are in.
- **Keyboard or mouse.** Every action has a key. You can also click rows and the keys in the bottom bar, drag the divider between the columns and use the scroll wheel.
- **Tasks come from your notes.** The Tasks panel lists the checkboxes you already wrote, and ticking one changes only that line.
- **Safe by default.** Deleted notes go to a trash for 20 days, and folders with content always ask first.

## What it does

### Notes and folders

One tree for folders and notes. Create, rename, move and delete with a single key; the cursor lands on what you just created.

<img src="assets/readme/feature-notes.gif" alt="Creating a folder, a note inside it and renaming the note" width="100%">

### Tasks

Every `- [ ]` and `- [x]` in your notes, in one list. `Space` ticks a task and rewrites only its line; `H` hides the finished ones. The preview shows the note at the task's line.

<img src="assets/readme/feature-tasks.gif" alt="Ticking tasks in the Tasks panel and hiding the finished ones" width="100%">

### Categories

Every `#tag` in your notes. `Enter` on a tag shows only the notes that have it.

<img src="assets/readme/feature-categories.gif" alt="Filtering the notes tree by the design tag" width="100%">

### Kanban board

The same tasks as three columns: To Do, In Progress and Done. Move a card with `H` and `L`; the change is written back to the note.

<img src="assets/readme/feature-kanban.gif" alt="Moving cards between the columns of the Kanban board" width="100%">

### Inline images

Images in a note show up in the preview, in place, in terminals that support the Kitty graphics protocol. `Ctrl+V` pastes the image on your clipboard into the note, and pasting a copied image file imports it.

<img src="assets/readme/feature-images.png" alt="The preview of a note showing an architecture diagram between two paragraphs of text" width="100%">

### Settings and themes

Press `,`. Seven themes, changed live. Popups can keep your terminal's background (so a translucent terminal stays translucent) or use the theme's. You can also pick the notes folder here.

<img src="assets/readme/feature-settings.gif" alt="Cycling the color theme live and switching the popup background" width="100%">

## Install

Install a [Nerd Font](https://www.nerdfonts.com/) in your terminal for the folder and note icons. Building needs Go 1.27.1 or newer.

**With Go**

```sh
go install github.com/MathiasDrizzy/lazymark/cmd/lazymark@latest
```

**From source**

```sh
git clone https://github.com/MathiasDrizzy/lazymark.git
cd lazymark
go build -o lazymark ./cmd/lazymark
```

**Homebrew and prebuilt binaries:** coming soon.

## Quick start

1. Run `lazymark`. It opens `~/Documents/notes`, creating it if needed. Use `lazymark --dir <folder>` for another folder, or pick one later in Settings.
2. Press `c`, type a name and press `Enter` to create a note. Press `e` to write in your editor (`$EDITOR`, `micro` if it is not set).
3. Press `?` any time to see the keys.

## Keys

The essentials. `?` shows the keys of the panel you are in, and [docs/keybindings.md](docs/keybindings.md) has all of them.

<!-- keys:start -->
| Keys | Action |
|---|---|
| `↑` `↓` | Up / Down |
| `Tab` | Next panel |
| `1` `2` `3` `4` | Jump to a panel |
| `c` `F` | New note / New folder |
| `r` `m` `d` | Rename / Move / Delete |
| `Space` | Toggle a task (Tasks panel) |
| `W` | Kanban board |
| `x` | Trash |
| `?` `,` | Keybindings / Settings |
| `q` | Quit |
<!-- keys:end -->

## Configuration

Most things are in Settings (`,`) and are saved at once. They live in a `config.json` in your user configuration folder (`~/Library/Application Support/lazymark/` on macOS, `~/.config/lazymark/` on Linux, `%AppData%\lazymark\` on Windows). A file with only what you want to change is enough:

```json
{
  "theme": "tokyo-night",
  "editor": "code --wait",
  "popup_background": "none"
}
```

See [docs/configuration.md](docs/configuration.md) for every setting, the command line and the commands for scripts and other tools (`lazymark task list`, `lazymark mcp`, and more).

## Compatibility

| | Status |
|---|---|
| macOS | Developed and tested here, including the interface. |
| Linux | Builds, and its tests run on every change. Not used interactively by the author yet. |
| Windows | Builds, and its tests run on every change. Not used interactively by the author yet. |

| Terminal | Inline images |
|---|---|
| Ghostty | ✓ tested |
| Kitty, WezTerm | ✓ support the protocol, not tested |
| Any other | ✗ each image shows as `[image: name.png]` |

Limitations:

- Tasks are the lines that start with `- [ ]` or `- [x]`; checkboxes inside an indented list are not listed yet.
- The Kanban board has three fixed columns.
- Pasting an image from the clipboard needs `osascript` or `pngpaste` (macOS), `wl-paste` or `xclip` (Linux) or PowerShell (Windows).
- The terminal must be at least 60 columns by 20 rows.

## Contributing

Issues and pull requests are welcome. Run `go test ./...` before sending a change, and `git config core.hooksPath scripts/hooks` to run `gofmt`, `go vet` and the tests before every commit. CI runs on Linux, macOS and Windows.

The GIFs are made from the tapes in [assets/readme](assets/readme/CAPTURES.md), so they can be redone with `vhs`.

## License

[MIT](LICENSE)
