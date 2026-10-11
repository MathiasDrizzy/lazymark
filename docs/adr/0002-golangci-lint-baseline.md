# golangci-lint baseline

* Status: accepted
* Date: 2026-10-11

## Context and problem statement

The development standard requires `golangci-lint` v2 with a passing run. The first run on the existing
code reported 60 findings: 50 `errcheck`, 1 `ineffassign` and 9 `staticcheck`.

## Decision drivers

* The lint step must pass so it can block regressions.
* Existing, working code is not rewritten only to satisfy a linter.
* Every exception must be written down with its reason.

## Considered options

* Fix all 60 findings in the code.
* Use the `standard` linter set and exclude the findings that are intentional, each with a reason.

## Decision outcome

Chosen option: the `standard` set (`errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`) plus
`gofmt`, with these exceptions in `.golangci.yml`:

* `errcheck` ignores `fmt.Fprint*` (output to the terminal or a pipe: if that fails there is nobody to
  tell) and `fmt.Sscanf` (it parses ids from a click payload and the zero value is the intended default).
  In `_test.go` files `errcheck` is off, because a failed setup call makes the test fail anyway.
* `staticcheck` skips `QF1001` (De Morgan), `QF1008` (embedded field selectors) and `ST1018` (escape
  sequences in literals): style suggestions with no behaviour change, some of them clearer as written
  (for example a literal U+202E in a test that checks it is rejected). It also skips the `ST10xx` naming
  and comment-style checks.
* Closing a file that is only read (`safeio.ReadRegular`) is marked in place with `//nolint:errcheck`,
  because a failed close of a read-only file loses nothing.
* `ineffassign` is off for `internal/storage/dates.go`, where a redundant assignment documents the default
  of the two branches.

Revised 2026-10-11 (order 2026-10-11-06): the first version also ignored `(*os.File).Close`, `os.Remove`
and `walkVault` everywhere. Some of those hid real failures, so the exceptions were removed and the code
fixed, each with a test that failed first:

* `ListNotes` swallowed every walk error, so a missing or unreadable notes folder looked like an empty one.
  It now returns the error for the folder itself and still skips unreadable subfolders and entries
  (`internal/storage/root_error_test.go`). `NotePaths` has no error to return and discards the walk result
  explicitly.
* `backupConfig` removed a half-written `config.json.bak` and ignored a failed removal, which left a corrupt
  copy that later backups would skip. `writeBackup` now reports the leftover file together with the write
  error (`internal/config/backup_error_test.go`).

### Consequences

* Good: `golangci-lint run` passes, any new finding fails the verification step, and the remaining
  exceptions are limited to output, parsing with a default, read-only closes and test setup.
* Bad: `fmt.Fprint*` failures are still ignored; if a command ever needs a reliable exit code for a broken
  pipe, that has to be handled in that command.
