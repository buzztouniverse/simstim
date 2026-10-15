# Install an agent on a host

simstim installs an agent with its vendor's own installer, in a session on the host, so you see the installer's output and can answer its questions. Then you log in with the agent's own login, in a session. simstim never logs in for you, and never sees your login.

## Install

<!-- test: help (it runs an installer on the host) -->
```sh
simstim agents install claude --host my-server
```

The client:

1. Checks the host. If the agent is already installed, it says where and stops; `--force` runs the installer again, for example to update the agent.
2. Prints the install command and the vendor page it comes from.
3. Starts a session named `install-claude` that runs the command, and attaches to it. Answer the installer's questions there. Detach with `ctrl+\` at any time; the installer keeps running, and `simstim attach my-server/install-claude` brings you back.
4. When the installer ends, checks the host again and says how to log in.

With one host, `--host` is not needed. `simstim agents ls` lists every agent and its install command, and [the agent reference](../reference/agents.md#install-and-log-in) lists each command's source.

## Log in

Each agent logs in its own way. Run its login in a session, and follow its prompts: most print a link to open in your browser and ask for the code it shows.

<!-- test: skip (it starts a login, and --help would go to the agent) -->
```sh
simstim new my-server -- claude auth login
simstim new my-server -- codex login --device-auth
```

For an agent whose first start is its login, such as Antigravity CLI, start the agent: `simstim new my-server --agent antigravity`.

An agent that takes an API key instead, such as Aider, can get it from your laptop at the start of each session: see [pass secrets to a session](secrets.md).

## Check the agents

`host add` and `host doctor` check each first-class and supported agent, and any other agent that is installed:

<!-- test: help (needs a host) -->
```sh
simstim host doctor my-server
```

```text
Agents:
  ✓ Claude Code      /home/me/.local/bin/claude, logged in (`claude auth status` says so)
  ! Codex            /home/me/.local/bin/codex, not logged in (`codex login status` says not)
    Try: simstim new my-server -- codex login --device-auth
  ✕ opencode         /home/me/.opencode/bin/opencode is not on PATH in a login shell
    Try: on the host, add it to PATH: echo 'export PATH="/home/me/.opencode/bin:$PATH"' >> ~/.profile
  ? Antigravity CLI  /home/me/.local/bin/agy, login unknown (the agent has no status command or credentials file to read)
```

| Mark | Meaning |
| --- | --- |
| ✓ | Installed, found by a login shell, and logged in |
| ! | Installed, but not logged in |
| ? | Installed; the host cannot tell whether it is logged in |
| ✕ | Not installed, or not on `PATH` in a login shell |

Sessions find agents through your login shell, the way `ssh my-server` does. An agent the installer put in a folder that is not on `PATH` there, such as `~/.local/bin` on some systems, is installed but cannot start: add the folder to `PATH` in `~/.profile` on the host.

The host decides "logged in" from the agent's own signs, kept as data in the agent table: a variable such as `ANTHROPIC_API_KEY` set in the login shell, the agent's own status command (`claude auth status`, `codex login status`), or its credentials file. It reads no file's content and shows no command's output.
