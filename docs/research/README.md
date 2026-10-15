# Research

The evidence behind the design, gathered on 14 and 15 October 2026. Each note marks what it could not verify.

| Note | What it covers |
| --- | --- |
| [Amp UX and Orbs](amp-ux-and-orbs.md) | Amp's public docs, design history and Orbs, and what users say. Source of the UX patterns, from public material. |
| [Amp's TUI, observed](amp-tui.md) | The real Amp CLI watched in a terminal, cell by cell, plus its official images: layout, palette in hex, glyphs, spacing, keys, empty and error states. The reference for the launcher. |
| [Herdr](herdr.md) | The closest existing tool: architecture, fidelity limits, and what to borrow |
| [Host runtime](host-runtime.md) | zmx, libghostty-vt, the holder design, status detection, resume commands, the manifest |
| [Connectivity and providers](connectivity-and-providers.md) | System SSH, multiplexing, bootstrap, provider recipes, reconnect, security |
| [Client stack](client-stack.md) | Language and TUI framework, attach hand-off, input filter, clipboard, distribution |

A clean-room study of the Amp CLI binary, and the screens and images captured for [Amp's TUI, observed](amp-tui.md), are kept outside git (`private/`, ignored). They must never enter the history of this repo. Only what they show, described in our own words, appears in the notes and in [the UX spec](../ux.md).
