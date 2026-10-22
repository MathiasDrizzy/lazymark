#!/usr/bin/env bash
# Rewrites the deprecated `postflight do ... end` stanza that GoReleaser v2.18 writes into a Homebrew cask
# (hooks.post.install) as the `postflight_steps` form that Homebrew 7 expects.
# Usage: cask-postflight-steps.sh <cask.rb>
# Does nothing if the cask has no `postflight do`. Fails if that block is not the known quarantine hook, so
# a different hook is never rewritten blindly. Remove this script when GoReleaser >= v2.19 is pinned and
# .goreleaser.yaml uses hooks.post.install_steps (goreleaser/goreleaser#6873).
set -euo pipefail

cask="${1:?usage: cask-postflight-steps.sh <cask.rb>}"

grep -q '^  postflight do$' "$cask" || exit 0

block="$(awk '/^  postflight do$/ { on = 1 } on { print } on && /^  end$/ { exit }' "$cask")"
if ! grep -q '/usr/bin/xattr' <<<"$block" || ! grep -q 'com.apple.quarantine' <<<"$block"; then
  echo "cask-postflight-steps: the postflight block is not the quarantine hook; not rewriting it" >&2
  exit 1
fi

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
awk '
  /^  postflight do$/ && !done {
    print "  postflight_steps do"
    print "    run \"/usr/bin/xattr\", args: [\"-dr\", \"com.apple.quarantine\", \"{{staged_path}}/lazymark\"], must_succeed: false"
    print "  end"
    skip = 1; done = 1; next
  }
  skip { if ($0 == "  end") skip = 0; next }
  { print }
' "$cask" >"$tmp"
cat "$tmp" >"$cask"
