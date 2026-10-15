# Amp's TUI, observed

- **Date:** 15 October 2026.
- **Version watched:** the Amp CLI `0.0.1792008042-gef0030` (released 14 October 2026), on the owner's Mac.
- **Purpose:** the reference for the launcher redesign ([decision 0009](../contributing/decisions/0009-light-launcher.md), [UX](../contributing/ux-spec.md)).

This note is clean-room. It describes what Amp's terminal UI looks like and how it behaves, in our own words. No Amp code, prompts, logos, fonts or screenshots are in this repo. The captured screens, cell dumps, images and video frames stay in `docs/research/private/` (git-ignored).

## How we looked

1. **The real CLI, in a terminal.**
   - We ran `amp` and `amp threads continue` in a pseudo-terminal behind a terminal emulator. The emulator reported the owner's Ghostty colours when Amp asked for them (OSC 4, 10 and 11).
   - At each step we saved the screen with every cell's text, colour and attributes. So the colours below are the exact SGR values Amp sends, not guesses from pixels.
   - We opened the welcome screen, the command palette (empty and filtered), the thread picker and its empty state, the shortcuts, the mode dial, the Ctrl+C chord, and the error exit with an unreachable server.
   - We sent no prompt, so no model ran and no thread was created.
2. **The public material.**
   - We read the docs and news posts listed at the end.
   - We sampled pixels from the official screenshots, after converting the ones that carry a monitor colour profile to sRGB, and from frames of the official demo videos (about ±15 per channel).
   - X did not load (HTTP 451), so nothing here comes from it.

## 1. Layout

- **The whole screen.** Amp takes the alternate screen. The transcript fills the top; one rounded prompt box spans the full width at the bottom, with no outer margin.
- **Notched frames.** Every box is a rounded frame (`╭╮╰╯─│`) with text set into its border:
  - the title at the top left: `╭─ Command Palette ───╮`;
  - a fact at the top right: `── medium ─╮` (the mode), or the cost and the mode;
  - hints or a location at the bottom right: `──── Esc close ─╯`, or `── ~/code/app (main) ─╯`;
  - a status at the bottom left of the prompt box: `╰ ≋ Connecting ───`.
  - Each label has one space on each side, and one `─` before the corner.
- **Padding.** One column inside each side border. Form dialogs add a blank row at the top and the bottom.
- **The welcome screen.**
  - A dotted orb, about 34 columns by 18 rows, on the left. Three to eight columns to its right is a block of text, centred on the orb's height.
  - The text is "Welcome to Amp" (bold, in the mode colour, "Amp" brighter), then one hint line: `ctrl+o for commands · ? for shortcuts`.
  - The prompt box sits under it all. In a narrow terminal the orb shrinks.
- **Dialogs.**
  - Dialogs are centred on the whole screen, and everything behind them is dimmed.
  - The command palette is 80 columns wide once the terminal has room for it, and the terminal width less 4 before that. The thread picker is the terminal width less 4.
  - Dialogs queue: they never stack.
- **The command palette.** Inside the frame, top to bottom:
  - a blank row;
  - `>` and a block cursor;
  - a blank row;
  - the rows.

  Each row is three columns:
  - the category, faint and right-aligned (`thread`, `amp`, `debug`);
  - two spaces, then the command in bold (`switch`, `new in orb`);
  - the key, right-aligned in bold blue (`Ctrl+X o`, `Ctrl+C Ctrl+C`).

  The palette has no hints in its bottom border.
- **The thread picker.** The same frame, filter line and blank row. When empty it shows a faint "No Threads", a blank row, and a hint line. A small "Thread Preview" box sits inside the picker. The bottom border carries the hints: `Opt+W/Ctrl+T filter by workspace · Opt+A show archived · Esc close`.
- **Docked popups.**
  - The mode dial is a 58-column box at the bottom right, just above the prompt box.
  - The Ctrl+C chord opens a 24-column box in the same place. Its title is `Ctrl+C then`, and its two rows are `Ctrl+C Quit` and `Esc cancel`.
- **Shortcuts.** `?` shows them inside the prompt box itself: two columns of key and words, then a `─` rule, then the input. No dialog opens.
- **Errors at start-up.** When Amp cannot start, it leaves the alternate screen and prints into the shell:
  - a bold red `Error:` and the message;
  - a blank line;
  - the next step, in blue.

## 2. Palette

Almost every colour Amp sends is a terminal palette slot or the terminal's default colour. So Amp takes on the user's theme. Neo (May 2026) removed custom themes, and screenshots in Gruvbox and in a One Dark–like theme show the same slots in different colours. Fixed RGB appears only in the mode colours and the gradients drawn from them.

| Element | What Amp sends | Owner's Ghostty theme | Amp's own screenshots (Gruvbox) |
| --- | --- | --- | --- |
| Text | default foreground | `#BFBDB6` | `#EADBB2` |
| Background | default background | `#0B0E14` | `#1E2021` |
| Secondary text: hint words, categories, paths, empty states | faint (SGR 2) | about `#656565`, half way to the background | `#867E6A`, the same half way |
| Borders | default foreground | `#BFBDB6` | `#EADBB2` |
| Titles in borders | slot 3, bold | `#F9AF4F` | `#D79A20` |
| Selection bar | background slot 3, text slot 0, the command bold | `#F9AF4F` behind `#1E232B` | `#D79A20` behind `#1E2021` |
| Keys | slot 4; bold in lists and popups, plain in borders | `#53BDFA` | `#468588` |
| Keys in an empty state | slot 4, faint | `#2F6587` (half way) | not seen |
| "Error:" | slot 1, bold | `#EA6C73` | not seen |
| The next step after an error | slot 4 | `#53BDFA` | not seen |
| Done (`✓`), user messages | slot 2 | `#7FD962` | `#99971A` |
| Text cursor | reverse video | `#BFBDB6` block | `#EADBB2` block |
| Mode colours | fixed RGB | low `#FFD700`, medium `#3DFFA6`, high `#3DD4FF`, ultra `#D8B3FF`, plugins `#14B8A6` | the same |
| Welcome title, medium mode | fixed RGB, bold | `#97B537` to `#B1CF3E` | the same idea, in each mode's colours |
| Orb, medium mode | fixed RGB gradient, lit from the top left | `#C2E042` to `#0A2A12` | `#66F6A7` to `#0A0666` in deep mode |
| Backdrop behind a dialog | faint on every cell, bold removed | half way to the background | measured half way |

Ghostty draws faint text at half opacity, so a faint colour lands half way between the colour and the background. That is what both lanes measured.

## 3. Glyphs

| Glyph | Use |
| --- | --- |
| `╭╮╰╯─│` | Every frame. Heavier lines are not used. |
| ` · ` | Between hints and between facts |
| `>` | The filter or prompt line |
| `∼ ≈ ≋` | The spinner before a status, a small wave |
| `⠁…⡀` braille | The spinner on a running tool row, and in the window title |
| `✓` green, `✕` or `×` red, `⋯` | Done, failed, queued or waiting |
| `▸` `▾` | Expand and collapse |
| `•` `·` | The dial's track: lit up to the chosen mode, unlit after it |
| `…` | Truncation, at the end of text or in the middle of a path (`private/tmp/…/amp-cwd/demo-project`) |
| `←→` `↑↓` | Arrow keys in hints |
| `● • · : .` | The orb, one glyph per cell |

## 4. Text weights

- **Bold:** titles, command names, the chosen mode, key names in the welcome hints, keys in lists and popups.
- **Faint:** hint words, categories, the local path in the prompt border, descriptions, empty-state text, and the `then` in the chord popup.
- **Plain:** borders, body text, keys in a border.
- **Italic:** the user's own messages, and asides such as "(exit code: 127)".
- **Underline:** links.

## 5. Spacing

- One column of padding inside every frame.
- In the palette: a blank row above the filter line and one below it, and two spaces between the category and the command.
- In the welcome text: the title, then the hint line straight under it in the version we watched. Older images show two blank rows between them.
- One blank row between blocks of the transcript.
- The shortcuts grid uses two columns, with the second about half way across.

## 6. Key flow

- **Ctrl+O** opens the palette. Typing filters it; ↑↓ move; Enter runs; Esc closes. Commands that cannot run stay listed as "(unavailable)", in faint bold.
- **?** shows or hides the shortcuts.
- **Ctrl+C** opens the chord popup. A second Ctrl+C quits, and Esc cancels. Ctrl+C Ctrl+N and Ctrl+C Ctrl+E are other chords on the same key.
- **Esc** closes a picker. If the picker was opened from the palette, Esc goes back to the palette.
- **Pickers:** ↑↓ move and Enter picks. Filter-as-you-type works when typing is the main job. Modifier keys (Opt+W, Opt+A) do the other jobs.
- **The dial:** ←→ turn it, Tab shows the list, Esc closes.
- **Ctrl+X** is a leader key, as in `Ctrl+X o`.
- **Rebinding:** every key can be changed in the settings.

## 7. Empty, loading and error states

- **Empty list:** a faint one-line statement ("No Threads"), a blank row, then a faint hint whose key is in the key colour.
- **Loading:** the spinner and a faint status in the prompt border ("Connecting", "Streaming"). Status changes cross-fade.
- **Unavailable command:** listed, in faint bold, with "(unavailable)".
- **Errors:** a bold red `Error:` and the message, then the next step in blue. In the transcript, a failed step shows a red `×` and its exit code in red.
- **Success:** a `✓` in green before the text.
- **Not seen anywhere:** an offline screen, the "?" panel in a public image, and a populated thread picker. This account had no threads.

## 8. What the launcher takes, and what it leaves

| Takes | Leaves, and why |
| --- | --- |
| The notched frame: title top left, a fact top right, hints bottom right | The prompt box: the launcher has nothing to type except a filter |
| Slot-based colour: yellow titles, blue keys, faint secondary text, borders in the text colour | The mode colours: we have one brand gradient of our own |
| The selection bar: yellow behind black, bold kept | The orb: we keep our own mark, in the same place beside the welcome text |
| The palette layout, "(unavailable)", no hints | The preview inside the picker: our preview opens on Space and closes again |
| Shortcuts shown inside the list's own box | Toasts at the top right: our notices are short lines under the list, like Amp's transcript lines |
| The Ctrl+C chord popup at the bottom right | The leader key: the launcher's keys are single letters |
| `Error:` in bold red, then the next step in blue | |
| The welcome layout: mark on the left, title and one hint line beside it | |
| The wave spinner and `✓ ✕` | |

## Sources

Read on 14 and 15 October 2026.

- **Docs:**
  - [CLI](https://ampcode.com/docs/cli)
  - [Keybindings](https://ampcode.com/docs/cli/keybindings)
  - [Remote control](https://ampcode.com/docs/cli/remote-control)
  - [Runners](https://ampcode.com/docs/cli/runners)
  - [Settings](https://ampcode.com/docs/cli/settings)
  - [The Dial](https://ampcode.com/docs/the-dial)
  - [Threads](https://ampcode.com/docs/threads)
- **News:**
  - [Amp, Rebuilt (Neo)](https://ampcode.com/news/neo)
  - [Drop the Neo](https://ampcode.com/news/drop-the-neo)
  - [Agents, Everywhere](https://ampcode.com/news/agents-everywhere)
  - [Agents in Orbs](https://ampcode.com/news/agents-in-orbs)
  - [Agents Anywhere](https://ampcode.com/news/agents-anywhere)
  - [The Dial](https://ampcode.com/news/the-dial)
  - [Build Your Own Dial](https://ampcode.com/news/build-your-own-dial)
  - [Steer, Don't Queue](https://ampcode.com/news/steer-dont-queue)
  - [Less Noise](https://ampcode.com/news/less-noise)
  - [So Long, TUI Sidebar](https://ampcode.com/news/so-long-tui-sidebar)
  - [Look Ma, No Flicker](https://ampcode.com/news/look-ma-no-flicker)
  - [Command Palette](https://ampcode.com/news/command-palette)
  - [CLI Tab Navigation](https://ampcode.com/news/cli-tab-navigation)
  - [Find Threads](https://ampcode.com/news/find-threads)
- **Official images on `static.ampcode.com`:**
  - `news/neo-ask-user-choice.png` (a dialog)
  - `docs/cli-command-palette.png` (the palette)
  - `news/dial-release.png`, `docs/the-dial-cli-dial.png` and `docs/the-dial-cli-mode-list.png` (the dial and the mode list)
  - `news/find-thread.png` (the transcript)
- **Official videos on `static.ampcode.com`:** the Neo introduction, getting started with the TUI, agents in orbs, steering and queueing, and runners. We pulled frames from them.
