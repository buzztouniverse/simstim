---
status: accepted
date: 2026-10-15
---

# 0011 The launcher remembers: it draws at once and opens where you left off

## Context

Each start of the launcher dials every host over SSH. Through a jump host that takes a second or two, and until then the list was empty: "Connecting…". Most of the time the owner wants the session they used last, and the launcher made them find it again.

## Options

1. Wait for the hosts, as before.
2. Keep a small memory on the laptop: each host's sessions as last seen, and the session last attached to. Draw it at once, quiet, and replace it when the host answers.
3. A background process on the laptop that keeps the hosts connected. That is faster still, but it adds a daemon, which the project avoids.

## Decision

Option 2. The memory is `launcher.json` in the client's state folder (`~/.local/state/<name>/`), private to the user (mode 0600). It is written when you attach and when the launcher closes.

- Remembered sessions show at once, with their glyph and state faint, while the host connects. The frame's border glows, and the detail line says "last seen before this run".
- The launcher opens on the sessions of the host last attached to, with that session selected. Enter attaches straight away: an attach dials its own connection, so it does not wait for the list.
- `attach -` attaches to the last session from the shell. The command line records its own attaches too.
- The demo never remembers anything, so its screens are the same on every run.

## Consequences

- Good, because `simstim` then Enter is the whole path back into the last session, even before SSH has answered.
- Good, because the launcher never opens on an empty screen once it has seen a host.
- Bad, because for a second or two a remembered session may be gone or changed. It is drawn quiet and marked as last seen. Attaching to a session that has gone fails with a clear error, and the live list replaces the memory as soon as it arrives.
- Bad, because the memory holds session names, folders and the agents' last questions on the laptop. It is a private file in the user's own state folder.
