# simstim UX

**Status:** accepted, 15 October 2026. This design replaces the first one, which had one home screen with panes and a permanent preview. [Decision 0009](decisions/0009-light-launcher.md) gives the reasons.

simstim has two kinds of screen, with opposite rules:

| | The launcher | An attached session |
| --- | --- | --- |
| What | Hosts, sessions, dialogs, the palette, the add-host wizard | The agent's real TUI |
| Who draws it | simstim, in the alternate screen | The agent, byte for byte |
| Job | Start or join a session in a few keys, then get out of the way | Everything else |
| Style | Amp's CLI, as observed: one framed list, calm and keyboard first | Untouched. Every row and column is the agent's. |
| Keys simstim reads | All of them | Only the detach key, and Ctrl+V when the clipboard holds an image |

The look and the key flow follow Amp's CLI, measured cell by cell in [Amp's TUI, observed](research/amp-tui.md). It is clean-room: we match what we observed, and copy no code, prompts, logos or fonts.

## 1. Principles

1. **One job per screen.** Pick a host, then pick or start a session. Each screen is one list in one column. There are no sidebars, split panes, permanent previews or dashboards.
2. **The session owns the terminal.** Once you attach, every row and column belongs to the agent. The client shows only two things of its own: the reconnect notice while the link is down, and system notifications (OSC 777), which draw nothing on the screen.
3. **Amp's look.** It uses notched frames, a yellow title, blue keys, faint secondary text, borders in the text colour, and a yellow selection bar with black text.
4. **The terminal's own colours.** Every colour is a palette slot or the terminal's default, so simstim matches your theme. Fixed RGB is kept for the brand only.
5. **Few keys.** Enter goes forward, Esc goes back, `n` starts a session, `?` shows the keys, and Ctrl+O shows everything else.
6. **Plain words.** Short labels, sentence case, no exclamation marks. Every error says what to try next.
7. **Motion with a purpose.** Things move when they change, are busy or need you, smoothly and in the terminal's own colours, and never in the way of a key. One switch turns all motion off.
8. **Fast paths.** The launcher draws at once from what it saw last time and opens where you left off, so `simstim` then Enter takes you back in. Numbers, the mouse and `attach -` shorten the common path further.

## 2. Screens

The mockups are 80 columns wide and come from the golden tests in `internal/tui/testdata`. Wider terminals get a wider box, up to 96 columns, centred. The box starts a few rows down, at the same row on every screen, so moving between screens does not shift it.

### 2.1 First run

With no hosts, the screen looks like Amp's welcome. Our own mark sits on the left, and a few lines sit beside it. In a narrow terminal the mark moves above the text.

```
          ⢀⡠⠤⣀  ⣀⣀⡀  ⢀⠔⠊⠉⠉⠒⢄⡀         Welcome to simstim
        ⢀⠔⠁   ⢑⡜  ⠘⢄⠔⠁    ⢀⢤⣚⣢⣄⡀
    ⠶⠶⣶⠪⢅    ⢠⠊⠈⢆ ⢠⠊⢆    ⣔⠕⠁  ⠈⠙⠳⠶    a to add a host (5 hosts in your SSH config)
       ⠉⠢⡑⠢⠤⠒⠁  ⢀⢷⠁  ⠣⡀⢀⡮⠊            ctrl+o for commands · ? for shortcuts
         ⠈⠑⠤⣀⣀⡠⠔⠁ ⠑⢄ ⣀⠜⠉
```

- **The mark** is three strands of a signal, woven into a ribbon that pinches at both ends. It is drawn in braille, so the curves are smooth, and coloured in the brand gradient. With motion on, the strands flow, a pulse runs along the ribbon, and the strands are brighter where they cross.
- **The title** is bold, in the brand gradient, and types itself in behind a cursor.
- **The hints:** the keys are bold and the words are faint. The first hint says how many hosts your SSH config already has.

### 2.2 Hosts

The first screen once hosts exist. With exactly one host, simstim skips it and opens that host's sessions.

```
  ╭─ Hosts ──────────────────────────────────────────────────────── 3 hosts ─╮
  │                                                                          │
  │ 1 ● atlas-dev  4 sessions · 1 needs you                            9ms  │
  │ 2 ◌ aws-gpu     offline                                                  │
  │ 3 ● gcp-dev     2 sessions                                         41ms  │
  │                                                                          │
  │ ssh atlas-dev · Debian 13 · 4 CPUs · claude, codex, opencode, shell     │
  │                                                                          │
  ╰────────────────── Enter open · n new session · a add host · ? shortcuts ─╯
```

- **Each row:**
  - Its number, faint: 1–9 open the row at once.
  - A state glyph: `●` green for online, the spinner while connecting, `◌` faint for offline, `✕` red for not ready.
  - The name, in bold.
  - A faint summary, in which the count that needs you is yellow.
  - The round trip, faint, on the right.
- **The selected row** is the selection bar. One faint line under the list says how the host is reached and what it runs.
- **Keys:**
  - Enter (or `l`, `→`, or its number) opens the host's sessions.
  - `n` starts a session on the selected host.
  - `a` adds a host, `i` shows its details, and `/` filters.

### 2.3 Sessions

One host's sessions. The host's name is the title, and its round trip (or "offline") sits at the top right.

```
  ╭─ atlas-dev ─────────────────────────────────────────────────────── 9ms ─╮
  │                                                                          │
  │ 1 ● auth-refactor    claude    ~/src/app            needs input      2m  │
  │ 2 ⢼ api-tests        codex     ~/src/api            working         14s  │
  │ 3 ○ notes            shell     ~                    idle             1h  │
  │ 4 ✓ docs-pass        opencode  ~/src/docs           exited 0         3h  │
  │                                                                          │
  │ Do you want to proceed? · ~/src/app · 6.1 of 8 GiB · s4                  │
  │                                                                          │
  ╰───────── Enter attach · n new · Space preview · Esc hosts · ? shortcuts ─╯
```

- **Each row:**
  - Its number, faint: 1–9 attach at once.
  - The state glyph, then the name in bold. A working session's glyph is a tiny Game of Life in one braille cell; the dot of a session that needs you breathes.
  - The agent and the folder, both faint.
  - The state in its colour: needs input yellow, working blue, failed red, others faint.
  - The time in that state, faint.
- **Order:** needs input, working, idle, then ended.
- **The detail line** under the list is about the selected session: what it waits for (in yellow), its folder, its memory and its ID.
- **Keys:**
  - Enter (or its number) attaches. On an empty list, Enter starts a new session instead.
  - `n` starts a session here.
  - `r` renames in place: the name becomes a text field and the hints change to `Enter rename · Esc cancel`.
  - `d` deletes, after a confirm.
  - `y` copies the attach command.
  - Space opens the preview.
  - `/` filters, and Alt+E (Opt+E on a Mac) hides or shows ended sessions.
  - Esc (or `h`, `←`) goes back to the hosts, on the same host.

**Empty:**

```
  │ No sessions on gcp-dev yet.                                              │
  │                                                                          │
  │ n new session                                                            │
```

**Offline**, in Amp's error form: a bold red "Error:", the message, then the next step in blue:

```
  ╭─ aws-gpu ────────────────────────────────────────────────────── offline ─╮
  │                                                                          │
  │ Error: Cannot reach aws-gpu: ssh: connect to host 198.51.100.4 port 22:      │
  │        Operation timed out.                                              │
  │                                                                          │
  │ ssh aws-gpu   (check that this works first)                              │
  │                                                                          │
  │ Retrying in the background.                                              │
```

**Every host.** `simstim flip`, or "view: all sessions" in the palette, shows one list of every host's sessions. Each host's name is a bold line above its sessions.

**Before the host answers.** The launcher remembers each host's sessions from the last run, and shows them at once, quiet (glyph and state faint), while SSH connects: the frame's border glows and the top right says "connecting". Enter attaches straight away. The detail line says "last seen before this run" until the live list arrives ([decision 0011](decisions/0011-instant-launcher.md)).

### 2.4 New session

`n`, or Enter on an empty list. It is a form dialog over the dimmed launcher.

```
  ╭─ New session on atlas-dev ──────────────────────────────────────────────╮
  │                                                                          │
  │ Agent      claude  codex  opencode  shell  other…   not installed: gemi… │
  │                                                                          │
  │ Folder     ~/src/app                                                     │
  │                                                                          │
  │ Name       auth                                                          │
  │                                                                          │
  │ Worktree   ● this folder   ○ new worktree                                │
  │                                                                          │
  ╰──────── Enter start · Tab next · Ctrl+Enter start detached · Esc cancel ─╯
```

- **Agent:** while the row has focus, the chosen agent is the selection bar. ←→ choose, and a letter jumps to an agent. The default is the agent used last on this host. Agents the host lacks are listed, faint. "other…" runs any command you type.
- **Folder:** recent folders and repositories show under the field. Tab completes a typed path from the host.
- **Name:** the folder's name by default.
- **Worktree:** this folder, or a new git worktree.
- **Enter** starts the session and attaches. Ctrl+Enter starts it and stays in the launcher. An error appears inside the dialog, in the "Error:" form.

### 2.5 Confirm a delete

A dialog in the danger tone: a red border and a red bold title.

```
  ╭─ Delete api-tests? ──────────────────────────────────────────────────────╮
  │                                                                          │
  │ This stops codex on atlas-dev and removes the session.                  │
  │ The folder ~/src/api stays. Unsaved work in the agent is lost.           │
  │                                                                          │
  ╰──────────────────────────────────────────────── Enter delete · Esc keep ─╯
```

### 2.6 Preview

Space opens the selected session's last screen in a dialog over the dimmed list. It loads when it opens, and refreshes while it stays open. It is never a permanent pane.

```
  ╭─ auth-refactor ──────────────────────────────────────────── needs input ─╮
  │ claude · ~/src/app · 6.1 of 8 GiB                                        │
  │ ──────────────────────────────────────────────────────────────────────── │
  │ (the session's screen, as the terminal would show it)                    │
  ╰───────────────────────────────────── Enter attach · ↑↓ next · Esc close ─╯
```

Inside the preview, ↑↓ move to the next session, Enter attaches, and Esc or Space closes it.

### 2.7 Command palette and shortcuts

**Ctrl+O** opens the palette, laid out like Amp's.

```
  ╭─ Commands ───────────────────────────────────────────────────────────────╮
  │                                                                          │
  │ > ses                                                                    │
  │                                                                          │
  │  session  new                                                         n  │
  │  session  attach                                                  Enter  │
  │  session  rename                                                      r  │
  │  session  delete                                                      d  │
  │  session  preview                                                 Space  │
  ╰──────────────────────────────────────────────────────────────────────────╯
```

- **Layout:** a blank row, the filter line, a blank row, then the rows. There are no hints in the bottom border.
- **Each row:** the noun, faint and aligned right; two spaces; the verb in bold; the key on the right in bold blue.
- **The selected row** is the selection bar.
- **Commands that cannot run** stay listed as "(unavailable)", in faint bold. Enter on one says why, in a notice.

**`?`** shows the shortcuts inside the list's own box, under a rule, the way Amp shows them inside its prompt box. Press `?` again to hide them. They are built from the live registry, so a rebound key shows its new name.

```
  │ ──────────────────────────────────────────────────────────────────────── │
  │ ↑↓ move                             a add host                           │
  │ Esc back to hosts                   i host details                       │
  │ n new session                       / filter                             │
  │ Enter attach                        Alt+E show ended                     │
```

### 2.8 Notices and the quit chord

**Notices** are short lines under the box. They are not framed and not in a corner, and they read like the lines of Amp's transcript. They go away after 4 s, or 8 s for an error, and at most four lines show at once.

```
  ╰───────── Enter attach · n new · Space preview · Esc hosts · ? shortcuts ─╯
    ✓ Copied: simstim attach atlas-dev/auth-refactor.
    Error: Cannot reach aws-gpu: timed out.
           ssh aws-gpu
```

**Ctrl+C** opens Amp's chord popup at the bottom right. A second Ctrl+C quits. Any other key, Esc included, cancels. `q` quits at once. Quitting never stops a session.

```
                                                        ╭─ Ctrl+C then ────────╮
                                                        │ Ctrl+C quit          │
                                                        │    Esc cancel        │
                                                        ╰──────────────────────╯
```

### 2.9 Attached

The agent owns the whole terminal: every row and every column. There is no border, status bar, sidebar or session list.

- **Size:** the agent's PTY is the terminal's size, and follows every resize.
- **Window title:** the agent's own. simstim saves the title before attaching and puts it back on detach.
- **Detach:** Ctrl+\ goes back to the launcher, on the same session. Press it twice quickly to send it to the agent.
- **Lost connection.** This is the only thing the client draws on the screen:
  - At once, the title shows `⧗ reconnecting · atlas-dev/s4`, and an OSC 9;4 progress mark appears.
  - After 3 s, one faint line is drawn on the bottom row:

    ```
     simstim · connection lost · reconnecting · keys are not sent · ctrl+\ to leave
    ```

  - On reconnect, a replay or a snapshot redraws the agent's screen over the line, and the title comes back. A notice follows in the launcher: "Reconnected to atlas-dev (missed 2.1 MiB, replayed)."
- **System notifications** (OSC 777) tell you when another session needs you or ends, and when an image upload fails. They appear outside the terminal, so nothing on the screen changes.

### 2.10 Host details

`i`, or "host: details" in the palette.

```
  ╭─ atlas-dev ─────────────────────────────────────────────────────── 9ms ─╮
  │                                                                          │
  │ Route      atlas-dev                                                    │
  │ System     Debian 13 · amd64 · 4 CPUs · 8 GiB · 61 GiB free              │
  │ Helper     simstim-host 0.1.0                                            │
  │ Agents     claude · codex · opencode · shell                             │
  │ Limits     3.2 GiB per session · 4 sessions                              │
  │                                                                          │
  ╰─────────────────── t test · s shell · r reconnect · x remove · Esc back ─╯
```

### 2.11 Fast paths

- **Where you left off.** The launcher opens on the sessions of the host you last attached to, with that session selected. `simstim` then Enter takes you back in. Esc goes to the hosts.
- **At once.** It draws the sessions it remembers before any host answers, and replaces them when the host does.
- **Numbers.** 1–9 open the row with that number.
- **The mouse.** A click selects a row; a click on the selected row opens it. The wheel moves the selection. In the palette a click runs a command. A click outside the palette, the preview, the host details or a confirm closes it.
- **The tab title** counts the sessions that need you: `● 1 · simstim · homelab`. You can see it from another tab.
- **When nobody is looking.** If a session starts to need you while the terminal is in the background, the launcher sends a desktop notification (OSC 777).
- **From the shell.** `simstim attach -` attaches to the last session.

### 2.12 Add a host

`a`, from any screen. It is a dialog with one step per screen; the step count sits at the top right, and Esc goes back one step.

```
  ╭─ Add a host ─────────────────────────────────────────────── step 1 of 4 ─╮
  │                                                                          │
  │ Where does it live?                                                      │
  │                                                                          │
  │ SSH target   het                                                         │
  │              From your SSH config: hetzner-1  (1 found)                  │
  │ Name         het                                                         │
  │ SSH config   optional: an extra SSH config file                          │
  │                                                                          │
  ╰───── ↑↓ pick from SSH config · Tab next field · Enter test · Esc cancel ─╯
```

The steps:

1. **Where.** A target from `~/.ssh/config` (↑↓ pick, and typing filters), a name, and an optional extra SSH config file.
2. **Test.** Each check shows the spinner, then `✓` or `✕` with the time it took. A failure shows the error and its next step.
3. **Install the helper.** This is the consent screen. It says what is installed, where it goes and what else changes (linger, terminfo), and that nothing listens on the network.
4. **Ready.** The agents found, the system, and the memory cap per session. Enter starts a session on the new host, whose sessions are now open.

## 3. Keymap

These keys work in the launcher. You can rebind any key in `config.toml`, and `simstim config keymap` prints the effective map.

| Key | Where | Action |
| --- | --- | --- |
| `↑` `↓` or `j` `k` | Lists | Move |
| `1`–`9` | Lists | Open the row with that number: a host's sessions, or attach |
| Click, wheel | Lists, palette | Select, open the selected row, move; run a palette command |
| `Enter`, `l`, `→` | Hosts | Open the host's sessions |
| `Esc`, `h`, `←` | Sessions | Back to the hosts |
| `Enter` | Sessions | Attach, or start a session on an empty list |
| `n` | Anywhere | New session on the host in play |
| `a` | Anywhere | Add a host |
| `i` | Anywhere | Host details |
| `Space` | Sessions | Preview |
| `r`, `F2` | Sessions | Rename |
| `d` | Sessions | Delete, with a confirm |
| `y` | Sessions | Copy the attach command |
| `/` | Lists | Filter as you type; Enter keeps the filter, Esc clears it |
| `Alt+E` (`Opt+E`) | Sessions | Hide or show ended sessions |
| `Ctrl+Enter`, `Alt+Enter`, `Ctrl+J` | New-session dialog | Start detached |
| `Ctrl+O` | Anywhere | Command palette |
| `?` | Anywhere | Show or hide the shortcuts |
| `Esc` | Dialogs | Back or cancel |
| `q`, or `Ctrl+C` `Ctrl+C` | Anywhere | Quit. Sessions keep running. |
| `Ctrl+\` | Attached | Detach. Press it twice to send it to the agent. |
| `Ctrl+V` | Attached | Paste an image if the clipboard holds one; otherwise the key passes through |

On screen, keys are written as Amp writes them: `Enter`, `Esc`, `Ctrl+O`, `Space`, single letters in lower case, and `Opt` for Alt on a Mac.

## 4. Visual language

### 4.1 Palette

Every token is a terminal palette slot, the terminal's default colour, or the faint attribute. So in 256- and 16-colour terminals the launcher looks the same as in truecolour; it is your theme that decides the actual colours. Amp works the same way ([research](research/amp-tui.md#2-palette)).

The truecolour column gives the colours in the reference theme: the owner's Ghostty theme, which the README images also use.

| Token | Sent as | Truecolour (reference theme) | 256 colours | 16 colours | Used for |
| --- | --- | --- | --- | --- | --- |
| text | default foreground | `#BFBDB6` | default | default | Body text, names, borders |
| background | default background | `#0B0E14` | default | default | Never painted |
| faint | SGR 2 on the text | `#656565` (half way to the background) | SGR 2 | SGR 2 | Hint words, agents, folders, ages, empty states, the dimmed backdrop |
| title | slot 3, bold | `#F9AF4F` | 3 | yellow | Titles in borders, the chord title |
| selection | background slot 3, text slot 0 | `#F9AF4F` behind `#1E232B` | 3 and 0 | yellow and black | The selected row, the chosen agent |
| key | slot 4 | `#53BDFA` | 4 | blue | Keys (bold in lists and popups), the next step after an error, the spinner, working |
| success | slot 2 | `#7FD962` | 2 | green | `✓`, an online host |
| warning | slot 3 | `#F9AF4F` | 3 | yellow | Needs input |
| danger | slot 1 | `#EA6C73` | 1 | red | "Error:", failed states, the delete dialog's frame |
| cursor | reverse video | `#BFBDB6` block | reverse | reverse | Text fields |

### 4.2 The brand

One fixed gradient, from deep teal to electric cyan, colours the welcome mark and the welcome title. On a light background the teal is deeper, so it keeps its contrast. The terminal's colour support decides what you see:

| | Dark background | Light background |
| --- | --- | --- |
| Truecolour | `#0E7C7B` to `#00E5FF` | `#045A5A` to `#0098B8` |
| 256 colours (nearest index) | 30 (`#008787`) to 45 (`#00D7FF`) | 23 (`#005F5F`) to 31 (`#0087AF`) |
| 16 colours | cyan (slot 6), no gradient | cyan (slot 6), no gradient |

### 4.3 Type, frames and spacing

- **Bold:** titles, names in lists, verbs in the palette, keys in lists and popups, and keys in the welcome hints.
- **Faint:** everything secondary (see the palette).
- **Frames:** rounded light lines (`╭╮╰╯─│`), in the text colour, or red in the danger tone.
  - The title sits in the top border at the left, and a fact at the right.
  - Key hints sit in the bottom border at the right, with a space each side and one `─` before the corner.
- **Spacing:**
  - One cell of padding inside each side of a frame, and a blank row inside the top and the bottom.
  - Two spaces between the columns of a row.
  - In the palette, a blank row above and below the filter line.
- **Truncation:** `…`. Paths are shortened in the middle and keep whole folders, `/home/dev/…/auth`; names are cut at the end.
- **Dialogs:** centred over the launcher, which is dimmed by setting the faint attribute on every cell, so the effect works with any theme. One dialog shows at a time.

### 4.4 Glyphs

| Glyph | Meaning |
| --- | --- |
| `●` green / yellow | An online host / a session that needs input (it breathes) |
| `∼ ≈ ≋` | The spinner: connecting, starting |
| `⢼` braille | A working session: a tiny Game of Life on the cell's eight dots |
| `○` | Idle |
| `✓` green | Done, exited 0, a check passed |
| `✕` red | Failed, exited with an error, a host that is not ready |
| `◌` | Offline, lost |
| `•` | An information notice |
| ` · ` | Between hints and between facts |
| `>` | The filter line |

### 4.5 Motion

Motion is there to show change, work in progress and things that need you. It is never decoration that a key has to wait for: keys act at once, whatever is moving.

| What moves | When | How |
| --- | --- | --- |
| The welcome mark | On the first-run screen | The strands flow and a pulse runs along the ribbon, at 30 fps |
| The welcome title | When the first-run screen opens | Types in, one letter every 38 ms, behind a cursor |
| List rows | When a list appears (start, hosts to sessions and back) | Fade in one after another: 22 ms apart, 140 ms each |
| A frame's border | While it is busy: a host connecting, a session starting, the wizard's checks | A soft glow travels round it, once every 1.6 s; the rest of the border dims |
| A dialog's border | When it opens, and the quit chord's popup | One lap of the glow in 0.5 s, fading as it goes |
| Notices | When they come and go | Fade in over 150 ms, and out over the last 300 ms |
| "Connecting…", "Starting…" | While they are true | A brighter band sweeps across the faint text |
| A session that needs you | While it waits | Its dot breathes, one breath every 1.6 s |
| A working session | While it works | Its braille glyph runs a tiny Game of Life |
| The spinner | Connecting, starting | A small wave, `∼ ≈ ≋`, every 200 ms |

**Colour.** To blend smoothly, the launcher asks the terminal for its real colours: the default foreground and background (OSC 10 and 11) and the 16 palette slots (OSC 4), as Amp does. Fades and glows then mix those colours in truecolour, so they match your theme. Without truecolour, or before the terminal answers, the same motion uses steps of faint, normal and bold instead.

**Cost.** One frame tick runs only while something moves: 30 fps for the welcome mark, glows and fades, 10 fps for breathing dots and working glyphs. When nothing moves, nothing repaints.

**One switch.** `ui.motion = auto | on | off`. `auto` turns motion off inside SSH sessions, in Emacs and JetBrains terminals, and when `NO_COLOR` or `TERM=dumb` is set. With motion off every animation draws one still frame: the spinner shows `≈` and a working session shows `≈`.

### 4.6 Capability probe

`simstim doctor terminal` sends one batch of queries, with DA1 as the end marker. It covers:

- the kitty keyboard protocol, synchronized output and OSC 52;
- XTVERSION;
- the OSC 10, 11 and 12 colours;
- modes 2027 and 2031.

It shows what the agent will get through the pass-through.

## 5. Feedback

**Notices** (section 2.8):
- "Detached. auth-refactor keeps running on atlas-dev."
- "Reconnected to atlas-dev."
- "Copied: simstim attach atlas-dev/auth-refactor."

A repeat of a notice extends it instead of adding a line.

**Errors:**

- Every error has a message and one next step.
- In the launcher and in dialogs it shows as a bold red "Error:" and the message, with the next step in blue.
- On the command line it prints the same way:

  ```
  Error: cannot reach atlas-dev through the jump host bastion.

  Try: simstim host test atlas-dev --verbose
  ```

**Empty states:** a faint statement, a blank row, then a faint hint with its key in blue. "No sessions on gcp-dev yet." / "n new session". "Nothing matches 'xyz'."

**Copy rules:**

- Use sentence case and plain words, with no exclamation marks.
- Say what keeps running: "Quit. Sessions keep running."
- Name the host and the session in every message.

## 6. CLI

The launcher is the home, and every action also has a command, for scripts and terminal keybindings.

| Command | Does |
| --- | --- |
| `simstim` | Opens the launcher |
| `simstim host add <name> [target] [--ssh-config f] [--provider-script s --wake] [--yes]` | Connects a host: probe, consent, install, setup, doctor |
| `simstim host ls`, `test`, `doctor`, `rm` | Host management |
| `simstim host restore <name> [--all]` | Starts again the sessions that a host restart stopped (or, with `--all`, a crash) |
| `simstim host start\|stop\|status <name>` | Runs the host's provider script |
| `simstim new <host> [--agent claude] [--dir ~/src/app] [--name x] [--worktree [--base ref]] [--detach] [-- cmd…]` | Starts a session, and attaches unless `--detach` |
| `simstim ls [--host h] [--json]` | Lists sessions, with their state and where the state came from |
| `simstim attach <host>/<session>` | Attaches directly. Good for a Ghostty keybinding. `simstim attach -` attaches to the last session. |
| `simstim flip [<host>/<session>]` | A quick switch: the sessions of every host in one list, then attach |
| `simstim send <host>/<session> <text> [--no-enter]` | Types text into a session without attaching |
| `simstim rename <host>/<session> <name>`, `kill <host>/<session>` | Renames; stops and removes (keeping a worktree that has work) |
| `simstim ports <host>/<session>`, `forward <host>/<session> [port…]` | Lists the ports a session listens on; forwards them to localhost |
| `simstim demo [--empty]` | The launcher on sample data; nothing connects anywhere |
| `simstim doctor keys` | Prints each key event as the terminal sends it, and marks the client's own keys |
| `simstim doctor terminal` | Asks the terminal what it supports: kitty keys, synchronized output, colours, and more |
| `simstim version`, `completion zsh\|bash\|fish` | Housekeeping |

**Exit codes:** 0 for success, 1 for a user error, 2 for a host or network error, 3 for a session or host that is not found.

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
# provider = { script = "~/.config/simstim/providers/gcp.sh" }
```

## 8. Not built yet

These parts of the spec are planned, and are not in the current build:

- using the capability probe for the launcher's own drawing (the probe itself is `doctor terminal`);
- live light and dark refresh through mode 2031 (the launcher reads OSC 11 once, at start);
- the ⧗ mark for hosts with a time limit (GCP IAP, AWS EICE);
- the disk check in the add-host wizard's test step;
- editing a host from host details.
