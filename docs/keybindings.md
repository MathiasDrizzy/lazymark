# Keybindings

<!-- Generated from the keymap by `go test ./internal/app -run TestKeybindingsDoc -update`. Do not edit by hand. -->

Press `?` inside lazymark to see the keys of the panel you are in. The same list is generated from the same keymap, so it never disagrees with this page.

`h` `j` `k` `l` `g` `G` `Ctrl+U` and `Ctrl+D` are Vim-style shortcuts. They work by default and can be turned off in Settings (Keybindings: `Lazy`). Nothing in lazymark needs Vim modes.

## Global

Work everywhere.

| Keys | Action |
|---|---|
| `q` `Ctrl+C` | Quit |
| `?` | Keybindings |
| `Ctrl+V` | Paste image |
| `,` | Settings |
| `x` | Trash |
| `/` | Search |
| `1` | Notes panel |
| `2` | Tasks panel |
| `3` | Categories panel |
| `4` | Preview panel |
| `Tab` | Next panel |
| `Shift+Tab` | Previous panel |
| `w` | Zoom panel |
| `W` | Kanban board |
| `[` | Narrow column |
| `]` | Widen column |
| `Esc` | Clear filter |

## Navigation

Move around lists, the preview and popups.

| Keys | Action |
|---|---|
| `↑` `k` | Up |
| `↓` `j` | Down |
| `←` `h` | Left |
| `→` `l` | Right |
| `Home` `g` | Top |
| `End` `G` | Bottom |
| `PgUp` `Ctrl+U` | Page up |
| `PgDn` `Ctrl+D` | Page down |

## Notes panel

Panel `[1]`: the tree of folders and notes.

| Keys | Action |
|---|---|
| `Enter` | Open |
| `e` | Edit |
| `c` | New note |
| `F` | New folder |
| `r` | Rename |
| `m` | Move |
| `d` | Delete |
| `v` | Select |
| `V` | Select all |

## Tasks panel

Panel `[2]`: the checkboxes found in your notes. The preview follows the selected task.

| Keys | Action |
|---|---|
| `Enter` | Open note |
| `Space` | Toggle task |
| `H` | Hide done |
| `f` | Filter tasks |

## Categories panel

Panel `[3]`: the `#tags` of your notes.

| Keys | Action |
|---|---|
| `Enter` | Filter notes |

## Preview panel

Panel `[4]`: the rendered note.

| Keys | Action |
|---|---|
| `Enter` `e` | Edit |
| `n` | Next link |
| `N` | Previous link |

## Kanban board

Opened with `W`. Cards are the tasks of your notes.

| Keys | Action |
|---|---|
| `←` `h` | Previous column |
| `→` `l` | Next column |
| `H` `shift+←` | Move left |
| `L` `shift+→` | Move right |
| `Space` | Toggle task |
| `Enter` `e` | Edit |
| `Esc` `W` | Back to notes |

## Lists inside popups

Settings, move and the folder picker.

| Keys | Action |
|---|---|
| `Enter` | Accept |

## Trash popup

Opened with `x`.

| Keys | Action |
|---|---|
| `r` | Restore |
| `d` | Delete forever |
| `D` | Empty trash |

## Confirmation popups

| Keys | Action |
|---|---|
| `y` `Enter` | Confirm |

## Mouse

- Click a note, task, tag, popup row or footer hint to select or run it.
- Click a task checkbox to toggle it. Click a tag to filter the notes tree.
- Drag the divider between the two columns to resize them.
- The scroll wheel scrolls the panel under the pointer.
- In popups the first click selects a row and the second click on a setting changes it. A click outside a popup does nothing; `Esc` closes it.

## Pasting images

- `Ctrl+V` saves the image on your clipboard into the `assets/` folder next to the note and adds `![](assets/…)` at the end of the note (or below the selected task when the Tasks panel is focused).
- Pasting a copied image file (for example with `Cmd+V` after copying it in Finder) imports that file the same way.

## Changing keys

Some actions can be rebound in the `keybindings` section of the configuration file. See [configuration.md](configuration.md).
