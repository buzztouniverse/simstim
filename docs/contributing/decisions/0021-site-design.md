---
status: proposed
date: 2026-10-15
---

# 0021 The site: a landing page in the launcher's look, docs kept on Hextra, and a three.js field that loads last

The evidence is in [the research note on websites for developer tools](../../research/website.md).

## Context

[0015](0015-docs-toolchain.md) chose Hugo with the Hextra theme. The site it built was Hextra as it comes: the docs index at the root, Hextra's colours and fonts, and no sidebar. Many links went to GitHub: a link from a page at the root of `docs/` (the index, the roadmap), and every link to a file outside `docs/`, such as `CONTRIBUTING.md` or `SECURITY.md`. 59 links in all.

At Gate 2, on 15 October 2026, the owner asked for more: match the launcher's look; a real landing page with detailed docs; pixel and ASCII art, three.js motion and interactions; navigation that stays on the site, not on GitHub; a look of our own.

The research found what the best sites share: the first screen shows the product; motion has a still twin and stops when unseen; the page stays light; the landing page and the docs share one domain, one header and one search. It also found two gaps no site fills: a visible pause button for a looping hero, and braille that lines up with text in a web font.

## Options

1. **Restyle Hextra with CSS only,** and write the landing page as Markdown with Hextra's hero shortcodes.
2. **Keep Hextra for what it does well, and replace the rest.** Keep the docs layout and the search. Replace the home page, the top bar, the footer and the look with our own templates and styles, and add a three.js piece.
3. **Replace Hextra** with our own templates and Pagefind for search.
4. **A separate landing site** in another tool, in front of the Hugo docs.

## Decision

Option 2. Option 1 cannot reach the launcher's look or a live product in the hero. Option 3 rebuilds search, the sidebar, the theme switch and the mobile menu, and adds Pagefind, a second tool. Option 4 means two headers and two searches, which the research shows readers notice.

### What we keep from Hextra

- **Search:** the FlexSearch dialog on `/` and Ctrl/Cmd+K, on every page, the landing page too. Its index loads when the dialog first opens.
- **The docs layout:** the sidebar tree, "on this page", breadcrumbs, previous and next, and the mobile menu. Every page now has the docs type, so the sidebar shows the whole tree.
- **Code blocks** with copy buttons, the theme switch and its no-flash script, and the skip link.

### What we replace

- **The landing page** (`site/layouts/landing.html`), at the root. The docs overview moves to `/docs/`; every other address stays the same.
- **The top bar and the footer,** our own markup on Hextra's class names, so its scripts still work. The top bar has the launcher's logo: its three braille cells, moving, and the name in the brand gradient. On a phone, the menu lists every section.
- **The look** (`site/assets/css/custom.css`), from [the UX spec](../ux-spec.md#4-visual-language):
  - dark: the reference theme's background `#0B0E14`, text `#C8C6BF`, accent `#90E1C6`, and the brand gradient `#0E7C7B` to `#00E5FF`;
  - light: a paper background `#F6F5F1`, text `#22262E`, accent `#0A6B6B`, and the light gradient `#045A5A` to `#0098B8`;
  - faint text brightened from the terminal's to 6.1:1 (dark) and 5.7:1 (light), so every text colour passes 4.5:1;
  - the selected page in the sidebar has the launcher's cyan marker and soft tint; rules instead of boxes; tables with mono heads.
- **Type,** self-hosted: JetBrains Mono for headings, labels, code and the terminal (cut to 20 KB); Inter at its text optical size for paragraphs (36 KB); and a braille font of our own (2 KB), drawn on the monospace cell, because no web font draws braille at the width of its letters.
- **The link hook.** Each page knows its path in the repository (`srcPath`, from `cmd/docgen`), and `data/pages.json` maps every hosted file to its page. A link between two Markdown files resolves to the page on the site wherever the page lives. The community files at the root become the Project section. Links to code and scripts still go to their source on GitHub, and each page ends with "View the source of this page". `scripts/check-site-links.py`, in `just docs-check`, fails if a built page links to GitHub for a page the site hosts.
- **An `llms.txt`** of every page, for coding agents.

### The landing page

1. **Hero:** the headline, one paragraph, the install line with a copy button, and two calls to action. Under them, the launcher, live: a copy that runs in the page with the `demo` command's sample data, built from the launcher's own rules (layout, glyphs, traces, the Game of Life glyph, decode, jack-in, search, the keys sheet). Without JavaScript, the page shows the real render, `docs/images/sessions.svg`.
2. **Why:** any agent, your own hosts, byte for byte, sleep and reboots, each card with a small moving braille screen; and six more features, each linked to its how-to.
3. **How it works:** the architecture as a text diagram, with packets on the ssh line; the five parts listed under it light up their part of the diagram.
4. **Quick start:** install, `host add`, `new`, each with a copy button.
5. **Agents:** from the agent table, by support level.
6. **How it compares:** the four things together, each alternative's gap and when to choose it, and where we are behind.
7. **Docs:** the four kinds of page, with their first pages.
8. **The end:** the welcome screen, its ribbon and the name decoding.

### three.js

- **Version:** three.js 0.186.1, released on 24 September 2026. Its two build files are kept unchanged in `site/assets/vendor/three/` with its MIT licence. `scripts/site-vendor.sh` fetches them and checks their SHA-256. Nothing loads from a CDN.
- **Build:** Hugo's own `js.Build` bundles the classes `field.js` uses into one minified, fingerprinted file: 527 KB, 133 KB with gzip. It is only on the landing page.
- **The piece:** a field of braille cells in perspective, with the welcome mark's three strands running through it as raised, brighter bands, and a pulse along them. One draw call of points; the vertex shader moves them, and each point is a braille glyph from an atlas the page draws itself, filling with dots as the signal rises. The pointer lifts the field; a click sends a ripple.
- **When it loads:** after the page has loaded and gone idle, and only if all hold: motion is on; the stage is on screen; the reader has not asked to save data; the screen is at least 720 px wide with a mouse or trackpad; WebGL runs on the GPU (`failIfMajorPerformanceCaveat`). Otherwise the braille ribbon stays, moving if motion is on.

### Motion

- **One setting** for the whole site, like the launcher's `ui.motion`: auto follows `prefers-reduced-motion`; the footer has auto, on and off, kept in this browser; a button under the launcher pauses and plays (WCAG 2.2.2).
- **A still twin for everything:** the welcome mark is text in the HTML; the launcher draws in full; with motion off, nothing moves, and three.js never loads.
- **Every loop stops** when its element is off screen or the tab is hidden, and the field is freed when the page goes away.
- **No content waits for motion:** nothing fades in on scroll.

### Measured

Built locally and served with gzip, as GitHub Pages serves it; Lighthouse 13.5.0 in Chrome, on 15 October 2026.

| Page | Transfer | Requests | Performance | Accessibility | Best practices | LCP | TBT | CLS |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Landing, phone (slow 4G, 4× slower CPU) | 149 KiB | 17 | 99 | 100 | 100 | 2.1 s | 0 ms | 0.001 |
| Landing, desktop | 282 KiB, 133 of it three.js | 18 | 100 | 100 | 100 | 0.4 s | 0 ms | 0.002 |
| A docs page, phone | 132 KiB | 14 | 99 | 100 | 100 | 2.0 s | 0 ms | 0 |

SEO scores 92 locally only because the canonical address is relative there; the deployed site has absolute addresses.

## Consequences

- Good, because the site looks like the product, shows it working in the first screen, and keeps readers on the site.
- Good, because search, the sidebar and the mobile menu stay Hextra's, maintained upstream.
- Good, because a link between pages can no longer leave the site by mistake: the check fails the build.
- Good, because the site makes no request to anyone else, and a phone never downloads three.js.
- Bad, because we own the top bar, the footer and the landing page. When Hextra changes its class names or scripts, they need care. Hextra is pinned, so that happens only when we update it.
- Bad, because the copy of the launcher in the page is a second implementation. It can drift from the real one. The README's images, made from the real launcher, are the reference; check the two side by side after a change to the launcher's look.
- Bad, because 2.1 MB of three.js source is in the repository, though only 133 KB reaches a reader.
- Bad, because `llms.txt` carries a build time, so two builds of one commit differ in that file.
- Revisit if three.js grows much larger, or if the field is the only reason to keep it: one shader in plain WebGL would be a few KB.
