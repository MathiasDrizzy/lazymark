package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const sampleChangelog = `# Changelog

## [Unreleased]

### Added
- Not released yet.

## [1.2.0] - 2026-10-11

### Added
- Thing two.

### Fixed
- Bug two.

## [1.1.0] - 2026-10-10

### Changed
- Thing one.

[Unreleased]: https://example.com/compare/v1.2.0...HEAD
[1.2.0]: https://example.com/compare/v1.1.0...v1.2.0
[1.1.0]: https://example.com/releases/tag/v1.1.0
`

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash script")
	}
	dir := t.TempDir()
	cl := filepath.Join(dir, "CHANGELOG.md")
	if err := os.WriteFile(cl, []byte(sampleChangelog), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{"release-notes.sh"}, append(args, cl)...)...)
	out, err := cmd.Output()
	return string(out), err
}

func TestReleaseNotesAreTheVersionSection(t *testing.T) {
	for _, v := range []string{"1.2.0", "v1.2.0"} {
		got, err := run(t, v)
		if err != nil {
			t.Fatalf("%s: %v", v, err)
		}
		want := "### Added\n- Thing two.\n\n### Fixed\n- Bug two.\n"
		if got != want {
			t.Errorf("%s: got %q, want %q", v, got, want)
		}
	}
}

func TestReleaseNotesStopBeforeTheLinkReferences(t *testing.T) {
	got, err := run(t, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "example.com") || got != "### Changed\n- Thing one.\n" {
		t.Errorf("got %q", got)
	}
}

func TestReleaseNotesFailWithoutTheSection(t *testing.T) {
	if _, err := run(t, "9.9.9"); err == nil {
		t.Error("a version with no section in CHANGELOG.md must fail")
	}
	if _, err := run(t); err == nil {
		t.Error("no version must fail")
	}
}

// TestGoreleaserKeepsItsChangelogStep guards a bug that shipped in v1.2.1: with `changelog: disable: true`, GoReleaser skips the step
// that loads --release-notes (checked with a real run: "generating changelog" does not appear), so the release was published with an
// empty body. The flag itself already turns off the generated changelog, so the config must not disable it.
func TestGoreleaserKeepsItsChangelogStep(t *testing.T) {
	cfg, err := os.ReadFile(filepath.Join("..", ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var inChangelog bool
	for _, line := range strings.Split(string(cfg), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			inChangelog = strings.HasPrefix(line, "changelog:")
			continue
		}
		if inChangelog && strings.HasPrefix(trimmed, "disable:") && strings.Contains(trimmed, "true") {
			t.Fatal(".goreleaser.yaml sets changelog.disable: true, which makes GoReleaser ignore --release-notes (empty release body)")
		}
	}
}
