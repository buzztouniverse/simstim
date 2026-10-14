# AGENTS.md

Instructions for any operator of this repo: a person or an AI agent.

## What this repo is

A client and a host helper that run AI coding agents in sessions on remote hosts, and show the agent's real terminal UI locally, byte for byte. [The plan](docs/plan.md) has the goals, [the architecture](docs/architecture.md) the design, and [the UX spec](docs/ux.md) the screens. The working goal and its finish state are in [GOAL.md](GOAL.md). The progress against it is in [docs/status.md](docs/status.md).

| Path | Contents |
| --- | --- |
| `cmd/cli` | The client binary |
| `cmd/host` | The host helper binary, installed on hosts by the client |
| `internal/brand` | The product name. **The only place that spells it.** |
| `internal/proto` | The wire protocol |
| `internal/ring` | The output ring with byte offsets |
| `internal/host` | Holder, bridge, launcher, session operations, doctor |
| `internal/agents` | The agent table: launch and resume commands, as data |
| `internal/term`, `internal/keys` | Mode tracking, reset and replay sanitising; the input filter |
| `internal/client` | Config, SSH, bootstrap, the attach pump, the clipboard |
| `internal/tui` | The screens, built with Bubble Tea v2 |
| `docs/` | Plan, architecture, UX, decisions, research, status |

## Rules

1. **Never spell the product name** outside `internal/brand`, the docs and the module path. Derive it. `just check-name` enforces this. The name may change ([docs/renaming.md](docs/renaming.md)).
2. **Byte-exact while attached.** Only the input filter's two keys (detach, image paste) may be intercepted. Never send RIS (`ESC c`) to the user's terminal.
3. **Agents are data** in `internal/agents`.
4. **Nothing listens, nothing phones home.** No new ports, relays or telemetry.
5. **Clean-room.** Never copy code, prompts or assets from closed-source tools. `docs/research/private/` is git-ignored and must stay out of history.
6. **Decisions** go in `docs/decisions/`. Do not edit an accepted record; supersede it.
7. **Errors** carry a message and a "Try:" hint.
8. **Commits:** one milestone or logical change each, with a DCO sign-off (`git commit -s`), pushed to `origin`. Run `just test lint` first.
9. **Docs:** plain English, short sentences, British spelling. Update `docs/status.md` when a finish-state item changes.

## Test hosts

End-to-end tests run against a real host. Set `E2E_HOST` to an SSH target, and `E2E_SSH_CONFIG` to an extra SSH config file if the host needs one. Then run `just e2e`.

Any change made on a test host by hand goes in [docs/sandbox-log.md](docs/sandbox-log.md). Ask the owner before rebooting a host.

## Commands

| Command | Does |
| --- | --- |
| `just build` | Host helpers, then the client, into `./bin` |
| `just test` | Unit tests with the race detector |
| `just lint` | `go vet`, golangci-lint, the name check |
| `just e2e` | End-to-end tests against `E2E_HOST` |
| `just snapshot` | A local release build with GoReleaser |
