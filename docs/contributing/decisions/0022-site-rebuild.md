---
status: proposed
date: 2026-10-15
---

# 0022 The site: Starlight for the docs, three tabs, and a landing page of figures

**Status:** proposed, for the owner's review. The site is built this way on the `website` branch. Once accepted, this record replaces two parts of earlier records:

- from [0015](0015-docs-toolchain.md), the site's toolchain, Hugo with the Hextra theme. The rest of 0015 stays: Diátaxis folders in `docs/`, the generated reference, the version layout, and the checks.
- from [0021](0021-site-design.md), the landing page's sections, the three.js field and "keep Hextra". The rest of 0021 stays: the launcher's look, one domain with one header and one search, links that stay on the site, a still frame for every animation, and no request to anyone else.

The evidence is in [the research note on websites](../../research/website.md), sections 11 to 16, and in [the audit of the site](../../research/website-audit.md).

## Context

The owner's review on 15 October 2026 found the site hard to use: the navigation had problems, the landing page was heavy with text and short of white space, and it explained too little. The owner asked for a framework made for docs, docs structured like a standard project, proper motion, explanations and copy.

The audit measured it. The landing page had 1,631 words in eight sections of cards, 14 screens tall on a phone, and cards that ran off a phone's screen. The docs had one sidebar with nine groups for two audiences, guides sorted by the alphabet, and four tabs that did not change the sidebar.

## Options

For the docs, each built against our real `docs/` (the research note, section 12):

1. **Keep Hugo with Hextra,** and fix the sidebar with cascades and weights.
2. **Starlight** (Astro): Pagefind search, accessible by default, component overrides, i18n built in. No versioning, breadcrumbs or `.md` links built in.
3. **Docusaurus:** versioning, breadcrumbs and `.md` links built in; 1,174 npm packages, no local search built in, and plain Markdown "experimental".
4. **Zensical:** small and fast, but at 0.0.x, with versioning from a fork.
5. **VitePress:** its stable line is on an unsupported Vite; our docs broke its build twice.

For the landing page: the eight sections restyled with more space; or a new page of a few sections, each one point and one figure.

## Decision

**Starlight** for the docs and the landing page, and a landing page of six sections, each with one figure that explains one thing.

### The docs

- **Content** stays in `docs/` as plain Markdown that reads well on GitHub and that `scripts/test-docs-commands.py` tests. `go run ./cmd/docgen -site site/src` writes it into Starlight's content folder: each README becomes its folder's page, each first heading a title, and each first paragraph a description. It rewrites every link between hosted files into a relative link to the page, so a link works under any base and never sends a reader to GitHub for a page the site has. Links to code still go to GitHub. A Go test covers the rewriting.
- **Three tabs, each with its own sidebar** (`src/route-data.js`): Docs, Reference and Project, with Changelog in the top bar too. Docs is for users: Overview, Get started, Guides (in groups: Hosts, Agents, Daily work, Your setup), Concepts and Help. Reference is generated: the commands as a tree, collapsed under their parents, then the configuration, keys, agents and provider scripts. Project holds the changelog, roadmap, help, security, the contributor guide, the decision records and the research notes. Every address stays as it was.
- **The order and labels come from each folder's README.** `cmd/docgen` reads the links in each README, in order, under its headings, and writes `src/generated/sidebar.json`; the build fails if the sidebar names a page that does not exist.
- **Where am I:** the current tab is lit; breadcrumbs (tab, group) over every title; the sidebar marks the page with the launcher's cyan marker and tint; "on this page" beside it; previous and next stay inside the tab.
- **Search** is Starlight's Pagefind: built at build time, loaded in small pieces when a reader searches, from our own site. `/` opens it too, as in the launcher. Decision records and research notes weigh less, so user pages come first.
- **New pages** where readers needed them: [How it works](../../explanation/how-it-works.md), the ideas in plain words, and [common questions](../../faq.md). The command reference is our own Markdown with a table of flags.
- **Versions** keep 0015's layout: the latest release at the root, each release at `/vX.Y/`, `main` at `/dev/`, each built on its own with its own base (`scripts/pages-assemble.sh`). A version menu in the top bar reads `versions.json` at the root, so an old version's menu lists newer ones.

### The landing page

Six sections, about 450 words, each a heading that states one point, one or two sentences, and one figure with a caption, as a datasheet sets them out:

1. **Your agents, on your hosts, in your terminal.** The install line, one button, one link, and figure 1: the live launcher with numbered notes on what each part shows.
2. **The session outlives your laptop.** Figure 2: forty minutes of output as two braille traces, the host's and the laptop's. The laptop's goes quiet while the lid is shut, then the missed output arrives in one burst.
3. **The real screen, not a copy.** Figure 3: the seven bytes of Shift+Enter, the same through the client, a plain Enter through tmux 3.8.
4. **Your machines. Your SSH.** Figure 4: the client, the helper and the holder, in a line.
5. **Three commands to a running agent,** each with a copy button.
6. **Use the agent you already have,** and the welcome mark at the end.

The comparison, the six extra features, the support tiers and the docs index move to the docs, where they already have pages.

### Motion

- **It explains.** Figures 2 and 3 are stories: an ordered list of steps beside the figure. With motion on, a figure plays its steps once when it is mostly in view, then rests on its last frame, with "Play again". Each step is a button that shows the figure at that step, so the story also works by hand, and with motion off.
- **The still frame is the whole story.** The HTML draws each figure's last frame, which shows before, during and after at once.
- **One setting** for the whole site, as in 0021: auto, on or off, in the footer, and a pause button under figure 1. Nothing moves with motion off; every loop stops off screen and in a hidden tab.
- **No three.js.** The field explained nothing and cost 133 KB with gzip. The welcome mark stays, as text.

### The look

0012's world, kept: the cyan accent, the brand gradient on the name, the braille mark, rules rather than boxes. Headings move from the monospace face to Schibsted Grotesk, cut to 19 KB, so long pages read lighter; mono stays for code, commands, labels and the launcher. No kickers or section numbers; figures are numbered because the captions refer to them.

### Supply chain

- Node 24.21.0 and pnpm 12.6.0, pinned with their SHA-256 in `scripts/tools.lock` and installed by `scripts/setup.sh`.
- Three direct dependencies, at exact versions: `astro` 7.3.8, `@astrojs/starlight` 0.42.6, and `@astrojs/markdown-satteri` 0.4.3, Astro's Markdown, named only to turn smart punctuation off. The lockfile pins 270 packages with their hashes. pnpm waits a week before it uses a new release and runs no install script but esbuild's. Dependabot watches them with the same seven-day cooldown as the rest.
- Astro's telemetry is turned off in every build (`ASTRO_TELEMETRY_DISABLED=1`).
- The fonts stay committed files, made by `scripts/site-vendor.sh` from pinned downloads.

### Measured

Built locally and served with gzip, as GitHub Pages serves it; Lighthouse 13.5.0 in Chrome, on 15 October 2026. The landing page moves 146 KiB in 16 requests and scores 99, 100, 100 and 100 on a phone (performance, accessibility, best practices, SEO); a guide moves 138 KiB and scores the same. The landing page has 499 words, down from 1,631, and six sizes of type, down from 15. [The research note](../../research/website.md#16-the-new-site-measured) has the full table.

## Consequences

- Good, because each reader sees their own tab: a user never meets a decision record on the way to a guide.
- Good, because the sidebar's order lives in the README a contributor already edits, and a missing page fails the build.
- Good, because search, the theme switch, the phone menu, "on this page" and the accessibility basics come from Starlight, maintained upstream.
- Good, because the landing page says what the product is in one screen and shows its main promise working.
- Bad, because the site now needs Node and 270 npm packages to build. Most contributors never build it; CI does, and `scripts/setup.sh` installs the pinned tools.
- Bad, because Starlight is 0.x and breaks in minor releases: five in seven months. We pin exact versions and update on purpose.
- Bad, because we own the top bar, the sidebar's tabs, the breadcrumbs and the footer as overrides; a Starlight update can move what they rest on.
- Bad, because the copy of the launcher in the page is still a second implementation, as 0021 said.
- Revisit when Starlight reaches 1.0, or if Zensical reaches a stable release with native versioning.
