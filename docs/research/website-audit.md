# An audit of the site, before the rebuild

- Date: 15 October 2026
- Site: the live site at [simstim.astrel.sh](https://simstim.astrel.sh), built from `main` with Hugo and Hextra ([decision 0021](../contributing/decisions/0021-site-design.md)).
- Why: the owner's review on 15 October 2026 found problems in the navigation, a landing page heavy with text and short of white space, and too little explanation. This note lists every problem we found, so the rebuild can answer each one. [Decision 0022](../contributing/decisions/0022-site-rebuild.md) has the answers.
- Method: Chrome, driven by hand through its extension, at 1440 × 900, and headless Chrome at 400 × 860 (a phone, with touch) and 1440 wide, in the dark and the light theme. We read the landing page from top to bottom; the docs overview; a tutorial (Getting started); a how-to guide (Add a host); the reference index, the command index and one command page; the changelog; the search dialog; the menu on a phone. A script measured words, sections, type sizes, transfer sizes and any element wider than the screen. The screenshots are kept outside the repository.

## Summary

The site has the right parts: a landing page that shows the launcher, docs with search, breadcrumbs and an "on this page" list, and no request to anyone else. The problems are in the arrangement.

- **The landing page says too much.** 1,631 words in eight sections, 8.6 screens tall on a laptop and 14 on a phone. Every section has the same shape (a label, a two-line heading, a paragraph, a grid of cards, a row of links), so nothing stands out and nothing rests.
- **The docs mix two audiences.** The sidebar puts the project's own notes (decisions, research, contributing) beside the user's guides, nine groups in all, and the top bar's tabs do not change it.
- **Some things are broken.** On a phone, four cards are wider than the screen and cut their text off. A command page cuts its option table off at the right. A backslash goes missing from the reference.

## 1. The landing page

Measured on the live page at 1440 × 900, in the dark theme.

| Measure | Value |
| --- | --- |
| Words in the page's main part | 1,631 |
| Sections | 8 |
| Height on a laptop (1440 wide) | 7,782 px: about 8.6 screens |
| Height on a phone (400 wide) | 11,932 px: about 14 screens |
| Words in the first section | 236 |
| Font sizes in the main part | 15 different sizes, from 11.4 px to 52 px |
| Boxed cards | 29 |
| Transfer, desktop, motion on | 268 KB in 16 requests; 135 KB of it is the three.js field |
| Requests to other sites | none |

### 1.1 The first screen

- **Five rows of text before the product.** A row of seven labels (pre-release, open source, Apache-2.0, no account, no relay, no telemetry), a 12-word headline over three lines, a 42-word paragraph, the install command, a note about the pre-release, two buttons and a hint about `/`. The launcher, the thing that shows the product, starts below the fold, at about 750 px.
- **The headline is set in the monospace face at 52 px.** Mono at that size is wide and dark, and a three-line headline in it reads as a block.
- **The install command cannot work yet.** The page offers it first, then explains in small grey text that it works only from the first release.
- **The launcher's frame is 60 % empty.** It shows four sessions in a window about 540 px tall. The caption under it has two sentences, five key chips and a pause button.

### 1.2 The sections

- **One template, eight times.** Each section has a numbered label ("01 why", "02 how it works"…), a heading in mono, a paragraph of two to four lines, a grid of boxed cards and a row of arrow links. Same spacing, same density, same weight. The eye finds no single point per section.
- **Cards carry the page.** Four feature cards, then six more features in three columns, five cards for the parts, three for the quick start, three for the agents, four plus four for the comparison, and four for the docs: 29 boxes.
- **Small type in narrow columns.** The five "parts" cards set 13–14 px text in columns 150 px wide.
- **Words a newcomer does not know, early.** PTY, holder, bridge, ControlMaster, IAP, SSM and systemd user service all appear before the quick start.
- **The comparison is a docs page.** Eight cards of prose with crosses and arrows. It belongs in the docs, with a short line on the landing page.
- **The docs section repeats the footer.** The four kinds of page and their first links appear in the "docs" section and again in the footer.
- **The closing links look like keys.** `enter`, `a` and `?` are drawn as keys, but pressing them does nothing; they are links.

### 1.3 Motion

- Six things move: the braille ribbon and the three.js field behind the hero, the live launcher, a braille animation in each feature card, packets in the diagram, and the closing mark.
- **Most of it decorates.** The field, the ribbon and the card animations do not explain anything. The launcher does: it shows sessions changing state.
- **The story that matters most has no picture.** The product's main promise is that the agent keeps working while the laptop is closed, and that coming back replays what you missed. No animation shows it.
- **What works:** motion follows `prefers-reduced-motion`, the footer has a motion switch, a button pauses the launcher, and every loop stops off screen and in a hidden tab ([0021](../contributing/decisions/0021-site-design.md#motion)).

### 1.4 On a phone (400 px)

- **Cards wider than the screen.** The four feature cards are 447 px wide in a 400 px screen. Their text runs past the right edge and is cut off: "Antigravity CLI", the end of "Nothing new listens on the host", and more.
- **Commands cut short.** The install command and the three quick-start commands are cut at the right with no sign that they scroll.
- **The launcher is too small to read.** It shrinks the whole window to fit, so its text is about 6 px high.
- **Fourteen screens of scrolling.** The comparison alone is two screens.

### 1.5 Light theme

The light theme works and passes contrast. The paper ground and the cards behave as in the dark theme, with the same density problems.

## 2. The docs

### 2.1 The top bar

- **Four tabs that do not change the sidebar.** Docs, Get started, Reference and Changelog all open pages in the one sidebar tree, so they are shortcuts, not sections.
- **The wrong tab lights up.** On the getting-started page "Get started" is lit and "Docs" is not; on a reference page "Reference" is lit and "Docs" is not, though both are in the docs. A reader cannot tell which part of the site they are in.
- **No version picker.** The footer says "Docs for dev" but offers no way to reach another version.

### 2.2 The sidebar

- **Nine top-level groups, for two audiences.** Overview, Tutorials, How-to guides, Reference, Explanation, Roadmap, Project, Contributing and Research. The last three hold the 21 decision records, the 12 research notes and the contributor guides, beside the user's guides.
- **How-to guides sorted by title.** Add a host, Bring a session's work…, Bring sessions back…, Change keys, Create a host, Get alerts, Install, … Uninstall. "Install" is seventh; "Create a host" comes after "Change keys". The order follows the alphabet, not the task.
- **Long titles wrap.** Ten of the fifteen how-to titles wrap to two lines in the 190 px column.
- **The command list swamps the tree.** 41 commands, each with the product's name in front ("… host alerts off"), many on two lines. Open, it pushes every other group off the screen, and the sidebar scrolls inside the page.
- **The open-and-close arrows sit off their rows.** Each chevron is about 12 px above its label, so it looks as if it belongs to the row above.

### 2.3 Pages

- **The overview (`/docs/`) is a flat table of 19 rows.** User pages and contributor pages share it: the UX spec, decisions, renaming, releasing and going public sit beside tutorials and how-to guides.
- **The reference index is four bullets.** It does not say which page answers which question.
- **The root command's page is titled "Commands"**, but it is the page for the bare command. Its sections are "Synopsis", "Options" and "SEE ALSO", in capitals, from the generator.
- **Option tables are cut off.** On `new`, the options are a code block; long lines run past the right edge, and the copy button sits on top of the text.
- **A backslash is lost.** "detach with Ctrl+\\" renders as "detach with Ctrl+)": the generator writes `Ctrl+\)` and Markdown reads `\)` as an escaped bracket.
- **Prev and next** are at the very bottom, after "View the source of this page" and "Suggest a change". Only "next" shows on the first page of a group.
- **What works:** breadcrumbs, the "on this page" list, copy buttons on code, the skip link, and the clear page titles.

### 2.4 Search

- Search works on `/` and Ctrl/Cmd+K, on every page, and loads its index only when first opened.
- **The dialog is a neutral grey** (`oklch(0.205 0 0)`) that does not match the site's blue-black.
- **Ranking is weak.** For "worktree", the first result is "Set up a repository when a session starts"; "Keep each task in its own worktree" is second.

### 2.5 On a phone

- **The menu is one long list.** It lists the top bar's items, then the whole sidebar tree, and under the open page every heading marked with "#".
- **No "on this page"** on a phone.

### 2.6 Type

- **Every heading is in the monospace face.** On long pages the headings compete with the code blocks, which use the same face.
- The body text is Inter at 16–17 px, with a comfortable measure.

## 3. What to keep

- The look of [decision 0012](../contributing/decisions/0012-own-look.md): the cyan accent, the brand gradient on the name, the braille mark, rules instead of boxes.
- No request to anyone else, no fonts from elsewhere, no analytics.
- A still frame for every animation, the pause button and the motion switch.
- The live launcher, made readable on a phone.
- Breadcrumbs, the "on this page" list, search on `/`, copy buttons, the skip link.
- Links between pages that stay on the site, and the check that enforces it.

## 4. Not checked

- Screen readers: we read the accessibility tree, but did not use VoiceOver or NVDA.
- Older browsers: only current Chrome.
- The versions at `/vX.Y/`: there is no release yet, so only `main`'s docs exist.
