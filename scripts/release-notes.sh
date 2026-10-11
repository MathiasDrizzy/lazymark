#!/usr/bin/env bash
# Prints the CHANGELOG.md section of one version, to use as the release notes.
# Usage: release-notes.sh <version|vversion> [changelog-file]
# Fails if the section does not exist or is empty, so a release cannot go out without notes.
set -euo pipefail

version="${1:?usage: release-notes.sh <version> [changelog-file]}"
version="${version#v}"
changelog="${2:-CHANGELOG.md}"

notes="$(awk -v heading="## [${version}]" '
  /^## \[/ { if (on) exit; on = (index($0, heading) == 1); next }
  /^\[[^]]+\]: / { if (on) exit }
  on { print }
' "$changelog")"

# trim leading and trailing blank lines
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"

if [ -z "${notes//[$'\n ']/}" ]; then
  echo "release-notes.sh: no section for version ${version} in ${changelog}" >&2
  exit 1
fi
printf '%s\n' "$notes"
