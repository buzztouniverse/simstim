# Common questions

Short answers, each with a link to the page that says more.

## Does my laptop need to stay on?

No. The agent runs in a session on your host, not on your laptop. Close the lid, and it keeps working. When you come back, simstim reconnects and replays what you missed ([how it works](explanation/how-it-works.md#reconnect-and-replay)).

## Is anything sent to you, or to anyone else?

No. There is no account, no relay and no telemetry. The client talks only to your hosts, over your own SSH. Nothing new listens on a host ([security model](explanation/security.md)).

## Which agents work?

Any program that runs in a terminal. Claude Code and Codex are tested end to end on every release. opencode, Antigravity CLI, Gemini CLI, Amp, Cursor CLI, Aider and Goose are in [the agent table](reference/agents.md) too, so simstim knows how to start them, resume them and tell when they need you.

## What do I need?

A Mac or a Linux machine with OpenSSH, and a Linux host that `ssh` reaches: a VPS, a cloud VM, a machine at home, or a VM on your laptop. The [guided setup](tutorials/getting-started.md) helps you find or make one.

## What does it put on my host?

A small helper, in `~/.local/share/simstim/bin/`, which it checks by its SHA-256. It turns on systemd's linger for your user, so sessions keep running when you log out, and adds a unit that brings sessions back after a reboot. Each session then runs as a systemd user service ([add a host](how-to/add-a-host.md#what-host-add-does)).

## Why not tmux?

tmux keeps its own copy of the screen and draws that, so its scrollback replaces your terminal's, and some keys, such as Shift+Enter, do not get through. It knows nothing about agents. simstim passes the agent's own bytes to your terminal, and shows each agent's state ([how it compares](explanation/comparison.md)).

## What happens when the host restarts?

The host starts each session again at boot, and each agent resumes its conversation with its own resume command. A turn that was running when the host went down is not resumed ([restore after a reboot](how-to/restore-after-reboot.md)).

## Can I attach from two computers?

Yes. Each computer attaches to the same session. The last one to type sets the size of the agent's terminal ([architecture](explanation/architecture.md)).

## What does it cost?

Nothing. It is open source, under the Apache License 2.0. You pay only for your host, if you rent one.

## Something is wrong. Where do I start?

Run `simstim doctor`, then `simstim host doctor my-server`. [Troubleshoot](how-to/troubleshoot.md) goes through the common problems, and [getting help](../SUPPORT.md) says where to ask.
