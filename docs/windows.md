# lazymark on Windows

Windows is built and tested on every change, but the author does not use it interactively. This page says what is checked automatically, and gives a checklist to check by hand the things a test cannot see. If you try it, a report in an issue (what worked, what did not, your terminal and Windows version) is the most useful thing you can send.

## What is checked automatically

On every push, the CI runs on `windows-latest`: `go vet ./...` and `go test -race ./...`, and a cross build for `windows`. The tests cover the notes folder rules (paths with `\`, `C:` drive letters, symlinks that leave the folder), the Kanban and task files, the CLI and MCP output (golden files with normalised paths), and the clipboard parsing.

What the CI **cannot** see: the real terminal (Windows Terminal, conhost, a PTY), the mouse, colours, the clipboard, an editor opening, and symlink privileges.

## Manual checklist

Use Windows Terminal (or another terminal with ANSI, mouse and true colour) and a fresh folder, never your real notes: `lazymark --dir C:\temp\notes-test`.

1. **Starts.** `lazymark --version` prints a version; `lazymark --dir C:\temp\notes-test` opens the three panels with no garbled borders.
2. **Notes.** `c` creates a note (`Nueva nota` / `New note`), `Enter` opens it in the preview, `r` renames it, `d` and `x` use the trash. The file shows up in Explorer with the name you typed.
3. **Mouse.** Click a note, scroll with the wheel, drag the divider between the columns, click a key in the bottom bar.
4. **Tasks and Kanban.** Write `- [ ] test #kb/doing` in a note, press `W`, move the card with `H`/`L` and with the mouse. Reorder two cards with `K`/`J`. Open the file in Notepad and check that only that line changed.
5. **Links and search.** `[[other note]]` is underlined, `n` then `Enter` follows it, `/` searches, renaming a linked note offers to update the links.
6. **Templates and daily note.** Put `templates\daily.md` with `# {{date}}` in the notes folder: `T` creates `journal\YYYY-MM-DD.md`; `lazymark daily` prints its path.
7. **Editor.** `e` opens the note in the `editor` setting, or `EDITOR`, or the first of `micro`, `vim` and `nano` found on the `PATH` (see [configuration.md](configuration.md)); on a bare Windows set it yourself, for example `"editor": "notepad"`. Check that the terminal comes back intact when you quit the editor.
8. **Paste an image.** Copy an image (Snipping Tool) and press `Ctrl+V` in a note: a file appears in an `assets\` folder next to the note and the note gets the image line. It uses PowerShell (`Windows.Forms`); if it fails the status bar says why.
9. **CLI.** In `cmd` and PowerShell: `lazymark task list --json`, `lazymark note new "x" --dir C:\temp\notes-test`, `lazymark search text`. Paths print with `\`; exit codes follow `docs/cli.md` (`echo %errorlevel%`).
10. **Config.** Change a setting (`,`): `%AppData%\lazymark\config.json` appears with only what changed.
11. **Images in the preview.** Windows Terminal does not draw the Kitty graphics protocol: an image must show as `[image: name.png]`, not as garbage.
12. **Symlinks.** Creating one needs Developer Mode or an administrator; a link that points outside the notes folder must not be read or written (test only if you can create one).

Write down anything that differs, with a screenshot of the terminal when it is visual.
