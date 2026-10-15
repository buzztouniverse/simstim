# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- `w` in the launcher goes to the next session that needs you, on any host, and says what it waits for; pressing it again goes to the one after (roadmap N3).
- `go install github.com/iAmLakshya/simstim@latest` gives a working client: with no helpers inside it, it fetches its release's helper bundle (a new release asset, `helpers_<version>.tar.gz`), checks the release's SSH signature and the bundle's SHA-256, and caches it. Its version comes from Go's build information.
- Recorded demos from tapes: `docs/demos/*.tape` drive the demo in `test/ttyshot`, which now records an animated SVG and an asciicast (`record start`, `record save`, and a `type` key that types like a person). `just screenshots` makes them with the stills. The README opens with the launcher demo and shows a session being joined and left.
- User docs in the Diátaxis shape: a getting-started tutorial, and how-to guides to install, add a host, use worktrees, open ports, paste images, restore after a reboot, change keys, troubleshoot and uninstall. Every command in them is tested (`just test-docs`, and in the docs checks).
- `just public-check` (also in CI on every push): the name check, no private details in the tree or the whole history (`check-private.sh --history`), no secrets in the history (gitleaks), licence information on every file (REUSE: `REUSE.toml` and `LICENSES/`), the workflows (zizmor), the docs checks, and the Scorecard checks the repository controls. It passes.
- Community files for a public repository: SUPPORT, GOVERNANCE and MAINTAINERS; issue forms for provider requests and docs problems; Discussion forms for ideas and questions; `.gitattributes` that marks generated files. AGENTS.md is shorter and points to the contributor docs, which gain a home page, a guide to how the docs work, and a style guide.
- A docs site: Hugo with the Hextra theme (`site/`), built from `docs/` with search and light and dark themes (`just docs-site`). CI builds it on every push and keeps it as an artifact; once the repository is public, GitHub Pages serves the latest release at the root, each release at `/vX.Y/` and `main` at `/dev/`, and each release carries its own docs.
- Docs checks (`just docs-check`, and CI): the generated reference is current, Markdown lint, British spelling (typos), plain English (Vale, with a small style of our own), and every internal link on the built site; links to other sites are checked on `main` and weekly.
- `scripts/setup.sh --only <tools>` installs only some tools, for CI; Hugo is built with `go install` at a pinned version.
- Reference docs generated from the code (`just docs`, with `cmd/docgen`): every command and flag, `config.toml` and `hosts.toml` from the settings structs' comments, the launcher's keys from its command registry, and the agent table. CI fails when they are stale. Releases now include man pages and shell completions.
- A docs-impact check: CI fails a pull request that changes commands, config, keys or agents without changing the docs and the CHANGELOG, unless it is labelled "no docs needed".
- Alerts with the laptop shut (decision 0019): each session's holder sends `needs_input`, `turn_done`, `failed`, `oom` (and on request `exited`, `started`, `restored`) to destinations you name in `~/.config/<name>/alerts.toml` on the host: an ntfy topic, a webhook (JSON, Slack or Discord) or a command, which gets the event as JSON on stdin and in environment variables. Nothing is sent until you name one. `needs_input` waits 10 s and stays quiet if the session moves on or you type; each session sends one event of a kind a minute, and a host 20 a minute. The agent's question is sent only if you turn `detail` on. `host alerts <host>` shows the settings and last deliveries, and `host alerts set|test|log|off` change, test, explain and remove them; `--token-cmd` takes an ntfy token from a command on the laptop, such as a keychain read, without printing it. See [docs/how-to/alerts.md](docs/how-to/alerts.md).
- Antigravity CLI (`agy`) in the agent table: Google moved Gemini CLI's consumer users to it on 18 June 2026.
- A support level for each agent: Claude Code and Codex are first-class (tested end to end on every release, with status hooks and resume), opencode, Antigravity CLI and the shell are supported, and the rest are best effort. The README lists them.
- A roadmap that scores every candidate feature by demand, impact and effort ([docs/roadmap.md](docs/roadmap.md)). Next and later items are GitHub issues, eleven of them good first issues, and the labels are kept as code in `.github/labels.yml` (synced by `scripts/sync-labels.sh` and the Labels workflow).
- One-command developer setup: `scripts/setup.sh` (or `just setup`) installs pinned, checksum-checked tools into `.tools/`, then builds and tests once. A devcontainer runs the same script, and CI checks both weekly and on every change to them.
- A versioning and release policy, with the steps to release and to check a release ([docs/contributing/releasing.md](docs/contributing/releasing.md)).
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
- `just check-private` (also in `just lint` and CI): fails when a tracked file holds a personal path, a private or Tailscale address, a tailnet name or an email. Maintainers can add private words of their own in the git-ignored `private/deny.txt`.

### Changed

- `go install github.com/iAmLakshya/simstim@latest` installs the client under its own name: its main moved to the root of the module, and the command tree to `internal/cli`.
- CI is hardened: every action is pinned to a full commit SHA, each job asks for the least token permissions, the checkout keeps no credentials, and zizmor lints the workflows. govulncheck runs on every push, and the toolchain is Go 1.26.9, which fixes two standard-library issues it found.
- Releases are signed and traceable: cosign signs `checksums.txt` without a key (a Sigstore bundle), each archive gets an SBOM, a GitHub attestation and a SLSA level 3 provenance file, and the notes come from this CHANGELOG (`scripts/release-notes.sh`). A Homebrew cask is built with every release, ready for a tap.
- Builds are reproducible: `just repro` (also in CI) builds every release binary twice from empty caches and checks they match byte for byte.
- CodeQL and OpenSSF Scorecard run once the repository is public; `just scorecard` scores it before then.
- SECURITY.md has a reporting link and a disclosure timeline.
- `install.sh` checks the release's signature before the checksum: the release job signs `checksums.txt` with an SSH key, and the script verifies it with `ssh-keygen`, which every machine with OpenSSH has. The public key is in `.github/release-signers`. `just test-install` (also in CI, on Linux and macOS) installs a signed snapshot release and checks that changed checksums, signatures and archives are refused.
- The launcher fits a 60×20 terminal: below 70 cells a session list drops its folder column instead of squeezing it to two letters and spilling past the margin. Golden screens now cover 60×20 too.
- Tests render the screens in every colour profile (truecolour, 256, 16, `NO_COLOR`, no terminal) and check the contrast of the fixed colours on dark and light backgrounds.
- The repo holds nothing private: the working goal, the status page, the test-host log and the owner's checklist moved to the git-ignored `private/` folder, and examples use generic host names, users and documentation addresses.
- Sessions are capped with `MemoryMax` only; `MemoryHigh` made a runaway process hang (decision 0008).
- Commits carry no sign-off or attribution lines, and contributions are covered by Apache-2.0 section 5 (decision 0010, which replaces the DCO).
- The home screen is now a light launcher (decisions 0009 and 0012). It has one column and one list: the hosts, then Enter for a host's sessions. The panes and the permanent preview are gone. The look is its own:
  - a header with an animated braille logo and the name in the brand gradient, and rules instead of boxes;
  - a cyan accent, and a cyan marker and tint on the selected row;
  - a braille trace of each session's recent output;
  - lower-case keys;
  - sheets that rise from the bottom for every dialog.

  Amp's CLI was the baseline, measured in docs/research/amp-tui.md: colours from the terminal's palette, faint secondary text and a command palette.
- The launcher moves, in the terminal's own colours:
  - on the welcome screen, a woven braille ribbon that opens out and flows, and a name that decodes out of noise;
  - a boot sequence, screens that slide, and a selection that glides;
  - host names and activity that decode in;
  - a scan along the header rule while something is busy, and a short jack-in sweep before an attach;
  - notices that slide and fade, a shimmer on work in progress, a breathing dot for a session that needs you, and a tiny Game of Life for one that works.

  It asks the terminal for its colours (OSC 4, 10 and 11) to blend smoothly, and falls back to faint and bold steps. One frame tick runs only while something moves, and `ui.motion = off` stills it all.
- The launcher is faster to use (decision 0011):
  - it remembers each host's sessions and draws them at once, before SSH answers, and opens on the session you last attached to;
  - `attach -` does the same from the shell;
  - `/` searches the sessions of every host, and `n` starts a session from one line (`claude ~/src/app auth`) with completion;
  - a live activity line says what the selected session is doing now, read from its screen;
  - rows have numbers (1–9 open them), and the mouse selects, opens and scrolls;
  - the tab title counts the sessions that need you, and a desktop notification goes out when one starts to need you while the terminal is in the background.
- While attached, the client draws nothing of its own. It sets no window title, and a failed upload is now a system notification instead of a line on the bottom row. The reconnect notice is the one exception. End-to-end checks prove that the agent's PTY is the terminal's size, and that the screen is the session's bytes alone.

### Fixed

- `scripts/setup.sh` had the same ellipsis bug as `install.sh`; `just lint` now checks every script for it.
- `install.sh` stopped with "unbound variable" in shells that read the `…` after `$arch` as part of the name.
- On a light background, the end of the brand gradient showed pink: blended colours with a channel near zero were sent as their 16-bit value. The gradient is now sent as plain 8-bit RGB.
