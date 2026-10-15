#!/usr/bin/env bash
# Creates or updates every label in .github/labels.yml on GitHub, with gh.
# Labels that are not in the file are left alone. --dry-run prints only.
set -euo pipefail
cd "$(dirname "$0")/.."

dry=false
[ "${1:-}" = "--dry-run" ] && dry=true

name="" color="" desc=""
flush() {
    [ -n "$name" ] || return 0
    if $dry; then
        printf 'label %-20s #%s  %s\n' "$name" "$color" "$desc"
    else
        gh label create "$name" --color "$color" --description "$desc" --force >/dev/null
        printf 'synced %s\n' "$name"
    fi
    name="" color="" desc=""
}
unquote() { local v=$1; v=${v#\"}; v=${v%\"}; printf '%s' "$v"; }
while IFS= read -r line; do
    case "$line" in
        "- name: "*) flush; name=$(unquote "${line#- name: }") ;;
        "  color: "*) color=$(unquote "${line#  color: }") ;;
        "  description: "*) desc=${line#  description: } ;;
    esac
done <.github/labels.yml
flush
