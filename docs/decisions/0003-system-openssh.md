---
status: accepted
date: 2026-10-15
---

# 0003 The system OpenSSH client as the only transport

## Context

Hosts live on any provider, behind ProxyJump, Tailscale, GCP IAP or AWS SSM. Users have keys in agents such as 1Password and Secretive, FIDO keys and certificates.

## Options

1. Run the system `ssh`, with a generated config that includes the user's own
2. An embedded SSH library (Go x/crypto/ssh, Rust russh)
3. mosh, Eternal Terminal or tsshd

## Decision

Option 1. Only OpenSSH honours the whole `~/.ssh/config` and every kind of key. Option 3 needs open ports on the host, and mosh breaks the kitty keyboard protocol and scrollback. The client owns a private ControlMaster per host for speed, and never edits `~/.ssh/config`.

## Consequences

- Good, because "if `ssh host` works, it works" is true.
- Good, because nothing new listens on the host.
- Bad, because TCP dies on a network change, so the client must reconnect and replay. Decision 0004 covers that.
