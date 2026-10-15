#!/usr/bin/env bash
# Compares the attach pump's benchmarks between a base commit and the working
# tree, on this machine, and fails on a clear slowdown.
#
#   scripts/bench.sh [base]       (default: origin/main)
#
# Both sides are built as test binaries first, then run in turns, so that a
# change in the machine's load hits both alike. benchstat (pinned) compares
# them. The check fails when a benchmark that exists on both sides got more
# than BENCH_MAX percent slower (default 10) with p below 0.05.
set -euo pipefail
cd "$(dirname "$0")/.."

base=${1:-origin/main}
rounds=${BENCH_ROUNDS:-6}
max=${BENCH_MAX:-10}
pkgs=(./internal/proto ./internal/ring ./internal/term ./internal/keys ./internal/host ./internal/client)
benchstat="golang.org/x/perf/cmd/benchstat@v0.0.0-20260929162123-406019bb8b68"

work=$(mktemp -d)
cleanup() {
    git worktree remove --force "$work/base" >/dev/null 2>&1 || true
    rm -rf "$work"
}
trap cleanup EXIT

git worktree add --detach "$work/base" "$base" >/dev/null 2>&1 || { echo "bench: no commit $base" >&2; exit 1; }
build() { # build <tree> <out dir>
    mkdir -p "$2"
    for p in "${pkgs[@]}"; do
        name=$(basename "$p")
        (cd "$1" && go test -c -o "$2/$name.test" "$p" >/dev/null 2>&1) || true
    done
}
echo "bench: building $base and the working tree"
build "$work/base" "$work/bin-base"
build "$PWD" "$work/bin-head"

run() { # run <bin dir> <out file> <package dir>
    for t in "$1"/*.test; do
        dir=${pkgs[0]}
        for p in "${pkgs[@]}"; do [ "$(basename "$p").test" = "$(basename "$t")" ] && dir=$p; done
        (cd "$3/$dir" && "$t" -test.run '^$' -test.bench . -test.benchmem -test.count 1 -test.benchtime 300ms) >>"$2"
    done
}
for i in $(seq "$rounds"); do
    printf '\rbench: round %d of %d' "$i" "$rounds"
    run "$work/bin-base" "$work/base.txt" "$work/base"
    run "$work/bin-head" "$work/head.txt" "$PWD"
done
echo

GOTOOLCHAIN=auto go run "$benchstat" -table '' -row .name base="$work/base.txt" head="$work/head.txt" 2>/dev/null >"$work/stat.txt" ||
    GOTOOLCHAIN=auto go run "$benchstat" "$work/base.txt" "$work/head.txt" >"$work/stat.txt"
cat "$work/stat.txt"

# Only the sec/op table counts: it comes first.
python3 -I - "$work/stat.txt" "$max" <<'PY'
import re, sys
text = open(sys.argv[1]).read()
limit = float(sys.argv[2])
first = text.split("\n\n")[0] if "sec/op" in text.split("\n\n")[0] else text
bad = []
for line in first.splitlines():
    m = re.search(r"([+-]\d+(?:\.\d+)?)%\s+\(p=(\d\.\d+)", line)
    if m and float(m.group(1)) > limit and float(m.group(2)) < 0.05:
        bad.append(line.strip())
if bad:
    print("bench: slower than the base by more than %g%%:" % limit)
    for b in bad:
        print("  " + b)
    sys.exit(1)
print("bench: no benchmark is more than %g%% slower" % limit)
PY
