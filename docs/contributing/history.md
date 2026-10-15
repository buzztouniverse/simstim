# How it was built

The first plan, kept for the reasons behind the code. It was written on 14 October 2026 and built over the following days: v0 was a spike, v1 the daily driver, v1.x and v2 the extras. The decisions it asked the owner for became records [0001](decisions/0001-go-and-bubble-tea.md) to [0008](decisions/0008-memory-max-only.md). What comes next is in [the roadmap](../roadmap.md).

## v0: spike (about one week, thrown away)

Prove the feel before building.

- Run zmx on a test host, and a throwaway Go client that runs `zmx attach` over the system `ssh`. Detach by closing the channel, never by zmx's own key: zmx's client writes a full reset (RIS), and that wipes Ghostty's scrollback.
- Check in Ghostty: Shift+Enter, Option keys, truecolour, OSC 8 links, OSC 52 copy, OSC 777 notifications, scrollback, Cmd+F, and resize.
- Record byte fixtures from Claude Code, Codex, opencode, Gemini CLI and Amp, for later tests.
- Measure the delay simstim adds per keystroke.
- Settle decision D1 (language).

**Output:** a spike report and the fixtures. Nothing ships on zmx.

## v1: the daily driver

**Host side (`simstim-host`):**

- One holder process per session. It owns the PTY, a Unix socket, an 8 MiB raw output ring with byte offsets, the manifest and the status engine.
- A shadow terminal (libghostty-vt) in a separate child process. It restores the screen on reattach and answers terminal queries while no client is attached. If it crashes, the holder restarts it from the ring, and the agent never notices.
- Each session runs as a systemd user service with `MemoryHigh` and `MemoryMax`. Linger must be on. Hosts without systemd get a daemonised holder and a memory watchdog.
- The environment comes from the user's login shell, so agents installed with nvm, mise or bun are found. Ghostty's terminfo is installed at host setup.
- Status layers L0 (process), L1 (the agent's own reports: OSC 7501 and hooks) and L2 (terminal hints: OSC 9;4, title, bell, notifications).
- Image staging: a private folder per session, files deleted after 24 hours.
- Commands: `bridge`, `new`, `attach`, `ls --json`, `watch --json`, `hook`, `kill`, `doctor`.

**Client side (`simstim`):**

- SSH through the system `ssh`. A generated config file includes the user's own config, so ProxyJump, IAP and SSM work. A private ControlMaster makes calls fast.
- One bridge channel per client process, with a heartbeat: 5 s pings, a 15 s deadline.
- Host setup over SSH: find the platform, ask before the first install, upload a static helper, check its SHA-256, install into a versioned folder, and handshake.
- Attach pump:
  - raw bytes in both directions;
  - an input filter that matches only the detach key and Ctrl+V, in every key encoding;
  - a passive mode tracker on the output;
  - an ordered reset list on detach, which never sends RIS.
- Reconnect after sleep or a network change: detect it fast, reconnect with backoff, and replay missed output from the last byte offset.
- Image paste and dropped files: upload over the same connection, then paste the remote path as one bracketed paste.
- The screens: setup wizard, host browser, session list with a live preview, new-session dialog, command palette, help, toasts. (Rebuilt later as a light launcher, one list at a time: [0009](../contributing/decisions/0009-light-launcher.md).)
- CLI: `--json` on every list, `simstim doctor` (including a key tester), and shell completions.

**Packaging:**

- Clients for macOS and Linux, on amd64 and arm64. Linux hosts on amd64 and arm64.
- GoReleaser, a Homebrew tap, a `curl | sh` installer and a Nix flake. Signed and notarised on macOS, with checksums and an SBOM.

## v1.x

- **Alerts for other sessions:** while you are attached to one session, show a native notification when another session needs you.
- **L3 status:** screen rules for each agent, kept as data. Herdr's rules can be reused with attribution.
- **Hosts:** macOS hosts.
- **Upgrades:** in-place holder upgrade (re-exec with the PTY kept open), so sessions survive a helper upgrade.
- **Several devices on one session:** a `take_size` rule decides which device sets the window size.

## v2: designed for now, built later

- **Respawn after a host reboot:**
  - The manifest stores each agent's resume command and its session ID, captured at launch or by a hook.
  - At boot, a user unit runs `simstim-host restore`, which starts each session again in its folder. Above the resumed agent it shows the old screen.
- **A git worktree per task:** created on request, branch `simstim/<name>`, locked while the session lives, and removed only when it is clean or merged.
- **Ports:** detect ports that a session's processes listen on, and offer a forward to `localhost` on the laptop.
- **Host scripts that work for any provider:** user scripts for `create`, `start`, `stop`, `delete` and `ssh-config`, with the job as JSON on stdin. With them, simstim wakes a cloud VM on attach and stops it after an idle rule. This is the same contract as Amp's self-hosted orb provider.
- **Setup and resume hooks** per host and per repo. They must be safe to run twice, have timeouts and write logs.
- **A Windows client**, with Alt+V for image paste.
- **Scripting:** `simstim new --detach` and `simstim send`.

## What v1 does now so that v2 stays cheap

| v2 feature | Prepared in v1 by |
| --- | --- |
| Respawn after reboot | Manifest schema 1 already has `resume_argv`, `continue_argv`, `agent_session_id`, `boot_id` and `restart`. The agent table has resume commands from day one. Holders save a screen snapshot when idle. |
| Worktree per task | The manifest has a `git` block. Session folders are per session. |
| Ports | The holder is a child subreaper, so it can map processes to a session. |
| Host scripts | A host record has an optional `provider` block. The client already checks reachability before any call. |
| Hook capture | `simstim-host hook` and `SIMSTIM_SESSION` exist in v1 for status, so ID capture is one more event. |
| Upgrades | The protocol is append-only, with a `hello` and feature flags. Binaries live in versioned folders. |

## Later, maybe

- Sync a session's work back to the laptop (git or rsync).
- Read-only observers of a session.
- A phone client. It is out of scope today.

## Milestones

Each milestone ends with its exit test passing. Sizes are relative (S, M, L), not dates.

| # | Milestone | Size | Exit test |
| --- | --- | --- | --- |
| M0 | Spike (v0) | S | The checklist in v0 passes in Ghostty. Fixtures recorded. D1 decided. |
| M1 | Host runtime: holder, vt child, protocol, manifests, systemd caps, `new`, `ls`, `kill`, `attach` | L | Replay tests pass: each fixture fed through snapshot-then-replay matches a reference terminal cell for cell. A killed client leaves the agent running. |
| M2 | Transport and attach: ssh config, master, bridge, bootstrap, attach pump, filter, reset list, reconnect, replay | L | Close the lid for an hour. On wake, the session is back in under 5 s and the missed output is in local scrollback. `simstim doctor keys` shows Shift+Enter unchanged through ProxyJump. |
| M3 | Screens: wizard, host browser, session list, preview, palette, help, toasts, errors | M | The owner's spec items 1–6 work end to end from the TUI. Golden tests for each screen. |
| M4 | Image paste, file drops, status L0–L2, doctor, `--json`, completions | M | Pasting a screenshot works in Claude Code, Codex and opencode on a remote host. The list shows "needs input" within 2 s of the agent asking. |
| M5 | Hardening and v1.0 release | M | The test matrix in section 9 passes. Notarised builds. Docs. |
| M6 | v1.x | M | As in v1.x. |
| M7 | v2 | L | Reboot the host: each session comes back with its conversation. Worktree and port features work. |
