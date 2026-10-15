#!/usr/bin/env bash
# Everything that must hold before the repository goes public
# (docs/contributing/going-public.md): the name check, no secrets and no
# private details in the tree or the history, licence information on every
# file (REUSE), the docs build and their links, the workflows' security, and
# the OpenSSF Scorecard checks this repository controls.
#
#   scripts/public-check.sh          (just public-check)
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
step() { printf '\n== %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "public-check: $1 not found. Try: scripts/setup.sh" >&2; exit 1; }; }
need gitleaks; need scorecard; need uvx

step "The name lives in internal/brand"
scripts/check-name.sh || fail=1

step "No private details in the tree"
scripts/check-private.sh || fail=1

step "No private details in the history"
scripts/check-private.sh --history || fail=1

step "No secrets in the history (gitleaks)"
gitleaks git . --redact --no-banner --exit-code 1 || fail=1

step "Licence information on every file (REUSE)"
uvx --from 'reuse[charset-normalizer]==6.2.0' reuse lint --lines >/dev/null && echo "reuse: ok" || {
    uvx --from 'reuse[charset-normalizer]==6.2.0' reuse lint --lines | head -20
    fail=1
}

step "Workflows (zizmor)"
uvx zizmor@1.16.0 --offline --min-severity medium .github || fail=1

step "Docs: build, lint, spelling, prose and links"
scripts/docs-check.sh || fail=1

step "OpenSSF Scorecard: the checks this repository controls"
token=${GITHUB_AUTH_TOKEN:-${GITHUB_TOKEN:-$(gh auth token 2>/dev/null || true)}}
if [ -z "$token" ]; then
    echo "skipped: needs a GitHub token (gh auth login, or GITHUB_TOKEN)"
    fail=1
else
    module=$(sed -n 's/^module //p' go.mod)
    out=$(mktemp)
    GITHUB_AUTH_TOKEN=$token scorecard --repo="$module" --format=json >"$out" 2>/dev/null || true
    python3 - "$out" <<'PY' || fail=1
import json, sys
r = json.load(open(sys.argv[1]))
# Checks the repository's own files decide. The rest need the repository to be
# public, a first release, time, or a step by the owner (going-public.md).
want = {"Binary-Artifacts": 10, "Dangerous-Workflow": 10, "Dependency-Update-Tool": 10,
        "Fuzzing": 10, "License": 10, "Pinned-Dependencies": 9, "SAST": 10,
        "Security-Policy": 10, "Token-Permissions": 10, "Vulnerabilities": 10}
bad = 0
for c in r["checks"]:
    need = want.get(c["name"])
    mark = "  " if need is None else ("ok" if c["score"] >= need else "NO")
    bad += mark == "NO"
    print(f"{mark} {c['score']:>3}  {c['name']}  ({c['reason'][:70]})")
print(f"Overall: {r['score']} (the badge needs the repository public)")
sys.exit(1 if bad else 0)
PY
    rm -f "$out"
fi

echo
if [ "$fail" -eq 0 ]; then echo "public-check: ok"; else echo "public-check: FAILED"; fi
exit "$fail"
