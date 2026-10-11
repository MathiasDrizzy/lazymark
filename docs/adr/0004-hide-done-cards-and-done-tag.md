# Hide finished cards and a done tag, in the Kanban board and the Tasks panel

* Status: accepted
* Date: 2026-10-11

## Context and problem statement

A board that accumulates finished cards becomes unusable: a user reported that "there can be too many piled up"
and asked for a way to hide the finished ones, and for a tag that puts a card away so it hides by itself, in the
Kanban board and in the pending tasks view, everything configurable. The Tasks panel already had `H`
(`hide_completed_tasks`); the board had nothing.

## Decision drivers

* Hiding must be view-only. The only write to a note is the tag the user asked for.
* The defaults must not change what existing users see.
* The user must always see that cards are hidden, so nothing looks lost.
* Reuse what exists: tags at the end of a task line (the same mechanism as `#kb/<column>`), the keymap table (it
  feeds the footer, the cheatsheet and the generated key docs) and the config normalization.

## Considered options

* A new column "Archive" for put-away cards (rejected: it changes the column model and the `#kb/<id>` contract).
* A frontmatter or sidecar file listing hidden tasks (rejected: a second source of truth that breaks when notes move).
* A configurable tag on the task line plus a per-view filter (chosen).

## Decision outcome

Chosen option: a configurable tag plus a filter applied while the board and the task list are built.

* Config: `hide_completed_kanban` (`false`), `done_tag` (`lm-done`, same shape as the Kanban tag prefix: a letter and
  up to 19 letters, digits, `-` or `_`) and `done_tag_hides` (`both` by default, `kanban`, `tasks` or `none`). Invalid
  values go back to the defaults. `hide_completed_tasks` keeps its meaning.
* Keys: in the board `o` hides or shows the finished cards (saved in the config; `x` was not used because it opens
  the Trash everywhere), `a` adds or removes the done tag on the selected card, `A` reveals what the tag hides for
  the session. In the Tasks panel `a` and `A` do the same and `H` stays. The board's top bar has buttons for `o`
  and `a` and the count of hidden cards; the Tasks panel title shows its count.
* The filter lives in `views.CollectKanbanHiding` and `views.CollectTasksHiding`, which take a predicate and return
  the number of hidden items. Indices into the board and the list stay consistent because hidden items are never in
  them, and the cursor is clamped after every reload.
* The tag is added or removed by `storage.ToggleTaskTag`, which rewrites only that line through the same
  atomic, mtime-checked path as moving a card (`ErrNoteChanged` if the note changed on disk).
* Defaults keep today's behaviour: finished cards are shown, and nothing carries the tag until the user adds it.

### Consequences

* Good: no new file format, no change to the note except the requested tag; the tag is a normal tag, visible in
  the Categories panel and searchable.
* Good: unit tests (`internal/storage`, `internal/config`, `internal/app`) and two `tuiqa` scenarios in
  `testdata/tuiqa/` that press the real keys in a pty; `verificar.sh` runs them.
* Bad: with `done_tag_hides` set to `both`, tagging a card in the Tasks panel also hides it in the board; the
  setting exists so people can choose.
* Bad: `A` does not undo the Pending filter of the Tasks panel (`H`/`f`); those are separate on purpose.
