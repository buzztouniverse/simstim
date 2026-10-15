---
status: accepted
date: 2026-10-15
---

# 0017 Release signatures that any machine with OpenSSH can check

## Context

The install script runs as `curl … | sh`. A checksum alone proves only that the archive matches a file from the same place, so a changed release could carry a changed `checksums.txt`. The script must also check a signature. It runs on a clean Mac or Linux box, which has OpenSSH, `curl` and `tar`, but no cosign, no `gh` and no GPG keys.

## Options

1. **cosign keyless only.** No key to keep, and every signature is logged in Rekor. But checking a Sigstore bundle needs cosign, a binary of tens of megabytes that a clean machine lacks, and is too much to do in `sh`.
2. **GitHub attestations only.** Checking needs `gh`, logged in.
3. **GPG.** Few machines have GPG, and its keyrings are hard to use from a script.
4. **An SSH signature** made with `ssh-keygen -Y sign` in the release job, from a key kept only as an Actions secret, and checked with `ssh-keygen -Y verify` against a public key in the repo and in the script.

## Decision

Option 4, with options 1 and 2 as further checks. Every machine that can use simstim has `ssh-keygen`, since simstim needs OpenSSH, so the script checks the signature everywhere and fails closed. cosign and GitHub attestations remain for anyone who wants a transparency log.

The owner agreed on 15 October 2026 that the key be made and kept only as the secret `RELEASE_SSH_KEY`. No other copy exists. The public key is `.github/release-signers`, and `install.sh` holds the same line.

## Consequences

- Good, because `curl … | sh` checks a signature on a clean machine, with nothing to install first.
- Good, because the key never leaves GitHub's secret store, and the release job checks its own signature before it uploads it.
- Bad, because a key kept by GitHub is only as safe as the repository's admin accounts and workflows. Pinned actions, least-privilege tokens and zizmor ([CI](../../.github/workflows/ci.yml)) reduce that risk.
- Bad, because rotating the key means a new key in the secret, a new `.github/release-signers`, and a new `install.sh`. Old releases then need the old key: the release notes say which key signed them.
