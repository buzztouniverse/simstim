# AGENTS.md

Instructions for any operator of this repo: a person or an AI agent.

## What this repo is

A client and a host helper that run AI coding agents in sessions on remote hosts, and show the agent's real terminal UI locally, byte for byte. [The plan](docs/plan.md) has the goals, [the architecture](docs/explanation/architecture.md) the design, and [the UX spec](docs/contributing/ux-spec.md) the screens.

| Path | Contents |
| --- | --- |
| `main.go` | The client's main, at the root so `go install` names it right |
| `internal/cli` | The client's command tree (cobra) |
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

1. **Never spell the product name** outside `internal/brand`, the docs and the module path. Derive it. `just check-name` enforces this. The name may change ([docs/contributing/renaming.md](docs/contributing/renaming.md)).
2. **Byte-exact while attached.** Only the input filter's two keys (detach, image paste) may be intercepted. Never send RIS (`ESC c`) to the user's terminal.
3. **Agents are data** in `internal/agents`.
4. **Nothing listens, nothing phones home.** No new ports, relays or telemetry.
5. **Clean-room and public-safe.** Never copy code, prompts or assets from closed-source tools. Keep private material (goals, status notes, logs, captures, host names, addresses, personal paths) in the git-ignored `private/` and `docs/research/private/` folders. Neither may enter the history.
6. **Decisions** go in `docs/contributing/decisions/`. Do not edit an accepted record; supersede it.
7. **Errors** carry a message and a "Try:" hint.
8. **Commits:** one milestone or logical change each, pushed to `origin`. Run `just test lint` first. The message is a subject, a blank line and a body, and nothing more. Never add `Signed-off-by:`, `Co-Authored-By:`, `Claude-Session:` or any other trailer or attribution line, and never use `git commit -s` ([0010](docs/contributing/decisions/0010-no-commit-trailers.md)).
9. **Docs:** plain English, short sentences, British spelling. A change ships with its docs and its CHANGELOG entry, in the same commit.

## Test hosts

End-to-end tests (`test/e2e/run.py`) drive the real client in a pseudo-terminal against a real host. Add the host with `host add`, set `E2E_HOST` to its name, then run `just e2e`.

To see a TUI the way a person would, use `test/ttyshot`: it runs a program in a pseudo-terminal, sends keys from a script, and prints the rendered screen (`go run ./test/ttyshot -script s.tty -- ./bin/<name>`).

Write each change you make by hand on a test host in a log of your own, outside the repo: a test host's name and address are private. Ask the host's owner before you reboot it.

## Commands

| Command | Does |
| --- | --- |
| `just build` | Host helpers (and the shadow terminal, if Zig 0.16 is installed), then the client, into `./bin` |
| `just vt` | The shadow terminal (`vt/`, Zig 0.16) for Linux amd64 and arm64 |
| `just screenshots` | The README's screenshots (`docs/images`), drawn from `demo` by `ttyshot` |
| `just test` | Unit tests with the race detector |
| `just lint` | `go vet`, golangci-lint, the name check |
| `just e2e` | End-to-end tests against `E2E_HOST` |
| `just snapshot` | A local release build with GoReleaser |
