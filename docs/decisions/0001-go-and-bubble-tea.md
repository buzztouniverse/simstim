---
status: accepted
date: 2026-10-15
---

# 0001 Go for the client and host helper, Bubble Tea v2 for the screens

## Context

The client needs polished screens, a raw byte pump for attach mode, SSH plumbing, and static binaries for every host platform. The host helper must be one static Linux binary that the client uploads.

## Options

1. Go, with Bubble Tea v2, Lip Gloss v2 and Bubbles v2
2. Rust, with Ratatui, and code borrowed from Herdr (Apache-2.0)
3. TypeScript on Bun, with OpenTUI or Ink
4. Zig, with libvaxis

## Decision

Option 1. Go cross-compiles static binaries for every target in one command, has the SSH and terminal-parsing libraries we need, and starts in about 4 ms. Bubble Tea v2 has been stable since February 2026 and Crush shows its visual ceiling. The shadow terminal (libghostty-vt) will be a small separate program, so its language does not bind the rest.

## Consequences

- Good, because one toolchain builds the client and the helper for every platform.
- Good, because the contributor pool is large.
- Bad, because Herdr's Rust modules cannot be reused directly; they serve as a reference.
- Bad, because a chat-grade text editor would be custom work (we do not need one).
