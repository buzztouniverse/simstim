---
status: proposed
date: 2026-10-16
---

# 0022 The site: Starlight for the docs, three tabs, and a landing page that tells its story in a drawing

**Status:** proposed, for the owner's review. The site is built this way on the `website` branch. Once accepted, this record replaces parts of three earlier records:

- from [0015](0015-docs-toolchain.md), the site's toolchain, Hugo with the Hextra theme. The rest of 0015 stays: Diátaxis folders in `docs/`, the generated reference, the version layout, and the checks.
- from [0021](0021-site-design.md), the landing page's sections, the three.js field, "keep Hextra", the site's look (the launcher's colours, from the UX spec) and its motion setting with a pause button. The rest of 0021 stays: one domain with one header and one search, links that stay on the site, a still frame for every animation, and no request to anyone else.
- from [0012](0012-own-look.md), only its hold on the website: the site no longer follows 0012's palette (the cyan accent, the brand gradient, the braille mark). The launcher keeps all of 0012, and so does the copy of the launcher that runs in the landing page.

The evidence is in [the research note on websites](../../research/website.md), sections 11 to 21, and in [the audit of the site](../../research/website-audit.md).

## Context

The owner's review on 15 October 2026 found the site hard to use: the navigation had problems, the landing page was heavy with text and short of white space, and it explained too little. The owner asked for a framework made for docs, docs structured like a standard project, proper motion, explanations and copy.

The audit measured it. The landing page had 1,631 words in eight sections of cards, 14 screens tall on a phone, and cards that ran off a phone's screen. The docs had one sidebar with nine groups for two audiences, guides sorted by the alphabet, and four tabs that did not change the sidebar.

The owner then reviewed the rebuilt site. The docs worked, but the landing page was still "text heavy", with "no visual story for user". They asked for an animation in the hero, no option to turn motion off on the page, and a site that is "simple and clean and clearly conveying". They did not like the teal colour or the UI. They named three sites for the look: Firecrawl, Greptile and turbopuffer. Three directions were drawn (the research note, section 21), and on 16 October 2026 the owner chose A, Blueprint, and no pause control for the hero.

## Options

For the docs, each built against our real `docs/` (the research note, section 12):

1. **Keep Hugo with Hextra,** and fix the sidebar with cascades and weights.
2. **Starlight** (Astro): Pagefind search, accessible by default, component overrides, i18n built in. No versioning, breadcrumbs or `.md` links built in.
3. **Docusaurus:** versioning, breadcrumbs and `.md` links built in; 1,174 npm packages, no local search built in, and plain Markdown "experimental".
4. **Zensical:** small and fast, but at 0.0.x, with versioning from a fork.
5. **VitePress:** its stable line is on an unsupported Vite; our docs broke its build twice.

For the landing page: the eight sections restyled with more space; or a new page of a few sections, each one point and one figure.

For the look, three directions from the owner's references: A, Blueprint, a ruled sheet with line diagrams and one orange; B, Halftone, a printed night sky drawn in dots; C, Terminal, box-drawing frames on a navy band.

For the hero's motion: inline SVG with CSS keyframes, HTML shapes, a canvas, three.js, Rive or Lottie, a video, or an animated image (the research note, section 20).

## Decision

**Starlight** for the docs and the landing page; the look of direction A, Blueprint, on both; and a landing page whose hero tells the whole story as a drawing, with a few short sections after it.

### The docs

- **Content** stays in `docs/` as plain Markdown that reads well on GitHub and that `scripts/test-docs-commands.py` tests. `go run ./cmd/docgen -site site/src` writes it into Starlight's content folder: each README becomes its folder's page, each first heading a title, and each first paragraph a description. It rewrites every link between hosted files into a relative link to the page, so a link works under any base and never sends a reader to GitHub for a page the site has. Links to code still go to GitHub. A Go test covers the rewriting.
- **Three tabs, each with its own sidebar** (`src/route-data.js`): Docs, Reference and Project, with Changelog in the top bar too. Docs is for users: Overview, Get started, Guides (in groups: Hosts, Agents, Daily work, Your setup), Concepts and Help. Reference is generated: the commands as a tree, collapsed under their parents, then the configuration, keys, agents and provider scripts. Project holds the changelog, roadmap, help, security, the contributor guide, the decision records and the research notes. Every address stays as it was.
- **The order and labels come from each folder's README.** `cmd/docgen` reads the links in each README, in order, under its headings, and writes `src/generated/sidebar.json`; the build fails if the sidebar names a page that does not exist.
- **Where am I:** the current tab has an orange rule under it; breadcrumbs (tab, group) over every title; the sidebar marks the page with a grey tint and the site's small orange square; "on this page" beside it; previous and next stay inside the tab.
- **Search** is Starlight's Pagefind: built at build time, loaded in small pieces when a reader searches, from our own site. `/` opens it too, as in the launcher. Decision records and research notes weigh less, so user pages come first.
- **New pages** where readers needed them: [How it works](../../explanation/how-it-works.md), the ideas in plain words, and [common questions](../../faq.md). The command reference is our own Markdown with a table of flags.
- **Versions** keep 0015's layout: the latest release at the root, each release at `/vX.Y/`, `main` at `/dev/`, each built on its own with its own base (`scripts/pages-assemble.sh`). A version menu in the top bar reads `versions.json` at the root, so an old version's menu lists newer ones.

### The landing page

About 200 words, on one ruled sheet. Each section after the hero is a short heading, at most one line, and a drawing:

1. **Walk away. Your agents keep working.** One line on what it is, two buttons (Get started, Read the docs), the install line with a copy button, a note that it is pre-release, and the story (below).
2. **One screen for every session.** The live launcher, with made-up sessions.
3. **The real screen, not a copy.** The seven bytes of Shift+Enter: the same seven through the client, a plain Enter through tmux 3.8.
4. **Your machines. Your SSH.** Your laptop, linked to a VPS, a cloud VM and a box at home.
5. **Three commands to a running agent,** each with a copy button.
6. **Use the agent you already have,** from the agent table.

Each figure sits on a plate of the sheet, ruled off by hairlines, with an orange plus mark at each corner where the rules cross. The page ends with the name, large, and the two buttons again. The comparison, the features, the support tiers and the docs index stay in the docs.

### The hero's story

A drawing on the sheet's grid tells it without words, in about 14 s. Your laptop is linked over SSH to your host, where three agents work, and packets run along the link. The lid shuts and the link drops. A clock runs through the night while the host's bars keep filling, cell by cell, and the agents finish one by one. The lid opens: packets run back, the laptop's stale bars jump to match the host's, a tag says "replayed 18 KiB", and both screens are outlined in orange, the same screen.

- **How it is built:** inline SVG, animated by CSS keyframes on `transform` and `opacity` only, so the browser runs it off the main thread. Every moving part reads its times from one table in `site/src/components/landing/HeroStory.astro`, and the build writes the keyframes from it. No library and no canvas. A wide drawing, and a tall one for phones with the laptop above the host. Both, with their keyframes, weigh 3.8 KB with gzip.
- **When it moves:** it plays once, while at least a third of it is on screen and the tab is visible, and pauses when either stops. It then rests on its last frame. It never loops. Without script, it plays once when the page loads.
- **With reduced motion,** it stands still on the frame that carries the claim: the lid shut, 02:10, the agents still working, one done.
- **For screen readers,** the drawing is one image with a description of the whole story.
- **Nothing else moves by itself.** The launcher plays its one-second start when it first comes into view, then moves only while the reader points at it or gives it focus.

### No pause control, and what it costs

The owner chose no pause or replay control for the story, visible or hidden, and no motion switch anywhere on the site. Motion follows the reader's system setting, `prefers-reduced-motion`, and nothing else.

So the hero does not meet WCAG 2.2.2, Pause, Stop, Hide (Level A), on a strict reading. That criterion covers motion that starts by itself, lasts more than five seconds and sits beside other content, and it asks for a way for the user to pause, stop or hide it. The story starts by itself, runs about 14 s, and sits beside the headline. What the site does instead does not count as that way:

- **Playing once** helps, but the criterion's limit is five seconds of motion, and the story is longer.
- **Moving only in view** does not count: W3C says content that starts when it is scrolled into view still starts automatically.
- **The reduced-motion setting** is listed by W3C as a technique for 2.3.3 (Level AAA), not for 2.2.2. Whether it can count for 2.2.2 is argued both ways, and it helps only people who know the setting exists.

The owner weighed this against a cleaner page: none of 14 product heroes we probed has a pause control either (the research note, section 20). The story is calm: it does not flash, never fills the screen with motion, and ends. The site makes no claim to WCAG AA. Everything else in this record is built to AA: contrast, focus, keyboard use, text alternatives and reduced motion. A later record can add a pause by making the drawing itself a button that pauses and replays it, which adds no visible control.

### The look

Direction A, Blueprint: the page is a ruled engineering sheet, and its drawings are wiring diagrams.

- **Colour:** an off-white ground, `#FBFBFA`, ruled by a 24 px hairline grid in the hero; ink `#16181D`; grey `#5F636C` for secondary text; one signal orange, `#EF5A1F`, for marks, fills and the headline's last words, and `#C2410C` for orange text. The dark theme is the same sheet at night: `#111215`, ink `#F2F2F0`, orange `#FF6A2B`. The site follows the reader's theme. Every text colour passes 4.5:1, except the light theme's orange in the headline, which is display-size text and passes the 3:1 for large text.
- **Type:** Schibsted Grotesk for headings, Inter for text, JetBrains Mono for labels in brackets (`[ ssh ]`), commands and code. No kickers and no section numbers.
- **Form:** hairlines instead of boxes; sections are cells of one framed sheet; orange plus marks at the hero's corners; no shadows and no gradients. The mark is a small orange square beside the name.
- **Docs:** the same ground, ink and orange: an orange rule under the current tab, the orange square at the current page in the sidebar, ink links with an orange underline, mono table heads, plain code blocks with a hairline frame, and asides with a hairline frame instead of a thick coloured bar.

### Supply chain

- Node 24.21.0 and pnpm 12.6.0, pinned with their SHA-256 in `scripts/tools.lock` and installed by `scripts/setup.sh`.
- Three direct dependencies, at exact versions: `astro` 7.3.8, `@astrojs/starlight` 0.42.6, and `@astrojs/markdown-satteri` 0.4.3, Astro's Markdown, named only to turn smart punctuation off. The lockfile pins 270 packages with their hashes. pnpm waits a week before it uses a new release and runs no package's install script. Dependabot watches them with the same seven-day cooldown as the rest.
- Astro's telemetry is turned off in every build (`ASTRO_TELEMETRY_DISABLED=1`).
- The fonts stay committed files, made by `scripts/site-vendor.sh` from pinned downloads.

### Measured

Built locally and served with gzip, as GitHub Pages serves it; Lighthouse 13.5.0 in Chrome, on 16 October 2026. A phone run is Lighthouse's mobile setting: a slow 4G network and a CPU four times slower.

| Page | Transfer | Requests | Performance | Accessibility | Best practices | SEO | LCP | TBT | CLS |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Landing, phone (three runs) | 143 KiB | 13 | 99, 99, 100 | 100 | 100 | 100 | 1.7 s | 0 ms | 0 |
| Landing, desktop | 143 KiB | 13 | 100 | 100 | 100 | 100 | 0.4 s | 0 ms | 0.002 |
| A docs page (Install), phone | 134 KiB | 16 | 99 | 100 | 100 | 100 | 1.7 s | 0 ms | 0.014 |

The landing page has 214 words outside its drawings, down from 499 in the first rebuild and 1,631 before it. [The research note](../../research/website.md#20-hero-animation-that-explains) has the measurements of each way to build the story.

## Consequences

- Good, because each reader sees their own tab: a user never meets a decision record on the way to a guide.
- Good, because the sidebar's order lives in the README a contributor already edits, and a missing page fails the build.
- Good, because search, the theme switch, the phone menu, "on this page" and the accessibility basics come from Starlight, maintained upstream.
- Good, because the landing page tells what the product does in one drawing, before a word is read, and stays light: the story is 3.8 KB with gzip and costs the main thread almost nothing.
- Good, because the story is one table of times: changing it is changing numbers, not redrawing frames.
- Bad, because the hero does not meet WCAG 2.2.2 (Level A) on a strict reading, so the site cannot claim AA. Adding a pause later is small.
- Bad, because the site now needs Node and 270 npm packages to build. Most contributors never build it; CI does, and `scripts/setup.sh` installs the pinned tools.
- Bad, because Starlight is 0.x and breaks in minor releases: five in seven months. We pin exact versions and update on purpose.
- Bad, because we own the top bar, the sidebar's tabs, the breadcrumbs and the footer as overrides; a Starlight update can move what they rest on.
- Bad, because the website and the launcher now look different. The launcher keeps 0012's look until a record changes it.
- Bad, because the copy of the launcher in the page is still a second implementation, as 0021 said.
- Revisit when Starlight reaches 1.0, if Zensical reaches a stable release with native versioning, or if the site needs to claim WCAG AA.
