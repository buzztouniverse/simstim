---
status: accepted
date: 2026-10-15
---

# 0012 Our own look: a signal, not a copy of Amp

This record replaces the look in [0009](0009-light-launcher.md): "Amp's look, from the terminal's palette". The rest of 0009 stays: one list per screen, no panes, a preview only on a key, and the attached session owning the whole terminal.

## Context

0009 matched Amp's CLI closely: notched boxes, a yellow title and a yellow selection bar, blue keys. When the owner reviewed it, they asked for a launcher that is unique, with a new layout and more animation and sequences, so that it does not look obviously inspired by Amp. Amp is the baseline, not the goal. They also asked for:

- a quick-start line;
- a live activity line;
- search across every host;
- smoother movement.

## Options

1. Keep Amp's look and add motion to it.
2. A look of our own, built on the product's idea: a signal you tune into. Keep only Amp's good habits: palette colours, faint secondary text, a command palette.

## Decision

Option 2 ([UX](../ux-spec.md)):

- **No box round the list.** A header holds an animated braille logo, the name in the brand gradient and where you are. A rule sits under it, and another rule above the prompt line and the keys.
- **One accent, cyan**, for keys, titles, the selection marker and work in progress. The selected row gets a cyan marker and a soft tint, not a bar.
- **A trace on every row:** eight braille cells of the session's output over the last half minute.
- **A live activity line** under the list: what the selected session is doing now, read from its screen.
- **A prompt line** for `/`, which searches every host, and for `n`, which starts a session from one line with completion.
- **Sheets** that rise from the bottom for every dialog, instead of centred boxes.
- **Sequences:**
  - a boot sequence;
  - names that decode out of braille noise;
  - screens that slide, and a selection that glides;
  - a scan along the rule while busy;
  - a short jack-in sweep before an attach;
  - a welcome whose ribbon opens out.
- **Keys in lower case**, as typed, with wide gaps instead of dots. The spinner is our own braille scan.

## Consequences

- Good, because the launcher has its own identity, and the motion says something: change, work, a session that needs you.
- Good, because the common paths are shorter: a number, one line to start a session, one search for every host.
- Bad, because the jack-in holds keys for 0.17 s before an attach. With motion off there is no delay.
- Bad, because the activity line reads the selected session's screen about every 1.5 s. That is one small request to its host each time.
