---
status: accepted
date: 2026-10-15
---

# 0007 The product name lives in one package

## Context

The owner may rename the product. The name appears in binaries, folders on hosts, environment variables, systemd units and the wire protocol.

## Decision

Only `internal/brand` spells the name. Everything else derives it: Go code uses `brand.Name`, `brand.HostBinary` and `brand.Env(...)`, and the justfile and the release workflow read it from that file. `scripts/check-name.sh` fails the build if the name appears anywhere else in code. `scripts/rename.sh` does a rename and lists the manual steps.

## Consequences

- Good, because a rename is one command and a review.
- Bad, because hosts that already run an old helper keep the old folder names until their sessions end.
