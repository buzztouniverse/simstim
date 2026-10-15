# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- The plan, architecture, UX and research notes.
- Project scaffolding: licence, community files, CI, release configuration and decision records.
- Host runtime: one holder process per session, with a PTY, an 8 MiB output ring with byte offsets, a manifest, and status from output signals. Each session runs as a systemd user service with a memory cap.
- Host helper commands: `bridge`, `new`, `ls`, `rm`, `doctor`, `version`.
- Wire protocol with a versioned hello, requests, attach streams and uploads.
- Terminal mode tracking, reset without RIS, restore on reattach, and query removal from replayed history.
- The input filter for the detach and image-paste keys, in legacy, kitty and modifyOtherKeys encodings.
- The client: system OpenSSH with a generated config and a private ControlMaster, helper install with consent and SHA-256 checks, automatic helper upgrades for saved hosts, the bridge connection, and the attach pump with heartbeat, reconnect and replay by byte offset.
- Image paste from the clipboard and dropped-file upload, into a private per-session folder.
- CLI commands: `host add/ls/rm/test/doctor`, `new`, `ls`, `attach`, `kill`, `rename`, `doctor keys`, `version`.
- End-to-end tests that drive the real client in a pseudo-terminal against a real host (`just e2e`).
- Agent status hooks, as data in the agent table: Claude Code through `--settings`, Codex through `notify`. They report prompts, finished turns, notifications and the agent's own session ID (which follows `/clear`).
- `send`: type text into a session without attaching.
- A git worktree per session (`new --worktree [--base ref]`): its own branch, locked while the session lives, `.worktreeinclude` files copied, removed on delete only when nothing would be lost.
- Setup and resume hooks: an executable `.<name>/setup` in the repository (or `~/.config/<name>/hooks/setup` on the host) runs in the session's terminal before the agent; `resume` runs before a respawn.
- `ports` and `forward`: find the ports a session listens on, and forward them to localhost through SSH.
- Provider scripts: `host add --provider-script <path> --wake`, and `host start|stop|status`. A host that does not answer is started first. Examples for GCP and AWS.
- Alerts while attached: a desktop notification when another session on the host needs input or ends.
- `--dir ~/…` on the command line maps the local home folder to the host's.
- Respawn after a host restart: `restore` brings sessions back with each agent's resume or continue command, and a boot-time user unit runs it. `host restore` runs it on demand.

### Changed

- Sessions are capped with `MemoryMax` only; `MemoryHigh` made a runaway process hang (decision 0008).
