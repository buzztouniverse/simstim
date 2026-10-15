# Routine commands. Run `just` to list them.

# The product name comes from internal/brand, so renaming needs no change here.
name := `sed -n 's/^const Name = "\(.*\)"/\1/p' internal/brand/brand.go`
module := `sed -n 's/^module //p' go.mod`
version := `git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev`
ldflags := "-s -w -X " + module + "/internal/version.Version=" + version
helpers := "internal/client/helpers"

default:
    @just --list

# Build the host helpers for every host platform, into the client's embed folder.
helpers:
    #!/usr/bin/env sh
    set -eu
    for arch in amd64 arm64; do
        CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build -trimpath -ldflags '{{ldflags}}' \
            -o {{helpers}}/host-linux-$arch ./cmd/host
    done
    ls -l {{helpers}}

# Build the shadow terminal (vt/, Zig 0.16) for every host platform, into the client's embed folder.
vt:
    #!/usr/bin/env sh
    set -eu
    (cd vt && zig build linux)
    for arch in amd64 arm64; do
        cp vt/zig-out/linux/shadow-vt-$arch {{helpers}}/vt-linux-$arch
    done
    ls -l {{helpers}}/vt-linux-*

# Build the shadow terminal if Zig 0.16 is installed; otherwise say so and go on.
_vt-if-zig:
    #!/usr/bin/env sh
    set -eu
    if ! command -v zig >/dev/null 2>&1; then
        echo "note: zig not found, so the shadow terminal was not built (see vt/README.md)"
    elif ! zig version | grep -q '^0\.16\.'; then
        echo "note: the shadow terminal needs Zig 0.16, not $(zig version), so it was not built (see vt/README.md)"
    else
        {{just_executable()}} vt
    fi

# Build the helpers, then the client for this machine, into ./bin.
build: helpers _vt-if-zig
    mkdir -p bin
    CGO_ENABLED=0 go build -trimpath -ldflags '{{ldflags}}' -o bin/{{name}} ./cmd/cli
    cp {{helpers}}/host-linux-amd64 bin/{{name}}-host-linux-amd64
    @echo "built bin/{{name}} {{version}}"

# Install the client into ~/.local/bin.
install: build
    install -m 0755 bin/{{name}} ~/.local/bin/{{name}}

# Unit tests with the race detector.
test:
    go test -race -count=1 ./...

# Fuzz the input filter for a while.
fuzz time="30s":
    go test ./internal/keys -run '^$' -fuzz FuzzFilter -fuzztime {{time}}

# Static checks.
lint: check-name
    go vet ./...
    golangci-lint run

# Fail if the product name is spelled outside the allowed places.
check-name:
    scripts/check-name.sh

# Format the code.
fmt:
    gofmt -w cmd internal

# End-to-end tests against a real host: E2E_HOST=<a host added with `host add`>.
e2e: build
    python3 e2e/run.py bin/{{name}}

# A local release build, with no publishing.
snapshot:
    APP_NAME={{name}} goreleaser release --snapshot --clean

# Rename the product (see docs/renaming.md).
rename new:
    scripts/rename.sh {{new}}
