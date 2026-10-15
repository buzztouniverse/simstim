#!/usr/bin/env sh
# Prints the unit tests' statement coverage of the packages under internal/,
# except the launcher (internal/tui, which golden screens test), counted
# across packages: a test in one package counts for the code it runs in
# another, and so does a session holder that a test starts as a child
# process. With a number, it fails when the coverage is below it.
# See docs/contributing/testing.md.
set -eu
cd "$(dirname "$0")/.."
min=${1:-}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
pkgs=$(go list ./internal/... | grep -v '/tui$' | tr '\n' ',')
if ! go test -count=1 -coverpkg="${pkgs%,}" -coverprofile="$tmp/cover.out" ./internal/... >"$tmp/test.log" 2>&1; then
    cat "$tmp/test.log"
    echo "Error: a test failed, so there is no coverage to report."
    exit 1
fi
total=$(go tool cover -func="$tmp/cover.out" | awk '$1 == "total:" { sub(/%/, "", $NF); print $NF }')
echo "Coverage of internal/ without the launcher: ${total} %"
if [ -n "$min" ] && awk -v t="$total" -v m="$min" 'BEGIN { exit !(t < m) }'; then
    echo "Error: the coverage is below ${min} %."
    echo "Try: add tests for the code you changed (docs/contributing/testing.md)."
    exit 1
fi
