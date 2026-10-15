# Herdr deep-dive

Research note for simstim. Written on 15 October 2026.

## About this note

- **Subject:** Herdr, <https://github.com/herdrdev/herdr>, docs at <https://herdr.dev/docs/>.
- **Snapshot:** commit `916fcf8de029c6cac965b50f3cdb9687cd7d1537` (14 October 2026), version `0.9.5`.
- **Method:** I read the docs and the source code. I treated the clone as untrusted data. I did **not** build or run Herdr. Every claim about behaviour comes from reading code, docs, or GitHub issues. Where I infer behaviour from code without a test or a doc that confirms it, I say so.
- **Citations:** `path:Lx-Ly` means that file and line range at the commit above. The permalink base is `https://github.com/herdrdev/herdr/blob/916fcf8de029c6cac965b50f3cdb9687cd7d1537/`.
- **Docs:** I read the unreleased drafts in `docs/next/website/src/content/docs/`. I compared them with the published `docs/versions/0.9.5/` tree. Only `connecting-machines.mdx` and `cli-reference.mdx` differ, and only in small details about retry clicks.
- **GitHub data** (stars, issues, discussions) comes from the GitHub API on 14 to 15 October 2026.

## Summary

Herdr is a Rust terminal workspace manager for coding agents. A background server owns the PTYs. Clients attach to it locally or over SSH. It is close to the simstim spec in scope:

- It connects to any SSH host, and it installs itself there.
- It shows several machines in one sidebar.
- Sessions survive a detach.
- It reconnects after sleep.
- It bridges local clipboard images to remote panes.
- It resumes 18 agents after a server restart.
- It has git worktrees per task.

It does **not** meet spec item 8 as written. Herdr is a multiplexer in the tmux mould. It emulates each pane with libghostty-vt on the server, then re-renders cells to the client. Its UI runs in the alternate screen. So:

- There is no native scrollback.
- Native Cmd+F cannot see history.
- OSC 777 and OSC 9 notifications from the agent are dropped.
- Visuals are a faithful re-render, not the agent's own bytes.

Shift+Enter and true colour do work. This is a design choice, not a bug. The docs say: "Herdr emulates the terminals in its panes" (`docs/next/website/src/content/docs/configuration.mdx`, section "Outer terminal window title").

**Verdict: (d).** Build simstim independently, around raw PTY passthrough. Borrow Herdr's remote bootstrap, SSH bridge, reconnect, image paste, resume table and detection ideas. If the owner can accept tmux-style fidelity, then **(a) is a real option: Herdr already meets the rest of the spec today.** Section 8 has the full mapping.

---

## 1. Architecture

### 1.1 Server, PTY ownership and VT library

- **One binary, two roles.** `herdr` starts or attaches to a background server process (`herdr server`). The server is "headless". It does not enter raw mode. It listens on two sockets and owns all PTYs (`src/server/headless.rs:L1-L15`):
  - `herdr.sock` carries the newline-delimited JSON API.
  - `herdr-client.sock` carries the binary client protocol.
- **Daemon start.** `spawn_server_daemon` re-executes itself as `herdr server` with stdio set to `/dev/null`. It then detaches with a platform helper (`src/server/autodetect.rs:L204-L242`). The server ignores `SIGHUP` on Unix (`src/platform/unix_common.rs:L325-L370`).
- **PTYs.** Herdr uses a vendored and patched `portable-pty 0.9.0` (MIT, from WezTerm). The patches are in `vendor/portable-pty.patches.md`, and the Unix PTY I/O actor is in `src/pty/actor/unix.rs`. Windows uses ConPTY with a pinned, hash-checked `conpty.dll`.
- **VT emulation.** The library is **libghostty-vt**, Ghostty's terminal core (MIT). It is vendored at upstream commit `44f2a44df7e8` (`vendor/libghostty-vt.vendor.json`). Herdr adds local patches, such as exposing the modifyOtherKeys mode (`vendor/libghostty-vt.patches.md`).
  - The Rust binding is `crates/ghostty-vt` (about 10k lines).
  - The build needs **Zig 0.16** (`crates/ghostty-vt/build.rs`, and the "Vendored libghostty-vt" section of `AGENTS.md`).
  - Its target list includes `aarch64-apple-ios`. An iOS client may be planned. UNVERIFIED.
- **Keyboard encoding.** Each pane has its own libghostty terminal. libghostty owns keyboard state and key encoding (`src/pane/kitty_keyboard.rs:L1-L4`).
- **Child environment.** Panes get the following (`src/pane.rs:L72-L76`, `src/pane.rs:L99-L136`):
  - `TERM=xterm-256color`, `COLORTERM=truecolor`, `TERM_PROGRAM=herdr`.
  - The host terminal handles are removed: `TMUX`, `KITTY_WINDOW_ID`, `WEZTERM_PANE` and others.
  - Agent nesting variables are also removed, such as `CLAUDECODE` and `CODEX_THREAD_ID` (`src/pane.rs:L184-L200`).
- **Render model.** The server "renders to a virtual ratatui Buffer in memory" and "streams frames to connected clients after each render" (`src/server/headless.rs:L8-L10`). Each client gets its own render baseline (`src/server/render_stream.rs:L14-L31`).
- **Direction.** Herdr is "migrating toward a server-owned runtime protocol with the TUI as one client" (`AGENTS.md`, section "Runtime/client boundary guardrail").

### 1.2 Client rendering and the remote protocol

**Framing.** Each message is `[u32 little-endian length][bincode 2 payload]`, using serde and the standard config (`src/protocol/wire.rs:L1656-L1727`). The limits are:

| Limit | Size | Where |
| --- | --- | --- |
| Normal frame | 2 MiB | `src/protocol/wire.rs:L24` |
| Graphics frame | 32 MiB | `src/protocol/wire.rs:L29` |
| Clipboard image | 16 MiB | `src/protocol/wire.rs:L32` |

**Two protocol layers.**

1. **Private numbered protocol**, `PROTOCOL_VERSION = 22` (`src/protocol/wire.rs:L20`). It is used for direct terminal attach, terminal observe and control, live handoff, and same-install tasks. The version check needs an **exact match**: older and newer clients are both rejected (`src/protocol/wire.rs:L1749-L1770`).
2. **Stable "endpoint generation 1"** for the client-rendered UI (`src/protocol/endpoint.rs:L1-L33`).
   - The client sends `EndpointControl{kind:"endpoint.hello.v1", data:<JSON>}`. The JSON lists the codecs it accepts: `shell.snapshot.v1`, `shell.surface.v1`, `shell.input.semantic.v1` and `shell.blob.v1`. It also sets optional flags for surface reuse, delta and scroll (`src/client/handshake.rs:L172-L232`).
   - The server answers with `endpoint.welcome.v1`. This carries its version, the chosen codecs, the API `methods` and the `capabilities` (`src/protocol/endpoint.rs:L94-L181`).
   - Client and server versions do not need to match. A missing method disables only that action. Frozen fixtures protect the wire shapes (`AGENTS.md`, section "Stable client endpoint contract").

**What the client receives.** The client never receives raw PTY bytes. There are two render encodings (`src/protocol/wire.rs:L41-L49`):

- **`SemanticFrame`**, used by the client-rendered shell. The server sends `PaneSurface(PaneSurfaceFrame)` and `PaneSurfacePatch`. These are grids of `CellData`: a grapheme, a packed fg and bg colour (RGB is kept as `0x02RRGGBB`), modifiers and hyperlinks (`src/protocol/wire.rs:L1268-L1308`, `src/protocol/wire.rs:L1516-L1570`). The **local** client draws the sidebar, tab bar and overlays with ratatui (`src/client/shell/render.rs`). It then blits the pane cells with a diff encoder. The diff encoder sends synchronised output (`?2026`), CUP and SGR (`src/protocol/render_ansi.rs:L1-L26`).
- **`TerminalAnsi`**, used by direct attach and terminal sessions. The server diffs cell frames into ANSI bytes. It sends them as `Terminal(TerminalFrame{seq,width,height,full,bytes})` (`src/protocol/wire.rs:L1326-L1338`).

**Input.** In the full UI, the client puts the host into kitty keyboard mode (`src/client/terminal_setup.rs:L730-L735`). It parses keys into semantic events (`ClientPaneInputEvent::Key{code, modifiers, kind, shifted_codepoint, physical_key_id, ...}`). The server then encodes them "for the child application's negotiated keyboard protocol" (`src/protocol/wire.rs:L156-L184`).

**Over SSH**, there is no special transport. It is an SSH stdio pipe:

```text
Ghostty <-> local `herdr client` (ratatui UI, alternate screen)
        <-> private local Unix socket (mode 0600)
        <-> ssh -C -T <host> 'printf marker; exec ~/.local/bin/herdr remote-client-bridge'
        <-> remote herdr-client.sock
        <-> remote `herdr server` (PTYs + one libghostty-vt per pane)
```

- `SshStdioBridge` binds the local socket. It runs one `ssh -T` per accepted connection, and it pumps bytes both ways in 16 KiB chunks (`src/remote/attach.rs:L2978-L3098`, `src/remote/attach.rs:L3241-L3415`).
- The remote command first prints a ready marker, `herdr-remote-output-ready:1`. The local side throws away everything before it, such as MOTD banners and shell noise (`src/remote/attach.rs:L554-L556`, `src/remote/attach.rs:L3441-L3482`).
- On the host, `remote-client-bridge` starts the server if it is not running. It checks that the endpoint generation is 1, then copies stdio to the client socket (`src/remote/host.rs:L1-L63`).
- The local client is launched as `herdr client`, with `HERDR_CLIENT_SOCKET_PATH` pointing at the forwarded socket (`src/remote/attach.rs:L3559-L3613`).
- Herdr also has a JSON-API bridge, `remote-api-bridge`. It lets `herdr --machine X <cmd>` run CLI commands remotely (`src/remote/attach.rs:L2926-L2942`).

### 1.3 Direct attach fidelity

`herdr agent attach <name>` and `herdr terminal attach <id>` attach one server-owned terminal to the current terminal, without the sidebar. Linux and macOS only.

- **Output** is re-rendered ANSI frames (`TerminalAnsi`), not raw PTY bytes. The docs say: "Direct attach streams the current rendered terminal state, then live ANSI frames" (`docs/next/website/src/content/docs/persistence-remote.mdx`, section "Direct terminal attach").
- **Alternate screen.** Direct attach uses the same terminal setup as the full UI, which calls `ratatui::init()` (`src/client/terminal_setup.rs:L40-L48`). That call enables raw mode and enters the alternate screen. So the outer terminal keeps no native scrollback. This is from code reading plus ratatui's documented `init()` behaviour; I did not run it.
- **Scrolling.** The mouse wheel and PageUp/PageDown scroll Herdr's server-side viewport (`ClientMessage::AttachScroll`, `src/client/attach.rs:L56-L140`). Typing jumps back to the bottom.
- **Prefix.** The prefix is hard-coded to `Ctrl+B`. `Ctrl+B q` detaches and `Ctrl+B Ctrl+B` sends a literal `Ctrl+B` (`src/client/attach.rs:L65-L140`).
- **Kitty keyboard protocol.** This mode is closer to passthrough. The server watches the pane's kitty flags and modifyOtherKeys level. It sends `DirectTerminalKeyboardProtocol{flags, modify_other_keys_level}` to the client, which pushes the same flags onto the outer terminal (`src/server/headless/render.rs:L182-L303`). The client then sends raw input bytes (`ClientMessage::Input{data}`). The server writes them to the PTY unchanged, except bracketed paste, which it re-wraps (`src/server/pane_input.rs:L187-L201`). So Shift+Enter should reach Claude Code as the outer terminal encoded it.
- **OSC 52.** It does **not** reach direct attach clients. Clipboard writes go only to the "foreground client" (`src/server/headless/notifications.rs:L334-L340`, `src/server/headless.rs:L1649-L1654`). This is open bug #4612, "OSC 52 clipboard writes from a pane never reach a `herdr terminal attach` client".
- **OSC 777 and OSC 9.** These do **not** reach the outer terminal in any mode. Pane OSC 0/2/9 bytes go to a passive tracker that feeds agent detection (`src/pane/osc.rs:L485-L530`). OSC 777 is ignored. This is open issue #3004, "OSC 777 notifications from a shell pane never reach the host terminal". I found no forwarding path for OSC 9 in the code. That OSC 9 is lost in practice is UNVERIFIED, because I did not run it.
- **Over SSH.** Direct attach is not forwarded through `--machine`: "interactive terminal attachment are not forwarded" (`docs/versions/0.9.5/.../cli-reference.mdx`, section "Saved SSH machines"). To attach directly to a remote agent you SSH in first. Then the local image bridge does not apply.
- **Scripting interface.** `herdr terminal session observe` and `herdr terminal session control` stream NDJSON `terminal.frame` records with base64 ANSI bytes. Control mode also reads JSON `terminal.input`, `terminal.resize`, `terminal.scroll`, `terminal.mouse` and `terminal.release` commands (`cli-reference.mdx`, section "Direct terminal attach").

### 1.4 Fidelity against simstim spec item 8

| simstim need | Herdr full UI (`herdr`, `--remote`, saved machines) | Herdr direct attach |
| --- | --- | --- |
| True colour | Yes. `COLORTERM=truecolor`, and RGB survives the wire (`src/protocol/wire.rs:L1516-L1570`). | Yes |
| The agent's real TUI, visually exact | The real program runs, but its output is re-rendered from libghostty-vt cells. Rendering bugs exist: 32 open `rendering` issues, for example #1178 (coloured underlines), #4808 (OSC 11 repaint) and #4600 (emoji background spill). | Re-rendered ANSI frames |
| Native scrollback | **No.** Alternate-screen TUI. Herdr keeps its own scrollback, 10 MB per pane by default (`src/config/model.rs:L1054-L1056`). | **No** (alternate screen) |
| Native Cmd+F | **No.** Copy mode has `/` search, and `prefix+e` opens the scrollback in `$EDITOR` (`src/config/model.rs:L410`). | **No** |
| Kitty keyboard (Shift+Enter) | Yes, by **re-encoding** through libghostty. This path has regressed often: #4401, #4835, #4985 (all closed), and #1844 (open, Grok Shift+Enter). | Yes, by mirroring the flags plus raw bytes |
| OSC 52 clipboard | Yes, for writes. The local client uses `pbcopy` or the native clipboard, or OSC 52 when it runs under SSH (`src/selection.rs:L337-L377`). | **No** (#4612) |
| OSC 9/777 from the agent | **No** (#3004). Instead, Herdr raises its own notifications from detected agent state. It writes OSC 9 for Ghostty, iTerm2 and WezTerm, and OSC 99 for kitty (`src/terminal_notify.rs:L11-L72`), or uses the OS notifier. | **No** (no forwarding path found) |
| OSC 0/2 title | Not passed through. Herdr writes its own title from a template that can include `{terminal_title}` (`configuration.mdx`). | UNVERIFIED |
| OSC 8 links | Rendered and clickable. But Claude Code stops emitting links because `TERM_PROGRAM=herdr` is not on its list (#4748, open, needs an upstream fix). | Same |
| Kitty graphics | Yes. Herdr re-hosts the images itself. | UNVERIFIED |

---

## 2. Remote

### 2.1 Setup flow, install and update, version handshake

**Entry points.**

| Command | What it does |
| --- | --- |
| `herdr --remote <ssh-target> [--session N] [--handoff] [--remote-keybindings server]` | One remote session in one local UI (`src/remote/attach.rs:L53-L106`). |
| `herdr machine add <ssh-target> [--label L] [--remote-session N]` | Saves a machine profile and shows it in the multi-machine sidebar (`src/remote/attach.rs:L153-L275`). |

**Targets.** A target is an SSH alias or an `ssh://user@host:port` URL. Herdr runs your normal OpenSSH and `Include`s your `~/.ssh/config` (`src/remote/attach.rs:L3140-L3182`). So `ProxyJump`, keys, agents and 1Password-style helpers all work unchanged. A homelab VM behind a jump host is fine.

**Platform probe.** Herdr runs `uname`, and has a separate Windows probe. It supports Linux and macOS on x86_64 and aarch64, and Windows on x86_64 (`src/remote/attach.rs:L1704-L1830`).

**Binary discovery.** Herdr prefers a compatible `herdr` on the remote `PATH`. If there is none, it tries Homebrew, mise, Nix and direct-install paths (`src/remote/attach.rs:L1830-L1995`). Each candidate is checked with `herdr status client --json`. The JSON reports `version`, `protocol`, `endpoint_protocol_generation`, `endpoint_capabilities`, `remote_host_bridge` and `remote_bridge_idle_timeout` (`src/remote/attach.rs:L2392-L2430`).

**Install.** Herdr installs only after an interactive yes/no prompt (`src/remote/attach.rs:L2830-L2865`). Non-interactive runs fail instead (`persistence-remote.mdx`).

- On Linux and macOS it writes `~/.local/bin/herdr`. It streams the bytes over `ssh ... tee <dest>.tmp.$$`, then runs `chmod 755` and `mv` (`src/remote/attach.rs:L1274-L1319`).
- If the platforms match, it sends **the local binary itself**.
- Otherwise it downloads the matching release asset for the client's version from `https://herdr.dev/latest.json`, or from `preview.json`. It checks a **SHA-256** from the manifest and refuses an asset without one (`src/remote/attach.rs:L2661-L2790`).
- `HERDR_REMOTE_BINARY` overrides the source.
- It warns if `~/.local/bin` is not on the remote `PATH`.

**Running server.** A version difference alone never restarts a server. `remote_server_restart_reason` asks for a stop only if one of these is true (`src/remote/restart_policy.rs:L16-L37`):

- the endpoint generation is not 1;
- the server is not a detached daemon;
- for saved machines, it lacks `surface_interest` or `health_check`.

The stop prompt defaults to **No**, because a stop kills the panes. `--handoff` tries an experimental live handoff instead. It passes the PTY file descriptors to the new server (`src/server/handoff.rs`, `session-state.mdx` section "Live handoff").

**Session choice.** `machine add` lists the running remote sessions (`herdr session list --json`) and lets you pick one with Up/Down and Enter. One profile maps to one remote Herdr session.

**Saved data.** A profile holds only an opaque ID, a label, the SSH target, the session and an enabled flag. No credentials are stored (`connecting-machines.mdx`, section "Updates and saved data").

### 2.2 Reconnect after sleep, keepalives, control sockets, many machines

**Keepalives.** The temporary SSH config adds `ServerAliveInterval 15` and `ServerAliveCountMax 4` after your own settings, so your own settings win (`src/remote/attach.rs:L3140-L3182`). Every SSH call also gets `-C` for compression.

**Control socket.** On Linux and macOS, managed connections add `-S <private ctl path> -o ControlMaster=auto -o ControlPersist=600` (`src/remote/attach.rs:L1405-L1425`). Herdr never touches your own ControlMaster. Background connections add more options (`src/remote/attach.rs:L1387-L1403`):

```text
BatchMode=yes
NumberOfPasswordPrompts=0
StrictHostKeyChecking=yes
ConnectTimeout=10
```

So they never prompt. Instead they show `! auth` in the sidebar, and `herdr machine reconnect` finishes the authentication in a real terminal.

**Health checks.** The endpoint sends a ping every 5 s and gives up after 10 s (`src/client/endpoint/health.rs:L3-L6`). The kinds are `endpoint.health.ping.v1` and `endpoint.health.pong.v1` (`src/protocol/endpoint.rs:L27-L29`).

**Remote bridge reaping.** With `--idle-timeout-v1`, the remote bridge exits after 60 s with no traffic in either direction. On Linux it measures time with `CLOCK_BOOTTIME`, so time spent asleep counts (`src/platform/remote_bridge.rs:L1-L60`, `src/platform/linux.rs:L10`). The remote server and its panes keep running.

**Reconnect ladder.** For saved machines, the delay starts at 0.5 s and grows to at most 120 s (30 s for Local). A connection must stay healthy for 60 s before the next drop gets a fast retry. After a wake there is a 30 s recovery window with 2 s retries (`src/client/endpoint/supervisor.rs:L11-L17`). macOS wake events come from IOKit `IORegisterForSystemPower` (`src/platform/macos/power.rs:L12-L37`).

While disconnected, the sidebar shows cached state dimmed, and input to those panes is blocked (`connecting-machines.mdx`, section "Switch between machines").

**Plain `herdr --remote` is different.** It does not auto-reconnect. On a lost connection the client exits and prints "Run `herdr --remote <host>` to reattach" (`src/client/mod.rs:L2278-L2283`, `src/client/errors.rs:L39-L67`). This is from code reading; not tested.

**Many machines in one sidebar.**

- Each saved machine gets its own `SshStdioBridge` and local socket (`src/remote/saved.rs:L20-L57`).
- Only the **selected** machine streams pane surfaces. The others send only workspace and agent state and notifications (the `surface_interest` capability).
- Agent rows show a `machine` token.
- IDs such as `w1:p1` are scoped per server.
- One machine failing does not affect the others.

**SSH agent forwarding.** Panes get a stable `SSH_AUTH_SOCK` indirection file next to the API socket. It follows whichever live attachment supplies an agent (`src/platform/ssh_agent.rs`). Git signing keeps working across reconnects. This is a detail worth copying.

**Known problems.**

- **#5170 (p1, open)**, "Linux `herdr --remote`: dropping the local SSH client kills all remote pane processes". The cause is that the server lands in the SSH logind `session-N.scope`, not in `user@.service`.
- **#4920 (p1, open)**, "Server shuts down ~1s after last client detaches". The 0.9.4 changelog cites #4920 for the `SIGHUP` fix, but the issue is still open. Whether it is fully fixed is UNVERIFIED.
- **#4291**: a saved-machine bridge restarts a server that was stopped on purpose.
- **#5254**: a stale ControlMaster connection breaks `--remote`.

---

## 3. Image paste bridge

This works only when the client is local and the server is remote. That means `herdr --remote` or a saved machine. In the code: `endpoint_accepts_local_images` returns true only for a remote client process or a non-local endpoint (`src/client/clipboard_images.rs:L57-L64`). For a local session the paste passes through, and the agent reads the clipboard itself.

**1. Key interception.** The client checks each raw stdin chunk (`src/client/mod.rs:L866-L901` for the full UI, `src/client/mod.rs:L1010-L1037` for direct attach, `src/client/mod.rs:L1195-L1215` for Windows events). There are three triggers:

- an **empty bracketed paste**, `ESC[200~ESC[201~`. Many terminals send this when Cmd+V meets an image-only clipboard. Which terminals send it, Ghostty included, is UNVERIFIED;
- the key `keys.remote_image_paste`, **default `ctrl+v`** (`src/config/model.rs:L1148`, parsed in `src/config.rs:L157-L165`);
- a terminal **file drop**: a bracketed paste of one absolute path ending in png, jpg/jpeg, gif, webp or bmp (`src/client/clipboard_images.rs:L140-L254`).

If the clipboard holds no image, the key goes to the pane as normal (it logs "local clipboard has no image").

**2. Clipboard read on each OS.**

- **macOS** runs `osascript` with `the clipboard as «class PNGf»`. It writes the result to a temporary PNG and reads it back with a 16 MiB cap. Only PNG is supported (`src/platform/macos.rs:L700-L742`).
- **Linux** uses `wl-paste --type <mime>` under Wayland and `xclip -selection clipboard -t <mime> -o` under X11. It tries png, jpeg, gif, webp and bmp in order, and it checks magic bytes (`src/platform/linux.rs:L886-L960`). Under WSL it uses PowerShell `System.Windows.Forms.Clipboard.GetImage()`, saved as PNG.
- **Windows** reads the Win32 clipboard. It converts DIB to PNG and validates the PNG. Image dimensions are capped at 16,384 px (`src/platform/windows.rs:L2913`, `src/platform/windows/clipboard_image.rs`).

**3. Upload path.** The client sends `ClientMessage::ClipboardImage{target, extension, data}` (`src/protocol/wire.rs:L482-L490`, `src/client/clipboard_images.rs:L12-L43`). The target is one of `DirectTerminal`, `Pane(id)` or `Popup(id)`. The message goes through the same bincode-framed socket, then the SSH stdio bridge. Images over 16 MiB are dropped. Issue #4800 (open) reports a 9.7 MB drop that made the remote client exit with "endpoint write timed out".

**4. Staging on the host.** The server checks that this client can see the target pane (`src/server/headless.rs:L2130-L2167`). It writes the file to `$TMPDIR/herdr-clipboard-images-<euid>/client-<id>-clipboard-<nanos>-<n>.<ext>`, with the file at mode `0600` and the directory at `0700`, using `create_new` (`src/server/clipboard_image.rs:L13-L50`, `src/server/clipboard_image.rs:L74-L116`). It sanitises the extension; an unknown one becomes png.

**5. Path injection.** The absolute path is sent to the pane as a **paste**. It is wrapped as a bracketed paste if the app turned that on (`src/server/headless.rs:L1168-L1227`, `src/server/terminal_attach.rs:L1-L10`). So Claude Code and Codex see a pasted image path.

**6. Cleanup.**

- Each client records its staged files. They are deleted when that client disconnects (`src/server/headless.rs:L971`) and when the server shuts down (`src/server/headless/lifecycle.rs:L358-L360`).
- Files older than 24 h are deleted at the next staging (`src/server/clipboard_image.rs:L118-L135`).
- Risk: a client that disconnects soon after pasting, for example on sleep, deletes the file. If the agent reads the path late, the image is gone. Whether agents read the image at paste time is UNVERIFIED.

**Other open issues:** #4515 (Linux clipboard holds a file reference, not image data) and #4500 (an Alacritty drop arrives as a local path).

---

## 4. Persistence across restarts

### 4.1 What survives each case

| Case | What comes back |
| --- | --- |
| **Detach** | Everything. The processes keep running in the server. |
| **Server restart** | The layout snapshot: workspaces, tabs, the pane tree, each cwd, focus, labels and agent session references. |
| **Screen history** (opt-in) | `[experimental] pane_history = true` writes `session-history.json`. It is replayed only if the layout matches exactly. |
| **Live handoff** (opt-in) | The processes themselves, on update. |

Source: `session-state.mdx`, published at <https://herdr.dev/docs/session-state/>.

**How saving works.**

- The snapshot is `session.json` under `~/.config/herdr/`, or `sessions/<name>/` for a named session. The schema is in `src/persist/snapshot.rs:L16-L129`.
- Saves are debounced by 5 s (`src/app/mod.rs:L44`). Each save is written in full to a temporary file before it replaces the old one.
- Up to 48 rotating snapshots are kept in `session-snapshots/`, at most one every 15 min.
- A `session.json` that fails to load is copied to `session-backups/` first.

**Shutdown hooks.** On Linux, a logind delay inhibitor saves before host shutdown (`src/platform/linux/shutdown.rs`). macOS and Windows have their own equivalents (`src/platform/macos/shutdown.rs`, changelog 0.9.4).

**After a host reboot,** nothing restarts at boot. The server starts on the next client or bridge connection; `remote-client-bridge` starts it (`src/remote/host.rs:L41-L62`). Agents resume once a client has attached and sent its terminal size and colours (`session-state.mdx`). There is a 750 ms wait for the theme (`src/app/mod.rs:L43`), and restores are spaced 100 ms apart by default (`[session] startup_per_agent_delay_ms`).

### 4.2 How resume works

1. **Capture the session ID.** Official integrations install hooks or plugins in each agent's config.
   - For Claude Code, `herdr integration install claude` adds a `SessionStart` hook to Claude's settings JSON (`src/integration/claude_settings.rs:L65-L95`).
   - The hook script reads the hook JSON from stdin. It skips sub-agents, and it sends `pane.report_agent_session{pane_id, source:"herdr:claude", agent_session_id, agent_session_path, session_start_source}` over the API socket with a 0.5 s timeout. It needs `python3` (`src/integration/assets/claude/herdr-agent-state.sh:L1-L100`).
   - Codex is similar, and it also reads `CODEX_THREAD_ID` (`src/integration/assets/codex/herdr-agent-state.sh`).
   - A pane launched as `codex resume <id>` is captured from its argv (`src/agent_resume.rs:L135-L151`).
2. **Persist.** The pane snapshot stores `agent_session{source, agent, kind: id|path, value}`, or an `agent_resume{source, agent, argv}` that the agent reported itself (`src/persist/snapshot.rs:L100-L129`).
3. **Plan.** The `plan()` function has a fixed table keyed on `(source, agent, kind)`. Only official `herdr:*` sources count (`src/agent_resume.rs:L199-L350`).
4. **Launch.** Herdr spawns the user's normal shell in the saved cwd. It then **types** the shell-quoted command and a CR into it, so the user's `PATH`, aliases and rc files apply (`src/app/agent_resume.rs:L226-L317`, quoting at `src/app/agent_resume.rs:L356-L380`). A missing cwd leaves the pane with an error, not in `$HOME`. Duplicate session IDs are not resumed twice.
5. **Third-party agents.** An agent can report its own resume argv with `herdr pane report-agent ... -- <cmd> <args>`. The rules are: a plain command name, not a path; at most 64 arguments and 8 KiB; no apostrophes and no control characters (`src/agent_resume.rs:L58-L92`, `add-herdr-support.mdx`).

### 4.3 Resume commands

These are the exact per-agent commands, from `src/agent_resume.rs:L204-L310`. They match the published table on herdr.dev, which I checked on 15 October 2026.

| Agent | Integration source | Resume argv | Minimum integration version |
| --- | --- | --- | --- |
| Claude Code | `herdr:claude` | `claude --resume <id>` | 6 |
| Codex | `herdr:codex` | `codex resume <id>` | 5 |
| OpenCode | `herdr:opencode` | `opencode --session <id>` | 5 |
| GitHub Copilot CLI | `herdr:copilot` | `copilot --resume=<id>` | 2 |
| Cursor Agent CLI | `herdr:cursor` | `cursor-agent --resume <id>` (`cursor-agent.cmd` on Windows) | 1 |
| Devin CLI | `herdr:devin` | `devin --resume <id>` | 2 |
| Droid | `herdr:droid` | `droid --resume <id>` | 2 |
| Kimi Code CLI | `herdr:kimi` | `kimi --session <id>` | 3 |
| MastraCode | `herdr:mastracode` | `mastracode --thread <id>` | 1 |
| Pi | `herdr:pi` | `pi --session <path-or-id>` | 2 |
| OMP | `herdr:omp` | `omp --resume=<path-or-id>` | 3 |
| Hermes Agent | `herdr:hermes` | `hermes --resume <id>` | 2 |
| Qoder CLI | `herdr:qodercli` | `qodercli --resume <id>` | 2 |
| Qwen Code | `herdr:qwen` | `qwen --resume <id>` | 1 |
| Kilo Code CLI | `herdr:kilo` | `kilo --session <id>` | 1 |
| Antigravity CLI | `herdr:antigravity_cli` | `agy --conversation <id>` | 1 |
| Grok CLI | `herdr:grok` | `grok --resume <id>` | 2 |
| Letta Code | `herdr:letta` | `letta --conversation <id>`, or `letta --conversation default --agent <agent-id>` for `default:<agent-id>` | 1 |

**Not resumable in Herdr:** Gemini CLI, Amp, Kiro, Cline and Maki. Herdr detects their state only (`agents.mdx`). For simstim, the candidates below come from my own knowledge, not from Herdr. All are UNVERIFIED and must be checked against current agent versions:

- Gemini CLI: `gemini --resume <id|latest>`.
- Amp: `amp threads continue <thread-id>`.

### 4.4 Caveats for simstim v2

- Official resume keeps **only the session ID**. Original flags such as the model or the permission mode are dropped. Only self-reported argv carries them (`src/agent_resume.rs:L204-L310`).
- Typing into a shell is fragile:
  - #4574: a slow `direnv` or Nix shell startup broke the resume.
  - #5104: a zsh plugin busy-looped after restore.
  - #5130: `claude --resume` was typed into a pane that had been running `claude rc`.
- Session IDs have leaked between panes: #5261, #4814 and #4785.

---

## 5. Agent detection and status

Herdr combines four signals. A fifth path lets third-party agents report themselves.

1. **Process tree.** Herdr finds the PTY's foreground process group with `tcgetpgrp`. It collects that job's processes and reads their names from argv[0]: `sysctl(KERN_PROCARGS2)` on macOS and `/proc` on Linux (`src/platform/macos.rs:L432-L610`, `src/platform/linux.rs:L351`). It matches each name against an alias table, such as `claude|claude-code` or `cursor|cursor-agent` (`src/detect/mod.rs:L198-L231`). It scores wrappers, so `node .../bin/codex` still counts as Codex (`src/detect/mod.rs:L249-L290`). There are 24 known agents (`src/detect/mod.rs:L43-L96`).
   - For wrappers and sandboxes, set `HERDR_AGENT=<agent>`.
   - For runtimes without a foreground group, set `HERDR_PROCESS_DETECTION=child-groups` (`agents.mdx`). Open issue #5289 reports that this mode costs about one CPU core.
2. **Screen manifests.** Each agent has a TOML file of rules in `src/detect/manifests/*.toml`, about 1,800 lines in all. Each rule has a `state` (working, idle or blocked), a `priority` and a `region`. Example regions are `bottom_non_empty_lines(12)`, `osc_title` and `after_last_horizontal_rule`. Matching uses `regex`, `line_regex`, `contains`, and `any` or `not` gates. Example: `src/detect/manifests/claude.toml:L1-L60`, where the OSC title spinner means working and "esc to cancel" plus "enter to confirm" means blocked.
   - Rules read the **live bottom of the buffer**, never the user's scrolled viewport.
   - OSC 0/2 titles and OSC 9 progress are inputs (`src/detect/mod.rs:L305-L327`).
   - Herdr fetches signed-off manifest updates from herdr.dev at run time. Local overrides go in `~/.config/herdr/agent-detection/<agent>.toml` (`agents.mdx`, section "Detection manifests").
3. **PTY activity.** This is "the normal working authority" (`src/detect/mod.rs:L35-L38`). To go from working to idle needs 3 confirmations 100 ms apart, capped at 700 ms. There is a 3 s start-up grace (`src/pane/agent_detection.rs:L5-L13`).
4. **Hooks and integrations.**
   - **Full lifecycle state** comes from pi, omp, mastracode, opencode, kilo and kimi; their reports override the screen (`src/detect/mod.rs:L330-L340`).
   - **Session identity only** comes from hermes, qwen, letta and agy (`src/detect/mod.rs:L342-L351`).
   - Claude and Codex report the session only; the screen decides the state. Changelog 0.9.5 says Codex turn hooks were removed again.
5. **Self-reporting agents.** Crush, Kiro, Command Code and others call `herdr pane report-agent --state working|idle|blocked --seq N` themselves. Reports with an older `--seq` are ignored (`add-herdr-support.mdx`).

**States.** `blocked`, `working`, `done` (finished and not yet seen by this client), `idle` and `unknown`. They roll up from pane to tab to workspace. `herdr agent explain <pane>` shows which rule matched.

Blocked detection is strict by design. An unknown prompt shows as `idle`, not `blocked`. The docs warn that automation must not treat idle as approval. There are 39 open `agent-detection` issues.

---

## 6. UX

- **Model.** A session is a server namespace. It holds workspaces, one per project. Each workspace holds tabs, and each tab holds panes, which are real PTYs. Several clients can view different tabs. When two clients view the same tab, the last one to interact sets its size (`concepts.mdx`).
- **Keys.** The prefix is `Ctrl+B`, tmux style, and every binding can be changed, the prefix included (`keyboard.mdx`, `configuration.mdx`). The main keys are:

  | Key | Action |
  | --- | --- |
  | `prefix+c` | new tab |
  | `prefix+v` / `prefix+minus` | split |
  | `prefix+h/j/k/l` | move focus |
  | `prefix+w` | navigate mode |
  | `prefix+g` | go-to picker, with fuzzy search and filters by state |
  | `prefix+[` | copy mode, with vi-style movement and `/` search |
  | `prefix+z` | zoom |
  | `prefix+q` | detach |
  | `prefix+?` | help |

  Prefix-free chords are possible; the docs suggest the `ctrl+alt` family. Remote attach uses your **local** keybindings by default.
- **Mouse.** The UI is mouse-first: click, drag borders, right-click menus. Mouse capture is on by default (`[ui] mouse_capture`). There is a mobile layout for phone SSH clients.
- **Sidebar.** It has a Spaces (workspaces) panel and an Agents panel. Rows are configurable with tokens such as `state_icon`, `machine`, `workspace`, `branch`, `git_status`, `terminal_title` and custom `$tokens`. Tokens can have conditional colour rules. Machines can be collapsed.
- **Notifications.** `[ui.toast] delivery = "herdr" | "terminal" | "system" | "off"`.
  - `terminal` writes OSC 9 or OSC 99 to the outer terminal.
  - `system` uses `terminal-notifier` or `osascript` on macOS, and Notification Center on Windows.
  - mp3 sounds can be set per agent.
  - The bell is forwarded to the foreground client.
- **Config.** `~/.config/herdr/config.toml` (TOML), or `%APPDATA%\herdr\config.toml` on Windows. `herdr --default-config` prints the defaults. `herdr server reload-config` reloads it. Invalid values fall back to defaults with a warning. Presentation settings come from the **client**. Pane defaults, worktrees and integrations come from the **server**.
- **Theming.** Built-in themes include catppuccin, tokyo-night and gruvbox. A `terminal` theme follows the host palette. Light and dark themes can switch automatically. Single colours can be overridden.
- **Plugins.** A plugin is a `herdr-plugin.toml` manifest plus any executable. It can declare actions, event hooks, panes and link handlers. The plugin API is the whole CLI. There is no sandbox, and a marketplace is at <https://herdr.dev/plugins/> (`plugins.mdx`).
- **Socket API.** Newline-delimited JSON over a Unix socket, or a named pipe on Windows. `herdr api schema --json` prints the JSON Schema. The API can control workspaces, tabs and panes. It can read panes, wait for output or state, and subscribe to events (`socket-api.mdx`). An "agent skill" (`skills/herdr/SKILL.md`) teaches agents to drive Herdr.
- **Worktrees.** "New worktree" from a git workspace creates `<dir>/<repo>/<branch-slug>` and opens it as a grouped workspace. Deleting runs `git worktree remove`; branches are kept (`configuration.mdx`, section "Worktrees").

---

## 7. Quality

**Size.** About 301k lines of Rust in 398 files, plus the vendored libghostty-vt (Zig) and portable-pty.

**Tests.**

- About 4,280 test functions (`#[test]` and `#[tokio::test]`), most next to the code.
- 32 integration test files under `tests/` (about 13k lines). Examples: `remote_attach.rs`, `detach_reattach.rs`, `session_restore.rs`, `live_handoff.rs`, `multi_client.rs` and `host_shutdown.rs`.
- Frozen endpoint fixtures guard the wire protocol.
- Render-scale and release performance benchmarks (`just bench-render-scale`, `just bench-release-smoke`).
- CI runs on Linux, macOS and Windows. It also runs a ConPTY package job (`.github/workflows/ci.yml`).
- Agent detection is checked by hand with "live smoke tests". Unit tests cover the engine, not each agent's screen (`AGENTS.md`, section "Agent Detection Updates").

**Releases.**

- 62 stable tags, from `v0.1.0` on 27 March 2026 to `v0.9.5` on 14 October 2026, plus 38 `preview-*` tags.
- April had near-daily releases. Since July there has been one every one to two weeks.
- Still 0.x.
- Stable releases are promoted from a published preview, and only admins can push tags (`AGENTS.md`, section "Release Channels").
- Assets: Linux and macOS on x86_64 and aarch64, and a Windows x86_64 zip. Also available through Homebrew, mise and Nix.

**Activity.**

- 1,861 commits on `master` since 27 March. The first commit is titled "initial release", so earlier history is private.
- Monthly commits: 249, 356, 302, 302, 215 and 299 (April to September).

**Maintainers and bus factor.**

- `.github/MAINTAINERS` lists `ogulcancelik`, `JJLiebig` and `kangal-bot`.
- **Can (Ogulcan) Celik wrote 1,301 of 1,861 commits (70%).**
- JJ Liebig wrote 144, all since late July. That is 24% of commits since 1 August.
- Bots wrote the rest. One account is the maintainer-controlled issue agent.
- So the bus factor is about 2, and was 1 until July.
- Contact: an email address for enterprise and partnership questions (`README.md`).

**Governance.**

- The project calls itself opinionated, with one owner.
- **Unsolicited pull requests are closed automatically.** Only maintainers and the 76 accounts in `.github/APPROVED_CONTRIBUTORS` may open them (`CONTRIBUTING.md:L1-L40`).
- Issues are for reproducible bugs only. An issue agent triages them.
- Features go to GitHub Discussions (1,626 discussions).
- There is no public roadmap file. Direction shows in Discussions with the `planned` label, and in `AGENTS.md`, which describes moving to a server-owned runtime protocol. The endpoint contract mentions "Local, SSH, and Cloud connections" (`src/protocol/endpoint.rs:L5`). A Cloud product is UNVERIFIED.
- I found no discussion that asks for raw passthrough or native scrollback, apart from closed #1304 and #1250, which were closed as off-template feature requests.

**Issues.**

- 342 open and 2,193 closed. 581 pull requests merged.
- Open labels: `input` 49, `remote` 43, `agent-detection` 39, `rendering` 32, `sessions` 31, `performance` 17 and `kitty-protocol` 9.
- Three open p1 issues: #5170 (Linux remote kills panes), #4920 (server exits after detach) and #4762 (server SIGSEGV in libghostty `Screen.cursorReload` on resize; every pane dies).

**Fidelity and remote issues worth reading:** #3004, #4612, #4748, #1844, #4476, #5265, #4833, #4673, #5094, #4614, #4800, #5170 and #4920.

**Performance.**

- The design limits the cost of many panes. Hidden panes parse their output but do not render, and only the selected machine streams surfaces.
- Open issues still report CPU and latency problems:
  - #5211 and #5194: idle server CPU.
  - #4570: higher Ghostty CPU for a one-cell spinner through Herdr.
  - #4840: about 200 ms to attach.
  - #4572: about 100 ms for some socket API requests.

---

## 8. Verdict for simstim

### 8.1 The spec against Herdr today

| simstim spec | Herdr 0.9.5 | Notes |
| --- | --- | --- |
| 1. Install on any device | **Yes** | macOS and Linux (x86_64, aarch64); Windows x86_64 (beta). |
| 2. Connect a host, any provider, SSH jump host | **Yes** | Any OpenSSH target. Uses `~/.ssh/config`, so `ProxyJump` works. Installs itself after a prompt. |
| 3. Browse connected hosts | **Yes** | Machines in the sidebar; `herdr machine list`. |
| 4. Select a host, press Enter, start a session | **Mostly** | `prefix+w`, arrows and Enter pick a workspace. New tabs and panes come from keys or the mouse. One profile is one remote Herdr session. |
| 5. View, delete, select and use running sessions | **Yes** | Workspaces, tabs and panes; `herdr session list/stop/delete`; go-to picker. |
| 6. Keeps running on exit or detach | **Yes, by design** | But see the open p1 bugs #5170 (Linux `--remote`) and #4920. |
| 7. Any agent, full access | **Yes** | Panes are real PTYs on the host. 24 agents are recognised; others run as plain terminals. |
| 8. Native fidelity | **Partly** | True colour and Shift+Enter: yes. Native scrollback, native Cmd+F, OSC 9/777 passthrough and byte-exact visuals: **no**. OSC 52 works, but not in direct attach. |
| v1: reconnect after sleep | **Yes** for saved machines | Plain `--remote` exits and prints a reattach command. |
| v1: image paste from the local clipboard | **Yes** | macOS (PNG only), Linux (Wayland or X11), Windows. |
| v2: respawn after host reboot | **Yes**, for 18 agents | Not Gemini CLI or Amp. Starts on the next connection, not at boot. Drops the original flags. |
| v2: git worktree per task | **Yes** | Grouped worktree workspaces. |

### 8.2 The options

**(a) Use Herdr as it is.** This is the honest fallback. If the owner accepts tmux-style rendering, with Herdr's scrollback and copy-mode search instead of native scrollback and Cmd+F, Herdr meets the spec today. It also goes beyond it: multi-machine sidebar, detection, socket API, plugins and worktrees. It does not meet "no compromise" fidelity. That gap comes from the architecture, so no setting closes it.

**(b) Build on Herdr's server or protocol.** Not recommended.

- The stable protocol (endpoint generation 1) carries **cell grids**. A simstim client on top of it still could not give native scrollback.
- Direct attach and `terminal session control` are nearer to bytes, but still re-rendered. They ride the private numbered protocol, which needs an **exact** `PROTOCOL_VERSION` match (`src/protocol/wire.rs:L1749-L1770`).
- simstim would tie itself to a fast-moving 0.x codebase that one person mostly owns.

**(c) Contribute the missing features upstream.** Not practical.

- Unsolicited pull requests are closed automatically (`CONTRIBUTING.md`).
- Feature requests go to Discussions, with no promise of a response.
- A raw passthrough mode goes against Herdr's core design ("Herdr emulates the terminals in its panes"). It would also break its server-side detection and multi-client model.

**(d) Build simstim independently and borrow.** **Recommended.**

- simstim's main difference is native fidelity. That needs a different core:
  - Forward the PTY bytes **raw** to the outer terminal while attached, so the agent's own bytes, scrollback, Cmd+F, OSC 52, OSC 9/777 and OSC 8 reach Ghostty.
  - Keep a server-side libghostty-vt mirror only to repaint the screen on reattach.
- Herdr has solved most of the plumbing around that core. Apache-2.0 allows reuse, so simstim can borrow the parts below.
- In the meantime, use Herdr as the benchmark for simstim's v1 and v2 features.

### 8.3 Modules worth borrowing

Licence duties: keep the Apache-2.0 licence text, add attribution, and mark the files you change. Herdr has no `NOTICE` file. The vendored libghostty-vt and portable-pty are MIT.

| Herdr module | What simstim gets |
| --- | --- |
| `src/remote/attach.rs:L3140-L3182`, `src/remote/attach.rs:L1387-L1425` | A managed SSH config. It `Include`s the user's config first, adds fallback keepalives and `-C`, uses a private ControlMaster with `ControlPersist 600`, and runs background connections with BatchMode. |
| `src/remote/attach.rs:L554-L556`, `src/remote/attach.rs:L3441-L3482` | A ready marker that skips MOTD and banner noise on an SSH stdio stream. |
| `src/remote/attach.rs:L2978-L3415`, `src/remote/host.rs`, `src/remote/saved.rs` | The local socket to `ssh -T host 'exec bin bridge'` stdio bridge. The remote side starts the daemon on demand. |
| `src/platform/remote_bridge.rs`, `src/platform/linux.rs:L10` | A bridge idle watchdog with a clock that counts suspend (`CLOCK_BOOTTIME`). |
| `src/remote/attach.rs:L1274-L1319`, `src/remote/attach.rs:L1487-L1587`, `src/remote/attach.rs:L2661-L2790` | Remote install: probe the platform and `PATH` candidates, ask for consent, send the local binary or download a SHA-256-checked release, then `tee`, `chmod` and `mv`. |
| `src/remote/restart_policy.rs`, `src/protocol/endpoint.rs` | A version handshake built on capabilities: named codecs, advertised methods and optional features. The rule "never restart a server only because the version differs". |
| `src/protocol/wire.rs:L1656-L1727` | Simple length-prefixed bincode framing, with size limits and a trailing-bytes check. |
| `src/client/endpoint/supervisor.rs`, `src/client/endpoint/health.rs`, `src/platform/macos/power.rs` | The reconnect ladder, wake detection through IOKit, and a 5 s / 10 s ping-pong health check. |
| `src/client/clipboard_images.rs`, `src/platform/macos.rs:L700-L742`, `src/platform/linux.rs:L886-L960`, `src/platform/windows/clipboard_image.rs`, `src/server/clipboard_image.rs` | The whole image paste bridge: triggers, clipboard reads on each OS with format checks, private staging and cleanup. |
| `src/selection.rs:L337-L377`, `src/terminal_notify.rs` | When to use native clipboard tools and when to use OSC 52; builders for OSC 9 and OSC 99 notifications. |
| `src/agent_resume.rs`, `src/app/agent_resume.rs:L226-L380` | The resume table, argv checks, shell quoting, "type into a fresh login shell in the saved cwd", and spaced restores. |
| `src/integration/assets/claude/*.sh`, `src/integration/assets/codex/*.sh`, `src/integration/claude_settings.rs` | Hooks that capture session IDs, and a careful edit of agent settings files. |
| `src/detect/` and `src/detect/manifests/*.toml`, plus `src/platform/*::foreground_job` | Agent identity from the foreground process group, and state from screen rules. The manifests are data you can reuse with attribution. |
| `src/pane.rs:L99-L136`, `src/pane.rs:L184-L200` | Child environment cleanup. Remove host terminal handles and agent nesting variables. |
| `src/platform/ssh_agent.rs` | A stable `SSH_AUTH_SOCK` indirection that survives reconnects. |
| `src/persist/writer.rs`, `src/persist/snapshot.rs`, `src/platform/linux/shutdown.rs` | Saves that write the whole file before replacing the old one, rotating snapshots, backups on a failed load, and a logind delay inhibitor that saves before host shutdown. |
| `crates/ghostty-vt` | A Rust binding to libghostty-vt, for the reattach mirror. It needs Zig 0.16. Herdr's patch notes show what upstream lacks. |

### 8.4 Lessons from Herdr's bugs

- **Linux server lifetime.** Do not leave the remote daemon in the SSH session scope. Use `systemd-run --user` or a user unit with lingering on (#5170).
- **macOS server context.** A server started over SSH may lack the GUI and Keychain context. Agents that use the Keychain then fail (`troubleshooting.mdx`, #966, #4100).
- **Terminal identity changes agent behaviour.** `TERM_PROGRAM=herdr` turned off Claude Code's OSC 8 links (#4748). A passthrough design can keep `xterm-ghostty`, but only if the host has its terminfo installed. Herdr forces `xterm-256color` for that reason (`src/pane.rs:L100-L104`).
- **One input owner per session.** Use a takeover rule for writes and resizes, as Herdr does for direct attach. Allow several read-only observers.
- **Keyboard re-encoding is a lasting source of bugs.** 49 open `input` issues and 9 open `kitty-protocol` issues. This is a strong argument for raw passthrough.

---

## UNVERIFIED items

- Whether Ghostty, and which other terminals, send an empty bracketed paste on Cmd+V with an image-only clipboard.
- Whether Claude Code and Codex read a pasted image path at once. If not, Herdr's cleanup on disconnect can delete the file first.
- That OSC 9 desktop notifications from an agent are lost in practice. I found no forwarding path in the code, but I did not run it.
- OSC 0/2 and kitty graphics behaviour in direct attach.
- Whether #4920 (server exits after the last detach) is fully fixed in 0.9.4 and 0.9.5.
- Plain `herdr --remote` behaviour after sleep. From the code it exits with a reattach hint; I did not test it.
- The "Cloud" endpoint and the iOS build target as product plans.
- The Gemini CLI and Amp resume commands in section 4.3. They are not in Herdr.

## Sources

**Repository and docs**

- Repository: <https://github.com/herdrdev/herdr> (Apache-2.0, created 27 March 2026, about 44k stars).
- Docs: <https://herdr.dev/docs/>. The pages used:
  - <https://herdr.dev/docs/concepts/>
  - <https://herdr.dev/docs/persistence-remote/>
  - <https://herdr.dev/docs/connecting-machines/>
  - <https://herdr.dev/docs/session-state/>
  - <https://herdr.dev/docs/agents/>
  - <https://herdr.dev/docs/add-herdr-support/>
  - <https://herdr.dev/docs/keyboard/>
  - <https://herdr.dev/docs/configuration/>
  - <https://herdr.dev/docs/socket-api/>
  - <https://herdr.dev/docs/plugins/>
  - <https://herdr.dev/docs/troubleshooting/>
  - <https://herdr.dev/docs/cli-reference/>
- Contribution policy: `CONTRIBUTING.md` and `AGENTS.md` at the commit above.

**Issues cited** (all at `https://github.com/herdrdev/herdr/issues/<n>`)

- Fidelity: #1178, #1844, #3004, #4401, #4476, #4600, #4612, #4748, #4808, #4835, #4985, #5265.
- Remote and session lifetime: #4291, #4500, #4515, #4673, #4800, #4833, #4920, #5094, #5170, #5254.
- Restore and session IDs: #4574, #4785, #4814, #5104, #5130, #5261.
- Crashes and performance: #4570, #4572, #4762, #4840, #5194, #5211, #5289.
- macOS server context: #966, #4100.
- Closed feature requests: #1250, #1304.
