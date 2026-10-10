# Tags and categories

Inline tags in notes fill the Categories panel and the `tags` field returned by
the CLI and MCP. Press `Enter` on a category to filter notes by it. Tags in task
text also contribute to the note's categories; templates do not (see
[templates.md](templates.md)).

## Inline tag rules

A tag starts with `#` at the beginning of a line or after whitespace. Its name
accepts Unicode letters and numbers, `_`, `-`, and `/` for nesting. A first
segment made entirely of numbers is ignored.

| Text | Category |
|---|---|
| `#Work #work` | `work` (once) |
| `#project/web` | `project` |
| `#café #日本` | `café`, `日本` |
| `#123 #123/abc` | none |
| `#1a #y2024` | `1a`, `y2024` |
| `C# foo#bar \#escaped` | none |
| `# Heading` | none |

Categories use only the first segment of nested tags, lowercased, deduplicated
and sorted. `#project/web` and `#project/mobile` therefore share one category.

## Exclusions

These do not contribute inline tags:

- Fenced code blocks using backticks or tildes, including fences within lists.
  An unclosed fence runs to the end of its block; a fence in a list ends when
  the list item ends.
- Inline code spans, closed by a backtick run of the same length. Spans can
  cross lines; an unmatched backtick is plain text.
- Wikilinks, URL fragments and Markdown link destinations.
- YAML frontmatter starting with `---` on the first line and ending with `---`
  or `...`. An unclosed frontmatter block excludes the rest of the note.
- Board column tags such as `#kb/doing`, using the configured `kanban_tag`
  prefix. The bare `#kb` remains a category.

Frontmatter `tags:` is not supported. HTML comments and Obsidian `%%` comments
are not specially excluded; inline tags inside them can still count. The tag
scanner does not specially exclude indented code blocks without fences.

See [configuration.md](configuration.md) for category scope and the board prefix,
and [cli.md](cli.md) for the JSON schema and Kanban operations.
