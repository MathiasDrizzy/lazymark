# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Sponsor button: `.github/FUNDING.yml`.
- Repository standards: `SECURITY.md`, this changelog, `.editorconfig`, Dependabot for GitHub Actions, Go
  modules and Docker, a `golangci-lint` v2 configuration and architecture decision records in `docs/adr/`.
- `govulncheck` runs in CI and in the local verification script.

### Changed
- CI builds the `Dockerfile` (without publishing it) on every push, so the image is checked on GitHub's runners.
- Dependencies: `chroma` 2.20.0 → 2.27.0, and the GitHub Actions `checkout` 4 → 7, `setup-go` 5 → 7 and
  `goreleaser-action` 6 → 7, pinned to the commit of each official release.
- README GIFs were re-recorded in an isolated home directory with the mascot turned off, so they no longer
  show a block-character smudge where the mascot would be.

### Fixed
- A missing or unreadable notes folder is reported as an error instead of looking like an empty one.
- If a half-written `config.json.bak` cannot be removed after a failed backup, the error now names the file
  that stays behind.
- Without a Kitty-compatible terminal, clicking the mascot no longer scales and queues animation images
  that would be discarded.

### Security
- Built with Go 1.27.2, which fixes 13 advisories in the Go 1.27.1 standard library reported by `govulncheck`
  (none of them reachable from lazymark's code). The Docker build image was moved to the matching Go 1.27.2
  digest.
- `golang.org/x/net` v0.59.0 → v0.61.0 (and `x/sync`, `x/sys`, `x/text` with it), which clears five advisories
  that `govulncheck` listed for it (none reachable from lazymark's code). `govulncheck` now reports nothing.

## [1.1.0] - 2026-10-10

### Added
- Custom themes: put a JSON file with the 14 colors in the `themes` folder next to `config.json`. A theme
  that is not valid is skipped with a warning in the status bar.

### Changed
- The mascot animation preloads its frames before showing them, which removes a flicker on the first click.

### Fixed
- A notes folder that is given as a relative path or as a symbolic link is now resolved and walked through
  its real path, and stays confined. Inline tags follow the tag rules.

### Security
- Theme names only accept `a-z`, `0-9`, dot, hyphen and underscore, and the warnings shown for a bad theme
  escape control characters, so a crafted theme file cannot inject terminal escape sequences.
- GitHub Actions workflows were hardened: checkouts no longer persist credentials and the release job does
  not use the build cache.

## [1.0.0] - 2026-10-07

First stable release: Markdown notes, tasks and a Kanban board in the terminal, with a CLI and an MCP server
over the same files.

### Added
- Eight new settings: date warnings and format notice, the cadence of the mascot prompt, `trash_days`,
  `daily_folder` / `daily_name` / `templates_folder`, `kanban_tag`, `notes_sort` / `tasks_sort` and
  `date_glyphs`, all validated and editable in the settings screen, with translations.

### Changed
- `config.json` is read tolerantly: a value of the wrong type falls back to its default with a warning, the
  rest is respected, unknown keys are kept, and an unreadable file is saved as `config.json.bak` before
  anything is written over it.

### Fixed
- Configured folders reject Windows device names (`CON`, `NUL`, `COM1`, ...).
- `trash_days` no longer loses data: retention is per item, and `0` always asks, showing how many files
  are affected.
- `--json` output escapes C1 control characters.
- The config backup neither overwrites an existing file nor follows links.

[Unreleased]: https://github.com/MathiasDrizzy/lazymark/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/MathiasDrizzy/lazymark/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/MathiasDrizzy/lazymark/releases/tag/v1.0.0
