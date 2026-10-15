# Getting started

In about five minutes you will install the client, connect a host, start a session, leave it running, and come back to it.

You need:

- a Mac or a Linux machine with OpenSSH (it has it already);
- a Linux host you can reach with `ssh`: a VPS, a cloud VM, or a machine at home. If `ssh my-server` logs you in without a password prompt, you are ready. This page calls the host `my-server`; use your own name for it.

## 1. Install the client

<!-- test: skip (needs a published release; scripts/test-install.sh tests the installer) -->
```sh
curl -fsSL https://raw.githubusercontent.com/iAmLakshya/simstim/main/install.sh | sh
```

The installer checks the release's signature and checksum, then puts the client in `~/.local/bin`. Check it:

```sh
simstim version
```

## 2. Connect the host

`host add` checks that the host answers, installs a small helper there (in your home folder, with nothing that listens on the network), and saves the host.

```sh
simstim host add my-server --yes
```

`--yes` agrees to the helper's install; leave it out to see what it installs first. Then check the host:

```sh
simstim host test my-server
```

## 3. Start a session

Start a shell session that keeps running on the host:

```sh
simstim new my-server --name first --detach
simstim ls
```

`ls` shows the session, with its state and folder.

To start an agent instead, name it. This opens the session in your terminal straight away:

<!-- test: help (opens the session) -->
```sh
simstim new my-server --agent claude --dir ~/src/app
```

While you are attached, the agent owns your terminal: its keys, colours and scrollback are its own. Press **Ctrl+\\** to detach. The session keeps running.

## 4. Come back

Close the laptop, change networks, come back tomorrow. Then attach again:

<!-- test: help (opens the session) -->
```sh
simstim attach my-server/first
```

Or run `simstim` with no arguments to open the launcher: Enter on a host shows its sessions, Enter on a session joins it, and `n` starts a new one. Press `?` in the launcher to see every key.

## 5. Clean up

```sh
simstim kill my-server/first
```

## Next

- [Add a host](../how-to/add-a-host.md) behind a jump host, or with a repo's own SSH config.
- [Keep each task in its own worktree](../how-to/worktrees.md).
- Read [how it works](../explanation/architecture.md).
