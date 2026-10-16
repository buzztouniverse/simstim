# Websites for developer tools: what works (research for the site)

- Date: 15 October 2026
- Lane: the public web. The landing pages and docs of terminal and developer tools that developers praise, their source where it is open, and well-known write-ups. Sections 1 to 10 were used for [decision 0021](../contributing/decisions/0021-site-design.md), the first design of this project's site. Sections 11 to 16 are a second round, after the owner's review of that site, and were used for [decision 0022](../contributing/decisions/0022-site-rebuild.md), which rebuilt it. Sections 17 to 21 are a third round, after the owner's review of the rebuilt site, and set the look and the hero of 0022 as it stands.
- Method: a script drove headless Chrome over the DevTools protocol, at desktop size (1440 × 1800) and phone size (390 × 844, at 2×, with touch), with the cache off. For each page it recorded every request and its size, took a screenshot, read the fonts, colours and contrast, pressed Tab to see the focus style, scrolled to the bottom, and counted animation frames at the top, at the bottom and under reduced motion. It also searched each page's own scripts for reduced-motion, IntersectionObserver and visibility handling. Where the source is public, I read it.
- Clean-room: this note describes what the sites do, in our own words. It copies no code, art or text, and the site copies nothing from them.
- Each weight is one run on one network, so treat the numbers as rough. Headless Chrome drew some WebGL heroes empty; I say where. Anything I could not confirm is marked **UNVERIFIED**.

## Summary

The pages developers praise share a few habits. The first screen shows the product: for a terminal tool, a terminal. There is one install command to copy, and at most two calls to action. Motion is an extra with a still twin: it stops under reduced motion, off screen and in a hidden tab. The page stays light, because weight comes from PNG screenshots and autoplay video, not from code. The landing page and the docs share one domain, one header and one search.

No site we studied offers a visible pause button for its looping hero, which WCAG 2.2.2 asks for. And no common web font draws braille at the width of its Latin letters, so braille art in page text needs a font of its own.

What we take, and how, is in [section 9](#9-what-we-take-for-our-site).

## 1. Sites read

All read on 15 October 2026.

| Site | Pages | Why |
| --- | --- | --- |
| Ghostty | [home](https://ghostty.org/), [docs](https://ghostty.org/docs), [config reference](https://ghostty.org/docs/config/reference), [website source](https://github.com/ghostty-org/website) (MIT) | A terminal; ASCII art hero |
| Charm | [home](https://charm.land/), [libraries](https://charm.land/libs/) | Terminal tools with a strong look |
| Warp | [home](https://www.warp.dev/), [docs](https://docs.warp.dev/) | A terminal |
| Zed | [home](https://zed.dev/), [docs](https://zed.dev/docs/) | An editor; keyboard-first |
| Bun | [home](https://bun.com/), [docs](https://bun.com/docs) | A CLI with a famous install line |
| Astro and Starlight | [Astro](https://astro.build/), [Astro docs](https://docs.astro.build/en/getting-started/), [Starlight](https://starlight.astro.build/), [Starlight docs](https://starlight.astro.build/getting-started/), [Starlight source](https://github.com/withastro/starlight) | Light pages; a docs theme |
| Linear | [home](https://linear.app/), [docs](https://linear.app/docs) | Restraint and type |
| Vercel | [home](https://vercel.com/), [docs](https://vercel.com/docs) | A three.js hero with a still twin |
| Tailwind CSS | [home](https://tailwindcss.com/), [docs](https://tailwindcss.com/docs/installation) | A hero that teaches |
| Raycast | [home](https://www.raycast.com/), [developer docs](https://developers.raycast.com/) | A polished launcher |
| Fly.io | [home](https://fly.io/), [docs](https://docs.fly.io/) | Illustration; an agent prompt to copy |
| Oxide | [home](https://oxide.computer/) | Many small ASCII animations, used to explain |
| ertdfgcvb Play | [playground](https://play.ertdfgcvb.xyz/), [play.core source](https://github.com/ertdfgcvb/play.core) (Apache-2.0) | The open engine behind Oxide's ASCII |
| OpenCode | [home](https://opencode.ai/) | A terminal agent's site, all monospace |
| terminal.shop | [home](https://www.terminal.shop/) | An SSH product whose page is its command |
| Nous Research | [home](https://nousresearch.com/) | One colour and WebGL dither |
| GitHub | [home](https://github.com/home), [how the globe was built](https://github.blog/engineering/engineering-principles/how-we-built-the-github-globe/) | WebGL done with care, written up |

Checked briefly: [Stripe](https://stripe.com/), [Poolside](https://poolside.ai/), [Amp](https://ampcode.com/), [Cursor](https://cursor.com/), [Factory](https://factory.ai/) and [Evervault](https://evervault.com/).

## 2. Page weight

Transfer size and number of requests, at first load and after scrolling to the bottom.

| Site | Desktop, first load | Desktop, scrolled | Phone, first load | Where the weight is |
| --- | --- | --- | --- | --- |
| Starlight | 134 KB / 22 | 138 KB / 24 | 115 KB / 14 | No web fonts; 24 KB of JS |
| terminal.shop | 218 KB / 18 | 218 KB / 18 | 324 KB / 22 | Fonts and a little JS |
| ertdfgcvb Play | 225 KB / 16 | 225 KB / 16 | – | A console font |
| Astro | 330 KB / 24 | 490 KB / 35 | 318 KB / 18 | 9 KB of JS; lazy images |
| Ghostty | 584 KB / 29 | 584 KB / 29 | 564 KB / 27 | A 285 KB sans font; frames in the HTML |
| Bun | 801 KB / 65 | 2.0 MB / 107 | 753 KB / 63 | A 727 KB video below the fold; analytics |
| Oxide | 1.0 MB / 92 | 1.8 MB / 100 | 1.2 MB / 90 | Photos, with tiny placeholders |
| Zed | 1.9 MB / 83 | 8.6 MB / 247 | 1.9 MB / 61 | A streamed demo video |
| Fly.io | 2.3 MB / 29 | 2.7 MB / 41 | 1.8 MB / 28 | A payment library and analytics on the landing page |
| Raycast | 2.6 MB / 128 | 41.7 MB / 307 | 4.2 MB / 123 | PNG screenshots of 4–5 MB each |
| Vercel | 2.7 MB / 132 | 3.1 MB / 141 | 1.1 MB / 77 | 1.8 MB of JS |
| Linear | 2.8 MB / 412 | 3.0 MB / 450 | 2.1 MB / 321 | 317 JS files |
| Tailwind CSS | 6.2 MB / 89 | 6.5 MB / 111 | 5.9 MB / 72 | PNGs, and 1.2 MB of fonts |
| Nous Research | 7.9 MB / 113 | 7.9 MB / 127 | 7.4 MB / 108 | Images and fonts |
| Charm | 8.5 MB / 47 | 12.4 MB / 48 | 30.8 MB / 47 | Autoplay video, fetched twice on the phone |
| OpenCode | 10.2 MB / 29 | 10.2 MB / 29 | 10.2 MB / 29 | One 9.9 MB video |
| Warp | 29.0 MB / 177 | 29.3 MB / 208 | 29.2 MB / 175 | 26 MB of PNG |

For scale, the whole of three.js r170 as one minified module is 692 KB, or 171 KB with gzip ([jsDelivr](https://cdn.jsdelivr.net/npm/three@0.170.0/build/three.module.min.js)). Hextra's search library, FlexSearch, is 15 KB with gzip.

The heavy pages are heavy because of pictures and video, not code.

## 3. Docs at a glance

| Docs | Engine | Search | Shortcut | Same domain and header as the landing page |
| --- | --- | --- | --- | --- |
| Ghostty | own | none: the config reference tells you to use the browser's find | – | yes |
| Bun | own | Pagefind | `/` | yes |
| Starlight | Starlight | Pagefind | Ctrl/Cmd+K | yes |
| Astro | Starlight | Algolia DocSearch | Ctrl/Cmd+K | no: a subdomain with another header |
| Warp | Starlight | Pagefind and an AI chat | Ctrl/Cmd+K | no: a subdomain |
| Zed | mdBook, own theme | built in | S | yes |
| Tailwind CSS | own | Algolia DocSearch | Ctrl/Cmd+K | yes |
| Vercel | own | own command menu and Ask AI | Ctrl/Cmd+K | yes |
| Linear | own | own | none shown | yes |
| Fly.io | Mintlify | Mintlify | Ctrl/Cmd+K | no: moved to another platform |
| Raycast | GitBook | GitBook | Ctrl/Cmd+K | no |
| Charm | none | none | – | no: libraries link to GitHub READMEs |
| Hextra (ours) | Hugo | FlexSearch; the index loads when the dialog first opens | Ctrl/Cmd+K and `/` | – |

An `llms.txt`, a plain list of the docs for coding agents, answered at Warp, Zed, Bun, Linear, Vercel, Raycast's developer docs, Fly.io and Amp. Ghostty, Astro, Starlight, Tailwind CSS and OpenCode had none.

## 4. Site by site

### 4.1 Ghostty

- **Hero.** The first screen is one terminal window on near-black. It shows the ghost mascot as animated ASCII art, 100 columns by 41 rows, in blue and light grey. Under it are one sentence and two buttons: Download and Documentation. The window's buttons match the visitor's platform.
- **How the art is made** ([the script](https://github.com/ghostty-org/website/blob/7962e91e190ff226be1a4983eb9368b7ddb4dff9/bin/video-to-terminal/video-to-terminal.sh) and [its README](https://github.com/ghostty-org/website/blob/7962e91e190ff226be1a4983eb9368b7ddb4dff9/bin/video-to-terminal/README.md)). A short 3D-rendered video goes through ffmpeg at 100 columns and 24 frames a second. Each frame is squashed by the font's aspect ratio, about 0.44. Each pixel is sorted by colour into blue, white or empty, and its brightness picks a character from a 10-step ramp. The result is 235 text frames that ship inside the page.
- **Motion** ([the component](https://github.com/ghostty-org/website/blob/7962e91e190ff226be1a4983eb9368b7ddb4dff9/src/components/animated-terminal/index.tsx)). A frame every 31 ms, about 32 a second. It starts only when the page is visible, pauses on window blur, and shows one still frame under reduced motion. The art is hidden from screen readers.
- **Docs.** A left tree, breadcrumbs, "on this page", an edit link and copy buttons. No search.
- **Weak points.** The server HTML's `<main>` is empty: without JavaScript the home page is blank. There is no `h1` and no skip link.
- **Works:** the product is a terminal, so the hero is a terminal, done with care. **Avoid:** a page drawn only by script, and docs without search.

### 4.2 Charm

- **Hero.** A full-screen violet WebGL scene with 3D blobs, a featured product card with a terminal video, and library cards, each with a mascot and a looping terminal recording.
- **Motion.** The WebGL background never stops, also at the bottom of the page and under reduced motion. Three videos autoplay.
- **Docs.** None on the site: libraries link to their GitHub READMEs.
- **Accessibility.** No `lang` attribute and no `<main>`. The first Tab lands on a link with no visible focus.
- **Works:** strong personality, and each library shown by its real terminal output. **Avoid:** tens of MB of video, motion that ignores the reader, and docs that leave the site.

### 4.3 Warp

- **Hero.** A light "engineering drawing" look: dotted grid paper, margin notes and captioned figures. "Figure 1" is a live grid of cells changing state, like a system diagram.
- **Motion.** The grid drops to about 1 frame a second off screen and to none under reduced motion. The site plays sounds, and its mute toggle is the first thing a keyboard user meets.
- **Weight.** 29 MB, 26 MB of it PNG.
- **Works:** a live diagram that explains the product, and motion that stops when unseen. **Avoid:** multi-MB PNGs, and sound.

### 4.4 Zed

- **Hero.** A centred headline, a Download button and a "Clone source" button, each with a one-letter key chip, then a large editor mock-up built in HTML. Zed served two different home pages across runs; **UNVERIFIED** whether that is a test.
- **Type.** A serif for headlines, a "duospace" face that reads like mono for body text, and its own mono for code.
- **Docs.** Same domain, mdBook with Zed's theme; press S to search.
- **Accessibility.** A strong focus ring. The key chips are only 2.2:1 contrast.
- **Works:** keyboard signals everywhere, and the product as real UI. **Avoid:** low-contrast key chips.

### 4.5 Bun

- **Hero.** On the left, a release eyebrow in mono, a heavy headline, then "Install Bun" with a one-liner, a `$` prompt in the accent colour, a copy button, OS tabs and a "then follow the quickstart" link. On the right, a benchmark with a "reproduce" link to its source.
- **Motion.** Scroll reveals exist only when motion is allowed, and the observer stops after each reveal. A terminal demo types, or shows all its lines at once under reduced motion. A video demo pauses below 30 % visibility. Prefetch is off under Save-Data, on slow connections and under `prefers-reduced-data`.
- **Docs.** Same header; Pagefind with `/`; "Copy page"; an `llms.txt`.
- **Accessibility.** A skip link first, and a 2 px focus ring in the accent colour.
- **Works:** the first screen shows the product working, and every claim links to the docs. **Avoid:** analytics weight.

### 4.6 Astro and Starlight

- **Astro.** A centred headline, a Get Started button and the create command in a mono pill with a copy icon. 330 KB and 9 KB of JS. Its docs are on a subdomain with another header.
- **Starlight.** The lightest site measured: 134 KB, no web fonts. An inline script in the head sets the theme before the first paint, so dark mode never flashes ([ThemeProvider](https://github.com/withastro/starlight/blob/b77aca9047de0f1f5213559def6ebd70c6158770/packages/starlight/src/components/ThemeProvider.astro)). Search lives in a native `<dialog>` and loads at idle time ([Search](https://github.com/withastro/starlight/blob/b77aca9047de0f1f5213559def6ebd70c6158770/packages/starlight/src/components/Search.astro)). Code blocks have terminal frames, synced package-manager tabs and copy buttons. A [skip link](https://github.com/withastro/starlight/blob/b77aca9047de0f1f5213559def6ebd70c6158770/packages/starlight/src/components/SkipLink.astro) goes to the title.
- **Works:** docs chrome that is light and accessible by default. **Avoid:** a generic hero nobody remembers.

### 4.7 Linear

- **Hero.** A large left-aligned headline, then a big product mock-up built in HTML on a soft glow.
- **Type.** One sans for everything, and a mono for small captions and code chips.
- **Weak points.** 412 requests. Some grey text is 3.1:1. 76 CSS animations still run under reduced motion, and one phone capture showed a blank first screen behind an entrance animation.
- **Works:** restraint, and the product as the hero. **Avoid:** grey text under 4.5:1, and content hidden behind entrance animations.

### 4.8 Vercel

- **Hero.** A three.js scene of a container on a crane hook. A still image of the same scene shows first.
- **Motion.** About 8 frames a second at the top and none off screen. **Under reduced motion it creates no WebGL context at all;** the still image stays.
- **Docs.** Same domain, a command menu on Ctrl/Cmd+K, Ask AI and an `llms.txt`.
- **Works:** heavy 3D with a matching still twin and a real stop. **Avoid:** 1.8 MB of JS on a marketing page.

### 4.9 Tailwind CSS

- **Hero.** A giant headline on grid paper. Faint mono labels above each element list the classes that style it: the page annotates itself with its own code. Search sits in the hero.
- **Motion.** A loop at 60 frames a second at the top, at the bottom and under reduced motion.
- **Works:** a hero that teaches the idea at a glance. **Avoid:** 1.2 MB of fonts, and a loop that never stops.

### 4.10 Raycast

- **Hero.** A large WebGL scene (empty in headless Chrome), a Download button with a Homebrew line, and a macOS desktop mock-up of the launcher.
- **Weak points.** 42 MB after scrolling, mostly PNG. No header, main, nav or footer landmarks. Developer docs on GitBook, on a subdomain.
- **Works:** a polished mock-up of the real app, and a path for developers. **Avoid:** huge PNGs and missing landmarks.

### 4.11 Fly.io

- **Hero.** A hand-drawn illustration, a serif headline, "Start for Free", and a second button that copies an onboarding prompt for your coding agent.
- **Weak points.** A payment library and analytics on the landing page. Docs moved to Mintlify on a subdomain. A cookie banner takes the first Tab stops.
- **Works:** a distinct voice, and a call to action for people who work with agents. **Avoid:** docs on another platform with another look.

### 4.12 Oxide

- **Hero.** A small terminal panel types a login and some commands. A leader line joins it to a photo of racks, captioned like a figure. ASCII shading glows behind.
- **Technique.** Each ASCII piece is a `<pre>` driven by play.core, hidden from screen readers, with an invisible `<pre>` of dots beside it that measures the cell size. Photos load as tiny blurred placeholders and fade into the full image.
- **Works:** ASCII that explains the system rather than decorates it, and mono labels that read like a datasheet. **Avoid:** a cookie banner over the hero.

### 4.13 ertdfgcvb Play and play.core

- **Technique** ([text renderer](https://github.com/ertdfgcvb/play.core/blob/b2d30ac618db573c228a4fa287ddedf48e642e79/src/core/textrenderer.js), [run loop](https://github.com/ertdfgcvb/play.core/blob/b2d30ac618db573c228a4fa287ddedf48e642e79/src/run.js)). A program is a function called once per cell, returning a character and a colour. The text renderer keeps the last frame and rebuilds only the rows that changed, one `innerHTML` string per row, with runs of the same style in one span. Cell size is measured once, so circles come out round in character space. The frame rate is capped at 30.
- **Works:** a simple model, and diffs by row. **Avoid:** taking its layout as a model; it is an art tool.

### 4.14 OpenCode, terminal.shop and Nous Research

- **OpenCode.** A pixel-block wordmark, an install box with tabs for six package managers, and features listed like a man page. One 9.9 MB video, which also plays under reduced motion.
- **terminal.shop.** The page is numbered lines like an editor buffer: the SSH command, then the server's host key so you can check it. 218 KB. No real headings or landmarks.
- **Nous Research.** One electric blue, WebGL dither and retro condensed type. It respects reduced motion and the visibility of the tab, but weighs 17 MB on a phone after scrolling.

### 4.15 Others, briefly

- **Stripe.** A WebGL ribbon that keeps about 140 frames a second under reduced motion.
- **Evervault.** Many small WebGL canvases, together about 650 frames a second, still about 520 under reduced motion.
- **Poolside.** A terminal mock-up hero; motion drops to about 2 frames a second under reduced motion.
- **Amp.** 575 requests; respects reduced motion.
- **Factory.** A 20 MB video. **Cursor.** 6.3 MB at first load.

## 5. Write-ups

- **Evil Martians, [a study of 100 developer-tool landing pages](https://evilmartians.com/chronicles/we-studied-100-devtool-landing-pages-here-is-what-actually-works-in-2025) (2025).** Centred heroes dominate. The hero visual depends on the product's maturity: real UI, code, or abstract art before launch. Use two calls to action, one product-specific and one for developers (docs, source). Lead features with problems, and end with a full-width call to action. Their [list of conversion killers](https://evilmartians.com/chronicles/how-to-kill-conversions-on-your-developer-tool-landing-page) adds: vague words, vague buttons and a crowded first screen.
- **GitHub, [the Copilot CLI's animated ASCII banner](https://github.blog/engineering/from-pixels-to-characters-the-engineering-behind-github-copilot-clis-animated-ascii-banner/) (2026).** Frames are plain text, and each character has a role (border, eyes), mapped to ANSI colours for light and dark themes. The banner is skipped for screen readers and never blocks start-up. Most of the work went into terminal differences and accessibility.
- **GitHub, [how the globe was built](https://github.blog/engineering/engineering-principles/how-we-built-the-github-globe/) (2020).** three.js, no textures: about 12,000 dots in one instanced draw call, antialiasing off, a light SVG placeholder that cross-fades into the canvas, and a frame-time watch that lowers the pixel ratio and the detail on slow machines.
- **Alex Harri, [ASCII characters are not pixels](https://alexharri.com/blog/ascii-rendering) (2026).** One brightness per cell blurs edges. Sampling the shape of each cell and picking the nearest glyph shape is much sharper, and fast enough on a GPU.
- **[WebGL ASCII](https://offscreencanvas.com/issues/webgl-ascii/), offscreencanvas.com.** Render a scene, cut it into cells, and in the fragment shader pick a glyph from an atlas drawn with canvas 2D, ordered from light to dense.
- **Maxime Heckel, [dithering and retro shading for the web](https://blog.maximeheckel.com/posts/the-art-of-dithering-and-retro-shading-web) (2024).** Ordered and blue-noise dithering as one post-processing pass in a shader.
- **three.js.** The [AsciiEffect example](https://raw.githubusercontent.com/mrdoob/three.js/dev/examples/jsm/effects/AsciiEffect.js) reads pixels back from the GPU and rewrites an HTML table every frame: easy, but too costly for a hero. The manual says to [dispose of](https://raw.githubusercontent.com/mrdoob/three.js/dev/manual/pages/how-to-dispose-of-objects.html) geometries, materials and textures by hand, to [cap the pixel ratio](https://raw.githubusercontent.com/mrdoob/three.js/dev/manual/pages/responsive.html), and to [render only when something changes](https://raw.githubusercontent.com/mrdoob/three.js/dev/manual/pages/rendering-on-demand.html). [`WebGLRenderer`](https://raw.githubusercontent.com/mrdoob/three.js/dev/docs/pages/WebGLRenderer.html.md) has `failIfMajorPerformanceCaveat`, which refuses software GL, so a slow machine can keep a still image.
- **web.dev, [prefers-reduced-motion](https://web.dev/articles/prefers-reduced-motion).** Build without motion first, and add it only when the reader has no preference. In script, listen for changes to the setting.
- **MDN, [Page Visibility](https://developer.mozilla.org/en-US/docs/Web/API/Page_Visibility_API) and [Intersection Observer](https://developer.mozilla.org/en-US/docs/Web/API/Intersection_Observer_API).** Browsers slow background tabs, but your own timers and loops need stopping by hand. Observers replace scroll handlers.
- **WCAG 2.2.**
  - [1.4.3 Contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html), AA: 4.5:1 for text, against what is really behind it.
  - [2.2.2 Pause, Stop, Hide](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html), A: motion that starts by itself, lasts more than 5 s and sits beside other content needs a way to pause it. A looping hero counts.
  - [2.3.3 Animation from Interactions](https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html), AAA: motion set off by pointer or scroll can be turned off.
  - [2.4.7 Focus Visible](https://www.w3.org/WAI/WCAG22/Understanding/focus-visible.html), AA, and [2.4.11 Focus Not Obscured](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum.html), AA: focus shows, and a sticky header never covers it.
- **Hextra**, our docs theme ([source](https://github.com/imfing/hextra)). Search is a native `<dialog>` on Ctrl/Cmd+K or `/`, and the index loads on first open. The base template has a skip link. Its asciinema shortcode loads the player from a CDN at `@latest`, so we do not use it.

## 6. Braille in web fonts

A braille hero drawn as page text needs every braille cell to be exactly as wide as a letter. A test page drew 20 braille cells and 20 Latin letters, at 16 px, in nine monospace fonts as Google Fonts serves them: JetBrains Mono, IBM Plex Mono, Geist Mono, Fira Code, Source Code Pro, Roboto Mono, Space Mono, Martian Mono and Noto Sans Mono.

- **Braille.** None of them drew braille. Every one fell back to the system's braille font, which was about 14 % wider: 218.8 px against 192 px. Braille columns would drift away from the text around them.
- **Blocks and box lines.** Most fell back to the system mono. Geist Mono and Fira Code drew them at the right width.

We checked the full JetBrains Mono 2.304 release file ourselves: it has the box lines, blocks, arrows and state glyphs the launcher uses, and no braille.

## 7. Patterns, ranked

1. **Show the product first.** For a terminal tool, the hero is a terminal: Ghostty, Oxide, Bun, OpenCode, Poolside, terminal.shop. Linear and Zed do the same for a GUI.
2. **One install command to copy,** with a link to the next step: Bun, Astro, OpenCode.
3. **Two calls to action at most:** Ghostty, Bun, Astro, Starlight.
4. **Motion with a still twin,** stopped under reduced motion, off screen and in a hidden tab: Vercel, Warp, Bun, GitHub, Ghostty. Not Charm, Tailwind CSS, Stripe or Evervault.
5. **A pause button for a looping hero.** None of the sites has one.
6. **A light page.** Under 1 MB is normal for the best: Starlight, Astro, Ghostty, Bun.
7. **One domain, one header, one search** for the landing page and the docs: Bun, Starlight, Zed, Vercel, Tailwind CSS, Linear.
8. **Keyboard first:** `/` and Ctrl/Cmd+K, key chips, copy buttons on every code block.
9. **Mono as a voice, not as body text:** mono for commands, labels and captions, and a readable face for paragraphs (Bun, Linear, Oxide, Zed).
10. **Pictures that explain:** Warp's live diagram, Oxide's figures, Tailwind's self-annotation.
11. **Evidence over claims:** Bun's "reproduce" link, terminal.shop's host key.
12. **Accessibility basics:** a skip link, visible focus, landmarks, art hidden from screen readers, and grey text at 4.5:1 or more.
13. **Dark and light,** following the system, with no flash: Starlight, Bun, Vercel.
14. **Content in the HTML,** with the art added on top; never an empty page without script.
15. **Docs for agents:** `llms.txt` and "copy page as Markdown".

## 8. Rendering a glyph field

| Way | Seen at | Good | Costs |
| --- | --- | --- | --- |
| Text frames made beforehand, swapped in a `<pre>` | Ghostty | Tiny at run time; crisp; one frame is the still | Fixed art; payload grows with frames |
| A live text grid, rebuilt by row | play.core, Oxide | Crisp; the font's own glyphs | Cost grows with cells and frames; braille falls back (section 6) |
| Canvas 2D, one `fillText` per cell | play.core | No page churn | Thousands of calls a frame |
| three.js `AsciiEffect` | three.js example | Turns any scene into text | A GPU read-back and a table rewrite every frame |
| One shader pass with a glyph atlas | offscreencanvas, Codrops | One draw call; all on the GPU | The atlas needs the font, or its own drawing |
| Points or instances, one per cell | the GitHub globe | Cells can move in 3D | More vertex work |

## 9. What we take for our site

[Decision 0021](../contributing/decisions/0021-site-design.md) has the choices and their reasons. In short:

- **The hero is the launcher,** live in the page with the `demo` command's sample data, built from the same rules as the real screens. Without script, the page shows the real render, `docs/images/sessions.svg`.
- **The three.js piece is a field of braille cells** with the welcome mark's three strands running through it. It is points in one draw call, each drawn as a braille glyph from an atlas the page draws itself, so no font is needed. It loads last, only with motion on, WebGL on the GPU (`failIfMajorPerformanceCaveat`), data saving off and the stage on screen. It stops off screen and in a hidden tab, and is freed when the page goes away.
- **A still twin everywhere:** the welcome mark as text in the HTML, and the launcher's real render.
- **A visible pause button** for all the page's motion, and a motion switch in the footer that works like the launcher's `ui.motion`.
- **A braille font of our own** (2 KB), drawn to the monospace face's cell, so braille lines up with text (section 6).
- **One domain, one header, one search:** the landing page uses the docs' header and Hextra's search. Every link between pages stays on the site, and a check enforces it.
- **Light:** self-hosted fonts cut to what the site uses, no images but the launcher's SVG, no analytics, no cookies, and no request to anyone else.
- **An `llms.txt`** for coding agents.
- **Not taken:** autoplay video, PNG screenshots, sound, scroll-triggered reveals of content, and anything that only works with script.

## 10. Not verified

- How Raycast, Zed and Charm draw their WebGL: their scenes were empty or minimal under software GL, and their sources are not public.
- Whether Zed's two home pages are a test or depend on the browser.
- The full Codrops article on shape-aware ASCII in three.js: only its listing was read.
- The weights are single runs. Our own site's numbers are in decision 0021.

## 11. The second round: why

The owner's review of the first site, on 15 October 2026, found problems in the navigation, a landing page heavy with text and short of white space, and too little explanation. [The audit](website-audit.md) measured each one. In short: 1,631 words in eight sections of cards on the landing page, cards wider than a phone's screen, one sidebar of nine groups for users and contributors together, and guides sorted by the alphabet.

Four research lanes followed, each with its own notes, sources and screenshots; each claim below was read on 15 October 2026. The notes themselves are kept outside the repository; this section keeps what they found, with the primary sources.

## 12. Docs frameworks

Each was built against our own `docs/` (118 pages) on an Apple M3 Pro. Versions are from the registries and release pages.

| | Starlight | Docusaurus | VitePress | Material for MkDocs | Zensical | Hugo with Hextra |
| --- | --- | --- | --- | --- | --- | --- |
| Version | 0.42.6, on Astro 7 | 3.10.2 | 1.6.4 (2.0 in alpha) | 9.7.7 | 0.0.70 | Hextra 0.13.0 |
| Search, local | Pagefind, in small pieces when used | none built in; a plugin loads 4 MB at once | MiniSearch, 220 KB with gzip at once | lunr, 988 KB at once | its own, 1 MB at once | FlexSearch, 871 KB at once |
| Versions | build each with its own base (works) | built in | build each with its own base | mike | a fork of mike | our own script |
| Relative `.md` links | not resolved | resolved | resolved; a README is not a folder's page | resolved | resolved; changes links that leave `docs/` | a render hook |
| Breadcrumbs | no | yes | no | yes | yes | yes |
| npm or Python packages | 270 | 1,174 | 126 | 29 | 11 | one binary |
| Build, about 120 pages | 2–5 s | 4–5 s | 3 s | 1 s | 1–1.5 s | 2–3 s |
| Requests to others | none | none | none | Google Fonts | Google Fonts | none |

What decided it:

- **Starlight** has local search that loads only when used ([site search](https://starlight.astro.build/guides/site-search/), [Pagefind](https://pagefind.app/docs/)), no requests to anyone else, a skip link and landmarks by default, overrides for every part of the page ([overriding components](https://starlight.astro.build/guides/overriding-components/)), i18n that keeps today's addresses ([i18n](https://starlight.astro.build/guides/i18n/)), and a moderate tree. Our plain Markdown built with no changes. It lacks `.md` links, breadcrumbs, `/` for search and a version menu; each is small to add.
- **Its risks:** it is 0.x, with five breaking minor releases between March and October 2026 ([changelog](https://github.com/withastro/starlight/blob/main/packages/starlight/CHANGELOG.md)); Astro's telemetry is on by default, in CI too, until `ASTRO_TELEMETRY_DISABLED=1` ([Astro telemetry](https://astro.build/telemetry/)); its landing-page template has no phone menu ([issue 977](https://github.com/withastro/starlight/issues/977)); and its Markdown turns `--` into a dash unless smart punctuation is off.
- **Docusaurus** has the most built in, but four times the packages, no local search of its own ([search resources](https://docusaurus.io/community/resources#search)), and plain Markdown only in a mode it calls experimental.
- **Material for MkDocs** reaches its end of life on 5 May 2027 ([getting started](https://squidfunk.github.io/mkdocs-material/getting-started/)). **Zensical**, its successor, is at 0.0.x ([repository](https://github.com/zensical/zensical)).
- **VitePress** 1.x depends on Vite 5, which no longer gets fixes ([Vite releases](https://vite.dev/releases)), and our docs broke its build twice: `<host>` in plain text and `{{ }}` in code are read as Vue.

## 13. Information architecture

Sixteen sites read: Ghostty, Bun, Deno, Astro, Tailwind CSS, Charm, Zed, Tauri, Biome, uv, mise, GoReleaser, Atuin, Fly.io and Tailscale, and our own.

- **Order by the reader's path.** All of them set the order by hand: install and a short first run, then guides, then concepts, then reference. None sorts guides by title ([uv](https://docs.astral.sh/uv/), [mise](https://mise.jdx.dev/), [Deno](https://docs.deno.com/runtime/)).
- **Tabs, each with its own sidebar,** once there is more than one kind of reader: Tauri, Deno, Bun, Fly.io and GoReleaser ([GoReleaser](https://goreleaser.com/), on Hextra).
- **Plain labels:** "Get started", "Guides", "Concepts", "Reference". No site uses "Explanation" as a label.
- **Long command lists** nest under their parent command, collapsed, with an overview of every command ([mise commands](https://mise.jdx.dev/cli/), [Fly.io](https://fly.io/docs/flyctl/)).
- **Decision records and research notes** are never in the user's path: they live on GitHub or in a separate project area.
- **Where-am-I cues:** breadcrumbs, the current item marked, "on this page", previous and next in the curated order.
- **Pages users look for:** a short concepts page in plain words ([uv concepts](https://docs.astral.sh/uv/concepts/)), and common questions ([Atuin FAQ](https://docs.atuin.sh/latest/faq/), [mise FAQ](https://mise.jdx.dev/faq.html)).

## 14. Landing pages: words and space

Twenty-three landing pages measured at 1440 × 900 and 400 × 860: words per screen, the hero, the gaps between sections, and the type sizes.

| Measure | Median of the 23 | Our first site |
| --- | --- | --- |
| Headline | 4 words | 13 words |
| Sub-headline | 17 words | 39 words |
| Words per screen | 106 | 161 |
| Gap above a section heading | about 155 px; the calm pages 210–284 px | 133 px |

- **The pages that read as calm** make one point per section: a heading that states it, one or two lines, then one picture, a short list or a command. Detail is one click away. Single-purpose CLI tools use about 20 to 950 words in all ([OpenCode](https://opencode.ai/), [Atuin](https://atuin.sh/), [Zellij](https://zellij.dev/), [Ghostty](https://ghostty.org/)).
- **The hero** is a short headline, one plain sentence, the install line and at most two actions, with the product in view on the first screen ([Bun](https://bun.com/), [Linear](https://linear.app/)).
- **Prose in boxes reads as stacks of grey lines.** Our first site had 23 text boxes, some inside others, a median of 34 words each, in columns of about 35 characters.
- **Copy:** say what only this tool does. A hosted agent now makes the "close the laptop" promise in its own words ([Amp](https://ampcode.com/)), so our first line says what is ours: any agent, on your own machines, in your own terminal.

## 15. Motion that explains

- **Examples that explain:** Electric's figure of a client catching up from an offset ([Electric Streams](https://electric.ax/streams)) is nearly our story, in about 4.4 KB with gzip; [Stripe Connect](https://stripe.com/connect) and [Restate](https://restate.dev/) draw their figures from one clock with named steps, and keep the meaning in real text beside them. Several fall back, under reduced motion, to their last frame alone, so the drop and the recovery vanish. A still frame for a recovery story must show before, during and after.
- **Standards:** WCAG 2.2.2 asks for a visible pause or stop for motion that starts by itself and lasts over 5 s; the reduced-motion setting alone is not one of its techniques ([Understanding 2.2.2](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html)). Scroll-driven animations are in Chrome and Safari but not yet in a released Firefox ([MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/animation-timeline)). SMIL in SVG ignores the reduced-motion setting.
- **Measured in headless Chrome:** CSS animations of properties other than `transform` and `opacity` keep costing style or layout work every frame, also off screen.
- **The three.js field** explained nothing and cost 133 KB with gzip; the same piece in plain WebGL is 2.5 KB. The rebuilt site drops it.
- **What we built:** the figures in [decision 0022](../contributing/decisions/0022-site-rebuild.md#motion). Each story plays once when mostly in view, rests on its last frame, and has its steps as buttons, so it also works by hand and with motion off.

## 16. The new site, measured

Built locally on 15 October 2026 and served with gzip, as GitHub Pages serves it; Lighthouse 13.5.0 in Chrome 155.

| Page | Transfer | Requests | Performance | Accessibility | Best practices | SEO | LCP | TBT | CLS |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Landing, phone (slow 4G, 4× slower CPU) | 146 KiB | 16 | 99 | 100 | 100 | 100 | 1.8 s | 0 ms | 0.006 |
| Landing, desktop | 146 KiB | 16 | 100 | 100 | 100 | 100 | 0.4 s | 0 ms | 0 |
| A guide, phone | 138 KiB | 19 | 99 | 100 | 100 | 100 | 1.8 s | 0 ms | 0 |
| A guide, desktop | 138 KiB | 19 | 100 | 100 | 100 | 100 | 0.4 s | 0 ms | 0.002 |

| Measure | First site | New site |
| --- | --- | --- |
| Landing page, words in its main part | 1,631 | 499 |
| Landing page, height on a laptop (1440 wide) | 7,782 px | 5,711 px |
| Landing page, height on a phone (400 wide) | 11,932 px | 6,942 px |
| Sizes of type on the landing page | 15 | 6 |
| three.js on a laptop | 133 KB with gzip | none |
| Requests to other sites | none | none |

The search's interface (26 KB with gzip) loads when the browser is idle, so search opens at once; its index loads only when a reader searches.

## 17. The third round: why

The owner's review of the rebuilt site, on 15 October 2026, found it still "text heavy" with "no visual story", disliked its teal accent and its UI, and asked for a hero animation, no visible motion controls, and a site that is "simple and clean and clearly conveying". They named three sites as visual references: [Firecrawl](https://www.firecrawl.dev/), [Greptile](https://www.greptile.com/) and [turbopuffer](https://turbopuffer.com/). Three research lanes followed, each with its own sources and screenshots, read on 15 and 16 October 2026. This section and the next three keep what they found.

## 18. How to design a clean website

From 88 pages by 30 publishers: design systems (IBM Carbon, Atlassian, GitHub Primer, Shopify Polaris, Material, Apple, Vercel Geist, Radix, GOV.UK), practitioners (Refactoring UI, Butterick, Utopia, Every Layout, Josh Comeau, Rauno Freiberg) and the Nielsen Norman Group's studies.

- **One spacing scale, few values.** Carbon uses 2, 4, 8, 12, 16, 24, 32, 40, 48, 64, 80, 96 and 160 px, and asks for nothing off the scale ([Carbon spacing](https://carbondesignsystem.com/elements/spacing/overview/)). Primer and Polaris use the same 4 px base. Small gaps group; large gaps separate ([Atlassian spacing](https://atlassian.design/foundations/spacing): 32–80 px between sections).
- **A short measure and calm text.** Body lines of 45–75 characters ([Practical Typography](https://practicaltypography.com/summary-of-key-rules.html)); body text 16–20 px with a line height near 1.5 ([Primer typography](https://primer.style/foundations/typography)).
- **Few sizes, two families, two or three weights.** No more than 3 sizes on a screen and 2 big elements ([NN/g, visual hierarchy](https://www.nngroup.com/articles/visual-hierarchy-ux-definition/)). Material's baseline styles use 2 weights; Apple asks for few typefaces ([Apple typography](https://developer.apple.com/design/human-interface-guidelines/typography)).
- **One focal point a screen.** 57 % of viewing time is above the fold and 74 % in the first two screens, from 130,000 fixations ([NN/g, scrolling and attention](https://www.nngroup.com/articles/scrolling-and-attention/)). The first two screens carry the story.
- **One grey ramp, one accent, each step with a job.** Radix gives 12 steps: backgrounds, fills, borders, solids, text ([Radix scale](https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale)); Geist gives 10 ([Geist colours](https://vercel.com/geist/colors)). Of 112 minimalist sites, 95 % had a limited palette, 49 % monochrome and 46 % with one or two accents ([NN/g, minimalism](https://www.nngroup.com/articles/characteristics-minimalism/)).
- **Contrast floors, not grey to look calm.** 4.5:1 for text, 3:1 for large text and control edges ([WCAG 2.2](https://www.w3.org/TR/WCAG22/)); low-contrast grey hurts reading and trust ([NN/g, low contrast](https://www.nngroup.com/articles/low-contrast/)).
- **Light by default.** Dark text on light reads better for people with normal vision; follow the system and design dark on purpose ([NN/g, dark mode](https://www.nngroup.com/articles/dark-mode/)).
- **Half the words.** Users read at most 28 % of a page's words, and about 18 % of each extra 100 ([NN/g, how little users read](https://www.nngroup.com/articles/how-little-do-users-read/)); a concise rewrite raised usability by 58 % ([NN/g, how users read](https://www.nngroup.com/articles/how-users-read-on-the-web/)).
- **Show the product; let words caption it.** Developer-tool heroes show product UI, code or a live tool ([Evil Martians, 100 landing pages](https://evilmartians.com/chronicles/we-studied-100-devtool-landing-pages-here-is-what-actually-works-in-2025)); a decorative hero that pushes content down hurts ([NN/g, image-focused design](https://www.nngroup.com/articles/image-focused-design/)).
- **One primary action a view** ([Carbon buttons](https://carbondesignsystem.com/components/button/usage/)), and a short navigation of 5 destinations or fewer.
- **Space first, borders second, shadows last; two or three radii.** Shadows for overlays only ([Atlassian elevation](https://atlassian.design/foundations/elevation)); no cards for lists of like items ([NN/g, cards](https://www.nngroup.com/articles/cards-component/)); about 6 px for controls and 12 px for containers ([Geist materials](https://vercel.com/geist/materials)).
- **Plain code blocks:** one command a line, scrolled sideways, a copy button, no shadow ([Carbon code snippet](https://carbondesignsystem.com/components/code-snippet/usage/)).
- **What reads as clutter:** walls of text, everything emphasised, many fonts and colours, decoration with no job, cards for every list, auto-advancing carousels, and off-scale values ([NN/g, aesthetic and minimalist design](https://www.nngroup.com/articles/aesthetic-minimalist-design/)).

**The site on 15 October, against these rules:** about 677 words in the landing page's main part, 9 section headings, 6 font weights, 11 corner radii, 17 colours in the landing page's styles and 10 widths. The targets for the redesign: 300 words or fewer in 5–6 sections, one accent, two families, two weights, four type sizes, one 4 px spacing scale, radii of 6 and 12 px, and one figure per section.

## 19. Landing pages that tell a story

Twenty-nine product sites studied live on 16 October 2026 in headless Chrome, at 1440 × 900 and 390 × 844: hero frames over 10 s, the network and canvas use behind each animation, words and placement by text box, and a second load with reduced motion on.

| Site | Headline words | Hero motion | Technique | Shows the product? | Words in main | Accent |
| --- | --- | --- | --- | --- | --- | --- |
| [Firecrawl](https://www.firecrawl.dev/) (owner's reference) | 7 | Typed queries in a search box; a page becoming JSON just below | Canvas 2D and WebGL, CSS | Yes | 2,971 | Orange-red #FA5D19 |
| [Greptile](https://www.greptile.com/) (owner's reference) | 4 | A halftone lizard; a fly circles; the tongue strikes | dotLottie | As a metaphor | 940 | Mint #28E99F |
| [turbopuffer](https://turbopuffer.com/) (owner's reference) | 3 | The wordmark decodes once; a still box diagram | Script, text | As a diagram | 542 | Peach #FDBA74 |
| [Linear](https://linear.app/) | 8 | The product window fades in; an agent takes a task, once | HTML, CSS, script | Yes | 1,265 | Indigo #5E6AD2 |
| [Warp](https://www.warp.dev/) | 6 | A grid of 224 task squares fills; a live counter | HTML grid, script | Yes | 1,381 | Blue #2A1EFF |
| [Arc](https://arc.net/) | 7 | Ask, think, answer: 6.7 s, once | WebM | Yes | 241 | Blue #3139FB |
| [Railway](https://railway.com/) | 3 | A five-chapter product demo with a chapter bar | HTML, SVG, script | Yes | 2,337 | Purple #553F83 |
| [Clerk](https://clerk.com/) | 5 | A cursor fills the real sign-up form, a 10 s loop | HTML, script | Yes | 1,794 | Violet #6C47FF |
| [Ghostty](https://ghostty.org/) | none | An ASCII ghost in a terminal window | Text frames, script | The medium itself | 54 | Blue #3551F3 |
| [Raycast](https://www.raycast.com/) | 4 | Red light slats, an endless shader | WebGL2 | No | 1,078 | Red #FF6363 |
| [Stripe](https://stripe.com/) | 23 | A colour ribbon, a live counter | WebGL | No | 1,825 | Blurple #533AFD |
| [Tailscale](https://tailscale.com/) | 9 | None | Still SVG | No | 442 | Red #D04841 |

The rest of the 29 (Vercel, Bun, Cursor, Turso, Superhuman, Charm, Tuple, Zed, Resend, Fly.io, Supabase, Neon, OrbStack, Framer, Zellij, Dia, Conductor) fit the same patterns.

What the owner's three references share:

- **A tinted neutral ground and one loud accent,** used on the primary button and one or two key words: #F8F7F7 with orange #FA5D19, #E9E9E9 with mint, navy and #F9FAFC with peach.
- **Mono as the machine's voice:** labels, navigation and captions in a monospace face, often in square brackets; a plainer sans for headlines.
- **Construction lines left visible:** 1 px hairlines with plus marks at the crossings, dashed frames, boxes that share their borders, square corners.
- **Texture made of dots and characters:** ASCII particles, a halftone engraving, dithered shadows. No photos, no glossy 3D, no gradients as decoration.
- **Few words above the fold:** headlines of 3–7 words, sub-headlines of 14–15, one accent button and one quiet one.
- **One moving thing in the hero, and it means something.** None uses video.
- **Diagrams and numbers as proof:** a box-and-arrow architecture, three numbered steps with small diagrams, benchmark tables, real code.

Patterns across all 29:

- **Short pages tell the best stories:** Arc has 241 words in its main part, Dia 301, turbopuffer 542; the median is 902.
- **About 12 of 29 heroes show the product doing its job** (a request, the work, a result, in one frame, with no captions): Linear about 5 s once, Arc 6.7 s once, Clerk a 10 s loop, Railway over 30 s. Decorative heroes (Raycast, Stripe, Vercel, Resend, Neon) look rich and say nothing about the product.
- **Left-aligned text on 20 of 29 pages;** paragraph lines with a median of about 53 characters; about 43 % of the first two screens empty.
- **One repeated unit carries the page:** a heading and one line over a full-width figure (Linear), one frame reused in every section (Arc).
- **Every site has one interface accent;** strong colour beyond it lives only in art.
- **Most heroes ignore reduced motion:** 15 of 29 keep moving under it, and four keep animation loops running at the bottom of the page. No site but ours had a visible pause button.

For simstim, the study recommends a wordless hero loop of about 14 s (a laptop and a host joined by an SSH line; the lid shuts; the host's sessions keep working while time passes; the lid opens; the laptop shows the same screen), then five short sections and about 200 words in all, each a short heading and one line over one figure.

## 20. Hero animation that explains

Six ways to build the hero's story, built as test pages on 15–16 October 2026 with the same 20 s scene, and measured with Lighthouse 13.5 (mobile preset) and with traces on an emulated phone (390 × 844, DPR 3, 4× slower CPU), three runs each.

| Technique | Size with gzip | Main thread, on screen | Off screen, with no help | Phone Lighthouse | Notes |
| --- | --- | --- | --- | --- | --- |
| Inline SVG with CSS keyframes | about 1.7 KB, scene and script | about 1 ms a second | the compositor keeps ticking until paused | 100 | Sharp, themed by CSS variables; SVG animations run on the GPU in Chrome 89 and later ([Chrome](https://developer.chrome.com/blog/hardware-accelerated-animations)) |
| HTML and CSS shapes | about 1.8 KB | about 1 ms a second | as SVG | 100 | Three times the GPU work of SVG |
| Canvas 2D | about 2.6 KB of code | about 34 ms a second | keeps drawing: 65 ms a second | 100 | Lighthouse cannot see the running cost; invisible to screen readers |
| Video, AV1 or H.264 | 50 KB AV1; 280–490 KB for the H.264 or HEVC that Safari needs on older devices | about 0.5 ms a second | a video started from script keeps playing | 100 with a poster | Soft on high-density screens, cannot follow the theme |
| Animated WebP or AVIF | 450 KB WebP; 128 KB AVIF | low | stopped by Chrome | **94**: the image becomes the LCP element (3.15 s) | No pause possible |
| Rive, Lottie, three.js | 40 KB (Lottie light) to 817 KB (Rive with WebGL); 155 KB for three.js | not measured | none by default | not measured | Rive and dotLottie fetch their WASM from a CDN by default ([Rive](https://rive.app/docs/runtimes/web/preloading-wasm), [LottieFiles](https://docs.lottiefiles.com/en/runtimes/distributions/js/v0.x/advanced/performance)); Rive's free exports play a splash screen ([pricing](https://rive.app/pricing)) |

Of 14 product heroes probed, none used Lottie or Rive and none had a visible pause control; Vercel loads no three.js at all under reduced motion, and Stripe draws a cheap still first.

**What WCAG 2.2.2 asks** ([Understanding 2.2.2](https://www.w3.org/WAI/WCAG22/Understanding/pause-stop-hide.html)): motion that starts by itself, lasts more than five seconds and sits beside other content needs "a mechanism for the user to pause", stop or hide it. A hero next to its headline counts. Playing once does not meet it if the motion lasts more than 5 s; neither does moving only while in view, because scrolling something into view still starts it "automatically". The reduced-motion setting is listed as a technique for 2.3.3 ([C39](https://www.w3.org/WAI/WCAG22/Techniques/css/C39)), not for 2.2.2; whether it can count for 2.2.2 is argued both ways ([Hidde de Vries](https://hidde.blog/meeting-2-22-pause-stop-hide-with-prefers-reduced-motion/), against: [WAI list](https://lists.w3.org/Archives/Public/w3c-wai-ig/2022JulSep/0121.html)).

**For simstim:** inline SVG in an Astro component, with CSS keyframes on `transform` and `opacity` only, generated from one table of named phases, with no library. About 14 s: connect, lid shuts, away (the hours pass, the bars keep growing, one agent finishes), lid opens, replay, in sync; then rest on the last frame. Pause when off screen or the tab is hidden, which takes the off-screen cost to zero. Under reduced motion, freeze one meaningful frame. With no visible button, the figure itself can be the mechanism: a tap, a click, or Enter or Space pauses it while it plays and replays it when it has finished.

## 21. Three directions

From sections 18 to 20, three directions for the redesign were drawn as static comps of the hero and one section, at 1440 and 390 px wide, each with a four-beat strip of its hero story. All three use the owner's references' vocabulary; none uses teal or cyan. The comps are kept outside the repository, for the owner's choice.

| Direction | Ground and ink | Accent | Type | The hero's story |
| --- | --- | --- | --- | --- |
| A, Blueprint | off-white #FBFBFA with a hairline grid and plus marks; ink #16181D | signal orange #EF5A1F (#C2410C for text) | Schibsted Grotesk headings, Inter text, JetBrains Mono labels | an SVG line diagram: a square packet runs along the ssh line; the lid folds flat; the host's bars keep filling cell by cell; the lid opens and both screens match, with "replayed 18 KiB" |
| B, Halftone | warm paper #F3F1EC; indigo ink #221E4A | amber #E28C1E | Archivo, expanded, and JetBrains Mono | a halftone drawn in the browser: the sky darkens dot by dot while the host's windows stay lit; at dawn the lid opens |
| C, Terminal | navy #0B1530 band over a light page | peach #FFB088 | JetBrains Mono throughout | box-drawing text frames: the laptop's box folds to a line, the link breaks, the host's bars keep filling, a status line counts the hours |

All three pass 4.5:1 for text (the orange headline words in A, at 72 px, pass the 3:1 for large text). The recommendation is A: it is the closest to the owner's three references, the lightest to build (an SVG of a few KB), the easiest to read, and the one that carries over to the docs' chrome without fighting the text.

The owner chose A on 16 October 2026, with no pause control for the hero; [decision 0022](../contributing/decisions/0022-site-rebuild.md) records the choice and its cost.
