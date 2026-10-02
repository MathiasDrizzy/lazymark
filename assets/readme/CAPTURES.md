# How the README media is made

| File | Made with | Source |
|---|---|---|
| `main.gif`, `feature-notes.gif`, `feature-tasks.gif`, `feature-categories.gif`, `feature-kanban.gif`, `feature-settings.gif` | [VHS](https://github.com/charmbracelet/vhs) | `tapes/*.tape` |
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

Only dark themes appear in the recordings: lazymark does not paint a background of its own, so
the light Latte theme is unreadable on the recorder's dark terminal.

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
