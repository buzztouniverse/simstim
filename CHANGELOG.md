# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- The plan, architecture, UX and research notes.
- Project scaffolding: licence, community files, CI, release configuration and decision records.
- Host runtime: one holder process per session, with a PTY, an 8 MiB output ring with byte offsets, a manifest, and status from output signals. Each session runs as a systemd user service with a memory cap.
- Host helper commands: `bridge`, `new`, `ls`, `rm`, `doctor`, `version`.
- The shadow terminal (libghostty-vt in its own process): a new client gets a snapshot of the screen, scrollback and modes instead of raw history, and queries are answered while no client is attached. A crash restarts it; the agent never notices.
- Wire protocol with a versioned hello, requests, attach streams and uploads.
- Terminal mode tracking, reset without RIS, restore on reattach, and query removal from replayed history.
- The input filter for the detach and image-paste keys, in legacy, kitty and modifyOtherKeys encodings.
- The client: system OpenSSH with a generated config and a private ControlMaster, helper install with consent and SHA-256 checks, automatic helper upgrades for saved hosts, the bridge connection, and the attach pump with heartbeat, reconnect and replay by byte offset.
- Image paste from the clipboard and dropped-file upload, into a private per-session folder.
- The launcher: the hosts, then one host's sessions, one list at a time, with the new-session dialog (agent, folder, name, worktree), rename, delete with a confirm, a preview on Space, the add-host wizard, a command palette, shortcuts and a first-run screen. Colours follow the terminal's own palette. Attach and detach keep the terminal's scrollback continuous within one run.
- Folder completion in the new-session dialog: recent session folders and git repositories as hints, and Tab completing a typed path from the host.
- `doctor terminal`: one batch of queries (DA1 last) that reports what the terminal supports, which is what agents get through the pass-through.
- `flip`: a quick switcher that opens on every host's sessions. Host details can open an SSH shell on the host (`s`).
- Running any command from the new-session dialog ("other…").
- Key rebinding through `[keymap]` in `config.toml`, and `config keymap` to print it.
- `demo`: the launcher with sample hosts and sessions, where nothing connects anywhere, and a made-up agent that fills the terminal when you attach. `just screenshots` uses it to make the README's screenshots, which `ttyshot` draws as SVG from the terminal's cells.
- CLI commands: `host add/ls/rm/test/doctor`, `new`, `ls`, `attach`, `kill`, `rename`, `doctor keys`, `version`.
- End-to-end tests that drive the real client in a pseudo-terminal against a real host (`just e2e`).
- Agent status hooks, as data in the agent table: Claude Code through `--settings`, Codex through `notify`. They report prompts, finished turns, notifications and the agent's own session ID (which follows `/clear`).
- `send`: type text into a session without attaching.
- A git worktree per session (`new --worktree [--base ref]`): its own branch, locked while the session lives, `.worktreeinclude` files copied, removed on delete only when nothing would be lost.
- Setup and resume hooks: an executable `.<name>/setup` in the repository (or `~/.config/<name>/hooks/setup` on the host) runs in the session's terminal before the agent; `resume` runs before a respawn.
- `ports` and `forward`: find the ports a session listens on, and forward them to localhost through SSH.
- Provider scripts: `host add --provider-script <path> --wake`, and `host start|stop|status`. A host that does not answer is started first. Examples for GCP and AWS.
- Status from screen rules, kept as data (one TOML file per agent, patterns adapted from Herdr): the holder keeps a rendered copy of the screen and the rules tell working from waiting for the user. Sessions say where their state came from.
- Alerts while attached: a desktop notification when another session on the host needs input or ends.
- `--dir ~/…` on the command line maps the local home folder to the host's.
- Respawn after a host restart: `restore` brings sessions back with each agent's resume or continue command, and a boot-time user unit runs it. `host restore` runs it on demand.

### Changed

- Sessions are capped with `MemoryMax` only; `MemoryHigh` made a runaway process hang (decision 0008).
- Commits carry no sign-off or attribution lines, and contributions are covered by Apache-2.0 section 5 (decision 0010, which replaces the DCO).
- The home screen is now a light launcher in the style of Amp's CLI (decision 0009). It has one column and one list: the hosts, then Enter for a host's sessions. The panes and the permanent preview are gone. The look is measured from Amp: notched frames, yellow titles, blue keys, faint secondary text and a yellow selection bar, all from the terminal's palette. Ctrl+C opens a chord popup, and a second Ctrl+C quits.
- While attached, the client draws nothing of its own. It sets no window title, and a failed upload is now a system notification instead of a line on the bottom row. The reconnect notice is the one exception. End-to-end checks prove that the agent's PTY is the terminal's size, and that the screen is the session's bytes alone.
