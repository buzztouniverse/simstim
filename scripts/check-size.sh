#!/usr/bin/env bash
# Fails when a hand-written Go source file is over 500 lines, the quality
# bar in docs/contributing/testing.md. Tests are not counted. Split a long
# file by topic: one file for each part of a feature.
set -euo pipefail
cd "$(dirname "$0")/.."
max=500
long=$(git ls-files -co --exclude-standard '*.go' | grep -v '_test\.go$' | while read -r f; do
    [ -f "$f" ] || continue
    n=$(wc -l <"$f" | tr -d ' ')
    [ "$n" -le "$max" ] || echo "$n $f"
done)
if [ -n "$long" ]; then
    echo "$long"
    echo
    echo "Error: these files are over $max lines."
    echo "Try: move a group of functions into a file of their own, by topic."
    exit 1
fi
echo "check-size: ok (no Go source file is over $max lines)"
