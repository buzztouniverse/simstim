#!/usr/bin/env bash
# Prints the CHANGELOG section for a version, for the release's notes.
# Usage: scripts/release-notes.sh v1.2.3   (or "Unreleased")
set -euo pipefail
cd "$(dirname "$0")/.."

want=${1:?usage: release-notes.sh <version|Unreleased>}
want=${want#v}

notes=$(awk -v want="$want" '
    /^## \[/ {
        v = $0
        sub(/^## \[/, "", v)
        sub(/\].*/, "", v)
        if (found) exit
        found = (v == want)
        next
    }
    found { print }
' CHANGELOG.md)

if [ -z "$(printf '%s' "$notes" | tr -d '[:space:]')" ]; then
    echo "release-notes: CHANGELOG.md has no section for [$want]" >&2
    echo "Try: rename the [Unreleased] section to [$want] - <date> before tagging." >&2
    exit 1
fi
printf '%s\n' "$notes" | sed -e '/./,$!d'
