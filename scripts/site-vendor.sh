#!/usr/bin/env bash
# Fetches the site's fonts at pinned versions, checks each download's
# SHA-256, cuts them down with scripts/site-fonts.py, and writes them into
# site/public/fonts/ (decision 0022):
#   - JetBrains Mono, for code, commands and the launcher (SIL OFL 1.1);
#   - Inter, for text (SIL OFL 1.1);
#   - Schibsted Grotesk, for headings (SIL OFL 1.1);
#   - the site's own braille font, drawn by scripts/site-fonts.py.
# The results are committed, so a build needs none of this. Run it only to
# change a version: change the pins below, run it, and commit the result.
# The site's code dependencies are npm packages, pinned in site/package.json
# and site/pnpm-lock.yaml. Needs curl, tar, unzip and uv (for fontTools).
set -euo pipefail
cd "$(dirname "$0")/.."

mono_url=https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip
mono_sha=6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf
sans_url=https://fonts.gstatic.com/s/inter/v20/UcCo3FwrK3iLTcviYwYZ8UA3.woff2 # Inter 4, Latin, from Google Fonts
sans_sha=dd8a4575be9806105ac3decd02805cd2782fe7c05abb02c582316bc436ce03ae
sans_licence=https://raw.githubusercontent.com/rsms/inter/v4.1/LICENSE.txt
display_url=https://registry.npmjs.org/@fontsource-variable/schibsted-grotesk/-/schibsted-grotesk-5.3.0.tgz
display_sha=dd729607754cfbf3f4c7f0872ba6d727e953a3d2b533184afe009b134f04f996
fonttools="uvx --from fonttools==4.60.1 --with brotli==1.1.0 python -I"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
fetch() { # url sha256 file
    curl -fsSL -o "$3" "$1"
    echo "$2  $3" | shasum -a 256 -c --status || { echo "site-vendor: $1 does not match its SHA-256" >&2; exit 1; }
}

fetch "$mono_url" "$mono_sha" "$tmp/mono.zip"
unzip -q "$tmp/mono.zip" -d "$tmp/mono"
fetch "$sans_url" "$sans_sha" "$tmp/sans.woff2"
fetch "$display_url" "$display_sha" "$tmp/display.tgz"
mkdir -p "$tmp/display"
tar -xzf "$tmp/display.tgz" -C "$tmp/display"

dst=site/public/fonts
mkdir -p "$dst"
$fonttools scripts/site-fonts.py mono "$tmp/mono/fonts/variable/JetBrainsMono[wght].ttf" "$dst/mono.woff2"
$fonttools scripts/site-fonts.py sans "$tmp/sans.woff2" "$dst/sans.woff2"
$fonttools scripts/site-fonts.py display "$tmp/display/package/files/schibsted-grotesk-latin-wght-normal.woff2" "$dst/display.woff2"
$fonttools scripts/site-fonts.py braille "$dst/braille.woff2"
cp "$tmp/mono/OFL.txt" "$dst/LICENSE-mono.txt"
curl -fsSL -o "$dst/LICENSE-sans.txt" "$sans_licence"
cp "$tmp/display/package/LICENSE" "$dst/LICENSE-display.txt"
ls -l "$dst"
echo "site-vendor: the fonts are in $dst"
