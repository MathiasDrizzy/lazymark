# Review of the perfectjared fork: lane boards are not adopted yet

* Status: accepted
* Date: 2026-10-11

## Context and problem statement

The public fork [`perfectjared/lazymark`](https://github.com/perfectjared/lazymark) carries four commits that
`main` does not have (compared on 2026-10-11 with `gh api repos/MathiasDrizzy/lazymark/compare/main...perfectjared:main`:
4 ahead, 36 behind, merge base `602c74b`). The fork did not open a pull request or an issue. This record decides
which of its ideas lazymark should take.

The four commits:

| Commit | What it does |
| ------ | ------------ |
| `4885cf5` Lane boards | A note shaped like an Obsidian Kanban plugin board (`## Lane` headings, `- [ ]` cards under them) is read and edited in place. Adds `--board <note>`, the MCP tools `get_board` and `move_card`, `internal/storage/lanes.go` and tests. |
| `de57b5b` | `--dir` relative to the working directory is made absolute. |
| `339db3e` | README link without an anchor. |
| `559689e` Checkpoint | A `.workbench/checkpoint.json` of the fork author's own tooling. |

License: the fork's `LICENSE` is MIT (`gh api repos/perfectjared/lazymark/license` returns `MIT`), the same as
this repository, so reusing code would be allowed with attribution.

## Decision drivers

* Keep the Kanban model that users already have: the column of a card is a tag (`#kb/doing`) on its task line.
* Do not widen the public surface (CLI flags, MCP tools, data formats) without a product decision.
* Prefer small, measurable changes that `main` can absorb without a large merge.

## Considered options

* Merge the fork as it is.
* Port the lane boards feature.
* Take nothing now, record the idea, and revisit it as a product decision.

## Decision outcome

Chosen option: take nothing now and record the lane boards idea for a product decision.

Measured on 2026-10-11:

* The lane boards commit is 704 added lines in 22 files (264 of them tests), and a trial merge with `main`
  (`git merge-tree`) conflicts in `cmd/lazymark/main.go`. It adds a second way to define columns (headings
  instead of tags), a `--board` flag, two MCP tools and a new section in `docs/cli.md`.
* The fork's code was written against another tool's rules (`wb`, from the author's own project): the
  parsing rules are shared with that tool on purpose, so lazymark would take on a compatibility contract
  with a project it does not control.
* `--dir` relative to the working directory is already handled on `main`: the notes folder is made
  absolute in `storage.New`, and `TestRelativeVaultOperations` (`internal/storage/relative_dir_test.go`)
  covers a relative path, `.` and a symbolic link. Nothing to take.
* The README change and the checkpoint file do not apply.

No code from the fork was reused, so no `Co-authored-by` trailer is needed. If lane boards are adopted later,
the commit that does it must credit the author (`Co-authored-by: perfectjared`) or cite the original commit.

### Consequences

* Good: no new public surface, no merge conflict to resolve, no external compatibility contract.
* Bad: people who keep Obsidian Kanban plugin boards cannot open them as boards in lazymark yet.
* To revisit: lane boards as an optional board source, decided by the maintainer as a product feature.
  The cheapest first step would be read-only (`get_board`) before any editing in place.

## Appendix: how the fork's lane boards work

Read on 2026-10-11 from commit `4885cf5` of the fork, kept here so a future decision does not need to start over.

* **Model.** One note is the whole board. Each `## Title` heading is a lane (at most 32), and each `- [ ]`, `- [x]` or
  `- [X]` line at the margin under it is a card. The card's block is its line plus the indented lines right below it,
  up to the first blank line. Front matter (`---` … `---`) and `%%` blocks (such as `%% kanban:settings %%`) are skipped.
  The rules are shared on purpose with `wb`, another tool of the fork's author.
* **Moving.** `storage.MoveCardToLane` moves the card's lines to the end of the target lane and leaves the checkbox and
  the text alone. It uses the same atomic, mtime-checked rewrite as lazymark (`ErrNoteChanged`).
* **Interface.** `lazymark --board <note>` opens that note as the board (without `--dir`, its folder becomes the notes
  folder). `H`/`L`, `Shift+←/→` and dragging move a card between lanes, `K`/`J` reorder inside a lane, and `Space` ticks
  a card in place without moving it. A lane called `Done` is drawn as the done column. MCP gets `get_board(board)` and
  `move_card(id, lane)`.
* **Difference with lazymark.** Here the column is a `#kb/<id>` tag and the board gathers the tasks of every note;
  ticking a task moves it to the Done column. In the fork the column is the position under a heading, the board is one
  note, and ticking never moves a card.
* **Cost of adopting it.** The code branches on `board.Lanes != nil` in `core.go`, `kanban.go` and `views/kanban.go`,
  and its comments and strings are in Spanish, so it would have to be rewritten in English first.
* **Not reviewed.** `internal/ops/ops.go` and the edge cases beyond what the fork's own tests cover.
