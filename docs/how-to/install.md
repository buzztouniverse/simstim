# Install

## With the installer (macOS and Linux)

<!-- test: skip (needs a published release; scripts/test-install.sh tests the installer) -->
```sh
curl -fsSL https://raw.githubusercontent.com/iAmLakshya/simstim/main/install.sh | sh
```

It downloads the release for your system, checks its signature with `ssh-keygen` and its SHA-256, and installs into `~/.local/bin`. It changes nothing else. Options, as environment variables:

| Variable | Does |
| --- | --- |
| `INSTALL_DIR` | Install somewhere else |
| `VERSION` | A release tag, such as `v0.1.0`, instead of the latest |

## With Go

<!-- test: skip (installs into your GOBIN) -->
```sh
go install github.com/iAmLakshya/simstim@latest
```

## With Homebrew

The tap opens with the first public release.

## From source

```sh
git clone https://github.com/iAmLakshya/simstim
cd simstim
scripts/setup.sh
just build
```

The client is `bin/simstim`.

## Check a release by hand

[Releasing](../contributing/releasing.md#checking-a-release) shows the three ways: the SSH signature, the cosign bundle and the GitHub attestation.

## Update

Run the installer again. The next time the client connects to a host, it installs the matching helper beside the old one; sessions that are running keep the helper they started with.
