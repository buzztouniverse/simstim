---
status: proposed
date: 2026-10-15
---

# 0020 A guided setup: one command and one sheet, over one engine

**Status:** Proposed. The owner asked for a guided setup at Gate 2 on 15 October 2026. This record is the design, for the owner's yes.

## Context

The owner approved the launcher at Gate 2, and asked for three changes to how a new user gets going:

1. one guided command that walks a new user through everything;
2. a checklist of agents to install;
3. steps to set up a host when the user has none yet.

Today the pieces exist, but the user must know them and run them in order: `host add` or `host create`, then `agents install` once per agent, then each agent's login in a session, and for a rented server the steps in [your first cloud host](../../tutorials/first-cloud-host.md). The launcher's first-run screen offers only the add-host wizard, which ends at "ready".

The rules stand: no secrets kept, no telemetry, no new listening port, and the attach path byte for byte. The product never sees a login, never rents anything, and never edits `~/.ssh/config` without asking.

## Options

**Where the logic lives:**

1. Each front end, the command and the launcher, has its own setup code.
2. One engine in a small package of its own, `internal/onboard`. The command and the launcher only ask its questions and show its answers.
3. The engine in `internal/client`.

**How the command asks:**

1. Full-screen, with Bubble Tea, like the launcher.
2. Plain lines and numbered answers, like `host add` and `doctor --fix` today.

**How agents get installed:**

1. Run each installer in the background and show a spinner.
2. Run each one in a session the user watches and can answer, one after another, as `agents install` does.

**Logins:**

1. Leave them to the user, with the commands printed.
2. Offer each one now, in a session the user attaches to, then check again.

**A stop (Ctrl+C or Esc) in the middle:**

1. Leave whatever was done.
2. Undo everything the setup did.
3. Keep what is whole and useful (a saved host, an installed agent), delete a machine a provider had only begun to make, and say what was done.

## Decision

Option 2 for where the logic lives: `internal/onboard`. It cannot live in `internal/client`, because making a host needs the provider packages, which import `internal/client`. It holds:

- **Places:** where agents can run, and whether this computer can use each now: a host in `~/.ssh/config` (`ssh` on `PATH`), a local VM with Lima (`limactl` on `PATH`), a test container with Docker (`docker` on `PATH`), an E2B sandbox (an API key in `E2B_API_KEY` or the E2B CLI's login; the key is never read out), or a server the user will rent. A place that is not ready comes with the exact steps to make it ready, from Lima's, Docker's and E2B's own install pages.
- **The rent walk-through:** the tutorial's steps, one at a time: an SSH key, a server, a user for agents, and a name in `~/.ssh/config`. Each step is checked where it can be: a public key in `~/.ssh`, `ssh -o BatchMode=yes user@address true`, then `ssh <name> true`. The config lines are shown in full, and added only after a yes.
- **Create:** `host create`'s checks, the provider's create, and saving the host at once. A stop while a provider works deletes the machine it had begun to make, but only for providers that name machines after their hosts (Lima, Docker), and only when no such machine was there before. For E2B, it says where to look.
- **The flow:** the checklist (first-class and supported agents first, best-effort ones folded under them; installed agents ticked), then each install, then each login, then a summary and the command for the first session. Each install and login runs in a session (`install-<agent>`, `login-<agent>`), and the host's agent checks after it decide how it went.

Option 2 for how the command asks: `simstim setup` prints numbered parts and reads plain answers. It reads stdin one byte at a time and keeps no reader while a session is attached, so the attach path stays byte for byte. It works the same over SSH, in any terminal, and in tests. Flags answer every question for scripts: `--host`, `--create`, `--name`, `--agents`, `--yes` and `--no-login`. Without a terminal and without enough flags, it stops before doing anything and says which flags to pass.

Option 2 for installs and option 2 for logins: the user watches each installer and each login in a session, as `agents install` does today. Without a terminal, the command waits for each installer to end, and skips the logins.

Option 3 for a stop.

**The launcher** drives the same engine. Its first-run screen offers the setup on Enter, and `s` opens it from anywhere. A sheet asks where agents will run. A host in the SSH config goes to the add-host wizard; Lima, Docker and E2B get a name and are made; a rented server gets one screen per step. The wizard gains three steps after "ready": the checklist, each install or login (Enter runs it, `s` skips it, Esc stops the rest), and a summary whose Enter opens the new-session form on the agent it names. A plain `a` (add a host) shows the checklist only when an agent is missing or not logged in. Quitting while a provider makes a machine waits for its clean-up.

**`host add`** shows the checklist too, in a terminal, when a first-class or supported agent is missing, unless `--yes` is given.

## Consequences

- Good, because a new user needs one command, or one key, from nothing to a first session, and the manual steps stay documented and keep working.
- Good, because the decisions live in one tested package; the two front ends cannot drift apart.
- Good, because nothing new listens, nothing is stored, and the product still never sees a login or a key.
- Bad, because the setup hands the terminal to each installer and each login. A user who leaves one running (Ctrl+\\) must come back to it with `attach`; the summary says how.
- Bad, because the readiness checks are shallow: `docker` on `PATH` does not prove the engine runs, and an E2B key does not prove the template exists. The create step's own errors cover those cases.
- Bad, because adding a host in the launcher takes one more screen when an agent is missing: the checklist, which Enter passes with nothing ticked.
