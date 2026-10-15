---
status: accepted
date: 2026-10-15
---

# 0002 Our own holder process per session, not zmx

## Context

Something on the host must keep each agent's PTY alive while no client is attached, and replay its output on reattach. zmx does this well, but research found that it has no stable interface, loses exit codes, wipes the terminal's scrollback on detach (RIS), has unbounded client buffers, and its VT engine crashed live sessions twice in 2026.

## Options

1. Depend on zmx at run time
2. Our own small holder per session, with the VT engine in a separate process
3. A central daemon that holds every session

## Decision

Option 2. One holder per session owns the PTY, an output ring with byte offsets, a socket and the manifest. The VT engine (libghostty-vt) runs in a child process, so a crash there cannot kill the agent. There is no central daemon, so one bug cannot take every session down.

## Consequences

- Good, because the interface is ours and versioned.
- Good, because each session is a systemd user service with its own memory cap.
- Bad, because we maintain the PTY and restore logic ourselves. zmx's restore rules serve as a checklist.
