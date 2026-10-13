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

// TestReleaseKeepsItsSBOMAndAttestation guards the supply-chain items of ED-11: an SPDX SBOM per archive in GoReleaser, and a build
// provenance attestation over checksums.txt in release.yml with the write permissions on the release job only. Dropping any of them
// would publish releases that `gh attestation verify` rejects, and a dry run would not notice.
func TestReleaseKeepsItsSBOMAndAttestation(t *testing.T) {
	read := func(parts ...string) string {
		b, err := os.ReadFile(filepath.Join(parts...))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	if cfg := read("..", ".goreleaser.yaml"); !strings.Contains(cfg, "\nsboms:\n") || !strings.Contains(cfg, "artifacts: archive") {
		t.Error(".goreleaser.yaml has no sboms entry for the archives")
	}
	wf := read("..", ".github", "workflows", "release.yml")
	for _, want := range []string{"actions/attest@", "subject-checksums: dist/checksums.txt", "id-token: write", "attestations: write", "download-syft@"} {
		if !strings.Contains(wf, want) {
			t.Errorf("release.yml lacks %q", want)
		}
	}
	top, _, _ := strings.Cut(wf, "\njobs:")
	if strings.Contains(top, "write") {
		t.Error("release.yml grants a write permission at workflow level: it belongs to the release job only")
	}
}
