#!/usr/bin/env bash
# Fails when a change touches the CLI, the config, the key map or the agent
# table without touching the docs and the CHANGELOG in the same change.
#
#   scripts/check-docs-impact.sh <base> [<head>]
#
# CI passes the pull request's base, or the commit before a push. A pull
# request labelled "no docs needed" skips the check (CI reads the label).
set -euo pipefail
cd "$(dirname "$0")/.."

base=${1:?usage: check-docs-impact.sh <base> [<head>]}
head=${2:-HEAD}
changed=$(git diff --name-only "$base" "$head")

# What users see: commands and flags, settings, keys, agents, screens.
impact='^(main\.go|internal/cli/|internal/client/config[^/]*\.go|internal/tui/(palette|keys|prompt|dialog|wizard)\.go|internal/agents/)'
hits=$(printf '%s\n' "$changed" | grep -E "$impact" | grep -v '_test\.go$' || true)
[ -n "$hits" ] || { echo "docs-impact: ok, no user-facing code changed"; exit 0; }

docs=$(printf '%s\n' "$changed" | grep -E '^docs/' | grep -v -E '^docs/(research|contributing/decisions)/' || true)
log=$(printf '%s\n' "$changed" | grep -E '^CHANGELOG\.md$' || true)
if [ -n "$docs" ] && [ -n "$log" ]; then
    echo "docs-impact: ok, user-facing code changed with its docs and CHANGELOG"
    exit 0
fi

echo "docs-impact: this change touches user-facing code:"
printf '  %s\n' $hits
[ -n "$docs" ] || echo "but no page under docs/ (outside research and decisions) changed."
[ -n "$log" ] || echo "but CHANGELOG.md did not change."
echo
echo "Try: update the how-to or explanation page, run \`just docs\` for the reference, and add a CHANGELOG line under Unreleased."
echo "If no user can notice the change, label the pull request \"no docs needed\"."
exit 1
