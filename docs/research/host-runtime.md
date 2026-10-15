# Host-side session runtime

Research note for simstim, written 14–15 October 2026. It covers the part of simstim that runs on each host: it holds the agent's terminal, keeps it alive, restores the screen, and reports status.

Method:

- **Sources:** I read the source code of zmx, Ghostty (libghostty-vt), herdr, agent-deck, Agent of Empires, Claude Squad, ccmanager, cmux, Codex, opencode, Gemini CLI, Goose and Aider at their October 2026 heads. I also used the systemd source and man pages, the kernel docs, and the official agent docs.
- **Repos:** cloned into a scratch folder and treated as untrusted data.
- **Unchecked claims** are marked UNVERIFIED and listed again at the end.

## Summary

- **zmx** (MIT, Zig, v0.8.1) proves the right idea: one small daemon per session, raw bytes passed straight through, and libghostty-vt used only to restore the screen. But simstim should not depend on it at run time:
  - **No stable interface.** Its list format and wire protocol have no version, and both broke in 2026.
  - **Exit codes are lost.**
  - **No event stream** for detached sessions.
  - **Its client wipes Ghostty's scrollback on detach.**
  - **Bus factor of one.**
  - **A VT crash kills the session.** libghostty-vt runs inside the PTY holder, so a crash there killed sessions twice in 2026: #182 during long Codex sessions, and #215 on a resize.
- **libghostty-vt** is the only screen-restore engine with good enough fidelity:
  - it tracks kitty keyboard flags, modes, palette and both screens;
  - it has a binary snapshot;
  - it can **answer terminal queries** while no client is attached;
  - it has callbacks for BEL, OSC 9/777, OSC 9;4 and the new **OSC 7501 program status**.

  Its API is unstable and untagged, and it needs Zig 0.16 to build. Bindings exist for Go (`go.mitchellh.com/libghostty`, cgo) and Rust (`libghostty-vt` 0.2.x). Every other engine fails on kitty keyboard state, so Shift+Enter would break after a re-attach.
- **Recommended design (option c):**
  - **One small holder process per session.** It owns the PTY, a socket, a raw ring buffer, the manifest and the status engine. The shadow terminal runs in a **separate child process**, so a VT crash cannot kill the agent.
  - **No central daemon in v1.** A stateless `simstim-host` CLI covers list, watch and hooks.
  - **systemd hosts:** each session is a **transient user service** with `MemoryHigh` and `MemoryMax` and `OOMPolicy=continue`. **Linger must be on.**
  - **Other hosts:** a zmx-style daemonised holder, plus an RSS watchdog.
  - **Upgrades:** binaries are installed side by side, and a **versioned hello** lets new clients drive old holders. In-place re-exec comes later.
- **Status:** use layers.
  1. Process state.
  2. Agent self-report: OSC 7501, and hooks injected per launch.
  3. Terminal hints: OSC 9;4, title, notifications.
  4. Screen rules kept as data, as in herdr.

  herdr stopped trusting hooks alone for "idle", after stale states caused by interrupts and subagents.
- **v2 respawn** is practical for all eight agents. Store a resume command template and the agent's own session ID in a per-session manifest, and capture the ID with hooks where the agent cannot take a chosen ID. Run Codex with `--no-daemon`, because its default shared daemon escapes the session.
- **v2 worktrees:**
  - Location: `$XDG_DATA_HOME/simstim/worktrees/<repo>/<slug>-<id>`.
  - Branch: `simstim/<slug>`.
  - Lock the worktree while the session lives. Copy `.worktreeinclude` files.
  - Remove the worktree only when it is clean or merged.

## 1. zmx

I read the source at `main` 2d23c0d (3 October 2026), the changelog, and the open and closed issues.

### 1.1 Facts at a glance

| Item | Value |
| --- | --- |
| Repo | github.com/neurosnap/zmx, docs at zmx.sh |
| Latest release | v0.8.1, 5 September 2026. Ten commits on `main` since then. |
| Language | Zig. It needs Zig 0.16 (`minimum_zig_version = "0.16.0"`). |
| VT library | Ghostty, pinned to commit 8af6897 ("ghostty-1.3.2-dev"). zmx uses the native Zig module `ghostty-vt`, not the C API. |
| Licence | MIT |
| Platforms | Linux (static musl binaries for x86_64 and aarch64) and macOS (arm64 and x86_64). No Windows. An OpenBSD port is an open PR (#277). |
| Binary size | Linux x86_64 v0.8.1: 15.7 MB, static, not stripped, with debug info. macOS arm64: 2.5 MB. |
| Build mode | Release binaries use `ReleaseSafe` for zmx and for ghostty-vt (`build.zig`). A failed safety check aborts the process. |
| Popularity | About 2,200 stars, 168 forks and 32 open issues on 14 October 2026 |
| Size of code | About 9,200 lines of Zig in `src/`. About half of `util.zig` is tests. |

### 1.2 Architecture

- Each session is one daemon process with one Unix socket. The idea comes from abduco.
- `zmx attach <name> [cmd...]` is an "upsert". If no socket answers, the client double-forks a daemon (`daemonize.zig`). The daemon calls `setsid()`, sends stdio to `/dev/null`, and runs `forkpty()`. The child execs the command, or a login shell (`-zsh`) when there is no command.
- The child inherits the environment of the `zmx attach` process. zmx adds `ZMX_SESSION=<name>`. It sets `TERM=xterm-256color` only when `TERM` is unset or `dumb`. It does not set `COLORTERM` or `TERM_PROGRAM`.
- The daemon and the client both use `poll(2)`. The daemon reads the PTY in 4 KiB chunks. It feeds each chunk to a ghostty-vt `Terminal`, and appends the same bytes to each client's output buffer.
- The only rewrite on the live path: `OSC 133;A` gets `redraw=0`, so Ghostty does not clear the prompt on resize (issue #111).
- Socket directory: `$ZMX_DIR`, else `$XDG_RUNTIME_DIR/zmx`, else `$TMPDIR/zmx-<uid>`, else `/tmp/zmx-<uid>`. Logs: `$XDG_STATE_HOME/zmx/logs` or `~/.local/state/zmx/logs`.
- The session name is the socket file name, so its length is limited by the Unix socket path limit (about 104 bytes on macOS, 108 on Linux).
- Scrollback in the shadow terminal: 10,000 lines (`cfg.zig`, `max_scrollback_lines`). There is no environment variable to change it.

### 1.3 Full CLI surface (v0.8.1 plus `main`)

| Command | What it does |
| --- | --- |
| `attach [--labels "k=v ..."] <name> [cmd...]` | Attach, and create the session if needed. Inside a session it switches the calling terminal instead. |
| `run <name> [-d] [cmd...]` | Types a command into a **bash** session (not `$SHELL`) and appends a `ZMX_TASK_COMPLETED:<id>:$?` marker. Synchronous unless `-d`. Not usable for interactive agents. |
| `send <name> <text...>` | Raw bytes into the PTY. No newline, no marker. Does not take leadership. |
| `print <name> <text...>` | Injects text into the display and scrollback only. The PTY sees nothing. |
| `write <name> <path>` | Writes stdin to a file through the session shell with `base64` and `printf`. Only works when a shell owns the PTY. |
| `detach` | Detach all clients. `Ctrl+\` detaches the current client, unless `ZMX_NO_DETACH_KEY` is set. |
| `list` / `ls [--short]` | List sessions (format in 1.4). |
| `get <name> [key]`, `set <name> k=v ...`, `clear <name>` | Session labels, kept in memory only. |
| `print-env [-s] <name> [key]` | Tracked environment of the leader client (see 1.6). |
| `kill <name>... [--force]` | SIGHUP to the child's process group, then SIGKILL after 500 ms. A trailing `*` matches a prefix. |
| `history <name> [--vt\|--html] [--screen\|--scrollback N]` | Dump the shadow terminal as plain text, VT or HTML. `--screen` and `--scrollback` are on `main` only. |
| `wait <name>...` | Wait for `run -d` tasks. |
| `tail <name>...` | Follow output. Since v0.7.0 it strips ANSI escapes, so OSC sequences are lost. |
| `completions <shell>`, `version`, `help` | Utilities. `version` prints the zmx and ghostty_vt versions and the socket and log directories. |

Environment variables: `SHELL`, `ZMX_DIR`, `ZMX_SESSION`, `ZMX_SESSION_PREFIX`, `ZMX_TRACK_ENV`, `ZMX_DIR_MODE`, `ZMX_LOG_MODE` and `ZMX_NO_DETACH_KEY`.

### 1.4 `zmx list` output and how stable it is

One line per session, tab-separated `key=value` fields, sorted by name (`util.zig` `writeSessionLine`):

```text
name=dev<TAB>pid=123<TAB>clients=2<TAB>created=1789495314<TAB>cwd=/home/u/src<TAB>cmd=claude<TAB>ended=…<TAB>exit_code=…<TAB>label1=v1
```

- `cwd`, `cmd`, `ended` and `exit_code` appear only when set. `ended` and `exit_code` exist only for `run` tasks.
- Labels follow as more `k=v` fields in the same key space. Only `name`, `start_dir` and `cmd` are reserved, so a label called `pid` or `cwd` would collide.
- Label keys and values may hold only `[A-Za-z0-9._-]`: no spaces, no slashes, no colons.
- When `ZMX_SESSION` is set, each line starts with `→` (current session) or two spaces.
- A session that does not answer prints `name=x<TAB>err=Timeout<TAB>status=unreachable`. A refused socket prints `status=cleaning up`, and zmx deletes it.
- `cmd` and `cwd` come from fixed 256-byte fields in the `Info` reply. A longer `cmd` is cut and ends in `...`.
- `created` is Unix seconds. There is no last-activity field and no status field. `clients` counts every connected socket, including `tail` clients.
- Stability: the format has changed twice in 2026. v0.4.2 renamed keys. v0.8.0 replaced `start_dir` with `cwd`. There is no JSON output; issue #220 asks for it, and the maintainer's answer is that tab-separated pairs are easy to parse. Treat the format as unstable. Parse it by key, never by position.
- `list` probes each socket in turn with a 1 s timeout, so one stuck daemon slows the whole list.

### 1.5 IPC protocol and versioning

- Wire format (`ipc.zig`): an 8-byte header, a packed `{tag: u8, len: u32}` in host byte order, then `len` bytes of payload. There are 23 tags, from `Input = 0` to `Capture = 22`.
- Since April 2026 the tags are frozen. The enum is non-exhaustive so that old daemons ignore unknown tags. Tests pin each tag value and the 552-byte size of the `Info` struct. The comment says: "Need a new field? → add a new `Tag` value".
- There is **no version handshake** and no capability negotiation. A client learns what a daemon supports only by trying.
- **Does an upgrade kill running sessions?** Not directly. Each daemon keeps running its old code after the binary on disk is replaced. The risk is that a new client cannot talk to an old daemon. That happened twice in 2026:
  - Before v0.6.0, a change to `Info` made old daemons unreachable. The README still lists "upgrading ... will kill all your sessions" as a known issue.
  - v0.7.0 clients hung against v0.6.0 daemons (#211). This was fixed in v0.7.1 by sending both the old 4-byte and the new 8-byte resize payload.
- Freezing the tags makes this rarer, but nothing enforces compatibility beyond the tests. The breaking CLI changes in v0.5.0 and v0.7.0 also show the project is pre-1.0.

### 1.6 Multi-client leader rules

From `loop.zig` (`handleInput`, `handleInit`, `handleResize`) and `util.zig` (`isUserInput`):

- One client at a time is the **leader**. Only the leader's input reaches the PTY, and only the leader's window size sets the PTY size.
- The first client to send `Init` becomes leader when there is none.
- A non-leader becomes leader when it sends **user input**:
  - printable characters, or CR, LF, tab or backspace;
  - kitty `CSI … u` key events (not releases), or `CSI … ~`;
  - modified arrow keys.
- Mouse reports, focus events, private-marker CSI replies and kitty keyboard status replies do not count. Other non-leader bytes are dropped, so only the leader's terminal answers the program's queries.
- On a leader change, the daemon asks the new leader for its size and resizes at once.
- When the leader leaves, there is no leader until someone types.
- `zmx send` never takes leadership (#201).
- Open gap (#267): a GUI client cannot say "size the PTY from me" without typing, and cannot learn whether it is the leader.

### 1.7 Environment tracking

- Each attaching client sends the values of the tracked variables as an `EnvSet` message.
- The defaults are `DISPLAY`, `SSH_AUTH_SOCK`, `SSH_AGENT_PID`, `SSH_CONNECTION`, `WINDOWID`, `XAUTHORITY`, `KITTY_LISTEN_ON`, `KITTY_PID` and `KITTY_WINDOW_ID`. `ZMX_TRACK_ENV` overrides the list.
- `zmx print-env [-s] <name>` prints the **leader's** values: `KEY=VALUE`, or `-KEY` for unset. With `-s` it prints `export` and `unset` lines.
- The running program does not change. A shell picks up new values only through a prompt hook (`eval "$(zmx print-env -s .)"`).
- An agent's environment cannot be patched from outside, so a stale `SSH_AUTH_SOCK` stays stale inside Claude Code. The usual fix is a stable symlink path, updated on each attach.

### 1.8 How restore works with libghostty-vt

- The daemon feeds every PTY byte into a ghostty-vt `Terminal` (10,000 lines of scrollback). ghostty-vt is never in the live path to the client.
- On **re-attach** (`handleInit`, only when `has_had_client` is true), zmx:
  1. resizes the shadow terminal to the new leader's size, so the snapshot is not wrapped twice;
  2. serialises with ghostty-vt's `TerminalFormatter` in two phases (`util.zig` `serializeTerminalState`, issue #31):
     - **Phase 1:** scrollback rows as styled VT text, with no modes and no cursor. These rows scroll into the client terminal's native scrollback, so Cmd+F works on them.
     - **Clear:** `ESC[2J ESC[H ESC[0m`.
     - **Phase 2:** the visible screen with extras: modes, scrolling region, kitty keyboard flags, both screens (`.screen = .all`), cursor, OSC 10/11/12 colour overrides, OSC 7 pwd and the window title (OSC 2).
     - **Left out on purpose:** synchronised output (mode 2026), because replaying it stalls the client; tabstops, because they move the cursor; and the palette.
  3. resizes the PTY. Then it sends SIGWINCH to the foreground process group even if the size did not change, so Ink-based TUIs such as Claude Code repaint.
- The zmx **client** clears the screen on attach. On detach it sends `ESC c` (RIS) and resets OSC 110/111/112. In Ghostty, RIS calls `Screen.reset()` and **wipes the local scrollback**. A simstim client that speaks the protocol itself can choose a softer reset.
- Not restored:
  - kitty graphics images (#14);
  - OSC 8 hyperlinks;
  - output written before the first client registers (#246, #264; the fix, PR #265, is still open);
  - application subscriptions such as `DECSET 2031`, when they were set before the first attach.
- Queries while detached: the daemon answers **only DA1 and DA2** (`respondToDeviceAttributes`). It does not answer the kitty keyboard query, OSC 10/11, XTVERSION or OSC 7501. An agent that starts with no client attached may switch off features.
- PR #243 (open) adds "libghostty snapshots": the daemon sends ghostty-vt's internal state to clients that are themselves built on libghostty, instead of a VT byte stream. Today only `monstar` can load them. The Ghostty app cannot.

### 1.9 Known bugs that matter for simstim

| Issue | Effect on simstim |
| --- | --- |
| #182 (fixed upstream in ghostty PR #13394) | `HyperlinkSetOutOfMemory` **aborted the daemon during long Codex sessions**. The agent died with it. |
| #215 (fixed in ghostty PR #13524) | SIGABRT on a resize to half the columns when scrollback holds OSC 8 links. The session and its scrollback were lost. |
| #246, #264 (open, PR #265) | Output before the first client registers is lost, for example a startup `DECSET 2031`. |
| #247 (open) | `attach <name> <cmd>` exits 0 whatever the command did. When the command exits, the session vanishes, so the exit code is lost. |
| #226 (open) | The daemon stays in the cgroup of whatever started it. Stopping that unit (here the foot terminal's service) kills the "detached" sessions. |
| #268 (open) | "session not ready" when a systemd unit creates a session at boot. |
| #259, #249 (still open, fixed on `main` in fca1964, not yet released) | A session switch with no leader killed the daemon. |
| #220, #221 (open) | No JSON output. No way to see which client is attached. |
| #76 (open) | Sessions and their metadata are lost on reboot. Users ask for an agent "resume" command on restore. |
| Code reading, no issue found | Each client's output buffer (`Client.write_buf`) has no size cap. A stalled client that never closes its socket makes the daemon's memory grow. PTY input is capped at 256 KiB. |

### 1.10 Release cadence and maintainers

- 14 tags in about 10 months: v0.0.1 on 28 November 2025, then roughly one a month, ending with v0.8.0 (3 September 2026) and v0.8.1 (5 September 2026).
- Eric Bower (neurosnap, pico.sh) wrote 371 of 485 commits (76 %). The next contributor has 12 commits (Ian Tay, who froze the wire tags). The bus factor is one.
- The README's "smol contract" says zmx aims to be a program "a single person can maintain" and "can be finished". Bigger additions, such as the zmosh UDP transport, are being proposed under `contrib/` (PR #271) rather than in the core.
- Zig itself is pre-1.0. The move to Zig 0.16 needed a rewrite (July 2026).

### 1.11 Bundle, embed or reimplement?

- **Bundle the binary.** The MIT licence allows it. Static binaries exist for Linux and macOS. simstim could install a pinned zmx on each host and drive it through `zmx attach` over `ssh -t`. Against it:
  - the list format and the protocol are unversioned;
  - there is no event stream for detached sessions (`tail` strips OSC);
  - exit codes are lost;
  - the RIS on detach wipes local scrollback;
  - a crash in the VT library kills the agent;
  - simstim would still need its own host helper for metadata, status, memory caps and hooks.
- **Embed as a library.** Not practical today. There is no C ABI. A Zig module "libzmx" is under discussion (#127). A proof of concept is in PR #271 (`src/lib.zig`), and a fork, 0xsend/zmx `sox-seams`, exposes about 130 lines of surface. The maintainer warns that the interface would carry zmx's choices: auto-daemonising, one daemon per session, `poll(2)` and the Unix socket. This only helps if simstim's host part is written in Zig.
- **Reimplement the idea.** The core is small: a daemon with a PTY, a socket and a shadow terminal. zmx's code and issues are a valuable checklist of the hard parts:
  - query replies must not take leadership;
  - answer DA when detached;
  - the `OSC 133;A redraw=0` rewrite;
  - two-phase serialisation;
  - leave out mode 2026;
  - the OSC 7 NUL byte;
  - replay the title;
  - send SIGWINCH even when the size is unchanged;
  - pop kitty keyboard flags on detach;
  - replay on the first attach.

  The MIT licence allows copying code with attribution.

**Verdict:** reimplement the holder, modelled closely on zmx, and keep zmx as a reference and as a test oracle. Do not depend on zmx at run time. Section 4 gives the reasons.

## 2. libghostty-vt

Sources: Ghostty `main` at ad97d331 (14 October 2026), its `include/ghostty/vt*.h` headers, build files and blog posts.

### 2.1 State of the C API

- **Licence:** MIT.
- **Version:** none of its own. The README says: "We haven't tagged libghostty with a version yet." `build_info.h` reports the Ghostty version, `1.3.2-dev`.
- **Ghostty releases:** the last ones are v1.3.0 (9 March 2026) and v1.3.1 (13 March 2026). The September 2025 post "Libghostty Is Coming" hoped for a tagged libghostty-vt within six months. That has not happened.
- **Where Ghostty lives:** the April 2026 post "Ghostty Is Leaving GitHub" says GitHub will stay as a read-only mirror. The mirror was still updated on 14 October 2026. The new home was UNVERIFIED at the time of writing.
- **Stability:** `vt.h` says: "This is an incomplete, work-in-progress API. It is not yet stable and is definitely going to change."
  - The README says the behaviour is very stable, but the signatures are still in flux.
  - `terminal.h` had 104 commits between March and October 2026.
  - The design tries to keep the ABI steady: sized structs, typed enums, set/get options, and a JSON type schema for binding generators.
- **Surface:** about 30 headers and about 210 functions. The parts simstim needs:
  - **Terminal** (`terminal.h`): `ghostty_terminal_new`, `_vt_write`, `_resize`, `_reset`, `_get` and `_set`. Scrollback has line and byte caps.
  - **Formatter** (`formatter.h`): plain text, VT or HTML, with extras for palette, modes, scrolling region, tab stops, pwd, cursor, style, hyperlink, kitty keyboard flags and charsets.
    - The C formatter covers only the active screen. Only the Zig API formats both screens, which zmx uses through `.screen = .all`.
  - **Snapshot** (`snapshot.h`, added August 2026): a binary format ("GHOSTSNP", CRC32C-checked records).
    - It covers both screens, full history, modes, palette, title, pwd, the kitty keyboard stack and even a half-received escape sequence.
    - It says: "format version 1 is a work in progress and does not yet carry a binary-compatibility guarantee."
    - The Ghostty app cannot load it. Only rockorager/monstar can, experimentally.
  - **Callbacks for status** (section 5): `OPT_BELL`, `OPT_DESKTOP_NOTIFICATION` (OSC 9 and 777), `OPT_PROGRESS_REPORT` (OSC 9;4), `OPT_PROGRAM_STATUS` (OSC 7501, merged in ghostty PR #14560 on 6 October 2026), title and pwd changes, and `CLIPBOARD_WRITE` (OSC 52).
  - **Answering queries while no client is attached.** Set `GHOSTTY_TERMINAL_OPT_WRITE_PTY`, and replies are written back through the callback.
    - Answered from the terminal's own state: DSR, CPR, DECRQM, DECRQSS, the kitty keyboard query `CSI ? u`, OSC 4/10/11/12 colour queries, XTGETTCAP and XTVERSION.
    - Answered only when you set a callback: DA1/2/3 (`OPT_DEVICE_ATTRIBUTES`), `CSI 14/16/18/19 t` (`OPT_SIZE`), the colour-scheme report (`OPT_COLOR_SCHEME`), OSC 52 reads and `OSC 7501;?` (`OPT_PROGRAM_STATUS`).
    - This is exactly what a holder needs when an agent starts with no client attached.
- **Known replay gaps:**
  - per-cell OSC 8 hyperlinks (PR #14711, open);
  - kitty graphics (PR #12182, open since April 2026);
  - a pending-wrap bug, fixed on 13 October 2026 (PR #14670).

### 2.2 Bindings

| Language | Option | State |
| --- | --- | --- |
| Go | `go.mitchellh.com/libghostty` (github.com/mitchellh/go-libghostty, MIT) | By Mitchell Hashimoto. Created April 2026, about 155 stars, pushed 14 October 2026. No tags, only pseudo-versions. Needs cgo and Go 1.26. Links statically through pkg-config and cross-compiles with `zig cc`. Wraps almost the whole C API, including snapshots. "I'm not promising any API stability yet." |
| Go, no cgo | seruman/hauntty (libghostty-vt to WASM to Go with `wasm2go`) | Hobby project, 7 stars. Needs Go 1.27 with `GOEXPERIMENT=simd`. Not for production. |
| Rust | `libghostty-vt` and `libghostty-vt-sys` 0.2.2 (Uzaaft/libghostty-rs, MIT or Apache-2.0) | Released 28 September 2026, about 165,000 downloads, about 394 stars. `build.rs` needs Zig 0.16 and fetches a pinned Ghostty commit. herdr uses its own in-tree crate built the same way, ReleaseFast by default, with a local patch. |
| Zig | Ghostty as a `build.zig.zon` dependency, module `ghostty-vt` | The richest API: the formatter can format both screens. Used by zmx. But Zig is pre-1.0, and the move to 0.16 broke builds. |

### 2.3 Build needs and size

- **Toolchain:** Zig 0.16.0. Build with `zig build -Demit-lib-vt`. Outputs are `.a`, `.so` or `.dylib`, pkg-config files, an xcframework and wasm.
- **SIMD:** comes from bundled simdutf and highway, built without libc++.
- **Release artefacts:** there is no official prebuilt native library. Ghostty's nightly CI publishes a signed source tarball, `libghostty-vt-source.tar.gz`, on the "tip" pre-release. Every consumer builds it with Zig.
- **Size:** wasm is about 0.8 to 1.1 MB. There is no official native figure. zmx, which links it in, is 2.5 MB on macOS arm64, and about 3.0 MB of code and data on Linux. So the library itself is probably 1.5 to 2.5 MB (UNVERIFIED estimate).
- **Rule for simstim:** pin one Ghostty commit. Wrap the library behind a narrow interface: feed bytes, resize, format as VT, encode and decode snapshots, set callbacks. Upgrade it on purpose, with replay tests.

## 3. Alternatives for screen restore

| Engine | Language, licence | Replays as VT bytes? | Kitty keyboard state | Notes |
| --- | --- | --- | --- | --- |
| **libghostty-vt** | Zig/C, MIT | Yes: modes, palette, kitty flags, cursor, scrollback. Also a binary snapshot. | Yes | Proven inside a shipping terminal. Also answers queries. The API is unstable. |
| shpool_vt100 0.1.3 (shpool's default) | Rust, fork of vt100 | Screen plus input modes (bracketed paste, cursor and keypad, mouse) | No | No OSC 8 and no mode 2026. Every row is 1024 columns wide, which wastes memory (#46). shpool's restore modes: `screen` (default), `lines=n`, `simple` (SIGWINCH only). |
| shpool-vterm 0.2.8 | Rust | History, screen or bottom lines | Writes the sequences but seems not to track the state (UNVERIFIED) | Experimental. Handles grapheme clusters. |
| vt100 0.16.2 | Rust, MIT | Yes (`contents_formatted`, `state_formatted`, `input_mode_formatted`) | No | Character width but not full grapheme clusters. No OSC 8. |
| alacritty_terminal 0.26 | Rust, Apache-2.0 | **No.** You would write the serialiser yourself. | Yes, a mode stack | Strong emulation and OSC 8. Grapheme handling looks limited (UNVERIFIED). |
| wezterm-term and termwiz | Rust, MIT | No | Yes, a stack per screen | wezterm-term is not on crates.io; only a fork is. termwiz 0.23.3 is from March 2025. |
| avt 0.18 (asciinema) | Rust, Apache-2.0 | Yes (`Vt::dump()`) | No | Small and clean. No OSC 8. |
| charmbracelet/x/vt 0.1.0 (first tag 12 October 2026) | Go, MIT | Cells with styles and links only. No modes, cursor or kitty state. | No ("TODO: Support Kitty, CSI u") | Young. |
| vito/midterm | Go, MIT | Yes, both screens (`MarshalBinary`) | No | Small. |
| taigrr/bubbleterm, hinshun/vt10x, liamg/darktile, go-vte | Go | Partial, or parser only | No | vt10x and darktile are stale (last pushed 2023). go-vte is only a parser. |

**Ranking for simstim:**

1. **libghostty-vt.** It is far ahead, and it is the same engine as the user's Ghostty, so the replayed screen matches what Ghostty would have drawn.
2. **alacritty_terminal or wezterm-term.** Their state is good, but you would have to write the replay yourself.
3. **avt, vt100 and shpool's engines.**
4. **The Go emulators**, which are too young or too small.

None of the alternatives restores kitty keyboard flags. Without them, Shift+Enter breaks after a re-attach until the agent re-enables the protocol. That one gap rules them out for simstim's fidelity goal.

## 4. Design options

### 4.1 What the host runtime must do

- Hold one PTY per session. Keep it alive with no client attached.
- Pass bytes through untouched to the attached client. This is what keeps true colour, the kitty keyboard protocol, OSC 52, OSC 9/777, native scrollback and Cmd+F.
- Keep a shadow terminal and send a snapshot on re-attach.
- Keep metadata: agent, cwd, created, last activity, status, attached clients and exit code.
- Detect status (section 5). Cap memory (section 6).
- Survive client loss, SSH loss, logout and simstim upgrades.
- Later (v2): survive reboots by respawning with each agent's resume command.

### 4.2 Options compared

| | (a) Wrap zmx behind an adapter | (b) One simstimd that holds all PTYs | (c) Hybrid: one small holder process per session, plus a stateless host CLI and an optional broker |
| --- | --- | --- | --- |
| Blast radius of a crash | One session | **All sessions** | One session |
| VT crash kills the agent? | Yes. ghostty-vt runs in the PTY holder in ReleaseSafe mode (#182, #215). | Yes, for all sessions, unless the VT runs elsewhere | No, if the shadow VT runs in a child process (see 4.4) |
| Fidelity | Very good pass-through. RIS on detach in zmx's own client. Only DA answered while detached. First-attach output lost (open bug). | Same as (c) | Same pass-through, plus our own fixes: answer queries while detached, replay on first attach, no RIS |
| Metadata and status | Labels only (`[A-Za-z0-9._-]`, in memory). No activity time. No exit code. No raw byte events unless we speak zmx's undocumented wire format as a passive client. | Full | Full: the holder sees every byte and owns the manifest |
| Memory caps | Wrap the command: `zmx attach X systemd-run --user --scope -p MemoryMax=… -- agent` | Daemon puts each agent in its own scope or cgroup | Same as (b), done per holder |
| Upgrade story | Old daemons keep running. A new client may not talk to an old daemon: no handshake, two breaks in 2026. | Restarting the daemon kills every session, unless it hands the PTY fds over. shpool kills all sessions on a protocol break (#377). herdr's fd handoff is still "experimental" after many fixes. | Old holders keep running old code. A versioned handshake lets new clients drive old holders. Optional in-place re-exec later. |
| Effort | Lowest to a first demo. But we still need a host helper for metadata, status, caps and hooks, and we must work around zmx bugs we cannot fix. | High: fd passing, state serialisation, reaping children we no longer parent | Medium: about 2,000 to 4,000 lines plus the VT binding |
| Dependency risk | One maintainer, Zig pre-1.0, unversioned interfaces | Ours | Ours. Still tied to libghostty-vt. |

### 4.3 Why one process per session

- **Isolation.** A bug, a leak or an abort in one holder costs one session.
- **Upgrades without a handoff.** A running holder keeps its old code in memory when the binary changes on disk. A new client needs to talk to it only through a small, versioned protocol.
- **Zero-downtime host CLI.** The host CLI (`simstim-host ls`, `new`, `kill` and so on) can be replaced at any time. It keeps no state in memory. The truth is in the holders and in the manifests on disk.
- zmx, abduco and dtach all use this model. Claude Code's own background supervisor runs each session as its own process. If the supervisor dies, each session process waits about a minute for a new supervisor to reconnect, and its state files keep fields that an older version does not know (Claude Code docs, "agent view").
- tmux, shpool and herdr use one server, and each has an upgrade problem:
  - tmux and shpool: a client of a new version cannot attach to an old server;
  - herdr: needs an experimental fd handoff.

### 4.4 Keep the VT parser out of the PTY holder

- The PTY holder should be tiny and boring. It needs:
  - `forkpty`;
  - the socket;
  - a ring buffer of raw output;
  - a small hand-written scanner for OSC, BEL and title.
- The ghostty-vt shadow terminal should run in a **child process** that the holder feeds through a pipe.
- If the shadow process crashes, the holder restarts it and replays the ring buffer, for example the last 8 MiB. The agent never notices. The next attach gets a best-effort snapshot, and simstim flags it as "rebuilt".
- If the shadow process is slow, the holder must never block the PTY. It drops to "resync from the ring buffer" mode instead.
- Cost: one more process per session. Its memory use is UNVERIFIED and should be measured. A rough estimate is tens of MB with 10,000 lines of scrollback.
- A cheaper first step is to run the VT in-process, but built so that it cannot abort (Rust `catch_unwind` around the feed, or the C API built ReleaseFast). This is weaker: ReleaseFast turns an abort into undefined behaviour.

### 4.5 Upgrade story in detail

1. **Install side by side.** Put each version at `~/.local/share/simstim/bin/<version>/simstim-host`, with a `current` symlink. Each holder records its version and binary path in its manifest. Delete an old version only when no live holder uses it. This is cmux's approach for `cmuxd-remote` (`~/.cmux/daemon/<version>/`).
2. **Versioned handshake.** The first frame each way is `hello {protocol: N, features: [...]}`.
   - Message types are append-only, and unknown types are ignored. zmx does this, but with no hello.
   - Integers are little-endian on the wire. zmx uses host byte order, which is fine on one host but fragile.
   - Control frames are JSON. Data frames are raw bytes.
   - A new client that needs a missing feature shows "restart this session to enable X" and keeps working. This is herdr's "endpoint generation" rule.
3. **Live holder upgrade (later, optional).** The holder re-execs itself in place:
   - it writes its state (metadata, client list, VT snapshot or ring buffer) to a memfd or temp file;
   - it clears `FD_CLOEXEC` on the PTY master and the listening socket;
   - it calls `execve` on the new binary with `--resume-fd`.

   The PID stays the same, the agent stays its child, and `waitpid` keeps working. If `execve` fails, the old code keeps running. This is the same idea as `systemctl daemon-reexec`.
   - Two alternatives are weaker. fd passing over `SCM_RIGHTS` to a new process (herdr) loses the parent link, so the exit status goes to the subreaper. systemd's `FileDescriptorStoreMax=` store works only on systemd hosts.

## 5. Agent status detection

Sources: herdr 916fcf8, agent-deck 8d0f1fe, Agent of Empires d3c5fb6, Claude Squad ce1ffb4, ccmanager 46fe720, cmux 9d01f95, Codex d292f72 (all October 2026), and the Claude Code docs.

### 5.1 How the existing tools do it

| Tool | Holds the PTY? | Main signal | Other signals | Hooks |
| --- | --- | --- | --- | --- |
| **herdr** (Rust, libghostty-vt) | Yes | Per-agent screen rules kept as data (TOML "detection manifests", `src/detect/manifests/*.toml`). Regions include `osc_title`, `osc_progress`, `bottom_non_empty_lines(N)` and `prompt_box_body`. Rules can update from herdr.dev without a release. | Foreground process group via `tcgetpgrp` plus `/proc` or `proc_pidinfo`. OSC 0/2 title and OSC 9 / 9;4 tracking (`src/pane/osc.rs`). Working to idle needs 3 confirmations within 700 ms. | Since 0.6.7 (June 2026), hooks for Claude, Codex and others report **session identity only**. Reasons: stale "working" after a cancel, stale permission reports, subagent hooks waking idle panes. Hooks stay authoritative for Pi, OpenCode, Kilo and a few others. |
| **Agent of Empires** | No (tmux) | The same manifest idea, with the title from tmux `pane_title` | — | Hook events write a status file. The status file is only one rule; live screen rules outrank it for Claude. |
| **agent-deck** (Go) | No (tmux) | 1. Title glyph. 2. tmux activity with spike filtering (about 1 s). 3. `capture-pane` plus per-agent patterns. 4. Error banners. 5. A SHA-256 content hash as the last fallback. | "Waiting" versus "idle" depends on whether the user has looked since the agent finished. | Hooks with freshness windows (45 s and 2 min). Code documents a case where a hook said running while the screen showed a finished turn. OpenCode via its SSE stream (`GET /event`). Codex via the legacy `notify`. |
| **Claude Squad** | No (tmux) | SHA-256 of the captured pane every ~500 ms: changed means Running, same means Ready | One prompt string per agent, used only for auto-yes | None |
| **ccmanager** (Bun, `@xterm/headless`) | Yes | Per-agent regexes every 100 ms, for example "esc to interrupt" means busy and "Do you want…" means waiting. Idle is debounced by 1.5 s. | — | None |
| **cmux** (macOS, Ghostty) | Yes | Hooks first. Injected per launch through PATH shims: Claude `--settings`, Codex `-c hooks.<event>=…`. States: running, idle, needsInput, error and backgroundWorkPending. | OSC 9/99/777 for attention rings. It removed its title and file-time fallbacks on purpose. | Yes, the main source |

### 5.2 What the agents emit themselves

- **OSC 7501 Program Status Protocol.** This is new (October 2026), from Ghostty and Superlogical. Format: `ESC ] 7501 ; state=idle|working|done|blocked|error|clear : kind=permission|question|auth : …`, with a support query `ESC ] 7501 ; ?`.
  - **Claude Code v2.1.295 and later** reports working, waiting and done "if your terminal implements" it. It sends nothing when there is no answer to the query, in background sessions, or with `CLAUDE_CODE_DISABLE_TERMINAL_TITLE=1` (code.claude.com/docs/en/terminal-config).
  - **Codex** emits it whenever stdout is a TTY (`codex-rs/tui/src/terminal_program_status.rs`, October 2026). Whether a stable Codex release includes it is UNVERIFIED.
  - libghostty-vt parses it and calls back. The Ghostty app ignores it today.
- **OSC 9;4 progress.** Claude Code sends it only to ConEmu, Ghostty 1.2.0+ and iTerm2 3.6.6+ (`terminalProgressBarEnabled`).
- **Desktop notifications.** Claude Code's `preferredNotifChannel: auto` sends them to Ghostty, Kitty and iTerm2. Codex `tui.notification_method` auto sends OSC 9 to Ghostty and others, and BEL elsewhere; by default only when the terminal is unfocused. Gemini CLI has notifications off by default (osc9, osc777 or bell).
- **Window titles.** Claude Code shows a spinner glyph while working. Gemini CLI shows ◇ ready, ✋ action required and ✦ working. Amp shows a spinner and "Plugin confirmation needed".
- **Hooks:**
  - **Claude Code:** about 30 events. Stop does not fire on a user interrupt. API errors fire StopFailure. `permission_prompt` notifications wait about 6 s and fire only when the user seems away; PermissionRequest is immediate.
  - **Codex:** behind `[features] hooks`. Events: PreToolUse, PermissionRequest, PostToolUse, SessionStart, SessionEnd, UserPromptSubmit, Stop and Interrupt. The legacy `notify` fires only on `agent-turn-complete`.
  - **Gemini CLI:** BeforeAgent, AfterAgent, BeforeTool, AfterTool and Notification.
  - **OpenCode:** plugin events (`session.status`, `session.idle`, `session.error`, `permission.asked`), and an HTTP server with an SSE stream at `/event` (use `--port`).

**Important for simstim:** Claude Code and Codex choose which signals to send by terminal identity. SSH does not forward `TERM_PROGRAM` unless the server accepts it. Ghostty's own ssh wrapper only *asks* with `SendEnv TERM_PROGRAM TERM_PROGRAM_VERSION` and sets `COLORTERM=truecolor`. So the holder must set `TERM_PROGRAM=ghostty`, `TERM_PROGRAM_VERSION=<client's>`, `COLORTERM=truecolor` and a working `TERM` in the agent's environment itself. Exactly how Claude Code detects Ghostty is UNVERIFIED. The docs say its notifications reach Ghostty over SSH.

### 5.3 Recommended layers

The holder sees every byte, so it should read the signals from the stream, not by polling the screen.

| Layer | Signal | Role |
| --- | --- | --- |
| L0 Process | Child exit (code, signal, OOM from `memory.events`). Foreground process group (`tcgetpgrp` on the PTY master plus `/proc/<pid>/stat` or `proc_pidinfo`). Last output time and last input time. | Always on. Authoritative for **exited**, **errored** and **which agent is running**. Activity times are hints only. |
| L1 Self-report | **OSC 7501** (answer `OSC 7501;?` from the holder, read the reports through libghostty-vt's callback). **Hooks** that call `simstim-host hook <event>` with `SIMSTIM_SESSION` set, injected per launch the way cmux does: Claude `--settings <file>`, Codex `-c hooks.…`, an OpenCode plugin or its SSE stream, Gemini hooks. A monotonic `seq` on each report. | Authoritative when present. Use hooks above all for **agent session id**, transcript path and "blocked" edges (PermissionRequest, Elicitation). Do not trust a hook alone for "idle", because of the interrupt and subagent gaps. |
| L2 Terminal hints | OSC 9;4 progress, title spinner, OSC 9/777 notifications, BEL | Strong hints. Also forward them to the client so Ghostty shows them natively. |
| L3 Screen rules | herdr-style per-agent rules kept as data and run on the libghostty-vt screen after output settles (debounce about 0.5 to 1.5 s). Read only the bottom region. | Fallback for agents with no L1. May override L1 only when it sees a clear blocker or idle prompt. |

- **Status values:** `starting`, `working`, `needs_input` (blocked: permission, question or auth), `idle`, `done`, `errored` and `exited`.
- **Done** means "idle after working, not yet seen by a client". The client clears it on view, as agent-deck does.
- **Errored** means one of: a non-zero exit, a signal, an OOM kill, StopFailure, OpenCode `session.error`, or OSC 7501 `error`.

**Traps to design for:**

- **Interrupts:** after Esc or Ctrl+C, hook state can stay "working". Re-check after forwarding those keys.
- **Subagents:** their hooks carry `agent_id`; ignore them for the session's state.
- **Background work:** Stop with pending background tasks is not done.
- **Focus gating:** Codex and Claude notify only when unfocused or away. With no client attached, the holder should report focus-out (`CSI O`) if the agent turned on mode 1004.
- **Glyph changes:** spinner glyphs change between versions (Claude changed at 2.1.228). Keep rules as data with tests.
- **Stale prompts:** transcript viewers and scrollback show old prompts. Read only the live bottom of the screen.
- **Hash changes:** a hash change alone is noise, because of cursor blink and clocks.
- **Double replies:** never let both the holder and an attached Ghostty answer the same query.
- **Blocked hooks:** `allowManagedHooksOnly` or `disableAllHooks` can block injected hooks, and Codex's single `notify` slot may already be taken. Fall back to L2 and L3.

## 6. Lifecycle and reliability on hosts

Sources: the systemd source tree and man pages (October 2026), distro packaging, the kernel cgroup-v2 docs, XNU source, Apple TN2083, the Docker and WSL docs, and zmx issue #226.

### 6.1 Logout, linger and where sessions live

- **Logout does not kill sessions by default.** Debian, Ubuntu, Fedora and Arch build systemd with `default-kill-user-processes=false`. A process started from SSH stays in `session-N.scope`, which is "abandoned", not killed, at logout. The upstream default became yes in v230, but distros override it.
- **Without linger, the user manager stops.** `user@UID.service` stops 10 s after the last session ends (`UserStopDelaySec=10s`). Everything started with `systemd-run --user` stops with it.
- **The runtime directory goes too.** `/run/user/UID` is removed "on the user's final logout" (pam_systemd(8)).
  - zmx puts its sockets there by default.
  - Its daemons can survive logout, but their sockets can vanish (UNVERIFIED end to end).
- **Linger is cheap to turn on.** `loginctl enable-linger` (for yourself) uses the polkit action `set-self-linger`, which allows any user. On hosts without polkitd, it needs `sudo loginctl enable-linger $USER`.
- **Rule:** on systemd hosts, simstim's setup turns linger on, and refuses the systemd path if it cannot.

### 6.2 One unit per session

- **Use a transient service per session.** Its parent is the user manager, not the SSH session, so stopping a terminal's service (zmx #226) or logging out does not touch it.

  ```sh
  systemd-run --user --unit=simstim-s-<id> --slice=simstim.slice --collect \
    -p Type=exec -p KillMode=control-group -p OOMPolicy=continue \
    -p MemoryHigh=6G -p MemoryMax=8G -p MemorySwapMax=0 \
    --working-directory=<dir> --setenv=K=V ... \
    -- simstim-host holder <id> -- <agent argv>
  ```

- **The unit gets a clean environment.** It does not inherit the caller's. Pass the agent's environment explicitly (see 9.6).
- **Why not `--scope`?** A scope keeps the caller's parent and environment. Scopes gained `OOMPolicy=` only in systemd v253. Before that, and by default today (`DefaultOOMPolicy=stop`), an OOM kill of the agent also stops the scope, which kills the holder and loses the screen.
- **Change limits live:** `systemctl --user set-property --runtime simstim-s-<id>.service MemoryMax=12G`.
- **Total cap across sessions:** `~/.config/systemd/user/simstim.slice` with `MemoryHigh=80%` and `MemoryMax=90%`.
- **Restarting is safe.** `systemctl --user restart simstimd` (if a broker exists) does not touch the session units. They are siblings, and the default `KillMode=control-group` kills only the unit's own cgroup.
- **Do not let a daemon hold the PTY master.** If it did, a restart would close the master, and the agents would get SIGHUP.

### 6.3 Memory caps and OOM

- **MemoryHigh** throttles and reclaims, and never calls the OOM killer. The man page calls it "the main mechanism".
- **MemoryMax** is the hard limit. When it is reached, the OOM killer runs inside the cgroup.
- **MemorySwapMax=0** stops a leaking agent from thrashing swap beyond MemoryMax.
- **Who gets killed:** with `memory.oom.group=0` (the default unless `OOMPolicy=kill`), the kernel kills one task, the one with the highest badness score. That is normally the agent, not the small holder.
  - The holder can make this certain by raising the agent's `oom_score_adj`, for example to 500. Raising it needs no privilege.
- **How to detect an OOM:**
  - The `oom_kill` counter in the unit's `memory.events`. The file emits inotify "modified" events.
  - Unit `Result=oom-kill`, and the `OOMKills` property since v259.
  - The holder records `errored: oom` and keeps the last screen.
- **systemd-oomd also kills.** Fedora enables it with `ManagedOOMMemoryPressure=kill` at 80 % for 20 s on user slices, and Ubuntu Desktop at 50 %. oomd kills a whole cgroup, holder included, so simstim must read the unit's result to explain a vanished session. Ubuntu Server defaults are UNVERIFIED.
- **Default caps:** MemoryHigh 6 GiB and MemoryMax 8 GiB, but never more than about 40 % of host RAM. They are per-session settings and can be raised live. Agents can leak to 10 to 14 GiB, so a cap below that is the point.

### 6.4 cgroup v2 without systemd, and containers

- **Detect v2:** `stat -fc %T /sys/fs/cgroup` gives `cgroup2fs`. The process's own path is the `0::/…` line in `/proc/self/cgroup`.
- **Use a delegated subtree when you can write to it:**
  1. Move the holder to a leaf first. The kernel's "no internal processes" rule requires this.
  2. Turn on `+memory +pids` in `cgroup.subtree_control`.
  3. `mkdir s-<id>`, then write `memory.high`, `memory.max`, `memory.swap.max`, and `memory.oom.group=0`.
  4. Start the agent there, ideally with `clone3(CLONE_INTO_CGROUP)`.
  5. To kill the whole session, write `1` to `cgroup.kill`.
- **Docker:** on cgroup v2 it uses a private cgroup namespace and mounts cgroupfs read-only, unless the container is `--privileged` or Engine 28+ runs with `--security-opt writable-cgroups=true`. Podman, Kubernetes and Codespaces are UNVERIFIED.
- **cgroup v1:** systemd v256+ refuses to boot on it, and v258 removed support. Unprivileged caps are impossible on v1.
- **rlimits are the wrong tool:**
  - **RLIMIT_AS** breaks V8 and Node, which reserve large virtual ranges (nodejs/node#25933, still open). Claude Code, Codex's npm shim, Gemini CLI and opencode all run on JS runtimes.
  - **RLIMIT_DATA** is per process, not per tree.
  - **RLIMIT_RSS** does nothing on Linux since 2.4.30.
- **Fallback everywhere:** a watchdog in the holder that sums RSS, or PSS from `/proc/<pid>/smaps_rollup`, over the session's process tree every few seconds. It sends SIGTERM above the cap and SIGKILL after a grace period.
- **Init in containers:** in a container where simstim is PID 1, run under `--init` (tini) or reap zombies. Holders should set `PR_SET_CHILD_SUBREAPER` so that orphaned grandchildren come back to them.

### 6.5 Hosts without systemd

- **macOS memory:**
  - There are no cgroups.
  - XNU enforces RLIMIT_AS and RLIMIT_DATA, which breaks JS runtimes, and ignores RLIMIT_RSS.
  - launchd's `ResidentSetSize` is only a hint.
  - Jetsam limits are private and need root.
  - So use the watchdog: sum `proc_pid_rusage(RUSAGE_INFO_V4).ri_phys_footprint` over the tree, using `proc_listchildpids`.
- **macOS launchd:**
  - `gui/<uid>` needs a GUI login. `user/<uid>` "may exist independently of a logged-in user".
  - Install a LaunchAgent for an optional broker with `launchctl bootstrap user/$(id -u) ~/Library/LaunchAgents/sh.simstim.host.plist`.
  - Set `ProcessType=Interactive`. Otherwise launchd throttles CPU and I/O.
  - launchd SIGTERMs a job's process group when the job exits, so holders must `setsid`.
  - Starting at boot without a login needs a root LaunchDaemon with `UserName`.
  - Whether `setsid` holders started over SSH survive logout, and whether agents started over SSH can reach the login Keychain, are UNVERIFIED. Test both before supporting macOS hosts.
- **WSL2:** turn on systemd with `[boot] systemd=true` in `/etc/wsl.conf`. The VM shuts down when idle (`instanceIdleTimeout` 15 s and `vmIdleTimeout` 60 s by default; `-1` disables), so set them in `.wslconfig`. Whether background processes count as activity is UNVERIFIED.
- **Alpine and OpenRC:** user services became non-experimental in OpenRC 0.62. There is no linger equivalent, so starting at boot needs a system service (UNVERIFIED).

### 6.6 Detection ladder at host setup

1. If `systemctl --user is-system-running` works, linger is on (`loginctl show-user $USER -p Linger --value`), and `memory` is in the user manager's `cgroup.controllers`: use **transient services per session** with memory caps.
2. Else, if `/sys/fs/cgroup` is cgroup2 and a subtree is writable: use **manual cgroups**.
3. Else (no linger, macOS, locked containers): use **setsid holders** like zmx, sockets outside `/run/user`, and the **RSS watchdog**. Warn the user that sessions may die at logout or reboot.

## 7. v2: respawn after a reboot

The processes do not survive a reboot. What can survive is the conversation, which each agent stores on disk or on its server. Respawn means: start the agent again, in the same directory, with its resume command. A turn that was running when the host went down is not resumed. The user must prompt again.

### 7.1 Resume commands

Checked on 14–15 October 2026 against source or official docs.

| Agent (version read) | Continue latest in this directory | Resume a given session | Start with an ID we choose, or learn the ID | Where sessions live |
| --- | --- | --- | --- | --- |
| **Claude Code** (docs, v2.1.28x+) | `claude --continue` (`-c`) | `claude --resume <uuid\|name\|/abs/path.jsonl>` (`-r`). Searches this project and its worktrees, then every project (v2.1.223+). | `claude --session-id <uuid>` (must be a UUID), plus `-n <name>`. **The ID changes on `/clear`**, so also capture it from the `SessionStart` hook (`source` = startup, resume, clear, compact or fork). | `~/.claude/projects/<cwd with non-alphanumerics as ->/<id>.jsonl`, kept 30 days (`cleanupPeriodDays`) |
| **Codex CLI** (0.162.1) | `codex resume --last`. Filters by cwd; `--all` drops the filter. | `codex resume <uuid\|name>` | No flag to choose the ID. Learn it from the `SessionStart` hook (hooks must be trusted, or use `--dangerously-bypass-hook-trust`), or scan the newest rollout file for the cwd. | `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-<ts>-<uuid>.jsonl` plus a SQLite state DB |
| **opencode** (v1.19.0 stable; `main` 2.0.29) | `opencode [dir] -c` | `opencode -s <ses_…>`. `--fork` is optional. | v1: none. On `main` (unreleased), `-s` also *creates* a missing session, and the ID must start with `ses`, so `-s ses_<simstim id>` is deterministic. | `~/.local/share/opencode/opencode.db` (SQLite) |
| **Gemini CLI** (v0.64.0) | `gemini --resume` (a bare flag means latest) | `gemini --resume <id\|index>` | `gemini --session-id <id>` (`[A-Za-z0-9_-]`). Cannot be combined with `--resume`. | `~/.gemini/tmp/<project>/chats/session-<ts>-<id8>.jsonl` |
| **Amp** (closed source; docs and CLI help) | `amp threads continue --last` ("last thread for this terminal and mode in this checkout") | `amp threads continue T-<uuid>` (a thread URL also works) | `amp threads new` prints a new thread's URL. Then continue it by ID. | On Amp's servers (`ampcode.com/threads/T-…`). Any local cache is UNVERIFIED. |
| **Cursor CLI** (binary `agent`, formerly `cursor-agent`) | `agent resume` or `agent --continue` (= `--resume=-1`). Whether it is scoped to the cwd is UNVERIFIED. | `agent --resume <chatId>` | `agent create-chat` prints a new chat ID. Then use `--resume <id>`. | `~/.cursor/chats/` (layout undocumented, UNVERIFIED) |
| **Aider** (0.86.x) | `aider --restore-chat-history` (reads `<git root>/.aider.chat.history.md`) | No session IDs. Give each session its own `--chat-history-file` and `--input-history-file`. `/save <file>` and `--load <file>` restore the file set. | Same: choose the file paths | Files in the git root, or the paths you give |
| **Goose** (v1.54.0) | `goose session -r`. It picks the latest session **globally, not by cwd**. | `goose session -r --session-id <id>` or `-r -n <name>` | `goose session -n <name>`, but names are not unique. If the cwd differs from the session's directory, Goose **asks interactively**, so `cd` first. | `~/.local/share/goose/sessions/sessions.db` (SQLite) |

**Two agents run a shared background daemon by default, which would escape the session:**

- **Codex.** `daemon_auto_start` is a Stable feature, on by default (`codex-rs/features/src/lib.rs`). Its daemon keeps the environment it started with and can auto-update and restart. Launch `codex --no-daemon`, so the work stays in the session's PTY, cgroup and environment.
- **opencode** v2 (unreleased). Use `--standalone` for the same reason.

**How to pin each agent's session ID, best first:**

1. **Choose the ID at launch:** Claude `--session-id`, Gemini `--session-id`, opencode v2 `-s ses_…`.
2. **Create first, then launch with the ID:** Amp `threads new`, Cursor `create-chat`.
3. **Capture it with a hook** that calls `simstim-host hook`: Claude and Codex `SessionStart`. This also follows `/clear`.
4. **Scan the agent's store** after launch, filtered by cwd and start time.
5. **Fall back to "continue latest in this directory".** This is safe only when one session uses the directory, which a worktree per task gives us.

### 7.2 What the per-session manifest should store

One JSON file per session, at `~/.local/state/simstim/sessions/<id>/manifest.json`, written atomically (write to a temp file, then rename).

- **Forward compatibility:** a reader must keep fields it does not know. Claude Code's own supervisor follows the same rule for its `state.json`.
- **Secrets:** store the names of secrets, never their values.

```json
{
  "schema": 1,
  "id": "01JAB3…", "name": "auth-refactor", "labels": {},
  "created_at": "…", "updated_at": "…", "last_activity_at": "…", "last_attached_at": "…",
  "host_version": "0.3.1", "holder_binary": "~/.local/share/simstim/bin/0.3.1/simstim-host",
  "agent": {
    "kind": "claude", "version": "2.1.290",
    "argv": ["claude", "--session-id", "6f1c…", "-n", "auth-refactor"],
    "resume_argv": ["claude", "--resume", "{agent_session_id}"],
    "continue_argv": ["claude", "--continue"],
    "agent_session_id": "9b2e…", "id_source": "hook",
    "id_history": [{"id": "6f1c…", "source": "startup", "at": "…"},
                   {"id": "9b2e…", "source": "clear", "at": "…"}],
    "transcript_path": "…"
  },
  "cwd": "~/.local/share/simstim/worktrees/app/auth-refactor-01jab3",
  "git": { "repo_root": "~/src/app", "worktree": "…", "branch": "simstim/auth-refactor",
           "base_ref": "origin/main", "base_sha": "…", "created_by_simstim": true, "locked": true },
  "env": { "allow": ["PATH", "HOME", "LANG"], "set": {"COLORTERM": "truecolor", "TERM_PROGRAM": "ghostty"},
           "secret_refs": ["ANTHROPIC_API_KEY"] },
  "term": { "TERM": "xterm-ghostty", "cols": 220, "rows": 60 },
  "limits": { "memory_high": "6G", "memory_max": "8G", "pids_max": 4096 },
  "runtime": { "holder_pid": 4242, "agent_pid": 4250, "unit": "simstim-s-01JAB3.service",
               "socket": "…/s/01JAB3.sock", "boot_id": "…", "started_at": "…",
               "respawn_count": 0, "last_respawn_at": null },
  "status": { "state": "working", "since": "…", "source": "osc7501", "viewed": false,
              "exit_code": null, "signal": null, "oom_killed": false, "reason": null },
  "restart": { "policy": "resume", "max_attempts": 3, "backoff_s": 10 },
  "snapshot_path": "…/01JAB3/screen.ghostsnp"
}
```

**How respawn works:**

- **Detect a reboot.** `runtime.boot_id` is `/proc/sys/kernel/random/boot_id` on Linux, or `sysctl kern.bootsessionuuid` on macOS. If it differs from the current boot and the holder is gone, the session was lost to a reboot.
- **Start at boot.** On a lingering systemd host, a user unit `simstimd.service` with `WantedBy=default.target` runs `simstim-host restore` at boot. On macOS, a LaunchAgent with `RunAtLoad` does the same, but only once the user's domain exists.
- **Rebuild each session.** For each such manifest with `restart.policy = resume`:
  1. Create a new holder in `cwd`.
  2. Run `resume_argv`, or `continue_argv` if the ID is unknown.
  3. Count `respawn_count`, with backoff and a maximum.
  4. Mark the session `restored`.
- **Old screen (optional).** If the holder saved a libghostty-vt snapshot on detach or when idle, print its scrollback above the resumed agent, so the user sees the old context. The snapshot format is not stable across Ghostty commits, so only load a snapshot written by the same pinned build.

## 8. v2: a git worktree per task

### 8.1 Git facts that matter

From git-scm.com/docs/git-worktree, plus a local test with git 2.50.1.

- **Create:** `git worktree add -b <branch> <path> <base>`. A branch checked out in one worktree is refused in another unless you pass `--force`.
- **Remove:** `git worktree remove <path>` works only when the worktree is clean, meaning no untracked files and no changes to tracked files.
  - Ignored files, such as `node_modules/` or `.env`, do **not** block removal. They are deleted silently (tested).
  - The branch survives. Delete it in a separate step.
- **Lock:** `git worktree lock --reason …` blocks `prune`, `move` and `remove`.
- **Prune:** `git worktree prune` cleans the admin data of worktrees that are gone. `gc` also prunes, after `gc.worktreePruneExpire`.
- **Repair:** `git worktree repair` fixes links after a move.
- **Submodules:** git's BUGS section says support "is incomplete" and multiple checkouts of a superproject are "NOT recommended". Agent of Empires runs `submodule update --init --recursive` after `add`, and `submodule deinit -f --all` before `remove`.
- **Relative paths:** `worktree.useRelativePaths=true` helps when a container mounts the worktree at another path. It turns on an extension that older git cannot read (git 2.48+, UNVERIFIED).

### 8.2 How other tools do it

| Tool | Location | Branch | Ignored files (`.env`) | Cleanup |
| --- | --- | --- | --- | --- |
| Claude Code `-w <name>` | `<repo>/.claude/worktrees/<name>` | `worktree-<name>` | `.worktreeinclude` (gitignore syntax; copies files that are both matched and ignored) | Removes a clean, unnamed worktree and its branch on exit. Otherwise asks. Locks while running. Sweeps old background worktrees but skips ones with work in them. |
| Codex `--worktree` | `$CODEX_HOME/worktrees/<4-hex>/<repo>` | detached HEAD | — | The CLI never cleans. The desktop app keeps 15. |
| Cursor | `~/.cursor/worktrees/<repo>/<name>` | — | Setup script in `.cursor/worktrees.json` | Keeps at most 25 per machine and may delete worktrees it did not create (scope UNVERIFIED). |
| Claude Squad | `~/.claude-squad/worktrees/<name>_<hex time>` | `<user>/<title>` | — | Pause: auto-commit, then remove the worktree but keep the branch. Delete: `remove -f` plus `branch -D`. |
| Agent of Empires | `../{repo-name}-worktrees/{branch}` (a template) | from the template | — | Locks every worktree. "Trash" moves it into `.aoe-trash/<id>`. Never deletes the branch by default. |
| agent-deck | sibling `repo-branch` (default) or `.worktrees/branch` | from the template | `.worktreeinclude` plus approved setup and teardown scripts | `worktree finish` merges, removes and deletes |
| ccmanager | `../{branch}` | — | `.worktreeinclude` | Can copy `~/.claude/projects/<src>` to the new path, because Claude's history is keyed by path |

`.worktreeinclude` is the de facto standard. Copy files; do not symlink them.

### 8.3 Shared caches

- **Go:** `GOMODCACHE` and `GOCACHE` are per user and safe to use at the same time. Nothing to do.
- **pnpm and uv:** global content-addressed stores. Each worktree still runs `pnpm install` or `uv sync`, but files are hard-linked or cloned from the store, so it is cheap.
  - This needs the store and the worktree on one filesystem. This is a reason to keep worktrees under `$XDG_DATA_HOME`, not on another mount.
- **Cargo:** a shared `CARGO_TARGET_DIR` serialises builds on its lock, and can rebuild when feature sets differ (UNVERIFIED).
  - Better: Cargo's `build.build-dir` with a per-workspace path template, or `sccache` as `RUSTC_WRAPPER`. sccache needs incremental builds off, and it does not cache bin, dylib or proc-macro crates.
- **Other collisions:**
  - `docker compose` takes its project name from the directory name, so worktrees with the same leaf name collide.
  - Dev server ports clash.
  - direnv needs `direnv allow` again for each new path.

### 8.4 Defaults for simstim

- **Path:** `$XDG_DATA_HOME/simstim/worktrees/<repo>/<task-slug>-<short id>`. The leaf is unique, so compose projects do not collide.
- **Branch:** `simstim/<task-slug>`. Record `base_ref` and `base_sha` in the manifest.
- **On create:**
  1. `git worktree add -b … <path> <base>`.
  2. `git worktree lock --reason "simstim <id>"`.
  3. Apply `.worktreeinclude`.
  4. Run an optional, user-approved setup script.
  5. Write a marker file, `.git/worktrees/<name>/simstim-id` or similar, so a sweep can tell simstim worktrees from others.
- **On session delete:** if the worktree is clean and has no commits beyond `base_sha` (or they are merged), unlock it, `git worktree remove`, `git branch -d`, then prune. Otherwise keep it and ask. Never `--force` without consent.
- **Sweep:** periodically remove orphans that have no manifest, but only ones that carry simstim's marker.
- **Resume interaction:** a worktree per task gives each agent its own cwd, so "continue latest in this directory" becomes safe for agents with no stable ID.

## 9. Recommended design

### 9.1 The short version

- **Choice:** option (c). There is one small **holder** process per session. There is **no central daemon** in v1.
- **What the holder owns:** the PTY, a socket, a raw ring buffer, the manifest and the status engine.
- **Shadow terminal:** libghostty-vt runs in a separate **child process**, so a VT bug cannot kill the agent.
- **What we take from zmx:** the design, its restore rules and its bug list. We do not depend on zmx at run time.
- **systemd hosts:** each session is a transient user service with a memory cap. Linger must be on.
- **Hosts without systemd:** the holder daemonises like zmx, and a watchdog enforces the cap.
- **Upgrades:** binaries are installed side by side, and a versioned handshake lets new clients drive old holders. Later, holders can re-exec in place.

### 9.2 Components on each host

One static binary, installed at `~/.local/share/simstim/bin/<version>/simstim-host`, with a `current` symlink. If the vt engine is written in Zig (option 1 below), it ships as a second small binary, `simstim-vt`, in the same folder.

| Subcommand | Role | Lives for |
| --- | --- | --- |
| `simstim-host new` | Builds the environment, creates the manifest, and starts a holder using the ladder in 6.6 | Seconds |
| `simstim-host holder <id>` (internal) | The PTY holder for one session | The session |
| `simstim-host vt <id>` or `simstim-vt` (internal) | The shadow terminal, a child of the holder | The session. Restarted on crash. |
| `simstim-host attach <id>` | Bridges SSH stdio to the holder socket with a framed protocol. Run with `ssh -T`; the Mac client does raw mode, the detach key and image paste. | One attach |
| `simstim-host ls --json` | Reads manifests and probes holder sockets in parallel (unlike zmx's serial probe) | Seconds |
| `simstim-host watch --json` | Streams status events from all holders as NDJSON, for the session browser. Stateless. | One client connection |
| `simstim-host hook <event>` | Called by agent hooks. Forwards the hook JSON to the holder named in `SIMSTIM_SESSION`. | Milliseconds |
| `simstim-host kill`, `rm` | Stops a session (kills the whole cgroup if there is one) and removes the manifest and worktree when safe | Seconds |
| `simstim-host doctor` | Runs the detection ladder (linger, cgroup, terminfo, agent binaries) and prints fixes | Seconds |
| `simstim-host restore`, `gc` (v2) | Respawn after reboot. Remove dead manifests, old binaries and orphan worktrees. | Run at boot by `simstimd.service` or a LaunchAgent, then exit |

**Language:** write the holder and the CLI in simstim's main language. Write the vt engine against libghostty-vt.

- **Option 1: Zig, on the native `ghostty-vt` module** (my first choice).
  - Its formatter covers both screens.
  - zmx's MIT-licensed restore code (`serializeTerminalState`) can be adapted directly.
  - Zig cross-compiles static binaries with no extra toolchain.
  - Zig must be installed to build libghostty-vt anyway.
- **Option 2: the project language**, through the Rust crate `libghostty-vt` 0.2.x or the Go module `go.mitchellh.com/libghostty`.
  - This keeps one language.
  - The cost: the C formatter covers only the active screen, and the Go module needs cgo.
- **Either way,** pin one Ghostty commit. Keep the holder-to-vt interface tiny: bytes in; resize; snapshot request; events out.

### 9.3 The holder

- **Spawn:**
  - `forkpty`, then exec the agent with the resolved environment (9.6).
  - Set `PR_SET_CHILD_SUBREAPER` on Linux.
  - Raise the agent's `oom_score_adj`.
  - Launch Codex with `--no-daemon`.
- **Pass-through:** PTY bytes go unchanged to each attached client, except for two rewrites:
  - zmx's `OSC 133;A` → `redraw=0` rewrite;
  - `OSC 7501;?`, which the holder answers itself (see "Queries" below).
- **Leader rule:** zmx's rule (last client to type real input leads; replies and mouse do not count), plus an explicit `take_size` message, so a GUI or a second device can claim the size without typing (zmx #267).
- **Re-attach:** ask the vt child for a snapshot, laid out at the new leader's size. Send it, then resize the PTY, then send SIGWINCH to the foreground process group even if the size did not change.
  - Replay on the **first** attach too, so early output is never lost (zmx #246, #264).
- **Detach:**
  - The client resets only what the session changed: pop the kitty flags, turn off the mouse and bracketed paste, reset the colours.
  - It never sends RIS, which in Ghostty wipes the scrollback.
- **Queries:**
  - With a client attached, queries pass through to the leader. Only the leader's replies reach the PTY.
  - With no client, the vt child answers through `GHOSTTY_TERMINAL_OPT_WRITE_PTY`, using a capability profile cached from the last Ghostty client: DA, XTVERSION, OSC 10/11 colours, the kitty keyboard query and size reports.
  - The holder always answers `OSC 7501;?` and strips it from the client stream, so Claude Code and Codex report status even with no client attached.
  - If the agent turned on mode 1004, the holder sends focus-out (`CSI O`) while no client is attached.
- **Back-pressure:**
  - Cap each client's output buffer, for example at 4 MiB.
  - When a client falls behind, drop its backlog and send a fresh snapshot when it drains. This fixes zmx's unbounded `write_buf`.
  - Never block reading the PTY.
- **Ring buffer:** keep the last 8 MiB of raw output. Use it to rebuild the vt child after a crash or a vt upgrade, and as the `history` source if the vt child is down.
- **Agent exit:** record the code, the signal and any OOM in the manifest.
  - Keep the holder and the final screen alive in state `exited`, until the user removes or respawns the session.
  - This fixes zmx's lost exit codes (#247).
- **Status:** the layers in 5.3.
  - L0 comes from `waitpid`, `tcgetpgrp` and `memory.events`.
  - L1 comes from OSC 7501 and hook messages on the socket.
  - L2 and L3 come from vt-child events and screen rules.
  - Status changes are written to the manifest and pushed to `watch` subscribers.

### 9.4 Socket protocol

- **Location:** one Unix socket per session, at `<runtime>/s/<id>.sock`, mode 0600.
  - `<runtime>` is `$XDG_RUNTIME_DIR/simstim` when linger is on.
  - Otherwise it is `/tmp/simstim-<uid>`, mode 0700 and owner checked. `/run/user` vanishes at the last logout without linger.
  - Keep IDs short. Socket paths are limited to about 104 bytes on macOS.
- **Frame:** `u32 little-endian length`, `u8 type`, payload.
- **Hello:** the first frame each way is `hello`, carrying `protocol` and `features`.
- **Compatibility:** types are append-only. Unknown types are ignored. Control frames are JSON. `input` and `output` frames are raw bytes.
- **Missing features:** if a feature is missing in an old holder, the client says "restart this session to enable X" and carries on.
- **Messages:**
  - `hello`, `attach{cols, rows, xpx, ypx, client_id, caps}`, `detach`, `take_size`
  - `input`, `resize`, `output`, `snapshot`
  - `info`, `subscribe`, `event`
  - `hook`, `set_meta`, `kill`

### 9.5 Lifecycle and limits

- **Start:** use the ladder in 6.6. On systemd it is one `systemd-run --user` transient service per session, in `simstim.slice`, with `OOMPolicy=continue`, `MemoryHigh`, `MemoryMax` and `MemorySwapMax=0`.
- **Default caps:** 6 GiB high and 8 GiB max, never above about 40 % of RAM. They can be raised live.
- **OOM:** read `memory.events` (inotify) and the unit result. Report `errored: oom` and keep the last screen.
- **Kill:** use `cgroup.kill` or stopping the unit when there is one. Otherwise send SIGHUP to the process group, then SIGKILL to the whole tree found under the subreaper.
- **Host setup turns linger on.** If it cannot, it warns that sessions die when the user's last login ends.

### 9.6 Environment and terminal identity

- **Resolve the environment.** Run the user's login shell once, as `$SHELL -lic 'env -0'` with a timeout and a marker, the way VS Code Remote resolves the shell environment. This finds agents installed through nvm, mise, bun or npm.
- **Add:**
  - `TERM=xterm-ghostty`, if setup installed Ghostty's terminfo with `infocmp -x xterm-ghostty | ssh host tic -x -`; else `xterm-256color`;
  - `COLORTERM=truecolor`;
  - `TERM_PROGRAM=ghostty` and `TERM_PROGRAM_VERSION`, from the client;
  - `SIMSTIM_SESSION` and `SIMSTIM_SOCK`.
- **SSH agent:** point `SSH_AUTH_SOCK` at a stable symlink that each attach updates. A running agent cannot have its environment changed, which is the problem zmx's `print-env` works around.

### 9.7 Upgrades

1. **Side-by-side install.** New sessions use `current`. Running holders keep their version, which is recorded in the manifest. `gc` removes a version when no holder uses it. cmux's `cmuxd-remote` does the same.
2. **Handshake.** New clients drive old holders through `hello` and feature flags. Never break a message type; add a new one.
3. **vt child.** It can be upgraded live: start the new binary and replay the ring buffer. This is the part most likely to need bug fixes, such as the Ghostty crash fixes zmx had to ship.
4. **In-place holder re-exec** (later). Serialise state to a memfd, clear `FD_CLOEXEC` on the PTY master and the socket, then `execve` the new holder. The PID, the parent link and `waitpid` all survive. This is the idea behind `systemctl daemon-reexec`.
   - Prefer this to `SCM_RIGHTS` handoff (herdr), which loses the parent link and so the exit codes.
   - Prefer it to systemd's fd store, which is Linux-only.

### 9.8 Build order

1. **Spike, about one week.**
   - Drive zmx from a prototype client over SSH into Ghostty, to check the end-to-end feel: Shift+Enter, OSC 52, OSC 9/777, scrollback and Cmd+F.
   - Record agent sessions (Claude Code, Codex, opencode) as byte fixtures.
   - Do not ship on zmx.
2. **v1:**
   - holder, vt child, `new`, `attach`, `ls`, `watch`, `hook`, `kill`;
   - transient-service caps on systemd and the watchdog elsewhere;
   - L0, L1 and L2 status;
   - image paste staging: the client uploads the file over the same SSH connection, the holder writes it under `sessions/<id>/paste/`, and the client pastes the path. herdr does this.
3. **Tests:** feed the recorded fixtures into a reference terminal and into snapshot-then-replay, then compare cell by cell, the way zmx's `expectScreensMatch` does. Run them on every Ghostty bump.
4. **v1.x:** L3 screen rules as data, a macOS host path, and in-place holder re-exec.
5. **v2:** `restore` at boot from manifests with resume commands, saved screen snapshots, and worktree per task (section 8).

### 9.9 Main risks

- **libghostty-vt has no stable API** and no release. Mitigation: pin a commit, keep a narrow wrapper, isolate it in a process, and run replay tests.
- **Agents change their signals often.** Spinner glyphs, hook names and flags all move. Mitigation: keep rules as data, and report what is detected, with its source, in the UI.
- **macOS and container hosts are less tested.** Keychain access over SSH, logout survival and cgroup write access are UNVERIFIED. Ship Linux VMs first.

## 10. UNVERIFIED items

These claims come from reading code or docs, or from estimates. I did not test them.

**zmx and libghostty-vt**

- zmx sockets in `$XDG_RUNTIME_DIR` vanishing at the last logout while the daemons live on. Each half is documented; I did not test them together.
- The memory use of one libghostty-vt shadow terminal with 10,000 lines of scrollback. My guess is tens of MB. Measure it.
- The native size of libghostty-vt on its own. My estimate is 1.5–2.5 MB.
- Where Ghostty's source will live after leaving GitHub.
- That the Ghostty app cannot load a libghostty snapshot.

**Agents**

- How Claude Code detects Ghostty over SSH, for notifications and OSC 9;4.
- Whether a stable Codex release emits OSC 7501.
- opencode's OSC output.
- Amp's hook semantics.
- Amp's local thread cache.
- Whether `agent resume` in Cursor CLI is scoped to the cwd, and how Cursor lays out its chat storage.
- The scope of Cursor's worktree sweep.
- What Claude Code does with `--session-id` when that UUID already exists.

**Screen-restore alternatives**

- That shpool-vterm restores kitty keyboard state. I believe it does not.
- How well alacritty_terminal handles grapheme clusters.

**Hosts**

- systemd-oomd defaults on Ubuntu Server.
- Whether cgroups are writable under Podman, Kubernetes and Codespaces.
- That Amazon Linux 2 uses cgroup v1.
- That RHEL 9 ships systemd 252.
- Which cloud images lack polkitd, which blocks self-service linger.
- macOS hosts: whether holders started over SSH with `setsid` survive logout, and whether agents started over SSH can reach the login Keychain.
- WSL2: whether background processes keep the VM awake.
- OpenRC: starting user services at boot.

**Git and build caches**

- The git version that added relative worktree paths (2.48?).
- Rebuilds with a shared `CARGO_TARGET_DIR`.
- sccache behaviour across different paths.

## Sources

**zmx**

- github.com/neurosnap/zmx at 2d23c0d. Files read: `README.md`, `CHANGELOG.md`, `src/ipc.zig`, `src/loop.zig`, `src/main.zig`, `src/util.zig`, `src/daemonize.zig`, `src/cfg.zig`, `src/label.zig` and `build.zig`.
- Issues #14, #76, #127, #182, #211, #215, #220, #221, #226, #246, #247, #249, #259, #264, #267 and #268.
- PRs #243, #265 and #271.
- Release binaries from zmx.sh/a/.

**Ghostty and libghostty-vt**

- github.com/ghostty-org/ghostty at ad97d331: `include/ghostty/vt.h`, `include/ghostty/vt/terminal.h`, `formatter.h`, `snapshot.h`, `src/terminal/Terminal.zig` (`fullReset`) and `src/terminal/Screen.zig` (`reset`).
- PRs #13394, #13524, #14560, #14670, #14711 and #12182.
- `.github/workflows/release-tip.yml`.
- Ghostty's zsh shell integration (the ssh wrapper).
- mitchellh.com posts "Libghostty Is Coming", "Ghostty Is Leaving GitHub" and the OSC 7501 post.

**Bindings and alternatives**

- github.com/mitchellh/go-libghostty
- github.com/Uzaaft/libghostty-rs and crates.io/crates/libghostty-vt
- herdr's `crates/ghostty-vt`, `scripts/vendor_libghostty_vt.py` and `src/server/handoff.rs`
- github.com/shell-pool/shpool (README, issues #46 and #377)
- crates.io: vt100, alacritty_terminal, termwiz and avt
- github.com/charmbracelet/x (vt) and github.com/vito/midterm

**Status detection**

- herdr 916fcf8: `src/detect/`, `src/pane/osc.rs`, CHANGELOG and docs
- agent-deck 8d0f1fe: `internal/tmux/tmux.go` and `cmd/agent-deck/hook_handler.go`
- Agent of Empires d3c5fb6: `src/tmux/detect/` and `src/agents.rs`
- Claude Squad ce1ffb4: `session/tmux/tmux.go`
- ccmanager 46fe720: `src/services/stateDetector/`
- cmux 9d01f95: `docs/agent-hooks.md` and `docs/codex-agent-detection-plan.md`
- Codex d292f72: `codex-rs/tui/src/terminal_program_status.rs`, `codex-rs/hooks/` and `codex-rs/features/src/lib.rs`
- Claude Code docs: terminal-config, hooks, settings, cli-reference, sessions and agent-view

**Host lifecycle**

- systemd source and man pages (October 2026): `systemd.resource-control`, `systemd-run`, `systemd.scope`, `systemd.service`, `logind.conf`, `loginctl`, `units/user@.service.in`, NEWS, and `src/core/scope.c` at v252
- man7.org pam_systemd(8) and getrlimit(2)
- Distro packaging: Debian, Ubuntu, Fedora and Arch systemd rules
- Kernel `Documentation/admin-guide/cgroup-v2.rst`
- nodejs/node#25933
- moby `daemon/oci_linux.go` and the Docker Engine 28 release notes
- Apple TN2083, XNU `bsd/kern/kern_resource.c`, launchd.plist(5) and launchctl(1)
- Microsoft WSL config docs
- OpenRC NEWS

**Resume and worktrees**

- Claude Code: code.claude.com/docs/en/cli-reference, sessions and worktrees
- openai/codex rust-v0.162.1
- anomalyco/opencode v1.19.0 and `main`
- google-gemini/gemini-cli v0.64.0
- ampcode.com/docs/threads
- cursor.com/docs/cli
- Aider-AI/aider `aider/args.py`
- block/goose v1.54.0
- git-scm.com/docs/git-worktree
- Worktree code and docs of Claude Squad, Agent of Empires, agent-deck and ccmanager
- doc.rust-lang.org/cargo/reference/config.html
