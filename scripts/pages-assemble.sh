#!/usr/bin/env bash
# Assembles the GitHub Pages site in build/pages: the latest release's docs at
# the root, each release's docs at /vX.Y/ (from the docs-site.tar.gz the
# release job attached), and main's docs at /dev/. Before the first release,
# main's docs are also at the root. Needs gh, logged in (GH_TOKEN in CI).
#
#   scripts/pages-assemble.sh https://docs.example.org
#
# CI passes the address that GitHub Pages reports: the custom domain, when
# the repo has one.
set -euo pipefail
cd "$(dirname "$0")/.."

site=${1:?usage: pages-assemble.sh <site URL>}
site=${site%/}
out=build/pages
rm -rf "$out"
mkdir -p "$out"

# main, at /dev/
scripts/docs-site.sh dev "$site/dev/"
cp -R build/site/public "$out/dev"

# Each release's own build, at /vX.Y/ (the newest patch of each minor).
latest=""
versions=()
while read -r tag; do
    [ -n "$tag" ] || continue
    minor=$(echo "$tag" | sed -E 's/^(v[0-9]+\.[0-9]+).*/\1/')
    [ -d "$out/$minor" ] && continue
    tmp=$(mktemp -d)
    if gh release download "$tag" --pattern docs-site.tar.gz --dir "$tmp" 2>/dev/null; then
        mkdir -p "$out/$minor"
        tar -xzf "$tmp/docs-site.tar.gz" -C "$out/$minor"
        versions+=("$minor")
        [ -n "$latest" ] || latest=$tag
    fi
    rm -rf "$tmp"
done < <(gh release list --exclude-drafts --exclude-pre-releases --limit 50 --json tagName --jq '.[].tagName')

# The root: the latest release's docs, built again for the root URL, or main's.
if [ -n "$latest" ]; then
    wt=$(mktemp -d)
    git worktree add --detach "$wt" "$latest" >/dev/null
    (cd "$wt" && scripts/docs-site.sh "$latest" "$site/")
    cp -R "$wt/build/site/public/." "$out/"
    git worktree remove --force "$wt"
else
    scripts/docs-site.sh dev "$site/"
    cp -R build/site/public/. "$out/"
fi

# The version menu in the site's top bar reads this list.
list=""
for v in "${versions[@]:-}"; do
    [ -n "$v" ] && list+="\"$v\","
done
printf '{"latest": "%s", "versions": [%s], "dev": "dev"}\n' "$latest" "${list%,}" >"$out/versions.json"
echo "pages-assemble: built $out (latest: ${latest:-none}; versions: ${versions[*]:-none})"
