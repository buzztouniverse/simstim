#!/usr/bin/env bash
# Checks the docs: generated pages are current, Markdown lint, British
# spelling, plain English, and links. --external also checks links to other
# sites (slow; CI runs it on main and weekly).
set -euo pipefail
cd "$(dirname "$0")/.."

external=false
[ "${1:-}" = "--external" ] && external=true
fail=0
step() { printf '\n== %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "docs-check: $1 not found. Try: scripts/setup.sh" >&2; exit 1; }; }
need typos; need vale; need lychee; need hugo

md=()
while IFS= read -r f; do [ -f "$f" ] && md+=("$f"); done < <(git ls-files -co --exclude-standard '*.md' | grep -v -E '^(private|build|\.tools)/')

step "Generated reference is current"
ref=$(mktemp -d)
trap 'rm -rf "$ref"' EXIT
go run ./cmd/docgen -out "$ref"
if ! diff -r -q "$ref" docs/reference; then
    echo "docs/reference is stale. Try: just docs."
    fail=1
fi

step "Markdown lint"
if command -v npx >/dev/null 2>&1; then
    npx --yes markdownlint-cli2@0.23.3 || fail=1
else
    echo "skipped: needs Node (npx); CI runs it"
fi

step "Spelling (British English)"
typos --force-exclude "${md[@]}" || fail=1

step "Plain English"
vale --minAlertLevel=error "${md[@]}" || fail=1

step "Site build and internal links"
scripts/docs-site.sh dev /
lychee --offline --no-progress --root-dir "$PWD/build/site/public" 'build/site/public/**/*.html' || fail=1

if $external; then
    step "External links"
    lychee --no-progress --config .lychee.toml "${md[@]}" || fail=1
fi

[ "$fail" -eq 0 ] && echo && echo "docs-check: ok"
exit "$fail"
