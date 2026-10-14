# simstim UX

**Status:** proposed, 15 October 2026.

simstim has two kinds of screen, with opposite rules:

| | simstim's own screens | An attached session |
| --- | --- | --- |
| What | Setup wizard, host browser, session list, dialogs, palette, help | The agent's real TUI |
| Who draws it | simstim, in the alternate screen | The agent, byte for byte |
| Style | Amp-grade: calm, fast, keyboard first | Untouched. No chrome, no colours, no keys added. |
| Keys simstim reads | All of them | Only the detach key and Ctrl+V with an image |

The patterns below come from studying the Amp CLI. They are clean-room: we copy ideas, never code, prompts or assets. The evidence is in [the Amp notes](research/README.md).

## 1. Principles

1. **One home screen.** Hosts on the left, sessions on the right, and a live preview below. Everything else is a dialog over it.
2. **Status lives in borders.** Context sits in the top border, key hints in the bottom border. No separate header or footer bars.
3. **The terminal's own colours.** Every colour token maps to an ANSI palette slot, so simstim matches your Ghostty theme with no config.
4. **Keyboard first, mouse welcome.** Every action has a key and a palette entry.
5. **Plain words.** Short labels, sentence case, no exclamation marks. Every error says what to try next.
6. **Calm motion.** One spinner style, short fades, and one switch to turn motion off.
7. **Never in the way.** Opening a session is one key, and leaving it is one key.

## 2. Screens

The mockups are 80 columns wide. The real screens grow with the terminal.

### 2.1 First run

```
                                ·  ·  ·
                          ·  ·  ∿∿∿∿∿∿∿  ·  ·
                     ·  ─────────╯     ╰─────────  ·
                          ·  ·  ·  ·  ·  ·  ·

                           jack in to any host

                   No hosts yet. Press a to add one.

                 a add host · ctrl+o commands · ? shortcuts
```

- **Hero:** a signal line that flattens, then pulses: the "flip" into a session. It is drawn with the dot ramp `· • ●` and box lines, in the accent gradient. It is still when motion is off.
- **Hint line:** exactly one, with at most three hints.

### 2.2 Add a host (wizard)

A notched dialog with one step per screen. Esc goes back one step, and a progress mark sits in the top border.

```
╭─ Add a host ─────────────────────────────────────────────────── step 1 of 4 ─╮
│                                                                               │
│  Where does it live?                                                          │
│                                                                               │
│  ▸ From my SSH config        atlas-dev, gcp-dev, hetzner-1   (12 found)      │
│    New SSH target            user@host, optional jump host                    │
│                                                                               │
╰─ ↑↓ choose · enter next · esc cancel ─────────────────────────────────────────╯
```

The steps:

1. **Where.** Pick from `~/.ssh/config` (with filter-as-you-type), or type a target and an optional jump host.
2. **Test.** Each check shows a spinner, then `✓` or `✕` with the time it took: connect, `uname`, disk, helper found. If a check fails, the error and a "Try:" line appear right there.
3. **Install the helper.** A clear consent screen:

   ```
   ╭─ Install simstim-host on atlas-dev? ─────────────────────────── step 3 of 4 ─╮
   │                                                                               │
   │  simstim needs a small helper on the host to keep sessions alive.             │
   │                                                                               │
   │    What    simstim-host 0.1.0 and simstim-vt, about 9 MB, SHA-256 checked     │
   │    Where   ~/.local/share/simstim/bin/0.1.0/                                  │
   │    Also    turn on linger, install Ghostty's terminfo                         │
   │                                                                               │
   │  Nothing listens on the network. Remove it any time with simstim host rm.     │
   │                                                                               │
   ╰─ enter install · d details · esc back ────────────────────────────────────────╯
   ```

4. **Ready.** A host card shows the agents found on `PATH`, the CPU and memory, and the memory cap per session. One button: "Start a session".

### 2.3 Home: hosts and sessions

```
╭─ simstim ─────────────────────────────────────────── 3 hosts · 5 sessions ───╮
│ HOSTS                  │ SESSIONS ─ atlas-dev                                │
│   all hosts        5   │ ● auth-refactor   claude   needs input        2m    │
│ ▸ atlas-dev   3   9ms │ ⠶ api-tests       codex    working           14s    │
│   gcp-dev      2  41ms │ ○ notes           zsh      idle               1h    │
│ ◌ aws-gpu      offline │ ✓ docs-pass       opencode exited 0           3h    │
│                        │╭─ auth-refactor ─────────────────────────────────╮  │
│                        ││ The last screen of the session, drawn exactly    │  │
│                        ││ as the shadow terminal holds it.                 │  │
│                        ││                                                  │  │
│                        │╰─ claude · ~/src/app · 6.1 of 8 GiB ─────────────╯  │
╰─ enter new session · tab sessions · a add host · ctrl+o commands · ? help ───╯
```

**Hosts pane:**

- **Rows:** name, session count, latency.
- **States:** offline hosts are dim, with `◌`. A host with a known time limit (IAP, EICE) shows a small `⧗` with a tooltip.
- **Keys:**
  - **Enter** starts a new session on the selected host (spec item 4).
  - **Tab** moves focus to the sessions pane.
  - **"all hosts"** lists sessions from every host, grouped by host.

**Sessions pane:**

- **Rows:** a status glyph, name, agent, state and age.
- **Order:** "needs input" first, then working, idle and exited.
- **Keys:**
  - **Enter** attaches.
  - **Space** shows or hides the preview.
  - **/** filters as you type.
  - **Pasting a session ID** jumps to it.

**Preview:**

- It shows the exact last screen, from the shadow terminal, after a 120 ms debounce.
- The bottom border shows the agent, folder and memory use.

**Status glyphs:**

| Glyph | State | Colour token |
| --- | --- | --- |
| `●` | Needs input | `warning` |
| spinner | Working | `primary` |
| `○` | Idle | `muted` |
| `✓` | Exited with 0 | `success` |
| `✕` | Exited with an error, or out of memory | `destructive` |
| `↻` | Restored after a reboot (v2) | `info` |
| `◌` | Host offline | `muted` |

The spinner is simstim's own: a braille "scan" (`⠁⠂⠄⡀⢀⠠⠐⠈`) at 120 ms.

### 2.4 New session

A notched dialog over the home screen. Enter starts the session and attaches at once.

```
╭─ New session on atlas-dev ──────────────────────────────────────────────────╮
│                                                                               │
│  Agent      claude   codex   opencode   gemini   amp   shell   other…         │
│             ▔▔▔▔▔▔                                                            │
│  Folder     ~/src/app                                    recent · repos · ↹    │
│  Name       auth-refactor                                                     │
│  Worktree   ● this folder   ○ new worktree  (v2)                              │
│                                                                               │
╰─ enter start · tab next field · ctrl+enter start detached · esc cancel ──────╯
```

- **Agent:** only agents found on the host can be chosen; a missing one shows "not installed". The last choice is remembered per host.
- **Folder:** completes from recent folders and from git repos found on the host.
- **Name:** the folder name by default. It can be changed later with `r`.

### 2.5 Confirm a delete

A notched dialog in the `destructive` tone. The backdrop is dimmed by setting the dim attribute on the cells underneath, so it works with any theme.

```
╭─ Delete api-tests? ──────────────────────────────────────────────────────────╮
│                                                                               │
│  This stops codex on atlas-dev and removes the session.                      │
│  The folder ~/src/api stays. Unsaved work in the agent is lost.               │
│                                                                               │
╰─ enter delete · esc keep ─────────────────────────────────────────────────────╯
```

### 2.6 Command palette

Ctrl+O opens it. Commands are "noun: verb", and one registry feeds the palette, help, keymap and shell completion. A disabled command stays visible, with its reason.

```
╭─ Commands ────────────────────────────────────────────────────────────────────╮
│ > ses                                                                          │
│   session: new                                                           n    │
│   session: attach                                                    enter    │
│   session: rename                                                        r    │
│   session: delete                                                        d    │
│   session: copy attach command                                           y    │
│   session: restart            not available: session is running             │
╰─ ↑↓ choose · enter run · esc close ───────────────────────────────────────────╯
```

The registry:

| Noun | Verbs |
| --- | --- |
| `host` | add, test, details, edit, remove, doctor, open SSH shell |
| `session` | new, attach, rename, delete, copy attach command, show log, restart (v2), forward port (v2) |
| `view` | all hosts, filter, toggle preview, toggle exited |
| `simstim` | settings (opens in `$EDITOR`), keymap, update, doctor keys, about |

### 2.7 Help

`?` opens a two-column grid built from the live keymap, so rebinding a key updates the help. The grid is 40 to 80 columns wide, and below 50 columns it stacks each key over its description.

### 2.8 Attached

No chrome at all. The agent owns the whole terminal.

- **Tab title:** simstim sets it to `simstim · atlas-dev/auth-refactor` before handing over. After that, the agent may change it freely.
- **Detach:** Ctrl+\ goes back to the home screen. Press it twice to send it to the agent.
- **Lost connection:**
  - At once, the title shows `⧗ reconnecting · atlas-dev/auth-refactor`, and an OSC 9;4 progress mark appears.
  - After 3 s, one dim line is drawn on the bottom row:

    ```
     ◌ connection lost · reconnecting in 4s · keys are not sent · ctrl+\ to leave
    ```

  - On reconnect, replay or a snapshot redraws the agent's screen over the banner. A toast follows on the home screen: "Reconnected to atlas-dev (missed 2.1 MB, replayed)".

### 2.9 Host details

```
╭─ atlas-dev ───────────────────────────────────────────────── online · 9 ms ─╮
│  Route      ProxyJump bastion over Tailscale                                     │
│  System     Debian 13 · x86_64 · 4 vCPU · 8 GiB · 61 GB free                  │
│  Helper     simstim-host 0.1.0 · linger on · terminfo xterm-ghostty           │
│  Agents     claude 2.1.300 · codex 0.162.1 · opencode 1.19.0 · zsh            │
│  Limits     8 GiB per session · 3 sessions · 6.4 GiB in use                   │
╰─ t test · e edit · x remove · s open shell · esc back ────────────────────────╯
```

## 3. Keymap

simstim's own screens use this keymap. Everything can be rebound in `config.toml`, and `simstim config keymap` prints the effective map.

| Key | Where | Action |
| --- | --- | --- |
| `↑` `↓` or `j` `k` | Lists | Move |
| `enter` | Hosts | New session on this host |
| `enter` | Sessions | Attach |
| `tab`, `shift+tab` | Home | Switch pane |
| `n` | Anywhere at home | New session |
| `a` | Anywhere at home | Add host |
| `d` | Sessions | Delete, with confirm |
| `r` | Sessions | Rename |
| `y` | Sessions | Copy the attach command |
| `space` | Sessions | Show or hide the preview |
| `/` | Lists | Filter as you type |
| `alt+e` | Sessions | Show or hide exited sessions |
| `ctrl+o` | Anywhere | Command palette |
| `?` | Anywhere | Help |
| `esc` | Dialogs | Back or cancel |
| `ctrl+c ctrl+c` or `q` | Home | Quit. Sessions keep running. |
| `ctrl+\` | Attached | Detach. Press twice to send it. |
| `ctrl+v` | Attached | Paste an image if the clipboard holds one; otherwise pass the key through |

## 4. Visual language

### 4.1 Colour tokens

Every token maps to the terminal's default colours or an ANSI slot (0–15), never to a fixed RGB value. Light or dark comes from OSC 11, and is refreshed live through mode 2031 where the terminal supports it.

| Token | Maps to | Used for |
| --- | --- | --- |
| `fg`, `bg` | Terminal default | Text, and a transparent background |
| `muted` | Bright black (8) | Hints, ages, secondary text |
| `primary` | Blue (4) | Focus, the working spinner, keys in hints |
| `secondary` | Cyan (6) | Host names, paths |
| `accent` | Magenta (5) | The hero, highlights |
| `border` | Bright black (8) | Frames |
| `success` | Green (2) | `✓`, done |
| `warning` | Yellow (3) | Needs input, time-limit marks |
| `destructive` | Red (1) | Errors, delete dialogs |
| `info` | Bright blue (12) | Restored, notices |
| `selection` | Reverse video | The selected row |

One fixed gradient is kept for brand moments only: the hero and the first-run title, from deep teal to electric cyan. It falls back to `accent` without truecolour.

### 4.2 Type, borders, spacing

- **Text styles:**
  - **Bold:** titles, key names in hints, the selected row's name.
  - **Dim:** hints, ages and the preview border.
  - **Underline:** links only.
- **Borders:** rounded light box lines (`╭╮╰╯─│`). The top border holds the title and context; the bottom border holds the key hints.
- **Spacing:** one cell of padding inside boxes, and one blank line between sections.
- **Truncation:** `…` in the middle for paths (`~/src/…/auth`), and at the end for names.

### 4.3 Motion

- The hero animates at up to 30 fps.
- Text changes cross-fade over 150 ms.
- The spinner ticks at 120 ms.
- Idle screens never repaint.
- One setting, `ui.motion = auto | on | off`, controls all of it. `auto` turns motion off inside SSH sessions, in Emacs and in JetBrains terminals, and when `NO_COLOR` or `TERM=dumb` is set.

### 4.4 Capability probe

At start-up, simstim sends one batch of queries, with DA1 as the end marker and a 300 ms timeout. It covers:

- the kitty keyboard protocol, synchronized output and OSC 52;
- XTVERSION;
- OSC 10, 11 and 12 colours;
- modes 2027 and 2031.

The result is cached per terminal, keyed by `TERM_PROGRAM` and XTVERSION. It drives simstim's own screens, and `simstim doctor` prints it, so the user can see what the agent will get through the pass-through.

## 5. Feedback

**Toasts:**

- At most three are visible. More wait in a queue, and a repeat extends the visible one.
- They last 3 s by default.
- Examples:
  - "Detached. auth-refactor keeps running on atlas-dev."
  - "Reconnected to atlas-dev."
  - "Copied: simstim attach atlas-dev/auth-refactor."

**Errors:**

- Every error has a message, an exit code and one "Try:" line.
- In the CLI it prints as a bold red `Error:` and the message, a blank line, then the suggestion in `primary`.

  ```
  Error: cannot reach atlas-dev through the jump host bastion.

  Try: simstim host test atlas-dev --verbose
  ```

**Empty states:** short and plain. "No hosts yet. Press a to add one." "No sessions on gcp-dev. Press enter to start one." "Nothing matches 'xyz'."

**Copy rules:**

- Sentence case, plain words, no exclamation marks.
- Say what keeps running: "Quit. Sessions keep running."
- Name the host and session in every message.

## 6. CLI

The TUI is the home, and every action also has a command, for scripts and terminal keybindings.

| Command | Does |
| --- | --- |
| `simstim` | Opens the home screen |
| `simstim host add [target]`, `ls`, `test`, `details`, `rm`, `doctor` | Host management |
| `simstim new <host> [--agent claude] [--dir ~/src/app] [--name x] [--detach]` | Starts a session, and attaches unless `--detach` |
| `simstim ls [--host h] [--json]` | Lists sessions |
| `simstim attach <host>/<session>` | Attaches directly. Good for a Ghostty keybinding. |
| `simstim flip` | A quick switcher: the session list on its own, then attach |
| `simstim kill <host>/<session>`, `rename` | Stops a session, or renames it |
| `simstim watch [--stream-jsonl]` | A live status stream, for scripts and status bars |
| `simstim doctor [keys]` | Local and host checks. `keys` prints each key event as JSON lines, here and through the pass-through. |
| `simstim update`, `version`, `completion zsh\|bash\|fish` | Housekeeping |

**Exit codes:** 0 for success, 1 for a user error, 2 for a host or network error, 3 for a session that is not found.

## 7. Config

```toml
# ~/.config/simstim/config.toml
[ui]
motion = "auto"          # auto | on | off

[attach]
detach_key = "ctrl+\\"
image_paste_key = "ctrl+v"
scrollback = "keep"      # keep | clear: what to do with local scrollback on a fresh attach

[sessions]
memory_max = "auto"      # auto = min(8 GiB, 40 % of host RAM)

[keymap]
"session: delete" = "d"
"session: rename" = ["r", "f2"]
```

```toml
# ~/.config/simstim/hosts.toml
[[host]]
name = "atlas-dev"
ssh = "atlas-dev"           # a Host from ~/.ssh/config, used as is
labels = { provider = "homelab" }
# provider = { script = "~/.config/simstim/providers/gcp.sh" }   # v2
```
