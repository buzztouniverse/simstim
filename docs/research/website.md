# Websites for developer tools: what works (research for the site)

- Date: 15 October 2026
- Lane: the public web. The landing pages and docs of terminal and developer tools that developers praise, their source where it is open, and well-known write-ups. Used for [decision 0021](../contributing/decisions/0021-site-design.md), the design of this project's site.
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
