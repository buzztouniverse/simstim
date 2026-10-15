#!/usr/bin/env bash
# Builds the docs site into build/site/public: the content from docs/ (through
# cmd/docgen), then Hugo with the Hextra theme (site/). The site's title and
# repository link come from internal/brand and go.mod.
#
#   scripts/docs-site.sh [version] [base-url]
set -euo pipefail
cd "$(dirname "$0")/.."

version=${1:-dev}
base=${2:-/}
name=$(sed -n 's/^const DisplayName = "\(.*\)"/\1/p' internal/brand/brand.go)
module=$(sed -n 's/^module //p' go.mod)

command -v hugo >/dev/null 2>&1 || { echo "docs-site: hugo not found. Try: scripts/setup.sh" >&2; exit 1; }
go run ./cmd/docgen -site build/site
HUGO_TITLE="$name" HUGO_PARAMS_REPO="https://$module" HUGO_PARAMS_VERSION="$version" \
    hugo --source site --destination ../build/site/public --baseURL "$base" --minify --cleanDestinationDir --panicOnWarning
echo "docs-site: built build/site/public ($version)"
