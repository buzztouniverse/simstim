#!/bin/sh
# Installs the client from the latest GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh | sh
#
# Options (environment variables):
#   INSTALL_DIR   where to put the binary (default: ~/.local/bin)
#   VERSION       a release tag, for example v0.1.0 (default: the latest)
#   GITHUB_TOKEN  needed only while the repository is private
set -eu

MODULE="github.com/iAmLakshya/simstim"
REPO="${MODULE#github.com/}"
NAME="${REPO##*/}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

say() { printf '%s\n' "$*" >&2; }
die() { say "Error: $*"; exit 1; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in darwin|linux) ;; *) die "this installer supports macOS and Linux, not $os" ;; esac
arch=$(uname -m)
case "$arch" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) die "unsupported architecture $arch" ;; esac

auth=""
[ -n "${GITHUB_TOKEN:-}" ] && auth="Authorization: Bearer $GITHUB_TOKEN"
get() { curl -fsSL ${auth:+-H "$auth"} "$@"; }

if [ -z "${VERSION:-}" ]; then
  VERSION=$(get "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
  [ -n "$VERSION" ] || die "could not find the latest release (is the repository private? set GITHUB_TOKEN)"
fi
ver="${VERSION#v}"
archive="${NAME}_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
say "Downloading $NAME $VERSION for $os/$arch…"
get -o "$tmp/$archive" "$base/$archive" || die "download failed: $base/$archive"
get -o "$tmp/checksums.txt" "$base/checksums.txt" || die "could not download checksums.txt"

want=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$want" ] || die "no checksum for $archive"
if command -v sha256sum >/dev/null 2>&1; then got=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
else got=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1); fi
[ "$want" = "$got" ] || die "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" "$NAME"
mkdir -p "$INSTALL_DIR"
mv -f "$tmp/$NAME" "$INSTALL_DIR/$NAME"
chmod 755 "$INSTALL_DIR/$NAME"
say "Installed $INSTALL_DIR/$NAME ($VERSION)."
case ":$PATH:" in *":$INSTALL_DIR:"*) ;; *) say "Add $INSTALL_DIR to your PATH to run it." ;; esac
