# Templates and the daily note

## Templates

A template is a note in the `templates/` folder of your notes folder: `templates/meeting.md`. New notes can start from it:

- In the app, `C` (in the Notes panel) lists the templates, then asks for the name of the new note.
- In the shell, `lazymark note new "Weekly meeting" --template meeting`.

Three variables are replaced, everywhere they appear:

| Variable | Replaced by | Example |
|---|---|---|
| `{{date}}` | today's date | `2026-10-03` |
| `{{time}}` | the current time | `09:05` |
| `{{title}}` | the title you typed | `Weekly meeting` |

The names do not care about case (`{{Date}}` and `{{TITLE}}` work), and there are no spaces inside the braces (`{{ title }}` is not a variable). Any other `{{...}}`, such as `{{fecha}}`, is **not** replaced: it stays as written and lazymark warns about it (`Unknown variable: {{fecha}}` in the status bar, `warning: unknown variable: {{fecha}}` on stderr and a `warnings` list in the `--json` output); the note is created anyway. A title that contains `{{date}}` is not expanded a second time. An empty template gives a note with just its title. Only the notes directly inside `templates/` are templates (not its subfolders or hidden files), and a symlink that leaves the notes folder is not one.

A template that is not UTF-8 text (a NUL byte, UTF-16, Latin-1…) or is bigger than 256 KB is refused: the note is not created, nothing is written (not even `journal/`), the status bar says why, and the command line exits with 2. A UTF-8 byte order mark at the start is dropped.

A template is still a note, so you see and edit it like any other, but its checkboxes and `#tags` do not count: they do not show up in the Tasks panel, the Kanban board or Categories.

## The daily note

`T` (anywhere in the app) and `lazymark daily` create today's note, `journal/YYYY-MM-DD.md`, or open it if it already exists; an existing note is never touched. The first time it is made from `templates/daily.md` (with the variables above; `{{title}}` is the date) and, without that template, it is just `# YYYY-MM-DD`. The `journal/` folder is created if it is missing, and always inside the notes folder (a `journal` symlink that points outside is refused).

```markdown
# Daily {{date}}

## Plan
- [ ] 

## Notes
```
