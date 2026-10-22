package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// deprecatedCask is what GoReleaser v2.18 writes for hooks.post.install (checked against the v1.2.2 cask).
const deprecatedCask = `cask "lazymark" do
  version "1.2.2"

  binary "lazymark"

  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr",
                     args: ["-dr", "com.apple.quarantine", "#{staged_path}/lazymark"]
    end
  end

  # No zap stanza required
end
`

const stepsBlock = `  postflight_steps do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/lazymark"], must_succeed: false
  end
`

func fixCask(t *testing.T, cask string) (string, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash script")
	}
	path := filepath.Join(t.TempDir(), "lazymark.rb")
	if err := os.WriteFile(path, []byte(cask), 0o600); err != nil {
		t.Fatal(err)
	}
	err := exec.Command("bash", "cask-postflight-steps.sh", path).Run()
	got, rerr := os.ReadFile(path)
	if rerr != nil {
		t.Fatal(rerr)
	}
	return string(got), err
}

func TestCaskPostflightBecomesPostflightSteps(t *testing.T) {
	got, err := fixCask(t, deprecatedCask)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(deprecatedCask, deprecatedCask[strings.Index(deprecatedCask, "  postflight do"):strings.Index(deprecatedCask, "  # No zap")], stepsBlock+"\n", 1)
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "postflight do") {
		t.Error("the deprecated stanza is still there")
	}
}

func TestCaskWithoutPostflightIsLeftAlone(t *testing.T) {
	clean := strings.Replace(deprecatedCask, deprecatedCask[strings.Index(deprecatedCask, "  postflight do"):strings.Index(deprecatedCask, "  # No zap")], stepsBlock+"\n", 1)
	got, err := fixCask(t, clean)
	if err != nil {
		t.Fatal(err)
	}
	if got != clean {
		t.Errorf("a cask that already uses postflight_steps must not change:\n%s", got)
	}
}

func TestCaskFixFailsOnAnUnknownPostflight(t *testing.T) {
	odd := strings.Replace(deprecatedCask, "xattr", "other-tool", 2)
	if _, err := fixCask(t, odd); err == nil {
		t.Error("a postflight block that is not the known xattr hook must fail instead of being rewritten blindly")
	}
}

// TestReleaseRewritesTheCaskHook guards the order 2026-10-22-17 fix: while GoReleaser v2.18 is pinned, release.yml must rewrite the
// cask it pushes, and ci.yml must check the dry-run cask. A dry run does not push, so only these two references keep the step alive.
func TestReleaseRewritesTheCaskHook(t *testing.T) {
	for _, name := range []string{"release.yml", "ci.yml"} {
		b, err := os.ReadFile(filepath.Join("..", ".github", "workflows", name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), "scripts/cask-postflight-steps.sh") {
			t.Errorf("%s does not run scripts/cask-postflight-steps.sh", name)
		}
	}
}
