# Keep your scrollback

Some agents draw full screen, on the terminal's alternate screen, the way `vim` does. Your terminal then keeps no scrollback of the conversation: `Cmd+F` and scrolling up show nothing of it. The agent keeps its own scrollback instead, inside its window. This is the agent's choice, and the session passes it through unchanged.

Claude Code and Codex can also draw inline, in the terminal's normal screen, so the conversation goes into your terminal's scrollback as it would with no session at all.

## For one session

<!-- test: help (opens the session) -->
```sh
simstim new my-server --agent codex --dir ~/src/app --inline
```

## For every session

In `config.toml` (`simstim config path` says where it is):

```toml
[agents]
inline = true
```

The [agents reference](../reference/agents.md) lists which agents have an inline mode, and how it is turned on: for Claude Code an environment variable, for Codex a flag. A session started inline comes back inline after a host reboot.

## Which sessions draw full screen

```sh
simstim host doctor my-server
```

`host doctor` lists the sessions that draw full screen now, and says how to start each one inline when its agent can. `ls --json` shows the same as `full_screen`.

## The agents' own scrollback

Full screen has its own strengths: Claude Code's fullscreen renderer does not flicker and keeps its memory flat in long conversations. To search its history, press `Ctrl+O` for its transcript, then `/` to search, or `[` to write the whole conversation into your terminal's scrollback.
