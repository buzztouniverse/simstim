#!/usr/bin/env bash
# Fetches the site's third-party files at pinned versions, checks each
# download's SHA-256, and writes them into site/ (decision 0021):
#   - three.js, unchanged, into site/assets/vendor/three/ (MIT); Hugo bundles
#     and minifies the parts the landing page uses;
#   - JetBrains Mono and Inter, cut down, into site/static/fonts/ (SIL OFL 1.1);
#   - the site's own braille font, drawn by scripts/site-fonts.py.
# The results are committed, so a build needs none of this. Run it only to
# change a version: change the pins below, run it, and commit the result.
# Needs curl, tar, unzip and uv (for fontTools).
set -euo pipefail
cd "$(dirname "$0")/.."

three_version=0.186.1
three_url="https://registry.npmjs.org/three/-/three-$three_version.tgz"
three_sha=8cd068708ea44f2c73c944b1cead2ba2f0d5c15c8fc194e5700f4e4f4a033fe7
mono_url=https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip
mono_sha=6f6376c6ed2960ea8a963cd7387ec9d76e3f629125bc33d1fdcd7eb7012f7bbf
sans_url=https://fonts.gstatic.com/s/inter/v20/UcCo3FwrK3iLTcviYwYZ8UA3.woff2 # Inter 4, Latin, from Google Fonts
sans_sha=dd8a4575be9806105ac3decd02805cd2782fe7c05abb02c582316bc436ce03ae
sans_licence=https://raw.githubusercontent.com/rsms/inter/v4.1/LICENSE.txt
fonttools="uvx --from fonttools==4.60.1 --with brotli==1.1.0 python -I"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
fetch() { # url sha256 file
    curl -fsSL -o "$3" "$1"
    echo "$2  $3" | shasum -a 256 -c --status || { echo "site-vendor: $1 does not match its SHA-256" >&2; exit 1; }
}

# three.js: the two module files of the build, and the licence.
fetch "$three_url" "$three_sha" "$tmp/three.tgz"
mkdir -p "$tmp/three"
tar -xzf "$tmp/three.tgz" -C "$tmp/three"
dst=site/assets/vendor/three
rm -rf "$dst"
mkdir -p "$dst"
cp "$tmp/three/package/build/three.core.js" "$tmp/three/package/build/three.module.js" "$dst/"
cp "$tmp/three/package/LICENSE" "$dst/LICENSE"
echo "$three_version" >"$dst/VERSION"

# The fonts.
fetch "$mono_url" "$mono_sha" "$tmp/mono.zip"
unzip -q "$tmp/mono.zip" -d "$tmp/mono"
fetch "$sans_url" "$sans_sha" "$tmp/sans.woff2"
dst=site/static/fonts
mkdir -p "$dst"
$fonttools scripts/site-fonts.py mono "$tmp/mono/fonts/variable/JetBrainsMono[wght].ttf" "$dst/mono.woff2"
$fonttools scripts/site-fonts.py sans "$tmp/sans.woff2" "$dst/sans.woff2"
$fonttools scripts/site-fonts.py braille "$dst/braille.woff2"
cp "$tmp/mono/OFL.txt" "$dst/LICENSE-mono.txt"
curl -fsSL -o "$dst/LICENSE-sans.txt" "$sans_licence"
ls -l site/assets/vendor/three "$dst"
echo "site-vendor: three.js $three_version and the fonts are in site/"
