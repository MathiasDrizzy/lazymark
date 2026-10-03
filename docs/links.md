# Links between notes

lazymark understands the wikilinks of [Obsidian](https://obsidian.md/help/links): `[[note]]`, `[[note|alias]]`, `[[note#Heading]]`, `[[note#^block]]`, `[[#Heading]]` (a heading of the same note) and `[[folder/note]]`. The `.md` extension is optional (`[[note]]` and `[[note.md]]` are the same). Embeds (`![[file]]`), escaped links (`\[[note]]`), empty ones and anything inside inline code or a fenced code block are not links.

## Which note a link points to

The name is compared without caring about case, against the file name and, if that fails, against the title (so `[[My Note]]` finds `my-note.md`). A link with a `/` is compared against the path from the notes folder, or its tail (`[[projects/plan]]`, `[[b/deep]]` for `a/b/deep.md`). With several candidates, the one in the same folder as the note that has the link wins, then the shortest path, then alphabetical order. The `#` of `[[note#Heading]]` is not a category.

## In the preview

Links show their text (the alias if there is one) underlined: blue if the note exists, peach if it does not. With the preview focused, `n` and `N` select the next and the previous link; `Enter` follows the selected one and `Esc` lets go of it; a click on a link follows it too (`e` still edits the note). Following `[[note#Heading]]` scrolls to that heading. Following a link to a note that does not exist asks whether to create it (next to the note that has the link).

The end of the preview lists the lines of other notes that link to the one you are reading ("Links to this note"), one entry per line; they are links too, and following one goes to that line.

## Renaming

Renaming a note (`r`) that other notes link to shows the lines that would change, before and after, and lets you update them (`y`), just rename (`n`) or cancel (`Esc`). Only those lines are rewritten, and only if they are still what you saw: a line changed by another program in the meantime is left alone and counted as not updated. A link keeps its alias and its heading (`[[old#Part|text]]` becomes `[[new#Part|text]]`), and a path link changes only its last part. Renaming a folder does not touch links.

## Limits

Obsidian does not say whether names are case-sensitive or whether an alias may contain `|` (the specification of this feature follows what the documentation does say; an alias here is everything after the first `|`). Links are read from the notes in memory, so a very large notes folder makes the preview of a note with backlinks scan all of them; there is no persistent index.
