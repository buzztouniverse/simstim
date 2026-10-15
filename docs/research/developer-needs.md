# What developers want from remote and background agent sessions

Research note for simstim. Researched on 15 October 2026.

This note asks what developers want when a coding agent runs somewhere other than the laptop in front of them, and what hurts today. It covers setup, persistence, fidelity over SSH, mosh and tmux, notifications, cost, lock-in, and the features that vendors keep behind an account or a paid plan. [Amp UX and Orbs](amp-ux-and-orbs.md) already covers Amp and its community, so this note only links to it.

The rules for the note:

- Every source is in the [source table](#sources) with its date, the date I read it, a link and its need codes. In the text, "S12" means row 12 of that table.
- **UNVERIFIED** marks a claim that I could not confirm from a primary source.
- Counts come first. Quotes are short, linked, and only illustrate a count.
- Reddit text and scores come from an archive copy, not the live page (see [Method](#method)). Each Reddit claim is therefore **UNVERIFIED** against reddit.com itself.

## Summary

1. **Reach comes first.** The two most common needs are to reach a session from a phone (40 sources) and to run it on a machine the developer owns (37 sources). Together they appear in 55 of the 132 sources that speak for developers, on 21 GitHub items with 2,715 reactions. The largest single requests are "remote development in the Codex desktop app" (876 reactions, S37) and "Codex remote control from the phone" (544 reactions, S38). People ask to reach *their own* machine and tools, not a vendor sandbox.
2. **Fidelity carries the most reactions.** Rendering, scrollback, keys, clipboard and image paste appear in 40 sources and 32 GitHub items with 3,937 reactions. The four most-reacted Claude Code rendering and scroll bugs alone hold 2,315 reactions (S1 to S4). In 2026 both Claude Code (April) and Codex (September) moved to a full-screen view by default. Users pushed back because it breaks native scrollback and copy (S6, S7, S47, S48: 8 of 10 reactions on the Codex announcement are thumbs down).
3. **The DIY stack is still the default answer.** tmux, mosh or zellij come up in 37 of the 63 Hacker News and Reddit threads I read, and Tailscale in 24. A typical guide joins 3 to 5 tools in 3 to 7 steps (S110, S134, S135, S93). Each tool loses something: mosh has had no scrollback since 2012 (280 reactions, still open, S75) and drops OSC 52 (S76). tmux has no kitty keyboard protocol as of October 2026 (S77, S78) and blocks agent notifications unless they are wrapped (S10).
4. **Vendor remote features are welcome, but fragile and locked.** Claude Code Remote Control drew 544 points on Hacker News (S79). Its top bugs are about dropped and idle-killed connections (195 reactions, S15, S16). Codex remote needs the ChatGPT desktop app awake on macOS or Windows, and does not support Linux hosts (S140, S40). Remote Control sends all traffic through the Anthropic API and needs a paid plan (S137). Codex remote uses a "secure relay layer" and a signed-in ChatGPT account; its docs do not say whose servers run the relay (S140).
5. **Lock-in and cost are the angriest topics.** Google's March 2026 limits on Gemini CLI drew 1,035 thumbs down (S68). Its May 2026 move of consumer users to the closed Antigravity CLI drew 300 thumbs down (S69). The repo for the new CLI has no source code (S70). Anthropic's legal action against OpenCode drew 483 points (S99). Claude Max usage limits drew 726 reactions and 1,499 comments (S32). Open-source tools that run several agents side by side hold about 132,000 GitHub stars between them (S144 to S150).
6. **The bottleneck moves to the human.** Parallel agents appear in 18 sources, notifications in 13 and an at-a-glance status in 4. The creator of Claude Code numbers five terminal tabs and relies on system notifications to know which one needs input (S128). A tmux dashboard that shows agent state got 691 points on r/ClaudeAI (S117). Agent notifications often fail inside tmux and zellij (S10, S61).
7. **Setup breaks in the same places.** These are: logging an agent in on a headless box (3 issues, 245 reactions, S20, S43, S63); the agent dying when SSH drops without tmux (S14, S134); cloud sandboxes that lack the developer's own tools (S81, S24); and phone keyboards that cannot send agent keys (S111, S85, S135). No source measured setup time.
8. **Phone control, cloud sessions and cloud agents all sit behind a vendor plan.** Claude Remote Control rejects API keys and Bedrock (S137; S19 has 135 reactions asking for Bedrock). Claude cloud sessions need a claude.ai plan and GitHub to push (S139). Cursor Cloud Agents need the Premium plan or higher (S141). Anthropic's terms forbid third-party tools from routing requests through a Pro or Max plan, or from handling Claude.ai credentials (S138). Free, open-source alternatives are popular: Happy (24,155 stars), Paseo (20,967) and CloudCLI (14,106).
9. **Some needs are rare.** Sharing a live session (0 sources), snapshots (1) and secrets handling (3) are rare in this evidence. Voice input (5) matters only on phones.
10. **For simstim:** most of the top needs are already in the v1 and v2 build: own host, survive sleep, byte-exact fidelity, image paste, status, worktrees, reboot restore and ports. The gaps with the most demand are: a phone path that needs no new client, alerts that reach a phone without a relay, a "who needs me" jump across hosts, agent login checks at host setup, and "continue when my usage limit resets". See [What this means for simstim](#what-this-means-for-simstim).

---

## Method

### Where the sources come from

| Channel | How I read it | What I took |
| --- | --- | --- |
| GitHub issues and discussions | The GitHub search API through `gh`. I ran 25 title searches, plus a few targeted ones (for example `tmux in:title`, `scrollback in:title`, `remote in:title`), sorted by reactions, on `anthropics/claude-code`, `openai/codex`, `anomalyco/opencode` (the current home of sst/opencode), `google-gemini/gemini-cli` and `Aider-AI/aider`. I also searched `tmux/tmux` and `mobile-shell/mosh`. I used GraphQL for discussions. | 69 issues, 4 discussions and 2 pull requests, with their reaction and comment counts on 15 October 2026 |
| Hacker News | The Algolia HN API: stories since 1 January 2025 for 25 queries, then the full comment tree of 42 threads | 32 threads coded as sources |
| Reddit | reddit.com and its JSON need a login now (HTTP 403 and a login redirect), and the search tool cannot read it. I read posts and comments through the Arctic Shift archive, which copies public Reddit data and needs no account. | 21 threads read, 18 coded as sources. Scores are as archived and may be lower than on the live page. |
| X | X pages do not render without a login. I read posts through X's own public oEmbed endpoint, which returns the post text and date. | 4 posts |
| Blogs, talks, docs | Web search and fetch | 4 blogs, 1 talk, 8 vendor pages |
| Repos | GitHub API | 12 tool repos, with stars as a demand signal, plus the Antigravity CLI repo and aider's releases page |

### How I coded the needs

- I fixed 27 need codes (see [Need codes](#need-codes)). I refined them from the evidence: `multi-device`, `status` and `voice` were added after reading, and `own-env` was split from `own-host`.
- An issue or post got a code when its title or body asks for that need, or complains that it is missing.
- A Hacker News or Reddit thread got a code when at least two comments raise the need. I found candidates with a keyword pass over all comments, then read the matching comments. The keyword counts and the coding script are kept outside git.
- **Voice sources** are where developers speak: GitHub issues, discussions and pull requests, HN and Reddit threads, posts by developers on X, blogs and talks. Vendor docs, vendor posts and tool repos are in the table, but the ranking does not count them as demand.
- **Reactions** are the total reactions on a GitHub item when I read it. For a vendor announcement in GitHub Discussions, I count only its thumbs-down reactions, because they are the users' answer to the announcement.

### Limits

- I chose the search terms, so the counts show what is common *within* the topic of remote and background sessions. They do not show what share of all developers care.
- Reactions measure how strongly people feel better than source counts do. A source count of 5 with 1,000 reactions can matter more than a count of 15 with none.
- GitHub search caps each query, and many Claude Code issues are closed as duplicates, so reaction totals undercount.

---

## Need codes

| Code | Need | What developers ask for |
| --- | --- | --- |
| `sleep` | persist-through-sleep | The session keeps running when the laptop sleeps, the lid closes or the client quits |
| `reboot` | resume-after-reboot | After a reboot or crash, the conversation comes back where it was |
| `own-host` | run-on-own-server | Run the agent on a VPS, a cloud VM or a home machine that the developer controls |
| `own-env` | own-environment | The developer's own tools, repos and Git host, not a vendor's fixed sandbox |
| `mobile` | mobile-access | Check, steer or approve from a phone or tablet |
| `notify` | notify-when-input-needed | An alert when an agent finishes or waits for input |
| `status` | status-at-a-glance | See which of many sessions is working, stuck or done without opening each one |
| `parallel` | parallel-agents | Run several agents at once |
| `worktree` | worktrees | Isolate each task in its own git worktree or branch |
| `sandbox` | sandbox-isolation | Run agents in a container, a VM or with limited permissions, so "yolo" mode is safe |
| `render` | tui-fidelity | No flicker, no redraw corruption, correct colours, inside tmux and over SSH |
| `scrollback` | scrollback | Native terminal scrollback and search keep working |
| `keys` | key-fidelity | Shift+Enter, Shift+Tab and other keys reach the agent |
| `clipboard` | clipboard | Copy out of the agent works, locally and over SSH (OSC 52) |
| `image` | image-paste | Paste a screenshot or drop a file into the agent, including when the agent is remote |
| `reconnect` | reliable-reconnect | The link comes back by itself after a drop, with nothing lost |
| `ports` | port-preview | See the app the agent is building, through a forwarded port or a preview |
| `login` | headless-login | Log the agent in on a machine with no browser |
| `secrets` | secrets | Scoped credentials for the agent on a remote box |
| `any-agent` | lock-in/any-agent | One tool for every agent; no vendor account, ban or shutdown can strand the work |
| `cost` | cost | Subscription limits, API spend and VM cost |
| `setup` | setup-time | Few steps, few tools, little to configure |
| `trust` | trust/no-relay | No third-party relay, no hidden daemon, no surprise data storage; end-to-end encryption |
| `multi-device` | multi-device-attach | The same session open from laptop and phone at once |
| `share` | sharing/watch | Teammates can watch or join a session |
| `snapshot` | snapshots | Save and restore a whole environment |
| `voice` | voice-input | Dictate prompts on a phone |

---

## Needs ranked

Ranked by the number of voice sources, then by GitHub reactions. The last column also counts tool repos and vendor pages.

| Rank | Need (code) | Voice sources | of which HN / Reddit | GitHub items | GitHub reactions | With tools and vendor pages |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | mobile-access (`mobile`) | 40 | 12 / 13 | 11 | 1,373 | 47 |
| 2 | run-on-own-server (`own-host`) | 37 | 9 / 10 | 15 | 2,261 | 41 |
| 3 | cost (`cost`) | 21 | 11 / 4 | 6 | 2,318 | 25 |
| 4 | parallel-agents (`parallel`) | 18 | 8 / 2 | 4 | 283 | 24 |
| 5 | scrollback (`scrollback`) | 17 | 3 / 1 | 11 | 2,302 | 17 |
| 6 | lock-in/any-agent (`any-agent`) | 16 | 10 / 1 | 3 | 1,470 | 31 |
| 7 | sandbox-isolation (`sandbox`) | 15 | 10 / 3 | 0 | 0 | 18 |
| 8 | reliable-reconnect (`reconnect`) | 14 | 3 / 7 | 3 | 242 | 16 |
| 9 | notify-when-input-needed (`notify`) | 13 | 4 / 2 | 6 | 225 | 18 |
| 10 | tui-fidelity (`render`) | 12 | 1 / 1 | 9 | 2,655 | 12 |
| 11 | setup-time (`setup`) | 12 | 5 / 3 | 2 | 389 | 12 |
| 12 | persist-through-sleep (`sleep`) | 11 | 4 / 3 | 3 | 162 | 13 |
| 13 | clipboard (`clipboard`) | 10 | 1 / 1 | 8 | 502 | 10 |
| 14 | worktrees (`worktree`) | 10 | 5 / 2 | 2 | 216 | 14 |
| 15 | trust/no-relay (`trust`) | 10 | 8 / 1 | 1 | 198 | 14 |
| 16 | key-fidelity (`keys`) | 8 | 1 / 1 | 5 | 251 | 8 |
| 17 | own-environment (`own-env`) | 7 | 2 / 2 | 3 | 1,235 | 8 |
| 18 | resume-after-reboot (`reboot`) | 6 | 1 / 0 | 5 | 226 | 7 |
| 19 | image-paste (`image`) | 6 | 0 / 0 | 6 | 175 | 8 |
| 20 | headless-login (`login`) | 5 | 1 / 0 | 3 | 245 | 6 |
| 21 | port-preview (`ports`) | 5 | 3 / 1 | 1 | 14 | 5 |
| 22 | voice-input (`voice`) | 5 | 3 / 2 | 0 | 0 | 5 |
| 23 | multi-device-attach (`multi-device`) | 4 | 1 / 1 | 2 | 51 | 5 |
| 24 | status-at-a-glance (`status`) | 4 | 0 / 3 | 0 | 0 | 4 |
| 25 | secrets (`secrets`) | 3 | 1 / 1 | 0 | 0 | 3 |
| 26 | snapshots (`snapshot`) | 1 | 0 / 1 | 0 | 0 | 1 |
| 27 | sharing/watch (`share`) | 0 | 0 / 0 | 0 | 0 | 1 |

### The same needs, grouped

A source counts once per group, even if it has several codes in the group.

| Group (codes) | Voice sources | GitHub items | GitHub reactions |
| --- | --- | --- | --- |
| Reach (`own-host`, `mobile`) | 55 | 21 | 2,715 |
| Fidelity (`render`, `scrollback`, `keys`, `clipboard`, `image`) | 40 | 32 | 3,937 |
| Freedom (`any-agent`, `cost`, `trust`) | 35 | 9 | 2,951 |
| Persistence (`sleep`, `reconnect`, `reboot`, `multi-device`) | 30 | 12 | 634 |
| Parallel work (`parallel`, `worktree`, `sandbox`) | 28 | 5 | 370 |
| Setup (`setup`, `login`, `own-env`) | 20 | 7 | 1,609 |
| Attention (`notify`, `status`) | 15 | 6 | 225 |

### What the ranking shows

- **Two kinds of evidence point in different directions.** Forums talk about reach, cost and lock-in. GitHub reactions pile up on fidelity. Fidelity has only 2 HN or Reddit sources for `render`, but 2,655 reactions. People do not start a forum thread about flicker. They add a thumbs-up to the bug.
- **Persistence is mostly a "how", not a "what".** `sleep` (11) and `reconnect` (14) are usually the reason people reach for tmux, mosh or a vendor relay. They rarely ask for them by name.
- **Sandboxing is talked about but not requested.** It has 15 forum sources but no GitHub item in this sample. It is a concern raised about other features ("run it isolated"), not a feature request on its own.

### Issue volume by topic

How many issues have the word in their title, in each agent's repo. This is a rough measure of how often each thing breaks. The counts include closed and duplicate issues, and some are unrelated (for example `ssh` matches Git over SSH). GitHub search, 15 October 2026.

| Word in title | claude-code | codex | opencode | gemini-cli | aider |
| --- | --- | --- | --- | --- | --- |
| tmux | 302 | 46 | 57 | 16 | 1 |
| zellij | 10 | 16 | 14 | 1 | 0 |
| mosh | 4 | 2 | 0 | 0 | 0 |
| ssh | 611 | 378 | 61 | 21 | 3 |
| flicker | 198 | 58 | 22 | 51 | 0 |
| scrollback | 153 | 36 | 9 | 3 | 0 |
| clipboard | 274 | 62 | 114 | 21 | 11 |
| image paste | 204 | 52 | 59 | 12 | 1 |
| notification | 944 | 182 | 107 | 38 | 3 |
| worktree | 1,204 | 308 | 171 | 14 | 1 |
| "remote control" | 1,613 | 298 | 4 | 3 | 0 |
| headless | 322 | 55 | 56 | 49 | 2 |

---

## Sentiment by tool

Each verdict is mine and comes from the counts and sources listed with it.

### Claude Code (CLI)

**Mixed. People love the agent and fight the terminal.**

- Rendering and scrolling are the most-reacted complaints in the whole sample. [#3648](https://github.com/anthropics/claude-code/issues/3648) (836 reactions), [#826](https://github.com/anthropics/claude-code/issues/826) (822), [#769](https://github.com/anthropics/claude-code/issues/769) (336) and [#1913](https://github.com/anthropics/claude-code/issues/1913) (321) total 2,315 reactions.
- A third-party flicker fix reached 192 points on [Hacker News](https://news.ycombinator.com/item?id=46699072). One reply notes it "breaks Ghostty's native scrollback" ([comment](https://news.ycombinator.com/item?id=46700190)).
- The April 2026 full-screen renderer fixed flicker but "completely destroys native terminal scrollback" ([#42670](https://github.com/anthropics/claude-code/issues/42670), [#41965](https://github.com/anthropics/claude-code/issues/41965)). It also broke tmux copy mode ([#60185](https://github.com/anthropics/claude-code/issues/60185)).
- Inside tmux there are separate bugs: rendering corruption ([#29937](https://github.com/anthropics/claude-code/issues/29937), 70 reactions), scrollback lag ([#4851](https://github.com/anthropics/claude-code/issues/4851), 97), and notifications that need DCS wrapping ([#19976](https://github.com/anthropics/claude-code/issues/19976), 26).
- Usage limits are the biggest cost complaint ([#16157](https://github.com/anthropics/claude-code/issues/16157), 726 reactions and 1,499 comments; [#13354](https://github.com/anthropics/claude-code/issues/13354), 215).
- On the positive side, a commenter writes that "no other agentic coding tool comes close" ([comment](https://news.ycombinator.com/item?id=45737290)). Its creator runs five sessions at once in numbered tabs ([X](https://x.com/bcherny/status/2007179833990885678)).

### Claude Code Remote Control

**Wanted, but seen as fragile and locked.**

- The launch reached 544 points and 297 comments on [Hacker News](https://news.ycombinator.com/item?id=47148454). In that thread, 63 comments talk about phones and 33 about tmux, mosh or zellij.
- One commenter called it an "extremely clunky and buggy prerelease" ([comment](https://news.ycombinator.com/item?id=47151466)). Others went back to Termux, or to self-hosted hapi with Tailscale ([comment](https://news.ycombinator.com/item?id=47150780)). One disliked that the Android app needs a GitHub connection ([comment](https://news.ycombinator.com/item?id=47150488)).
- Its top bugs are connection bugs: no automatic reconnect ([#34255](https://github.com/anthropics/claude-code/issues/34255), 109 reactions) and sessions that die after about 20 minutes idle ([#32982](https://github.com/anthropics/claude-code/issues/32982), 86). A small HN thread is titled "Anyone else having a terrible experience with Claude Code's remote control?" ([HN](https://news.ycombinator.com/item?id=47645915)).
- Users ask for it on Bedrock ([#28795](https://github.com/anthropics/claude-code/issues/28795), 135 reactions), on headless servers ([#30447](https://github.com/anthropics/claude-code/issues/30447), 46) and as a service that survives a reboot ([#88166](https://github.com/anthropics/claude-code/issues/88166)).
- The docs say it needs a Pro, Max, Team or Enterprise plan. API keys and third-party providers are not supported, and all traffic goes through the Anthropic API ([docs](https://code.claude.com/docs/en/remote-control)).

### Claude Code on the web (cloud sessions)

**Mixed. Good for quick tasks from a phone, weak for real environments.**

- The launch reached 578 points on [Hacker News](https://news.ycombinator.com/item?id=45647166) and 346 on [r/ClaudeAI](https://www.reddit.com/r/ClaudeAI/comments/1obpbbs/). A follow-up post, "I've been loving Claude Code on the web", reached 161 points ([HN](https://news.ycombinator.com/item?id=45735264)).
- The environment is the most common complaint. One commenter wrote "it doesn't have support for devcontainers" ([comment](https://news.ycombinator.com/item?id=45736685)). Another said their tools were missing and "the LLM spends 10 minutes just trying to get the environment set up... On every interaction" ([comment](https://news.ycombinator.com/item?id=45737045)). A third said "yarn install just hangs" ([comment](https://news.ycombinator.com/item?id=45649030)). Several repositories per session is an open request with 99 reactions ([#23627](https://github.com/anthropics/claude-code/issues/23627)).
- Some find the clone, change, test and PR loop "too slow of a loop" for anything but small tasks ([comment](https://news.ycombinator.com/item?id=45736850)).
- On Reddit, a top comment asked for the opposite of a cloud copy: "remotely run/continue my claude code sessions against the repos already on my PC" ([r/ClaudeAI](https://www.reddit.com/r/ClaudeAI/comments/1obpbbs/)).
- The docs require a claude.ai plan, and GitHub for cloning and pushing. A setup script is cached only if it finishes in about five minutes ([docs](https://code.claude.com/docs/en/claude-code-on-the-web)).

### Codex CLI

**Positive on speed and stability. Mixed on terminal behaviour since the switch to full screen.**

- Users compare its resource use with Claude Code's. One wrote: "I can run 5-10 codex sessions and my system holds up fine ... but 8 CC instances would grind things to a halt" ([comment](https://news.ycombinator.com/item?id=45737220)).
- Peter Steinberger reports no flicker, but says it "loses" lines when scrolling quickly ([blog](https://steipete.me/posts/just-talk-to-it)). Scrolling inside zellij truncated output ([#2558](https://github.com/openai/codex/issues/2558), 135 reactions). Native scroll in tmux broke after an update ([#48315](https://github.com/openai/codex/issues/48315)).
- The September 2026 full-screen default ([announcement](https://github.com/openai/codex/discussions/49129)) got 8 thumbs down out of 10 reactions. An issue asks to keep native terminal behaviour by default ([#48398](https://github.com/openai/codex/issues/48398)).
- A built-in `--worktree --tmux` was asked for with 129 reactions ([#12862](https://github.com/openai/codex/issues/12862)). A background daemon that starts by itself drew 198 reactions asking to make it opt-in ([#48195](https://github.com/openai/codex/issues/48195)).

### Codex cloud and Codex remote

**High demand, uneven delivery.**

- Codex cloud is the web agent that "stuck" for Peter Steinberger ([blog](https://steipete.me/posts/just-talk-to-it)). Simon Willison uses it for riskier asynchronous tasks, often started from his phone ([blog](https://simonwillison.net/2025/Oct/5/parallel-coding-agents/)).
- The remote-control request explains why people wanted more: "I cannot set up the environment from the mobile app", "I prefer using my desktop environment" ([discussion #9200](https://github.com/openai/codex/discussions/9200), 260 reactions).
- Demand for remote work on one's own machine is the largest in the sample: [#10450](https://github.com/openai/codex/issues/10450) (876 reactions) and [#9224](https://github.com/openai/codex/issues/9224) (544).
- The shipped remote has complaints. Two r/codex threads report constant disconnects and failed connections ([1](https://www.reddit.com/r/codex/comments/1wt0qt3/), [2](https://www.reddit.com/r/codex/comments/1vu8gvz/)). A reply in the first thread complains of a "session is active on another device" error after moving from the desktop to the phone. Resuming a thread can fail with "already has an active writer" ([#37403](https://github.com/openai/codex/issues/37403), 48 reactions).
- The docs say the host must run the ChatGPT desktop app on macOS or Windows, and stay awake and online. Linux hosts are not listed ([docs](https://learn.chatgpt.com/docs/remote-connections)). Users ask for headless Linux hosts ([#23200](https://github.com/openai/codex/issues/23200), 69 reactions).
- One user on r/codex says their cloud workspace restarted and lost its work ([r/codex](https://www.reddit.com/r/codex/comments/1x3wc14/)). A reply prefers "my existing VM setup in my home lab". What "Dot" in that thread refers to is **UNVERIFIED**.

### opencode

**Positive on openness. Mixed on security and terminal fidelity.**

- It is very popular: 213,734 GitHub stars and a 1,274-point launch thread ([HN](https://news.ycombinator.com/item?id=47460525)). Version 2.0 reached 703 points on 14 October 2026 ([HN](https://news.ycombinator.com/item?id=50094994)).
- Security worries are recurring: an unauthenticated remote code execution bug (432 points, [HN](https://news.ycombinator.com/item?id=46581095)) and "Annoying and alarming things about OpenCode" (420 points, [HN](https://news.ycombinator.com/item?id=48978112)).
- Lock-in hit it directly. After a lawyers' letter, opencode stopped loading the Claude Max plugin ([X](https://x.com/thdxr/status/2034730036759339100); 483 points on [HN](https://news.ycombinator.com/item?id=47444748)).
- Its remote story is `opencode serve` plus a tunnel. Users warn that quick tunnels are "ephemeral" and that "Tailscale + SSH + tmux is the most reliable setup" ([r/opencodeCLI](https://www.reddit.com/r/opencodeCLI/comments/1rfw70p/)).
- Its terminal issues match the others: clipboard ([#4283](https://github.com/anomalyco/opencode/issues/4283), 134 reactions), Shift+Enter ([#1505](https://github.com/anomalyco/opencode/issues/1505), 104), and notifications under tmux or zellij ([#29099](https://github.com/anomalyco/opencode/issues/29099)). Users also ask for SSH from the desktop app ([#7790](https://github.com/anomalyco/opencode/issues/7790), 116) and a mobile version ([#10288](https://github.com/anomalyco/opencode/issues/10288), 103).

### Gemini CLI

**Negative, because of churn and lock-in.**

- The launch was the biggest thread in the sample: 1,428 points ([HN](https://news.ycombinator.com/item?id=44376919)). The free tier was a large part of the draw.
- In March 2026 Google added abuse detection for "using Gemini CLI oAuth with third-party software", and limited free users to Flash models. Of 1,396 reactions, 1,035 were thumbs down ([discussion #22970](https://github.com/google-gemini/gemini-cli/discussions/22970)).
- In May 2026 Google announced that Gemini CLI would stop serving Pro, Ultra and free users on 18 June 2026, and moved them to Antigravity CLI ([Google blog](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/)). The announcement drew 300 thumbs down out of 374 reactions ([discussion #27274](https://github.com/google-gemini/gemini-cli/discussions/27274)) and 406 points on [HN](https://news.ycombinator.com/item?id=48196867). One commenter wrote: "google can kill your account for using an unsupported harness" ([comment](https://news.ycombinator.com/item?id=48204257)).
- The Antigravity CLI repo holds a README, a changelog and examples, with no source and no licence ([repo](https://github.com/google-antigravity/antigravity-cli)). Gemini CLI itself stays Apache-2.0 and is still updated.
- Remote pains were the usual ones: login over SSH ([#1437](https://github.com/google-gemini/gemini-cli/issues/1437), 64 reactions) and flicker in tmux ([#22004](https://github.com/google-gemini/gemini-cli/issues/22004)).

### aider

**Quiet. Respected for control, but slowing down.**

- The last release is v0.86.0, from 9 August 2025 ([releases](https://github.com/Aider-AI/aider/releases/tag/v0.86.0)). A community fork, Aider-CE, became "official" in November 2025 ([HN](https://news.ycombinator.com/item?id=45945379)). One commenter there calls aider their "Goldilocks zone" for reviewing each line.
- It has very few remote complaints: 1 issue with "tmux" in the title and 3 with "ssh". It has had notifications since March 2025 ([docs](https://aider.chat/docs/usage/notifications.html)). A request to run several instances in parallel dates from 2023 ([#302](https://github.com/Aider-AI/aider/issues/302)).
- My reading is that its line-by-line interface suits tmux well, and few people run it as a background agent. This is an inference, not a count.

### Amp

**Mixed. Covered in detail in [Amp UX and Orbs](amp-ux-and-orbs.md).**

- In short, the Orbs thread (116 points, [HN](https://news.ycombinator.com/item?id=49437117)) splits between "the best $20/month" and "a thin wrapper over a remote virtual machine". One commenter asks for plain OpenSSH. Price is the most common complaint in that note.

### Cursor background agents (Cloud Agents)

**Negative to mixed.**

- When Background Agent reached general availability with Cursor 1.0, r/cursor was sceptical ([thread](https://www.reddit.com/r/cursor/comments/1l3wwtr/)). One reply said it could not be used in privacy mode. A May 2026 post saying it "actually works now" drew replies that doubt it ([thread](https://www.reddit.com/r/cursor/comments/1t81b3a/)).
- On Hacker News, one user said Cursor's agents "would make entirely new mistakes" that the local IDE did not ([comment](https://news.ycombinator.com/item?id=45736734)).
- In June 2026, installing the Cursor iOS app switched one user's account to a softer privacy mode, whose terms allow storing code "for Background Agents". The post reached 249 points ([HN](https://news.ycombinator.com/item?id=48737226)). Replies say the mobile app "cannot start an agent session on your computer" and that cloud agents are "significantly more limiting" ([1](https://news.ycombinator.com/item?id=48737982), [2](https://news.ycombinator.com/item?id=48738246)).
- Cursor's docs say Cloud Agents need the Premium plan or higher ([pricing](https://cursor.com/docs/account/pricing)). They are the only feature for which Cursor keeps encrypted copies of the repository while the agent runs ([privacy docs](https://cursor.com/docs/enterprise/privacy-and-data-governance)).

### tmux, zellij and mosh setups

**Trusted and robust, but fiddly and lossy.**

- This is the default answer. tmux comes up in 35 of the 63 HN and Reddit threads I read (178 comments), Tailscale in 24, mosh in 17, Termux in 14, Termius in 9 and zellij in 8. The r/tmux subreddit complained that it had become "just claude code tmux plugins" (110 points, [thread](https://www.reddit.com/r/tmux/comments/1saciat/)).
- People trust it more than vendor relays. One HN commenter runs "the agent in tmux in a colo" and uses Remote Control only on the go ([comment](https://news.ycombinator.com/item?id=47150668)). Another, under a mobile-agent launch, asked: "So you've reinvented SSH+screen except slower and much less flexible?" ([comment](https://news.ycombinator.com/item?id=44879312)).
- What it loses:
  - Scrollback under mosh: "Mosh paints the screen, meaning no native scrollback" ([comment](https://news.ycombinator.com/item?id=45054124)). The mosh issue is open since 2012 with 280 reactions ([#122](https://github.com/mobile-shell/mosh/issues/122)). A Rust port with scrollback appeared on 14 October 2026 ([HN](https://news.ycombinator.com/item?id=50080836)).
  - Clipboard under mosh: no OSC 52 ([#637](https://github.com/mobile-shell/mosh/issues/637)). One user said this "sent me back to tmux" ([comment](https://news.ycombinator.com/item?id=45053868)). Over mosh, repeated copies of the same text are deduplicated ([Claude Code #89030](https://github.com/anthropics/claude-code/issues/89030)).
  - Keys under tmux: the kitty keyboard pull request was closed unmerged ([#4912](https://github.com/tmux/tmux/pull/4912)). The roadmap lists kitty keys as "potentially" 3.9, and save and restore as "probably 4.0" ([#5406](https://github.com/tmux/tmux/issues/5406)). A newer kitty keyboard pull request was still open on 15 October 2026 ([#5615](https://github.com/tmux/tmux/pull/5615), S156). Shift+Enter needs extended keys inside tmux ([Codex #21699](https://github.com/openai/codex/issues/21699)).
  - Phones: "tmux keybindings can be awkward on a phone keyboard" ([Twingate](https://www.twingate.com/blog/claude-code-termius-tmux)). Users remap Shift+Tab by hand ([r/ClaudeAI](https://www.reddit.com/r/ClaudeAI/comments/1l96zlw/)). They also report that "Mosh has lots of visual artifacts" (same thread).
  - Reboots: "Such a pain to get back to before a reboot" ([comment](https://news.ycombinator.com/item?id=50082557)).

---

## Setup pains

### The usual stack

These are the parts a developer joins together to get a persistent remote agent today. The count is how many of the 63 HN and Reddit threads name each one.

| Part | Typical choice | Threads naming it | What breaks |
| --- | --- | --- | --- |
| A machine that stays on | VPS (Hetzner, DigitalOcean), a Mac mini, a home server | (see `own-host`: 37 sources) | Laptops sleep. Users run `caffeinate` or turn off sleep (6 threads) |
| A private network | Tailscale (24), Twingate, WireGuard | 24 for Tailscale | One more account and client on each device. Quick tunnels change their URL on restart (4 threads name cloudflared or ngrok) |
| A link that survives drops | mosh (17) or plain SSH | 17 for mosh | mosh loses scrollback and OSC 52, and shows artefacts |
| A session that outlives the link | tmux (35), zellij (8), screen (5) | 35 for tmux | Agent rendering bugs inside tmux, blocked notifications, and "its own world of key bindings" ([S111](https://www.reddit.com/r/ClaudeAI/comments/1l96zlw/)) |
| A terminal on the phone | Termux (14), Termius (9), Blink (6), Echo | 14 for Termux | Missing keys (Shift+Tab, backtick), small screen, hard copy and paste |
| Alerts | ntfy, Pushover, Telegram, Home Assistant (6) | 6 | Built by hand from agent hooks |
| Agent login | The vendor's login flow on a box with no browser | (see `login`) | Copy a code between devices. Tokens cannot be moved from the laptop |

### Steps and time

| Guide | Steps | Tools |
| --- | --- | --- |
| Reddit post on moving Claude Code to a remote box ([S110](https://www.reddit.com/r/ClaudeAI/comments/1w5bx9x/)) | 5 parts, then a custom CLI so the team could repeat it | Hetzner, Tailscale, mosh, tmux, Tailscale Funnel |
| Twingate guide ([S135](https://www.twingate.com/blog/claude-code-termius-tmux)) | 4 steps, one of them with 4 sub-steps, plus tmux mouse mode | macOS Remote Login, Twingate, tmux, Termius |
| Product Compass VPS guide ([S134](https://www.productcompass.pm/p/claude-code-vps)) | 5 steps: user, install, login, tmux or systemd, repo access | VPS, installer, tmux or systemd, GitHub CLI or a deploy key |
| Ask HN answer ([S93](https://news.ycombinator.com/item?id=45787595)) | 3 steps | Tailscale, tmux, Termux |

- **Time:** no source measured setup time. One HN commenter guessed "a couple of hours" when calling a paid relay "expensive" ([comment](https://news.ycombinator.com/item?id=46992372)). That is one claim, not a measure.
- **Repeatability:** the Reddit author built a CLI because sharing the setup with a team "was challenging" ([S110](https://www.reddit.com/r/ClaudeAI/comments/1w5bx9x/)). Codeman (176 points on [r/selfhosted](https://www.reddit.com/r/selfhosted/comments/1vebymy/)) and Herdr (44,140 stars) exist to package the same job.

### What breaks, with evidence

| Break | Evidence | Count |
| --- | --- | --- |
| Logging the agent in on a headless box | [Claude Code #22992](https://github.com/anthropics/claude-code/issues/22992), [Codex #2798](https://github.com/openai/codex/issues/2798), [Gemini CLI #1437](https://github.com/google-gemini/gemini-cli/issues/1437). Anthropic's terms say "sign-in to a Claude account must complete through Anthropic's own flow" ([S138](https://code.claude.com/docs/en/legal-and-compliance)), so a token cannot be copied from the laptop ([S134](https://www.productcompass.pm/p/claude-code-vps)). | 3 issues, 245 reactions |
| The agent dies when the client disconnects | Claude Desktop's SSH remote: "the Claude Code process on the remote server terminates with it" ([#49790](https://github.com/anthropics/claude-code/issues/49790)). Without tmux, an SSH drop can kill the process ([S134](https://www.productcompass.pm/p/claude-code-vps)). | 47 reactions, plus 2 guides |
| The agent's own UI under tmux, zellij or mosh | [S8](https://github.com/anthropics/claude-code/issues/4851), [S9](https://github.com/anthropics/claude-code/issues/29937), [S42](https://github.com/openai/codex/issues/2558), [S54](https://github.com/openai/codex/issues/48315), [S66](https://github.com/google-gemini/gemini-cli/issues/22004), [S75](https://github.com/mobile-shell/mosh/issues/122) | 6 issues, 602 reactions |
| Notifications inside a multiplexer | [Claude Code #19976](https://github.com/anthropics/claude-code/issues/19976), [opencode #29099](https://github.com/anomalyco/opencode/issues/29099) | 2 issues |
| Cloud sandbox lacks the developer's tools | HN comments on devcontainers, mise and yarn ([S80](https://news.ycombinator.com/item?id=45647166), [S81](https://news.ycombinator.com/item?id=45735264)); several repos per session ([#23627](https://github.com/anthropics/claude-code/issues/23627), 99) | 7 `own-env` sources, 1,235 reactions |
| Vendor relay drops or idles out | [#34255](https://github.com/anthropics/claude-code/issues/34255), [#32982](https://github.com/anthropics/claude-code/issues/32982), two r/codex threads | 2 issues with 195 reactions, plus 2 threads |
| The host must be a desktop that stays awake | Codex remote needs the ChatGPT desktop app on macOS or Windows ([docs](https://learn.chatgpt.com/docs/remote-connections)); [#23200](https://github.com/openai/codex/issues/23200) | 69 reactions |
| Seeing what the agent built | "I'm flying blind, approving changes to a button I've never seen" ([HN](https://news.ycombinator.com/item?id=47645915)); auto-forward ports ([Codex #21294](https://github.com/openai/codex/issues/21294)) | 5 `ports` sources |

---

## Paywalled and account-locked features

These are features that developers say they want, and that a vendor keeps behind a paid plan, a vendor account, a vendor relay or a vendor platform.

| Feature | Where it is locked | What it needs | Evidence that developers want it open or free |
| --- | --- | --- | --- |
| Steer a local session from a phone | Claude Code Remote Control | Pro, Max, Team or Enterprise. No API keys, no Bedrock, Vertex or Foundry. Traffic goes through the Anthropic API ([docs](https://code.claude.com/docs/en/remote-control)). | Bedrock and SSO request, 135 reactions ([#28795](https://github.com/anthropics/claude-code/issues/28795)); headless request, 46 ([#30447](https://github.com/anthropics/claude-code/issues/30447)); free tools with stars: Happy 24,155, Paseo 20,967, CloudCLI 14,106, hapi 5,244 |
| Steer Codex from a phone; remote SSH projects | Codex remote | A ChatGPT account with Codex access. The host runs the ChatGPT desktop app on macOS or Windows ([docs](https://learn.chatgpt.com/docs/remote-connections)). | Linux hosts, 69 reactions ([#23200](https://github.com/openai/codex/issues/23200)). Third-party reports say the mobile preview reached all plans, including Free (**UNVERIFIED**). |
| Agent sessions that run while the laptop is closed | Claude Code cloud sessions | A claude.ai plan. GitHub to clone and push. Shares the plan's usage limits ([docs](https://code.claude.com/docs/en/claude-code-on-the-web)). | Several repos per session, 99 reactions ([#23627](https://github.com/anthropics/claude-code/issues/23627)); "run/continue my claude code sessions against the repos already on my PC" ([r/ClaudeAI](https://www.reddit.com/r/ClaudeAI/comments/1obpbbs/)) |
| Background agents in a cloud VM | Cursor Cloud Agents | Premium plan or higher ([pricing](https://cursor.com/docs/account/pricing)). Code is stored while the agent runs ([privacy docs](https://cursor.com/docs/enterprise/privacy-and-data-governance)). | Privacy mode conflict, 249 points ([HN](https://news.ycombinator.com/item?id=48737226)) |
| A sandbox VM per task | Amp Orbs | Pay per minute, an Amp account (see [Amp UX and Orbs](amp-ux-and-orbs.md)) | "Just give me OpenSSH" (in that note) |
| A managed relay to a phone | Omnara | A subscription. A commenter puts it at about $20 a month on top of a $100 Claude plan (**UNVERIFIED** price). No end-to-end encryption yet, by Omnara's own account ([comment](https://news.ycombinator.com/item?id=46992557)). | "Feels expensive for something that an engineer can hack in a couple of hours" ([comment](https://news.ycombinator.com/item?id=46992372)) |
| Use a subscription in a third-party client | Anthropic (Claude Max), Google (Gemini CLI OAuth) | First-party clients only. Anthropic does not let third-party developers "route requests through Free, Pro, or Max plan credentials on behalf of their users", or "collect, store, or intermediate Claude.ai credentials" ([terms](https://code.claude.com/docs/en/legal-and-compliance)). Google detects "using Gemini CLI oAuth with third-party software" ([#22970](https://github.com/google-gemini/gemini-cli/discussions/22970)). | 483 points ([HN](https://news.ycombinator.com/item?id=47444748)); 1,035 thumbs down ([#22970](https://github.com/google-gemini/gemini-cli/discussions/22970)) |
| The open-source Gemini terminal agent on a consumer plan | Google | From 18 June 2026, Pro, Ultra and free users must use Antigravity CLI, whose repo has no source code ([repo](https://github.com/google-antigravity/antigravity-cli)). Gemini CLI still works with paid API keys ([Google blog](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/)). | 300 thumbs down ([#27274](https://github.com/google-gemini/gemini-cli/discussions/27274)); 406 points ([HN](https://news.ycombinator.com/item?id=48196867)) |
| Phone alerts for approvals and finished tasks | Claude app (Remote Control); ChatGPT app (Codex) | The vendor's app and account | Push for approvals, 69 reactions ([#29438](https://github.com/anthropics/claude-code/issues/29438)); home-made ntfy and Home Assistant bridges ([HN](https://news.ycombinator.com/item?id=47111171), [comment](https://news.ycombinator.com/item?id=47150647)) |
| Headless and scripted use on a plan | Anthropic | Since 15 June 2026, paid plans get a separate monthly credit for `claude -p` and the Agent SDK ([X](https://x.com/ClaudeDevs/status/2054610152817619388)) | Headless runs use about 1.8 times more of the 5-hour window ([#97074](https://github.com/anthropics/claude-code/issues/97074)) |

The pattern is clear. Each vendor ships phone access and background sessions for its own agent, through its own relay, on its own paid account. The open-source tools that do the same job for any agent are among the fastest-growing repos in this space: Herdr (44,140 stars since March 2026), Happy (24,155), Paseo (20,967), Superset (15,219) and CloudCLI (14,106).

---

## What this means for simstim

The table maps the top needs to what simstim does today and to candidate features. Demand is the number of voice sources, then the GitHub reactions. "Built" refers to the v1 and v2 work in the [README](../../README.md) and [the plan](../plan.md).

| Rank | Need | Demand (sources / reactions) | simstim today | Candidate feature |
| --- | --- | --- | --- | --- |
| 1 | mobile-access | 40 / 1,373 | Not a goal: no phone client | **Phone path without a new client.** Make the host helper usable straight from any phone SSH app (Termius, Blink, Echo, Termux): a small session picker on the host, then attach. This keeps the non-goal. |
| 2 | run-on-own-server | 37 / 2,261 | Built: any host reachable by `ssh` | Say plainly that Linux hosts work, which Codex remote does not support ([S140](https://learn.chatgpt.com/docs/remote-connections)) |
| 3 | cost | 21 / 2,318 | Free, no account; v2 provider scripts stop idle VMs | **Continue when the limit resets**: `simstim send --at <time>` or "after the limit resets" (Claude Code #13354 and Codex #21073: 291 reactions). An optional cost hint per cloud host. |
| 4 | parallel-agents | 18 / 283 | Built: many sessions, `flip`, search across hosts | **A "who needs me" jump**: one key goes to the next session that waits for input, on any host (S117: 691 points; S128) |
| 5 | scrollback | 17 / 2,302 | Built: byte-exact pass-through, replay into native scrollback | Per-agent hints in the agent table for inline mode (for example Codex `--no-alt-screen`, which issue titles mention). `doctor terminal` warns when an agent runs full screen. simstim cannot restore scrollback inside an agent's own alternate screen. |
| 6 | lock-in/any-agent | 16 / 1,470 (about 132,000 stars on multi-agent tools) | Built: agents are rows in a table | Add rows for Antigravity CLI, Pi, Cursor CLI and Copilot CLI. Say in the docs that simstim runs each vendor's own, unmodified CLI, and that the user logs in through the vendor's own flow inside the session. simstim must never collect, store or pass on an agent's credentials ([S138](https://code.claude.com/docs/en/legal-and-compliance)). |
| 7 | sandbox-isolation | 15 / no GitHub items | Partly: memory caps, worktrees | An optional isolation level per session (a separate Unix user, or systemd sandboxing), for agents run without approvals |
| 8 | reliable-reconnect | 14 / 242 | Built: heartbeat, reconnect with backoff, replay | Show the link state and the replay size after a reconnect |
| 9 | notify-when-input-needed | 13 / 225 | Built: desktop alert when another session needs you | **A host-side alert hook**: a command per host (for example `curl` to the user's own ntfy topic) so alerts reach a phone with no simstim relay |
| 10 | tui-fidelity | 12 / 2,655 | Built: no re-rendering, no multiplexer | Keep the fixture tests for each agent's full-screen mode. Publish the fidelity contract as a selling point. |
| 11 | setup-time | 12 / 389 | Built: `host add` wizard and helper install | Target and measure "first session in N minutes" on a fresh VPS |
| 12 | persist-through-sleep | 11 / 162 | Built | None |
| 13 | clipboard | 10 / 502 | Built: OSC 52 passes through | Test OSC 52 in each agent's full-screen mode |
| 14 | worktrees | 10 / 216 | Built: `new --worktree` | Choose the base branch (Claude Code #23622: 87 reactions) |
| 15 | trust/no-relay | 10 / 198 | Built: nothing listens, no relay, no telemetry | Keep host units opt-in with consent (Codex #48195: 198 reactions against an automatic daemon) |
| 16 | key-fidelity | 8 / 251 | Built: kitty keyboard protocol passes through | None |
| 17 | own-environment | 7 / 1,235 | Built: the agent runs in the user's login-shell environment | None |
| 18 | resume-after-reboot | 6 / 226 | Built in v2: restore with each agent's resume command | None |
| 19 | image-paste | 6 / 175 | Built: Ctrl+V uploads and pastes the remote path | None |
| 20 | headless-login | 5 / 245 | Not covered | **Agent checks at host setup**: is each agent installed, on `PATH` and logged in? If not, show the vendor's own login command to run in a session. |
| 21 | port-preview | 5 / 14 | Built: `ports`, `forward` | None |
| 23 | multi-device-attach | 4 / 51 | Planned for v1.x: several devices, a `take_size` rule | Make laptop and phone attach together a tested case |

Needs not to build, from this evidence:

- **Sharing and watching** (0 voice sources), **snapshots** (1) and **secrets management** (3) have little demand here. Secrets also conflict with simstim's rule of storing no secrets.
- **Voice input** (5) belongs to the phone's keyboard, not to simstim.
- **A web or phone UI** would meet the top need, but the evidence also warns against it. The sample has an unauthenticated remote code execution bug in opencode's server (S102), a widely shared critique of opencode's security (S103), and a mobile app that changed a user's privacy setting (S108). The plan already rules a web or phone UI out. The phone path above reaches the same users through plain SSH.

---

## Unverified and open points

- **Reddit (UNVERIFIED against the live site):** all Reddit text and scores come from the Arctic Shift archive, not from reddit.com, which needs a login. The live pages may differ. Scores can be lower than today's. Two threads I fetched had been removed by moderators, and one more had nothing on topic. None of the three is used.
- **X:** I read X posts through X's oEmbed endpoint, which returns the post text. I could not see replies or view counts.
- **Codex "Dot" and "Dots" (UNVERIFIED):** the r/codex thread (S123) uses the word for a cloud workspace. I could not confirm what the product is.
- **Codex mobile plans (UNVERIFIED):** third-party sites say the preview reached all plans, including Free. OpenAI's page does not say.
- **Omnara's price (UNVERIFIED):** taken from a commenter, not from Omnara.
- **Claude Code full-screen opt-out (UNVERIFIED):** I did not confirm which setting turns the full-screen renderer off in the current version.
- **Setup time:** no source measured it.
- **Talk date (UNVERIFIED):** the talk page names AI Engineer World's Fair 2025, not the month. I give June 2025 from secondary sources.
- **Usage terms:** Anthropic says the advertised Pro and Max limits "assume ordinary, individual usage" ([S138](https://code.claude.com/docs/en/legal-and-compliance)). Whether many parallel, always-on sessions fit that is for Anthropic to say, not this note.
- **Selection bias:** I chose the queries. A different set of queries would change the counts, but probably not the order of the top five, because those lead on both source counts and reactions.

---

## Sources

All sources were read on 15 October 2026. Reactions and comment counts are as of that day. "Vendor" rows (docs and vendor posts) and "Repo" rows are not counted as demand in the ranking.

| # | Source (evidence) | Type | Date | Read | Link | Needs |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Claude Code #826: view jumps to the top when Claude adds text (822 reactions, 354 comments) | GH issue | 2025-04-19 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/826) | `render`, `scrollback` |
| 2 | Claude Code #3648: terminal scrolls uncontrollably (836 reactions, 337 comments) | GH issue | 2025-07-16 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/3648) | `render`, `scrollback` |
| 3 | Claude Code #769: screen flickers during a call (336 reactions, 307 comments) | GH issue | 2025-04-12 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/769) | `render` |
| 4 | Claude Code #1913: terminal flickering (321 reactions, 187 comments) | GH issue | 2025-06-10 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/1913) | `render` |
| 5 | Claude Code #18170: copied text includes indentation and trailing spaces (300 reactions, 138 comments) | GH issue | 2026-01-14 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/18170) | `clipboard` |
| 6 | Claude Code #42670: alternate screen kills all scrollback (v2.1.89+) (53 reactions, 16 comments) | GH issue | 2026-04-02 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/42670) | `scrollback` |
| 7 | Claude Code #41965: flicker-free rendering destroys scrollback (31 reactions, 17 comments) | GH issue | 2026-04-01 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/41965) | `render`, `scrollback` |
| 8 | Claude Code #4851: scrollback lag in tmux after long use (97 reactions, 28 comments) | GH issue | 2025-07-31 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/4851) | `scrollback`, `render` |
| 9 | Claude Code #29937: rendering corruption in tmux (70 reactions, 28 comments) | GH issue | 2026-03-02 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/29937) | `render` |
| 10 | Claude Code #19976: notifications do not work inside tmux (26 reactions, 12 comments) | GH issue | 2026-01-22 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/19976) | `notify` |
| 11 | Claude Code #5277: image paste over SSH (33 reactions, 25 comments) | GH issue | 2025-08-06 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/5277) | `image`, `own-host` |
| 12 | Claude Code #81472: copy and paste broken, tracking 42 open issues (14 reactions, 7 comments) | GH issue | 2026-07-26 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/81472) | `clipboard` |
| 13 | Claude Code #1282: Shift+Enter in Ghostty (90 reactions, 25 comments) | GH issue | 2025-05-24 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/1282) | `keys` |
| 14 | Claude Code #49790: Desktop SSH session should survive a client disconnect (47 reactions, 17 comments) | GH issue | 2026-04-17 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/49790) | `sleep`, `reconnect`, `own-host` |
| 15 | Claude Code #34255: Remote Control does not reconnect (109 reactions, 72 comments) | GH issue | 2026-03-14 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/34255) | `reconnect`, `mobile` |
| 16 | Claude Code #32982: Remote Control sessions die after about 20 minutes idle (86 reactions, 21 comments) | GH issue | 2026-03-10 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/32982) | `reconnect`, `mobile` |
| 17 | Claude Code #30447: headless, daemon Remote Control for servers (46 reactions, 9 comments) | GH issue | 2026-03-03 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/30447) | `own-host`, `sleep`, `mobile` |
| 18 | Claude Code #29438: push notifications on iOS for approvals (69 reactions, 15 comments) | GH issue | 2026-02-27 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/29438) | `notify`, `mobile` |
| 19 | Claude Code #28795: Remote Control with Bedrock and AWS SSO (135 reactions, 14 comments) | GH issue | 2026-02-26 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/28795) | `any-agent` |
| 20 | Claude Code #22992: device-code login for headless machines (39 reactions, 19 comments) | GH issue | 2026-02-04 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/22992) | `login`, `own-host` |
| 21 | Claude Code #24122: Agent Teams in zellij, not only tmux (95 reactions, 11 comments) | GH issue | 2026-02-08 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/24122) | `parallel` |
| 22 | Claude Code #23622: choose the base branch of a worktree (87 reactions, 22 comments) | GH issue | 2026-02-06 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/23622) | `worktree` |
| 23 | Claude Code #13354: continue when the session limit is reached (215 reactions, 80 comments) | GH issue | 2025-12-08 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/13354) | `cost` |
| 24 | Claude Code #23627: several repositories in web sessions (99 reactions, 15 comments) | GH issue | 2026-02-06 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/23627) | `own-env` |
| 25 | Claude Code #88166: Remote Control as a service that survives reboot (4 reactions, 1 comment) | GH issue | 2026-08-20 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/88166) | `reboot`, `own-host` |
| 26 | Claude Code #57543: SSH or mosh from the mobile app (0 reactions, 2 comments) | GH issue | 2026-05-09 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/57543) | `mobile`, `own-host` |
| 27 | Claude Code #13738: clipboard image paste does not work in WSL (44 reactions, 35 comments) | GH issue | 2025-12-11 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/13738) | `image` |
| 28 | Claude Code #1361: cannot paste an image from the clipboard (50 reactions, 70 comments) | GH issue | 2025-05-27 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/1361) | `image` |
| 29 | Claude Code #50982: remote SSH session loses its history after a reboot (7 reactions, 9 comments) | GH issue | 2026-04-20 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/50982) | `reboot`, `own-host` |
| 30 | Claude Code #5768: resume works only from the starting folder (61 reactions, 24 comments) | GH issue | 2025-08-14 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/5768) | `reboot` |
| 31 | Claude Code #40310: attach a terminal to a phone-started session (3 reactions, 5 comments) | GH issue | 2026-03-28 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/40310) | `multi-device`, `mobile` |
| 32 | Claude Code #16157: hitting usage limits at once on the Max plan (726 reactions, 1,499 comments) | GH issue | 2026-01-03 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/16157) | `cost` |
| 33 | Claude Code #97074: headless claude -p uses about 1.8 times more of the 5-hour window (6 reactions, 8 comments) | GH issue | 2026-09-25 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/97074) | `cost` |
| 34 | Claude Code #29214: Remote Control shows permission prompts on the phone despite skip-permissions (84 reactions, 32 comments) | GH issue | 2026-02-27 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/29214) | `mobile` |
| 35 | Claude Code #60185: new scroll UI breaks tmux copy mode (24 reactions, 4 comments) | GH issue | 2026-05-18 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/60185) | `scrollback`, `clipboard` |
| 36 | Claude Code #89030: over mosh, OSC 52 copy works only once per string (0 reactions, 0 comments) | GH issue | 2026-08-23 | 2026-10-15 | [link](https://github.com/anthropics/claude-code/issues/89030) | `clipboard` |
| 37 | Codex #10450: remote development (SSH) in the desktop app (876 reactions, 179 comments) | GH issue | 2026-02-03 | 2026-10-15 | [link](https://github.com/openai/codex/issues/10450) | `own-host`, `own-env` |
| 38 | Codex #9224: remote control from the phone (544 reactions, 59 comments) | GH issue | 2026-01-14 | 2026-10-15 | [link](https://github.com/openai/codex/issues/9224) | `mobile`, `own-host` |
| 39 | Codex discussion #9200: remote control Codex from the ChatGPT app (194 upvotes, 260 reactions, 51 comments) | GH discussion | 2026-01-14 | 2026-10-15 | [link](https://github.com/openai/codex/discussions/9200) | `mobile`, `own-host`, `own-env`, `cost`, `setup` |
| 40 | Codex #23200: headless Linux hosts for Codex mobile (69 reactions, 23 comments) | GH issue | 2026-05-17 | 2026-10-15 | [link](https://github.com/openai/codex/issues/23200) | `own-host`, `sleep`, `mobile` |
| 41 | Codex #12862: --worktree and --tmux flags (129 reactions, 24 comments) | GH issue | 2026-02-26 | 2026-10-15 | [link](https://github.com/openai/codex/issues/12862) | `worktree`, `parallel`, `setup` |
| 42 | Codex #2558: output truncated when scrolling in zellij (135 reactions, 66 comments) | GH issue | 2025-08-21 | 2026-10-15 | [link](https://github.com/openai/codex/issues/2558) | `scrollback`, `render` |
| 43 | Codex #2798: remote and headless sign-in (142 reactions, 49 comments) | GH issue | 2025-08-27 | 2026-10-15 | [link](https://github.com/openai/codex/issues/2798) | `login`, `own-host` |
| 44 | Codex #21294: forward remote dev-server ports (14 reactions, 1 comment) | GH issue | 2026-05-06 | 2026-10-15 | [link](https://github.com/openai/codex/issues/21294) | `ports`, `own-host` |
| 45 | Codex #21073: resume automatically when the usage limit resets (76 reactions, 19 comments) | GH issue | 2026-05-04 | 2026-10-15 | [link](https://github.com/openai/codex/issues/21073) | `cost` |
| 46 | Codex #48195: make the background daemon opt-in (198 reactions, 19 comments) | GH issue | 2026-09-25 | 2026-10-15 | [link](https://github.com/openai/codex/issues/48195) | `trust` |
| 47 | Codex #48398: keep native terminal behaviour by default; fullscreen opt-in (3 reactions, 5 comments) | GH issue | 2026-09-26 | 2026-10-15 | [link](https://github.com/openai/codex/issues/48398) | `scrollback`, `clipboard` |
| 48 | Codex discussion #49129: Codex CLI goes fullscreen (announcement; user reactions) (8 of 10 reactions are thumbs down, 7 comments) | GH discussion | 2026-09-29 | 2026-10-15 | [link](https://github.com/openai/codex/discussions/49129) | `scrollback` |
| 49 | Codex #4306: notify when a task completes (19 reactions, 1 comment) | GH issue | 2025-09-26 | 2026-10-15 | [link](https://github.com/openai/codex/issues/4306) | `notify` |
| 50 | Codex #3939: paste images from Windows into WSL (12 reactions, 25 comments) | GH issue | 2025-09-19 | 2026-10-15 | [link](https://github.com/openai/codex/issues/3939) | `image` |
| 51 | Codex #4163: named sessions for --resume (133 reactions, 6 comments) | GH issue | 2025-09-24 | 2026-10-15 | [link](https://github.com/openai/codex/issues/4163) | `reboot` |
| 52 | Codex #37403: cannot resume a thread, 'already has an active writer' (48 reactions, 65 comments) | GH issue | 2026-08-07 | 2026-10-15 | [link](https://github.com/openai/codex/issues/37403) | `multi-device` |
| 53 | Codex #21699: Shift+Enter not seen inside tmux (4 reactions, 3 comments) | GH issue | 2026-05-08 | 2026-10-15 | [link](https://github.com/openai/codex/issues/21699) | `keys` |
| 54 | Codex #48315: tmux native scroll broken after update (13 reactions, 5 comments) | GH issue | 2026-09-26 | 2026-10-15 | [link](https://github.com/openai/codex/issues/48315) | `scrollback` |
| 55 | opencode #7790: SSH connections from the desktop app (116 reactions, 20 comments) | GH issue | 2026-01-11 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/7790) | `own-host` |
| 56 | opencode #10288: a mobile version (103 reactions, 15 comments) | GH issue | 2026-01-23 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/10288) | `mobile` |
| 57 | opencode #4283: copy to clipboard not working (134 reactions, 147 comments) | GH issue | 2025-11-13 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/4283) | `clipboard` |
| 58 | opencode #15907: clipboard over SSH and tmux in Ghostty (12 reactions, 10 comments) | GH issue | 2026-03-03 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/15907) | `clipboard` |
| 59 | opencode #1505: Shift+Enter does not work (104 reactions, 128 comments) | GH issue | 2025-08-01 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/1505) | `keys` |
| 60 | opencode #213: notifications (30 reactions, 7 comments) | GH issue | 2025-06-19 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/213) | `notify` |
| 61 | opencode #29099: notifications do not fire under zellij or tmux (2 reactions, 8 comments) | GH issue | 2026-05-24 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/29099) | `notify` |
| 62 | opencode #906: paste to attach an image (22 reactions, 37 comments) | GH issue | 2025-07-12 | 2026-10-15 | [link](https://github.com/anomalyco/opencode/issues/906) | `image` |
| 63 | Gemini CLI #1437: Google login fails over SSH, headless (64 reactions, 49 comments) | GH issue | 2025-06-25 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/issues/1437) | `login`, `own-host` |
| 64 | Gemini CLI #4310: sound notifications (79 reactions, 13 comments) | GH issue | 2025-07-16 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/issues/4310) | `notify` |
| 65 | Gemini CLI #4168: autonomous background agents (56 reactions, 15 comments) | GH issue | 2025-07-14 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/issues/4168) | `parallel` |
| 66 | Gemini CLI #22004: spinner flicker inside tmux (7 reactions, 7 comments) | GH issue | 2026-03-11 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/issues/22004) | `render` |
| 67 | Gemini CLI #1452: image paste support (14 reactions, 4 comments) | GH issue | 2025-06-25 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/issues/1452) | `image` |
| 68 | Gemini CLI discussion #22970: abuse detection, third-party OAuth, Pro models paid only (announcement; user reactions) (1,396 reactions, 1,035 of them thumbs down, 401 comments) | GH discussion | 2026-03-18 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/discussions/22970) | `any-agent`, `cost` |
| 69 | Gemini CLI discussion #27274: Gemini CLI moves to Antigravity CLI (announcement; user reactions) (374 reactions, 300 of them thumbs down, 82 comments) | GH discussion | 2026-05-19 | 2026-10-15 | [link](https://github.com/google-gemini/gemini-cli/discussions/27274) | `any-agent` |
| 70 | google-antigravity/antigravity-cli: README, changelog and examples, no source, no licence (2,561 stars) | Repo | 2026-05-13 | 2026-10-15 | [link](https://github.com/google-antigravity/antigravity-cli) | `any-agent` |
| 71 | aider #302: several aider instances in parallel (3 reactions, 8 comments) | GH issue | 2023-10-31 | 2026-10-15 | [link](https://github.com/Aider-AI/aider/issues/302) | `parallel` |
| 72 | aider docs: notifications when aider waits for input (page added 6 March 2025) | Vendor | 2025-03-06 | 2026-10-15 | [link](https://aider.chat/docs/usage/notifications.html) | `notify` |
| 73 | aider releases: last release v0.86.0 (no release since) | Repo | 2025-08-09 | 2026-10-15 | [link](https://github.com/Aider-AI/aider/releases/tag/v0.86.0) | none (sentiment only) |
| 74 | The new Aider-CE fork of aider is now official (14 points, 3 comments) | HN | 2025-11-16 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45945379) | none (sentiment only) |
| 75 | mosh #122: mosh prevents the use of scrollback (280 reactions, 118 comments, still open) | GH issue | 2012-04-10 | 2026-10-15 | [link](https://github.com/mobile-shell/mosh/issues/122) | `scrollback` |
| 76 | mosh #637: mosh does not pass OSC 52 to the client (15 reactions, 14 comments, still open) | GH issue | 2015-06-17 | 2026-10-15 | [link](https://github.com/mobile-shell/mosh/issues/637) | `clipboard` |
| 77 | tmux PR #4912: kitty keyboard protocol (closed, not merged) (32 reactions, 74 comments) | GH PR | 2026-03-05 | 2026-10-15 | [link](https://github.com/tmux/tmux/pull/4912) | `keys` |
| 78 | tmux #5406: 3.9 and 4.0 roadmap (kitty keys, save and restore state) (21 reactions, 5 comments) | GH issue | 2026-07-17 | 2026-10-15 | [link](https://github.com/tmux/tmux/issues/5406) | `keys`, `reboot` |
| 79 | Claude Code Remote Control (544 points, 297 comments) | HN | 2026-02-25 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47148454) | `mobile`, `own-host`, `notify`, `reconnect`, `any-agent`, `setup`, `trust`, `voice` |
| 80 | Claude Code on the web (578 points, 390 comments) | HN | 2025-10-20 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45647166) | `mobile`, `sandbox`, `own-env`, `own-host`, `parallel`, `trust`, `cost` |
| 81 | I've been loving Claude Code on the web (161 points, 114 comments) | HN | 2025-10-28 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45735264) | `own-env`, `mobile`, `sandbox`, `setup` |
| 82 | Show HN: Omnara, run Claude Code from anywhere (310 points, 168 comments) | HN | 2025-08-12 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=44878650) | `mobile`, `notify`, `own-host`, `trust`, `setup`, `voice`, `sleep` |
| 83 | Launch HN: Omnara, run Claude Code and Codex from anywhere (147 points, 161 comments) | HN | 2026-02-12 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=46991591) | `mobile`, `own-host`, `notify`, `trust`, `cost`, `sandbox`, `voice`, `sleep`, `any-agent`, `multi-device` |
| 84 | Claude Chill: fix Claude Code's flickering (192 points, 147 comments) | HN | 2026-01-20 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=46699072) | `render`, `scrollback` |
| 85 | Show HN: Echo, an iOS SSH and mosh client built on Ghostty (122 points, 75 comments) | HN | 2026-02-18 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47064787) | `mobile`, `keys` |
| 86 | Show HN: mosh-rs, mosh in Rust with scrollback (20 points, 10 comments) | HN | 2026-10-14 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=50080836) | `scrollback`, `reboot` |
| 87 | Mosh, the mobile shell (155 points, 78 comments) | HN | 2025-08-28 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45053040) | `scrollback`, `clipboard`, `reconnect`, `mobile` |
| 88 | Parallel coding agents with tmux and Markdown specs (189 points, 122 comments) | HN | 2026-03-02 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47218318) | `parallel`, `worktree` |
| 89 | Embracing the parallel coding agent lifestyle (174 points, 138 comments) | HN | 2025-10-06 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45489884) | `parallel`, `worktree`, `sandbox`, `setup`, `own-host` |
| 90 | LLM codegen go brrr: parallelisation with git worktrees and tmux (156 points, 73 comments) | HN | 2025-05-28 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=44116872) | `parallel`, `worktree`, `cost` |
| 91 | Show HN: Superset, a terminal to run 10 parallel coding agents (96 points, 90 comments) | HN | 2025-12-23 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=46368739) | `parallel`, `worktree`, `sandbox` |
| 92 | Ask HN: Is it possible to run several coding sessions in parallel? (11 points, 15 comments) | HN | 2026-03-30 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47573483) | `parallel`, `worktree`, `ports` |
| 93 | Ask HN: What is the best way to use Claude Code from my phone? (6 points, 6 comments) | HN | 2025-11-02 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=45787595) | `mobile`, `own-host`, `sleep`, `setup` |
| 94 | Anyone else having a terrible experience with Remote Control? (5 points, 4 comments) | HN | 2026-04-05 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47645915) | `reconnect`, `ports`, `mobile` |
| 95 | Show HN: Approve Claude Code permission requests from your phone via ntfy (5 points, 2 comments) | HN | 2026-02-22 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47111171) | `notify`, `mobile` |
| 96 | Show HN: Shellular, run Claude Code, Codex, Pi from your phone (33 points, 41 comments) | HN | 2026-07-07 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48818124) | `mobile`, `own-host`, `sleep`, `any-agent` |
| 97 | Show HN: Boxes.dev, run Claude Code and Codex in the cloud (105 points, 79 comments) | HN | 2026-06-04 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48399358) | `own-host`, `sandbox`, `ports`, `parallel`, `any-agent`, `cost` |
| 98 | Orbs (Amp) (116 points, 43 comments) | HN | 2026-08-25 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=49437117) | `sandbox`, `own-host`, `cost` |
| 99 | Anthropic takes legal action against OpenCode (483 points, 389 comments) | HN | 2026-03-19 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47444748) | `any-agent`, `cost` |
| 100 | Gemini CLI will stop working from June 18, 2026 (406 points, 208 comments) | HN | 2026-05-19 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48196867) | `any-agent`, `cost` |
| 101 | OpenCode, open source AI coding agent (1,274 points, 585 comments) | HN | 2026-03-20 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=47460525) | `any-agent`, `cost`, `trust` |
| 102 | Unauthenticated remote code execution in OpenCode (432 points, 142 comments) | HN | 2026-01-11 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=46581095) | `trust` |
| 103 | Annoying and alarming things about OpenCode (420 points, 288 comments) | HN | 2026-07-20 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48978112) | `trust`, `sandbox` |
| 104 | OpenCode 2.0 (703 points, 295 comments) | HN | 2026-10-14 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=50094994) | `any-agent`, `sandbox` |
| 105 | Claude Code sends 33k tokens before reading the prompt; OpenCode sends 7k (706 points, 391 comments) | HN | 2026-07-12 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48883275) | `cost`, `any-agent` |
| 106 | Gemini CLI (launch) (1,428 points, 788 comments) | HN | 2025-06-25 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=44376919) | `cost`, `any-agent`, `login` |
| 107 | Wasting inferences with aider (139 points, 105 comments) | HN | 2025-04-13 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=43672712) | `parallel`, `cost` |
| 108 | Tell HN: Installing Cursor on iOS changes your privacy settings (249 points, 34 comments) | HN | 2026-06-30 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=48737226) | `trust`, `mobile` |
| 109 | Ask HN: How do you safely give LLMs SSH and database access? (85 points, 104 comments) | HN | 2026-01-14 | 2026-10-15 | [link](https://news.ycombinator.com/item?id=46620990) | `secrets`, `sandbox` |
| 110 | r/ClaudeAI: why I moved Claude Code off my laptop to a remote box (Tailscale, mosh, tmux) (61 comments) | Reddit | 2026-09-02 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1w5bx9x/) | `sleep`, `own-host`, `mobile`, `sandbox`, `secrets`, `setup`, `reconnect`, `clipboard`, `cost` |
| 111 | r/ClaudeAI: Termius, tmux and Claude Code on my iPhone (58 points, 50 comments) | Reddit | 2025-06-11 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1l96zlw/) | `mobile`, `keys`, `render`, `voice`, `scrollback` |
| 112 | r/ClaudeAI: Claude Code on my phone over SSH with a-Shell (136 points, 80 comments) | Reddit | 2025-06-28 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1lmqayh/) | `mobile`, `own-host`, `reconnect`, `voice` |
| 113 | r/ClaudeAI: convince me mobile dev isn't just theatre (74 comments) | Reddit | 2025-10-10 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1o32uzh/) | `mobile`, `ports`, `own-host` |
| 114 | r/ClaudeAI: does anyone run Claude on a 24/7 VPS? (22 comments) | Reddit | 2026-10-06 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1wzhgi3/) | `own-host`, `sleep`, `cost`, `mobile` |
| 115 | r/ClaudeAI: we need a cross-device notification system for the CLI (5 comments) | Reddit | 2026-01-08 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1q7jtj2/) | `notify`, `status`, `mobile` |
| 116 | r/ClaudeAI: Claude Code on the web (launch thread) (346 points, 111 comments) | Reddit | 2025-10-20 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1obpbbs/) | `own-env`, `mobile`, `own-host`, `sandbox`, `cost` |
| 117 | r/ClaudeAI: Claude Code agents as Tamagotchis, monitored from tmux (691 points, 76 comments) | Reddit | 2026-03-15 | 2026-10-15 | [link](https://www.reddit.com/r/ClaudeAI/comments/1ru9yda/) | `status`, `parallel`, `worktree` |
| 118 | r/tmux: is this subreddit just Claude Code tmux plugins now? (110 points, 20 comments) | Reddit | 2026-04-02 | 2026-10-15 | [link](https://www.reddit.com/r/tmux/comments/1saciat/) | `status` |
| 119 | r/ChatGPTCoding: who runs Codex CLI on a server to SSH in from a phone? (46 comments) | Reddit | 2025-09-13 | 2026-10-15 | [link](https://www.reddit.com/r/ChatGPTCoding/comments/1nga48y/) | `own-host`, `mobile`, `reconnect` |
| 120 | r/ChatGPTCoding: running Codex from a terminal on my phone (37 points, 28 comments) | Reddit | 2025-09-28 | 2026-10-15 | [link](https://www.reddit.com/r/ChatGPTCoding/comments/1nsz5ri/) | `mobile`, `own-host`, `reconnect` |
| 121 | r/codex: remote ChatGPT/Codex is practically unusable (17 points, 9 comments) | Reddit | 2026-09-29 | 2026-10-15 | [link](https://www.reddit.com/r/codex/comments/1wt0qt3/) | `reconnect`, `mobile`, `multi-device` |
| 122 | r/codex: is Codex remote not working? (22 points, 43 comments) | Reddit | 2026-08-21 | 2026-10-15 | [link](https://www.reddit.com/r/codex/comments/1vu8gvz/) | `reconnect`, `mobile` |
| 123 | r/codex: my Dot lost its work, cloud workspace restarted (31 points, 26 comments) | Reddit | 2026-10-12 | 2026-10-15 | [link](https://www.reddit.com/r/codex/comments/1x3wc14/) | `snapshot`, `own-host`, `cost` |
| 124 | r/opencodeCLI: an opencode Remote Control app? (13 points, 12 comments) | Reddit | 2026-02-27 | 2026-10-15 | [link](https://www.reddit.com/r/opencodeCLI/comments/1rfw70p/) | `mobile`, `own-host`, `setup`, `reconnect` |
| 125 | r/cursor: Background Agent, game changer or hype? (13 points, 9 comments) | Reddit | 2025-06-05 | 2026-10-15 | [link](https://www.reddit.com/r/cursor/comments/1l3wwtr/) | `trust`, `setup`, `own-env` |
| 126 | r/cursor: Cursor's background agent actually works now (7 points, 7 comments) | Reddit | 2026-05-09 | 2026-10-15 | [link](https://www.reddit.com/r/cursor/comments/1t81b3a/) | `worktree`, `sandbox` |
| 127 | r/selfhosted: Codeman, self-hosted mission control for coding agents (176 points, 43 comments) | Reddit | 2026-08-03 | 2026-10-15 | [link](https://www.reddit.com/r/selfhosted/comments/1vebymy/) | `own-host`, `mobile`, `notify`, `any-agent`, `sleep`, `parallel` |
| 128 | Creator of Claude Code: "I run 5 Claudes in parallel in my terminal" | X | 2026-01-02 | 2026-10-15 | [link](https://x.com/bcherny/status/2007179833990885678) | `parallel`, `notify`, `status` |
| 129 | opencode maintainer: opencode stops loading the Claude Max plugin after lawyers' letter | X | 2026-03-19 | 2026-10-15 | [link](https://x.com/thdxr/status/2034730036759339100) | `any-agent` |
| 130 | Anthropic staff member on X: Agent SDK use with Max plans unchanged | Vendor | 2026-02-18 | 2026-10-15 | [link](https://x.com/trq212/status/2024212378402095389) | `any-agent` |
| 131 | ClaudeDevs on X: monthly credit for programmatic use on paid plans | Vendor | 2026-05-13 | 2026-10-15 | [link](https://x.com/ClaudeDevs/status/2054610152817619388) | `cost` |
| 132 | Simon Willison: Embracing the parallel coding agent lifestyle | Blog | 2025-10-05 | 2026-10-15 | [link](https://simonwillison.net/2025/Oct/5/parallel-coding-agents/) | `parallel`, `sandbox`, `mobile` |
| 133 | Peter Steinberger: Just Talk To It | Blog | 2025-10-14 | 2026-10-15 | [link](https://steipete.me/posts/just-talk-to-it) | `parallel`, `worktree`, `render`, `scrollback`, `mobile` |
| 134 | Product Compass: Claude Code on a VPS, run it 24/7 and reach it from your phone (updated 29 Sep 2026) | Blog | 2026-09-04 | 2026-10-15 | [link](https://www.productcompass.pm/p/claude-code-vps) | `own-host`, `sleep`, `login`, `reconnect`, `setup`, `mobile` |
| 135 | Twingate: run Claude Code from your phone with Termius and tmux | Blog | 2026-03-05 | 2026-10-15 | [link](https://www.twingate.com/blog/claude-code-termius-tmux) | `mobile`, `setup`, `keys`, `scrollback`, `own-host` |
| 136 | Solomon Hykes: Containing Agent Chaos (AI Engineer World's Fair) | Talk | 2025 (June, **UNVERIFIED** month) | 2026-10-15 | [link](https://ai.engineer/talks/containing-agent-chaos) | `parallel`, `sandbox`, `secrets`, `own-host`, `any-agent` |
| 137 | Claude Code docs: Remote Control | Vendor | undated | 2026-10-15 | [link](https://code.claude.com/docs/en/remote-control) | `mobile`, `notify`, `reconnect`, `any-agent`, `trust`, `multi-device` |
| 138 | Claude Code docs: legal and compliance (credentials and third-party clients) | Vendor | undated | 2026-10-15 | [link](https://code.claude.com/docs/en/legal-and-compliance) | `any-agent`, `login` |
| 139 | Claude Code docs: cloud sessions (Claude Code on the web) | Vendor | undated | 2026-10-15 | [link](https://code.claude.com/docs/en/claude-code-on-the-web) | `own-env`, `sandbox`, `any-agent`, `share`, `cost` |
| 140 | Codex docs: remote connections | Vendor | undated | 2026-10-15 | [link](https://learn.chatgpt.com/docs/remote-connections) | `own-host`, `sleep`, `mobile`, `notify`, `any-agent` |
| 141 | Cursor docs: pricing (Cloud Agents) | Vendor | undated | 2026-10-15 | [link](https://cursor.com/docs/account/pricing) | `cost`, `any-agent` |
| 142 | Cursor docs: privacy and data governance (Cloud Agents store code) | Vendor | undated | 2026-10-15 | [link](https://cursor.com/docs/enterprise/privacy-and-data-governance) | `trust`, `sandbox` |
| 143 | Google Developers Blog: Gemini CLI to Antigravity CLI | Vendor | 2026-05-19 | 2026-10-15 | [link](https://developers.googleblog.com/an-important-update-transitioning-gemini-cli-to-antigravity-cli/) | `any-agent`, `cost` |
| 144 | herdrdev/herdr (44,140 stars) | Repo | 2026-03-27 | 2026-10-15 | [link](https://github.com/herdrdev/herdr) | `own-host`, `parallel`, `reboot`, `reconnect`, `image`, `worktree`, `any-agent` |
| 145 | slopus/happy (24,155 stars) | Repo | 2025-07-18 | 2026-10-15 | [link](https://github.com/slopus/happy) | `mobile`, `any-agent`, `trust`, `notify` |
| 146 | getpaseo/paseo (20,967 stars) | Repo | 2025-10-13 | 2026-10-15 | [link](https://github.com/getpaseo/paseo) | `mobile`, `parallel`, `any-agent` |
| 147 | superset-sh/superset (15,219 stars) | Repo | 2025-10-21 | 2026-10-15 | [link](https://github.com/superset-sh/superset) | `parallel`, `worktree`, `any-agent` |
| 148 | siteboon/claudecodeui (CloudCLI) (14,106 stars) | Repo | 2025-06-25 | 2026-10-15 | [link](https://github.com/siteboon/claudecodeui) | `mobile`, `any-agent` |
| 149 | smtg-ai/claude-squad (8,607 stars) | Repo | 2025-03-09 | 2026-10-15 | [link](https://github.com/smtg-ai/claude-squad) | `parallel`, `worktree`, `any-agent` |
| 150 | tiann/hapi (5,244 stars) | Repo | 2025-12-24 | 2026-10-15 | [link](https://github.com/tiann/hapi) | `mobile`, `any-agent`, `own-host` |
| 151 | amantus-ai/vibetunnel (4,688 stars) | Repo | 2025-06-15 | 2026-10-15 | [link](https://github.com/amantus-ai/vibetunnel) | `mobile`, `own-host` |
| 152 | dagger/container-use (4,070 stars) | Repo | 2025-05-23 | 2026-10-15 | [link](https://github.com/dagger/container-use) | `sandbox`, `parallel` |
| 153 | raine/workmux (2,839 stars) | Repo | 2025-11-04 | 2026-10-15 | [link](https://github.com/raine/workmux) | `worktree`, `parallel` |
| 154 | neurosnap/zmx (2,210 stars) | Repo | 2025-10-10 | 2026-10-15 | [link](https://github.com/neurosnap/zmx) | `sleep` |
| 155 | kzahel/yepanywhere (537 stars) | Repo | 2025-12-28 | 2026-10-15 | [link](https://github.com/kzahel/yepanywhere) | `notify`, `image`, `trust` |
| 156 | tmux PR #5615: kitty keyboard protocol with autodetect (open, not merged; 5 reactions, 7 comments). Used only to confirm that tmux has no kitty keys yet. | GH PR | 2026-09-19 | 2026-10-15 | [link](https://github.com/tmux/tmux/pull/5615) | none (verification only) |
