# Troubleshoot

## Start here

```sh
simstim doctor terminal
simstim host test my-server
simstim host doctor my-server
```

- `doctor terminal` asks your terminal what it supports: the keyboard protocol, colours, synchronized output.
- `host test` says whether the host answers, and how fast.
- `host doctor` checks the host: the helper, systemd and linger, the agents, and memory.

## Common problems

| What you see | Try |
| --- | --- |
| `cannot reach my-server` | Run `ssh my-server true`. If that fails, the problem is in SSH, not here. |
| Asked for a password every time | Load your key into an SSH agent, or use a key without a passphrase for this host. |
| A key does nothing in the agent | `simstim doctor keys` shows what your terminal sends. Some terminals keep keys for themselves. |
| An agent is "not installed" | `simstim agents install <agent> --host my-server` runs its installer in a session ([install agents](install-agents.md)). An agent installed by hand must be on your `PATH` in a login shell. |
| An agent asks you to log in | Run its own login in a session; `host doctor` prints the command. |
| Sessions stop when you log out | Linger is off: `host doctor` says so. Run `loginctl enable-linger` on the host. |
| The screen looks wrong after attaching | Detach and attach again. If it stays wrong, open an issue with your terminal's name and version. |

## Logs

On the host, each session keeps a log at `~/.local/state/simstim/sessions/<id>/holder.log`.

## Still stuck

Open an issue with the bug form, and include the output of the three commands at the top, with anything private removed.
