# Text roles that read in every theme

* Status: accepted
* Date: 2026-10-13

## Context and problem statement

Every color the app paints comes from the active palette (`TestThemeColorsComeFromPalette`), but a palette names one color per
role and some palettes cannot read it: the selected row (Text over Surface1) was 1.7:1 in solarized-dark and the accent text on
it 1.2:1; the dim text (Overlay0) was 1.2:1 to 1.7:1 on the base in nord and solarized-dark; the active panel title (Green) and
the footer keys (Mauve) were 2.6:1 to 3.0:1 in catppuccin-latte and solarized-light. A sweep of the main screens in the 14
built-in themes found 146 runs of text under 3:1.

## Considered options

* Mix colors (blend the dim text toward the background): it breaks the rule that every color is a palette color and the
  identity of each theme.
* Raise every role to 4.5:1: it changes the look of the themes that were fine (the accent of nord and frappe is 3.0 to 3.4 on
  the selected row, by design).
* Per role, keep the color the theme always used and move to the next color of the same palette only where it falls under
  3:1 on the backgrounds it is drawn on (chosen).

## Decision outcome

`internal/ui/theme/roles.go` resolves five roles when a palette is applied: `ColorSelection` (Surface1, or Surface0 where Text
does not read on Surface1: solarized-dark), `ColorMuted` (Overlay0, then Subtext0, then Text), `ColorSoft` (Subtext0, then Text),
`ColorActive` (Green, then Teal, Blue, Mauve, Peach, Yellow) and `ColorAccent` (Peach, then Yellow, Mauve, Teal, Blue, Green),
the last one on the base and on the selected row. Borders keep their own colors. The floor is 3:1, the WCAG level for bold
text and interface that dates already use and that `theme.Button` falls back to; the buttons keep asking for 4.5:1 where the
palette allows it because they are the only text a user must find. `TestTextReadsInEveryTheme` sweeps the notes, Kanban,
Tasks, settings and keybindings screens in all themes; `TestRolesKeepTheThemeColorWhereItReads` fails if a role leaves the
theme's color without a reason.

## Consequences

Where a palette reads fine nothing changes. Measured: 7 of the 14 themes change, only in the roles that failed (latte: muted, active, accent; frappe, tokyo-night, nord, one-dark: muted; solarized-dark: selection, accent; solarized-light: muted, soft, active). Borders keep the theme's color (existing tests and the tuiqa theme scenario pin it); only text moves. Dates
keep their own policy (`views/datecolors.go`). The Markdown preview and the code-block colors use `readable` (the theme's color where it reads on its background, else Text) and a level-1 heading takes the first accent on which Base or Text reads (latte). The sweep covers a note with every Markdown element, the dates and move popups and the Kanban dates popup.
