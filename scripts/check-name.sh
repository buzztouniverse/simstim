#!/usr/bin/env bash
# Fails when the product name is spelled in code outside internal/brand.
# The module path and import paths may contain it; docs may mention it.
set -euo pipefail
cd "$(dirname "$0")/.."

name=$(sed -n 's/^const Name = "\(.*\)"/\1/p' internal/brand/brand.go)
module=$(sed -n 's/^module //p' go.mod)
if [ -z "$name" ]; then
    echo "check-name: cannot read the name from internal/brand/brand.go" >&2
    exit 2
fi

fail=0
while IFS= read -r file; do
    case "$file" in
        internal/brand/*) continue ;;
    esac
    # Drop the module path, then look for the name in any case.
    hits=$(sed "s#${module}##g" "$file" | grep -n -i -- "$name" || true)
    if [ -n "$hits" ]; then
        echo "$file:"
        echo "$hits" | sed 's/^/  /'
        fail=1
    fi
done < <(git ls-files --cached --others --exclude-standard -- '*.go' 'justfile' '.goreleaser.yaml' '.github/workflows/*.yml' 'scripts/*.sh' 'install.sh' 'test/e2e/*.py' 'vt/*.zig' 'vt/src/*.zig' 'vt/build.zig.zon' 'internal/*/testdata/*/*.py' | grep -v '^scripts/check-name.sh$' | grep -v '^scripts/rename.sh$')

if [ "$fail" -ne 0 ]; then
    echo
    echo "Error: the product name is spelled outside internal/brand."
    echo "Try: use brand.Name, brand.HostBinary or brand.Env(...) instead (see docs/contributing/renaming.md)."
    exit 1
fi
echo "check-name: ok (name \"$name\" appears only in internal/brand)"
