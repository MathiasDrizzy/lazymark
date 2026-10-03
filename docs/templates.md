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

Anything else, `{{other}}` included, stays as written, and a title that contains `{{date}}` is not expanded a second time. An empty template gives a note with just its title. Only the notes directly inside `templates/` are templates (not its subfolders or hidden files), and a symlink that leaves the notes folder is not one.

A template is still a note, so you see and edit it like any other, but its checkboxes and `#tags` do not count: they do not show up in the Tasks panel, the Kanban board or Categories.

## The daily note

`T` (anywhere in the app) and `lazymark daily` create today's note, `journal/YYYY-MM-DD.md`, or open it if it already exists; an existing note is never touched. The first time it is made from `templates/daily.md` (with the variables above; `{{title}}` is the date) and, without that template, it is just `# YYYY-MM-DD`. The `journal/` folder is created if it is missing, and always inside the notes folder (a `journal` symlink that points outside is refused).

```markdown
# Daily {{date}}

## Plan
- [ ] 

## Notes
```
