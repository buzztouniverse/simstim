# Competitors: running coding agents remotely or in the background

Researched on 15 October 2026.

Research note for simstim. It compares hosted and open tools that run coding agents away from the laptop, or keep them running in the background.

## About this note

- **Scope.** 28 rows: 11 hosted products, 3 sandbox providers, 13 open or local tools, and simstim itself. Each row has 14 facts, split over three tables.
- **Sources.** Every cell links a primary source: official docs, pricing pages, repositories, changelogs or official posts. All of them were read on 15 October 2026. Prices are list prices in US dollars on that day, without tax.
- **Method.** Five research passes ran in parallel, one for each group of rows. I then re-read the most surprising claims myself: Cursor's new plans, OpenAI's purchase of Ona, Daytona's move to closed source, Conductor's Pro plan, the end of Vibe Kanban's company, Claude Code's cloud sessions and Codex's prices. GitHub figures come from `gh repo view` on the same day.
- **Rules.** No sign-ups, accounts, API keys, downloads or paid resources. Public pages only. Closed products are described in my own words.
- **Reuse.** This note does not repeat what other notes already cover. Amp's orbs are in [Amp UX and Orbs][n-amp]. Herdr's design is in [Herdr][n-herdr]. mosh, Eternal Terminal and Tailscale SSH as transports are in [Connectivity and providers][n-conn]. zmx is in [Host runtime][n-host].
- **UNVERIFIED** marks a claim I could not confirm from a primary source. A "?" in a table cell means the same.

### Legend

| Mark | Meaning |
| --- | --- |
| ✅ | Yes, fully |
| ◐ | Partly. The cell says what is missing. |
| ❌ | No |
| ? | Unknown. Always marked **UNVERIFIED**. |

What each column asks:

| Column | ✅ means | ◐ means |
| --- | --- | --- |
| Self-host | Everything runs on your machines | Only the runtime is yours; the control plane is the vendor's, or it needs an Enterprise plan |
| Any agent | Any terminal program runs | A fixed list, or the vendor's agent plus one or two others |
| Real TUI pass-through | The agent's own bytes reach your terminal unchanged | You see the real TUI, but a multiplexer or an embedded terminal re-draws it, or it works only through a separate SSH path |
| Persistence | Work goes on with the laptop closed or offline, with no action from you | It goes on, but with idle stops, time limits, manual reattach, or only while your own computer stays awake |
| Notifications | You hear when an agent needs input or is done | Only some events, only in a browser, or only the tool's own alerts |
| Ports / previews | Forwarded ports or preview links | Local only, or limited |
| Worktrees | Each task gets its own isolated checkout or machine, made for you | You can isolate tasks, but by hand |
| Sharing / watch | Others can watch or join live | Static links, one user's devices, or paid only |
| Snapshots and hibernation | Saved environments and pause or resume | One of the two, or limited |
| Secrets | A managed store: encrypted, injected, hidden or proxied | Plain variables or copied `.env` files, or no store by design |

## Summary

1. **No product meets simstim's whole brief.** simstim is the only row with ✅ in all four of: any agent, real TUI pass-through, persistence, and self-hosting. Four rows miss only one:
   - **zmx and Eternal Terminal** pass the bytes through, but they lose sessions on reboot, and they manage no agents.
   - **Herdr** persists and manages agents, but it re-draws the screen.
   - **Coder** gives a real SSH terminal on your own servers, but it stops idle workspaces. It needs tmux to survive a disconnect, and tmux re-draws the screen.

   Codespaces and Daytona are like Coder, but they also run on the vendor's machines. **The hosted agents** run one vendor's agent in a web or app UI: Amp, Claude Code on the web, Codex cloud, Cursor, Jules, Devin, Factory and Warp.
2. **The market churned hard in 2025–2026.** In this matrix alone:
   - Terragon shut down (9 February 2026).
   - Crystal was deprecated in favour of Nimbalyst (February 2026).
   - Vibe Kanban's company shut down (10 April 2026).
   - Daytona went closed source (11 June 2026).
   - OpenAI bought Ona, formerly Gitpod (deal closed 10 August 2026). Ona closed new sign-ups, and its built-in agent is now Codex only.
   - SpaceX bought Cursor (14 August 2026). Cursor's cloud agents now need its $30 Premium plan.
   - Coder removed its Tasks feature and archived AgentAPI.
   - Devin retired its Core and Team plans.

   The full list is in [Changes in 2025–2026](#changes-in-20252026).
3. **Vendors now run their own agent on your machine, but they keep the control plane.** Examples: Amp runners, Claude Desktop SSH sessions, Codex's "Connect to an SSH host", Cursor's My Machines, Devin Outposts, Factory's Remote Control and Warp's Remote Control. Each is tied to one vendor's agent, account and UI. See [Near misses](#near-misses-vendor-agents-on-your-own-machine).
4. **simstim already gives away what others sell:**
   - an agent that keeps working with the laptop closed (Conductor Pro $50 a month, Cursor Premium $30, Claude and Codex from $20);
   - runs on your own machines (Enterprise plans at most vendors);
   - no session time limits (E2B: 1 hour free, 24 hours on the $150 Pro plan; Modal: 24 hours on every plan);
   - no limit on parallel agents (Jules: 3 free, 15 on the $19.99 plan);
   - a shell and port forwards into the agent's machine (Devin and Factory, from $20).
5. **Cheap wins remain** (section [Paid features](#paid-features-simstim-can-give-for-free)):
   - a host-side notify hook that reaches a phone or Slack (S);
   - stopping cloud VMs after an idle rule (S–M);
   - read-only watchers (M);
   - `pull` to bring a session's branch back to the laptop (S);
   - recipes for scheduled runs (S).
6. **simstim is behind on reach and isolation.** Nothing is released yet, and you must bring a Linux host. There is no phone or web client, and no sharing with other people. There are no per-task sandboxes, no environment snapshots and no managed secrets.
7. **simstim is ahead on fidelity, cost and control.** It passes bytes through with replay and keeps native scrollback. It has no time, concurrency or compute caps. It works with any agent login and any SSH path. It needs no account, relay or open port, and carries no vendor risk.
8. **Updates to [the design page](../explanation/design.md#why-build-it).** Daytona is now closed source, not just archived. zellij forwards OSC 9 and 777 since 0.45, though it still re-draws the screen. One fact is confirmed: tmux 3.8 still has no kitty keyboard protocol, which is planned for 3.9.

## Changes in 2025–2026

| Product | Change | Date | Source |
| --- | --- | --- | --- |
| Terragon | Shut down. Sandboxes deleted, credits refunded. An Apache-2.0 snapshot of the code was published in January. | 9 Feb 2026 | [shutdown][tg-shutdown], [site][tg-home] |
| Sculptor | Open-sourced under MIT. It moved from one container per agent to git worktrees. | 12 Jun 2026 | [commit][sc-epoch], [history][sc-history] |
| Crystal | Deprecated, with no further updates. Its successor Nimbalyst was proprietary from 21 January and MIT from 28 April 2026. | Feb 2026 | [release][cr-last], [commit][nb-mit] |
| Vibe Kanban | Its company, bloop, shut down. The project goes on as community-maintained open source. Cloud features and paid plans were removed. | 10 Apr 2026 | [post][vk-shutdown] |
| Ona (Gitpod) | Gitpod renamed Ona. | 2 Sep 2025 | [post][ona-rename] |
| Ona (Gitpod) | Gitpod Classic pay-as-you-go shut down. | 15 Oct 2025 | [post][ona-classic] |
| Ona (Gitpod) | Agreed to join OpenAI's Codex team. The deal closed on 10 August. New sign-ups are now closed, and the built-in agent is Codex only. | 11 Jun 2026 | [post][ona-openai], [docs][ona-start], [agents][ona-agents] |
| Codex cloud | Cloud tasks started to count towards plan usage. Credits for extra use followed on 30 October. | 20 Oct 2025 | [changelog][cx-changelog] |
| Codex cloud | A new Codex Cloud with published environments. The old one is now "Legacy". Docs moved to learn.chatgpt.com. | Sep–Oct 2026 | [what's new][cx-new], [environments][cx-env] |
| Claude Code on the web | Launched as a research preview for Pro and Max. | 20 Oct 2025 | [post][cc-blog] |
| Claude Code on the web | Now called "cloud sessions". `--remote` is a deprecated alias of `--cloud`. Self-hosted environments entered public beta. | Aug 2026 | [docs][cc-cloud], [week 32][cc-w32] |
| Cursor | Pro moved to usage-based billing. Cursor apologised and refunded charges on 4 July. | 16 Jun 2025 | [post][cu-june] |
| Cursor | Background Agents renamed Cloud Agents (Cursor 2.0). | 29 Oct 2025 | [changelog][cu-2] |
| Cursor | Acquired by SpaceX. | 14 Aug 2026 | [post][cu-spacex] |
| Cursor | New plans sold through "the X Subscription": Free, Plus $8, Premium $30, Super $100, Ultra $200. Cloud agents moved from Pro ($20) to Premium ($30). The old plans stay for existing subscribers. | 12 Oct 2026 | [pricing][cu-pricing], [announcement][cu-x] |
| Warp | Build plan at $20 a month replaced Pro, Turbo and Lightspeed. | 30 Oct 2025 | [post][wp-build] |
| Warp | Oz cloud agents (10 February). The client was open-sourced under AGPL-3.0 (27 April). Oz runs Claude Code and Codex (18 May). | 2026 | [changelog][wp-log26] |
| Warp | Warp Factories announced as a closed beta. | 18 Aug 2026 | [post][wp-factories] |
| Devin | Core and Team plans retired. Now Free, Pro $20, Max $200 and Teams. | 14 Apr 2026 | [post][dv-plans] |
| Devin | Devin Desktop succeeded Windsurf (2 June). Outposts launched (22 July). `devin --cloud` and `devin ssh` arrived (21 September). | 2026 | [post][dv-desktop], [notes][dv-notes], [CLI][dv-clilog] |
| Factory | factory.ai now redirects to factory.com (date UNVERIFIED). Pro gained managed computers (9 September). Bring-your-own-computer was renamed Remote Control (10 October). Environments became generally available (12 October). | 2026 | [changelog][fa-notes], [post][fa-fleet] |
| Amp | Split from Sourcegraph as Amp Inc. | 2 Dec 2025 | [post][amp-inc] |
| Amp | Orb prices cut and new a1 sizes (July and August). "Free Agent" pricing made runners free (13 September). Claude Code mode added (10 October). | 2026 | [sizes][amp-sizes-news], [post][amp-free], [post][amp-claude] |
| Coder | Tasks deprecated in v2.34 (2 June) and removed in v2.36 (4 August). AgentAPI archived. `coder/mux` renamed Xum (20 August). | 2026 | [migration][co-tasks], [AgentAPI][co-agentapi], [Xum][gh-xum] |
| Daytona | Went closed source, giving security as the reason. The repository is archived, and the SDKs moved to a new organisation. | 11 Jun 2026 | [post][dt-closed], [repo][gh-daytona], [clients][dt-clients] |
| Jules | Left beta, with paid tiers. The changelog's last entry is 9 March 2026. | 6 Aug 2025 | [changelog][jl-changelog] |
| E2B | `e2b-dev/infra` became `e2b-dev/runtime`, with a single-machine self-host package (date UNVERIFIED). | 2026 | [repo][gh-e2b-runtime] |
| Modal | Filesystem snapshots now expire after 30 days by default. | 2026 | [docs][md-snap] |
| Conductor | Pro plan at $50 a month (23 July). Conductor Cloud followed (30 July), then an iOS app (2 October). | 2026 | [changelog][cd-changelog] |
| Tailscale | Pricing v4. Personal Plus retired; Starter became Standard at $8. SSH session recording via tsrecorder is no longer offered. | 8 Apr 2026 | [post][ts-v4], [recording][ts-rec] |
| Herdr | "Herdr Cloud" waitlist, with no price. | Oct 2026 | [page][hd-cloud] |

## Table 1: business

| Product | List price (read 15 Oct 2026) | What paid tiers add | Self-host | Open source (licence) | Any agent |
| --- | --- | --- | --- | --- | --- |
| **Amp** (orbs, runners) | Hobby $0, with orbs pay-as-you-go at $0.08–2.13 an hour. Individual $20 a month includes 45,000 orb-minutes. A $200 top tier appears only in the page's script (UNVERIFIED) ([pricing][amp-pricing], [sizes][amp-sizes]) | Included orb time, custom portal domains, the largest orb size; Enterprise adds SCIM and audit logs ([pricing][amp-pricing]) | ◐ Runners run threads on your machine for free; the control plane stays at Amp ([runners][amp-runners]) | Closed; the CLI is proprietary ([npm][amp-npm]) | ◐ Amp's agent, plus a Claude Code mode ([news][amp-claude], [runners][amp-runners]) |
| **Claude Code on the web** | No extra charge. Needs Pro $20 a month ($17 yearly), Max $100 or $200, Team from $25 a seat, or Enterprise ([pricing][cc-pricing], [docs][cc-cloud]) | Max: 5x or 20x usage, which bounds parallel sessions. Team and Enterprise: self-hosted and shared environments ([environments][cc-env], [self-hosted][cc-selfhost]) | ◐ Self-hosted environments (Team and Enterprise, beta); the control plane stays at Anthropic ([docs][cc-selfhost]) | Closed; the CLI is proprietary ([licence][cc-licence]) | ❌ Claude Code only ([docs][cc-cloud]) |
| **Codex cloud** | Plus $20 a month; Pro from $100; Business $25 a user monthly; Enterprise by quote. Whether Free and Go include cloud is UNVERIFIED ([pricing][cx-pricing]) | Bigger VMs: 2 vCPU and 8 GiB on Plus, 4 vCPU and 16 GiB on Pro and Business, custom on Enterprise ([environments][cx-env]) | ❌ OpenAI runs it ([environments][cx-env]) | CLI Apache-2.0 ([repo][gh-codex]); cloud closed | ❌ Codex only ([docs][cx-cloud]) |
| **Cursor cloud agents** | Free $0, Plus $8, Premium $30, Super $100, Ultra $200 a month; Teams from $30 a seat. Cloud agents from Premium, billed at the model's API price ([pricing][cu-pricing], [billing][cu-billing]) | Premium: cloud agents, automations, phone remote control. Teams: CI auto-fix. Enterprise: bigger VMs, team pools ([docs][cu-agent], [setup][cu-setup]) | ◐ "My Machines" runs tools on your host; the agent loop stays in Cursor's cloud ([docs][cu-self]) | Closed; only a sample worker chart is open ([repo][cu-k8s]) | ❌ Cursor's agent, with a choice of models ([docs][cu-agent]) |
| **Jules** | Free: 15 tasks a day, 3 at once. Google AI Pro $19.99 a month: 100 and 15. Ultra $99.99 or $199.99: 300 and 60 ([limits][jl-limits], [plans][jl-subs]) | More tasks and concurrency, newer models, suggested tasks ([limits][jl-limits]) | ❌ Google's VMs only ([docs][jl-env]) | Closed; the CLI fetches a closed binary ([npm][jl-npm]) | ❌ Jules only ([docs][jl-docs]) |
| **Devin** | Free $0, Pro $20, Max $200 a month; Teams from $80 a month ($40 a seat); Enterprise in ACUs ([pricing][dv-pricing], [docs][dv-selfserve]) | Pro: cloud VM sessions, 10 at once. Teams: unlimited sessions, sharing. Enterprise: VPC, idle timeout of 5–120 min ([pricing][dv-pricing], [usage][dv-usage]) | ◐ Outposts run commands on your machines; the agent loop stays at Cognition ([docs][dv-outposts]) | Closed; small helpers are MIT ([handoff][dv-handoff]) | ❌ Devin only; other agents can hand it tasks ([docs][dv-handoff]) |
| **Factory** | Pro $20, Plus $100, Max $200 a month; Teams $60 a month plus $40 a seat; Business and Enterprise custom ([pricing][fa-pricing], [docs][fa-ind]) | Managed "Droid Computers": Pro with 5 hours a month per the docs, but Plus per the pricing page (conflict). Enterprise: on-premises and air-gapped ([organisations][fa-org], [changelog][fa-notes]) | ◐ Remote Control runs Droid on your machine through Factory's relay ([docs][fa-rc]) | Closed ("UNLICENSED") ([npm][fa-npm]) | ❌ Droid only, with many models ([docs][fa-computers]) |
| **Warp** | Free $0; Build $20; Max $200 a month; Business $50 a user; Enterprise custom. Factories: usage at 20% over API rates ([pricing][wp-pricing]) | Build: "extended" cloud agents. Claude Code and Codex in the cloud need a paid plan per the docs, but the pricing page says Free (conflict). Enterprise: self-hosted workers ([harnesses][wp-harness]) | ◐ Workers on your hosts; the control plane is Warp's ([FAQ][wp-faqs]) | Client AGPL-3.0 since April 2026; server and Oz closed ([FAQ][wp-faq-lic]) | ◐ The terminal runs any CLI agent; the cloud runs Warp's agent, Claude Code and Codex ([docs][wp-cli-agents]) |
| **GitHub Codespaces** | Free: 120 core-hours and 15 GB-month a month (Pro account: 180 and 20). Then $0.18 an hour (2 cores) to $2.88 (32 cores), storage $0.07 a GB-month ([billing][cs-billing]) | Pay-as-you-go hours, up to 32 cores ([billing][cs-billing]) | ❌ GitHub-hosted only ([billing][cs-billing]) | Service closed; `gh` CLI MIT ([manual][cs-ssh]) | ✅ Install any CLI; Copilot CLI is preinstalled ([changelog][cs-copilot]) |
| **Ona** (formerly Gitpod) | No public prices; new sign-ups are closed. Core was "from $20 a month" in September 2025 and is closed to new buyers ([pricing][ona-pricing], [billing][ona-billing], [start][ona-start]) | Core: Codex agent, auto-stop policies. Enterprise: runners in your AWS or GCP account, SSO, audit ([runners][ona-runners]) | ◐ Runners in your cloud account (Enterprise); the management plane is Ona's ([docs][ona-runners]) | Gitpod Classic AGPL-3.0, last release 2023 ([repo][gh-gitpod]); Ona closed (UNVERIFIED) | ◐ Built-in agent is Codex only; you may run Claude Code yourself ([docs][ona-agents]) |
| **Coder** | Community free; Premium per user per year, price not public; AI Premium custom ([pricing][co-pricing]) | Premium: AI gateway, audit logs, HA, dormancy, prebuilt workspaces. AI Premium: more than 5 concurrent Coder Agents ([licensing][co-licensing]) | ✅ Control plane and workspaces on your infrastructure ([docs][co-agents]) | AGPL-3.0 ([repo][gh-coder]) | ✅ Claude Code, Codex and others through modules; also its own agent ([docs][co-ai]) |
| **E2B** | Hobby $0 with $100 one-off credit; Pro $150 a month plus usage; Enterprise from $3,000 a month. vCPU $0.000014 a second ([pricing][e2b-pricing], [billing][e2b-billing]) | Pro: runs of up to 24 h (Hobby 1 h), 100 sandboxes at once (Hobby 20). Enterprise: your own cloud ([billing][e2b-billing]) | ✅ The open runtime runs on one KVM host ([repo][gh-e2b-runtime]) | Apache-2.0 SDK, CLI and runtime ([repo][gh-e2b]) | ✅ Guides for Claude Code, Codex, Amp and others ([docs][e2b-claude]) |
| **Daytona** | Usage only: vCPU $0.0504 an hour, memory $0.0162 a GiB-hour, disk free to 5 GiB; $200 free credit ([pricing][dt-pricing]) | Higher resource tiers unlock with top-ups of $25 to $2,000. Enterprise: SSO, audit, your own compute ([limits][dt-limits]) | ◐ Runners on your machines; the control plane is Daytona's ([docs][dt-byoc]) | Closed since June 2026; clients Apache-2.0 ([post][dt-closed], [clients][dt-clients]) | ✅ Default image has Claude Code; guides for Codex, Gemini, OpenCode ([guide][dt-claude]) |
| **Modal** | Starter $0 with $30 a month of credit; Team $250 a month; Enterprise custom. Sandbox CPU $0.00003942 a core-second ([pricing][md-pricing]) | Team: unlimited seats, 5,000 containers. Enterprise: custom limits ([pricing][md-pricing]) | ❌ None found ([pricing][md-pricing]) | Client Apache-2.0; platform closed ([repo][gh-modal]) | ✅ Any image or CLI ([docs][md-sandboxes]) |
| **Conductor** | Free $0; Pro $50 a month; Teams $60 a user; Enterprise custom ([pricing][cd-pricing]) | Pro: Conductor Cloud (agents keep going with the Mac closed), live collaboration for up to 5, API, iOS app ([pricing][cd-pricing]) | ◐ The app runs on your Mac only; the cloud is Conductor's; your own machines "not at the moment" ([pricing][cd-pricing]) | Closed ([terms][cd-terms]) | ◐ Chat drives four agents; an experimental terminal mode runs any CLI ([harnesses][cd-harness], [terminal][cd-bigterm]) |
| **Terragon** (shut down 9 Feb 2026; cells describe the service before that) | Was Core $25 and Pro $50 a user a month ([billing][tg-billing], [shutdown][tg-shutdown]) | Was: hosted sandboxes, 3 or 10 tasks at once, automations ([plans][tg-plans]) | ◐ Was hosted only; the open snapshot needs Postgres, Redis, a GitHub App and a sandbox provider ([repo][gh-terragon]) | Apache-2.0 snapshot ([repo][gh-terragon]) | ◐ Claude Code, Codex, Amp, Gemini, OpenCode ([docs][tg-agents]) |
| **Sculptor** | Free; you pay only your model provider ([product][sc-product]) | None | ✅ Desktop app on Mac and Linux; experimental remote backend over SSH or Docker ([docs][sc-backend]) | MIT ([licence][sc-licence]) | ✅ Claude Code and Pi built in; any terminal agent by a TOML entry ([spec][sc-spec]) |
| **claude-squad** | Free ([site][sq-site]) | None | ✅ Local TUI; runs on any host you SSH to ([repo][gh-squad]) | AGPL-3.0 ([licence][sq-licence]) | ✅ Any program, by flag or profile ([repo][gh-squad]) |
| **Crystal** (now Nimbalyst) | Crystal free, deprecated. Nimbalyst: Individual $0; Teams $20 a user a month, free in beta ([release][cr-last], [pricing][nb-pricing]) | Teams: live shared docs. Enterprise: self-hosted collaboration server ([pricing][nb-pricing]) | ✅ Desktop app on macOS, Windows, Linux ([repo][gh-nimbalyst]) | Both MIT ([repo][gh-crystal], [Nimbalyst][nb-oss]) | ◐ Claude Code, Codex, OpenCode, Copilot, Gemini ([repo][gh-nimbalyst]) |
| **Vibe Kanban** | Free. A $30 Pro plan still shows but is not sold ([post][vk-shutdown], [pricing][vk-pricing]) | None now ([post][vk-shutdown]) | ✅ Local or on a VPS; the server runs in Docker ([docs][vk-selfhost]) | Apache-2.0 ([repo][gh-vk]) | ◐ A fixed list of 10 agents ([docs][vk-agents]) |
| **tmux** | Free ([repo][gh-tmux]) | None | ✅ On the host ([man page][tm-man]) | ISC ([licence][tm-licence]) | ✅ Any program ([man page][tm-man]) |
| **zellij** | Free, with donations ([release][zj-0451]) | None. A sharing service, zellij.online, is in beta with no price ([site][zj-online]) | ✅ On the host, with a built-in web server ([docs][zj-web]) | MIT ([repo][gh-zellij]) | ✅ Any program ([release][zj-0451]) |
| **zmx** | Free ([site][zmx-site]) | None | ✅ On the host ([repo][gh-zmx]) | MIT ([repo][gh-zmx]) | ✅ Any program ([repo][gh-zmx]) |
| **Herdr** | Free; "Herdr Cloud" waitlist with no price ([site][hd-site], [cloud][hd-cloud]) | None yet | ✅ Its server runs on your machines and installs itself over SSH ([docs][hd-remote]) | Apache-2.0 ([repo][gh-herdr]) | ✅ Any program; 24 agents recognised ([docs][hd-agents]) |
| **mosh** | Free ([site][mosh-site]) | None | ✅ On the host ([site][mosh-site]) | GPL-3.0 ([repo][gh-mosh]) | ✅ Any command ([site][mosh-site]) |
| **Eternal Terminal** | Free ([repo][gh-et]) | None | ✅ A server on the host ([repo][gh-et]) | Apache-2.0 ([repo][gh-et]) | ✅ A shell or a command ([client][et-client]) |
| **Tailscale SSH** | Personal $0 (6 users); Standard $8 and Premium $18 a user a month; Enterprise custom ([pricing][ts-pricing]) | Premium: a custom re-login period. Session recording is a sales-only add-on ([pricing][ts-pricing], [recording][ts-rec]) | ◐ The daemon runs on your machines; the coordination server is Tailscale's ([open source][ts-oss]) | Client BSD-3-Clause; control plane closed ([repo][gh-tailscale]) | ✅ An SSH server: any program ([docs][ts-ssh]) |
| **simstim** (today) | Free. Pre-release: nothing published yet ([README][ss-readme]) | None; no account ([plan][ss-plan]) | ✅ Only your laptop and your hosts; no relay ([plan][ss-plan]) | Apache-2.0 ([licence][ss-licence]) | ✅ Agents are rows in a table; any command runs ([README][ss-readme]) |

## Table 2: session

| Product | Real TUI pass-through | Persistence | Notifications | Ports / previews | Worktrees |
| --- | --- | --- | --- | --- | --- |
| **Amp** | ◐ Amp's CLI follows an orb thread as a client, not as a byte stream. The web has its own UI and a browser terminal ([orbs][amp-what]) | ✅ Orbs work with the laptop closed. They pause after 5 min of agent idle and 20 min with no person, then wake with files and services. Runners stop with their machine ([customising][amp-custom], [runners][amp-runners]) | ◐ The macOS and iOS apps notify when a thread waits ([apps][amp-apps]) | ✅ Portals give a URL per HTTP port, but not on runners ([portals][amp-portals]) | ✅ An orb per thread; runners can make a worktree per thread ([runners][amp-runners]) |
| **Claude Code on the web** | ❌ Web, desktop and phone UI. `--teleport` copies a session into the local CLI; it is not a live attach ([docs][cc-cloud]) | ✅ Runs with the laptop closed. Idle VMs are reclaimed: the conversation comes back, running processes do not ([docs][cc-cloud]) | ◐ The desktop app notifies when a session ends; phone push is documented for Remote Control ([desktop][cc-desktop], [mobile][cc-mobile]) | ❌ No shell into the VM and no port forward ([environments][cc-env]) | ✅ A VM and a branch per session ([quick start][cc-quick]) |
| **Codex cloud** | ❌ Web, desktop and phone. `codex cloud` lists tasks; `codex apply` takes the diff ([commands][cx-cmds]) | ✅ Runs while the computer sleeps; task state kept for 7 days ([environments][cx-env]) | ✅ Push, email or SMS, and desktop alerts ([docs][cx-notify]) | ❌ None documented ([environments][cx-env]) | ✅ An isolated workspace per task ([modes][cx-modes]) |
| **Cursor cloud agents** | ❌ Web, desktop, iOS and Slack. The CLI can hand a task off, but cannot attach ([CLI][cu-cli]) | ✅ A cloud VM; idle VMs hibernate and can resume for 30 days ([security][cu-security]) | ✅ iOS push and Slack ([mobile][cu-mobile]) | ✅ Forwarded ports and a preview browser ([setup][cu-setup]) | ✅ A VM and a branch per agent ([docs][cu-agent]) |
| **Jules** | ❌ Web UI; the CLI is a dashboard and diff viewer ([CLI][jl-cli]) | ◐ Runs in a cloud VM while you are away; VM lifetime not documented ([docs][jl-tasks]) | ◐ Browser notifications only ([docs][jl-docs]) | ❌ Screenshots only ([changelog][jl-preview]) | ✅ A VM per task, with a branch on publish ([docs][jl-tasks]) |
| **Devin** | ◐ `devin --cloud` streams a session into Devin's own CLI; `devin ssh` gives a real shell ([CLI][dv-cli-cloud]) | ✅ Runs on its VM; sleeps after 30 min idle, and a message wakes it ([usage][dv-usage]) | ✅ Slack, and an inbox with phone push ([Slack][dv-slack], [notes][dv-notes]) | ✅ `devin forward` or `ssh -L` ([docs][dv-ssh]) | ✅ A fresh VM per session ([docs][dv-env]) |
| **Factory** | ◐ The local Droid CLI is a TUI. Remote sessions use its app, web or Slack; `droid computer ssh` gives a shell ([docs][fa-computers]) | ✅ Droid Computers keep files and services, and pause and resume ([docs][fa-computers]) | ◐ Desktop alerts and Slack; phone push UNVERIFIED ([changelog][fa-notes]) | ✅ `droid computer port-forward` ([docs][fa-computers]) | ✅ A worktree per session ([docs][fa-worktrees]) |
| **Warp** | ◐ Local and SSH sessions show the agent's own TUI inside Warp. Cloud runs and remote control use Warp's own viewer ([remote control][wp-rc]) | ◐ Cloud runs survive a closed laptop, but idle environments shut down. Remote control stops with the session ([docs][wp-runs]) | ✅ Desktop alerts for Claude Code, Codex, OpenCode and Warp's agent ([docs][wp-notify]) | ? None found for cloud runs. UNVERIFIED ([docs][wp-envs]) | ✅ Local worktrees; a fresh workspace per cloud run ([docs][wp-worktrees]) |
| **GitHub Codespaces** | ✅ `gh codespace ssh`, or plain OpenSSH ([manual][cs-ssh]) | ◐ Stops after 30 min idle by default (4 h at most), which kills processes. Deleted 30 days after a stop ([timeout][cs-timeout], [lifecycle][cs-lifecycle]) | ❌ None ([lifecycle][cs-lifecycle]) | ✅ Private, organisation or public ports ([ports][cs-ports]) | ◐ Many codespaces per repo, made by hand ([docs][cs-create]) |
| **Ona** | ✅ `ona environment ssh`, or plain SSH ([CLI][ona-cli]) | ◐ Auto-stop after 30 min by default. Background processes do not count as activity, but a keep-alive command exists ([timeout][ona-timeout]) | ◐ Slack updates for its own agent ([Slack][ona-slack]) | ✅ HTTPS port sharing ([ports][ona-ports]) | ◐ An environment per task, made by hand ([docs][ona-envs]) |
| **Coder** | ✅ `coder ssh`, or plain SSH ([docs][co-access]) | ◐ Stops after 1 h idle by default; SSH counts as activity ([docs][co-sched]) | ◐ Workspace events by email, webhook, Slack or Teams. No "agent needs input" event since Tasks went (UNVERIFIED) ([docs][co-notify]) | ✅ Shared ports and URLs ([docs][co-ports]) | ◐ Many workspaces; worktrees only in its separate Xum app ([Xum][xum-ssh]) |
| **E2B** | ◐ `e2b sandbox connect` gives a PTY over E2B's API; byte-exactness is not documented ([docs][e2b-connect]) | ◐ Default timeout 5 min; runs capped at 1 h (Hobby) or 24 h (Pro). Pause keeps memory ([docs][e2b-life]) | ◐ Lifecycle webhooks only ([docs][e2b-hooks]) | ✅ A public URL per port ([docs][e2b-url]) | ◐ A sandbox or fork per task, made by hand ([docs][e2b-fork]) |
| **Daytona** | ✅ `daytona ssh` ([docs][dt-ssh]) | ◐ Auto-stop after 15 min by default, even with processes running; it can be turned off ([docs][dt-sandboxes]) | ◐ Lifecycle webhooks only ([docs][dt-hooks]) | ✅ Preview URLs and signed links ([docs][dt-preview]) | ◐ A sandbox or fork per task, made by hand ([docs][dt-sandboxes]) |
| **Modal** | ◐ `modal shell` gives a PTY over Modal's API ([docs][md-shell]) | ◐ Default 5 min, 24 h at most ([docs][md-sandboxes]) | ❌ None ([docs][md-sandboxes]) | ✅ Tunnels with public URLs ([docs][md-tunnels]) | ◐ A sandbox per task, made by hand ([docs][md-sandboxes]) |
| **Conductor** | ◐ Its own chat UI. An experimental embedded terminal runs agent TUIs, locally only ([docs][cd-bigterm]) | ◐ Local sessions end when the Mac sleeps or the app closes. Pro cloud sleeps after 4 h idle and stops at 23 h 50 min ([docs][cd-cloud]) | ✅ Sounds, OS notifications and an iOS app ([changelog][cd-ios], [sounds][cd-sounds]) | ✅ Ports per workspace and preview URLs; cloud forwarding ([settings][cd-settings], [changelog][cd-ports]) | ✅ A worktree and branch per workspace ([docs][cd-worktrees]) |
| **Terragon** (shut down) | ❌ Web chat UI; `terry pull` resumed the work locally ([docs][tg-cli]) | ✅ Cloud sandboxes that hibernated when idle ([docs][tg-security]) | ✅ Browser push when a task ended ([notes][tg-notes]) | ? None found. UNVERIFIED | ✅ A sandbox and branch per task ([docs][tg-git]) |
| **Sculptor** | ◐ Chat UI by default. Terminal agents show their real TUI in a PTY inside the app ([sample][sc-term]) | ◐ Stops when the machine sleeps; restores and resumes agents after a restart ([requirements][sc-req]) | ✅ App notifications and status dots ([docs][sc-harness]) | ◐ No port feature; a web recipe proxies ports ([recipe][sc-openhost]) | ✅ A worktree by default ([docs][sc-workspaces]) |
| **claude-squad** | ◐ Attaches to the agent's tmux session, so tmux re-draws it. Local only ([repo][gh-squad]) | ◐ Agents keep running in tmux after the TUI quits, but stop when the laptop sleeps. After a reboot they are restarted (v1.0.20) ([code][sq-app], [PR][sq-322]) | ❌ Status in the TUI only; the request was closed ([issue][sq-185]) | ❌ An open request ([issue][sq-260]) | ✅ A worktree and branch per session ([repo][gh-squad]) |
| **Crystal** (now Nimbalyst) | ❌ Its own transcript UI; a separate embedded terminal ([docs][nb-claude]) | ❌ The desktop must stay awake; processes stop on quit ([mobile][nb-mobile], [terminal][nb-terminal]) | ✅ OS notifications; a free iOS app pushes while you are away ([docs][nb-notify]) | ◐ A built-in browser preview; no port handling ([docs][nb-browser]) | ✅ Worktree sessions ([docs][nb-worktrees]) |
| **Vibe Kanban** | ❌ Its own chat UI over the agents' JSON logs ([docs][vk-chat]) | ◐ A local server survives a closed browser, but stops when its host sleeps ([docs][vk-general]) | ✅ Sounds and browser push ([docs][vk-general]) | ✅ A dev server script and a preview browser ([docs][vk-browser]) | ✅ A worktree per repo per workspace ([docs][vk-start]) |
| **tmux** | ◐ Re-draws from its own grid. Its own scrollback; no kitty keyboard protocol in 3.8; the agent's OSC 52 dropped by default; OSC 9/777 only with passthrough on ([options][tm-options], [kitty PR][tm-kitty]) | ◐ Survives detach, sleep and network loss; you reattach by hand; lost on reboot ([man page][tm-man]) | ◐ Its own bell and activity alerts; the agent's own need passthrough ([options][tm-options]) | ❌ ([man page][tm-man]) | ❌ ([man page][tm-man]) |
| **zellij** | ◐ Re-draws, with its own scrollback. Only the first kitty keyboard level. Forwards OSC 52 ([options][zj-options], [0.41][zj-0410]) | ◐ Survives detach and network loss. After a reboot, layouts and commands come back, processes do not ([options][zj-options]) | ✅ Forwards OSC 9, 99 and 777 since 0.45 ([0.45][zj-0450]) | ❌ ([options][zj-options]) | ❌ ([options][zj-options]) |
| **zmx** | ✅ Raw bytes. But its client sends a full reset on detach, which wipes Ghostty's scrollback ([repo][gh-zmx], [note][n-host]) | ◐ Survives disconnect and sleep; lost on reboot ([issue][zmx-76]) | ◐ The agent's alerts pass only while attached ([repo][gh-zmx]) | ❌ ([repo][gh-zmx]) | ❌ ([repo][gh-zmx]) |
| **Herdr** | ◐ Re-draws through libghostty-vt in its own UI: no native scrollback, and the agent's OSC 9/777 are dropped ([issue][hd-3004], [note][n-herdr]) | ✅ Survives detach and sleep and reconnects by itself; resumes 18 agents after a restart ([docs][hd-state]) | ◐ Its own alerts from detected state; the agent's own are not forwarded ([docs][hd-config]) | ❌ None found ([docs][hd-remote]) | ✅ Worktree workspaces ([docs][hd-config]) |
| **mosh** | ❌ Syncs only the visible screen: no scrollback, no kitty keyboard, OSC 9/777 dropped ([source][mosh-term]) | ◐ Survives roaming and sleep; not a client restart, another device or a reboot ([issue][mosh-394]) | ◐ The bell only ([source][mosh-term]) | ❌ And it needs UDP 60000–61000 open ([site][mosh-site]) | ❌ ([site][mosh-site]) |
| **Eternal Terminal** | ✅ A raw byte stream; kitty keys and OSC untested (UNVERIFIED) ([protocol][et-proto]) | ◐ Survives sleep and network change with catch-up. No reattach after a client restart in v7.0.0; lost on reboot ([protocol][et-proto]) | ◐ The agent's alerts should pass as bytes (UNVERIFIED); none of its own ([protocol][et-proto]) | ✅ Forward and reverse tunnels ([client][et-client]) | ❌ ([repo][gh-et]) |
| **Tailscale SSH** | ✅ A plain SSH channel ([docs][ts-ssh]) | ❌ A transport only; a daemon restart or policy change ends sessions ([docs][ts-ssh]) | ◐ The agent's alerts pass while connected; none of its own ([docs][ts-ssh]) | ✅ `tailscale serve` on the tailnet, Funnel in public ([serve][ts-serve], [Funnel][ts-funnel]) | ❌ ([docs][ts-ssh]) |
| **simstim** (today) | ✅ Raw PTY bytes; the client draws nothing while attached. End-to-end checks prove OSC 8, 52 and 777, truecolour and kitty keys byte-identical ([README][ss-readme], [architecture][ss-arch]) | ✅ Survives detach, a killed client, sleep and network loss, and replays missed output by byte offset. After a reboot, a boot unit restarts each agent with its resume command: the conversation comes back, not the running turn ([changelog][ss-changelog]) | ✅ The agent's own OSC 9/777 pass through; missed ones are collapsed on reconnect. A desktop alert when another session needs you. No phone or Slack ([architecture][ss-arch]) | ◐ `ports` and `forward` to 127.0.0.1 over SSH; never public ([changelog][ss-changelog]) | ✅ `new --worktree`: its own branch, kept if it has work ([changelog][ss-changelog]) |

## Table 3: collaboration and lifecycle

| Product | Sharing / watch | Snapshots and hibernation | Secrets | Setup steps to a first running agent |
| --- | --- | --- | --- | --- |
| **Amp** | ✅ Thread links, multiplayer orbs, shared runners ([orbs explained][amp-explained], [runners][amp-runners]) | ✅ A setup snapshot reused for 72 h; paused orbs cost nothing ([docs][amp-custom]) | ✅ Personal, project and workspace secrets, masked in output; short-lived OIDC tokens ([docs][amp-secrets]) | 6: sign in, connect GitHub, create a project, buy credits or a tier, start an orb thread, prompt ([docs][amp-start]) |
| **Claude Code on the web** | ◐ A link to a view that does not update live ([docs][cc-cloud]) | ✅ Setup result cached for about 7 days; idle VMs pause, then are reclaimed ([environments][cc-env]) | ◐ Environment variables are readable by all users of the environment. Network secrets are added by a proxy outside the VM, and GitHub tokens never enter it ([environments][cc-env]) | 5: buy a plan, sign in, authorise GitHub, pick a repo, type the task ([quick start][cc-quick]) |
| **Codex cloud** | ◐ Environments shared in Enterprise; task sharing UNVERIFIED ([environments][cx-env]) | ✅ A published environment snapshot; task state kept 7 days ([environments][cx-env]) | ✅ Variables, proxy-injected network secrets, a personal vault ([environments][cx-env]) | 6: sign in on Plus, create an environment, connect GitHub, let it prepare, publish, start a task ([docs][cx-cloud]) |
| **Cursor cloud agents** | ◐ Teammates with repo access can view; follow-ups need an admin setting ([docs][cu-agent]) | ✅ Pre-warmed disk snapshots; idle VMs hibernate ([docs][cu-builds]) | ✅ User, team or environment secrets, encrypted; runtime secrets redacted ([docs][cu-secrets]) | 5: buy Premium, connect GitHub, set a spend limit, pick a repo, type the task ([docs][cu-agent]) |
| **Jules** | ? None documented. UNVERIFIED ([docs][jl-tasks]) | ✅ An environment snapshot per repo; pause and resume ([docs][jl-env]) | ◐ Repo variables, enabled per task; encryption not documented ([changelog][jl-envvars]) | 5: sign in, connect GitHub, pick a repo, write a prompt, approve the plan ([docs][jl-docs]) |
| **Devin** | ◐ Session links and edit access, on Teams only ([pricing][dv-pricing]) | ✅ A blueprint built into a snapshot; sessions sleep and wake ([docs][dv-blueprints]) | ✅ Organisation, personal, repo or session secrets, encrypted ([docs][dv-secrets]) | 5: sign up, choose Pro, connect GitHub, approve the environment, type the task ([docs][dv-first]) |
| **Factory** | ✅ Session URLs for the organisation ([CLI][fa-cli]) | ✅ Environment snapshots rebuilt every few hours; computers pause and resume ([docs][fa-envs]) | ✅ Personal or organisation secrets; not readable again ([docs][fa-secrets]) | 5: sign up for Pro, connect GitHub, create a computer, open a session on it, type the task ([docs][fa-computers]) |
| **Warp** | ✅ View and edit roles, many viewers, web links ([docs][wp-share]) | ◐ A fresh Docker environment per run; diffs saved for handoff; no VM pause ([docs][wp-snap]) | ✅ Team or personal secrets, not readable back ([docs][wp-secrets]) | 5: install Warp, sign in, open a cloud agent, create an environment, send the task. Claude Code in the cloud adds a paid plan and an API key ([quick start][wp-quick]) |
| **GitHub Codespaces** | ◐ VS Code Live Share and port visibility; no terminal watch ([docs][cs-collab]) | ◐ A stop keeps the disk, not processes; prebuilds ([docs][cs-prebuilds]) | ✅ Per-user secrets scoped to repos ([docs][cs-secrets]) | 6: GitHub account, `gh auth login`, create, `gh codespace ssh`, install the agent, run and log in ([docs][cs-cli]) |
| **Ona** | ◐ Port and organisation sharing; live watch UNVERIFIED ([ports][ona-ports]) | ◐ Prebuilds expire after 5 days; a stop keeps the disk ([docs][ona-prebuilds]) | ✅ User, project and organisation secrets, encrypted ([docs][ona-secrets]) | Closed to new users; 6 steps for existing ones ([docs][ona-start]) |
| **Coder** | ✅ Share a workspace with users or groups ([docs][co-share]) | ◐ A stop keeps the volume; prebuilt workspaces on Premium ([docs][co-prebuilt]) | ✅ User secrets as variables or files; plaintext in the database unless encryption is set up ([docs][co-secrets]) | 7: Docker, install, start the server, make a template, a workspace, `coder ssh`, run the agent ([docs][co-start]) |
| **E2B** | ◐ Reconnect to a PTY; no multi-viewer ([docs][e2b-pty]) | ✅ Pause with memory, snapshots, forks, auto-pause ([docs][e2b-persist]) | ✅ A proxy injects secrets into outbound HTTPS; write-only ([docs][e2b-secrets]) | 5: sign up for a key, install the CLI, log in, create a Claude sandbox, run and log in ([docs][e2b-claude]) |
| **Daytona** | ◐ Signed preview links and a web terminal ([docs][dt-preview]) | ✅ Snapshots, VM pause with memory, archive to object storage ([docs][dt-billing]) | ✅ A proxy swaps placeholders for values ([docs][dt-secrets]) | 6: sign up for a key, install the CLI, log in, create a sandbox, `daytona ssh`, run the agent ([guide][dt-claude]) |
| **Modal** | ◐ Tunnel URLs; attach by sandbox ID ([docs][md-shell]) | ◐ Filesystem and memory snapshots restore into a new sandbox; no pause ([docs][md-snap]) | ✅ Secrets as variables; HTTPS injection in beta ([docs][md-secrets]) | 6: sign up, install, set up, start a sandbox, install the agent, run and log in ([docs][md-shell]) |
| **Conductor** | ◐ Shared chats and follow, on paid plans only ([docs][cd-collab]) | ◐ Per-turn checkpoints, archive and restore; cloud sleeps and wakes with files, not processes ([docs][cd-checkpoints]) | ✅ Copies ignored `.env` files into workspaces; organisation secrets in the cloud ([docs][cd-files]) | 5: download, install, pass the setup checks, add a repo, prompt ([docs][cd-first]) |
| **Terragon** (shut down) | ✅ Private, link or repo visibility ([code][tg-share]) | ✅ Idle sandboxes hibernated and resumed ([docs][tg-security]) | ✅ Global or per-repo variables, encrypted ([docs][tg-env]) | 4 (UNVERIFIED): sign in with GitHub, grant repos, connect an agent plan, type a task ([docs][tg-quick]) |
| **Sculptor** | ❌ None found ([spec][sc-spec]) | ❌ Workspaces can only be deleted ([docs][sc-workspaces]) | ◐ Loads `.env` files from home and repo; no copy into worktrees ([docs][sc-workspaces]) | 5: download, enter a name, pass a dependency check, add a repo, prompt ([docs][sc-start]) |
| **claude-squad** | ❌ ([repo][gh-squad]) | ◐ Commit and pause a session to free its worktree, then resume ([repo][gh-squad]) | ❌ No `.env` copy and no setup hook ([issue][sq-260]) | 4: install tmux and gh, install it, run it in a repo, press `n` ([repo][gh-squad]) |
| **Crystal** (now Nimbalyst) | ✅ Free read-only, expiring share links; live collaboration on Teams ([docs][nb-share]) | ◐ File history and worktree archive ([docs][nb-history]) | ❌ No `.env` copy into worktrees found ([repo][gh-nimbalyst]) | 4: install, open a project, set the AI provider, start a session ([docs][nb-quick]) |
| **Vibe Kanban** | ◐ Phone access through its cloud relay; UNVERIFIED since the shutdown ([docs][vk-remote]) | ◐ Archive keeps the chat and the files ([docs][vk-manage]) | ✅ Copies `.env`-style files into worktrees; setup scripts ([code][vk-copy]) | 4: log in to the agent, run it with `npx`, skip sign-in, create a workspace ([docs][vk-start]) |
| **tmux** | ✅ Many clients, read-only attach, other users by permission ([man page][tm-man]) | ❌ None built in; a third-party plugin restores layouts ([plugin][tm-resurrect]) | ❌ None; it uses the SSH login ([man page][tm-man]) | 4: install on the host, `ssh`, start tmux, run the agent ([wiki][tm-start]) |
| **zellij** | ✅ Multiplayer, and a web client with read-only tokens ([news][zj-multi], [0.44][zj-0440]) | ◐ Saves layouts; commands wait for Enter after a reboot ([docs][zj-resurrect]) | ❌ None for agents ([docs][zj-web]) | 4: install on the host, `ssh`, attach, run the agent ([docs][zj-install]) |
| **zmx** | ◐ Many clients; typing takes the lead; no read-only mode ([repo][gh-zmx]) | ❌ No save across a reboot ([issue][zmx-76]) | ❌ None ([repo][gh-zmx]) | 3: install on the host, `ssh -t` into an attach, run the agent ([repo][gh-zmx]) |
| **Herdr** | ◐ Many clients per server; a frame stream; no read-only share ([CLI][hd-cli]) | ◐ Rotating layout snapshots; agents resumed after a restart ([docs][hd-state]) | ❌ None; your SSH config and keys ([docs][hd-connect]) | 4: install, connect with `--remote`, confirm the install, run the agent ([docs][hd-install]) |
| **mosh** | ❌ One client ([issue][mosh-394]) | ❌ ([site][mosh-site]) | ❌ None; it starts over SSH ([site][mosh-site]) | 5: install on host and laptop, open UDP ports, connect, run the agent ([site][mosh-site]) |
| **Eternal Terminal** | ❌ One client per session ([client][et-client]) | ❌ ([repo][gh-et]) | ❌ None; SSH plus a key per session ([protocol][et-proto]) | 5: install the server, open TCP 2022, install the client, connect, run the agent ([repo][gh-et]) |
| **Tailscale SSH** | ❌ No shared terminal; a browser console for admins, in beta ([docs][ts-console]) | ❌ ([docs][ts-ssh]) | ◐ No SSH keys to manage; nothing for agents ([docs][ts-ssh]) | 5: install and log in on the host, turn SSH on, install and log in on the laptop, `ssh`, run the agent ([docs][ts-ssh]) |
| **simstim** (today) | ◐ One user on several devices; the last to type sets the size. No read-only watchers and no other people ([architecture][ss-arch]) | ◐ Screen snapshots for new clients and after a reboot. Provider scripts stop and start a VM and wake it on attach. No idle auto-stop and no disk snapshots ([changelog][ss-changelog]) | ◐ Stores none by design. Agent logins live on the host; `.worktreeinclude` copies `.env` into a worktree ([architecture][ss-arch]) | 4: build or install the client, `host add`, `new <host> --agent claude`, log in to the agent once. Needs a host you can already reach with `ssh` ([README][ss-readme]) |

## Open-source repositories

From `gh repo view <repo> --json stargazerCount,licenseInfo,latestRelease,pushedAt` on 15 October 2026.

| Repository | Stars | Licence | Latest release | Last push | Note |
| --- | --- | --- | --- | --- | --- |
| [anthropics/claude-code][cc-licence] | 150,993 | none (proprietary) | v2.1.300, 14 Oct 2026 | 14 Oct 2026 | Issues and releases only |
| [openai/codex][gh-codex] | 129,354 | Apache-2.0 | 0.162.1, 9 Oct 2026 | 15 Oct 2026 | The CLI; the cloud is closed |
| [daytonaio/daytona][gh-daytona] | 71,544 | none shown | v0.190.0, 23 Jun 2026 | 24 Jul 2026 | Archived; was AGPL-3.0 |
| [warpdotdev/warp][wp-faq-lic] | 65,468 | AGPL-3.0 | 3 Jun 2026 (stable tag) | 15 Oct 2026 | Client only |
| [tmux/tmux][gh-tmux] | 49,844 | ISC | 3.8, 9 Sep 2026 | 15 Oct 2026 | |
| [herdrdev/herdr][gh-herdr] | 44,136 | Apache-2.0 | v0.9.5, 14 Oct 2026 | 14 Oct 2026 | Repo created 27 Mar 2026 |
| [tailscale/tailscale][gh-tailscale] | 37,568 | BSD-3-Clause | v1.104.1, 7 Oct 2026 | 15 Oct 2026 | |
| [zellij-org/zellij][gh-zellij] | 35,738 | MIT | v0.45.1, 28 Aug 2026 | 14 Oct 2026 | |
| [BloopAI/vibe-kanban][gh-vk] | 28,373 | Apache-2.0 | v0.1.44, 24 Apr 2026 | 19 Sep 2026 | Community-maintained |
| [coder/coder][gh-coder] | 17,022 | AGPL-3.0 | v2.37.4, 5 Oct 2026 | 15 Oct 2026 | |
| [mobile-shell/mosh][gh-mosh] | 14,570 | GPL-3.0 | 1.4.0, 27 Oct 2022 | 22 Mar 2026 | |
| [e2b-dev/E2B][gh-e2b] | 14,364 | Apache-2.0 | 13 Oct 2026 (SDK) | 14 Oct 2026 | |
| [gitpod-io/gitpod][gh-gitpod] | 13,798 | AGPL-3.0 | 2022.11.3, 5 Jun 2023 | 14 Oct 2026 | Gitpod Classic |
| [smtg-ai/claude-squad][gh-squad] | 8,607 | AGPL-3.0 | v1.0.20, 20 Aug 2026 | 20 Aug 2026 | |
| [MisterTea/EternalTerminal][gh-et] | 3,917 | Apache-2.0 | v7.0.0, 7 Jul 2026 | 13 Oct 2026 | |
| [stravu/crystal][gh-crystal] | 3,130 | MIT | v0.3.5, 26 Feb 2026 | 26 Feb 2026 | Deprecated |
| [neurosnap/zmx][gh-zmx] | 2,210 | MIT | v0.8.1, 5 Sep 2026 | 3 Oct 2026 | |
| [coder/xum][gh-xum] | 2,054 | AGPL-3.0 | v0.31.0, 9 Oct 2026 | 15 Oct 2026 | Formerly `coder/mux` |
| [Nimbalyst/nimbalyst][gh-nimbalyst] | 1,905 | MIT | v0.80.6, 9 Oct 2026 | 14 Oct 2026 | Crystal's successor |
| [e2b-dev/runtime][gh-e2b-runtime] | 1,721 | Apache-2.0 | 2026.30, 10 Sep 2026 | 15 Oct 2026 | Formerly `e2b-dev/infra` |
| [modal-labs/modal-client][gh-modal] | 525 | Apache-2.0 | none | 14 Oct 2026 | Client only |
| [terragon-labs/terragon-oss][gh-terragon] | 261 | Apache-2.0 | none | 10 Feb 2026 | Snapshot after shutdown |
| [imbue-ai/sculptor][gh-sculptor] | 239 | MIT | sculptor-v0.48.0, 21 Sep 2026 | 14 Oct 2026 | Open since June 2026 |

Conductor has no public source. Its release repository, `meltylabs/conductor-releases`, has 9 stars and no licence.

## Near misses: vendor agents on your own machine

These are not rows in the matrix, but they come closest to simstim's idea of "your agent on your host". Each one keeps one vendor's agent, account and UI.

| Offer | What it does | What it lacks against simstim |
| --- | --- | --- |
| Claude Desktop SSH sessions ([docs][cc-desktop]) | Runs Claude Code on your SSH host; it keeps going while the app is away | A GUI, not the TUI; Claude Code only |
| Claude Remote Control ([docs][cc-rc]) | Steer a local or SSH-host CLI session from the web or a phone, with a worktree per session | Claude Code only; the web and phone show their own view |
| ChatGPT desktop "Connect to an SSH host" ([docs][cx-remote]) | Runs Codex on your host, driven from desktop or phone | Codex only; the app's UI |
| Amp runners ([docs][amp-runners]) | Threads run on your machine, free since September 2026 | Amp's account and control plane; no portals |
| Cursor My Machines ([docs][cu-self]) | Cursor's tools run on your host | The agent loop stays in Cursor's cloud; needs Premium ($30) |
| Devin Outposts ([docs][dv-outposts]) | Devin's commands run on your machines | The agent loop stays at Cognition; enabled per organisation |
| Factory Remote Control ([docs][fa-rc]) | Droid on your machine through Factory's relay | Droid only; a relay |
| Warp Remote Control ([docs][wp-rc]) | Publishes any CLI agent's session to the web and phone | Needs Warp open; Warp's viewer; ends with the session |
| Xum, by Coder ([docs][xum-ssh]) | AGPL desktop app with SSH, worktree and Docker runtimes, and notifications | Runs its own agent loop, not the agents' TUIs |

## Paid features simstim can give for free

Effort is for what simstim still lacks: **S** is days, **M** is a week or two, and **L** is a month or more. "Done" means it is built today.

| Feature | Who sells it, and the price | simstim today | Effort |
| --- | --- | --- | --- |
| The agent keeps working with the laptop closed | Conductor Pro $50 a month ([pricing][cd-pricing]); Cursor Premium $30 ([pricing][cu-pricing]); Claude cloud sessions from Pro $20 ([pricing][cc-pricing]); Codex cloud from Plus $20 ([pricing][cx-pricing]); Devin Pro $20 ([pricing][dv-pricing]); Amp orbs $0.08–2.13 an hour ([sizes][amp-sizes]) | ✅ Built | Done |
| Runs on your own machines | Claude self-hosted environments, Team and Enterprise ([docs][cc-selfhost]); Cursor My Machines, Premium $30 ([docs][cu-self]); Warp workers, Enterprise ([FAQ][wp-faqs]); Ona runners, Enterprise ([docs][ona-runners]); E2B BYOC, Enterprise from $3,000 a month ([billing][e2b-billing]); Daytona BYOC, Enterprise ([docs][dt-byoc]) | ✅ The only mode | Done |
| No session time limit | E2B: 1 h free, 24 h on Pro $150 a month ([billing][e2b-billing]); Modal: 24 h on every plan ([docs][md-sandboxes]); Conductor Cloud: 23 h 50 min on Pro $50 ([docs][cd-cloud]); Codespaces: idle stop at 4 h at most ([docs][cs-timeout]) | ✅ No limit; only the memory cap you set | Done |
| More agents at once | Jules: 3 free, 15 on $19.99, 60 on $99.99 and up ([limits][jl-limits]); Devin: 10 on Pro, unlimited on Teams from $80 ([pricing][dv-pricing]); E2B: 20 free, 100 on Pro $150 ([billing][e2b-billing]); Coder: 5 Coder Agents, more on AI Premium ([licensing][co-licensing]) | ✅ As many as the host holds | Done |
| Bigger machines | Codex: 4 vCPU from Pro $100 or Business $25 a user ([environments][cx-env]); Cursor: larger VMs on Enterprise ([docs][cu-agent]); Amp: the largest orb on the top tier ([pricing][amp-pricing]) | ✅ Whatever host you bring | Done |
| A shell and port forwards into the agent's machine | Devin Pro $20 ([docs][dv-ssh]); Factory Pro $20 ([docs][fa-computers]); Cursor Premium $30 ([setup][cu-setup]); Amp custom portal domains on paid tiers ([portals][amp-portals]) | ✅ `ports`, `forward`, and an SSH shell from host details | Done |
| A worktree per task, with `.env` copied in | Free in Conductor, Vibe Kanban and Nimbalyst; part of the paid cloud plans elsewhere | ✅ `new --worktree` and `.worktreeinclude` | Done |
| Alerts on a phone, in Slack or by email | Cursor iOS push and Slack, Premium $30 ([mobile][cu-mobile]); Devin Slack and push, Pro $20 ([Slack][dv-slack]); Codex push, email or SMS, Plus $20 ([docs][cx-notify]); Conductor iOS app, Pro $50 ([pricing][cd-pricing]) | ❌ Desktop only, and only while a client runs | **S**: a `notify` command in the host config, run on "needs input" and "ended" with the event as JSON. The user points it at ntfy, Pushover or a Slack webhook. It runs on the host, so it fires while the laptop sleeps, with no relay of ours. |
| Idle sleep of cloud VMs | Amp orbs pause when idle and bill per minute ([docs][amp-custom]); Coder dormancy and autostop, Premium ([licensing][co-licensing]); Ona auto-stop policies, Core ([runners][ona-runners]); Devin custom idle timeout, Enterprise ([pricing][dv-pricing]) | ◐ Provider scripts start and stop a VM and wake it on attach; no automatic stop | **S–M**: stop the VM when every session is idle or waiting and no client is attached for N minutes. Restore brings the conversations back. It needs care never to stop a working agent. |
| Watch or share a session | Conductor Multiplayer, Pro $50 ([docs][cd-collab]); Devin, Teams from $80 ([pricing][dv-pricing]); Cursor team follow-ups, Teams from $30 a seat ([docs][cu-agent]); Factory, Teams $60 plus $40 a seat ([pricing][fa-pricing]) | ◐ One user, many devices | **M**: a read-only attach that drops input (the holder already serves many clients). Other people need their own host accounts and socket permissions. Live web links stay out of scope. |
| Prebuilt environments | Cursor Builds, Premium $30 ([docs][cu-builds]); Coder prebuilt workspaces, Premium ([docs][co-prebuilt]); Devin blueprints, Pro $20 ([docs][dv-blueprints]); Codex published environments, Plus $20 ([environments][cx-env]) | ◐ A setup hook per repo; no image cache | **M**: `snapshot` and `create --from` actions in the provider script, so the provider does the work. **L** for anything generic. |
| Scheduled and triggered runs | Cursor automations, Premium $30 ([pricing][cu-pricing]); Terragon automations, was $25 ([plans][tg-plans]); Amp automations ([note][n-amp]) | ◐ `new --detach` and `send` already work from cron or CI | **S**: document recipes |
| Bring the work back to the laptop | Claude `--teleport` ([docs][cc-cloud]); `codex apply` ([commands][cx-cmds]); Terragon `terry pull` ([docs][tg-cli]). Each comes with the paid plan that gives cloud sessions. | ❌ | **S**: `pull <session>` fetches the session's branch over the same SSH |
| Session recording and audit | Tailscale SSH recording, sales-only add-on ([recording][ts-rec]); Coder audit logs, Premium ([licensing][co-licensing]) | ◐ An 8 MiB output ring per session | **S**: opt-in saving of a session's output to a file; off by default for privacy |
| Managed secrets with redaction or proxy injection | Cursor runtime secrets, Premium $30 ([docs][cu-secrets]); Codex network secrets, Plus $20 ([environments][cx-env]); Claude network secrets, Pro $20 ([environments][cc-env]); E2B and Daytona proxies ([E2B][e2b-secrets], [Daytona][dt-secrets]) | ◐ Stores none, by design | **L**, and it goes against principle 5 ("no stored secrets"). Better: document a per-host environment file. |

## Where simstim is behind

1. **It is not released.** Today you build it from source with Go and just; there is no Homebrew tap and no notarised build ([README][ss-readme]). Herdr installs with one `curl` line or `brew` ([docs][hd-install]). Hosted products need only a browser.
2. **You bring the host.** simstim needs a Linux host that you can already reach with `ssh` ([architecture][ss-arch]). Hosted products need only a GitHub login. The first run also needs you to install the agents and log in to them on the host.
3. **No phone or web client.** This is a non-goal ([plan][ss-plan]), but Claude, Codex, Cursor, Devin, Amp, Conductor and Nimbalyst all have phone apps or a web UI. simstim's alerts reach only the desktop, and only while a client runs.
4. **No sharing with other people.** tmux has read-only attach ([man page][tm-man]). zellij has read-only web tokens ([0.44][zj-0440]). Warp has view and edit roles ([docs][wp-share]). Amp has multiplayer orbs ([note][n-amp]). simstim serves one user on several devices.
5. **Weaker isolation.** All sessions on a host share one OS user, one filesystem and one network. An agent with broad permissions can read another session's files and the host's logins. Hosted tools give each task its own VM: Claude, Codex, Cursor, Jules, Devin. E2B and Amp give each task a microVM. Claude and Codex also control network egress ([environments][cc-env], [environments][cx-env]). simstim gives a memory cap and a worktree, not a sandbox.
6. **No environment snapshots, prebuilds or memory hibernation.** Amp, Cursor, Devin, Factory, E2B and Daytona start a task from a saved image, and some pause with memory intact. After a reboot, simstim brings back the conversation, not the running turn. That matches Claude's cloud sessions after a reclaim ([docs][cc-cloud]), but not an E2B or Amp pause.
7. **No review flow.** There is no diff view, no pull-request button, no CI auto-fix, and no Slack, Linear or GitHub triggers. This is by design, since simstim is not an agent wrapper, but teams do buy these features.
8. **No automatic idle stop** for cloud VMs. Amp's orbs pause after 5 to 20 minutes and cost nothing while paused ([note][n-amp]).
9. **Community.** Herdr gathered 44,000 stars in seven months. tmux has 50,000 and zellij 36,000. simstim is private and has no users yet.

## Where simstim is ahead

1. **The whole brief, in one tool.** simstim is the only row with ✅ in all four of: any agent, real TUI pass-through, persistence, and self-hosting. zmx, Eternal Terminal, Herdr and Coder each miss one (Summary item 1).
2. **Fidelity.** The agent's bytes are passed through, and missed output is replayed by byte offset, so native scrollback stays continuous ([architecture][ss-arch]). The other tools fall short:
   - tmux 3.8 has no kitty keyboard protocol and drops the agent's OSC 52 by default ([options][tm-options]).
   - zellij supports only the first kitty level ([0.41][zj-0410]).
   - mosh supports none ([source][mosh-term]).
   - Herdr re-draws the screen and drops OSC 9/777 ([issue][hd-3004]).
   - zmx wipes the scrollback on detach ([note][n-host]).
3. **No caps and no compute markup.** There are no time limits, no concurrency limits and no per-minute fees; you pay only for your host. For scale:
   - An always-on Amp a1.medium orb (4 vCPU, 8 GB) at $0.33 an hour would cost about $241 a month ([sizes][amp-sizes]). Orbs pause when idle, so real bills are lower.
   - A 4-core codespace costs $0.36 an hour ([billing][cs-billing]).
   - [The first plan](../contributing/history.md) cites a 4 vCPU, 8 GB VPS at €8.49 a month.
4. **Any agent login.** simstim runs the agent's own CLI on your host, with whatever subscription or key you give it. Claude's cloud sessions refuse Bedrock, Vertex and API-key logins ([docs][cc-cloud]). Warp's cloud needs an Anthropic API key stored as a Warp secret ([quick start][wp-quick]).
5. **Any network path.** ProxyJump, GCP IAP, AWS SSM and Tailscale all work through `~/.ssh/config` ([connectivity][n-conn]), including private networks behind a jump host. Hosted runtimes reach a private network only through special features, such as Codex's Tailscale option ([environments][cx-env]).
6. **Nothing listens, and nothing phones home.** Other tools need an open port, a relay or telemetry:
   - Eternal Terminal needs TCP 2022 open and sends telemetry by default ([repo][gh-et]).
   - mosh needs UDP ports ([site][mosh-site]).
   - Factory's Remote Control goes through a relay ([docs][fa-rc]).
   - Cursor's and Devin's own-machine modes keep the agent loop in their cloud ([Cursor][cu-self], [Devin][dv-outposts]).
7. **Your code and credentials stay on your hosts.** Every hosted product holds your repository, and most hold your secrets.
8. **No vendor risk.** Several products in this matrix were shut down, sold, closed or re-priced in 2025–2026 (see [Changes in 2025–2026](#changes-in-20252026)). A self-hosted Apache-2.0 tool over plain SSH cannot be withdrawn the same way.
9. **Remote image paste.** Ctrl+V uploads a screenshot and pastes its path ([README][ss-readme]). Herdr does the same ([note][n-herdr]). The terminal tools (tmux, zellij, zmx, mosh, Eternal Terminal) have no equivalent.

## Unverified

- Amp's $200 top tier: it appears only in the pricing page's script, not in its text.
- Whether Codex's Free and Go plans include cloud tasks.
- Factory: whether Droid Computers come with Pro or Plus (the docs and the pricing page disagree), and when factory.ai moved to factory.com.
- Warp: whether Claude Code and Codex in the cloud need a paid plan (the docs and the pricing page disagree), whether cloud runs have port forwarding, and whether there is phone push.
- Factory phone push.
- Ona's current licence, and the exact date its own agent was dropped.
- Coder: whether any "agent needs input" notification remains after Tasks was removed.
- Jules: session sharing; Jules has had no changelog entry since March 2026.
- Terragon: its setup steps and whether it had port previews.
- Vibe Kanban: whether phone access through its relay still works.
- Eternal Terminal: kitty keys and OSC 9/777 pass-through in practice.
- E2B: when `infra` became `runtime`.
- Tailscale: a pricing table cell that limits Basic SSH to 5 hosts on Standard but not on Personal.

## Sources

Each table cell links its own primary source. The link targets are defined at the end of this file. All were read on 15 October 2026.

[n-amp]: amp-ux-and-orbs.md
[n-herdr]: herdr.md
[n-conn]: connectivity-and-providers.md
[n-host]: host-runtime.md

[ss-readme]: ../../README.md
[ss-changelog]: ../../CHANGELOG.md
[ss-arch]: ../explanation/architecture.md
[ss-plan]: ../explanation/design.md#non-goals
[ss-licence]: ../../LICENSE

[amp-pricing]: https://ampcode.com/pricing
[amp-sizes]: https://ampcode.com/docs/orbs/sizes-and-costs
[amp-runners]: https://ampcode.com/docs/cli/runners
[amp-npm]: https://www.npmjs.com/package/@sourcegraph/amp
[amp-what]: https://ampcode.com/what-are-orbs
[amp-start]: https://ampcode.com/docs/orbs/getting-started
[amp-custom]: https://ampcode.com/docs/orbs/customizing
[amp-apps]: https://ampcode.com/docs/macos-and-ios
[amp-portals]: https://ampcode.com/docs/orbs/portals
[amp-explained]: https://ampcode.com/notes/orbs-explained
[amp-secrets]: https://ampcode.com/docs/orbs/handling-secrets
[amp-free]: https://ampcode.com/news/free-agent
[amp-inc]: https://ampcode.com/news/amp-inc
[amp-claude]: https://ampcode.com/news/use-your-claude-plan
[amp-sizes-news]: https://ampcode.com/news/size-the-orbs-of-production

[cc-pricing]: https://claude.com/pricing
[cc-cloud]: https://code.claude.com/docs/en/claude-code-on-the-web
[cc-env]: https://code.claude.com/docs/en/cloud-environments
[cc-selfhost]: https://code.claude.com/docs/en/self-hosted-environments
[cc-licence]: https://github.com/anthropics/claude-code/blob/main/LICENSE.md
[cc-desktop]: https://code.claude.com/docs/en/desktop
[cc-mobile]: https://code.claude.com/docs/en/mobile
[cc-quick]: https://code.claude.com/docs/en/web-quickstart
[cc-rc]: https://code.claude.com/docs/en/remote-control
[cc-blog]: https://claude.com/blog/claude-code-on-the-web
[cc-w32]: https://code.claude.com/docs/en/whats-new/2026-w32

[cx-pricing]: https://learn.chatgpt.com/docs/pricing
[cx-env]: https://learn.chatgpt.com/docs/environments/cloud-environments
[cx-cloud]: https://learn.chatgpt.com/docs/cloud
[cx-cmds]: https://learn.chatgpt.com/docs/developer-commands
[cx-notify]: https://learn.chatgpt.com/docs/notifications
[cx-modes]: https://learn.chatgpt.com/docs/environments/modes
[cx-remote]: https://learn.chatgpt.com/docs/remote-connections
[cx-changelog]: https://learn.chatgpt.com/docs/changelog
[cx-new]: https://learn.chatgpt.com/docs/whats-new
[gh-codex]: https://github.com/openai/codex

[jl-limits]: https://jules.google/docs/usage-limits/
[jl-subs]: https://gemini.google/us/subscriptions/?hl=en
[jl-env]: https://jules.google/docs/environment/
[jl-npm]: https://www.npmjs.com/package/@google/jules
[jl-docs]: https://jules.google/docs
[jl-cli]: https://jules.google/docs/cli/reference
[jl-tasks]: https://jules.google/docs/tasks-repos/
[jl-preview]: https://jules.google/docs/changelog/2025-08-07/
[jl-envvars]: https://jules.google/docs/changelog/2025-10-01/
[jl-changelog]: https://jules.google/docs/changelog/

[cu-pricing]: https://cursor.com/pricing
[cu-billing]: https://cursor.com/help/account-and-billing/pricing
[cu-agent]: https://cursor.com/docs/cloud-agent
[cu-setup]: https://cursor.com/docs/cloud-agent/setup
[cu-self]: https://cursor.com/docs/cloud-agent/self-hosted
[cu-k8s]: https://github.com/anysphere/k8s-workers
[cu-cli]: https://cursor.com/docs/cli/overview
[cu-security]: https://cursor.com/docs/cloud-agent/security
[cu-mobile]: https://cursor.com/docs/cloud-agent/mobile
[cu-builds]: https://cursor.com/docs/cloud-agent/builds
[cu-secrets]: https://cursor.com/docs/cloud-agent/security-network
[cu-2]: https://cursor.com/changelog/2-0
[cu-june]: https://cursor.com/blog/june-2025-pricing
[cu-spacex]: https://cursor.com/blog/joining-spacex
[cu-x]: https://x.ai/news/x-subscription

[dv-pricing]: https://devin.ai/pricing
[dv-selfserve]: https://docs.devin.ai/admin/billing/self-serve
[dv-usage]: https://docs.devin.ai/admin/billing/usage
[dv-outposts]: https://docs.devin.ai/cloud/outposts/overview
[dv-handoff]: https://docs.devin.ai/work-with-devin/devin-handoff
[dv-cli-cloud]: https://docs.devin.ai/cli/cloud
[dv-slack]: https://docs.devin.ai/integrations/slack
[dv-ssh]: https://docs.devin.ai/cli/ssh
[dv-env]: https://docs.devin.ai/onboard-devin/environment
[dv-blueprints]: https://docs.devin.ai/onboard-devin/environment/blueprints
[dv-secrets]: https://docs.devin.ai/product-guides/secrets
[dv-first]: https://docs.devin.ai/get-started/first-run
[dv-plans]: https://cognition.com/blog/new-self-serve-plans-for-devin
[dv-desktop]: https://cognition.com/blog/introducing-devin-desktop
[dv-notes]: https://docs.devin.ai/release-notes/2026
[dv-clilog]: https://docs.devin.ai/cli/changelog/stable

[fa-pricing]: https://factory.com/pricing
[fa-ind]: https://docs.factory.com/pricing/individuals
[fa-org]: https://docs.factory.com/pricing/organizations
[fa-rc]: https://docs.factory.com/remote-control
[fa-npm]: https://registry.npmjs.org/droid/latest
[fa-computers]: https://docs.factory.com/droid-computers/overview
[fa-notes]: https://docs.factory.com/changelog/release-notes
[fa-worktrees]: https://docs.factory.com/factory-app/worktrees
[fa-cli]: https://docs.factory.com/droid-cli/cli-reference
[fa-envs]: https://docs.factory.com/environments/overview
[fa-secrets]: https://docs.factory.com/environments/prerequisites
[fa-fleet]: https://factory.com/news/droids-now-have-a-fleet-of-sandboxes

[wp-pricing]: https://www.warp.dev/pricing
[wp-harness]: https://docs.warp.dev/platform/harnesses/
[wp-faqs]: https://docs.warp.dev/platform/faqs/
[wp-faq-lic]: https://github.com/warpdotdev/warp/blob/main/FAQ.md
[wp-cli-agents]: https://docs.warp.dev/agents/cli-agents/overview/
[wp-rc]: https://docs.warp.dev/agents/cli-agents/remote-control/
[wp-runs]: https://docs.warp.dev/platform/viewing-cloud-agent-runs/
[wp-notify]: https://docs.warp.dev/agents/capabilities/agent-notifications/
[wp-envs]: https://docs.warp.dev/platform/environments/
[wp-worktrees]: https://docs.warp.dev/code/git-worktrees/
[wp-share]: https://docs.warp.dev/agents/local-agents/session-sharing/
[wp-snap]: https://docs.warp.dev/platform/handoff/snapshots/
[wp-secrets]: https://docs.warp.dev/platform/secrets/
[wp-quick]: https://docs.warp.dev/platform/quickstart/
[wp-build]: https://www.warp.dev/blog/warp-new-pricing-flexibility-byok
[wp-log26]: https://docs.warp.dev/changelog/2026
[wp-factories]: https://www.warp.dev/blog/open-infrastructure-for-building-a-software-factory

[cs-billing]: https://docs.github.com/en/billing/concepts/product-billing/github-codespaces
[cs-ssh]: https://cli.github.com/manual/gh_codespace_ssh
[cs-copilot]: https://github.blog/changelog/2026-01-14-github-copilot-cli-enhanced-agents-context-management-and-new-ways-to-install/
[cs-timeout]: https://docs.github.com/en/codespaces/setting-your-user-preferences/setting-your-timeout-period-for-github-codespaces
[cs-lifecycle]: https://docs.github.com/en/codespaces/about-codespaces/understanding-the-codespace-lifecycle
[cs-cli]: https://docs.github.com/en/codespaces/developing-in-a-codespace/using-github-codespaces-with-github-cli
[cs-create]: https://docs.github.com/en/codespaces/developing-in-a-codespace/creating-a-codespace-for-a-repository
[cs-collab]: https://docs.github.com/en/codespaces/developing-in-a-codespace/working-collaboratively-in-a-codespace
[cs-prebuilds]: https://docs.github.com/en/codespaces/prebuilding-your-codespaces/about-github-codespaces-prebuilds
[cs-secrets]: https://docs.github.com/en/codespaces/managing-your-codespaces/managing-your-account-specific-secrets-for-github-codespaces
[cs-ports]: https://docs.github.com/en/codespaces/developing-in-a-codespace/forwarding-ports-in-your-codespace

[ona-rename]: https://ona.com/stories/gitpod-is-now-ona
[ona-classic]: https://ona.com/stories/gitpod-classic-payg-sunset
[ona-openai]: https://ona.com/stories/ona-joins-openai
[ona-start]: https://ona.com/docs/ona/getting-started
[ona-billing]: https://ona.com/docs/ona/billing/overview
[ona-agents]: https://ona.com/docs/ona/agents/overview
[ona-pricing]: https://ona.com/pricing
[ona-runners]: https://ona.com/docs/ona/runners/overview
[ona-cli]: https://ona.com/docs/ona/integrations/cli
[ona-timeout]: https://ona.com/docs/ona/organizations/policies/environment-timeout
[ona-slack]: https://ona.com/docs/ona/integrations/configure-slack
[ona-ports]: https://ona.com/docs/ona/integrations/ports
[ona-envs]: https://ona.com/docs/ona/environments/overview
[ona-prebuilds]: https://ona.com/docs/ona/projects/prebuilds
[ona-secrets]: https://ona.com/docs/ona/configuration/secrets/overview
[gh-gitpod]: https://github.com/gitpod-io/gitpod

[co-pricing]: https://coder.com/pricing
[co-tasks]: https://coder.com/docs/ai-coder/agents/tasks-to-chats-migration
[co-agentapi]: https://github.com/coder/agentapi
[co-agents]: https://coder.com/docs/ai-coder/agents
[co-licensing]: https://coder.com/docs/ai-coder/agents/licensing-usage
[co-ai]: https://coder.com/docs/ai-coder
[co-access]: https://coder.com/docs/user-guides/workspace-access
[co-sched]: https://coder.com/docs/user-guides/workspace-scheduling
[co-notify]: https://coder.com/docs/admin/monitoring/notifications
[co-ports]: https://coder.com/docs/user-guides/workspace-access/port-forwarding
[co-share]: https://coder.com/docs/user-guides/shared-workspaces
[co-prebuilt]: https://coder.com/docs/admin/templates/extending-templates/prebuilt-workspaces
[co-secrets]: https://coder.com/docs/user-guides/user-secrets
[co-start]: https://coder.com/docs/get-started
[gh-coder]: https://github.com/coder/coder
[gh-xum]: https://github.com/coder/xum
[xum-ssh]: https://xum.cdr.dev/runtime/ssh.md

[e2b-pricing]: https://e2b.dev/pricing
[e2b-billing]: https://docs.e2b.dev/billing.md
[gh-e2b-runtime]: https://github.com/e2b-dev/runtime
[gh-e2b]: https://github.com/e2b-dev/E2B
[e2b-claude]: https://docs.e2b.dev/agents/claude-code.md
[e2b-connect]: https://docs.e2b.dev/cli/connect-to-sandbox.md
[e2b-pty]: https://docs.e2b.dev/sandbox/pty.md
[e2b-life]: https://docs.e2b.dev/faq/sandbox-lifetime.md
[e2b-hooks]: https://docs.e2b.dev/sandbox/lifecycle-events-webhooks.md
[e2b-url]: https://docs.e2b.dev/network/public-url.md
[e2b-fork]: https://docs.e2b.dev/sandbox/fork.md
[e2b-persist]: https://docs.e2b.dev/sandbox/persistence.md
[e2b-secrets]: https://docs.e2b.dev/secrets.md

[dt-closed]: https://www.daytona.io/dotfiles/updates/daytona-is-going-closed-source
[gh-daytona]: https://github.com/daytonaio/daytona
[dt-clients]: https://github.com/daytona/clients
[dt-pricing]: https://www.daytona.io/pricing
[dt-limits]: https://www.daytona.io/docs/en/limits.md
[dt-byoc]: https://www.daytona.io/docs/en/bring-your-own-compute.md
[dt-claude]: https://www.daytona.io/docs/en/guides/claude/claude-code-run-cli-sandbox.md
[dt-ssh]: https://www.daytona.io/docs/en/ssh-access.md
[dt-sandboxes]: https://www.daytona.io/docs/en/sandboxes.md
[dt-hooks]: https://www.daytona.io/docs/en/webhooks.md
[dt-preview]: https://www.daytona.io/docs/en/preview.md
[dt-billing]: https://www.daytona.io/docs/en/billing.md
[dt-secrets]: https://www.daytona.io/docs/en/secrets.md

[md-pricing]: https://modal.com/pricing
[gh-modal]: https://github.com/modal-labs/modal-client
[md-sandboxes]: https://modal.com/docs/guide/sandboxes.md
[md-shell]: https://modal.com/docs/cli/latest/shell.md
[md-tunnels]: https://modal.com/docs/guide/tunnels.md
[md-snap]: https://modal.com/docs/guide/sandbox-snapshots.md
[md-secrets]: https://modal.com/docs/guide/sandbox-secret-injection.md

[cd-pricing]: https://www.conductor.build/pricing
[cd-terms]: https://www.conductor.build/terms
[cd-harness]: https://www.conductor.build/docs/reference/harnesses
[cd-bigterm]: https://www.conductor.build/docs/reference/big-terminal-mode
[cd-cloud]: https://www.conductor.build/docs/cloud/working-with-cloud-workspaces
[cd-ios]: https://www.conductor.build/changelog/0.90.0-conductor-for-ios
[cd-sounds]: https://www.conductor.build/changelog/0.52.0-sounds-colors
[cd-settings]: https://www.conductor.build/docs/reference/settings/reference
[cd-ports]: https://www.conductor.build/changelog/0.83.0-auto-port-forwarding
[cd-worktrees]: https://www.conductor.build/docs/concepts/git-worktrees
[cd-collab]: https://www.conductor.build/docs/cloud/collaboration
[cd-checkpoints]: https://www.conductor.build/docs/reference/checkpoints
[cd-files]: https://www.conductor.build/docs/reference/files-to-copy
[cd-first]: https://www.conductor.build/docs/first-workspace
[cd-changelog]: https://www.conductor.build/changelog

[tg-home]: https://www.terragonlabs.com/
[tg-shutdown]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/resources/shutdown.mdx
[tg-billing]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/resources/billing.mdx
[tg-plans]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/www/src/lib/subscription-plan-config.ts
[gh-terragon]: https://github.com/terragon-labs/terragon-oss
[tg-agents]: https://github.com/terragon-labs/terragon-oss/tree/main/apps/docs/content/docs/agent-providers
[tg-cli]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/integrations/cli.mdx
[tg-security]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/security-permissions.mdx
[tg-notes]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/resources/release-notes.mdx
[tg-git]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/configuration/git-checkpointing.mdx
[tg-share]: https://github.com/terragon-labs/terragon-oss/blob/main/packages/shared/src/db/types.ts
[tg-env]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/configuration/environment-setup/environment-variables.mdx
[tg-quick]: https://github.com/terragon-labs/terragon-oss/blob/main/apps/docs/content/docs/quick-start.mdx

[sc-product]: https://imbue.com/product/sculptor
[gh-sculptor]: https://github.com/imbue-ai/sculptor
[sc-epoch]: https://github.com/imbue-ai/sculptor/commit/d2ab02fe
[sc-history]: https://github.com/imbue-ai/sculptor/blob/main/docs/history.md
[sc-backend]: https://github.com/imbue-ai/sculptor/blob/main/docs/help/experimental/container_backend.md
[sc-licence]: https://github.com/imbue-ai/sculptor/blob/main/LICENSE.md
[sc-spec]: https://github.com/imbue-ai/sculptor/blob/main/docs/specs/SPEC.md
[sc-term]: https://github.com/imbue-ai/sculptor/blob/main/samples/terminal_agents/claude-code/README.md
[sc-req]: https://github.com/imbue-ai/sculptor/blob/main/docs/specs/requirements.md
[sc-harness]: https://github.com/imbue-ai/sculptor/blob/main/docs/help/integrated_harnesses.md
[sc-openhost]: https://github.com/imbue-ai/sculptor/blob/main/openhost.toml
[sc-workspaces]: https://github.com/imbue-ai/sculptor/blob/main/docs/help/workspaces.md
[sc-start]: https://github.com/imbue-ai/sculptor/blob/main/docs/help/getting_started.md

[gh-squad]: https://github.com/smtg-ai/claude-squad
[sq-site]: https://smtg-ai.github.io/claude-squad/
[sq-licence]: https://github.com/smtg-ai/claude-squad/blob/main/LICENSE.md
[sq-app]: https://github.com/smtg-ai/claude-squad/blob/main/app/app.go
[sq-322]: https://github.com/smtg-ai/claude-squad/pull/322
[sq-185]: https://github.com/smtg-ai/claude-squad/issues/185
[sq-260]: https://github.com/smtg-ai/claude-squad/issues/260

[gh-crystal]: https://github.com/stravu/crystal
[cr-last]: https://github.com/stravu/crystal/releases/tag/v0.3.5
[gh-nimbalyst]: https://github.com/Nimbalyst/nimbalyst
[nb-mit]: https://github.com/Nimbalyst/nimbalyst/commit/2269c0ca
[nb-pricing]: https://nimbalyst.com/pricing/
[nb-oss]: https://nimbalyst.com/open-source/
[nb-claude]: https://nimbalyst.com/docs/developer-features/coding-with-claude-code-and-nimbalyst/
[nb-mobile]: https://nimbalyst.com/docs/mobile/mobile-app/
[nb-terminal]: https://nimbalyst.com/docs/developer-features/terminal-window/
[nb-notify]: https://nimbalyst.com/docs/setup-nimbalyst/ai-provider-setup-and-notifications/
[nb-browser]: https://nimbalyst.com/docs/visual-editors-powered-by-ai/browser/
[nb-worktrees]: https://nimbalyst.com/docs/developer-features/worktrees/
[nb-share]: https://nimbalyst.com/docs/session-management/share-link-to-a-session/
[nb-history]: https://nimbalyst.com/docs/file-management/file-history-and-restore/
[nb-quick]: https://nimbalyst.com/docs/getting-started/quickstart/

[vk-shutdown]: https://www.vibekanban.com/blog/shutdown
[gh-vk]: https://github.com/BloopAI/vibe-kanban
[vk-pricing]: https://www.vibekanban.com/pricing
[vk-selfhost]: https://vibekanban.com/docs/self-hosting/deploy-docker
[vk-agents]: https://vibekanban.com/docs/supported-coding-agents
[vk-chat]: https://vibekanban.com/docs/workspaces/chat-interface
[vk-general]: https://vibekanban.com/docs/settings/general
[vk-browser]: https://vibekanban.com/docs/browser-testing
[vk-start]: https://vibekanban.com/docs/getting-started
[vk-remote]: https://vibekanban.com/docs/remote-access
[vk-manage]: https://vibekanban.com/docs/workspaces/managing-workspaces
[vk-copy]: https://github.com/BloopAI/vibe-kanban/blob/main/crates/local-deployment/src/copy.rs

[gh-tmux]: https://github.com/tmux/tmux
[tm-man]: https://github.com/tmux/tmux/blob/3.8/tmux.1
[tm-licence]: https://github.com/tmux/tmux/blob/master/COPYING
[tm-options]: https://github.com/tmux/tmux/blob/3.8/options-table.c
[tm-kitty]: https://github.com/tmux/tmux/pull/5615
[tm-resurrect]: https://github.com/tmux-plugins/tmux-resurrect
[tm-start]: https://github.com/tmux/tmux/wiki/Getting-Started

[gh-zellij]: https://github.com/zellij-org/zellij
[zj-options]: https://zellij.dev/documentation/options.html
[zj-web]: https://zellij.dev/documentation/web-client.html
[zj-0451]: https://github.com/zellij-org/zellij/releases/tag/v0.45.1
[zj-0450]: https://github.com/zellij-org/zellij/releases/tag/v0.45.0
[zj-0440]: https://github.com/zellij-org/zellij/releases/tag/v0.44.0
[zj-0410]: https://github.com/zellij-org/zellij/releases/tag/v0.41.0
[zj-multi]: https://zellij.dev/news/multiplayer-sessions/
[zj-resurrect]: https://zellij.dev/documentation/session-resurrection.html
[zj-install]: https://zellij.dev/documentation/installation.html
[zj-online]: https://zellij.online

[gh-zmx]: https://github.com/neurosnap/zmx
[zmx-site]: https://zmx.sh
[zmx-76]: https://github.com/neurosnap/zmx/issues/76

[gh-herdr]: https://github.com/herdrdev/herdr
[hd-site]: https://herdr.dev
[hd-cloud]: https://herdr.dev/cloud/
[hd-remote]: https://herdr.dev/docs/persistence-remote/
[hd-agents]: https://herdr.dev/docs/agents/
[hd-3004]: https://github.com/herdrdev/herdr/issues/3004
[hd-state]: https://herdr.dev/docs/session-state/
[hd-config]: https://herdr.dev/docs/configuration/
[hd-cli]: https://herdr.dev/docs/cli-reference/
[hd-connect]: https://herdr.dev/docs/connecting-machines/
[hd-install]: https://herdr.dev/docs/install/

[gh-mosh]: https://github.com/mobile-shell/mosh
[mosh-site]: https://mosh.org
[mosh-term]: https://github.com/mobile-shell/mosh/blob/mosh-1.4.0/src/terminal/terminalfunctions.cc
[mosh-394]: https://github.com/mobile-shell/mosh/issues/394

[gh-et]: https://github.com/MisterTea/EternalTerminal
[et-proto]: https://github.com/MisterTea/EternalTerminal/blob/et-v7.0.0/docs/protocol.md
[et-client]: https://github.com/MisterTea/EternalTerminal/blob/et-v7.0.0/src/terminal/TerminalClientMain.cpp

[ts-pricing]: https://tailscale.com/pricing
[ts-v4]: https://tailscale.com/blog/pricing-v4
[ts-ssh]: https://tailscale.com/kb/1193/tailscale-ssh
[ts-oss]: https://tailscale.com/opensource/
[gh-tailscale]: https://github.com/tailscale/tailscale
[ts-serve]: https://tailscale.com/kb/1312/serve
[ts-funnel]: https://tailscale.com/kb/1223/funnel
[ts-console]: https://tailscale.com/docs/features/tailscale-ssh/tailscale-ssh-console
[ts-rec]: https://tailscale.com/docs/features/tailscale-ssh/tailscale-ssh-session-recording
