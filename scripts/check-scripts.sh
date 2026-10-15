#!/usr/bin/env bash
# Fails when a shell script puts a non-ASCII character right after a variable
# name, as in "$arch…": in some locales the shell reads the character as part
# of the name and stops with "unbound variable". Write "${arch}…" instead.
set -euo pipefail
cd "$(dirname "$0")/.."
if hits=$(git ls-files '*.sh' | xargs grep -n -P '\$[A-Za-z_][A-Za-z0-9_]*[^\x00-\x7F]' 2>/dev/null); then
    echo "$hits"
    echo
    echo "Error: a variable name runs into a non-ASCII character."
    echo "Try: wrap the name in braces, as in \${name}…"
    exit 1
fi
echo "check-scripts: ok"
