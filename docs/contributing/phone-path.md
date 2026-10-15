# The phone path (experimental)

**Status:** scaffolded, not launched. It is hidden from the help and from the user docs until the owner launches it ([decision 0019](../decisions/0019-alerts-and-hooks.md), roadmap item N4). Do not mention it in the README, the CHANGELOG or `docs/how-to/`.

## What it is

A person who gets an alert on their phone should be able to answer the agent from the phone. Every phone has an SSH app, so the phone path needs no app of ours. The host helper has a hidden command, `pick`:

```sh
ssh -t my-vps '~/.local/share/simstim/bin/current/simstim-host pick'
```

It lists the user's running sessions as a plain numbered menu, with the name, agent, state and age. It uses no colours and no TUI library, so it works in any terminal and in a 40-column phone screen. The person types a number (or a session's name or ID), and the terminal attaches to that session. Ctrl+\ detaches, and the menu comes back. `q` quits.

## How it works

- `RunPick` (in `internal/host/pick.go`) reads the sessions with `List`, the same as `ls`, and keeps those whose holder runs.
- The attach talks to the holder's own Unix socket, with the same protocol as the bridge (`internal/proto`). Nothing new listens: sshd is still the only door.
- `pickPump` (in `internal/host/pick_pump.go`) puts the terminal in raw mode and passes bytes both ways unchanged. Only Ctrl+\ is kept back, found by the same input filter as the client (`internal/keys`), in every encoding. Window size changes go to the holder as resize frames.
- It asks for a snapshot (`last_offset` −1), as a new client does. Replayed history goes through `term.Catchup`, which the client's attach pump also uses, so old terminal queries are not answered again.
- On detach it writes the reset sequence from `term.Tracker`, never RIS, so the phone's scrollback stays.
- At the menu, the terminal is in its normal mode, where Ctrl+\ would send SIGQUIT. `pick` ignores SIGQUIT.

The phone becomes the leader when it types, so the session takes the phone's size, as with any second client. The laptop's client gets the size back when its user types.

## What is missing before a launch

- A way to get there in one tap: an alert that opens the SSH app on the right command (ntfy's click action can carry an `ssh://` link in some apps).
- Double-tap Ctrl+\ to send the key itself, as the client does. Phone keyboards may also need another detach key.
- Image paste and dropped files, which the client does through uploads.
- Reconnect after the phone's network drops. Today the SSH session ends, and the person runs `pick` again; the session itself keeps running.
- End-to-end tests. Unit tests cover the menu, the choice parser and the pump against a fake holder (`internal/host/pick_test.go`).

## Try it

Build and install a helper (`just build`, then any client command on the host installs it), then from any terminal:

```sh
ssh -t <host> '~/.local/share/simstim/bin/current/simstim-host pick'
```
