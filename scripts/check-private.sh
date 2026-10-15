#!/usr/bin/env bash
# Fails when a tracked file holds a private detail: a personal path, a private
# or Tailscale address, a tailnet name or an email address. Examples in the
# docs use the documentation ranges (192.0.2.0/24, 198.51.100.0/24,
# 203.0.113.0/24), example.com and the users "me", "you" and "dev".
#
# A maintainer can add words of their own (machine names, a homelab's name)
# to private/deny.txt, one extended regex per line. That file is git-ignored,
# so the words never enter the repo. CI runs without it.
set -euo pipefail
cd "$(dirname "$0")/.."

patterns=(
    # Home folders of a real user.
    '/Users/[A-Za-z][A-Za-z0-9._-]*'
    '/home/[A-Za-z][A-Za-z0-9._-]*'
    # Private and carrier-grade NAT (Tailscale) IPv4 addresses.
    '(^|[^0-9.])10\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}($|[^0-9.])'
    '(^|[^0-9.])192\.168\.[0-9]{1,3}\.[0-9]{1,3}($|[^0-9.])'
    '(^|[^0-9.])172\.(1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3}($|[^0-9.])'
    '(^|[^0-9.])100\.(6[4-9]|[7-9][0-9]|1[01][0-9]|12[0-7])\.[0-9]{1,3}\.[0-9]{1,3}($|[^0-9.])'
    # Tailnet names.
    '[A-Za-z0-9-]+\.ts\.net'
    # Email addresses.
    '[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}'
)
if [ -f private/deny.txt ]; then
    while IFS= read -r line; do
        case "$line" in '' | '#'*) continue ;; esac
        patterns+=("$line")
    done <private/deny.txt
fi

# Matches that are fine: generic users, documentation names and addresses,
# and the bots' addresses.
allow='/Users/(me|you|runner|shared)([/" ]|$)|/home/(me|you|u|dev|user|runner|linuxbrew|admin|ubuntu|debian|vscode|node)([/" .:;)\]|$)|example-tailnet\.ts\.net|@([A-Za-z0-9-]+\.)*(example|example\.(com|org|net)|openssh\.com|users\.noreply\.github\.com)([^A-Za-z0-9.-]|$)|git@github\.com|noreply@|user@UID|@v[0-9]|@sha256|@latest'

fail=0
while IFS= read -r file; do
    [ -f "$file" ] || continue
    grep -I -q . "$file" 2>/dev/null || continue # binary or empty
    # Remove the allowed forms first, so a line with one of them and a real
    # hit still fails.
    text=$(sed -E "s#${allow}##g" "$file")
    for p in "${patterns[@]}"; do
        hits=$(printf '%s\n' "$text" | grep -n -E -- "$p" || true)
        if [ -n "$hits" ]; then
            echo "$file: $p"
            echo "$hits" | cut -c1-160 | sed 's/^/  /'
            fail=1
        fi
    done
done < <(git ls-files --cached | grep -v -E '^(go\.sum|vt/build\.zig\.zon|scripts/check-private\.sh)$')

if [ "$fail" -ne 0 ]; then
    echo
    echo "Error: a tracked file holds a private detail."
    echo "Try: use a generic name, example.com, or a documentation address such as 203.0.113.7."
    exit 1
fi
echo "check-private: ok (no personal paths, private addresses or emails)"
