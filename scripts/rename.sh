#!/usr/bin/env bash
# Renames the product. Usage: scripts/rename.sh <new-name> [<new-module-path>]
#
# It changes internal/brand, the module path and imports (if given), and the docs.
# It prints the steps that need a person: the GitHub repo, the folder and any
# hosts that already run an old helper.
set -euo pipefail
cd "$(dirname "$0")/.."

new=${1:-}
newmod=${2:-}
if ! [[ "$new" =~ ^[a-z][a-z0-9]{1,30}$ ]]; then
    echo "Usage: scripts/rename.sh <new-name> [<new-module-path>]" >&2
    echo "The name must be lowercase letters and digits, starting with a letter." >&2
    exit 2
fi
old=$(sed -n 's/^const Name = "\(.*\)"/\1/p' internal/brand/brand.go)
oldmod=$(sed -n 's/^module //p' go.mod)
if [ "$old" = "$new" ]; then
    echo "The name is already $new."
    exit 0
fi
OLD=$(echo "$old" | tr a-z A-Z)
NEW=$(echo "$new" | tr a-z A-Z)

echo "Renaming $old to $new"

# 1. The one place in the code.
sed -i.bak "s/^const Name = \"$old\"/const Name = \"$new\"/; s/^const DisplayName = \"$old\"/const DisplayName = \"$new\"/" internal/brand/brand.go
rm -f internal/brand/brand.go.bak

# 2. The module path, if a new one is given.
if [ -n "$newmod" ]; then
    go mod edit -module "$newmod"
    git ls-files '*.go' | xargs sed -i.bak "s#\"$oldmod/#\"$newmod/#g"
    git ls-files '*.go' | sed 's/$/.bak/' | xargs rm -f
fi

# 3. The docs.
git ls-files '*.md' 'NOTICE' | while read -r f; do
    sed -i.bak "s/$old/$new/g; s/${OLD}_/${NEW}_/g" "$f" && rm -f "$f.bak"
done

gofmt -l cmd internal || true
scripts/check-name.sh

cat <<EOF

Done in the repo. Steps for a person:
  1. Rename the GitHub repo:  gh repo rename $new
  2. Rename the local folder, and update the git remote if needed.
  3. Hosts that run the old helper keep their sessions under ~/.local/share/$old
     and ~/.local/state/$old. Delete them, or move them, after the old sessions end.
  4. Update the Homebrew tap and any install script URLs.
  5. Review the diff:  git diff --stat
EOF
