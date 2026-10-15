# Amp: TUI, DX and Orbs (research for simstim)

- Date: 15 October 2026
- Lane: the public web (docs, Chronicle news, notes, official media, community posts). A separate note covers the local Amp binary.
- Method: I read the ampcode.com docs and Chronicle posts on 14 and 15 October 2026. I downloaded official screenshots and demo videos from `static.ampcode.com` and looked at frames myself. I read the install scripts and the Homebrew formula but did not run them. I used the Hacker News Algolia API for community threads. X (twitter.com) did not load (HTTP 402 and 451), so X posts appear here only through search snippets, and I mark them.
- Anything I could not confirm is marked **UNVERIFIED**.

## Summary

Amp in October 2026 is no longer a terminal-first product. It is a cloud agent platform. Each thread can run in an "orb": a cloud microVM that Amp creates, puts to sleep and wakes again. A "runner" is your own machine, joined to Amp's cloud. The web and native apps are now the main place to manage many agents. The TUI is kept lean and local, and Amp removed its TUI sidebar in August 2026.

The Amp TUI is still a strong model for simstim's own screens. It runs on Amp's own renderer, with no flicker. It has one bordered prompt box that carries status in its border. Dialogs have the title in the top border and key hints in the bottom border. A command palette (Ctrl+O) replaces slash commands. The keymap uses chords and a leader key and can be changed. Tool steps fold into one line each.

Orbs show what people value in remote sessions: zero setup, sessions that survive a closed laptop, sleep when idle, one sandbox per task, ports exposed as links, and one place to watch everything. simstim should copy the lifecycle and the visibility. It should differ on the control plane: no cloud account, plain SSH, any agent, and the agent's real TUI passed through byte for byte.

---

## Background: Amp in October 2026

### Timeline of the changes that matter for simstim

| Date | Change | Source |
| --- | --- | --- |
| 23 Jul 2025 | "Towards a New CLI": new npm CLI, "agents in the terminal" | news/towards-a-new-cli |
| 2 Sep 2025 | "Look Ma, No Flicker": full-screen TUI on Amp's own framework | news/look-ma-no-flicker |
| 21 Oct 2025 | Edit, restore and fork messages in the CLI (Tab, then e, r, f) | news/cli-tab-navigation |
| 23 Oct 2025 | Handoff replaces compaction | news/handoff |
| 28 Oct 2025 | Command palette replaces slash commands | news/command-palette |
| 19 Feb 2026 | "The Coding Agent Is Dead": editor extensions shut down on 5 March; CLI kept "for now" as "a ladder" | news/the-coding-agent-is-dead |
| 6 May 2026 | "Amp, Rebuilt" (codename Neo): new architecture, remote control, plugins, no permission prompts by default | news/neo |
| 14 May 2026 | CLI ships as a single Bun-compiled binary; npm package renamed to `@ampcode/cli` | news/npm-package-changes |
| 27 May 2026 | Neo for everyone; the old CLI is gone | news/drop-the-neo |
| 2 Jun 2026 | Public, discoverable threads removed (unlisted links remain) | news/end-of-public-threads |
| 4 Jun 2026 | "Agents, Everywhere": one UI for all agents on web, mobile and CLI; CLI sidebar on Ctrl+\ | news/agents-everywhere |
| 30 Jun 2026 | **Orbs launch** ("Agents in Orbs") | news/agents-in-orbs |
| 8 Jul 2026 | Runners: start threads on any machine that runs `amp --no-tui` | news/agents-anywhere |
| 20 to 23 Jul 2026 | Puck assistant, Slack, multiplayer, event-driven orbs | Chronicle |
| 6 Aug 2026 | Portals into orbs | news/portals |
| 27 Aug 2026 | **TUI sidebar removed**; the web and native apps manage many threads | news/so-long-tui-sidebar |
| 28 Aug 2026 | macOS and iOS apps | news/amp-on-ios-and-macos |
| 8 Sep 2026 | "Steer, Don't Queue": messages reach a busy agent at the next chance | news/steer-dont-queue |
| 13 Sep 2026 | "Free Agent": free with your own compute and model keys | news/free-agent |
| 17 to 24 Sep 2026 | One runner serves many folders; runners update themselves; the Mac app is a runner; shared runners | Chronicle |
| 25 Sep 2026 | "Less Noise": agent steps fold more aggressively | news/less-noise |
| 10 Oct 2026 | Claude Code mode: use a Claude plan inside Amp through the Claude Agent SDK | news/use-your-claude-plan |
| 13 Oct 2026 | Puck over MCP from ChatGPT and Claude | news/take-puck-with-you |

### 1. The Amp CLI TUI

#### 1.1 The rewrites, and the stated reasons

- **The first TUI (September 2025).** Amp moved from a line-based CLI on Ink (React for terminals) to a full-screen TUI on its own framework. The post lists what the framework makes possible: smooth scrolling while tool calls stream, mouse use, overlays, popups and clickable buttons. It claims "no flicker. No jarring redraws. No stuttering text." Thorsten Ball posted on X: "Amp's CLI is now a proper, full-on TUI. Best part for me personally? No more Ink, no more flicker." (Seen only as a search snippet. X did not load.)
- **The framework.** Tim Culverhouse (Amp) says it is a TypeScript port of his Zig library libvaxis, and that about 90% of it was written by Amp itself. It is "inspired by Flutter", with Widgets, StatefulWidgets, Intents and Bindings. It is double-buffered: a front screen (the last frame) and a back screen (the next). It diffs the two, prints only the changes, then swaps them. Test file names in an official October 2025 video show `src/tui/framework/` with flex layout, scrolling, selection and multi-click selection.
- **Why they built their own.** No post gives a full reason. The evidence is Ink's flicker, the features listed above, and a team member who already wrote a TUI library. **UNVERIFIED:** whether Neo (May 2026) kept the same framework. Neo screenshots look like the same widget style.
- **The Neo rebuild (May 2026).** Its stated principles:
  1. Build for what frontier models can do now, not for their past limits.
  2. Let the agent run longer with fewer interruptions: queueing, automatic compaction at 90% of context, and permissions as a plugin.
  3. Remove features that need babysitting.
  Neo is "remote-controllable, compaction-first, plugin-powered, and much faster". On a thread of about 5,000 messages, mean CPU fell from 84.1% to 17.4% and idle memory from 1,814 MB to 540 MB. Neo removed: handoff (replaced by compaction), file rollback, skill management commands, user-invoked skills, **custom themes**, `$`/`$$` shell mode, and the `--take-me-back` flag.
- **Later principles.**
  - "Less Noise" (September 2026): people care about the outcome, not the path. Hide the steps by default and keep them one action away. Watching every step keeps agents on "too short a leash".
  - "So Long, TUI Sidebar" (August 2026): Amp's own multiplexing on top of terminal tabs and panes "proved to be a less-than-optimal user experience". The TUI now covers local, interactive threads in one environment, and the terminal does the multiplexing.
- **Known cost of the approach.** Peter Steinberger (late 2025) explains that alternate-screen TUIs avoid flicker but lose native text selection, scrollback and search. He criticises Amp's own find (it searches only visible text), its non-native selection and its custom scrollbar. He prefers the inline approach of Claude Code and pi.

#### 1.2 What the TUI looks like (official screenshots and video frames)

- **Welcome screen (Neo, May 2026).**
  - On the left is a large sphere of dots. It fades from bright green at the top to dim blue at the bottom, like a planet lit from above.
  - On the right are three lines: "Welcome to Amp" in green, "Ctrl+O for help" (key in bold cream, words in grey), and a rotating quote in dim grey.
  - At the bottom is the prompt box. It is a full-width box with a cream border and a block cursor. The mode name "deep" sits in green in the top-right of the border. The working folder "~/work/amp2" sits in the bottom-right of the border.
- **Welcome screen in an orb (June 2026).**
  - Scattered dot clusters drift across the screen.
  - The text reads "ctrl+o for commands", "? for shortcuts" and "NEW: Agents in Orbs" (underlined link).
  - The bottom-right of the prompt border reads "In the Orb (tucan)" in bold instead of a local path. The top-right reads "deep²". The superscript shows the effort level.
- **A running thread.**
  - The transcript is markdown. Headings are green. Inline code and file names are amber. Links are amber and underlined. Box-drawing diagrams use arrows.
  - The user's own messages are yellow-olive italics with a thick bar on the left.
  - Tool steps fold to one line, with a ▸ to expand: "✓ Explored 3 searches ▸", "✓ Edited 2 files +28 -3 ▸", "$ cargo test (exit code: 127) ▸". The `$` and the exit code are red.
  - A thin scrollbar sits on the right edge.
  - Status lives in the prompt border:
    - bottom-left: a small animated glyph and "Thinking 78 tok", "Streaming", "Running tools" or "Enter to steer";
    - top-right: running cost and mode, for example "$0.003 — deep²".
- **Queued message.** A second bordered box stacks just above the prompt. It holds the queued text, with "queued" right-aligned in dim grey. The prompt border hint changes to "Enter to steer".
- **Dialog (`ask_user_choice`, May 2026).**
  - A centred box with a light grey border. The question is the title, set into the top border in bold amber.
  - Below it are numbered options, the line "Type the number of your choice", and an amber ">" prompt.
  - The key hints are set into the bottom border: "Enter submit · Esc cancel". Key names are teal, actions are grey, and a middle dot separates them.
  - The transcript behind the dialog is dimmed.
- **The Dial (July 2026).**
  - A box with a row of dots across the top. Filled cyan dots run up to the chosen position, then dim dots continue.
  - Under the dots are four labels: low, medium, **high**, ultra. The chosen one is bold cyan.
  - Then come "Agent: GPT-5.5 xhigh" and "Oracle: Fable 5 high" (labels in cream, values grey), and a one-line description, "Deep reasoning for hard tasks".
  - The bottom border reads "↔ turn · tab plugins · esc".
- **Command palette (October 2025 video).**
  - A centred box titled "Command Palette" in amber, with a "> " filter line.
  - Each row has a dim category word, then a bold action: "thread switch thread", "screen show costs", "custom command work-on-linear-issue".
  - The palette uses the current prompt as context. With "then run the build" typed, it offers 'prompt queue prompt "then run the build"' and 'thread handoff to new thread "then run the build"'.
  - Key bindings are right-aligned in blue ("Alt d", "Ctrl g"). The selected row has an amber background.
- **Old footer (pre-Neo).** A status line under the prompt: "~ Waiting for response... Esc to cancel" on the left. On the right: "3 IDEs available  1 file changed +8 ~4 -12  5% of 968k".
- **CLI sidebar (June to August 2026, now removed).** A narrow left column lists thread titles with relative ages ("1m", "5m", "2h"). The current thread is highlighted.
- **Terminal title.** Amp sets the window title to "amp - <thread title> - <cwd>". A spinner glyph is prefixed while the agent works.
- **`amp sync` output.**
  - A small orb glyph next to "Sync <thread title>", with the thread ID and the local path below it in dim text.
  - Green "✓ Updated <file>" lines.
  - Then: "Watching this thread for changes. Press Ctrl-C to stop."
- **Colour.**
  - In the May 2026 screenshot, the sampled colours match the Gruvbox palette closely: background #1e2021, text #eadbb2, amber #d79a20, teal #468588.
  - The border colour differs between official screenshots: cream, then grey, then grey-green.
  - This suggests Amp paints mostly with the terminal's own ANSI palette, so it takes on the user's terminal theme, and uses RGB for gradients such as the logo. **UNVERIFIED.** The binary lane can confirm.
  - A third-party Pi package, `amp-themes`, offers "Amp-inspired" dark and light themes that follow the system appearance. It is not Amp's own.

#### 1.3 Keys, palette, modes and settings (current docs)

| Keys | Action |
| --- | --- |
| Ctrl+O | Command palette (typing `/` also opens it) |
| `?` | Shortcuts help (seen on the June 2026 welcome screen) |
| Ctrl+G | Edit the prompt in `$EDITOR` |
| Ctrl+V | Paste an image (not Cmd+V) |
| Ctrl+S | Turn the Dial before the first message; after that, show the locked mode |
| Ctrl+R | Prompt history |
| ↑ / ↓ | Move through queued and earlier messages to edit them |
| Alt+T | Expand thinking and tool blocks |
| Alt+R | Fast mode on or off |
| Enter | Send. If the agent is busy, it steers the agent. |
| Enter Enter | On a queued message, steer it sooner |
| Esc Esc | Interrupt the agent's turn |
| Ctrl+C Ctrl+N | Archive the thread and start a new one |
| Ctrl+C Ctrl+E | Archive the thread and quit |
| Ctrl+C Ctrl+C | Quit |
| Shift+Enter, Ctrl+J, `\` then Return | New line |
| `@` / `@@` | Mention a file / search for a thread to mention |
| Ctrl+X P | Open Puck (Ctrl+X is the default leader key) |

- **Keymap.**
  - The `amp.keymap` setting changes bindings.
  - Chords are keys separated by spaces, for example `"<leader> u"`. Use an array to keep the default key and add another. Use `null` to unbind.
  - `amp config keymap` prints the full map and every command.
  - User settings override workspace settings.
- **Terminal support.**
  - The docs list Ghostty, WezTerm, Kitty, iTerm2, and tmux with `extended-keys` on, for Shift+Enter.
  - The docs give the exact tmux lines to set this up, and say Amp will do it for you if you ask.
  - On Windows (WSL), the docs recommend WezTerm or Alacritty over Windows Terminal, for the clipboard.
  - The docs do not mention the kitty keyboard protocol or true colour.
- **Modes: "The Dial".**
  - The positions are low, medium (the default), high and ultra. Each position chooses the model, reasoning effort, system prompt, tools and "oracle".
  - The mode is fixed at the first message. The reasons given are that models get confused when they take over another model's conversation, and that a change would break the prompt cache.
  - You can build your own dial from two to four modes.
  - "Claude Code" is now a mode as well. It runs the Claude Agent SDK, and on a runner it runs the `claude` binary.
- **Slash commands.** Removed in October 2025 in favour of the palette, because the palette "run[s] commands without editing your prompt". Plugins add palette commands with `amp.registerCommand`.
- **Thread picker.** The October 2025 palette had "thread switch thread". Ctrl+\ opened a sidebar from June to August 2026. **UNVERIFIED:** the current way to switch threads inside the TUI. `amp threads continue T-…` attaches from the shell.
- **Settings.**
  - JSON or JSONC at `~/.config/amp/settings.json`, plus the nearest `.amp/settings.json` going up to the repository root.
  - `amp config edit`, or `amp config edit --workspace`, opens them.
  - Admins can deploy managed settings.
  - Keys include `amp.notifications.enabled`, `amp.showCosts`, `amp.updates.mode` (auto, warn or disabled), `amp.terminal.copyOnSelect` and `amp.thread.autoArchiveOnQuit`.
- **Themes.** Custom themes were removed in Neo (May 2026). No theme setting is documented now.

### 2. Threads

- **What a thread is.** One conversation with prompts, replies, tool calls and file changes. Its URL is `https://ampcode.com/threads/T-…`. Where the agent runs (local CLI, orb or runner) is separate from where you view the thread.
- **Sync across devices.** The same thread opens on the web, in the CLI and in the macOS and iOS apps.
  - "Cross-Client Access" lets the web drive a thread that runs in your local CLI. You can send, queue, dequeue and cancel messages. The web shows "● Connected Amp CLI · /Users/…/project · main".
  - It can be turned off per workspace. A recent passkey can be required.
  - `amp --remote-control-terminal` also exposes a web terminal.
  - `amp.remoteThreadCreation.enabled` lets the web open new threads in your open TUI.
- **Transport.** The security page says clients connect over secure WebSockets to `production.ampworkers.com`. A community analysis of the Neo binary says remote control runs over Rivet actors, with CBOR over WebSocket. **UNVERIFIED.**
- **CLI commands.** `amp threads list`, `search`, `continue`, `share T-… --visibility unlisted`, `visibility private`, `archive`, `label`, `rename`, `delete`, `markdown`, `export`.
  - `amp threads continue T-… -ox "…"` sends one message to an orb thread and exits.
  - Commits carry an `Amp-Thread-ID` trailer.
- **Sharing.** Private, Workspace, Group (Enterprise) or Unlisted.
  - Public, discoverable threads were removed in June 2026, because models now read so much that a review cannot reliably find leaked secrets.
  - "Contribute" access turns on multiplayer for seven days. Every workspace member can then use the thread's orb, secrets, files and terminal.
- **Housekeeping.** Archive hides a thread but keeps its URL working. Snooze hides it until the next agent message, 8 am or Monday. There are also pin, colour, labels, rename and delete.
- **Background and remote work.** See `-ox`, `--executor` and runners in section (b).

### 3. DX

- **Install.**
  - `curl -fsSL https://ampcode.com/install.sh | bash` for macOS, Linux and WSL.
  - `https://ampcode.com/install.ps1` for Windows.
  - A Homebrew tab on the site.
  - npm still works but is not recommended.
  - The binary is a single Bun-compiled file. The version format is `0.0.<unix time>-g<git sha>`, and a new release comes out about every day.
  - **Homebrew:** the official tap `ampcode/homebrew-tap` has the formula `ampcode`, which downloads the same per-platform binaries. The exact command on the site, probably `brew install ampcode/tap/ampcode`, is **UNVERIFIED**.
- **What the install script does** (read, not run):
  - It checks for a TTY, `NO_COLOR` and UTF-8. It draws an orb header and step glyphs (`·──◍`, `◍──·`, `·─!`, `·─╳`), with ASCII fallbacks.
  - It checks prerequisites. It downloads the gzip-compressed binary to a staging file, checks its SHA-256, then moves it into place in one step.
  - It installs to `~/.amp/bin`, links the binary from a folder already on `PATH`, and asks before it edits the shell profile.
  - `AMP_VERSION` pins a version. `AMP_HOME` moves the install.
  - The minisign signature check is in the script but commented out.
- **Updates.**
  - Amp updates in the background, and you restart to use the new version.
  - `amp update` and `amp version` exist.
  - `amp.updates.mode` and `AMP_SKIP_UPDATE_CHECK=1` control updates.
  - Runners check about once an hour. They restart into the new version only when no thread is running, at most once every 12 hours, and keep their ID and flags.
- **Auth.**
  - `amp login` adds an account through the browser.
  - `amp account list` and `amp account switch` manage accounts. `amp logout --all` removes them all.
  - A running process keeps the account it started with.
  - CI uses `AMP_API_KEY` (tokens start with `sgamp_`). The short-lived login token is refused in that role.
- **Config and plugins.**
  - Settings are as above.
  - Plugins are TypeScript files in `.amp/plugins/`, in `~/.config/amp/plugins/`, or set per person or per workspace.
  - Plugins can handle events (`session.start`, `agent.start`, `tool.call`, `tool.result`, `agent.end`) and add tools, commands and skills.
  - Plugins can show UI with `ctx.ui.notify`, `confirm`, `input` and `select`. `requireHuman: true` stops an agent from answering an approval prompt.
- **MCP.**
  - `amp mcp add <name> -- <command>` or `amp mcp add <name> <url>`.
  - Remote definitions are stored on ampcode.com with `amp mcp remote --personal|--workspace|--project`.
  - OAuth opens a browser. On a headless runner, the sign-in appears as a dialog in the web thread.
  - Servers in a workspace's `.amp/settings.json` need approval: `amp mcp approve`. `amp mcp doctor` shows "awaiting approval".
  - Skills can bundle an `mcp.json`, so the tools load only when the skill loads.
  - **Toolboxes** (August 2025, "Bring Your Own Tools") are not in the current docs. **UNVERIFIED** whether they still exist after Neo.
- **Permissions.**
  - September 2025: ordered rules with allow, ask, reject and `delegate` to an external program. The program's exit code means 0 allow, 1 ask, 2 reject.
  - Since Neo (May 2026), "Amp does not ask for approval before running tools". The old behaviour is a built-in plugin, and it switches on if you set the old keys.
  - The reason given: a tool call's name and arguments no longer show whether it is destructive, so checks like `rm -rf` give "a false sense of security".
- **Execute mode.**
  - `amp -x "prompt"` runs one turn, prints the final message and exits.
  - If stdout is redirected, execute mode switches on automatically.
  - Piped stdin becomes the prompt, or is added to it.
  - Other flags: `--attach <file>`, `--mcp-config`, `--timings` (to stderr), `--fast`, `--executor runner:<id>`.
- **Streaming JSON.**
  - `--stream-json` prints one JSON object per line: `system` (init), `user`, `assistant`, `result`. Errors have subtypes such as `error_max_turns`.
  - `--stream-json-input` reads JSONL user messages from stdin. `steer: true` interrupts a busy agent.
  - The docs warn that `--stream-json-thinking` "isn't compatible with Claude Code". That implies the base schema follows Claude Code's stream-json. This is my inference.
- **Shell completions.** `amp completions bash|zsh|fish|pwsh` work offline and without sign-in.

---

## (a) Catalogue of UX and DX patterns, and how they map to simstim

Each pattern has three parts: what Amp does, why it works, and where it fits in simstim. The simstim screens are: the setup wizard, the host browser, the session list, attach and detach, notifications, and errors.

**P1. One anchor box that carries status in its border.**

- Amp: the prompt box is the only fixed element. Its border holds the mode (top-right), the cost (top-right), live status (bottom-left: "Thinking 78 tok", "Running tools", "Enter to steer") and the location (bottom-right: "~/code/app (main)" or "In the Orb (tucan)").
- Why it works: there is no separate status bar, the eye has one place to look, and the frame changes when the state changes.
- simstim:
  - In the host browser and session list, use one framed panel. Put the selected host or session and its state (●/○, uptime) in the top border, the location (for example "homelab-vm via bastion") in the bottom-right, and key hints in the bottom-left.
  - Do **not** draw this frame while attached. The agent owns the whole screen there.

**P2. Dialogs with the title in the top border and hints in the bottom border.**

- Amp: "I scanned the project. Which improvement should I implement?" sits in the top border. "Enter submit · Esc cancel" sits in the bottom border. Key names are in an accent colour, verbs are dim, and a middle dot separates them. The background is dimmed.
- Why it works: the dialog is compact and the keys are always visible. Every dialog looks the same, so users learn it once.
- simstim: use this for every confirmation and form: "Kill session `api-refactor` on gcp-dev?", the add-host fields, and rename. Use the same hint grammar everywhere: `Enter confirm · Esc cancel · Tab next field`.

**P3. A command palette instead of slash commands.**

- Amp: Ctrl+O (and `/`) opens a fuzzy palette. Rows read "category action". Bindings are right-aligned. Entries can take context from what is selected (`prompt queue prompt "then run the build"`). Plugins add entries.
- Why it works: you can always find every action. You do not have to remember syntax, and the palette teaches the shortcuts.
- simstim:
  - One palette on every non-attached screen, with entries such as `session new`, `session attach`, `session kill`, `session rename`, `host add`, `host test`, `host remove`, `settings open`, `keymap show`. Each shows its key.
  - Entries depend on the selection, for example `session kill "api-refactor"`.
  - Plain `simstim` with no arguments should open on this model.

**P4. A welcome or empty state that teaches the three things you need.**

- Amp: an animated orb, "Welcome to Amp", "ctrl+o for commands", "? for shortcuts", and one "NEW:" line.
- Why it works: it is calm and it shows exactly the next step. The "NEW" line announces changes without a changelog popup.
- simstim:
  - First run, with no hosts: a short mark, "No hosts yet", and "Enter to add one · ? for shortcuts".
  - A host with no sessions: "Enter to start a session · Ctrl+O for commands".
  - One "NEW:" line after an update.

**P5. Fold the steps and show the outcome ("Less Noise").**

- Amp: tool steps fold to one line ("✓ Edited 2 files +28 -3 ▸"), and Alt+T expands them. The reason given is that people run many agents and care about the result.
- Why it works: you can scan the screen, and the detail is one key away.
- simstim: each session row is one line: name, agent, host, state, last activity, and a short tail of recent output or the agent's title. Space expands a preview of the last screen, like Amp's Agents Panel. Enter attaches.

**P6. A list with keys that do one job each (Amp's Agents Panel, January 2026).**

- Amp: `j`/`k` or arrows to move, Space to expand the last message, Enter to open, `e` to archive or unarchive, Esc to switch focus between the list and an input that starts a new thread.
- Why it works: the keys are vim-like and need no mode, so the list and creating a new item feel like one place.
- simstim: use this as the session list keymap:
  - `j`/`k`, Space preview, Enter attach, `x` kill (with confirmation), `r` rename, `a` archive or hide exited sessions, and `/` filter.
  - Esc or `n` opens a bottom input: "new session on gcp-dev: claude".
  - Group the rows the way the Amp web sidebar does: "Running", "Waiting for input", "Idle", "Exited", with relative ages ("6m", "2h").

**P7. Chips and pickers to set up new work (the Amp web runner picker).**

- Amp: the new-thread box has chips for the machine ("mac-mini"), the folder and the mode ("medium"). The folder picker (Ctrl+J) lists name, origin badge ("GitHub", "Amp") and path. A Tab toggle chooses "Current" or "New Worktree".
- Why it works: it takes three choices, each with a good default, and Tab cycles a binary choice without a menu.
- simstim: the new-session flow is Host ▸ Folder ▸ Agent ▸ (Current folder | New worktree).
  - The host picker shows a source badge ("ssh config", "added by simstim") and the jump path ("via bastion").
  - The agent picker lists the agents found on the host (claude, codex, opencode, gemini, amp, shell), each with its version.

**P8. Always show where the work runs.**

- Amp: the prompt border says "In the Orb (tucan)". The web shows "● Connected Amp CLI · /path · main". The Mac app shows "This Mac" in the runner picker. `--executor local|orb|runner:<id>` makes the choice explicit.
- Why it works: remote and local look the same, so a label stops mistakes.
- simstim:
  - Show "host · path · branch" on every row and in the terminal title.
  - On attach, set the terminal title to `simstim · <host> · <session> · <agent>`, with a state glyph in front.
  - Pass the agent's own OSC title through in a way the user can see, for example in the session name column.

**P9. Set the terminal title.**

- Amp: "amp - <thread title> - <cwd>", with a spinner glyph while it works.
- Why it works: Ghostty tabs become a status display at no cost.
- simstim: set the title in the browser and list. While attached, let the agent's own title through, or prefix it, as the user chooses.

**P10. A keymap with a leader key, chords and double-press guards.**

- Amp: Ctrl+X is the leader. Destructive actions need a double press (Ctrl+C Ctrl+C to quit, Esc Esc to interrupt). JSON settings take chords, arrays add keys, and `null` unbinds. `amp config keymap` prints everything.
- Why it works: few clashes with other keys, hard to quit by accident, and the user stays in control.
- simstim:
  - Detaching is the one key simstim takes while attached, and it must not clash with any agent. Copy the leader idea: a default prefix chord (to be chosen, and rebindable), plus a double press to send the literal key on to the agent.
  - Show the detach chord once, briefly, when you attach, then show nothing.
  - Ship `simstim keymap` to print the map.

**P11. Write down what the terminal and multiplexer need.**

- Amp: the docs name the terminals that support Shift+Enter, give the exact tmux `extended-keys` lines, and say Amp can set them up for you.
- Why it works: it heads off the most common "my key does nothing" bug report.
- simstim: the owner wants agent TUIs exactly as if they were local in Ghostty. Whatever simstim uses on the host must pass through extended keys (or the kitty keyboard protocol), true colour, OSC 8 links, OSC 52 clipboard and focus events. Check each one with `simstim doctor` and fix them on the host on its own, without the user asking. Amp's docs do not cover the kitty protocol or true colour, so simstim can do better here.

**P12. Leave multiplexing to the terminal.**

- Amp: removed its TUI sidebar because doing its own multiplexing on top of terminal tabs and panes was poor UX.
- Why it works: Ghostty already has excellent tabs and splits.
- simstim: attach mode is a clean full-screen passthrough. To watch two sessions, open two Ghostty tabs or splits, each running `simstim attach <id>`. Keep the session list as a fast, separate screen to switch from, not a frame around the agent.

**P13. Start in the background and print a handle.**

- Amp: `amp -ox "…"` creates a thread, prints its URL and exits. `amp threads continue T-…` attaches later. `--stream-json` streams JSON until the turn ends.
- Why it works: it fits scripts and fits "send the prompt, close the laptop".
- simstim:
  - `simstim new <host> --agent claude --detach` prints a short session ID and exits.
  - `simstim attach <id>` attaches. `simstim send <id> "text"` types into a session without attaching. Amp's `threads continue -ox` is the model for this.
  - `simstim ls --json` lists sessions as JSON.

**P14. Behave well when not on a TTY.**

- Amp: execute mode turns on when stdout is redirected. Piped stdin becomes the prompt. `--timings` goes to stderr, so stdout stays clean. `--stream-json` prints JSONL.
- Why it works: the same binary works by hand and in scripts.
- simstim: when stdout is not a TTY, print plain text or JSON and no TUI. Respect `NO_COLOR`. Put diagnostics on stderr. Pick stable exit codes, and document them.

**P15. An installer that is polite and safe.**

- Amp: one line to install. It checks the TTY, `NO_COLOR` and UTF-8, with ASCII fallbacks. It shows step glyphs, checks prerequisites, verifies SHA-256, stages the file and then moves it into place in one step, asks before editing the shell profile, and supports a pinned version. There is a Homebrew tap that downloads the same binaries.
- Why it works: it builds trust and does not surprise the user.
- simstim: copy all of it for the setup wizard's first step and for `install.sh`. Also turn on signature checking, which Amp's script has commented out. Publish a Homebrew tap from the start, since macOS comes first.

**P16. Updates that never break running work.**

- Amp: runners update in the background but restart only when no thread is running, at most every 12 hours, and keep their flags. Clients warn or auto-update according to `amp.updates.mode`.
- Why it works: an update never cuts off a running agent.
- simstim: this is a hard rule for simstim. Updating the client or the host helper must never kill a session. Design the session process so that it outlives the helper. Offer `updates.mode = auto | warn | off`.

**P17. Lifecycle hooks with timeouts and logs.**

- Amp: `.agents/setup` runs once per snapshot, must be idempotent, and is stopped after 20 minutes. `.agents/resume` runs after each wake, gets 10 seconds, and does not block. Each logs to a fixed path (`~/.cache/amp/logs/setup.log` and `resume.log`).
- Why it works: setup becomes predictable, and when it fails you know where to look.
- simstim:
  - A host setup hook: install tmux and the agents, and set up the terminal features.
  - A per-session start hook.
  - Each with a timeout and a log at a known path, which the wizard and `simstim doctor` print on failure.

**P18. Notifications that respect attention.**

- Amp: a sound when a task completes or needs input. Over SSH, or with `AMP_FORCE_BEL`, it uses the terminal bell. The apps send "ready for your next message" and never notify for the thread you are viewing. You can turn them off while the app is in front.
- Why it works: you get the signal without the noise.
- simstim: when a detached session rings the bell or emits an OSC 9 or OSC 777 notification, or goes from "busy" to "quiet" for N seconds, raise a local notification. Ghostty shows OSC 9 and 777 as desktop notifications. Never notify for the session you are attached to. Mark the state in the list as "waiting".

**P19. Error and doctor UX.**

- Amp: error and warning glyphs in the installer. Inline red exit codes, for example "(exit code: 127)". The installer prints the exact `export PATH=…` line to add. `amp mcp doctor` explains "awaiting approval". When orbs are paced, the thread shows the estimated wait instead of failing.
- Why it works: every error says what happened and what to do next.
- simstim:
  - `simstim doctor [host]` checks SSH, the jump host, the host helper, terminal features, and the agent binaries and their versions.
  - Every error gives one next step ("Run `ssh-add`", "Host unreachable through bastion; check `ssh bastion` first").
  - An exited session shows its exit code inline.

**P20. Stable IDs, archive versus delete, labels.**

- Amp: `T-…` IDs. Archive keeps the history and stops the orb. Delete removes it. Labels and snooze.
- Why it works: cleaning up is safe, and you can always refer to a session.
- simstim: give each session a short, stable ID and a name the user can change. "Kill" ends the process. "Hide" or "archive" keeps a record and the scrollback log of an exited session. Rows can carry labels.

**P21. Opinionated choices, sometimes taken too far.**

- Amp: removed themes, the sidebar, editor extensions, handoff, Amp Tab and public threads, each with a public reason.
- Why it works: fewer features, each done well, and users trust a team that explains its cuts.
- simstim: keep the scope small. But the owner's rule is "no compromise on keybindings or visuals", so simstim must never remove the ability to rebind keys or to fit the user's terminal theme.

**P22. Colour that fits the user's terminal.**

- Amp: the official screenshots appear to take on the terminal's palette (Gruvbox in May 2026), with accents in amber, teal, green and red, and dim grey for secondary text. **UNVERIFIED.**
- Why it works: the TUI looks native in any theme.
- simstim: use ANSI palette roles (fg, bg, dim, accent, success, warning, error) for all chrome, so it matches the user's Ghostty theme. Use 24-bit colour only for small decorative accents. Test on light and dark themes.

---

## (b) Orbs explained, and what simstim should learn

### What an orb is

An orb is a remote machine that Amp creates for one thread. It gets a fresh clone of the repository, the user's plugins and tools, and the thread's context. The agent then works there unsupervised. Thorsten Ball calls them "Remote machines that we spin up for you, put your code into, and start the Amp agent on", and says they are "more than just VMs".

- **Technology.**
  - Orbs run Debian 12. "Default E2B orbs run as microVMs in Google Cloud `us-west1`".
  - The security page says E2B "provides ephemeral compute instances for Amp orbs".
  - Firecracker is not named. A third-party write-up suggests it. **UNVERIFIED.**
- **Preinstalled software.** `amp`, `gh` (signed in), git, SSH, tmux, Bun, Node, pnpm, Yarn, Python, `uv`, `agent-browser`, ffmpeg, ripgrep, fzf, vim and more. Docker is not installed.
- **Features around an orb.**
  - A Changes pane with a Ship button: commit, rebase, run tests and push, or push to a branch.
  - A Files pane.
  - A **Terminal**: "a shell in a shared tmux session" that uses the same filesystem as the agent.
  - **Portals**: authenticated URLs on `onamp.dev` for any HTTP port. They can be made public for one hour to seven days.
  - A Linux **Desktop** tab.
  - **Secrets and env vars** at three levels: workspace, project and personal. Values are masked in output and every change is recorded.
  - **OIDC workload identity**: `amp orb id-token --audience …` for GCP, AWS and Tailscale.
  - **Agent to agent**: agents start other orbs and send messages and files.
  - **Automations**: schedules.
  - **Event-driven wakes**: GitHub CI, Linear, Discord and monitor webhooks.
  - **Multiplayer.**
  - **Puck**, an assistant that starts and watches agents.

### Lifecycle

| Stage | How it works | Interfaces |
| --- | --- | --- |
| Create | A new thread with executor "New Orb". Amp restores a matching snapshot or builds one: pre-clone script, clone, pre-setup, `.agents/setup` (20-minute limit), then saves the snapshot. A snapshot is reused for up to 72 hours. | Web "New Thread"; `amp -ox "…"`; `amp --executor orb` (the TUI waits for the first prompt); palette "thread: new in orb"; agents, Puck, automations, webhooks |
| Run | The agent works unsupervised. Long-running services are declared in `.amp/services.yaml`, not started from setup. | Web, apps, CLI |
| Watch | Open the thread URL on any device. The CLI can stream JSON. | `amp -ox … --stream-json`; `amp threads continue T-…` |
| Attach | No SSH is documented. You "attach" by opening the thread, the web Terminal (shared tmux) or a Portal. **UNVERIFIED** whether there is any SSH or local-terminal attach into an orb. | Web, apps; `amp threads continue` for the conversation |
| Sleep | Pauses when **both** are true: the agent has done no work for 5 minutes, **and** no person has interacted for 20 minutes. A running dev server does not keep it awake. Paused orbs cost nothing, and can sleep "for weeks". | Automatic; "Orb Saver Mode" stops a mere open from waking it |
| Wake | Opening the thread, opening a portal, or sending a message. `.agents/resume` runs (10 seconds, does not block). Files, memory and services come back. | Any client |
| Stop | Archiving the thread pauses its orb at once. | `amp threads archive`; web |
| Delete | Deleting the thread deletes its orb, "usually within seconds". | `amp threads delete`; web |

### Moving work between local and remote

- **Remote to local:** `amp sync <thread>` mirrors the orb's changes into your local checkout and keeps watching until Ctrl+C. The Ship button pushes from the orb.
- **Local to remote:** run `amp -ox` in a repository. The CLI matches the git remote to an Amp project. If none matches, it warns you and starts an orb with no project. Piped input such as `git diff main | amp -ox "…"` carries local context.
- **Steering a remote thread from the shell:** `amp threads continue T-… -ox "…"`.
- **Runners: the self-hosted analogue.**
  - `amp --no-tui --runner-id <id> [--dir …] [--discover-dirs] [--share] [--remote-control-terminal] [--desktop] [--amp-env]` turns any machine into a target for threads. Examples include a Raspberry Pi, a server or a cloud dev box.
  - Code runs as the OS user that started the runner. Amp "doesn't create, size, pause, or wake it".
  - Portals work only on orbs.
  - The Mac app runs a runner for you, with "Keep This Mac Awake".
  - Runners still depend on the ampcode.com control plane. Amp does not offer self-hosted deployments.
  - The portals page mentions "self-hosted orbs" once. **UNVERIFIED**, and it conflicts with the security page.

### Pricing and limits (October 2026)

- **Billing.** Per minute while running. Each size includes 60 GB of disk.

| Size | vCPU | Memory | Price per hour |
| --- | --- | --- | --- |
| a1.tiny | 1 | 2 GB | $0.08 |
| a1.small | 2 | 4 GB | $0.17 |
| a1.medium (default) | 4 | 8 GB | $0.33 |
| a1.large | 8 | 16 GB | $0.66 |
| a1.xxlarge | 16 | 32 GB | $1.32 |
| a1.3xlarge (top tiers only) | 16 | 44 GB | $2.13 |

- **Plans.**
  - Hobby is free: orbs are pay-as-you-go, and your own runners are free.
  - Individual is $20 a month ("Megawatt/Gigawatt") and includes "45,000 minutes of orbin' time".
  - Teams costs no extra. Enterprise is custom.
  - A third-party review says Gigawatt is $200 a month. **UNVERIFIED.**
- **Starting orbs.** Each user can start a burst of 20 metered orbs, then one new orb every five minutes. The thread shows the estimated wait. Higher limits come through support.
- **Uploads.** Up to 100 MB per file, or 512 MB with the higher limit.

### Security model

- **Isolation.** One sandboxed machine per thread. Filesystems and snapshots are encrypted at rest. Amp's servers do not keep a copy of the orb filesystem, but files the agent reads end up in the thread data.
- **Secrets.** They reach new orbs as environment variables. `amp orb restart-processes` reloads them. They are masked in output. Setup scripts get only a credential limited to the project or workspace, and it is removed before the snapshot.
- **Access.** Anyone who can view a thread can reach its portals. While multiplayer is on, "every workspace member can access the thread, its orb or shared runner, secrets, files, and terminal". Passkeys can be required for remote control.
- **Agent permissions.** No approval prompts by default since Neo. The docs suggest a policy plugin, or an isolated environment for untrusted repositories.
- **Not documented:** network egress controls. **UNVERIFIED.** Amp is SOC 2 Type II and hosted on GCP in the US.

### Comparison: what simstim should learn, and where it should differ

simstim is self-hosted, works with any agent, and runs sessions on any host the user can reach by SSH. Orbs are a hosted, Amp-only, per-thread sandbox. They overlap most in lifecycle and visibility.

**Learn from Orbs**

1. **Sessions outlive the client.** "Send prompt, close laptop" is the headline promise on the Amp homepage. simstim's sessions must survive detach, Ctrl+C in the client, network loss and a client crash. The orb terminal is itself a shared tmux session, which supports a design built on a terminal multiplexer on the host.
2. **One sandbox per task, as an option.** Amp sells "no worktrees to juggle, no port conflicts". simstim can offer "New worktree" when it creates a session (pattern P7), so parallel agents on one host do not collide.
3. **A clear lifecycle with plain-English states.** Amp's hero animation shows rows such as "paused · $0". simstim's list should show running, waiting for input, idle, exited (code) and host offline, with ages.
4. **Sleep and wake for cloud hosts, without tying simstim to a provider.** A user can give a host optional `wake` and `sleep` commands (for example `gcloud compute instances start …`). simstim runs `wake` on attach or new, and `sleep` after an idle rule the user sets. It stays provider-agnostic because the user writes the commands. This is a suggestion and needs a design record.
5. **Setup and resume hooks.** Copy `.agents/setup` and `.agents/resume`: idempotent, with timeouts, logged to known paths. Use them per host and per session.
6. **Ports as links.** Portals are popular. simstim's version is local only: detect ports the session is listening on and offer an SSH `-L` forward with a `localhost` URL. Never make one public by default.
7. **Sync back.** `amp sync` (mirror and watch) is a good model for a later `simstim sync <session>` built on git or rsync. Not for the first version.
8. **Start and steer from the shell.** `-ox` and `threads continue -ox` map to `simstim new --detach` and `simstim send`.
9. **Pace starts instead of failing.** If a host is at capacity, simstim should queue the new session and show the estimated wait, as orbs do.
10. **Show where it runs, and its cost.** "In the Orb (tucan)", plus a cost counter. simstim could show an optional hourly rate per cloud host, set by the user, as a cost hint for VMs that are running.

**Where simstim should differ**

1. **No cloud control plane and no account.** Orbs and runners both go through ampcode.com, over WebSockets. simstim talks to the user's hosts directly over SSH, including `ProxyJump` through a homelab jump host. On Hacker News, dbmikus asked for exactly this: "Just give me OpenSSH" with normal key management. simstim needs no inbound ports beyond SSH.
2. **The agent's real TUI, byte for byte.** Amp's web and apps re-render a thread as their own UI. simstim must pass the agent's PTY output through unchanged, so Claude Code, Codex, opencode, Gemini CLI, Amp or a shell look and behave as they do locally in Ghostty. Re-rendering is out of scope.
3. **Any agent, and the agent decides its own policy.** Amp's permissions, compaction and modes belong to Amp's agent. simstim should not add agent logic. Each agent's own permission prompts appear inside its TUI. simstim's own confirmations are only for simstim actions, such as killing a session or removing a host.
4. **Terminal first.** Amp moved the "many agents" view to web and native apps, and removed the TUI sidebar. Community tools such as `lazyamp` (a TUI for managing runners) and `manager` (a TUI for many Amp CLIs) appeared to fill that gap. simstim's host browser and session list should be the fast terminal home that Amp's TUI no longer is.
5. **Honest persistence.** Orb state is durable in a cloud system, and paused orbs keep memory. simstim sessions are processes on the user's host. They survive a disconnect, but not a host reboot, unless the user adds a restore step. The UI and docs must say so plainly.
6. **Private by default.** There is no multiplayer, there are no shareable links and there are no public portals. If several clients attach to one session, that must be a deliberate choice, with a clear rule for window size.
7. **No secrets inside simstim.** Amp stores and injects secrets. simstim should rely on the host's own environment and the user's SSH agent, and store nothing secret. Agent forwarding should be off by default. Its risk deserves its own decision record.
8. **Cost.** simstim is open source and free. The user pays only for their own VMs.

---

## Community verdict

**What people like**

- **Polish and craft.** Adam Wathan, quoted on the homepage: "Use it, and be ruined." Taylor Otwell, quoted on the homepage: "Feels more like having an actual computer in the cloud vs. a restricted sandbox." On Hacker News, usef- (July 2026) wrote: "Their CLI seemed very well thought out to make you do things 'the correct way'".
- **Speed.** tecoholic on HN (January 2026): "Amp finishes the task in the time it takes Codex to build context."
- **Threads.** Hamel Husain calls threads his favourite feature: server-side, resumable on any device, and shareable. Alex Dunlop (June to August 2026) says Amp "puts [history] in front of you", and that subagent progress is visible "out of the box", where Claude Code needs plugins.
- **Orbs.** On HN (August 2026, 116 points, 43 comments), altschuler called orbs "just fantastic" and "the best $20/month I'm spending on any dev tools". josevalerio said they are "more consistent" than Codex cloud.
- **Opinionated design.** asadjb (February 2026) praises the team's "love of the craft" and its willingness to remove features.

**What people dislike**

- **Price.** This is the most common complaint, in the HN thread on the Amp Inc spin-out (December 2025), from Hamel ("it's not cheap"), from asadjb (about $10 a day) and from Alex Dunlop ("felt expensive"). The September 2026 "Free Agent" plan answers part of it.
- **Alternate-screen trade-offs.** Peter Steinberger criticises Amp's find, which searches only visible text, its selection, which does not feel native, and its custom scrollbar.
- **Churn.** Features keep disappearing: themes, handoff, the sidebar, the editor extensions, Amp Tab and public threads. A third-party guide (Verdent, which sells a competing product) warns that Amp moves fast without backward compatibility. An X post by Ian Hsiao says Thread Map was removed, and supports the decision (search snippet only, **UNVERIFIED**).
- **Orbs as "just a VM".** On HN, fwlr called them "a thin wrapper over a remote virtual machine". fishtoaster said they are "an increasingly-commodity experience". jchook said: "really dislike having to do my work in a web browser". dbmikus wants plain OpenSSH. Some disliked a phone-number step at sign-up, and Amp's sqs replied that it had been turned off.
- **No approvals by default.** Commentators called the Neo default a safety trade-off. Multiplayer gives wide access to secrets and the terminal.

## Unverified and open points

- The exact Homebrew command shown on ampcode.com (the tap and formula exist).
- Whether Neo kept the libvaxis-port framework.
- How a user switches threads inside the current TUI.
- Whether toolboxes still exist.
- Whether Amp's TUI deliberately uses the terminal's ANSI palette.
- Whether there is SSH or local-terminal access into an orb.
- Whether orbs use Firecracker; what egress controls exist.
- What "self-hosted orbs" means on the portals page.
- The WebSocket and Rivet transport details.
- The price of Gigawatt.
- X posts: I could not open X, so tweet texts come from search snippets.

---

## (c) Sources

Amp primary sources (fetched 14 and 15 October 2026):

- https://ampcode.com/
- https://ampcode.com/docs
- https://ampcode.com/docs/cli
- https://ampcode.com/docs/cli/keybindings
- https://ampcode.com/docs/cli/execute-mode
- https://ampcode.com/docs/cli/spawning-orbs
- https://ampcode.com/docs/cli/streaming-json
- https://ampcode.com/docs/cli/settings
- https://ampcode.com/docs/cli/runners
- https://ampcode.com/docs/cli/remote-control
- https://ampcode.com/docs/cli/clone
- https://ampcode.com/docs/threads
- https://ampcode.com/docs/the-dial
- https://ampcode.com/docs/tools
- https://ampcode.com/docs/customize/mcp
- https://ampcode.com/docs/customize/plugins
- https://ampcode.com/docs/orbs
- https://ampcode.com/docs/orbs/getting-started
- https://ampcode.com/docs/orbs/customizing
- https://ampcode.com/docs/orbs/sizes-and-costs
- https://ampcode.com/docs/orbs/handling-secrets
- https://ampcode.com/docs/orbs/portals
- https://ampcode.com/docs/orbs/shipping
- https://ampcode.com/docs/orbs/agent-to-agent
- https://ampcode.com/docs/orbs/desktop
- https://ampcode.com/docs/orbs/files-and-attachments
- https://ampcode.com/docs/collaborate/multiplayer
- https://ampcode.com/docs/macos-and-ios
- https://ampcode.com/docs/puck
- https://ampcode.com/security
- https://ampcode.com/pricing
- https://ampcode.com/install
- https://ampcode.com/install.sh
- https://ampcode.com/install.ps1
- https://github.com/ampcode/homebrew-tap (Formula/ampcode.rb)
- https://ampcode.com/what-are-orbs
- https://ampcode.com/notes/orbs-explained
- https://ampcode.com/notes/what-i-want-to-tell-you-about-orbs
- https://ampcode.com/notes/by-an-agent-for-an-agent
- https://ampcode.com/notes/permissions
- https://ampcode.com/chronicle
- https://ampcode.com/news/towards-a-new-cli
- https://ampcode.com/news/look-ma-no-flicker
- https://ampcode.com/news/through-the-agent-into-the-shell
- https://ampcode.com/news/new-threads-without-leaving-the-cli
- https://ampcode.com/news/cli-tab-navigation
- https://ampcode.com/news/handoff
- https://ampcode.com/news/command-palette
- https://ampcode.com/news/thread-map
- https://ampcode.com/news/agents-panel
- https://ampcode.com/news/the-coding-agent-is-dead
- https://ampcode.com/news/neo
- https://ampcode.com/news/npm-package-changes
- https://ampcode.com/news/drop-the-neo
- https://ampcode.com/news/end-of-public-threads
- https://ampcode.com/news/agents-everywhere
- https://ampcode.com/news/agents-in-orbs
- https://ampcode.com/news/agents-anywhere
- https://ampcode.com/news/pass-the-orb-to-the-left-hand-side
- https://ampcode.com/news/so-long-tui-sidebar
- https://ampcode.com/news/steer-dont-queue
- https://ampcode.com/news/free-agent
- https://ampcode.com/news/one-runner-is-now-enough
- https://ampcode.com/news/the-mac-app-is-your-runner
- https://ampcode.com/news/less-noise
- https://ampcode.com/news/use-your-claude-plan
- https://ampcode.com/news/take-puck-with-you
- https://ampcode.com/news/the-dial and https://ampcode.com/news/one-runner-many-worktrees (screenshots)

Official media viewed (screenshots and video frames):

- https://static.ampcode.com/news/neo-ask-user-choice.png
- https://static.ampcode.com/news/neo-intro-2.mp4
- https://static.ampcode.com/news/neo-queueing-1.mp4
- https://static.ampcode.com/news/neo-steering-1.mp4
- https://static.ampcode.com/news/neo-remote-control-2.mp4
- https://static.ampcode.com/news/agents-in-orbs-demo-new-in-orb-2.mp4
- https://static.ampcode.com/news/agents-in-orbs-demo-sync.mp4
- https://static.ampcode.com/news/cli-cmd-palette-demo-3.mp4
- https://static.ampcode.com/news/tui.mp4
- https://static.ampcode.com/news/new-look-tour1.mp4
- https://static.ampcode.com/news/dial-release.png
- https://static.ampcode.com/news/one-runner-many-worktrees-picker-dark.png

Community and third-party sources:

- https://news.ycombinator.com/item?id=49437117 (Orbs, August 2026)
- https://news.ycombinator.com/item?id=46124649 (Amp Inc spin-out, December 2025)
- https://news.ycombinator.com/item?id=48742031 (usef-)
- https://news.ycombinator.com/item?id=46738736 (tecoholic)
- https://news.ycombinator.com/item?id=47860985 (hamish-b, who says he works at Amp)
- https://hn.algolia.com/api/v1/search?query=ampcode (thread index)
- https://hackerlinks.cc/items/amp-orbs/
- https://steipete.me/posts/2025/signature-flicker
- https://hamel.dev/notes/coding-agents/amp.html
- https://asadjb.com/blog/2026-02-08-i-keep-coming-back-to-amp
- https://www.alexdunlop.com/writing/amp-vs-claude-code-worth-switching-2026
- https://www.verdent.ai/guides/agents/amp-coding-agent (a competitor; not neutral)
- https://github.com/router-for-me/CLIProxyAPI/issues/3258 (community analysis of the Neo binary)
- https://github.com/miltonparedes/lazyamp
- https://github.com/ctrl-cheeb-del/manager (seen in search results only)
- https://pi.dev/packages/amp-themes (third-party Amp-style theme)
- https://github.com/rockorager/libvaxis
- https://x.com/thorstenball/status/1962892799767883812 (could not open; text from a search snippet)
- https://x.com/ian_hsiao_tw/status/2083869123050926125 (could not open; search snippet)
