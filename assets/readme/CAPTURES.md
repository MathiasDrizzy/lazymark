# How the README media is made

| File | Made with | Source |
|---|---|---|
| `main.gif`, `feature-notes.gif`, `feature-tasks.gif`, `feature-categories.gif`, `feature-kanban.gif`, `feature-settings.gif`, `feature-paste.gif` | [VHS](https://github.com/charmbracelet/vhs) | `tapes/*.tape` |
| `feature-images.png` | A real Ghostty window | the command below |

## GIFs

Run from the repository root. Each tape builds lazymark into a temporary folder, uses a
temporary `HOME` with a copy of `demo-home/` (invented notes, nothing personal) and fixed
note dates, so recording never touches real notes or writes inside the repository.

```sh
for t in assets/readme/tapes/*.tape; do vhs "$t"; done
```

All tapes share `tapes/lib/common.tape`: Catppuccin Mocha, JetBrainsMono Nerd Font (install it
first, the plain "JetBrains Mono" shows boxes instead of icons), 1200x700, font size 16, same
typing speed. `demo-env.sh` prepares the isolated environment.

lazymark paints the whole screen with the theme's base color (Screen background = theme), so light
themes such as Solarized Light are readable on the recorder's dark terminal and appear in the
recordings.

The dates of the demo tasks are written for 2026-10-06 and `demo-env.sh` moves them by as many days as separate that day from the day you record, so the colors of the dates (overdue in red, due today in yellow, on time in blue, started in teal, completed in green) look the same whichever day you record. `inbox.md` is the newest note, so its dated tasks come first in the Tasks panel.

The Reddit demo and the MCP demo are not part of the README: they live in `assets/social/` (`reddit-demo.tape` makes `reddit-demo.mp4` and `reddit-demo.gif`, 1280x720, font size 22 so it reads on a phone; `mcp-demo.tape` makes `mcp-demo.gif`). Record them from the repository root with `vhs assets/social/<name>.tape`.

`demo-env.sh` also fakes the clipboard (`pngpaste`, `wl-paste` and `osascript` stand-ins that "have
copied" the demo architecture diagram, so the real clipboard is never read) and installs the micro
plugin in the temporary `HOME`. VHS cannot send `Alt-i` to micro (it types an `i`), so
`feature-paste.tape` runs the plugin's `pasteimage` command, which `Alt-i` also runs. The demo uses the
fixed folder `/tmp/lazymark-demo`, because micro shows the full path of the note.

## Image screenshot

VHS renders with xterm.js, which does not draw Kitty graphics, so the inline-image screenshot
is a screenshot of a real terminal window (Ghostty, 120x35, Catppuccin Mocha, opaque
background). To redo it:

1. Copy `demo-home/` somewhere and start lazymark with that copy as `HOME` and
   `LANG=en_US.UTF-8`, so the interface is in English and your real notes stay untouched.
2. Press `down` six times: the cursor lands on `architecture.md`, whose preview shows
   `assets/architecture.png`.
3. Take a screenshot of the window and save it as `feature-images.png`.

The dates shown in that screenshot are the day it was taken, because copying the fixtures does
not keep their modification dates.
