# Release notes come from CHANGELOG.md

* Status: accepted
* Date: 2026-10-11

## Context and problem statement

The notes of v1.1.0 and v1.2.0 were the list of commit subjects that GoReleaser builds by default: hashes, dependency
bumps and a subject in Spanish. The development standard asks for notes derived from `CHANGELOG.md`, written by a
person and readable by anyone (ED-6, ED-10).

## Considered options

* Keep GoReleaser's generated changelog and filter it (still commit subjects, not what a user needs).
* A separate `RELEASE_NOTES.md` per version (a second file to keep in sync).
* Extract the version's section from `CHANGELOG.md` and pass it with `goreleaser release --release-notes <file>` (chosen).

## Decision outcome

`scripts/release-notes.sh <version>` prints the `## [version]` section of `CHANGELOG.md`, without the link
references at the bottom, and exits with an error if the section is missing or empty. `release.yml` runs it before
GoReleaser and passes the file with `--release-notes`, so a tag without a changelog section cannot be released.
`.goreleaser.yaml` turns its own changelog off. The `release dry run` job of `ci.yml` runs the same script for the
newest released section and prints the notes in its log. The script has Go tests in `scripts/` (section, link
references, missing section).

### Consequences

* Good: one source of truth; the notes are what a person wrote for users; a missing section stops the release.
* Bad: the changelog must be updated before the tag (the release step now enforces it).
* Releases published before this change had their notes replaced with `gh release edit --notes-file`; tags and assets were not touched.
