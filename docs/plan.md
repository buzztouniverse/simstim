# simstim plan

**Status:** accepted and built, 15 October 2026. v1, v1.x and v2 are implemented; progress against the goal, with evidence, is in [the status](status.md). The plan below is kept as written, for the reasons behind the design.

This page is the plan. [Architecture](architecture.md) says how it works, [UX](ux.md) says how it looks and feels, and [the research notes](research/README.md) hold the evidence.

## 1. In one paragraph

simstim lets you run any coding agent in a session on a host you own, on any provider, and use it from your laptop as if it ran there. The agent can be Claude Code, Codex, opencode, Gemini CLI, Amp or a plain shell. Close the laptop, change networks or switch devices: the session keeps running on the host. When you come back, you flip straight back in. Your own terminal draws the agent's real screen, byte for byte, so keys, colours, scrollback, clipboard and notifications work as they do locally.

The name comes from William Gibson's *Neuromancer* (1984). Simstim lets Case flip into Molly's senses, live, from far away, as if he were there.

## 2. Why build it

Today all work is tied to one laptop. If the lid closes or the owner travels, the agent stops.

Research in October 2026 found no tool that meets the whole brief:

| Tool | What breaks the brief |
| --- | --- |
| tmux, Zellij, screen | Their own scrollback replaces the terminal's (no Cmd+F). Their prefix keys clash with agent keys. tmux has no kitty keyboard protocol. |
| mosh | No kitty keyboard protocol (Shift+Enter becomes Enter), no scrollback, no OSC 9/777 notifications. |
| Herdr | A full multiplexer. It redraws panes from its own terminal model, direct attach uses the alternate screen, and it drops OSC 52 and OSC 9/777. Pull requests from outside are closed automatically. |
| Web and phone UIs (CloudCLI, Happy, Paseo) | They replace the agent's real interface. Several had critical security bugs in 2026. |
| Amp Orbs, Claude Remote Control, Codex remote | One vendor's cloud and one agent each. |
| Coder, DevPod, Daytona | Coder's agent features now need its own agent and API keys. DevPod has stalled and Daytona is archived. |

The gap: **any agent, on any host you can reach by SSH, with the agent's bytes passed through untouched, sessions that survive sleep, and image paste that works remotely.**

## 3. Goals

The owner's spec, as given:

1. Install the CLI on any device.
2. At setup, connect a host: a VM on any provider (GCP, AWS, homelab, a VPS).
3. Browse connected hosts.
4. Select a host and press Enter to start a session.
5. View running sessions, delete them, select one and interact with it.
6. After exit, the session keeps running.
7. Run Codex, Claude Code or any other agent, with full access to the session runtime, as if it ran on the device.
8. See the actual colours and TUI, as if it ran natively.

Added by this plan:

9. The client reconnects by itself after sleep or a network change, and shows the output that was missed.
10. Ctrl+V pastes a screenshot from the laptop's clipboard into a remote agent.
11. The session list shows which sessions are working, which need input and which have finished.
12. One command attaches straight to a session, for scripts and terminal keybindings.
13. One agent cannot take down the host: each session has a memory cap.
14. simstim's own screens feel as good as the Amp CLI: calm, fast, keyboard first.

## 4. Non-goals

- **Not a terminal multiplexer.** No panes, splits or tabs. The terminal does that. Amp removed its TUI sidebar in August 2026 for the same reason.
- **Not an agent or an agent wrapper.** simstim never re-renders an agent, sends prompts for it or changes its permissions.
- **No cloud service.** No account, no relay and no telemetry.
- **No phone or web client** in v1 or v2.
- **No VM provisioning in v1.** v2 adds optional host scripts that the user writes.
- **No promise that a process survives a host reboot.** v2 brings the conversation back, not the turn that was running.

## 5. Principles

Ranked. Use them to settle trade-offs.

1. **Byte-exact while attached.** simstim adds no modes, colours or key handling. It watches for two keys only: detach, and Ctrl+V when the clipboard holds an image.
2. **The session outlives the client.** A closed tab, a crash, sleep or a network change never stops a session.
3. **Plain OpenSSH is the only network path.** If `ssh host` works, simstim works.
4. **Any agent.** What simstim knows about each agent is data in a table, not code paths.
5. **Nothing listens and nothing phones home.** No open ports on hosts, no relay, no telemetry, no stored secrets.
6. **Amp-grade screens.** Colours come from the terminal's own palette. Keyboard first. Every error suggests a next step.
7. **Small and boring.** One binary on each side. Files, not databases. Everything carries a version.

## 6. Scope by version

### v0: spike (about one week, thrown away)

Prove the feel before building.

- Run zmx on a test host, and a throwaway Go client that runs `zmx attach` over the system `ssh`. Detach by closing the channel, never by zmx's own key: zmx's client writes a full reset (RIS), and that wipes Ghostty's scrollback.
- Check in Ghostty: Shift+Enter, Option keys, truecolour, OSC 8 links, OSC 52 copy, OSC 777 notifications, scrollback, Cmd+F, and resize.
- Record byte fixtures from Claude Code, Codex, opencode, Gemini CLI and Amp, for later tests.
- Measure the delay simstim adds per keystroke.
- Settle decision D1 (language).

**Output:** a spike report and the fixtures. Nothing ships on zmx.

### v1: the daily driver

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
- The screens: setup wizard, host browser, session list with a live preview, new-session dialog, command palette, help, toasts. (Rebuilt later as a light launcher, one list at a time: [0009](decisions/0009-light-launcher.md).)
- CLI: `--json` on every list, `simstim doctor` (including a key tester), and shell completions.

**Packaging:**

- Clients for macOS and Linux, on amd64 and arm64. Linux hosts on amd64 and arm64.
- GoReleaser, a Homebrew tap, a `curl | sh` installer and a Nix flake. Signed and notarised on macOS, with checksums and an SBOM.

### v1.x

- **Alerts for other sessions:** while you are attached to one session, show a native notification when another session needs you.
- **L3 status:** screen rules for each agent, kept as data. Herdr's rules can be reused with attribution.
- **Hosts:** macOS hosts.
- **Upgrades:** in-place holder upgrade (re-exec with the PTY kept open), so sessions survive a helper upgrade.
- **Several devices on one session:** a `take_size` rule decides which device sets the window size.

### v2: designed for now, built later

- **Respawn after a host reboot:**
  - The manifest stores each agent's resume command and its session ID, captured at launch or by a hook.
  - At boot, a user unit runs `simstim-host restore`, which starts each session again in its folder. Above the resumed agent it shows the old screen.
- **A git worktree per task:** created on request, branch `simstim/<name>`, locked while the session lives, and removed only when it is clean or merged.
- **Ports:** detect ports that a session's processes listen on, and offer a forward to `localhost` on the laptop.
- **Host scripts that work for any provider:** user scripts for `create`, `start`, `stop`, `delete` and `ssh-config`, with the job as JSON on stdin. With them, simstim wakes a cloud VM on attach and stops it after an idle rule. This is the same contract as Amp's self-hosted orb provider.
- **Setup and resume hooks** per host and per repo. They must be safe to run twice, have timeouts and write logs.
- **A Windows client**, with Alt+V for image paste.
- **Scripting:** `simstim new --detach` and `simstim send`.

### What v1 does now so that v2 stays cheap

| v2 feature | Prepared in v1 by |
| --- | --- |
| Respawn after reboot | Manifest schema 1 already has `resume_argv`, `continue_argv`, `agent_session_id`, `boot_id` and `restart`. The agent table has resume commands from day one. Holders save a screen snapshot when idle. |
| Worktree per task | The manifest has a `git` block. Session folders are per session. |
| Ports | The holder is a child subreaper, so it can map processes to a session. |
| Host scripts | A host record has an optional `provider` block. The client already checks reachability before any call. |
| Hook capture | `simstim-host hook` and `SIMSTIM_SESSION` exist in v1 for status, so ID capture is one more event. |
| Upgrades | The protocol is append-only, with a `hello` and feature flags. Binaries live in versioned folders. |

### Later, maybe

- Sync a session's work back to the laptop (git or rsync).
- Read-only observers of a session.
- A phone client. It is out of scope today.

## 7. Milestones

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

## 8. Decisions for the owner

Each is proposed. Once decided, write it as a decision record in `docs/decisions/`.

| # | Decision | Recommendation | Why, in short |
| --- | --- | --- | --- |
| D1 | Language | **Go** for the client and the host helper, with Bubble Tea v2 for the screens. The shadow terminal is a small Zig program on libghostty-vt, adapting zmx's MIT-licensed restore code. | Go makes one-command static builds for every target, and is strong for SSH and terminal parsing. Bubble Tea v2 (stable since February 2026) gives Amp-level polish, as Crush shows. Startup is about 4 ms. Zig is needed to build libghostty-vt anyway. **The serious alternative is Rust**: Herdr's Apache-2.0 modules (SSH bridge, reconnect, image paste, resume table, detection rules) could be reused almost directly, and there is a Rust binding for libghostty-vt. Choose Rust if reuse beats Go's simpler builds and wider contributor pool. |
| D2 | Session runtime | Our own holder per session. zmx only for the spike and as a reference. | zmx has no stable interface, loses exit codes, wipes scrollback on detach, has unbounded client buffers, and its VT engine crashed live sessions twice in 2026. |
| D3 | Transport | System OpenSSH, a private ControlMaster, one bridge channel per client process. No mosh, Eternal Terminal or tsshd. | Only OpenSSH honours the whole `~/.ssh/config` and every key type. The others need open ports on the host, and mosh breaks fidelity. |
| D4 | Scrollback on reattach | Replay from the client's last byte offset when it is still in the ring, so nothing is duplicated. Otherwise redraw from a snapshot under one dim separator line. A setting `attach.scrollback = keep \| clear`. | Keeps native scrollback continuous on the same device, and stays honest on a new device. |
| D5 | Licence | Apache-2.0 with a DCO | Patent grant, common for developer tools, and compatible with borrowing from Herdr (Apache-2.0) and zmx (MIT). |
| D6 | macOS signing | Get an Apple Developer ID (about $99 a year) | Homebrew now effectively requires signed and notarised casks. |
| D7 | Visibility | Keep the repo private until the clean-room review in section 11 is done | The Amp research note comes from analysing a proprietary binary. |
| D8 | Name | Keep "simstim". Check trademarks before 1.0. | It was free on GitHub, crates.io, npm and Homebrew on 14 October 2026. "Calcifer" was the first idea, but a pre-alpha coding-agent wrapper already uses it on crates.io. |
| D9 | Defaults | Detach on Ctrl+\ (press twice to send it to the agent). Image paste on Ctrl+V (Alt+V on Windows). Memory cap of 8 GiB or 40 % of the host's RAM, whichever is smaller. | Ctrl+\ clashes with no agent's defaults. The cap stops a leaking agent before it starves the host. |

## 9. Testing

- **Fidelity fixtures:** recorded agent byte streams, fed through snapshot-then-replay and compared cell by cell with a reference terminal. They run on every Ghostty pin bump and every release.
- **Input filter:** table tests, a property test that splits streams at random points, and Go fuzzing.
- **Mode reset:** after detach, a Go terminal emulator (`x/vt`) must be back to its baseline modes.
- **Screens:** golden files with teatest v2, and VHS tapes for the README.
- **End-to-end matrix:**

| Dimension | Values |
| --- | --- |
| Agents | Claude Code, Codex, opencode, Gemini CLI, Amp, a shell |
| Terminals | Ghostty (first), iTerm2, WezTerm, kitty |
| Transports | homelab (ProxyJump over Tailscale), Tailscale, plain SSH, GCP IAP, AWS SSM |

- **Chaos:** kill the client, sleep the laptop (`pmset sleepnow`), drop the network, push an agent into its memory cap, and reboot the host (v2).
- **Test hosts:**
  - An homelab bench VM. This goes through homelab's own plan, in the homelab repo, not here.
  - One cheap VPS, for example a Hetzner CX33: 4 vCPU and 8 GB for €8.49 a month.
  - Small GCP and AWS VMs, stopped when not in use, to check the provider recipes.

## 10. Risks

| Risk | Mitigation |
| --- | --- |
| libghostty-vt has no stable API and no tags | Pin one Ghostty commit. Keep the interface tiny (bytes in, resize, snapshot, events out). Run it in its own process. Replay tests on every bump. |
| Agents change their signals and flags often | Keep agent knowledge as data. Show the source of each status in the UI. Test against recorded fixtures. |
| A terminal or agent update breaks fidelity | The fixture suite and `doctor keys` catch it. Pin and test before release. |
| Provider tunnels drop sessions (IAP after 1 hour idle, EICE at 1 hour, Cloudflare on edge restart) | Auto-reconnect with replay. Warn about each recipe's limit in the host details view. |
| Scope creep towards a multiplexer or an agent UI | Principles 1 and 4, and the non-goals. |
| Clean-room and licence mistakes | No Amp code or assets, ever. Attribution for anything taken from Herdr or zmx. A review before going public. |
| Bus factor of one | Small code base, data-driven agent table, good tests, plain docs. |
| A compromised host attacks the client through escape sequences | An "untrusted host" mode that filters OSC 52, OSC 8 and report sequences. Never forward the SSH agent by default. |

## 11. Before the repo goes public

- [ ] Clean-room review of every doc and file.
- [x] Keep the Amp binary study out of git: it lives in `docs/research/private/`, which is ignored.
- [ ] Add `LICENSE`, `NOTICE` and attribution for any code taken from Herdr or zmx.
- [ ] Trademark check for "simstim".
- [ ] `SECURITY.md`, and `CONTRIBUTING.md` with the DCO.
- [ ] Owner decisions D1–D9 recorded.

## 12. Research

| Note | What it covers |
| --- | --- |
| Amp CLI dissection (kept outside git) | How Amp's TUI is built, its screens, keys and visual language, and Orbs, from the shipped binary. Never committed. |
| [Amp UX and Orbs](research/amp-ux-and-orbs.md) | Amp's public docs, design history, Orbs, and what users say |
| [Herdr](research/herdr.md) | The closest existing tool: architecture, fidelity limits, and what to borrow |
| [Host runtime](research/host-runtime.md) | zmx, libghostty-vt, the holder design, status detection, resume commands, manifest |
| [Connectivity and providers](research/connectivity-and-providers.md) | System SSH, multiplexing, bootstrap, provider recipes, reconnect, security |
| [Client stack](research/client-stack.md) | Language and TUI framework, attach hand-off, input filter, clipboard, distribution |
