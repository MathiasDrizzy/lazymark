# Paste images from your editor

Copy an image (a screenshot, or an image file in the file manager), open a note in micro, vim or GNU nano and press one key: the editor inserts `![](assets/…)` at the cursor, and the image is saved in the `assets/` folder next to the note. lazymark then shows it inline.

It works through `lazymark paste`, which any editor can call:

```sh
lazymark paste [--no-newline] [<note.md>]
```

- It saves the copied image in `<note's folder>/assets/<note>-<date>-<time>.<ext>` (a copied image file is copied; the original is not touched) and prints only `![](assets/…)` on stdout.
- Without a note it uses `$LAZYMARK_NOTE`, which lazymark defines when it opens your editor with `e` or `Enter`. It also puts its own folder in the editor's `PATH`, so the plugins find `lazymark` without any setup.
- If nothing is copied it exits with an error, prints the reason on stderr and nothing on stdout, so the editor inserts nothing.
- On macOS it reads the clipboard with `pngpaste` if you have it, or with `osascript`; on Linux with `wl-paste` or `xclip`; on Windows with PowerShell.

## Install

```sh
lazymark editor-plugins install            # micro, vim and nano
lazymark editor-plugins install micro      # or only one of them
lazymark editor-plugins uninstall          # takes everything back
```

Installing twice is the same as installing once. It never overwrites a file of yours that is not lazymark's, and `uninstall` leaves your configuration as it was.

| Editor | What is written | Key | Why this key |
|---|---|---|---|
| micro | The plugin `plug/lazymark/` in micro's configuration folder (`$MICRO_CONFIG_HOME`, or `$XDG_CONFIG_HOME/micro`, or `~/.config/micro`). | `Alt-i`, or the command `pasteimage` (`Ctrl-e`, then `pasteimage`) | micro has no default binding for `Alt-i`, and the plugin binds it with `TryBindKey(…, false)`, which never replaces a binding you already have. |
| vim | A native package, `~/.vim/pack/lazymark/start/lazymark/plugin/lazymark.vim`. | `<Leader>ip` in normal mode (`\ip` by default), or `:LazymarkPaste` | The mapping is only created if `<Leader>ip` is free, and you can remap it with `nmap <Leader>x <Plug>(lazymark-paste)`. The image goes after the character under the cursor, like `p`. |
| GNU nano | A block between `# >>> lazymark` and `# <<< lazymark <<<` in your `nanorc` (`~/.nanorc`, or `~/.config/nano/nanorc`). | `Alt-7` | `bind` is not allowed in an included file, so the block goes in your `nanorc`; the rest of the file is not touched. nano has no default `M-7`, and the install refuses if your `nanorc` already binds it. |

On macOS terminals, `Alt` needs "Use Option as Meta" (Terminal and iTerm2) or `macos-option-as-alt = true` (Ghostty).

## Notes about nano

- The `nano` that ships with macOS is UW Pico 5.09: it has no `bind`. Install GNU nano with `brew install nano`.
- nano does not know the file it is editing from a command, so the key needs `$LAZYMARK_NOTE`: open the note from lazymark. If you start nano by hand, export `LAZYMARK_NOTE=/path/to/note.md` first.
- The command runs with `2>/dev/null` because nano inserts everything a command writes. If there is no copied image, nothing is inserted.
- When nano runs as root it reads the home folder from `/etc/passwd`, not `$HOME`.

## References

- micro plugins: <https://github.com/zyedidia/micro/blob/master/runtime/help/plugins.md>
- vim packages: <https://vimhelp.org/repeat.txt.html#packages> and `:help write-plugin`
- nano `bind`, `execute`: <https://www.nano-editor.org/dist/latest/nanorc.5.html>
