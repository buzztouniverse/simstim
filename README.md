# simstim

**Your coding agents, on any host, as if they ran on your laptop.**

simstim runs Claude Code, Codex, opencode, Gemini CLI, Amp or a plain shell in a session on a host you own: a VPS, a cloud VM or a homelab machine. Close the laptop, change networks, switch devices: the session keeps running. Come back, and you flip straight back in. Your terminal draws the agent's real screen, byte for byte, so keys, colours, scrollback, clipboard and notifications work as they do locally.

> **Status:** pre-release. The client, the host helper and the home screen work end to end on a Linux test host; the screen-restore engine is in progress. Nothing is released yet. See [the status](docs/status.md).

## Why

- **Any agent.** Agents are rows in a table, not code paths.
- **Any host you can reach with `ssh`.** simstim uses your own OpenSSH and your `~/.ssh/config`, so ProxyJump, Tailscale, GCP IAP and AWS SSM all work. Nothing new listens on the host, and there is no relay or cloud account.
- **The real thing, byte for byte.** No re-rendering and no multiplexer in the way. The kitty keyboard protocol, truecolour, OSC 8 links, OSC 52 clipboard and desktop notifications pass straight through. Native scrollback and search keep working.
- **Survives sleep.** The client reconnects by itself, and replays exactly the output you missed.
- **Paste screenshots.** Ctrl+V with an image on your laptop's clipboard uploads it and pastes the remote path into the agent. Dropped files work the same way.
- **Knows what each agent is doing.** Sessions show working, needs input, idle or exited, from the agents' own hooks and screen rules kept as data. You get a desktop notification when another session needs you.
- **A worktree per task.** `new --worktree` gives a session its own git branch and folder, and delete keeps it if it has work.
- **Comes back after a reboot.** A boot-time unit restarts each agent with its own resume command.

## How it works

```
 laptop                                   host
 ┌──────────────────────┐  your ssh   ┌───────────────────────────────────────────┐
 │ simstim (client)     │ ──────────▶ │ simstim-host bridge                       │
 │  screens, attach,    │             │  ├─ holder ── PTY ── claude   (8 MiB ring) │
 │  reconnect, paste    │             │  ├─ holder ── PTY ── codex                 │
 └──────────────────────┘             │  └─ systemd user service per session      │
                                      └───────────────────────────────────────────┘
```

Read [the architecture](docs/architecture.md) for the details.

## Build from source

You need Go 1.26 and [just](https://github.com/casey/just).

```sh
just build        # builds the host helpers, then the client, into ./bin
./bin/simstim --help
```

## Quick start

```sh
simstim host add my-vps                          # a Host from ~/.ssh/config: checks it, installs the helper
simstim                                          # the home screen: Enter on a host starts a session
simstim new my-vps --agent claude --dir ~/src/app --worktree
simstim attach my-vps/app                        # attach from a script or a terminal keybinding
```

While attached, the agent owns your terminal. **Ctrl+\\** detaches (press it twice to send it to the agent). **Ctrl+V** with an image on the clipboard pastes it into the agent.

| Command | Does |
| --- | --- |
| `host add/ls/rm/test/doctor/restore` | Manage hosts; `restore` starts again sessions a reboot stopped |
| `host start/stop/status` | Start or stop a cloud VM through your provider script ([examples](examples/providers)) |
| `new <host> [--agent a] [--dir d] [--worktree] [--detach] [-- cmd…]` | Start a session |
| `ls [--json]`, `attach`, `send`, `rename`, `kill` | Work with sessions |
| `ports`, `forward` | See and forward the ports a session listens on |
| `doctor keys` | Show exactly what your terminal sends for each key |

## Docs

| Page | What it covers |
| --- | --- |
| [Plan](docs/plan.md) | Goals, scope by version, milestones, decisions, risks |
| [Architecture](docs/architecture.md) | Components, flows, protocol, fidelity contract, security |
| [UX](docs/ux.md) | Screens, keys, visual language, CLI |
| [Decisions](docs/decisions/) | The decision records |
| [Research](docs/research/README.md) | The evidence behind the design |

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). Commits need a DCO sign-off. Report security issues privately, as [SECURITY.md](SECURITY.md) describes.

## Licence

[Apache-2.0](LICENSE).

The name is a working name, from William Gibson's *Neuromancer*. It may change, so the code keeps it in one place ([docs/renaming.md](docs/renaming.md)).
