---
status: accepted
date: 2026-10-15
---

# 0009 A light launcher, and a session that owns the whole terminal

This record replaces the first UI design: the "one home screen" of the first [UX spec](../ux.md), which had hosts on the left, sessions on the right and a permanent live preview below. That design had no record of its own. The rest of the design stays as it was, including Bubble Tea v2 ([0001](0001-go-and-bubble-tea.md)), the detach key ([0006](0006-defaults.md)) and byte-exact pass-through.

## Context

The first home screen worked, but it looked and behaved like a GUI. Panes, a preview and every host's sessions all competed for attention, and the screen was busy even when you only wanted one session. The owner wants the tool to get out of the way: start or join a session in a few keys, then give full focus to the agent.

Amp's CLI sets the bar for a calm terminal UI. Amp itself removed its TUI sidebar in August 2026 and left multiplexing to the terminal. We watched the real Amp CLI cell by cell ([research](../research/amp-tui.md)). Almost everything it draws uses the terminal's palette: notched frames, yellow titles, blue keys, faint secondary text, and a yellow selection bar.

## Options

1. **Keep the panes and restyle them.** This is the least work, but the screen stays busy, and the preview still competes with the list.
2. **A launcher: one list per screen.** Show the hosts, then one host's sessions. Dialogs and the palette open over the list and close again, and the preview opens on a key. Once attached, the session has the whole terminal.
3. **No launcher: commands only** (`new`, `attach`, `flip`). This is the lightest of all, but it is hard to learn and slow for picking among many sessions.

## Decision

Option 2, in Amp's style:

- **One column and one list:** hosts, then Enter for that host's sessions. With one host, the launcher opens on its sessions. `flip` and the palette can still show every host's sessions in one list.
- **No permanent panes:** the preview is a dialog on Space. The shortcuts show inside the list's own box on `?`. Notices are short lines under the box.
- **Amp's look, from the terminal's palette:** a notched frame, a yellow bold title, hints at the bottom right, blue keys, faint secondary text, borders in the text colour, and a yellow selection bar with black text. Fixed RGB is kept for the brand gradient only, as Amp keeps it for its mode colours.
- **The attached session owns every row and column.** The client sets no window title of its own while attached, and moves the "upload failed" message from the bottom row to a system notification (OSC 777). The reconnect notice is the only thing it still draws, and only while the link is down.

## Consequences

- Good, because each screen has one job, and the common path takes few keys: Enter on a host, then Enter on a session (or `n`).
- Good, because the agent gets the whole terminal, and its own window title.
- Good, because the launcher matches any terminal theme. The colours are slots, so 256- and 16-colour terminals look the same as truecolour.
- Bad, because there is no longer one screen with every host's sessions at a glance. The hosts list shows counts and "needs you", and `flip` and system notifications cover the rest.
- Bad, because the tab no longer says which session it holds, unless the agent sets a title. Shells on the host usually do not.
