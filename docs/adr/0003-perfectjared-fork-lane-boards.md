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
