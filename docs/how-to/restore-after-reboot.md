# Bring sessions back after the host restarts

When the host restarts, its processes stop, and so do the agents. At boot, a user unit that `host add` installed starts each session again, in its folder, with the agent's own resume command: Claude Code and Codex continue the same conversation. Above it, the session shows the screen it had before.

To do it by hand, for example when the host has no systemd user manager:

<!-- test: help (restarts sessions) -->
```sh
simstim host restore my-server
```

`--all` also restarts sessions whose holder stopped for another reason. A process that was running when the host stopped does not come back; the conversation does.

A session that had [secrets](secrets.md) comes back without them, because nothing on the host keeps them. Its terminal and the launcher say so; start a new session to pass them again.

Which agents resume, and with which command, is in the [agent table](../reference/agents.md).
