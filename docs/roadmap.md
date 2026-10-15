# Roadmap

**Status:** proposed, 15 October 2026, until the owner's yes at Gate 1. It replaces the scope lists in [the plan](plan.md#6-scope-by-version), which is kept for the reasons behind the design.

Every candidate feature is scored from the evidence in the research notes:

- [Developer needs](research/developer-needs.md): 132 developer sources, coded by need and ranked by count and GitHub reactions.
- [Competitors](research/competitors.md): what 28 tools offer, and which features they charge for.
- [Sandboxes and hosts](research/sandboxes.md): how each provider could be a host.

## How a feature is scored

**Score = demand × impact ÷ effort.**

| Part | Scale |
| --- | --- |
| **Demand** | 0 to 5, from the number of developer sources that raise the need: 30 or more is 5, 15 to 29 is 4, 8 to 14 is 3, 4 to 7 is 2, 1 to 3 is 1. Add 1 (up to 5) when the need's GitHub items carry 1,000 reactions or more. A feature that others only sell, with no voice in the sample, gets 1. |
| **Impact** | 3: it changes who can use the tool, or keeps its core promise. 2: a clear gain for current users. 1: a polish. |
| **Effort** | S (days) = 1, M (a week or two) = 2, L (a month or more) = 3. |

**Horizons.** **Now** is built in the current goal: every item the goal requires, plus every other feature that scores 4 or more. **Next** scores 2 to 4. **Later** scores under 2. Next and later items are GitHub issues with the labels `horizon: next` or `horizon: later`.

## Now

### Required by the goal

| # | Feature | Need (sources / reactions) | D | I | E | Score |
| --- | --- | --- | --- | --- | --- | --- |
| R1 | **Install in one command:** `curl … \| sh` that checks a signature and a checksum, `go install`, and a Homebrew cask ready for a tap. Clean Mac to a running agent in under 5 minutes, measured. | setup-time [12 / 389](research/developer-needs.md#needs-ranked); "not released" is the first place we are behind ([competitors](research/competitors.md#where-simstim-is-behind)) | 3 | 3 | M | 4.5 |
| R2 | **`host create local` / `host destroy`:** a local VM made, bootstrapped and connected in one command, proven live | run-on-own-server [37 / 2,261](research/developer-needs.md#needs-ranked); Lima ranks first for a local host ([sandboxes](research/sandboxes.md#14-recommendation)) | 5 | 3 | L | 5.0 |
| R3 | **Cloud providers**, E2B first, built from public API docs and tested against recorded responses | cost [21 / 2,318](research/developer-needs.md#needs-ranked); E2B fits over its process API ([sandboxes](research/sandboxes.md#2-e2b)) | 5 | 2 | L | 3.3 |
| R4 | **Idle stop, wake on attach** for cloud hosts | cost 21 / 2,318; sold by Amp, Coder Premium, Ona ([competitors](research/competitors.md#paid-features-simstim-can-give-for-free)) | 5 | 2 | M | 5.0 |
| R5 | **`agents install <agent>`** with the official installers, and agent checks at host setup: installed, on `PATH`, logged in | headless-login 5 / 245, setup-time 12 / 389 | 3 | 3 | M | 4.5 |
| R6 | **Per-session secrets** from the laptop's keychain or a password manager CLI, optional and never stored | secrets 3 / 0 | 1 | 1 | M | 0.5 |
| R7 | **`doctor` finds and fixes** the common problems | setup-time 12 / 389 | 3 | 2 | M | 3.0 |

R6 scores low: few developers ask for it, and simstim stores no secrets by design. It stays because the goal asks for it, and it is built as small as it can be: names in the config, values fetched at session start.

### Chosen by score

| # | Feature | Need (sources / reactions) | D | I | E | Score |
| --- | --- | --- | --- | --- | --- | --- |
| N1 | **Antigravity CLI in the agent table.** Google moved consumer Gemini CLI users to it on 18 June 2026. | lock-in/any-agent [16 / 1,470](research/developer-needs.md#needs-ranked) | 5 | 2 | S | 10.0 |
| N2 | **Scrollback hints per agent:** the agent table records each agent's inline-mode flag (for example Codex `--no-alt-screen`), and `doctor` warns when an agent runs full screen and so keeps its own scrollback | scrollback 17 / 2,302 | 5 | 2 | S | 10.0 |
| N3 | **"Who needs me" jump:** one key goes to the next session that waits for input, on any host | parallel-agents 18 / 283 | 4 | 2 | S | 8.0 |
| N4 | **A phone path with no new app:** a small session picker in the host helper, so any phone SSH app (Termius, Blink, Termux) can pick a session and attach. No relay, no web UI. *Needs the owner's yes: the goal keeps phones out unless the owner agrees.* | mobile-access 40 / 1,373, the top need | 5 | 3 | M | 7.5 |
| N5 | **A host-side alert hook:** a command per host, run when a session needs input or finishes (for example `curl` to the user's own ntfy topic), so alerts reach a phone with no simstim relay. *Needs the owner's yes, as N4.* | notify-when-input-needed 13 / 225; sold by Cursor, Devin, Codex, Conductor at $20 to $50 a month | 3 | 2 | S | 6.0 |
| N6 | **Repo setup hooks:** a repo's own setup script runs when a session or worktree starts, with its output in the session | setup-time 12 / 389; prebuilt environments are sold by Cursor, Coder, Devin, Codex | 3 | 2 | S | 6.0 |
| N7 | **`pull <session>`:** bring a session's branch back to the laptop over the same SSH | worktrees 10 / 216; sold as Claude `--teleport`, `codex apply`, Terragon `terry pull` | 3 | 2 | S | 6.0 |

## Next

| # | Feature | Need (sources / reactions) | D | I | E | Score |
| --- | --- | --- | --- | --- | --- | --- |
| X1 | More agent rows: Pi, Cursor CLI, GitHub Copilot CLI | lock-in/any-agent 16 / 1,470 | 5 | 1 | S | 5.0 (small; left for contributors) |
| X2 | Continue when a usage limit resets: `send --at <time>` or "after the reset" | cost; [Claude Code #13354](https://github.com/anthropics/claude-code/issues/13354) and [Codex #21073](https://github.com/openai/codex/issues/21073), 291 reactions | 3 | 2 | M | 3.0 |
| X3 | Show the link state and the replayed bytes after a reconnect | reliable-reconnect 14 / 242 | 3 | 1 | S | 3.0 |
| X4 | Choose the worktree's base branch in the new-session form | worktrees 10 / 216; [Claude Code #23622](https://github.com/anthropics/claude-code/issues/23622), 87 reactions | 3 | 1 | S | 3.0 |
| X5 | Tests of OSC 52 copy in each agent's full-screen mode | clipboard 10 / 502 | 3 | 1 | S | 3.0 |
| X6 | Per-session isolation: a separate Unix user or systemd sandboxing, for agents run without approvals | sandbox-isolation 15 / 0 | 4 | 2 | L | 2.7 |
| X7 | Laptop and phone attached together, as a tested case | multi-device-attach 4 / 51 | 2 | 2 | M | 2.0 |
| X8 | Sessions inside the repo's devcontainer on the host | own-environment 7 / 1,235 | 3 | 2 | L | 2.0 |
| X9 | Background runs with a result: start an agent with a prompt, get its exit and summary when it ends | parallel-agents; scheduled runs are sold by Cursor and Amp | 2 | 2 | M | 2.0 |
| X10 | A cost hint per cloud host | cost 21 / 2,318 | 4 | 1 | M | 2.0 |

## Later

| # | Feature | Need (sources / reactions) | D | I | E | Score |
| --- | --- | --- | --- | --- | --- | --- |
| L1 | Read-only watch: an attach that drops input, for a second screen or a colleague with SSH access | sharing 0 sources; sold by Conductor ($50), Devin Teams | 1 | 2 | M | 1.0 |
| L2 | Opt-in recording of a session's output to a file, and replay | 0 sources; sold as Tailscale SSH recording | 1 | 1 | S | 1.0 |
| L3 | Port preview: notice a new listening port and offer to forward it | port-preview 5 / 14 (`ports` and `forward` exist) | 2 | 1 | M | 1.0 |
| L4 | Snapshots and hibernation: save a host and create from it | snapshots 1 source; providers' own pause covers idle hosts (R4) | 1 | 2 | L | 0.7 |

## Not planned

| Feature | Why |
| --- | --- |
| A web or phone UI | A non-goal of [the plan](plan.md#4-non-goals). The evidence warns against it too: an unauthenticated remote code execution in another tool's server, and a phone app that changed a privacy setting ([needs](research/developer-needs.md#what-this-means-for-simstim)). N4 reaches phones through SSH apps instead. |
| Managed secrets with redaction or a proxy | simstim stores no secrets. R6 fetches values from the user's own store at session start. |
| A relay, accounts or telemetry | Project rule: nothing listens and nothing phones home. |

## Good first issues

Small, well-bounded changes for new contributors, each a GitHub issue with the label `good first issue`: X1 (one issue per agent), X3, X4, X5, and these:

- `ports --json`, like `ls --json`.
- Shell completion of host and session names.
- Recipes in the docs for scheduled runs with `new --detach` and `send` from cron or CI.
- A key in the launcher that copies the selected session's folder.
- The agent's version in host details.
