# Client stack, attach-mode fidelity and distribution

Research note for simstim. Written 2026-10-15.

This note covers the local side of simstim: the language and TUI framework, the raw "attach" mode, input interception, clipboard images, and how we ship the binary. Host-side choices (zmx and its alternatives) are out of scope here, except where they change what the client must do.

All versions and dates were checked on 2026-10-15 against repos, release pages, registries and source code. Source links pin a commit where it matters. Anything we could not check is marked **UNVERIFIED**.

## Summary

- **Stack:** Go, with Bubble Tea v2, Lip Gloss v2 and Bubbles v2. Bubble Tea v2 has been stable since 2026-02-24, and v2.1.0 shipped on 2026-10-08. It has the hand-off hook we need (`tea.Exec`), and its renderer already resets the terminal in the right order. A minimal static binary is about 4 MB and starts in about 4 ms.
- **Attach mode:** simstim runs its own thin pass-through pump. It runs `zmx attach` on the host over the system `ssh`. It detaches by closing the channel, never through zmx's detach key, because zmx's client sends RIS (`ESC c`) on detach, and in Ghostty RIS wipes all scrollback. simstim watches the output stream passively to track modes, and on detach it writes an explicit, ordered reset list.
- **Input:** a small purpose-built filter of about 170 lines recognises the detach key and Ctrl+V in every encoding: legacy, kitty CSI u (with event types, lock bits and alternate keys) and modifyOtherKeys. It forwards every other byte unchanged and never waits for a timer on a real key. A prototype passes a split-stream property test.
- **Clipboard:** `golang.design/x/clipboard` v0.11, which needs no cgo since v0.8.
  - macOS: NSPasteboard through `purego`. In our test a warm check takes about 20 µs, against 164 ms for `osascript`.
  - Linux: X11 or Wayland, with `wl-paste` and `xclip` as fallbacks.
  - Windows: Win32. Use Alt+V there, because Windows Terminal takes Ctrl+V.
  - Claude Code, Codex, Gemini CLI, opencode and Amp all attach an image when you paste its path. So "upload, then paste the remote path" needs no agent-specific code.
  - A dropped file path inside a bracketed paste is uploaded and rewritten.
- **Shipping:** GoReleaser, a Homebrew tap (cask), a `curl | sh` installer, Nix, and Scoop/winget later. macOS signing and notarisation, an SBOM, checksums and provenance. Apache-2.0.

The full recommendation is at the end.

## 1. Language and TUI framework

### 1.1 What the notable agent CLIs use

| CLI | Language and TUI | Shipped as | Licence |
| --- | --- | --- | --- |
| Amp | TypeScript. "Built on our own framework" (Amp post "Look Ma, No Flicker", 2025-09-02). The binary holds Flutter-style names and `_vaxis`, which hints at a vxfw-like design. That last point is **UNVERIFIED**. | Bun single-file executable (110 MB on this Mac); `curl` installer, npm, brew | Proprietary |
| Claude Code | TypeScript and React. A team member describes a scene graph, then layout, rasterise, diff and ANSI, within a ~16 ms frame budget ([HN](https://news.ycombinator.com/item?id=46699072)). The binary holds Ink host names, so it is likely a heavily changed Ink. That is **UNVERIFIED**. | Bun single-file executable (250 MB); `curl` installer, brew cask, winget | Proprietary |
| Codex CLI | Rust, Ratatui 0.30.2 and a patched crossterm 0.29. An inline viewport keeps history in normal scrollback. About 1,400 insta snapshot files. | Native binary; `curl` installer, npm shim, brew | Apache-2.0 |
| opencode | TypeScript, OpenTUI with the Solid reconciler | Bun-compiled binary (144 MB); `curl`, npm, brew, scoop, AUR, nix | MIT |
| Crush | Go, Bubble Tea v2.1.0, Bubbles v2.2.1, Lip Gloss v2.0.6, Ultraviolet, Fang v2 | Go binary (90 MB, many deps); brew tap, npm, winget, scoop, AUR, nix | FSL-1.1-MIT (not OSI) |
| Gemini CLI | TypeScript and React, on an Ink fork (`@jrichman/ink`) | npm bundle, needs Node 20 or later | Apache-2.0 |
| Herdr | Rust, Ratatui 0.30, crossterm 0.29, portable-pty, and a vendored libghostty-vt | `curl` installer, brew, mise | Apache-2.0 |

Sources: each project's `go.mod`, `Cargo.toml` or `package.json` at HEAD on 2026-10-15, release pages, and the vendor posts named above.

Two lessons stand out:

1. The polished TUIs all own their renderer. Each one diffs cells and writes only changes, inside synchronised output (mode 2026). Claude Code, Amp and Codex all had to fight flicker. Bubble Tea v2 and Ratatui give us this out of the box.
2. Codex and Claude Code both keep the conversation in the terminal's own scrollback, on the main screen. This is the "native scrollback and Cmd+F" property the owner wants. simstim must not break it.

### 1.2 Framework status (October 2026)

- **Go: Bubble Tea v2.**
  - v2.0.0 went stable on 2026-02-24, after alphas from 2024-09 and betas through 2025. v2.1.0 shipped on 2026-10-08 and adds OSC 7501 "Program Status" reports.
  - The import path is `charm.land/bubbletea/v2`. It needs Go 1.26.
  - Lip Gloss v2 is at v2.0.6 (2026-08-11). Bubbles v2 is at v2.2.1 (2026-08-24).
  - The renderer comes from Ultraviolet (`github.com/charmbracelet/ultraviolet`). It is a cell-based diffing renderer "inspired by ncurses", and it has no tags, so you pin a pseudo-version.
  - `charmbracelet/x/vt` v0.1.0 (2026-10-12) is a Go terminal emulator, useful for tests.
- **Rust: Ratatui and crossterm.**
  - Ratatui 0.30.0 came out on 2025-12-26 and split into modular crates. The latest is 0.30.2 (2026-06-19).
  - crossterm 0.29.0 (2025-04-05) is still the latest. Codex patches it in a fork.
  - portable-pty 0.9.0 and termwiz have not had a release since early 2025.
  - The common text area moved to `ratatui-textarea` 0.9.3.
- **TypeScript: OpenTUI or Ink.**
  - OpenTUI (`anomalyco/opentui`, formerly `sst/opentui`) is at v0.6.6. It is still before 1.0 and releases several times a day.
  - The core is Zig 0.16, with a TypeScript API and React and Solid reconcilers. It runs on Bun 1.3.14 or later, or on Node 26.4 or later with `--experimental-ffi`.
  - Ink v8.0.0 shipped on 2026-10-03, with React 19.3 and incremental rendering.
- **Zig: libvaxis.**
  - The last tag is v0.5.1 (2024-11-12). Main declares 0.6.0 and needs Zig 0.17.0, which was released 2026-10-01. So in practice you track main.
  - Its feature list is the best of the four for terminal fidelity. It has no terminfo; it queries the terminal. It supports the kitty keyboard and graphics protocols, OSC 8, 52, 22, 9, 777 and 7501, and modes 2026, 2027, 2031 and 2048.
  - It includes vxfw, a Flutter-like widget layer, and a terminal widget.

### 1.3 Measurements on this Mac (arm64)

We built a minimal program in each stack. It imports the TUI library and an input widget and exits on `--version`. The time is the median of 30 runs, including process spawn.

| Stack | Binary | Startup |
| --- | --- | --- |
| Go 1.26, Bubble Tea v2.1.0, Lip Gloss v2.0.6, Bubbles textarea, `CGO_ENABLED=0 -trimpath -s -w` | 3.9 MB (darwin/arm64); 4.1 MB static ELF for linux/amd64 from the same Mac | 4.4 ms |
| Rust 1.98, Ratatui 0.30.2, crossterm 0.29, LTO, `opt-level=z`, stripped | 0.44 MB | 3.5 ms |
| Node 24, Ink 8.0.0, import only | needs Node (23 MB of `node_modules`) | 234 ms (bare `node -e 0` takes 41 ms) |

Installed agent CLIs, for scale (median of 7 runs):

| Binary | Size | `--help` |
| --- | --- | --- |
| claude 2.1.300 (Bun) | 250 MB | 94 ms |
| amp (Bun) | 110 MB | 152 ms |
| codex 0.162.1 (Rust) | 247 MB | 13 ms |

A real simstim binary will add SSH, SFTP and a TOML parser. Expect about 10–15 MB in Go. That is our estimate; we did not build it. Startup stays under 10 ms in Go or Rust. A Bun build would be 60 MB or more and take 50–150 ms to start.

### 1.4 Comparison

| Criterion | Go + Bubble Tea v2 | Rust + Ratatui | TS + OpenTUI / Ink | Zig + libvaxis |
| --- | --- | --- | --- | --- |
| Amp-style polish | Strong. Crush proves it. Diffing cell renderer, sync output on by default, 60 fps cap (`WithFPS`), declarative `tea.View` (cursor, mouse, focus, title, colours), and `BackgroundColorMsg.IsDark()` with `lipgloss.LightDark`. | Strong. Codex proves it. You build more yourself: no built-in event loop or component model. | Strongest for rich UI (React or Solid, Yoga layout). opencode, Claude Code and Gemini CLI prove it. | Strong primitives (vxfw). Fewer ready-made widgets. |
| Rich input editor | `bubbles/textarea` and `textinput` are enough for wizards and filters. A chat-grade editor would be custom work. | `ratatui-textarea` | Best ecosystem | `TextField` in vxfw |
| Light and dark | `RequestBackgroundColor`, `lipgloss.HasDarkBackground`. Ultraviolet decodes mode 2031 reports, but Bubble Tea does not expose them as messages yet. | Manual (OSC 11 query) | Manual or library | Built in (2031) |
| Single static binary | Yes, `CGO_ENABLED=0`. Cross-compiles for every target from one machine. | Yes, but musl or zigbuild is needed for static Linux. | Bun `--compile`, 60–150 MB | Yes, tiny, and cross-compiles well |
| Startup | ~4 ms | ~3 ms | 50–250 ms | ~1–3 ms (expected, not measured) |
| PTY and raw mode | `x/term`, `creack/pty` v1.1.24 (2024; works, slow releases), `charmbracelet/x/xpty` (adds Windows ConPTY) | crossterm raw mode; portable-pty is stale | node-pty is a native addon and awkward in a single binary. Bun's PTY support is **UNVERIFIED**. | Own PTY code |
| SSH | `golang.org/x/crypto/ssh` (client, agent, known_hosts), `pkg/sftp` v1.13.11 (2026-07), `kevinburke/ssh_config` | `russh` v0.64.1 (2026-10), or the system `ssh` through the `openssh` crate | `ssh2` (Node) | None. You shell out to `ssh`. |
| Terminal parsing | `x/ansi` parser, Ultraviolet `EventDecoder.Decode(buf) (n, Event)`, `x/vt` emulator | `vte`, `vt100`, termwiz | xterm.js headless | libghostty-vt |
| Testing | teatest v2 (experimental, pseudo-version only), `x/exp/golden`, `x/vt`, VHS v0.12.1 | `TestBackend` and insta, the most mature snapshot story | ink-testing-library; `@microsoft/tui-test` | Thin |
| Contributors | Large pool. Go is quick to learn. The Charm community is active. | Large pool, steeper curve | Largest pool. Bun lock-in. | Small pool. Zig still breaks APIs between versions. |
| Shipping the host helper | `GOOS=linux GOARCH=arm64` from any machine | Needs a cross toolchain | Bun cross-compiles, but binaries are huge | Excellent |

### 1.5 Testing approach for a Go client

- **Screens:** Use teatest v2 to drive the model and compare the final output with golden files. Pin its pseudo-version, because it is experimental.
- **Attach mode:** Feed the output byte stream and the detach reset into `x/vt` (a Go emulator). Assert that every mode is back to its baseline. Then do the same against libghostty-vt in CI through zmx's own tests or a small C harness. That second part is **UNVERIFIED** as an easy setup.
- **Input filter:** Use table tests and a property test that splits streams at random. Fuzz it with Go's native fuzzing, using Ultraviolet's decoder as an oracle.
- **End to end:** Use VHS tapes for the setup wizard and the session list. VHS can write `.txt` golden output as well as GIFs for the README.

### 1.6 Verdict on the framework

Bubble Tea v2 is the best fit. It is now stable, and its v2 renderer was built for exactly the problems Amp and Claude Code fought: diffing, sync output, kitty keys and cursor control. Crush shows the visual ceiling.

The deciding factor for simstim is not the screens. It is the attach pump, the SSH plumbing and shipping a host helper to any Linux VM. Go is the strongest there:

- `x/crypto/ssh` and `pkg/sftp`
- `x/ansi` and `x/vt` for parsing and testing
- one-command cross-compiles to static binaries

Rust would also work well. It would cost more contributor time and need more build tooling. TypeScript would make a 60 MB or larger binary with slow startup, for a tool that should feel instant. Zig is attractive for fidelity, but the pool is too small and the toolchain too unsettled for an open-source project that wants contributors.

## 2. Attach mode

### 2.1 The hand-off, step by step

Bubble Tea v2 already has the hook. `tea.Exec(cmd ExecCommand, fn)` calls `releaseTerminal`, runs `cmd.Run()` with the program's input and output, then calls `RestoreTerminal` and repaints (`exec.go`, `tea.go#L1363-L1420` in v2.1.0). simstim implements `ExecCommand` with a `Run()` that is the attach pump.

What Bubble Tea's renderer does on release (`cursed_renderer.go`, `close()`, v2.1.0), in order:

1. Writes `CSI > 4 m` (reset modifyOtherKeys) and `CSI < 1 u` (pops its own kitty entry). It pops on each switch between the main and alternate screens, because each screen has its own stack.
2. Moves the cursor to the bottom, then leaves the alternate screen (`CSI ? 1049 l`).
3. Shows the cursor, and turns off bracketed paste, focus and mouse modes 1002, 1003 and 1006.
4. Clears the title, resets the cursor shape (`CSI 0 SP q`), and resets cursor, foreground and background colours (OSC 112, 110, 111).
5. Resets the OSC 9;4 progress bar and mode 2027.
6. Restores termios.

`RestoreTerminal` pushes a fresh kitty entry, re-enables modifyOtherKeys level 2, re-enters the alternate screen and repaints. It also checks the size, because SIGWINCH may have been missed.

So the attach pump starts with a clean, cooked terminal on the **main screen**. That is what we want: the remote app's output lands on the main screen and flows into Ghostty's own scrollback.

The attach `Run()`:

1. **Cancellable input.** Open a cancellable reader on stdin. Use Ultraviolet's `NewCancelReader`, or `muesli/cancelreader`. A stray goroutine must not keep reading stdin after detach, or it will steal keys from the TUI.
2. **Baseline.** Optionally query the baseline before raw pass-through: `CSI ? u` (kitty flags) and DECRQM (`CSI ? Pm $ p`) for modes whose default differs between terminals (7, 12, 1007, 2027). Use a 100 ms timeout. This lets detach restore rather than guess.
3. **Raw mode.** Put the tty in raw mode with `x/term.MakeRaw`.
4. **Push an entry.** Push one kitty entry with all flags off (`CSI > 0 u`). This gives the remote's replayed "set" (`CSI = flags ; 1 u`, see 2.3) an entry of ours to modify, on every terminal.
5. **Start the session channel** (see 2.6 on transport), and send the window size first.
6. **Run three loops:**
   - output: remote → passive mode tracker → stdout
   - input: stdin → filter → remote
   - SIGWINCH → window change
7. **On detach:**
   1. Stop forwarding output at a sequence boundary.
   2. Close the channel.
   3. Write the reset list (2.2).
   4. Restore termios.
   5. Return. Bubble Tea then repaints the TUI.

**Passive output tracker.** This is a small VT parser, such as `x/ansi`'s parser. It reads a copy of the output after the bytes have been written, and never changes them. It records:

- which DEC private modes the remote set or reset
- kitty push, pop and set per screen
- modifyOtherKeys
- alternate-screen state
- whether the stream is in the middle of an escape sequence or inside an OSC, DCS or APC string

It costs microseconds per read and adds no latency, because the write happens first.

### 2.2 What to reset on detach, in order

The remote app talks straight to Ghostty, so anything it turned on stays on after we leave. OpenSSH shows the failure mode. On exit it restores termios only and writes no escape sequences (`sshtty.c#L57-L66`). That is why a dropped `ssh` session often leaves mouse reporting on.

The list below merges what tmux, Zellij, Herdr and Bubble Tea do, and fills their gaps. tmux never turns off titles or OSC 10/11. Zellij leaves bracketed paste on. zmx uses RIS. The "Seen in" column says which tools do the same thing.

| # | What | Bytes to write | Why and notes | Seen in |
| --- | --- | --- | --- | --- |
| 0 | Leave any half-sent sequence | If the tracker is not in the ground state: `CAN` (`0x18`), then `ESC \` if inside a string | Otherwise Ghostty's parser swallows our reset as part of an unfinished OSC or APC (for example a kitty image chunk). CAN returns DEC-style parsers to ground from any state (Ghostty `parse_table.zig#L62-L63`). | – |
| 1 | Synchronised output | `CSI ? 2026 l` | If we cut between BSU and ESU, Ghostty freezes rendering until its 1000 ms safety timeout (`termio/Thread.zig#L38`). | zmx excludes 2026 from replay |
| 2 | Kitty flags, alternate screen | if on alt: `CSI = 0 ; 1 u` then `CSI < 99 u` | Each screen has its own stack (spec; Ghostty `Screen.zig#L75`). Set to 0, then pop past the bottom. Ghostty's stack is a ring of 8, and `pop(n ≥ 8)` clears it. The kitty spec says a pop that empties the stack resets the flags. | Bubble Tea pops per screen; Zellij `CSI < 1 u` |
| 3 | Alternate screen | `CSI ? 1049 l`, only if the tracker says alt is active | On the primary screen, `1049 l` restores a stale saved cursor. | tmux `rmcup`, Zellij, Bubble Tea |
| 4 | Kitty flags, main screen | `CSI = 0 ; 1 u` then `CSI < 99 u`. If the baseline query showed non-zero flags, push them back. | Drops the remote's entries and our own. | Herdr owns exactly one entry |
| 5 | modifyOtherKeys | `CSI > 4 ; 0 m` | Ghostty supports level 2 only, and resets on `>4m` or `>4;0m` (`stream.zig#L2096-L2139`). | tmux `>4m`, Herdr `>4;0m`, Bubble Tea `>4m` |
| 6 | Mouse | `CSI ? 9 l`, `? 1000 l`, `? 1001 l`, `? 1002 l`, `? 1003 l`, `? 1005 l`, `? 1006 l`, `? 1015 l`, `? 1016 l` | Every tracking and encoding mode. 1016 is SGR pixels. | Herdr (full list); tmux and Zellij a subset |
| 7 | Focus events | `CSI ? 1004 l` | – | tmux, Zellij, Herdr |
| 8 | Bracketed paste | `CSI ? 2004 l` | Bubble Tea turns it back on for itself. | tmux, Herdr, Bubble Tea; **not** Zellij |
| 9 | Colour-scheme reports and in-band resize | `CSI ? 2031 l`, `CSI ? 2048 l` | Ghostty sends unsolicited `CSI ? 997 ; n n` and `CSI 48 ; … t` while these are on. | tmux and Zellij 2031 |
| 10 | Modes with terminal-specific defaults | restore from the baseline query: 2027, 7 (wrap), 12 (blink), 1007 (alternate scroll) | Do not guess defaults. Ghostty and xterm differ. | Bubble Tea 2027 |
| 11 | Soft reset | `CSI ! p` (DECSTR) | In Ghostty this resets cursor visibility, insert and origin modes, wrap, cursor and keypad keys, margins, SGR, charsets, the saved cursor, modifyOtherKeys, cursor style and the palette. It **does not** clear the screen or the scrollback (`Terminal.zig`, `softReset`). It is VT220-standard. | – |
| 12 | Cursor | `CSI ? 25 h`, `CSI 0 SP q`, `OSC 112 ST` | Belt and braces after DECSTR. 0 means "the user's configured default". | tmux `Se` and `Cr`; Herdr `CSI 0 SP q` |
| 13 | Dynamic colours | `OSC 110 ST`, `OSC 111 ST`, `OSC 104 ST` | Foreground, background and palette. Optionally also `OSC 117`/`OSC 119` for highlight colours; Ghostty support is **UNVERIFIED**. | zmx, Bubble Tea; **not** tmux |
| 14 | Pointer shape (OSC 22) | `OSC 22 ; ST` (empty value) | Ghostty treats an empty value as a reset (`stream.zig`, `mouse_shape_reset`). | – |
| 15 | Hyperlink | `OSC 8 ; ; ST` | Closes any open link. | – |
| 16 | Progress and status | `OSC 9 ; 4 ; 0 ST`; OSC 7501 `clear` if the remote used it | Stops a stale progress bar in the tab. The exact OSC 7501 clear syntax is spec v0.4, so pin it. | Bubble Tea progress bar |
| 17 | Title | `OSC 2 ; simstim ST` (Bubble Tea's view sets it again) | Do **not** pop the title stack (`CSI 23 t`). The remote may have pushed without popping, so the stack depth is unknown. | – |
| 18 | SGR and cursor position | `CSI 0 m`, then move to the last row and print a newline | Leaves the session's last frame visible in scrollback, below everything else. | tmux, Zellij |

**Never use RIS (`ESC c`)** as a shortcut. In Ghostty, RIS calls `fullReset`, then `Screen.reset`, then `PageList.reset`, which "discards all scrollback" (`PageList.zig#L984-L988`). That destroys the native scrollback the owner asked for. zmx's own client does exactly this on detach (`src/main.zig#L1642-L1651`, v0.8.1): `ESC c` followed by OSC 110, 111 and 112.

**XTSAVE and XTRESTORE** (`CSI ? Pm s`, `CSI ? Pm r`) look tempting. But the remote app can use the same single save slot per mode and overwrite ours. Do not rely on them.

Write the whole reset list in **one** `write()`, so that it lands as one chunk and Ghostty renders it once.

### 2.3 How other tools handle this

- **zmx** (Zig, MIT, v0.8.1, 2026-09-05; pins libghostty-vt 1.3.2-dev)
  - **Client setup:** The client sets raw termios and turns off `VQUIT` and `VLNEXT`, then writes `CSI 2 J CSI H` on connect.
  - **Detach:** RIS plus OSC 110, 111 and 112, as above.
  - **Detach key:** fixed at `Ctrl+\`. Custom keys were declined in issue #179. Detection takes legacy `0x1C` only as the **first byte of a read**, or a CSI u `92;…u`, or modifyOtherKeys `27;…;92~`. It masks the lock bits and ignores releases. The whole read chunk is dropped on detach.
  - **Layout bug:** issue #197 shows a British-PC layout in Ghostty sending `CSI 96 ; 5 u` for the physical `\` key. Layout-dependent detach keys break.
  - **On reattach** it replays state from libghostty-vt:
    - up to 10,000 lines of scrollback as plain styled text, so the local scrollback fills again
    - then a clear, and the visible screen
    - every DEC mode that differs from its default
    - the kitty flags as a **set** (`CSI = flags ; 1 u`) on the active screen only
    - modifyOtherKeys
    - OSC 10, 11 and 12, and the title
  - **Not replayed:** cursor shape (DECSCUSR) and mode 2026.
  - **Many clients:** the leader is the last client that typed. Only the leader resizes the PTY.
  - **While detached:** the daemon answers DA1 and DA2 itself, and injects focus in/out if 1004 is on.
  - **SSH:** a first-class flow (`ssh -t host zmx attach x`).
- **tmux** (3.8)
  - tmux is a full emulator, so it never passes bytes through. On detach, `tty_stop_tty` (`tty.c#L462-L531`) writes:
    - scroll region, `rmacs`, `sgr0`, `rmkx`
    - cursor style and colour resets, `cnorm`
    - mouse off (1000, 1002, 1003, 1005, 1006)
    - `?2004l`, `?7727l`, `?1004l`, `>4m`, `?69l`
    - `rmcup`, `?2031l`
  - It uses modifyOtherKeys on the outer terminal, and never kitty flags.
  - Suspend: the server calls `tty_stop_tty`, then the client sends itself SIGTSTP. On SIGCONT the client sends `MSG_WAKEUP`, and the server calls `tty_start_tty` and redraws.
  - Resize: the client sends `MSG_RESIZE`, and the server reads `TIOCGWINSZ` on the client tty.
- **Zellij** (0.45.1)
  - On start it writes `?1049h`, a "clear attributes" string (`?1l`, `ESC =`, `CSI r`, mouse off, `?12l`), `>1u`, `?2031h` and `?2004h`.
  - On exit it writes mouse off, `<1u`, `?2031l`, `?1004l`, `?2027l`, `?1049l`, `CSI m` and `?25h`. It leaves bracketed paste on.
- **Herdr** (Rust, Apache-2.0, v0.9.5)
  - Herdr is the closest design to simstim. Its "direct attach" streams the rendered terminal state, then live frames.
  - It mirrors the pane's kitty flags and modifyOtherKeys onto the host, and **owns exactly one kitty stack entry** (`<1u` then `>{n}u`).
  - It matches its prefix key as raw `0x02`, kitty `CSI 98;5u` (releases ignored) and modifyOtherKeys `CSI 27;5;98~`, and it leaves bracketed paste contents alone (`src/client/attach.rs#L57-L140`).
  - Its restore list is `>4;0m`, a kitty pop, `?7h`, `?1004l`, `?2004l`, the full mouse-off list, `?2031l`, `?25h` and `CSI 0 SP q`.
- **OpenSSH** (10.6)
  - On exit it restores termios only, with no escape sequences.
  - The `~` escape fires only after `\r` or `\n`. With kitty flag 8, Enter becomes `CSI 13 u`, so `~.` stops working. With flag 1, Ctrl+Z is `CSI 122;5u`, so `~^Z` stops working.
  - Window size is read with `TIOCGWINSZ` on **stdin** and sent as a `window-change` request with columns, rows and pixels (`channels.c#L5051-L5072`).

**What this means for simstim:**

- Use zmx (or a zmx-like daemon) on the host for state and replay.
- Never let zmx's client write its detach bytes to Ghostty. simstim detaches by closing the SSH channel. The remote `zmx attach` gets SIGHUP and its daemon keeps running. Even if zmx writes RIS, simstim has already stopped forwarding.
- The detach key must be simstim's own, matched on the local side.

### 2.4 Resize (SIGWINCH)

1. simstim catches SIGWINCH and reads `TIOCGWINSZ` on the **tty** (stdout or `/dev/tty`), including pixel sizes.
2. It sends the new size at once. Coalesce within one frame (about 16 ms) to avoid a storm while the user drags the window.
3. The size goes to the remote over the transport:
   - With `x/crypto/ssh`: `session.WindowChange(rows, cols)`. This API sends a fake pixel size of `cols × 8` by `rows × 8`, and `RequestPty` does the same (`ssh/session.go#L190-L252` at `cf169cd`). Apps that size images from pixel dimensions get wrong numbers. To send real pixels, you would open the session channel yourself and marshal your own `window-change` request.
   - With the system `ssh` running on a local PTY (see 2.6): `TIOCSWINSZ` on the PTY master. The kernel sends SIGWINCH to `ssh`, and `ssh` sends `window-change`.
4. On the host, sshd resizes the PTY. The `zmx attach` client gets SIGWINCH and tells the daemon. If this client is the leader, the daemon resizes the session PTY and the app redraws.
5. On attach, send the size **before** asking for the replay, so the replay is rendered at the right width. zmx sends SIGWINCH to the app on every reattach, even with no size change, which forces a fresh frame.
6. Mode 2048 (in-band resize) reports from Ghostty are ordinary input. Forward them unchanged.

### 2.5 Suspend (Ctrl+Z) and other signals

- **In attach mode**, the tty is raw, so Ctrl+Z is not a signal. It goes to the remote as `0x1A`, or as `CSI 122;5u` under kitty flags. The remote app or shell decides what to do. This matches local behaviour. Claude Code and shells suspend on the host.
- **SIGTSTP from outside** while attached (`kill -TSTP`): follow tmux.
  1. Write the reset list, restore termios, set SIGTSTP to the default and re-raise it.
  2. On SIGCONT, go back to raw mode and push our kitty entry.
  3. Re-attach to get a replay, because the screen may have changed meanwhile. A fresh attach is simpler and more robust than trying to redraw in place.
- **In TUI mode**, Bubble Tea's `tea.Suspend` and `ResumeMsg` handle this already.
- **SIGHUP, SIGTERM and a lost connection:** always run the reset list, then return to the TUI with a "connection lost, reconnecting" state. The session lives on the host, so reconnecting is cheap. Laptops changing WiFi will hit this often.
- **SIGINT:** it cannot happen in raw mode; Ctrl+C goes to the remote.

### 2.6 Transport: system `ssh` or in-process SSH

| | System OpenSSH | `x/crypto/ssh` in-process |
| --- | --- | --- |
| Works with the user's `~/.ssh/config` (ProxyJump, `Include`, `Match`), FIDO keys, 1Password or Secretive agents, certificates, GSSAPI, Tailscale SSH | Yes, all of it | Partly. Needs `ssh_config` parsing, and many directives are missing. |
| Fast attach | ControlMaster with our own ControlPath gives roughly one round trip per attach | Keep one connection per host inside a background simstim process |
| Window size | Needs a local PTY between simstim and `ssh` (`ssh` reads the size from stdin). Real pixel sizes pass through. | `WindowChange` call, but with fake pixel sizes (see 2.4) |
| Uploads | `ssh host 'cat > path'` over the master, or `sftp -b` | `pkg/sftp` on the same connection |
| Escape char | Must pass `-e none`, or the `~` handling eats keys | None |
| Windows | Built-in OpenSSH | Same code |

**Recommendation:** use the system `ssh` as the default transport. Run it with `-e none`, `-o ControlMaster=auto`, a simstim-owned `ControlPath`, `ControlPersist`, and `ServerAliveInterval` for fast loss detection. Run it on a local PTY from `creack/pty` or `x/xpty`.

Keep `x/crypto/ssh` behind an interface for later. Users bring complex SSH setups, and "it works like `ssh`" is the fastest way to support any provider. The extra PTY hop costs microseconds. Note that zmx itself documents `ssh -t host zmx attach` as its remote flow, so this path is well trodden.

**Scrollback duplication on reattach.** zmx replays up to 10,000 lines of history on each attach. Attaching to the same session three times puts three copies into Ghostty's scrollback.

The options:

1. Clear the scrollback on attach (`CSI 3 J`). This loses the user's earlier shell history in that tab.
2. Ask zmx for less replay. Its `history --scrollback N` exists for `zmx history`; whether `attach` takes a replay limit is **UNVERIFIED**.
3. Accept duplicates.

We suggest option 1 as a setting, default on, with a clear note in the docs. This is an open question for the owner.

## 3. Input interception in pass-through

### 3.1 What we must recognise

The detach key defaults to **`Ctrl+\`**, as in zmx, dtach and abduco. It must be configurable. Users must also be able to send a literal `Ctrl+\` by pressing it twice.

| Encoding | Ctrl+V | `Ctrl+\` (detach) | When |
| --- | --- | --- | --- |
| Legacy C0 | `0x16` | `0x1C` | No keyboard protocol on |
| Kitty, flag 1 (disambiguate) | `CSI 118 ; 5 u` | `CSI 92 ; 5 u` | Claude Code, Codex, Crush and Herdr turn this on |
| + caps lock / num lock | `CSI 118 ; 69 u` / `CSI 118 ; 133 u` | `… 69 u` / `… 133 u` | Modifiers are 1 + bits; caps lock is 64 and num lock is 128 |
| + flag 2 (event types) | press `CSI 118 ; 5 u` (Ghostty omits `:1`), repeat `CSI 118 ; 5 : 2 u`, release `CSI 118 ; 5 : 3 u` | same pattern | |
| + flag 4 (alternate keys), non-Latin layout | `CSI 1084 :: 118 ; 5 u` (Russian `м`, base key `v`) | base key 92 | The exact Ghostty bytes for this case are **UNVERIFIED**; this follows the spec |
| + flag 8 (all keys as escapes) | same as flag 1 | same | Enter and Tab also become CSI u |
| + flag 16 (text) | optional third field; Ctrl suppresses text in Ghostty | same | |
| xterm modifyOtherKeys 2 | `CSI 27 ; 5 ; 118 ~` | `CSI 27 ; 5 ; 92 ~` | Ghostty supports level 2 only; kitty flags win if both are on |
| Inside bracketed paste | literal | literal | **Never intercept** between `CSI 200 ~` and `CSI 201 ~` |

Sources: the kitty keyboard protocol spec (`docs/keyboard-protocol.rst`), Ghostty `src/input/key_encode.zig` and `src/terminal/kitty/key.zig`, and zmx `src/util.zig#L521-L525` with its tests.

### 3.2 The filter

The goal is "never break other keys, add no latency". The design:

1. **Stream in, stream out.** For each read, scan the bytes once. Forward everything that is not a match, unchanged and in order, in the same call. There is no per-key timer.
2. **Legacy bytes.** `0x1C` and `0x16` are matched anywhere in the chunk, not just at its start as zmx does.
3. **CSI.** On `ESC [`, find the end of the sequence (a final byte from `0x40` to `0x7E`). Classify only two shapes:
   - `CSI digits[:digits[:digits]] ; mods[:event] [; text] u`
   - `CSI 27 ; mods ; code ~`
   Compute `(mods − 1) &^ (64 | 128)` and require exactly Ctrl (4). Match the key code **or** the base-layout key. Anything else is forwarded unchanged. That includes mouse reports, focus events, OSC 11 replies, `CSI ? flags u` replies, mode 2048 reports and Shift+Enter (`CSI 13 ; 2 u`).
4. **Event types.** A press (no event field, or `:1`) triggers the action. Repeats and releases of a key we took are dropped, so the remote never sees a release without a press. A release of a key we did not take is forwarded.
5. **Bracketed paste.** On `CSI 200 ~`, buffer until `CSI 201 ~`, then hand the whole paste to the drop-rewrite step (section 4.3). Pastes arrive in one burst from the terminal, so buffering adds no visible delay. Cap the buffer (for example at 1 MB), and above the cap forward unchanged.
6. **Split reads.** If a chunk ends in the middle of a CSI (`ESC [ 1 1`), hold only that tail, and flush it unchanged after about 10 ms if nothing follows. Real terminals write one key per `write()`, so a real key press never waits. A lone `ESC` at the end of a chunk is forwarded at once. Under kitty flags, Esc is `CSI 27 u` anyway. The worst case of a rare split is a missed detection, never a changed byte.
7. **OSC, DCS and APC replies** from the terminal pass through untouched. They contain no `0x16` or `0x1C`.
8. **Ctrl+V with no image.** When Ctrl+V matches, ask the clipboard whether it holds an image. With NSPasteboard through `purego` this took 12–23 µs once warm in our test, and about 6 ms on the first call (see 4.1). If it holds no image, forward the **original bytes** exactly as received, so the remote app sees its own encoding.

**Prototype.** We wrote this filter in Go (about 170 lines of code, kept in the scratchpad, not in the repo). The tests pass:

- every encoding in the table above
- non-matches forwarded byte for byte: Ctrl+Shift+V, Alt+V, Shift+Enter, Esc, arrows, SGR mouse, focus, an OSC 11 reply and a kitty flags reply
- releases swallowed only after our own press
- paste contents never scanned
- a property test: 2,000 random streams of mixed keys and replies, split at random points, come out byte-identical

We suggest building the real filter the same way and adding Go fuzzing with Ultraviolet's `EventDecoder.Decode(buf) (n int, Event)` as an oracle. Do not use Ultraviolet directly as the filter. It turns a lone `ESC` into an Escape key and decodes far more than we need. The rule "forward `buf[:n]` unchanged" would still keep it safe, but the small filter is easier to reason about.

### 3.3 How to paste the remote path

After the upload, simstim types the remote path into the session:

- **The remote app turned on mode 2004** (the tracker knows): send `CSI 200 ~` + path + `CSI 201 ~`. Claude Code and Codex treat a pasted image path as an image attachment (see section 4.2).
- **Otherwise:** send the path as plain text, shell-quoted, with a trailing space.

## 4. Clipboard images and dropped files

### 4.1 Reading an image from the local clipboard

**Use `golang.design/x/clipboard`** (MIT, v0.11.0, 2026-09-26). Since v0.8.0 (2026-06-07) it needs **no cgo** on any desktop platform:

- macOS uses `purego/objc` and NSPasteboard.
- Linux uses pure-Go X11, with native Wayland data-control and an XWayland fallback.
- Windows reads `CF_DIBV5`, falling back to `CF_DIB`.
- `Read(FmtImage)` always returns PNG, and converts TIFF itself.

So the release build stays `CGO_ENABLED=0` and cross-compiles from one Linux runner.

| OS | Primary | Fallback | Notes |
| --- | --- | --- | --- |
| macOS | NSPasteboard through `purego`: `public.png`, then `public.tiff` converted to PNG | `osascript -e 'the clipboard as «class PNGf»'` | We tested `purego` NSPasteboard reads on this Mac with `CGO_ENABLED=0` and purego v0.11.1. The first call took about 6 ms; later calls took 12–23 µs. `osascript -e 'clipboard info'` took 164 ms. Do **not** rely on `«class PNGf»` alone: Claude Code issue #30934 shows it fails for copies from Chromium apps, which put only `public.png` on the pasteboard. `pngpaste` is a third-party Homebrew tool, unchanged since 2023; do not depend on it. |
| Linux, Wayland | data-control protocol (library) | `wl-paste --list-types`, then `wl-paste --type image/png` | Data-control needs compositor support. Whether GNOME supports it is **UNVERIFIED**, so keep the `wl-paste` fallback. |
| Linux, X11 | X11 selections (library) | `xclip -selection clipboard -t TARGETS -o`, then `-t image/png -o` | `xsel` cannot read images. Claude Code issue #46249: paste fails silently when `xclip` is missing, so show a clear error that names the package. |
| Windows | Win32 `OpenClipboard`/`GetClipboardData` (`CF_DIBV5`, `CF_DIB`; `"PNG"` is a registered format by convention only) | PowerShell `Get-Clipboard -Format Image` (slow) | **Windows Terminal binds Ctrl+V to its own paste**, so the key never reaches us. Default the image key to **Alt+V** on Windows, as Claude Code does. What Windows Terminal sends on Ctrl+V when the clipboard holds only an image is **UNVERIFIED**. |

Other points:

- **Warm-up.** Warm the pasteboard up when attach starts, so the first Ctrl+V costs microseconds, not 6 ms.
- **Image types.** Which UTI types a macOS screenshot writes is **UNVERIFIED**. Reports say `public.png` and `public.tiff`, so reading both covers it.
- **Copied files.** Files copied in Finder are a file list, not image data. Codex reads them first, through arboard's `file_list()`. simstim should treat a copied file list like a drop (4.3).

### 4.2 What the agents do with a pasted path

Every major agent turns a pasted image path into an attachment. So "upload, then paste the remote path" works without any agent-specific code.

| Agent | Image key | Path handling |
| --- | --- | --- |
| Claude Code | Ctrl+V (Alt+V on Windows and WSL) | Inside a bracketed paste, it strips quotes and undoes backslash escapes. If the result ends in `.png`, `.jpg`, `.jpeg`, `.gif` or `.webp` and exists, it attaches the file. v2.1.300 handles one path per line. Docs: code.claude.com/docs/en/interactive-mode. |
| Codex | Ctrl+V, Ctrl+Alt+V | `normalize_pasted_path` takes `file://` URLs and one shell-escaped token, and `handle_paste_image_path` attaches it (`codex-rs/tui/src/clipboard_paste.rs`, `bottom_pane/chat_composer`). More than one token is not treated as a path. |
| Gemini CLI | Ctrl+V, Cmd+V, Alt+V | Pasted or dropped paths become `@path` references (`clipboardUtils.ts`). |
| opencode | Ctrl+V | Local paths become attachments: images and PDF, up to 20 MiB and 32 paths (`prompt/local-attachment.ts`). |
| Amp | Ctrl+V | Drag and drop of image files works (ampcode.com/docs/cli). Amp's own Orbs terminal already does what simstim plans: paste an image, upload it, insert the absolute path (ampcode.com/docs/orbs). |

So:

- **Send one path per paste event, wrapped in bracketed paste.** Codex accepts only a single token.
- **Keep the original extension.** Claude Code matches on it.
- **Shell-escape the path** the way Ghostty does.

Upload target: `~/.cache/simstim/uploads/<session>/<sha256-prefix>-<name>.png` on the host. Use the content hash so a repeated paste does not upload again, and let the host side prune old files after a few days.

Upload over the existing ControlMaster connection, for example `ssh host 'mkdir -p … && cat > …'`. This adds no new authentication round trip.

**Prior art.** iTerm2 already offers "Upload and Paste Path(s)" and "Save to Temp File and Paste Path" for remote sessions (`iTermNonTextPasteHelper.swift`). Small tools such as cc-clip use an `xclip` shim and an SSH `RemoteForward`. simstim's approach is cleaner: it needs nothing on the host's display stack.

### 4.3 Dropped files: rewrite local paths in a paste

What terminals insert when a file is dropped:

| Terminal | Inserted text | Bracketed? |
| --- | --- | --- |
| Ghostty 1.3.1, macOS | Each path with a backslash before spaces, tabs and shell metacharacters (`Ghostty.Shell.escape`; full list below), joined with spaces | Yes, if 2004 is on (through `insertText` → `completeClipboardPaste`) |
| Ghostty main (after 2026-10-13) | Same text, sent with `pasteText` and paste protection | Yes, if 2004 is on. Paste protection may now ask before a multi-line drop. |
| Ghostty GTK | Each path escaped and followed by `\n` | Yes, if 2004 is on |
| iTerm2 | Backslash-escaped, or double-quoted by preference; spaces between; trailing space | Yes |
| kitty | `file://` URIs turned into paths. Quoted only at a shell prompt; inside a TUI, raw paths separated by newlines | Yes |
| WezTerm | Only spaces escaped (`SpacesOnly`); trailing space | Yes |

Ghostty escapes these characters with a backslash: space, tab, `\`, `(`, `)`, `[`, `]`, `{`, `}`, `<`, `>`, `"`, `'`, backtick, `!`, `#`, `$`, `&`, `;`, `|`, `*` and `?` (`macos/Sources/Ghostty/Ghostty.Shell.swift#L4`).

The rewrite rule, applied to each complete bracketed paste from the filter (3.2):

1. **Tokenise** with POSIX shell rules (backslash escapes, single and double quotes). Split on spaces and newlines. Turn `file://` URIs into paths.
2. **Rewrite only if every token** is an absolute (or `~`) path to an existing local regular file, and the paste holds nothing else. Otherwise forward the paste unchanged. This keeps ordinary text pastes, including prose that mentions a path, safe.
3. **Upload** each file. Enforce a size cap (for example 50 MB, with a confirm above it) and refuse directories in v1.
4. **Rebuild the paste** with the remote paths, using the same escaping style and separators as the original, and forward it wrapped in `CSI 200 ~` / `CSI 201 ~`.
5. **Show a short notice.** Use the terminal title or an OSC 9 notification, for example "uploaded 2 files". The user needs to know the paste changed.
6. **Make it a setting** (`attach.rewrite_dropped_paths = true`).

Two limits:

- If the remote app has bracketed paste off, a drop arrives as plain typed text and cannot be told apart from typing. We do not rewrite in that case.
- Codex takes only one token per paste. A multi-file drop into Codex attaches nothing, just as it would locally.

## 5. Distribution and developer experience

### 5.1 Release tooling: GoReleaser

GoReleaser v2.18.3 (2026-10-09). The free version covers everything we need:

| Need | GoReleaser feature |
| --- | --- |
| Builds for darwin, linux and windows on amd64 and arm64 | `builds` with `CGO_ENABLED=0` |
| Homebrew | `homebrew_casks` in our own tap. The old `brews` key is deprecated since v2.10. |
| deb, rpm, apk | `nfpms` |
| Scoop, winget, Nix (NUR), AUR | `scoops`, `winget`, `nix`, `aurs` |
| SBOM | `sboms` (syft, SPDX JSON per archive) |
| Signatures | `signs` with cosign keyless (`--bundle`, `.sigstore.json`), plus `checksums` |
| macOS signing and notarisation | `notarize.macos` (anchore/quill). It runs on a **Linux** runner and signs and notarises bare binaries. The native DMG/pkg flow is Pro only and not needed. |
| Reproducible builds | `-trimpath`, `mod_timestamp: "{{ .CommitTimestamp }}"`, `-X main.date={{ .CommitDate }}`, `gomod.proxy: true`, and a `toolchain` line in `go.mod` |
| npm package | Pro only (`npms`). Not needed. |

Add GitHub's build-provenance attestation (`actions/attest-build-provenance`) in the release workflow. Users can then run `gh attestation verify`.

Note that macOS signing embeds a signature in the binary. So "reproducible" for darwin means the binary before signing. Document the check.

**cargo-dist** (now "dist", v0.33.0, 2026-09-10) would be the Rust equivalent. It shipped nothing between January and July 2025, while Astral kept a fork. It is active again, but the company's status is **UNVERIFIED**. This is one more small point for Go.

### 5.2 Install channels, in order

1. **`curl -fsSL https://<domain>/install.sh | sh`.**
   - The script is hand-written; godownloader was archived in 2021.
   - It detects OS and arch, downloads from GitHub Releases, verifies the SHA-256 (and the cosign bundle if `cosign` is present), and installs to `~/.local/bin` without sudo.
   - It prints the PATH line if needed. Keep it short enough to read in one screen.
   - curl does not set `com.apple.quarantine`, so Gatekeeper does not check this path. The Go linker ad-hoc signs arm64 binaries, so they run. Apple has no formal document stating this, but we confirmed the ad-hoc signature locally.
2. **Homebrew tap with a cask** (`brew install <org>/tap/simstim`).
   - Homebrew quarantines cask downloads. Homebrew 5.0 (2025-11-12) deprecated unsigned casks and `--no-quarantine`. Homebrew 6.0 (2026-06-11) added **tap trust**: users must trust a third-party tap once.
   - So the cask binary **must be signed with a Developer ID and notarised**. That needs an Apple Developer Program membership.
   - Move to homebrew-core once the project meets Homebrew's notability rules. Formulae there build from source.
3. **Nix:** a `flake.nix` in the repo from day one (Go builds are easy in Nix). Then nixpkgs.
4. **Later:** Scoop and winget (with Windows support), AUR, and deb/rpm through `nfpms`.
5. **Skip npm.** It is how the TypeScript agents ship. For a Go binary it adds a wrapper and Pro licensing for no gain.

Reference points: zmx ships tarballs, a tap, mise, nixpkgs and AUR. Crush ships a tap, npm, AUR, NUR, winget, scoop and apt/yum, signed and notarised through GoReleaser Pro.

### 5.3 Self-update

- **`simstim update`** uses `creativeprojects/go-selfupdate` (MIT, v1.6.0, July 2026). It checks the checksum and an ECDSA signature before it swaps the binary.
- **Do not self-update a package-managed install.** Detect it by the binary path:
  - Homebrew prefix: compare with `brew --prefix`, as `gh` does in `internal/ghcmd/cmd.go`
  - `/nix/store`
  - the Scoop or winget directories
  In those cases print the right command, for example `brew upgrade simstim`.
- **Update notice.** Check for a new version at most once a day. Show it in the TUI footer. Allow it to be turned off (`update_check = false`, `SIMSTIM_NO_UPDATE_CHECK=1`). Skip it when `DO_NOT_TRACK=1`.
- **Host helper.** Updating the client should also offer to update the host-side helper (zmx or our own), with a version handshake on connect.

### 5.4 Licence

Both MIT and Apache-2.0 are fine for a dev tool, and every dependency we would use is MIT or Apache-2.0.

- **MIT:** shortest. Used by Charm, Ghostty, zmx and opencode.
- **Apache-2.0:**
  - Adds an explicit patent grant with a retaliation clause.
  - States that contributions come in under the same licence (section 5), so no CLA is needed.
  - Requires changed files to say so.
  - Used by Codex, Gemini CLI and Herdr.

**Recommendation:** Apache-2.0 with a DCO sign-off (no CLA). simstim lives next to large companies' agent tools, and contributors from those companies get a clear patent position. Avoid source-available licences such as Crush's FSL-1.1-MIT, which is not OSI open source. The owner should confirm this choice.

### 5.5 Config, completions, scripting and telemetry

- **Config format: TOML.** It has comments, is easy to hand-edit and has good Go libraries (`pelletier/go-toml/v2` or `BurntSushi/toml`). Publish a JSON Schema so editors can complete keys (Taplo reads it).
- **Location:**
  - `$XDG_CONFIG_HOME/simstim/config.toml`, defaulting to `~/.config/simstim/` on **both** macOS and Linux. Do not use `os.UserConfigDir`: it returns `~/Library/Application Support` on macOS, where CLI users do not look. `adrg/xdg` has the same default unless overridden.
  - On Windows, `%APPDATA%\simstim`.
  - State (the host list, the session cache, logs) goes in `$XDG_STATE_HOME/simstim`, and the cache in `$XDG_CACHE_HOME/simstim`.
  - ControlMaster sockets: use a **short** path with `%C`, for example `~/.ssh/simstim-%C` or `$XDG_RUNTIME_DIR/simstim/%C`. macOS limits Unix socket paths to about 104 bytes.
- **Precedence:** flags, then `SIMSTIM_*` env vars, then the config file, then defaults (clig.dev).
- **Hosts:** store an SSH alias where one exists, so `~/.ssh/config` stays the single source of truth for connection details.
- **Command layer:** Cobra with Charm's **Fang** v2.0.1 (MIT). It gives styled help and errors in the same look as the TUI, `--version`, a `completion` command for bash, zsh, fish and PowerShell, and man pages. Add dynamic completion of host and session names from the local cache, so pressing Tab never touches the network.
- **Scripting:**
  - Every list or show command takes `--json` with a documented, versioned schema (`{"version":1,…}`), for example `simstim hosts --json` and `simstim sessions --json`.
  - When stdout is not a terminal, never start the TUI. Use stable exit codes.
  - `simstim attach <host>/<session>` goes straight to attach mode.
  - Respect `NO_COLOR`.
- **Telemetry:** none in v1. If it is ever added:
  - opt-in only, with the exact event list documented
  - a `simstim telemetry status` command
  - `DO_NOT_TRACK=1` as a hard off
  - never host names, paths, commands or session content
  Note: consoledonottrack.com is now an unrelated spam site, so do not link to it. Cite tools that honour `DO_NOT_TRACK`, such as `gh`.

## 6. Risks and open questions

1. **Scrollback duplicates on reattach.** zmx replays up to 10,000 lines each time (2.6). Owner decision: clear the scrollback on attach (default), or accept duplicates.
2. **zmx replay gaps.**
   - Kitty flags are replayed as a "set" on the active screen only.
   - The cursor shape is not replayed.
   - Its README lists a known issue: programs such as `psql` echo CSI u after a replay.
   These may need fixes upstream in zmx, or our own host helper. This belongs to the host-side note.
3. **Detach key and keyboard layouts.** zmx issue #197 shows a British-PC layout sending a different code for the `\` key in Ghostty. We match the base-layout key when flag 4 is on, but the remote app chooses the flags.
   - Mitigation: make the key configurable.
   - Document a Ghostty binding that sends the detach byte from a chord of the user's choice: `keybind = <chord>=text:\x1c`. Ghostty's `text:` action takes Zig string escapes (`src/input/Binding.zig`). Pick a chord Ghostty does not already use. For example, `cmd+shift+d` is taken: it is the default "new split down" on macOS (`src/config/Config.zig`).
4. **Ghostty drop behaviour is changing.** 1.3.1 and main differ (4.3). Test against both, and against iTerm2, kitty and WezTerm.
5. **Pinned pseudo-versions.** Ultraviolet and teatest v2 have no tags. Pin them, and expect small API breaks.
6. **Mode 2031.** Bubble Tea v2.1 does not yet surface live light/dark changes as messages, though Ultraviolet decodes them. Query once at start. Upstream a message type if we want live switching.
7. **Pixel sizes.** Only the system-`ssh` path forwards real pixel sizes (2.4).
8. **Windows.** Ctrl+V is taken by Windows Terminal. ConPTY and the raw console need their own testing. Treat Windows as a later milestone.
9. **Notifications while detached.** OSC 9/777 and OSC 7501 (Program Status, spec v0.4, 2026-10-10; Bubble Tea v2.1 and Crush emit it) pass straight through while attached. While detached they are lost, unless the host side records them. The host-side note should decide whether to record them, so the session list can show "working", "blocked" or "done" for each session.
10. **Apple Developer account.** Notarisation needs a paid Apple Developer Program membership and a Developer ID certificate in CI secrets. The owner needs to set these up before the first cask release.

## 7. Recommendation

1. **Language and TUI:** Go 1.26 with Bubble Tea v2.1, Lip Gloss v2 and Bubbles v2. Pin Ultraviolet. Use Fang and Cobra for the command layer. Build all of simstim's own screens (wizard, host browser, session list, dialogs) in Bubble Tea, in the alternate screen. Use Crush's code as a reference for Amp-level polish.
2. **Attach mode:** implement attach as a `tea.ExecCommand`. Bubble Tea then releases the terminal in the right order and repaints afterwards. Inside `Run()`:
   - raw mode and a cancellable stdin
   - one kitty entry of our own
   - an output pump with a **passive** mode tracker
   - the input filter
   - SIGWINCH forwarding
   - on any exit path (detach key, lost connection, SIGTERM, SIGTSTP), the ordered reset list from section 2.2, in one write
   **Never send RIS.** Detach by closing the SSH channel, so zmx's own RIS never reaches Ghostty.
3. **Transport:** the system OpenSSH client on a local PTY, with `-e none`, our own ControlMaster socket and keep-alives. Keep `x/crypto/ssh` behind an interface for later.
4. **Input filter:** the small purpose-built filter from section 3.2. It matches the detach key (default `Ctrl+\`, configurable, double-press sends it through) and Ctrl+V in legacy, kitty CSI u (lock bits masked, event types, base-layout key) and modifyOtherKeys forms. It forwards everything else byte for byte and never scans inside a bracketed paste. Test it with table tests, a split-stream property test and Go fuzzing.
5. **Images and drops:**
   - Use `golang.design/x/clipboard` (no cgo), with `wl-paste`, `xclip` and `osascript` fallbacks.
   - Upload over the ControlMaster connection to a content-addressed path, and paste the remote path as one bracketed paste.
   - Rewrite a bracketed paste only when every token is an existing local file.
   - Default the image key to Alt+V on Windows.
6. **Distribution:**
   - GoReleaser with `CGO_ENABLED=0` builds, a `curl | sh` installer, and a Homebrew tap cask that is signed and notarised through `notarize.macos` (quill) from Linux CI.
   - A Nix flake, cosign bundles, a syft SBOM, checksums, GitHub build provenance and reproducible-build flags.
   - Scoop, winget and deb/rpm come with Windows and Linux packaging later.
   - `simstim update` only for script installs.
7. **DX:**
   - Config at `~/.config/simstim/config.toml` (XDG on every Unix) with a JSON Schema.
   - `--json` on every list or show command, and shell completions through Fang.
   - `NO_COLOR` respected, and no telemetry.
8. **Licence:** Apache-2.0 with DCO, pending the owner's confirmation.

## Sources

Pinned commits used in this note:

- zmx: `github.com/neurosnap/zmx` at `2d23c0d` (v0.8.1). Files: `src/main.zig`, `src/util.zig`, `src/loop.zig`, `src/cfg.zig` and `README.md`.
- tmux: `github.com/tmux/tmux` at `9ad0056`. Files: `tty.c`, `client.c`, `server-client.c` and `tty-features.c`.
- Zellij: `github.com/zellij-org/zellij` at `519a237`. Files: `zellij-client/src/lib.rs` and `os_input_output.rs`.
- OpenSSH: `github.com/openssh/openssh-portable` at `75ef208` (10.6). Files: `sshtty.c`, `clientloop.c`, `channels.c` and `ssh.c`.
- Herdr: `github.com/herdrdev/herdr` at `916fcf8` (v0.9.5). Files: `src/client/attach.rs`, `src/terminal_modes.rs` and `src/client/terminal_setup.rs`.
- Ghostty: `github.com/ghostty-org/ghostty` at `ad97d33` (main, 1.4.0-dev). Files: `src/terminal/Terminal.zig` (`fullReset`, `softReset`), `PageList.zig`, `Screen.zig`, `kitty/key.zig`, `stream.zig`, `osc/parsers/mouse_shape.zig`, `src/input/key_encode.zig`, `src/input/paste.zig` and `macos/Sources/…/SurfaceView_AppKit.swift`.
- Kitty keyboard protocol: `github.com/kovidgoyal/kitty`, `docs/keyboard-protocol.rst` at `0d006c4`.
- Bubble Tea v2.1.0: `github.com/charmbracelet/bubbletea`. Files: `exec.go`, `tea.go`, `tty.go` and `cursed_renderer.go`. Releases: v2.0.0 (2026-02-24) and v2.1.0 (2026-10-08).
- Ultraviolet: `github.com/charmbracelet/ultraviolet`. Files: `decoder.go` (`EventDecoder.Decode`) and `event.go`.
- `golang.org/x/crypto/ssh` at `cf169cd`, `session.go` (`RequestPty` and `WindowChange`).
- `charmbracelet/x/ansi` v0.11.9: `kitty.go`, `xterm.go` and `mode.go`.

Docs and pages:

- OSC 7501 Program Status Protocol: superlogical.com/rex/docs/build/program-status (v0.4)
- Claude Code: code.claude.com/docs/en/interactive-mode, /keybindings, /fullscreen; issues #30934, #46249, #42712
- Codex: `codex-rs/tui/src/clipboard_paste.rs`, `bottom_pane/chat_composer/`, `custom_terminal.rs`
- Amp: ampcode.com/docs/cli, ampcode.com/docs/orbs, and the posts "Look Ma, No Flicker" (2025-09-02) and "Amp, Rebuilt" (2026-05-06)
- GoReleaser: goreleaser.com (customisation pages for `homebrew_casks`, `notarize` and `sboms`; blog post on reproducible builds)
- Homebrew: 5.0.0 and 6.0.0 release notes; `docs/Homebrew-Security-and-Supply-Chain.md`
- `golang.design/x/clipboard` v0.11.0 and v0.8.0 release notes; `ebitengine/purego` v0.11.1
- clig.dev
